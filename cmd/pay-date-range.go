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

var payDateRangeCmd = &cobra.Command{
	Use:   "pay-date-range",
	Short: "EXPERIMENTAL: payout for date range",
	Long:  "EXPERIMENTAL: runs payout for date range",
	Run: func(cmd *cobra.Command, args []string) {
		config, collector, signer, transactor := assertRunWithResult(loadConfigurationEnginesExtensions, EXIT_CONFIGURATION_LOAD_FAILURE).Unwrap()
		defer extension.CloseExtensions()

		skipBalanceCheck, _ := cmd.Flags().GetBool(SKIP_BALANCE_CHECK_FLAG)
		confirmed, _ := cmd.Flags().GetBool(CONFIRM_FLAG)
		mixInContractCalls, _ := cmd.Flags().GetBool(DISABLE_SEPARATE_SC_PAYOUTS_FLAG)
		mixInFATransfers, _ := cmd.Flags().GetBool(DISABLE_SEPARATE_FA_PAYOUTS_FLAG)
		isDryRun, _ := cmd.Flags().GetBool(DRY_RUN_FLAG)

		fsReporter := reporter_engines.NewFileSystemReporter(config, &common.ReporterEngineOptions{
			DryRun: isDryRun,
		})
		stdioReporter := reporter_engines.NewStdioReporter(config)

		if !state.Global.IsDonationPromptDisabled() && !config.IsDonatingToTezCapital() {
			assertRequireConfirmation("⚠️  With your current configuration you are not going to donate to tez.capital.😔 Do you want to proceed?")
		}

		startDate, endDate, err := parseDateFlags(cmd)
		if err != nil {
			slog.Error("failed to parse date flags", "error", err.Error())
			os.Exit(EXIT_OPERTION_FAILED)
		}

		if endDate.After(time.Now()) {
			slog.Error("end date cannot be in the future")
			os.Exit(EXIT_OPERTION_FAILED)
		}

		assertRequireConfirmation(fmt.Sprintf("NOTE: The payout for date ranges is an EXPERIMENTAL feature. Exercise caution!\n\nDo you want to generate payouts for date range: %s - %s?", startDate.Format(time.RFC3339), endDate.Format(time.RFC3339)))

		cycles, err := collector.GetCyclesInDateRange(startDate, endDate)
		if err != nil {
			slog.Error("failed to get cycles in date selected range", "error", err.Error())
			os.Exit(EXIT_OPERTION_FAILED)
		}

		slog.Info("acquiring lock", "cycles", cycles, "phase", "acquiring_lock")
		unlock, err := lockCyclesWithTimeout(time.Minute*10, cycles...)
		if err != nil {
			slog.Error("failed to acquire lock", "error", err.Error())
			os.Exit(EXIT_OPERTION_FAILED)
		}
		defer unlock()

		slog.Info("generating payouts for cycles in the date range", "date_range", fmt.Sprintf("%s - %s", startDate.Format(time.RFC3339), endDate.Format(time.RFC3339)), "cycles", cycles)
		generationResults := assertRunWithErrorHandler(func() (common.CyclePayoutBlueprints, error) {
			return generatePayoutsForCycles(cycles, config, collector, signer, &common.GeneratePayoutsOptions{})
		}, handleGeneratePayoutsFailure)

		slog.Info("checking reports of past payouts")
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
			slog.Info("nothing to pay out")
			notificator, _ := cmd.Flags().GetString(NOTIFICATOR_FLAG)
			if notificator != "" { // rerun notification through notificator if specified manually
				notifyPayoutsProcessed(config, utils.GeneratePayoutSummaryFromPreparationResult(preparationResult), notificator)
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

		slog.Info("executing payout")
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
			slog.Error("failed operations detected", "failed_count", failedCount, "total_count", len(executionResult.BatchResults))
			os.Exit(EXIT_OPERTION_FAILED)
		}
		if silent, _ := cmd.Flags().GetBool(SILENT_FLAG); !silent && !isDryRun {
			notifyPayoutsProcessedThroughAllNotificators(config, &executionResult.Summary)
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
	payDateRangeCmd.Flags().Bool(CONFIRM_FLAG, false, "automatically confirms generated payouts")
	payDateRangeCmd.Flags().String(START_DATE_FLAG, "", "start date for payout generation (format: 2024-02-01)")
	payDateRangeCmd.Flags().String(END_DATE_FLAG, "", "end date for payout generation (format: 2024-02-01)")
	payDateRangeCmd.Flags().String(MONTH_FLAG, "", "month to generate payout for (format: 2024-02)")
	payDateRangeCmd.Flags().Bool(REPORT_TO_STDOUT, false, "prints them to stdout (wont write to file)")
	payDateRangeCmd.Flags().Bool(DISABLE_SEPARATE_SC_PAYOUTS_FLAG, false, "disables smart contract separation (mixes txs and smart contract calls within batches)")
	payDateRangeCmd.Flags().Bool(DISABLE_SEPARATE_FA_PAYOUTS_FLAG, false, "disables fa transfers separation (mixes txs and fa transfers within batches)")
	payDateRangeCmd.Flags().BoolP(SILENT_FLAG, "s", false, "suppresses notifications")
	payDateRangeCmd.Flags().String(NOTIFICATOR_FLAG, "", "Notify through specific notificator")
	payDateRangeCmd.Flags().Bool(SKIP_BALANCE_CHECK_FLAG, false, "skips payout wallet balance check")
	payDateRangeCmd.Flags().Bool(DRY_RUN_FLAG, false, "Performs all actions except sending transactions. Reports are stored in 'reports/dry' folder")

	RootCmd.AddCommand(payDateRangeCmd)
}
