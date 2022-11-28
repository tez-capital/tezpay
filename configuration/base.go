package configuration

import (
	"encoding/json"
	"math"
	"os"
	"path"
	"strconv"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/configuration/migrations"
	tezpay_configuration "github.com/alis-is/tezpay/configuration/tezpay"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/notifications"
	"github.com/alis-is/tezpay/state"
	"github.com/hjson/hjson-go/v4"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

func FloatAmountToMutez(amount float64) tezos.Z {
	mutez := amount * constants.MUTEZ_FACTOR
	return tezos.NewZ(int64(math.Floor(mutez)))
}

func ConfigurationToRuntimeConfiguration(configuration *LatestConfigurationType) (*RuntimeConfiguration, error) {
	delegatorFeeOverrides := make(map[string]float64)
	for k, addresses := range configuration.Delegators.FeeOverrides {
		for _, a := range addresses {
			fee, err := strconv.ParseFloat(k, 64)
			if err != nil {
				return nil, err
			}
			delegatorFeeOverrides[a.String()] = fee
		}
	}

	delegatorOverrides := lo.MapEntries(configuration.Delegators.Overrides, func(k string, delegatorOverride tezpay_configuration.DelegatorOverrideV0) (string, RuntimeDelegatorOverride) {
		actualFee := delegatorOverride.Fee
		noFee := delegatorOverride.NoFee
		if feeFromFeeOverrides, ok := delegatorFeeOverrides[k]; actualFee == 0 && ok {
			actualFee = feeFromFeeOverrides
			noFee = feeFromFeeOverrides == 0
		}
		return k, RuntimeDelegatorOverride{
			Recipient:      delegatorOverride.Recipient,
			Fee:            actualFee,
			NoFee:          noFee,
			MinimumBalance: FloatAmountToMutez(delegatorOverride.MinimumBalance),
		}
	})
	for k, v := range delegatorFeeOverrides {
		if _, ok := delegatorOverrides[k]; !ok {
			delegatorOverrides[k] = RuntimeDelegatorOverride{
				Fee:   v,
				NoFee: v == 0,
			}
		}
	}

	return &RuntimeConfiguration{
		BakerPKH: configuration.BakerPKH,
		PayoutConfiguration: RuntimePayoutConfiguration{
			WalletMode:          configuration.PayoutConfiguration.WalletMode,
			Fee:                 configuration.PayoutConfiguration.Fee,
			IsPayingTxFee:       configuration.PayoutConfiguration.IsPayingTxFee,
			MinimumAmount:       FloatAmountToMutez(configuration.PayoutConfiguration.MinimumAmount),
			IgnoreEmptyAccounts: configuration.PayoutConfiguration.IgnoreEmptyAccounts,
		},
		Delegators: RuntimeDelegatorsConfiguration{
			Requirements: RuntimeDelegatorRequirements{
				MinimumBalance: FloatAmountToMutez(configuration.Delegators.Requirements.MinimumBalance),
			},
			Overrides: delegatorOverrides,
			Ignore:    configuration.Delegators.Ignore,
		},
		IncomeRecipients: configuration.IncomeRecipients,
		Network:          configuration.Network,
		Overdelegation:   configuration.Overdelegation,
		NotificationConfigurations: lo.Map(configuration.NotificationConfigurations, func(item map[string]interface{}, index int) RuntimeNotificatorConfiguration {
			var isValid bool
			var notificatorType string
			if notificatorType, isValid = item["type"].(string); !isValid {
				log.Warnf("invalid notificator type %v", item["type"])
			}

			configuration, _ := json.Marshal(item)

			return RuntimeNotificatorConfiguration{
				Type:          notifications.NotificatorKind(notificatorType),
				Configuration: configuration,
				Options:       item,
				IsValid:       isValid,
			}
		}),
		SourceBytes: []byte{},
	}, nil
}

func Load() (*RuntimeConfiguration, error) {
	workingDirectory := state.Global.GetWorkingDirectory()
	hasInjectedConfiguration, configurationBytes := state.Global.GetInjectedConfiguration()
	if !hasInjectedConfiguration {
		log.Debugf("loading configuration from '%s'", constants.CONFIG_FILE_NAME)
		// we load configuration from file if it wasnt injected
		var err error
		configurationBytes, err = os.ReadFile(path.Join(workingDirectory, "config.hjson"))
		if err != nil {
			return nil, err
		}
	} else {
		log.Debug("using injected configuration")
	}

	log.Debug("loading version info")
	versionInfo := migrations.ConfigurationVersionInfo{}
	err := hjson.Unmarshal(configurationBytes, &versionInfo)
	if err != nil {
		return nil, err
	}

	log.Trace("migrating if required")
	configuration, err := Migrate(configurationBytes, &versionInfo, !hasInjectedConfiguration)
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
