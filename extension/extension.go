package extension

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/constants/enums"
)

type Extension interface {
	IsLoaded() bool
	GetEndpoint() EndpointClient
	Load() error
	Close() error
	GetDefinition() common.ExtensionDefinition
	GetTimeout() time.Duration
}

func RegisterExtension(ctx context.Context, def common.ExtensionDefinition) (Extension, error) {
	if def.Kind == enums.EXTENSION_STDIO_RPC {
		slog.Info("initializing extension", "kind", def.Kind, "command", def.Command)
	} else {
		slog.Info("initializing extension", "kind", def.Kind, "url", def.Url)
	}

	switch def.Kind {
	case enums.EXTENSION_STDIO_RPC:
		return newStdioExtension(ctx, def), nil
	case enums.EXTENSION_TCP_RPC:
		return newTcpExtension(ctx, def), nil
		// TODO: http and WS
	default:
		return nil, errors.Join(constants.ErrUnsupportedExtensionKind, fmt.Errorf("kind - \"%s\"", def.Kind))
	}
}

func LoadExtension(ext Extension) error {
	if ext.IsLoaded() {
		return nil
	}
	def := ext.GetDefinition()
	slog.Debug("loading extension", "name", def.Name, "command", def.Command, "url", def.Url)
	if err := ext.Load(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), ext.GetTimeout())
	defer cancel()
	response, err := Request[common.ExtensionInitializationMessage, common.ExtensionInitializationResult](ctx, ext.GetEndpoint(), string(enums.EXTENSION_INIT_CALL), common.ExtensionInitializationMessage{
		OwnerId:    extensionStore.id.String(),
		Definition: ext.GetDefinition(),
		BakerPKH:   extensionStore.environment.BakerPKH,
		PayoutPKH:  extensionStore.environment.PayoutPKH,
		RpcPool:    extensionStore.environment.RpcPool,
	})
	if err != nil {
		return err
	}
	result, err := response.Unwrap()
	if err != nil {
		return err
	}
	if result.Success {
		return nil
	}
	return errors.Join(constants.ErrExtensionLoadFailed, errors.New(result.Message))
}

type ExtensionBase struct {
	definition common.ExtensionDefinition
	endpoint   EndpointClient
	loaded     bool
}

func (e *ExtensionBase) GetDefinition() common.ExtensionDefinition {
	return e.definition
}

func (e *ExtensionBase) GetEndpoint() EndpointClient {
	return e.endpoint
}

func (e *ExtensionBase) GetTimeout() time.Duration {
	timeout := e.definition.Timeout
	if timeout == nil || *timeout <= 0 {
		return time.Minute * 1
	}
	return time.Duration(*timeout) * time.Second
}
