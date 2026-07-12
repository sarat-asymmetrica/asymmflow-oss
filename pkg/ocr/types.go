// Package ocr provides GPU-accelerated OCR with Trinity optimization.
// σ: ACE-OCR | ρ: pkg/ocr | γ: Production | κ: O(√n×log₂n) | λ: Zen_Gardener
//
// Features:
// - GPU-first preprocessing (Level Zero / Vulkan / CUDA fallback)
// - Trinity optimization (Tesla + Ramanujan + Madhava)
// - AIMLAPI integration for tough cases
// - Pandoc multi-format support
// - ZIP folder processing with checkpoints
// - Full observability (metrics, logs, streaming)
//
// Author: Research Dyad (the maintainer + Claude)
// Date: December 17, 2025
package ocr

import (
	"context"
	"io"
	"sync"
	"time"
)

// ========================================================================
// CORE TYPES
// ========================================================================

// ProcessingTier defines the OCR processing tier
type ProcessingTier int

const (
	// TierLocal uses local GPU + Tesseract (fastest, cheapest)
	TierLocal ProcessingTier = iota
	// TierAIMLAPI uses AIMLAPI for tough cases
	TierAIMLAPI
	// TierConsensus uses multi-model consensus (most accurate)
	TierConsensus
)

// String returns the string representation of ProcessingTier
func (t ProcessingTier) String() string {
	switch t {
	case TierLocal:
		return "local"
	case TierAIMLAPI:
		return "aimlapi"
	case TierConsensus:
		return "consensus"
	default:
		return "unknown"
	}
}

// DocumentType represents the type of document being processed
type DocumentType string

const (
	DocTypeUnknown      DocumentType = "unknown"
	DocTypePassport     DocumentType = "passport"
	DocTypeInvoice      DocumentType = "invoice"
	DocTypeContract     DocumentType = "contract"
	DocTypeDiploma      DocumentType = "diploma"
	DocTypeBirthCert    DocumentType = "birth_certificate"
	DocTypeIDCard       DocumentType = "id_card"
	DocTypeReceipt      DocumentType = "receipt"
	DocTypeBOQ          DocumentType = "bill_of_quantities"
	DocTypeLetterhead   DocumentType = "letterhead"
	DocTypeRFQ          DocumentType = "rfq"   // Request for Quotation
	DocTypeQuote        DocumentType = "quote" // Quotation/Offer
	DocTypePO           DocumentType = "purchase_order"
	DocTypeDeliveryNote DocumentType = "delivery_note"
	DocTypeGeneric      DocumentType = "generic"
)

// Language represents supported languages
type Language string

const (
	LangEnglish    Language = "eng"
	LangArabic     Language = "ara"
	LangHindi      Language = "hin"
	LangChinese    Language = "chi_sim"
	LangJapanese   Language = "jpn"
	LangKorean     Language = "kor"
	LangRussian    Language = "rus"
	LangFrench     Language = "fra"
	LangSpanish    Language = "spa"
	LangGerman     Language = "deu"
	LangDutch      Language = "nld"
	LangPortuguese Language = "por"
	LangAuto       Language = "auto" // Auto-detect
)

// ========================================================================
// REQUEST/RESPONSE TYPES
// ========================================================================

// ProcessRequest represents an OCR processing request
type ProcessRequest struct {
	// Source can be: file path, URL, or io.Reader
	Source     any
	SourceType SourceType

	// Document metadata
	DocumentType DocumentType
	Language     Language
	CountryCode  string // ISO 3166-1 alpha-2

	// Processing options
	EnableGPU           bool
	EnablePreprocessing bool
	EnableTranslation   bool
	TargetLanguage      Language

	// Tier selection
	Tier              ProcessingTier
	FallbackToAIMLAPI bool

	// Checkpointing
	CheckpointDir        string
	ResumeFromCheckpoint bool

	// Context for cancellation
	Context context.Context
}

// SourceType defines the type of input source
type SourceType int

const (
	SourceFile SourceType = iota
	SourceURL
	SourceReader
	SourceZIP
	SourceDirectory
)

// ProcessResponse represents the OCR processing result
type ProcessResponse struct {
	// Extracted content
	Text       string
	Fields     map[string]string
	Confidence float64

	// Metadata
	DocumentType     DocumentType
	DetectedLanguage Language
	PageCount        int

	// Performance
	ProcessingTime time.Duration
	Tier           ProcessingTier
	GPUUsed        bool

	// Costs
	EstimatedCostUSD float64

	// Trinity metrics
	TrinityMetrics *TrinityMetrics

	// Validation
	VedicValidation *VedicValidation

	// Errors (if any)
	Errors   []ProcessingError
	Warnings []string
}

// ProcessingError represents an error during processing
type ProcessingError struct {
	Stage     string
	Message   string
	Timestamp time.Time
	Fatal     bool
}

// ========================================================================
// BATCH PROCESSING
// ========================================================================

// BatchRequest represents a batch OCR request
type BatchRequest struct {
	Requests       []*ProcessRequest
	MaxConcurrency int
	ProgressChan   chan<- BatchProgress
	Context        context.Context
}

// BatchResponse represents batch processing results
type BatchResponse struct {
	Results           []*ProcessResponse
	TotalTime         time.Duration
	SuccessCount      int
	FailureCount      int
	AverageConfidence float64
}

// BatchProgress reports progress during batch processing
type BatchProgress struct {
	Completed    int
	Total        int
	CurrentFile  string
	Percentage   float64
	EstimatedETA time.Duration
}

// ========================================================================
// ZIP PROCESSING
// ========================================================================

// ZIPRequest represents a ZIP folder processing request
type ZIPRequest struct {
	ZIPPath        string
	ExtractDir     string
	ProcessOptions *ProcessRequest

	// Checkpoint support
	CheckpointPath string
	Resume         bool

	// Progress streaming
	ProgressChan chan<- ZIPProgress
	Context      context.Context
}

// ZIPResponse represents ZIP processing results
type ZIPResponse struct {
	TotalFiles     int
	ProcessedFiles int
	SkippedFiles   int
	Results        map[string]*ProcessResponse
	Checkpoint     *Checkpoint
	TotalTime      time.Duration
}

// ZIPProgress reports progress during ZIP processing
type ZIPProgress struct {
	CurrentFile    string
	FilesProcessed int
	TotalFiles     int
	Percentage     float64
	BytesProcessed int64
	TotalBytes     int64
}

// Checkpoint stores processing state for resumption
type Checkpoint struct {
	ProcessedFiles []string
	LastFile       string
	Timestamp      time.Time
	State          map[string]any
	mu             sync.RWMutex
}

// ========================================================================
// TRINITY OPTIMIZATION
// ========================================================================

// TrinityMetrics contains optimization metrics
type TrinityMetrics struct {
	// Tesla resonance
	TeslaFrequency float64
	TeslaHarmonic  int

	// Ramanujan partitioning
	RamanujanPartition int
	OptimalBatchSize   int

	// Madhava convergence
	MadhavaIterations int
	ConvergenceRate   float64

	// Combined efficiency
	EfficiencyGain float64 // Multiplier vs naive approach
	DigitalRoot    int
	Regime         int // 1, 2, or 3
}

// VedicValidation contains Vedic math validation results
type VedicValidation struct {
	HarmonicMean     float64
	DharmaIndex      float64
	NikhilamValid    bool
	DigitalRootCheck bool
	ConfidenceBoost  float64
}

// ========================================================================
// BABEL LANGUAGE MAPPING
// ========================================================================

// FieldMapping maps local terminology to standard fields
type FieldMapping struct {
	LocalTerm      string
	StandardTerm   string
	Confidence     float64
	AlternateTerms []string
	Explanation    string
}

// BabelResult contains language mapping results
type BabelResult struct {
	CountryCode    string
	DocumentType   DocumentType
	Mappings       []FieldMapping
	UnmappedFields []string
}

// ========================================================================
// OBSERVABILITY
// ========================================================================

// MetricsCollector collects OCR pipeline metrics
type MetricsCollector interface {
	RecordProcessingTime(duration time.Duration, tier ProcessingTier)
	RecordConfidence(confidence float64, docType DocumentType)
	RecordError(stage string, err error)
	RecordGPUUsage(used bool, duration time.Duration)
	RecordCost(costUSD float64, tier ProcessingTier)
	RecordBatchProgress(completed, total int)
}

// Logger provides structured logging
type Logger interface {
	Debug(msg string, fields map[string]any)
	Info(msg string, fields map[string]any)
	Warn(msg string, fields map[string]any)
	Error(msg string, fields map[string]any)
}

// StreamHandler handles streaming output
type StreamHandler interface {
	OnPageComplete(pageNum int, text string, confidence float64)
	OnDocumentComplete(response *ProcessResponse)
	OnError(err error)
	OnProgress(progress float64)
}

// ========================================================================
// GPU ACCELERATION
// ========================================================================

// GPUBackend represents the GPU acceleration backend
type GPUBackend int

const (
	GPUNone GPUBackend = iota
	GPULevelZero
	GPUVulkan
	GPUCUDA
	GPUOpenCL
)

// GPUConfig configures GPU acceleration
type GPUConfig struct {
	Backend         GPUBackend
	DeviceID        int
	MaxMemoryMB     int
	EnableProfiling bool
	FallbackToCPU   bool
}

// GPUStats contains GPU utilization statistics
type GPUStats struct {
	Backend            GPUBackend
	DeviceName         string
	MemoryUsedMB       int
	MemoryTotalMB      int
	UtilizationPercent float64
	KernelTime         time.Duration
}

// ========================================================================
// PANDOC INTEGRATION
// ========================================================================

// PandocFormat represents supported input formats
type PandocFormat string

const (
	FormatPDF      PandocFormat = "pdf"
	FormatDOCX     PandocFormat = "docx"
	FormatDOC      PandocFormat = "doc"
	FormatODT      PandocFormat = "odt"
	FormatRTF      PandocFormat = "rtf"
	FormatHTML     PandocFormat = "html"
	FormatMarkdown PandocFormat = "markdown"
	FormatTXT      PandocFormat = "txt"
	FormatXLSX     PandocFormat = "xlsx"
	FormatCSV      PandocFormat = "csv"
	FormatImage    PandocFormat = "image" // PNG, JPG, TIFF, etc.
)

// ConversionResult represents format conversion output
type ConversionResult struct {
	OutputPath string
	Format     PandocFormat
	Success    bool
	Error      error
	PageCount  int
}

// ========================================================================
// INTERFACES
// ========================================================================

// Engine is the main OCR engine interface
type Engine interface {
	// Single document processing
	Process(ctx context.Context, req *ProcessRequest) (*ProcessResponse, error)

	// Batch processing
	ProcessBatch(ctx context.Context, req *BatchRequest) (*BatchResponse, error)

	// ZIP folder processing
	ProcessZIP(ctx context.Context, req *ZIPRequest) (*ZIPResponse, error)

	// Health check
	HealthCheck(ctx context.Context) error

	// Cleanup
	Close() error
}

// Preprocessor handles image preprocessing
type Preprocessor interface {
	Preprocess(ctx context.Context, input io.Reader) (io.Reader, error)
	PreprocessGPU(ctx context.Context, input io.Reader, config *GPUConfig) (io.Reader, error)
}

// Extractor performs the actual OCR extraction
type Extractor interface {
	Extract(ctx context.Context, input io.Reader, opts *ProcessRequest) (*ProcessResponse, error)
}

// Postprocessor handles post-processing and validation
type Postprocessor interface {
	Validate(ctx context.Context, response *ProcessResponse) (*VedicValidation, error)
	MapFields(ctx context.Context, response *ProcessResponse, countryCode string) (*BabelResult, error)
}
