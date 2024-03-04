package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/constants"
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
		return nil, errors.Join(constants.ErrConfigurationLoadFailed, err)
	}

	signerEngine := state.Global.SignerOverride
	if signerEngine == nil {
		signerEngine, err = signer_engines.Load(string(config.PayoutConfiguration.WalletMode))
		if err != nil {
			return nil, errors.Join(constants.ErrSignerLoadFailed, err)
		}
	}
	// for testing point transactor to testnet
	// transactorEngine, err := clients.InitDefaultTransactor("https://rpc.tzkt.io/ghostnet/", "https://api.ghostnet.tzkt.io/") // (config.Network.RpcUrl, config.Network.TzktUrl)
	transactorEngine, err := transactor_engines.InitDefaultTransactor(config)
	if err != nil {
		return nil, errors.Join(constants.ErrTransactorLoadFailed, err)
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
		return nil, errors.Join(constants.ErrExtensionStoreInitializationFailed, err)
	}

	return &configurationAndEngines{
		Configuration: config,
		Collector:     collector,
		Signer:        signerEngine,
		Transactor:    transactorEngine,
	}, nil
}

func loadGeneratedPayoutsResultFromFile(fromFile string) (*common.CyclePayoutBlueprint, error) {
	log.Infof("reading payouts from '%s'", fromFile)
	data, err := os.ReadFile(fromFile)
	if err != nil {
		return nil, errors.Join(constants.ErrPayoutsFromFileLoadFailed, err)
	}
	payouts, err := utils.PayoutBlueprintFromJson(data)
	if err != nil {
		return nil, errors.Join(constants.ErrPayoutsFromFileLoadFailed, err)
	}
	return payouts, nil
}

func writePayoutBlueprintToFile(toFile string, blueprint *common.CyclePayoutBlueprint) error {
	log.Infof("writing payouts to '%s'", toFile)
	err := os.WriteFile(toFile, utils.PayoutBlueprintToJson(blueprint), 0644)
	if err != nil {
		return errors.Join(constants.ErrPayoutsSaveToFileFailed, err)
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
