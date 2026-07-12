// Observability for ACE OCR pipeline.
// σ: Observability | ρ: pkg/ocr | γ: Production | κ: O(1)
//
// Provides:
// - Prometheus metrics (processing time, confidence, throughput)
// - Structured logging (JSON format)
// - Streaming output handler
// - Performance profiling
package ocr

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// ========================================================================
// PROMETHEUS METRICS
// ========================================================================

// PrometheusMetrics implements MetricsCollector with Prometheus-style metrics
type PrometheusMetrics struct {
	// Counters
	documentsProcessed int64
	documentsSucceeded int64
	documentsFailed    int64
	totalProcessingMs  int64

	// Histograms (simplified)
	processingTimes  []float64
	confidenceScores []float64

	// By tier
	tierCounts map[ProcessingTier]int64
	tierCosts  map[ProcessingTier]float64

	// GPU metrics
	gpuUsageCount int64
	gpuTotalMs    int64

	mu sync.RWMutex
}

// NewPrometheusMetrics creates a new Prometheus metrics collector
func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		processingTimes:  make([]float64, 0, 1000),
		confidenceScores: make([]float64, 0, 1000),
		tierCounts:       make(map[ProcessingTier]int64),
		tierCosts:        make(map[ProcessingTier]float64),
	}
}

func (m *PrometheusMetrics) RecordProcessingTime(duration time.Duration, tier ProcessingTier) {
	m.mu.Lock()
	defer m.mu.Unlock()

	atomic.AddInt64(&m.documentsProcessed, 1)
	atomic.AddInt64(&m.totalProcessingMs, duration.Milliseconds())

	// Record for histogram
	m.processingTimes = append(m.processingTimes, duration.Seconds())

	// Track by tier
	m.tierCounts[tier]++
}

func (m *PrometheusMetrics) RecordConfidence(confidence float64, docType DocumentType) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.confidenceScores = append(m.confidenceScores, confidence)

	if confidence >= MIN_CONFIDENCE {
		atomic.AddInt64(&m.documentsSucceeded, 1)
	} else {
		atomic.AddInt64(&m.documentsFailed, 1)
	}
}

func (m *PrometheusMetrics) RecordError(stage string, err error) {
	atomic.AddInt64(&m.documentsFailed, 1)
}

func (m *PrometheusMetrics) RecordGPUUsage(used bool, duration time.Duration) {
	if used {
		atomic.AddInt64(&m.gpuUsageCount, 1)
		atomic.AddInt64(&m.gpuTotalMs, duration.Milliseconds())
	}
}

func (m *PrometheusMetrics) RecordCost(costUSD float64, tier ProcessingTier) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tierCosts[tier] += costUSD
}

func (m *PrometheusMetrics) RecordBatchProgress(completed, total int) {
	// Could emit to progress channel if needed
}

// GetStats returns current statistics
func (m *PrometheusMetrics) GetStats() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalDocs := atomic.LoadInt64(&m.documentsProcessed)
	totalMs := atomic.LoadInt64(&m.totalProcessingMs)

	avgProcessingMs := float64(0)
	if totalDocs > 0 {
		avgProcessingMs = float64(totalMs) / float64(totalDocs)
	}

	avgConfidence := float64(0)
	if len(m.confidenceScores) > 0 {
		sum := float64(0)
		for _, c := range m.confidenceScores {
			sum += c
		}
		avgConfidence = sum / float64(len(m.confidenceScores))
	}

	// Calculate total cost
	totalCost := float64(0)
	for _, cost := range m.tierCosts {
		totalCost += cost
	}

	return map[string]any{
		"documents_processed":   totalDocs,
		"documents_succeeded":   atomic.LoadInt64(&m.documentsSucceeded),
		"documents_failed":      atomic.LoadInt64(&m.documentsFailed),
		"average_processing_ms": avgProcessingMs,
		"average_confidence":    avgConfidence,
		"gpu_usage_count":       atomic.LoadInt64(&m.gpuUsageCount),
		"gpu_total_ms":          atomic.LoadInt64(&m.gpuTotalMs),
		"total_cost_usd":        totalCost,
		"tier_breakdown":        m.tierCounts,
	}
}

// ToPrometheusFormat exports metrics in Prometheus text format
func (m *PrometheusMetrics) ToPrometheusFormat() string {
	stats := m.GetStats()

	output := ""
	output += "# HELP ace_ocr_documents_processed Total documents processed\n"
	output += "# TYPE ace_ocr_documents_processed counter\n"
	output += fmt.Sprintf("ace_ocr_documents_processed %d\n\n", stats["documents_processed"])

	output += fmt.Sprintf("# HELP ace_ocr_documents_succeeded Documents with confidence >= %.2f\n", MIN_CONFIDENCE)
	output += "# TYPE ace_ocr_documents_succeeded counter\n"
	output += fmt.Sprintf("ace_ocr_documents_succeeded %d\n\n", stats["documents_succeeded"])

	output += "# HELP ace_ocr_average_processing_ms Average processing time in milliseconds\n"
	output += "# TYPE ace_ocr_average_processing_ms gauge\n"
	output += fmt.Sprintf("ace_ocr_average_processing_ms %.2f\n\n", stats["average_processing_ms"])

	output += "# HELP ace_ocr_average_confidence Average OCR confidence score\n"
	output += "# TYPE ace_ocr_average_confidence gauge\n"
	output += fmt.Sprintf("ace_ocr_average_confidence %.4f\n\n", stats["average_confidence"])

	output += "# HELP ace_ocr_total_cost_usd Total cost in USD\n"
	output += "# TYPE ace_ocr_total_cost_usd counter\n"
	output += fmt.Sprintf("ace_ocr_total_cost_usd %.6f\n\n", stats["total_cost_usd"])

	return output
}

// ========================================================================
// STRUCTURED LOGGER
// ========================================================================

// StructuredLogger implements Logger with JSON output
type StructuredLogger struct {
	output    io.Writer
	level     LogLevel
	component string
	mu        sync.Mutex
}

// LogLevel defines logging levels
type LogLevel int

const (
	LogDebug LogLevel = iota
	LogInfo
	LogWarn
	LogError
)

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	Component string         `json:"component,omitempty"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(output io.Writer, level LogLevel, component string) *StructuredLogger {
	if output == nil {
		output = os.Stdout
	}
	return &StructuredLogger{
		output:    output,
		level:     level,
		component: component,
	}
}

func (l *StructuredLogger) log(level LogLevel, levelStr string, msg string, fields map[string]any) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     levelStr,
		Component: l.component,
		Message:   msg,
		Fields:    fields,
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if data, err := json.Marshal(entry); err == nil {
		fmt.Fprintln(l.output, string(data))
	}
}

func (l *StructuredLogger) Debug(msg string, fields map[string]any) {
	l.log(LogDebug, "DEBUG", msg, fields)
}

func (l *StructuredLogger) Info(msg string, fields map[string]any) {
	l.log(LogInfo, "INFO", msg, fields)
}

func (l *StructuredLogger) Warn(msg string, fields map[string]any) {
	l.log(LogWarn, "WARN", msg, fields)
}

func (l *StructuredLogger) Error(msg string, fields map[string]any) {
	l.log(LogError, "ERROR", msg, fields)
}

// ========================================================================
// STREAMING OUTPUT HANDLER
// ========================================================================

// StreamingHandler implements StreamHandler for real-time output
type StreamingHandler struct {
	onPageComplete     func(pageNum int, text string, confidence float64)
	onDocumentComplete func(response *ProcessResponse)
	onError            func(err error)
	onProgress         func(progress float64)
	mu                 sync.Mutex
}

// NewStreamingHandler creates a new streaming handler
func NewStreamingHandler() *StreamingHandler {
	return &StreamingHandler{}
}

// OnPageComplete sets the page completion callback
func (s *StreamingHandler) SetOnPageComplete(fn func(pageNum int, text string, confidence float64)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onPageComplete = fn
}

// SetOnDocumentComplete sets the document completion callback
func (s *StreamingHandler) SetOnDocumentComplete(fn func(response *ProcessResponse)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onDocumentComplete = fn
}

// SetOnError sets the error callback
func (s *StreamingHandler) SetOnError(fn func(err error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onError = fn
}

// SetOnProgress sets the progress callback
func (s *StreamingHandler) SetOnProgress(fn func(progress float64)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onProgress = fn
}

func (s *StreamingHandler) OnPageComplete(pageNum int, text string, confidence float64) {
	s.mu.Lock()
	fn := s.onPageComplete
	s.mu.Unlock()

	if fn != nil {
		fn(pageNum, text, confidence)
	}
}

func (s *StreamingHandler) OnDocumentComplete(response *ProcessResponse) {
	s.mu.Lock()
	fn := s.onDocumentComplete
	s.mu.Unlock()

	if fn != nil {
		fn(response)
	}
}

func (s *StreamingHandler) OnError(err error) {
	s.mu.Lock()
	fn := s.onError
	s.mu.Unlock()

	if fn != nil {
		fn(err)
	}
}

func (s *StreamingHandler) OnProgress(progress float64) {
	s.mu.Lock()
	fn := s.onProgress
	s.mu.Unlock()

	if fn != nil {
		fn(progress)
	}
}

// ========================================================================
// PERFORMANCE PROFILER
// ========================================================================

// Profiler tracks detailed performance metrics
type Profiler struct {
	stages  map[string][]time.Duration
	current string
	start   time.Time
	mu      sync.RWMutex
}

// NewProfiler creates a new performance profiler
func NewProfiler() *Profiler {
	return &Profiler{
		stages: make(map[string][]time.Duration),
	}
}

// StartStage begins timing a stage
func (p *Profiler) StartStage(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current = name
	p.start = time.Now()
}

// EndStage ends timing the current stage
func (p *Profiler) EndStage() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.current != "" {
		duration := time.Since(p.start)
		p.stages[p.current] = append(p.stages[p.current], duration)
		p.current = ""
	}
}

// GetStageSummary returns summary statistics for all stages
func (p *Profiler) GetStageSummary() map[string]map[string]any {
	p.mu.RLock()
	defer p.mu.RUnlock()

	summary := make(map[string]map[string]any)

	for stage, durations := range p.stages {
		if len(durations) == 0 {
			continue
		}

		// Calculate statistics
		var total time.Duration
		min := durations[0]
		max := durations[0]

		for _, d := range durations {
			total += d
			if d < min {
				min = d
			}
			if d > max {
				max = d
			}
		}

		avg := total / time.Duration(len(durations))

		summary[stage] = map[string]any{
			"count":      len(durations),
			"total_ms":   total.Milliseconds(),
			"average_ms": avg.Milliseconds(),
			"min_ms":     min.Milliseconds(),
			"max_ms":     max.Milliseconds(),
		}
	}

	return summary
}

// PrintSummary prints a formatted summary to stdout
func (p *Profiler) PrintSummary() {
	summary := p.GetStageSummary()

	fmt.Println("\n=== Performance Profile ===")
	fmt.Println()

	for stage, stats := range summary {
		fmt.Printf("Stage: %s\n", stage)
		fmt.Printf("  Count:   %v\n", stats["count"])
		fmt.Printf("  Total:   %vms\n", stats["total_ms"])
		fmt.Printf("  Average: %vms\n", stats["average_ms"])
		fmt.Printf("  Min:     %vms\n", stats["min_ms"])
		fmt.Printf("  Max:     %vms\n", stats["max_ms"])
		fmt.Println()
	}
}

// ========================================================================
// FIVE TIMBRES QUALITY METRICS
// ========================================================================

// FiveTimbresMetrics represents the Five Timbres quality assessment
type FiveTimbresMetrics struct {
	Correctness  float64 // Did processing succeed?
	Performance  float64 // Processing speed (<100ms/page target)
	Reliability  float64 // Error rate
	Synergy      float64 // Pattern detection quality
	Elegance     float64 // Extraction quality
	UnifiedScore float64 // Harmonic mean
	Verdict      string  // Overall assessment
}

// CalculateFiveTimbres computes Five Timbres metrics
func CalculateFiveTimbres(response *ProcessResponse, targetMsPerPage int) *FiveTimbresMetrics {
	metrics := &FiveTimbresMetrics{}

	// Correctness (0-1): Did we get results?
	if response != nil && response.Text != "" {
		metrics.Correctness = 1.0
		if len(response.Errors) > 0 {
			metrics.Correctness -= float64(len(response.Errors)) * 0.1
		}
		if metrics.Correctness < 0 {
			metrics.Correctness = 0
		}
	}

	// Performance (0-1): Speed relative to target
	if response != nil && response.PageCount > 0 && targetMsPerPage > 0 {
		actualMsPerPage := float64(response.ProcessingTime.Milliseconds()) / float64(response.PageCount)
		metrics.Performance = float64(targetMsPerPage) / actualMsPerPage
		if metrics.Performance > 1.0 {
			metrics.Performance = 1.0
		}
	}

	// Reliability (0-1): Based on confidence
	if response != nil {
		metrics.Reliability = response.Confidence
	}

	// Synergy (0-1): Based on Trinity metrics
	if response != nil && response.TrinityMetrics != nil {
		// Higher regime = better synergy
		metrics.Synergy = float64(response.TrinityMetrics.Regime) / 3.0
	}

	// Elegance (0-1): Based on field extraction quality
	if response != nil && len(response.Fields) > 0 {
		// More fields = more elegant extraction
		metrics.Elegance = float64(len(response.Fields)) / 10.0
		if metrics.Elegance > 1.0 {
			metrics.Elegance = 1.0
		}
	}

	// Unified Score: Harmonic mean of all five
	values := []float64{
		metrics.Correctness,
		metrics.Performance,
		metrics.Reliability,
		metrics.Synergy,
		metrics.Elegance,
	}

	sum := float64(0)
	for _, v := range values {
		if v > 0 {
			sum += 1.0 / v
		} else {
			sum += 100 // Penalty for zero
		}
	}
	metrics.UnifiedScore = float64(len(values)) / sum

	// Verdict
	switch {
	case metrics.UnifiedScore >= 0.9:
		metrics.Verdict = "LEGENDARY RUN"
	case metrics.UnifiedScore >= 0.8:
		metrics.Verdict = "Production Ready"
	case metrics.UnifiedScore >= 0.7:
		metrics.Verdict = "Acceptable"
	case metrics.UnifiedScore >= 0.5:
		metrics.Verdict = "Needs Improvement"
	default:
		metrics.Verdict = "Critical Issues"
	}

	return metrics
}
