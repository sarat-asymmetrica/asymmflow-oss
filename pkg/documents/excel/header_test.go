package excel

import "testing"

func TestHeaderIndex_FindAndCell(t *testing.T) {
	h := IndexHeader([]string{" Folder No ", "Customer Name", "AMOUNT", "Amount"}) // dup: last wins
	if got := h.Find("folder no"); got != 0 {
		t.Errorf("Find(folder no) = %d", got)
	}
	if got := h.Find("missing", "customer name"); got != 1 {
		t.Errorf("variant fallback = %d", got)
	}
	if got := h.Find("nope"); got != -1 {
		t.Errorf("missing = %d", got)
	}
	if got := h.Find("amount"); got != 3 {
		t.Errorf("duplicate header must keep LAST occurrence (map-overwrite), got %d", got)
	}
	row := []string{"F-001", "  Wasela Café  "}
	if got := h.Cell(row, "customer name"); got != "Wasela Café" {
		t.Errorf("Cell = %q", got)
	}
	if got := h.Cell(row, "amount"); got != "" { // row shorter than column index
		t.Errorf("short-row Cell = %q", got)
	}
}

// The two lookup shapes tie-break differently BY DESIGN — this test is the
// documentation that stops someone "unifying" them.
func TestColumnVsVariantPriority(t *testing.T) {
	header := []string{"B", "A"}
	if got := FindInHeader(header, "A", "B"); got != 0 {
		t.Errorf("FindInHeader is column-priority: want 0 (B), got %d", got)
	}
	if got := IndexHeader(header).Find("A", "B"); got != 1 {
		t.Errorf("HeaderIndex.Find is variant-priority: want 1 (A), got %d", got)
	}
}
