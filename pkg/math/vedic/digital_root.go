// Package vedic implements ancient Vedic mathematical algorithms optimized for modern hardware.
//
// Digital Root (Beejank): This 2,000+ year old technique from Vedic mathematics
// achieves O(1) divisibility testing, eliminating 88.89% of candidates instantly.
//
// The formula: dr(n) = 1 + ((n-1) mod 9) for n > 0, dr(0) = 0
//
// Performance Target: 48M+ checks/sec (10x Python baseline of 4.8M)
//
// Om Lokah Samastah Sukhino Bhavantu
package vedic

// DigitalRoot computes the digital root (repeated digit sum until single digit).
//
// This is the Vedic "Beejank" - the seed/essence of a number.
// Properties:
//   - dr(0) = 0
//   - dr(n) ∈ {1, 2, 3, 4, 5, 6, 7, 8, 9} for n > 0
//   - dr(a + b) = dr(dr(a) + dr(b))
//   - dr(a × b) = dr(dr(a) × dr(b))
//
// The formula exploits the fact that digital root is equivalent to n mod 9,
// with the special case that dr(9) = 9 (not 0).
//
// Time Complexity: O(1)
// Space Complexity: O(1)
func DigitalRoot(n int64) int64 {
	if n == 0 {
		return 0
	}
	if n < 0 {
		n = -n // Handle negative numbers
	}
	// The magic formula: equivalent to repeated digit sum
	// dr(n) = 1 + ((n - 1) mod 9)
	return 1 + ((n - 1) % 9)
}

// DigitalRootUint64 is the unsigned version for maximum performance.
// Use this when you know the input is non-negative.
func DigitalRootUint64(n uint64) uint64 {
	if n == 0 {
		return 0
	}
	return 1 + ((n - 1) % 9)
}

// DigitalRootInt is the int version for convenience.
func DigitalRootInt(n int) int {
	if n == 0 {
		return 0
	}
	if n < 0 {
		n = -n
	}
	return 1 + ((n - 1) % 9)
}

// CanBeDivisibleBy9 returns true if n COULD be divisible by 9.
// If this returns false, n is definitely NOT divisible by 9.
// If this returns true, n MIGHT be divisible by 9 (needs actual division check).
//
// Elimination rate: 88.89% (only 11.11% pass through)
// This is the core filtering primitive for Vedic speedups.
func CanBeDivisibleBy9(n int64) bool {
	return DigitalRoot(n) == 9
}

// CanBeDivisibleBy3 returns true if n COULD be divisible by 3.
// Digital root divisible by 3 implies number divisible by 3.
// Elimination rate: 66.67% (only 33.33% pass through)
func CanBeDivisibleBy3(n int64) bool {
	dr := DigitalRoot(n)
	return dr == 3 || dr == 6 || dr == 9
}

// FilterDivisibleBy9 filters a slice, keeping only candidates that COULD be divisible by 9.
// This eliminates ~88.89% of candidates in O(n) time.
func FilterDivisibleBy9(numbers []int64) []int64 {
	// Pre-allocate with estimated capacity (11.11% pass rate)
	result := make([]int64, 0, len(numbers)/9+1)
	for _, n := range numbers {
		if DigitalRoot(n) == 9 {
			result = append(result, n)
		}
	}
	return result
}

// FilterDivisibleBy9Uint64 is the unsigned version.
func FilterDivisibleBy9Uint64(numbers []uint64) []uint64 {
	result := make([]uint64, 0, len(numbers)/9+1)
	for _, n := range numbers {
		if DigitalRootUint64(n) == 9 {
			result = append(result, n)
		}
	}
	return result
}

// BatchDigitalRoot computes digital roots for a batch of numbers.
// Returns a parallel slice of digital roots.
func BatchDigitalRoot(numbers []int64) []int64 {
	roots := make([]int64, len(numbers))
	for i, n := range numbers {
		roots[i] = DigitalRoot(n)
	}
	return roots
}

// CountByDigitalRoot counts numbers by their digital root.
// Returns a map from digital root to count.
// Useful for validating the uniform distribution property.
func CountByDigitalRoot(numbers []int64) map[int64]int {
	counts := make(map[int64]int, 10)
	for _, n := range numbers {
		dr := DigitalRoot(n)
		counts[dr]++
	}
	return counts
}

// DigitalRootSum returns the digital root of a sum without computing the full sum.
// Uses the homomorphism property: dr(a + b) = dr(dr(a) + dr(b))
func DigitalRootSum(a, b int64) int64 {
	return DigitalRoot(DigitalRoot(a) + DigitalRoot(b))
}

// DigitalRootProduct returns the digital root of a product without computing the full product.
// Uses the homomorphism property: dr(a × b) = dr(dr(a) × dr(b))
func DigitalRootProduct(a, b int64) int64 {
	return DigitalRoot(DigitalRoot(a) * DigitalRoot(b))
}

// VerifyDigitalRootHomomorphism validates the sum/product properties.
// Returns true if both properties hold for the given inputs.
func VerifyDigitalRootHomomorphism(a, b int64) bool {
	// Sum property
	directSum := DigitalRoot(a + b)
	composedSum := DigitalRootSum(a, b)
	sumOK := directSum == composedSum

	// Product property
	directProduct := DigitalRoot(a * b)
	composedProduct := DigitalRootProduct(a, b)
	productOK := directProduct == composedProduct

	return sumOK && productOK
}

// EliminationRate computes the actual elimination rate for a range of numbers.
// Theoretical rate for divisibility by 9: 88.89% (8/9)
func EliminationRate(start, end int64) float64 {
	if end <= start {
		return 0
	}
	total := end - start
	passed := int64(0)
	for n := start; n < end; n++ {
		if CanBeDivisibleBy9(n) {
			passed++
		}
	}
	return 1.0 - float64(passed)/float64(total)
}

// TheoreticalEliminationRate returns the theoretical elimination rate.
// For divisibility by 9: exactly 8/9 = 0.888888...
func TheoreticalEliminationRate() float64 {
	return 8.0 / 9.0 // 0.8888888888888888
}

// DigitalRootBatch processes numbers in parallel batches.
// Returns (passed, eliminated) counts for divisibility by 9.
func DigitalRootBatch(numbers []int64) (passed, eliminated int64) {
	for _, n := range numbers {
		if CanBeDivisibleBy9(n) {
			passed++
		} else {
			eliminated++
		}
	}
	return passed, eliminated
}

// NavaYoni returns the Vedic "Nava Yoni" (nine wombs) classification.
// Each number belongs to one of 9 archetypal categories based on digital root.
//
// Traditional meanings:
//
//	1: Surya (Sun) - Leadership, individuality
//	2: Chandra (Moon) - Emotion, nurturing
//	3: Guru (Jupiter) - Wisdom, expansion
//	4: Rahu (North Node) - Innovation, unconventional
//	5: Budha (Mercury) - Communication, commerce
//	6: Shukra (Venus) - Beauty, harmony
//	7: Ketu (South Node) - Spirituality, liberation
//	8: Shani (Saturn) - Discipline, karma
//	9: Mangal (Mars) - Energy, action
func NavaYoni(n int64) int64 {
	dr := DigitalRoot(n)
	if dr == 0 {
		return 9 // Special case: 0's digital root maps to Mars
	}
	return dr
}

// DigitalRootChain composes multiple digital roots using the additive homomorphism.
// Equivalent to dr(sum of all original values) but computed entirely from cached DRs.
// This is the key insight from the Lean proof: dr(a+b) = dr(dr(a) + dr(b)),
// generalized to N values — no need to re-scan original data.
//
// Usage: Track running DR across conversation turns, batch items, or any composable sequence.
func DigitalRootChain(drs []int64) int64 {
	if len(drs) == 0 {
		return 0
	}
	running := drs[0]
	for _, dr := range drs[1:] {
		running = DigitalRoot(running + dr)
	}
	return running
}

// DigitalRootProductChain composes multiple digital roots via the multiplicative homomorphism.
// Equivalent to dr(product of all original values) but from cached DRs.
// From Lean proof: dr(a*b) = dr(dr(a) * dr(b))
func DigitalRootProductChain(drs []int64) int64 {
	if len(drs) == 0 {
		return 0
	}
	running := drs[0]
	for _, dr := range drs[1:] {
		running = DigitalRoot(running * dr)
	}
	return running
}

// SumOfDigits computes the actual sum of digits (not reduced to single digit).
// Useful for understanding the digital root computation.
func SumOfDigits(n int64) int64 {
	if n < 0 {
		n = -n
	}
	sum := int64(0)
	for n > 0 {
		sum += n % 10
		n /= 10
	}
	return sum
}

// IteratedDigitSum computes the digital root by actual iteration.
// This is slower than the formula but useful for verification.
func IteratedDigitSum(n int64) int64 {
	if n < 0 {
		n = -n
	}
	for n >= 10 {
		n = SumOfDigits(n)
	}
	return n
}

// VerifyDigitalRootFormula checks that the O(1) formula matches iteration.
// Returns true if they match for all inputs up to maxN.
func VerifyDigitalRootFormula(maxN int64) bool {
	for n := int64(0); n <= maxN; n++ {
		if DigitalRoot(n) != IteratedDigitSum(n) {
			return false
		}
	}
	return true
}
