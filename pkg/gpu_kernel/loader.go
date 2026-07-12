package gpu_kernel

import (
	"fmt"
	"os"
	"path/filepath"
)

// Kernel represents a loadable GPU compute unit
type Kernel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	Path        string `json:"path"`
	MemoryUsage int64  `json:"memory_usage"`
}

// LoadKernels scans the filesystem for the sovereign engines
// APOLLO-GRADE ERROR HANDLING - Path resolution failures are CRITICAL
func LoadKernels() ([]Kernel, error) {
	// Base path assumption: We are in ph_holdings_app/sovereign_ui
	// Kernels are in asymm_mathematical_organism/...
	// Relative path: ../../asymm_mathematical_organism

	// CRITICAL: Check working directory - if this fails, all paths are wrong!
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("CRITICAL: failed to determine working directory: %w", err)
	}

	// Try multiple locations for asymm_mathematical_organism
	candidatePaths := []string{
		filepath.Join(wd, "..", "..", "asymm_mathematical_organism"),
		filepath.Join(filepath.Dir(filepath.Dir(wd)), "asymm_mathematical_organism"),
		os.Getenv("ASYMM_MATH_ROOT"),
	}

	var root string
	for _, p := range candidatePaths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			root = p
			break
		}
	}

	// SAFETY: Verify we found a valid kernel root path
	if root == "" {
		return nil, fmt.Errorf("asymm_mathematical_organism not found in standard locations. Set ASYMM_MATH_ROOT env var to specify location")
	}

	kernels := []Kernel{
		{
			ID:      "K01_SPIRV",
			Name:    "SPIR-V Geometry Engine",
			Version: "v1.2.0",
			Path:    filepath.Join(root, "geometric_consciousness_imaging", "quaternion_os_level_zero_go", "kernels"),
		},
		{
			ID:      "K02_QUAT",
			Name:    "Quaternion Dynamics",
			Version: "v0.9.5",
			Path:    filepath.Join(root, "geometric_consciousness_imaging", "quaternion_os_level_zero_go", "mega_scale_benchmark.exe"),
		},
		{
			ID:      "K03_FLUID",
			Name:    "Navier-Stokes Solver (Python)",
			Version: "v2.1.0",
			Path:    filepath.Join(root, "geometric_consciousness_imaging", "qgif_visualizer.py"),
		},
	}

	loadedKernels := make([]Kernel, 0)

	for _, k := range kernels {
		if _, err := os.Stat(k.Path); err == nil {
			k.Status = "ACTIVE"
			k.MemoryUsage = 1024 * 1024 * 128 // Placeholder for now, would read Process RAM in real scenario
		} else {
			k.Status = "MISSING"
			k.MemoryUsage = 0
		}
		loadedKernels = append(loadedKernels, k)
	}

	return loadedKernels, nil
}

func InjectKernel(id string, params map[string]any) error {
	// In a real scenario, this would use exec.Command to pass JSON args to the exe/python script
	fmt.Printf("Injecting parameters into kernel %s: %v\n", id, params)
	return nil
}
