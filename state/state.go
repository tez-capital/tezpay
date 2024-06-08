package state

import (
	"errors"
	"log/slog"
	"os"
	"path"

	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
)

var (
	Global                 *State
	CONFIG_FILE_NAME       = "config.hjson"
	PRIVATE_KEY_FILE_NAME  = "payout_wallet_private.key"
	REMOTE_SPECS_FILE_NAME = "remote_signer.hjson"
)

type StateInitOptions struct {
	WantsJsonOutput       bool
	InjectedConfiguration *string
	SignerOverride        common.SignerEngine
	Debug                 bool
	DisableDonationPrompt bool
	PayOnlyAddressPrefix  string
}

type State struct {
	workingDirectory         string
	wantsJsonOutput          bool
	injectedConfiguration    []byte
	hasInjectedConfiguration bool
	SignerOverride           common.SignerEngine
	disableDonationPrompt    bool

	payOnlyAddressPrefix string
}

func Init(workingDirectory string, options StateInitOptions) error {
	injectedConfiguration, hasInjectedConfiguration := []byte{}, false
	if options.InjectedConfiguration != nil {
		injectedConfiguration, hasInjectedConfiguration = []byte(*options.InjectedConfiguration), true
	}

	Global = &State{
		workingDirectory:         workingDirectory,
		injectedConfiguration:    injectedConfiguration,
		wantsJsonOutput:          options.WantsJsonOutput,
		hasInjectedConfiguration: hasInjectedConfiguration,
		SignerOverride:           options.SignerOverride,
		disableDonationPrompt:    options.DisableDonationPrompt,
		payOnlyAddressPrefix:     options.PayOnlyAddressPrefix,
	}

	return errors.Join(Global.validateReportsDirectory())
}

func (state *State) validateReportsDirectory() error {
	reportsDirectoryPath := state.GetReportsDirectory()
	if _, err := os.Stat(reportsDirectoryPath); os.IsNotExist(err) {
		slog.Debug("reports directory does not exist, creating", "path", reportsDirectoryPath)
		if err := os.Mkdir(reportsDirectoryPath, 0755); err != nil {
			return err
		}
	}
	// write test file
	testFilePath := path.Join(reportsDirectoryPath, ".test")
	slog.Debug("writing test file to reports directory", "path", testFilePath)
	if err := os.WriteFile(testFilePath, []byte("test"), 0644); err != nil {
		return err
	}
	return nil
}

func (state *State) GetWorkingDirectory() string {
	return state.workingDirectory
}

func (state *State) GetWantsOutputJson() bool {
	return state.wantsJsonOutput
}

func (state *State) GetInjectedConfiguration() (bool, []byte) {
	return state.hasInjectedConfiguration, state.injectedConfiguration
}

func (state *State) GetReportsDirectory() string {
	reportsDirectoryPath := os.Getenv("REPORTS_DIRECTORY")
	if reportsDirectoryPath != "" {
		return reportsDirectoryPath
	}
	return path.Join(state.GetWorkingDirectory(), constants.REPORTS_DIRECTORY)
}

func (state *State) GetConfigurationFilePath() string {
	configurationFilePath := os.Getenv("CONFIGURATION_FILE")
	if configurationFilePath != "" {
		return configurationFilePath
	}
	return path.Join(state.GetWorkingDirectory(), CONFIG_FILE_NAME)
}

func (state *State) GetPrivateKeyFilePath() string {
	privateKeyFilePath := os.Getenv("PRIVATE_KEY_FILE")
	if privateKeyFilePath != "" {
		return privateKeyFilePath
	}
	return path.Join(state.GetWorkingDirectory(), PRIVATE_KEY_FILE_NAME)
}

func (state *State) GetRemoteSpecsFilePath() string {
	remoteSpecsConfigurationFile := os.Getenv("REMOTE_SIGNER_CONFIGURATION_FILE")
	if remoteSpecsConfigurationFile != "" {
		return remoteSpecsConfigurationFile
	}
	return path.Join(state.GetWorkingDirectory(), REMOTE_SPECS_FILE_NAME)
}

func (state *State) GetPayOnlyAddressPrefix() string {
	return state.payOnlyAddressPrefix
}

func (state *State) IsDonationPromptDisabled() bool {
	return state.disableDonationPrompt
}
