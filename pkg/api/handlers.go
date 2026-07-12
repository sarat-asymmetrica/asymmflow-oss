package api

import (
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
)

// ========== UI ALCHEMY HANDLERS ==========

func (s *Server) getScreen(w http.ResponseWriter, r *http.Request) {
	screenID := chi.URLParam(r, "screenID")

	// Parse context from query params
	ctx := ContextVector{
		TimeOfDay: r.URL.Query().Get("time_of_day"),
		FlowRate:  parseFloat(r.URL.Query().Get("flow_rate"), 45.0),
		Urgency:   parseFloat(r.URL.Query().Get("urgency"), 0.2),
	}

	// Default time of day
	if ctx.TimeOfDay == "" {
		ctx.TimeOfDay = "morning"
	}

	start := time.Now()
	screen, err := s.uiAlchemy.GetScreen(r.Context(), screenID, ctx)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorWithDetails(
			"SCREEN_NOT_FOUND",
			"Screen layout not found",
			err.Error(),
		))
		return
	}

	resp := SuccessWithMeta(screen, r.Header.Get("X-Request-ID"), duration)
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) generateScreen(w http.ResponseWriter, r *http.Request) {
	var req GenerateScreenRequest
	if err := parseJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Error(
			"INVALID_REQUEST",
			"Failed to parse request body: "+err.Error(),
		))
		return
	}

	start := time.Now()
	screen, err := s.uiAlchemy.GenerateScreen(r.Context(), req)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorWithDetails(
			"GENERATION_FAILED",
			"Failed to generate screen",
			err.Error(),
		))
		return
	}

	resp := SuccessWithMeta(screen, r.Header.Get("X-Request-ID"), duration)
	writeJSON(w, http.StatusOK, resp)
}

// ========== GPU HANDLERS ==========

func (s *Server) computeSLERP(w http.ResponseWriter, r *http.Request) {
	var req SLERPRequest
	if err := parseJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Error(
			"INVALID_REQUEST",
			"Failed to parse SLERP request: "+err.Error(),
		))
		return
	}

	// Validate input
	if req.T < 0.0 || req.T > 1.0 {
		writeJSON(w, http.StatusBadRequest, Error(
			"INVALID_PARAMETER",
			"Parameter 't' must be in range [0, 1]",
		))
		return
	}

	// Increment GPU operations counter
	atomic.AddInt64(&s.counters.gpuOperations, 1)

	result, err := s.gpu.ComputeSLERP(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorWithDetails(
			"SLERP_FAILED",
			"GPU SLERP computation failed",
			err.Error(),
		))
		return
	}

	writeJSON(w, http.StatusOK, Success(result))
}

func (s *Server) gpuStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.gpu.GetStatus(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorWithDetails(
			"GPU_STATUS_FAILED",
			"Failed to get GPU status",
			err.Error(),
		))
		return
	}

	writeJSON(w, http.StatusOK, Success(status))
}

func (s *Server) quaternionMultiply(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Q1 Quaternion `json:"q1"`
		Q2 Quaternion `json:"q2"`
	}

	if err := parseJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Error(
			"INVALID_REQUEST",
			"Failed to parse quaternion multiply request: "+err.Error(),
		))
		return
	}

	// Increment GPU operations counter
	atomic.AddInt64(&s.counters.gpuOperations, 1)

	result, err := s.gpu.MultiplyQuaternions(r.Context(), req.Q1, req.Q2)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorWithDetails(
			"QUATERNION_MULTIPLY_FAILED",
			"Quaternion multiplication failed",
			err.Error(),
		))
		return
	}

	writeJSON(w, http.StatusOK, Success(result))
}

func (s *Server) quaternionNormalize(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Q Quaternion `json:"q"`
	}

	if err := parseJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Error(
			"INVALID_REQUEST",
			"Failed to parse quaternion normalize request: "+err.Error(),
		))
		return
	}

	// Increment GPU operations counter
	atomic.AddInt64(&s.counters.gpuOperations, 1)

	result, err := s.gpu.NormalizeQuaternion(r.Context(), req.Q)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorWithDetails(
			"QUATERNION_NORMALIZE_FAILED",
			"Quaternion normalization failed",
			err.Error(),
		))
		return
	}

	writeJSON(w, http.StatusOK, Success(result))
}

// ========== VQC HANDLERS ==========

func (s *Server) vqcOptimize(w http.ResponseWriter, r *http.Request) {
	var req OptimizationRequest
	if err := parseJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Error(
			"INVALID_REQUEST",
			"Failed to parse VQC optimization request: "+err.Error(),
		))
		return
	}

	// Increment VQC optimizations counter
	atomic.AddInt64(&s.counters.vqcOptimizations, 1)

	result, err := s.vqc.Optimize(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorWithDetails(
			"VQC_OPTIMIZATION_FAILED",
			"VQC optimization failed",
			err.Error(),
		))
		return
	}

	writeJSON(w, http.StatusOK, Success(result))
}

func (s *Server) vqcClassify(w http.ResponseWriter, r *http.Request) {
	var req ClassificationRequest
	if err := parseJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Error(
			"INVALID_REQUEST",
			"Failed to parse VQC classification request: "+err.Error(),
		))
		return
	}

	// Increment VQC optimizations counter
	atomic.AddInt64(&s.counters.vqcOptimizations, 1)

	result, err := s.vqc.Classify(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorWithDetails(
			"VQC_CLASSIFICATION_FAILED",
			"VQC cancer classification failed",
			err.Error(),
		))
		return
	}

	writeJSON(w, http.StatusOK, Success(result))
}

func (s *Server) vqcClimate(w http.ResponseWriter, r *http.Request) {
	var req ClimateRequest
	if err := parseJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Error(
			"INVALID_REQUEST",
			"Failed to parse VQC climate request: "+err.Error(),
		))
		return
	}

	// Increment VQC optimizations counter
	atomic.AddInt64(&s.counters.vqcOptimizations, 1)

	result, err := s.vqc.AnalyzeClimate(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorWithDetails(
			"VQC_CLIMATE_FAILED",
			"VQC climate analysis failed",
			err.Error(),
		))
		return
	}

	writeJSON(w, http.StatusOK, Success(result))
}

// ========== SURVIVAL GARDEN HANDLERS ==========

func (s *Server) simulateSurvivalGarden(w http.ResponseWriter, r *http.Request) {
	var params SimulationParams
	if err := parseJSON(r, &params); err != nil {
		writeJSON(w, http.StatusBadRequest, Error(
			"INVALID_REQUEST",
			"Failed to parse simulation parameters: "+err.Error(),
		))
		return
	}

	result, err := s.survivalGarden.Simulate(r.Context(), params)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorWithDetails(
			"SIMULATION_FAILED",
			"Survival garden simulation failed",
			err.Error(),
		))
		return
	}

	writeJSON(w, http.StatusOK, Success(result))
}

func (s *Server) getEquilibrium(w http.ResponseWriter, r *http.Request) {
	var params SimulationParams
	if err := parseJSON(r, &params); err != nil {
		writeJSON(w, http.StatusBadRequest, Error(
			"INVALID_REQUEST",
			"Failed to parse simulation parameters: "+err.Error(),
		))
		return
	}

	result, err := s.survivalGarden.GetEquilibrium(r.Context(), params)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorWithDetails(
			"EQUILIBRIUM_FAILED",
			"Failed to calculate equilibrium",
			err.Error(),
		))
		return
	}

	writeJSON(w, http.StatusOK, Success(result))
}

// ========== VISUAL REGIME HANDLERS ==========

func (s *Server) listRegimes(w http.ResponseWriter, r *http.Request) {
	// This will be implemented by the VisualRegimeService
	// For now, return a placeholder
	writeJSON(w, http.StatusOK, Success(map[string]any{
		"regimes": []string{"MorningCalm", "HighVelocity"},
	}))
}

func (s *Server) getRegime(w http.ResponseWriter, r *http.Request) {
	regimeName := chi.URLParam(r, "regimeName")

	// This will be implemented by the VisualRegimeService
	// For now, return a placeholder
	writeJSON(w, http.StatusOK, Success(map[string]any{
		"name":    regimeName,
		"message": "Regime retrieval not yet implemented",
	}))
}

func (s *Server) computeRegime(w http.ResponseWriter, r *http.Request) {
	var ctx ContextVector
	if err := parseJSON(r, &ctx); err != nil {
		writeJSON(w, http.StatusBadRequest, Error(
			"INVALID_REQUEST",
			"Failed to parse context vector: "+err.Error(),
		))
		return
	}

	// This will be implemented by the VisualRegimeService
	// For now, return a placeholder
	writeJSON(w, http.StatusOK, Success(map[string]any{
		"context": ctx,
		"message": "Regime computation not yet implemented",
	}))
}

// ========== HELPER FUNCTIONS ==========

// parseFloat parses a string to float64, returning defaultVal if empty or invalid
func parseFloat(s string, defaultVal float64) float64 {
	if s == "" {
		return defaultVal
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		// Invalid float, return default
		return defaultVal
	}
	return f
}
