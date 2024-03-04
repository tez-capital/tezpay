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

func parseDateFlags(cmd *cobra.Command) (time.Time, time.Time, error) {
	startDateFlag, _ := cmd.Flags().GetString(START_DATE_FLAG)
	endDateFlag, _ := cmd.Flags().GetString(END_DATE_FLAG)
	monthFlag, _ := cmd.Flags().GetString(MONTH_FLAG)

	if startDateFlag != "" && endDateFlag != "" && monthFlag != "" {
		return time.Time{}, time.Time{}, errors.New("only start date and end date or month can be specified")
	}
	if startDateFlag != "" && endDateFlag != "" {
		startDate, err := time.Parse("2006-01-02", startDateFlag)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("failed to parse start date - %s", err)
		}
		endDate, err := time.Parse("2006-01-02", endDateFlag)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("failed to parse end date - %s", err)
		}
		return startDate, endDate.Add(-time.Nanosecond), nil
	}
	if monthFlag != "" {
		month, err := time.Parse("2006-01", monthFlag)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("failed to parse month - %s", err)
		}
		startDate := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
		endDate := startDate.AddDate(0, 1, 0).Add(-time.Nanosecond)
		return startDate, endDate, nil
	}
	return time.Time{}, time.Time{}, errors.New("invalid date range")
}

var payDateRangeCmd = &cobra.Command{
	Use:   "pay-date-range",
	Short: "EXPERIMENTAL: payout for date range",
	Long:  "EXPERIMENTAL: runs payout for date range",
	Run: func(cmd *cobra.Command, args []string) {
		config, collector, signer, transactor := assertRunWithResult(loadConfigurationEnginesExtensions, EXIT_CONFIGURATION_LOAD_FAILURE).Unwrap()
		defer extension.CloseExtensions()

		skipBalanceCheck, _ := cmd.Flags().GetBool(SKIP_BALANCE_CHECK_FLAG)
		confirmed, _ := cmd.Flags().GetBool(CONFIRM_FLAG)
		mixInContractCalls, _ := cmd.Flags().GetBool(DISABLE_SEPERATE_SC_PAYOUTS_FLAG)
		mixInFATransfers, _ := cmd.Flags().GetBool(DISABLE_SEPERATE_FA_PAYOUTS_FLAG)

		fsReporter := reporter_engines.NewFileSystemReporter(config)
		stdioReporter := reporter_engines.NewStdioReporter(config)

		if !state.Global.IsDonationPromptDisabled() && !config.IsDonatingToTezCapital() {
			assertRequireConfirmation("With your current configuration you are not going to donate to tez.capital. Do you want to proceed?")
		}

		startDate, endDate, err := parseDateFlags(cmd)
		if err != nil {
			log.Error(err.Error())
			os.Exit(EXIT_OPERTION_FAILED)
		}

		assertRequireConfirmation(fmt.Sprintf("NOTE: The payout for date ranges is an EXPERIMENTAL feature. Exercise caution!\n\nDo you want to generate payouts for date range: %s - %s?", startDate.Format(time.RFC3339), endDate.Format(time.RFC3339)))

		cycles, err := collector.GetCyclesInDateRange(startDate, endDate)
		if err != nil {
			log.Errorf("failed to get cycles in date range - %s", err)
			os.Exit(EXIT_OPERTION_FAILED)
		}

		log.Infof("Generating payouts for cycles: %s", utils.FormatCycleNumbers(cycles))
		generationResults := make(common.CyclePayoutBlueprints, 0, len(cycles))

		channels := make([]chan *common.CyclePayoutBlueprint, 0, len(cycles))

		for _, cycle := range cycles {
			ch := make(chan *common.CyclePayoutBlueprint)
			channels = append(channels, ch)
			go func() {
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
				ch <- generationResult
			}()
		}
		for _, ch := range channels {
			generationResult := <-ch
			if generationResult != nil {
				generationResults = append(generationResults, generationResult)
			}
		}

		log.Info("checking past reports")
		preparationResult := assertRunWithResult(func() (*common.PreparePayoutsResult, error) {
			return core.PreparePayouts(generationResults, config, common.NewPreparePayoutsEngineContext(collector, fsReporter, notifyAdminFactory(config)), &common.PreparePayoutsOptions{
				Accumulate: true,
			})
		}, EXIT_OPERTION_FAILED)

		if state.Global.GetWantsOutputJson() {
			utils.PrintPayoutsAsJson(preparationResult.ReportsOfPastSuccesfulPayouts)
			utils.PrintPayoutsAsJson(preparationResult.AccumulatedPayouts)
			utils.PrintPayoutsAsJson(preparationResult.ValidPayouts)
		} else {
			utils.PrintPayouts(preparationResult.InvalidPayouts, fmt.Sprintf("Invalid - %s", utils.FormatCycleNumbers(cycles)), false)
			utils.PrintPayouts(preparationResult.AccumulatedPayouts, fmt.Sprintf("Accumulated - %s", utils.FormatCycleNumbers(cycles)), false)
			utils.PrintReports(preparationResult.ReportsOfPastSuccesfulPayouts, fmt.Sprintf("Already Successfull - %s", utils.FormatCycleNumbers(cycles)), true)
			utils.PrintPayouts(preparationResult.ValidPayouts, fmt.Sprintf("Valid - %s", utils.FormatCycleNumbers(cycles)), true)
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
	payDateRangeCmd.Flags().String(START_DATE_FLAG, "", "start date for payout generation (format: 2024-02-01)")
	payDateRangeCmd.Flags().String(END_DATE_FLAG, "", "end date for payout generation (format: 2024-02-01)")
	payDateRangeCmd.Flags().String(MONTH_FLAG, "", "month to generate payout for (format: 2024-02)")
	payDateRangeCmd.Flags().Bool(REPORT_TO_STDOUT, false, "prints them to stdout (wont write to file)")
	payDateRangeCmd.Flags().Bool(DISABLE_SEPERATE_SC_PAYOUTS_FLAG, false, "disables smart contract separation (mixes txs and smart contract calls within batches)")
	payDateRangeCmd.Flags().Bool(DISABLE_SEPERATE_FA_PAYOUTS_FLAG, false, "disables fa transfers separation (mixes txs and fa transfers within batches)")
	payDateRangeCmd.Flags().BoolP(SILENT_FLAG, "s", false, "suppresses notifications")
	payDateRangeCmd.Flags().String(NOTIFICATOR_FLAG, "", "Notify through specific notificator")
	payDateRangeCmd.Flags().Bool(SKIP_BALANCE_CHECK_FLAG, false, "skips payout wallet balance check")

	RootCmd.AddCommand(payDateRangeCmd)
}
