// Batch OCR processor for Acme Instrumentation offer folders.
// σ: Batch-Offers-OCR | ρ: pkg/ocr | γ: Production | κ: O(√n×log₂n)
package ocr

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
)

// ========================================================================
// TYPES
// ========================================================================

// BatchOfferRequest represents a batch offer processing request
type BatchOfferRequest struct {
	// Source folder (e.g., "C:\...\Offers No 1-50 (2025)")
	OffersFolder string

	// Optional filters
	OfferNumberFilter string // Process only specific offer (e.g., "01")

	// Processing options
	MaxConcurrency int  // 0 = auto (Williams formula)
	EnableGPU      bool // Use GPU preprocessing
	StopOnError    bool // Stop entire batch on first error (default: false, continue)

	// Database persistence
	DB *gorm.DB // GORM database for storing results

	// Progress reporting
	ProgressChan chan<- BatchOfferProgress
}

// BatchOfferProgress represents processing progress
type BatchOfferProgress struct {
	CurrentFile    string  `json:"current_file"`
	FilesProcessed int     `json:"files_processed"`
	TotalFiles     int     `json:"total_files"`
	Percentage     float64 `json:"percentage"`
	OfferNumber    string  `json:"offer_number"`
	CustomerName   string  `json:"customer_name"`
	Stage          string  `json:"stage"` // "RFQ", "OFFER", "EXECUTION"
	Error          error   `json:"error,omitempty"`
}

// BatchOfferResult represents the final batch processing result
type BatchOfferResult struct {
	TotalOffers       int           `json:"total_offers"`
	TotalFiles        int           `json:"total_files"`
	ProcessedFiles    int           `json:"processed_files"`
	SkippedFiles      int           `json:"skipped_files"`
	FailedFiles       int           `json:"failed_files"`
	TotalTime         time.Duration `json:"total_time"`
	AverageConfidence float64       `json:"average_confidence"`

	// Per-offer results
	OfferResults map[string]*OfferProcessResult `json:"offer_results"`

	// Aggregate stats
	DocumentsByType map[string]int `json:"documents_by_type"`
	TotalCostUSD    float64        `json:"total_cost_usd"`
	GPUUsagePercent float64        `json:"gpu_usage_percent"`
}

// OfferProcessResult represents results for one offer
type OfferProcessResult struct {
	OfferNumber       string            `json:"offer_number"`
	CustomerName      string            `json:"customer_name"`
	FilesProcessed    int               `json:"files_processed"`
	FilesSkipped      int               `json:"files_skipped"`
	FilesFailed       int               `json:"files_failed"`
	Documents         []*DocumentResult `json:"documents"`
	ExtractedEntities map[string]any    `json:"extracted_entities"`
}

// DocumentResult represents a single processed document
type DocumentResult struct {
	FileName         string  `json:"file_name"`
	FilePath         string  `json:"file_path"`
	Stage            string  `json:"stage"` // RFQ, OFFER, EXECUTION
	DocumentType     string  `json:"document_type"`
	ExtractedText    string  `json:"extracted_text"`
	Confidence       float64 `json:"confidence"`
	ProcessingTimeMS int64   `json:"processing_time_ms"`
	Engine           string  `json:"engine"`
	TierUsed         string  `json:"tier_used"`
	Cost             float64 `json:"cost"`
	GPUUsed          bool    `json:"gpu_used"`
	Error            error   `json:"error,omitempty"`
}

// OfferMetadata represents parsed metadata from folder name
type OfferMetadata struct {
	OfferNumber  string // e.g., "01"
	CustomerName string // e.g., "NGA-FIT(FEED)"
	FullName     string // e.g., "01 NGA-FIT(FEED)"
}

// fileTask represents a single file to process in batch
type fileTask struct {
	offerMeta *OfferMetadata
	filePath  string
	stage     string // RFQ, OFFER, EXECUTION
}

// ========================================================================
// BATCH PROCESSOR
// ========================================================================

// ProcessOffersBatch processes an entire offers folder
func (e *ACEEngine) ProcessOffersBatch(ctx context.Context, req *BatchOfferRequest) (*BatchOfferResult, error) {
	startTime := time.Now()

	e.logger.Info("Starting batch offer processing", map[string]any{
		"offers_folder": req.OffersFolder,
		"filter":        req.OfferNumberFilter,
	})

	// Validate request
	if req.OffersFolder == "" {
		return nil, fmt.Errorf("offers_folder is required")
	}
	if _, err := os.Stat(req.OffersFolder); os.IsNotExist(err) {
		return nil, fmt.Errorf("offers_folder does not exist: %s", req.OffersFolder)
	}

	// Scan offer folders
	offerFolders, err := e.scanOfferFolders(req.OffersFolder, req.OfferNumberFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to scan offer folders: %w", err)
	}

	if len(offerFolders) == 0 {
		return &BatchOfferResult{}, nil
	}

	e.logger.Info("Found offer folders", map[string]any{
		"count": len(offerFolders),
	})

	// Collect all files to process
	var fileTasks []fileTask
	for _, meta := range offerFolders {
		offerPath := filepath.Join(req.OffersFolder, meta.FullName)

		// Walk subdirectories (RFQ, OFFER, EXECUTION)
		stages := []string{"RFQ", "OFFER", "EXECUTION"}
		for _, stage := range stages {
			stagePath := filepath.Join(offerPath, stage)
			if _, err := os.Stat(stagePath); err == nil {
				// Recursively walk stage folder
				err := filepath.Walk(stagePath, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return nil // Skip errors
					}
					if !info.IsDir() && e.isProcessableFile(path) {
						fileTasks = append(fileTasks, fileTask{
							offerMeta: meta,
							filePath:  path,
							stage:     stage,
						})
					}
					return nil
				})
				if err != nil {
					e.logger.Warn("Failed to walk stage folder", map[string]any{
						"stage": stage,
						"error": err.Error(),
					})
				}
			}
		}
	}

	totalFiles := len(fileTasks)
	if totalFiles == 0 {
		return &BatchOfferResult{
			TotalOffers: len(offerFolders),
			TotalTime:   time.Since(startTime),
		}, nil
	}

	e.logger.Info("Files to process", map[string]any{
		"total_files": totalFiles,
	})

	// Calculate optimal workers using Williams formula: O(√n × log₂n)
	optimalWorkers := int(math.Sqrt(float64(totalFiles)) * math.Log2(float64(totalFiles)))
	if optimalWorkers < 1 {
		optimalWorkers = 1
	}
	if optimalWorkers > e.config.MaxWorkers {
		optimalWorkers = e.config.MaxWorkers
	}
	if req.MaxConcurrency > 0 && req.MaxConcurrency < optimalWorkers {
		optimalWorkers = req.MaxConcurrency
	}

	e.logger.Info("Worker pool configuration", map[string]any{
		"workers": optimalWorkers,
		"formula": "Williams O(√n×log₂n)",
	})

	// Create worker pool
	type taskResult struct {
		task   fileTask
		result *DocumentResult
	}

	results := make(chan taskResult, totalFiles)
	sem := make(chan struct{}, optimalWorkers)
	var wg sync.WaitGroup
	var completed int64
	var totalConfidence float64
	var confidenceMu sync.Mutex
	var gpuUsageCount int64

	// Process files concurrently
	for i, task := range fileTasks {
		wg.Add(1)
		go func(idx int, t fileTask) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			// Check context cancellation
			select {
			case <-ctx.Done():
				results <- taskResult{
					task: t,
					result: &DocumentResult{
						FileName: filepath.Base(t.filePath),
						FilePath: t.filePath,
						Stage:    t.stage,
						Error:    ctx.Err(),
					},
				}
				return
			default:
			}

			// Process document
			docResult := e.processOfferDocument(ctx, &t, req.EnableGPU)
			results <- taskResult{task: t, result: docResult}

			// Track GPU usage
			if docResult.GPUUsed {
				atomic.AddInt64(&gpuUsageCount, 1)
			}

			// Update average confidence
			if docResult.Error == nil {
				confidenceMu.Lock()
				totalConfidence += docResult.Confidence
				confidenceMu.Unlock()
			}

			// Update progress
			c := atomic.AddInt64(&completed, 1)
			if req.ProgressChan != nil {
				req.ProgressChan <- BatchOfferProgress{
					CurrentFile:    docResult.FileName,
					FilesProcessed: int(c),
					TotalFiles:     totalFiles,
					Percentage:     float64(c) / float64(totalFiles) * 100,
					OfferNumber:    t.offerMeta.OfferNumber,
					CustomerName:   t.offerMeta.CustomerName,
					Stage:          t.stage,
					Error:          docResult.Error,
				}
			}

			// Persist to database if provided
			if req.DB != nil && docResult.Error == nil {
				e.persistOfferDocument(req.DB, docResult, t.offerMeta)
			}
		}(i, task)
	}

	// Close results channel when all done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results by offer
	offerResults := make(map[string]*OfferProcessResult)
	documentsByType := make(map[string]int)
	var processedFiles, skippedFiles, failedFiles int
	var totalCostUSD float64

	for res := range results {
		offerNum := res.task.offerMeta.OfferNumber

		// Initialize offer result if needed
		if _, exists := offerResults[offerNum]; !exists {
			offerResults[offerNum] = &OfferProcessResult{
				OfferNumber:       res.task.offerMeta.OfferNumber,
				CustomerName:      res.task.offerMeta.CustomerName,
				Documents:         []*DocumentResult{},
				ExtractedEntities: make(map[string]any),
			}
		}

		offerRes := offerResults[offerNum]
		offerRes.Documents = append(offerRes.Documents, res.result)

		if res.result.Error != nil {
			failedFiles++
			offerRes.FilesFailed++
		} else {
			processedFiles++
			offerRes.FilesProcessed++
			totalCostUSD += res.result.Cost

			// Track document types
			documentsByType[res.result.DocumentType]++
		}
	}

	// Calculate average confidence
	avgConfidence := 0.0
	if processedFiles > 0 {
		avgConfidence = totalConfidence / float64(processedFiles)
	}

	// Calculate GPU usage percentage
	gpuUsagePercent := 0.0
	if totalFiles > 0 {
		gpuUsagePercent = float64(gpuUsageCount) / float64(totalFiles) * 100
	}

	result := &BatchOfferResult{
		TotalOffers:       len(offerFolders),
		TotalFiles:        totalFiles,
		ProcessedFiles:    processedFiles,
		SkippedFiles:      skippedFiles,
		FailedFiles:       failedFiles,
		TotalTime:         time.Since(startTime),
		AverageConfidence: avgConfidence,
		OfferResults:      offerResults,
		DocumentsByType:   documentsByType,
		TotalCostUSD:      totalCostUSD,
		GPUUsagePercent:   gpuUsagePercent,
	}

	e.logger.Info("Batch offer processing complete", map[string]any{
		"total_offers":      result.TotalOffers,
		"total_files":       result.TotalFiles,
		"processed":         result.ProcessedFiles,
		"failed":            result.FailedFiles,
		"avg_confidence":    result.AverageConfidence,
		"total_time":        result.TotalTime.String(),
		"gpu_usage_percent": result.GPUUsagePercent,
	})

	return result, nil
}

// ========================================================================
// HELPER METHODS
// ========================================================================

// scanOfferFolders scans the offers directory and extracts metadata
func (e *ACEEngine) scanOfferFolders(offersFolder string, filter string) ([]*OfferMetadata, error) {
	entries, err := os.ReadDir(offersFolder)
	if err != nil {
		return nil, err
	}

	// Pattern: "NN CUSTOMER_NAME" (e.g., "01 NGA-FIT(FEED)")
	offerPattern := regexp.MustCompile(`^(\d+)\s+(.+)$`)

	var offers []*OfferMetadata
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		matches := offerPattern.FindStringSubmatch(name)
		if matches == nil {
			// Not a valid offer folder
			continue
		}

		offerNum := matches[1]
		customerName := matches[2]

		// Apply filter if specified
		if filter != "" && offerNum != filter {
			continue
		}

		offers = append(offers, &OfferMetadata{
			OfferNumber:  offerNum,
			CustomerName: customerName,
			FullName:     name,
		})
	}

	return offers, nil
}

// processOfferDocument processes a single document in an offer
func (e *ACEEngine) processOfferDocument(ctx context.Context, task *fileTask, enableGPU bool) *DocumentResult {

	// Open file
	file, err := os.Open(task.filePath)
	if err != nil {
		return &DocumentResult{
			FileName: filepath.Base(task.filePath),
			FilePath: task.filePath,
			Stage:    task.stage,
			Error:    fmt.Errorf("failed to open file: %w", err),
		}
	}
	defer file.Close()

	// Create process request
	procReq := &ProcessRequest{
		Source:              file,
		SourceType:          SourceReader,
		Context:             ctx,
		EnableGPU:           enableGPU,
		EnablePreprocessing: true,
		DocumentType:        e.detectDocumentTypeFromStage(task.stage),
		Language:            LangEnglish, // Acme Instrumentation is primarily English
	}

	// Process with OCR engine
	response, err := e.Process(ctx, procReq)
	if err != nil {
		return &DocumentResult{
			FileName: filepath.Base(task.filePath),
			FilePath: task.filePath,
			Stage:    task.stage,
			Error:    err,
		}
	}

	// Convert to DocumentResult
	return &DocumentResult{
		FileName:         filepath.Base(task.filePath),
		FilePath:         task.filePath,
		Stage:            task.stage,
		DocumentType:     string(response.DocumentType),
		ExtractedText:    response.Text,
		Confidence:       response.Confidence,
		ProcessingTimeMS: response.ProcessingTime.Milliseconds(),
		Engine:           response.Tier.String(), // Use tier as engine indicator
		TierUsed:         response.Tier.String(),
		Cost:             response.EstimatedCostUSD,
		GPUUsed:          response.GPUUsed,
		Error:            nil,
	}
}

// detectDocumentTypeFromStage infers document type from folder stage
func (e *ACEEngine) detectDocumentTypeFromStage(stage string) DocumentType {
	switch strings.ToUpper(stage) {
	case "RFQ":
		return DocTypeRFQ
	case "OFFER":
		return DocTypeQuote
	case "EXECUTION":
		// Could be PO, invoice, delivery note - use generic
		return DocTypeGeneric
	default:
		return DocTypeUnknown
	}
}

// persistOfferDocument persists an OCR document to the database
func (e *ACEEngine) persistOfferDocument(db *gorm.DB, doc *DocumentResult, meta *OfferMetadata) error {
	ocrDoc := OCRDocument{
		FileName:         doc.FileName,
		FilePath:         doc.FilePath,
		DocumentType:     doc.DocumentType,
		ExtractedText:    doc.ExtractedText,
		Confidence:       doc.Confidence,
		ProcessingTimeMS: doc.ProcessingTimeMS,
		Engine:           doc.Engine,
		TierUsed:         doc.TierUsed,
		Cost:             doc.Cost,
		DNACacheHit:      false, // Future: implement DNA cache
		TableDetected:    false, // Future: implement table detection
		GPUUsed:          doc.GPUUsed,
		ProcessedAt:      time.Now(),
	}

	result := db.Create(&ocrDoc)
	if result.Error != nil {
		e.logger.Error("Failed to persist OCR document", map[string]any{
			"file_name": doc.FileName,
			"error":     result.Error.Error(),
		})
		return result.Error
	}

	e.logger.Debug("Persisted OCR document", map[string]any{
		"id":            ocrDoc.ID,
		"file_name":     doc.FileName,
		"offer_number":  meta.OfferNumber,
		"customer_name": meta.CustomerName,
	})

	return nil
}

// OCRDocument database model (must match app.go definition)
type OCRDocument struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	FileName         string    `json:"file_name" gorm:"not null"`
	FilePath         string    `json:"file_path" gorm:"not null"`
	DocumentType     string    `json:"document_type" gorm:"index"`
	ExtractedText    string    `json:"extracted_text" gorm:"type:text"`
	Confidence       float64   `json:"confidence"`
	ProcessingTimeMS int64     `json:"processing_time_ms"`
	Engine           string    `json:"engine"`
	TierUsed         string    `json:"tier_used"`
	Cost             float64   `json:"cost"`
	DNACacheHit      bool      `json:"dna_cache_hit"`
	TableDetected    bool      `json:"table_detected"`
	GPUUsed          bool      `json:"gpu_used"`
	ProcessedAt      time.Time `json:"processed_at"`
	CreatedAt        time.Time `json:"created_at"`
}

// TableName overrides the table name
func (OCRDocument) TableName() string {
	return "ocr_documents"
}

// ========================================================================
// ENTITY EXTRACTION (FUTURE ENHANCEMENT)
// ========================================================================

// ExtractOfferEntities extracts structured entities from batch results
// This is a placeholder for future entity extraction logic
func ExtractOfferEntities(result *BatchOfferResult) map[string]any {
	entities := make(map[string]any)

	// TODO: Implement entity extraction:
	// - Offer numbers from documents
	// - Product codes and descriptions
	// - Prices and quantities
	// - Dates (RFQ received, quotation, validity)
	// - Customer contact information

	entities["total_offers"] = result.TotalOffers
	entities["total_documents"] = result.ProcessedFiles

	return entities
}
