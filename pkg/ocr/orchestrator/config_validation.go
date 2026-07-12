// Config Validation - Bounds checking for HybridPipeline and GPUPreprocessor configurations
//
// This file provides validation methods that ensure configuration values are within safe ranges.
// All invalid values are automatically clamped to the nearest valid boundary, preventing
// undefined behavior from out-of-range parameters.
//
// Validation ranges:
//   - DenoiseStrength: [0, 1] - Controls noise reduction intensity
//   - ContrastFactor: [0.5, 3.0] - Controls contrast enhancement (1.0 = no change)
//   - DenoiseRadius: [0, 5] - Neighborhood size for denoising
//   - MaxConcurrent: [1, 64] - Parallel processing limit
//   - BatchSize: [0, 10000] - Batch processing size (0 = auto Williams optimization)
//   - ParallelImages: [1, 16] - Image processing parallelism
//   - MaxImageSize: [100000, 16000000] - Max pixels (0.1MP to 16MP)
package orchestrator

// Validate validates and clamps HybridConfig values to safe ranges
func (c *HybridConfig) Validate() error {
	// Clamp DenoiseStrength to [0, 1]
	if c.DenoiseStrength < 0 {
		c.DenoiseStrength = 0
	}
	if c.DenoiseStrength > 1 {
		c.DenoiseStrength = 1
	}

	// Clamp ContrastFactor to [0.5, 3.0]
	if c.ContrastFactor < 0.5 {
		c.ContrastFactor = 0.5
	}
	if c.ContrastFactor > 3.0 {
		c.ContrastFactor = 3.0
	}

	// Clamp MaxConcurrent to [1, 64]
	if c.MaxConcurrent < 1 {
		c.MaxConcurrent = 1
	}
	if c.MaxConcurrent > 64 {
		c.MaxConcurrent = 64
	}

	// Clamp BatchSize to [0, 10000] (0 = auto Williams)
	if c.BatchSize < 0 {
		c.BatchSize = 0
	}
	if c.BatchSize > 10000 {
		c.BatchSize = 10000
	}

	return nil
}

// Validate validates and clamps GPUPreprocessConfig values to safe ranges
func (c *GPUPreprocessConfig) Validate() error {
	// DenoiseStrength [0, 1]
	if c.DenoiseStrength < 0 {
		c.DenoiseStrength = 0
	}
	if c.DenoiseStrength > 1 {
		c.DenoiseStrength = 1
	}

	// ContrastFactor [0.5, 3.0]
	if c.ContrastFactor < 0.5 {
		c.ContrastFactor = 0.5
	}
	if c.ContrastFactor > 3.0 {
		c.ContrastFactor = 3.0
	}

	// DenoiseRadius [0, 5]
	if c.DenoiseRadius < 0 {
		c.DenoiseRadius = 0
	}
	if c.DenoiseRadius > 5 {
		c.DenoiseRadius = 5
	}

	// ParallelImages [1, 16]
	if c.ParallelImages < 1 {
		c.ParallelImages = 1
	}
	if c.ParallelImages > 16 {
		c.ParallelImages = 16
	}

	// MaxImageSize [100000, 16000000] (0.1MP to 16MP)
	if c.MaxImageSize < 100000 {
		c.MaxImageSize = 100000
	}
	if c.MaxImageSize > 16000000 {
		c.MaxImageSize = 16000000
	}

	return nil
}
