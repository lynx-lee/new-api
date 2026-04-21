// Package alerting provides rule-based alerting with multiple notification channels
// for monitoring relay health, error rates, latency, and resource usage.
package alerting

import (
	"time"

	"github.com/QuantumNous/new-api/common"
)

// RuleType defines the category of an alert rule.
type RuleType int

const (
	RuleErrorRate  RuleType = iota // Channel/provider error rate threshold
	RuleLatencyP99                 // P99 latency exceeds threshold
	RuleChannelDown                // Channel consecutive failure detection
	RuleQuotaLow                   // User quota running low
)

func (t RuleType) String() string {
	switch t {
	case RuleErrorRate:
		return "error_rate"
	case RuleLatencyP99:
		return "latency_p99"
	case RuleChannelDown:
		return "channel_down"
	case RuleQuotaLow:
		return "quota_low"
	default:
		return "unknown"
	}
}

// Severity indicates the urgency level of an alert.
type Severity int

const (
	SeverityInfo     Severity = iota // Informational
	SeverityWarning                  // Warning, needs attention
	SeverityCritical                 // Critical, immediate action required
)

func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// AlertRule defines a condition that triggers notifications when met.
type AlertRule struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      RuleType  `json:"type"`
	Enabled   bool      `json:"enabled"`

	// Condition
	Threshold float64 `json:"threshold"` // value threshold (rate, ms, count)
	Duration  int     `json:"duration"`  // seconds the condition must persist

	// Notification
	Channels   []string `json:"channels"`   // ["log","webhook"]
	WebhookURL string   `json:"webhook_url"`
	Cooldown   int      `json:"cooldown"`   // seconds between repeated alerts for same rule

	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AlertEvent is a fired alert notification.
type AlertEvent struct {
	RuleID    string            `json:"rule_id"`
	RuleName  string            `json:"rule_name"`
	Type      RuleType          `json:"type"`
	Severity  Severity          `json:"severity"`
	Message   string            `json:"message"`
	Labels    map[string]string `json:"labels"`
	Value     float64           `json:"value"`
	Timestamp time.Time         `json:"timestamp"`
	Resolved  bool              `json:"resolved"` // true if this resolves a prior firing
}

// DefaultRules returns built-in default alert rules.
func DefaultRules() []AlertRule {
	now := time.Now()
	return []AlertRule{
		{
			ID:        "default-error-rate",
			Name:      "Channel Error Rate High",
			Type:      RuleErrorRate,
			Enabled:   common.AlertingEnabled,
			Threshold: 50.0,  // 50% error rate
			Duration:  300,    // 5 minutes
			Channels:  []string{"log"},
			Cooldown:  600,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "default-latency-p99",
			Name:      "Model P99 Latency High",
			Type:      RuleLatencyP99,
			Enabled:   common.AlertingEnabled,
			Threshold: 30000, // 30 seconds
			Duration:  600,    // 10 minutes
			Channels:  []string{"log"},
			Cooldown:  900,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "default-channel-down",
			Name:      "Channel Down Detection",
			Type:      RuleChannelDown,
			Enabled:   common.AlertingEnabled,
			Threshold: 5.0,   // 5 consecutive failures
			Duration:  60,     // 1 minute
			Channels:  []string{"log", "webhook"},
			Cooldown:  300,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "default-quota-low",
			Name:      "User Quota Low Warning",
			Type:      RuleQuotaLow,
			Enabled:   common.AlertingEnabled,
			Threshold: float64(common.AlertingQuotaThreshold),
			Duration:  0,       // immediate (single check)
			Channels:  []string{"log"},
			Cooldown:  3600,    // 1 hour
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}
