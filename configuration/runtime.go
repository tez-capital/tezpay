package configuration

import (
	"blockwatch.cc/tzgo/tezos"
	tezpay_configuration "github.com/alis-is/tezpay/configuration/tezpay"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/core/signer"
)

type RuntimeDelegatorRequirements struct {
	MinimumBalance tezos.Z
}

type RuntimeDelegatorOverride struct {
	Recipient      tezos.Address `json:"recipient,omitempty"`
	Fee            float32       `json:"fee,omitempty"`
	NoFee          bool          `json:"no_fee,omitempty"`
	MinimumBalance tezos.Z       `json:"minimum_balance,omitempty"`
}

type RuntimeDelegatorsConfiguration struct {
	Requirements RuntimeDelegatorRequirements        `json:"requirements,omitempty"`
	Overrides    map[string]RuntimeDelegatorOverride `json:"overrides,omitempty"`
	Ignore       []tezos.Address                     `json:"ignore,omitempty"`
}

type RuntimeNotificatorConfiguration struct {
	Type          string                 `json:"type,omitempty"`
	Configuration []byte                 `json:"-"`
	Options       map[string]interface{} `json:"configuration,omitempty"`
	IsValid       bool                   `json:"-"`
}

type RuntimePayoutConfiguration struct {
	WalletMode              enums.WalletMode `json:"wallet_mode,omitempty"`
	Fee                     float32          `json:"fee,omitempty"`
	IsPayingTxFee           bool             `json:"baker_pays_transaction_fee,omitempty"`
	IsPayingAllocationTxFee bool             `json:"baker_pays_allocation_fee,omitempty"`
	MinimumAmount           tezos.Z          `json:"minimum_payout_amount,omitempty"`
	IgnoreEmptyAccounts     bool             `json:"ignore_empty_accounts,omitempty"`
}

type RuntimeConfiguration struct {
	BakerPKH                   tezos.Address
	PayoutConfiguration        RuntimePayoutConfiguration
	Delegators                 RuntimeDelegatorsConfiguration
	IncomeRecipients           tezpay_configuration.IncomeRecipientsV0
	Network                    tezpay_configuration.TezosNetworkConfigurationV0
	Overdelegation             tezpay_configuration.OverdelegationConfigurationV0
	NotificationConfigurations []RuntimeNotificatorConfiguration
	SourceBytes                []byte `json:"-"`
}

func GetDefaultRuntimeConfiguration() RuntimeConfiguration {
	return RuntimeConfiguration{
		BakerPKH: tezos.InvalidKey.Address(),
		PayoutConfiguration: RuntimePayoutConfiguration{
			WalletMode:              enums.WALLET_MODE_LOCAL_PRIVATE_KEY,
			Fee:                     constants.DEFAULT_BAKER_FEE,
			IsPayingTxFee:           false,
			IsPayingAllocationTxFee: false,
			MinimumAmount:           FloatAmountToMutez(constants.DEFAULT_PAYOUT_MINIMUM_AMOUNT),
			IgnoreEmptyAccounts:     false,
		},
		Delegators: RuntimeDelegatorsConfiguration{
			Requirements: RuntimeDelegatorRequirements{
				MinimumBalance: FloatAmountToMutez(constants.DEFAULT_DELEGATOR_MINIMUM_BALANCE),
			},
			Overrides: make(map[string]RuntimeDelegatorOverride),
			Ignore:    make([]tezos.Address, 0),
		},
		Network: tezpay_configuration.TezosNetworkConfigurationV0{
			RpcUrl:                 constants.DEFAULT_RPC_URL,
			TzktUrl:                constants.DEFAULT_TZKT_URL,
			DoNotPaySmartContracts: false,
		},
		Overdelegation: tezpay_configuration.OverdelegationConfigurationV0{
			IsProtectionEnabled: true,
		},

		NotificationConfigurations: make([]RuntimeNotificatorConfiguration, 0),
		SourceBytes:                []byte{},
	}
}

func (configuration *RuntimeConfiguration) LoadSigner() (common.SignerEngine, error) {
	return signer.Load(string(configuration.PayoutConfiguration.WalletMode))
}
