package hospitality

// Wave 5 C.2 (stretch) — a minimal print spooler. Kitchen tickets and
// invoices enqueue print jobs at issuance; a worker claims the oldest
// queued job and marks it printed or failed. There is deliberately NO
// printer driver here — the seam is the point: a deployment plugs its
// driver into the claim/mark loop, and everything upstream (what gets
// printed, when, with which payload reference) is already decided.

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Print job kinds and states.
const (
	PrintKindKitchenTicket = "kot"
	PrintKindInvoice       = "invoice"

	PrintQueued  = "queued"
	PrintPrinted = "printed"
	PrintFailed  = "failed"
)

// PrintJob is one spooled document. Target names the physical station
// ("kitchen", "counter"); PayloadRef carries the document identity the
// driver renders from (ticket number / invoice number) — the spooler
// stores references, never rendered bytes.
type PrintJob struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	Kind       string     `gorm:"size:12;not null;index" json:"kind"`
	Target     string     `gorm:"size:20;not null" json:"target"`
	PayloadRef string     `gorm:"size:40;not null" json:"payload_ref"`
	Status     string     `gorm:"size:12;not null;index" json:"status"`
	Error      string     `gorm:"size:300" json:"error"`
	Attempts   int        `gorm:"not null;default:0" json:"attempts"`
	EnqueuedAt time.Time  `json:"enqueued_at"`
	PrintedAt  *time.Time `json:"printed_at"`
}

func (PrintJob) TableName() string { return "hosp_print_jobs" }

// enqueuePrintTx spools one job inside the caller's transaction, so a
// document and its print job commit (or roll back) together.
func (s *Service) enqueuePrintTx(tx *gorm.DB, kind, target, payloadRef string) error {
	job := PrintJob{
		Kind:       kind,
		Target:     target,
		PayloadRef: payloadRef,
		Status:     PrintQueued,
		EnqueuedAt: s.now(),
	}
	return tx.Create(&job).Error
}

// ClaimNextPrintJob hands the worker the oldest queued job for a target
// (empty target = any station). Returns nil when the queue is empty.
func (s *Service) ClaimNextPrintJob(target string) (*PrintJob, error) {
	query := s.db.Where("status = ?", PrintQueued).Order("enqueued_at ASC, id ASC")
	if target != "" {
		query = query.Where("target = ?", target)
	}
	var job PrintJob
	switch err := query.First(&job).Error; {
	case err == nil:
		return &job, nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return nil, nil
	default:
		return nil, err
	}
}

// MarkPrintJobPrinted records a successful print.
func (s *Service) MarkPrintJobPrinted(jobID uint) error {
	now := s.now()
	result := s.db.Model(&PrintJob{}).
		Where("id = ? AND status = ?", jobID, PrintQueued).
		Updates(map[string]any{
			"status":     PrintPrinted,
			"printed_at": &now,
			"attempts":   gorm.Expr("attempts + 1"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("hospitality: print job %d is not queued", jobID)
	}
	return nil
}

// MarkPrintJobFailed records a failed attempt. With requeue the job goes
// back to the queue tail (re-stamped EnqueuedAt) for another try;
// otherwise it lands in failed for operator attention.
func (s *Service) MarkPrintJobFailed(jobID uint, reason string, requeue bool) error {
	status := PrintFailed
	updates := map[string]any{
		"status":   status,
		"error":    reason,
		"attempts": gorm.Expr("attempts + 1"),
	}
	if requeue {
		updates["status"] = PrintQueued
		updates["enqueued_at"] = s.now()
	}
	result := s.db.Model(&PrintJob{}).
		Where("id = ? AND status = ?", jobID, PrintQueued).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("hospitality: print job %d is not queued", jobID)
	}
	return nil
}
