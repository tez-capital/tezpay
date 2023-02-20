package trd_seed

import "gopkg.in/yaml.v3"

type TrdRewardsType string

const (
	TrdRewardsTypeActual TrdRewardsType = "actual"
	TrdRewardsTypeIdeal  TrdRewardsType = "ideal"
)

type TelegramPluginConfigurationV1 struct {
	Type           string  `json:"type" yaml:"-"`
	AdminChatsIds  []int64 `yaml:"admin_chats_ids"`                       // chat ids to send admin messages to
	PayoutChatsIds []int64 `yaml:"payout_chats_ids"`                      // chat ids to send payout messages to
	BotApiKey      string  `json:"api_token" yaml:"bot_api_key"`          // telegram bot api key
	TelegramText   string  `json:"message_template" yaml:"telegram_text"` // telegram text
}

// twitter and discord are easily mapable to tp format
// we just parse as yaml fill in the type and marshal to json

type TwitterPluginConfigurationV1 struct {
	Type         string `json:"type" yaml:"-"`
	ApiKey       string `json:"consumer_key" yaml:"api_key"`
	ApiSecret    string `json:"consumer_secret" yaml:"api_secret"`
	AccessToken  string `json:"access_token" yaml:"access_token"`
	AccessSecret string `json:"access_token_secret" yaml:"access_secret"`
	TweetText    string `json:"message_template" yaml:"tweet_text"`
}

type DiscordPluginConfigurationV1 struct {
	Type        string `json:"type" yaml:"-"`
	Endpoint    string `json:"webhook_url" yaml:"endpoint"`
	IsAdmin     bool   `json:"admin" yaml:"send_admin"`
	DiscordText string `json:"message_template" yaml:"discord_text"`
}

type ConfigurationV1 struct {
	Version        string               `yaml:"version"`
	BakingAddress  string               `yaml:"baking_address"`
	PaymentAddress string               `yaml:"payment_address"`
	RewardsType    TrdRewardsType       `yaml:"rewards_type"`
	ServiceFee     float64              `yaml:"service_fee"`
	FoundersMap    map[string]float64   `yaml:"founders_map"`
	OwnersMap      map[string]float64   `yaml:"owners_map"`
	SpecialsMap    map[string]float64   `yaml:"specials_map"`
	SupportersSet  map[string]string    `yaml:"supporters_set"`
	MinDelegation  float64              `yaml:"min_delegation_amt"`
	MinPayment     float64              `yaml:"min_payment_amt"`
	ReactivateZero bool                 `yaml:"reactivate_zeroed"`
	DelPaysXferFee bool                 `yaml:"delegator_pays_xfer_fee"`
	DelPaysRaFee   bool                 `yaml:"delegator_pays_ra_fee"`
	PayDenRewards  bool                 `yaml:"pay_denunciation_rewards"`
	RulesMap       map[string]yaml.Node `yaml:"rules_map"`
	Plugins        map[string]yaml.Node `yaml:"plugins"`
}

func GetDefault() ConfigurationV1 {
	return ConfigurationV1{
		Version:        "1.0",
		BakingAddress:  "",
		PaymentAddress: "",
		RewardsType:    TrdRewardsTypeActual,
		ServiceFee:     5,
		FoundersMap:    make(map[string]float64),
		OwnersMap:      make(map[string]float64),
		SpecialsMap:    make(map[string]float64),
		SupportersSet:  make(map[string]string),
		MinDelegation:  0,
		MinPayment:     0,
		ReactivateZero: false,
		DelPaysXferFee: false,
		DelPaysRaFee:   false,
		PayDenRewards:  false,
		RulesMap:       make(map[string]yaml.Node),
		Plugins:        make(map[string]yaml.Node),
	}
}
