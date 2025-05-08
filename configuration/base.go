package configuration

import (
	"encoding/json"
	"log/slog"
	"math"
	"os"
	"slices"
	"strconv"

	"github.com/trilitech/tzgo/tezos"

	"github.com/hjson/hjson-go/v4"
	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/common"
	tezpay_configuration "github.com/tez-capital/tezpay/configuration/v"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/state"
)

func FloatAmountToMutez(amount float64) tezos.Z {
	mutez := amount * constants.MUTEZ_FACTOR
	return tezos.NewZ(int64(math.Floor(mutez)))
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
		var stakeLimit *tezos.Z = nil
		if delegatorOverride.MaximumBalance != nil {
			sl := FloatAmountToMutez(*delegatorOverride.MaximumBalance)
			stakeLimit = &sl
		}
		return k, RuntimeDelegatorOverride{
			Recipient:                    delegatorOverride.Recipient,
			Fee:                          delegatorOverride.Fee,
			MinimumBalance:               FloatAmountToMutez(delegatorOverride.MinimumBalance),
			IsBakerPayingTxFee:           delegatorOverride.IsBakerPayingTxFee,
			IsBakerPayingAllocationTxFee: delegatorOverride.IsBakerPayingAllocationTxFee,
			MaximumBalance:               stakeLimit,
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
	balanceCheckMode := configuration.PayoutConfiguration.BalanceCheckMode
	if balanceCheckMode == "" {
		balanceCheckMode = enums.PROTOCOL_BALANCE_CHECK_MODE
	}

	gasLimitBuffer := int64(constants.DEFAULT_TX_GAS_LIMIT_BUFFER)
	if configuration.PayoutConfiguration.TxGasLimitBuffer != nil {
		gasLimitBuffer = *configuration.PayoutConfiguration.TxGasLimitBuffer
	}

	ktGasLimitBuffer := int64(constants.DEFAULT_KT_TX_GAS_LIMIT_BUFFER)
	if configuration.PayoutConfiguration.KtTxGasLimitBuffer != nil {
		ktGasLimitBuffer = *configuration.PayoutConfiguration.KtTxGasLimitBuffer
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

	donate := constants.DEFAULT_DONATION_PERCENTAGE
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

	minimumPayoutDelayBlocks := constants.DEFAULT_CYCLE_MONITOR_MINIMUM_DELAY
	if configuration.PayoutConfiguration.MinimumDelayBlocks != nil && *configuration.PayoutConfiguration.MaximumDelayBlocks > 0 {
		minimumPayoutDelayBlocks = *configuration.PayoutConfiguration.MinimumDelayBlocks
	}

	maximumPayoutDelayBlocks := constants.DEFAULT_CYCLE_MONITOR_MAXIMUM_DELAY
	if configuration.PayoutConfiguration.MaximumDelayBlocks != nil && *configuration.PayoutConfiguration.MaximumDelayBlocks > 0 {
		maximumPayoutDelayBlocks = *configuration.PayoutConfiguration.MaximumDelayBlocks
	}

	simulationBatchSize := constants.DEFAULT_SIMULATION_TX_BATCH_SIZE
	if configuration.PayoutConfiguration.SimulationBatchSize != nil && *configuration.PayoutConfiguration.SimulationBatchSize > 0 {
		simulationBatchSize = *configuration.PayoutConfiguration.SimulationBatchSize
	}

	rpcPool := make([]string, 0, len(configuration.Network.RpcPool)+1)
	if configuration.Network.RpcUrl != "" {
		rpcPool = append(rpcPool, configuration.Network.RpcUrl)
	}
	for _, rpc := range configuration.Network.RpcPool {
		if !slices.Contains(rpcPool, rpc) {
			rpcPool = append(rpcPool, rpc)
		}
	}

	return &RuntimeConfiguration{
		BakerPKH: configuration.BakerPKH,
		PayoutConfiguration: RuntimePayoutConfiguration{
			WalletMode:                 walletMode,
			PayoutMode:                 payoutMode,
			BalanceCheckMode:           balanceCheckMode,
			Fee:                        configuration.PayoutConfiguration.Fee,
			IsPayingTxFee:              configuration.PayoutConfiguration.IsPayingTxFee,
			IsPayingAllocationTxFee:    configuration.PayoutConfiguration.IsPayingAllocationTxFee,
			MinimumAmount:              FloatAmountToMutez(configuration.PayoutConfiguration.MinimumAmount),
			IgnoreEmptyAccounts:        configuration.PayoutConfiguration.IgnoreEmptyAccounts,
			TxGasLimitBuffer:           gasLimitBuffer,
			KtTxGasLimitBuffer:         ktGasLimitBuffer,
			TxDeserializationGasBuffer: deserializaGasBuffer,
			TxFeeBuffer:                feeBuffer,
			KtTxFeeBuffer:              ktFeeBuffer,
			MinimumDelayBlocks:         minimumPayoutDelayBlocks,
			MaximumDelayBlocks:         maximumPayoutDelayBlocks,
			SimulationBatchSize:        simulationBatchSize,
		},
		Delegators: RuntimeDelegatorsConfiguration{
			Requirements: RuntimeDelegatorRequirements{
				MinimumBalance:                        FloatAmountToMutez(configuration.Delegators.Requirements.MinimumBalance),
				BellowMinimumBalanceRewardDestination: delegatorBellowMinimumBalanceRewardDestination,
			},
			Overrides: delegatorOverrides,
			Ignore:    configuration.Delegators.Ignore,
			Prefilter: configuration.Delegators.Prefilter,
		},
		IncomeRecipients: RuntimeIncomeRecipients{
			Bonds:       configuration.IncomeRecipients.Bonds,
			Fees:        configuration.IncomeRecipients.Fees,
			Donations:   preprocessDonationMap(configuration.IncomeRecipients.Donations),
			DonateFees:  donateFees,
			DonateBonds: donateBonds,
		},
		Network: RuntimeNetworkConfiguration{
			RpcPool:                rpcPool,
			TzktUrl:                configuration.Network.TzktUrl,
			ProtocolRewardsUrl:     configuration.Network.ProtocolRewardsUrl,
			Explorer:               configuration.Network.Explorer,
			DoNotPaySmartContracts: configuration.Network.DoNotPaySmartContracts,
			IgnoreProtocolChanges:  configuration.Network.IgnoreProtocolChanges,
		},
		Overdelegation: configuration.Overdelegation,
		NotificationConfigurations: lo.Map(configuration.NotificationConfigurations, func(item json.RawMessage, index int) RuntimeNotificatorConfiguration {
			var isValid bool
			var notificatorConfigurationBase tezpay_configuration.NotificatorConfigurationBase
			if err := json.Unmarshal(item, &notificatorConfigurationBase); err != nil {
				slog.Warn("invalid notificator configuration", "error", err.Error())
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
		slog.Debug("loading configuration from file", "path", state.Global.GetConfigurationFilePath())
		// we load configuration from file if it wasnt injected
		var err error
		configurationBytes, err = os.ReadFile(state.Global.GetConfigurationFilePath())
		if err != nil {
			return nil, err
		}
	} else {
		slog.Debug("using injected configuration")
	}

	slog.Debug("loading version info")
	versionInfo := common.ConfigurationVersionInfo{}
	err := hjson.Unmarshal(configurationBytes, &versionInfo)
	if err != nil {
		return nil, err
	}
	slog.Debug("loading configuration")
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
	slog.Debug("loading version info")
	versionInfo := common.ConfigurationVersionInfo{}
	err := hjson.Unmarshal(configurationBytes, &versionInfo)
	if err != nil {
		return nil, err
	}
	slog.Debug("loading configuration")
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
