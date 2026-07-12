package otel

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// StartDomainSpan begins a span with domain context.
func (p *Provider) StartDomainSpan(ctx context.Context, domain, operation string) (context.Context, func()) {
	if p == nil || p.tracer == nil {
		return ctx, func() {}
	}
	ctx, span := p.tracer.Start(ctx, domain+"."+operation,
		trace.WithAttributes(
			attribute.String("domain", domain),
			attribute.String("operation", operation),
		))
	return ctx, func() { span.End() }
}

// RecordRegime records the current system regime as metrics.
func (p *Provider) RecordRegime(ctx context.Context, domain string, r1, r2, r3 float64) {
	if p == nil || p.meter == nil {
		return
	}
	p.recordGauge(ctx, "asymmflow.regime.exploration", r1, domain, "")
	p.recordGauge(ctx, "asymmflow.regime.optimization", r2, domain, "")
	p.recordGauge(ctx, "asymmflow.regime.stabilization", r3, domain, "")
}

// RecordLatency records an operation's duration.
func (p *Provider) RecordLatency(ctx context.Context, domain, operation string, durationMs float64) {
	if p == nil || p.meter == nil {
		return
	}
	histogram, err := p.meter.Float64Histogram(
		"asymmflow.operation.latency",
		metric.WithUnit("ms"),
		metric.WithDescription("Operation latency in milliseconds"),
	)
	if err != nil {
		return
	}
	histogram.Record(ctx, durationMs, metric.WithAttributes(
		attribute.String("domain", domain),
		attribute.String("operation", operation),
	))
}

// RecordCount increments a counter for an operation.
func (p *Provider) RecordCount(ctx context.Context, domain, operation string, count int64) {
	if p == nil || p.meter == nil {
		return
	}
	counter, err := p.meter.Int64Counter(
		"asymmflow.operation.count",
		metric.WithDescription("Operation count"),
	)
	if err != nil {
		return
	}
	counter.Add(ctx, count, metric.WithAttributes(
		attribute.String("domain", domain),
		attribute.String("operation", operation),
	))
}

func (p *Provider) recordGauge(ctx context.Context, name string, value float64, domain, operation string) {
	gauge, err := p.meter.Float64Gauge(name, metric.WithDescription("Three-regime system metric"))
	if err != nil {
		return
	}
	attrs := []attribute.KeyValue{attribute.String("domain", domain)}
	if operation != "" {
		attrs = append(attrs, attribute.String("operation", operation))
	}
	gauge.Record(ctx, value, metric.WithAttributes(attrs...))
}
