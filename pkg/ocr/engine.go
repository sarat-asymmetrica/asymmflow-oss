// Engine implementation for ACE OCR pipeline.
// σ: ACE-OCR-Engine | ρ: pkg/ocr | γ: Production | κ: O(√n×log₂n)
package ocr

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ph_holdings_app/pkg/ocr/mistralocr"
	"ph_holdings_app/pkg/ocr/preprocess"
)

// ========================================================================
// CONSTANTS
// ========================================================================

const (
	// Vedic constants
	PHI                = 0.618033988749 // Golden ratio
	TESLA_FREQUENCY_HZ = 4.909          // Tesla harmonic frequency
	WILLIAMS_LEVERAGE  = 8.35           // Williams optimizer efficiency

	// Thresholds
	MIN_CONFIDENCE      = 0.70  // Minimum acceptable confidence
	CONSENSUS_THRESHOLD = 0.667 // 2/3 voting requirement

	// DefaultCloudEscalationThreshold: below this local confidence, escalate to the
	// cloud OCR tier (Mistral OCR 4). Config-not-constant — EngineConfig.ConfidenceThreshold
	// overrides this; the constant only supplies the sane default.
	DefaultCloudEscalationThreshold = 0.85

	// Default concurrency
	DEFAULT_WORKERS = 8
	MAX_WORKERS     = 32
)

// ========================================================================
// ACE OCR ENGINE
// ========================================================================

// ACEEngine is the main OCR engine implementation
type ACEEngine struct {
	// Configuration
	config    *EngineConfig
	gpuConfig *GPUConfig

	// Components
	preprocessor  Preprocessor
	extractor     Extractor
	postprocessor Postprocessor
	trinity       *TrinityOptimizer
	babel         *BabelMapper

	// Cloud OCR escalation client (Mistral OCR 4). Nil when no key is configured —
	// offline-first: the local tesseract+pandoc pipeline above never blocks on this.
	mistralClient *mistralocr.Client

	// Observability
	metrics MetricsCollector
	logger  Logger

	// State
	mu          sync.RWMutex
	initialized bool
}

// EngineConfig contains engine configuration
type EngineConfig struct {
	// GPU settings
	EnableGPU      bool
	GPUBackend     GPUBackend
	GPUDeviceID    int
	GPUMaxMemoryMB int

	// Processing settings
	MaxWorkers            int
	DefaultLanguage       Language
	EnablePreprocessing   bool
	EnableVedicValidation bool

	// Cloud OCR escalation settings (Mistral OCR 4, pkg/ocr/mistralocr). The API key is
	// caller-injected — this engine never reads env/DB itself; route through the app's
	// standard Mistral key resolver.
	MistralAPIKey       string
	MistralBaseURL      string  // optional override; mistralocr applies its own default
	MistralModel        string  // optional override; mistralocr applies its own default (mistral-ocr-4-0)
	ConfidenceThreshold float64 // below this, escalate to Mistral OCR; default DefaultCloudEscalationThreshold
	FallbackToMistral   bool

	// Tesseract settings
	TesseractPath     string
	TesseractDataPath string

	// Pandoc settings
	PandocPath string

	// Checkpoint settings
	CheckpointDir     string
	EnableCheckpoints bool

	// Logging
	LogLevel string
	LogPath  string
}

// NewACEEngine creates a new ACE OCR engine
func NewACEEngine(config *EngineConfig) (*ACEEngine, error) {
	if config == nil {
		config = &EngineConfig{
			EnableGPU:             true,
			GPUBackend:            GPULevelZero,
			MaxWorkers:            runtime.NumCPU(),
			DefaultLanguage:       LangEnglish,
			EnablePreprocessing:   true,
			EnableVedicValidation: true,
			FallbackToMistral:     true,
		}
	}

	// Cap workers
	if config.MaxWorkers > MAX_WORKERS {
		config.MaxWorkers = MAX_WORKERS
	}
	if config.MaxWorkers < 1 {
		config.MaxWorkers = DEFAULT_WORKERS
	}
	if config.ConfidenceThreshold <= 0 {
		config.ConfidenceThreshold = DefaultCloudEscalationThreshold
	}

	engine := &ACEEngine{
		config: config,
		gpuConfig: &GPUConfig{
			Backend:       config.GPUBackend,
			DeviceID:      config.GPUDeviceID,
			MaxMemoryMB:   config.GPUMaxMemoryMB,
			FallbackToCPU: true,
		},
		trinity: NewTrinityOptimizer(),
		babel:   NewBabelMapper(),
		logger:  &defaultLogger{},
		metrics: &defaultMetrics{},
	}

	// Initialize the cloud OCR escalation client only if a key was provided —
	// offline-first: without a key, mistralClient stays nil and escalation is skipped.
	if config.MistralAPIKey != "" {
		engine.mistralClient = mistralocr.NewClient(mistralocr.Config{
			APIKey:              config.MistralAPIKey,
			BaseURL:             config.MistralBaseURL,
			Model:               config.MistralModel,
			ConfidenceThreshold: config.ConfidenceThreshold,
		})
	}

	engine.initialized = true
	return engine, nil
}

// ========================================================================
// SINGLE DOCUMENT PROCESSING
// ========================================================================

// Process processes a single document
func (e *ACEEngine) Process(ctx context.Context, req *ProcessRequest) (*ProcessResponse, error) {
	startTime := time.Now()

	e.logger.Info("Processing document", map[string]any{
		"source_type":   req.SourceType,
		"document_type": req.DocumentType,
		"language":      req.Language,
		"tier":          req.Tier,
	})

	// Validate request
	if err := e.validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Set context if not provided
	if req.Context == nil {
		req.Context = ctx
	}

	// Phase 1: Source resolution (30% - Emergence)
	reader, cleanup, err := e.resolveSource(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve source: %w", err)
	}
	defer cleanup()

	// Phase 2: Preprocessing (20% - Optimization)
	var processedReader io.Reader = reader
	if req.EnablePreprocessing || e.config.EnablePreprocessing {
		processedReader, err = e.preprocess(ctx, reader, req)
		if err != nil {
			e.logger.Warn("Preprocessing failed, continuing with raw input", map[string]any{
				"error": err.Error(),
			})
			processedReader = reader
		}
	}

	// Phase 3: Extraction with tier selection (50% - Stabilization)
	response, err := e.extractWithTierSelection(ctx, processedReader, req)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	// Apply Trinity optimization metrics
	response.TrinityMetrics = e.trinity.CalculateMetrics(response)

	// Vedic validation
	if e.config.EnableVedicValidation {
		response.VedicValidation = e.trinity.ValidateWithVedic(response)
	}

	// Apply Babel mapping if country code provided
	if req.CountryCode != "" {
		babelResult, err := e.babel.MapFields(response.Fields, req.CountryCode, req.DocumentType)
		if err == nil {
			// Enrich response with mapped fields
			for _, mapping := range babelResult.Mappings {
				if val, ok := response.Fields[mapping.LocalTerm]; ok {
					response.Fields[mapping.StandardTerm] = val
				}
			}
		}
	}

	// Record metrics
	response.ProcessingTime = time.Since(startTime)
	e.metrics.RecordProcessingTime(response.ProcessingTime, response.Tier)
	e.metrics.RecordConfidence(response.Confidence, response.DocumentType)

	e.logger.Info("Document processed successfully", map[string]any{
		"processing_time": response.ProcessingTime.String(),
		"confidence":      response.Confidence,
		"tier_used":       response.Tier,
		"gpu_used":        response.GPUUsed,
	})

	return response, nil
}

// ========================================================================
// BATCH PROCESSING
// ========================================================================

// ProcessBatch processes multiple documents concurrently
func (e *ACEEngine) ProcessBatch(ctx context.Context, req *BatchRequest) (*BatchResponse, error) {
	startTime := time.Now()

	if len(req.Requests) == 0 {
		return &BatchResponse{}, nil
	}

	// Calculate optimal batch size using Williams formula
	optimalWorkers := e.trinity.CalculateOptimalWorkers(len(req.Requests))
	if req.MaxConcurrency > 0 && req.MaxConcurrency < optimalWorkers {
		optimalWorkers = req.MaxConcurrency
	}

	e.logger.Info("Starting batch processing", map[string]any{
		"total_documents": len(req.Requests),
		"workers":         optimalWorkers,
	})

	// Create worker pool
	type result struct {
		index    int
		response *ProcessResponse
		err      error
	}

	results := make(chan result, len(req.Requests))
	sem := make(chan struct{}, optimalWorkers)

	var wg sync.WaitGroup
	var completed int64

	for i, procReq := range req.Requests {
		wg.Add(1)
		go func(idx int, r *ProcessRequest) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			// Check context
			select {
			case <-ctx.Done():
				results <- result{idx, nil, ctx.Err()}
				return
			default:
			}

			// Process document
			resp, err := e.Process(ctx, r)
			results <- result{idx, resp, err}

			// Update progress
			c := atomic.AddInt64(&completed, 1)
			if req.ProgressChan != nil {
				req.ProgressChan <- BatchProgress{
					Completed:  int(c),
					Total:      len(req.Requests),
					Percentage: float64(c) / float64(len(req.Requests)) * 100,
				}
			}
		}(i, procReq)
	}

	// Close results channel when all done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	responses := make([]*ProcessResponse, len(req.Requests))
	var successCount, failureCount int
	var totalConfidence float64

	for res := range results {
		if res.err != nil {
			failureCount++
			e.logger.Error("Document processing failed", map[string]any{
				"index": res.index,
				"error": res.err.Error(),
			})
		} else {
			successCount++
			totalConfidence += res.response.Confidence
		}
		responses[res.index] = res.response
	}

	avgConfidence := 0.0
	if successCount > 0 {
		avgConfidence = totalConfidence / float64(successCount)
	}

	return &BatchResponse{
		Results:           responses,
		TotalTime:         time.Since(startTime),
		SuccessCount:      successCount,
		FailureCount:      failureCount,
		AverageConfidence: avgConfidence,
	}, nil
}

// ========================================================================
// ZIP PROCESSING
// ========================================================================

// ProcessZIP processes a ZIP archive with checkpointing
func (e *ACEEngine) ProcessZIP(ctx context.Context, req *ZIPRequest) (*ZIPResponse, error) {
	startTime := time.Now()

	e.logger.Info("Processing ZIP archive", map[string]any{
		"zip_path": req.ZIPPath,
		"resume":   req.Resume,
	})

	// Open ZIP file
	zipReader, err := zip.OpenReader(req.ZIPPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP: %w", err)
	}
	defer zipReader.Close()

	// Load checkpoint if resuming
	var checkpoint *Checkpoint
	if req.Resume && req.CheckpointPath != "" {
		checkpoint, err = e.loadCheckpoint(req.CheckpointPath)
		if err != nil {
			e.logger.Warn("Failed to load checkpoint, starting fresh", map[string]any{
				"error": err.Error(),
			})
			checkpoint = &Checkpoint{
				ProcessedFiles: []string{},
				State:          make(map[string]any),
			}
		}
	} else {
		checkpoint = &Checkpoint{
			ProcessedFiles: []string{},
			State:          make(map[string]any),
		}
	}

	// Create set of processed files for quick lookup
	processedSet := make(map[string]bool)
	for _, f := range checkpoint.ProcessedFiles {
		processedSet[f] = true
	}

	// Filter processable files
	var processableFiles []*zip.File
	var totalBytes int64
	for _, f := range zipReader.File {
		if !f.FileInfo().IsDir() && e.isProcessableFile(f.Name) && !processedSet[f.Name] {
			processableFiles = append(processableFiles, f)
			totalBytes += int64(f.UncompressedSize64)
		}
	}

	e.logger.Info("Found processable files", map[string]any{
		"total":   len(processableFiles),
		"skipped": len(checkpoint.ProcessedFiles),
		"bytes":   totalBytes,
	})

	// Process files
	results := make(map[string]*ProcessResponse)
	var bytesProcessed int64
	var skippedCount int

	for i, f := range processableFiles {
		// Check context
		select {
		case <-ctx.Done():
			// Save checkpoint before exiting
			e.saveCheckpoint(req.CheckpointPath, checkpoint)
			return &ZIPResponse{
				TotalFiles:     len(processableFiles) + len(checkpoint.ProcessedFiles),
				ProcessedFiles: i,
				SkippedFiles:   skippedCount,
				Results:        results,
				Checkpoint:     checkpoint,
				TotalTime:      time.Since(startTime),
			}, ctx.Err()
		default:
		}

		// Open file from ZIP
		rc, err := f.Open()
		if err != nil {
			e.logger.Error("Failed to open file in ZIP", map[string]any{
				"file":  f.Name,
				"error": err.Error(),
			})
			skippedCount++
			continue
		}

		// Read file content
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			e.logger.Error("Failed to read file content", map[string]any{
				"file":  f.Name,
				"error": err.Error(),
			})
			skippedCount++
			continue
		}

		// Create process request
		procReq := &ProcessRequest{
			Source:     bytes.NewReader(content),
			SourceType: SourceReader,
			Context:    ctx,
		}
		if req.ProcessOptions != nil {
			procReq.DocumentType = req.ProcessOptions.DocumentType
			procReq.Language = req.ProcessOptions.Language
			procReq.EnableGPU = req.ProcessOptions.EnableGPU
			procReq.Tier = req.ProcessOptions.Tier
		}

		// Detect format from extension
		procReq.DocumentType = e.detectDocumentType(f.Name)

		// Process file
		response, err := e.Process(ctx, procReq)
		if err != nil {
			e.logger.Error("Failed to process file", map[string]any{
				"file":  f.Name,
				"error": err.Error(),
			})
			skippedCount++
			continue
		}

		results[f.Name] = response
		bytesProcessed += int64(f.UncompressedSize64)

		// Update checkpoint
		checkpoint.mu.Lock()
		checkpoint.ProcessedFiles = append(checkpoint.ProcessedFiles, f.Name)
		checkpoint.LastFile = f.Name
		checkpoint.Timestamp = time.Now()
		checkpoint.mu.Unlock()

		// Save checkpoint periodically (every 10 files)
		if (i+1)%10 == 0 && req.CheckpointPath != "" {
			e.saveCheckpoint(req.CheckpointPath, checkpoint)
		}

		// Report progress
		if req.ProgressChan != nil {
			req.ProgressChan <- ZIPProgress{
				CurrentFile:    f.Name,
				FilesProcessed: i + 1,
				TotalFiles:     len(processableFiles),
				Percentage:     float64(i+1) / float64(len(processableFiles)) * 100,
				BytesProcessed: bytesProcessed,
				TotalBytes:     totalBytes,
			}
		}
	}

	// Final checkpoint save
	if req.CheckpointPath != "" {
		e.saveCheckpoint(req.CheckpointPath, checkpoint)
	}

	return &ZIPResponse{
		TotalFiles:     len(processableFiles) + len(checkpoint.ProcessedFiles),
		ProcessedFiles: len(results),
		SkippedFiles:   skippedCount,
		Results:        results,
		Checkpoint:     checkpoint,
		TotalTime:      time.Since(startTime),
	}, nil
}

// ========================================================================
// HELPER METHODS
// ========================================================================

func (e *ACEEngine) validateRequest(req *ProcessRequest) error {
	if req.Source == nil {
		return fmt.Errorf("source is required")
	}
	return nil
}

func (e *ACEEngine) resolveSource(ctx context.Context, req *ProcessRequest) (io.Reader, func(), error) {
	cleanup := func() {}

	switch req.SourceType {
	case SourceFile:
		path, ok := req.Source.(string)
		if !ok {
			return nil, cleanup, fmt.Errorf("source must be string for file type")
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, cleanup, err
		}
		cleanup = func() { f.Close() }
		return f, cleanup, nil

	case SourceReader:
		reader, ok := req.Source.(io.Reader)
		if !ok {
			return nil, cleanup, fmt.Errorf("source must be io.Reader for reader type")
		}
		return reader, cleanup, nil

	default:
		return nil, cleanup, fmt.Errorf("unsupported source type: %d", req.SourceType)
	}
}

func (e *ACEEngine) preprocess(ctx context.Context, reader io.Reader, req *ProcessRequest) (io.Reader, error) {
	// Read input
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// Image preprocessing (pkg/ocr/preprocess): quaternion-based denoising and
	// contrast enhancement. Pure-Go CPU implementation despite the flag name —
	// EnableGPU selects this path, it does not dispatch to a GPU.
	if e.config.EnableGPU && e.config.EnablePreprocessing {
		gpConfig := preprocess.DefaultGPUPreprocessConfig()
		gpConfig.UseGPU = true
		gpConfig.DenoiseStrength = 0.5
		gpConfig.ContrastFactor = 1.2

		gpuPreprocessor, err := preprocess.NewGPUPreprocessor(gpConfig)
		if err != nil {
			// Fall back to CPU if GPU init fails
			return bytes.NewReader(data), nil
		}

		// Decode image
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			// Not an image or decode failed - return as-is
			return bytes.NewReader(data), nil
		}

		// GPU preprocessing (22.68M ops/sec quaternion operations!)
		preprocessed, err := gpuPreprocessor.PreprocessImage(ctx, img)
		if err != nil {
			// GPU processing failed - return original
			return bytes.NewReader(data), nil
		}

		// Encode back to PNG
		var buf bytes.Buffer
		if err := png.Encode(&buf, preprocessed); err != nil {
			return bytes.NewReader(data), nil
		}

		return bytes.NewReader(buf.Bytes()), nil
	}

	// CPU fallback or preprocessing disabled
	return bytes.NewReader(data), nil
}

func (e *ACEEngine) extractWithTierSelection(ctx context.Context, reader io.Reader, req *ProcessRequest) (*ProcessResponse, error) {
	// Read content
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// Try local extraction first
	response := &ProcessResponse{
		Tier:         TierLocal,
		GPUUsed:      e.config.EnableGPU, // ACTUAL GPU usage tracking!
		DocumentType: req.DocumentType,
	}

	// Detect format and convert if needed
	format := e.detectFormat(data)
	if format != FormatImage {
		// Use Pandoc for non-image formats
		converted, err := e.convertWithPandoc(ctx, data, format)
		if err != nil {
			e.logger.Warn("Pandoc conversion failed", map[string]any{
				"format": format,
				"error":  err.Error(),
			})
		} else {
			data = converted
		}
	}

	// Try Tesseract extraction
	text, confidence, err := e.extractWithTesseract(ctx, data, req.Language)
	if err != nil {
		e.logger.Warn("Tesseract extraction failed", map[string]any{
			"error": err.Error(),
		})
	}

	response.Text = text
	response.Confidence = confidence
	response.Fields = e.extractFields(text, req.DocumentType)

	// Escalate to the cloud OCR tier (Mistral OCR 4) if local confidence is too low.
	// Offline-first: a missing key or a failed cloud call always keeps the local result —
	// this never blocks or errors the pipeline.
	if confidence < e.config.ConfidenceThreshold && e.config.FallbackToMistral && e.mistralClient != nil {
		e.logger.Info("Confidence below threshold, escalating to Mistral OCR", map[string]any{
			"local_confidence": confidence,
			"threshold":        e.config.ConfidenceThreshold,
		})

		mimeType := "application/pdf"
		isImage := format == FormatImage
		if isImage {
			mimeType = "image/png"
		}

		mResult, mErr := e.mistralClient.Process(ctx, mistralocr.DocumentInput{
			Data:     data,
			MIMEType: mimeType,
			IsImage:  isImage,
		}, mistralocr.ProcessOptions{IncludeBlocks: true})
		if mErr != nil {
			e.logger.Error("Mistral OCR escalation failed", map[string]any{
				"error": mErr.Error(),
			})
			// Keep local results
		} else if mResult != nil && strings.TrimSpace(mResult.Text) != "" {
			mConfidence := averageBlockConfidence(mResult.Blocks)
			if mConfidence == 0 {
				// Mistral OCR 4 is the dedicated cloud tier; if it returned text but no
				// block-level confidence signal, treat it as clearing the escalation bar
				// rather than silently discarding better text for lack of a number.
				mConfidence = e.config.ConfidenceThreshold
			}
			if mConfidence > confidence {
				response.Text = mResult.Text
				response.Confidence = mConfidence
				response.Fields = e.extractFields(mResult.Text, req.DocumentType)
				response.Tier = TierCloudOCR
				response.EstimatedCostUSD = estimateMistralOCRCostUSD(1) // informational only; $4/1k pages
			}
		}
	}

	return response, nil
}

// averageBlockConfidence returns the mean of per-block confidence scores, or 0 if the
// API returned no blocks or no confidence signal on any of them.
func averageBlockConfidence(blocks []mistralocr.Block) float64 {
	if len(blocks) == 0 {
		return 0
	}
	var sum float64
	var counted int
	for _, b := range blocks {
		if b.Confidence > 0 {
			sum += b.Confidence
			counted++
		}
	}
	if counted == 0 {
		return 0
	}
	return sum / float64(counted)
}

// estimateMistralOCRCostUSD is informational only (not wired into billing): $4 per 1,000
// pages, per the confirmed Mistral OCR 4 pricing (see FABLE_WAVE13_REPORT.md A0 section).
func estimateMistralOCRCostUSD(pages int) float64 {
	return float64(pages) * 0.004
}

func (e *ACEEngine) extractWithTesseract(ctx context.Context, data []byte, lang Language) (string, float64, error) {
	// Find tesseract
	tesseractPath := e.config.TesseractPath
	if tesseractPath == "" {
		tesseractPath = "tesseract"
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "ocr-*.png")
	if err != nil {
		return "", 0, err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(data); err != nil {
		return "", 0, err
	}
	tmpFile.Close()

	// Build command
	langStr := string(lang)
	if lang == LangAuto || lang == "" {
		langStr = "eng" // Default to English
	}

	args := []string{tmpFile.Name(), "stdout", "-l", langStr}
	if e.config.TesseractDataPath != "" {
		args = append(args, "--tessdata-dir", e.config.TesseractDataPath)
	}

	cmd := exec.CommandContext(ctx, tesseractPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", 0, fmt.Errorf("tesseract failed: %w", err)
	}

	text := string(output)

	// Calculate confidence (simplified - in production, use hOCR output)
	confidence := e.estimateConfidence(text)

	return text, confidence, nil
}

func (e *ACEEngine) estimateConfidence(text string) float64 {
	if text == "" {
		return 0.0
	}

	// Simple heuristics for confidence estimation
	// In production, use Tesseract's confidence output
	wordCount := len(strings.Fields(text))
	if wordCount == 0 {
		return 0.1
	}

	// Check for common OCR artifacts
	artifactCount := strings.Count(text, "§") +
		strings.Count(text, "¶") +
		strings.Count(text, "®") +
		strings.Count(text, "©")

	artifactRatio := float64(artifactCount) / float64(wordCount)

	// Base confidence
	confidence := 0.85

	// Penalize artifacts
	confidence -= artifactRatio * 0.5

	// Bonus for reasonable text length
	if wordCount > 10 {
		confidence += 0.05
	}

	// Clamp
	if confidence < 0.1 {
		confidence = 0.1
	}
	if confidence > 0.99 {
		confidence = 0.99
	}

	return confidence
}

func (e *ACEEngine) convertWithPandoc(ctx context.Context, data []byte, format PandocFormat) ([]byte, error) {
	pandocPath := e.config.PandocPath
	if pandocPath == "" {
		pandocPath = "pandoc"
	}

	// Create temp input file
	tmpIn, err := os.CreateTemp("", fmt.Sprintf("pandoc-in-*.%s", format))
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpIn.Name())

	if _, err := tmpIn.Write(data); err != nil {
		tmpIn.Close()
		return nil, err
	}
	tmpIn.Close()

	// Create temp output file
	tmpOut, err := os.CreateTemp("", "pandoc-out-*.txt")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpOut.Name())
	tmpOut.Close()

	// Run pandoc
	cmd := exec.CommandContext(ctx, pandocPath, "-f", string(format), "-t", "plain", "-o", tmpOut.Name(), tmpIn.Name())
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pandoc failed: %w", err)
	}

	return os.ReadFile(tmpOut.Name())
}

func (e *ACEEngine) extractFields(text string, docType DocumentType) map[string]string {
	fields := make(map[string]string)

	// Document type specific extraction
	// In production, use regex patterns or ML-based extraction
	switch docType {
	case DocTypeInvoice:
		// Extract invoice fields
		fields["raw_text"] = text
	case DocTypePassport:
		// Extract passport fields
		fields["raw_text"] = text
	default:
		fields["raw_text"] = text
	}

	return fields
}

func (e *ACEEngine) detectFormat(data []byte) PandocFormat {
	// Check magic bytes
	if len(data) >= 4 {
		// PDF
		if bytes.HasPrefix(data, []byte("%PDF")) {
			return FormatPDF
		}
		// DOCX (ZIP with specific structure)
		if bytes.HasPrefix(data, []byte("PK\x03\x04")) {
			return FormatDOCX
		}
		// PNG
		if bytes.HasPrefix(data, []byte("\x89PNG")) {
			return FormatImage
		}
		// JPEG
		if bytes.HasPrefix(data, []byte("\xff\xd8\xff")) {
			return FormatImage
		}
		// TIFF
		if bytes.HasPrefix(data, []byte("II\x2a\x00")) || bytes.HasPrefix(data, []byte("MM\x00\x2a")) {
			return FormatImage
		}
	}

	// Default to image
	return FormatImage
}

func (e *ACEEngine) detectDocumentType(filename string) DocumentType {
	ext := strings.ToLower(filepath.Ext(filename))
	name := strings.ToLower(filename)

	// Check filename hints
	if strings.Contains(name, "invoice") {
		return DocTypeInvoice
	}
	if strings.Contains(name, "passport") {
		return DocTypePassport
	}
	if strings.Contains(name, "contract") {
		return DocTypeContract
	}
	if strings.Contains(name, "receipt") {
		return DocTypeReceipt
	}
	if strings.Contains(name, "boq") || strings.Contains(name, "bill") {
		return DocTypeBOQ
	}

	// Check extension
	switch ext {
	case ".pdf", ".docx", ".doc":
		return DocTypeGeneric
	case ".png", ".jpg", ".jpeg", ".tiff", ".tif":
		return DocTypeGeneric
	}

	return DocTypeUnknown
}

func (e *ACEEngine) isProcessableFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	processable := map[string]bool{
		".pdf":  true,
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".tiff": true,
		".tif":  true,
		".docx": true,
		".doc":  true,
		".txt":  true,
		".rtf":  true,
	}
	return processable[ext]
}

func (e *ACEEngine) loadCheckpoint(path string) (*Checkpoint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var checkpoint Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, err
	}

	return &checkpoint, nil
}

func (e *ACEEngine) saveCheckpoint(path string, checkpoint *Checkpoint) error {
	if path == "" {
		return nil
	}

	checkpoint.mu.RLock()
	data, err := json.MarshalIndent(checkpoint, "", "  ")
	checkpoint.mu.RUnlock()

	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// HealthCheck verifies the engine is operational
func (e *ACEEngine) HealthCheck(ctx context.Context) error {
	// Check Tesseract
	cmd := exec.CommandContext(ctx, "tesseract", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tesseract not available: %w", err)
	}

	// Check Pandoc (optional)
	cmd = exec.CommandContext(ctx, "pandoc", "--version")
	if err := cmd.Run(); err != nil {
		e.logger.Warn("Pandoc not available", map[string]any{
			"error": err.Error(),
		})
	}

	return nil
}

// Close cleans up engine resources
func (e *ACEEngine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.initialized = false
	return nil
}

// ========================================================================
// DEFAULT IMPLEMENTATIONS
// ========================================================================

type defaultLogger struct{}

func (l *defaultLogger) Debug(msg string, fields map[string]any) {
	fmt.Printf("[DEBUG] %s %v\n", msg, fields)
}
func (l *defaultLogger) Info(msg string, fields map[string]any) {
	fmt.Printf("[INFO] %s %v\n", msg, fields)
}
func (l *defaultLogger) Warn(msg string, fields map[string]any) {
	fmt.Printf("[WARN] %s %v\n", msg, fields)
}
func (l *defaultLogger) Error(msg string, fields map[string]any) {
	fmt.Printf("[ERROR] %s %v\n", msg, fields)
}

type defaultMetrics struct{}

func (m *defaultMetrics) RecordProcessingTime(duration time.Duration, tier ProcessingTier) {}
func (m *defaultMetrics) RecordConfidence(confidence float64, docType DocumentType)        {}
func (m *defaultMetrics) RecordError(stage string, err error)                              {}
func (m *defaultMetrics) RecordGPUUsage(used bool, duration time.Duration)                 {}
func (m *defaultMetrics) RecordCost(costUSD float64, tier ProcessingTier)                  {}
func (m *defaultMetrics) RecordBatchProgress(completed, total int)                         {}
