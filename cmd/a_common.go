package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alis-is/tezpay/clients"
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/state"
	"github.com/alis-is/tezpay/utils"
	log "github.com/sirupsen/logrus"
)

type configurationAndEngines struct {
	Configuration *configuration.RuntimeConfiguration
	Collector     common.CollectorEngine
	Signer        common.SignerEngine
	Transactor    common.TransactorEngine
}

func (cae *configurationAndEngines) Unwrap() (*configuration.RuntimeConfiguration, common.CollectorEngine, common.SignerEngine, common.TransactorEngine) {
	return cae.Configuration, cae.Collector, cae.Signer, cae.Transactor
}

func loadConfigurationAndEngines() (*configurationAndEngines, error) {
	config, err := configuration.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration - %s", err.Error())
	}

	signerEngine := state.Global.SignerOverride
	if signerEngine == nil {
		signerEngine, err = config.LoadSigner()
		if err != nil {
			return nil, fmt.Errorf("failed to load signer - %s", err.Error())
		}
	}
	// for testing point transactor to testnet
	// transactorEngine, err := clients.InitDefaultTransactor("https://rpc.tzkt.io/ghostnet/", "https://api.ghostnet.tzkt.io/") // (config.Network.RpcUrl, config.Network.TzktUrl)
	transactorEngine, err := clients.InitDefaultTransactor(config.Network.RpcUrl, config.Network.TzktUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to load transactor - %s", err.Error())
	}

	collector, err := clients.InitDefaultRpcAndTzktColletor(config.Network.RpcUrl, config.Network.TzktUrl)
	if err != nil {
		return nil, err
	}

	if utils.IsTty() && state.Global.GetIsInDebugMode() {
		marshaled, _ := json.MarshalIndent(config, "", "\t")
		fmt.Println("Loaded configuration:", string(marshaled))
	}

	return &configurationAndEngines{
		Configuration: config,
		Collector:     collector,
		Signer:        signerEngine,
		Transactor:    transactorEngine,
	}, nil
}

func loadPayoutBlueprintFromFile(fromFile string) (*common.CyclePayoutBlueprint, error) {
	log.Infof("reading payouts from '%s'", fromFile)
	data, err := os.ReadFile(fromFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read payouts from file - %s", err.Error())
	}
	payouts, err := utils.PayoutBlueprintFromJson(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse payouts from file - %s", err.Error())
	}
	return payouts, nil
}

func writePayoutBlueprintToFile(toFile string, blueprint *common.CyclePayoutBlueprint) error {
	log.Infof("writing payouts to '%s'", toFile)
	err := os.WriteFile(toFile, utils.PayoutBlueprintToJson(blueprint), 0644)
	if err != nil {
		return fmt.Errorf("failed to write generated payouts to file - %s", err.Error())
	}
	return nil
}

func printPayoutCycleReport(report *common.PayoutCycleReport) error {
	data, err := json.Marshal(report)
	if err != nil {
		return err
	}
	fmt.Println("REPORT:", string(data))
	return nil
}

type versionInfo struct {
	Version string `json:"tag_name"`
}
