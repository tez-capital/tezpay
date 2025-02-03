package main

type Exchanger interface {
	RefreshExchangeRate() error
	ExchangeToToken(amount int64) int64
}

type FixedAmountExchanger struct {
	Amount int64
	Token  TokenConfiguration
}

func (e FixedAmountExchanger) RefreshExchangeRate() error {
	return nil
}

func (e FixedAmountExchanger) ExchangeToToken(_ int64) int64 {
	return e.Amount * e.Token.Decimals
}

type FixedRateExchanger struct {
	Rate  float64
	Token TokenConfiguration
	Fee   float64
}

func (e FixedRateExchanger) RefreshExchangeRate() error {
	return nil
}

func (e FixedRateExchanger) ExchangeToToken(amount int64) int64 {
	token_amount := float64(amount) * e.Rate * (1 - e.Fee)

	result := token_amount * float64(e.Token.Decimals)

	return int64(result)
}
