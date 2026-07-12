package graph

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// =============================================================================
// SSOT MODELS (simplified references to avoid circular imports)
// =============================================================================

// These are lightweight references to the main database models
// Used for graph building without importing the entire app

type SSOTCustomer struct {
	ID           uint
	CustomerID   string
	BusinessName string
	Industry     string
	Country      string
	PaymentGrade string
}

type SSOTContact struct {
	ID          uint
	CustomerID  uint
	ContactName string
	Email       string
	JobTitle    string
}

type SSOTOffer struct {
	ID             uint
	OfferNumber    string
	CustomerID     uint
	CustomerName   string
	TotalValueBHD  float64
	Stage          string
	WinProbability float64
	CreatedAt      time.Time
}

type SSOTOfferItem struct {
	ID          uint
	OfferID     uint
	ProductID   *uint
	ProductCode string
	Description string
	Quantity    float64
}

type SSOTProduct struct {
	ID              uint
	ProductCode     string
	ProductName     string
	ProductCategory string
	SupplierCode    string
}

type SSOTSupplier struct {
	ID           uint
	SupplierCode string
	SupplierName string
	Country      string
}

// =============================================================================
// SSOT BUILDER
// =============================================================================

// SSOTBuilder builds the entity graph from the SSOT database
type SSOTBuilder struct {
	db           *gorm.DB
	graphService *GraphService
	stats        BuildStats
}

// BuildStats tracks graph building progress
type BuildStats struct {
	NodesCreated int            `json:"nodes_created"`
	EdgesCreated int            `json:"edges_created"`
	Errors       int            `json:"errors"`
	Duration     time.Duration  `json:"duration"`
	NodesByType  map[string]int `json:"nodes_by_type"`
	EdgesByType  map[string]int `json:"edges_by_type"`
	StartTime    time.Time      `json:"start_time"`
	EndTime      time.Time      `json:"end_time"`
}

// NewSSOTBuilder creates a new SSOT builder
func NewSSOTBuilder(db *gorm.DB) *SSOTBuilder {
	return &SSOTBuilder{
		db:           db,
		graphService: NewGraphService(db),
		stats: BuildStats{
			NodesByType: make(map[string]int),
			EdgesByType: make(map[string]int),
		},
	}
}

// BuildGraph scans all SSOT tables and creates graph relationships
func (b *SSOTBuilder) BuildGraph() (*BuildStats, error) {
	b.stats.StartTime = time.Now()
	log.Println("🔨 Starting SSOT graph build...")

	// Build in order: entities first, then relationships
	steps := []struct {
		name string
		fn   func() error
	}{
		{"Customers", b.buildCustomers},
		{"Contacts", b.buildContacts},
		{"Suppliers", b.buildSuppliers},
		{"Products", b.buildProducts},
		{"Industries", b.buildIndustries},
		{"Countries", b.buildCountries},
		{"Categories", b.buildCategories},
		{"Offers", b.buildOffers},
		{"Offer Items", b.buildOfferItems},
	}

	for _, step := range steps {
		log.Printf("  ├─ Building %s...", step.name)
		if err := step.fn(); err != nil {
			log.Printf("  │  ⚠️ Error: %v", err)
			b.stats.Errors++
		}
	}

	b.stats.EndTime = time.Now()
	b.stats.Duration = b.stats.EndTime.Sub(b.stats.StartTime)

	log.Printf("✅ Graph build complete!")
	log.Printf("   Nodes: %d, Edges: %d, Errors: %d", b.stats.NodesCreated, b.stats.EdgesCreated, b.stats.Errors)
	log.Printf("   Duration: %v", b.stats.Duration)

	return &b.stats, nil
}

// buildCustomers creates customer nodes
func (b *SSOTBuilder) buildCustomers() error {
	var customers []SSOTCustomer
	if err := b.db.Table("customers").Find(&customers).Error; err != nil {
		return err
	}

	for _, customer := range customers {
		props := map[string]any{
			"business_name": customer.BusinessName,
			"industry":      customer.Industry,
			"country":       customer.Country,
			"payment_grade": customer.PaymentGrade,
		}

		_, err := b.graphService.CreateNode(
			NodeTypeCustomer,
			fmt.Sprintf("customer:%s", customer.CustomerID),
			customer.BusinessName,
			props,
		)
		if err == nil {
			b.stats.NodesCreated++
			b.stats.NodesByType[NodeTypeCustomer]++
		}
	}

	return nil
}

// buildContacts creates contact nodes and links to customers
func (b *SSOTBuilder) buildContacts() error {
	var contacts []SSOTContact
	if err := b.db.Table("customer_contacts").Find(&contacts).Error; err != nil {
		return err
	}

	for _, contact := range contacts {
		props := map[string]any{
			"email":     contact.Email,
			"job_title": contact.JobTitle,
		}

		// Create contact node
		contactNode, err := b.graphService.CreateNode(
			NodeTypeContact,
			fmt.Sprintf("contact:%d", contact.ID),
			contact.ContactName,
			props,
		)
		if err == nil {
			b.stats.NodesCreated++
			b.stats.NodesByType[NodeTypeContact]++

			// Link to customer
			var customer SSOTCustomer
			if err := b.db.Table("customers").Where("id = ?", contact.CustomerID).First(&customer).Error; err == nil {
				customerNode, err := b.graphService.GetNodeByExternalID(NodeTypeCustomer, fmt.Sprintf("customer:%s", customer.CustomerID))
				if err == nil {
					_, err = b.graphService.CreateEdge(customerNode.ID, contactNode.ID, EdgeHasContact, 1.0, nil)
					if err == nil {
						b.stats.EdgesCreated++
						b.stats.EdgesByType[EdgeHasContact]++
					}
				}
			}
		}
	}

	return nil
}

// buildSuppliers creates supplier nodes
func (b *SSOTBuilder) buildSuppliers() error {
	var suppliers []SSOTSupplier
	if err := b.db.Table("suppliers").Find(&suppliers).Error; err != nil {
		return err
	}

	for _, supplier := range suppliers {
		props := map[string]any{
			"supplier_code": supplier.SupplierCode,
			"country":       supplier.Country,
		}

		_, err := b.graphService.CreateNode(
			NodeTypeSupplier,
			fmt.Sprintf("supplier:%s", supplier.SupplierCode),
			supplier.SupplierName,
			props,
		)
		if err == nil {
			b.stats.NodesCreated++
			b.stats.NodesByType[NodeTypeSupplier]++
		}
	}

	return nil
}

// buildProducts creates product nodes and links to suppliers
func (b *SSOTBuilder) buildProducts() error {
	var products []SSOTProduct
	if err := b.db.Table("products").Find(&products).Error; err != nil {
		return err
	}

	for _, product := range products {
		props := map[string]any{
			"product_code":     product.ProductCode,
			"product_category": product.ProductCategory,
			"supplier_code":    product.SupplierCode,
		}

		// Create product node
		productNode, err := b.graphService.CreateNode(
			NodeTypeProduct,
			fmt.Sprintf("product:%s", product.ProductCode),
			product.ProductName,
			props,
		)
		if err == nil {
			b.stats.NodesCreated++
			b.stats.NodesByType[NodeTypeProduct]++

			// Link to supplier
			if product.SupplierCode != "" {
				supplierNode, err := b.graphService.GetNodeByExternalID(NodeTypeSupplier, fmt.Sprintf("supplier:%s", product.SupplierCode))
				if err == nil {
					_, err = b.graphService.CreateEdge(supplierNode.ID, productNode.ID, EdgeSupplies, 1.0, nil)
					if err == nil {
						b.stats.EdgesCreated++
						b.stats.EdgesByType[EdgeSupplies]++
					}
				}
			}
		}
	}

	return nil
}

// buildIndustries creates industry nodes and links to customers
func (b *SSOTBuilder) buildIndustries() error {
	// Get unique industries from customers
	var industries []string
	b.db.Table("customers").Distinct("industry").Pluck("industry", &industries)

	for _, industry := range industries {
		if industry == "" {
			continue
		}

		// Create industry node
		industryNode, err := b.graphService.CreateNode(
			NodeTypeIndustry,
			fmt.Sprintf("industry:%s", industry),
			industry,
			nil,
		)
		if err == nil {
			b.stats.NodesCreated++
			b.stats.NodesByType[NodeTypeIndustry]++

			// Link all customers in this industry
			var customers []SSOTCustomer
			b.db.Table("customers").Where("industry = ?", industry).Find(&customers)
			for _, customer := range customers {
				customerNode, err := b.graphService.GetNodeByExternalID(NodeTypeCustomer, fmt.Sprintf("customer:%s", customer.CustomerID))
				if err == nil {
					_, err = b.graphService.CreateEdge(customerNode.ID, industryNode.ID, EdgeBelongsToIndustry, 1.0, nil)
					if err == nil {
						b.stats.EdgesCreated++
						b.stats.EdgesByType[EdgeBelongsToIndustry]++
					}
				}
			}
		}
	}

	return nil
}

// buildCountries creates country nodes and links to customers
func (b *SSOTBuilder) buildCountries() error {
	// Get unique countries from customers
	var countries []string
	b.db.Table("customers").Distinct("country").Pluck("country", &countries)

	for _, country := range countries {
		if country == "" {
			continue
		}

		// Create country node
		countryNode, err := b.graphService.CreateNode(
			NodeTypeCountry,
			fmt.Sprintf("country:%s", country),
			country,
			nil,
		)
		if err == nil {
			b.stats.NodesCreated++
			b.stats.NodesByType[NodeTypeCountry]++

			// Link all customers in this country
			var customers []SSOTCustomer
			b.db.Table("customers").Where("country = ?", country).Find(&customers)
			for _, customer := range customers {
				customerNode, err := b.graphService.GetNodeByExternalID(NodeTypeCustomer, fmt.Sprintf("customer:%s", customer.CustomerID))
				if err == nil {
					_, err = b.graphService.CreateEdge(customerNode.ID, countryNode.ID, EdgeLocatedIn, 1.0, nil)
					if err == nil {
						b.stats.EdgesCreated++
						b.stats.EdgesByType[EdgeLocatedIn]++
					}
				}
			}
		}
	}

	return nil
}

// buildCategories creates product category nodes and links to products
func (b *SSOTBuilder) buildCategories() error {
	// Get unique categories from products
	var categories []string
	b.db.Table("products").Distinct("product_category").Pluck("product_category", &categories)

	for _, category := range categories {
		if category == "" {
			continue
		}

		// Create category node
		categoryNode, err := b.graphService.CreateNode(
			NodeTypeCategory,
			fmt.Sprintf("category:%s", category),
			category,
			nil,
		)
		if err == nil {
			b.stats.NodesCreated++
			b.stats.NodesByType[NodeTypeCategory]++

			// Link all products in this category
			var products []SSOTProduct
			b.db.Table("products").Where("product_category = ?", category).Find(&products)
			for _, product := range products {
				productNode, err := b.graphService.GetNodeByExternalID(NodeTypeProduct, fmt.Sprintf("product:%s", product.ProductCode))
				if err == nil {
					_, err = b.graphService.CreateEdge(productNode.ID, categoryNode.ID, EdgeBelongsToIndustry, 1.0, nil)
					if err == nil {
						b.stats.EdgesCreated++
						b.stats.EdgesByType[EdgeBelongsToIndustry]++
					}
				}
			}
		}
	}

	return nil
}

// buildOffers creates offer nodes and links to customers
func (b *SSOTBuilder) buildOffers() error {
	var offers []SSOTOffer
	if err := b.db.Table("offers").Find(&offers).Error; err != nil {
		return err
	}

	for _, offer := range offers {
		props := map[string]any{
			"offer_number":    offer.OfferNumber,
			"total_value_bhd": offer.TotalValueBHD,
			"stage":           offer.Stage,
			"win_probability": offer.WinProbability,
			"created_at":      offer.CreatedAt.Format("2006-01-02"),
		}

		// Create offer node
		offerNode, err := b.graphService.CreateNode(
			NodeTypeOffer,
			fmt.Sprintf("offer:%s", offer.OfferNumber),
			fmt.Sprintf("%s (%s)", offer.OfferNumber, offer.CustomerName),
			props,
		)
		if err == nil {
			b.stats.NodesCreated++
			b.stats.NodesByType[NodeTypeOffer]++

			// Link to customer
			var customer SSOTCustomer
			if err := b.db.Table("customers").Where("id = ?", offer.CustomerID).First(&customer).Error; err == nil {
				customerNode, err := b.graphService.GetNodeByExternalID(NodeTypeCustomer, fmt.Sprintf("customer:%s", customer.CustomerID))
				if err == nil {
					// Edge weight based on offer value and win probability
					weight := (offer.TotalValueBHD / 1000.0) * offer.WinProbability
					edgeProps := map[string]any{
						"value_bhd":       offer.TotalValueBHD,
						"win_probability": offer.WinProbability,
					}

					_, err = b.graphService.CreateEdge(offerNode.ID, customerNode.ID, EdgeOfferedTo, weight, edgeProps)
					if err == nil {
						b.stats.EdgesCreated++
						b.stats.EdgesByType[EdgeOfferedTo]++
					}
				}
			}
		}
	}

	return nil
}

// buildOfferItems links offers to products
func (b *SSOTBuilder) buildOfferItems() error {
	var items []SSOTOfferItem
	if err := b.db.Table("offer_items").Find(&items).Error; err != nil {
		return err
	}

	for _, item := range items {
		// Get offer node
		var offer SSOTOffer
		if err := b.db.Table("offers").Where("id = ?", item.OfferID).First(&offer).Error; err != nil {
			continue
		}

		offerNode, err := b.graphService.GetNodeByExternalID(NodeTypeOffer, fmt.Sprintf("offer:%s", offer.OfferNumber))
		if err != nil {
			continue
		}

		// Get product node
		var productCode string
		if item.ProductID != nil {
			var product SSOTProduct
			if err := b.db.Table("products").Where("id = ?", *item.ProductID).First(&product).Error; err == nil {
				productCode = product.ProductCode
			}
		} else {
			productCode = item.ProductCode
		}

		if productCode == "" {
			continue
		}

		productNode, err := b.graphService.GetNodeByExternalID(NodeTypeProduct, fmt.Sprintf("product:%s", productCode))
		if err != nil {
			continue
		}

		// Link offer to product (weighted by quantity)
		edgeProps := map[string]any{
			"quantity":    item.Quantity,
			"description": item.Description,
		}

		_, err = b.graphService.CreateEdge(offerNode.ID, productNode.ID, EdgeContainsProduct, item.Quantity, edgeProps)
		if err == nil {
			b.stats.EdgesCreated++
			b.stats.EdgesByType[EdgeContainsProduct]++
		}
	}

	return nil
}

// RebuildGraph clears the existing graph and rebuilds from scratch
func (b *SSOTBuilder) RebuildGraph() (*BuildStats, error) {
	log.Println("🗑️ Clearing existing graph...")

	// Delete all edges
	if err := b.db.Exec("DELETE FROM graph_edges").Error; err != nil {
		return nil, fmt.Errorf("failed to clear edges: %w", err)
	}

	// Delete all nodes
	if err := b.db.Exec("DELETE FROM graph_nodes").Error; err != nil {
		return nil, fmt.Errorf("failed to clear nodes: %w", err)
	}

	log.Println("✅ Graph cleared. Building fresh graph...")

	return b.BuildGraph()
}

// UpdateGraph incrementally updates the graph with recent changes
func (b *SSOTBuilder) UpdateGraph(since time.Time) (*BuildStats, error) {
	// For now, just rebuild (can optimize later with incremental updates)
	log.Printf("📊 Updating graph (changes since %v)", since)
	return b.BuildGraph()
}
