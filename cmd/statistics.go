package cmd

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/tez-capital/tezpay/common"
	reporter_engines "github.com/tez-capital/tezpay/engines/reporter"
	"github.com/tez-capital/tezpay/state"
	"github.com/tez-capital/tezpay/utils"
)

var statisticsCmd = &cobra.Command{
	Use:   "statistics",
	Short: "prints earning stats",
	Long:  "prints out earning statiscs",
	Run: func(cmd *cobra.Command, args []string) {
		n, _ := cmd.Flags().GetInt(CYCLES_FLAG)
		lastCycle, _ := cmd.Flags().GetInt64(LAST_CYCLE_FLAG)

		config, collector, _, _ := assertRunWithResult(loadConfigurationEnginesExtensions, EXIT_CONFIGURATION_LOAD_FAILURE).Unwrap()
		if lastCycle == 0 {
			lastCycle = assertRunWithResult(collector.GetLastCompletedCycle, EXIT_OPERTION_FAILED)
		}
		fsReporter := reporter_engines.NewFileSystemReporter(config, &common.ReporterEngineOptions{})

		var total common.CyclePayoutSummary
		ok := 0
		collectedCycles := make([]int64, 0, n)
		for i := 0; i < n; i++ {
			cycle := lastCycle - int64(i)
			summary, err := fsReporter.GetExistingCycleSummary(cycle)
			if err != nil {
				slog.Warn("failed to read report", "cycle", cycle, "error", err.Error())
				continue
			}
			total = *total.CombineNumericData(summary)
			collectedCycles = append(collectedCycles, cycle)
			ok++
		}

		firstCycle := lastCycle - int64(n-1)
		header := fmt.Sprintf("Statistics #%d - #%d", firstCycle, lastCycle)
		if firstCycle == lastCycle {
			header = fmt.Sprintf("Statistics #%d", lastCycle)
		}

		if state.Global.GetWantsOutputJson() {
			slog.Info("statistics generated", "result", total, "cycles", collectedCycles, "phase", "result")
			return
		}
		utils.PrintCycleSummary(total, header)
	},
}

func init() {
	statisticsCmd.Flags().Int(CYCLES_FLAG, 10, "number of cycles to collect statistics from")
	statisticsCmd.Flags().Int64(LAST_CYCLE_FLAG, 0, "last cycle to collect statistics from (has priority over --cycles)")
	RootCmd.AddCommand(statisticsCmd)
}
