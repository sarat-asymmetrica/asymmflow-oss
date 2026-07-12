package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

// =============================================================================
// ASYNC JOB QUEUE SYSTEM
// =============================================================================
//
// PURPOSE:
// - Handle long-running operations without blocking UI
// - Track job progress and completion
// - Support report generation, OCR batches, exports
//
// DESIGN:
// - SQLite-backed (no external dependencies like Redis/Sidekiq)
// - In-memory worker pool (restart-safe via status column)
// - Polling-based frontend (WebSocket would be overkill for PH scale)
//
// BUSINESS CONTEXT:
// - Report generation: 2-10 seconds for large datasets
// - OCR batch processing: Minutes for 100+ documents
// - Excel exports: 1-5 seconds for complex reports
// =============================================================================

// Job is defined in database.go
// This file only contains the JobQueue infrastructure

// JobQueue manages background job processing
type JobQueue struct {
	db       *gorm.DB
	workers  int
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.RWMutex
	handlers map[string]JobHandler
}

// JobHandler processes a specific job type
type JobHandler func(ctx context.Context, job *Job) error

// NewJobQueue creates a new job queue with n worker goroutines
func NewJobQueue(db *gorm.DB, workers int) *JobQueue {
	ctx, cancel := context.WithCancel(context.Background())
	return &JobQueue{
		db:       db,
		workers:  workers,
		ctx:      ctx,
		cancel:   cancel,
		handlers: make(map[string]JobHandler),
	}
}

// RegisterHandler registers a job type handler
func (jq *JobQueue) RegisterHandler(jobType string, handler JobHandler) {
	jq.mu.Lock()
	defer jq.mu.Unlock()
	jq.handlers[jobType] = handler
}

// Enqueue adds a job to the queue
func (jq *JobQueue) Enqueue(jobType string, input any, createdBy string) (string, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("failed to marshal job input: %w", err)
	}

	job := Job{
		Type:        jobType,
		Status:      "pending",
		Input:       string(inputJSON),
		Progress:    0,
		Attempts:    0,
		MaxAttempts: 3, // Retry up to 3 times
	}

	if err := jq.db.Create(&job).Error; err != nil {
		return "", fmt.Errorf("failed to create job: %w", err)
	}

	return job.ID, nil
}

// GetJob retrieves job status by ID
func (jq *JobQueue) GetJob(id string) (*Job, error) {
	var job Job
	if err := jq.db.First(&job, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// GetJobsByStatus retrieves jobs with specific status
func (jq *JobQueue) GetJobsByStatus(status string, limit int) ([]Job, error) {
	var jobs []Job
	query := jq.db.Where("status = ?", status).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&jobs).Error; err != nil {
		return nil, err
	}

	return jobs, nil
}

// GetRecentJobs retrieves recent jobs (all statuses)
func (jq *JobQueue) GetRecentJobs(limit int) ([]Job, error) {
	var jobs []Job
	query := jq.db.Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&jobs).Error; err != nil {
		return nil, err
	}

	return jobs, nil
}

// CancelJob cancels a pending job
func (jq *JobQueue) CancelJob(id string) error {
	return jq.db.Model(&Job{}).Where("id = ? AND status = ?", id, "pending").
		Update("status", "cancelled").Error
}

// CleanupOldJobs removes completed/failed jobs older than retention period
func (jq *JobQueue) CleanupOldJobs(retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	return jq.db.Where("status IN ? AND created_at < ?",
		[]string{"completed", "failed", "cancelled"}, cutoff).
		Delete(&Job{}).Error
}

// updateJob saves the current job state to the database
func (jq *JobQueue) updateJob(job *Job) error {
	return jq.db.Save(job).Error
}

// =============================================================================
// JOB INPUT/OUTPUT TYPES
// =============================================================================

// ReportGenerateInput defines input for report generation jobs
type ReportGenerateInput struct {
	Category  string `json:"category"` // sales, customers, financial, etc.
	Format    string `json:"format"`   // pdf, excel, csv
	DateRange struct {
		Start string `json:"start"` // YYYY-MM-DD
		End   string `json:"end"`   // YYYY-MM-DD
	} `json:"date_range"`
	Options map[string]any `json:"options,omitempty"`
}

// ReportGenerateOutput defines output for report generation jobs
type ReportGenerateOutput struct {
	FilePath  string    `json:"file_path"`
	FileSize  int64     `json:"file_size"`
	RowCount  int       `json:"row_count,omitempty"`
	Generated time.Time `json:"generated"`
}

// OCRBatchInput defines input for batch OCR jobs
type OCRBatchInput struct {
	FilePaths []string       `json:"file_paths"`
	Options   map[string]any `json:"options,omitempty"`
}

// OCRBatchOutput defines output for batch OCR jobs
type OCRBatchOutput struct {
	ProcessedCount int      `json:"processed_count"`
	FailedCount    int      `json:"failed_count"`
	Results        []string `json:"results"`
}
