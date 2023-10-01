//go:build js && wasm

package notificator_engines

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/wasm"
)

type NotificatorConfiguration struct {
	Type string   `json:"type"`
	Path string   `json:"path"`
	Args []string `json:"args,omitempty"`
}

type JsNotificator struct {
	notificator js.Value
}

var (
	notificatorLoader js.Value
)

func RegisterNotificatorLoader(loader js.Value) error {
	if loader.Type() != js.TypeObject {
		return fmt.Errorf("invalid loader object")
	}
	return nil
}

func LoadNotificators(kind string, configuration []byte) (common.NotificatorEngine, error) {
	funcId := "loadNotificators"

	result, err := wasm.CallJsFuncExpectResultType(notificatorLoader, funcId, js.TypeObject, kind, string(configuration))
	if err != nil {
		return nil, err
	}

	return &JsNotificator{
		notificator: result,
	}, nil
}

func ValidateNotificatorConfiguration(kind string, configuration []byte) error {
	funcId := "validateNotificatorConfiguration"

	_, err := wasm.CallJsFunc(notificatorLoader, funcId, kind, string(configuration))
	return err
}

func (jn *JsNotificator) PayoutSummaryNotify(summary *common.CyclePayoutSummary, additionalData map[string]string) error {
	var additionalDataJson string
	if additionalData != nil {
		data, err := json.Marshal(additionalData)
		if err != nil {
			return err
		}
		additionalDataJson = string(data)
	}

	_, err := wasm.CallJsFunc(jn.notificator, "send", fmt.Sprintf("Report of cycle #%d", summary.Cycle), additionalDataJson)
	return err
}

func (jn *JsNotificator) AdminNotify(msg string) error {
	_, err := wasm.CallJsFunc(jn.notificator, "send", string(ADMIN_NOTIFICATION), msg)
	return err
}

func (jn *JsNotificator) TestNotify() error {
	_, err := wasm.CallJsFunc(jn.notificator, "send", "test notification", "js notificator test")
	return err
}
