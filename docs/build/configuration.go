package main

import (
	"encoding/json"
	"os"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	tezpay_configuration "github.com/alis-is/tezpay/configuration/v"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/hjson/hjson-go/v4"
)

func GenerateDefaultHJson() {
	config := tezpay_configuration.GetDefaultV0()

	sample, _ := hjson.Marshal(config)
	_ = os.WriteFile("docs/configuration/config.default.hjson", sample, 0644)
}
func genrateSample() *tezpay_configuration.ConfigurationV0 {
	logExtensionConfiguration := json.RawMessage(`{"LOG_FILE": "path/to/my/extension.log"}`)
	feeExtensionConfiguration := json.RawMessage(`{"FEE": 0, "TOKEN": "1", "CONTRACT": "KT1Hkg6qgV3VykjgUXKbWcU3h6oJ1qVxUxZV"}`)

	fee := 0.0
	return &tezpay_configuration.ConfigurationV0{
		Version:  0,
		BakerPKH: tezos.InvalidAddress,
		Delegators: tezpay_configuration.DelegatorsConfigurationV0{
			Requirements: tezpay_configuration.DelegatorRequirementsV0{
				MinimumBalance: float64(0.5),
			},
			Overrides: map[string]tezpay_configuration.DelegatorOverrideV0{
				"tz1P6WKJu2rcbxKiKRZHKQKmKrpC9TfW1AwM": {
					Recipient:      tezos.InvalidAddress,
					Fee:            &fee,
					MinimumBalance: 2.5,
				},
			},
			FeeOverrides: map[string][]tezos.Address{
				"1":  {tezos.ZeroAddress, tezos.BurnAddress},
				".5": {tezos.InvalidAddress},
			},
			Ignore: []tezos.Address{tezos.ZeroAddress, tezos.BurnAddress},
		},
		Network: tezpay_configuration.TezosNetworkConfigurationV0{
			RpcUrl:                 constants.DEFAULT_RPC_URL,
			TzktUrl:                constants.DEFAULT_TZKT_URL,
			Explorer:               "https://tzstats.com/",
			DoNotPaySmartContracts: true,
		},
		Overdelegation: tezpay_configuration.OverdelegationConfigurationV0{
			IsProtectionEnabled: true,
		},
		PayoutConfiguration: tezpay_configuration.PayoutConfigurationV0{
			WalletMode:              enums.WALLET_MODE_LOCAL_PRIVATE_KEY,
			PayoutMode:              enums.PAYOUT_MODE_IDEAL,
			Fee:                     .075,
			IsPayingTxFee:           true,
			IsPayingAllocationTxFee: true,
			MinimumAmount:           10.5,
		},
		NotificationConfigurations: []map[string]interface{}{
			{
				"type":             "discord",
				"webhook_url":      "https://my-discord-webhook.com/",
				"message_template": "my awesome message",
			},
			{
				"type":             "discord",
				"webhook_url":      "https://my-admin-discord-webhook.com/",
				"message_template": "my awesome message",
				"admin":            true,
			},
			{
				"type":             "discord",
				"webhook_id":       "webhook id",
				"webhook_token":    "webhook token",
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
				"smtp_server":      "smtp.gmail.com:443",
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
			Donate: 0.025,
			Donations: map[string]float64{
				"tz1P6WKJu2rcbxKiKRZHKQKmKrpC9TfW1AwM": 0.10,
				"tz1UGkfyrT9yBt6U5PV7Qeui3pt3a8jffoWv": 0.90,
			},
		},
		Extensions: []tezpay_configuration.ExtensionConfigurationV0{
			common.ExtensionDefinition{
				Id:      "log-extension",
				Command: "python3",
				Args:    []string{"/path/to/my/extension.py"},
				Kind:    enums.EXTENSION_STDIO_RPC,
				Hooks: []common.ExtensionHook{{
					Id:   enums.EXTENSION_HOOK_ALL,
					Mode: enums.EXTENSION_HOOK_MODE_READ_ONLY,
				}},
				Configuration: &logExtensionConfiguration,
			},
			common.ExtensionDefinition{
				Id:      "fee-extension",
				Command: `/path/to/my/extension.bin`,
				Args:    []string{"--config", "/path/to/my/extension.config"},
				Kind:    enums.EXTENSION_STDIO_RPC,
				Hooks: []common.ExtensionHook{{
					Id:   enums.EXTENSION_HOOK_AFTER_CANDIDATE_GENERATED,
					Mode: enums.EXTENSION_HOOK_MODE_READ_WRITE,
				}},
				Configuration: &feeExtensionConfiguration,
			},
		},
	}
}

func GenerateSampleHJson() {
	config := genrateSample()

	sample, _ := hjson.Marshal(config)
	_ = os.WriteFile("docs/configuration/config.sample.hjson", sample, 0644)
}
