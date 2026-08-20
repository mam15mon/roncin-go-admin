package telemetry

import (
	"context"
	"fmt"
	"strings"

	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Setup configures the process-wide trace provider. When enabled, endpoint
// and sample ratio are required so an exporter failure cannot be hidden by a
// silent local fallback.
func Setup(ctx context.Context, config *conf.Telemetry, serviceName, serviceVersion string) (func(context.Context) error, error) {
	if config == nil || !config.GetEnabled() {
		return func(context.Context) error { return nil }, nil
	}
	endpoint := strings.TrimSpace(config.GetEndpoint())
	if endpoint == "" {
		return nil, fmt.Errorf("telemetry endpoint is required when telemetry is enabled")
	}
	sampleRatio := config.GetSampleRatio()
	if sampleRatio <= 0 || sampleRatio > 1 {
		return nil, fmt.Errorf("telemetry sample_ratio must be greater than 0 and no greater than 1")
	}

	exporterOptions := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(endpoint)}
	if config.GetInsecure() {
		exporterOptions = append(exporterOptions, otlptracegrpc.WithInsecure())
	}
	exporter, err := otlptracegrpc.New(ctx, exporterOptions...)
	if err != nil {
		return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
	}
	serviceResource, err := resource.New(ctx, resource.WithAttributes(
		attribute.String("service.name", serviceName),
		attribute.String("service.version", serviceVersion),
	))
	if err != nil {
		return nil, fmt.Errorf("create telemetry resource: %w", err)
	}
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(serviceResource),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleRatio))),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	return provider.Shutdown, nil
}
