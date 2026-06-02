package messaging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	commonlogger "github.com/pocwithmehul/common-go-lib/pkg/logger"
)

type Client struct {
	webhookURL string
	httpClient *http.Client
	logger     *commonlogger.Logger
}

func NewClient(webhookURL string, httpClient *http.Client, logger *commonlogger.Logger) *Client {
	return &Client{
		webhookURL: webhookURL,
		httpClient: httpClient,
		logger:     logger,
	}
}

func (c *Client) SendEvent(ctx context.Context, event interface{}) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("post event: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Error("close webhook response body failed", map[string]interface{}{"error": closeErr.Error()})
		}
	}()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}
