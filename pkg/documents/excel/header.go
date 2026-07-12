package excel

// Header-column matching (Wave 3 B.3): the one implementation of "find the
// column named any of X" that used to exist three times at the root —
// import_2026_data.go findColumnIndex, tally_importer.go findColumn (plus
// four hand-rolled colMap loops), and etl_service.go's inline map.
//
// Two lookup shapes exist ON PURPOSE — they tie-break differently and both
// semantics have callers:
//
//   - FindInHeader scans COLUMNS in order and returns the first column whose
//     name matches ANY variant (column priority — a header [B, A] queried
//     with (A, B) yields B's index).
//   - HeaderIndex.Find tries VARIANTS in order against the index (variant
//     priority — the same query yields A's index).
//
// Don't "unify" them; converting a caller between the two silently changes
// which column wins when a sheet carries more than one candidate.

import "strings"

// Normalize is the shared header normalization: lowercase, trimmed.
func Normalize(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// HeaderIndex maps normalized column names to their positions. Duplicate
// names keep the LAST occurrence (map-overwrite), matching the historical
// colMap loops it replaces.
type HeaderIndex map[string]int

// IndexHeader builds a HeaderIndex from a header row.
func IndexHeader(header []string) HeaderIndex {
	idx := make(HeaderIndex, len(header))
	for i, name := range header {
		idx[Normalize(name)] = i
	}
	return idx
}

// Find returns the position of the first VARIANT present in the index, or -1.
func (h HeaderIndex) Find(variants ...string) int {
	for _, v := range variants {
		if i, ok := h[Normalize(v)]; ok {
			return i
		}
	}
	return -1
}

// Cell returns the trimmed value of the named column in row, or "" when the
// column is absent or the row is short — the etl getCell contract.
func (h HeaderIndex) Cell(row []string, name string) string {
	if i, ok := h[Normalize(name)]; ok && i < len(row) {
		return strings.TrimSpace(row[i])
	}
	return ""
}

// FindInHeader returns the index of the first COLUMN whose normalized name
// equals any of the normalized names, or -1 (column-priority scan).
func FindInHeader(header []string, names ...string) int {
	for i, col := range header {
		colNorm := Normalize(col)
		for _, name := range names {
			if colNorm == Normalize(name) {
				return i
			}
		}
	}
	return -1
}
