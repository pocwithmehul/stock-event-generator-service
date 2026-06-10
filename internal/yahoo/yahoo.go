package yahoo

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type yahooQuoteResponse struct {
	QuoteResponse struct {
		Result []struct {
			Symbol             string  `json:"symbol"`
			RegularMarketPrice float64 `json:"regularMarketPrice"`
		} `json:"result"`
	} `json:"quoteResponse"`
}

type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	if e == nil {
		return "yahoo rate limited request"
	}
	if e.RetryAfter > 0 {
		return fmt.Sprintf("yahoo returned 429, retry after %s", e.RetryAfter)
	}
	return "yahoo returned 429"
}

var httpClient = &http.Client{
	Timeout: 5 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
	},
}

func FetchTickerPrices(tickers []string) (map[string]float64, error) {
	if len(tickers) == 0 {
		return map[string]float64{}, nil
	}

	endpoint := "https://query1.finance.yahoo.com/v7/finance/quote"
	params := url.Values{}
	params.Set("symbols", strings.Join(tickers, ","))
	tickerURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	req, err := http.NewRequest(http.MethodGet, tickerURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "stock-event-generator-service/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, &RateLimitError{RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After"))}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo returned %d", resp.StatusCode)
	}

	var result yahooQuoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	prices := make(map[string]float64, len(result.QuoteResponse.Result))
	for _, quote := range result.QuoteResponse.Result {
		if quote.Symbol == "" {
			continue
		}
		prices[quote.Symbol] = quote.RegularMarketPrice
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("missing quotes for %s", strings.Join(tickers, ","))
	}

	return prices, nil
}

func FetchTickerPrice(ticker string) (float64, error) {
	prices, err := FetchTickerPrices([]string{ticker})
	if err != nil {
		return 0, err
	}

	price, ok := prices[ticker]
	if !ok {
		return 0, fmt.Errorf("missing quote for %s", ticker)
	}

	return price, nil
}

func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}

	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return 0
	}

	return time.Duration(seconds) * time.Second
}
