package middleware

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/pkg/tracing"

	"github.com/gin-gonic/gin"
	otelgin "github.com/opentelemetry/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const otelSpanKey = "otel_span"

// TracingMiddleware creates a Gin middleware for OpenTelemetry distributed tracing.
// When OTEL_ENABLED=false, this is a no-op middleware with zero overhead.
func TracingMiddleware() gin.HandlerFunc {
	if !common.OtelEnabled {
		return func(c *gin.Context) { c.Next() }
	}

	handler := otelgin.Middleware(common.OtelServiceName,
		otelgin.WithSpanAttributes(
			attribute.String("service", common.OtelServiceName),
		),
	)
	return handler
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
