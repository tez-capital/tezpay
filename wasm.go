//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/alis-is/tezpay/cmd"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/core"
	collector_engines "github.com/alis-is/tezpay/engines/collector"
	notificator_engines "github.com/alis-is/tezpay/engines/notificator"
	signer_engines "github.com/alis-is/tezpay/engines/signer"
	log "github.com/sirupsen/logrus"
)

var (
	terminationChannel = make(chan struct{})
)

type WasmExecutionResult struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func (result WasmExecutionResult) ToJsValue() js.Value {
	jsResult := js.Global().Get("Object").New()
	jsResult.Set("success", js.ValueOf(result.Success))
	if result.Success {
		jsResult.Set("data", js.ValueOf(result.Data))
	} else {
		jsResult.Set("error", js.ValueOf(result.Error))
	}
	return jsResult
}

func main() {
	tezpay := js.Global().Get("Object").New()
	js.Global().Set("tezpay", tezpay)

	tezpay.Set("terminate", js.FuncOf(terminate))
	tezpay.Set("generatePayouts", js.FuncOf(generate_payouts))

	log.Infof("tezpay wasm v%s loaded", constants.VERSION)
	<-terminationChannel
	tezpay.Set("exited", js.ValueOf(true))
}

func terminate(this js.Value, args []js.Value) interface{} {
	terminationChannel <- struct{}{}
	return nil
}

func generate_payouts(this js.Value, args []js.Value) (result interface{}) {
	if len(args) < 2 {
		log.Error("invalid number of arguments (expects payout wallet, payout amount, and payout recipients)")
		return nil
	}

	// defer recover PanicStatus
	defer func() {
		if r := recover(); r != nil {
			if panicStatus, ok := r.(cmd.PanicStatus); ok {
				result = WasmExecutionResult{
					Success: false,
					Error:   panicStatus.Error.Error(),
				}.ToJsValue()
				return
			} else {
				log.Fatal("Unhandled panic")
			}
		}
	}()

	jsConfigurationAndEngines := args[0]
	jsConfiguration := jsConfigurationAndEngines.Get("configuration")
	if jsConfiguration.Type() != js.TypeString {
		log.Error("invalid configuration")
		return nil
	}
	runtimeConfiguration, err := configuration.LoadFromString([]byte(jsConfiguration.String()))
	if err != nil {
		log.Errorf("failed to load configuration - %s", err.Error())
		return nil
	}

	jsCollectorEngine, err := collector_engines.InitJsColletor(jsConfigurationAndEngines.Get("collectorEngine"))
	if err != nil {
		log.Errorf("failed to initialize collector engine - %s", err.Error())
		return nil
	}
	jsSignerEngine, err := signer_engines.InitJsSigner(jsConfigurationAndEngines.Get("signerEngine"))
	if err != nil {
		log.Errorf("failed to initialize signer engine - %s", err.Error())
		return nil
	}
	notificatorLoader := jsConfigurationAndEngines.Get("notificatorLoader")
	err = notificator_engines.RegisterNotificatorLoader(notificatorLoader)
	if err != nil {
		log.Errorf("failed to initialize signer engine - %s", err.Error())
		return nil
	}
	// jsTransactorEngine, err := transactor_engines.InitJsTransactor(jsConfigurationAndEngines.Get("transactorEngine"))
	// if err != nil {
	// 	log.Errorf("failed to transactor signer engine - %s", err.Error())
	// 	return nil
	// }

	jsGeneratePayoutsOptions := args[1]
	if jsGeneratePayoutsOptions.Type() != js.TypeString {
		log.Error("invalid payout options")
		return nil
	}
	var options common.GeneratePayoutsOptions
	err = json.Unmarshal([]byte(jsGeneratePayoutsOptions.String()), &options)

	if err != nil {
		log.Errorf("failed to unmarshal payout options - %s", err.Error())
		return nil
	}

	blueprint, err := core.GeneratePayouts(runtimeConfiguration, common.NewGeneratePayoutsEngines(jsCollectorEngine, jsSignerEngine, func(string) {}),
		&options)
	return WasmExecutionResult{
		Success: true,
		Data:    blueprint,
	}.ToJsValue()
}
