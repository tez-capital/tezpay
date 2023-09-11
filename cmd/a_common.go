package cmd

import (
	"time"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/configuration"
	log "github.com/sirupsen/logrus"
)

type ConfigurationAndEngines struct {
	Configuration *configuration.RuntimeConfiguration
	Collector     common.CollectorEngine
	Signer        common.SignerEngine
	Transactor    common.TransactorEngine
}

func (cae *ConfigurationAndEngines) Unwrap() (*configuration.RuntimeConfiguration, common.CollectorEngine, common.SignerEngine, common.TransactorEngine) {
	return cae.Configuration, cae.Collector, cae.Signer, cae.Transactor
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
