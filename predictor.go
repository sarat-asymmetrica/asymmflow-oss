package main

import butlerprediction "ph_holdings_app/pkg/butler/prediction"

// PaymentPrediction represents prediction output.
type PaymentPrediction = butlerprediction.PaymentPrediction

// PaymentPredictor predicts customer payment behavior using the M79 manifold.
type PaymentPredictor = butlerprediction.PaymentPredictor

func NewPaymentPredictor(customer *Customer) *PaymentPredictor {
	return butlerprediction.NewPaymentPredictor(customer)
}
