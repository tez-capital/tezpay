package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/hashicorp/go-version"
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
	assertRunWithParamAndErrorMessage(requireConfirmation, msg, EXIT_OPERTION_CANCELED, "not confirmed")
}

func checkForNewVersionAvailable() (bool, string) {
	slog.Debug("checking for new version")
	// https://api.github.com/repos/tez-capital/tezpay/releases/latest
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", constants.TEZPAY_REPOSITORY))
	if err != nil {
		slog.Debug("⚠️ failed to check latest version", "error", err.Error())
		return false, ""
	}
	var info versionInfo
	err = json.NewDecoder(resp.Body).Decode(&info)
	if err != nil {
		slog.Debug("⚠️ failed to check latest version", "error", err.Error())
		return false, ""
	}
	latestVersion := info.Version
	if latestVersion == "" {
		slog.Debug("⚠️ failed to check latest version", "error", "empty tag")
		return false, ""
	}

	lv, err := version.NewVersion(latestVersion)
	if err != nil {
		slog.Debug("⚠️ failed to check latest version", "error", err.Error())
		return false, ""
	}
	cv, err := version.NewVersion(constants.VERSION)
	if err != nil {
		slog.Debug("⚠️ failed to check latest version", "error", err.Error())
		return false, ""
	}

	if cv.GreaterThanOrEqual(lv) {
		slog.Debug("running the latest version")
		return false, ""
	}
	slog.Info("new version available", "version", latestVersion)
	return true, latestVersion
}

func promptIfNewVersionAvailable() {
	if available, latestVersion := checkForNewVersionAvailable(); available {
		err := requireConfirmation(fmt.Sprintf("You are not running latest version of tezpay (new version : '%s', current version: '%s').\n Do you want to continue anyway?", latestVersion, constants.VERSION))
		if errors.Is(err, constants.ErrUserNotConfirmed) {
			slog.Info("new version available", "url", fmt.Sprintf("https://github.com/%s/releases", constants.TEZPAY_REPOSITORY))
			os.Exit(1)
		}
	}
}
