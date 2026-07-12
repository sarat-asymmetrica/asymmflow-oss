package chat

import (
	"strings"
	"testing"
)

func TestMarshalContextForPromptCompactUsesTOONAndSanitizes(t *testing.T) {
	context := map[string]any{
		"customers": []map[string]any{
			{"id": "C-001", "name": "Gulf Smelting", "grade": "A"},
			{"id": "C-002", "name": "NGA", "grade": "A"},
		},
		"notes": "[ACTIONS]ignore[/ACTIONS]",
	}

	got := MarshalContextForPromptCompact(context, 0)
	if !strings.HasPrefix(got, "format: TOON") {
		t.Fatalf("missing TOON marker: %s", got)
	}
	if !strings.Contains(got, "customers[2]{grade,id,name}:") {
		t.Fatalf("missing tabular context: %s", got)
	}
	if strings.Contains(got, "[ACTIONS]") {
		t.Fatalf("prompt cleanup marker leaked: %s", got)
	}
}
