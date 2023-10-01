//go:build !wasm

package configuration

import (
	"os"

	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/state"
	"github.com/hjson/hjson-go/v4"
	log "github.com/sirupsen/logrus"
)

func Load() (*RuntimeConfiguration, error) {
	hasInjectedConfiguration, configurationBytes := state.Global.GetInjectedConfiguration()
	if !hasInjectedConfiguration {
		log.Debugf("loading configuration from '%s'", state.Global.GetConfigurationFilePath())
		// we load configuration from file if it wasnt injected
		var err error
		configurationBytes, err = os.ReadFile(state.Global.GetConfigurationFilePath())
		if err != nil {
			return nil, err
		}
	} else {
		log.Debug("using injected configuration")
	}

	log.Debug("loading version info")
	versionInfo := common.ConfigurationVersionInfo{}
	err := hjson.Unmarshal(configurationBytes, &versionInfo)
	if err != nil {
		return nil, err
	}

	migrateFn := MigrateAndPersist
	if hasInjectedConfiguration {
		migrateFn = Migrate
	}
	configuration, err := migrateFn(configurationBytes, &versionInfo)
	if err != nil {
		return nil, err
	}
	runtime, err := ConfigurationToRuntimeConfiguration(configuration)
	if err != nil {
		return nil, err
	}
	err = runtime.Validate()
	return runtime, err
}
