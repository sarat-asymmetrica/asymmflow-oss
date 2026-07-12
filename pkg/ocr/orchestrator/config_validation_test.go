package orchestrator

import "testing"

func TestHybridConfigValidate(t *testing.T) {
	tests := []struct {
		name     string
		input    HybridConfig
		expected HybridConfig
	}{
		{
			name: "Valid config - no changes",
			input: HybridConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  1.2,
				MaxConcurrent:   4,
				BatchSize:       100,
			},
			expected: HybridConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  1.2,
				MaxConcurrent:   4,
				BatchSize:       100,
			},
		},
		{
			name: "DenoiseStrength too low - clamped to 0",
			input: HybridConfig{
				DenoiseStrength: -0.5,
				ContrastFactor:  1.2,
				MaxConcurrent:   4,
				BatchSize:       100,
			},
			expected: HybridConfig{
				DenoiseStrength: 0.0,
				ContrastFactor:  1.2,
				MaxConcurrent:   4,
				BatchSize:       100,
			},
		},
		{
			name: "DenoiseStrength too high - clamped to 1",
			input: HybridConfig{
				DenoiseStrength: 1.5,
				ContrastFactor:  1.2,
				MaxConcurrent:   4,
				BatchSize:       100,
			},
			expected: HybridConfig{
				DenoiseStrength: 1.0,
				ContrastFactor:  1.2,
				MaxConcurrent:   4,
				BatchSize:       100,
			},
		},
		{
			name: "ContrastFactor too low - clamped to 0.5",
			input: HybridConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  0.1,
				MaxConcurrent:   4,
				BatchSize:       100,
			},
			expected: HybridConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  0.5,
				MaxConcurrent:   4,
				BatchSize:       100,
			},
		},
		{
			name: "ContrastFactor too high - clamped to 3.0",
			input: HybridConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  5.0,
				MaxConcurrent:   4,
				BatchSize:       100,
			},
			expected: HybridConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  3.0,
				MaxConcurrent:   4,
				BatchSize:       100,
			},
		},
		{
			name: "MaxConcurrent too low - clamped to 1",
			input: HybridConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  1.2,
				MaxConcurrent:   0,
				BatchSize:       100,
			},
			expected: HybridConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  1.2,
				MaxConcurrent:   1,
				BatchSize:       100,
			},
		},
		{
			name: "MaxConcurrent too high - clamped to 64",
			input: HybridConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  1.2,
				MaxConcurrent:   100,
				BatchSize:       100,
			},
			expected: HybridConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  1.2,
				MaxConcurrent:   64,
				BatchSize:       100,
			},
		},
		{
			name: "BatchSize negative - clamped to 0",
			input: HybridConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  1.2,
				MaxConcurrent:   4,
				BatchSize:       -10,
			},
			expected: HybridConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  1.2,
				MaxConcurrent:   4,
				BatchSize:       0,
			},
		},
		{
			name: "BatchSize too high - clamped to 10000",
			input: HybridConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  1.2,
				MaxConcurrent:   4,
				BatchSize:       50000,
			},
			expected: HybridConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  1.2,
				MaxConcurrent:   4,
				BatchSize:       10000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.input
			err := config.Validate()
			if err != nil {
				t.Errorf("Validate() returned unexpected error: %v", err)
			}

			if config.DenoiseStrength != tt.expected.DenoiseStrength {
				t.Errorf("DenoiseStrength = %v, want %v", config.DenoiseStrength, tt.expected.DenoiseStrength)
			}
			if config.ContrastFactor != tt.expected.ContrastFactor {
				t.Errorf("ContrastFactor = %v, want %v", config.ContrastFactor, tt.expected.ContrastFactor)
			}
			if config.MaxConcurrent != tt.expected.MaxConcurrent {
				t.Errorf("MaxConcurrent = %v, want %v", config.MaxConcurrent, tt.expected.MaxConcurrent)
			}
			if config.BatchSize != tt.expected.BatchSize {
				t.Errorf("BatchSize = %v, want %v", config.BatchSize, tt.expected.BatchSize)
			}
		})
	}
}

func TestGPUPreprocessConfigValidate(t *testing.T) {
	tests := []struct {
		name     string
		input    GPUPreprocessConfig
		expected GPUPreprocessConfig
	}{
		{
			name: "Valid config - no changes",
			input: GPUPreprocessConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  1.2,
				DenoiseRadius:   1,
				ParallelImages:  2,
				MaxImageSize:    4000000,
			},
			expected: GPUPreprocessConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  1.2,
				DenoiseRadius:   1,
				ParallelImages:  2,
				MaxImageSize:    4000000,
			},
		},
		{
			name: "DenoiseRadius too high - clamped to 5",
			input: GPUPreprocessConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  1.2,
				DenoiseRadius:   10,
				ParallelImages:  2,
				MaxImageSize:    4000000,
			},
			expected: GPUPreprocessConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  1.2,
				DenoiseRadius:   5,
				ParallelImages:  2,
				MaxImageSize:    4000000,
			},
		},
		{
			name: "ParallelImages too low - clamped to 1",
			input: GPUPreprocessConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  1.2,
				DenoiseRadius:   1,
				ParallelImages:  0,
				MaxImageSize:    4000000,
			},
			expected: GPUPreprocessConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  1.2,
				DenoiseRadius:   1,
				ParallelImages:  1,
				MaxImageSize:    4000000,
			},
		},
		{
			name: "ParallelImages too high - clamped to 16",
			input: GPUPreprocessConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  1.2,
				DenoiseRadius:   1,
				ParallelImages:  32,
				MaxImageSize:    4000000,
			},
			expected: GPUPreprocessConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  1.2,
				DenoiseRadius:   1,
				ParallelImages:  16,
				MaxImageSize:    4000000,
			},
		},
		{
			name: "MaxImageSize too low - clamped to 100000",
			input: GPUPreprocessConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  1.2,
				DenoiseRadius:   1,
				ParallelImages:  2,
				MaxImageSize:    1000,
			},
			expected: GPUPreprocessConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  1.2,
				DenoiseRadius:   1,
				ParallelImages:  2,
				MaxImageSize:    100000,
			},
		},
		{
			name: "MaxImageSize too high - clamped to 16000000",
			input: GPUPreprocessConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  1.2,
				DenoiseRadius:   1,
				ParallelImages:  2,
				MaxImageSize:    50000000,
			},
			expected: GPUPreprocessConfig{
				DenoiseStrength: 0.5,
				ContrastFactor:  1.2,
				DenoiseRadius:   1,
				ParallelImages:  2,
				MaxImageSize:    16000000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.input
			err := config.Validate()
			if err != nil {
				t.Errorf("Validate() returned unexpected error: %v", err)
			}

			if config.DenoiseStrength != tt.expected.DenoiseStrength {
				t.Errorf("DenoiseStrength = %v, want %v", config.DenoiseStrength, tt.expected.DenoiseStrength)
			}
			if config.ContrastFactor != tt.expected.ContrastFactor {
				t.Errorf("ContrastFactor = %v, want %v", config.ContrastFactor, tt.expected.ContrastFactor)
			}
			if config.DenoiseRadius != tt.expected.DenoiseRadius {
				t.Errorf("DenoiseRadius = %v, want %v", config.DenoiseRadius, tt.expected.DenoiseRadius)
			}
			if config.ParallelImages != tt.expected.ParallelImages {
				t.Errorf("ParallelImages = %v, want %v", config.ParallelImages, tt.expected.ParallelImages)
			}
			if config.MaxImageSize != tt.expected.MaxImageSize {
				t.Errorf("MaxImageSize = %v, want %v", config.MaxImageSize, tt.expected.MaxImageSize)
			}
		})
	}
}
