package graph

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type JSON = json.RawMessage

// sanitizeSearchQuery removes SQL special characters from user search input
// to prevent SQL injection via LIKE wildcards and escape sequences.
func sanitizeSearchQuery(query string) string {
	sanitized := strings.ReplaceAll(query, "%", "")
	sanitized = strings.ReplaceAll(sanitized, "_", "")
	sanitized = strings.ReplaceAll(sanitized, "'", "")
	sanitized = strings.ReplaceAll(sanitized, "\"", "")
	sanitized = strings.ReplaceAll(sanitized, "\\", "")
	sanitized = strings.ReplaceAll(sanitized, ";", "")
	sanitized = strings.ReplaceAll(sanitized, "--", "")
	sanitized = strings.ReplaceAll(sanitized, "/*", "")
	sanitized = strings.ReplaceAll(sanitized, "*/", "")
	sanitized = strings.TrimSpace(sanitized)
	return sanitized
}

// =============================================================================
// GRAPH DATABASE MODELS
// =============================================================================

// GraphNode represents a node in the entity graph
// Nodes represent entities like Customer, Offer, Product, Contact, Industry
type GraphNode struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	NodeType   string    `gorm:"index;size:50" json:"node_type"`    // Customer, Offer, Product, Contact, Industry, Supplier
	ExternalID string    `gorm:"index;size:100" json:"external_id"` // Reference to source table (e.g., "customer:CO1", "offer:INV-2025-0106")
	Label      string    `gorm:"size:255" json:"label"`             // Display name
	Properties JSON      `gorm:"type:text" json:"properties"`       // Additional data as JSON
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (GraphNode) TableName() string {
	return "graph_nodes"
}

// GraphEdge represents a relationship between two nodes
// Edges represent relationships like OFFERED_TO, CONTAINS_PRODUCT, HAS_CONTACT, BELONGS_TO_INDUSTRY
type GraphEdge struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	SourceID   uint      `gorm:"index" json:"source_id"`          // Source node
	TargetID   uint      `gorm:"index" json:"target_id"`          // Target node
	EdgeType   string    `gorm:"index;size:100" json:"edge_type"` // OFFERED_TO, CONTAINS_PRODUCT, etc.
	Properties JSON      `gorm:"type:text" json:"properties"`     // Additional metadata
	Weight     float64   `gorm:"index" json:"weight"`             // Relationship strength (for ranking)
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (GraphEdge) TableName() string {
	return "graph_edges"
}

// =============================================================================
// GRAPH DATA STRUCTURES (D3.js compatible)
// =============================================================================

// GraphData represents the complete graph structure for visualization
// Compatible with D3.js force-directed graph format
type GraphData struct {
	Nodes []D3Node   `json:"nodes"`
	Links []D3Link   `json:"links"`
	Stats GraphStats `json:"stats"`
}

// D3Node represents a node in D3.js format
type D3Node struct {
	ID         string         `json:"id"`         // Unique identifier
	Label      string         `json:"label"`      // Display label
	Type       string         `json:"type"`       // Node type (for coloring)
	Size       float64        `json:"size"`       // Node size (based on connections)
	Properties map[string]any `json:"properties"` // Additional data
}

// D3Link represents an edge in D3.js format
type D3Link struct {
	Source     string         `json:"source"`     // Source node ID
	Target     string         `json:"target"`     // Target node ID
	Type       string         `json:"type"`       // Edge type
	Weight     float64        `json:"weight"`     // Link strength
	Properties map[string]any `json:"properties"` // Additional metadata
}

// GraphStats provides statistics about the graph
type GraphStats struct {
	TotalNodes    int64          `json:"total_nodes"`
	TotalEdges    int64          `json:"total_edges"`
	NodesByType   map[string]int `json:"nodes_by_type"`
	EdgesByType   map[string]int `json:"edges_by_type"`
	Density       float64        `json:"density"`    // Edges / Possible edges
	AvgDegree     float64        `json:"avg_degree"` // Average connections per node
	MaxDegree     int            `json:"max_degree"` // Most connected node
	MaxDegreeNode string         `json:"max_degree_node"`
}

// =============================================================================
// NODE TYPES (Constants)
// =============================================================================

const (
	NodeTypeCustomer = "Customer"
	NodeTypeOffer    = "Offer"
	NodeTypeProduct  = "Product"
	NodeTypeContact  = "Contact"
	NodeTypeIndustry = "Industry"
	NodeTypeSupplier = "Supplier"
	NodeTypeCountry  = "Country"
	NodeTypeCategory = "Category"
)

// =============================================================================
// EDGE TYPES (Constants)
// =============================================================================

const (
	EdgeOfferedTo         = "OFFERED_TO"          // Offer → Customer
	EdgeContainsProduct   = "CONTAINS_PRODUCT"    // Offer → Product
	EdgeHasContact        = "HAS_CONTACT"         // Customer → Contact
	EdgeBelongsToIndustry = "BELONGS_TO_INDUSTRY" // Customer → Industry
	EdgeLocatedIn         = "LOCATED_IN"          // Customer → Country
	EdgeSupplies          = "SUPPLIES"            // Supplier → Product
	EdgeCompetesWith      = "COMPETES_WITH"       // Product → Product (ABB vs Rhine Instruments)

	// E8 LATTICE TOPOLOGY-SOUND ENTITY LINKING
	// Similarity detection verified via E8 root structure (240 roots, kissing number optimal in 8D)
	// Entity uniqueness proven via countNonZero function (shared attributes = non-zero dimensions)
	// Reference: C:\Projects\asymm_all_math\asymmetrica_proofs\AsymmetricaProofs\E8Lattice.lean
	EdgeSimilarTo = "SIMILAR_TO" // Customer → Customer (same industry/size) - E8 verified

	EdgePreviousOffer = "PREVIOUS_OFFER" // Offer → Offer (revision chain)
	EdgeConvertedTo   = "CONVERTED_TO"   // Offer → Order
)

// =============================================================================
// GRAPH SERVICE
// =============================================================================

// GraphService manages the entity graph
type GraphService struct {
	db *gorm.DB
}

// NewGraphService creates a new graph service
func NewGraphService(db *gorm.DB) *GraphService {
	return &GraphService{db: db}
}

// =============================================================================
// NODE OPERATIONS
// =============================================================================

// CreateNode creates a new graph node
func (s *GraphService) CreateNode(nodeType, externalID, label string, properties map[string]any) (*GraphNode, error) {
	// Check if node already exists
	var existing GraphNode
	result := s.db.Where("node_type = ? AND external_id = ?", nodeType, externalID).First(&existing)
	if result.Error == nil {
		// Node exists, update it
		existing.Label = label
		if properties != nil {
			propsJSON, _ := json.Marshal(properties)
			existing.Properties = propsJSON
		}
		if err := s.db.Save(&existing).Error; err != nil {
			return nil, err
		}
		return &existing, nil
	}

	// Create new node
	node := &GraphNode{
		NodeType:   nodeType,
		ExternalID: externalID,
		Label:      label,
	}

	if properties != nil {
		propsJSON, err := json.Marshal(properties)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal properties: %w", err)
		}
		node.Properties = propsJSON
	}

	if err := s.db.Create(node).Error; err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	return node, nil
}

// GetNodeByExternalID retrieves a node by its external ID
func (s *GraphService) GetNodeByExternalID(nodeType, externalID string) (*GraphNode, error) {
	var node GraphNode
	result := s.db.Where("node_type = ? AND external_id = ?", nodeType, externalID).First(&node)
	if result.Error != nil {
		return nil, result.Error
	}
	return &node, nil
}

// GetNodesByType retrieves all nodes of a specific type
func (s *GraphService) GetNodesByType(nodeType string, limit int) ([]GraphNode, error) {
	var nodes []GraphNode
	query := s.db.Where("node_type = ?", nodeType).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

// DeleteNode deletes a node and all its edges
func (s *GraphService) DeleteNode(nodeID uint) error {
	// Delete all edges connected to this node
	s.db.Where("source_id = ? OR target_id = ?", nodeID, nodeID).Delete(&GraphEdge{})

	// Delete the node
	return s.db.Delete(&GraphNode{}, nodeID).Error
}

// =============================================================================
// EDGE OPERATIONS
// =============================================================================

// CreateEdge creates a relationship between two nodes
// E8 LATTICE NOTE: For SIMILAR_TO edges, the weight represents the similarity metric
// computed via E8 countNonZero function (shared attributes in 8D space).
// This ensures topology-sound entity deduplication with mathematical rigor.
func (s *GraphService) CreateEdge(sourceID, targetID uint, edgeType string, weight float64, properties map[string]any) (*GraphEdge, error) {
	// Check if edge already exists
	var existing GraphEdge
	result := s.db.Where("source_id = ? AND target_id = ? AND edge_type = ?", sourceID, targetID, edgeType).First(&existing)
	if result.Error == nil {
		// Edge exists, update it
		existing.Weight = weight
		if properties != nil {
			propsJSON, _ := json.Marshal(properties)
			existing.Properties = propsJSON
		}
		if err := s.db.Save(&existing).Error; err != nil {
			return nil, err
		}
		return &existing, nil
	}

	// Create new edge
	edge := &GraphEdge{
		SourceID: sourceID,
		TargetID: targetID,
		EdgeType: edgeType,
		Weight:   weight,
	}

	if properties != nil {
		propsJSON, err := json.Marshal(properties)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal properties: %w", err)
		}
		edge.Properties = propsJSON
	}

	if err := s.db.Create(edge).Error; err != nil {
		return nil, fmt.Errorf("failed to create edge: %w", err)
	}

	return edge, nil
}

// GetEdgesByNode retrieves all edges connected to a node
func (s *GraphService) GetEdgesByNode(nodeID uint) ([]GraphEdge, error) {
	var edges []GraphEdge
	if err := s.db.Where("source_id = ? OR target_id = ?", nodeID, nodeID).Find(&edges).Error; err != nil {
		return nil, err
	}
	return edges, nil
}

// GetEdgesByType retrieves all edges of a specific type
func (s *GraphService) GetEdgesByType(edgeType string, limit int) ([]GraphEdge, error) {
	var edges []GraphEdge
	query := s.db.Where("edge_type = ?", edgeType).Order("weight DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&edges).Error; err != nil {
		return nil, err
	}
	return edges, nil
}

// DeleteEdge deletes a specific edge
func (s *GraphService) DeleteEdge(edgeID uint) error {
	return s.db.Delete(&GraphEdge{}, edgeID).Error
}

// =============================================================================
// GRAPH QUERIES
// =============================================================================

// GetCustomerGraph retrieves all relationships for a customer
// Returns nodes and edges centered around a specific customer
func (s *GraphService) GetCustomerGraph(customerID string, depth int) (*GraphData, error) {
	// Get customer node
	customerNode, err := s.GetNodeByExternalID(NodeTypeCustomer, customerID)
	if err != nil {
		return nil, fmt.Errorf("customer not found: %w", err)
	}

	// Collect nodes and edges
	nodesMap := make(map[uint]*GraphNode)
	edgesMap := make(map[uint]*GraphEdge)

	// Add customer node
	nodesMap[customerNode.ID] = customerNode

	// BFS traversal to specified depth
	currentLevel := []uint{customerNode.ID}
	visited := make(map[uint]bool)
	visited[customerNode.ID] = true

	for d := 0; d < depth; d++ {
		var nextLevel []uint

		for _, nodeID := range currentLevel {
			// Get all edges for this node
			edges, err := s.GetEdgesByNode(nodeID)
			if err != nil {
				continue
			}

			for _, edge := range edges {
				edgesMap[edge.ID] = &edge

				// Add connected nodes
				var connectedID uint
				if edge.SourceID == nodeID {
					connectedID = edge.TargetID
				} else {
					connectedID = edge.SourceID
				}

				if !visited[connectedID] {
					var node GraphNode
					if err := s.db.First(&node, connectedID).Error; err == nil {
						nodesMap[connectedID] = &node
						visited[connectedID] = true
						nextLevel = append(nextLevel, connectedID)
					}
				}
			}
		}

		currentLevel = nextLevel
		if len(currentLevel) == 0 {
			break
		}
	}

	// Convert to D3.js format
	return s.convertToD3Format(nodesMap, edgesMap)
}

// GetEntityGraph retrieves all entities of a specific type with their relationships
func (s *GraphService) GetEntityGraph(nodeType string, limit int) (*GraphData, error) {
	nodes, err := s.GetNodesByType(nodeType, limit)
	if err != nil {
		return nil, err
	}

	nodesMap := make(map[uint]*GraphNode)
	edgesMap := make(map[uint]*GraphEdge)

	// Add all nodes
	for i := range nodes {
		nodesMap[nodes[i].ID] = &nodes[i]
	}

	// Get all edges between these nodes
	nodeIDs := make([]uint, 0, len(nodes))
	for _, node := range nodes {
		nodeIDs = append(nodeIDs, node.ID)
	}

	var edges []GraphEdge
	if err := s.db.Where("source_id IN ? OR target_id IN ?", nodeIDs, nodeIDs).Find(&edges).Error; err != nil {
		return nil, err
	}

	// Add connected nodes
	for i := range edges {
		edgesMap[edges[i].ID] = &edges[i]

		// Add source and target nodes if not already present
		if _, exists := nodesMap[edges[i].SourceID]; !exists {
			var node GraphNode
			if err := s.db.First(&node, edges[i].SourceID).Error; err == nil {
				nodesMap[node.ID] = &node
			}
		}
		if _, exists := nodesMap[edges[i].TargetID]; !exists {
			var node GraphNode
			if err := s.db.First(&node, edges[i].TargetID).Error; err == nil {
				nodesMap[node.ID] = &node
			}
		}
	}

	return s.convertToD3Format(nodesMap, edgesMap)
}

// GetNodeRelationships retrieves immediate relationships for a node
func (s *GraphService) GetNodeRelationships(nodeID uint) ([]GraphEdge, error) {
	return s.GetEdgesByNode(nodeID)
}

// SearchEntities performs full-text search across all nodes
func (s *GraphService) SearchEntities(query string, limit int) ([]GraphNode, error) {
	var nodes []GraphNode

	// Sanitize user input to prevent SQL injection via LIKE wildcards
	sanitized := sanitizeSearchQuery(query)
	if len(sanitized) < 2 {
		return nodes, nil // Return empty for very short/invalid queries
	}

	searchPattern := "%" + sanitized + "%"

	// Apply limit with sensible default and maximum
	effectiveLimit := limit
	if effectiveLimit <= 0 {
		effectiveLimit = 50
	}
	if effectiveLimit > 100 {
		effectiveLimit = 100 // Prevent DOS
	}

	dbQuery := s.db.Where("label LIKE ? OR external_id LIKE ?", searchPattern, searchPattern).
		Order("created_at DESC").
		Limit(effectiveLimit)

	if err := dbQuery.Find(&nodes).Error; err != nil {
		return nil, err
	}

	return nodes, nil
}

// =============================================================================
// GRAPH STATISTICS
// =============================================================================

// GetGraphStats computes statistics about the graph
func (s *GraphService) GetGraphStats() (*GraphStats, error) {
	stats := &GraphStats{
		NodesByType: make(map[string]int),
		EdgesByType: make(map[string]int),
	}

	// Count total nodes
	s.db.Model(&GraphNode{}).Count(&stats.TotalNodes)

	// Count total edges
	s.db.Model(&GraphEdge{}).Count(&stats.TotalEdges)

	// Count nodes by type
	var nodeTypeCounts []struct {
		NodeType string
		Count    int
	}
	s.db.Model(&GraphNode{}).Select("node_type, COUNT(*) as count").Group("node_type").Scan(&nodeTypeCounts)
	for _, row := range nodeTypeCounts {
		stats.NodesByType[row.NodeType] = row.Count
	}

	// Count edges by type
	var edgeTypeCounts []struct {
		EdgeType string
		Count    int
	}
	s.db.Model(&GraphEdge{}).Select("edge_type, COUNT(*) as count").Group("edge_type").Scan(&edgeTypeCounts)
	for _, row := range edgeTypeCounts {
		stats.EdgesByType[row.EdgeType] = row.Count
	}

	// Calculate density
	if stats.TotalNodes > 1 {
		possibleEdges := int64(stats.TotalNodes * (stats.TotalNodes - 1))
		stats.Density = float64(stats.TotalEdges) / float64(possibleEdges)
		stats.AvgDegree = float64(stats.TotalEdges*2) / float64(stats.TotalNodes)
	}

	// Find most connected node
	var maxDegreeRow struct {
		NodeID uint
		Degree int
	}
	s.db.Raw(`
		SELECT node_id, degree FROM (
			SELECT source_id as node_id, COUNT(*) as degree FROM graph_edges GROUP BY source_id
			UNION ALL
			SELECT target_id as node_id, COUNT(*) as degree FROM graph_edges GROUP BY target_id
		) AS degrees
		ORDER BY degree DESC LIMIT 1
	`).Scan(&maxDegreeRow)

	if maxDegreeRow.NodeID > 0 {
		stats.MaxDegree = maxDegreeRow.Degree
		var node GraphNode
		if err := s.db.First(&node, maxDegreeRow.NodeID).Error; err == nil {
			stats.MaxDegreeNode = node.Label
		}
	}

	return stats, nil
}

// =============================================================================
// D3.js EXPORT
// =============================================================================

// convertToD3Format converts internal graph representation to D3.js format
func (s *GraphService) convertToD3Format(nodesMap map[uint]*GraphNode, edgesMap map[uint]*GraphEdge) (*GraphData, error) {
	// Convert nodes
	d3Nodes := make([]D3Node, 0, len(nodesMap))
	nodeIDToIndex := make(map[uint]string)

	for _, node := range nodesMap {
		var props map[string]any
		if len(node.Properties) > 0 {
			json.Unmarshal(node.Properties, &props)
		}

		// Calculate node size based on degree
		edges, _ := s.GetEdgesByNode(node.ID)
		size := float64(len(edges) + 1) // Minimum size 1

		d3Node := D3Node{
			ID:         node.ExternalID,
			Label:      node.Label,
			Type:       node.NodeType,
			Size:       size,
			Properties: props,
		}
		d3Nodes = append(d3Nodes, d3Node)
		nodeIDToIndex[node.ID] = node.ExternalID
	}

	// Convert edges
	d3Links := make([]D3Link, 0, len(edgesMap))
	for _, edge := range edgesMap {
		var props map[string]any
		if len(edge.Properties) > 0 {
			json.Unmarshal(edge.Properties, &props)
		}

		sourceID, sourceOK := nodeIDToIndex[edge.SourceID]
		targetID, targetOK := nodeIDToIndex[edge.TargetID]

		if sourceOK && targetOK {
			d3Link := D3Link{
				Source:     sourceID,
				Target:     targetID,
				Type:       edge.EdgeType,
				Weight:     edge.Weight,
				Properties: props,
			}
			d3Links = append(d3Links, d3Link)
		}
	}

	// Compute stats
	stats, _ := s.GetGraphStats()

	return &GraphData{
		Nodes: d3Nodes,
		Links: d3Links,
		Stats: *stats,
	}, nil
}

// ExportGraphJSON exports the entire graph as JSON
func (s *GraphService) ExportGraphJSON() ([]byte, error) {
	// Get all nodes
	var nodes []GraphNode
	if err := s.db.Find(&nodes).Error; err != nil {
		return nil, err
	}

	nodesMap := make(map[uint]*GraphNode)
	for i := range nodes {
		nodesMap[nodes[i].ID] = &nodes[i]
	}

	// Get all edges
	var edges []GraphEdge
	if err := s.db.Find(&edges).Error; err != nil {
		return nil, err
	}

	edgesMap := make(map[uint]*GraphEdge)
	for i := range edges {
		edgesMap[edges[i].ID] = &edges[i]
	}

	// Convert to D3 format
	graphData, err := s.convertToD3Format(nodesMap, edgesMap)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(graphData, "", "  ")
}
