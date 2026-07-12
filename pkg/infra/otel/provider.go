package otel

import (
	"context"
	"errors"
	"io"

	gootel "go.opentelemetry.io/otel"
	stdoutmetric "go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	stdouttrace "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// Config controls OTel behavior.
type Config struct {
	ServiceName    string
	ServiceVersion string
	TraceOutput    io.Writer
	MetricOutput   io.Writer
	Enabled        bool
}

// Provider holds initialized OTel resources.
type Provider struct {
	tracer   trace.Tracer
	meter    metric.Meter
	shutdown func(context.Context) error
	config   Config
}

// New creates and registers an OTel provider.
func New(cfg Config) (*Provider, error) {
	if cfg.ServiceName == "" {
		cfg.ServiceName = "asymmflow"
	}
	if !cfg.Enabled || cfg.TraceOutput == nil || cfg.MetricOutput == nil {
		tracerProvider := tracenoop.NewTracerProvider()
		meterProvider := metricnoop.NewMeterProvider()
		return &Provider{
			tracer:   tracerProvider.Tracer(cfg.ServiceName),
			meter:    meterProvider.Meter(cfg.ServiceName),
			shutdown: func(context.Context) error { return nil },
			config:   cfg,
		}, nil
	}

	traceExporter, err := stdouttrace.New(
		stdouttrace.WithWriter(cfg.TraceOutput),
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		return nil, err
	}
	metricExporter, err := stdoutmetric.New(
		stdoutmetric.WithWriter(cfg.MetricOutput),
		stdoutmetric.WithPrettyPrint(),
	)
	if err != nil {
		return nil, err
	}

	traceProvider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(traceExporter))
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
	)

	gootel.SetTracerProvider(traceProvider)
	gootel.SetMeterProvider(meterProvider)

	return &Provider{
		tracer: traceProvider.Tracer(cfg.ServiceName),
		meter:  meterProvider.Meter(cfg.ServiceName),
		shutdown: func(ctx context.Context) error {
			return errors.Join(traceProvider.Shutdown(ctx), meterProvider.Shutdown(ctx))
		},
		config: cfg,
	}, nil
}

// Tracer returns the configured tracer.
func (p *Provider) Tracer() trace.Tracer {
	return p.tracer
}

// Meter returns the configured meter.
func (p *Provider) Meter() metric.Meter {
	return p.meter
}

// Shutdown gracefully flushes and closes exporters.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p == nil || p.shutdown == nil {
		return nil
	}
	return p.shutdown(ctx)
}
