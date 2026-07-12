package toon

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMarshalTabularArrayReducesCompactJSONSize(t *testing.T) {
	data := map[string]any{
		"context": map[string]any{
			"scope": "dashboard",
			"year":  2026,
		},
		"invoices": []map[string]any{
			{"id": "INV-001", "customer": "Gulf Smelting", "amount": 120.5, "status": "Sent"},
			{"id": "INV-002", "customer": "NGA", "amount": 89.25, "status": "Paid"},
			{"id": "INV-003", "customer": "NPC", "amount": 55.75, "status": "Overdue"},
		},
	}

	encoded, err := Marshal(data)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(encoded, "invoices[3]{amount,customer,id,status}:") {
		t.Fatalf("expected tabular invoices output, got:\n%s", encoded)
	}

	compact, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if len(encoded) >= len(compact) {
		t.Fatalf("expected TOON to be smaller than compact JSON: toon=%d json=%d\n%s", len(encoded), len(compact), encoded)
	}
	t.Logf("sample compact JSON=%d bytes (~%d tokens), TOON=%d bytes (~%d tokens), reduction=%.1f%%",
		len(compact), EstimatedTokens(string(compact)), len(encoded), EstimatedTokens(encoded),
		100*(float64(len(compact)-len(encoded))/float64(len(compact))))
}

func TestEstimatedTokens(t *testing.T) {
	if EstimatedTokens("12345") != 2 {
		t.Fatalf("unexpected token estimate")
	}
}

func BenchmarkMarshalVsCompactJSON(b *testing.B) {
	data := map[string]any{
		"customers": []map[string]any{
			{"id": "C-001", "name": "Gulf Smelting Co.", "grade": "A", "outstanding": 1200.5},
			{"id": "C-002", "name": "NGA", "grade": "A", "outstanding": 950.25},
			{"id": "C-003", "name": "NPC", "grade": "B", "outstanding": 700.75},
			{"id": "C-004", "name": "DPC", "grade": "A", "outstanding": 425.0},
		},
	}
	b.Run("toon", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = Marshal(data)
		}
	})
	b.Run("json", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(data)
		}
	})
}
