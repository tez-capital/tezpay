package mock

import (
	"github.com/tez-capital/tezpay/common"
)

type EmptyReporter struct {
}

func (engine *EmptyReporter) GetExistingReports(cycle int64) ([]common.PayoutReport, error) {
	panic("not implemented")
}

func (engine *EmptyReporter) ReportPayouts(payouts []common.PayoutReport) error {
	panic("not implemented")
}
func (engine *EmptyReporter) ReportInvalidPayouts(payouts []common.PayoutReport) error {
	panic("not implemented")
}

func (engine *EmptyReporter) ReportCycleSummary(cycle int64, summary common.CyclePayoutSummary) error {
	panic("not implemented")
}

func (engine *EmptyReporter) GetExistingCycleSummary(cycle int64) (*common.CyclePayoutSummary, error) {
	panic("not implemented")
}
