package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

type cmcResponse struct {
	Data map[string]struct {
		Slug  string `json:"slug"`
		Quote map[string]struct {
			Price float64 `json:"price"`
		} `json:"quote"`
	} `json:"data"`
}

func get_cmc_exchange_rate(token_slug string, exchange_fee float64, apiKey string) (float64, error) {
	if token_slug == "" || token_slug == "tezos" {
		return 0, errors.New("Invalid token slug")
	}

	if apiKey == "" {
		return 0, errors.New("Invalid API key")
	}

	if exchange_fee <= 0 {
		exchange_fee = 0
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
		return 0, err
	}

	if cmcResp.Data == nil {
		return 0, errors.New("Invalid response from CMC")
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
		return 0, errors.New("Invalid price data")
	}

	exchangeRate := tezosPrice / tokenPrice
	return exchangeRate * (1 - exchange_fee), nil
}

type CMCExchangeRateProvider struct {
	slug    string
	api_key string
	fee     float64
	token   TokenConfiguration

	rate float64
}

func NewCMCExchangeRateProvider(slug, api_key string, fee float64, token TokenConfiguration) *CMCExchangeRateProvider {
	return &CMCExchangeRateProvider{
		slug:    slug,
		api_key: api_key,
		fee:     fee,
		token:   token,
	}
}

func (p *CMCExchangeRateProvider) RefreshExchangeRate() error {
	rate, err := get_cmc_exchange_rate(p.slug, p.fee, p.api_key)
	if err != nil {
		return err
	}
	p.rate = rate
	return nil
}

func (p *CMCExchangeRateProvider) ExchangeToToken(amount int64) int64 {
	token_amount := float64(amount) * p.rate * (1 - p.fee)

	if p.token.Decimals == 0 {
		return int64(token_amount)
	}

	return int64(token_amount * float64(p.token.Decimals))
}
