package extension

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/alis-is/jsonrpc2/endpoints"
	"github.com/alis-is/jsonrpc2/rpc"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/echa/log"
	"github.com/google/uuid"
)

type ExtensionStore struct {
	id         uuid.UUID
	extensions []Extension
}

var (
	extensionStore ExtensionStore
)

func InitializeExtensionStore(ctx context.Context, es []common.ExtensionDefinition) error {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return err
	}

	extensions := make([]Extension, 0, len(es))
	for _, def := range es {
		ext, err := RegisterExtension(ctx, def)
		if err != nil {
			return err
		}
		extensions = append(extensions, ext)
	}
	extensionStore = ExtensionStore{
		id:         uuid,
		extensions: extensions,
	}
	return nil
}

// sends close to all scoped extensions
func EndExecutionScope() {
	for _, ext := range extensionStore.extensions {
		if ext.IsLoaded() && ext.GetDefinition().GetLifespan() == enums.EXTENSION_LIFESPAN_SCOPED {
			ext.Close()
		}
	}
}

func ExecuteHook[TData rpc.ResultType](hook enums.EExtensionHook, version string, data *TData) error {
	if data == nil {
		return errors.New("no data forwarded to hook, cannot execute")
	}

	for _, ext := range extensionStore.extensions {
		def := ext.GetDefinition()
		matchedMode := enums.EXTENSION_HOOK_MODE_UNKNOWN
		for _, h := range def.Hooks {
			if h.Id == enums.EExtensionHook(hook) {
				matchedMode = h.Mode
				break
			}
			if h.Id == enums.EXTENSION_HOOK_ALL {
				matchedMode = h.Mode
				// we do not break here in order to allow more specific hooks to override
			}
		}

		// for testing purposes
		if hook == enums.EXTENSION_HOOK_TEST_NOTIFY {
			matchedMode = enums.EXTENSION_HOOK_MODE_READ_ONLY
		}
		if hook == enums.EXTENSION_HOOK_TEST_REQUEST {
			matchedMode = enums.EXTENSION_HOOK_MODE_READ_WRITE
		}
		log.Debugf("executing hook %s with mode %s on extension %s", hook, matchedMode, def.Id)

		var err error
		for i := 0; i < def.GetRetry(); i++ {
			if i > 0 {
				time.Sleep(time.Second * time.Duration(def.GetRetryDelay()))
			}
			// init or continue
			switch matchedMode {
			case enums.EXTENSION_HOOK_MODE_READ_ONLY:
				fallthrough
			case enums.EXTENSION_HOOK_MODE_READ_WRITE:
				err := LoadExtension(ext)
				if err != nil {
					return err
				}
			default:
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), ext.GetTimeout())
			defer cancel()

			switch matchedMode {
			case enums.EXTENSION_HOOK_MODE_READ_ONLY:
				err = endpoints.Notify(ctx, ext.GetEndpoint(), string(hook), common.ExtensionHookData[TData]{
					Version: version,
					Data:    data,
				})
			case enums.EXTENSION_HOOK_MODE_READ_WRITE:
				var response rpc.Response[TData]
				err = endpoints.RequestTo(ctx, ext.GetEndpoint(), string(hook), common.ExtensionHookData[TData]{
					Version: version,
					Data:    data,
				}, &response)
				responseResult, err := response.Unwrap()

				if err != nil && strings.Contains(err.Error(), string(rpc.MethodNotFoundKind)) {
					// extensions are not required to implement all hooks
					// so we continue if error is MethodNotFound
					err = nil
					break
				}
				if err == nil {
					*data = responseResult
				}
			default:
				// no hook matched
				continue
			}
			if err == nil {
				break
			}
		}
		if err != nil {
			switch def.ErrorAction {
			case enums.EXTENSION_ERROR_ACTION_CONTINUE:
			default:
				return err
			}
		}
	}
	return nil
}
