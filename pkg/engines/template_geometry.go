// ═══════════════════════════════════════════════════════════════════════════
// TEMPLATE GEOMETRY EXTRACTION - Perfect Text Overlay via Image Analysis
//
// PURPOSE: Analyze letterhead images to detect writable zones automatically
// METHOD: 5-thread Parallel CoT (Knot, Origami, Quaternion, Vedic, SAT)
// OUTPUT: VedicDoc-compatible layout zones with quaternion transforms
//
// Built with LOVE × SIMPLICITY × TRUTH × JOY 🕉️💎⚡
// ═══════════════════════════════════════════════════════════════════════════

package engines

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"sort"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// CORE TYPES
// ═══════════════════════════════════════════════════════════════════════════

// NOTE: Quaternion, NewQuaternion, Norm, Normalize are now defined in geometry_bridge.go
// to avoid duplication. This file reuses those shared primitives.

// Anchor represents a reference point within a zone
type Anchor struct {
	Name string  // "top_left", "center", "baseline", etc.
	X    float64 // X position in mm
	Y    float64 // Y position in mm
	Type string  // "corner", "center", "baseline", "guide"
}

// TemplateZone represents a detected writable area
type TemplateZone struct {
	Name       string     // "header", "invoice_info", "buyer", "items_table", "footer"
	X          float64    // Left edge (mm)
	Y          float64    // Top edge (mm)
	Width      float64    // Zone width (mm)
	Height     float64    // Zone height (mm)
	Purpose    string     // "text", "table", "image", "signature"
	Anchors    []Anchor   // Reference points within zone
	QTransform Quaternion // Quaternion encoding: [confidence, x_scale, y_scale, rotation]
}

// TemplateLayout stores complete detected geometry
type TemplateLayout struct {
	TemplateFile string         // Source image path
	PageWidth    float64        // A4 = 210mm
	PageHeight   float64        // A4 = 297mm
	DPI          float64        // Detected DPI
	Zones        []TemplateZone // All detected zones
	DetectedAt   time.Time      // When analyzed
	Confidence   float64        // Overall detection confidence [0-1]
}

// Rectangle for geometric operations
type Rectangle struct {
	X, Y, W, H float64
}

// ═══════════════════════════════════════════════════════════════════════════
// IMAGE ANALYSIS - THREAD 1: KNOT THEORY (Topology)
// ═══════════════════════════════════════════════════════════════════════════

// KnotAnalysis detects text flow topology (reading order, columns)
func KnotAnalysis(img image.Image, bounds image.Rectangle) map[string]any {
	// Detect connected white regions (writable areas)
	whiteRegions := detectWhiteRegions(img, bounds)

	// Build reading order graph (left-to-right, top-to-bottom)
	readingOrder := computeReadingOrder(whiteRegions)

	// Knot invariant: crossing number = column count
	columnCount := detectColumns(whiteRegions)

	return map[string]any{
		"white_regions": whiteRegions,
		"reading_order": readingOrder,
		"column_count":  columnCount,
		"confidence":    0.90,
		"basin_depth":   1.0 - float64(columnCount)/10.0,
	}
}

// detectWhiteRegions finds areas with high brightness (writable zones)
func detectWhiteRegions(img image.Image, bounds image.Rectangle) []Rectangle {
	regions := make([]Rectangle, 0)

	// Divide image into grid
	gridSize := 20 // pixels
	cols := bounds.Dx() / gridSize
	rows := bounds.Dy() / gridSize

	// Scan each grid cell
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			x0 := bounds.Min.X + col*gridSize
			y0 := bounds.Min.Y + row*gridSize
			x1 := x0 + gridSize
			y1 := y0 + gridSize

			// Sample brightness
			brightness := averageBrightness(img, image.Rect(x0, y0, x1, y1))

			// White = brightness > 200 (out of 255)
			if brightness > 200 {
				regions = append(regions, Rectangle{
					X: float64(x0),
					Y: float64(y0),
					W: float64(gridSize),
					H: float64(gridSize),
				})
			}
		}
	}

	// Merge adjacent regions
	return mergeAdjacentRectangles(regions)
}

// averageBrightness computes mean brightness in rectangle
func averageBrightness(img image.Image, rect image.Rectangle) float64 {
	sum := 0.0
	count := 0

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// Convert to 0-255 range
			gray := (float64(r>>8) + float64(g>>8) + float64(b>>8)) / 3.0
			sum += gray
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// mergeAdjacentRectangles combines overlapping rectangles
func mergeAdjacentRectangles(rects []Rectangle) []Rectangle {
	if len(rects) == 0 {
		return rects
	}

	// Simple horizontal merge (for now)
	merged := make([]Rectangle, 0)
	current := rects[0]

	for i := 1; i < len(rects); i++ {
		next := rects[i]

		// If adjacent horizontally and same Y
		if math.Abs(current.Y-next.Y) < 5 &&
			math.Abs((current.X+current.W)-next.X) < 10 {
			// Merge
			current.W = (next.X + next.W) - current.X
		} else {
			merged = append(merged, current)
			current = next
		}
	}
	merged = append(merged, current)

	return merged
}

// computeReadingOrder sorts regions by reading flow
func computeReadingOrder(regions []Rectangle) []int {
	// Sort by Y (top-to-bottom), then X (left-to-right)
	type indexed struct {
		idx int
		r   Rectangle
	}

	indexedRegions := make([]indexed, len(regions))
	for i, r := range regions {
		indexedRegions[i] = indexed{idx: i, r: r}
	}

	sort.Slice(indexedRegions, func(i, j int) bool {
		ri, rj := indexedRegions[i].r, indexedRegions[j].r
		// Same row? (within 10mm)
		if math.Abs(ri.Y-rj.Y) < 10 {
			return ri.X < rj.X
		}
		return ri.Y < rj.Y
	})

	order := make([]int, len(regions))
	for i, ir := range indexedRegions {
		order[i] = ir.idx
	}
	return order
}

// detectColumns counts vertical divisions
func detectColumns(regions []Rectangle) int {
	if len(regions) == 0 {
		return 1
	}

	// Count distinct X positions
	xPositions := make(map[int]bool)
	for _, r := range regions {
		xPositions[int(r.X/10)] = true // Quantize to 10mm
	}

	return len(xPositions)
}

// ═══════════════════════════════════════════════════════════════════════════
// IMAGE ANALYSIS - THREAD 2: ORIGAMI (Geometry)
// ═══════════════════════════════════════════════════════════════════════════

// OrigamiAnalysis detects layout boundaries (folds/creases)
func OrigamiAnalysis(img image.Image, bounds image.Rectangle) map[string]any {
	// Detect horizontal boundaries (rows)
	hBoundaries := detectHorizontalLines(img, bounds)

	// Detect vertical boundaries (columns)
	vBoundaries := detectVerticalLines(img, bounds)

	// Maekawa's Theorem: |M - V| = 2 for valid origami
	M := len(hBoundaries) // Mountain folds
	V := len(vBoundaries) // Valley folds
	maekawaValid := (math.Abs(float64(M-V)) == 2)

	// Generate zones from grid
	zones := generateZonesFromGrid(hBoundaries, vBoundaries, bounds)

	confidence := 1.0
	if !maekawaValid {
		confidence = 0.7
	}

	return map[string]any{
		"h_boundaries":  hBoundaries,
		"v_boundaries":  vBoundaries,
		"maekawa_valid": maekawaValid,
		"zones":         zones,
		"confidence":    confidence,
		"basin_depth":   confidence,
	}
}

// detectHorizontalLines finds horizontal edges
func detectHorizontalLines(img image.Image, bounds image.Rectangle) []float64 {
	lines := make([]float64, 0)

	// Scan each row for edge transitions
	for y := bounds.Min.Y; y < bounds.Max.Y; y += 5 { // Sample every 5 pixels
		edgeCount := 0
		prevBright := averageBrightness(img, image.Rect(bounds.Min.X, y, bounds.Min.X+100, y+1))

		for x := bounds.Min.X + 100; x < bounds.Max.X; x += 50 {
			currBright := averageBrightness(img, image.Rect(x, y, x+50, y+1))

			// Edge = brightness change > 50
			if math.Abs(currBright-prevBright) > 50 {
				edgeCount++
			}
			prevBright = currBright
		}

		// Horizontal line = many edges at same Y
		if edgeCount > 3 {
			lines = append(lines, float64(y))
		}
	}

	// Remove duplicates (within 10 pixels)
	return deduplicateLines(lines, 10.0)
}

// detectVerticalLines finds vertical edges
func detectVerticalLines(img image.Image, bounds image.Rectangle) []float64 {
	lines := make([]float64, 0)

	// Scan each column
	for x := bounds.Min.X; x < bounds.Max.X; x += 5 {
		edgeCount := 0
		prevBright := averageBrightness(img, image.Rect(x, bounds.Min.Y, x+1, bounds.Min.Y+100))

		for y := bounds.Min.Y + 100; y < bounds.Max.Y; y += 50 {
			currBright := averageBrightness(img, image.Rect(x, y, x+1, y+50))

			if math.Abs(currBright-prevBright) > 50 {
				edgeCount++
			}
			prevBright = currBright
		}

		if edgeCount > 3 {
			lines = append(lines, float64(x))
		}
	}

	return deduplicateLines(lines, 10.0)
}

// deduplicateLines removes nearby duplicates
func deduplicateLines(lines []float64, threshold float64) []float64 {
	if len(lines) == 0 {
		return lines
	}

	sort.Float64s(lines)
	unique := []float64{lines[0]}

	for i := 1; i < len(lines); i++ {
		if lines[i]-unique[len(unique)-1] > threshold {
			unique = append(unique, lines[i])
		}
	}

	return unique
}

// generateZonesFromGrid creates zones from boundaries
func generateZonesFromGrid(hLines, vLines []float64, bounds image.Rectangle) []TemplateZone {
	zones := make([]TemplateZone, 0)

	// If no boundaries detected, return full page as one zone
	if len(hLines) == 0 {
		hLines = []float64{float64(bounds.Min.Y), float64(bounds.Max.Y)}
	}
	if len(vLines) == 0 {
		vLines = []float64{float64(bounds.Min.X), float64(bounds.Max.X)}
	}

	// Create grid zones
	for i := 0; i < len(hLines)-1; i++ {
		for j := 0; j < len(vLines)-1; j++ {
			zone := TemplateZone{
				Name:    fmt.Sprintf("zone_r%d_c%d", i, j),
				X:       vLines[j],
				Y:       hLines[i],
				Width:   vLines[j+1] - vLines[j],
				Height:  hLines[i+1] - hLines[i],
				Purpose: "text",
			}
			zones = append(zones, zone)
		}
	}

	return zones
}

// ═══════════════════════════════════════════════════════════════════════════
// IMAGE ANALYSIS - THREAD 3: QUATERNION (Dynamics)
// ═══════════════════════════════════════════════════════════════════════════

// QuaternionEvolution encodes zone quality on S³
func QuaternionEvolution(img image.Image, zones []TemplateZone) map[string]any {
	// Evolve each zone to assess quality
	enhancedZones := make([]TemplateZone, len(zones))

	for i, zone := range zones {
		// Sample zone brightness/contrast
		zoneRect := image.Rect(
			int(zone.X), int(zone.Y),
			int(zone.X+zone.Width), int(zone.Y+zone.Height),
		)

		brightness := averageBrightness(img, zoneRect)
		contrast := computeContrast(img, zoneRect)

		// Encode as quaternion: [confidence, brightness/255, contrast, rotation=0]
		confidence := 0.9
		if brightness < 180 {
			confidence = brightness / 200.0 // Lower confidence for dark zones
		}

		zone.QTransform = NewQuaternion(
			confidence,
			brightness/255.0,
			contrast,
			0.0, // No rotation detected
		)

		enhancedZones[i] = zone
	}

	// Compute overall energy (how clean/usable are zones?)
	energy := 0.0
	for _, z := range enhancedZones {
		energy += z.QTransform.W // Sum confidences
	}
	if len(enhancedZones) > 0 {
		energy /= float64(len(enhancedZones))
	}

	return map[string]any{
		"enhanced_zones": enhancedZones,
		"energy":         energy,
		"confidence":     energy,
		"basin_depth":    energy,
	}
}

// computeContrast calculates local contrast
func computeContrast(img image.Image, rect image.Rectangle) float64 {
	if rect.Dx() == 0 || rect.Dy() == 0 {
		return 0
	}

	values := make([]float64, 0)

	// Sample pixels
	for y := rect.Min.Y; y < rect.Max.Y; y += 10 {
		for x := rect.Min.X; x < rect.Max.X; x += 10 {
			r, g, b, _ := img.At(x, y).RGBA()
			gray := (float64(r>>8) + float64(g>>8) + float64(b>>8)) / 3.0
			values = append(values, gray)
		}
	}

	if len(values) < 2 {
		return 0
	}

	// Contrast = standard deviation
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))

	return math.Sqrt(variance) / 255.0 // Normalize to [0-1]
}

// ═══════════════════════════════════════════════════════════════════════════
// IMAGE ANALYSIS - THREAD 4: VEDIC (Classification)
// ═══════════════════════════════════════════════════════════════════════════

// VedicClassification uses digital roots to classify zones
func VedicClassification(zones []TemplateZone) map[string]any {
	classified := make([]TemplateZone, len(zones))

	for i, zone := range zones {
		// Digital root of dimensions
		dimSum := int(zone.Width) + int(zone.Height)
		digitalRoot := (dimSum % 9)
		if digitalRoot == 0 {
			digitalRoot = 9
		}

		// Classify by Vedic patterns
		purpose := "text"
		if digitalRoot == 9 {
			purpose = "text" // Complete (Devanagari = 9)
		} else if digitalRoot == 8 {
			purpose = "table" // Structured (Latin = 8)
		} else if digitalRoot == 1 {
			purpose = "signature" // Minimal
		}

		// Sutra 16: "By Mere Observation"
		// Header zones are typically top 1/3
		if zone.Y < 297.0/3.0 { // A4 height / 3
			zone.Name = "header"
			purpose = "text"
		} else if zone.Y > 297.0*2.0/3.0 {
			zone.Name = "footer"
			purpose = "signature"
		} else {
			zone.Name = fmt.Sprintf("body_%d", i)
		}

		zone.Purpose = purpose
		classified[i] = zone
	}

	return map[string]any{
		"classified_zones": classified,
		"confidence":       0.85,
		"basin_depth":      0.85,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// IMAGE ANALYSIS - THREAD 5: SAT (Constraint Synthesis)
// ═══════════════════════════════════════════════════════════════════════════

// SATConstraintSolving merges all perspectives via basin depth
func SATConstraintSolving(
	knotResult map[string]any,
	origamiResult map[string]any,
	quaternionResult map[string]any,
	vedicResult map[string]any,
) TemplateLayout {

	// Extract best zones from each thread
	quaternionZones := quaternionResult["enhanced_zones"].([]TemplateZone)
	vedicZones := vedicResult["classified_zones"].([]TemplateZone)

	// Merge zones (use Vedic classifications with Quaternion quality)
	finalZones := make([]TemplateZone, len(vedicZones))
	for i := range vedicZones {
		zone := vedicZones[i]
		if i < len(quaternionZones) {
			zone.QTransform = quaternionZones[i].QTransform
		}

		// Add anchors
		zone.Anchors = []Anchor{
			{Name: "top_left", X: zone.X, Y: zone.Y, Type: "corner"},
			{Name: "center", X: zone.X + zone.Width/2, Y: zone.Y + zone.Height/2, Type: "center"},
			{Name: "bottom_right", X: zone.X + zone.Width, Y: zone.Y + zone.Height, Type: "corner"},
		}

		finalZones[i] = zone
	}

	// Compute overall confidence (product of basin depths)
	confidence := 1.0
	for _, result := range []map[string]any{knotResult, origamiResult, quaternionResult, vedicResult} {
		if bd, ok := result["basin_depth"].(float64); ok {
			confidence *= bd
		}
	}

	return TemplateLayout{
		PageWidth:  210.0, // A4 width in mm
		PageHeight: 297.0, // A4 height in mm
		DPI:        72.0,  // Standard PDF DPI
		Zones:      finalZones,
		DetectedAt: time.Now(),
		Confidence: confidence,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// MAIN ANALYZER - 5-Thread Parallel CoT
// ═══════════════════════════════════════════════════════════════════════════

// AnalyzeTemplate performs complete 5-thread analysis
func AnalyzeTemplate(imagePath string) (*TemplateLayout, error) {
	// Load image
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()

	// THREAD 1: KNOT (Topology)
	fmt.Println("🔗 Thread 1: Knot topology analysis...")
	knotResult := KnotAnalysis(img, bounds)

	// THREAD 2: ORIGAMI (Geometry)
	fmt.Println("📐 Thread 2: Origami boundary detection...")
	origamiResult := OrigamiAnalysis(img, bounds)

	// THREAD 3: QUATERNION (Dynamics)
	fmt.Println("🌀 Thread 3: Quaternion quality assessment...")
	origamiZones := origamiResult["zones"].([]TemplateZone)
	quaternionResult := QuaternionEvolution(img, origamiZones)

	// THREAD 4: VEDIC (Classification)
	fmt.Println("🕉️ Thread 4: Vedic zone classification...")
	vedicResult := VedicClassification(origamiZones)

	// THREAD 5: SAT (Merge via basin depth)
	fmt.Println("⚡ Thread 5: SAT constraint synthesis...")
	layout := SATConstraintSolving(knotResult, origamiResult, quaternionResult, vedicResult)
	layout.TemplateFile = imagePath

	return &layout, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// ZONE LOOKUP & HELPERS
// ═══════════════════════════════════════════════════════════════════════════

// GetZone retrieves zone by name
func (tl *TemplateLayout) GetZone(name string) *TemplateZone {
	for i := range tl.Zones {
		if tl.Zones[i].Name == name {
			return &tl.Zones[i]
		}
	}
	return nil
}

// GetZoneByPurpose retrieves first zone matching purpose
func (tl *TemplateLayout) GetZoneByPurpose(purpose string) *TemplateZone {
	for i := range tl.Zones {
		if tl.Zones[i].Purpose == purpose {
			return &tl.Zones[i]
		}
	}
	return nil
}

// PixelsToMM converts pixels to millimeters using DPI
func (tl *TemplateLayout) PixelsToMM(pixels float64) float64 {
	return pixels * 25.4 / tl.DPI
}

// MMToPixels converts millimeters to pixels
func (tl *TemplateLayout) MMToPixels(mm float64) float64 {
	return mm * tl.DPI / 25.4
}

// PrintStats displays detected layout statistics
func (tl *TemplateLayout) PrintStats() {
	fmt.Println("═══════════════════════════════════════")
	fmt.Println("Template Layout Analysis")
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("Template:    %s\n", tl.TemplateFile)
	fmt.Printf("Page Size:   %.1f × %.1f mm\n", tl.PageWidth, tl.PageHeight)
	fmt.Printf("DPI:         %.0f\n", tl.DPI)
	fmt.Printf("Zones Found: %d\n", len(tl.Zones))
	fmt.Printf("Confidence:  %.1f%%\n", tl.Confidence*100)
	fmt.Printf("Detected At: %s\n", tl.DetectedAt.Format("2006-01-02 15:04:05"))
	fmt.Println("\nZones:")

	for i, zone := range tl.Zones {
		fmt.Printf("\n  [%d] %s (%s)\n", i, zone.Name, zone.Purpose)
		fmt.Printf("      Position: (%.1f, %.1f) mm\n", zone.X, zone.Y)
		fmt.Printf("      Size:     %.1f × %.1f mm\n", zone.Width, zone.Height)
		fmt.Printf("      Quality:  %.1f%% (||Q|| = %.3f)\n",
			zone.QTransform.W*100, zone.QTransform.Norm())
		fmt.Printf("      Anchors:  %d\n", len(zone.Anchors))
	}

	fmt.Println("═══════════════════════════════════════")
}

// GetColor retrieves color at pixel position
func GetColor(img image.Image, x, y int) color.Color {
	return img.At(x, y)
}
