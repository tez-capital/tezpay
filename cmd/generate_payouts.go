package cmd

import (
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/core/payout"
	"github.com/alis-is/tezpay/state"
	"github.com/alis-is/tezpay/utils"
	"github.com/spf13/cobra"
)

var generatePayoutsCmd = &cobra.Command{
	Use:   "generate-payouts",
	Short: "generate payouts",
	Long:  "generates payouts without further processing",
	Run: func(cmd *cobra.Command, args []string) {
		cycle, _ := cmd.Flags().GetInt64(CYCLE_FLAG)
		skipBalanceCheck, _ := cmd.Flags().GetBool(SKIP_BALANCE_CHECK_FLAG)
		config, _, signerEngine, _ := assertRunWithResult(loadConfigurationAndEngines, EXIT_CONFIGURATION_LOAD_FAILURE).Unwrap()

		payoutBlueprint := assertRunWithResultAndErrFmt(func() (*common.CyclePayoutBlueprint, error) {
			return payout.GeneratePayoutsWithPayoutAddress(signerEngine.GetKey(), cycle, config, common.GeneratePayoutsOptions{
				SkipBalanceCheck: skipBalanceCheck,
			})
		}, EXIT_OPERTION_FAILED, "failed to generate payouts - %s")

		targetFile, _ := cmd.Flags().GetString(TO_FILE_FLAG)
		if targetFile != "" {
			assertRun(func() error {
				return writePayoutBlueprintToFile(targetFile, payoutBlueprint)
			}, EXIT_PAYOUT_WRITE_FAILURE)
			return
		}

		if state.Global.GetWantsOutputJson() {
			utils.PrintPayoutsAsJson(payoutBlueprint.Payouts)
			return
		}

		utils.PrintInvalidPayoutRecipes(payoutBlueprint.Payouts, payoutBlueprint.Cycle)
		utils.PrintValidPayoutRecipes(payoutBlueprint.Payouts, payoutBlueprint.Cycle)
	},
}

func init() {
	generatePayoutsCmd.Flags().Int64P(CYCLE_FLAG, "c", 0, "cycle to generate payouts for")
	generatePayoutsCmd.Flags().String(TO_FILE_FLAG, "", "saves generated payouts to specified file")
	generatePayoutsCmd.Flags().Bool(SKIP_BALANCE_CHECK_FLAG, false, "skips payout wallet balance check")
	RootCmd.AddCommand(generatePayoutsCmd)
}
