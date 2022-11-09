package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/core/common"
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
	DEFAULT_TWITTER_MESSAGE_TEMPLATE = "A total of <DistributedRewards> was distributed for cycle <Cycle> to <Delegators> delegators using #tezpay on the #tezos blockchain."
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

	log.Trace("twitter plugin initialized")

	return &TwitterNotificator{
		client:          client,
		messageTemplate: msgTemplate,
	}, nil
}

func (tn *TwitterNotificator) Notify(summary *common.CyclePayoutSummary) error {
	_, err := tn.client.CreateTweet(context.Background(), twitter.CreateTweetRequest{
		Text: PopulateMessageTemplate(tn.messageTemplate, summary),
	})
	return err
}

func (tn *TwitterNotificator) TestNotify() error {
	_, err := tn.client.CreateTweet(context.Background(), twitter.CreateTweetRequest{
		Text: fmt.Sprintf("Notification test from %s (%s) 👀", constants.CODENAME, constants.VERSION),
	})
	return err
}
