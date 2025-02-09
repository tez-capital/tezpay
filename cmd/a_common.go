package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/configuration"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/constants/enums"
	collector_engines "github.com/tez-capital/tezpay/engines/collector"
	signer_engines "github.com/tez-capital/tezpay/engines/signer"
	transactor_engines "github.com/tez-capital/tezpay/engines/transactor"
	"github.com/tez-capital/tezpay/extension"
	"github.com/tez-capital/tezpay/state"
	"github.com/tez-capital/tezpay/utils"
	"github.com/trilitech/tzgo/tezos"
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

	if utils.IsTty() {
		slog.Debug("loaded configuration", "configuration", config)
	}

	extEnv := &extension.ExtensionStoreEnviromnent{
		BakerPKH:  config.BakerPKH.String(),
		PayoutPKH: signerEngine.GetPKH().String(),
		RpcPool:   config.Network.RpcPool,
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

func loadGeneratedPayoutsFromBytes(data []byte) (*common.CyclePayoutBlueprint, error) {
	payouts, err := utils.PayoutBlueprintFromJson(data)
	if err != nil {
		return nil, errors.Join(constants.ErrPayoutsFromBytesLoadFailed, err)
	}
	return payouts, nil
}

func loadGeneratedPayoutsFromStdin() (*common.CyclePayoutBlueprint, error) {
	slog.Info("reading payouts from stdin")
	scanner := bufio.NewScanner(os.Stdin) // by default reads line by line
	if !scanner.Scan() {
		return nil, errors.Join(constants.ErrPayoutsFromStdinLoadFailed, errors.New("no data available"))
	}
	return loadGeneratedPayoutsFromBytes(scanner.Bytes())
}

func loadGeneratedPayoutsFromFile(fromFile string) (*common.CyclePayoutBlueprint, error) {
	slog.Info("reading payouts from file", "path", fromFile)
	data, err := os.ReadFile(fromFile)
	if err != nil {
		return nil, errors.Join(constants.ErrPayoutsFromFileLoadFailed, err)
	}
	return loadGeneratedPayoutsFromBytes(data)
}

func writePayoutBlueprintToFile(toFile string, blueprint *common.CyclePayoutBlueprint) error {
	slog.Info("writing payouts to file", "path", toFile)
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
		slog.Warn("failed to get protocol", "error", err.Error())
		slog.Info("retrying in 10 seconds")
		time.Sleep(time.Second * 10)
		protocol, err = collector.GetCurrentProtocol()
	}
	return protocol
}

func PrintPreparationResults(preparationResult *common.PreparePayoutsResult, cyclesForTitle ...int64) {
	title := utils.FormatCycleNumbers(cyclesForTitle...)

	utils.PrintPayouts(preparationResult.InvalidPayouts, fmt.Sprintf("Invalid - %s", title), false)
	utils.PrintPayouts(preparationResult.AccumulatedPayouts, fmt.Sprintf("Accumulated - %s", title), false)
	utils.PrintReports(preparationResult.ReportsOfPastSuccesfulPayouts, fmt.Sprintf("Already Successfull - %s", title), true)
	utils.PrintPayouts(preparationResult.ValidPayouts, fmt.Sprintf("Valid - %s", title), true)
}

func PrintPayoutWalletRemainingBalance(collector common.CollectorEngine, signer common.SignerEngine) {
	addr := signer.GetPKH()
	balance, err := collector.GetBalance(addr)
	if err != nil {
		slog.Error("failed to get balance", "error", err.Error())
		return
	}

	slog.Info("the payout wallet remaining balance", "wallet", addr.String(), "balance", common.FormatAmount(enums.PAYOUT_TX_KIND_TEZ, balance.Int64(), ""), "phase", "payout_wallet_remaining_balance")
}
