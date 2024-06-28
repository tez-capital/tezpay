package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/alis-is/jsonrpc2/rpc"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/core/prepare"
	"github.com/tez-capital/tezpay/extension"
)

type rwCloser struct {
	io.ReadCloser
	io.WriteCloser
}

func (rw rwCloser) Close() error {
	return errors.Join(rw.WriteCloser.Close(), rw.ReadCloser.Close())
}

func appendToFile(data []byte) error {
	f, err := os.OpenFile("tezpay-fixer.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
	endpoint := extension.NewStreamEndpoint(context.Background(), extension.NewPlainObjectStream(rwCloser{os.Stdin, os.Stdout}))

	extension.RegisterEndpointMethod(endpoint, string(enums.EXTENSION_INIT_CALL), func(ctx context.Context, params common.ExtensionInitializationMessage) (common.ExtensionInitializationResult, *rpc.Error) {
		return common.ExtensionInitializationResult{
			Success: true,
		}, nil
	})

	extension.RegisterEndpointMethod(endpoint, string(enums.EXTENSION_HOOK_TEST_NOTIFY), func(ctx context.Context, params common.ExtensionHookData[any]) (any, *rpc.Error) {
		return params.Data, nil
	})

	type testHookData struct {
		Message string `json:"message"`
	}
	extension.RegisterEndpointMethod(endpoint, string(enums.EXTENSION_HOOK_TEST_REQUEST), func(ctx context.Context, params common.ExtensionHookData[testHookData]) (*testHookData, *rpc.Error) {
		data := params.Data
		data.Message = "Hello from FIX extension!"
		return data, nil
	})

	extension.RegisterEndpointMethod(endpoint, string(enums.EXTENSION_HOOK_AFTER_PAYOUTS_PREPARED), func(ctx context.Context, params common.ExtensionHookData[prepare.AfterPayoutsPreapered]) (any, *rpc.Error) {
		data := params.Data
		newValidPayments := data.ValidPayouts
		for _, report := range data.ReportsOfPastSuccesfulPayouts {
			for _, recipe := range data.Recipes {
				if recipe.Kind == enums.PAYOUT_KIND_DONATION {
					continue
				}
				if report.Baker == recipe.Baker && report.Cycle == recipe.Cycle && report.FAContract == recipe.FAContract && report.FATokenId.Equal(recipe.FATokenId) && report.Delegator == recipe.Delegator && report.Kind == recipe.Kind && report.TxKind == recipe.TxKind {
					if report.Amount.IsLess(recipe.Amount) {
						appendToFile([]byte(fmt.Sprintf("injecting transaction fix for %s for extra %d\n", recipe.Delegator.String(), recipe.Amount.Sub(report.Amount).Int64())))
						recipe.Amount = recipe.Amount.Sub(report.Amount)
						recipe.Kind = recipe.Kind + " (fix)"
						newValidPayments = append(newValidPayments, recipe)
					}
				}
			}
		}

		data.ValidPayouts = newValidPayments
		return data, nil
	})

	closeChannel := make(chan struct{})
	extension.RegisterEndpointMethod(endpoint, string(enums.EXTENSION_CLOSE_CALL), func(ctx context.Context, params any) (any, *rpc.Error) {
		close(closeChannel)
		return nil, nil
	})
	<-closeChannel

}
