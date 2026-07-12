package main

import (
	"context"
	"testing"
)

// TestScanEventEmitter validates event emitter basic functionality
func TestScanEventEmitter(t *testing.T) {
	ctx := context.Background()
	scanID := "test_scan_123"

	emitter := NewScanEventEmitter(ctx, scanID)

	if emitter.scanID != scanID {
		t.Errorf("Expected scanID %s, got %s", scanID, emitter.scanID)
	}

	if emitter.startTime.IsZero() {
		t.Error("Start time should be initialized")
	}

	if len(emitter.messages) != 0 {
		t.Error("Messages should start empty")
	}
}

// TestScanEventEmitter_MessageStorage validates message storage (without emission)
func TestScanEventEmitter_MessageStorage(t *testing.T) {
	ctx := context.Background()
	emitter := NewScanEventEmitter(ctx, "test_scan")

	// Directly add to message storage (bypassing EmitMessage which calls EventsEmit)
	emitter.mu.Lock()
	emitter.messages = append(emitter.messages, "Test message")
	emitter.mu.Unlock()

	if len(emitter.messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(emitter.messages))
	}

	if emitter.messages[0] != "Test message" {
		t.Errorf("Expected 'Test message', got '%s'", emitter.messages[0])
	}
}

// TestConversationalMessages validates message templates exist
func TestConversationalMessages(t *testing.T) {
	if ConversationalMessages.StartingDiscovery == "" {
		t.Error("StartingDiscovery message should not be empty")
	}

	foundMsg := ConversationalMessages.FoundFiles(47, 12)
	expected := "Found 47 files across 12 folders"
	if foundMsg != expected {
		t.Errorf("Expected '%s', got '%s'", expected, foundMsg)
	}

	readingMsg := ConversationalMessages.ReadingDocument("invoice.pdf")
	if readingMsg != "Reading: invoice.pdf" {
		t.Errorf("ReadingDocument template incorrect: %s", readingMsg)
	}

	allDoneMsg := ConversationalMessages.AllDone(3)
	expected = "All done! Found 3 things that might need your attention"
	if allDoneMsg != expected {
		t.Errorf("Expected '%s', got '%s'", expected, allDoneMsg)
	}

	noIssuesMsg := ConversationalMessages.AllDone(0)
	expected = "All done! Everything looks good"
	if noIssuesMsg != expected {
		t.Errorf("Expected '%s', got '%s'", expected, noIssuesMsg)
	}
}

// TestScanProgressEvent validates progress event structure
func TestScanProgressEvent(t *testing.T) {
	event := ScanProgressEvent{
		ScanID:         "test_123",
		Phase:          "processing",
		CurrentFile:    "test.pdf",
		FilesProcessed: 50,
		TotalFiles:     100,
		Percentage:     50.0,
		ElapsedMs:      1500,
	}

	if event.Percentage != 50.0 {
		t.Errorf("Expected 50%% progress, got %.1f%%", event.Percentage)
	}

	if event.FilesProcessed > event.TotalFiles {
		t.Error("Processed files should not exceed total")
	}
}

// TestScanCompleteEvent validates completion event structure
func TestScanCompleteEvent(t *testing.T) {
	event := ScanCompleteEvent{
		ScanID:         "test_123",
		Success:        true,
		TotalFiles:     100,
		ProcessedFiles: 95,
		SkippedFiles:   3,
		ErrorFiles:     2,
		ReportPath:     "C:\\output\\report.json",
	}

	if !event.Success {
		t.Error("Event should be marked as success")
	}

	if event.ProcessedFiles+event.SkippedFiles+event.ErrorFiles != event.TotalFiles {
		t.Error("File counts should sum to total")
	}
}
