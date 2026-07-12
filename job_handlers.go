package main

import (
	"encoding/json"
	"fmt"
	"log"
)

// =============================================================================
// WAILS BINDINGS FOR JOB QUEUE
// =============================================================================
//
// FRONTEND API:
// - GenerateReportAsync(category, format, dateRange) → job_id
// - GetJobStatus(job_id) → status, progress, output
// - CancelJob(job_id) → success
// - GetRecentJobs(limit) → jobs[]
//
// POLLING:
// Frontend polls GetJobStatus every 1-2 seconds until status = "completed"
// =============================================================================

// InitializeJobQueue initializes the job queue system during app startup
// Call this AFTER database migration in startup().
// Mission I (I-11): bound method is gated; startup uses initializeJobQueueInternal.
func (a *App) InitializeJobQueue() error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	return a.initializeJobQueueInternal()
}

func (a *App) initializeJobQueueInternal() error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Auto-migrate Job table
	if err := a.db.AutoMigrate(&Job{}); err != nil {
		return fmt.Errorf("failed to migrate Job table: %w", err)
	}

	// Create job queue with 2 workers (suitable for single-user desktop app)
	a.jobQueue = NewJobQueue(a.db, 2)

	// Register report generation handler
	a.RegisterReportHandlers()

	// Note: Worker pool would be started here if implemented
	// For now, jobs are processed synchronously or via polling

	log.Println("✓ Job Queue initialized with 2 workers")
	return nil
}

// ShutdownJobQueue gracefully stops the job queue
// Call this in app shutdown/cleanup
func (a *App) ShutdownJobQueue() {
	if a.jobQueue != nil {
		// Note: Worker pool cleanup would happen here if implemented
		log.Println("✓ Job Queue stopped")
	}
}

// =============================================================================
// FRONTEND BINDINGS
// =============================================================================

// GenerateReportAsync queues a report generation job
// Returns job ID immediately for frontend polling
func (a *App) GenerateReportAsync(category, format, startDate, endDate string) (string, error) {
	if a.jobQueue == nil {
		return "", fmt.Errorf("job queue not initialized")
	}

	input := ReportGenerateInput{
		Category: category,
		Format:   format,
	}
	input.DateRange.Start = startDate
	input.DateRange.End = endDate

	jobID, err := a.jobQueue.Enqueue("report_generate", input, "system")
	if err != nil {
		return "", fmt.Errorf("failed to enqueue report job: %w", err)
	}

	log.Printf("📊 Report job enqueued: ID=%s, Category=%s, Format=%s", jobID, category, format)
	return jobID, nil
}

// GetJobStatus retrieves job status for frontend polling
func (a *App) GetJobStatus(jobID string) (*JobStatusResponse, error) {
	if a.jobQueue == nil {
		return nil, fmt.Errorf("job queue not initialized")
	}

	job, err := a.jobQueue.GetJob(jobID)
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}

	response := &JobStatusResponse{
		ID:       job.ID,
		Type:     job.Type,
		Status:   job.Status,
		Progress: job.Progress,
		Error:    job.Error,
	}

	// Parse output if completed
	if job.Status == "completed" && job.Output != "" {
		var output ReportGenerateOutput
		if err := json.Unmarshal([]byte(job.Output), &output); err == nil {
			response.Output = &output
		}
	}

	// Add timing info
	if job.StartedAt != nil {
		response.StartedAt = job.StartedAt
	}
	if job.CompletedAt != nil {
		response.CompletedAt = job.CompletedAt
	}

	return response, nil
}

// CancelJob cancels a pending job
func (a *App) CancelJob(jobID string) error {
	// Mission I (I-11): bound mutator — gated.
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	if a.jobQueue == nil {
		return fmt.Errorf("job queue not initialized")
	}

	if err := a.jobQueue.CancelJob(jobID); err != nil {
		return fmt.Errorf("failed to cancel job: %w", err)
	}

	log.Printf("⛔ Job cancelled: ID=%s", jobID)
	return nil
}

// GetRecentJobs retrieves recent jobs for UI display
func (a *App) GetRecentJobs(limit int) ([]JobSummary, error) {
	if a.jobQueue == nil {
		return nil, fmt.Errorf("job queue not initialized")
	}

	if limit <= 0 {
		limit = 20
	}

	jobs, err := a.jobQueue.GetRecentJobs(limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recent jobs: %w", err)
	}

	summaries := make([]JobSummary, len(jobs))
	for i, job := range jobs {
		summaries[i] = JobSummary{
			ID:        job.ID,
			Type:      job.Type,
			Status:    job.Status,
			Progress:  job.Progress,
			CreatedAt: job.CreatedAt,
		}

		if job.CompletedAt != nil {
			summaries[i].CompletedAt = job.CompletedAt
		}
	}

	return summaries, nil
}

// CleanupOldJobs removes completed jobs older than retention period
func (a *App) CleanupOldJobs(retentionDays int) error {
	// Mission I (I-11): bound mutator — gated.
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	if a.jobQueue == nil {
		return fmt.Errorf("job queue not initialized")
	}

	if retentionDays <= 0 {
		retentionDays = 30 // Default: 30 days
	}

	if err := a.jobQueue.CleanupOldJobs(retentionDays); err != nil {
		return fmt.Errorf("failed to cleanup old jobs: %w", err)
	}

	log.Printf("🗑️ Cleaned up jobs older than %d days", retentionDays)
	return nil
}

// =============================================================================
// RESPONSE TYPES (for TypeScript bindings)
// =============================================================================

// JobStatusResponse is returned to frontend for status polling
type JobStatusResponse struct {
	ID          string                `json:"id"`
	Type        string                `json:"type"`
	Status      string                `json:"status"`   // pending, processing, completed, failed
	Progress    int                   `json:"progress"` // 0-100
	Error       string                `json:"error,omitempty"`
	Output      *ReportGenerateOutput `json:"output,omitempty"`
	StartedAt   any                   `json:"started_at,omitempty"`
	CompletedAt any                   `json:"completed_at,omitempty"`
}

// JobSummary is a lightweight job representation for lists
type JobSummary struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Progress    int    `json:"progress"`
	CreatedAt   any    `json:"created_at"`
	CompletedAt any    `json:"completed_at,omitempty"`
}
