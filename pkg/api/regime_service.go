package api

import (
	"context"
	"errors"
	"sync"

	"ph_holdings_app/pkg/ui_alchemy"
)

// VisualRegimeServiceImpl is the THREAD-SAFE implementation
// FIXES VIOLATION #2: No more global DefaultRegimes variable!
type VisualRegimeServiceImpl struct {
	regimes map[string]ui_alchemy.VisualRegime
	mu      sync.RWMutex // Thread-safe access
}

// NewVisualRegimeService creates a new regime service with default regimes
func NewVisualRegimeService() VisualRegimeService {
	return &VisualRegimeServiceImpl{
		regimes: ui_alchemy.DefaultRegimes, // Copy from existing, but now instance-based!
	}
}

// NewVisualRegimeServiceWithCustom allows custom regime configurations
// This enables multi-tenant customization!
func NewVisualRegimeServiceWithCustom(regimes map[string]ui_alchemy.VisualRegime) VisualRegimeService {
	return &VisualRegimeServiceImpl{
		regimes: regimes,
	}
}

func (v *VisualRegimeServiceImpl) GetRegime(ctx context.Context, name string) (*VisualRegime, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	regime, ok := v.regimes[name]
	if !ok {
		return nil, errors.New("regime not found: " + name)
	}

	apiRegime := convertToAPIVisualRegime(regime)
	return &apiRegime, nil
}

func (v *VisualRegimeServiceImpl) ListRegimes(ctx context.Context) ([]VisualRegime, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	result := make([]VisualRegime, 0, len(v.regimes))
	for _, regime := range v.regimes {
		result = append(result, convertToAPIVisualRegime(regime))
	}

	return result, nil
}

func (v *VisualRegimeServiceImpl) ComputeRegime(ctx context.Context, contextVec ContextVector) (*VisualRegime, error) {
	// Convert API ContextVector to ui_alchemy.ContextVector
	alchemyCtx := ui_alchemy.ContextVector{
		TimeOfDay: contextVec.TimeOfDay,
		FlowRate:  contextVec.FlowRate,
		Urgency:   contextVec.Urgency,
	}

	// Use existing GetVisualRegime logic
	regime := ui_alchemy.GetVisualRegime(alchemyCtx)

	apiRegime := convertToAPIVisualRegime(regime)
	return &apiRegime, nil
}

// AddRegime allows dynamic regime addition (hot-reload configurations!)
func (v *VisualRegimeServiceImpl) AddRegime(name string, regime ui_alchemy.VisualRegime) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.regimes[name] = regime
}

// RemoveRegime allows dynamic regime removal
func (v *VisualRegimeServiceImpl) RemoveRegime(name string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	delete(v.regimes, name)
}

// UpdateRegime allows dynamic regime modification
func (v *VisualRegimeServiceImpl) UpdateRegime(name string, regime ui_alchemy.VisualRegime) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if _, ok := v.regimes[name]; !ok {
		return errors.New("regime not found: " + name)
	}

	v.regimes[name] = regime
	return nil
}
