package engines

import (
	"math"
	"testing"

	prediction "ph_holdings_app/pkg/butler/prediction"
)

func TestQuaternion_Methods(t *testing.T) {
	q := NewQuaternion(1.0, 2.0, 3.0, 4.0)

	// NewQuaternion normalizes, so norm should be 1.0
	norm := q.Norm()
	if math.Abs(norm-1.0) > 1e-9 {
		t.Errorf("Norm() = %v, want 1.0", norm)
	}

	dot := q.Dot(q)
	// Dot product of a unit quaternion with itself is 1.0
	if math.Abs(dot-1.0) > 1e-9 {
		t.Errorf("Dot() = %v, want 1.0", dot)
	}
}

func TestGeometryBridge_Basic(t *testing.T) {
	gb := NewGeometryBridge()
	if gb == nil {
		t.Fatal("NewGeometryBridge() returned nil")
	}

	customer := &prediction.Customer{
		ID:             "CUST001",
		PaymentHistory: []int{30, 35, 40},
		RelationYears:  5,
	}

	c360, err := gb.GetCustomer360("CUST001", customer)
	if err != nil {
		t.Errorf("GetCustomer360 failed: %v", err)
	}

	if c360.Grade == "" {
		t.Error("Expected non-empty grade")
	}
}

func TestGeometryBridge_CheckCompliance(t *testing.T) {
	gb := NewGeometryBridge()

	data := ComplianceData{
		Type:    "invoice",
		RuleSet: "bahrain",
		Data: map[string]any{
			"customer_id": "CUST001",
			"amount":      10000.0,
		},
	}

	res, err := gb.CheckCompliance(data)
	if err != nil {
		t.Errorf("CheckCompliance failed: %v", err)
	}

	if res.Score < 0 || res.Score > 1 {
		t.Errorf("Invalid compliance score: %v", res.Score)
	}
}

func TestTemplateLayout_Scaling(t *testing.T) {
	tl := &TemplateLayout{
		DPI: 300,
	}

	// 1 inch = 25.4 mm
	// 300 pixels at 300 DPI = 1 inch = 25.4 mm
	mm := tl.PixelsToMM(300)
	if math.Abs(mm-25.4) > 1e-9 {
		t.Errorf("PixelsToMM(300) = %v, want 25.4", mm)
	}

	pixels := tl.MMToPixels(25.4)
	if math.Abs(pixels-300) > 1e-9 {
		t.Errorf("MMToPixels(25.4) = %v, want 300", pixels)
	}
}

func TestTemplateLayout_GetZone(t *testing.T) {
	tl := &TemplateLayout{
		Zones: []TemplateZone{
			{Name: "header", Purpose: "text"},
			{Name: "footer", Purpose: "image"},
		},
	}

	z1 := tl.GetZone("header")
	if z1 == nil || z1.Name != "header" {
		t.Error("Failed to get header zone")
	}

	z2 := tl.GetZoneByPurpose("image")
	if z2 == nil || z2.Purpose != "image" {
		t.Error("Failed to get zone by purpose")
	}
}
