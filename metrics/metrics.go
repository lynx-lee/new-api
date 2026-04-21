// Package metrics provides Prometheus metrics instrumentation for New-API.
// It exposes HTTP request latency, error rates, relay duration, and system health
// metrics at the /metrics endpoint for Prometheus scraping.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ---- Request Metrics ----

var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "newapi",
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "newapi",
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path", "status"},
	)

HTTPRequestInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "newapi",
		Name:      "http_requests_in_flight",
		Help:      "Number of HTTP requests currently being processed",
	})

	HTTPResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "newapi",
			Name:      "http_response_size_bytes",
			Help:      "HTTP response size in bytes",
			Buckets:   prometheus.ExponentialBuckets(256, 4, 8), // 256B ~ 256KB
		},
		[]string{"method", "path"},
	)
)

// ---- Relay (Upstream API) Metrics ----

var (
	RelayRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "newapi",
			Name:      "relay_requests_total",
			Help:      "Total number of upstream AI API relay requests",
		},
		[]string{"provider", "model", "status"}, // status: success, error, stream_error
	)

	RelayRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "newapi",
			Name:      "relay_request_duration_seconds",
			Help:      "Duration of upstream AI API requests in seconds",
			Buckets:   []float64{0.1, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300}, // LLM calls can be long
		},
		[]string{"provider", "model", "status"},
	)

	RelayTokensTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "newapi",
			Name:      "relay_tokens_total",
			Help:      "Total number of tokens processed (prompt + completion)",
		},
		[]string{"provider", "model", "type"}, // type: prompt, completion
	)

	RelayStreamEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "newapi",
			Name:      "relay_stream_events_total",
			Help:      "Total number of streaming events sent to clients",
		},
		[]string{"provider", "model", "event_type"},
	)
)

// ---- Billing / Quota Metrics ----

var (
	BillingOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "newapi",
			Name:      "billing_operations_total",
			Help:      "Total billing operations (pre-consume, settle, refund)",
		},
		[]string{"operation"}, // operation: pre_consume, settle, refund
	)

	UserQuotaGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "newapi",
			Name:      "user_quota_remaining",
			Help:      "Remaining quota for users (sampled)",
		},
		[]string{"user_tier"}, // tier: root, admin, common, guest
	)

	ChannelQuotaUsed = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "newapi",
			Name:      "channel_quota_used",
			Help:      "Used quota per channel (sampled)",
		},
		[]string{"channel_id", "provider"},
	)
)

// ---- Cache & Channel Metrics ----

var (
	CacheSyncDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "newapi",
			Name:      "channel_cache_sync_duration_seconds",
			Help:      "Duration of channel cache sync operations",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
	)

	ChannelCacheSize = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "newapi",
		Name:      "channel_cache_size",
		Help:      "Number of channels loaded in local cache",
	})

	PubSubMessagesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "newapi",
			Name:      "channel_cache_pubsub_messages_total",
			Help:      "Number of Pub/Sub sync messages received",
		},
		[]string{"action"},
	)
)

// ---- Rate Limiting Metrics ----

var (
	RateLimitRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "newapi",
			Name:      "rate_limit_requests_total",
			Help:      "Total rate limit checks",
		},
		[]string{"type", "result"}, // type: api, web, critical; result: allowed, rejected
	)
)

// ---- System / Health Metrics ----

var (
	SystemInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "newapi",
			Name:      "system_info",
			Help:      "System information (labels: version, node_name). Value is always 1.",
		},
		[]string{"version", "node_name", "node_type"},
	)

	DiskCacheFilesActive = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "newapi",
		Name:      "disk_cache_files_active",
		Help:      "Number of active disk cache files",
	})

	DiskCacheUsageBytes = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "newapi",
		Name:      "disk_cache_usage_bytes",
		Help:      "Current disk cache usage in bytes",
	})
)
