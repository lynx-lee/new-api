package metrics

import "time"

// recordTime records the current time for duration measurement
func recordTime() time.Time {
	return time.Now()
}

// sinceSeconds returns the elapsed seconds since start
func sinceSeconds(start time.Time) float64 {
	return time.Since(start).Seconds()
}

// statusLabel converts HTTP status code to a label string for Prometheus
func statusLabel(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	default:
		return "5xx"
	}
}
