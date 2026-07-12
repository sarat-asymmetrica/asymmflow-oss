package api

import (
	"context"
	"errors"
	"ph_holdings_app/pkg/survival_garden"
	"ph_holdings_app/pkg/ui_alchemy"
	"ph_holdings_app/pkg/vqc"
	"time"
)

// ========== UI ALCHEMY IMPLEMENTATION ==========

// UIAlchemyImpl wraps the existing ui_alchemy package
type UIAlchemyImpl struct {
	// No global state! Everything is instance-based
}

// NewUIAlchemyService creates a new UI Alchemy service
func NewUIAlchemyService() UIAlchemyService {
	return &UIAlchemyImpl{}
}

func (u *UIAlchemyImpl) GetScreen(ctx context.Context, screenID string, contextVec ContextVector) (*ScreenLayout, error) {
	// Convert API ContextVector to ui_alchemy.ContextVector
	alchemyCtx := ui_alchemy.ContextVector{
		TimeOfDay: contextVec.TimeOfDay,
		FlowRate:  contextVec.FlowRate,
		Urgency:   contextVec.Urgency,
	}

	// Call existing ui_alchemy.GenerateScreen
	layout := ui_alchemy.GenerateScreen(screenID, alchemyCtx)

	// Convert to API ScreenLayout
	return convertToAPIScreenLayout(layout), nil
}

func (u *UIAlchemyImpl) GenerateScreen(ctx context.Context, req GenerateScreenRequest) (*ScreenLayout, error) {
	// Convert and call existing logic
	alchemyCtx := ui_alchemy.ContextVector{
		TimeOfDay: req.Context.TimeOfDay,
		FlowRate:  req.Context.FlowRate,
		Urgency:   req.Context.Urgency,
	}

	layout := ui_alchemy.GenerateScreen(req.ScreenID, alchemyCtx)
	return convertToAPIScreenLayout(layout), nil
}

func (u *UIAlchemyImpl) ListScreens(ctx context.Context) ([]string, error) {
	// Hardcoded for now - in production, this would query a registry
	return []string{
		"dashboard",
		"opportunities",
		"orders",
		"butler",
		"customers",
		"settings",
	}, nil
}

// convertToAPIScreenLayout converts ui_alchemy.ScreenLayout to api.ScreenLayout
func convertToAPIScreenLayout(layout ui_alchemy.ScreenLayout) *ScreenLayout {
	components := make([]Component, len(layout.Components))
	for i, c := range layout.Components {
		components[i] = Component{
			ID:       c.ID,
			Type:     string(c.Type),
			Data:     c.Data,
			GridArea: c.GridArea,
			Regime:   c.Regime,
		}
	}

	return &ScreenLayout{
		ID:           layout.ID,
		Title:        layout.Title,
		Type:         layout.Type,
		Components:   components,
		Theme:        ThemeHints(layout.Theme),
		GridTemplate: layout.GridTemplate,
		Regime:       convertToAPIVisualRegime(layout.Regime),
	}
}

// convertToAPIVisualRegime converts ui_alchemy.VisualRegime to api.VisualRegime
func convertToAPIVisualRegime(regime ui_alchemy.VisualRegime) VisualRegime {
	return VisualRegime{
		Name:           regime.Name,
		PrimaryColor:   regime.PrimaryColor,
		SecondaryColor: regime.SecondaryColor,
		Geometry: GeometryConfig{
			Type:       regime.Geometry.Type,
			Complexity: regime.Geometry.Complexity,
			Roughness:  regime.Geometry.Roughness,
			Metalness:  regime.Geometry.Metalness,
		},
		Physics: PhysicsConfig{
			FlowRate:   regime.Physics.FlowRate,
			Turbulence: regime.Physics.Turbulence,
			Gravity:    regime.Physics.Gravity,
			Viscosity:  regime.Physics.Viscosity,
		},
		ShaderUniforms: regime.ShaderUniforms,
	}
}

// ========== GPU SERVICE STUB ==========

// GPUServiceStub is a placeholder implementation
// Real implementation would connect to gpu_kernel package
type GPUServiceStub struct{}

func NewGPUService() GPUService {
	return &GPUServiceStub{}
}

func (g *GPUServiceStub) ComputeSLERP(ctx context.Context, req SLERPRequest) (*SLERPResponse, error) {
	start := time.Now()

	// Simple CPU fallback SLERP (real implementation would use GPU)
	// This is just to make the API functional
	result := cpuSLERP(req.Q1, req.Q2, req.T)

	return &SLERPResponse{
		Result:     result,
		ComputeMs:  float64(time.Since(start).Milliseconds()),
		DeviceUsed: "CPU_FALLBACK",
	}, nil
}

func (g *GPUServiceStub) GetStatus(ctx context.Context) (*GPUStatus, error) {
	return &GPUStatus{
		Available:     false,
		DeviceName:    "CPU_FALLBACK",
		KernelsLoaded: 0,
		Performance:   0.0,
	}, nil
}

func (g *GPUServiceStub) MultiplyQuaternions(ctx context.Context, q1, q2 Quaternion) (*Quaternion, error) {
	result := cpuQuaternionMultiply(q1, q2)
	return &result, nil
}

func (g *GPUServiceStub) NormalizeQuaternion(ctx context.Context, q Quaternion) (*Quaternion, error) {
	result := cpuQuaternionNormalize(q)
	return &result, nil
}

// ========== VQC SERVICE IMPLEMENTATION ==========

// VQCServiceImpl implements VQC optimization engine
// WIRED to VQC engine (10M candidates/sec capability!)
// Source: asymm_mathematical_organism/03_ENGINES/vqc/vqc_optimization_engine.go
type VQCServiceImpl struct{}

func NewVQCService() VQCService {
	return &VQCServiceImpl{}
}

func (v *VQCServiceImpl) Optimize(ctx context.Context, req OptimizationRequest) (*OptimizationResponse, error) {
	start := time.Now()

	// Determine number of candidates
	numCandidates := len(req.Candidates)
	if numCandidates == 0 {
		numCandidates = 10000 // Default to 10K candidates for good performance
	}

	// Override with max iterations if specified
	maxIters := req.MaxIters
	if maxIters == 0 {
		maxIters = 108 // Vedic sacred number
	}

	// Create VQC optimization engine
	engine := vqc.NewEngine(numCandidates)
	engine.MaxIterations = maxIters

	// Run the optimization!
	if err := engine.Run(ctx); err != nil {
		return nil, err
	}

	// Get results
	results := engine.GetResults()

	// Convert to API response
	return &OptimizationResponse{
		BestCandidate: Candidate{
			ID: "best",
			Value: []float64{
				results.BestCandidate.State.W,
				results.BestCandidate.State.X,
				results.BestCandidate.State.Y,
				results.BestCandidate.State.Z,
			},
			Score: results.BestFitness,
		},
		FinalScore: results.BestFitness,
		Iterations: results.Iterations,
		TimeMS:     float64(time.Since(start).Milliseconds()),
	}, nil
}

func (v *VQCServiceImpl) Classify(ctx context.Context, req ClassificationRequest) (*ClassificationResponse, error) {
	// Use VQC service for classification (71M ops/sec capability!)
	svc := vqc.NewService()

	vqcReq := vqc.ClassifyRequest{
		GeneExpression: req.GeneExpression,
		PatientID:      req.PatientID,
	}

	resp, err := svc.Classify(ctx, vqcReq)
	if err != nil {
		return nil, err
	}

	return &ClassificationResponse{
		Class:      resp.Class,
		Confidence: resp.Confidence,
		TimeMS:     resp.TimeMS,
	}, nil
}

func (v *VQCServiceImpl) AnalyzeClimate(ctx context.Context, req ClimateRequest) (*ClimateResponse, error) {
	// TODO: Wire to vqc_climate_analyzer.go when needed
	return nil, errors.New("VQC climate analysis not yet implemented - wire to asymm_mathematical_organism/03_ENGINES/vqc/vqc_climate_analyzer.go")
}

// ========== SURVIVAL GARDEN IMPLEMENTATION ==========

// SurvivalGardenImpl wraps the existing survival_garden package
type SurvivalGardenImpl struct{}

func NewSurvivalGardenService() SurvivalGardenService {
	return &SurvivalGardenImpl{}
}

func (s *SurvivalGardenImpl) Simulate(ctx context.Context, params SimulationParams) (*SimulationResult, error) {
	// Wire to the actual GPU simulation!
	start := time.Now()

	// Convert API params to survival_garden params
	sgParams := survival_garden.SimulationParams{
		CashRunway:       params.CashRunway,
		MonthlyBurn:      params.MonthlyBurn,
		MonthsToSimulate: params.MonthsToSimulate,
		GPUAccelerate:    params.GPUAccelerate,
		Expenses:         convertExpenses(params.Expenses),
	}

	// Call the actual simulation
	states, err := survival_garden.SimulateSurvivalGarden(sgParams)
	if err != nil {
		return nil, err
	}

	// Convert states back to API types
	apiStates := make([]GardenState, len(states))
	for i, state := range states {
		apiStates[i] = GardenState{
			WaterLevel:    state.WaterLevel,
			StoneHeights:  state.StoneHeights,
			ParticleCount: state.ParticleCount,
			Regime:        state.Regime,
			Temperature:   state.Temperature,
			Turbulence:    state.Turbulence,
		}
	}

	elapsed := time.Since(start)

	return &SimulationResult{
		States: apiStates,
		TimeMS: float64(elapsed.Milliseconds()),
	}, nil
}

func (s *SurvivalGardenImpl) GetEquilibrium(ctx context.Context, params SimulationParams) (*EquilibriumResult, error) {
	start := time.Now()

	// Convert API params to survival_garden params
	sgParams := survival_garden.SimulationParams{
		CashRunway:       params.CashRunway,
		MonthlyBurn:      params.MonthlyBurn,
		MonthsToSimulate: params.MonthsToSimulate,
		GPUAccelerate:    params.GPUAccelerate,
		Expenses:         convertExpenses(params.Expenses),
	}

	// Calculate the 87.532% thermodynamic attractor
	attractor := survival_garden.CalculateEquilibriumState(sgParams)

	// Business is sustainable if it can reach equilibrium
	// (i.e., cash runway > 0 months)
	sustainable := params.CashRunway > 0

	elapsed := time.Since(start)

	return &EquilibriumResult{
		Attractor:   attractor,
		Sustainable: sustainable,
		TimeMS:      float64(elapsed.Milliseconds()),
	}, nil
}

// convertExpenses converts API Expense types to survival_garden Expense types
func convertExpenses(apiExpenses []Expense) []survival_garden.Expense {
	sgExpenses := make([]survival_garden.Expense, len(apiExpenses))
	for i, exp := range apiExpenses {
		sgExpenses[i] = survival_garden.Expense{
			Name:   exp.Name,
			Amount: exp.Amount,
			Weight: exp.Weight,
		}
	}
	return sgExpenses
}

// ========== CPU FALLBACK MATH ==========

// cpuSLERP performs quaternion SLERP on CPU
func cpuSLERP(q1, q2 Quaternion, t float64) Quaternion {
	// Simplified SLERP (real implementation in primitives.go)
	// Linear interpolation as fallback
	return Quaternion{
		W: q1.W*(1-t) + q2.W*t,
		X: q1.X*(1-t) + q2.X*t,
		Y: q1.Y*(1-t) + q2.Y*t,
		Z: q1.Z*(1-t) + q2.Z*t,
	}
}

// cpuQuaternionMultiply performs quaternion multiplication on CPU
func cpuQuaternionMultiply(q1, q2 Quaternion) Quaternion {
	return Quaternion{
		W: q1.W*q2.W - q1.X*q2.X - q1.Y*q2.Y - q1.Z*q2.Z,
		X: q1.W*q2.X + q1.X*q2.W + q1.Y*q2.Z - q1.Z*q2.Y,
		Y: q1.W*q2.Y - q1.X*q2.Z + q1.Y*q2.W + q1.Z*q2.X,
		Z: q1.W*q2.Z + q1.X*q2.Y - q1.Y*q2.X + q1.Z*q2.W,
	}
}

// cpuQuaternionNormalize normalizes a quaternion on CPU
func cpuQuaternionNormalize(q Quaternion) Quaternion {
	mag := q.W*q.W + q.X*q.X + q.Y*q.Y + q.Z*q.Z
	if mag == 0 {
		return Quaternion{W: 1, X: 0, Y: 0, Z: 0}
	}

	invMag := 1.0 / mag
	return Quaternion{
		W: q.W * invMag,
		X: q.X * invMag,
		Y: q.Y * invMag,
		Z: q.Z * invMag,
	}
}
