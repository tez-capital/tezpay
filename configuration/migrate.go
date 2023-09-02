package configuration

import (
	"github.com/alis-is/tezpay/common"
	tezpay_configuration "github.com/alis-is/tezpay/configuration/v"

	"github.com/hjson/hjson-go/v4"
)

type LatestConfigurationType = tezpay_configuration.ConfigurationV0

func migrate(sourceBytes []byte, versionInfo *common.ConfigurationVersionInfo) ([]byte, error) {
	switch versionInfo.TPVersion {
	/* here go future migrations */
	}

	return sourceBytes, nil
}

func Migrate(sourceBytes []byte, versionInfo *common.ConfigurationVersionInfo) (*LatestConfigurationType, error) {
	sourceBytes, err := migrate(sourceBytes, versionInfo)
	if err != nil {
		return nil, err
	}

	// load final config
	configuration := tezpay_configuration.GetDefaultV0()
	err = hjson.Unmarshal(sourceBytes, &configuration)
	if err != nil {
		return nil, err
	}
	configuration.SourceBytes = sourceBytes // inject bytes for processing in future

	return &configuration, nil
}
