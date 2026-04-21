package tracing

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.16"
)

// newResource creates a resource with service metadata.
func newResource() *resource.Resource {
	attrs := []resource.Option{
		resource.WithAttributes(
			semconv.ServiceName(common.OtelServiceName),
			semconv.ServiceVersion(common.Version),
		),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
	}

	r, err := NewResource(attrs...)
	if err != nil {
		// fallback to minimal resource
		return resource.Default()
	}
	return r
}

// NewResource is a thin wrapper to avoid import cycle; delegates to resource.New.
func NewResource(opts ...resource.Option) (*resource.Resource, error) {
	return resource.New(opts...)
}
