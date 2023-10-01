//go:build !wasm

package reporter_engines

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/configuration"
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

type PayoutsReport struct {
	Payouts []common.PayoutReport `json:"payouts"`
}

func (engine *StdioReporter) ReportPayouts(payouts []common.PayoutReport) error {
	sort.Slice(payouts, func(i, j int) bool {
		return !payouts[i].Amount.IsLess(payouts[j].Amount)
	})
	data, err := json.Marshal(PayoutsReport{Payouts: payouts})
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}

type InvalidPayoutsReport struct {
	InvalidPayouts []common.PayoutRecipe `json:"invalid_payouts"`
}

func (engine *StdioReporter) ReportInvalidPayouts(reports []common.PayoutRecipe) error {
	data, err := json.Marshal(InvalidPayoutsReport{InvalidPayouts: reports})
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}

type CycleSummaryReport struct {
	CycleSummary common.CyclePayoutSummary `json:"cycle_summary"`
}

func (engine *StdioReporter) ReportCycleSummary(summary common.CyclePayoutSummary) error {
	data, err := json.Marshal(CycleSummaryReport{CycleSummary: summary})
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}

func (engine *StdioReporter) GetExistingCycleSummary(cycle int64) (*common.CyclePayoutSummary, error) {
	return &common.CyclePayoutSummary{}, nil
}
