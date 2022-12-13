//go:build js && wasm

package main

import (
	"encoding/json"
	"errors"
	"syscall/js"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/core/payout"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.Infof("tezpay wasm v%s loaded", constants.VERSION)
}

//export generate_payouts
func generate_payouts(key js.Value, cycle int64, configurationJs js.Value) (js.Value, error) {
	configurationBytes := []byte(configurationJs.String())
	config, err := configuration.LoadFromString(configurationBytes)
	if err != nil {
		return js.Null(), err
	}

	bakerKey, err := tezos.ParseKey(key.String())
	if err != nil {
		return js.Null(), err
	}

	payoutBlueprint, err := payout.GeneratePayoutsWithPayoutAddress(bakerKey, cycle, config, common.GeneratePayoutsOptions{
		SkipBalanceCheck: true,
	})
	if err != nil {
		return js.Null(), err
	}

	result, err := json.Marshal(payoutBlueprint)

	return js.ValueOf(string(result)), err
}

//export test
func test(data js.Value) (js.Value, error) {
	x := data.String()
	log.Info(x)
	return js.ValueOf(x), errors.New("test")
}
