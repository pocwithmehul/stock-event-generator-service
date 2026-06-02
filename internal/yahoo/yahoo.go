package yahoo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type yahooQuoteResponse struct {
	QuoteResponse struct {
		Result []struct {
			Symbol             string  `json:"symbol"`
			RegularMarketPrice float64 `json:"regularMarketPrice"`
		} `json:"result"`
	} `json:"quoteResponse"`
}

func FetchTickerPrice(ticker string) (float64, error) {
	endpoint := "https://query1.finance.yahoo.com/v7/finance/quote"
	params := url.Values{}
	params.Set("symbols", ticker)
	tickerURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	resp, err := http.Get(tickerURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("yahoo returned %d", resp.StatusCode)
	}

	var result yahooQuoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if len(result.QuoteResponse.Result) == 0 {
		return 0, fmt.Errorf("missing quote for %s", ticker)
	}

	return result.QuoteResponse.Result[0].RegularMarketPrice, nil
}
