package seed

import (
	"encoding/json"
	"fmt"

	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/echa/log"
	"github.com/hjson/hjson-go/v4"
	"gopkg.in/yaml.v3"
)

func getSerializeHjsonOptions() hjson.EncoderOptions {
	encoderOptions := hjson.DefaultOptions()
	encoderOptions.IndentBy = "\t"
	return encoderOptions
}

func Generate(sourceBytes []byte, kind enums.EConfigurationSeedKind) ([]byte, error) {
	var versionInfo common.ConfigurationVersionInfo
	switch kind {
	case enums.BC_CONFIGURATION_SEED:
		err := json.Unmarshal(sourceBytes, &versionInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal version info: %w", err)
		}
		if versionInfo.TPVersion == nil && versionInfo.Version == nil {
			return MigrateBcv0ToTPv0(sourceBytes)
		}

		/*future bc generators*/

		return nil, fmt.Errorf("unsupported bc version: (%v/%v)", versionInfo.TPVersion, versionInfo.Version)
	case enums.TRD_CONFIGURATION_SEED:
		err := yaml.Unmarshal(sourceBytes, &versionInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal version info: %w", err)
		}
		if versionInfo.Version == nil {
			defVer := "1.0"
			log.Warnf("trd version is not defined, assuming version: %v", defVer)
			versionInfo.Version = &defVer
		}
		switch *versionInfo.Version {
		case "1.0":
			return MigrateTrdv1ToTPv0(sourceBytes)

		/* future TRD generators*/

		default:
			return nil, fmt.Errorf("unsupported trd version: %v", versionInfo.Version)
		}

	default:
		return nil, fmt.Errorf("invalid seed kind: %s", kind)
	}
}
