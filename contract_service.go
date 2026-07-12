// ═══════════════════════════════════════════════════════════════════════════
// CONTRACT GENERATION SERVICE — root façade
//
// Wave 5 A.1: the contract generation body (grade-based clause selection,
// numbering, PDF rendering, seeds) lives in pkg/crm/contract. These aliases
// keep the table shapes, JSON contracts, model registry, and every root
// call site unchanged.
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	crmcontract "ph_holdings_app/pkg/crm/contract"
	"ph_holdings_app/pkg/kernel/text"

	"gorm.io/gorm"
)

// ContractService handles contract generation and management
type ContractService = crmcontract.Service

// NewContractService creates a new contract service
func NewContractService(db *gorm.DB) *ContractService {
	return crmcontract.New(db)
}

// ContractTemplate defines predefined contract templates
type ContractTemplate = crmcontract.Template

// ContractClause defines individual clauses that can be included in contracts
type ContractClause = crmcontract.Clause

// Contract represents a generated contract document
type Contract = crmcontract.Contract

// ContractGenerationRequest represents request to generate a contract
type ContractGenerationRequest = crmcontract.GenerationRequest

// ContractClauseSelection represents selected clauses for a contract
type ContractClauseSelection = crmcontract.ClauseSelection

// wrapText wraps text to the specified width. Canonical implementation:
// pkg/kernel/text.Wrap (Wave 5); the costing export surfaces still call
// this root name.
func wrapText(s string, width int) []string {
	return text.Wrap(s, width)
}
