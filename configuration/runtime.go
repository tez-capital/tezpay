package configuration

import (
	"encoding/json"
	"math"

	tezpay_configuration "github.com/tez-capital/tezpay/configuration/v"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/notifications"
	"github.com/trilitech/tzgo/tezos"
)

type RuntimeDelegatorRequirements struct {
	MinimumBalance                        tezos.Z
	BellowMinimumBalanceRewardDestination enums.ERewardDestination
}

type RuntimeDelegatorOverride struct {
	Recipient                    tezos.Address `json:"recipient,omitempty"`
	Fee                          *float64      `json:"fee,omitempty"`
	MinimumBalance               tezos.Z       `json:"minimum_balance,omitempty"`
	IsBakerPayingTxFee           *bool         `json:"baker_pays_transaction_fee,omitempty"`
	IsBakerPayingAllocationTxFee *bool         `json:"baker_pays_allocation_fee,omitempty"`
	MaximumBalance               *tezos.Z      `json:"maximum_balance,omitempty"`
}

type RuntimeDelegatorsConfiguration struct {
	Requirements RuntimeDelegatorRequirements        `json:"requirements,omitempty"`
	Overrides    map[string]RuntimeDelegatorOverride `json:"overrides,omitempty"`
	Ignore       []tezos.Address                     `json:"ignore,omitempty"`
	Prefilter    []tezos.Address                     `json:"prefilter,omitempty"`
}

type RuntimeNotificatorConfiguration struct {
	Type          notifications.NotificatorKind `json:"type,omitempty"`
	Configuration json.RawMessage               `json:"-"`
	IsValid       bool                          `json:"-"`
	IsAdmin       bool                          `json:"admin"`
}

type RuntimePayoutConfiguration struct {
	WalletMode                 enums.EWalletMode `json:"wallet_mode,omitempty"`
	PayoutMode                 enums.EPayoutMode `json:"payout_mode,omitempty"`
	Fee                        float64           `json:"fee,omitempty"`
	IsPayingTxFee              bool              `json:"baker_pays_transaction_fee,omitempty"`
	IsPayingAllocationTxFee    bool              `json:"baker_pays_allocation_fee,omitempty"`
	MinimumAmount              tezos.Z           `json:"minimum_payout_amount,omitempty"`
	IgnoreEmptyAccounts        bool              `json:"ignore_empty_accounts,omitempty"`
	TxGasLimitBuffer           int64             `json:"transaction_gas_limit_buffer,omitempty"`
	KtTxGasLimitBuffer         int64             `json:"kt_transaction_gas_limit_buffer,omitempty"`
	TxDeserializationGasBuffer int64             `json:"transaction_deserialization_gas_buffer,omitempty"`
	TxFeeBuffer                int64             `json:"transaction_fee_buffer,omitempty"`
	KtTxFeeBuffer              int64             `json:"kt_transaction_fee_buffer,omitempty"`
	MinimumDelayBlocks         int64             `json:"minimum_delay_blocks,omitempty"`
	MaximumDelayBlocks         int64             `json:"maximum_delay_blocks,omitempty"`
	SimulationBatchSize        int               `json:"simulation_batch_size,omitempty"`
}

type RuntimeIncomeRecipients struct {
	Bonds       map[string]float64 `json:"bonds,omitempty"`
	Fees        map[string]float64 `json:"fees,omitempty"`
	DonateFees  float64            `json:"donate_fees,omitempty"`
	DonateBonds float64            `json:"donate_bonds,omitempty"`
	Donations   map[string]float64 `json:"donations,omitempty"`
}

type RuntimeNetworkConfiguration struct {
	RpcPool                []string `json:"rpc_pool,omitempty" comment:"Url to rpc endpoint"`
	TzktUrl                string   `json:"tzkt_url,omitempty" comment:"Url to tzkt endpoint"`
	Explorer               string   `json:"explorer,omitempty" comment:"Url to block explorer"`
	DoNotPaySmartContracts bool     `json:"ignore_kt,omitempty" comment:"if true, smart contracts will not be paid out (used for testing)"`
	IgnoreProtocolChanges  bool     `json:"ignore_protocol_changes,omitempty" comment:"if true, protocol changes will be ignored, otherwise the payout will be stopped if the protocol changes"`
}

type RuntimeConfiguration struct {
	BakerPKH                   tezos.Address
	PayoutConfiguration        RuntimePayoutConfiguration
	Delegators                 RuntimeDelegatorsConfiguration
	IncomeRecipients           RuntimeIncomeRecipients
	Network                    RuntimeNetworkConfiguration
	Overdelegation             tezpay_configuration.OverdelegationConfigurationV0
	NotificationConfigurations []RuntimeNotificatorConfiguration
	Extensions                 []tezpay_configuration.ExtensionConfigurationV0
	SourceBytes                []byte `json:"-"`
	DisableAnalytics           bool   `json:"disable_analytics,omitempty"`
	DisableKillSwitch          bool   `json:"disable_kill_switch,omitempty"`
}

func GetDefaultRuntimeConfiguration() RuntimeConfiguration {
	return RuntimeConfiguration{
		BakerPKH: tezos.InvalidKey.Address(),
		PayoutConfiguration: RuntimePayoutConfiguration{
			WalletMode:                 enums.WALLET_MODE_LOCAL_PRIVATE_KEY,
			PayoutMode:                 enums.PAYOUT_MODE_ACTUAL,
			Fee:                        constants.DEFAULT_BAKER_FEE,
			IsPayingTxFee:              false,
			IsPayingAllocationTxFee:    false,
			MinimumAmount:              FloatAmountToMutez(constants.DEFAULT_PAYOUT_MINIMUM_AMOUNT),
			IgnoreEmptyAccounts:        false,
			TxGasLimitBuffer:           constants.DEFAULT_TX_GAS_LIMIT_BUFFER,
			KtTxGasLimitBuffer:         constants.DEFAULT_KT_TX_GAS_LIMIT_BUFFER,
			TxDeserializationGasBuffer: constants.DEFAULT_TX_DESERIALIZATION_GAS_BUFFER,
			TxFeeBuffer:                constants.DEFAULT_TX_FEE_BUFFER,
			KtTxFeeBuffer:              constants.DEFAULT_KT_TX_FEE_BUFFER,
			MinimumDelayBlocks:         constants.DEFAULT_CYCLE_MONITOR_MINIMUM_DELAY,
			MaximumDelayBlocks:         constants.DEFAULT_CYCLE_MONITOR_MAXIMUM_DELAY,
			SimulationBatchSize:        constants.DEFAULT_SIMULATION_TX_BATCH_SIZE,
		},
		Delegators: RuntimeDelegatorsConfiguration{
			Requirements: RuntimeDelegatorRequirements{
				MinimumBalance:                        FloatAmountToMutez(constants.DEFAULT_DELEGATOR_MINIMUM_BALANCE),
				BellowMinimumBalanceRewardDestination: enums.REWARD_DESTINATION_NONE,
			},
			Overrides: make(map[string]RuntimeDelegatorOverride),
			Ignore:    make([]tezos.Address, 0),
			Prefilter: make([]tezos.Address, 0),
		},
		Network: RuntimeNetworkConfiguration{
			RpcPool:                constants.DEFAULT_RPC_POOL,
			TzktUrl:                constants.DEFAULT_TZKT_URL,
			Explorer:               constants.DEFAULT_EXPLORER_URL,
			DoNotPaySmartContracts: false,
			IgnoreProtocolChanges:  false,
		},
		Overdelegation: tezpay_configuration.OverdelegationConfigurationV0{
			IsProtectionEnabled: true,
		},
		NotificationConfigurations: make([]RuntimeNotificatorConfiguration, 0),
		SourceBytes:                []byte{},
		DisableAnalytics:           false,
	}
}

func (configuration *RuntimeConfiguration) IsDonatingToTezCapital() bool {
	total := float64(0)
	for k, v := range configuration.IncomeRecipients.Donations {
		if constants.DEFAULT_DONATION_ADDRESS == k {
			continue
		}
		total += v
	}
	portion := int64(math.Floor(float64(total) * 10000))
	return portion < 10000 && (configuration.IncomeRecipients.DonateBonds > 0 || configuration.IncomeRecipients.DonateFees > 0)
}
