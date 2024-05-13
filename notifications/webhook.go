package notifications

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants"
	log "github.com/sirupsen/logrus"
)

type webhookAuthorization string

const (
	WebhookAuthNone   webhookAuthorization = "none"
	WebhookAuthBearer webhookAuthorization = "bearer"
)

type webhookNotificatorConfiguration struct {
	Type  string               `json:"type"`
	Url   string               `json:"url"`
	Token string               `json:"token"`
	Auth  webhookAuthorization `json:"auth"`
}

type WebhookNotificator struct {
	url   string
	token string
	auth  webhookAuthorization
}

func InitWebhookNotificator(configurationBytes []byte) (*WebhookNotificator, error) {
	configuration := webhookNotificatorConfiguration{}
	err := json.Unmarshal(configurationBytes, &configuration)
	if err != nil {
		return nil, err
	}

	log.Trace("webhook notificator initialized")

	return &WebhookNotificator{
		url:   configuration.Url,
		token: configuration.Token,
		auth:  configuration.Auth,
	}, nil
}

func ValidateWebhookConfiguration(configurationBytes []byte) error {
	configuration := webhookNotificatorConfiguration{}
	err := json.Unmarshal(configurationBytes, &configuration)
	if err != nil {
		return err
	}
	if configuration.Url == "" {
		return errors.Join(constants.ErrInvalidNotificatorConfiguration, errors.New("invalid url"))
	}
	_, err = url.Parse(configuration.Url)
	if err != nil {
		return errors.Join(constants.ErrInvalidNotificatorConfiguration, errors.New("invalid url"))
	}
	if configuration.Auth == WebhookAuthBearer && configuration.Token == "" {
		return errors.Join(constants.ErrInvalidNotificatorConfiguration, errors.New("invalid bearer token"))
	}
	return nil
}

func (wn *WebhookNotificator) post(data interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", wn.url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if wn.auth == WebhookAuthBearer {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", wn.token))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to make request, status code: %d", resp.StatusCode)
	}
	return nil
}

func (wn *WebhookNotificator) PayoutSummaryNotify(summary *common.CyclePayoutSummary, additionalData map[string]string) error {
	return wn.post(summary)
}

func (wn *WebhookNotificator) AdminNotify(msg string) error {
	return wn.post(msg)
}

func (wn *WebhookNotificator) TestNotify() error {
	return wn.post(common.CyclePayoutSummary{})
}
