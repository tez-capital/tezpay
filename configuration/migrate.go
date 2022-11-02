package configuration

import (
	"bytes"
	"fmt"
	"os"

	tezpay_configuration "github.com/alis-is/tezpay/configuration/tezpay"

	"github.com/alis-is/tezpay/configuration/migrations"
	"github.com/alis-is/tezpay/constants"
	"github.com/hjson/hjson-go/v4"
)

type LatestConfigurationType = tezpay_configuration.ConfigurationV0

func WriteMigratedConfiguration(configuration LatestConfigurationType) error {
	encoderOptions := hjson.DefaultOptions()
	encoderOptions.IndentBy = "\t"
	marshaled, err := hjson.MarshalWithOptions(configuration, encoderOptions)
	if err != nil {
		return err
	}
	err = os.WriteFile(constants.CONFIG_FILE_NAME, marshaled, 0644)
	return err
}

func Migrate(sourceBytes []byte, versionInfo *migrations.ConfigurationVersionInfo, persist bool) (*LatestConfigurationType, error) {
	originalSourceBytes := sourceBytes

	// migrations by version in order
	if versionInfo.Version == nil {
		var err error
		sourceBytes, _ /*new versionInfo*/, err = migrations.MigrateBcToTPv0(sourceBytes)
		if err != nil {
			return nil, err
		}
	}

	/* here goes future migrations */

	// load final config
	configuration := tezpay_configuration.GetDefaultV0()
	err := hjson.Unmarshal(sourceBytes, &configuration)
	if err != nil {
		return nil, err
	}
	configuration.SourceBytes = sourceBytes // inject bytes for processing in future

	// persist migrated config
	isMigrated := !bytes.Equal(originalSourceBytes, sourceBytes)
	if isMigrated && persist {
		os.Rename(constants.CONFIG_FILE_NAME, constants.BACKUP_CONFIG_FILE_NAME)
		err := WriteMigratedConfiguration(configuration)
		if err != nil {
			return nil, fmt.Errorf("failed to write migrated configuration - %s", err.Error())
		}
	}

	return &configuration, nil
}
