//go:build !wasm

package configuration

import (
	"bytes"
	"fmt"
	"os"

	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/state"
	"github.com/hjson/hjson-go/v4"
)

func WriteMigratedConfiguration(source string, configuration *LatestConfigurationType) error {
	var marshaled []byte
	var err error
	encoderOptions := hjson.DefaultOptions()
	encoderOptions.IndentBy = "\t"
	marshaled, err = hjson.MarshalWithOptions(configuration, encoderOptions)
	if err != nil {
		return err
	}
	err = os.WriteFile(source, marshaled, 0644)
	return err
}

func MigrateAndPersist(sourceBytes []byte, versionInfo *common.ConfigurationVersionInfo) (*LatestConfigurationType, error) {
	originalSourceBytes := sourceBytes

	configuration, err := Migrate(sourceBytes, versionInfo)
	if err != nil {
		return nil, err
	}

	// persist migrated config
	isMigrated := !bytes.Equal(originalSourceBytes, configuration.SourceBytes)
	if isMigrated {
		source := state.Global.GetConfigurationFilePath()
		os.Rename(source, source+constants.CONFIG_FILE_BACKUP_SUFFIX)
		err := WriteMigratedConfiguration(source, configuration)
		if err != nil {
			return nil, fmt.Errorf("failed to write migrated configuration - %s", err.Error())
		}
	}

	return configuration, nil
}
