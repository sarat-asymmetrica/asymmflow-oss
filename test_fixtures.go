package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"
)

// TestFixtureGenerator creates realistic test fixtures from scanned offer data
type TestFixtureGenerator struct {
	Scanner       *OfferScanner
	Random        *rand.Rand
	BaseCustomers map[string]CustomerProfile // Profiles built from real data
}

// CustomerProfile represents aggregated customer behavior from offer history
type CustomerProfile struct {
	Name              string
	Industry          string
	Country           string
	TotalOrders       int
	ExecutedOrders    int
	AvgOrderValue     float64
	AvgPaymentDays    float64
	PaymentVariance   float64
	RelationshipYears int
	IsHighValue       bool
	IsReliable        bool
	ProductPreference string
}

// NewTestFixtureGenerator creates generator from scanned offers
func NewTestFixtureGenerator(scanner *OfferScanner) *TestFixtureGenerator {
	return &TestFixtureGenerator{
		Scanner:       scanner,
		Random:        rand.New(rand.NewSource(time.Now().UnixNano())),
		BaseCustomers: make(map[string]CustomerProfile),
	}
}

// BuildCustomerProfiles analyzes scanned offers to create realistic customer profiles
func (tfg *TestFixtureGenerator) BuildCustomerProfiles() {
	// Group offers by customer
	customerOffers := make(map[string][]OfferMetadata)

	for _, offer := range tfg.Scanner.Offers {
		if offer.CustomerName == "" {
			continue
		}

		customerOffers[offer.CustomerName] = append(customerOffers[offer.CustomerName], offer)
	}

	// Build profile for each customer
	for customerName, offers := range customerOffers {
		profile := CustomerProfile{
			Name:        customerName,
			TotalOrders: len(offers),
			Industry:    tfg.InferIndustry(offers),
			Country:     "Bahrain", // Default (Acme Instrumentation is Bahrain-based)
		}

		// Count executed orders
		for _, offer := range offers {
			if offer.HasExecution {
				profile.ExecutedOrders++
			}
		}

		// Estimate order values (using heuristics from product types)
		totalValue := 0.0
		for _, offer := range offers {
			estimatedValue := tfg.EstimateOfferValue(offer)
			totalValue += estimatedValue
		}

		if profile.TotalOrders > 0 {
			profile.AvgOrderValue = totalValue / float64(profile.TotalOrders)
		}

		// Payment behavior (estimated from execution rate and customer type)
		if profile.ExecutedOrders > 0 {
			// Good customers: 30-45 days, low variance
			// Medium customers: 45-60 days, medium variance
			// Risky customers: 60-90 days, high variance
			winRate := float64(profile.ExecutedOrders) / float64(profile.TotalOrders)

			if winRate > 0.7 {
				// Reliable customer
				profile.AvgPaymentDays = 30 + tfg.Random.Float64()*15
				profile.PaymentVariance = 5 + tfg.Random.Float64()*5
				profile.IsReliable = true
			} else if winRate > 0.4 {
				// Medium reliability
				profile.AvgPaymentDays = 45 + tfg.Random.Float64()*15
				profile.PaymentVariance = 10 + tfg.Random.Float64()*10
				profile.IsReliable = false
			} else {
				// Risky customer
				profile.AvgPaymentDays = 60 + tfg.Random.Float64()*30
				profile.PaymentVariance = 15 + tfg.Random.Float64()*20
				profile.IsReliable = false
			}
		} else {
			// Never executed - highly risky
			profile.AvgPaymentDays = 75 + tfg.Random.Float64()*25
			profile.PaymentVariance = 20 + tfg.Random.Float64()*25
			profile.IsReliable = false
		}

		// Relationship tenure (estimate 1-15 years based on offer volume)
		if profile.TotalOrders >= 10 {
			profile.RelationshipYears = 10 + tfg.Random.Intn(6)
		} else if profile.TotalOrders >= 5 {
			profile.RelationshipYears = 5 + tfg.Random.Intn(6)
		} else {
			profile.RelationshipYears = 1 + tfg.Random.Intn(4)
		}

		// High-value determination
		profile.IsHighValue = profile.AvgOrderValue > 5000 || profile.TotalOrders >= 5

		// Product preference (most common product type)
		profile.ProductPreference = tfg.FindPreferredProduct(offers)

		tfg.BaseCustomers[customerName] = profile
	}
}

// InferIndustry guesses industry from customer name and product types
func (tfg *TestFixtureGenerator) InferIndustry(offers []OfferMetadata) string {
	// Count product types to infer industry
	aitCount := 0
	fitCount := 0
	litCount := 0

	for _, offer := range offers {
		switch offer.ProductType {
		case "AIT":
			aitCount++
		case "FIT":
			fitCount++
		case "LIT":
			litCount++
		}
	}

	// Heuristic industry mapping
	if aitCount > fitCount && aitCount > litCount {
		return "Water & Wastewater" // pH, conductivity = water treatment
	} else if fitCount > 0 {
		return "Oil & Gas" // Flow measurement = O&G
	} else if litCount > 0 {
		return "Storage & Terminals" // Level = tank farms
	}

	return "Industrial"
}

// EstimateOfferValue estimates order value from product type
func (tfg *TestFixtureGenerator) EstimateOfferValue(offer OfferMetadata) float64 {
	// Rough estimates based on typical instrument costs
	baseValue := 2000.0 // BHD

	switch offer.ProductType {
	case "AIT":
		baseValue = 3000 + tfg.Random.Float64()*2000 // 3K-5K BHD
	case "FIT":
		baseValue = 4000 + tfg.Random.Float64()*3000 // 4K-7K BHD
	case "LIT":
		baseValue = 2500 + tfg.Random.Float64()*2000 // 2.5K-4.5K BHD
	case "SP":
		baseValue = 500 + tfg.Random.Float64()*1000 // 500-1500 BHD
	case "FEED":
		baseValue = 10000 + tfg.Random.Float64()*20000 // 10K-30K BHD
	}

	// Multiply by revision count (more revisions = larger/complex order)
	revisionMultiplier := 1.0 + float64(offer.RevisionCount)*0.1

	return baseValue * revisionMultiplier
}

// FindPreferredProduct finds most common product type for customer
func (tfg *TestFixtureGenerator) FindPreferredProduct(offers []OfferMetadata) string {
	counts := make(map[string]int)

	for _, offer := range offers {
		if offer.ProductType != "" {
			counts[offer.ProductType]++
		}
	}

	// Find max
	maxCount := 0
	preferred := "AIT" // default

	for productType, count := range counts {
		if count > maxCount {
			maxCount = count
			preferred = productType
		}
	}

	return preferred
}

// GenerateTestCustomers creates realistic test customers for payment predictor
func (tfg *TestFixtureGenerator) GenerateTestCustomers(count int) []Customer {
	customers := make([]Customer, 0, count)

	// Use real customer profiles
	profileList := make([]CustomerProfile, 0, len(tfg.BaseCustomers))
	for _, profile := range tfg.BaseCustomers {
		profileList = append(profileList, profile)
	}

	for i := 0; i < count; i++ {
		var customer Customer

		if i < len(profileList) {
			// Use real customer profile
			profile := profileList[i]
			customer = tfg.CustomerFromProfile(profile)
		} else {
			// Generate synthetic customer based on random profile
			randomProfile := profileList[tfg.Random.Intn(len(profileList))]
			customer = tfg.SyntheticCustomer(randomProfile, i-len(profileList)+1)
		}

		customers = append(customers, customer)
	}

	return customers
}

// CustomerFromProfile converts profile to Customer struct
func (tfg *TestFixtureGenerator) CustomerFromProfile(profile CustomerProfile) Customer {
	// Generate payment history based on profile statistics
	historyLength := 5 + tfg.Random.Intn(10) // 5-15 past payments

	paymentHistory := make([]int, historyLength)
	for j := 0; j < historyLength; j++ {
		// Normal distribution around avg with variance
		days := int(profile.AvgPaymentDays + tfg.Random.NormFloat64()*math.Sqrt(profile.PaymentVariance))

		// Clamp to reasonable range
		if days < 15 {
			days = 15
		}
		if days > 120 {
			days = 120
		}

		paymentHistory[j] = days
	}

	// Generate order history
	orderHistory := make([]float64, profile.TotalOrders)
	for j := 0; j < profile.TotalOrders; j++ {
		// Vary around average with ±30%
		orderValue := profile.AvgOrderValue * (0.7 + tfg.Random.Float64()*0.6)
		orderHistory[j] = orderValue
	}

	return Customer{
		ID:             fmt.Sprintf("CUST-%s", profile.Name),
		BusinessName:   profile.Name,
		OrderValue:     profile.AvgOrderValue * (0.8 + tfg.Random.Float64()*0.4), // Current order varies
		OrderHistory:   orderHistory,
		PaymentHistory: paymentHistory,
		RelationYears:  profile.RelationshipYears,
		Industry:       profile.Industry,
		Country:        profile.Country,
		IsEmergency:    boolToInt(tfg.Random.Float64() < 0.15), // 15% emergency orders
		HasABB:         boolToInt(tfg.Random.Float64() < 0.25), // 25% have ABB competition
		DisputeCount:   tfg.PoissonSample(0.5),                 // Avg 0.5 disputes
	}
}

// SyntheticCustomer creates new customer based on template
func (tfg *TestFixtureGenerator) SyntheticCustomer(template CustomerProfile, id int) Customer {
	// Create variation on template
	profile := template
	profile.Name = fmt.Sprintf("%s-VARIANT-%d", template.Name, id)

	// Add noise to values (±20%)
	profile.AvgOrderValue *= (0.8 + tfg.Random.Float64()*0.4)
	profile.AvgPaymentDays *= (0.8 + tfg.Random.Float64()*0.4)
	profile.PaymentVariance *= (0.7 + tfg.Random.Float64()*0.6)

	return tfg.CustomerFromProfile(profile)
}

// PoissonSample samples from Poisson distribution (for count data like disputes)
func (tfg *TestFixtureGenerator) PoissonSample(lambda float64) int {
	// Simple Poisson sampler using inverse transform
	L := math.Exp(-lambda)
	k := 0
	p := 1.0

	for p > L {
		k++
		p *= tfg.Random.Float64()
	}

	return k - 1
}

// ExportToJSON writes test fixtures to JSON file
func (tfg *TestFixtureGenerator) ExportToJSON(customers []Customer, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	return encoder.Encode(customers)
}

// GeneratePaymentPredictorFixtures creates complete test suite for payment predictor
func (tfg *TestFixtureGenerator) GeneratePaymentPredictorFixtures() map[string][]Customer {
	fixtures := make(map[string][]Customer)

	// Grade A: Excellent customers (30-40 days, low variance, high tenure)
	gradeA := make([]Customer, 0, 5)
	for _, profile := range tfg.BaseCustomers {
		if profile.IsReliable && profile.AvgPaymentDays < 40 && profile.RelationshipYears >= 5 {
			gradeA = append(gradeA, tfg.CustomerFromProfile(profile))
		}
		if len(gradeA) >= 5 {
			break
		}
	}
	fixtures["grade_a"] = gradeA

	// Grade B: Good customers (40-55 days, medium variance)
	gradeB := make([]Customer, 0, 5)
	for _, profile := range tfg.BaseCustomers {
		if profile.AvgPaymentDays >= 40 && profile.AvgPaymentDays < 55 {
			gradeB = append(gradeB, tfg.CustomerFromProfile(profile))
		}
		if len(gradeB) >= 5 {
			break
		}
	}
	fixtures["grade_b"] = gradeB

	// Grade C: Moderate risk (55-75 days)
	gradeC := make([]Customer, 0, 5)
	for _, profile := range tfg.BaseCustomers {
		if profile.AvgPaymentDays >= 55 && profile.AvgPaymentDays < 75 {
			gradeC = append(gradeC, tfg.CustomerFromProfile(profile))
		}
		if len(gradeC) >= 5 {
			break
		}
	}
	fixtures["grade_c"] = gradeC

	// Grade D: High risk (75+ days, high variance, disputes)
	gradeD := make([]Customer, 0, 5)
	for _, profile := range tfg.BaseCustomers {
		if !profile.IsReliable && profile.AvgPaymentDays >= 75 {
			customer := tfg.CustomerFromProfile(profile)
			customer.DisputeCount = 1 + tfg.Random.Intn(3) // Add disputes
			gradeD = append(gradeD, customer)
		}
		if len(gradeD) >= 5 {
			break
		}
	}
	fixtures["grade_d"] = gradeD

	return fixtures
}

// GenerateGeometryRoutingFixtures creates events for geometry routing tests
func (tfg *TestFixtureGenerator) GenerateGeometryRoutingFixtures() []map[string]any {
	events := make([]map[string]any, 0, 20)

	// RFQ Received events
	for _, offer := range tfg.Scanner.Offers {
		if offer.HasRFQ {
			events = append(events, map[string]any{
				"event_type":    "RFQ_RECEIVED",
				"offer_id":      offer.OfferID,
				"customer_name": offer.CustomerName,
				"product_type":  offer.ProductType,
				"timestamp":     time.Now().Add(-time.Duration(offer.CycleDays+30) * 24 * time.Hour),
			})
		}
	}

	// Offer Submitted events
	for _, offer := range tfg.Scanner.Offers {
		if offer.HasOffer {
			events = append(events, map[string]any{
				"event_type":    "OFFER_SUBMITTED",
				"offer_id":      offer.OfferID,
				"customer_name": offer.CustomerName,
				"revision":      offer.RevisionCount,
				"timestamp":     time.Now().Add(-time.Duration(offer.CycleDays+15) * 24 * time.Hour),
			})
		}
	}

	// PO Received events
	for _, offer := range tfg.Scanner.Offers {
		if offer.HasExecution {
			events = append(events, map[string]any{
				"event_type":    "PO_RECEIVED",
				"offer_id":      offer.OfferID,
				"customer_name": offer.CustomerName,
				"value":         tfg.EstimateOfferValue(offer),
				"timestamp":     time.Now().Add(-time.Duration(offer.CycleDays) * 24 * time.Hour),
			})
		}
	}

	return events
}

// GenerateCustomer360Fixtures creates multi-source customer profiles
func (tfg *TestFixtureGenerator) GenerateCustomer360Fixtures() []map[string]any {
	profiles := make([]map[string]any, 0, len(tfg.BaseCustomers))

	for _, profile := range tfg.BaseCustomers {
		// Aggregate data from multiple sources
		customer360 := map[string]any{
			"customer_name": profile.Name,
			"sources": map[string]any{
				"offers": map[string]any{
					"total_offers":       profile.TotalOrders,
					"executed_offers":    profile.ExecutedOrders,
					"win_rate":           float64(profile.ExecutedOrders) / float64(profile.TotalOrders),
					"avg_order_value":    profile.AvgOrderValue,
					"product_preference": profile.ProductPreference,
				},
				"payments": map[string]any{
					"avg_payment_days": profile.AvgPaymentDays,
					"payment_variance": profile.PaymentVariance,
					"is_reliable":      profile.IsReliable,
				},
				"relationship": map[string]any{
					"tenure_years":  profile.RelationshipYears,
					"industry":      profile.Industry,
					"country":       profile.Country,
					"is_high_value": profile.IsHighValue,
				},
			},
			"risk_grade": tfg.CalculateRiskGrade(profile),
		}

		profiles = append(profiles, customer360)
	}

	return profiles
}

// CalculateRiskGrade assigns A/B/C/D grade to customer
func (tfg *TestFixtureGenerator) CalculateRiskGrade(profile CustomerProfile) string {
	score := 0.0

	// Payment speed (40% weight)
	if profile.AvgPaymentDays < 40 {
		score += 40
	} else if profile.AvgPaymentDays < 55 {
		score += 30
	} else if profile.AvgPaymentDays < 75 {
		score += 15
	}

	// Reliability (30% weight)
	if profile.IsReliable {
		score += 30
	}

	// Tenure (20% weight)
	if profile.RelationshipYears >= 10 {
		score += 20
	} else if profile.RelationshipYears >= 5 {
		score += 15
	} else if profile.RelationshipYears >= 2 {
		score += 10
	}

	// Win rate (10% weight)
	winRate := float64(profile.ExecutedOrders) / float64(profile.TotalOrders)
	score += winRate * 10

	// Assign grade
	if score >= 80 {
		return "A"
	} else if score >= 60 {
		return "B"
	} else if score >= 40 {
		return "C"
	}
	return "D"
}
