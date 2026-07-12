package main

// Wave 8 P3 slice 3: customer data-quality review ledger (Bucket F).
//
// Ported from ph_holdings/user_feedback_hardening_service.go (the
// data_quality_reviews surface, methods 693–942). This is an admin
// data-hygiene queue: PreviewCustomerDataQuality computes issues live over
// customers/opportunities/offers, ReviewDataQualityIssue dispositions one
// (admin-only), and GetDataQualityReviewHistory returns the audit trail.
//
// OSS adaptation (per docs/MISSION_I_DEFERRED_MODEL_SPECS.md §2): PH
// self-provisions the table at call time via ensureDataQualityReviewFoundation
// (a raw CREATE TABLE + ensureSyncBaseColumns loop). ensureSyncBaseColumns does
// not exist in OSS, and the sovereign substrate migrates schema through
// tradingModels() + AutoMigrate with a pinned golden. So the self-migration is
// dropped entirely: &DataQualityReview{} is registered in tradingModels(), and
// the three ensureDataQualityReviewFoundation() call sites are removed. No other
// divergence from PH — the issue scan and review upsert are faithful.

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

// DataQualityIssue is the transient (non-persisted) shape the preview returns
// and the review consumes. It is a computed view over the scanned entities,
// overlaid with any existing review disposition.
type DataQualityIssue struct {
	ID            string `json:"id"`
	Severity      string `json:"severity"`
	IssueType     string `json:"issue_type"`
	EntityType    string `json:"entity_type"`
	EntityID      string `json:"entity_id"`
	Summary       string `json:"summary"`
	Detail        string `json:"detail"`
	PrimaryAction string `json:"primary_action"`
	ReviewStatus  string `json:"review_status"`
	ReviewNote    string `json:"review_note"`
	ReviewedBy    string `json:"reviewed_by"`
	ReviewedAt    string `json:"reviewed_at"`
}

// DataQualityReview is the persisted disposition of a computed issue. It is a
// self-contained ledger: entity_type + entity_id are loose pointers, no FK.
type DataQualityReview struct {
	Base
	IssueID       string     `gorm:"uniqueIndex;size:180" json:"issue_id"`
	IssueType     string     `gorm:"index;size:80" json:"issue_type"`
	Severity      string     `gorm:"index;size:40" json:"severity"`
	EntityType    string     `gorm:"index;size:80" json:"entity_type"`
	EntityID      string     `gorm:"index;size:100" json:"entity_id"`
	Summary       string     `gorm:"size:500" json:"summary"`
	Detail        string     `gorm:"type:text" json:"detail"`
	PrimaryAction string     `gorm:"size:255" json:"primary_action"`
	Status        string     `gorm:"index;size:40" json:"status"`
	ReviewNote    string     `gorm:"type:text" json:"review_note"`
	ReviewedByID  string     `gorm:"index;size:100" json:"reviewed_by_id"`
	ReviewedBy    string     `gorm:"size:255" json:"reviewed_by"`
	ReviewedAt    *time.Time `json:"reviewed_at"`
}

func (DataQualityReview) TableName() string { return "data_quality_reviews" }

// PreviewCustomerDataQuality computes data-hygiene issues live over customers,
// opportunities, and offers, overlays any existing review disposition, and
// suppresses issues already resolved/dismissed from the queue. Read-only.
func (a *App) PreviewCustomerDataQuality(limit int) ([]DataQualityIssue, error) {
	if err := a.requirePermission("customers:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	reviews, err := a.dataQualityReviewsByIssueID()
	if err != nil {
		return nil, err
	}
	issues := make([]DataQualityIssue, 0)
	appendIssue := func(issue DataQualityIssue) {
		if review, ok := reviews[issue.ID]; ok {
			switch strings.ToLower(strings.TrimSpace(review.Status)) {
			case "dismissed", "resolved":
				return
			default:
				issue.ReviewStatus = review.Status
				issue.ReviewNote = review.ReviewNote
				issue.ReviewedBy = review.ReviewedBy
				if review.ReviewedAt != nil {
					issue.ReviewedAt = review.ReviewedAt.Format(time.RFC3339)
				}
			}
		}
		if len(issues) < limit {
			issues = append(issues, issue)
		}
	}

	var customers []CustomerMaster
	_ = a.db.Where("deleted_at IS NULL").Find(&customers).Error
	byName := map[string][]CustomerMaster{}
	for _, customer := range customers {
		key := normalizeDataQualityName(customer.BusinessName)
		if key == "" {
			appendIssue(DataQualityIssue{ID: "customer-blank-" + customer.ID, Severity: "high", IssueType: "blank_customer_name", EntityType: "customer", EntityID: customer.ID, Summary: "Customer has no business name", Detail: customer.CustomerID, PrimaryAction: "Complete customer name"})
			continue
		}
		byName[key] = append(byName[key], customer)
	}
	for _, group := range byName {
		if len(group) < 2 {
			continue
		}
		names := make([]string, 0, len(group))
		for _, customer := range group {
			names = append(names, customer.BusinessName)
		}
		appendIssue(DataQualityIssue{ID: "customer-duplicate-" + group[0].ID, Severity: "medium", IssueType: "duplicate_customer", EntityType: "customer", EntityID: group[0].ID, Summary: "Possible duplicate customer records", Detail: strings.Join(names, " | "), PrimaryAction: "Review merge candidates"})
	}

	var opps []Opportunity
	_ = a.db.Limit(limit).Find(&opps).Error
	for _, opp := range opps {
		if strings.TrimSpace(opp.Title) == "" && strings.TrimSpace(opp.FolderName) == "" {
			appendIssue(DataQualityIssue{ID: "opp-blank-title-" + opp.ID, Severity: "medium", IssueType: "blank_opportunity_name", EntityType: "opportunity", EntityID: opp.ID, Summary: "Opportunity has no clear name", Detail: firstNonEmpty(opp.FolderNumber, opp.CustomerName), PrimaryAction: "Add project/opportunity title"})
		}
		if strings.TrimSpace(opp.CustomerID) == "" || strings.TrimSpace(opp.CustomerName) == "" {
			appendIssue(DataQualityIssue{ID: "opp-missing-customer-" + opp.ID, Severity: "high", IssueType: "missing_customer_link", EntityType: "opportunity", EntityID: opp.ID, Summary: "Opportunity is missing a customer link", Detail: firstNonEmpty(opp.FolderNumber, opp.Title), PrimaryAction: "Assign correct customer"})
		}
	}

	var offers []Offer
	_ = a.db.Where("customer_id = '' OR customer_name = ''").Limit(limit).Find(&offers).Error
	for _, offer := range offers {
		appendIssue(DataQualityIssue{ID: "offer-missing-customer-" + offer.ID, Severity: "high", IssueType: "offer_missing_customer", EntityType: "offer", EntityID: offer.ID, Summary: "Offer is missing customer data", Detail: firstNonEmpty(offer.OfferNumber, offer.CustomerReference), PrimaryAction: "Reassign offer customer"})
	}

	return issues, nil
}

// ReviewDataQualityIssue dispositions a computed issue (reviewed/resolved/
// dismissed) as an upsert keyed on issue_id. Admin-only — not a plain
// permission, since it suppresses items from every reviewer's queue.
func (a *App) ReviewDataQualityIssue(issue DataQualityIssue, action string, note string) (DataQualityReview, error) {
	if !a.currentSessionHasAdminRoleOnly() {
		return DataQualityReview{}, fmt.Errorf("data quality review requires admin permission")
	}
	if a.db == nil {
		return DataQualityReview{}, fmt.Errorf("database not initialized")
	}

	issue.ID = strings.TrimSpace(issue.ID)
	if issue.ID == "" {
		return DataQualityReview{}, fmt.Errorf("issue id is required")
	}
	status := strings.ToLower(strings.TrimSpace(action))
	switch status {
	case "reviewed", "resolved", "dismissed":
	default:
		return DataQualityReview{}, fmt.Errorf("unsupported data quality review action %q", action)
	}

	now := time.Now()
	reviewerID := strings.TrimSpace(a.getCurrentUserID())
	reviewerName := firstNonEmpty(a.getCurrentUserDisplayName(), reviewerID, "Admin")
	var review DataQualityReview
	err := a.db.Where("issue_id = ?", issue.ID).First(&review).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return DataQualityReview{}, fmt.Errorf("failed to load data quality review: %w", err)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		review = DataQualityReview{IssueID: issue.ID}
	}

	review.IssueType = strings.TrimSpace(issue.IssueType)
	review.Severity = strings.TrimSpace(issue.Severity)
	review.EntityType = strings.TrimSpace(issue.EntityType)
	review.EntityID = strings.TrimSpace(issue.EntityID)
	review.Summary = trimToLength(issue.Summary, 500)
	review.Detail = strings.TrimSpace(issue.Detail)
	review.PrimaryAction = trimToLength(issue.PrimaryAction, 255)
	review.Status = status
	review.ReviewNote = strings.TrimSpace(note)
	review.ReviewedByID = reviewerID
	review.ReviewedBy = reviewerName
	review.ReviewedAt = &now
	review.CreatedBy = firstNonEmpty(review.CreatedBy, reviewerID)

	if err := a.db.Save(&review).Error; err != nil {
		return DataQualityReview{}, fmt.Errorf("failed to save data quality review: %w", err)
	}
	a.logAudit(&reviewerID, "data_quality_"+status, "data_quality_issue", &review.IssueID, review.ReviewNote)
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "data-quality:updated", map[string]any{"issue_id": review.IssueID, "status": review.Status})
	}
	return review, nil
}

// GetDataQualityReviewHistory returns recent review dispositions, newest first.
func (a *App) GetDataQualityReviewHistory(limit int) ([]DataQualityReview, error) {
	if err := a.requirePermission("customers:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var reviews []DataQualityReview
	if err := a.db.Order("updated_at DESC, created_at DESC").Limit(limit).Find(&reviews).Error; err != nil {
		return nil, fmt.Errorf("failed to load data quality review history: %w", err)
	}
	return reviews, nil
}

// dataQualityReviewsByIssueID maps issue_id → review for the preview overlay.
func (a *App) dataQualityReviewsByIssueID() (map[string]DataQualityReview, error) {
	var reviews []DataQualityReview
	if err := a.db.Find(&reviews).Error; err != nil {
		return nil, fmt.Errorf("failed to load data quality reviews: %w", err)
	}
	out := make(map[string]DataQualityReview, len(reviews))
	for _, review := range reviews {
		out[review.IssueID] = review
	}
	return out, nil
}

// trimToLength clamps a trimmed string to maxLen bytes.
func trimToLength(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	if maxLen <= 0 || len(value) <= maxLen {
		return value
	}
	return value[:maxLen]
}

// normalizeDataQualityName is the dedup key: lowercase, punctuation-stripped,
// with common Bahrain company suffixes removed (wll / bsc / company / co) so
// name variations of the same entity collide. Domain context, not client data.
func normalizeDataQualityName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(".", "", ",", "", "(", "", ")", "", "-", " ", "_", " ")
	value = replacer.Replace(value)
	for _, suffix := range []string{" wll", " w l l", " bsc", " b s c", " company", " co"} {
		value = strings.TrimSuffix(value, suffix)
	}
	return strings.Join(strings.Fields(value), " ")
}
