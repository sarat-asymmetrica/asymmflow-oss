package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	cashflowevidence "ph_holdings_app/pkg/cashflow/evidence"
	"ph_holdings_app/pkg/kernel/approval"

	"gorm.io/gorm"
)

const (
	CashflowProposalStatusPending    = "pending_review"
	CashflowProposalStatusApproved   = "approved"
	CashflowProposalStatusRejected   = "rejected"
	CashflowProposalStatusNeedsInput = "needs_input"
	CashflowProposalStatusSuperseded = "superseded"
)

type CashflowEvidenceProposalReview struct {
	Base
	ProposalKey                  string     `gorm:"uniqueIndex;size:255" json:"proposal_key"`
	Action                       string     `gorm:"index;size:120" json:"action"`
	Label                        string     `gorm:"size:255" json:"label"`
	Reason                       string     `gorm:"type:varchar(1000)" json:"reason"`
	Priority                     string     `gorm:"index;size:20" json:"priority"`
	SourceType                   string     `gorm:"index;size:80" json:"source_type"`
	MutatesState                 bool       `json:"mutates_state"`
	RequiredDeterministicService string     `gorm:"size:160" json:"required_deterministic_service"`
	Status                       string     `gorm:"index;size:30" json:"status"`
	ReviewNote                   string     `gorm:"type:varchar(1000)" json:"review_note"`
	ReviewedBy                   string     `gorm:"size:160" json:"reviewed_by"`
	ReviewedAt                   *time.Time `json:"reviewed_at"`
	WindowLabel                  string     `gorm:"size:80" json:"window_label"`
	WindowStart                  time.Time  `json:"window_start"`
	WindowEnd                    time.Time  `json:"window_end"`
	LastSeenAt                   time.Time  `gorm:"index" json:"last_seen_at"`
}

func (CashflowEvidenceProposalReview) TableName() string {
	return "cashflow_evidence_proposal_reviews"
}

func (a *App) SyncCashflowEvidenceProposalReviews(days int) ([]CashflowEvidenceProposalReview, error) {
	if err := a.requirePermission("finance:update"); err != nil {
		return nil, err
	}
	if err := a.ensureCashflowEvidenceProposalReviewStore(); err != nil {
		return nil, err
	}
	center, err := a.GetCashflowEvidenceCommandCenter(days)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	keys := make([]string, 0, len(center.ActionProposals))
	for _, proposal := range center.ActionProposals {
		key := cashflowevidence.ProposalReviewKey(proposal)
		if strings.TrimSpace(key) == "" {
			continue
		}
		keys = append(keys, key)
		row := CashflowEvidenceProposalReview{
			ProposalKey:                  key,
			Action:                       proposal.Action,
			Label:                        proposal.Label,
			Reason:                       proposal.Reason,
			Priority:                     string(proposal.Priority),
			SourceType:                   proposal.SourceType,
			MutatesState:                 proposal.MutatesState,
			RequiredDeterministicService: proposal.RequiredDeterministicService,
			Status:                       CashflowProposalStatusPending,
			WindowLabel:                  center.Window.Label,
			WindowStart:                  center.Window.Start,
			WindowEnd:                    center.Window.End,
			LastSeenAt:                   now,
		}
		if err := a.upsertCashflowEvidenceProposalReview(row); err != nil {
			return nil, err
		}
	}

	return a.listCashflowEvidenceProposalReviews(keys, false)
}

func (a *App) ListCashflowEvidenceProposalReviews(days int, includeResolved bool) ([]CashflowEvidenceProposalReview, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	if err := a.ensureCashflowEvidenceProposalReviewStore(); err != nil {
		return nil, err
	}
	center, err := a.GetCashflowEvidenceCommandCenter(days)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(center.ActionProposals))
	for _, proposal := range center.ActionProposals {
		key := cashflowevidence.ProposalReviewKey(proposal)
		if strings.TrimSpace(key) != "" {
			keys = append(keys, key)
		}
	}
	return a.listCashflowEvidenceProposalReviews(keys, includeResolved)
}

func (a *App) ReviewCashflowEvidenceProposal(proposalReviewID string, status string, note string) (CashflowEvidenceProposalReview, error) {
	if err := a.requirePermission("finance:update"); err != nil {
		return CashflowEvidenceProposalReview{}, err
	}
	if err := a.ensureCashflowEvidenceProposalReviewStore(); err != nil {
		return CashflowEvidenceProposalReview{}, err
	}
	status = normalizeCashflowProposalReviewStatus(status)
	if status == "" {
		return CashflowEvidenceProposalReview{}, fmt.Errorf("unsupported cashflow proposal review status")
	}
	var row CashflowEvidenceProposalReview
	if err := a.db.First(&row, "id = ?", strings.TrimSpace(proposalReviewID)).Error; err != nil {
		return CashflowEvidenceProposalReview{}, err
	}

	if !validCashflowProposalTransition(row.Status, status) {
		return CashflowEvidenceProposalReview{}, fmt.Errorf(
			"invalid proposal review transition from %q to %q", row.Status, status)
	}

	now := time.Now().UTC()
	row.Status = status
	row.ReviewNote = strings.TrimSpace(note)
	if status == CashflowProposalStatusPending {
		row.ReviewedAt = nil
		row.ReviewedBy = ""
	} else {
		row.ReviewedAt = &now
		row.ReviewedBy = a.getCurrentUserDisplayName()
	}
	if err := a.db.Save(&row).Error; err != nil {
		return CashflowEvidenceProposalReview{}, err
	}
	return row, nil
}

func (a *App) ensureCashflowEvidenceProposalReviewStore() error {
	if a == nil || a.db == nil {
		return fmt.Errorf("database connection not available")
	}
	return a.db.AutoMigrate(&CashflowEvidenceProposalReview{})
}

func (a *App) upsertCashflowEvidenceProposalReview(next CashflowEvidenceProposalReview) error {
	var existing CashflowEvidenceProposalReview
	err := a.db.First(&existing, "proposal_key = ?", next.ProposalKey).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return a.db.Create(&next).Error
	}
	if err != nil {
		return err
	}
	updates := map[string]any{
		"action":                         next.Action,
		"label":                          next.Label,
		"reason":                         next.Reason,
		"priority":                       next.Priority,
		"source_type":                    next.SourceType,
		"mutates_state":                  next.MutatesState,
		"required_deterministic_service": next.RequiredDeterministicService,
		"window_label":                   next.WindowLabel,
		"window_start":                   next.WindowStart,
		"window_end":                     next.WindowEnd,
		"last_seen_at":                   next.LastSeenAt,
	}
	if existing.Status == "" {
		updates["status"] = CashflowProposalStatusPending
	}
	return a.db.Model(&existing).Updates(updates).Error
}

func (a *App) listCashflowEvidenceProposalReviews(keys []string, includeResolved bool) ([]CashflowEvidenceProposalReview, error) {
	if len(keys) == 0 {
		return []CashflowEvidenceProposalReview{}, nil
	}
	query := a.db.Where("proposal_key IN ?", keys)
	if !includeResolved {
		query = query.Where("status NOT IN ?", []string{CashflowProposalStatusRejected, CashflowProposalStatusSuperseded})
	}
	var rows []CashflowEvidenceProposalReview
	if err := query.Order("last_seen_at DESC, priority DESC, updated_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func validCashflowProposalTransition(from, to string) bool {
	return approval.ValidTransition(approval.Decision(from), approval.Decision(to))
}

func normalizeCashflowProposalReviewStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "pending", "pending_review":
		return CashflowProposalStatusPending
	case "approve", "approved":
		return CashflowProposalStatusApproved
	case "reject", "rejected":
		return CashflowProposalStatusRejected
	case "needs_input", "needs input", "input":
		return CashflowProposalStatusNeedsInput
	case "superseded":
		return CashflowProposalStatusSuperseded
	default:
		return ""
	}
}
