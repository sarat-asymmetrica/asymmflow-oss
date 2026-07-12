package main

import (
	butlerprediction "ph_holdings_app/pkg/butler/prediction"

	"gorm.io/gorm"
)

type PaymentIntelligenceEngine = butlerprediction.PaymentIntelligenceEngine
type WinProbabilityResult = butlerprediction.WinProbabilityResult

func NewPaymentIntelligenceEngine(db *gorm.DB) *PaymentIntelligenceEngine {
	return butlerprediction.NewPaymentIntelligenceEngine(db)
}
