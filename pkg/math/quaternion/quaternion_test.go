package quaternion

import (
	"math"
	"testing"
)

func TestIdentityIsUnit(t *testing.T) {
	if !Identity().IsUnit(1e-12) {
		t.Fatalf("identity quaternion should be unit")
	}
}

func TestNormalize(t *testing.T) {
	q := New(2, 0, 0, 0).Normalize()
	if !q.IsUnit(1e-12) {
		t.Fatalf("normalized quaternion magnitude = %f, want unit", q.Magnitude())
	}
	if q.W != 1 || q.X != 0 || q.Y != 0 || q.Z != 0 {
		t.Fatalf("Normalize() = %+v, want identity direction", q)
	}
}

func TestSlerpEndpoints(t *testing.T) {
	start := Identity()
	end := FromAxisAngle(0, 0, 1, math.Pi/2)

	if got := Slerp(start, end, 0); !got.EqualsRotation(start, 1e-12) {
		t.Fatalf("Slerp(t=0) = %+v, want start %+v", got, start)
	}
	if got := Slerp(start, end, 1); !got.EqualsRotation(end, 1e-12) {
		t.Fatalf("Slerp(t=1) = %+v, want end %+v", got, end)
	}
}

func TestSlerpMidpointIsUnit(t *testing.T) {
	start := Identity()
	end := FromAxisAngle(1, 0, 0, math.Pi)
	mid := Slerp(start, end, 0.5)

	if !mid.IsUnit(1e-12) {
		t.Fatalf("midpoint magnitude = %f, want unit", mid.Magnitude())
	}
}

func TestGeodesicDistanceSymmetric(t *testing.T) {
	a := FromAxisAngle(1, 0, 0, math.Pi/3)
	b := FromAxisAngle(0, 1, 0, math.Pi/5)

	d1 := a.GeodesicDistance(b)
	d2 := b.GeodesicDistance(a)
	if math.Abs(d1-d2) > 1e-12 {
		t.Fatalf("distance symmetry mismatch: %f vs %f", d1, d2)
	}
}
