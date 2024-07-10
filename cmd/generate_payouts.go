package cmd

import (
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/core"
	"github.com/tez-capital/tezpay/extension"
	"github.com/tez-capital/tezpay/state"
	"github.com/tez-capital/tezpay/utils"
)

var generatePayoutsCmd = &cobra.Command{
	Use:   "generate-payouts",
	Short: "generate payouts",
	Long:  "generates payouts without further processing",
	Run: func(cmd *cobra.Command, args []string) {
		cycle, _ := cmd.Flags().GetInt64(CYCLE_FLAG)
		skipBalanceCheck, _ := cmd.Flags().GetBool(SKIP_BALANCE_CHECK_FLAG)
		config, collector, signer, _ := assertRunWithResult(loadConfigurationEnginesExtensions, EXIT_CONFIGURATION_LOAD_FAILURE).Unwrap()
		defer extension.CloseExtensions()

		if cycle <= 0 {
			lastCompletedCycle := assertRunWithResultAndErrorMessage(collector.GetLastCompletedCycle, EXIT_OPERTION_FAILED, "failed to get last completed cycle")
			cycle = lastCompletedCycle + cycle
		}

		if !state.Global.IsDonationPromptDisabled() && !config.IsDonatingToTezCapital() {
			slog.Warn("⚠️  With your current configuration you are not going to donate to tez.capital 😔")
			time.Sleep(time.Second * 5)
		}

		generationResult, err := core.GeneratePayouts(config, common.NewGeneratePayoutsEngines(collector, signer, notifyAdminFactory(config)),
			&common.GeneratePayoutsOptions{
				Cycle:            cycle,
				SkipBalanceCheck: skipBalanceCheck,
			})
		if errors.Is(err, constants.ErrNoCycleDataAvailable) {
			slog.Info("no data available, nothing to pay out", "cycle", cycle)
			return
		}
		if err != nil {
			slog.Error("failed to generate payouts", "error", err.Error())
			os.Exit(EXIT_OPERTION_FAILED)
		}

		targetFile, _ := cmd.Flags().GetString(TO_FILE_FLAG)
		if targetFile != "" {
			assertRunWithErrorMessage(func() error {
				return writePayoutBlueprintToFile(targetFile, generationResult)
			}, EXIT_PAYOUT_WRITE_FAILURE, "failed to write payouts to file")
			return
		}

		cycles := []int64{generationResult.Cycle}

		switch {
		case state.Global.GetWantsOutputJson():
			slog.Info(constants.LOG_MESSAGE_PAYOUTS_GENERATED, constants.LOG_FIELD_CYCLES, cycles, constants.LOG_FIELD_PAYOUTS, generationResult.Payouts, constants.LOG_FIELD_SUMMARY, generationResult.Summary, "phase", "result")
		default:
			utils.PrintPayouts(utils.OnlyInvalidPayouts(generationResult.Payouts), utils.FormatCycleNumbers(cycles...), false)
			utils.PrintPayouts(utils.OnlyValidPayouts(generationResult.Payouts), utils.FormatCycleNumbers(cycles...), true)
		}
	},
}

func init() {
	generatePayoutsCmd.Flags().Int64P(CYCLE_FLAG, "c", 0, "cycle to generate payouts for")
	generatePayoutsCmd.Flags().String(TO_FILE_FLAG, "", "saves generated payouts to specified file")
	generatePayoutsCmd.Flags().Bool(SKIP_BALANCE_CHECK_FLAG, false, "skips payout wallet balance check")
	RootCmd.AddCommand(generatePayoutsCmd)
}
