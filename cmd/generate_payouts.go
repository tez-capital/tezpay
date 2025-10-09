package cmd

import (
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/core"
	reporter_engines "github.com/tez-capital/tezpay/engines/reporter"
	"github.com/tez-capital/tezpay/extension"
	"github.com/tez-capital/tezpay/state"
)

var generatePayoutsCmd = &cobra.Command{
	Use:   "generate-payouts",
	Short: "generate payouts",
	Long:  "generates payouts without further processing",
	Run: func(cmd *cobra.Command, args []string) {
		cycle, _ := cmd.Flags().GetInt64(CYCLE_FLAG)
		skipBalanceCheck, _ := cmd.Flags().GetBool(SKIP_BALANCE_CHECK_FLAG)
		payoutPeriod, _ := cmd.Flags().GetInt64(PAYOUT_PERIOD_FLAG)
		payoutPeriod = getBoundedPayoutPeriod(payoutPeriod)
		config, collector, signer, _ := assertRunWithResult(loadConfigurationEnginesExtensions, EXIT_CONFIGURATION_LOAD_FAILURE).Unwrap()
		defer extension.CloseExtensions()

		if cycle <= 0 {
			lastCompletedCycle := assertRunWithResultAndErrorMessage(collector.GetLastCompletedCycle, EXIT_OPERTION_FAILED, "failed to get last completed cycle")
			cycle = lastCompletedCycle + cycle
		}

		cycles, isEndOfThePeriod := getCyclesInCompletedPeriod(cycle, payoutPeriod)
		if !isEndOfThePeriod {
			slog.Error("cycle is not at the end of the specified payout period", "cycle", cycle, "payout_period", payoutPeriod)
			os.Exit(EXIT_OPERTION_FAILED)
		}

		if !state.Global.IsDonationPromptDisabled() && !config.IsDonatingToTezCapital() {
			slog.Warn("⚠️  With your current configuration you are not going to donate to tez.capital 😔")
			time.Sleep(time.Second * 5)
		}
		generationResults := assertRunWithErrorHandler(func() (common.CyclePayoutBlueprints, error) {
			return generatePayoutsForCycles(cycles, config, collector, signer, &common.GeneratePayoutsOptions{
				SkipBalanceCheck: skipBalanceCheck,
			})
		}, handleGeneratePayoutsFailure)

		targetFile, _ := cmd.Flags().GetString(TO_FILE_FLAG)
		if targetFile != "" {
			assertRunWithErrorMessage(func() error {
				return writePayoutBlueprintToFile(targetFile, generationResults)
			}, EXIT_PAYOUT_WRITE_FAILURE, "failed to write payouts to file")
			return
		}

		switch {
		case state.Global.GetWantsOutputJson():
			slog.Info(constants.LOG_MESSAGE_PAYOUTS_GENERATED, constants.LOG_FIELD_CYCLES, cycles, constants.LOG_FIELD_CYCLE_PAYOUT_BLUEPRINT, generationResults, "phase", "result")
		default:
			fsReporter := reporter_engines.NewFileSystemReporter(config, &common.ReporterEngineOptions{
				IsReadOnly: true,
			})
			preparationResult := assertRunWithResult(func() (*common.PreparePayoutsResult, error) {
				return core.PreparePayouts(generationResults, config, common.NewPreparePayoutsEngineContext(collector, signer, fsReporter, notifyAdminFactory(config)), &common.PreparePayoutsOptions{
					Accumulate: true,
				})
			}, EXIT_OPERTION_FAILED)
			PrintPreparationResults(preparationResult, cycles, &PrintPreparationResultsOptions{AutoMergeRecords: true})
		}
	},
}

func init() {
	generatePayoutsCmd.Flags().Int64P(CYCLE_FLAG, "c", 0, "cycle to generate payouts for")
	generatePayoutsCmd.Flags().String(TO_FILE_FLAG, "", "saves generated payouts to specified file")
	generatePayoutsCmd.Flags().Int64(PAYOUT_PERIOD_FLAG, 1, "payout period")
	generatePayoutsCmd.Flags().Bool(SKIP_BALANCE_CHECK_FLAG, false, "skips payout wallet balance check")
	RootCmd.AddCommand(generatePayoutsCmd)
}
