package messaging

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	commonlogger "github.com/pocwithmehul/common-go-lib/pkg/logger"
)

func newTestLogger() *commonlogger.Logger {
	return commonlogger.NewLogger("test", commonlogger.DatadogConfig{})
}

type testEvent struct {
	Ticker string  `json:"ticker"`
	Price  float64 `json:"price"`
}

func TestSendEvent_Success(t *testing.T) {
	var received testEvent

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, &http.Client{}, newTestLogger())
	event := testEvent{Ticker: "AAPL", Price: 150.0}

	if err := client.SendEvent(context.Background(), event); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received.Ticker != "AAPL" || received.Price != 150.0 {
		t.Errorf("unexpected received event: %+v", received)
	}
}

func TestSendEvent_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient(server.URL, &http.Client{}, newTestLogger())

	err := client.SendEvent(context.Background(), testEvent{Ticker: "AAPL"})
	if err == nil {
		t.Fatal("expected error for 400 status, got nil")
	}
}

func TestSendEvent_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, &http.Client{}, newTestLogger())

	err := client.SendEvent(context.Background(), testEvent{Ticker: "AAPL"})
	if err == nil {
		t.Fatal("expected error for 500 status, got nil")
	}
}

func TestSendEvent_InvalidWebhookURL(t *testing.T) {
	client := NewClient("http://127.0.0.1:1", &http.Client{}, newTestLogger())

	err := client.SendEvent(context.Background(), testEvent{Ticker: "AAPL"})
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
}

func TestSendEvent_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := NewClient(server.URL, &http.Client{}, newTestLogger())
	err := client.SendEvent(ctx, testEvent{Ticker: "AAPL"})
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

func TestSendEvent_UnmarshalableEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, &http.Client{}, newTestLogger())

	// Channels cannot be marshalled to JSON
	err := client.SendEvent(context.Background(), make(chan int))
	if err == nil {
		t.Fatal("expected marshal error, got nil")
	}
}

func TestSendEvent_StatusBoundary(t *testing.T) {
	tests := []struct {
		status  int
		wantErr bool
	}{
		{http.StatusOK, false},
		{http.StatusCreated, false},
		{299, false},
		{http.StatusMultipleChoices, true}, // 300
		{http.StatusBadRequest, true},
		{http.StatusInternalServerError, true},
	}

	for _, tt := range tests {
		status := tt.status
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		}))

		client := NewClient(server.URL, &http.Client{}, newTestLogger())
		err := client.SendEvent(context.Background(), testEvent{Ticker: "T"})

		if tt.wantErr && err == nil {
			t.Errorf("status %d: expected error, got nil", tt.status)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("status %d: unexpected error: %v", tt.status, err)
		}

		server.Close()
	}
}
