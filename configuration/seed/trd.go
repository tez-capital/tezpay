package seed

import (
	"encoding/json"
	"strings"

	"blockwatch.cc/tzgo/tezos"
	trd_seed "github.com/alis-is/tezpay/configuration/seed/trd"
	tezpay_configuration "github.com/alis-is/tezpay/configuration/v"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/hjson/hjson-go/v4"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// %CYCLE%, %NDELEGATORS%, %TREWARDS%
func trdAliasing(configuration []byte) []byte {
	config := string(configuration)
	config = strings.ReplaceAll(config, "%TREWARDS%", "<DistributedRewards>")
	config = strings.ReplaceAll(config, "%CYCLE%", "<Cycle>")
	config = strings.ReplaceAll(config, "%NDELEGATORS%", "<Delegators>")
	return []byte(config)
}

func MigrateTrdv1ToTPv0(sourceBytes []byte) ([]byte, error) {
	log.Debug("migrating trd configuration to tezpay")
	configuration := trd_seed.GetDefault()
	err := yaml.Unmarshal(trdAliasing(sourceBytes), &configuration)
	if err != nil {
		return []byte{}, err
	}

	address, err := tezos.ParseAddress(configuration.BakingAddress)
	if err != nil {
		return []byte{}, err
	}

	feeRecipients := make(map[string]float64, len(configuration.FoundersMap))
	if len(configuration.FoundersMap) > 0 {
		for recipient, share := range configuration.FoundersMap {
			feeRecipients[recipient] = share
		}
	}

	bondRecipients := make(map[string]float64, len(configuration.OwnersMap))
	if len(configuration.OwnersMap) > 0 {
		for recipient, share := range configuration.OwnersMap {
			bondRecipients[recipient] = share
		}
	}

	delegatorOverrides := make(map[string]tezpay_configuration.DelegatorOverrideV0, len(configuration.SpecialsMap)+len(configuration.SupportersSet))
	if len(configuration.SpecialsMap) > 0 {
		for recipient, share := range configuration.SpecialsMap {
			if addr, err := tezos.ParseAddress(recipient); err == nil {
				delegatorOverrides[recipient] = tezpay_configuration.DelegatorOverrideV0{
					Recipient:      addr,
					Fee:            &share,
					MinimumBalance: 0,
				}
			}
		}
	}

	if len(configuration.SupportersSet) > 0 {
		fee := 0.0
		for recipient := range configuration.SupportersSet {
			if _, err := tezos.ParseAddress(recipient); err == nil {
				if v, ok := delegatorOverrides[recipient]; ok {
					if v.Fee == nil {
						v.Fee = &fee
					}
					continue
				}
				delegatorOverrides[recipient] = tezpay_configuration.DelegatorOverrideV0{
					Fee: &fee,
				}
			}
		}
	}

	if len(configuration.RulesMap) > 0 {
		// TODO: rules
		log.Warnf("we do not support migration of rules right now")
	}

	notificationConfigurations := make([]map[string]interface{}, 0)
	if configuration.Plugins != nil {
		for t, plugin := range configuration.Plugins {
			switch t {
			case "email":
				log.Warnf("we are not able to migrate email plugin configuration right now, please check your configuration file and migrate it manually")
			case "webhook":
				log.Warnf("we do not support webhook notificators right now")
			case "telegram":
				var configuration trd_seed.TelegramPluginConfigurationV1
				err := plugin.Decode(&configuration)
				if err != nil {
					// log and skip
					log.Warnf("we are not able to migrate telegram plugin configuration right now, please check your configuration file and migrate it manually")
					continue
				}
				if len(configuration.AdminChatsIds) > 0 {
					notificationConfigurations = append(notificationConfigurations, map[string]interface{}{
						"type":             "telegram",
						"admin":            true,
						"recipients":       configuration.AdminChatsIds,
						"api_token":        configuration.BotApiKey,
						"message_template": configuration.TelegramText,
					})
				}
				if len(configuration.AdminChatsIds) > 0 {
					notificationConfigurations = append(notificationConfigurations, map[string]interface{}{
						"type":             "telegram",
						"admin":            false,
						"recipients":       configuration.PayoutChatsIds,
						"api_token":        configuration.BotApiKey,
						"message_template": configuration.TelegramText,
					})
				}
			case "twitter":
				var configuration trd_seed.TwitterPluginConfigurationV1
				err := plugin.Decode(&configuration)
				if err != nil {
					// log and skip
					log.Warnf("we are not able to migrate twitter plugin configuration right now, please check your configuration file and migrate it manually")
					continue
				}
				configuration.Type = "twitter"
				result, err := json.Marshal(configuration)
				if err != nil {
					log.Warnf("we are not able to migrate twitter plugin configuration right now, please check your configuration file and migrate it manually")
					continue
				}
				var notificationConfiguration map[string]interface{}
				err = json.Unmarshal(result, &notificationConfiguration)
				if err != nil {
					log.Warnf("we are not able to migrate twitter plugin configuration right now, please check your configuration file and migrate it manually")
					continue
				}
				notificationConfigurations = append(notificationConfigurations, notificationConfiguration)
			case "discord":
				var configuration trd_seed.DiscordPluginConfigurationV1
				err := plugin.Decode(&configuration)
				if err != nil {
					// log and skip
					log.Warnf("we are not able to migrate discord plugin configuration right now, please check your configuration file and migrate it manually")
					continue
				}
				configuration.Type = "discord"
				result, err := json.Marshal(configuration)
				if err != nil {
					log.Warnf("we are not able to migrate discord plugin configuration right now, please check your configuration file and migrate it manually")
					continue
				}
				var notificationConfiguration map[string]interface{}
				err = json.Unmarshal(result, &notificationConfiguration)
				if err != nil {
					log.Warnf("we are not able to migrate discord plugin configuration right now, please check your configuration file and migrate it manually")
					continue
				}
				notificationConfigurations = append(notificationConfigurations, notificationConfiguration)
			}
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
				MinimumBalance: configuration.MinDelegation,
			},
			Overrides: delegatorOverrides,
		},
		Network: tezpay_configuration.TezosNetworkConfigurationV0{
			RpcUrl:                 constants.DEFAULT_TZKT_URL,
			TzktUrl:                constants.DEFAULT_TZKT_URL,
			DoNotPaySmartContracts: false,
		},
		Overdelegation: tezpay_configuration.OverdelegationConfigurationV0{
			IsProtectionEnabled: true,
		},
		PayoutConfiguration: tezpay_configuration.PayoutConfigurationV0{
			Fee:           configuration.ServiceFee / 100,
			IsPayingTxFee: !configuration.DelPaysXferFee,
			WalletMode:    enums.WALLET_MODE_REMOTE_SIGNER,
			PayoutMode:    enums.EPayoutMode(configuration.RewardsType),
			MinimumAmount: configuration.MinPayment,
		},
		NotificationConfigurations: notificationConfigurations,
	}

	migratedBytes, err := hjson.MarshalWithOptions(migrated, getSerializeHjsonOptions())
	if err != nil {
		return []byte{}, err
	}
	log.Debug("migrated bc configuration successfully")
	return migratedBytes, nil
}
