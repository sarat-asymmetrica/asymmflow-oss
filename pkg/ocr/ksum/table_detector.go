package ksum

import (
	"image"
	"image/color"
)

// TableRegion represents a detected table in a document.
type TableRegion struct {
	Bounds      image.Rectangle  // Bounding box in original image
	Rows        int              // Number of detected rows
	Cols        int              // Number of detected columns
	Fingerprint *KsumFingerprint // K-sum fingerprint of region
	Confidence  float64          // Detection confidence (0-1)
}

// TableDetector detects tables using k-sum orthogonal analysis.
// Uses multi-scale sliding window with orthogonal fingerprinting.
type TableDetector struct {
	config *KsumConfig
}

// NewTableDetector creates a new table detector with given config.
func NewTableDetector(config *KsumConfig) *TableDetector {
	if config == nil {
		config = DefaultKsumConfig()
	}
	return &TableDetector{config: config}
}

// DetectTables finds all tables in an image.
//
// Algorithm:
// 1. Multi-scale sliding window (scales: 1.0, 0.5, 0.25)
// 2. For each window, compute k-sum fingerprint
// 3. If fingerprint indicates table, record detection
// 4. Merge overlapping detections (keep highest confidence)
//
// Returns: List of detected table regions sorted by confidence (descending)
func (d *TableDetector) DetectTables(img image.Image) []TableRegion {
	bounds := img.Bounds()
	var tables []TableRegion

	// Multi-scale approach to handle varying table sizes
	// Scale 1.0 = large tables, 0.5 = medium, 0.25 = small
	scales := []float64{1.0, 0.5, 0.25}

	for _, scale := range scales {
		windowW := int(float64(bounds.Dx()) * scale)
		windowH := int(float64(bounds.Dy()) * scale)

		// Skip if window too small
		if windowW < 100 || windowH < 100 {
			continue
		}

		// 50% overlap for robustness
		stepW := windowW / 2
		stepH := windowH / 2

		for y := bounds.Min.Y; y+windowH <= bounds.Max.Y; y += stepH {
			for x := bounds.Min.X; x+windowW <= bounds.Max.X; x += stepW {
				rect := image.Rect(x, y, x+windowW, y+windowH)
				subImg := extractSubImage(img, rect)

				fp := ComputeFingerprint(subImg, d.config)

				if fp.IsTable(d.config) {
					tables = append(tables, TableRegion{
						Bounds:      rect,
						Rows:        len(fp.RowPeaks),
						Cols:        len(fp.ColPeaks),
						Fingerprint: fp,
						Confidence:  fp.Orthogonality,
					})
				}
			}
		}
	}

	// Merge overlapping detections
	tables = mergeOverlapping(tables)

	return tables
}

// DetectTablesQuick is a faster version for large documents.
// Uses single-scale detection with larger step size (less overlap).
func (d *TableDetector) DetectTablesQuick(img image.Image) []TableRegion {
	bounds := img.Bounds()
	var tables []TableRegion

	// Single scale, larger steps for speed
	windowW := bounds.Dx() / 2
	windowH := bounds.Dy() / 2
	stepW := windowW // No overlap
	stepH := windowH

	for y := bounds.Min.Y; y+windowH <= bounds.Max.Y; y += stepH {
		for x := bounds.Min.X; x+windowW <= bounds.Max.X; x += stepW {
			rect := image.Rect(x, y, x+windowW, y+windowH)
			subImg := extractSubImage(img, rect)

			fp := ComputeFingerprint(subImg, d.config)

			if fp.IsTable(d.config) {
				tables = append(tables, TableRegion{
					Bounds:      rect,
					Rows:        len(fp.RowPeaks),
					Cols:        len(fp.ColPeaks),
					Fingerprint: fp,
					Confidence:  fp.Orthogonality,
				})
			}
		}
	}

	return tables
}

// extractSubImage extracts a rectangular region from an image.
// Returns a view (no copying) for efficiency.
func extractSubImage(img image.Image, rect image.Rectangle) image.Image {
	return &subImage{img: img, bounds: rect}
}

// subImage is a view into a rectangular region of another image.
type subImage struct {
	img    image.Image
	bounds image.Rectangle
}

func (s *subImage) ColorModel() color.Model { return s.img.ColorModel() }
func (s *subImage) Bounds() image.Rectangle { return s.bounds }
func (s *subImage) At(x, y int) color.Color { return s.img.At(x, y) }

// mergeOverlapping merges overlapping table detections.
// Strategy: Keep highest confidence detection, mark overlapping as used.
func mergeOverlapping(tables []TableRegion) []TableRegion {
	if len(tables) == 0 {
		return tables
	}

	var result []TableRegion
	used := make([]bool, len(tables))

	// Process in order (confidence not yet sorted)
	for i := range tables {
		if used[i] {
			continue
		}

		best := tables[i]
		used[i] = true

		// Find all overlapping detections
		for j := i + 1; j < len(tables); j++ {
			if !used[j] && overlaps(best.Bounds, tables[j].Bounds) {
				// Keep higher confidence
				if tables[j].Confidence > best.Confidence {
					best = tables[j]
				}
				used[j] = true
			}
		}

		result = append(result, best)
	}

	return result
}

// overlaps checks if two rectangles overlap.
func overlaps(a, b image.Rectangle) bool {
	return a.Overlaps(b)
}

// iou computes intersection-over-union between two rectangles (0-1).
// Useful for more sophisticated merging strategies.
func iou(a, b image.Rectangle) float64 {
	inter := a.Intersect(b)
	if inter.Empty() {
		return 0
	}

	interArea := float64(inter.Dx() * inter.Dy())
	unionArea := float64(a.Dx()*a.Dy()+b.Dx()*b.Dy()) - interArea

	if unionArea == 0 {
		return 0
	}

	return interArea / unionArea
}
