package overlay

import (
	"encoding/json"
	"testing"
)

// TestDeploymentSlug_EmptyFallsBackToDefault verifies that a zero-value
// CompanyOverlay (no Deployment.Slug configured) yields the built-in
// "AsymmFlow-Dev" default.
func TestDeploymentSlug_EmptyFallsBackToDefault(t *testing.T) {
	o := &CompanyOverlay{}
	got := o.DeploymentSlug()
	want := "AsymmFlow-Dev"
	if got != want {
		t.Errorf("DeploymentSlug() = %q, want %q", got, want)
	}
}

// TestDeploymentSlug_ConfiguredSlugWins verifies that an explicit slug is
// returned as-is.
func TestDeploymentSlug_ConfiguredSlugWins(t *testing.T) {
	o := &CompanyOverlay{Deployment: DeploymentConfig{Slug: "AsymmFlow-PH"}}
	got := o.DeploymentSlug()
	want := "AsymmFlow-PH"
	if got != want {
		t.Errorf("DeploymentSlug() = %q, want %q", got, want)
	}
}

// TestDeploymentSlug_WhitespaceOnlyFallsBackToDefault verifies that a
// whitespace-only slug is treated as blank and falls back to the default.
func TestDeploymentSlug_WhitespaceOnlyFallsBackToDefault(t *testing.T) {
	o := &CompanyOverlay{Deployment: DeploymentConfig{Slug: "  "}}
	got := o.DeploymentSlug()
	want := "AsymmFlow-Dev"
	if got != want {
		t.Errorf("DeploymentSlug() = %q, want %q", got, want)
	}
}

// TestDeploymentSlug_JSONRoundTrip verifies the json tag path: unmarshalling
// overlay.json's "deployment.slug" into a CompanyOverlay produces the same
// slug via DeploymentSlug().
func TestDeploymentSlug_JSONRoundTrip(t *testing.T) {
	var o CompanyOverlay
	data := []byte(`{"deployment":{"slug":"AsymmFlow-PH"}}`)
	if err := json.Unmarshal(data, &o); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	got := o.DeploymentSlug()
	want := "AsymmFlow-PH"
	if got != want {
		t.Errorf("DeploymentSlug() after JSON round-trip = %q, want %q", got, want)
	}
}
