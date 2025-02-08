package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/core"
	reporter_engines "github.com/tez-capital/tezpay/engines/reporter"
	"github.com/tez-capital/tezpay/extension"
	"github.com/tez-capital/tezpay/state"
	"github.com/tez-capital/tezpay/utils"
)

var payCmd = &cobra.Command{
	Use:   "pay",
	Short: "manual payout",
	Long:  "runs manual payout",
	Run: func(cmd *cobra.Command, args []string) {
		config, collector, signer, transactor := assertRunWithResult(loadConfigurationEnginesExtensions, EXIT_CONFIGURATION_LOAD_FAILURE).Unwrap()
		defer extension.CloseExtensions()

		cycle, _ := cmd.Flags().GetInt64(CYCLE_FLAG)
		skipBalanceCheck, _ := cmd.Flags().GetBool(SKIP_BALANCE_CHECK_FLAG)
		confirmed, _ := cmd.Flags().GetBool(CONFIRM_FLAG)
		mixInContractCalls, _ := cmd.Flags().GetBool(DISABLE_SEPERATE_SC_PAYOUTS_FLAG)
		mixInFATransfers, _ := cmd.Flags().GetBool(DISABLE_SEPERATE_FA_PAYOUTS_FLAG)
		isDryRun, _ := cmd.Flags().GetBool(DRY_RUN_FLAG)

		fsReporter := reporter_engines.NewFileSystemReporter(config, &common.ReporterEngineOptions{
			DryRun: isDryRun,
		})
		stdioReporter := reporter_engines.NewStdioReporter(config)

		if !state.Global.IsDonationPromptDisabled() && !config.IsDonatingToTezCapital() {
			assertRequireConfirmation("⚠️  With your current configuration you are not going to donate to tez.capital.😔 Do you want to proceed?")
		}

		var generationResult *common.CyclePayoutBlueprint
		fromFile, _ := cmd.Flags().GetString(FROM_FILE_FLAG)
		fromStdin, _ := cmd.Flags().GetBool(FROM_STDIN_FLAG)
		switch {
		case fromStdin:
			generationResult = assertRunWithResult(func() (*common.CyclePayoutBlueprint, error) {
				return loadGeneratedPayoutsFromStdin()
			}, EXIT_PAYOUTS_READ_FAILURE)
		case fromFile != "":
			generationResult = assertRunWithResult(func() (*common.CyclePayoutBlueprint, error) {
				return loadGeneratedPayoutsFromFile(fromFile)
			}, EXIT_PAYOUTS_READ_FAILURE)
		default:
			if cycle <= 0 {
				lastCompletedCycle := assertRunWithResultAndErrorMessage(collector.GetLastCompletedCycle, EXIT_OPERTION_FAILED, "failed to get last completed cycle")
				cycle = lastCompletedCycle + cycle
			}

			var err error
			generationResult, err = core.GeneratePayouts(config, common.NewGeneratePayoutsEngines(collector, signer, notifyAdminFactory(config)),
				&common.GeneratePayoutsOptions{
					Cycle:            cycle,
					SkipBalanceCheck: skipBalanceCheck,
				})
			if errors.Is(err, constants.ErrNoCycleDataAvailable) {
				slog.Info("no data available for cycle, skipping", "cycle", cycle)
				return
			}
			if err != nil {
				slog.Error("failed to generate payouts", "error", err.Error())
				time.Sleep(time.Minute * 5)
				os.Exit(EXIT_OPERTION_FAILED)
			}
		}

		cycles := lo.Reduce(generationResult.Payouts, func(acc []int64, cp common.PayoutRecipe, _ int) []int64 {
			if lo.Contains(acc, cp.Cycle) {
				return acc
			}
			return append(acc, cp.Cycle)
		}, []int64{})

		slog.Info("acquiring lock", "cycles", cycles, "phase", "acquiring_lock")
		unlock, err := lockCyclesWithTimeout(time.Minute*10, cycles...)
		if err != nil {
			slog.Error("failed to acquire lock", "error", err.Error())
			os.Exit(EXIT_OPERTION_FAILED)
		}
		defer unlock()

		slog.Info("checking past reports")
		preparationResult := assertRunWithResult(func() (*common.PreparePayoutsResult, error) {
			return core.PrepareCyclePayouts(generationResult, config, common.NewPreparePayoutsEngineContext(collector, signer, fsReporter, notifyAdminFactory(config)), &common.PreparePayoutsOptions{})
		}, EXIT_OPERTION_FAILED)

		switch {
		case state.Global.GetWantsOutputJson():
			slog.Info(constants.LOG_MESSAGE_PREPAYOUT_SUMMARY,
				constants.LOG_FIELD_CYCLES, cycles,
				constants.LOG_FIELD_REPORTS_OF_PAST_PAYOUTS, preparationResult.ReportsOfPastSuccesfulPayouts,
				constants.LOG_FIELD_ACCUMULATED_PAYOUTS, preparationResult.AccumulatedPayouts,
				constants.LOG_FIELD_VALID_PAYOUTS, preparationResult.ValidPayouts,
				constants.LOG_FIELD_INVALID_PAYOUTS, preparationResult.InvalidPayouts,
			)
		default:
			PrintPreparationResults(preparationResult, cycles...)
		}

		if len(preparationResult.ValidPayouts) == 0 {
			slog.Info("nothing to pay out", "phase", "result")
			notificator, _ := cmd.Flags().GetString(NOTIFICATOR_FLAG)
			if notificator != "" { // rerun notification through notificator if specified manually
				notifyPayoutsProcessed(config, &generationResult.Summary, notificator)
			}
			os.Exit(0)
		}

		if !confirmed {
			msg := "Do you want to pay out above VALID payouts?"
			if isDryRun {
				msg = msg + " (dry-run)"
			}
			assertRequireConfirmation(msg)
		}

		slog.Info("executing payouts")
		executionResult := assertRunWithResult(func() (*common.ExecutePayoutsResult, error) {
			var reporter common.ReporterEngine
			reporter = fsReporter
			if reportToStdout, _ := cmd.Flags().GetBool(REPORT_TO_STDOUT); reportToStdout {
				reporter = stdioReporter
			}
			return core.ExecutePayouts(preparationResult, config, common.NewExecutePayoutsEngineContext(signer, transactor, reporter, notifyAdminFactory(config)), &common.ExecutePayoutsOptions{
				MixInContractCalls: mixInContractCalls,
				MixInFATransfers:   mixInFATransfers,
				DryRun:             isDryRun,
			})
		}, EXIT_OPERTION_FAILED)

		// notify
		failedCount := lo.CountBy(executionResult.BatchResults, func(br common.BatchResult) bool { return !br.IsSuccess })
		if len(executionResult.BatchResults) > 0 && failedCount > 0 {
			slog.Error("failed operations detected", "failed", failedCount, "total", len(executionResult.BatchResults))
			os.Exit(EXIT_OPERTION_FAILED)
		}
		if silent, _ := cmd.Flags().GetBool(SILENT_FLAG); !silent && !isDryRun {
			notifyPayoutsProcessedThroughAllNotificators(config, &generationResult.Summary)
		}
		switch {
		case state.Global.GetWantsOutputJson():
			slog.Info(constants.LOG_MESSAGE_PAYOUTS_EXECUTED, constants.LOG_FIELD_CYCLES, cycles, "phase", "result")
		default:
			utils.PrintBatchResults(executionResult.BatchResults, fmt.Sprintf("Results of #%s", utils.FormatCycleNumbers(cycles...)), config.Network.Explorer)
		}
		PrintPayoutWalletRemainingBalance(collector, signer)
	},
}

func init() {
	payCmd.Flags().Bool(CONFIRM_FLAG, false, "automatically confirms generated payouts")
	payCmd.Flags().Int64P(CYCLE_FLAG, "c", 0, "cycle to generate payouts for")
	payCmd.Flags().Bool(REPORT_TO_STDOUT, false, "prints them to stdout (wont write to file)")
	payCmd.Flags().String(FROM_FILE_FLAG, "", "loads payouts from file instead of generating on the fly")
	payCmd.Flags().Bool(FROM_STDIN_FLAG, false, "loads payouts from stdin instead of generating on the fly")
	payCmd.Flags().Bool(DISABLE_SEPERATE_SC_PAYOUTS_FLAG, false, "disables smart contract separation (mixes txs and smart contract calls within batches)")
	payCmd.Flags().Bool(DISABLE_SEPERATE_FA_PAYOUTS_FLAG, false, "disables fa transfers separation (mixes txs and fa transfers within batches)")
	payCmd.Flags().BoolP(SILENT_FLAG, "s", false, "suppresses notifications")
	payCmd.Flags().String(NOTIFICATOR_FLAG, "", "Notify through specific notificator")
	payCmd.Flags().Bool(SKIP_BALANCE_CHECK_FLAG, false, "skips payout wallet balance check")
	payCmd.Flags().Bool(DRY_RUN_FLAG, false, "skips payout wallet balance check")

	RootCmd.AddCommand(payCmd)
}
