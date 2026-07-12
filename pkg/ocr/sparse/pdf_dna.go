package sparse

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"sync"
)

// PDFElement represents a recurring document element
type PDFElement struct {
	Hash        string  // SHA256 of visual/text content
	Type        string  // "header", "footer", "logo", "table_header", "boilerplate"
	Text        string  // Extracted text (if any)
	BoundingBox [4]int  // x, y, width, height
	Frequency   int     // How often seen
	Confidence  float64 // OCR confidence when learned (0.0-1.0)
}

// PDFDNA stores the "DNA" fingerprint database
type PDFDNA struct {
	Elements map[string]*PDFElement // hash -> element
	mu       sync.RWMutex
	Version  string // DNA database version
	Stats    DNAStats
}

// DNAStats tracks DNA database statistics
type DNAStats struct {
	TotalElements     int
	RecurringElements int
	TotalQueries      int64
	CacheHits         int64
	CacheMisses       int64
	TimeSavedMs       int64
}

// NewPDFDNA creates a new DNA database
func NewPDFDNA() *PDFDNA {
	return &PDFDNA{
		Elements: make(map[string]*PDFElement),
		Version:  "1.0.0",
		Stats:    DNAStats{},
	}
}

// HashRegion computes SHA256 of image region (128-bit hash for speed)
func HashRegion(pixels []byte) string {
	h := sha256.Sum256(pixels)
	return hex.EncodeToString(h[:16]) // 128-bit hash (32 hex chars)
}

// HashRegionFuzzy computes perceptual hash for fuzzy matching
// Uses average hash (aHash) algorithm - resistant to minor changes
func HashRegionFuzzy(pixels []byte, width, height int) string {
	if len(pixels) == 0 || width == 0 || height == 0 {
		return ""
	}

	// Downsample to 8x8 grid
	gridSize := 8
	cellWidth := width / gridSize
	cellHeight := height / gridSize

	var averages [64]byte
	for gy := 0; gy < gridSize; gy++ {
		for gx := 0; gx < gridSize; gx++ {
			// Compute average brightness in cell
			var sum uint32
			count := 0
			for y := gy * cellHeight; y < (gy+1)*cellHeight && y < height; y++ {
				for x := gx * cellWidth; x < (gx+1)*cellWidth && x < width; x++ {
					idx := (y*width + x) * 3
					if idx+2 < len(pixels) {
						// Grayscale: 0.299*R + 0.587*G + 0.114*B
						r, g, b := uint32(pixels[idx]), uint32(pixels[idx+1]), uint32(pixels[idx+2])
						brightness := (299*r + 587*g + 114*b) / 1000
						sum += brightness
						count++
					}
				}
			}
			if count > 0 {
				averages[gy*gridSize+gx] = byte(sum / uint32(count))
			}
		}
	}

	// Compute overall average
	var totalAvg uint32
	for _, v := range averages {
		totalAvg += uint32(v)
	}
	avgBrightness := byte(totalAvg / 64)

	// Create hash: 1 if above average, 0 if below
	var hashBits uint64
	for i, v := range averages {
		if v > avgBrightness {
			hashBits |= (1 << uint(i))
		}
	}

	// Return as hex string
	return hex.EncodeToString([]byte{
		byte(hashBits >> 56), byte(hashBits >> 48), byte(hashBits >> 40), byte(hashBits >> 32),
		byte(hashBits >> 24), byte(hashBits >> 16), byte(hashBits >> 8), byte(hashBits),
	})
}

// RegisterElement adds or updates an element in the DNA database
func (d *PDFDNA) RegisterElement(hash, elemType, text string, bbox [4]int, confidence float64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if elem, exists := d.Elements[hash]; exists {
		elem.Frequency++
		// Update text if new confidence is higher
		if confidence > elem.Confidence {
			elem.Text = text
			elem.Confidence = confidence
		}
	} else {
		d.Elements[hash] = &PDFElement{
			Hash:        hash,
			Type:        elemType,
			Text:        text,
			BoundingBox: bbox,
			Frequency:   1,
			Confidence:  confidence,
		}
		d.Stats.TotalElements++
	}
}

// LookupElement checks if element exists in DNA
func (d *PDFDNA) LookupElement(hash string) (*PDFElement, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	d.Stats.TotalQueries++

	elem, exists := d.Elements[hash]
	if exists {
		d.Stats.CacheHits++
	} else {
		d.Stats.CacheMisses++
	}

	return elem, exists
}

// IsRecurring checks if element is seen frequently (threshold: 3+)
func (d *PDFDNA) IsRecurring(hash string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if elem, exists := d.Elements[hash]; exists {
		return elem.Frequency >= 3
	}
	return false
}

// GetStats returns DNA database statistics
func (d *PDFDNA) GetStats() (total, recurring int, hitRate float64) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	total = d.Stats.TotalElements
	for _, elem := range d.Elements {
		if elem.Frequency >= 3 {
			recurring++
		}
	}

	if d.Stats.TotalQueries > 0 {
		hitRate = float64(d.Stats.CacheHits) / float64(d.Stats.TotalQueries)
	}

	return total, recurring, hitRate
}

// RecordTimeSaved adds to cumulative time saved metric
func (d *PDFDNA) RecordTimeSaved(ms int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Stats.TimeSavedMs += ms
}

// GetTimeSaved returns total time saved in milliseconds
func (d *PDFDNA) GetTimeSaved() int64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Stats.TimeSavedMs
}

// Save persists DNA database to disk
func (d *PDFDNA) Save(filepath string) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath, data, 0644)
}

// Load restores DNA database from disk
func (d *PDFDNA) Load(filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	return json.Unmarshal(data, d)
}

// Prune removes low-frequency elements to keep database lean
func (d *PDFDNA) Prune(minFrequency int) int {
	d.mu.Lock()
	defer d.mu.Unlock()

	pruned := 0
	for hash, elem := range d.Elements {
		if elem.Frequency < minFrequency {
			delete(d.Elements, hash)
			pruned++
		}
	}

	d.Stats.TotalElements -= pruned
	return pruned
}

// Merge combines another DNA database into this one
func (d *PDFDNA) Merge(other *PDFDNA) int {
	d.mu.Lock()
	defer d.mu.Unlock()

	other.mu.RLock()
	defer other.mu.RUnlock()

	merged := 0
	for hash, elem := range other.Elements {
		if existing, exists := d.Elements[hash]; exists {
			// Merge frequencies
			existing.Frequency += elem.Frequency
			// Keep higher confidence text
			if elem.Confidence > existing.Confidence {
				existing.Text = elem.Text
				existing.Confidence = elem.Confidence
			}
		} else {
			// Add new element
			d.Elements[hash] = &PDFElement{
				Hash:        elem.Hash,
				Type:        elem.Type,
				Text:        elem.Text,
				BoundingBox: elem.BoundingBox,
				Frequency:   elem.Frequency,
				Confidence:  elem.Confidence,
			}
			merged++
			d.Stats.TotalElements++
		}
	}

	return merged
}

// GetRecurringElements returns all elements with frequency >= threshold
func (d *PDFDNA) GetRecurringElements(threshold int) []*PDFElement {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var recurring []*PDFElement
	for _, elem := range d.Elements {
		if elem.Frequency >= threshold {
			recurring = append(recurring, elem)
		}
	}

	return recurring
}

// Clear resets the DNA database
func (d *PDFDNA) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.Elements = make(map[string]*PDFElement)
	d.Stats = DNAStats{}
}
