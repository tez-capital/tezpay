package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/utils"
	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
)

func requireConfirmation(msg string) error {
	proceed := false
	if utils.IsTty() {
		prompt := &survey.Confirm{
			Message: msg,
		}
		survey.AskOne(prompt, &proceed)
	}
	if !proceed {
		return errors.New("not confirmed")
	}
	return nil
}

func assertRequireConfirmation(msg string) {
	assertRunWithParam(requireConfirmation, msg, EXIT_OPERTION_CANCELED)
}

func checkLatestVersion() {
	log.Info("checking for new version")
	// https://api.github.com/repos/tez-capital/tezpay/releases/latest
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", constants.TEZPAY_REPOSITORY))
	if err != nil {
		log.Warnf("Failed to check latest version!")
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Warnf("Failed to check latest version!")
		return
	}
	var info versionInfo
	err = json.Unmarshal(body, &info)
	if err != nil {
		log.Warnf("Failed to check latest version!")
		return
	}
	latestVersion := info.Version
	if latestVersion == "" {
		log.Warnf("failed to check latest version - empty tag!")
		return
	}
	currentVersion := constants.VERSION
	lv, err := version.NewVersion(latestVersion)
	if err != nil {
		log.Warnf("failed to check latest version - invalid version from remote!")
		return
	}
	cv, err := version.NewVersion(currentVersion)
	if err != nil {
		log.Warnf("failed to check latest version - invalid binary version!")
		return
	}

	if cv.GreaterThanOrEqual(lv) {
		log.Info("you are running latest version")
		return
	}
	err = requireConfirmation(fmt.Sprintf("You are not running latest version of tezpay (new version : '%s', current version: '%s').\n Do you want to continue anyway?", latestVersion, currentVersion))
	if err != nil && err.Error() == "not confirmed" {
		log.Infof("You can download new version here:\n\nhttps://github.com/%s/releases\n", constants.TEZPAY_REPOSITORY)
		os.Exit(1)
	}
}
