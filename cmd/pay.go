package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/core"
	reporter_engines "github.com/alis-is/tezpay/engines/reporter"
	"github.com/alis-is/tezpay/extension"
	"github.com/alis-is/tezpay/state"
	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
			assertRequireConfirmation("With your current configuration you are not going to donate to tez.capital. Do you want to proceed?")
		}

		if cycle <= 0 {
			lastCompletedCycle := assertRunWithResultAndErrFmt(collector.GetLastCompletedCycle, EXIT_OPERTION_FAILED, "failed to get last completed cycle")
			cycle = lastCompletedCycle + cycle
		}

		var generationResult *common.CyclePayoutBlueprint
		fromFile, _ := cmd.Flags().GetString(TO_FILE_FLAG)
		if fromFile != "" {
			generationResult = assertRunWithResult(func() (*common.CyclePayoutBlueprint, error) {
				return loadGeneratedPayoutsResultFromFile(fromFile)
			}, EXIT_PAYOUTS_READ_FAILURE)
		} else {
			var err error
			generationResult, err = core.GeneratePayouts(config, common.NewGeneratePayoutsEngines(collector, signer, notifyAdminFactory(config)),
				&common.GeneratePayoutsOptions{
					Cycle:            cycle,
					SkipBalanceCheck: skipBalanceCheck,
				})
			if errors.Is(err, constants.ErrNoCycleDataAvailable) {
				log.Infof("no data available for cycle %d, nothing to pay out...", cycle)
				return
			}
			if err != nil {
				log.Errorf("failed to generate payouts - %s", err)
				os.Exit(EXIT_OPERTION_FAILED)
			}
		}
		log.Info("checking past reports")
		preparationResult := assertRunWithResult(func() (*common.PreparePayoutsResult, error) {
			return core.PrepareCyclePayouts(generationResult, config, common.NewPreparePayoutsEngineContext(collector, signer, fsReporter, notifyAdminFactory(config)), &common.PreparePayoutsOptions{})
		}, EXIT_OPERTION_FAILED)

		cycles := []int64{generationResult.Cycle}
		if state.Global.GetWantsOutputJson() {
			utils.PrintPayoutsAsJson(preparationResult.ReportsOfPastSuccesfulPayouts)
			utils.PrintPayoutsAsJson(preparationResult.ValidPayouts)
		} else {
			utils.PrintPayouts(preparationResult.InvalidPayouts, fmt.Sprintf("Invalid - %s", utils.FormatCycleNumbers(cycles)), false)
			utils.PrintPayouts(preparationResult.AccumulatedPayouts, fmt.Sprintf("Accumulated - %s", utils.FormatCycleNumbers(cycles)), false)
			utils.PrintReports(preparationResult.ReportsOfPastSuccesfulPayouts, fmt.Sprintf("Already Successfull - #%s", utils.FormatCycleNumbers(cycles)), true)
			utils.PrintPayouts(preparationResult.ValidPayouts, fmt.Sprintf("Valid - %s", utils.FormatCycleNumbers(cycles)), true)
		}

		if len(preparationResult.ValidPayouts) == 0 {
			log.Info("nothing to pay out")
			notificator, _ := cmd.Flags().GetString(NOTIFICATOR_FLAG)
			if notificator != "" { // rerun notification through notificator if specified manually
				notifyPayoutsProcessed(config, &generationResult.Summary, notificator)
			}
			os.Exit(0)
		}

		if !confirmed {
			assertRequireConfirmation("Do you want to pay out above VALID payouts?")
		}

		log.Info("executing payout")
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
			log.Errorf("%d of operations failed", failedCount)
			os.Exit(EXIT_OPERTION_FAILED)
		}
		if silent, _ := cmd.Flags().GetBool(SILENT_FLAG); !silent {
			notifyPayoutsProcessedThroughAllNotificators(config, &generationResult.Summary)
		}
		utils.PrintBatchResults(executionResult.BatchResults, fmt.Sprintf("Results of #%s", utils.FormatCycleNumbers(cycles)), config.Network.Explorer)
	},
}

func init() {
	payCmd.Flags().Bool(CONFIRM_FLAG, false, "automatically confirms generated payouts")
	payCmd.Flags().Int64P(CYCLE_FLAG, "c", 0, "cycle to generate payouts for")
	payCmd.Flags().Bool(REPORT_TO_STDOUT, false, "prints them to stdout (wont write to file)")
	payCmd.Flags().String(FROM_FILE_FLAG, "", "loads payouts from file instead of generating on the fly")
	payCmd.Flags().Bool(DISABLE_SEPERATE_SC_PAYOUTS_FLAG, false, "disables smart contract separation (mixes txs and smart contract calls within batches)")
	payCmd.Flags().Bool(DISABLE_SEPERATE_FA_PAYOUTS_FLAG, false, "disables fa transfers separation (mixes txs and fa transfers within batches)")
	payCmd.Flags().BoolP(SILENT_FLAG, "s", false, "suppresses notifications")
	payCmd.Flags().String(NOTIFICATOR_FLAG, "", "Notify through specific notificator")
	payCmd.Flags().Bool(SKIP_BALANCE_CHECK_FLAG, false, "skips payout wallet balance check")
	payCmd.Flags().Bool(DRY_RUN_FLAG, false, "skips payout wallet balance check")

	RootCmd.AddCommand(payCmd)
}
