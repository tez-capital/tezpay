package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"

	rpc "github.com/alis-is/jsonrpc2"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/core/generate"
	"github.com/tez-capital/tezpay/core/prepare"
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

func main() {
	endpoint := extension.NewStreamEndpoint(context.Background(), extension.NewPlainObjectStream(rwCloser{os.Stdin, os.Stdout}))

	extension.RegisterEndpointMethod(endpoint, string(enums.EXTENSION_INIT_CALL), func(ctx context.Context, params common.ExtensionInitializationMessage) (common.ExtensionInitializationResult, *rpc.Error) {
		var err error
		if runtimeContext, err = Initialize(ctx, params); err != nil {
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
		tokenTxs := make([]generate.PayoutCandidateWithBondAmount, 0, len(params.Data.Candidates))
		slog.Info("calculating token rewards", "candidates", len(params.Data.Candidates))
		err := runtimeContext.ExchangeRateProvider.RefreshExchangeRate()
		if err != nil {
			slog.Error("failed to refresh exchange rate", "error", err.Error())
			return nil, rpc.NewServerError(1000)
		}

		for _, candidate := range params.Data.Candidates {
			if candidate.GetAmount().IsLessEqual(tezos.Zero) {
				continue
			}

			tokenAmount := runtimeContext.ExchangeRateProvider.ExchangeTezToToken(candidate.GetAmount()) // we calculated with mutez so we need to divide by 1_000_000
			slog.Debug("calculated token reward", "delegate", candidate.Source, "recipient", candidate.Recipient, "amount", tokenAmount)
			if tokenAmount.IsLessEqual(tezos.Zero) || tokenAmount.IsLess(runtimeContext.MinimumTokenAmount) {
				continue
			}

			txKind := enums.PAYOUT_TX_KIND_FA2
			switch runtimeContext.TokenConfiguration.Kind {
			case TokenKindFA1_2:
				txKind = enums.PAYOUT_TX_KIND_FA1_2
			}

			tokenTxs = append(tokenTxs, generate.PayoutCandidateWithBondAmount{
				PayoutCandidate: candidate.PayoutCandidate,
				BondsAmount:     tokenAmount,
				TxKind:          txKind,
				FATokenId:       tezos.NewZ(runtimeContext.TokenConfiguration.Id),
				FAContract:      tezos.MustParseAddress(runtimeContext.TokenConfiguration.Contract),
				FAAlias:         runtimeContext.TokenConfiguration.Alias,
				FADecimals:      runtimeContext.TokenConfiguration.Decimals,
			})
		}

		rewards := params.Data.Candidates
		switch runtimeContext.RewardMode {
		case RewardModeBonus:
			rewards = append(rewards, tokenTxs...)
		case RewardModeReplace:
			rewards = tokenTxs
		}
		slog.Info("successfully calculated token rewards", "rewards_count", len(rewards))
		params.Data.Candidates = rewards
		return params.Data, nil
	})

	extension.RegisterEndpointMethod(endpoint, string(enums.EXTENSION_HOOK_CHECK_BALANCE), func(ctx context.Context, params common.ExtensionHookData[prepare.CheckBalanceHookData]) (*prepare.CheckBalanceHookData, *rpc.Error) {
		total := tezos.Zero

		balance, err := runtimeContext.Contract.GetBalance(ctx)
		if err != nil {
			slog.Error("failed to get balance", "error", err.Error())
			return nil, rpc.NewServerError(1000)
		}
		slog.Info("successfully checked available token balance", "balance", balance)

		for _, candidate := range params.Data.Payouts {
			if candidate.TxKind != enums.PAYOUT_TX_KIND_FA1_2 && candidate.TxKind != enums.PAYOUT_TX_KIND_FA2 {
				continue
			}

			if !candidate.FAContract.Equal(tezos.MustParseAddress(runtimeContext.TokenConfiguration.Contract)) || !candidate.FATokenId.Equal(tezos.NewZ(runtimeContext.TokenConfiguration.Id)) {
				continue
			}

			if candidate.GetAmount().IsLessEqual(tezos.Zero) {
				continue
			}

			total = total.Add(candidate.GetAmount())
		}

		slog.Info("total amount to be paid", "total", total, "is_sufficient", total.IsLess(balance))
		if balance.IsLess(total) {
			params.Data.IsSufficient = false
			params.Data.Message = "Insufficient balance of FA tokens"
			return params.Data, nil
		}

		return params.Data, nil
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

	closeChannel := make(chan struct{})
	extension.RegisterEndpointMethod(endpoint, string(enums.EXTENSION_CLOSE_CALL), func(ctx context.Context, params any) (any, *rpc.Error) {
		close(closeChannel)
		return nil, nil
	})
	<-closeChannel
}
