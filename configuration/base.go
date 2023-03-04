package configuration

import (
	"encoding/json"
	"math"
	"os"
	"strconv"

	"blockwatch.cc/tzgo/tezos"

	"github.com/alis-is/tezpay/common"
	tezpay_configuration "github.com/alis-is/tezpay/configuration/v"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
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
		return k, RuntimeDelegatorOverride{
			Recipient:                    delegatorOverride.Recipient,
			Fee:                          delegatorOverride.Fee,
			MinimumBalance:               FloatAmountToMutez(delegatorOverride.MinimumBalance),
			IsBakerPayingTxFee:           delegatorOverride.IsBakerPayingTxFee,
			IsBakerPayingAllocationTxFee: delegatorOverride.IsBakerPayingAllocationTxFee,
		}
	})
	for k, v := range delegatorFeeOverrides {
		fee := v
		if delegatorOverride, ok := delegatorOverrides[k]; ok {
			if delegatorOverride.Fee == nil {
				delegatorOverride.Fee = &fee
			}
			continue
		}
		delegatorOverrides[k] = RuntimeDelegatorOverride{
			Fee: &fee,
		}
	}

	walletMode := configuration.PayoutConfiguration.WalletMode
	if walletMode == "" {
		walletMode = enums.WALLET_MODE_LOCAL_PRIVATE_KEY
	}
	payoutMode := configuration.PayoutConfiguration.PayoutMode
	if payoutMode == "" {
		payoutMode = enums.PAYOUT_MODE_ACTUAL
	}

	return &RuntimeConfiguration{
		BakerPKH: configuration.BakerPKH,
		PayoutConfiguration: RuntimePayoutConfiguration{
			WalletMode:              walletMode,
			PayoutMode:              payoutMode,
			Fee:                     configuration.PayoutConfiguration.Fee,
			IsPayingTxFee:           configuration.PayoutConfiguration.IsPayingTxFee,
			IsPayingAllocationTxFee: configuration.PayoutConfiguration.IsPayingAllocationTxFee,
			MinimumAmount:           FloatAmountToMutez(configuration.PayoutConfiguration.MinimumAmount),
			IgnoreEmptyAccounts:     configuration.PayoutConfiguration.IgnoreEmptyAccounts,
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
			isAdmin := false
			if admin, ok := item["admin"].(bool); ok {
				isAdmin = admin
			}

			configuration, _ := json.Marshal(item)

			return RuntimeNotificatorConfiguration{
				Type:          notifications.NotificatorKind(notificatorType),
				IsAdmin:       isAdmin,
				Configuration: configuration,
				Options:       item,
				IsValid:       isValid,
			}
		}),
		Extensions:  configuration.Extensions,
		SourceBytes: []byte{},
	}, nil
}

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

func LoadFromString(configurationBytes []byte) (*RuntimeConfiguration, error) {
	log.Debug("loading version info")
	versionInfo := common.ConfigurationVersionInfo{}
	err := hjson.Unmarshal(configurationBytes, &versionInfo)
	if err != nil {
		return nil, err
	}

	log.Trace("migrating if required")
	configuration, err := Migrate(configurationBytes, &versionInfo, false)
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
