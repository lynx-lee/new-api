package alerting

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
)

// EventHandler defines the interface for alert notification handlers.
type EventHandler interface {
	Handle(event AlertEvent) error
	Name() string
}

// LogHandler writes alerts to the application log at error/warning level.
type LogHandler struct{}

func (h *LogHandler) Name() string { return "log" }

func (h *LogHandler) Handle(event AlertEvent) error {
	msg := fmt.Sprintf("[ALERT] [%s] %s | rule=%s type=%s value=%.2f",
		event.Severity.String(), event.Message, event.RuleName, event.Type.String(), event.Value)
	if event.Resolved {
		msg += " [RESOLVED]"
	}
	for k, v := range event.Labels {
		msg += fmt.Sprintf(" %s=%s", k, v)
	}

	if event.Severity == SeverityCritical {
		logger.LogError(context.Background(), msg)
	} else {
		logger.LogWarn(context.Background(), msg)
	}
	return nil
}

// WebhookHandler sends HTTP POST notifications to a configured webhook URL.
type WebhookHandler struct {
	client  *http.Client
	timeout time.Duration
}

// NewWebhookHandler creates a new webhook notification handler.
func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		timeout: 10 * time.Second,
	}
}

func (h *WebhookHandler) Name() string { return "webhook" }

func (h *WebhookHandler) Handle(event AlertEvent) error {
	url := common.AlertingWebhookURL
	if url == "" {
		return nil // no URL configured, silently skip
	}

	payload, err := common.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal alert event: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "new-api-alerting/1.0")

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d for url %s", resp.StatusCode, url)
	}
	return nil
}
