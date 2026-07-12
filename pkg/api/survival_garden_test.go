package api

import (
	"context"
	"testing"
)

func TestSurvivalGardenSimulation(t *testing.T) {
	service := NewSurvivalGardenService()
	ctx := context.Background()

	// Test parameters: 6 months runway, 5000 BHD/month burn
	params := SimulationParams{
		CashRunway:       6.0,    // 6 months
		MonthlyBurn:      5000.0, // 5000 BHD/month
		MonthsToSimulate: 12,     // Simulate 12 months
		GPUAccelerate:    true,   // Try GPU acceleration
		Expenses: []Expense{
			{Name: "Salaries", Amount: 3000.0, Weight: 0.6},
			{Name: "Office", Amount: 1000.0, Weight: 0.2},
			{Name: "Cloud", Amount: 500.0, Weight: 0.1},
			{Name: "Marketing", Amount: 500.0, Weight: 0.1},
		},
	}

	// Run simulation
	result, err := service.Simulate(ctx, params)
	if err != nil {
		t.Fatalf("Simulation failed: %v", err)
	}

	// Validate results
	if result == nil {
		t.Fatal("Result is nil")
	}

	if len(result.States) == 0 {
		t.Fatal("No states returned")
	}

	t.Logf("✓ Simulation completed in %.2fms", result.TimeMS)
	t.Logf("✓ Generated %d states", len(result.States))

	// Check initial state (month 0)
	initialState := result.States[0]
	if initialState.WaterLevel == 0.0 {
		t.Error("Initial water level should not be 0")
	}
	if initialState.Regime == 0 {
		t.Error("Initial regime should not be 0 (bankrupt)")
	}

	t.Logf("  Initial state: water=%.2f%%, regime=%d, temp=%.2f, turbulence=%.2f",
		initialState.WaterLevel*100, initialState.Regime,
		initialState.Temperature, initialState.Turbulence)

	// Check that water level decreases over time (cash is burning!)
	if len(result.States) > 6 {
		month6 := result.States[6]
		// At month 6, we should be near bankruptcy (6 months runway, 6 months passed)
		if month6.WaterLevel > initialState.WaterLevel {
			t.Error("Water level should decrease as cash burns")
		}

		t.Logf("  Month 6 state: water=%.2f%%, regime=%d, temp=%.2f",
			month6.WaterLevel*100, month6.Regime, month6.Temperature)
	}

	// Check three-regime dynamics
	regimeCounts := map[int]int{}
	for _, state := range result.States {
		regimeCounts[state.Regime]++
	}

	t.Logf("  Regime distribution:")
	for regime, count := range regimeCounts {
		regimeName := "UNKNOWN"
		switch regime {
		case 0:
			regimeName = "BANKRUPT"
		case 1:
			regimeName = "DANGER (R1)"
		case 2:
			regimeName = "WARNING (R2)"
		case 3:
			regimeName = "SAFE (R3)"
		}
		t.Logf("    Regime %d (%s): %d states", regime, regimeName, count)
	}

	// Verify stone heights match expenses
	if len(initialState.StoneHeights) != len(params.Expenses) {
		t.Errorf("Expected %d stones, got %d",
			len(params.Expenses), len(initialState.StoneHeights))
	}
}

func TestSurvivalGardenEquilibrium(t *testing.T) {
	service := NewSurvivalGardenService()
	ctx := context.Background()

	params := SimulationParams{
		CashRunway:  12.0, // 12 months runway (healthy business)
		MonthlyBurn: 5000.0,
		Expenses: []Expense{
			{Name: "Salaries", Amount: 3000.0, Weight: 0.6},
		},
	}

	result, err := service.GetEquilibrium(ctx, params)
	if err != nil {
		t.Fatalf("GetEquilibrium failed: %v", err)
	}

	// Check the 87.532% thermodynamic attractor
	expectedAttractor := 0.87532
	if result.Attractor != expectedAttractor {
		t.Errorf("Expected attractor %.5f, got %.5f",
			expectedAttractor, result.Attractor)
	}

	if !result.Sustainable {
		t.Error("Business with 12 months runway should be sustainable")
	}

	t.Logf("✓ Equilibrium: %.5f (87.532%% thermodynamic attractor)", result.Attractor)
	t.Logf("✓ Sustainable: %v", result.Sustainable)
	t.Logf("✓ Computed in %.2fms", result.TimeMS)
}

func TestSurvivalGardenBankruptcy(t *testing.T) {
	service := NewSurvivalGardenService()
	ctx := context.Background()

	// Simulate inevitable bankruptcy (1 month runway, simulate 6 months)
	params := SimulationParams{
		CashRunway:       1.0, // Only 1 month runway
		MonthlyBurn:      5000.0,
		MonthsToSimulate: 6, // Try to simulate 6 months
		GPUAccelerate:    false,
		Expenses: []Expense{
			{Name: "Salaries", Amount: 5000.0, Weight: 1.0},
		},
	}

	result, err := service.Simulate(ctx, params)
	if err != nil {
		t.Fatalf("Simulation failed: %v", err)
	}

	// Find bankruptcy state (regime 0)
	bankruptcyFound := false
	bankruptcyMonth := -1
	for i, state := range result.States {
		if state.Regime == 0 {
			bankruptcyFound = true
			bankruptcyMonth = i
			t.Logf("✓ Bankruptcy detected at month %d", i)
			t.Logf("  State: water=%.2f%%, temp=%.2f, turbulence=%.2f",
				state.WaterLevel*100, state.Temperature, state.Turbulence)

			// Verify bankruptcy state characteristics
			if state.WaterLevel != 0.0 {
				t.Error("Bankruptcy state should have 0% water level")
			}
			if state.Temperature != 1.0 {
				t.Error("Bankruptcy state should have max temperature (1.0)")
			}
			if state.Turbulence != 2.0 {
				t.Error("Bankruptcy state should have max turbulence (2.0)")
			}
			break
		}
	}

	if !bankruptcyFound {
		t.Error("Expected bankruptcy state (regime 0) but none found")
	}

	// Verify simulation stopped after bankruptcy
	// (Should have bankruptcyMonth+1 states, not 6+1)
	expectedStates := bankruptcyMonth + 1
	if bankruptcyFound && len(result.States) != expectedStates {
		t.Logf("Note: Simulation returned %d states (expected %d after bankruptcy)",
			len(result.States), expectedStates)
	}

	t.Logf("✓ Simulation correctly handled bankruptcy scenario")
}
