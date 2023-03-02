package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/alis-is/jsonrpc2/endpoints"
	"github.com/alis-is/jsonrpc2/rpc"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants/enums"
)

type rwCloser struct {
	io.ReadCloser
	io.WriteCloser
}

func (rw rwCloser) Close() error {
	return errors.Join(rw.WriteCloser.Close(), rw.ReadCloser.Close())
}

type configuration struct {
	LogFile string `json:"LOG_FILE"`
}

var (
	config configuration = configuration{}
)

func appendToFile(data []byte) error {
	f, err := os.OpenFile(config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return err
	}
	return nil
}

func main() {
	endpoint := endpoints.NewStreamEndpoint(context.Background(), endpoints.NewPlainObjectStream(rwCloser{os.Stdin, os.Stdout}))

	endpoints.RegisterEndpointMethod(endpoint, string(enums.EXTENSION_INIT_CALL), func(ctx context.Context, params common.ExtensionInitializationMessage) (common.ExtensionInitializationResult, *rpc.Error) {
		def := params.Definition
		if def.Configuration == nil {
			return common.ExtensionInitializationResult{
				Success: false,
				Message: "no configuration provided",
			}, nil
		}
		err := json.Unmarshal([]byte(*def.Configuration), &config)
		if err != nil {
			return common.ExtensionInitializationResult{
				Success: false,
				Message: "invalid configuration provided",
			}, nil
		}

		return common.ExtensionInitializationResult{
			Success: true,
		}, nil
	})

	endpoints.RegisterEndpointMethod(endpoint, string(enums.EXTENSION_HOOK_TEST_NOTIFY), func(ctx context.Context, params common.ExtensionHookData[interface{}]) (interface{}, *rpc.Error) {
		return params.Data, nil
	})

	type testHookData struct {
		Message string `json:"message"`
	}
	endpoints.RegisterEndpointMethod(endpoint, string(enums.EXTENSION_HOOK_TEST_REQUEST), func(ctx context.Context, params common.ExtensionHookData[testHookData]) (*testHookData, *rpc.Error) {
		data := params.Data
		data.Message = "Hello from GO!"
		return data, nil
	})

	for _, hook := range enums.SUPPORTED_EXTENSION_HOOKS {
		h := hook
		endpoints.RegisterEndpointMethod(endpoint, string(hook), func(ctx context.Context, params common.ExtensionHookData[interface{}]) (interface{}, *rpc.Error) {
			messageData, err := json.Marshal(params)
			if err != nil {
				return nil, rpc.NewInternalErrorWithData(err.Error())
			}
			appendToFile([]byte(fmt.Sprintf("%s: %s\n", string(h), string(messageData))))
			return params.Data, nil
		})
	}

	closeChannel := make(chan struct{})
	endpoints.RegisterEndpointMethod(endpoint, string(enums.EXTENSION_CLOSE_CALL), func(ctx context.Context, params interface{}) (interface{}, *rpc.Error) {
		close(closeChannel)
		return nil, nil
	})
	<-closeChannel

}
