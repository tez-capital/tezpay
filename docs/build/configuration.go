package main

import (
	"os"

	"blockwatch.cc/tzgo/tezos"
	tezpay_configuration "github.com/alis-is/tezpay/configuration/tezpay"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/hjson/hjson-go/v4"
)

func GenerateDefault() {
	config := tezpay_configuration.GetDefaultV0()

	sample, _ := hjson.Marshal(config)
	_ = os.WriteFile("docs/configuration/config.default.hjson", sample, 0644)
}

func GenerateSample() {
	config := tezpay_configuration.ConfigurationV0{
		Version:  0,
		BakerPKH: tezos.InvalidAddress,
		Delegators: tezpay_configuration.DelegatorsConfigurationV0{
			Requirements: tezpay_configuration.DelegatorRequirementsV0{
				MinimumBalance: float64(0.5),
			},
			Overrides: map[string]tezpay_configuration.DelegatorOverrideV0{
				"tz1P6WKJu2rcbxKiKRZHKQKmKrpC9TfW1AwM": {
					Recipient:      tezos.InvalidAddress,
					Fee:            0.5,
					NoFee:          true,
					MinimumBalance: 2.5,
				},
			},
			Ignore: []tezos.Address{tezos.ZeroAddress, tezos.BurnAddress},
		},
		Network: tezpay_configuration.TezosNetworkConfigurationV0{
			RpcUrl:                 "https://mainnet.api.tez.ie",
			TzktUrl:                "https://api.tzkt.io/v1/",
			DoNotPaySmartContracts: true,
		},
		Overdelegation: tezpay_configuration.OverdelegationConfigurationV0{
			IsProtectionEnabled: true,
		},
		PayoutConfiguration: tezpay_configuration.PayoutConfigurationV0{
			WalletMode:              enums.WALLET_MODE_LOCAL_PRIVATE_KEY,
			Fee:                     7.5,
			IsPayingTxFee:           true,
			IsPayingAllocationTxFee: true,
			MinimumAmount:           10.5,
		},
		NotificationConfigurations: []map[string]interface{}{
			{"type": "discord", "webhook": "https://my-discord-webhook.com/"},
		},
		IncomeRecipients: tezpay_configuration.IncomeRecipientsV0{
			Bonds: map[string]float32{
				"tz1P6WKJu2rcbxKiKRZHKQKmKrpC9TfW1AwM": 45.5,
				"tz1X7U9XxVz6NDxL4DSZhijME61PW45bYUJE": 54.5,
			},
			Fees: map[string]float32{
				"tz1P6WKJu2rcbxKiKRZHKQKmKrpC9TfW1AwM": 45.5,
				"tz1X7U9XxVz6NDxL4DSZhijME61PW45bYUJE": 54.5,
			},
			Donate: 2.5,
			Donations: map[string]float32{
				"tz1P6WKJu2rcbxKiKRZHKQKmKrpC9TfW1AwM": 10,
				"tz1UGkfyrT9yBt6U5PV7Qeui3pt3a8jffoWv": 90,
			},
		},
	}

	sample, _ := hjson.Marshal(config)
	_ = os.WriteFile("docs/configuration/config.sample.hjson", sample, 0644)
}
