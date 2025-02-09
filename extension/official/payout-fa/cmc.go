package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"

	"github.com/trilitech/tzgo/tezos"
)

type cmcResponse struct {
	Data map[string]struct {
		Slug  string `json:"slug"`
		Quote map[string]struct {
			Price float64 `json:"price"`
		} `json:"quote"`
	} `json:"data"`
}

func get_cmc_exchange_rate(token_slug string, apiKey string) (tezos.Z, error) {
	if token_slug == "" || token_slug == "tezos" {
		return tezos.Zero, errors.New("Invalid token slug")
	}

	if apiKey == "" {
		return tezos.Zero, errors.New("Invalid API key")
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://pro-api.coinmarketcap.com/v2/cryptocurrency/quotes/latest", nil)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	q := url.Values{}
	q.Add("slug", fmt.Sprintf("tezos,%s", token_slug))
	q.Add("aux", "num_market_pairs")

	req.Header.Set("Accepts", "application/json")
	req.Header.Add("X-CMC_PRO_API_KEY", apiKey)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request to server")
		os.Exit(1)
	}

	respBody, _ := io.ReadAll(resp.Body)

	var cmcResp cmcResponse
	if err := json.Unmarshal(respBody, &cmcResp); err != nil {
		return tezos.Zero, err
	}

	if cmcResp.Data == nil {
		return tezos.Zero, errors.New("Invalid response from CMC")
	}

	var tezosPrice float64
	var tokenPrice float64
	for _, data := range cmcResp.Data {
		for _, quote := range data.Quote {
			switch data.Slug {
			case "tezos":
				tezosPrice = quote.Price
			case token_slug:
				tokenPrice = quote.Price
			}
		}
	}

	if tezosPrice == 0 || tokenPrice == 0 {
		return tezos.Zero, errors.New("Invalid price data")
	}

	exchangeRate := tezosPrice / tokenPrice
	return tezos.NewZ(int64(exchangeRate * float64(PRECISION))), nil
}

type CMCExchangeRateProvider struct {
	slug    string
	api_key string
	fee     tezos.Z
	token   TokenConfiguration

	rate tezos.Z
}

func NewCMCExchangeRateProvider(slug, api_key string, fee float64, token TokenConfiguration) *CMCExchangeRateProvider {
	return &CMCExchangeRateProvider{
		slug:    slug,
		api_key: api_key,
		fee:     tezos.NewZ(int64(fee * float64(PRECISION))),
		token:   token,
	}
}

func (p *CMCExchangeRateProvider) RefreshExchangeRate() error {
	rate, err := get_cmc_exchange_rate(p.slug, p.api_key)
	if err != nil {
		return err
	}
	slog.Info("Exchange rate updated", "rate", rate)
	p.rate = rate
	return nil
}

func (p *CMCExchangeRateProvider) ExchangeTezToToken(mutez tezos.Z) tezos.Z {
	decimalsMultiplier := int64(math.Pow10(p.token.Decimals))

	token_amount := mutez.Mul(p.rate).Mul64(decimalsMultiplier).Div(tezos.NewZ(PRECISION).Sub(p.fee)).Div64(MUTEZ_FACTOR)
	slog.Debug("Exchanging amount", "amount", mutez, "token_amount", token_amount, "rate", p.rate, "fee", p.fee)
	return token_amount
}
