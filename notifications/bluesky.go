package notifications

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
)

type blueskyNotificatorConfiguration struct {
	Type            string `json:"type"`
	MessageTemplate string `json:"message_template"`
	Handle          string `json:"handle"`
	AppPassword     string `json:"app_password"`
}

type BlueskyNotificator struct {
	handle          string
	appPassword     string
	messageTemplate string
	client          *http.Client
	accessJwt       string
}

const (
	DEFAULT_BLUESKY_MESSAGE_TEMPLATE = "A total of <DistributedRewards> was distributed for cycle <Cycle> to <Delegators> delegators and donated <DonatedTotal> using #tezpay on the #tezos blockchain."
	BLUESKY_API_URL                 = "https://bsky.social/xrpc"
)

func InitBlueskyNotificator(configurationBytes []byte) (*BlueskyNotificator, error) {
	configuration := blueskyNotificatorConfiguration{}
	err := json.Unmarshal(configurationBytes, &configuration)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	bn := &BlueskyNotificator{
		handle:          configuration.Handle,
		appPassword:     configuration.AppPassword,
		client:          client,
		messageTemplate: configuration.MessageTemplate,
	}

	if bn.messageTemplate == "" {
		bn.messageTemplate = DEFAULT_BLUESKY_MESSAGE_TEMPLATE
	}

	// Initial authentication
	err = bn.authenticate()
	if err != nil {
		return nil, fmt.Errorf("bluesky authentication failed: %w", err)
	}

	slog.Debug("bluesky notificator initialized")
	return bn, nil
}

func (bn *BlueskyNotificator) authenticate() error {
	data := map[string]string{
		"identifier": bn.handle,
		"password":   bn.appPassword,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", BLUESKY_API_URL+"/com.atproto.server.createSession", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := bn.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed with status: %d", resp.StatusCode)
	}

	var result struct {
		AccessJwt string `json:"accessJwt"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	bn.accessJwt = result.AccessJwt
	return nil
}

func ValidateBlueskyConfiguration(configurationBytes []byte) error {
	configuration := blueskyNotificatorConfiguration{}
	err := json.Unmarshal(configurationBytes, &configuration)
	if err != nil {
		return err
	}
	if configuration.Handle == "" {
		return errors.Join(constants.ErrInvalidNotificatorConfiguration, errors.New("invalid bluesky handle"))
	}
	if configuration.AppPassword == "" {
		return errors.Join(constants.ErrInvalidNotificatorConfiguration, errors.New("invalid bluesky app password"))
	}

	return nil
}

func (bn *BlueskyNotificator) createPost(text string) error {
	data := map[string]interface{}{
		"collection": "app.bsky.feed.post",
		"repo":      bn.handle,
		"record": map[string]interface{}{
			"text":      text,
			"createdAt": time.Now().Format(time.RFC3339),
			"$type":     "app.bsky.feed.post",
		},
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", BLUESKY_API_URL+"/com.atproto.repo.createRecord", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bn.accessJwt)

	resp, err := bn.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// Token expired, try to re-authenticate
		if err := bn.authenticate(); err != nil {
			return err
		}
		// Retry the request
		return bn.createPost(text)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create post with status: %d", resp.StatusCode)
	}

	return nil
}

func (bn *BlueskyNotificator) PayoutSummaryNotify(summary *common.CyclePayoutSummary, additionalData map[string]string) error {
	text := PopulateMessageTemplate(bn.messageTemplate, summary, additionalData)
	return bn.createPost(text)
}

func (bn *BlueskyNotificator) AdminNotify(msg string) error {
	return bn.createPost(msg)
}

func (bn *BlueskyNotificator) TestNotify() error {
	return bn.createPost(fmt.Sprintf("Notification test from %s (%s) 👀", constants.CODENAME, constants.VERSION))
}
