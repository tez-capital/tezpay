package tezpay_configuration

import (
	"encoding/json"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/notifications"
)

type IncomeRecipientsV0 struct {
	Bonds       map[string]float64 `json:"bonds,omitempty" comment:"list of addresses and their share of the bonds"`
	Fees        map[string]float64 `json:"fees,omitempty" comment:"list of addresses and their share of the fees"`
	Donate      *float64           `json:"donate,omitempty" comment:"share of the rewards to donate"`
	DonateFees  *float64           `json:"donate_fees,omitempty" comment:"share of the fees to donate (if not set, 'donate' is used)"`
	DonateBonds *float64           `json:"donate_bonds,omitempty" comment:"share of the bonds to donate (if not set, 'donate' is used)"`
	Donations   map[string]float64 `json:"donations,omitempty" comment:"list of addresses and their share of the donations"`
}

type DelegatorRequirementsV0 struct {
	MinimumBalance                        float64                   `json:"minimum_balance,omitempty" comment:"Minimum balance of tez a delegator has to have to be considered for payout"`
	BellowMinimumBalanceRewardDestination *enums.ERewardDestination `json:"below_minimum_reward_destination,omitempty" comment:"Reward destination for delegators with balance below the minimum balance (possible values: 'none', 'everyone')"`
}

type DelegatorOverrideV0 struct {
	Recipient                    tezos.Address `json:"recipient,omitempty" comment:"Redirects payout to the recipient 'address'"`
	Fee                          *float64      `json:"fee,omitempty" comment:"Overrides the fee for the delegator"`
	MinimumBalance               float64       `json:"minimum_balance,omitempty" comment:"Overrides the minimum balance requirement for the delegator"`
	IsBakerPayingTxFee           *bool         `json:"baker_pays_transaction_fee,omitempty" comment:"Overrides the baker paying the transaction fee"`
	IsBakerPayingAllocationTxFee *bool         `json:"baker_pays_allocation_fee,omitempty" comment:"Overrides the baker paying the allocation transaction fee"`
	MaximumBalance               *float64      `json:"maximum_balance,omitempty" comment:"The maximum balance for the delegator (for overdelegation situation you can limit how much of a delegator balance is taken into account)"`
}

type DelegatorsConfigurationV0 struct {
	Requirements DelegatorRequirementsV0        `json:"requirements,omitempty" comment:"Requirements delegators have to meet"`
	Ignore       []tezos.Address                `json:"ignore,omitempty" comment:"List of delegator addresses to ignore"`
	Overrides    map[string]DelegatorOverrideV0 `json:"overrides,omitempty" comment:"Overrides for specific delegators"`
	FeeOverrides map[string][]tezos.Address     `json:"fee_overrides,omitempty" comment:"Shortcuts for overriding fees for specific delegators"`
}

type TezosNetworkConfigurationV0 struct {
	// RpcUrl represents the URL to the RPC node.
	RpcUrl                 string `json:"rpc_url,omitempty" comment:"Url to rpc endpoint"`
	TzktUrl                string `json:"tzkt_url,omitempty" comment:"Url to tzkt endpoint"`
	Explorer               string `json:"explorer,omitempty" comment:"Url to block explorer"`
	DoNotPaySmartContracts bool   `json:"ignore_kt,omitempty" comment:"if true, smart contracts will not be paid out (used for testing)"`
	IgnoreProtocolChanges  bool   `json:"ignore_protocol_changes,omitempty" comment:"if true, protocol changes will be ignored, otherwise the payout will be stopped if the protocol changes"`
}

type OverdelegationConfigurationV0 struct {
	IsProtectionEnabled bool `json:"protect,omitempty"`
}

type PayoutConfigurationV0 struct {
	WalletMode                 enums.EWalletMode `json:"wallet_mode" comment:"wallet mode to use for signing transactions, can be 'local-private-key' or 'remote-signer'"`
	PayoutMode                 enums.EPayoutMode `json:"payout_mode" comment:"payout mode to use, can be 'actual' or 'ideal'"`
	Fee                        float64           `json:"fee,omitempty" comment:"fee to charge delegators for the payout (portion of the reward as decimal, e.g. 0.075 for 7.5%)" validate:"required,min=0,max=1"`
	IsPayingTxFee              bool              `json:"baker_pays_transaction_fee,omitempty" comment:"if true, baker pays the transaction fee"`
	IsPayingAllocationTxFee    bool              `json:"baker_pays_allocation_fee,omitempty" comment:"if true, baker pays the allocation transaction fee"`
	MinimumAmount              float64           `json:"minimum_payout_amount,omitempty" comment:"minimum amount to pay out to delegators, if the amount is less, the payout will be ignored"`
	IgnoreEmptyAccounts        bool              `json:"ignore_empty_accounts,omitempty" comment:"if true, empty accounts will be ignored"`
	TxGasLimitBuffer           *int64            `json:"transaction_gas_limit_buffer,omitempty" comment:"buffer for transaction gas limit"`
	TxDeserializationGasBuffer *int64            `json:"transaction_deserialization_gas_buffer,omitempty" comment:"buffer for transaction deserialization gas"`
	TxFeeBuffer                *int64            `json:"transaction_fee_buffer,omitempty" comment:"buffer for transaction fee"`
	KtTxFeeBuffer              *int64            `json:"kt_transaction_fee_buffer,omitempty" comment:"buffer for KT transaction fee"`
	MinimumDelayBlocks         *int64            `json:"minimum_delay_blocks,omitempty" comment:"minimum delay in blocks before the payout is executed"`
	MaximumDelayBlocks         *int64            `json:"maximum_delay_blocks,omitempty" comment:"maximum delay in blocks before the payout is executed"`
	SimulationBatchSize        *int              `json:"simulation_batch_size,omitempty" comment:"size of the batch for simulation (number of transactions, higher usually means faster simulation but in case of failure, more transactions will be lost and need to be simulated again)"`
}

type ExtensionConfigurationV0 = common.ExtensionDefinition

type ConfigurationV0 struct {
	Version                    uint                          `json:"tezpay_config_version" comment:"version of the configuration file"`
	BakerPKH                   tezos.Address                 `json:"baker" comment:"baker's public key hash"`
	PayoutConfiguration        PayoutConfigurationV0         `json:"payouts" comment:"payout configuration"`
	Delegators                 DelegatorsConfigurationV0     `json:"delegators,omitempty" comment:"delegators configuration"`
	IncomeRecipients           IncomeRecipientsV0            `json:"income_recipients,omitempty" comment:"income recipients configuration"`
	Network                    TezosNetworkConfigurationV0   `json:"network,omitempty" comment:"tezos network configuration"`
	Overdelegation             OverdelegationConfigurationV0 `json:"overdelegation,omitempty" comment:"overdelegation protection configuration"`
	NotificationConfigurations []json.RawMessage             `json:"notifications,omitempty" comment:"notification configurations"`
	Extensions                 []ExtensionConfigurationV0    `json:"extensions,omitempty" comment:"extensions (for custom functionality)"`
	SourceBytes                []byte                        `json:"-"`
	DisableAnalytics           bool                          `json:"disable_analytics,omitempty" comment:"disables analytics, please consider leaving it enabled🙏"`
}

type NotificatorConfigurationBase struct {
	Type  notifications.NotificatorKind `json:"type" comment:"type of the notificator"`
	Admin bool                          `json:"admin" comment:"if true, the notificator is used for admin notifications"`
}

func GetDefaultV0() ConfigurationV0 {
	gasLimitBuffer := int64(constants.DEFAULT_TX_GAS_LIMIT_BUFFER)
	deserializaGasBuffer := int64(constants.DEFAULT_TX_DESERIALIZATION_GAS_BUFFER)
	minimumPayoutDelayBlocks := constants.DEFAULT_CYCLE_MONITOR_MINIMUM_DELAY
	maximumPayoutDelayBlocks := constants.DEFAULT_CYCLE_MONITOR_MAXIMUM_DELAY
	simulationBatchSize := constants.DEFAULT_SIMULATION_TX_BATCH_SIZE

	delegatorBellowMinimumBalanceRewardDestination := enums.REWARD_DESTINATION_NONE

	return ConfigurationV0{
		Version:  0,
		BakerPKH: tezos.InvalidKey.Address(),
		Delegators: DelegatorsConfigurationV0{
			Requirements: DelegatorRequirementsV0{
				MinimumBalance:                        constants.DEFAULT_DELEGATOR_MINIMUM_BALANCE,
				BellowMinimumBalanceRewardDestination: &delegatorBellowMinimumBalanceRewardDestination,
			},
			Overrides: make(map[string]DelegatorOverrideV0),
			Ignore:    make([]tezos.Address, 0),
		},
		Network: TezosNetworkConfigurationV0{
			RpcUrl:                 constants.DEFAULT_RPC_URL,
			TzktUrl:                constants.DEFAULT_TZKT_URL,
			Explorer:               constants.DEFAULT_EXPLORER_URL,
			DoNotPaySmartContracts: false,
			IgnoreProtocolChanges:  false,
		},
		Overdelegation: OverdelegationConfigurationV0{
			IsProtectionEnabled: true,
		},
		PayoutConfiguration: PayoutConfigurationV0{
			WalletMode:                 enums.WALLET_MODE_LOCAL_PRIVATE_KEY,
			PayoutMode:                 enums.PAYOUT_MODE_ACTUAL,
			Fee:                        constants.DEFAULT_BAKER_FEE,
			IsPayingTxFee:              false,
			IsPayingAllocationTxFee:    false,
			MinimumAmount:              constants.DEFAULT_PAYOUT_MINIMUM_AMOUNT,
			TxGasLimitBuffer:           &gasLimitBuffer,
			TxDeserializationGasBuffer: &deserializaGasBuffer,
			MinimumDelayBlocks:         &minimumPayoutDelayBlocks,
			MaximumDelayBlocks:         &maximumPayoutDelayBlocks,
			SimulationBatchSize:        &simulationBatchSize,
		},
		IncomeRecipients:           IncomeRecipientsV0{},
		NotificationConfigurations: make([]json.RawMessage, 0),
		SourceBytes:                []byte{},
		DisableAnalytics:           false,
	}
}
