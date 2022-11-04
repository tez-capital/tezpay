package state

import (
	"github.com/alis-is/tezpay/core/common"
)

var (
	Global State
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

func (state *State) GetIsInDebugMode() bool {
	return state.debug
}
