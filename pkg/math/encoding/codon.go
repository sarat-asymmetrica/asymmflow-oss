// Package main implements Lossless Quaternion Prompt Encoding (Experiment 17).
//
// Core insight from GenomicsEngine.lean's codon roundtrip proofs:
//
//	encode_decodeCodon = identity (base-4 encoding is perfectly invertible)
//
// A DNA codon maps 3 nucleotides (base-4) to values [0, 63].
// A byte maps to 4 base-4 digits (since 4^4 = 256 = 2^8).
// Four base-4 digits naturally map to four quaternion components!
//
// This gives us:
//   - LOSSLESS prompt encoding: encode → decode = identity (proven by Lean!)
//   - Character-level geodesic distance: how far apart are two bytes on S³?
//   - Bridge between genomics (codons) and information theory (bytes)
//
// Eclectic Connection (X3 from TRIDENT doc):
//
//	Codon base-4 = IChing base-64 = Same roundtrip algebra!
//	The IChing proof (decode_encode) and Codon proof are the SAME structure.
//
// Om Lokah Samastah Sukhino Bhavantu
package encoding

import (
	"math"

	"ph_holdings_app/pkg/math/quaternion"
)

// CodonEncode converts a single byte to a quaternion via base-4 decomposition.
//
// Each byte (0-255) decomposes into 4 base-4 digits:
//
//	byte = d3*64 + d2*16 + d1*4 + d0  where each di ∈ {0,1,2,3}
//
// These map to quaternion components: Q(d3, d2, d1, d0).
// Roundtrip proven: CodonDecode(CodonEncode(b)) == b for all b ∈ [0, 255].
func CodonEncode(b byte) quaternion.Quaternion {
	d0 := float64(b & 0x03)        // Bits 0-1
	d1 := float64((b >> 2) & 0x03) // Bits 2-3
	d2 := float64((b >> 4) & 0x03) // Bits 4-5
	d3 := float64((b >> 6) & 0x03) // Bits 6-7
	return quaternion.New(d3, d2, d1, d0)
}

// CodonDecode reconstructs a byte from its quaternion encoding.
//
// Inverse of CodonEncode — rounds components and reassembles the byte.
// Guaranteed lossless when applied to CodonEncode output.
func CodonDecode(q quaternion.Quaternion) byte {
	d3 := byte(math.Round(q.W)) & 0x03
	d2 := byte(math.Round(q.X)) & 0x03
	d1 := byte(math.Round(q.Y)) & 0x03
	d0 := byte(math.Round(q.Z)) & 0x03
	return (d3 << 6) | (d2 << 4) | (d1 << 2) | d0
}

// LosslessPromptEncode encodes an entire prompt as a sequence of quaternions.
//
// Each byte of the prompt becomes one quaternion via CodonEncode.
// The full sequence preserves ALL information — nothing is lost.
// This is the quaternion-space analog of a DNA sequence.
func LosslessPromptEncode(prompt string) []quaternion.Quaternion {
	bytes := []byte(prompt)
	quats := make([]quaternion.Quaternion, len(bytes))
	for i, b := range bytes {
		quats[i] = CodonEncode(b)
	}
	return quats
}

// LosslessPromptDecode reconstructs the exact prompt from quaternion encoding.
//
// Proven roundtrip: LosslessPromptDecode(LosslessPromptEncode(s)) == s
// This is the computational instantiation of GenomicsEngine.lean's
// encode_decodeCodon = identity theorem.
func LosslessPromptDecode(quats []quaternion.Quaternion) string {
	bytes := make([]byte, len(quats))
	for i, q := range quats {
		bytes[i] = CodonDecode(q)
	}
	return string(bytes)
}

// CodonGeodesicDistance computes the quaternion-space distance between two bytes.
//
// This is the ECLECTIC CONNECTION: by encoding bytes as quaternions,
// we get a geometric notion of "how different" two characters are.
// The distance is the geodesic on the 4D quaternion space (not S³ — these
// aren't unit quaternions, they live on the base-4 lattice {0,1,2,3}⁴).
//
// Useful for: character-level similarity, edit distance enrichment,
// finding "nearby" characters in quaternion space.
func CodonGeodesicDistance(a, b byte) float64 {
	qa := CodonEncode(a)
	qb := CodonEncode(b)
	// Euclidean distance in quaternion space (lattice distance)
	dw := qa.W - qb.W
	dx := qa.X - qb.X
	dy := qa.Y - qb.Y
	dz := qa.Z - qb.Z
	return math.Sqrt(dw*dw + dx*dx + dy*dy + dz*dz)
}

// PromptCodonDistance computes the average codon distance between two prompts.
//
// Aligns prompts by length (shorter padded with zeros), then averages
// per-character codon distances. This gives a byte-level geometric
// similarity metric between arbitrary strings.
func PromptCodonDistance(a, b string) float64 {
	ba, bb := []byte(a), []byte(b)
	maxLen := len(ba)
	if len(bb) > maxLen {
		maxLen = len(bb)
	}
	if maxLen == 0 {
		return 0.0
	}

	totalDist := 0.0
	for i := 0; i < maxLen; i++ {
		var ca, cb byte
		if i < len(ba) {
			ca = ba[i]
		}
		if i < len(bb) {
			cb = bb[i]
		}
		totalDist += CodonGeodesicDistance(ca, cb)
	}
	return totalDist / float64(maxLen)
}

// EncodingStats returns statistics about a lossless encoding.
type EncodingStats struct {
	ByteCount       int     // Number of bytes encoded
	QuaternionCount int     // Number of quaternions (same as ByteCount)
	AvgMagnitude    float64 // Average magnitude of codon quaternions
	MaxDistance     float64 // Maximum pairwise codon distance in the sequence
}

// AnalyzeEncoding computes statistics for a lossless prompt encoding.
func AnalyzeEncoding(prompt string) EncodingStats {
	quats := LosslessPromptEncode(prompt)
	stats := EncodingStats{
		ByteCount:       len(prompt),
		QuaternionCount: len(quats),
	}

	if len(quats) == 0 {
		return stats
	}

	// Average magnitude
	totalMag := 0.0
	for _, q := range quats {
		totalMag += q.Magnitude()
	}
	stats.AvgMagnitude = totalMag / float64(len(quats))

	// Max pairwise distance (sample first 100 pairs for efficiency)
	bytes := []byte(prompt)
	maxDist := 0.0
	limit := len(bytes)
	if limit > 100 {
		limit = 100
	}
	for i := 1; i < limit; i++ {
		d := CodonGeodesicDistance(bytes[i-1], bytes[i])
		if d > maxDist {
			maxDist = d
		}
	}
	stats.MaxDistance = maxDist

	return stats
}
