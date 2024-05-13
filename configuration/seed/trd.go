package seed

import (
	"encoding/json"
	"fmt"
	"strings"

	"blockwatch.cc/tzgo/tezos"
	trd_seed "github.com/alis-is/tezpay/configuration/seed/trd"
	tezpay_configuration "github.com/alis-is/tezpay/configuration/v"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/notifications"
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

	delegatorBellowMinimumBalanceRewardDestination := enums.REWARD_DESTINATION_NONE
	feeOverrides := make(map[string][]tezos.Address, 0)
	ignores := make([]tezos.Address, 0)
	if len(configuration.RulesMap) > 0 {
		for k, v := range configuration.RulesMap {
			if k == "mindelegation" {
				if v == "TOE" {
					delegatorBellowMinimumBalanceRewardDestination = enums.REWARD_DESTINATION_EVERYONE
				}
				continue
			}

			switch v {
			// if TOE -> ignore
			case "TOE":
				if addr, err := tezos.ParseAddress(k); err == nil {
					ignores = append(ignores, addr)
				} else {
					log.Warnf("failed to parse address %s, please adjust configuration manually", k)
				}
				// if TOB -> fee 1
			case "TOB":
				fallthrough
				// if TOF -> fee 1
			case "TOF":
				if addr, err := tezos.ParseAddress(k); err == nil {
					if _, ok := feeOverrides["1"]; !ok {
						feeOverrides["1"] = make([]tezos.Address, 0)
					}
					feeOverrides["1"] = append(feeOverrides["1"], addr)
				} else {
					log.Warnf("failed to parse address %s, please adjust configuration manually", k)
				}
			case "Dexter":
				log.Warnf("we do not support dexter right now, please check your configuration file and migrate it manually")
			default:
				targetAddr, err := tezos.ParseAddress(v)
				if err == nil { // if address redirect
					if sourceAddr, err := tezos.ParseAddress(k); err == nil {
						if v, ok := delegatorOverrides[sourceAddr.String()]; ok {
							if v.Recipient != tezos.ZeroAddress {
								log.Warnf("address %s already has recipient %s, please adjust configuration manually", k, v.Recipient)
							} else {
								v.Recipient = targetAddr
							}
						} else {
							delegatorOverrides[sourceAddr.String()] = tezpay_configuration.DelegatorOverrideV0{
								Recipient: targetAddr,
							}
						}
					} else {
						log.Warnf("failed to parse address %s, please adjust configuration manually", k)
					}
				} else {
					log.Warnf("failed to parse '%s' - unknown rules map value, please adjust configuration manually", v)
				}
			}
		}
	}

	notificationConfigurations := make([]json.RawMessage, 0)
	if configuration.Plugins != nil {
		for t, plugin := range configuration.Plugins {
			switch t {
			case "email":
				oldConfig := trd_seed.EmailPluginConfigurationV1{}
				err := plugin.Decode(&oldConfig)
				if err != nil {
					log.Warnf("we are not able to migrate email plugin configuration right now, please check your configuration file and migrate it manually")
					continue
				}
				configuration := notifications.EmailNotificatorConfiguration{
					Type:       "email",
					Sender:     oldConfig.SMTPSender,
					SmtpUser:   oldConfig.SMTPUser,
					SmtpPass:   oldConfig.SMTPPass,
					Recipients: oldConfig.SMTPRecipients,
				}
				if oldConfig.SMTPPort == 0 {
					configuration.SmtpServer = oldConfig.SMTPHost
				} else {
					configuration.SmtpServer = fmt.Sprintf("%s:%d", oldConfig.SMTPHost, oldConfig.SMTPPort)
				}
				result, err := json.Marshal(configuration)
				if err != nil {
					log.Warnf("we are not able to migrate twitter plugin configuration right now, please check your configuration file and migrate it manually")
					continue
				}
				notificationConfigurations = append(notificationConfigurations, result)
			case "webhook":
				log.Warnf("POST request by trd webhook plugin differs from tezpay, you may have to adjust your webhook logic")
				var configuration trd_seed.WebhookPluginConfigurationV1
				err := plugin.Decode(&configuration)
				if err != nil {
					// log and skip
					log.Warnf("we are not able to migrate webhook plugin configuration right now, please check your configuration file and migrate it manually")
					continue
				}
				result, err := json.Marshal(map[string]interface{}{
					"type":  "webhook",
					"url":   configuration.Endpoint,
					"token": configuration.Token,
				})
				if err != nil {
					log.Warnf("we are not able to migrate webhook plugin configuration right now, please check your configuration file and migrate it manually")
					continue
				}
				notificationConfigurations = append(notificationConfigurations, result)
			case "telegram":
				var configuration trd_seed.TelegramPluginConfigurationV1
				err := plugin.Decode(&configuration)
				if err != nil {
					// log and skip
					log.Warnf("we are not able to migrate telegram plugin configuration right now, please check your configuration file and migrate it manually")
					continue
				}
				if len(configuration.AdminChatsIds) > 0 {
					config, err := json.Marshal(map[string]interface{}{
						"type":             "telegram",
						"admin":            true,
						"recipients":       configuration.AdminChatsIds,
						"api_token":        configuration.BotApiKey,
						"message_template": configuration.TelegramText,
					})
					if err == nil {
						notificationConfigurations = append(notificationConfigurations, config)
					} else {
						log.Warnf("we are not able to migrate telegram plugin configuration right now, please check your configuration file and migrate it manually\n - %s", err.Error())
					}
				}
				if len(configuration.ChatIds) > 0 {
					configuration.PayoutChatsIds = append(configuration.PayoutChatsIds, configuration.ChatIds...)
				}
				if len(configuration.PayoutChatsIds) > 0 {
					config, err := json.Marshal(map[string]interface{}{
						"type":             "telegram",
						"admin":            false,
						"recipients":       configuration.PayoutChatsIds,
						"api_token":        configuration.BotApiKey,
						"message_template": configuration.TelegramText,
					})
					if err == nil {
						notificationConfigurations = append(notificationConfigurations, config)
					} else {
						log.Warnf("we are not able to migrate telegram plugin configuration right now, please check your configuration file and migrate it manually\n - %s", err.Error())
					}
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
				notificationConfigurations = append(notificationConfigurations, result)
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
				notificationConfigurations = append(notificationConfigurations, result)
			}
		}
	}

	donate := 0.05

	migrated := tezpay_configuration.ConfigurationV0{
		Version:  0,
		BakerPKH: address,
		IncomeRecipients: tezpay_configuration.IncomeRecipientsV0{
			Bonds:  bondRecipients,
			Fees:   feeRecipients,
			Donate: &donate,
		},
		Delegators: tezpay_configuration.DelegatorsConfigurationV0{
			Requirements: tezpay_configuration.DelegatorRequirementsV0{
				MinimumBalance:                        configuration.MinDelegation,
				BellowMinimumBalanceRewardDestination: &delegatorBellowMinimumBalanceRewardDestination,
			},
			Overrides:    delegatorOverrides,
			FeeOverrides: feeOverrides,
			Ignore:       ignores,
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
			Fee:                     configuration.ServiceFee / 100,
			IsPayingTxFee:           !configuration.DelPaysXferFee,
			IsPayingAllocationTxFee: !configuration.DelPaysRaFee,
			IgnoreEmptyAccounts:     !configuration.ReactivateZero,
			WalletMode:              enums.WALLET_MODE_LOCAL_PRIVATE_KEY,
			PayoutMode:              enums.EPayoutMode(configuration.RewardsType),
			MinimumAmount:           configuration.MinPayment,
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
