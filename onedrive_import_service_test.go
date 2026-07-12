package main

import "testing"

func TestClassifyDiscoveredFile_RecognizesCostWorkbookVariants(t *testing.T) {
	cases := []string{
		"Acme Instrumentation Costing MasterFile.xlsx",
		"COST 01-26_rev-1.xlsx",
		"Cost-06-26.xlsx",
		"cost_42-26.xls",
	}

	for _, name := range cases {
		if got := classifyDiscoveredFile(name); got != "costing_sheet" {
			t.Fatalf("expected %q to classify as costing_sheet, got %q", name, got)
		}
	}
}

func TestIsOneDriveSectionDir_IncludesWorking(t *testing.T) {
	if !isOneDriveSectionDir("WORKING") {
		t.Fatal("expected WORKING to be treated as a OneDrive section directory")
	}
}
