// Copyright (c) 2025 True Tickets, Inc.
// SPDX-License-Identifier: MIT

package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Provider provides the telemetry components
type Provider struct {
	tracer         trace.Tracer
	meter          metric.Meter
	tracerProvider *sdktrace.TracerProvider
	shutdownFuncs  []func(context.Context) error
}

// Config holds telemetry configuration
type Config struct {
	ServiceName     string
	TracingEnabled  bool
	TracingEndpoint string
	MetricsEnabled  bool
}

// NewProvider creates a new telemetry Provider
func NewProvider(cfg Config) (*Provider, error) {
	t := &Provider{
		shutdownFuncs: make([]func(context.Context) error, 0),
	}

	// Create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Setup tracing if enabled
	if cfg.TracingEnabled {
		tracerProvider, err := t.setupTracing(cfg, res)
		if err != nil {
			return nil, fmt.Errorf("failed to setup tracing: %w", err)
		}
		t.tracerProvider = tracerProvider
		t.tracer = tracerProvider.Tracer(cfg.ServiceName)
	} else {
		// Use noop tracer
		t.tracer = noop.NewTracerProvider().Tracer(cfg.ServiceName)
	}

	// Setup metrics (for now, always using global meter provider)
	// TODO: Implement conditional metrics setup when metrics exporter is added
	t.meter = otel.GetMeterProvider().Meter(cfg.ServiceName)

	return t, nil
}

// setupTracing sets up the tracing provider
func (t *Provider) setupTracing(cfg Config, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	// Create OTLP exporter
	client := otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(cfg.TracingEndpoint),
		otlptracegrpc.WithInsecure(),
	)

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	t.shutdownFuncs = append(t.shutdownFuncs, tp.Shutdown)

	return tp, nil
}

// Tracer returns the tracer
func (t *Provider) Tracer() trace.Tracer {
	return t.tracer
}

// Meter returns the meter
func (t *Provider) Meter() metric.Meter {
	return t.meter
}

// Shutdown shuts down the telemetry
func (t *Provider) Shutdown(ctx context.Context) error {
	var err error
	for _, fn := range t.shutdownFuncs {
		if shutdownErr := fn(ctx); shutdownErr != nil {
			if err == nil {
				err = shutdownErr
			} else {
				err = fmt.Errorf("%v; %w", err, shutdownErr)
			}
		}
	}
	return err
}
