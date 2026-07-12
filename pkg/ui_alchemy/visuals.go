package ui_alchemy

// VisualRegime defines the high-level visual state of the application
// derived from the business context.
type VisualRegime struct {
	Name           string             `json:"name"`            // e.g., "Deep Ocean", "Solar Flare", "Zen Garden"
	PrimaryColor   string             `json:"primary_color"`   // Hex code
	SecondaryColor string             `json:"secondary_color"` // Hex code
	Geometry       GeometryConfig     `json:"geometry"`        // Shape parameters
	Physics        PhysicsConfig      `json:"physics"`         // Movement parameters
	ShaderUniforms map[string]float32 `json:"shader_uniforms"` // Raw values for WebGL
}

type GeometryConfig struct {
	Type       string  `json:"type"`       // "Sphere", "Torus", "Icosahedron", "FluidPlane"
	Complexity float64 `json:"complexity"` // Vertex count / tessellation level
	Roughness  float64 `json:"roughness"`  // Surface texture
	Metalness  float64 `json:"metalness"`  // Reflectivity
}

type PhysicsConfig struct {
	FlowRate   float64 `json:"flow_rate"`  // Speed of animation
	Turbulence float64 `json:"turbulence"` // Randomness/Noise amplitude
	Gravity    float64 `json:"gravity"`    // Directional pull
	Viscosity  float64 `json:"viscosity"`  // Resistance to motion
}

// DefaultRegimes provides a palette of predefined states
var DefaultRegimes = map[string]VisualRegime{
	"MorningCalm": {
		Name:           "Morning Calm",
		PrimaryColor:   "#fdfbf7", // Paper
		SecondaryColor: "#e2e8f0", // Slate-200
		Geometry: GeometryConfig{
			Type:       "FluidPlane",
			Complexity: 0.2,
			Roughness:  0.9,
			Metalness:  0.1,
		},
		Physics: PhysicsConfig{
			FlowRate:   0.1,
			Turbulence: 0.05,
			Gravity:    0.0,
			Viscosity:  0.8,
		},
		ShaderUniforms: map[string]float32{
			"u_time_scale": 0.1,
			"u_distortion": 0.1,
		},
	},
	"HighVelocity": {
		Name:           "High Velocity",
		PrimaryColor:   "#0f172a", // Slate-900
		SecondaryColor: "#3b82f6", // Blue-500
		Geometry: GeometryConfig{
			Type:       "Torus",
			Complexity: 0.8,
			Roughness:  0.2,
			Metalness:  0.9,
		},
		Physics: PhysicsConfig{
			FlowRate:   2.5,
			Turbulence: 0.4,
			Gravity:    0.5,
			Viscosity:  0.1,
		},
		ShaderUniforms: map[string]float32{
			"u_time_scale": 2.0,
			"u_distortion": 1.5,
		},
	},
}

// GetVisualRegime calculates the visual state based on the context vector
func GetVisualRegime(ctx ContextVector) VisualRegime {
	// Start with a base regime
	regime := DefaultRegimes["MorningCalm"]

	// Morph based on context
	// 1. Time of day affects color and lighting
	if ctx.TimeOfDay == "evening" || ctx.TimeOfDay == "night" {
		// Night mode / Deep thought
		regime.PrimaryColor = "#0f172a"
		regime.SecondaryColor = "#475569"
		regime.Geometry.Metalness = 0.8
	}

	// 2. Flow Rate affects physics
	regime.Physics.FlowRate = ctx.FlowRate * 2.0
	regime.Physics.Turbulence = ctx.FlowRate * 0.5

	// 3. Urgency affects distortion
	if ctx.Urgency > 0.7 {
		regime.ShaderUniforms["u_distortion"] = 2.0
		regime.Geometry.Roughness = 0.8 // Jagged edges
	}

	return regime
}
