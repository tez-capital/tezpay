//go:build js && wasm

package reporter_engines

import (
	"errors"

	"github.com/alis-is/tezpay/common"
)

type DummyReporter struct {
}

func NewDummyReporter() *DummyReporter {
	return &DummyReporter{}
}

func (engine *DummyReporter) GetExistingReports(cycle int64) ([]common.PayoutReport, error) {
	return []common.PayoutReport{}, nil
}

func (engine *DummyReporter) ReportPayouts(payouts []common.PayoutReport) error {
	return nil
}

func (engine *DummyReporter) ReportInvalidPayouts(reports []common.PayoutRecipe) error {
	return nil
}

func (engine *DummyReporter) ReportCycleSummary(summary common.CyclePayoutSummary) error {
	return nil
}

func (engine *DummyReporter) GetExistingCycleSummary(cycle int64) (*common.CyclePayoutSummary, error) {
	return nil, errors.New("not found")
}
