package migrations

import (
	"encoding/json"
	"strings"

	"blockwatch.cc/tzgo/tezos"
	bc_configuration "github.com/alis-is/tezpay/configuration/bc"
	tezpay_configuration "github.com/alis-is/tezpay/configuration/tezpay"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/hjson/hjson-go/v4"
	log "github.com/sirupsen/logrus"
)

func aliasing(configuration []byte) []byte {
	config := string(configuration)
	//notification aliasing
	config = strings.ReplaceAll(config, "messageTemplate", "message_template")
	// twitter
	config = strings.ReplaceAll(config, "api_key_secret", "consumer_secret")
	config = strings.ReplaceAll(config, "api_key", "consumer_key")
	// discord
	config = strings.ReplaceAll(config, "webhook", "webhook_url")
	// message template aliasing
	config = strings.ReplaceAll(config, "<T_REWARDS>", "<DistributedRewards>")
	config = strings.ReplaceAll(config, "<CYCLE>", "<Cycle>")
	config = strings.ReplaceAll(config, "<N_DELEGATORS>", "<Delegators>")
	return []byte(config)
}

func MigrateBcToTPv0(sourceBytes []byte) ([]byte, *ConfigurationVersionInfo, error) {
	log.Debug("migrating bc configuration to tezpay")
	configuration := bc_configuration.GetDefault()
	err := hjson.Unmarshal(aliasing(sourceBytes), &configuration)
	if err != nil {
		return []byte{}, nil, err
	}
	version := uint(0)

	address, err := tezos.ParseAddress(configuration.BakerPKH)
	if err != nil {
		return []byte{}, nil, err
	}

	feeRecipients := make(map[string]float64, len(configuration.IncomeRecipients.FeeRewards))
	if len(configuration.IncomeRecipients.FeeRewards) > 0 {
		for recipient, share := range configuration.IncomeRecipients.FeeRewards {
			feeRecipients[recipient] = share
		}
	}

	bondRecipients := make(map[string]float64, len(configuration.IncomeRecipients.BondRewards))
	if len(configuration.IncomeRecipients.BondRewards) > 0 {
		for recipient, share := range configuration.IncomeRecipients.BondRewards {
			bondRecipients[recipient] = share
		}
	}

	overdelegationExcludedAddresses := make([]tezos.Address, len(configuration.Overdelegation.ExcludedAddresses))
	for index, pkh := range configuration.Overdelegation.ExcludedAddresses {
		if addr, err := tezos.ParseAddress(pkh); err == nil {
			overdelegationExcludedAddresses[index] = addr
		} else {
			log.Warnf("invalid PKH in overdelegation protections address list: '%s'", pkh)
			continue
		}
	}

	delegatorOverrides := make(map[string]tezpay_configuration.DelegatorOverrideV0)
	for k, delegatorOverride := range configuration.DelegatorOverrides {
		if addr, err := tezos.ParseAddress(delegatorOverride.Recipient); err == nil {
			delegatorOverrides[k] = tezpay_configuration.DelegatorOverrideV0{
				Recipient:      addr,
				Fee:            delegatorOverride.Fee / 100,
				NoFee:          delegatorOverride.Fee == 0.,
				MinimumBalance: 0,
			}
		} else {
			log.Warnf("invalid PKH in delegator overrides: '%s'", delegatorOverride.Recipient)
			continue
		}
	}

	migrated := tezpay_configuration.ConfigurationV0{
		Version:  0,
		BakerPKH: address,
		IncomeRecipients: tezpay_configuration.IncomeRecipientsV0{
			Bonds:  bondRecipients,
			Fees:   feeRecipients,
			Donate: 0.05,
		},
		Delegators: tezpay_configuration.DelegatorsConfigurationV0{
			Requirements: tezpay_configuration.DelegatorRequirementsV0{
				MinimumBalance: configuration.DelegatorRequirements.MinimumBalance,
			},
			Overrides: delegatorOverrides,
			Ignore:    overdelegationExcludedAddresses,
		},
		Network: tezpay_configuration.TezosNetworkConfigurationV0{
			RpcUrl:                 configuration.Network.RpcUrl,
			TzktUrl:                constants.DEFAULT_TZKT_URL,
			DoNotPaySmartContracts: configuration.Network.DoNotPaySmartContracts,
		},
		Overdelegation: tezpay_configuration.OverdelegationConfigurationV0{
			IsProtectionEnabled: configuration.Overdelegation.IsProtectionEnabled,
		},
		PayoutConfiguration: tezpay_configuration.PayoutConfigurationV0{
			Fee:           configuration.Fee / 100,
			IsPayingTxFee: configuration.PaymentRequirements.IsPayingTxFee,
			WalletMode:    enums.WalletMode(configuration.WalletMode),
			MinimumAmount: configuration.PaymentRequirements.MinimumAmount,
		},
		NotificationConfigurations: configuration.NotificationConfigurations,
	}

	migratedBytes, err := json.Marshal(migrated)
	if err != nil {
		return []byte{}, nil, err
	}
	log.Debug("migrated bc configuration successfully")
	return migratedBytes, &ConfigurationVersionInfo{
		Version: &version,
	}, nil
}
