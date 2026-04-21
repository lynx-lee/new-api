package tracing

import (
	"context"

	"github.com/QuantumNous/new-api/common"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.16.0"
)

// newResource creates a resource with service metadata.
func newResource() *resource.Resource {
	r, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(common.OtelServiceName),
			semconv.ServiceVersionKey.String(common.Version),
		),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
	)
	if err != nil {
		return resource.Default()
	}
	return r
}
