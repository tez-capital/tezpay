//go:build !wasm

package cmd

import (
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/core"
	"github.com/alis-is/tezpay/extension"
	"github.com/alis-is/tezpay/state"
	"github.com/alis-is/tezpay/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type GeneratePayoutsOptions struct {
	Cycle            int64 `json:"cycle"`
	SkipBalanceCheck bool  `json:"skip_balance_check"`
}

func GeneratePayouts(configurationAndEngines *ConfigurationAndEngines, options GeneratePayoutsOptions) *common.CyclePayoutBlueprint {
	cycle := options.Cycle
	config, collector, signer, _ := configurationAndEngines.Unwrap()
	defer extension.CloseExtensions()

	if cycle <= 0 {
		lastCompletedCycle := assertRunWithResultAndErrFmt(collector.GetLastCompletedCycle, common.EXIT_OPERTION_FAILED, "failed to get last completed cycle")
		cycle = lastCompletedCycle + cycle
	}

	if !config.IsDonatingToTezCapital() {
		log.Warn("With your current configuration you are not going to donate to tez.capital")
	}

	return assertRunWithResultAndErrFmt(func() (*common.CyclePayoutBlueprint, error) {
		return core.GeneratePayouts(config, common.NewGeneratePayoutsEngines(collector, signer, notifyAdminFactory(config)),
			&common.GeneratePayoutsOptions{
				Cycle:            cycle,
				SkipBalanceCheck: options.SkipBalanceCheck,
			})
	}, common.EXIT_OPERTION_FAILED, "failed to generate payouts - %s")
}

var generatePayoutsCmd = &cobra.Command{
	Use:   "generate-payouts",
	Short: "generate payouts",
	Long:  "generates payouts without further processing",
	Run: func(cmd *cobra.Command, args []string) {
		cycle, _ := cmd.Flags().GetInt64(CYCLE_FLAG)
		skipBalanceCheck, _ := cmd.Flags().GetBool(SKIP_BALANCE_CHECK_FLAG)

		configurationAndEngines := assertRunWithResult(loadConfigurationEnginesExtensions, common.EXIT_CONFIGURATION_LOAD_FAILURE)
		generationResult := GeneratePayouts(configurationAndEngines, GeneratePayoutsOptions{
			Cycle:            cycle,
			SkipBalanceCheck: skipBalanceCheck,
		})

		targetFile, _ := cmd.Flags().GetString(TO_FILE_FLAG)
		if targetFile != "" {
			assertRun(func() error {
				return writePayoutBlueprintToFile(targetFile, generationResult)
			}, common.EXIT_PAYOUT_WRITE_FAILURE)
			return
		}

		if state.Global.GetWantsOutputJson() {
			utils.PrintPayoutsAsJson(generationResult.Payouts)
			return
		}

		utils.PrintInvalidPayoutRecipes(generationResult.Payouts, generationResult.Cycle)
		utils.PrintValidPayoutRecipes(generationResult.Payouts, generationResult.Cycle)
	},
}

func init() {
	generatePayoutsCmd.Flags().Int64P(CYCLE_FLAG, "c", 0, "cycle to generate payouts for")
	generatePayoutsCmd.Flags().String(TO_FILE_FLAG, "", "saves generated payouts to specified file")
	generatePayoutsCmd.Flags().Bool(SKIP_BALANCE_CHECK_FLAG, false, "skips payout wallet balance check")
	RootCmd.AddCommand(generatePayoutsCmd)
}
