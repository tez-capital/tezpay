package main

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/alis-is/jsonrpc2/rpc"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/core/generate"
	"github.com/tez-capital/tezpay/extension"
	"github.com/trilitech/tzgo/tezos"
)

var (
	runtimeContext *RuntimeContext = &RuntimeContext{}
)

type rwCloser struct {
	io.ReadCloser
	io.WriteCloser
}

func (rw rwCloser) Close() error {
	return errors.Join(rw.WriteCloser.Close(), rw.ReadCloser.Close())
}

// func appendToFile(data []byte) error {
// 	f, err := os.OpenFile("payouts-fa.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
// 	if err != nil {
// 		return err
// 	}
// 	defer f.Close()
// 	if _, err := f.Write(data); err != nil {
// 		return err
// 	}
// 	return nil
// }

// func main() {
// 	apiKey := "2c080226-b70b-44e4-8b4c-f9efcd8b985a"
// 	tokenSlug := "bitcoin"
// 	exchangeFee := 0.02
// 	exchange_rate, _ := get_cmc_exchange_rate(tokenSlug, exchangeFee, apiKey)
// 	fmt.Printf("Exchange rate: %f\n", exchange_rate)
// 	fmt.Println(120000 * exchange_rate)
// }

func main() {
	endpoint := extension.NewStreamEndpoint(context.Background(), extension.NewPlainObjectStream(rwCloser{os.Stdin, os.Stdout}))

	extension.RegisterEndpointMethod(endpoint, string(enums.EXTENSION_INIT_CALL), func(ctx context.Context, params common.ExtensionInitializationMessage) (common.ExtensionInitializationResult, *rpc.Error) {
		var err error
		if runtimeContext, err = Initialize(*params.Definition.Configuration); err != nil {
			return common.ExtensionInitializationResult{
				Success: false,
				Message: err.Error(),
			}, nil
		}

		return common.ExtensionInitializationResult{
			Success: true,
		}, nil
	})

	extension.RegisterEndpointMethod(endpoint, string(enums.EXTENSION_HOOK_AFTER_BONDS_DISTRIBUTED), func(ctx context.Context, params common.ExtensionHookData[generate.AfterBondsDistributedHookData]) (*generate.AfterBondsDistributedHookData, *rpc.Error) {
		extra := make([]generate.PayoutCandidateWithBondAmount, 0, len(params.Data.Candidates))

		err := runtimeContext.ExchangeRateProvider.RefreshExchangeRate()
		if err != nil {
			return nil, rpc.NewServerError(1000)
		}

		for _, candidate := range params.Data.Candidates {
			if candidate.GetAmount().IsLessEqual(tezos.Zero) {
				continue
			}

			tokenAmount := runtimeContext.ExchangeRateProvider.ExchangeToToken(candidate.GetAmount().Int64())
			if tokenAmount <= 0 {
				continue
			}

			txKind := enums.PAYOUT_TX_KIND_FA2
			switch runtimeContext.TokenConfiguration.Kind {
			case TokenKindFA1_2:
				txKind = enums.PAYOUT_TX_KIND_FA1_2
			}

			switch runtimeContext.RewardMode {
			case RewardModeBonus:
				bonusTx := generate.PayoutCandidateWithBondAmount{
					PayoutCandidate: candidate.PayoutCandidate,
					BondsAmount:     tezos.Zero,
					TxKind:          txKind,
					FATokenId:       tezos.NewZ(runtimeContext.TokenConfiguration.Id),
					FAContract:      tezos.MustParseAddress(runtimeContext.TokenConfiguration.Contract),
				}
				bonusTx.BondsAmount = tezos.NewZ(tokenAmount)

				extra = append(extra, bonusTx)
			case RewardModeReplace:
				candidate.BondsAmount = tezos.NewZ(tokenAmount)
				candidate.TxKind = txKind
				candidate.FATokenId = tezos.NewZ(runtimeContext.TokenConfiguration.Id)
				candidate.FAContract = tezos.MustParseAddress(runtimeContext.TokenConfiguration.Contract)
			}
		}

		rewards := append(params.Data.Candidates, extra...)
		params.Data.Candidates = rewards
		return params.Data, nil
	})

	extension.RegisterEndpointMethod(endpoint, string(enums.EXTENSION_HOOK_CHECK_BALANCE), func(ctx context.Context, params common.ExtensionHookData[generate.CheckBalanceHookData]) (*generate.CheckBalanceHookData, *rpc.Error) {
		// TODO:
		return params.Data, rpc.NewInternalError()
	})

	extension.RegisterEndpointMethod(endpoint, string(enums.EXTENSION_HOOK_TEST_NOTIFY), func(ctx context.Context, params common.ExtensionHookData[any]) (any, *rpc.Error) {
		return params.Data, nil
	})

	type testHookData struct {
		Message string `json:"message"`
	}
	extension.RegisterEndpointMethod(endpoint, string(enums.EXTENSION_HOOK_TEST_REQUEST), func(ctx context.Context, params common.ExtensionHookData[testHookData]) (*testHookData, *rpc.Error) {
		data := params.Data
		data.Message = "Hello from FA extension!"
		return data, nil
	})

	// extension.RegisterEndpointMethod(endpoint, string(enums.EXTENSION_HOOK_AFTER_PAYOUTS_PREPARED), func(ctx context.Context, params common.ExtensionHookData[prepare.AfterPayoutsPreapered]) (any, *rpc.Error) {
	// 	data := params.Data
	// 	newValidPayments := data.ValidPayouts
	// 	for _, report := range data.ReportsOfPastSuccesfulPayouts {
	// 		for _, recipe := range data.Recipes {
	// 			if recipe.Kind == enums.PAYOUT_KIND_DONATION {
	// 				continue
	// 			}
	// 			if report.Baker == recipe.Baker && report.Cycle == recipe.Cycle && report.FAContract == recipe.FAContract && report.FATokenId.Equal(recipe.FATokenId) && report.Delegator == recipe.Delegator && report.Kind == recipe.Kind && report.TxKind == recipe.TxKind {
	// 				if report.Amount.IsLess(recipe.Amount) {
	// 					appendToFile([]byte(fmt.Sprintf("injecting transaction fix for %s for extra %d\n", recipe.Delegator.String(), recipe.Amount.Sub(report.Amount).Int64())))
	// 					recipe.Amount = recipe.Amount.Sub(report.Amount)
	// 					recipe.Kind = recipe.Kind + " (fix)"
	// 					newValidPayments = append(newValidPayments, recipe)
	// 				}
	// 			}
	// 		}
	// 	}

	// 	data.ValidPayouts = newValidPayments
	// 	return data, nil
	// })

	closeChannel := make(chan struct{})
	extension.RegisterEndpointMethod(endpoint, string(enums.EXTENSION_CLOSE_CALL), func(ctx context.Context, params any) (any, *rpc.Error) {
		close(closeChannel)
		return nil, nil
	})
	<-closeChannel

}
