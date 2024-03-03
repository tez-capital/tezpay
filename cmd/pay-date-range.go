package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

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

var payDateRangeCmd = &cobra.Command{
	Use:   "pay-date-range",
	Short: "manual payout",
	Long:  "runs manual payout",
	Run: func(cmd *cobra.Command, args []string) {
		config, collector, signer, transactor := assertRunWithResult(loadConfigurationEnginesExtensions, EXIT_CONFIGURATION_LOAD_FAILURE).Unwrap()
		defer extension.CloseExtensions()

		startDateFlag, _ := cmd.Flags().GetString(START_DATE_FLAG)
		endDateFlag, _ := cmd.Flags().GetString(END_DATE_FLAG)
		skipBalanceCheck, _ := cmd.Flags().GetBool(SKIP_BALANCE_CHECK_FLAG)
		confirmed, _ := cmd.Flags().GetBool(CONFIRM_FLAG)
		mixInContractCalls, _ := cmd.Flags().GetBool(DISABLE_SEPERATE_SC_PAYOUTS_FLAG)
		mixInFATransfers, _ := cmd.Flags().GetBool(DISABLE_SEPERATE_FA_PAYOUTS_FLAG)

		fsReporter := reporter_engines.NewFileSystemReporter(config)
		stdioReporter := reporter_engines.NewStdioReporter(config)

		if !state.Global.IsDonationPromptDisabled() && !config.IsDonatingToTezCapital() {
			assertRequireConfirmation("With your current configuration you are not going to donate to tez.capital. Do you want to proceed?")
		}

		startDate, err := time.Parse("2006-01-02", startDateFlag)
		if err != nil {
			log.Errorf("failed to parse start date - %s", err)
			os.Exit(EXIT_OPERTION_FAILED)
		}
		endDate, err := time.Parse("2006-01-02", endDateFlag)
		if err != nil {
			log.Errorf("failed to parse end date - %s", err)
			os.Exit(EXIT_OPERTION_FAILED)
		}

		cycles, err := collector.GetCyclesInDateRange(startDate, endDate)
		if err != nil {
			log.Errorf("failed to get cycles in date range - %s", err)
			os.Exit(EXIT_OPERTION_FAILED)
		}

		generationResults := make(common.CyclePayoutBlueprints, 0, len(cycles))

		for _, cycle := range cycles {
			generationResult, err := core.GeneratePayouts(config, common.NewGeneratePayoutsEngines(collector, signer, notifyAdminFactory(config)),
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
			generationResults = append(generationResults, generationResult)
		}
		log.Info("checking past reports")
		preparationResult := assertRunWithResult(func() (*common.PreparePayoutsResult, error) {
			return core.PreparePayouts(generationResults, config, common.NewPreparePayoutsEngineContext(collector, fsReporter, notifyAdminFactory(config)), &common.PreparePayoutsOptions{})
		}, EXIT_OPERTION_FAILED)

		if state.Global.GetWantsOutputJson() {
			utils.PrintPayoutsAsJson(preparationResult.ReportsOfPastSuccesfulPayouts)
			utils.PrintPayoutsAsJson(preparationResult.ValidPayouts)
		} else {
			utils.PrintInvalidPayoutRecipes(preparationResult.ValidPayouts, cycles)
			utils.PrintReports(preparationResult.ReportsOfPastSuccesfulPayouts, fmt.Sprintf("Already Successfull - %s", utils.FormatCycleNumbers(cycles)), true)
			utils.PrintValidPayoutRecipes(preparationResult.ValidPayouts, cycles)
		}

		if len(utils.OnlyValidPayouts(preparationResult.ValidPayouts)) == 0 {
			log.Info("nothing to pay out")
			notificator, _ := cmd.Flags().GetString(NOTIFICATOR_FLAG)
			if notificator != "" { // rerun notification through notificator if specified manually
				notifyPayoutsProcessed(config, generationResults.GetSummary(), notificator)
			}
			os.Exit(0)
		}

		if !confirmed {
			assertRequireConfirmation("Do you want to pay out above VALID payouts?")
		}
		os.Exit(0) // TODO: remove this line

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
			})
		}, EXIT_OPERTION_FAILED)

		// notify
		failedCount := lo.CountBy(executionResult.BatchResults, func(br common.BatchResult) bool { return !br.IsSuccess })
		if len(executionResult.BatchResults) > 0 && failedCount > 0 {
			log.Errorf("%d of operations failed", failedCount)
			os.Exit(EXIT_OPERTION_FAILED)
		}
		if silent, _ := cmd.Flags().GetBool(SILENT_FLAG); !silent {
			summary := generationResults.GetSummary()
			summary.PaidDelegators = executionResult.PaidDelegators
			notifyPayoutsProcessedThroughAllNotificators(config, summary)
		}
		utils.PrintBatchResults(executionResult.BatchResults, fmt.Sprintf("Results of #%s", utils.FormatCycleNumbers(cycles)), config.Network.Explorer)
	},
}

func init() {
	payDateRangeCmd.Flags().Bool(CONFIRM_FLAG, false, "automatically confirms generated payouts")
	//payCmd.Flags().Int64P(CYCLE_FLAG, "c", 0, "cycle to generate payouts for")
	payDateRangeCmd.Flags().String(START_DATE_FLAG, "", "start date for payout generation")
	payDateRangeCmd.Flags().String(END_DATE_FLAG, "", "end date for payout generation")
	payDateRangeCmd.Flags().Bool(REPORT_TO_STDOUT, false, "prints them to stdout (wont write to file)")
	payDateRangeCmd.Flags().Bool(DISABLE_SEPERATE_SC_PAYOUTS_FLAG, false, "disables smart contract separation (mixes txs and smart contract calls within batches)")
	payDateRangeCmd.Flags().Bool(DISABLE_SEPERATE_FA_PAYOUTS_FLAG, false, "disables fa transfers separation (mixes txs and fa transfers within batches)")
	payDateRangeCmd.Flags().BoolP(SILENT_FLAG, "s", false, "suppresses notifications")
	payDateRangeCmd.Flags().String(NOTIFICATOR_FLAG, "", "Notify through specific notificator")
	payDateRangeCmd.Flags().Bool(SKIP_BALANCE_CHECK_FLAG, false, "skips payout wallet balance check")

	RootCmd.AddCommand(payCmd)
}
