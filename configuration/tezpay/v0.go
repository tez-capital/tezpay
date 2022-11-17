package tezpay_configuration

import (
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
)

type IncomeRecipientsV0 struct {
	Bonds     map[string]float64 `json:"bonds,omitempty"`
	Fees      map[string]float64 `json:"fees,omitempty"`
	Donate    float64            `json:"donate,omitempty"`
	Donations map[string]float64 `json:"donations,omitempty"`
}

type DelegatorRequirementsV0 struct {
	MinimumBalance float64 `json:"minimum_balance,omitempty"`
}

type DelegatorOverrideV0 struct {
	Recipient      tezos.Address `json:"recipient,omitempty"`
	Fee            float64       `json:"fee,omitempty"`
	NoFee          bool          `json:"no_fee,omitempty"`
	MinimumBalance float64       `json:"minimum_balance,omitempty"`
}

type DelegatorsConfigurationV0 struct {
	Requirements DelegatorRequirementsV0        `json:"requirements,omitempty"`
	Ignore       []tezos.Address                `json:"ignore,omitempty"`
	Overrides    map[string]DelegatorOverrideV0 `json:"overrides,omitempty"`
}

type TezosNetworkConfigurationV0 struct {
	RpcUrl                 string `json:"rpc_url,omitempty"`
	TzktUrl                string `json:"tzkt_url,omitempty"`
	Explorer               string `json:"explorer,omitempty"`
	DoNotPaySmartContracts bool   `json:"ignore_kt,omitempty"`
}

type OverdelegationConfigurationV0 struct {
	IsProtectionEnabled bool `json:"protect,omitempty"`
}

type PayoutConfigurationV0 struct {
	WalletMode              enums.WalletMode `json:"wallet_mode"`
	Fee                     float64          `json:"fee,omitempty"`
	IsPayingTxFee           bool             `json:"baker_pays_transaction_fee,omitempty"`
	IsPayingAllocationTxFee bool             `json:"baker_pays_allocation_fee,omitempty"`
	MinimumAmount           float64          `json:"minimum_payout_amount,omitempty"`
	IgnoreEmptyAccounts     bool             `json:"ignore_empty_accounts,omitempty"`
}

type NotificatorConfigurationV0 struct {
	Type string `json:"type"`
}

type ConfigurationV0 struct {
	Version                    uint                          `json:"tezpay_config_version"`
	BakerPKH                   tezos.Address                 `json:"baker"`
	PayoutConfiguration        PayoutConfigurationV0         `json:"payouts"`
	Delegators                 DelegatorsConfigurationV0     `json:"delegators,omitempty"`
	IncomeRecipients           IncomeRecipientsV0            `json:"income_recipients,omitempty"`
	Network                    TezosNetworkConfigurationV0   `json:"network,omitempty"`
	Overdelegation             OverdelegationConfigurationV0 `json:"overdelegation,omitempty"`
	NotificationConfigurations []map[string]interface{}      `json:"notifications,omitempty"`
	SourceBytes                []byte                        `json:"-"`
}

func GetDefaultV0() ConfigurationV0 {
	return ConfigurationV0{
		Version:  0,
		BakerPKH: tezos.InvalidKey.Address(),
		Delegators: DelegatorsConfigurationV0{
			Requirements: DelegatorRequirementsV0{
				MinimumBalance: constants.DEFAULT_DELEGATOR_MINIMUM_BALANCE,
			},
			Overrides: make(map[string]DelegatorOverrideV0),
			Ignore:    make([]tezos.Address, 0),
		},
		Network: TezosNetworkConfigurationV0{
			RpcUrl:                 constants.DEFAULT_RPC_URL,
			TzktUrl:                constants.DEFAULT_TZKT_URL,
			Explorer:               constants.DEFAULT_EXPLORER_URL,
			DoNotPaySmartContracts: false,
		},
		Overdelegation: OverdelegationConfigurationV0{
			IsProtectionEnabled: true,
		},
		PayoutConfiguration: PayoutConfigurationV0{
			WalletMode:              enums.WALLET_MODE_LOCAL_PRIVATE_KEY,
			Fee:                     constants.DEFAULT_BAKER_FEE,
			IsPayingTxFee:           false,
			IsPayingAllocationTxFee: false,
			MinimumAmount:           constants.DEFAULT_PAYOUT_MINIMUM_AMOUNT,
		},
		NotificationConfigurations: make([]map[string]interface{}, 0),
		SourceBytes:                []byte{},
	}
}
