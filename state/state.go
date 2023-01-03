package state

import (
	"os"
	"path"

	"github.com/alis-is/tezpay/core/common"
)

var (
	Global           State
	CONFIG_FILE_NAME = "config.hjson"
)

type StateInitOptions struct {
	WantsJsonOutput       bool
	InjectedConfiguration *string
	SignerOverride        common.SignerEngine
	Debug                 bool
}

type State struct {
	workingDirectory         string
	wantsJsonOutput          bool
	injectedConfiguration    []byte
	hasInjectedConfiguration bool
	SignerOverride           common.SignerEngine
	debug                    bool
}

func Init(workingDirectory string, options StateInitOptions) {
	injectedConfiguration, hasInjectedConfiguration := []byte{}, false
	if options.InjectedConfiguration != nil {
		injectedConfiguration, hasInjectedConfiguration = []byte(*options.InjectedConfiguration), true
	}
	Global = State{
		workingDirectory:         workingDirectory,
		injectedConfiguration:    injectedConfiguration,
		wantsJsonOutput:          options.WantsJsonOutput,
		hasInjectedConfiguration: hasInjectedConfiguration,
		SignerOverride:           options.SignerOverride,
		debug:                    options.Debug,
	}
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

func (state *State) GetConfigurationFilePath() string {
	configurationFilePath := os.Getenv("CONFIGURATION_FILE")
	if configurationFilePath != "" {
		return configurationFilePath
	}
	return path.Join(state.GetWorkingDirectory() + CONFIG_FILE_NAME)
}

func (state *State) GetConfigurationFileBackupPath() string {
	configurationFilePath := state.GetConfigurationFilePath()
	extension := path.Ext(configurationFilePath)

	return configurationFilePath[0:len(configurationFilePath)-len(extension)] + ".backup" + extension
}

func (state *State) GetIsInDebugMode() bool {
	return state.debug
}
