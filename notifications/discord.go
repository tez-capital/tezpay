package notifications

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
)

type discordNotificatorConfiguration struct {
	Type            string `json:"type"`
	MessageTemplate string `json:"message_template"`
	WebhookUrl      string `json:"webhook_url"`
	WebhookId       string `json:"webhook_id"`
	WebhookToken    string `json:"webhook_token"`
}

type DiscordNotificator struct {
	session         *discordgo.Session
	messageTemplate string
	token           string
	id              string
}

const (
	DEFAULT_DISCORD_MESSAGE_TEMPLATE = "Report of cycle #<Cycle>"
	// https://github.com/discordjs/discord.js/blob/aec44a0c93f620b22242f35e626d817e831fc8cb/packages/discord.js/src/util/Util.js#L517
	DISCORD_WEBHOOK_REGEX = `https?:\/\/(?:ptb\.|canary\.)?discord\.com\/api(?:\/v\d{1,2})?\/webhooks\/(\d{17,19})\/([\w-]{68})`
)

func InitDiscordNotificator(configurationBytes []byte) (*DiscordNotificator, error) {
	configuration := discordNotificatorConfiguration{}
	err := json.Unmarshal(configurationBytes, &configuration)
	if err != nil {
		return nil, err
	}
	msgTemplate := configuration.MessageTemplate
	if msgTemplate == "" {
		msgTemplate = DEFAULT_DISCORD_MESSAGE_TEMPLATE
	}

	id := configuration.WebhookId
	token := configuration.WebhookToken
	if configuration.WebhookUrl != "" {
		wr, err := regexp.Compile(DISCORD_WEBHOOK_REGEX)
		if err != nil {
			return nil, err
		}
		matched := wr.FindStringSubmatch(configuration.WebhookUrl)
		if len(matched) > 2 {
			id = matched[1]
			token = matched[2]
		} else {
			slog.Warn("failed to parse discord webhook")
		}
	}

	session, err := discordgo.New("")
	if err != nil {
		return nil, err
	}

	slog.Debug("discord notificator initialized")

	return &DiscordNotificator{
		session:         session,
		messageTemplate: msgTemplate,
		id:              id,
		token:           token,
	}, nil
}

func ValidateDiscordConfiguration(configurationBytes []byte) error {
	configuration := discordNotificatorConfiguration{}
	err := json.Unmarshal(configurationBytes, &configuration)
	if err != nil {
		return err
	}
	id := configuration.WebhookId
	token := configuration.WebhookToken
	if configuration.WebhookUrl != "" {
		wr, err := regexp.Compile(DISCORD_WEBHOOK_REGEX)
		if err != nil {
			return err
		}
		matched := wr.FindStringSubmatch(configuration.WebhookUrl)
		if len(matched) > 2 {
			id = matched[1]
			token = matched[2]
		} else {
			return errors.Join(constants.ErrInvalidNotificatorConfiguration, errors.New("failed to parse discord webhook"))
		}
	}
	if id == "" {
		if configuration.WebhookUrl != "" {
			return errors.Join(constants.ErrInvalidNotificatorConfiguration, errors.New("invalid discord webhook url - failed to parse id"))
		}
		return errors.Join(constants.ErrInvalidNotificatorConfiguration, errors.New("invalid discord webhook id"))
	}
	if token == "" {
		if configuration.WebhookUrl != "" {
			return errors.Join(constants.ErrInvalidNotificatorConfiguration, errors.New("invalid discord webhook url - failed to parse token"))
		}
		return errors.Join(constants.ErrInvalidNotificatorConfiguration, errors.New("invalid discord webhook token"))
	}
	return nil
}

func (dn *DiscordNotificator) PayoutSummaryNotify(summary *common.CyclePayoutSummary, additionalData map[string]string) error {

	_, err := dn.session.WebhookExecute(dn.id, dn.token, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title: PopulateMessageTemplate(dn.messageTemplate, summary, additionalData),
				Color: 261239,
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf(`%s v%s`, constants.CODENAME, constants.VERSION),
				},
				Timestamp: time.Now().Format(time.RFC3339),
				Fields: []*discordgo.MessageEmbedField{
					{Name: "Staking Balance", Value: common.MutezToTezS(summary.OwnStakingBalance.Int64())},
					{Name: "Distributed", Value: common.MutezToTezS(summary.DistributedRewards.Int64())},
					{Name: "Delegators", Value: fmt.Sprintf("%d", summary.Delegators)},
					{Name: "Donated", Value: common.MutezToTezS(summary.DonatedTotal.Int64())},
				},
			},
		},
	})
	return err
}

func (dn *DiscordNotificator) AdminNotify(msg string) error {
	_, err := dn.session.WebhookExecute(dn.id, dn.token, true, &discordgo.WebhookParams{
		Content: msg,
	})
	return err
}

func (dn *DiscordNotificator) TestNotify() error {
	_, err := dn.session.WebhookExecute(dn.id, dn.token, true, &discordgo.WebhookParams{
		//Content: fmt.Sprintf("Notification test from %s (%s) 👀", constants.CODENAME, constants.VERSION),
		Embeds: []*discordgo.MessageEmbed{
			{
				Title: "Test Cycle Report",
				Color: 261239,
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf(`%s v%s`, constants.CODENAME, constants.VERSION),
				},
				Timestamp: time.Now().Format(time.RFC3339),
				Fields: []*discordgo.MessageEmbedField{
					{Name: "Staking Blanace", Value: "test value"},
					{Name: "Distributed", Value: "test value"},
					{Name: "Delegators", Value: "test value"},
					{Name: "Donated", Value: "test value"},
				},
			},
		},
	})
	return err
}
