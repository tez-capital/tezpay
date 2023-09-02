//go:build js && wasm

package notificator_engines

import (
	"encoding/json"
	"errors"
	"fmt"
	"syscall/js"

	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/utils"
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
	if !utils.HasJsFunc(notificatorLoader, funcId) {
		return nil, fmt.Errorf("function %s not found", funcId)
	}
	result := notificatorLoader.Call(funcId, kind, string(configuration))
	if result.Type() != js.TypeObject {
		return nil, fmt.Errorf("%s returned invalid data", funcId)
	}
	return &JsNotificator{
		notificator: result,
	}, nil
}

func ValidateNotificatorConfiguration(kind string, configuration []byte) error {
	funcId := "validateNotificatorConfiguration"
	if !utils.HasJsFunc(notificatorLoader, funcId) {
		return fmt.Errorf("function %s not found", funcId)
	}
	result := notificatorLoader.Call(funcId, kind, string(configuration))
	if result.Type() == js.TypeString {
		return errors.New(result.String())
	}
	return nil
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

	result, err := utils.CallJsFunc(jn.notificator, "send", fmt.Sprintf("Report of cycle #%d", summary.Cycle), additionalDataJson)
	if err != nil {
		return err
	}
	if result.Type() == js.TypeString {
		return errors.New(result.String())
	}
	return nil
}

func (jn *JsNotificator) AdminNotify(msg string) error {
	result, err := utils.CallJsFunc(jn.notificator, "send", string(ADMIN_NOTIFICATION), msg)
	if err != nil {
		return err
	}
	if result.Type() == js.TypeString {
		return errors.New(result.String())
	}
	return nil
}

func (jn *JsNotificator) TestNotify() error {
	result, err := utils.CallJsFunc(jn.notificator, "send", "test notification", "js notificator test")
	if err != nil {
		return err
	}
	if result.Type() == js.TypeString {
		return errors.New(result.String())
	}
	return nil
}
