// Package quaternion provides high-performance quaternion mathematics on S³.
//
// This is the Go implementation of the Vedic Qiskit resonant quaternion library.
// All quaternions are unit quaternions living on the 3-sphere (S³), enabling
// geodesic navigation through state space.
//
// Performance Targets:
//   - SLERP: 1.9M+ ops/sec (10x Python baseline)
//   - Norm: 10M+ ops/sec
//   - Multiply: 5M+ ops/sec
//
// Built with: Love × Simplicity × Truth × Joy
// Om Lokah Samastah Sukhino Bhavantu
package quaternion

import (
	"math"
	"strconv"
)

// Quaternion represents a unit quaternion on S³ with optional resonance frequency.
// The quaternion q = w + xi + yj + zk where i² = j² = k² = ijk = -1.
//
// For unit quaternions: ||q|| = sqrt(w² + x² + y² + z²) = 1
// This constraint keeps us on the 3-sphere.
type Quaternion struct {
	W, X, Y, Z float64 // Components: real part W, imaginary parts X, Y, Z
	Frequency  float64 // Resonance frequency in Hz (default 7.83 = Schumann)
	Phase      float64 // Phase angle in radians
}

// New creates a quaternion with the given components.
// The quaternion is NOT automatically normalized - call Normalize() if needed.
func New(w, x, y, z float64) Quaternion {
	return Quaternion{W: w, X: x, Y: y, Z: z, Frequency: 7.83}
}

// NewWithFrequency creates a quaternion with a specific resonance frequency.
func NewWithFrequency(w, x, y, z, freq float64) Quaternion {
	return Quaternion{W: w, X: x, Y: y, Z: z, Frequency: freq}
}

// Identity returns the identity quaternion (1, 0, 0, 0).
func Identity() Quaternion {
	return Quaternion{W: 1, X: 0, Y: 0, Z: 0, Frequency: 7.83}
}

// FromAxisAngle creates a unit quaternion from axis-angle representation.
// The axis should be a unit vector, angle is in radians.
func FromAxisAngle(ax, ay, az, angle float64) Quaternion {
	halfAngle := angle / 2
	s := math.Sin(halfAngle)
	return Quaternion{
		W:         math.Cos(halfAngle),
		X:         ax * s,
		Y:         ay * s,
		Z:         az * s,
		Frequency: 7.83,
	}
}

// Magnitude returns the L2 norm of the quaternion.
// For unit quaternions, this should always be 1.0.
//
// Formula: ||q|| = sqrt(w² + x² + y² + z²)
func (q Quaternion) Magnitude() float64 {
	return math.Sqrt(q.W*q.W + q.X*q.X + q.Y*q.Y + q.Z*q.Z)
}

// MagnitudeSquared returns ||q||² without the sqrt (faster for comparisons).
func (q Quaternion) MagnitudeSquared() float64 {
	return q.W*q.W + q.X*q.X + q.Y*q.Y + q.Z*q.Z
}

// Normalize returns a unit quaternion (projected onto S³).
// If the quaternion is zero, returns identity to avoid NaN.
func (q Quaternion) Normalize() Quaternion {
	mag := q.Magnitude()
	if mag < 1e-10 {
		return Identity()
	}
	invMag := 1.0 / mag
	return Quaternion{
		W:         q.W * invMag,
		X:         q.X * invMag,
		Y:         q.Y * invMag,
		Z:         q.Z * invMag,
		Frequency: q.Frequency,
		Phase:     q.Phase,
	}
}

// NormalizeInPlace normalizes the quaternion in place (no allocation).
// Returns the original magnitude for reference.
func (q *Quaternion) NormalizeInPlace() float64 {
	mag := q.Magnitude()
	if mag < 1e-10 {
		q.W, q.X, q.Y, q.Z = 1, 0, 0, 0
		return 0
	}
	invMag := 1.0 / mag
	q.W *= invMag
	q.X *= invMag
	q.Y *= invMag
	q.Z *= invMag
	return mag
}

// Conjugate returns the conjugate q* = w - xi - yj - zk.
// For unit quaternions: q* = q⁻¹ (inverse).
func (q Quaternion) Conjugate() Quaternion {
	return Quaternion{
		W:         q.W,
		X:         -q.X,
		Y:         -q.Y,
		Z:         -q.Z,
		Frequency: q.Frequency,
		Phase:     q.Phase,
	}
}

// Inverse returns the multiplicative inverse q⁻¹ = q*/||q||².
// For unit quaternions, this equals the conjugate.
func (q Quaternion) Inverse() Quaternion {
	magSq := q.MagnitudeSquared()
	if magSq < 1e-20 {
		return Identity()
	}
	invMagSq := 1.0 / magSq
	return Quaternion{
		W:         q.W * invMagSq,
		X:         -q.X * invMagSq,
		Y:         -q.Y * invMagSq,
		Z:         -q.Z * invMagSq,
		Frequency: q.Frequency,
		Phase:     q.Phase,
	}
}

// Dot returns the 4D dot product of two quaternions.
// For unit quaternions, dot = cos(θ/2) where θ is the geodesic distance.
func (q Quaternion) Dot(other Quaternion) float64 {
	return q.W*other.W + q.X*other.X + q.Y*other.Y + q.Z*other.Z
}

// Mul performs quaternion multiplication (Hamilton product).
// This is the core operation for composing rotations.
//
// Formula: q1 × q2 (non-commutative!)
func (q Quaternion) Mul(other Quaternion) Quaternion {
	return Quaternion{
		W: q.W*other.W - q.X*other.X - q.Y*other.Y - q.Z*other.Z,
		X: q.W*other.X + q.X*other.W + q.Y*other.Z - q.Z*other.Y,
		Y: q.W*other.Y - q.X*other.Z + q.Y*other.W + q.Z*other.X,
		Z: q.W*other.Z + q.X*other.Y - q.Y*other.X + q.Z*other.W,
		// Average frequency for combined state
		Frequency: (q.Frequency + other.Frequency) / 2,
		Phase:     q.Phase + other.Phase,
	}
}

// Scale multiplies the quaternion by a scalar.
func (q Quaternion) Scale(s float64) Quaternion {
	return Quaternion{
		W:         q.W * s,
		X:         q.X * s,
		Y:         q.Y * s,
		Z:         q.Z * s,
		Frequency: q.Frequency,
		Phase:     q.Phase,
	}
}

// Add returns q + other (component-wise).
func (q Quaternion) Add(other Quaternion) Quaternion {
	return Quaternion{
		W:         q.W + other.W,
		X:         q.X + other.X,
		Y:         q.Y + other.Y,
		Z:         q.Z + other.Z,
		Frequency: (q.Frequency + other.Frequency) / 2,
		Phase:     (q.Phase + other.Phase) / 2,
	}
}

// Sub returns q - other (component-wise).
func (q Quaternion) Sub(other Quaternion) Quaternion {
	return Quaternion{
		W:         q.W - other.W,
		X:         q.X - other.X,
		Y:         q.Y - other.Y,
		Z:         q.Z - other.Z,
		Frequency: q.Frequency,
		Phase:     q.Phase,
	}
}

// Negate returns -q.
func (q Quaternion) Negate() Quaternion {
	return Quaternion{
		W:         -q.W,
		X:         -q.X,
		Y:         -q.Y,
		Z:         -q.Z,
		Frequency: q.Frequency,
		Phase:     q.Phase,
	}
}

// GeodesicDistance returns the angle between two unit quaternions on S³.
// This is the length of the shortest path (geodesic) between them.
//
// Formula: d(q1, q2) = 2 × arccos(|q1 · q2|)
// Range: [0, π]
func (q Quaternion) GeodesicDistance(other Quaternion) float64 {
	dot := q.Dot(other)
	// Clamp to avoid numerical issues with arccos
	if dot > 1.0 {
		dot = 1.0
	} else if dot < -1.0 {
		dot = -1.0
	}
	// Take absolute value because q and -q represent the same rotation
	if dot < 0 {
		dot = -dot
	}
	return 2 * math.Acos(dot)
}

// BeatFrequency returns the interference pattern frequency between two quaternions.
// This is the resonance beating effect when two frequencies interact.
func (q Quaternion) BeatFrequency(other Quaternion) float64 {
	return math.Abs(q.Frequency - other.Frequency)
}

// Slerp performs Spherical Linear Interpolation between two unit quaternions.
// This traces the geodesic (shortest path) on S³.
//
// Parameters:
//   - q0, q1: Start and end quaternions (should be unit quaternions)
//   - t: Interpolation parameter in [0, 1]
//
// Returns: The quaternion at parameter t along the geodesic.
//
// Mathematical guarantee: ||result|| = 1.0 (stays on S³)
//
// This is the core navigation primitive - like light following the
// shortest path through spacetime!
func Slerp(q0, q1 Quaternion, t float64) Quaternion {
	// Compute the cosine of the angle between quaternions
	dot := q0.Dot(q1)

	// If q0 and q1 are nearly the same, use linear interpolation
	const threshold = 0.9995
	if dot > threshold {
		// Linear interpolation for very close quaternions
		result := Quaternion{
			W:         q0.W + t*(q1.W-q0.W),
			X:         q0.X + t*(q1.X-q0.X),
			Y:         q0.Y + t*(q1.Y-q0.Y),
			Z:         q0.Z + t*(q1.Z-q0.Z),
			Frequency: q0.Frequency + t*(q1.Frequency-q0.Frequency),
			Phase:     q0.Phase + t*(q1.Phase-q0.Phase),
		}
		return result.Normalize()
	}

	// If dot is negative, negate q1 to take the shorter path
	if dot < 0 {
		q1 = q1.Negate()
		dot = -dot
	}

	// Clamp dot for numerical stability
	if dot > 1.0 {
		dot = 1.0
	}

	// Calculate the angle and its sine
	theta := math.Acos(dot)
	sinTheta := math.Sin(theta)

	// Compute interpolation factors
	// s0 = sin((1-t)θ) / sin(θ)
	// s1 = sin(tθ) / sin(θ)
	s0 := math.Sin((1-t)*theta) / sinTheta
	s1 := math.Sin(t*theta) / sinTheta

	// Interpolate
	return Quaternion{
		W:         s0*q0.W + s1*q1.W,
		X:         s0*q0.X + s1*q1.X,
		Y:         s0*q0.Y + s1*q1.Y,
		Z:         s0*q0.Z + s1*q1.Z,
		Frequency: q0.Frequency + t*(q1.Frequency-q0.Frequency),
		Phase:     q0.Phase + t*(q1.Phase-q0.Phase),
	}
}

// SlerpBatch performs SLERP for multiple t values efficiently.
// Pre-computes the angle and factors for better performance.
func SlerpBatch(q0, q1 Quaternion, ts []float64) []Quaternion {
	results := make([]Quaternion, len(ts))

	dot := q0.Dot(q1)

	// Handle near-identical quaternions
	if dot > 0.9995 {
		for i, t := range ts {
			results[i] = Quaternion{
				W:         q0.W + t*(q1.W-q0.W),
				X:         q0.X + t*(q1.X-q0.X),
				Y:         q0.Y + t*(q1.Y-q0.Y),
				Z:         q0.Z + t*(q1.Z-q0.Z),
				Frequency: q0.Frequency + t*(q1.Frequency-q0.Frequency),
				Phase:     q0.Phase + t*(q1.Phase-q0.Phase),
			}
			results[i] = results[i].Normalize()
		}
		return results
	}

	// Take shorter path
	if dot < 0 {
		q1 = q1.Negate()
		dot = -dot
	}

	if dot > 1.0 {
		dot = 1.0
	}

	// Pre-compute angle factors
	theta := math.Acos(dot)
	sinTheta := math.Sin(theta)

	for i, t := range ts {
		s0 := math.Sin((1-t)*theta) / sinTheta
		s1 := math.Sin(t*theta) / sinTheta

		results[i] = Quaternion{
			W:         s0*q0.W + s1*q1.W,
			X:         s0*q0.X + s1*q1.X,
			Y:         s0*q0.Y + s1*q1.Y,
			Z:         s0*q0.Z + s1*q1.Z,
			Frequency: q0.Frequency + t*(q1.Frequency-q0.Frequency),
			Phase:     q0.Phase + t*(q1.Phase-q0.Phase),
		}
	}

	return results
}

// Squad performs Spherical Quadrangle interpolation for smooth curves.
// This creates a C¹ continuous path through multiple quaternions.
func Squad(q0, q1, q2, q3 Quaternion, t float64) Quaternion {
	// Compute intermediate control points
	s1 := SquadIntermediate(q0, q1, q2)
	s2 := SquadIntermediate(q1, q2, q3)

	// Double SLERP for smooth interpolation
	slerp1 := Slerp(q1, q2, t)
	slerp2 := Slerp(s1, s2, t)

	return Slerp(slerp1, slerp2, 2*t*(1-t))
}

// SquadIntermediate computes the intermediate control point for SQUAD.
func SquadIntermediate(qPrev, q, qNext Quaternion) Quaternion {
	qInv := q.Conjugate()
	// Compute log of differences
	diff1 := qInv.Mul(qPrev)
	diff2 := qInv.Mul(qNext)

	// Average in tangent space (simplified)
	avg := diff1.Add(diff2).Scale(-0.25)

	// Exp map back (simplified for unit quaternions)
	return q.Mul(Exp(avg))
}

// Exp computes the quaternion exponential e^q.
// Used for tangent space operations.
func Exp(q Quaternion) Quaternion {
	// For pure quaternions (w=0), this is the axis-angle formula
	vNorm := math.Sqrt(q.X*q.X + q.Y*q.Y + q.Z*q.Z)
	if vNorm < 1e-10 {
		return Quaternion{W: math.Exp(q.W), X: 0, Y: 0, Z: 0, Frequency: q.Frequency}
	}

	expW := math.Exp(q.W)
	s := expW * math.Sin(vNorm) / vNorm

	return Quaternion{
		W:         expW * math.Cos(vNorm),
		X:         s * q.X,
		Y:         s * q.Y,
		Z:         s * q.Z,
		Frequency: q.Frequency,
		Phase:     q.Phase,
	}
}

// Log computes the quaternion logarithm ln(q).
// Inverse of Exp.
func Log(q Quaternion) Quaternion {
	norm := q.Magnitude()
	vNorm := math.Sqrt(q.X*q.X + q.Y*q.Y + q.Z*q.Z)

	if vNorm < 1e-10 {
		return Quaternion{W: math.Log(norm), X: 0, Y: 0, Z: 0, Frequency: q.Frequency}
	}

	theta := math.Atan2(vNorm, q.W)
	s := theta / vNorm

	return Quaternion{
		W:         math.Log(norm),
		X:         s * q.X,
		Y:         s * q.Y,
		Z:         s * q.Z,
		Frequency: q.Frequency,
		Phase:     q.Phase,
	}
}

// Power computes q^t (quaternion power).
// Useful for smooth interpolation and path parameterization.
func (q Quaternion) Power(t float64) Quaternion {
	return Exp(Log(q).Scale(t))
}

// IsUnit returns true if the quaternion is approximately unit length.
func (q Quaternion) IsUnit(epsilon float64) bool {
	magSq := q.MagnitudeSquared()
	return math.Abs(magSq-1.0) < epsilon
}

// Equals returns true if two quaternions are approximately equal.
func (q Quaternion) Equals(other Quaternion, epsilon float64) bool {
	return math.Abs(q.W-other.W) < epsilon &&
		math.Abs(q.X-other.X) < epsilon &&
		math.Abs(q.Y-other.Y) < epsilon &&
		math.Abs(q.Z-other.Z) < epsilon
}

// EqualsRotation returns true if two quaternions represent the same rotation.
// Note: q and -q represent the same rotation!
func (q Quaternion) EqualsRotation(other Quaternion, epsilon float64) bool {
	return q.Equals(other, epsilon) || q.Equals(other.Negate(), epsilon)
}

// ToAxisAngle converts the quaternion to axis-angle representation.
// Returns (ax, ay, az, angle) where (ax, ay, az) is a unit vector.
func (q Quaternion) ToAxisAngle() (float64, float64, float64, float64) {
	// Ensure we're working with a unit quaternion
	qn := q.Normalize()

	angle := 2 * math.Acos(qn.W)
	s := math.Sqrt(1 - qn.W*qn.W)

	if s < 1e-10 {
		// No rotation, arbitrary axis
		return 1, 0, 0, 0
	}

	return qn.X / s, qn.Y / s, qn.Z / s, angle
}

// Vibrate returns the phase-shifted quaternion at time t based on frequency.
// This models oscillation/resonance on S³.
func (q Quaternion) Vibrate(t float64) Quaternion {
	phase := q.Phase + 2*math.Pi*q.Frequency*t
	// Rotate around the w-axis by the phase
	return FromAxisAngle(1, 0, 0, phase).Mul(q)
}

// String returns a string representation for debugging.
func (q Quaternion) String() string {
	return "Quaternion{W: " + formatFloat(q.W) +
		", X: " + formatFloat(q.X) +
		", Y: " + formatFloat(q.Y) +
		", Z: " + formatFloat(q.Z) +
		", Freq: " + formatFloat(q.Frequency) + "}"
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', 4, 64)
}
