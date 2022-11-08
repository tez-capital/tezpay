package cmd

import (
	"fmt"

	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/core/reports"
	"github.com/alis-is/tezpay/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var statisticsCmd = &cobra.Command{
	Use:   "statistics",
	Short: "prints earning stats",
	Long:  "prints out earning statiscs",
	Run: func(cmd *cobra.Command, args []string) {
		n, _ := cmd.Flags().GetInt(CYCLES_FLAG)
		lastCycle, _ := cmd.Flags().GetInt64(LAST_CYCLE_FLAG)

		if lastCycle == 0 {
			_, collector, _, _ := assertRunWithResult(loadConfigurationAndEngines, EXIT_CONFIGURATION_LOAD_FAILURE).Unwrap()
			lastCycle = assertRunWithResult(collector.GetLastCompletedCycle, EXIT_OPERTION_FAILED)
		}

		var total common.CyclePayoutSummary
		ok := 0
		for i := 0; i < n; i++ {
			cycle := lastCycle - int64(i)
			summary, err := reports.ReadCycleSummary(cycle)
			if err != nil {
				log.Warnf("failed to read report of #%d, skipping...", cycle)
				continue
			}
			total = *total.Add(summary)
			ok++
		}

		firstCycle := lastCycle - int64(n-1)
		header := fmt.Sprintf("Statistics #%d - #%d", firstCycle, lastCycle)
		if firstCycle == lastCycle {
			header = fmt.Sprintf("Statistics #%d", lastCycle)
		}
		utils.PrintCycleSummary(total, header)
	},
}

func init() {
	statisticsCmd.Flags().Int(CYCLES_FLAG, 10, "number of cycles to collect statistics from")
	statisticsCmd.Flags().Int64(LAST_CYCLE_FLAG, 0, "last cycle to collect statistics from (has priority over --cycles)")
	RootCmd.AddCommand(statisticsCmd)
}
