package cmd

import (
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
		mixInContractCalls, _ := cmd.Flags().GetBool(DISABLE_SEPARATE_SC_PAYOUTS_FLAG)
		mixInFATransfers, _ := cmd.Flags().GetBool(DISABLE_SEPARATE_FA_PAYOUTS_FLAG)
		isDryRun, _ := cmd.Flags().GetBool(DRY_RUN_FLAG)

		payoutInterval, _ := cmd.Flags().GetInt64(PAYMENT_INTERVAL_CYCLES_FLAG)
		payoutInterval = getBoundedPayoutInterval(payoutInterval)
		intervalTriggerOffset, _ := cmd.Flags().GetInt64(INTERVAL_TRIGGER_OFFSET_FLAG)
		intervalTriggerOffset = boundToInterval(intervalTriggerOffset, payoutInterval, "interval-trigger-offset")
		includePrevious, _ := cmd.Flags().GetInt64(INCLUDE_PREVIOUS_CYCLES_FLAG)
		includePrevious = boundToInterval(includePrevious, payoutInterval*2, "include-previous-cycles")

		fsReporter := reporter_engines.NewFileSystemReporter(config, &common.ReporterEngineOptions{
			DryRun: isDryRun,
		})
		stdioReporter := reporter_engines.NewStdioReporter(config)

		if !state.Global.IsDonationPromptDisabled() && !config.IsDonatingToTezCapital() {
			assertRequireConfirmation("⚠️  With your current configuration you are not going to donate to tez.capital.😔 Do you want to proceed?")
		}

		var generationResults common.CyclePayoutBlueprints
		fromFile, _ := cmd.Flags().GetString(FROM_FILE_FLAG)
		fromStdin, _ := cmd.Flags().GetBool(FROM_STDIN_FLAG)

		cycles := make([]int64, 0, payoutInterval)
		switch {
		case fromStdin:
			generationResults = assertRunWithResult(func() (common.CyclePayoutBlueprints, error) {
				return loadGeneratedPayoutsFromStdin()
			}, EXIT_PAYOUTS_READ_FAILURE)

			cycles = generationResults.GetCycles()
		case fromFile != "":
			generationResults = assertRunWithResult(func() (common.CyclePayoutBlueprints, error) {
				return loadGeneratedPayoutsFromFile(fromFile)
			}, EXIT_PAYOUTS_READ_FAILURE)

			cycles = generationResults.GetCycles()
		default:
			if cycle <= 0 {
				lastCompletedCycle := assertRunWithResultAndErrorMessage(collector.GetLastCompletedCycle, EXIT_OPERTION_FAILED, "failed to get last completed cycle")
				cycle = lastCompletedCycle + cycle
			}

			var isEndOfThePeriod bool
			cycles, isEndOfThePeriod = getCyclesInCompletedPeriod(cycle, payoutInterval, intervalTriggerOffset, includePrevious)
			if !isEndOfThePeriod {
				slog.Error("cycle is not at the end of the specified payout interval", "cycle", cycle, "payout_interval", payoutInterval, "interval_trigger_offset", intervalTriggerOffset, "include_previous", includePrevious)
				os.Exit(EXIT_OPERTION_FAILED)
			}

			generationResults = assertRunWithErrorHandler(func() (common.CyclePayoutBlueprints, error) {
				return generatePayoutsForCycles(cycles, config, collector, signer, &common.GeneratePayoutsOptions{})
			}, handleGeneratePayoutsFailure)
		}

		slog.Info("acquiring lock", "cycles", cycles, "phase", "acquiring_lock")
		unlock, err := lockCyclesWithTimeout(time.Minute*10, cycles...)
		if err != nil {
			slog.Error("failed to acquire lock", "error", err.Error())
			os.Exit(EXIT_OPERTION_FAILED)
		}
		defer unlock()

		slog.Info("checking past reports")
		preparationResult := assertRunWithResult(func() (*common.PreparePayoutsResult, error) {
			return core.PreparePayouts(generationResults, config, common.NewPreparePayoutsEngineContext(collector, signer, fsReporter, notifyAdminFactory(config)), &common.PreparePayoutsOptions{
				Accumulate:       true,
				SkipBalanceCheck: skipBalanceCheck,
			})
		}, EXIT_OPERTION_FAILED)

		switch {
		case state.Global.GetWantsOutputJson():
			slog.Info(constants.LOG_MESSAGE_PREPAYOUT_SUMMARY,
				constants.LOG_FIELD_CYCLES, cycles,
				constants.LOG_FIELD_REPORTS_OF_PAST_PAYOUTS, preparationResult.ReportsOfPastSuccessfulPayouts,
				constants.LOG_FIELD_ACCUMULATED_PAYOUTS, preparationResult.ValidPayouts,
				constants.LOG_FIELD_VALID_PAYOUTS, preparationResult.ValidPayouts,
				constants.LOG_FIELD_INVALID_PAYOUTS, preparationResult.InvalidPayouts,
			)
		default:
			utils.PrintPreparePayoutsResult(preparationResult, &utils.PrintPreparePayoutsResultOptions{AutoMergeRecords: true})
		}

		if len(preparationResult.ValidPayouts) == 0 {
			slog.Info("nothing to pay out", "phase", "result")
			notificator, _ := cmd.Flags().GetString(NOTIFICATOR_FLAG)
			if notificator != "" { // rerun notification through notificator if specified manually
				notifyPayoutsProcessed(config, utils.GeneratePayoutSummaryFromPreparationResult(preparationResult), notificator)
			}
			os.Exit(0)
		}

		if !confirmed {
			msg := "Do you want to pay out above VALID payouts?"
			if isDryRun {
				msg = msg + " " + constants.DRY_RUN_NOTE
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
			notifyPayoutsProcessedThroughAllNotificators(config, &executionResult.Summary)
		}
		switch {
		case state.Global.GetWantsOutputJson():
			slog.Info(constants.LOG_MESSAGE_PAYOUTS_EXECUTED, constants.LOG_FIELD_CYCLES, cycles, "phase", "result")
		default:
			utils.PrintBatchResults(executionResult.BatchResults, fmt.Sprintf("Results of %s", utils.FormatCycleNumbers(cycles...)), config.Network.Explorer)
		}
		PrintPayoutWalletRemainingBalance(collector, signer)
	},
}

func init() {
	payCmd.Flags().Bool(CONFIRM_FLAG, false, "automatically confirms generated payouts")
	payCmd.Flags().Int64P(CYCLE_FLAG, "c", 0, "cycle to generate payouts for")
	payCmd.Flags().Int64(PAYMENT_INTERVAL_CYCLES_FLAG, 1, "number of cycles between consecutive payouts")
	payCmd.Flags().Int64(INTERVAL_TRIGGER_OFFSET_FLAG, 0, "offset (in cycles) to trigger payouts within the interval")
	payCmd.Flags().Int64(INCLUDE_PREVIOUS_CYCLES_FLAG, 0, "number of previous cycles to reevaluate for missed or failed payouts")
	payCmd.Flags().Bool(REPORT_TO_STDOUT, false, "prints them to stdout (wont write to file)")
	payCmd.Flags().String(FROM_FILE_FLAG, "", "loads payouts from file instead of generating on the fly")
	payCmd.Flags().Bool(FROM_STDIN_FLAG, false, "loads payouts from stdin instead of generating on the fly")
	payCmd.Flags().Bool(DISABLE_SEPARATE_SC_PAYOUTS_FLAG, false, "disables smart contract separation (mixes txs and smart contract calls within batches)")
	payCmd.Flags().Bool(DISABLE_SEPARATE_FA_PAYOUTS_FLAG, false, "disables fa transfers separation (mixes txs and fa transfers within batches)")
	payCmd.Flags().BoolP(SILENT_FLAG, "s", false, "suppresses notifications")
	payCmd.Flags().String(NOTIFICATOR_FLAG, "", "Notify through specific notificator")
	payCmd.Flags().Bool(SKIP_BALANCE_CHECK_FLAG, false, "skips payout wallet balance check")
	payCmd.Flags().Bool(DRY_RUN_FLAG, false, "Performs all actions except sending transactions. Reports are stored in 'reports/dry' folder")

	RootCmd.AddCommand(payCmd)
}
