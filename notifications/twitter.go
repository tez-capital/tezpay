package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants"
	"github.com/dghubble/oauth1"
	twitter "github.com/g8rswimmer/go-twitter/v2"
	log "github.com/sirupsen/logrus"
)

type twitterNotificatorConfiguration struct {
	Type              string `json:"type"`
	MessageTemplate   string `json:"message_template"`
	ConsumerKey       string `json:"consumer_key"`
	ConsumerSecret    string `json:"consumer_secret"`
	AccessToken       string `json:"access_token"`
	AccessTokenSecret string `json:"access_token_secret"`
}

type TwitterNotificator struct {
	client          *twitter.Client
	messageTemplate string
}

type authorize struct{}

func (a authorize) Add(req *http.Request) {}

const (
	DEFAULT_TWITTER_MESSAGE_TEMPLATE = "A total of <DistributedRewards> was distributed for cycle <Cycle> to <Delegators> delegators and donated <DonatedTotal> using #tezpay on the #tezos blockchain."
)

func InitTwitterNotificator(configurationBytes []byte) (*TwitterNotificator, error) {
	configuration := twitterNotificatorConfiguration{}
	err := json.Unmarshal(configurationBytes, &configuration)
	if err != nil {
		return nil, err
	}
	config := oauth1.NewConfig(configuration.ConsumerKey, configuration.ConsumerSecret)
	// Pass in your Access Token and your Access Token Secret
	token := oauth1.NewToken(configuration.AccessToken, configuration.AccessTokenSecret)

	httpClient := config.Client(oauth1.NoContext, token)

	client := &twitter.Client{
		Authorizer: authorize{},
		Client:     httpClient,
		Host:       "https://api.twitter.com",
	}
	msgTemplate := configuration.MessageTemplate
	if msgTemplate == "" {
		msgTemplate = DEFAULT_TWITTER_MESSAGE_TEMPLATE
	}

	log.Trace("twitter notificator initialized")

	return &TwitterNotificator{
		client:          client,
		messageTemplate: msgTemplate,
	}, nil
}

func ValidateTwitterConfiguration(configurationBytes []byte) error {
	configuration := twitterNotificatorConfiguration{}
	err := json.Unmarshal(configurationBytes, &configuration)
	if err != nil {
		return err
	}
	if configuration.AccessToken == "" {
		return errors.Join(constants.ErrInvalidNotificatorConfiguration, errors.New("invalid twitter access token"))
	}
	if configuration.AccessTokenSecret == "" {
		return errors.Join(constants.ErrInvalidNotificatorConfiguration, errors.New("invalid twitter access token secret"))
	}
	if configuration.ConsumerKey == "" {
		return errors.Join(constants.ErrInvalidNotificatorConfiguration, errors.New("invalid twitter consumer key"))
	}
	if configuration.ConsumerSecret == "" {
		return errors.Join(constants.ErrInvalidNotificatorConfiguration, errors.New("invalid twitter consumer secret"))
	}

	return nil
}

func (tn *TwitterNotificator) PayoutSummaryNotify(summary *common.CyclePayoutSummary, additionalData map[string]string) error {
	_, err := tn.client.CreateTweet(context.Background(), twitter.CreateTweetRequest{
		Text: PopulateMessageTemplate(tn.messageTemplate, summary, additionalData),
	})
	return err
}

func (tn *TwitterNotificator) AdminNotify(msg string) error {
	_, err := tn.client.CreateTweet(context.Background(), twitter.CreateTweetRequest{
		Text: msg,
	})
	return err
}

func (tn *TwitterNotificator) TestNotify() error {
	_, err := tn.client.CreateTweet(context.Background(), twitter.CreateTweetRequest{
		Text: fmt.Sprintf("Notification test from %s (%s) 👀", constants.CODENAME, constants.VERSION),
	})
	return err
}
