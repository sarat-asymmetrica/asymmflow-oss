package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"ph_holdings_app/pkg/data"
	"ph_holdings_app/pkg/engines"
	"ph_holdings_app/pkg/graph"
)

func (a *App) GenerateContract(customerID string, templateName, contractType string, valueBHD float64, orderID string) (*Contract, error) {
	if err := a.requirePermission("contracts:create"); err != nil {
		return nil, err
	}
	contractService := NewContractService(a.db)

	req := ContractGenerationRequest{
		CustomerID:   customerID,
		TemplateName: templateName,
		ContractType: contractType,
		ValueBHD:     valueBHD,
		OrderID:      orderID,
	}

	contract, err := contractService.GenerateContract(req)
	if err != nil {
		log.Printf("❌ Contract generation failed: %v", err)
		return nil, err
	}

	log.Printf("✅ Contract generated: %s for customer ID %s", contract.ContractNo, customerID)
	return contract, nil
}

// GetContracts retrieves all contracts (paginated)
func (a *App) GetContracts(limit, offset int) ([]Contract, error) {
	if err := a.requirePermission("contracts:view"); err != nil {
		return nil, err
	}
	var contracts []Contract
	err := a.db.Preload("Customer").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&contracts).Error

	if err != nil {
		return nil, err
	}

	return contracts, nil
}

// GetContractsByCustomer retrieves all contracts for a specific customer
func (a *App) GetContractsByCustomer(customerID uint) ([]Contract, error) {
	if err := a.requirePermission("contracts:view"); err != nil {
		return nil, err
	}
	var contracts []Contract
	err := a.db.Where("customer_id = ?", customerID).
		Order("created_at DESC").
		Find(&contracts).Error

	if err != nil {
		return nil, err
	}

	return contracts, nil
}

// GetContract retrieves a single contract by ID
func (a *App) GetContract(contractID uint) (*Contract, error) {
	if err := a.requirePermission("contracts:view"); err != nil {
		return nil, err
	}
	var contract Contract
	err := a.db.Preload("Customer").
		First(&contract, contractID).Error

	if err != nil {
		return nil, err
	}

	return &contract, nil
}

// GetContractTemplates retrieves available contract templates
func (a *App) GetContractTemplates() ([]ContractTemplate, error) {
	if err := a.requirePermission("contracts:view"); err != nil {
		return nil, err
	}
	var templates []ContractTemplate
	err := a.db.Where("is_active = ?", true).
		Order("name ASC").
		Find(&templates).Error

	if err != nil {
		return nil, err
	}

	return templates, nil
}

// SeedContractData seeds contract templates and clauses
func (a *App) SeedContractData() error {
	// SECURITY: Admin-only permission for seed functions
	if err := a.requirePermission("*"); err != nil {
		return err
	}
	contractService := NewContractService(a.db)

	// Seed templates
	if err := contractService.SeedContractTemplates(); err != nil {
		return fmt.Errorf("failed to seed templates: %v", err)
	}

	// Seed clauses
	if err := contractService.SeedContractClauses(); err != nil {
		return fmt.Errorf("failed to seed clauses: %v", err)
	}

	log.Println("✅ Contract templates and clauses seeded successfully")
	return nil
}

// DownloadContract returns the PDF path for downloading
func (a *App) DownloadContract(contractID uint) (string, error) {
	if err := a.requirePermission("contracts:view"); err != nil {
		return "", err
	}
	var contract Contract
	err := a.db.First(&contract, contractID).Error
	if err != nil {
		return "", fmt.Errorf("contract not found: %v", err)
	}

	if contract.PDFPath == "" {
		return "", fmt.Errorf("PDF not generated for this contract")
	}

	// Verify file exists
	if _, err := os.Stat(contract.PDFPath); os.IsNotExist(err) {
		return "", fmt.Errorf("PDF file not found: %s", contract.PDFPath)
	}

	return contract.PDFPath, nil
}

// =============================================================================
// SSOT DATA IMPORT
// =============================================================================

// ImportSSOTData imports all Single Source of Truth data from data/ssot
//
// PURPOSE:
// Import critical business data into the database:
// - Customer master data from CSV (311 customers)
// - Opportunities from Excel
// - Supplier payments from Excel
// - Product costing from Excel
//
// PERFORMANCE:
// Uses Williams batching (sqrt(n) * log2(n)) for optimal batch sizes
// Typical import time: ~2-5 seconds for full dataset
//
// IDEMPOTENCY:
// Can be run multiple times safely - duplicates are skipped
//
// RETURNS:
// Detailed statistics including:
// - Total records processed per category
// - Successful imports
// - Skipped duplicates
// - Errors encountered
func (a *App) ImportSSOTData() (*data.ImportResult, error) {
	// SECURITY: Admin-only permission for data import
	if err := a.requirePermission("*"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Get project root path
	paths := a.getAppPaths()
	dataDir := filepath.Join(paths.ProjectRoot, "data/ssot")

	// Verify directory exists
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("SSOT data directory not found: %s", dataDir)
	}

	log.Printf("🌟 Starting SSOT data import from: %s", dataDir)

	// Execute import
	result, err := data.ImportAllSSOT(a.db, dataDir)
	if err != nil {
		log.Printf("❌ SSOT import failed: %v", err)
		return result, err
	}

	// Log statistics
	log.Printf("✅ SSOT import completed in %v:", result.Duration)
	log.Printf("   📊 Customers: %d/%d imported (%d skipped)",
		result.CustomersImported, result.CustomersTotal, result.CustomersSkipped)
	log.Printf("   💼 Opportunities: %d/%d imported (%d skipped)",
		result.OpportunitiesImported, result.OpportunitiesTotal, result.OpportunitiesSkipped)
	log.Printf("   💰 Payments: %d/%d imported (%d skipped)",
		result.PaymentsImported, result.PaymentsTotal, result.PaymentsSkipped)
	log.Printf("   📦 Products: %d/%d imported (%d skipped)",
		result.ProductsImported, result.ProductsTotal, result.ProductsSkipped)

	if len(result.CustomerErrors) > 0 {
		log.Printf("   ⚠️  Customer errors: %d", len(result.CustomerErrors))
	}
	if len(result.OpportunityErrors) > 0 {
		log.Printf("   ⚠️  Opportunity errors: %d", len(result.OpportunityErrors))
	}
	if len(result.PaymentErrors) > 0 {
		log.Printf("   ⚠️  Payment errors: %d", len(result.PaymentErrors))
	}
	if len(result.ProductErrors) > 0 {
		log.Printf("   ⚠️  Product errors: %d", len(result.ProductErrors))
	}

	return result, nil
}

// GetSSOTImportStatus returns a user-friendly summary of the last import
func (a *App) GetSSOTImportStatus() map[string]any {
	if err := a.requirePermission("settings:view"); err != nil {
		return map[string]any{
			"status":  "error",
			"message": err.Error(),
		}
	}
	if a.db == nil {
		return map[string]any{
			"status":  "error",
			"message": "Database not initialized",
		}
	}

	// Count current records in each table
	var customerCount int64
	var opportunityCount int64
	var paymentCount int64
	var productCount int64

	a.db.Model(&data.CustomerMaster{}).Count(&customerCount)
	a.db.Model(&data.OpportunitySSOT{}).Count(&opportunityCount)
	a.db.Model(&data.PaymentSSOT{}).Count(&paymentCount)
	a.db.Model(&data.ProductCostingSSOT{}).Count(&productCount)

	return map[string]any{
		"status": "ready",
		"counts": map[string]int64{
			"customers":     customerCount,
			"opportunities": opportunityCount,
			"payments":      paymentCount,
			"products":      productCount,
		},
		"total_records": customerCount + opportunityCount + paymentCount + productCount,
	}
}

// =============================================================================
// ENTITY GRAPH BINDINGS (Customer360)
// =============================================================================

// GetEntityGraph retrieves entities of a specific type with their relationships
func (a *App) GetEntityGraph(nodeType string, limit int) (*graph.GraphData, error) {
	if err := a.requirePermission("customers:view"); err != nil {
		return nil, err
	}
	if a.graphService == nil {
		return nil, fmt.Errorf("graph service not initialized")
	}
	return a.graphService.GetEntityGraph(nodeType, limit)
}

// GetCustomerGraph retrieves all relationships for a specific customer
func (a *App) GetCustomerGraph(customerID string, depth int) (*graph.GraphData, error) {
	if err := a.requirePermission("customers:view"); err != nil {
		return nil, err
	}
	if a.graphService == nil {
		return nil, fmt.Errorf("graph service not initialized")
	}
	return a.graphService.GetCustomerGraph(customerID, depth)
}

// GetNodeRelationships retrieves immediate relationships for a node
func (a *App) GetNodeRelationships(nodeID uint) ([]graph.GraphEdge, error) {
	if err := a.requirePermission("customers:view"); err != nil {
		return nil, err
	}
	if a.graphService == nil {
		return nil, fmt.Errorf("graph service not initialized")
	}
	return a.graphService.GetNodeRelationships(nodeID)
}

// SearchGraphEntities performs full-text search across all graph nodes
func (a *App) SearchGraphEntities(query string, limit int) ([]graph.GraphNode, error) {
	if err := a.requirePermission("customers:view"); err != nil {
		return nil, err
	}
	if a.graphService == nil {
		return nil, fmt.Errorf("graph service not initialized")
	}
	return a.graphService.SearchEntities(query, limit)
}

// GetGraphStats returns statistics about the entity graph
func (a *App) GetGraphStats() (*graph.GraphStats, error) {
	if err := a.requirePermission("customers:view"); err != nil {
		return nil, err
	}
	if a.graphService == nil {
		return nil, fmt.Errorf("graph service not initialized")
	}
	return a.graphService.GetGraphStats()
}

// BuildEntityGraph builds the entity graph from SSOT tables
func (a *App) BuildEntityGraph() (*graph.BuildStats, error) {
	if err := a.requirePermission("settings:update"); err != nil {
		return nil, err
	}
	if a.graphService == nil {
		return nil, fmt.Errorf("graph service not initialized")
	}

	log.Println("🔨 Building entity graph from SSOT...")
	builder := graph.NewSSOTBuilder(a.db)
	stats, err := builder.BuildGraph()
	if err != nil {
		return nil, fmt.Errorf("failed to build graph: %w", err)
	}

	log.Printf("✅ Graph build complete: %d nodes, %d edges", stats.NodesCreated, stats.EdgesCreated)
	return stats, nil
}

// RebuildEntityGraph clears and rebuilds the entire entity graph
func (a *App) RebuildEntityGraph() (*graph.BuildStats, error) {
	if err := a.requirePermission("settings:update"); err != nil {
		return nil, err
	}
	if a.graphService == nil {
		return nil, fmt.Errorf("graph service not initialized")
	}

	log.Println("🔄 Rebuilding entity graph...")
	builder := graph.NewSSOTBuilder(a.db)
	stats, err := builder.RebuildGraph()
	if err != nil {
		return nil, fmt.Errorf("failed to rebuild graph: %w", err)
	}

	log.Printf("✅ Graph rebuild complete: %d nodes, %d edges", stats.NodesCreated, stats.EdgesCreated)
	return stats, nil
}

// ExportGraphJSON exports the entire graph as JSON (for D3.js visualization)
func (a *App) ExportGraphJSON() (string, error) {
	if err := a.requirePermission("customers:view"); err != nil {
		return "", err
	}
	if a.graphService == nil {
		return "", fmt.Errorf("graph service not initialized")
	}

	jsonBytes, err := a.graphService.ExportGraphJSON()
	if err != nil {
		return "", fmt.Errorf("failed to export graph: %w", err)
	}

	return string(jsonBytes), nil
}

// ═══════════════════════════════════════════════════════════════════════════
// WILLIAMS OPTIMIZATION METRICS
// ═══════════════════════════════════════════════════════════════════════════

// GetWilliamsMetrics calculates Williams batching optimization metrics
// Formula: O(√n × log₂n) space complexity
// Proof: asymmetrica_proofs/AsymmetricaProofs/WilliamsBatching.lean
func (a *App) GetWilliamsMetrics(totalItems int) engines.WilliamsMetrics {
	return engines.GetWilliamsMetrics(totalItems)
}

// CompareWilliamsLinear compares Williams optimization vs linear approach
func (a *App) CompareWilliamsLinear(totalItems int) map[string]any {
	optimizer := engines.NewWilliamsOptimizer()
	return optimizer.CompareWithLinear(totalItems)
}

// =============================================================================
// ROLE-BASED ACCESS CONTROL (RBAC) API
// =============================================================================

// Permission constants - fine-grained access control
const (
	// Customer Management
	PermCustomersView   = "customers:view"
	PermCustomersEdit   = "customers:edit"
	PermCustomersDelete = "customers:delete"

	// Supplier Management
	PermSuppliersView   = "suppliers:view"
	PermSuppliersEdit   = "suppliers:edit"
	PermSuppliersDelete = "suppliers:delete"

	// Invoice Management
	PermInvoicesView    = "invoices:view"
	PermInvoicesCreate  = "invoices:create"
	PermInvoicesApprove = "invoices:approve"

	// Order Management
	PermOrdersView   = "orders:view"
	PermOrdersCreate = "orders:create"
	PermOrdersEdit   = "orders:edit"

	// Payment Management
	PermPaymentsView   = "payments:view"
	PermPaymentsRecord = "payments:record"

	// Reporting
	PermReportsView     = "reports:view"
	PermReportsGenerate = "reports:generate"

	// Settings
	PermSettingsView   = "settings:view"
	PermSettingsManage = "settings:manage"

	// Data Operations
	PermImportData    = "data:import"
	PermDeleteRecords = "records:delete"

	// Intelligence & Chat
	PermIntelligenceChat = "intelligence:chat"
	PermFinanceView      = "finance:view" // Access to financial data through Butler/reports

	// Offers Management
	PermOffersView   = "offers:view"
	PermOffersCreate = "offers:create"

	// User Management
	PermUsersManage = "users:manage"
)

// SeedDefaultRoles creates the 5 default roles: admin, manager, sales, operations, staff
// Called automatically on first startup or manually from admin
