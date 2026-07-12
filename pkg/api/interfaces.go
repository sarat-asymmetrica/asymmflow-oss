package api

import (
	"context"
)

// UIAlchemyService interface defines UI generation operations
// This DECOUPLES business logic from UI generation (Violation #3 Fix!)
type UIAlchemyService interface {
	// GetScreen retrieves a pre-generated screen layout
	GetScreen(ctx context.Context, screenID string, contextVec ContextVector) (*ScreenLayout, error)

	// GenerateScreen creates a new screen layout dynamically
	GenerateScreen(ctx context.Context, req GenerateScreenRequest) (*ScreenLayout, error)

	// ListScreens returns all available screen IDs
	ListScreens(ctx context.Context) ([]string, error)
}

// GPUService interface defines GPU computation operations
// Externalizes the 7K LOC GPU stack (Violation #5 Fix!)
type GPUService interface {
	// ComputeSLERP performs quaternion SLERP interpolation
	ComputeSLERP(ctx context.Context, req SLERPRequest) (*SLERPResponse, error)

	// GetStatus returns GPU availability and performance metrics
	GetStatus(ctx context.Context) (*GPUStatus, error)

	// MultiplyQuaternions performs quaternion multiplication
	MultiplyQuaternions(ctx context.Context, q1, q2 Quaternion) (*Quaternion, error)

	// NormalizeQuaternion normalizes a quaternion to unit length
	NormalizeQuaternion(ctx context.Context, q Quaternion) (*Quaternion, error)
}

// VQCService interface defines VQC engine operations
// Externalizes the 2K LOC VQC engines (Violation #6 Fix!)
type VQCService interface {
	// Optimize runs the VQC optimization engine (10M candidates/sec)
	Optimize(ctx context.Context, req OptimizationRequest) (*OptimizationResponse, error)

	// Classify runs the VQC cancer classifier (71M genes/sec)
	Classify(ctx context.Context, req ClassificationRequest) (*ClassificationResponse, error)

	// AnalyzeClimate runs the VQC climate analyzer (13.7M records/sec)
	AnalyzeClimate(ctx context.Context, req ClimateRequest) (*ClimateResponse, error)
}

// SurvivalGardenService interface defines business simulation operations
// Externalizes the survival garden physics (Violation #7 Fix!)
type SurvivalGardenService interface {
	// Simulate runs the survival garden cash flow simulation
	Simulate(ctx context.Context, params SimulationParams) (*SimulationResult, error)

	// GetEquilibrium calculates the 87.532% thermodynamic attractor
	GetEquilibrium(ctx context.Context, params SimulationParams) (*EquilibriumResult, error)
}

// VisualRegimeService interface defines visual regime operations
// Replaces global DefaultRegimes variable (Violation #2 Fix!)
type VisualRegimeService interface {
	// GetRegime retrieves a visual regime by name
	GetRegime(ctx context.Context, name string) (*VisualRegime, error)

	// ListRegimes returns all available regimes
	ListRegimes(ctx context.Context) ([]VisualRegime, error)

	// ComputeRegime calculates a regime based on context vector
	ComputeRegime(ctx context.Context, contextVec ContextVector) (*VisualRegime, error)
}

// ========== REQUEST/RESPONSE TYPES ==========

// ContextVector represents business/user context
type ContextVector struct {
	TimeOfDay string  `json:"time_of_day"` // "morning", "afternoon", "evening"
	FlowRate  float64 `json:"flow_rate"`   // Activity level (MB/s)
	Urgency   float64 `json:"urgency"`     // Risk level (0.0 - 1.0)
}

// GenerateScreenRequest for dynamic screen generation
type GenerateScreenRequest struct {
	ScreenID string         `json:"screen_id"`
	Context  ContextVector  `json:"context"`
	Data     map[string]any `json:"data,omitempty"` // Optional business data
}

// ScreenLayout represents a complete UI layout
type ScreenLayout struct {
	ID           string       `json:"id"`
	Title        string       `json:"title"`
	Type         string       `json:"type"`
	Components   []Component  `json:"components"`
	Theme        ThemeHints   `json:"theme"`
	GridTemplate string       `json:"grid_template"`
	Regime       VisualRegime `json:"regime"`
}

// Component represents a UI component
type Component struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Data     map[string]any `json:"data"`
	GridArea string         `json:"grid_area"`
	Regime   int            `json:"regime"`
}

// ThemeHints provides color palette hints
type ThemeHints struct {
	PrimaryColor    string `json:"primary_color"`
	AccentColor     string `json:"accent_color"`
	BackgroundColor string `json:"background_color"`
}

// VisualRegime defines visual state
type VisualRegime struct {
	Name           string             `json:"name"`
	PrimaryColor   string             `json:"primary_color"`
	SecondaryColor string             `json:"secondary_color"`
	Geometry       GeometryConfig     `json:"geometry"`
	Physics        PhysicsConfig      `json:"physics"`
	ShaderUniforms map[string]float32 `json:"shader_uniforms"`
}

// GeometryConfig defines shape parameters
type GeometryConfig struct {
	Type       string  `json:"type"`
	Complexity float64 `json:"complexity"`
	Roughness  float64 `json:"roughness"`
	Metalness  float64 `json:"metalness"`
}

// PhysicsConfig defines movement parameters
type PhysicsConfig struct {
	FlowRate   float64 `json:"flow_rate"`
	Turbulence float64 `json:"turbulence"`
	Gravity    float64 `json:"gravity"`
	Viscosity  float64 `json:"viscosity"`
}

// Quaternion represents a quaternion (W, X, Y, Z)
type Quaternion struct {
	W float64 `json:"w"`
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// SLERPRequest for quaternion interpolation
type SLERPRequest struct {
	Q1 Quaternion `json:"q1"` // Start quaternion
	Q2 Quaternion `json:"q2"` // End quaternion
	T  float64    `json:"t"`  // Interpolation factor (0-1)
}

// SLERPResponse with performance metrics
type SLERPResponse struct {
	Result     Quaternion `json:"result"`
	ComputeMs  float64    `json:"compute_ms"`
	DeviceUsed string     `json:"device_used"` // "GPU" or "CPU_FALLBACK"
}

// GPUStatus provides GPU availability info
type GPUStatus struct {
	Available     bool    `json:"available"`
	DeviceName    string  `json:"device_name"`
	KernelsLoaded int     `json:"kernels_loaded"`
	Performance   float64 `json:"performance"` // Operations per second
}

// OptimizationRequest for VQC optimizer
type OptimizationRequest struct {
	Candidates []Candidate `json:"candidates"`
	MaxIters   int         `json:"max_iters"`
}

// Candidate for optimization
type Candidate struct {
	ID    string    `json:"id"`
	Value []float64 `json:"value"` // N-dimensional vector
	Score float64   `json:"score"`
}

// OptimizationResponse with results
type OptimizationResponse struct {
	BestCandidate Candidate `json:"best_candidate"`
	FinalScore    float64   `json:"final_score"`
	Iterations    int       `json:"iterations"`
	TimeMS        float64   `json:"time_ms"`
}

// ClassificationRequest for VQC cancer classifier
type ClassificationRequest struct {
	GeneExpression []float64 `json:"gene_expression"` // 7,129 genes
	PatientID      string    `json:"patient_id"`
}

// ClassificationResponse with cancer prediction
type ClassificationResponse struct {
	Class      string  `json:"class"`      // "ALL" or "AML"
	Confidence float64 `json:"confidence"` // 0-1
	TimeMS     float64 `json:"time_ms"`
}

// ClimateRequest for VQC climate analyzer
type ClimateRequest struct {
	Temperatures []float64 `json:"temperatures"` // Time series
	Location     string    `json:"location"`
}

// ClimateResponse with regime classification
type ClimateResponse struct {
	Regime int     `json:"regime"` // 1 (WARMING), 2 (STABLE), 3 (COOLING)
	Trend  float64 `json:"trend"`  // °C per year
	TimeMS float64 `json:"time_ms"`
}

// SimulationParams for survival garden
type SimulationParams struct {
	CashRunway       float64   `json:"cash_runway"`  // Months
	MonthlyBurn      float64   `json:"monthly_burn"` // BHD/month
	Expenses         []Expense `json:"expenses"`
	MonthsToSimulate int       `json:"months_to_simulate"`
	GPUAccelerate    bool      `json:"gpu_accelerate"`
}

// Expense represents a business expense
type Expense struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"` // BHD/month
	Weight float64 `json:"weight"` // Visual weight
}

// SimulationResult contains garden states over time
type SimulationResult struct {
	States []GardenState `json:"states"`
	TimeMS float64       `json:"time_ms"`
}

// GardenState represents survival garden at a point in time
type GardenState struct {
	WaterLevel    float64   `json:"waterLevel"`
	StoneHeights  []float64 `json:"stoneHeights"`
	ParticleCount int       `json:"particleCount"`
	Regime        int       `json:"regime"`
	Temperature   float64   `json:"temperature"`
	Turbulence    float64   `json:"turbulence"`
}

// EquilibriumResult contains the 87.532% attractor
type EquilibriumResult struct {
	Attractor   float64 `json:"attractor"` // 0.87532
	Sustainable bool    `json:"sustainable"`
	TimeMS      float64 `json:"time_ms"`
}
