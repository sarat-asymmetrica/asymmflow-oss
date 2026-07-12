package vedic

import "testing"

func TestWilliamsBatchSize(t *testing.T) {
	tests := []struct {
		n    int64
		want int64
	}{
		{0, 1},
		{1, 1},
		{100, 66},
		{1000, 315},
	}

	for _, tt := range tests {
		if got := WilliamsBatchSize(tt.n); got != tt.want {
			t.Fatalf("WilliamsBatchSize(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}

	got := WilliamsBatchSize(1000000)
	if got < 17100 || got > 20900 {
		t.Fatalf("WilliamsBatchSize(1000000) = %d, want within 10%% of 19000", got)
	}
}
