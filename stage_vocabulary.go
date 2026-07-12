package main

import (
	"fmt"
	"log"
	"sort"
	"strings"
)

// canonicalOpportunityStages is the single source of truth for the
// Opportunity/RFQ pipeline-stage vocabulary. Terminal stages that feed the
// dashboard financial aggregates (win rate, pipeline value) are "Won",
// "Lost", and "Expired" — everything else is treated as active pipeline.
//
// Offer.Stage is intentionally NOT unified with this enum: it is a separate,
// DB-level CHECK-constrained vocabulary ('RFQ','Quoted','Won','Lost','Expired')
// defined in pkg/crm/domain.go and must not be changed here.
var canonicalOpportunityStages = []string{
	"New", "Qualified", "Proposal", "Quoted", "Won", "Lost", "Expired", "On Hold",
}

// legacyOpportunityStageMap is the owner-ratified historical migration map
// from legacy/ad-hoc Opportunity/RFQ stage vocabulary to the canonical enum.
// Values already canonical, or values not present in this map, are left
// alone by canonicalizeOpportunityStage (caller decides how to handle
// unrecognized values).
var legacyOpportunityStageMap = map[string]string{
	"RFQ Received":     "New",
	"Costing":          "Proposal",
	"Tender":           "Proposal",
	"Offer Sent":       "Quoted",
	"Follow-up/Eval":   "Quoted",
	"PO/LOI Received":  "Won",
	"Order Placed":     "Won",
	"In Process":       "Won",
	"Delivered":        "Won",
	"Closed (Payment)": "Won",
	"Closed (Lost)":    "Lost",
	"In Progress":      "Proposal",
}

// isCanonicalOpportunityStage reports whether s is exactly one of the 8
// canonical Opportunity/RFQ pipeline stages.
func isCanonicalOpportunityStage(s string) bool {
	for _, c := range canonicalOpportunityStages {
		if c == s {
			return true
		}
	}
	return false
}

// canonicalizeOpportunityStage trims raw and applies the ratified migration
// map. Already-canonical values are returned unchanged with mapped=false.
// Empty (after trim) maps to "New" with mapped=true. Unrecognized values are
// returned UNCHANGED with mapped=false — callers decide whether to reject,
// coerce, or log them; this function never guesses.
func canonicalizeOpportunityStage(raw string) (canonical string, mapped bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "New", true
	}
	if isCanonicalOpportunityStage(trimmed) {
		return trimmed, false
	}
	if target, ok := legacyOpportunityStageMap[trimmed]; ok {
		return target, true
	}
	return trimmed, false
}

// migrateOpportunityStageVocabulary is the idempotent historical migration
// that rewrites legacy Opportunity.Stage / RFQData.Stage values to the
// canonical enum in existing (already-populated) databases. It is safe to
// run on every boot: once a row's stage has been rewritten to a canonical
// value, none of the legacy WHERE clauses match it again, so a second run
// affects 0 rows.
func (a *App) migrateOpportunityStageVocabulary() error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	type migrationTarget struct {
		label string
		model any
	}
	targets := []migrationTarget{
		{"opportunities", &Opportunity{}},
		{"rfq_data", &RFQData{}},
	}

	// Deterministic iteration order for stable, readable logs.
	legacyKeys := make([]string, 0, len(legacyOpportunityStageMap))
	for k := range legacyOpportunityStageMap {
		legacyKeys = append(legacyKeys, k)
	}
	sort.Strings(legacyKeys)

	for _, target := range targets {
		for _, legacy := range legacyKeys {
			canonical := legacyOpportunityStageMap[legacy]
			result := a.db.Model(target.model).Where("stage = ?", legacy).Update("stage", canonical)
			if result.Error != nil {
				log.Printf("⚠️ stage migration: %s %q→%q failed: %v", target.label, legacy, canonical, result.Error)
				continue
			}
			if result.RowsAffected > 0 {
				log.Printf("stage migration: %s %q→%q: %d rows", target.label, legacy, canonical, result.RowsAffected)
			}
		}

		// Ratified: empty/NULL stage → "New".
		emptyResult := a.db.Model(target.model).Where("stage = ? OR stage IS NULL", "").Update("stage", "New")
		if emptyResult.Error != nil {
			log.Printf("⚠️ stage migration: %s \"\"→\"New\" failed: %v", target.label, emptyResult.Error)
		} else if emptyResult.RowsAffected > 0 {
			log.Printf("stage migration: %s \"\"→\"New\": %d rows", target.label, emptyResult.RowsAffected)
		}

		// Surface any residual non-canonical values left behind (unmapped
		// legacy stages, e.g. bare "Closed") so the owner can see them.
		// These are deliberately left UNCHANGED — we never guess.
		var residuals []string
		if err := a.db.Model(target.model).
			Where("stage NOT IN ?", canonicalOpportunityStages).
			Distinct().
			Pluck("stage", &residuals).Error; err != nil {
			log.Printf("⚠️ stage migration: %s residual scan failed: %v", target.label, err)
		} else if len(residuals) > 0 {
			log.Printf("⚠️ stage migration: %s has %d unmapped residual stage value(s), left unchanged: %v", target.label, len(residuals), residuals)
		}
	}

	return nil
}
