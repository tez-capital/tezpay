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
					Fee:            0.005,
					NoFee:          true,
					MinimumBalance: 2.5,
				},
			},
			Ignore: []tezos.Address{tezos.ZeroAddress, tezos.BurnAddress},
		},
		Network: tezpay_configuration.TezosNetworkConfigurationV0{
			RpcUrl:                 "https://mainnet.api.tez.ie",
			TzktUrl:                "https://api.tzkt.io/",
			Explorer:               "https://tzstats.com/",
			DoNotPaySmartContracts: true,
		},
		Overdelegation: tezpay_configuration.OverdelegationConfigurationV0{
			IsProtectionEnabled: true,
		},
		PayoutConfiguration: tezpay_configuration.PayoutConfigurationV0{
			WalletMode:              enums.WALLET_MODE_LOCAL_PRIVATE_KEY,
			Fee:                     .075,
			IsPayingTxFee:           true,
			IsPayingAllocationTxFee: true,
			MinimumAmount:           10.5,
			Explorer:                "https://tzstats.com/",
		},
		NotificationConfigurations: []map[string]interface{}{
			{
				"type":             "discord",
				"webhook_url":      "https://my-discord-webhook.com/",
				"message_template": "my awesome message",
			},
			{
				"type":                "twitter",
				"access_token":        "your access token",
				"access_token_secret": "your access token secret",
				"consumer_key":        "your consumer key",
				"consumer_secret":     "your consumer secret",
				"message_template":    "my awesome message",
			},
			{
				"type":             "telegram",
				"api_token":        "your api token",
				"receivers":        []interface{}{"list of chat numbers without quotes", -1234567890},
				"message_template": "my awesome message",
			},
			{
				"type":             "email",
				"sender":           "my@email.is",
				"smtp_server":      "smtp.gmail.com",
				"smtp_identity":    "",
				"smtp_username":    "my@email.is",
				"smtp_password":    "password123",
				"recipients":       []string{"my-follower1@email.is", "my-follower2@email.is"},
				"message_template": "my awesome message",
			},
			{
				"type": "external",
				"path": "path to external notificator binary",
				"args": []string{"--kind", "<kind>", "<data>"},
			},
		},
		IncomeRecipients: tezpay_configuration.IncomeRecipientsV0{
			Bonds: map[string]float64{
				"tz1P6WKJu2rcbxKiKRZHKQKmKrpC9TfW1AwM": 0.455,
				"tz1X7U9XxVz6NDxL4DSZhijME61PW45bYUJE": 0.545,
			},
			Fees: map[string]float64{
				"tz1P6WKJu2rcbxKiKRZHKQKmKrpC9TfW1AwM": 0.455,
				"tz1X7U9XxVz6NDxL4DSZhijME61PW45bYUJE": 0.545,
			},
			Donate: 2.5,
			Donations: map[string]float64{
				"tz1P6WKJu2rcbxKiKRZHKQKmKrpC9TfW1AwM": 0.10,
				"tz1UGkfyrT9yBt6U5PV7Qeui3pt3a8jffoWv": 0.90,
			},
		},
	}

	sample, _ := hjson.Marshal(config)
	_ = os.WriteFile("docs/configuration/config.sample.hjson", sample, 0644)
}
