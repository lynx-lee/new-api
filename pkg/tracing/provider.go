// Package tracing provides OpenTelemetry trace provider initialization and configuration.
package tracing

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/QuantumNous/new-api/common"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
)

var (
	globalTracerProvider *trace.TracerProvider
	globalTracer         trace.Tracer
)

const tracerName = "github.com/QuantumNous/new-api"

// Init initializes the global TracerProvider based on environment config.
// Returns a shutdown function that should be called on application exit.
func Init() (func(context.Context) error, error) {
	if !common.OtelEnabled {
		log.Println("[OTEL] tracing disabled, using no-op provider")
		tp := trace.NewTracerProvider()
		globalTracerProvider = tp
		otel.SetTracerProvider(tp)
		globalTracer = tp.Tracer(tracerName)
		return func(ctx context.Context) error { return nil }, nil
	}

	ctx := context.Background()

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(common.OtelExporterEndpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	sampler := trace.TraceIDRatioBased(common.OtelSamplingRatio)

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter,
			trace.WithBatchTimeout(5*time.Second),
			trace.WithMaxExportBatchSize(512),
		),
		trace.WithSampler(sampler),
		trace.WithResource(newResource()),
	)

	globalTracerProvider = tp
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	globalTracer = tp.Tracer(tracerName)

	common.SysLog(fmt.Sprintf("[OTEL] tracing enabled, endpoint=%s, service=%s, sampling=%.2f",
		common.OtelExporterEndpoint, common.OtelServiceName, common.OtelSamplingRatio))

	return tp.Shutdown, nil
}

// Tracer returns the global tracer instance.
func Tracer() trace.Tracer {
	return globalTracer
}

// Provider returns the global TracerProvider.
func Provider() *trace.TracerProvider {
	return globalTracerProvider
}

// StartSpan is a convenience wrapper for starting a span with the global tracer.
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return globalTracer.Start(ctx, name, opts...)
}
