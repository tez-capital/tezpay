package reporter_engines

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"

	"github.com/gocarina/gocsv"
	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/configuration"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/state"
	"github.com/tez-capital/tezpay/utils"
)

type FsReporter struct {
	configuration *configuration.RuntimeConfiguration
	options       *common.ReporterEngineOptions
}

func NewFileSystemReporter(config *configuration.RuntimeConfiguration, options *common.ReporterEngineOptions) *FsReporter {
	return &FsReporter{
		configuration: config,
		options:       options,
	}
}

func (engine *FsReporter) getReportsDirectory() (string, error) {
	var directory string
	if engine.options.DryRun {
		directory = path.Join(state.Global.GetReportsDirectory(), "dry")
	} else {
		directory = state.Global.GetReportsDirectory()
	}
	return directory, os.MkdirAll(directory, 0700)
}

func (engine *FsReporter) GetExistingReports(cycle int64) ([]common.PayoutReport, error) {
	reportsDirectory, err := engine.getReportsDirectory()
	if err != nil {
		return []common.PayoutReport{}, err
	}
	sourceFile := path.Join(reportsDirectory, fmt.Sprintf("%d", cycle), constants.PAYOUT_REPORT_FILE_NAME)
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

	reportsDirectory, err := engine.getReportsDirectory()
	if err != nil {
		return err
	}
	for _, cycle := range cyclesToBeWritten {
		targetFile := path.Join(reportsDirectory, fmt.Sprintf("%d", cycle), constants.PAYOUT_REPORT_FILE_NAME)
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

	reportsDirectory, err := engine.getReportsDirectory()
	if err != nil {
		return err
	}
	for _, cycle := range cyclesToBeWritten {
		targetFile := path.Join(reportsDirectory, fmt.Sprintf("%d", cycle), constants.INVALID_REPORT_FILE_NAME)
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
	reportsDirectory, err := engine.getReportsDirectory()
	if err != nil {
		return err
	}
	targetFile := path.Join(reportsDirectory, fmt.Sprintf("%d", summary.Cycle), constants.REPORT_SUMMARY_FILE_NAME)
	data, err := json.MarshalIndent(summary, "", "\t")
	if err != nil {
		return err
	}
	err = os.WriteFile(targetFile, data, 0644)
	return err
}

func (engine *FsReporter) GetExistingCycleSummary(cycle int64) (*common.CyclePayoutSummary, error) {
	reportsDirectory, err := engine.getReportsDirectory()
	if err != nil {
		return nil, err
	}
	sourceFile := path.Join(reportsDirectory, fmt.Sprintf("%d", cycle), constants.REPORT_SUMMARY_FILE_NAME)
	data, err := os.ReadFile(sourceFile)
	if err != nil {
		return nil, err
	}
	var summary common.CyclePayoutSummary
	err = json.Unmarshal(data, &summary)
	return &summary, err
}
