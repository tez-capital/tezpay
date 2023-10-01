//go:build js && wasm

package reporter_engines

import (
	"encoding/json"
	"syscall/js"

	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/wasm"
)

type JsReporter struct {
	reporter js.Value
}

func NewJsReporter(reporter js.Value) *JsReporter {
	return &JsReporter{
		reporter: reporter,
	}
}

func (engine *JsReporter) GetExistingReports(cycle int64) ([]common.PayoutReport, error) {
	funcId := "getExistingReports"
	result, err := wasm.CallJsFuncExpectResultType(engine.reporter, funcId, js.TypeString, cycle)
	if err != nil {
		return nil, err
	}

	reports := make([]common.PayoutReport, 0)
	err = json.Unmarshal([]byte(result.String()), &reports)
	return reports, err
}

func (engine *JsReporter) ReportPayouts(payouts []common.PayoutReport) error {
	funcId := "reportPayouts"

	data, err := json.Marshal(payouts)
	if err != nil {
		return err
	}

	_, err = wasm.CallJsFunc(engine.reporter, funcId, string(data))
	return err
}

func (engine *JsReporter) ReportInvalidPayouts(reports []common.PayoutRecipe) error {
	funcId := "reportInvalidPayouts"

	data, err := json.Marshal(reports)
	if err != nil {
		return err
	}

	_, err = wasm.CallJsFunc(engine.reporter, funcId, string(data))
	return err
}

func (engine *JsReporter) ReportCycleSummary(summary common.CyclePayoutSummary) error {
	funcId := "reportCycleSummary"

	data, err := json.Marshal(summary)
	if err != nil {
		return err
	}

	_, err = wasm.CallJsFunc(engine.reporter, funcId, string(data))
	return err
}

func (engine *JsReporter) GetExistingCycleSummary(cycle int64) (*common.CyclePayoutSummary, error) {
	funcId := "getExistingCycleSummary"
	result, err := wasm.CallJsFuncExpectResultType(engine.reporter, funcId, js.TypeString, cycle)
	if err != nil {
		return nil, err
	}

	var summary common.CyclePayoutSummary
	err = json.Unmarshal([]byte(result.String()), &summary)
	if err != nil {
		return nil, err
	}
	return &summary, nil
}
