package graph

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Migrate schema
	err = db.AutoMigrate(&GraphNode{}, &GraphEdge{})
	if err != nil {
		t.Fatalf("Failed to migrate schema: %v", err)
	}

	return db
}

func TestGraphNodeCreation(t *testing.T) {
	db := setupTestDB(t)
	service := NewGraphService(db)

	// Create a customer node
	props := map[string]any{
		"business_name": "National Petroleum Refinery",
		"industry":      "Oil & Gas",
		"payment_grade": "A",
	}

	node, err := service.CreateNode(NodeTypeCustomer, "customer:CO1", "National Petroleum Refinery", props)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}

	if node.NodeType != NodeTypeCustomer {
		t.Errorf("Expected NodeType %s, got %s", NodeTypeCustomer, node.NodeType)
	}

	if node.Label != "National Petroleum Refinery" {
		t.Errorf("Expected Label 'National Petroleum Refinery', got '%s'", node.Label)
	}

	// Verify properties
	var retrievedProps map[string]any
	json.Unmarshal(node.Properties, &retrievedProps)

	if retrievedProps["industry"] != "Oil & Gas" {
		t.Errorf("Expected industry 'Oil & Gas', got '%v'", retrievedProps["industry"])
	}
}

func TestGraphEdgeCreation(t *testing.T) {
	db := setupTestDB(t)
	service := NewGraphService(db)

	// Create two nodes
	customerNode, _ := service.CreateNode(NodeTypeCustomer, "customer:CO1", "National Petroleum", nil)
	offerNode, _ := service.CreateNode(NodeTypeOffer, "offer:INV-2025-0106", "INV-2025-0106", nil)

	// Create edge
	edgeProps := map[string]any{
		"value_bhd":       15000.0,
		"win_probability": 0.85,
	}

	edge, err := service.CreateEdge(offerNode.ID, customerNode.ID, EdgeOfferedTo, 12.75, edgeProps)
	if err != nil {
		t.Fatalf("Failed to create edge: %v", err)
	}

	if edge.EdgeType != EdgeOfferedTo {
		t.Errorf("Expected EdgeType %s, got %s", EdgeOfferedTo, edge.EdgeType)
	}

	if edge.Weight != 12.75 {
		t.Errorf("Expected Weight 12.75, got %f", edge.Weight)
	}
}

func TestGetNodesByType(t *testing.T) {
	db := setupTestDB(t)
	service := NewGraphService(db)

	// Create multiple customer nodes
	service.CreateNode(NodeTypeCustomer, "customer:CO1", "National Petroleum", nil)
	service.CreateNode(NodeTypeCustomer, "customer:CO2", "Gulf Smelting", nil)
	service.CreateNode(NodeTypeCustomer, "customer:CO3", "Delta Petrochemicals", nil)
	service.CreateNode(NodeTypeProduct, "product:FL90", "Flowmeter FL90", nil)

	// Query customers
	customers, err := service.GetNodesByType(NodeTypeCustomer, 10)
	if err != nil {
		t.Fatalf("Failed to get nodes: %v", err)
	}

	if len(customers) != 3 {
		t.Errorf("Expected 3 customers, got %d", len(customers))
	}
}

func TestSearchEntities(t *testing.T) {
	db := setupTestDB(t)
	service := NewGraphService(db)

	// Create test data
	service.CreateNode(NodeTypeCustomer, "customer:CO1", "National Petroleum Refinery", nil)
	service.CreateNode(NodeTypeCustomer, "customer:CO2", "Gulf Smelting Co.", nil)
	service.CreateNode(NodeTypeProduct, "product:FL90", "Flowmeter FL90", nil)

	// Search for "National Petroleum"
	results, err := service.SearchEntities("National Petroleum", 10)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'National Petroleum', got %d", len(results))
	}

	if results[0].Label != "National Petroleum Refinery" {
		t.Errorf("Expected 'National Petroleum Refinery', got '%s'", results[0].Label)
	}
}

func TestGetCustomerGraph(t *testing.T) {
	db := setupTestDB(t)
	service := NewGraphService(db)

	// Create customer
	customerNode, _ := service.CreateNode(NodeTypeCustomer, "customer:CO1", "National Petroleum", nil)

	// Create related entities
	offerNode, _ := service.CreateNode(NodeTypeOffer, "offer:INV-2025-0106", "INV-2025-0106", nil)
	contactNode, _ := service.CreateNode(NodeTypeContact, "contact:1", "Omar Hassan", nil)
	industryNode, _ := service.CreateNode(NodeTypeIndustry, "industry:Oil & Gas", "Oil & Gas", nil)

	// Create relationships
	service.CreateEdge(offerNode.ID, customerNode.ID, EdgeOfferedTo, 1.0, nil)
	service.CreateEdge(customerNode.ID, contactNode.ID, EdgeHasContact, 1.0, nil)
	service.CreateEdge(customerNode.ID, industryNode.ID, EdgeBelongsToIndustry, 1.0, nil)

	// Get customer graph (depth 1)
	graphData, err := service.GetCustomerGraph("customer:CO1", 1)
	if err != nil {
		t.Fatalf("Failed to get customer graph: %v", err)
	}

	// Should have 4 nodes (customer + 3 related)
	if len(graphData.Nodes) != 4 {
		t.Errorf("Expected 4 nodes, got %d", len(graphData.Nodes))
	}

	// Should have 3 edges
	if len(graphData.Links) != 3 {
		t.Errorf("Expected 3 edges, got %d", len(graphData.Links))
	}
}

func TestGraphStats(t *testing.T) {
	db := setupTestDB(t)
	service := NewGraphService(db)

	// Create test graph
	customerNode, _ := service.CreateNode(NodeTypeCustomer, "customer:CO1", "National Petroleum", nil)
	offerNode, _ := service.CreateNode(NodeTypeOffer, "offer:INV-2025-0106", "INV-2025-0106", nil)
	productNode, _ := service.CreateNode(NodeTypeProduct, "product:FL90", "FL90", nil)

	service.CreateEdge(offerNode.ID, customerNode.ID, EdgeOfferedTo, 1.0, nil)
	service.CreateEdge(offerNode.ID, productNode.ID, EdgeContainsProduct, 1.0, nil)

	// Get stats
	stats, err := service.GetGraphStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.TotalNodes != 3 {
		t.Errorf("Expected 3 nodes, got %d", stats.TotalNodes)
	}

	if stats.TotalEdges != 2 {
		t.Errorf("Expected 2 edges, got %d", stats.TotalEdges)
	}

	if stats.NodesByType[NodeTypeCustomer] != 1 {
		t.Errorf("Expected 1 customer, got %d", stats.NodesByType[NodeTypeCustomer])
	}

	if stats.EdgesByType[EdgeOfferedTo] != 1 {
		t.Errorf("Expected 1 OFFERED_TO edge, got %d", stats.EdgesByType[EdgeOfferedTo])
	}
}

func TestD3JSONExport(t *testing.T) {
	db := setupTestDB(t)
	service := NewGraphService(db)

	// Create simple graph
	customerNode, _ := service.CreateNode(NodeTypeCustomer, "customer:CO1", "National Petroleum", nil)
	offerNode, _ := service.CreateNode(NodeTypeOffer, "offer:INV-2025-0106", "INV-2025-0106", nil)
	service.CreateEdge(offerNode.ID, customerNode.ID, EdgeOfferedTo, 1.0, nil)

	// Export to JSON
	jsonBytes, err := service.ExportGraphJSON()
	if err != nil {
		t.Fatalf("Failed to export JSON: %v", err)
	}

	// Parse JSON
	var graphData GraphData
	if err := json.Unmarshal(jsonBytes, &graphData); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if len(graphData.Nodes) != 2 {
		t.Errorf("Expected 2 nodes in JSON, got %d", len(graphData.Nodes))
	}

	if len(graphData.Links) != 1 {
		t.Errorf("Expected 1 link in JSON, got %d", len(graphData.Links))
	}

	// Verify D3 format
	if graphData.Nodes[0].ID == "" {
		t.Error("D3 node missing ID")
	}

	if graphData.Links[0].Source == "" || graphData.Links[0].Target == "" {
		t.Error("D3 link missing source or target")
	}
}

func TestSSOTBuilder(t *testing.T) {
	db := setupTestDB(t)

	// Create SSOT tables
	db.Exec(`CREATE TABLE customers (
		id INTEGER PRIMARY KEY,
		customer_id TEXT,
		business_name TEXT,
		industry TEXT,
		country TEXT,
		payment_grade TEXT
	)`)

	db.Exec(`CREATE TABLE offers (
		id INTEGER PRIMARY KEY,
		offer_number TEXT,
		customer_id INTEGER,
		customer_name TEXT,
		total_value_bhd REAL,
		stage TEXT,
		win_probability REAL,
		created_at DATETIME
	)`)

	// Insert test data
	db.Exec(`INSERT INTO customers (id, customer_id, business_name, industry, country, payment_grade)
		VALUES (1, 'CO1', 'National Petroleum Refinery', 'Oil & Gas', 'Bahrain', 'A')`)

	db.Exec(`INSERT INTO offers (id, offer_number, customer_id, customer_name, total_value_bhd, stage, win_probability, created_at)
		VALUES (1, 'INV-2025-0106', 1, 'National Petroleum Refinery', 15000.0, 'Quoted', 0.85, ?)`, time.Now())

	// Build graph
	builder := NewSSOTBuilder(db)
	stats, err := builder.BuildGraph()
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	if stats.NodesCreated < 2 {
		t.Errorf("Expected at least 2 nodes created, got %d", stats.NodesCreated)
	}

	if stats.EdgesCreated < 1 {
		t.Errorf("Expected at least 1 edge created, got %d", stats.EdgesCreated)
	}

	t.Logf("Build stats: %d nodes, %d edges in %v", stats.NodesCreated, stats.EdgesCreated, stats.Duration)
}

// Benchmark tests
func BenchmarkNodeCreation(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&GraphNode{}, &GraphEdge{})
	service := NewGraphService(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.CreateNode(NodeTypeCustomer, "customer:TEST", "Test Customer", nil)
	}
}

func BenchmarkEdgeCreation(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&GraphNode{}, &GraphEdge{})
	service := NewGraphService(db)

	node1, _ := service.CreateNode(NodeTypeCustomer, "customer:1", "Customer 1", nil)
	node2, _ := service.CreateNode(NodeTypeOffer, "offer:1", "Offer 1", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.CreateEdge(node1.ID, node2.ID, EdgeOfferedTo, 1.0, nil)
	}
}

func BenchmarkGraphQuery(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&GraphNode{}, &GraphEdge{})
	service := NewGraphService(db)

	// Create test graph
	for i := 0; i < 100; i++ {
		service.CreateNode(NodeTypeCustomer, "customer:TEST", "Test Customer", nil)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.GetNodesByType(NodeTypeCustomer, 10)
	}
}

// Example test (executable documentation)
func ExampleGraphService() {
	// Setup in-memory database
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&GraphNode{}, &GraphEdge{})

	service := NewGraphService(db)

	// Create customer node
	customer, _ := service.CreateNode(NodeTypeCustomer, "customer:CO1", "National Petroleum Refinery", map[string]any{
		"industry":      "Oil & Gas",
		"payment_grade": "A",
	})

	// Create offer node
	offer, _ := service.CreateNode(NodeTypeOffer, "offer:INV-2025-0106", "INV-2025-0106", map[string]any{
		"value_bhd": 15000.0,
		"stage":     "Quoted",
	})

	// Link offer to customer
	service.CreateEdge(offer.ID, customer.ID, EdgeOfferedTo, 12.75, map[string]any{
		"win_probability": 0.85,
	})

	// Get customer graph
	_, _ = service.GetCustomerGraph("customer:CO1", 1)

	// Export to JSON for D3.js
	jsonBytes, _ := service.ExportGraphJSON()
	os.Stdout.Write(jsonBytes)
}
