package observability

/*
// OpenTelemetry setup (commented out, ready for future use)

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// InitOpenTelemetry initializes OpenTelemetry tracing
func InitOpenTelemetry(ctx context.Context, serviceName, jaegerEndpoint string) (*trace.TracerProvider, error) {
	// Create Jaeger exporter
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndpoint)))
	if err != nil {
		return nil, err
	}

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	// Create tracer provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	return tp, nil
}

// ShutdownOpenTelemetry shuts down OpenTelemetry
func ShutdownOpenTelemetry(ctx context.Context, tp *trace.TracerProvider) error {
	return tp.Shutdown(ctx)
}
*/
