package configuration

import (
	"encoding/json"
	"math"
	"os"
	"strconv"
	"time"

	"blockwatch.cc/tzgo/tezos"

	"github.com/alis-is/tezpay/common"
	tezpay_configuration "github.com/alis-is/tezpay/configuration/v"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/state"
	"github.com/hjson/hjson-go/v4"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

func FloatAmountToMutez(amount float64) tezos.Z {
	mutez := amount * constants.MUTEZ_FACTOR
	return tezos.NewZ(int64(math.Floor(mutez)))
}

func getDefaultDonatePercentageRelativeToDate(currentDate time.Time) float64 {
	startDate := time.Date(2023, time.June, 28, 0, 0, 0, 0, time.UTC)
	donate := float64(0.0)
	daysPassed := int(currentDate.Sub(startDate).Hours() / 24)
	increments := daysPassed / 30

	for i := 0; i < increments; i++ {
		if donate < 0.05 {
			donate += 0.01
		} else {
			break
		}
	}

	return donate
}

func getDefaultDonatePercentage() float64 {
	return getDefaultDonatePercentageRelativeToDate(time.Now())
}

func preprocessDonationMap(donations map[string]float64) map[string]float64 {
	if len(donations) == 0 {
		return map[string]float64{
			constants.DEFAULT_DONATION_ADDRESS: 1,
		}
	}
	total := 0.0
	for _, value := range donations {
		total += value
	}
	if total < 1 {
		donations[constants.DEFAULT_DONATION_ADDRESS] = 1 - total
	}
	return donations
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
	gasLimitBuffer := int64(constants.DEFAULT_TX_GAS_LIMIT_BUFFER)
	if configuration.PayoutConfiguration.TxGasLimitBuffer != nil {
		gasLimitBuffer = *configuration.PayoutConfiguration.TxGasLimitBuffer
	}

	deserializaGasBuffer := int64(constants.DEFAULT_TX_DESERIALIZATION_GAS_BUFFER)
	if configuration.PayoutConfiguration.TxDeserializationGasBuffer != nil {
		deserializaGasBuffer = *configuration.PayoutConfiguration.TxDeserializationGasBuffer
	}

	feeBuffer := int64(constants.DEFAULT_TX_FEE_BUFFER)
	if configuration.PayoutConfiguration.TxFeeBuffer != nil {
		feeBuffer = *configuration.PayoutConfiguration.TxFeeBuffer
	}

	ktFeeBuffer := int64(constants.DEFAULT_KT_TX_FEE_BUFFER)
	if configuration.PayoutConfiguration.KtTxFeeBuffer != nil {
		ktFeeBuffer = *configuration.PayoutConfiguration.KtTxFeeBuffer
	}

	donate := getDefaultDonatePercentage()
	if configuration.IncomeRecipients.Donate != nil {
		donate = *configuration.IncomeRecipients.Donate
	}

	donateBonds := donate
	if configuration.IncomeRecipients.DonateBonds != nil {
		donateBonds = *configuration.IncomeRecipients.DonateBonds
	}

	donateFees := donate
	if configuration.IncomeRecipients.DonateFees != nil {
		donateFees = *configuration.IncomeRecipients.DonateFees
	}

	delegatorBellowMinimumBalanceRewardDestination := enums.REWARD_DESTINATION_NONE
	if configuration.Delegators.Requirements.BellowMinimumBalanceRewardDestination != nil {
		delegatorBellowMinimumBalanceRewardDestination = *configuration.Delegators.Requirements.BellowMinimumBalanceRewardDestination
	}

	return &RuntimeConfiguration{
		BakerPKH: configuration.BakerPKH,
		PayoutConfiguration: RuntimePayoutConfiguration{
			WalletMode:                 walletMode,
			PayoutMode:                 payoutMode,
			Fee:                        configuration.PayoutConfiguration.Fee,
			IsPayingTxFee:              configuration.PayoutConfiguration.IsPayingTxFee,
			IsPayingAllocationTxFee:    configuration.PayoutConfiguration.IsPayingAllocationTxFee,
			MinimumAmount:              FloatAmountToMutez(configuration.PayoutConfiguration.MinimumAmount),
			IgnoreEmptyAccounts:        configuration.PayoutConfiguration.IgnoreEmptyAccounts,
			TxGasLimitBuffer:           gasLimitBuffer,
			TxDeserializationGasBuffer: deserializaGasBuffer,
			TxFeeBuffer:                feeBuffer,
			KtTxFeeBuffer:              ktFeeBuffer,
		},
		Delegators: RuntimeDelegatorsConfiguration{
			Requirements: RuntimeDelegatorRequirements{
				MinimumBalance:                        FloatAmountToMutez(configuration.Delegators.Requirements.MinimumBalance),
				BellowMinimumBalanceRewardDestination: delegatorBellowMinimumBalanceRewardDestination,
			},
			Overrides: delegatorOverrides,
			Ignore:    configuration.Delegators.Ignore,
		},
		IncomeRecipients: RuntimeIncomeRecipients{
			Bonds:       configuration.IncomeRecipients.Bonds,
			Fees:        configuration.IncomeRecipients.Fees,
			Donations:   preprocessDonationMap(configuration.IncomeRecipients.Donations),
			DonateFees:  donateFees,
			DonateBonds: donateBonds,
		},
		Network:        configuration.Network,
		Overdelegation: configuration.Overdelegation,
		NotificationConfigurations: lo.Map(configuration.NotificationConfigurations, func(item json.RawMessage, index int) RuntimeNotificatorConfiguration {
			var isValid bool
			var notificatorConfigurationBase tezpay_configuration.NotificatorConfigurationBase
			if err := json.Unmarshal(item, &notificatorConfigurationBase); err != nil {
				log.Warnf("invalid notificator configuration %v", err)
			}

			return RuntimeNotificatorConfiguration{
				Type:          notificatorConfigurationBase.Type,
				IsAdmin:       notificatorConfigurationBase.Admin,
				Configuration: item,
				IsValid:       isValid,
			}
		}),
		Extensions:       configuration.Extensions,
		SourceBytes:      []byte{},
		DisableAnalytics: configuration.DisableAnalytics,
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
