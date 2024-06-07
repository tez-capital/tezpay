package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/utils"
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
		return constants.ErrUserNotConfirmed
	}
	return nil
}

func assertRequireConfirmation(msg string) {
	assertRunWithParam(requireConfirmation, msg, EXIT_OPERTION_CANCELED)
}

func checkForNewVersionAvailable() (bool, string) {
	log.Debugf("checking for new version")
	// https://api.github.com/repos/tez-capital/tezpay/releases/latest
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", constants.TEZPAY_REPOSITORY))
	if err != nil {
		log.Debugf("Failed to check latest version!")
		return false, ""
	}
	var info versionInfo
	err = json.NewDecoder(resp.Body).Decode(&info)
	if err != nil {
		log.Debugf("Failed to check latest version!")
		return false, ""
	}
	latestVersion := info.Version
	if latestVersion == "" {
		log.Debugf("failed to check latest version - empty tag!")
		return false, ""
	}

	lv, err := version.NewVersion(latestVersion)
	if err != nil {
		log.Debugf("failed to check latest version - invalid version from remote!")
		return false, ""
	}
	cv, err := version.NewVersion(constants.VERSION)
	if err != nil {
		log.Debugf("failed to check latest version - invalid binary version!")
		return false, ""
	}

	if cv.GreaterThanOrEqual(lv) {
		log.Debugf("you are running latest version")
		return false, ""
	}
	log.Infof("new version available: %s", latestVersion)
	return true, latestVersion
}

func promptIfNewVersionAvailable() {
	if available, latestVersion := checkForNewVersionAvailable(); available {
		err := requireConfirmation(fmt.Sprintf("You are not running latest version of tezpay (new version : '%s', current version: '%s').\n Do you want to continue anyway?", latestVersion, constants.VERSION))
		if errors.Is(err, constants.ErrUserNotConfirmed) {
			log.Infof("You can download new version here:\n\nhttps://github.com/%s/releases\n", constants.TEZPAY_REPOSITORY)
			os.Exit(1)
		}
	}
}
