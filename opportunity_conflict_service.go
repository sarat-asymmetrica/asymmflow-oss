package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	opportunityConflictStatusPending  = "pending"
	opportunityConflictStatusApplied  = "applied"
	opportunityConflictStatusRejected = "rejected"
)

type OpportunityEditConflict struct {
	Base
	OpportunityID       string     `gorm:"index;size:36" json:"opportunity_id"`
	FolderNumber        string     `gorm:"index;size:50" json:"folder_number"`
	Operation           string     `gorm:"index;size:50" json:"operation"`
	Status              string     `gorm:"index;size:20;default:'pending'" json:"status"`
	ExpectedVersion     int        `json:"expected_version"`
	CurrentVersion      int        `json:"current_version"`
	AttemptedBy         string     `gorm:"index;size:255" json:"attempted_by"`
	AttemptedRole       string     `gorm:"index;size:100" json:"attempted_role"`
	ProposedChangesJSON string     `gorm:"type:text" json:"proposed_changes_json"`
	CurrentSnapshotJSON string     `gorm:"type:text" json:"current_snapshot_json"`
	BaseSnapshotJSON    string     `gorm:"type:text" json:"base_snapshot_json"`
	ResolutionAction    string     `gorm:"size:30" json:"resolution_action"`
	ResolutionNote      string     `gorm:"type:text" json:"resolution_note"`
	ResolvedBy          string     `gorm:"size:255" json:"resolved_by"`
	ResolvedAt          *time.Time `json:"resolved_at"`
}

func (OpportunityEditConflict) TableName() string { return "opportunity_edit_conflicts" }

type opportunityConflictChange struct {
	Stage      string `json:"stage,omitempty"`
	Comment    string `json:"comment,omitempty"`
	OwnerNotes string `json:"owner_notes,omitempty"`
}

type opportunityConflictSnapshot struct {
	ID           string  `json:"id"`
	FolderNumber string  `json:"folder_number"`
	Title        string  `json:"title"`
	CustomerName string  `json:"customer_name"`
	Stage        string  `json:"stage"`
	Comment      string  `json:"comment"`
	OwnerNotes   string  `json:"owner_notes"`
	RevenueBHD   float64 `json:"revenue_bhd"`
	Version      int     `json:"version"`
	UpdatedAt    string  `json:"updated_at"`
}

type OpportunityConflictResolutionResult struct {
	Conflict    OpportunityEditConflict `json:"conflict"`
	Opportunity Opportunity             `json:"opportunity"`
}

// Mission I (I-11): bound DDL is gated; startup uses the internal.
func (a *App) EnsureOpportunityConflictFoundation() error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	return a.ensureOpportunityConflictFoundationInternal()
}

func (a *App) ensureOpportunityConflictFoundationInternal() error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return a.db.AutoMigrate(&OpportunityEditConflict{})
}

func (a *App) CanResolveOpportunityConflicts() bool {
	return a.currentSessionIsAdministrator()
}

func (a *App) ListOpportunityEditConflicts(status string, limit int) ([]OpportunityEditConflict, error) {
	if !a.currentSessionIsAdministrator() {
		return nil, fmt.Errorf("only administrators can review opportunity edit conflicts")
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" || status == "all" {
		status = ""
	} else if !validOpportunityConflictStatus(status) {
		return nil, fmt.Errorf("invalid conflict status: %s", status)
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}

	query := a.db.Order("created_at DESC").Limit(limit)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var conflicts []OpportunityEditConflict
	if err := query.Find(&conflicts).Error; err != nil {
		return nil, fmt.Errorf("failed to list opportunity conflicts: %w", err)
	}
	return conflicts, nil
}

func (a *App) ResolveOpportunityEditConflict(conflictID, action, note string) (*OpportunityConflictResolutionResult, error) {
	if !a.currentSessionIsAdministrator() {
		return nil, fmt.Errorf("only administrators can resolve opportunity edit conflicts")
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	conflictID = strings.TrimSpace(conflictID)
	action = strings.ToLower(strings.TrimSpace(action))
	note = strings.TrimSpace(note)
	if conflictID == "" {
		return nil, fmt.Errorf("conflict ID is required")
	}
	if action != "apply" && action != "reject" {
		return nil, fmt.Errorf("resolution action must be apply or reject")
	}
	if len(note) > 2000 {
		note = note[:2000]
	}
	resolvedBy := a.getCurrentUserDisplayName()

	var result OpportunityConflictResolutionResult
	err := a.db.Transaction(func(tx *gorm.DB) error {
		var conflict OpportunityEditConflict
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&conflict, "id = ?", conflictID).Error; err != nil {
			return fmt.Errorf("opportunity conflict not found: %w", err)
		}
		if conflict.Status != opportunityConflictStatusPending {
			return fmt.Errorf("opportunity conflict is already %s", conflict.Status)
		}

		var opp Opportunity
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&opp, "id = ?", conflict.OpportunityID).Error; err != nil {
			return fmt.Errorf("opportunity not found: %w", err)
		}

		now := time.Now()
		updates := map[string]any{
			"status":            opportunityConflictStatusRejected,
			"resolution_action": action,
			"resolution_note":   note,
			"resolved_by":       resolvedBy,
			"resolved_at":       &now,
			"updated_at":        now,
		}

		if action == "apply" {
			changes, err := parseOpportunityConflictChanges(conflict.ProposedChangesJSON)
			if err != nil {
				return err
			}
			oppUpdates := map[string]any{}
			if changes.Stage != "" {
				if err := validateOpportunityStageTransition(opp.Stage, changes.Stage); err != nil {
					return err
				}
				oppUpdates["stage"] = changes.Stage
				if changes.Stage == "Won" || changes.Stage == "Lost" {
					oppUpdates["closed_date"] = &now
				}
			}
			if changes.Comment != "" || conflict.Operation == "details_update" {
				oppUpdates["comment"] = trimOpportunityComment(changes.Comment)
			}
			if changes.OwnerNotes != "" || conflict.Operation == "details_update" {
				oppUpdates["owner_notes"] = trimOpportunityComment(changes.OwnerNotes)
			}
			if len(oppUpdates) > 0 {
				oppUpdates["version"] = gorm.Expr("version + ?", 1)
				oppUpdates["updated_at"] = now
				if err := tx.Model(&Opportunity{}).Where("id = ?", opp.ID).Updates(oppUpdates).Error; err != nil {
					return fmt.Errorf("failed to apply opportunity conflict: %w", err)
				}
				if err := tx.First(&opp, "id = ?", opp.ID).Error; err != nil {
					return fmt.Errorf("failed to reload opportunity: %w", err)
				}
			}
			updates["status"] = opportunityConflictStatusApplied
		}

		if err := tx.Model(&OpportunityEditConflict{}).Where("id = ?", conflict.ID).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to resolve opportunity conflict: %w", err)
		}
		if err := tx.First(&conflict, "id = ?", conflict.ID).Error; err != nil {
			return fmt.Errorf("failed to reload opportunity conflict: %w", err)
		}

		result.Conflict = conflict
		result.Opportunity = opp
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *App) UpdateOpportunityStageWithVersion(opportunityID, stage string, expectedVersion int) (*Opportunity, error) {
	if err := a.requirePermission("offers:edit"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	opportunityID = strings.TrimSpace(opportunityID)
	stage = strings.TrimSpace(stage)
	if opportunityID == "" {
		return nil, fmt.Errorf("opportunity ID is required")
	}
	if err := validateOpportunityStageValue(stage); err != nil {
		return nil, err
	}
	attemptedBy := a.getCurrentUserDisplayName()
	attemptedRole := a.GetCurrentUserRole()

	var updated Opportunity
	var conflictErr error
	err := a.db.Transaction(func(tx *gorm.DB) error {
		current, err := loadOpportunityForUpdate(tx, opportunityID)
		if err != nil {
			return err
		}
		if err := validateOpportunityStageTransition(current.Stage, stage); err != nil {
			return err
		}
		if expectedVersion > 0 && current.Version != expectedVersion {
			if err := createOpportunityEditConflict(tx, current, "stage_update", expectedVersion, attemptedBy, attemptedRole, opportunityConflictChange{Stage: stage}); err != nil {
				return err
			}
			conflictErr = fmt.Errorf("opportunity edit conflict detected for %s; admin review required", current.FolderNumber)
			return nil
		}

		updates := map[string]any{
			"stage":      stage,
			"version":    gorm.Expr("version + ?", 1),
			"updated_at": time.Now(),
		}
		if stage == "Won" || stage == "Lost" {
			now := time.Now()
			updates["closed_date"] = &now
		}

		result := tx.Model(&Opportunity{}).Where("id = ?", current.ID).Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("failed to update opportunity stage: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("opportunity not found")
		}
		return tx.First(&updated, "id = ?", current.ID).Error
	})
	if err != nil {
		return nil, err
	}
	if conflictErr != nil {
		return nil, conflictErr
	}
	return &updated, nil
}

func (a *App) UpdateOpportunityDetailsWithVersion(opportunityID string, expectedVersion int, comment, ownerNotes string) (*Opportunity, error) {
	if err := a.requirePermission("offers:edit"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	opportunityID = strings.TrimSpace(opportunityID)
	if opportunityID == "" {
		return nil, fmt.Errorf("opportunity ID is required")
	}

	comment = trimOpportunityComment(comment)
	ownerNotes = trimOpportunityComment(ownerNotes)
	attemptedBy := a.getCurrentUserDisplayName()
	attemptedRole := a.GetCurrentUserRole()

	var updated Opportunity
	var conflictErr error
	err := a.db.Transaction(func(tx *gorm.DB) error {
		current, err := loadOpportunityForUpdate(tx, opportunityID)
		if err != nil {
			return err
		}
		if !a.currentSessionIsManagementOrAbove() {
			ownerNotes = current.OwnerNotes
		}
		if expectedVersion > 0 && current.Version != expectedVersion {
			if err := createOpportunityEditConflict(tx, current, "details_update", expectedVersion, attemptedBy, attemptedRole, opportunityConflictChange{
				Comment:    comment,
				OwnerNotes: ownerNotes,
			}); err != nil {
				return err
			}
			conflictErr = fmt.Errorf("opportunity edit conflict detected for %s; admin review required", current.FolderNumber)
			return nil
		}

		updates := map[string]any{
			"comment":     comment,
			"owner_notes": ownerNotes,
			"version":     gorm.Expr("version + ?", 1),
			"updated_at":  time.Now(),
		}
		result := tx.Model(&Opportunity{}).Where("id = ?", current.ID).Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("failed to update opportunity details: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("opportunity not found")
		}
		return tx.First(&updated, "id = ?", current.ID).Error
	})
	if err != nil {
		return nil, err
	}
	if conflictErr != nil {
		return nil, conflictErr
	}
	return &updated, nil
}

func loadOpportunityForUpdate(tx *gorm.DB, opportunityID string) (Opportunity, error) {
	var current Opportunity
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, "id = ?", opportunityID).Error; err != nil {
		return current, fmt.Errorf("opportunity not found: %w", err)
	}
	return current, nil
}

func createOpportunityEditConflict(tx *gorm.DB, current Opportunity, operation string, expectedVersion int, attemptedBy, attemptedRole string, changes opportunityConflictChange) error {
	proposedJSON, err := json.Marshal(changes)
	if err != nil {
		return fmt.Errorf("failed to encode proposed opportunity changes: %w", err)
	}
	currentJSON, err := json.Marshal(snapshotOpportunityForConflict(current))
	if err != nil {
		return fmt.Errorf("failed to encode current opportunity snapshot: %w", err)
	}

	conflict := OpportunityEditConflict{
		OpportunityID:       current.ID,
		FolderNumber:        current.FolderNumber,
		Operation:           operation,
		Status:              opportunityConflictStatusPending,
		ExpectedVersion:     expectedVersion,
		CurrentVersion:      current.Version,
		AttemptedBy:         strings.TrimSpace(attemptedBy),
		AttemptedRole:       strings.TrimSpace(attemptedRole),
		ProposedChangesJSON: string(proposedJSON),
		CurrentSnapshotJSON: string(currentJSON),
		BaseSnapshotJSON:    fmt.Sprintf(`{"version":%d}`, expectedVersion),
	}
	if conflict.AttemptedBy == "" {
		conflict.AttemptedBy = "unknown"
	}
	if conflict.AttemptedRole == "" {
		conflict.AttemptedRole = "unknown"
	}
	if err := tx.Create(&conflict).Error; err != nil {
		return fmt.Errorf("failed to flag opportunity edit conflict: %w", err)
	}
	return nil
}

func snapshotOpportunityForConflict(opp Opportunity) opportunityConflictSnapshot {
	return opportunityConflictSnapshot{
		ID:           opp.ID,
		FolderNumber: opp.FolderNumber,
		Title:        opp.Title,
		CustomerName: opp.CustomerName,
		Stage:        opp.Stage,
		Comment:      opp.Comment,
		OwnerNotes:   opp.OwnerNotes,
		RevenueBHD:   opp.RevenueBHD,
		Version:      opp.Version,
		UpdatedAt:    opp.UpdatedAt.Format(time.RFC3339),
	}
}

func parseOpportunityConflictChanges(raw string) (opportunityConflictChange, error) {
	var changes opportunityConflictChange
	if err := json.Unmarshal([]byte(raw), &changes); err != nil {
		return changes, fmt.Errorf("failed to parse proposed opportunity changes: %w", err)
	}
	return changes, nil
}

func validateOpportunityStageValue(stage string) error {
	if !isCanonicalOpportunityStage(stage) {
		return fmt.Errorf("invalid stage: %s", stage)
	}
	return nil
}

func validateOpportunityStageTransition(currentStage, nextStage string) error {
	if currentStage == "Lost" && nextStage != "Lost" {
		return fmt.Errorf("invalid transition: lost opportunities cannot be reopened")
	}
	if currentStage == "Won" && nextStage != "Won" && nextStage != "Lost" {
		return fmt.Errorf("invalid transition: won opportunities cannot revert to %s", nextStage)
	}
	return nil
}

func trimOpportunityComment(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 2000 {
		return value[:2000]
	}
	return value
}

func validOpportunityConflictStatus(status string) bool {
	switch status {
	case opportunityConflictStatusPending, opportunityConflictStatusApplied, opportunityConflictStatusRejected:
		return true
	default:
		return false
	}
}
