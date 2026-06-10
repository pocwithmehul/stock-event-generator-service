package yahoo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func withMockClient(statusCode int, body interface{}, header http.Header) func() {
	original := httpClient
	httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			var bodyStr string
			switch v := body.(type) {
			case string:
				bodyStr = v
			default:
				b, _ := json.Marshal(v)
				bodyStr = string(b)
			}
			h := make(http.Header)
			for k, vals := range header {
				h[k] = vals
			}
			return &http.Response{
				StatusCode: statusCode,
				Body:       io.NopCloser(strings.NewReader(bodyStr)),
				Header:     h,
			}, nil
		}),
	}
	return func() { httpClient = original }
}

func withErrorClient(err error) func() {
	original := httpClient
	httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return nil, err
		}),
	}
	return func() { httpClient = original }
}

func makeQuoteResponse(prices map[string]float64) yahooQuoteResponse {
	var resp yahooQuoteResponse
	for sym, price := range prices {
		resp.QuoteResponse.Result = append(resp.QuoteResponse.Result, struct {
			Symbol             string  `json:"symbol"`
			RegularMarketPrice float64 `json:"regularMarketPrice"`
		}{Symbol: sym, RegularMarketPrice: price})
	}
	return resp
}

func TestFetchTickerPrices_EmptyTickers(t *testing.T) {
	prices, err := FetchTickerPrices([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prices) != 0 {
		t.Errorf("expected empty map, got %v", prices)
	}
}

func TestFetchTickerPrices_Success(t *testing.T) {
	restore := withMockClient(http.StatusOK, makeQuoteResponse(map[string]float64{
		"AAPL": 150.0,
		"MSFT": 300.0,
	}), nil)
	defer restore()

	prices, err := FetchTickerPrices([]string{"AAPL", "MSFT"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prices["AAPL"] != 150.0 {
		t.Errorf("expected AAPL=150.0, got %v", prices["AAPL"])
	}
	if prices["MSFT"] != 300.0 {
		t.Errorf("expected MSFT=300.0, got %v", prices["MSFT"])
	}
}

func TestFetchTickerPrices_RateLimit_NoRetryAfter(t *testing.T) {
	restore := withMockClient(http.StatusTooManyRequests, "", nil)
	defer restore()

	_, err := FetchTickerPrices([]string{"AAPL"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	rlErr, ok := err.(*RateLimitError)
	if !ok {
		t.Fatalf("expected *RateLimitError, got %T: %v", err, err)
	}
	if rlErr.RetryAfter != 0 {
		t.Errorf("expected RetryAfter=0, got %v", rlErr.RetryAfter)
	}
}

func TestFetchTickerPrices_RateLimit_WithRetryAfter(t *testing.T) {
	h := http.Header{"Retry-After": []string{"60"}}
	restore := withMockClient(http.StatusTooManyRequests, "", h)
	defer restore()

	_, err := FetchTickerPrices([]string{"AAPL"})
	rlErr, ok := err.(*RateLimitError)
	if !ok {
		t.Fatalf("expected *RateLimitError, got %T", err)
	}
	if rlErr.RetryAfter != 60*time.Second {
		t.Errorf("expected RetryAfter=60s, got %v", rlErr.RetryAfter)
	}
}

func TestFetchTickerPrices_NonOKStatus(t *testing.T) {
	restore := withMockClient(http.StatusInternalServerError, "", nil)
	defer restore()

	_, err := FetchTickerPrices([]string{"AAPL"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to mention status code 500, got: %v", err)
	}
}

func TestFetchTickerPrices_EmptyResult(t *testing.T) {
	restore := withMockClient(http.StatusOK, makeQuoteResponse(map[string]float64{}), nil)
	defer restore()

	_, err := FetchTickerPrices([]string{"AAPL"})
	if err == nil {
		t.Fatal("expected error for empty result, got nil")
	}
}

func TestFetchTickerPrices_SkipsEmptySymbol(t *testing.T) {
	var resp yahooQuoteResponse
	resp.QuoteResponse.Result = []struct {
		Symbol             string  `json:"symbol"`
		RegularMarketPrice float64 `json:"regularMarketPrice"`
	}{
		{Symbol: "", RegularMarketPrice: 99.9},
		{Symbol: "AAPL", RegularMarketPrice: 150.0},
	}
	restore := withMockClient(http.StatusOK, resp, nil)
	defer restore()

	prices, err := FetchTickerPrices([]string{"AAPL"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := prices[""]; ok {
		t.Error("expected empty symbol to be skipped")
	}
	if prices["AAPL"] != 150.0 {
		t.Errorf("expected AAPL=150.0, got %v", prices["AAPL"])
	}
}

func TestFetchTickerPrices_InvalidJSON(t *testing.T) {
	restore := withMockClient(http.StatusOK, "not-json", nil)
	defer restore()

	_, err := FetchTickerPrices([]string{"AAPL"})
	if err == nil {
		t.Fatal("expected JSON decode error, got nil")
	}
}

func TestFetchTickerPrices_TransportError(t *testing.T) {
	restore := withErrorClient(fmt.Errorf("connection refused"))
	defer restore()

	_, err := FetchTickerPrices([]string{"AAPL"})
	if err == nil {
		t.Fatal("expected transport error, got nil")
	}
}

func TestFetchTickerPrice_Success(t *testing.T) {
	restore := withMockClient(http.StatusOK, makeQuoteResponse(map[string]float64{"AAPL": 175.5}), nil)
	defer restore()

	price, err := FetchTickerPrice("AAPL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price != 175.5 {
		t.Errorf("expected 175.5, got %v", price)
	}
}

func TestFetchTickerPrice_MissingTicker(t *testing.T) {
	restore := withMockClient(http.StatusOK, makeQuoteResponse(map[string]float64{"MSFT": 300.0}), nil)
	defer restore()

	_, err := FetchTickerPrice("AAPL")
	if err == nil {
		t.Fatal("expected missing quote error, got nil")
	}
	if !strings.Contains(err.Error(), "AAPL") {
		t.Errorf("expected error to mention AAPL, got: %v", err)
	}
}

func TestParseRetryAfter_Empty(t *testing.T) {
	if d := parseRetryAfter(""); d != 0 {
		t.Errorf("expected 0, got %v", d)
	}
}

func TestParseRetryAfter_ValidSeconds(t *testing.T) {
	if d := parseRetryAfter("30"); d != 30*time.Second {
		t.Errorf("expected 30s, got %v", d)
	}
}

func TestParseRetryAfter_Zero(t *testing.T) {
	if d := parseRetryAfter("0"); d != 0 {
		t.Errorf("expected 0, got %v", d)
	}
}

func TestParseRetryAfter_Negative(t *testing.T) {
	if d := parseRetryAfter("-5"); d != 0 {
		t.Errorf("expected 0, got %v", d)
	}
}

func TestParseRetryAfter_Invalid(t *testing.T) {
	if d := parseRetryAfter("abc"); d != 0 {
		t.Errorf("expected 0, got %v", d)
	}
}

func TestRateLimitError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *RateLimitError
		contains string
	}{
		{"nil", nil, "yahoo rate limited request"},
		{"no retry-after", &RateLimitError{}, "yahoo returned 429"},
		{"with retry-after", &RateLimitError{RetryAfter: 30 * time.Second}, "30s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			if !strings.Contains(msg, tt.contains) {
				t.Errorf("expected %q to contain %q", msg, tt.contains)
			}
		})
	}
}
