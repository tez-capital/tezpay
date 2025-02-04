package main

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/tez-capital/tezpay/common"
	"github.com/trilitech/tzgo/tezos"
)

type ExchangeRateKind string

const (
	ExchangeRateFixed     ExchangeRateKind = "fixed"
	ExchangeRateFixedRate ExchangeRateKind = "fixed-rate"
	ExchangeRateDynamic   ExchangeRateKind = "dynamic"
)

type ExchangeRateProviderKind string

const (
	CMCExchangeRateProviderKind ExchangeRateProviderKind = "cmd"
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

	if len(params.RpcPool) == 0 {
		return nil, errors.New("rpc pool is empty")
	}

	rpcs, err := InitializeRpcClients(ctx, params.RpcPool, nil)
	if err != nil {
		return nil, err
	}

	contract, err := NewContract(ctx, rpcs, params.PayoutPKH, config.Token)
	if err != nil {
		return nil, err
	}

	result := &RuntimeContext{
		ExchangeRateProvider: nil,
		TokenConfiguration:   config.Token,
		Contract:             contract,
	}

	switch config.ExchangeRateKind {
	case ExchangeRateFixed:
		if config.RewardAmount <= 0 {
			return nil, errors.New("invalid reward amount, must be greater than 0")
		}
		result.ExchangeRateProvider = FixedAmountExchanger{Amount: config.RewardAmount, Token: config.Token}
	case ExchangeRateFixedRate:
		if config.ExchangeRate == 0 {
			return nil, errors.New("invalid exchange rate, must be greater than 0")
		}
		result.ExchangeRateProvider = FixedRateExchanger{Rate: config.ExchangeRate, Token: config.Token, Fee: config.ExchangeFee}
	case ExchangeRateDynamic:
		switch config.ExchangeRateProvider {
		case CMCExchangeRateProviderKind:
			var providerConfig CMCExchangeRateProviderConfiguration
			if err := json.Unmarshal(config.ExchangeRateProviderConfiguration, &providerConfig); err != nil {
				return nil, errors.Join(err, errors.New("invalid CMC configuration"))
			}
			if providerConfig.APIKey == "" {
				return nil, errors.New("invalid CMC API key")
			}
			if providerConfig.Slug == "" || providerConfig.Slug == "tezos" {
				return nil, errors.New("invalid CMC token slug")
			}
			result.ExchangeRateProvider = NewCMCExchangeRateProvider(providerConfig.Slug, providerConfig.APIKey, config.ExchangeFee, config.Token)
		default:
			return nil, errors.New("invalid exchange rate provider")
		}

		if config.ExchangeFee <= 0 {
			config.ExchangeFee = 0
		}

	default:
		return nil, errors.New("invalid exchange rate kind")
	}

	if _, err := tezos.ParseAddress(config.Token.Contract); err != nil {
		return nil, errors.New("invalid token contract")
	}

	if config.Token.Id < 0 {
		return nil, errors.New("invalid token id")
	}

	if config.Token.Decimals < 0 {
		return nil, errors.New("invalid token decimals")
	}

	switch config.RewardMode {
	case RewardModeBonus:
		result.RewardMode = RewardModeBonus
	case RewardModeReplace:
		result.RewardMode = RewardModeReplace
	default:
		return nil, errors.New("invalid reward mode")
	}

	switch config.Token.Kind {
	case TokenKindFA1_2:
		result.TokenConfiguration.Kind = TokenKindFA1_2
	case TokenKindFA2:
		result.TokenConfiguration.Kind = TokenKindFA2
	default:
		return nil, errors.New("invalid token kind")
	}

	return result, nil
}
