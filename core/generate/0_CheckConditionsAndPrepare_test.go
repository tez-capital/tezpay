package generate

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/configuration"
	"github.com/tez-capital/tezpay/constants"
)

func TestCheckKillSwitch_DoNotPayEnabled(t *testing.T) {
	assert := assert.New(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/DO_NOT_PAY", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/UPGRADE_REQUIRED", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	oldDoNotPayURL := killSwitchDoNotPayURL
	oldUpgradeRequiredURL := killSwitchUpgradeRequiredURL
	killSwitchDoNotPayURL = server.URL + "/DO_NOT_PAY"
	killSwitchUpgradeRequiredURL = server.URL + "/UPGRADE_REQUIRED"
	t.Cleanup(func() {
		killSwitchDoNotPayURL = oldDoNotPayURL
		killSwitchUpgradeRequiredURL = oldUpgradeRequiredURL
	})

	config := configuration.GetDefaultRuntimeConfiguration()
	ctx := &PayoutGenerationContext{configuration: &config}

	_, err := checkKillSwitch(ctx, &common.GeneratePayoutsOptions{})
	assert.Error(err)
	assert.Contains(err.Error(), KILL_SWITCH_DETECTED_MESSAGE)
}

func TestCheckKillSwitch_UpgradeRequired(t *testing.T) {
	assert := assert.New(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/DO_NOT_PAY", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/UPGRADE_REQUIRED", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("9.9.9"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	oldVersion := constants.VERSION
	oldDoNotPayURL := killSwitchDoNotPayURL
	oldUpgradeRequiredURL := killSwitchUpgradeRequiredURL
	constants.VERSION = "0.25.0"
	killSwitchDoNotPayURL = server.URL + "/DO_NOT_PAY"
	killSwitchUpgradeRequiredURL = server.URL + "/UPGRADE_REQUIRED"
	t.Cleanup(func() {
		constants.VERSION = oldVersion
		killSwitchDoNotPayURL = oldDoNotPayURL
		killSwitchUpgradeRequiredURL = oldUpgradeRequiredURL
	})

	config := configuration.GetDefaultRuntimeConfiguration()
	ctx := &PayoutGenerationContext{configuration: &config}

	_, err := checkKillSwitch(ctx, &common.GeneratePayoutsOptions{})
	assert.Error(err)
	assert.Contains(err.Error(), "kill switch activated: upgrade required")
}

func TestCheckKillSwitch_FailsClosedOnNetworkError(t *testing.T) {
	assert := assert.New(t)

	oldDoNotPayURL := killSwitchDoNotPayURL
	oldUpgradeRequiredURL := killSwitchUpgradeRequiredURL
	killSwitchDoNotPayURL = "http://127.0.0.1:1/DO_NOT_PAY"
	killSwitchUpgradeRequiredURL = "http://127.0.0.1:1/UPGRADE_REQUIRED"
	t.Cleanup(func() {
		killSwitchDoNotPayURL = oldDoNotPayURL
		killSwitchUpgradeRequiredURL = oldUpgradeRequiredURL
	})

	config := configuration.GetDefaultRuntimeConfiguration()
	ctx := &PayoutGenerationContext{configuration: &config}

	_, err := checkKillSwitch(ctx, &common.GeneratePayoutsOptions{})
	assert.Error(err)
	assert.Contains(err.Error(), "kill switch check failed: could not fetch DO_NOT_PAY")
}

func TestCheckKillSwitch_IgnoreWhenDisabledInConfig(t *testing.T) {
	assert := assert.New(t)

	oldDoNotPayURL := killSwitchDoNotPayURL
	oldUpgradeRequiredURL := killSwitchUpgradeRequiredURL
	killSwitchDoNotPayURL = "http://127.0.0.1:1/DO_NOT_PAY"
	killSwitchUpgradeRequiredURL = "http://127.0.0.1:1/UPGRADE_REQUIRED"
	t.Cleanup(func() {
		killSwitchDoNotPayURL = oldDoNotPayURL
		killSwitchUpgradeRequiredURL = oldUpgradeRequiredURL
	})

	config := configuration.GetDefaultRuntimeConfiguration()
	config.DisableKillSwitch = true
	ctx := &PayoutGenerationContext{configuration: &config}

	_, err := checkKillSwitch(ctx, &common.GeneratePayoutsOptions{})
	assert.NoError(err)
}

func TestCheckKillSwitch_FailsOnUnexpectedDoNotPayStatusCode(t *testing.T) {
	assert := assert.New(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/DO_NOT_PAY", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	mux.HandleFunc("/UPGRADE_REQUIRED", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	oldDoNotPayURL := killSwitchDoNotPayURL
	oldUpgradeRequiredURL := killSwitchUpgradeRequiredURL
	killSwitchDoNotPayURL = server.URL + "/DO_NOT_PAY"
	killSwitchUpgradeRequiredURL = server.URL + "/UPGRADE_REQUIRED"
	t.Cleanup(func() {
		killSwitchDoNotPayURL = oldDoNotPayURL
		killSwitchUpgradeRequiredURL = oldUpgradeRequiredURL
	})

	config := configuration.GetDefaultRuntimeConfiguration()
	ctx := &PayoutGenerationContext{configuration: &config}

	_, err := checkKillSwitch(ctx, &common.GeneratePayoutsOptions{})
	assert.Error(err)
	assert.Contains(err.Error(), "failed to determine DO_NOT_PAY status")
}

func TestCheckKillSwitch_FailsOnUnexpectedUpgradeRequiredStatusCode(t *testing.T) {
	assert := assert.New(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/DO_NOT_PAY", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/UPGRADE_REQUIRED", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	oldDoNotPayURL := killSwitchDoNotPayURL
	oldUpgradeRequiredURL := killSwitchUpgradeRequiredURL
	killSwitchDoNotPayURL = server.URL + "/DO_NOT_PAY"
	killSwitchUpgradeRequiredURL = server.URL + "/UPGRADE_REQUIRED"
	t.Cleanup(func() {
		killSwitchDoNotPayURL = oldDoNotPayURL
		killSwitchUpgradeRequiredURL = oldUpgradeRequiredURL
	})

	config := configuration.GetDefaultRuntimeConfiguration()
	ctx := &PayoutGenerationContext{configuration: &config}

	_, err := checkKillSwitch(ctx, &common.GeneratePayoutsOptions{})
	assert.Error(err)
	assert.Contains(err.Error(), "failed to determine UPGRADE_REQUIRED status")
}
