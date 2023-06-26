package reporter_engines

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"

	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/state"
	"github.com/alis-is/tezpay/utils"
	"github.com/gocarina/gocsv"
	"github.com/samber/lo"
)

type FsReporter struct {
	configuration *configuration.RuntimeConfiguration
}

func NewFileSystemReporter(config *configuration.RuntimeConfiguration) *FsReporter {
	return &FsReporter{
		configuration: config,
	}
}

func (engine *FsReporter) GetExistingReports(cycle int64) ([]common.PayoutReport, error) {
	sourceFile := path.Join(state.Global.GetReportsDirectory(), fmt.Sprintf("%d", cycle), constants.PAYOUT_REPORT_FILE_NAME)
	data, err := os.ReadFile(sourceFile)
	if err != nil {
		return []common.PayoutReport{}, err
	}
	reports := make([]common.PayoutReport, 0)
	err = gocsv.UnmarshalBytes(data, &reports)
	return reports, err
}

func (engine *FsReporter) ReportPayouts(payouts []common.PayoutReport) error {
	if len(payouts) == 0 {
		return nil
	}
	cyclesToBeWritten := lo.Uniq(lo.Map(payouts, func(pr common.PayoutReport, _ int) int64 {
		return pr.Cycle
	}))

	for _, cycle := range cyclesToBeWritten {
		targetFile := path.Join(state.Global.GetReportsDirectory(), fmt.Sprintf("%d", cycle), constants.PAYOUT_REPORT_FILE_NAME)
		err := os.MkdirAll(path.Dir(targetFile), 0700)
		if err != nil {
			return err
		}

		sort.Slice(payouts, func(i, j int) bool {
			return !payouts[i].Amount.IsLess(payouts[j].Amount)
		})

		reports := lo.Filter(payouts, func(payout common.PayoutReport, _ int) bool {
			return payout.Cycle == cycle
		})
		csv, err := gocsv.MarshalBytes(reports)
		if err != nil {
			return err
		}
		err = os.WriteFile(targetFile, csv, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

func mapPayoutRecipeToPayoutReport(pr common.PayoutRecipe, _ int) common.PayoutReport {
	return pr.ToPayoutReport()
}

func (engine *FsReporter) ReportInvalidPayouts(payouts []common.PayoutRecipe) error {
	invalid := utils.OnlyInvalidPayouts(payouts)
	if len(invalid) == 0 {
		return nil
	}
	cyclesToBeWritten := lo.Uniq(lo.Map(invalid, func(pr common.PayoutRecipe, _ int) int64 {
		return pr.Cycle
	}))

	for _, cycle := range cyclesToBeWritten {
		targetFile := path.Join(state.Global.GetReportsDirectory(), fmt.Sprintf("%d", cycle), constants.INVALID_REPORT_FILE_NAME)
		err := os.MkdirAll(path.Dir(targetFile), 0700)
		if err != nil {
			return err
		}
		reports := lo.Map(utils.FilterPayoutsByCycle(invalid, cycle), mapPayoutRecipeToPayoutReport)
		csv, err := gocsv.MarshalBytes(reports)
		if err != nil {
			return err
		}
		err = os.WriteFile(targetFile, csv, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

func (engine *FsReporter) ReportCycleSummary(summary common.CyclePayoutSummary) error {
	targetFile := path.Join(state.Global.GetReportsDirectory(), fmt.Sprintf("%d", summary.Cycle), constants.REPORT_SUMMARY_FILE_NAME)
	data, err := json.MarshalIndent(summary, "", "\t")
	if err != nil {
		return err
	}
	err = os.WriteFile(targetFile, data, 0644)
	return err
}

func (engine *FsReporter) GetExistingCycleSummary(cycle int64) (*common.CyclePayoutSummary, error) {
	sourceFile := path.Join(state.Global.GetReportsDirectory(), fmt.Sprintf("%d", cycle), constants.REPORT_SUMMARY_FILE_NAME)
	data, err := os.ReadFile(sourceFile)
	if err != nil {
		return nil, err
	}
	var summary common.CyclePayoutSummary
	err = json.Unmarshal(data, &summary)
	return &summary, err
}
