package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/configuration"
	collector_engines "github.com/alis-is/tezpay/engines/collector"
	signer_engines "github.com/alis-is/tezpay/engines/signer"
	transactor_engines "github.com/alis-is/tezpay/engines/transactor"
	"github.com/alis-is/tezpay/extension"
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

func loadConfigurationEnginesExtensions() (*configurationAndEngines, error) {
	config, err := configuration.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration - %s", err.Error())
	}

	signerEngine := state.Global.SignerOverride
	if signerEngine == nil {
		signerEngine, err = signer_engines.Load(string(config.PayoutConfiguration.WalletMode))
		if err != nil {
			return nil, fmt.Errorf("failed to load signer - %s", err.Error())
		}
	}
	// for testing point transactor to testnet
	// transactorEngine, err := clients.InitDefaultTransactor("https://rpc.tzkt.io/ghostnet/", "https://api.ghostnet.tzkt.io/") // (config.Network.RpcUrl, config.Network.TzktUrl)
	transactorEngine, err := transactor_engines.InitDefaultTransactor(config)
	if err != nil {
		return nil, fmt.Errorf("failed to load transactor - %s", err.Error())
	}

	collector, err := collector_engines.InitDefaultRpcAndTzktColletor(config)
	if err != nil {
		return nil, err
	}

	if utils.IsTty() && state.Global.GetIsInDebugMode() {
		marshaled, _ := json.MarshalIndent(config, "", "\t")
		fmt.Println("Loaded configuration:", string(marshaled))
	}

	extEnv := &extension.ExtensionStoreEnviromnent{
		BakerPKH:  config.BakerPKH.String(),
		PayoutPKH: signerEngine.GetPKH().String(),
	}
	if err = extension.InitializeExtensionStore(context.Background(), config.Extensions, extEnv); err != nil {
		return nil, fmt.Errorf("failed to initialize extension store - %s", err.Error())
	}

	return &configurationAndEngines{
		Configuration: config,
		Collector:     collector,
		Signer:        signerEngine,
		Transactor:    transactorEngine,
	}, nil
}

func loadGeneratePayoutsResultFromFile(fromFile string) (*common.CyclePayoutBlueprint, error) {
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

type versionInfo struct {
	Version string `json:"tag_name"`
}

func GetProtocolWithRetry(collector common.CollectorEngine) tezos.ProtocolHash {
	protocol, err := collector.GetCurrentProtocol()
	for err != nil {
		log.Warnf("failed to get protocol - %s", err.Error())
		log.Warnf("retrying in 10 seconds")
		time.Sleep(time.Second * 10)
		protocol, err = collector.GetCurrentProtocol()
	}
	return protocol
}
