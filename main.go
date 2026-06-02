package main

import (
	"context"
	"log"
	"net/http"
	"time"

	commonlib "github.com/pocwithmehul/common-go-lib"
	"github.com/pocwithmehul/stock-event-generator-service/internal/messaging"
	"github.com/pocwithmehul/stock-event-generator-service/internal/yahoo"
)

type StockEvent struct {
	Ticker    string    `json:"ticker"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
	Volume    int64     `json:"volume"`
}

func main() {
	cfg, err := commonlib.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := commonlib.NewLogger("stock-event-generator-service", cfg.Datadog)
	if cfg.IntervalMs <= 0 {
		cfg.IntervalMs = 10
	}

	webhookURL := cfg.Server.WebhookURL
	if webhookURL == "" {
		webhookURL = "http://stock-webhook-service:8090/v1/stock/callbacks"
	}

	logger.Info("starting stock-event-generator-service", map[string]interface{}{"webhook": webhookURL, "intervalMs": cfg.IntervalMs})
	poster := messaging.NewClient(webhookURL, &http.Client{Timeout: 5 * time.Second}, logger)

	for _, ticker := range cfg.Tickers {
		if ticker == "" {
			continue
		}
	}

	for {
		for _, ticker := range cfg.Tickers {
			price, err := yahoo.FetchTickerPrice(ticker)
			if err != nil {
				logger.Error("failed to fetch price", map[string]interface{}{"ticker": ticker, "error": err.Error()})
				continue
			}

			event := StockEvent{
				Ticker:    ticker,
				Price:     price,
				Timestamp: time.Now().UTC(),
				Volume:    0,
			}

			if err := poster.SendEvent(context.Background(), event); err != nil {
				logger.Error("post event failed", map[string]interface{}{"ticker": ticker, "error": err.Error()})
				continue
			}

			logger.Info("sent stock event", map[string]interface{}{"ticker": ticker, "price": price})
			if cfg.IntervalMs > 0 {
				time.Sleep(time.Duration(cfg.IntervalMs) * time.Millisecond)
			}
		}
	}
}
