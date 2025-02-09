package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"

	"github.com/tez-capital/tezpay/common"
	"github.com/trilitech/tzgo/tezos"
	"gopkg.in/natefinch/lumberjack.v2"
)

type ExchangeRateKind string

const (
	ExchangeRateFixedAmount ExchangeRateKind = "fixed-amount"
	ExchangeRateFixed       ExchangeRateKind = "fixed"
	ExchangeRateDynamic     ExchangeRateKind = "dynamic"
)

type ExchangeRateProviderKind string

const (
	CMCExchangeRateProviderKind ExchangeRateProviderKind = "cmc"
)

type CMCExchangeRateProviderConfiguration struct {
	Slug   string `json:"slug"`
	APIKey string `json:"api_key"`
}

type RewardMode string

const (
	RewardModeBonus   RewardMode = "bonus"
	RewardModeReplace RewardMode = "replace"
)

type TokenKind string

const (
	TokenKindFA1_2 TokenKind = "fa1.2"
	TokenKindFA2   TokenKind = "fa2"
)

type TokenConfiguration struct {
	Id       int64     `json:"id"`
	Contract string    `json:"contract"`
	Decimals int64     `json:"decimals"`
	Kind     TokenKind `json:"kind"`
	Alias    string    `json:"alias,omitempty"`
}

type Configuration struct {
	ExchangeRateKind                  ExchangeRateKind         `json:"exchange_rate_kind,omitempty"`
	RewardAmount                      int64                    `json:"reward_amount,omitempty"`
	BalanceRequired                   int64                    `json:"balance_required,omitempty"`
	ExchangeRate                      float64                  `json:"exchange_rate,omitempty"`
	ExchangeRateProvider              ExchangeRateProviderKind `json:"exchange_rate_provider,omitempty"`
	ExchangeRateProviderConfiguration json.RawMessage          `json:"exchange_rate_provider_configuration,omitempty"`
	ExchangeFee                       float64                  `json:"exchange_fee,omitempty"`
	Token                             TokenConfiguration       `json:"token"`
	RewardMode                        RewardMode               `json:"reward_mode,omitempty"`
	LogFile                           string                   `json:"log_file,omitempty"`
}

type RuntimeContext struct {
	ExchangeRateProvider Exchanger
	TokenConfiguration   TokenConfiguration
	RewardMode           RewardMode
	Contract             Contract
}

func Initialize(ctx context.Context, params common.ExtensionInitializationMessage) (*RuntimeContext, error) {
	var config Configuration
	if err := json.Unmarshal(*params.Definition.Configuration, &config); err != nil {
		return nil, err
	}

	if config.LogFile != "" {
		logFile := &lumberjack.Logger{
			Filename:   config.LogFile,
			MaxSize:    10, // megabytes
			MaxBackups: 3,
			MaxAge:     28, // days
			Compress:   true,
		}
		logger := slog.New(slog.NewJSONHandler(logFile, &slog.HandlerOptions{Level: slog.LevelDebug}))
		slog.SetDefault(logger)
	} else {
		slog.SetDefault(slog.New(slog.NewJSONHandler(io.Discard, nil)))
	}

	if len(params.RpcPool) == 0 {
		slog.Error("rpc pool is empty")
		return nil, errors.New("rpc pool is empty")
	}

	rpcs, err := InitializeRpcClients(ctx, params.RpcPool, nil)
	if err != nil {
		slog.Error("failed to initialize rpc clients", "error", err.Error())
		return nil, err
	}

	contract, err := NewContract(ctx, rpcs, params.PayoutPKH, config.Token)
	if err != nil {
		slog.Error("failed to create contract", "error", err.Error())
		return nil, err
	}

	result := &RuntimeContext{
		ExchangeRateProvider: nil,
		TokenConfiguration:   config.Token,
		Contract:             contract,
	}

	switch config.ExchangeRateKind {
	case ExchangeRateFixedAmount:
		if config.RewardAmount <= 0 {
			slog.Error("invalid reward amount, must be greater than 0")
			return nil, errors.New("invalid reward amount, must be greater than 0")
		}
		result.ExchangeRateProvider = FixedAmountExchanger{Amount: config.RewardAmount, Token: config.Token}
	case ExchangeRateFixed:
		if config.ExchangeRate == 0 {
			slog.Error("invalid exchange rate, must be greater than 0")
			return nil, errors.New("invalid exchange rate, must be greater than 0")
		}
		result.ExchangeRateProvider = FixedRateExchanger{Rate: config.ExchangeRate, Token: config.Token, Fee: config.ExchangeFee}
	case ExchangeRateDynamic:
		switch config.ExchangeRateProvider {
		case CMCExchangeRateProviderKind:
			var providerConfig CMCExchangeRateProviderConfiguration
			if err := json.Unmarshal(config.ExchangeRateProviderConfiguration, &providerConfig); err != nil {
				slog.Error("invalid CMC configuration", "error", err.Error())
				return nil, errors.Join(err, errors.New("invalid CMC configuration"))
			}
			if providerConfig.APIKey == "" {
				slog.Error("invalid CMC API key")
				return nil, errors.New("invalid CMC API key")
			}
			if providerConfig.Slug == "" || providerConfig.Slug == "tezos" {
				slog.Error("invalid CMC token slug")
				return nil, errors.New("invalid CMC token slug")
			}
			result.ExchangeRateProvider = NewCMCExchangeRateProvider(providerConfig.Slug, providerConfig.APIKey, config.ExchangeFee, config.Token)
		default:
			slog.Error("invalid exchange rate provider")
			return nil, errors.New("invalid exchange rate provider")
		}

		if config.ExchangeFee <= 0 {
			config.ExchangeFee = 0
		}

	default:
		slog.Error("invalid exchange rate kind")
		return nil, errors.New("invalid exchange rate kind")
	}

	if _, err := tezos.ParseAddress(config.Token.Contract); err != nil {
		slog.Error("invalid token contract")
		return nil, errors.New("invalid token contract")
	}

	if config.Token.Id < 0 {
		slog.Error("invalid token id")
		return nil, errors.New("invalid token id")
	}

	if config.Token.Decimals < 0 {
		slog.Error("invalid token decimals")
		return nil, errors.New("invalid token decimals")
	}

	switch config.RewardMode {
	case RewardModeBonus:
		result.RewardMode = RewardModeBonus
	case RewardModeReplace:
		result.RewardMode = RewardModeReplace
	default:
		slog.Error("invalid reward mode")
		return nil, errors.New("invalid reward mode")
	}

	switch config.Token.Kind {
	case TokenKindFA1_2:
		result.TokenConfiguration.Kind = TokenKindFA1_2
	case TokenKindFA2:
		result.TokenConfiguration.Kind = TokenKindFA2
	default:
		slog.Error("invalid token kind")
		return nil, errors.New("invalid token kind")
	}

	slog.Info("configuration loaded")
	return result, nil
}
