package reports

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/state"
	"github.com/alis-is/tezpay/utils"
	"github.com/gocarina/gocsv"
	"github.com/samber/lo"
)

func mapPayoutRecipeToPayoutReport(pr common.PayoutRecipe, _ int) common.PayoutReport {
	return pr.PayoutRecipeToPayoutReport()
}

func WriteInvalidPayoutRecipesReport(payouts []common.PayoutRecipe) error {
	invalid := utils.OnlyInvalidPayouts(payouts)
	if len(invalid) == 0 {
		return nil
	}
	cyclesToBeWritten := lo.Uniq(lo.Map(invalid, func(pr common.PayoutRecipe, _ int) int64 {
		return pr.Cycle
	}))

	workingDirectory := state.Global.GetWorkingDirectory()

	for _, cycle := range cyclesToBeWritten {
		targetFile := path.Join(workingDirectory, constants.REPORTS_DIRECTORY, fmt.Sprintf("%d", cycle), constants.INVALID_REPORT_FILE_NAME)
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

func WritePayoutsReport(payouts []common.PayoutReport) error {
	cyclesToBeWritten := lo.Uniq(lo.Map(payouts, func(pr common.PayoutReport, _ int) int64 {
		return pr.Cycle
	}))

	workingDirectory := state.Global.GetWorkingDirectory()

	for _, cycle := range cyclesToBeWritten {
		targetFile := path.Join(workingDirectory, constants.REPORTS_DIRECTORY, fmt.Sprintf("%d", cycle), constants.PAYOUT_REPORT_FILE_NAME)
		err := os.MkdirAll(path.Dir(targetFile), 0700)
		if err != nil {
			return err
		}
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

// Returns payouts reports, true if error is
func ReadPayoutReports(cycle int64) ([]common.PayoutReport, error) {
	workingDirectory := state.Global.GetWorkingDirectory()
	sourceFile := path.Join(workingDirectory, constants.REPORTS_DIRECTORY, fmt.Sprintf("%d", cycle), constants.PAYOUT_REPORT_FILE_NAME)
	data, err := os.ReadFile(sourceFile)
	if err != nil {
		return []common.PayoutReport{}, err
	}
	reports := make([]common.PayoutReport, 0)
	err = gocsv.UnmarshalBytes(data, &reports)
	return reports, err
}

func WriteCycleSummary(summary common.CyclePayoutSummary) error {
	workingDirectory := state.Global.GetWorkingDirectory()
	targetFile := path.Join(workingDirectory, constants.REPORTS_DIRECTORY, fmt.Sprintf("%d", summary.Cycle), constants.REPORT_SUMMARY_FILE_NAME)
	data, err := json.MarshalIndent(summary, "", "\t")
	if err != nil {
		return err
	}
	err = os.WriteFile(targetFile, data, 0644)
	return err
}

func ReadCycleSummary(cycle int64) (*common.CyclePayoutSummary, error) {
	workingDirectory := state.Global.GetWorkingDirectory()
	sourceFile := path.Join(workingDirectory, constants.REPORTS_DIRECTORY, fmt.Sprintf("%d", cycle), constants.REPORT_SUMMARY_FILE_NAME)
	data, err := os.ReadFile(sourceFile)
	if err != nil {
		return nil, err
	}
	var summary common.CyclePayoutSummary
	err = json.Unmarshal(data, &summary)
	return &summary, err
}
