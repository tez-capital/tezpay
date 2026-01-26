package generate

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
)

const KILL_SWITCH_DETECTED_MESSAGE = "kill switch detected, please check TzC support channels for more information"

func checkKillSwitch(ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) (*PayoutGenerationContext, error) {
	configuration := ctx.GetConfiguration()

	if os.Getenv("DISABLE_TEZPAY_KILL_SWITCH") == "true" {
		return ctx, nil
	}

	if configuration.DisableKillSwitch {
		return ctx, nil
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// there are 2 types of kill switches
	// - both hosted at https://raw.githubusercontent.com/tez-capital/tezpay/refs/heads/main/

	// 1. DO_NOT_PAY
	// - https://raw.githubusercontent.com/tez-capital/tezpay/refs/heads/main/DO_NOT_PAY
	// - content of the file does not matter, the file just needs to exist to trigger the kill switch
	resp, err := client.Get("https://raw.githubusercontent.com/tez-capital/tezpay/refs/heads/main/DO_NOT_PAY")
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return ctx, errors.New(KILL_SWITCH_DETECTED_MESSAGE)
		}
	}

	// 2. UPGRADE_REQUIRED
	// - https://raw.githubusercontent.com/tez-capital/tezpay/refs/heads/main/UPGRADE_REQUIRED
	// - the version will be the raw content of the file and is supposed to be compared against constants.VERSION
	resp, err = client.Get("https://raw.githubusercontent.com/tez-capital/tezpay/refs/heads/main/UPGRADE_REQUIRED")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return ctx, fmt.Errorf("kill switch check failed: found UPGRADE_REQUIRED but failed to read body: %w", err)
			}

			requiredVersionStr := strings.TrimSpace(string(body))
			if requiredVersionStr == "" {
				return ctx, errors.New(KILL_SWITCH_DETECTED_MESSAGE)
			}

			if constants.VERSION == "dev" {
				return ctx, nil
			}

			currentVersion, err := version.NewVersion(constants.VERSION)
			if err != nil {
				return ctx, fmt.Errorf("kill switch check failed: invalid current version format '%s': %w", constants.VERSION, err)
			}

			requiredVersion, err := version.NewVersion(requiredVersionStr)
			if err != nil {
				return ctx, fmt.Errorf("kill switch check failed: found UPGRADE_REQUIRED but failed to parse version '%s': %w", requiredVersionStr, err)
			}

			if currentVersion.LessThan(requiredVersion) {
				return ctx, fmt.Errorf("kill switch activated: upgrade required (current: %s, required: %s)", constants.VERSION, requiredVersionStr)
			}
		}
	}

	return ctx, nil
}

func CheckConditionsAndPrepare(ctx *PayoutGenerationContext, options *common.GeneratePayoutsOptions) (*PayoutGenerationContext, error) {
	ctx, err := checkKillSwitch(ctx, options)
	if err != nil {
		return ctx, err
	}

	collector := ctx.GetCollector()
	logger := ctx.logger.With("phase", "check_conditions_and_prepare")
	logger.Info("checking conditions and preparing")
	logger.Debug("checking if payout address is revealed")
	payoutAddress := ctx.PayoutKey.Address()
	revealed, err := collector.IsRevealed(payoutAddress)
	if err != nil {
		return ctx, errors.Join(constants.ErrRevealCheckFailed, fmt.Errorf("address - %s", payoutAddress), err)
	}
	if !revealed {
		return ctx, errors.Join(constants.ErrNotRevealed, fmt.Errorf("address - %s", payoutAddress))
	}

	return ctx, nil
}
