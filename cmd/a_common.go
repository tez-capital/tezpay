package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"blockwatch.cc/tzgo/tezos"
	"github.com/AlecAivazis/survey/v2"
	"github.com/alis-is/tezpay/clients"
	"github.com/alis-is/tezpay/clients/interfaces"
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/core/reports"
	"github.com/alis-is/tezpay/notifications"
	"github.com/alis-is/tezpay/state"
	"github.com/alis-is/tezpay/utils"
	log "github.com/sirupsen/logrus"
)

type configurationAndEngines struct {
	Configuration *configuration.RuntimeConfiguration
	Collector     interfaces.CollectorEngine
	Signer        interfaces.SignerEngine
	Transactor    interfaces.TransactorEngine
}

func (cae *configurationAndEngines) Unwrap() (*configuration.RuntimeConfiguration, interfaces.CollectorEngine, interfaces.SignerEngine, interfaces.TransactorEngine) {
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
	transactorEngine, err := clients.InitDefaultTransactor(config.Network.RpcUrl)
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

func loadPastPayoutReports(baker tezos.Address, cycle int64) ([]common.PayoutReport, error) {
	reports, err := reports.ReadPayoutReports(cycle)
	if err == nil || os.IsNotExist(err) {
		return utils.FilterReportsByBaker(reports, baker), nil
	}
	return []common.PayoutReport{}, err
}

func requireConfirmation(msg string) error {
	proceed := false
	if utils.IsTty() {
		prompt := &survey.Confirm{
			Message: msg,
		}
		survey.AskOne(prompt, &proceed)
	}
	if !proceed {
		return errors.New("not confirmed")
	}
	return nil
}

func notifyPayoutsProcessed(configuration *configuration.RuntimeConfiguration, summary *common.CyclePayoutSummary, filter string) {
	for _, notificatorConfiguration := range configuration.NotificationConfigurations {
		if filter != "" && notificatorConfiguration.Type != filter {
			continue
		}

		log.Infof("Sending notification with %s", notificatorConfiguration.Type)
		notificator, err := notifications.LoadNotificatior(notificatorConfiguration.Type, notificatorConfiguration.Configuration)
		if err != nil {
			log.Warnf("failed to send notification - %s", err.Error())
			continue
		}

		err = notificator.Notify(summary)
		if err != nil {
			log.Warnf("failed to send notification - %s", err.Error())
			continue
		}
	}
	log.Info("Notifications sent.")
}
func notifyPayoutsProcessedThroughAllNotificators(configuration *configuration.RuntimeConfiguration, summary *common.CyclePayoutSummary) {
	notifyPayoutsProcessed(configuration, summary, "")
}

func printPayoutCycleReport(report *common.PayoutCycleReport) error {
	data, err := json.Marshal(report)
	if err != nil {
		return err
	}
	fmt.Println("REPORT:", string(data))
	return nil
}
