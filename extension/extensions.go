package extension

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/alis-is/jsonrpc2/rpc"
	"github.com/google/uuid"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/constants/enums"
)

type ExtensionStoreEnviromnent struct {
	BakerPKH  string   `json:"baker_pkh"`
	PayoutPKH string   `json:"payout_pkh"`
	RpcPool   []string `json:"rpc_pool"`
}

type ExtensionStore struct {
	id          uuid.UUID
	extensions  []Extension
	environment *ExtensionStoreEnviromnent
}

var (
	extensionStore     ExtensionStore
	IsStoreInitialized bool
)

func InitializeExtensionStore(ctx context.Context, es []common.ExtensionDefinition, env *ExtensionStoreEnviromnent) error {
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
		id:          uuid,
		extensions:  extensions,
		environment: env,
	}
	IsStoreInitialized = true
	return nil
}

func closeExtension(ext Extension) {
	if ext.IsLoaded() {
		Notify[any](context.Background(), ext.GetEndpoint(), string(enums.EXTENSION_CLOSE_CALL), nil)
		ext.Close()
	}
}

// sends close to all scoped extensions
func CloseScopedExtensions() {
	for _, ext := range extensionStore.extensions {
		if ext.GetDefinition().GetLifespan() == enums.EXTENSION_LIFESPAN_SCOPED {
			closeExtension(ext)
		}
	}
}

func CloseExtensions() {
	for _, ext := range extensionStore.extensions {
		closeExtension(ext)
	}
}

func ExecuteHook[TData rpc.ResultType](hook enums.EExtensionHook, version string, data *TData) error {
	if data == nil {
		return constants.ErrExtensionHookMissingData
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
				err = LoadExtension(ext)
				if err != nil {
					if ext.GetDefinition().Name != "" {
						return fmt.Errorf("failed to load extension %s: %w", ext.GetDefinition().Name, err)
					}
					return err
				}
			default:
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), ext.GetTimeout())
			defer cancel()

			switch matchedMode {
			case enums.EXTENSION_HOOK_MODE_READ_ONLY:
				slog.Debug("executing hook", "hook", hook, "mode", matchedMode, "extension", def.Name)
				err = Notify(ctx, ext.GetEndpoint(), string(hook), common.ExtensionHookData[TData]{
					Version: version,
					Data:    data,
				})
			case enums.EXTENSION_HOOK_MODE_READ_WRITE:
				slog.Debug("executing hook", "hook", hook, "mode", matchedMode, "extension", def.Name)
				var response rpc.Response[TData]
				err = RequestTo(ctx, ext.GetEndpoint(), string(hook), common.ExtensionHookData[TData]{
					Version: version,
					Data:    data,
				}, &response)
				if err == nil {
					var responseResult TData
					responseResult, err = response.Unwrap()

					if err != nil && strings.Contains(err.Error(), string(rpc.MethodNotFoundKind)) {
						// extensions are not required to implement all hooks
						// so we continue if error is MethodNotFound
						err = nil
						break
					}
					if err == nil {
						*data = responseResult
					}
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
				if ext.GetDefinition().Name != "" {
					return fmt.Errorf("[%s] %w", ext.GetDefinition().Name, err)
				}
				return err
			}
		}
	}
	return nil
}
