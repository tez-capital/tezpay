package reporter_engines

import (
	"encoding/json"
	"log/slog"
	"sort"

	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/configuration"
)

type StdioReporter struct {
	configuration *configuration.RuntimeConfiguration
}

func NewStdioReporter(config *configuration.RuntimeConfiguration) *StdioReporter {
	return &StdioReporter{
		configuration: config,
	}
}

func (engine *StdioReporter) GetExistingReports(cycle int64) ([]common.PayoutReport, error) {
	return []common.PayoutReport{}, nil
}

func (engine *StdioReporter) ReportPayouts(payouts []common.PayoutReport) error {
	sort.Slice(payouts, func(i, j int) bool {
		return !payouts[i].Amount.IsLess(payouts[j].Amount)
	})

	slog.Info("REPORT", "payouts", payouts)
	return nil
}

type InvalidPayoutsReport struct {
	InvalidPayouts []common.PayoutRecipe `json:"invalid_payouts"`
}

func (engine *StdioReporter) ReportInvalidPayouts(payouts []common.PayoutReport) error {
	for _, inv := range payouts {
		if len(inv.Accumulated) > 0 {
			panic("invalid payout report contains accumulated reports")
		}
	}
	slog.Info("REPORT", "invalid_payouts", payouts)
	return nil
}

type CycleSummaryReport struct {
	Cycle        int64                     `json:"cycle"`
	CycleSummary common.CyclePayoutSummary `json:"cycle_summary"`
}

func (engine *StdioReporter) ReportCycleSummary(cycle int64, summary common.CyclePayoutSummary) error {
	data, err := json.Marshal(CycleSummaryReport{Cycle: cycle, CycleSummary: summary})
	if err != nil {
		return err
	}

	slog.Info("REPORT", "cycle_summary", string(data))
	return nil
}

func (engine *StdioReporter) GetExistingCycleSummary(cycle int64) (*common.CyclePayoutSummary, error) {
	return &common.CyclePayoutSummary{}, nil
}
