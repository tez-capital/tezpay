package common

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tez-capital/tezpay/constants/enums"
)

type extensionHook struct {
	Id   enums.EExtensionHook     `json:"id"`
	Mode enums.EExtensionHookMode `json:"mode"`
}

type ExtensionHook struct {
	Id   enums.EExtensionHook     `json:"id"`
	Mode enums.EExtensionHookMode `json:"mode"`
}

func (h *ExtensionHook) UnmarshalJSON(b []byte) error {
	raw := extensionHook{
		Id:   enums.EXTENSION_HOOK_UNKNOWN,
		Mode: enums.EXTENSION_HOOK_MODE_UNKNOWN,
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		split := strings.Split(s, ":")
		switch len(split) {
		case 1:
			split = append(split, string(enums.EXTENSION_HOOK_MODE_UNKNOWN))
		case 2:
			break
		default:
			return err
		}
		raw.Id = enums.EExtensionHook(strings.Trim(split[0], " "))
		raw.Mode = enums.EExtensionHookMode(strings.Trim(split[1], " "))
	}
	// we could validate here but validation during unmarshalling does not
	// provide that good user experience as validation during configuration validation
	h.Id = raw.Id
	h.Mode = raw.Mode
	return nil
}

func (h *ExtensionHook) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s:%s\"", h.Id, h.Mode)), nil
}

type ExtensionHookData[TData any] struct {
	Version string `json:"version"`
	Data    *TData `json:"data"`
}

type ExtensionDefinition struct {
	// name is optional, it is used for debugging purposes only
	Name    string                  `json:"name"`
	Command string                  `json:"command,omitempty"`
	Args    []string                `json:"args,omitempty"`
	Url     string                  `json:"url,omitempty"`
	Kind    enums.EExtensionRpcKind `json:"kind"`
	// configuration is passed through to the extension as is
	Configuration *json.RawMessage            `json:"configuration,omitempty"`
	Hooks         []ExtensionHook             `json:"hooks,omitempty"`
	Lifespan      *enums.EExtensionLifespan   `json:"lifespan,omitempty"`
	ErrorAction   enums.EExtensionErrorAction `json:"error_action,omitempty"`
	Retry         *int                        `json:"retry,omitempty"`
	RetryDelay    *int                        `json:"retry_delay,omitempty"`
	Timeout       *int                        `json:"timeout,omitempty"`
	WaitForStart  int                         `json:"wait_for_start,omitempty"`
}

func (d ExtensionDefinition) GetLifespan() enums.EExtensionLifespan {
	if d.Lifespan != nil {
		return *d.Lifespan
	}
	switch d.Kind {
	case enums.EXTENSION_STDIO_RPC:
		return enums.EXTENSION_LIFESPAN_SCOPED
	default:
		return enums.EXTENSION_LIFESPAN_TRANSIENT
	}
}

func (d ExtensionDefinition) GetRetry() int {
	if d.Retry != nil {
		return *d.Retry
	}
	return 2
}

func (d ExtensionDefinition) GetRetryDelay() int {
	if d.RetryDelay != nil {
		return *d.RetryDelay
	}
	return 5
}

type ExtensionInitializationResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ExtensionInitializationMessage struct {
	// the id of the extension store
	OwnerId    string              `json:"owner_id"`
	BakerPKH   string              `json:"baker_pkh"`
	PayoutPKH  string              `json:"payout_pkh"`
	Definition ExtensionDefinition `json:"definition"`
}
