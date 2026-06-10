package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	commonlogger "github.com/pocwithmehul/common-go-lib/pkg/logger"
	commontracer "github.com/pocwithmehul/common-go-lib/pkg/tracer"
	"github.com/pocwithmehul/stock-event-generator-service/internal/config"
	"github.com/pocwithmehul/stock-event-generator-service/internal/handler"
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
	commontracer.Start(commontracer.Config{
		Service: "stock-event-generator-service",
	})
	defer commontracer.Stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := commonlogger.NewLogger("stock-event-generator-service", cfg.Datadog)
	if cfg.IntervalMs <= 0 {
		cfg.IntervalMs = 1000
	}

	webhookURL := cfg.Server.WebhookURL
	if webhookURL == "" {
		webhookURL = "http://stock-webhook-service:8090/v1/stock/callbacks"
	}

	logger.Info("starting stock-event-generator-service", map[string]interface{}{"webhook": webhookURL, "intervalMs": cfg.IntervalMs})
	poster := messaging.NewClient(webhookURL, &http.Client{Timeout: 5 * time.Second}, logger)

	tickers := make([]string, 0, len(cfg.Tickers))
	for _, ticker := range cfg.Tickers {
		if ticker == "" {
			continue
		}
		tickers = append(tickers, ticker)
	}

	if len(tickers) == 0 {
		log.Fatal("no tickers configured")
	}

	port := cfg.Server.Port
	if port == 0 {
		port = 8081
	}
	startHealthServer(port, logger)

	for {
		prices, err := yahoo.FetchTickerPrices(tickers)
		if err != nil {
			var rateLimitErr *yahoo.RateLimitError
			if errors.As(err, &rateLimitErr) {
				backoff := 30 * time.Second
				if rateLimitErr.RetryAfter > 0 {
					backoff = rateLimitErr.RetryAfter
				}
				logger.Error("failed to fetch prices", map[string]interface{}{"error": err.Error(), "backoff": backoff.String()})
				time.Sleep(backoff)
				continue
			}

			logger.Error("failed to fetch prices", map[string]interface{}{"error": err.Error()})
			time.Sleep(time.Duration(cfg.IntervalMs) * time.Millisecond)
			continue
		}

		for _, ticker := range tickers {
			price, ok := prices[ticker]
			if !ok {
				logger.Error("missing fetched price", map[string]interface{}{"ticker": ticker})
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
		}

		if cfg.IntervalMs > 0 {
			time.Sleep(time.Duration(cfg.IntervalMs) * time.Millisecond)
		}
	}
}

func startHealthServer(port int, logger *commonlogger.Logger) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handler.HealthHandler())

	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("starting health server", map[string]interface{}{"addr": addr})
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("health server failed", map[string]interface{}{"error": err.Error()})
			log.Fatalf("health server failed: %v", err)
		}
	}()
}
