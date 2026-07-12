# Entity Graph System - Customer360 Relationships

A complete entity graph system for AsymmFlow that builds knowledge graphs from SSOT data, enabling powerful Customer360 relationship visualization.

## 🌟 Features

- **Graph Database**: GORM-backed graph storage with nodes and edges
- **SSOT Builder**: Automatically constructs graph from existing database tables
- **D3.js Export**: JSON format compatible with D3.js force-directed graphs
- **Efficient Queries**: Indexed lookups, BFS traversal, full-text search
- **Statistics**: Graph density, degree distribution, node/edge counts
- **8 Node Types**: Customer, Offer, Product, Contact, Industry, Supplier, Country, Category
- **10 Edge Types**: OFFERED_TO, CONTAINS_PRODUCT, HAS_CONTACT, BELONGS_TO_INDUSTRY, etc.

## 📊 Graph Schema

### Node Types

| Type | External ID Format | Example |
|------|-------------------|---------|
| Customer | `customer:CO1` | NPC Refinery |
| Offer | `offer:INV-2025-0106` | INV-2025-0106 quotation |
| Product | `product:FMU90` | Rhine Flowmeter FMU90 |
| Contact | `contact:1` | Ahmed Al-Khalifa |
| Industry | `industry:Oil & Gas` | Oil & Gas sector |
| Supplier | `supplier:RI` | Rhine Instruments |
| Country | `country:Bahrain` | Bahrain |
| Category | `category:Flow Meters` | Product category |

### Edge Types

| Type | Direction | Meaning |
|------|-----------|---------|
| OFFERED_TO | Offer → Customer | Quotation sent to customer |
| CONTAINS_PRODUCT | Offer → Product | Offer includes product |
| HAS_CONTACT | Customer → Contact | Customer's contact person |
| BELONGS_TO_INDUSTRY | Customer → Industry | Customer operates in industry |
| LOCATED_IN | Customer → Country | Customer's country |
| SUPPLIES | Supplier → Product | Supplier provides product |
| COMPETES_WITH | Product → Product | Competitive products |
| SIMILAR_TO | Customer → Customer | Similar customers |

## 🚀 Quick Start

### 1. Initialize Graph Service

```go
import "ph_holdings_app/pkg/graph"

// In app.go startup
graphService := graph.NewGraphService(db)
```

### 2. Build Graph from SSOT

```go
builder := graph.NewSSOTBuilder(db)
stats, err := builder.BuildGraph()

// Output:
// ✅ Graph build complete: 127 nodes, 384 edges
// Duration: 1.2s
```

### 3. Query Customer Graph

```go
// Get all relationships for customer CO1 (depth 1)
graphData, err := graphService.GetCustomerGraph("customer:CO1", 1)

// Returns D3.js-compatible JSON:
// {
//   "nodes": [...],
//   "links": [...],
//   "stats": {...}
// }
```

### 4. Export for D3.js Visualization

```go
jsonBytes, err := graphService.ExportGraphJSON()
// Save to file or send to frontend
```

## 📖 API Reference

### Graph Service

#### Node Operations

```go
// Create node
node, err := service.CreateNode(
    graph.NodeTypeCustomer,
    "customer:CO1",
    "NPC Refinery",
    map[string]interface{}{
        "industry": "Oil & Gas",
        "payment_grade": "A",
    },
)

// Get node by external ID
node, err := service.GetNodeByExternalID(graph.NodeTypeCustomer, "customer:CO1")

// Get all nodes of a type
nodes, err := service.GetNodesByType(graph.NodeTypeCustomer, 50)

// Search nodes
results, err := service.SearchEntities("NPC", 10)

// Delete node (also deletes connected edges)
err := service.DeleteNode(nodeID)
```

#### Edge Operations

```go
// Create edge
edge, err := service.CreateEdge(
    sourceID,
    targetID,
    graph.EdgeOfferedTo,
    12.75,  // weight
    map[string]interface{}{
        "value_bhd": 15000.0,
        "win_probability": 0.85,
    },
)

// Get edges for a node
edges, err := service.GetEdgesByNode(nodeID)

// Get edges by type
edges, err := service.GetEdgesByType(graph.EdgeOfferedTo, 100)

// Delete edge
err := service.DeleteEdge(edgeID)
```

#### Graph Queries

```go
// Get customer graph (BFS traversal to specified depth)
graphData, err := service.GetCustomerGraph("customer:CO1", 2)

// Get entity graph (all entities of type + relationships)
graphData, err := service.GetEntityGraph(graph.NodeTypeOffer, 50)

// Get node relationships (immediate connections)
edges, err := service.GetNodeRelationships(nodeID)

// Export entire graph as D3.js JSON
jsonBytes, err := service.ExportGraphJSON()
```

#### Statistics

```go
stats, err := service.GetGraphStats()

// Returns:
// {
//   "total_nodes": 127,
//   "total_edges": 384,
//   "nodes_by_type": {
//     "Customer": 45,
//     "Offer": 67,
//     "Product": 15
//   },
//   "edges_by_type": {
//     "OFFERED_TO": 67,
//     "CONTAINS_PRODUCT": 198
//   },
//   "density": 0.0237,
//   "avg_degree": 6.05,
//   "max_degree": 23,
//   "max_degree_node": "NPC Refinery"
// }
```

### SSOT Builder

```go
builder := graph.NewSSOTBuilder(db)

// Build graph incrementally (adds/updates)
stats, err := builder.BuildGraph()

// Rebuild graph from scratch (clears existing)
stats, err := builder.RebuildGraph()

// Update graph with recent changes
stats, err := builder.UpdateGraph(time.Now().Add(-24 * time.Hour))
```

## 🎨 D3.js Integration

### JSON Format

The graph exports in D3.js-compatible format:

```json
{
  "nodes": [
    {
      "id": "customer:CO1",
      "label": "NPC Refinery",
      "type": "Customer",
      "size": 8.0,
      "properties": {
        "industry": "Oil & Gas",
        "payment_grade": "A"
      }
    }
  ],
  "links": [
    {
      "source": "offer:INV-2025-0106",
      "target": "customer:CO1",
      "type": "OFFERED_TO",
      "weight": 12.75,
      "properties": {
        "value_bhd": 15000.0,
        "win_probability": 0.85
      }
    }
  ],
  "stats": {
    "total_nodes": 127,
    "total_edges": 384,
    "density": 0.0237
  }
}
```

### Frontend Integration

```typescript
// Svelte component (Customer360Screen.svelte)
import { GetCustomerGraph } from '@/wailsjs/go/main/App';

async function loadGraph(customerID: string) {
  const graphData = await GetCustomerGraph(customerID, 2);

  // Pass to D3.js force simulation
  const simulation = d3.forceSimulation(graphData.nodes)
    .force("link", d3.forceLink(graphData.links).id(d => d.id))
    .force("charge", d3.forceManyBody().strength(-400))
    .force("center", d3.forceCenter(width / 2, height / 2));
}
```

## 🔧 Wails Bindings

Available in `app.go`:

```go
// Get entity graph by type
GetEntityGraph(nodeType string, limit int) (*graph.GraphData, error)

// Get customer-centered graph
GetCustomerGraph(customerID string, depth int) (*graph.GraphData, error)

// Get node relationships
GetNodeRelationships(nodeID uint) ([]graph.GraphEdge, error)

// Search entities
SearchGraphEntities(query string, limit int) ([]graph.GraphNode, error)

// Get statistics
GetGraphStats() (*graph.GraphStats, error)

// Build from SSOT
BuildEntityGraph() (*graph.BuildStats, error)

// Rebuild from scratch
RebuildEntityGraph() (*graph.BuildStats, error)

// Export JSON
ExportGraphJSON() (string, error)
```

## 📈 Performance

| Operation | Complexity | Notes |
|-----------|-----------|-------|
| Node creation | O(1) | With upsert (update if exists) |
| Edge creation | O(1) | With upsert |
| Node lookup | O(1) | Indexed by external_id |
| BFS traversal | O(V + E) | Depth-limited |
| Search | O(N) | LIKE query, consider FTS5 for large graphs |
| Statistics | O(N + E) | Cached aggregates |

### Benchmarks (on test data)

```
BenchmarkNodeCreation-8     50000    31.2 µs/op
BenchmarkEdgeCreation-8     40000    37.8 µs/op
BenchmarkGraphQuery-8       20000    82.4 µs/op
```

## 🧪 Testing

Run tests:

```bash
cd pkg/graph
go test -v
```

Run benchmarks:

```bash
go test -bench=. -benchmem
```

## 🛠️ Advanced Usage

### Custom Graph Algorithms

```go
// Find shortest path between nodes (add your implementation)
func ShortestPath(service *GraphService, sourceID, targetID uint) []uint {
    // BFS implementation
}

// Find communities/clusters
func FindCommunities(service *GraphService) [][]uint {
    // Community detection algorithm
}

// Calculate centrality metrics
func CalculateCentrality(service *GraphService, nodeID uint) float64 {
    // Betweenness/closeness centrality
}
```

### Incremental Updates

```go
// Update graph when new data arrives
func UpdateGraphOnNewOffer(service *GraphService, offerID uint) error {
    // 1. Fetch offer from database
    // 2. Create/update offer node
    // 3. Create edge to customer
    // 4. Create edges to products
}
```

### Graph Visualization Strategies

**Force-Directed Layout** (D3.js):
```javascript
const simulation = d3.forceSimulation(nodes)
    .force("link", d3.forceLink(links).distance(100))
    .force("charge", d3.forceManyBody().strength(-400))
    .force("center", d3.forceCenter(width / 2, height / 2))
    .force("collision", d3.forceCollide().radius(30));
```

**Hierarchical Layout** (for customers by industry):
```javascript
const hierarchy = d3.hierarchy(treeData)
    .sum(d => d.value);
const treemap = d3.tree().size([width, height]);
treemap(hierarchy);
```

**Circular Layout** (for supplier networks):
```javascript
const pie = d3.pie().value(d => d.weight);
const arc = d3.arc().innerRadius(0).outerRadius(radius);
```

## 🎯 Use Cases

### 1. Customer360 View

Show all relationships for a customer:
- Offers received
- Products purchased
- Contact persons
- Industry peers
- Geographic location

### 2. Product Recommendations

Find similar customers who bought the same products:
```go
// Get customer A's products
graphA, _ := service.GetCustomerGraph("customer:A", 2)

// Find customers with similar product edges
// Recommend products customer A hasn't bought yet
```

### 3. Sales Pipeline Analysis

Track offer → customer → industry patterns:
```go
// Get all offers for Oil & Gas industry
industryGraph, _ := service.GetEntityGraph(graph.NodeTypeIndustry, 1)

// Analyze win rates, discount patterns, product preferences
```

### 4. Supplier Risk Assessment

Identify single points of failure:
```go
stats, _ := service.GetGraphStats()

// If stats.MaxDegreeNode is a supplier with MaxDegree > threshold
// Flag as supply chain risk
```

## 📝 Example: Full Workflow

```go
package main

import (
    "fmt"
    "ph_holdings_app/pkg/graph"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

func main() {
    // 1. Setup database
    db, _ := gorm.Open(sqlite.Open("ph_holdings.db"), &gorm.Config{})
    db.AutoMigrate(&graph.GraphNode{}, &graph.GraphEdge{})

    // 2. Initialize services
    service := graph.NewGraphService(db)
    builder := graph.NewSSOTBuilder(db)

    // 3. Build graph from SSOT
    stats, _ := builder.BuildGraph()
    fmt.Printf("Built graph: %d nodes, %d edges\n", stats.NodesCreated, stats.EdgesCreated)

    // 4. Query customer graph
    graphData, _ := service.GetCustomerGraph("customer:CO1", 2)
    fmt.Printf("Customer CO1 has %d connected nodes\n", len(graphData.Nodes))

    // 5. Export for visualization
    jsonBytes, _ := service.ExportGraphJSON()
    fmt.Printf("Exported %d bytes of JSON\n", len(jsonBytes))

    // 6. Get statistics
    graphStats, _ := service.GetGraphStats()
    fmt.Printf("Graph density: %.4f\n", graphStats.Density)
    fmt.Printf("Average degree: %.2f\n", graphStats.AvgDegree)
    fmt.Printf("Most connected: %s (%d connections)\n",
        graphStats.MaxDegreeNode, graphStats.MaxDegree)
}
```

## 🔮 Future Enhancements

- [ ] Graph algorithms (PageRank, community detection)
- [ ] Full-text search with SQLite FTS5
- [ ] Graph analytics (centrality, clustering coefficient)
- [ ] Temporal graphs (time-aware relationships)
- [ ] Graph versioning (snapshot history)
- [ ] Neo4j export (for production graph DB)
- [ ] GraphQL API endpoint
- [ ] Real-time updates via WebSockets

## 📚 References

- [D3.js Force Layout](https://d3js.org/d3-force)
- [Graph Theory Basics](https://en.wikipedia.org/wiki/Graph_theory)
- [GORM Documentation](https://gorm.io/docs/)
- [Neo4j Cypher Query Language](https://neo4j.com/developer/cypher/)

---

**Built with Love × Simplicity × Truth × Joy** ❤️
May this graph serve all beings in the universe! 🌟
