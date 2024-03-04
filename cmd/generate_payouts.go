package cmd

import (
	"errors"
	"os"

	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/core"
	"github.com/alis-is/tezpay/extension"
	"github.com/alis-is/tezpay/state"
	"github.com/alis-is/tezpay/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
			lastCompletedCycle := assertRunWithResultAndErrFmt(collector.GetLastCompletedCycle, EXIT_OPERTION_FAILED, "failed to get last completed cycle")
			cycle = lastCompletedCycle + cycle
		}

		if !config.IsDonatingToTezCapital() {
			log.Warn("With your current configuration you are not going to donate to tez.capital")
		}

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

		targetFile, _ := cmd.Flags().GetString(TO_FILE_FLAG)
		if targetFile != "" {
			assertRun(func() error {
				return writePayoutBlueprintToFile(targetFile, generationResult)
			}, EXIT_PAYOUT_WRITE_FAILURE)
			return
		}

		if state.Global.GetWantsOutputJson() {
			utils.PrintPayoutsAsJson(generationResult.Payouts)
			return
		}
		cycles := []int64{generationResult.Cycle}
		utils.PrintPayouts(generationResult.Payouts, utils.FormatCycleNumbers(cycles), false)
		utils.PrintPayouts(generationResult.Payouts, utils.FormatCycleNumbers(cycles), true)
	},
}

func init() {
	generatePayoutsCmd.Flags().Int64P(CYCLE_FLAG, "c", 0, "cycle to generate payouts for")
	generatePayoutsCmd.Flags().String(TO_FILE_FLAG, "", "saves generated payouts to specified file")
	generatePayoutsCmd.Flags().Bool(SKIP_BALANCE_CHECK_FLAG, false, "skips payout wallet balance check")
	RootCmd.AddCommand(generatePayoutsCmd)
}
