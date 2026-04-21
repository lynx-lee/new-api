package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/pkg/tracing"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.16.0"
	"go.opentelemetry.io/otel/trace"
)

const otelSpanKey = "otel_span"

var propagator = propagation.TraceContext{}

// TracingMiddleware creates a Gin middleware for OpenTelemetry distributed tracing.
// When OTEL_ENABLED=false, this is a no-op middleware with zero overhead.
// Custom implementation to avoid external otelgin dependency issues.
func TracingMiddleware() gin.HandlerFunc {
	if !common.OtelEnabled {
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		start := time.Now()
		ctx := propagatorExtract(c.Request)
		ctx, span := tracing.StartSpan(ctx, "http_request",
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPRequestMethodKey.String(c.Request.Method),
				semconv.URLFull(c.Request.URL.String()),
				semconv.NetworkProtocolVersion("1.1"),
				semconv.UserAgentOriginal(c.Request.UserAgent()),
				attribute.String("service", common.OtelServiceName),
				attribute.String("client.ip", c.ClientIP()),
			),
		)
		defer span.End()

		c.Request = c.Request.WithContext(ctx)
		c.Set(otelSpanKey, span)

		c.Next()

		attributes := []attribute.KeyValue{
			semconv.HTTPResponseStatusCode(c.Writer.Status()),
			attribute.Float64("http.duration_ms", float64(time.Since(start).Microseconds())/1000),
		}

		if len(c.Errors) > 0 {
			span.SetStatus(codes.Error, c.Errors.String())
			span.RecordError(fmt.Errorf("gin errors: %s", c.Errors.String()))
			attributes = append(attributes,
				attribute.Int("error.count", len(c.Errors)),
				attribute.String("error.message", c.Errors.Last().Error()),
			)
		} else if c.Writer.Status() >= http.StatusBadRequest {
			span.SetStatus(codes.Error, strconv.Itoa(c.Writer.Status()))
		} else {
			span.SetStatus(codes.Ok, "")
		}

		if requestId := c.GetString(common.RequestIdKey); requestId != "" {
			attributes = append(attributes, attribute.String("request.id", requestId))
		}

		span.SetAttributes(attributes...)

		propagatorInject(ctx, c.Writer)
	}
}

// propagatorExtract extracts trace context from incoming HTTP request headers.
func propagatorExtract(r *http.Request) context.Context {
	return propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
}

// propagatorInject injects trace context into outgoing HTTP response headers.
func propagatorInject(ctx context.Context, w http.ResponseWriter) {
	propagator.Inject(ctx, propagation.HeaderCarrier(w.Header()))
}

// EnhanceSpanWithRequestContext adds business context attributes to the current span.
// Should be called after auth/distribute middleware to capture user, channel, model info.
func EnhanceSpanWithRequestContext(c *gin.Context) {
	if !common.OtelEnabled {
		return
	}

	ctx := c.Request.Context()
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	attrs := []attribute.KeyValue{}

	if requestId := c.GetString(common.RequestIdKey); requestId != "" {
		attrs = append(attrs, attribute.String("request.id", requestId))
	}
	if userId := c.GetInt("id"); userId > 0 {
		attrs = append(attrs, attribute.Int("user.id", userId))
	}
	if groupId := common.GetContextKeyString(c, constant.ContextKeyUsingGroup); groupId != "" {
		attrs = append(attrs, attribute.String("group", groupId))
	}
	if channelId := common.GetContextKeyString(c, constant.ContextKeyChannelId); channelId != "" {
		attrs = append(attrs, attribute.String("channel.id", channelId))
	}
	if channelType := common.GetContextKeyInt(c, constant.ContextKeyChannelType); channelType > 0 {
		attrs = append(attrs, attribute.Int("channel.type", channelType))
	}
	if modelName := c.GetString("original_model"); modelName != "" {
		attrs = append(attrs, attribute.String("model.name", modelName))
	}

	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
}

// SetSpanError marks the current span as errored with message.
func SetSpanError(c *gin.Context, errMsg string) {
	if !common.OtelEnabled {
		return
	}
	span := trace.SpanFromContext(c.Request.Context())
	if span.IsRecording() {
		span.SetStatus(codes.Error, errMsg)
		span.RecordError(fmtErr(errMsg))
		span.SetAttribute("error.message", errMsg)
	}
}

// StartRelaySpan starts a child span for upstream relay requests.
// Returns (context, span) - caller must call span.End().
func StartRelaySpan(c *gin.Context, provider, modelName, upstreamURL string) (context.Context, trace.Span) {
	if !common.OtelEnabled {
		return c.Request.Context(), trace.NoopSpan{}
	}

	ctx, span := tracing.StartSpan(c.Request.Context(), "relay_upstream",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("relay.provider", provider),
			attribute.String("relay.model", modelName),
			attribute.String("relay.upstream_url", upstreamURL),
			attribute.Int("relay.channel_id",
				common.GetContextKeyInt(c, constant.ContextKeyChannelId)),
		),
	)
	return ctx, span
}

func fmtErr(s string) error { return errStr{s} }

type errStr struct{ s string }

func (e errStr) Error() string { return e.s }
