package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Propagator returns the global text map propagator (W3C TraceContext).
func Propagator() propagation.TextMapPropagator {
	return otel.GetTextMapPropagator()
}

// Extract extracts trace context from a carrier map.
func Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	return Propagator().Extract(ctx, carrier)
}

// Inject injects trace context into a carrier map.
func Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	Propagator().Inject(ctx, carrier)
}

// SpanFromContext returns the current span from context, or a noop span if none exists.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}
