package main

import (
	"log/slog"
	"math"

	"github.com/trilitech/tzgo/tezos"
)

type Exchanger interface {
	RefreshExchangeRate() error
	ExchangeTezToToken(mutez tezos.Z) tezos.Z
}

type FixedAmountExchanger struct {
	amount float64
	token  TokenConfiguration
}

func NewFixedAmountExchanger(amount float64, token TokenConfiguration) FixedAmountExchanger {
	return FixedAmountExchanger{
		amount: amount,
		token:  token,
	}
}

func (e FixedAmountExchanger) RefreshExchangeRate() error {
	return nil
}

func (e FixedAmountExchanger) ExchangeTezToToken(_ tezos.Z) tezos.Z {
	slog.Debug("Exchanging fixed amount", "amount", e.amount, "token", e.token)
	// we need to multiply by 1_000_000 because other functions assume we calculated with mutez
	return tezos.NewZ(int64(e.amount * math.Pow10(e.token.Decimals)))
}

type FixedRateExchanger struct {
	rate  tezos.Z
	fee   tezos.Z
	token TokenConfiguration
}

func NewFixedRateExchanger(rate, fee float64, token TokenConfiguration) FixedRateExchanger {
	return FixedRateExchanger{
		rate:  tezos.NewZ(int64(rate * float64(PRECISION))),
		fee:   tezos.NewZ(int64(fee * float64(PRECISION))),
		token: token,
	}
}

func (e FixedRateExchanger) RefreshExchangeRate() error {
	return nil
}

func (e FixedRateExchanger) ExchangeTezToToken(mutez tezos.Z) tezos.Z {
	decimalsMultiplier := int64(math.Pow10(e.token.Decimals))

	token_amount := mutez.Mul(e.rate).Mul64(decimalsMultiplier).Div(tezos.NewZ(PRECISION).Sub(e.fee)).Div64(MUTEZ_FACTOR)
	slog.Debug("Exchanging amount", "amount", mutez, "token_amount", token_amount, "rate", e.rate, "fee", e.fee)
	return token_amount
}
