//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/core"
	collector_engines "github.com/alis-is/tezpay/engines/collector"
	notificator_engines "github.com/alis-is/tezpay/engines/notificator"
	reporter_engines "github.com/alis-is/tezpay/engines/reporter"
	signer_engines "github.com/alis-is/tezpay/engines/signer"
	transactor_engines "github.com/alis-is/tezpay/engines/transactor"
	"github.com/alis-is/tezpay/state"
	"github.com/alis-is/tezpay/wasm"
	log "github.com/sirupsen/logrus"
)

var (
	jobChannel = make(chan wasm.Job)
)

func main() {
	tezpay := js.Global().Get("Object").New()
	js.Global().Set("tezpay", tezpay)

	tezpay.Set("terminate", js.FuncOf(terminate))
	tezpay.Set("generatePayouts", js.FuncOf(initGeneratePayouts))
	tezpay.Set("executePayouts", js.FuncOf(initExecutePayouts))
	log.SetLevel(log.TraceLevel)
	state.InitWASMState(state.StateInitOptions{})
	log.Infof("tezpay wasm v%s loaded", constants.VERSION)
	for job := range jobChannel {
		if job.Id == "terminate" {
			break
		}

		switch job.Id {
		case "generate_payouts":
			job.ResultChannel <- generatePayouts(job.This, job.Args)
		case "execute_payouts":
			job.ResultChannel <- executePayouts(job.This, job.Args)
		}
	}

	tezpay.Set("exited", js.ValueOf(true))
}

func terminate(this js.Value, args []js.Value) interface{} {
	jobChannel <- wasm.NewJob("terminate", this, args)
	return nil
}

func initGeneratePayouts(this js.Value, args []js.Value) interface{} {
	job := wasm.NewJob("generate_payouts", this, args)
	jobChannel <- job
	return job.GetPromise()
}

func initExecutePayouts(this js.Value, args []js.Value) interface{} {
	job := wasm.NewJob("execute_payouts", this, args)
	jobChannel <- job
	return job.GetPromise()
}

type ConfigurationAndEngines struct {
	Configuration *configuration.RuntimeConfiguration
	Collector     common.CollectorEngine
	Signer        common.SignerEngine
	Transactor    common.TransactorEngine
}

func (cae *ConfigurationAndEngines) Unwrap() (*configuration.RuntimeConfiguration, common.CollectorEngine, common.SignerEngine, common.TransactorEngine) {
	return cae.Configuration, cae.Collector, cae.Signer, cae.Transactor
}

func loadConfigurationAndEngines(this js.Value, args []js.Value) (*ConfigurationAndEngines, *wasm.WasmExecutionResult) {
	log.Trace("loading configuration and engines")
	if len(args) < 2 {
		log.Error("invalid number of arguments (expects configurationAndEngines and generatePayoutsOptions)")
		return nil, wasm.NewErrorResult(wasm.ErrorInvalidArguments)
	}

	jsConfigurationAndEngines := args[0]
	jsConfiguration := jsConfigurationAndEngines.Get("configuration")
	if jsConfiguration.Type() != js.TypeString {
		log.Error("invalid configuration")
		return nil, wasm.NewErrorResult(wasm.ErrorInvalidConfiguration)
	}
	runtimeConfiguration, err := configuration.LoadFromString([]byte(jsConfiguration.String()))
	if err != nil {
		log.Errorf("failed to load configuration - %s", err.Error())
		return nil, wasm.NewErrorResult(wasm.ErrorFailedToLoadConfiguration)
	}

	jsCollectorEngine, err := collector_engines.InitJsColletor(jsConfigurationAndEngines.Get("collectorEngine"))
	if err != nil {
		log.Errorf("failed to initialize collector engine - %s", err.Error())
		return nil, wasm.NewErrorResult(wasm.ErrorFailedToInitiaiizeCollector)
	}

	jsSignerEngine, err := signer_engines.InitJsSigner(jsConfigurationAndEngines.Get("signerEngine"))
	if err != nil {
		log.Errorf("failed to initialize signer engine - %s", err.Error())
		return nil, wasm.NewErrorResult(wasm.ErrorFailedToInitiaiizeSigner)
	}

	jsTransactorEngine, err := transactor_engines.InitJsTransactor(jsConfigurationAndEngines.Get("transactorEngine"))
	if err != nil {
		log.Errorf("failed to transactor signer engine - %s", err.Error())
		return nil, wasm.NewErrorResult(wasm.ErrorFailedToInitiaiizeTransactor)
	}

	notificatorLoader := jsConfigurationAndEngines.Get("notificatorLoader")
	err = notificator_engines.RegisterNotificatorLoader(notificatorLoader)
	if err != nil {
		log.Errorf("failed to initialize signer engine - %s", err.Error())
		return nil, wasm.NewErrorResult(wasm.ErrorFailedToInitiaiizeNotificatorLoader)
	}
	log.Trace("configuration and engines loaded")
	return &ConfigurationAndEngines{
		Configuration: runtimeConfiguration,
		Collector:     jsCollectorEngine,
		Signer:        jsSignerEngine,
		Transactor:    jsTransactorEngine,
	}, nil
}

func generatePayouts(this js.Value, args []js.Value) (result *wasm.WasmExecutionResult) {
	// defer recover PanicStatus
	defer func() {
		if r := recover(); r != nil {
			if panicStatus, ok := r.(common.PanicStatus); ok {
				result = wasm.NewErrorResult(panicStatus.Error)
				return
			} else {
				result = wasm.NewErrorResult(wasm.ErrorUnhandledPanic)
				return
			}
		}
	}()

	configurationAndEngines, execResult := loadConfigurationAndEngines(this, args)
	if execResult != nil {
		return execResult
	}

	runtimeConfiguration, jsCollectorEngine, jsSignerEngine, _ := configurationAndEngines.Unwrap()

	jsGeneratePayoutsOptions := args[1]
	if jsGeneratePayoutsOptions.Type() != js.TypeString {
		log.Error("invalid payout options")
		return wasm.NewErrorResult(wasm.ErrorInvalidPayoutOptions)
	}

	var options common.GeneratePayoutsOptions
	err := json.Unmarshal([]byte(jsGeneratePayoutsOptions.String()), &options)
	if err != nil {
		log.Errorf("failed to unmarshal payout options - %s", err.Error())
		return wasm.NewErrorResult(wasm.ErrorFailedToUnmarshalPayoutOptions)
	}

	blueprint, err := core.GeneratePayouts(runtimeConfiguration, common.NewGeneratePayoutsEngines(jsCollectorEngine, jsSignerEngine, func(string) {}),
		&options)
	if err != nil {
		log.Errorf("failed to generate payouts - %s", err.Error())
		return wasm.NewErrorResult(fmt.Errorf("failed to generate payouts - %s", err.Error()))
	}

	blueprintJson, err := json.Marshal(blueprint)
	if err != nil {
		log.Errorf("failed to marshal blueprint - %s", err.Error())
		return wasm.NewErrorResult(fmt.Errorf("failed to marshal blueprint - %s", err.Error()))
	}

	return wasm.NewSuccessResult(string(blueprintJson))
}

func executePayouts(this js.Value, args []js.Value) (result *wasm.WasmExecutionResult) {
	// defer recover PanicStatus
	defer func() {
		if r := recover(); r != nil {
			if panicStatus, ok := r.(common.PanicStatus); ok {
				result = wasm.NewErrorResult(panicStatus.Error)
				return
			} else {
				fmt.Println(r)
				result = wasm.NewErrorResult(wasm.ErrorUnhandledPanic)
				return
			}
		}
	}()

	configurationAndEngines, execResult := loadConfigurationAndEngines(this, args)
	if execResult != nil {
		return execResult
	}
	runtimeConfiguration, _, jsSignerEngine, jsTransactorEngine := configurationAndEngines.Unwrap()

	log.Trace("loading execute payouts options")
	jsExecutePayoutsOptions := args[1]
	if jsExecutePayoutsOptions.Type() != js.TypeString {
		log.Error("invalid payout options")
		return wasm.NewErrorResult(wasm.ErrorInvalidPayoutOptions)
	}
	options := common.ExecutePayoutsOptions{
		MixInContractCalls: false,
		MixInFATransfers:   false,
	}
	err := json.Unmarshal([]byte(jsExecutePayoutsOptions.String()), &options)
	if err != nil {
		log.Errorf("failed to unmarshal payout options - %s", err.Error())
		return wasm.NewErrorResult(wasm.ErrorFailedToUnmarshalPayoutOptions)
	}

	log.Trace("loading preparation result")
	jsPreparationResult := args[2]
	if jsExecutePayoutsOptions.Type() != js.TypeString {
		log.Error("invalid payout options")
		return wasm.NewErrorResult(wasm.ErrorInvalidPayoutOptions)
	}
	var preparationResult common.PreparePayoutsResult
	err = json.Unmarshal([]byte(jsPreparationResult.String()), &preparationResult)
	if err != nil {
		log.Errorf("failed to unmarshal preparation result - %s", err.Error())
		return wasm.NewErrorResult(wasm.ErrorFailedToUnmarshalPayoutOptions)
	}
	log.Trace("succesfully loadded all parameters")
	payoutResult, err := core.ExecutePayouts(&preparationResult, runtimeConfiguration, common.NewExecutePayoutsEngineContext(jsSignerEngine, jsTransactorEngine, reporter_engines.NewDummyReporter(), func(string) {}),
		&options)

	if err != nil {
		log.Errorf("failed to execute payouts - %s", err.Error())
		return wasm.NewErrorResult(fmt.Errorf("failed to execute payouts - %s", err.Error()))
	}

	payoutResultJson, err := json.Marshal(payoutResult)
	if err != nil {
		log.Errorf("failed to marshal payouts result - %s", err.Error())
		return wasm.NewErrorResult(fmt.Errorf("failed to marshal payouts result - %s", err.Error()))
	}

	return wasm.NewSuccessResult(string(payoutResultJson))
}
