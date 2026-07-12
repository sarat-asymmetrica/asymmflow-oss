package main

import butlerprediction "ph_holdings_app/pkg/butler/prediction"

// BatchSummary generates summary statistics for batch predictions.
type BatchSummary = butlerprediction.BatchSummary

// BusinessValue captures the financial impact of M79 payment predictions.
type BusinessValue = butlerprediction.BusinessValue

func BatchPredictCustomers(customers []*Customer) []PaymentPrediction {
	return butlerprediction.BatchPredictCustomers(customers)
}

func SummarizeBatch(customers []*Customer, predictions []PaymentPrediction) BatchSummary {
	return butlerprediction.SummarizeBatch(customers, predictions)
}

func CalculateBusinessValue(customers []*Customer, predictions []PaymentPrediction) BusinessValue {
	return butlerprediction.CalculateBusinessValue(customers, predictions)
}
