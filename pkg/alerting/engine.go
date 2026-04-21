package alerting

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
)

// Engine is the central alerting engine that evaluates rules and fires events.
type Engine struct {
	rules     []AlertRule
	active    sync.Map // map[string]time.Time  (ruleID -> last fired timestamp)
	eventChan chan AlertEvent
	handlers  []EventHandler
	running   atomic.Bool
	mu        sync.RWMutex
}

var globalEngine *Engine
var engineOnce sync.Once

// InitEngine initializes the global alerting engine with default rules and handlers.
func InitEngine() {
	engineOnce.Do(func() {
		e := NewEngine()

		// Register built-in handlers
		e.AddHandler(&LogHandler{})
		if common.AlertingWebhookURL != "" {
			e.AddHandler(NewWebhookHandler())
		}

		// Load default rules
		for _, rule := range DefaultRules() {
			if rule.Enabled {
				e.AddRule(rule)
			}
		}

		globalEngine = e
		common.SysLog(fmt.Sprintf("alerting engine initialized, %d rules, %d handlers",
			len(e.rules), len(e.handlers)))
	})
}

// GetEngine returns the global alerting engine.
func GetEngine() *Engine {
	return globalEngine
}

// NewEngine creates a new alerting engine.
func NewEngine() *Engine {
	return &Engine{
		rules:     make([]AlertRule, 0),
		eventChan: make(chan AlertEvent, 256),
		handlers:  make([]EventHandler, 0),
		running:   atomic.Bool{},
	}
}

// AddRule appends a new alert rule to the engine.
func (e *Engine) AddRule(rule AlertRule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = append(e.rules, rule)
}

// AddHandler registers a notification handler.
func (e *Engine) AddHandler(handler EventHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers = append(e.handlers, handler)
}

// Start launches the background event processing goroutine.
func (e *Engine) Start(ctx context.Context) {
	if !common.AlertingEnabled {
		logger.LogInfo(ctx, "alerting engine disabled (no-op)")
		return
	}
	if e.running.Swap(true) {
		return // already running
	}

	go func() {
		logger.LogInfo(ctx, "alerting engine started, processing events")
		for {
			select {
			case <-ctx.Done():
				e.running.Store(false)
				logger.LogInfo(ctx, "alerting engine stopped")
				return
			case event := <-e.eventChan:
				e.dispatchEvent(event)
			}
		}
	}()
}

// Stop halts the background processing.
func (e *Engine) Stop() {
	e.running.Store(false)
	close(e.eventChan)
}

// FireEvent queues an alert event for processing.
func (e *Engine) FireEvent(event AlertEvent) {
	if !common.AlertingEnabled || !e.running.Load() {
		return
	}
	// Cooldown check
	if v, ok := e.active.Load(event.RuleID); ok {
		lastFired := v.(time.Time)
		cooldown := getCooldownForRule(e.rules, event.RuleID)
		if time.Since(lastFired) < cooldown {
			return // still in cooldown
		}
	}
	e.active.Store(event.RuleID, time.Now())

	select {
	case e.eventChan <- event:
	default:
		logger.LogWarn(context.Background(),
			fmt.Sprintf("alerting event channel full, dropping event for rule %s", event.RuleName))
	}
}

// dispatchEvent sends an alert to all registered handlers.
func (e *Engine) dispatchEvent(event AlertEvent) {
	for _, h := range e.handlers {
		if shouldHandle(h, event.Channels) {
			if err := h.Handle(event); err != nil {
				logger.LogError(context.Background(),
					fmt.Sprintf("alert handler '%s' failed: %v", h.Name(), err))
			}
		}
	}
}

// FireRelayError checks error-rate/channel-down rules and fires if conditions met.
// This is called from controller/relay.go after each relay attempt failure.
func FireRelayError(provider, modelName string, channelId int, err error) {
	if !common.AlertingEnabled || globalEngine == nil {
		return
	}

	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	globalEngine.FireEvent(AlertEvent{
		RuleID:   "default-channel-down",
		RuleName: "Channel Down Detection",
		Type:     RuleChannelDown,
		Severity: SeverityCritical,
		Message:  fmt.Sprintf("Relay error on channel %d [%s/%s]: %s", channelId, provider, modelName, errMsg),
		Labels: map[string]string{
			"provider":   provider,
			"model":      modelName,
			"channel_id": fmt.Sprintf("%d", channelId),
		},
		Value:     1,
		Timestamp: time.Now(),
	})
}

// FireLatencyHigh fires when relay latency exceeds threshold.
func FireLatencyHigh(provider, modelName string, durationMs float64) {
	if !common.AlertingEnabled || globalEngine == nil {
		return
	}

	severity := SeverityWarning
	if durationMs > 60000 { // >60s is critical
		severity = SeverityCritical
	}

	globalEngine.FireEvent(AlertEvent{
		RuleID:   "default-latency-p99",
		RuleName: "Model P99 Latency High",
		Type:     RuleLatencyP99,
		Severity: severity,
		Message:  fmt.Sprintf("High latency detected for %s/%s: %.0fms", provider, modelName, durationMs),
		Labels: map[string]string{
			"provider": provider,
			"model":    modelName,
		},
		Value:     durationMs,
		Timestamp: time.Now(),
	})
}

// FireQuotaLow fires when user quota falls below threshold.
func FireQuotaLow(userId int, username string, quotaRemaining float64) {
	if !common.AlertingEnabled || globalEngine == nil {
		return
	}

	globalEngine.FireEvent(AlertEvent{
		RuleID:   "default-quota-low",
		RuleName: "User Quota Low Warning",
		Type:     RuleQuotaLow,
		Severity: SeverityInfo,
		Message:  fmt.Sprintf("User %s (id=%d) quota low: %.2f", username, userId, quotaRemaining),
		Labels: map[string]string{
			"user_id":  fmt.Sprintf("%d", userId),
			"username": username,
		},
		Value:     quotaRemaining,
		Timestamp: time.Now(),
	})
}

// GetAllStatuses returns current status of all active alerts.
func (e *Engine) GetAllStatuses() []map[string]interface{} {
	var result []map[string]interface{}
	e.active.Range(func(key, value any) bool {
		result = append(result, map[string]interface{}{
			"rule_id":     key.(string),
			"last_fired": value.(time.Time).Format(time.RFC3339),
		})
		return true
	})
	return result
}

// MetricPoint represents a single metric sample for evaluation.
type MetricPoint struct {
	Name      string
	Labels    map[string]string
	Value     float64
	Timestamp time.Time
}

// Evaluate checks all rules against a given metric point and fires alerts if needed.
func (e *Engine) Evaluate(point MetricPoint) {
	if !common.AlertingEnabled {
		return
	}
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}
		if matchesRuleType(point.Name, rule.Type) && point.Value >= rule.Threshold {
			e.FireEvent(AlertEvent{
				RuleID:    rule.ID,
				RuleName:  rule.Name,
				Type:      rule.Type,
				Severity:  inferSeverity(rule),
				Message:   fmt.Sprintf("%s exceeded threshold: %.2f >= %.2f", point.Name, point.Value, rule.Threshold),
				Labels:   point.Labels,
				Value:    point.Value,
				Timestamp: time.Now(),
			})
		}
	}
}

// Helper functions

func shouldHandle(handler EventHandler, channels []string) bool {
	name := handler.Name()
	for _, ch := range channels {
		if strings.EqualFold(ch, name) {
			return true
		}
	}
	return len(channels) == 0
}

func getCooldownForRule(rules []AlertRule, ruleID string) time.Duration {
	for _, r := range rules {
		if r.ID == ruleID && r.Cooldown > 0 {
			return time.Duration(r.Cooldown) * time.Second
		}
	}
	return time.Duration(common.AlertingCooldownSeconds) * time.Second
}

func matchesRuleType(metricName string, ruleType RuleType) bool {
	switch ruleType {
	case RuleErrorRate:
		return metricName == "relay_error_rate"
	case RuleLatencyP99:
		return metricName == "relay_duration_p99" || metricName == "relay_latency"
	case RuleChannelDown:
		return metricName == "channel_consecutive_failures"
	case RuleQuotaLow:
		return metricName == "user_quota_remaining"
	default:
		return false
	}
}

func inferSeverity(rule AlertRule) Severity {
	switch rule.Type {
	case RuleChannelDown:
		return SeverityCritical
	case RuleLatencyP99:
		if rule.Threshold > 30000 { // >30s critical
			return SeverityCritical
		}
		return SeverityWarning
	case RuleErrorRate:
		return SeverityWarning
	default:
		return SeverityInfo
	}
}
