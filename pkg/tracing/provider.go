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
	oteltrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.16"
)

var (
	globalTracerProvider *oteltrace.TracerProvider
	globalTracer         otel.Tracer
)

const tracerName = "github.com/QuantumNous/new-api"

// Init initializes the global TracerProvider based on environment config.
// Returns a shutdown function that should be called on application exit.
func Init() (func(context.Context) error, error) {
	if !common.OtelEnabled {
		log.Println("[OTEL] tracing disabled, using no-op provider")
		globalTracerProvider = oteltrace.NewTracerProvider()
		otel.SetTracerProvider(globalTracerProvider)
		globalTracer = globalTracerProvider.Tracer(tracerName)
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

	sampler := oteltrace.TraceIDRatioBased(common.OtelSamplingRatio)

	tp := oteltrace.NewTracerProvider(
		oteltrace.WithBatcher(exporter,
			oteltrace.WithBatchTimeout(5*time.Second),
			oteltrace.WithMaxExportBatchSize(512),
		),
		oteltrace.WithSampler(sampler),
		oteltrace.WithResource(newResource()),
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
func Tracer() otel.Tracer {
	return globalTracer
}

// Provider returns the global TracerProvider.
func Provider() *oteltrace.TracerProvider {
	return globalTracerProvider
}
