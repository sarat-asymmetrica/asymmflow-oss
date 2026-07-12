package health

import (
	"testing"

	"ph_holdings_app/pkg/math/trident"
)

func TestNewMonitor(t *testing.T) {
	if got := NewMonitor(); got == nil {
		t.Fatalf("NewMonitor returned nil")
	}
}

func TestHealthySystemIsStabilization(t *testing.T) {
	monitor := NewMonitor()
	for i := 0; i < 100; i++ {
		monitor.RecordRequest(25, false)
	}

	health := monitor.Health()
	if health.Regime != trident.RegimeStabilization {
		t.Fatalf("regime = %s, want Stabilization", health.Regime)
	}
}

func TestHighErrorRateIsExploration(t *testing.T) {
	monitor := NewMonitor()
	for i := 0; i < 10; i++ {
		monitor.RecordRequest(100, i%2 == 0)
	}

	health := monitor.Health()
	if health.Regime != trident.RegimeExploration {
		t.Fatalf("regime = %s, want Exploration", health.Regime)
	}
}

func TestHealthScore(t *testing.T) {
	healthy := NewMonitor()
	for i := 0; i < 100; i++ {
		healthy.RecordRequest(25, false)
	}
	if score := healthy.Health().Score; score <= 0.8 {
		t.Fatalf("healthy score = %f, want > 0.8", score)
	}

	unhealthy := NewMonitor()
	for i := 0; i < 10; i++ {
		unhealthy.RecordRequest(250, i%2 == 0)
	}
	if score := unhealthy.Health().Score; score >= 0.5 {
		t.Fatalf("unhealthy score = %f, want < 0.5", score)
	}
}

func TestReset(t *testing.T) {
	monitor := NewMonitor()
	monitor.RecordRequest(1000, true)
	monitor.Reset()

	health := monitor.Health()
	if health.Regime != trident.RegimeStabilization {
		t.Fatalf("regime after reset = %s, want Stabilization", health.Regime)
	}
	if health.AvgLatencyMs != 0 {
		t.Fatalf("avg latency after reset = %f, want 0", health.AvgLatencyMs)
	}
}
