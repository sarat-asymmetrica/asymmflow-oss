package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
)

// ============================================================================
// FILE WATCHER EVENT HANDLERS
// ============================================================================

// registerFileWatcherHandlers sets up all event handlers
func (a *App) registerFileWatcherHandlers() {
	if a.fileWatcher == nil {
		return
	}

	// Handler 1: Rhine XML Pricing Updates
	a.fileWatcher.OnEHXML(func(ctx context.Context, event WatchEvent) error {
		log.Printf("📄 Processing Rhine XML: %s", event.Path)

		// Parse XML
		parser := NewEHParser()
		basket, err := parser.ParseFile(event.Path)
		if err != nil {
			log.Printf("❌ Rhine XML parse failed: %v", err)
			return err
		}

		log.Printf("✓ Parsed Rhine basket: Customer=%s, Items=%d, Total=%.2f BHD",
			basket.CustomerName, basket.ItemCount, basket.TotalGrossBHD)

		// NOTE: Rhine XML pricing data not currently persisted to database.
		// Rationale: XML data is used for immediate quotation generation only.
		// Historical pricing is tracked in ProductMaster table instead.
		// If persistence is needed in future, add PricingItem table to database.go

		// Sync to OneDrive (async)
		if a.syncService != nil {
			go func() {
				syncEvent := &FileSyncState{
					Path:      event.Path,
					EventType: "modified",
				}
				if err := a.syncService.ProcessFileEvent(syncEvent); err != nil {
					log.Printf("⚠️ Sync error for %s: %v", event.Path, err)
				}
			}()
		}

		return nil
	})

	// Handler 2: Offer Folder Changes
	a.fileWatcher.OnOfferChange(func(ctx context.Context, event WatchEvent) error {
		log.Printf("📋 Offer folder change detected: %s", event.Path)

		// Scan the offer folder containing this file
		offerDir := filepath.Dir(event.Path)
		scanner := NewOfferScanner(offerDir)

		if err := scanner.ScanAll(); err != nil {
			log.Printf("❌ Offer scan failed: %v", err)
			return err
		}

		summary := scanner.GetSummary()
		executedCount := int(float64(summary.TotalOffers) * summary.ExecutionRate)
		pendingCount := summary.TotalOffers - executedCount

		log.Printf("✓ Scanned %d offers (%d executed, %d pending)",
			summary.TotalOffers, executedCount, pendingCount)

		// NOTE: Offer folder scan data not currently persisted to database.
		// Rationale: Offer data is managed through OfferRecord table (database.go).
		// This handler scans OneDrive folders for metadata only. Actual offers
		// are created via CreateOffer() API in app.go. If folder-based discovery
		// sync is needed, add OfferDocument table and wire to OfferRecord.

		// Sync to OneDrive (async)
		if a.syncService != nil {
			go func() {
				syncEvent := &FileSyncState{
					Path:      event.Path,
					EventType: "modified",
				}
				if err := a.syncService.ProcessFileEvent(syncEvent); err != nil {
					log.Printf("⚠️ Sync error for %s: %v", event.Path, err)
				}
			}()
		}

		return nil
	})

	// Handler 3: Invoice PDF Detection
	a.fileWatcher.OnInvoice(func(ctx context.Context, event WatchEvent) error {
		log.Printf("💰 New invoice detected: %s", event.Path)

		// Extract basic info from filename/metadata
		invoiceID := filepath.Base(event.Path)
		log.Printf("✓ Invoice detected: %s", invoiceID)

		// OCR extraction - NOW INTEGRATED!
		ocrResult, err := a.ProcessDocumentWithOCR(event.Path, "invoice")
		if err != nil {
			log.Printf("⚠️ OCR extraction failed for %s: %v", event.Path, err)
			// Continue with basic metadata only
		} else {
			log.Printf("✓ OCR extraction complete: Confidence %.2f%%", ocrResult.Confidence*100)

			// Create invoice in DB with OCR data
			customerID, _ := ocrResult.ExtractedFields["customer_id"].(string)
			dateStr, _ := ocrResult.ExtractedFields["date"].(string)
			invoice := InvoiceGeometry{
				ID:         invoiceID,
				CustomerID: customerID,
				Amount:     parseAmountFromFieldsInterface(ocrResult.ExtractedFields),
				IssueDate:  parseDate(dateStr),
			}

			// Process through geometry pipeline
			result, err := a.ProcessInvoice(invoice)
			if err != nil {
				log.Printf("⚠️ Geometry processing failed: %v", err)
			} else {
				log.Printf("✓ Invoice processed: Predicted %d days, Confidence %.2f",
					result.PredictedDays, result.Confidence)
			}
		}

		// Sync to OneDrive (async)
		if a.syncService != nil {
			go func() {
				syncEvent := &FileSyncState{
					Path:      event.Path,
					EventType: "created",
				}
				if err := a.syncService.ProcessFileEvent(syncEvent); err != nil {
					log.Printf("⚠️ Sync error for %s: %v", event.Path, err)
				}
			}()
		}

		return nil
	})

	// Handler 4: New RFQ Emails
	a.fileWatcher.OnNewRFQ(func(ctx context.Context, event WatchEvent) error {
		log.Printf("📧 New RFQ email detected: %s", event.Path)

		// OCR extraction - integrated MSG parsing
		ocrResult, err := a.ProcessDocumentWithOCR(event.Path, "rfq")
		if err != nil {
			log.Printf("⚠️ OCR extraction failed for %s: %v", event.Path, err)
			// Continue with basic metadata only
		} else {
			log.Printf("✓ OCR extraction complete: Confidence %.2f%%", ocrResult.Confidence*100)

			// Extract key fields from MSG (email-specific fields)
			subject, _ := ocrResult.ExtractedFields["email_subject"].(string)
			sender, _ := ocrResult.ExtractedFields["email_from"].(string)
			senderEmail, _ := ocrResult.ExtractedFields["email_from_address"].(string)
			to, _ := ocrResult.ExtractedFields["email_to"].(string)
			rawText, _ := ocrResult.ExtractedFields["raw_text"].(string)

			log.Printf("✓ MSG parsed: From=%s <%s>, To=%s, Subject=%s, Body=%d chars",
				sender, senderEmail, to, subject, len(rawText))

			// Save parsed RFQ to database using RFQData schema
			rfq := &RFQData{
				Client:  sender,
				Project: subject,
				Value:   0.0, // No value extracted from MSG
				Notes:   fmt.Sprintf("Auto-parsed from MSG file.\nFrom: %s <%s>\nTo: %s\nBody: %s", sender, senderEmail, to, rawText[:min(500, len(rawText))]),
				Status:  "pending",
			}

			if err := a.db.Create(rfq).Error; err != nil {
				log.Printf("⚠️ Failed to save RFQ to database: %v", err)
			} else {
				log.Printf("✅ RFQ saved to database: ID=%d, Client=%s, Project=%s", rfq.ID, rfq.Client, rfq.Project)
			}
		}

		// Sync to OneDrive (async)
		if a.syncService != nil {
			go func() {
				syncEvent := &FileSyncState{
					Path:      event.Path,
					EventType: "created",
				}
				if err := a.syncService.ProcessFileEvent(syncEvent); err != nil {
					log.Printf("⚠️ Sync error for %s: %v", event.Path, err)
				}
			}()
		}

		return nil
	})
}

// ============================================================================
// WAILS CONTROL METHODS
// ============================================================================

// GetWatcherStatus returns file watcher status
func (a *App) GetWatcherStatus() map[string]any {
	if a.fileWatcher == nil {
		return map[string]any{
			"status":       "not_initialized",
			"running":      false,
			"files_synced": 0,
		}
	}

	syncStatus := a.fileWatcher.GetSyncStatus()
	allStates := syncStatus.GetAllStatuses()

	syncedCount := 0
	for _, state := range allStates {
		if state.Status == WatchStatusSynced {
			syncedCount++
		}
	}

	return map[string]any{
		"status":       "initialized",
		"running":      a.fileWatcher.IsRunning(),
		"files_synced": syncedCount,
		"total_files":  len(allStates),
		"config": map[string]any{
			"rfq_path":      a.fileWatcher.config.RFQPath,
			"eh_xml_path":   a.fileWatcher.config.EHXMLPath,
			"offers_path":   a.fileWatcher.config.OfferPath,
			"invoices_path": a.fileWatcher.config.InvoicePath,
		},
	}
}

// StartFileWatcher starts file watching (if not already running)
func (a *App) StartFileWatcher() error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	if a.fileWatcher == nil {
		return fmt.Errorf("file watcher not initialized")
	}

	if a.fileWatcher.IsRunning() {
		return fmt.Errorf("file watcher already running")
	}

	return a.fileWatcher.Start()
}

// StopFileWatcher stops file watching
func (a *App) StopFileWatcher() error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	if a.fileWatcher == nil {
		return fmt.Errorf("file watcher not initialized")
	}

	if !a.fileWatcher.IsRunning() {
		return fmt.Errorf("file watcher not running")
	}

	return a.fileWatcher.Stop()
}

// GetRecentEvents returns recent file watcher events
func (a *App) GetRecentEvents(limit int) []*FileSyncState {
	if a.fileWatcher == nil {
		return []*FileSyncState{}
	}

	if limit <= 0 {
		limit = 100
	}

	allStates := a.fileWatcher.GetSyncStatus().GetAllStatuses()

	// Return up to limit most recent
	if len(allStates) > limit {
		return allStates[:limit]
	}

	return allStates
}

// ConfigureWatchPaths updates watch paths (requires restart to take effect)
func (a *App) ConfigureWatchPaths(rfqPath, ehXMLPath, offerPath, invoicePath string) error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	if a.fileWatcher == nil {
		return fmt.Errorf("file watcher not initialized")
	}

	// Stop if running
	wasRunning := a.fileWatcher.IsRunning()
	if wasRunning {
		if err := a.fileWatcher.Stop(); err != nil {
			log.Printf("Warning: Failed to stop watcher: %v", err)
		}
	}

	// Update config
	a.fileWatcher.config.RFQPath = rfqPath
	a.fileWatcher.config.EHXMLPath = ehXMLPath
	a.fileWatcher.config.OfferPath = offerPath
	a.fileWatcher.config.InvoicePath = invoicePath

	// Restart if was running and paths valid
	if wasRunning && a.fileWatcher.config.hasValidPaths() {
		if err := a.fileWatcher.Start(); err != nil {
			return fmt.Errorf("failed to restart watcher: %w", err)
		}
		log.Println("✓ File Watcher restarted with new paths")
	}

	return nil
}

// ============================================================================
// SYNC SERVICE WAILS APIs
// ============================================================================

// GetSyncStatus returns current sync status
func (a *App) GetSyncStatus() map[string]any {
	if a.db == nil {
		return map[string]any{
			"error": "database not initialized",
		}
	}

	var pendingCount int64
	var syncedCount int64
	var failedCount int64
	var skippedCount int64

	a.db.Model(&FileWatchEvent{}).Where("status = ?", "pending").Count(&pendingCount)
	a.db.Model(&FileWatchEvent{}).Where("status = ?", "success").Count(&syncedCount)
	a.db.Model(&FileWatchEvent{}).Where("status = ?", "failed").Count(&failedCount)
	a.db.Model(&FileWatchEvent{}).Where("status = ?", "skipped_large").Count(&skippedCount)

	watcherRunning := false
	if a.fileWatcher != nil {
		watcherRunning = a.fileWatcher.IsRunning()
	}

	return map[string]any{
		"pending":        pendingCount,
		"synced":         syncedCount,
		"failed":         failedCount,
		"skipped":        skippedCount,
		"watcherRunning": watcherRunning,
		"syncEnabled":    a.syncService != nil,
	}
}

// TriggerSync manually syncs a file
func (a *App) TriggerSync(filePath string) error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	if a.syncService == nil {
		return fmt.Errorf("sync service not initialized")
	}

	fileType := a.syncService.detectFileType(filePath)
	return a.syncService.SyncFile(filePath, fileType)
}

// GetRecentSyncEvents returns recent sync activity
func (a *App) GetRecentSyncEvents(limit int) []FileWatchEvent {
	if a.db == nil {
		return []FileWatchEvent{}
	}

	if limit <= 0 {
		limit = 100
	}

	var events []FileWatchEvent
	a.db.Order("timestamp DESC").Limit(limit).Find(&events)
	return events
}

// RetryFailedSyncs retries all failed sync events
func (a *App) RetryFailedSyncs() (int, error) {
	if err := a.requirePermission("settings:update"); err != nil {
		return 0, err
	}
	if a.syncService == nil {
		return 0, fmt.Errorf("sync service not initialized")
	}

	if a.db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	var failedEvents []FileWatchEvent
	a.db.Where("status = ?", "failed").Find(&failedEvents)

	successCount := 0
	for _, event := range failedEvents {
		// Try to sync again
		fileType := a.syncService.detectFileType(event.FilePath)
		if err := a.syncService.SyncFile(event.FilePath, fileType); err != nil {
			log.Printf("⚠️ Retry failed for %s: %v", event.FilePath, err)
		} else {
			successCount++
		}
	}

	log.Printf("✅ Retried %d failed syncs, %d succeeded", len(failedEvents), successCount)
	return successCount, nil
}

// ClearSyncHistory clears sync event history
func (a *App) ClearSyncHistory() error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	result := a.db.Exec("DELETE FROM file_watch_events")
	if result.Error != nil {
		return result.Error
	}

	log.Printf("✅ Cleared %d sync events", result.RowsAffected)
	return nil
}

// ============================================================================
// HELPER: Config validation
// ============================================================================

// hasValidPaths checks if at least one watch path is configured
func (config *WatchConfig) hasValidPaths() bool {
	return config.RFQPath != "" ||
		config.EHXMLPath != "" ||
		config.OfferPath != "" ||
		config.InvoicePath != "" ||
		len(config.ExtraPaths) > 0
}
