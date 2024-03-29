package extension

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
	log "github.com/sirupsen/logrus"
)

type Extension interface {
	IsLoaded() bool
	GetEndpoint() IEndpointClient
	Load() error
	Close() error
	GetDefinition() common.ExtensionDefinition
	GetTimeout() time.Duration
}

func RegisterExtension(ctx context.Context, def common.ExtensionDefinition) (Extension, error) {
	if def.Kind == enums.EXTENSION_STDIO_RPC {
		log.Infof("Initialization of extension %s (kind '%s')", def.Command, def.Kind)
	} else {
		log.Infof("Initialization of extension %s (kind '%s')", def.Url, def.Kind)
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
	log.Tracef("loading extension - %s#%s@%s", def.Name, def.Command, def.Url)
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
	endpoint   IEndpointClient
	loaded     bool
}

func (e *ExtensionBase) GetDefinition() common.ExtensionDefinition {
	return e.definition
}

func (e *ExtensionBase) GetEndpoint() IEndpointClient {
	return e.endpoint
}

func (e *ExtensionBase) GetTimeout() time.Duration {
	timeout := e.definition.Timeout
	if timeout == nil || *timeout <= 0 {
		return time.Minute * 1
	}
	return time.Duration(*timeout) * time.Second
}
