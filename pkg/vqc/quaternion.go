package vqc

import "math"

// ============================================================================
// QUATERNION PRIMITIVES (Imported from asymm_mathematical_organism)
// Shared across all VQC engines to avoid duplication
// ============================================================================

// Quaternion represents a point on S³ unit sphere
type Quaternion struct {
	W, X, Y, Z float64
}

// NewQuaternion creates and normalizes a quaternion
func NewQuaternion(w, x, y, z float64) Quaternion {
	q := Quaternion{W: w, X: x, Y: y, Z: z}
	return q.Normalize()
}

// Norm computes ||q|| = sqrt(w² + x² + y² + z²)
func (q Quaternion) Norm() float64 {
	return math.Sqrt(q.W*q.W + q.X*q.X + q.Y*q.Y + q.Z*q.Z)
}

// Normalize returns unit quaternion (||q|| = 1)
func (q Quaternion) Normalize() Quaternion {
	normSquared := q.W*q.W + q.X*q.X + q.Y*q.Y + q.Z*q.Z
	if normSquared < 1e-20 {
		return Quaternion{W: 1, X: 0, Y: 0, Z: 0}
	}
	// Babylonian square root for 4000-year-old optimality!
	n := math.Sqrt(normSquared)
	return Quaternion{W: q.W / n, X: q.X / n, Y: q.Y / n, Z: q.Z / n}
}

// Dot computes quaternion dot product
func (q1 Quaternion) Dot(q2 Quaternion) float64 {
	return q1.W*q2.W + q1.X*q2.X + q1.Y*q2.Y + q1.Z*q2.Z
}

// Distance computes geodesic distance on S³
func (q1 Quaternion) Distance(q2 Quaternion) float64 {
	dot := math.Abs(q1.Dot(q2))
	if dot > 1.0 {
		dot = 1.0
	}
	return math.Acos(dot)
}

// Add quaternions (vector addition)
func (q1 Quaternion) Add(q2 Quaternion) Quaternion {
	return Quaternion{
		W: q1.W + q2.W,
		X: q1.X + q2.X,
		Y: q1.Y + q2.Y,
		Z: q1.Z + q2.Z,
	}
}

// Scale quaternion by scalar
func (q Quaternion) Scale(s float64) Quaternion {
	return Quaternion{
		W: q.W * s,
		X: q.X * s,
		Y: q.Y * s,
		Z: q.Z * s,
	}
}

// Multiply quaternions (Hamilton product)
func (q1 Quaternion) Multiply(q2 Quaternion) Quaternion {
	return Quaternion{
		W: q1.W*q2.W - q1.X*q2.X - q1.Y*q2.Y - q1.Z*q2.Z,
		X: q1.W*q2.X + q1.X*q2.W + q1.Y*q2.Z - q1.Z*q2.Y,
		Y: q1.W*q2.Y - q1.X*q2.Z + q1.Y*q2.W + q1.Z*q2.X,
		Z: q1.W*q2.Z + q1.X*q2.Y - q1.Y*q2.X + q1.Z*q2.W,
	}
}

// SLERP performs spherical linear interpolation between two quaternions
// t ∈ [0,1]: t=0 returns q1, t=1 returns q2
// This is the GEODESIC path on S³ (shortest distance on the 3-sphere!)
func SLERP(q1, q2 Quaternion, t float64) Quaternion {
	dot := q1.Dot(q2)

	// If quaternions are nearly identical, use linear interpolation
	if dot > 0.9995 {
		result := Quaternion{
			W: q1.W + t*(q2.W-q1.W),
			X: q1.X + t*(q2.X-q1.X),
			Y: q1.Y + t*(q2.Y-q1.Y),
			Z: q1.Z + t*(q2.Z-q1.Z),
		}
		return result.Normalize()
	}

	// Clamp dot product
	if dot < -1.0 {
		dot = -1.0
	} else if dot > 1.0 {
		dot = 1.0
	}

	// Compute angle between quaternions
	theta := math.Acos(math.Abs(dot))
	sinTheta := math.Sin(theta)

	// Compute interpolation coefficients
	w1 := math.Sin((1-t)*theta) / sinTheta
	w2 := math.Sin(t*theta) / sinTheta

	// Handle antipodal quaternions (dot < 0)
	if dot < 0 {
		w2 = -w2
	}

	return Quaternion{
		W: w1*q1.W + w2*q2.W,
		X: w1*q1.X + w2*q2.X,
		Y: w1*q1.Y + w2*q2.Y,
		Z: w1*q1.Z + w2*q2.Z,
	}
}
