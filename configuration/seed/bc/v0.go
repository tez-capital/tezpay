package bc_seed

import (
	"encoding/json"

	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/constants/enums"
)

type IncomeRecipientsV0 struct {
	BondRewards map[string]float64 `json:"bond_rewards"`
	FeeRewards  map[string]float64 `json:"fee_income"`
}

type DelegatorRequirementsV0 struct {
	MinimumBalance float64 `json:"minimum_balance,omitempty"`
}

type DelegatorOverrideV0 struct {
	Recipient string  `json:"recipient,omitempty"`
	Fee       float64 `json:"fee,omitempty"`
}

type TezosNetworkConfigurationV0 struct {
	RpcUrl                 string `json:"rpc_url,omitempty"`
	DoNotPaySmartContracts bool   `json:"suppress_KT_payments,omitempty"`
}

type OverdelegationConfigurationV0 struct {
	ExcludedAddresses   []string `json:"excluded_addresses,omitempty"`
	IsProtectionEnabled bool     `json:"guard,omitempty"`
}

type PaymentRequirementsV0 struct {
	IsPayingTxFee bool    `json:"baker_pays_transaction_fee,omitempty"`
	MinimumAmount float64 `json:"minimum_amount,omitempty"`
}

type ConfigurationV0 struct {
	BakerPKH                   string                         `json:"baking_address"`
	Fee                        float64                        `json:"default_fee,omitempty"`
	WalletMode                 string                         `json:"payout_wallet_mode"`
	DelegatorRequirements      DelegatorRequirementsV0        `json:"delegator_requirements,omitempty"`
	IncomeRecipients           IncomeRecipientsV0             `json:"income_recipients,omitempty"`
	DelegatorOverrides         map[string]DelegatorOverrideV0 `json:"delegator_overrides,omitempty"`
	Network                    TezosNetworkConfigurationV0    `json:"network_configuration,omitempty"`
	Overdelegation             OverdelegationConfigurationV0  `json:"overdelegation,omitempty"`
	PaymentRequirements        PaymentRequirementsV0          `json:"payment_requirements,omitempty"`
	NotificationConfigurations []json.RawMessage              `json:"notifications,omitempty"`
}

func GetDefault() ConfigurationV0 {
	return ConfigurationV0{
		BakerPKH:   "",
		Fee:        constants.DEFAULT_BAKER_FEE,
		WalletMode: string(enums.WALLET_MODE_LOCAL_PRIVATE_KEY),
		DelegatorRequirements: DelegatorRequirementsV0{
			MinimumBalance: constants.DEFAULT_DELEGATOR_MINIMUM_BALANCE,
		},
		DelegatorOverrides: make(map[string]DelegatorOverrideV0),
		Network: TezosNetworkConfigurationV0{
			RpcUrl:                 constants.DEFAULT_RPC_URL,
			DoNotPaySmartContracts: false,
		},
		Overdelegation: OverdelegationConfigurationV0{
			ExcludedAddresses:   make([]string, 0),
			IsProtectionEnabled: true,
		},
		PaymentRequirements: PaymentRequirementsV0{
			IsPayingTxFee: false,
			MinimumAmount: constants.DEFAULT_PAYOUT_MINIMUM_AMOUNT,
		},
		NotificationConfigurations: make([]json.RawMessage, 0),
	}
}
