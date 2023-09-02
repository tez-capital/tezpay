//go:build js && wasm

package reporter_engines

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/utils"
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
	if !utils.HasJsFunc(engine.reporter, funcId) {
		return nil, fmt.Errorf("function %s not found", funcId)
	}

	result := engine.reporter.Call(funcId, cycle)
	if result.Type() != js.TypeString {
		return nil, fmt.Errorf("%s returned invalid data", funcId)
	}
	reports := make([]common.PayoutReport, 0)
	err := json.Unmarshal([]byte(result.String()), &reports)
	return reports, err
}

func (engine *JsReporter) ReportPayouts(payouts []common.PayoutReport) error {
	funcId := "reportPayouts"
	if !utils.HasJsFunc(engine.reporter, funcId) {
		return fmt.Errorf("function %s not found", funcId)
	}

	data, err := json.Marshal(payouts)
	if err != nil {
		return err
	}

	engine.reporter.Call(funcId, string(data))
	return nil
}

func (engine *JsReporter) ReportInvalidPayouts(reports []common.PayoutRecipe) error {
	funcId := "reportInvalidPayouts"
	if !utils.HasJsFunc(engine.reporter, funcId) {
		return fmt.Errorf("function %s not found", funcId)
	}

	data, err := json.Marshal(reports)
	if err != nil {
		return err
	}

	engine.reporter.Call(funcId, string(data))
	return nil
}

func (engine *JsReporter) ReportCycleSummary(summary common.CyclePayoutSummary) error {
	funcId := "reportCycleSummary"
	if !utils.HasJsFunc(engine.reporter, funcId) {
		return fmt.Errorf("function %s not found", funcId)
	}

	data, err := json.Marshal(summary)
	if err != nil {
		return err
	}

	engine.reporter.Call(funcId, string(data))
	return nil
}

func (engine *JsReporter) GetExistingCycleSummary(cycle int64) (*common.CyclePayoutSummary, error) {
	funcId := "getExistingCycleSummary"
	if !utils.HasJsFunc(engine.reporter, funcId) {
		return nil, fmt.Errorf("function %s not found", funcId)
	}

	result := engine.reporter.Call(funcId, cycle)
	if result.Type() != js.TypeString {
		return nil, fmt.Errorf("%s returned invalid data", funcId)
	}

	var summary common.CyclePayoutSummary
	err := json.Unmarshal([]byte(result.String()), &summary)
	if err != nil {
		return nil, err
	}
	return &summary, nil
}
