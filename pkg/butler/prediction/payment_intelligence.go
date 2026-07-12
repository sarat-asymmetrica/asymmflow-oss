package prediction

import (
	"fmt"

	butler "ph_holdings_app/pkg/butler"
	crm "ph_holdings_app/pkg/crm"

	"gorm.io/gorm"
)

// PaymentIntelligenceEngine handles predictive analytics for payments.
type PaymentIntelligenceEngine struct {
	db *gorm.DB
}

// NewPaymentIntelligenceEngine creates a new engine.
func NewPaymentIntelligenceEngine(db *gorm.DB) *PaymentIntelligenceEngine {
	return &PaymentIntelligenceEngine{db: db}
}

// WinProbabilityResult represents the prediction outcome.
type WinProbabilityResult struct {
	Probability float64  `json:"probability"`
	Grade       string   `json:"grade"`
	Factors     []string `json:"factors"`
}

// PredictWinProbability calculates likelihood of winning an offer.
func (s *PaymentIntelligenceEngine) PredictWinProbability(offerID string) (float64, error) {
	if s.db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	var offer crm.Offer
	if err := s.db.First(&offer, "id = ?", offerID).Error; err != nil {
		return 0, fmt.Errorf("offer not found")
	}

	var customer crm.CustomerMaster
	if err := s.db.First(&customer, "id = ?", offer.CustomerID).Error; err != nil {
		return 0, fmt.Errorf("customer not found")
	}

	// Base probability
	probability := 0.5

	// Grade adjustment
	switch customer.PaymentGrade {
	case "A":
		probability += 0.3
	case "B":
		probability += 0.1
	case "C":
		probability -= 0.1
	case "D":
		probability -= 0.3
	}

	// Competition adjustment
	if offer.HasABBCompetition {
		probability -= 0.2
	}

	// Cap probability
	if probability > 0.95 {
		probability = 0.95
	}
	if probability < 0.05 {
		probability = 0.05
	}

	// Save prediction
	prediction := &butler.WinProbabilityPrediction{
		OfferID:              offer.ID,
		PredictedProbability: probability,
	}

	// Mission I (I-16): surface persistence failure instead of returning a
	// probability the caller believes was recorded.
	if err := s.db.Create(prediction).Error; err != nil {
		return probability, err
	}

	return probability, nil
}

// GenerateDiscountRecommendation suggests optimal discount.
func (s *PaymentIntelligenceEngine) GenerateDiscountRecommendation(offerID string) (float64, error) {
	// Placeholder logic
	return 0.05, nil
}
