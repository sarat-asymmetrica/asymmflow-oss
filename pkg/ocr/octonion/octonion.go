package octonion

import (
	"math"
)

// Octonion represents an 8-dimensional hypercomplex number
// e0=1, e1=i, e2=j, e3=k, e4=l, e5=li, e6=lj, e7=lk
// Used for color document processing: RGB + spatial + confidence + context
type Octonion struct {
	E [8]float64
}

// NewOctonion creates a new octonion
func NewOctonion(e0, e1, e2, e3, e4, e5, e6, e7 float64) Octonion {
	return Octonion{E: [8]float64{e0, e1, e2, e3, e4, e5, e6, e7}}
}

// Zero returns the zero octonion
func Zero() Octonion {
	return Octonion{}
}

// One returns the unit octonion (1,0,0,0,0,0,0,0)
func One() Octonion {
	return Octonion{E: [8]float64{1, 0, 0, 0, 0, 0, 0, 0}}
}

// FromPixel creates an octonion from pixel data
// e0=R, e1=G, e2=B, e3=A, e4=x_norm, e5=y_norm, e6=confidence, e7=context
func FromPixel(r, g, b, a uint8, x, y, width, height int, confidence, context float64) Octonion {
	return Octonion{E: [8]float64{
		float64(r) / 255.0,           // e0: Red (normalized)
		float64(g) / 255.0,           // e1: Green
		float64(b) / 255.0,           // e2: Blue
		float64(a) / 255.0,           // e3: Alpha
		float64(x) / float64(width),  // e4: X position
		float64(y) / float64(height), // e5: Y position
		confidence,                   // e6: OCR confidence
		context,                      // e7: Context score
	}}
}

// ToRGBA converts octonion back to RGBA values
func (o Octonion) ToRGBA() (r, g, b, a uint8) {
	return uint8(clamp(o.E[0]*255, 0, 255)),
		uint8(clamp(o.E[1]*255, 0, 255)),
		uint8(clamp(o.E[2]*255, 0, 255)),
		uint8(clamp(o.E[3]*255, 0, 255))
}

// Norm computes the magnitude ||o||
func (o Octonion) Norm() float64 {
	var sum float64
	for i := 0; i < 8; i++ {
		sum += o.E[i] * o.E[i]
	}
	return math.Sqrt(sum)
}

// Normalize returns unit octonion
func (o Octonion) Normalize() Octonion {
	n := o.Norm()
	if n < 1e-10 {
		return One()
	}
	var result Octonion
	for i := 0; i < 8; i++ {
		result.E[i] = o.E[i] / n
	}
	return result
}

// Conjugate returns the octonion conjugate
func (o Octonion) Conjugate() Octonion {
	return Octonion{E: [8]float64{
		o.E[0], -o.E[1], -o.E[2], -o.E[3],
		-o.E[4], -o.E[5], -o.E[6], -o.E[7],
	}}
}

// Add adds two octonions
func (o Octonion) Add(other Octonion) Octonion {
	var result Octonion
	for i := 0; i < 8; i++ {
		result.E[i] = o.E[i] + other.E[i]
	}
	return result
}

// Sub subtracts two octonions
func (o Octonion) Sub(other Octonion) Octonion {
	var result Octonion
	for i := 0; i < 8; i++ {
		result.E[i] = o.E[i] - other.E[i]
	}
	return result
}

// Scale multiplies by scalar
func (o Octonion) Scale(s float64) Octonion {
	var result Octonion
	for i := 0; i < 8; i++ {
		result.E[i] = o.E[i] * s
	}
	return result
}

// Mul performs octonion multiplication (non-associative!)
// Uses Cayley-Dickson construction
func (o Octonion) Mul(other Octonion) Octonion {
	// Split into two quaternions: o = (a, b), other = (c, d)
	// Product = (ac - d*b, da + bc*)
	a := [4]float64{o.E[0], o.E[1], o.E[2], o.E[3]}
	b := [4]float64{o.E[4], o.E[5], o.E[6], o.E[7]}
	c := [4]float64{other.E[0], other.E[1], other.E[2], other.E[3]}
	d := [4]float64{other.E[4], other.E[5], other.E[6], other.E[7]}

	// Quaternion multiplication helper
	qMul := func(p, q [4]float64) [4]float64 {
		return [4]float64{
			p[0]*q[0] - p[1]*q[1] - p[2]*q[2] - p[3]*q[3],
			p[0]*q[1] + p[1]*q[0] + p[2]*q[3] - p[3]*q[2],
			p[0]*q[2] - p[1]*q[3] + p[2]*q[0] + p[3]*q[1],
			p[0]*q[3] + p[1]*q[2] - p[2]*q[1] + p[3]*q[0],
		}
	}

	// Quaternion conjugate
	qConj := func(q [4]float64) [4]float64 {
		return [4]float64{q[0], -q[1], -q[2], -q[3]}
	}

	// Quaternion subtraction
	qSub := func(p, q [4]float64) [4]float64 {
		return [4]float64{p[0] - q[0], p[1] - q[1], p[2] - q[2], p[3] - q[3]}
	}

	// Quaternion addition
	qAdd := func(p, q [4]float64) [4]float64 {
		return [4]float64{p[0] + q[0], p[1] + q[1], p[2] + q[2], p[3] + q[3]}
	}

	// First half: ac - d*b*
	ac := qMul(a, c)
	bConj := qConj(b)
	db := qMul(d, bConj)
	first := qSub(ac, db)

	// Second half: da + bc*
	da := qMul(d, a)
	cConj := qConj(c)
	bc := qMul(b, cConj)
	second := qAdd(da, bc)

	return Octonion{E: [8]float64{
		first[0], first[1], first[2], first[3],
		second[0], second[1], second[2], second[3],
	}}
}

// Dot computes dot product
func (o Octonion) Dot(other Octonion) float64 {
	var sum float64
	for i := 0; i < 8; i++ {
		sum += o.E[i] * other.E[i]
	}
	return sum
}

// Distance computes Euclidean distance between octonions
func (o Octonion) Distance(other Octonion) float64 {
	var sum float64
	for i := 0; i < 8; i++ {
		d := o.E[i] - other.E[i]
		sum += d * d
	}
	return math.Sqrt(sum)
}

// Project projects onto the RGB subspace (first 3 components)
func (o Octonion) ProjectRGB() Octonion {
	return Octonion{E: [8]float64{o.E[0], o.E[1], o.E[2], 0, 0, 0, 0, 0}}
}

// ProjectSpatial projects onto spatial subspace (components 4-5)
func (o Octonion) ProjectSpatial() Octonion {
	return Octonion{E: [8]float64{0, 0, 0, 0, o.E[4], o.E[5], 0, 0}}
}

// ColorMagnitude returns RGB magnitude
func (o Octonion) ColorMagnitude() float64 {
	return math.Sqrt(o.E[0]*o.E[0] + o.E[1]*o.E[1] + o.E[2]*o.E[2])
}

// Grayscale converts RGB to grayscale using standard weights
func (o Octonion) Grayscale() float64 {
	return 0.299*o.E[0] + 0.587*o.E[1] + 0.114*o.E[2]
}

// IsInk determines if pixel is likely ink (vs background)
// Uses color magnitude and distance from white
func (o Octonion) IsInk(threshold float64) bool {
	// Distance from white (1,1,1)
	white := Octonion{E: [8]float64{1, 1, 1, 1, 0, 0, 0, 0}}
	dist := o.ProjectRGB().Distance(white.ProjectRGB())
	return dist > threshold
}

// InkStrength returns ink strength (0-1, higher = darker ink)
func (o Octonion) InkStrength() float64 {
	// Inverse of grayscale (dark = high strength)
	return 1.0 - o.Grayscale()
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
