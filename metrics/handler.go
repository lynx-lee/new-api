package metrics

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler returns a gin.HandlerFunc that serves Prometheus metrics at /metrics
func Handler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// InitMetrics sets initial system info gauge values.
// Should be called after common.InitEnv() and before starting the HTTP server.
func InitMetrics() {
	SystemInfo.WithLabelValues(common.Version, common.NodeName, nodeTypeLabel()).Set(1)
	common.SysLog("Prometheus metrics initialized")
}

func nodeTypeLabel() string {
	if common.IsMasterNode {
		return "master"
	}
	return "slave"
}

// Middleware returns a Gin middleware that instruments each request with:
// - http_requests_total counter
// - http_request_duration_seconds histogram
// - http_requests_in_flight gauge
// - http_response_size_bytes histogram
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
			// Normalize path to avoid high cardinality from IDs in URLs
			if len(path) > 64 {
				path = path[:64] + "..."
			}
		}

		HTTPRequestInFlight.Inc()
		start := recordTime()

		c.Next()

		status := statusLabel(c.Writer.Status())
		method := c.Request.Method

		HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
		HTTPRequestDuration.WithLabelValues(method, path, status).Observe(sinceSeconds(start))
		HTTPResponseSize.WithLabelValues(method, path).Observe(float64(c.Writer.Size()))
		HTTPRequestInFlight.Dec()
	}
}
