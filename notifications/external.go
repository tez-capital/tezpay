package notifications

import (
	"encoding/json"
	"errors"
	"os/exec"
	"strings"

	"github.com/alis-is/tezpay/core/common"
	log "github.com/sirupsen/logrus"
)

type extedrnalNotificatorConfiguration struct {
	Type string   `json:"type"`
	Path string   `json:"path"`
	Args []string `json:"args,omitempty"`
}

type ExternalNotificator struct {
	path string
	args []string
}

func InitExternalNotificator(configurationBytes []byte) (*ExternalNotificator, error) {
	configuration := extedrnalNotificatorConfiguration{}
	err := json.Unmarshal(configurationBytes, &configuration)
	if err != nil {
		return nil, err
	}
	log.Trace("external notificator initialized")

	args := configuration.Args
	if len(args) == 0 {
		args = []string{"<kind>", "<data>"}
	}

	return &ExternalNotificator{
		path: configuration.Path,
		args: args,
	}, nil
}

func ValidateExternalConfiguration(configurationBytes []byte) error {
	configuration := extedrnalNotificatorConfiguration{}
	err := json.Unmarshal(configurationBytes, &configuration)
	if err != nil {
		return err
	}

	if configuration.Path == "" {
		return errors.New("invalid external notificator path")
	}
	return nil
}

func (en *ExternalNotificator) mapArgs(kind NotificationKind, data string) []string {
	args := make([]string, len(en.args))
	for i, v := range en.args {
		mappedArg := v
		mappedArg = strings.ReplaceAll(mappedArg, "<kind>", string(PAYOUT_SUMMARY_NOTIFICATION))
		mappedArg = strings.ReplaceAll(mappedArg, "<data>", data)
		args[i] = mappedArg
	}
	return args
}

func (en *ExternalNotificator) PayoutSummaryNotify(summary *common.CyclePayoutSummary) error {
	data, _ := json.Marshal(summary)
	args := en.mapArgs(PAYOUT_SUMMARY_NOTIFICATION, string(data))
	cmd := exec.Command(en.path, args...)
	return cmd.Run()
}

func (en *ExternalNotificator) AdminNotify(msg string) error {
	args := en.mapArgs(ADMIN_NOTIFICATION, msg)
	cmd := exec.Command(en.path, args...)
	return cmd.Run()
}

func (en *ExternalNotificator) TestNotify() error {
	args := en.mapArgs(TEST_NOTIFICATION, "test notification")
	cmd := exec.Command(en.path, args...)
	return cmd.Run()
}
