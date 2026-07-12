package main

import (
	"fmt"
	"strings"
	"time"

	butlerintent "ph_holdings_app/pkg/butler/intent"
)

type butlerARProjectionScope struct {
	start          time.Time
	end            time.Time
	label          string
	orderStart     time.Time
	includeOrders  bool
	includeOffers  bool
	invoiceOnly    bool
	needsClarify   bool
	intentDetected bool
}

type butlerProjectionStageRow struct {
	Stage    string  `gorm:"column:stage"`
	Count    int64   `gorm:"column:count"`
	TotalBHD float64 `gorm:"column:total_bhd"`
}

func (a *App) tryButlerIntentClarificationFastPath(intent Intent, message string, hasFinanceAccess bool) (string, []ButlerAction, bool) {
	return butlerintent.TryClarificationFastPath(intent, message, hasFinanceAccess)
}

func shouldAskButlerClarifyingQuestion(intent Intent, q string) bool {
	return butlerintent.ShouldAskClarifyingQuestion(intent, q)
}

func hasButlerReferenceToken(q string) bool {
	return butlerintent.HasReferenceToken(q)
}

func buildButlerClarificationActions(hasFinanceAccess bool) []ButlerAction {
	return butlerintent.BuildClarificationActions(hasFinanceAccess)
}

func (a *App) tryGroundedARProjectionFastPath(intent Intent, message string, hasFinanceAccess bool) (string, []ButlerAction, bool) {
	if a != nil {
		if msg, actions, handled := a.butlerFastpathService().TryARProjection(message, hasFinanceAccess); handled {
			return msg, actions, true
		}
	}

	if a == nil || a.db == nil {
		return "", nil, false
	}

	scope := parseButlerARProjectionScope(message)
	if !scope.intentDetected || scope.needsClarify {
		return "", nil, false
	}
	if !hasFinanceAccess {
		return "AR projections require finance:view permission. I can still help with non-financial order or follow-up context.", []ButlerAction{}, true
	}

	now := time.Now()
	openStatuses := []string{"Sent", "PartiallyPaid", "Overdue"}

	var currentOpenAR float64
	_ = a.db.Model(&Invoice{}).
		Where("status IN ? AND outstanding_bhd > 0", openStatuses).
		Select("COALESCE(SUM(outstanding_bhd), 0)").
		Scan(&currentOpenAR).Error

	var overdueAR float64
	_ = a.db.Model(&Invoice{}).
		Where("status IN ? AND outstanding_bhd > 0 AND due_date < ?", openStatuses, now).
		Select("COALESCE(SUM(outstanding_bhd), 0)").
		Scan(&overdueAR).Error

	var invoiceDueInWindow float64
	_ = a.db.Model(&Invoice{}).
		Where("status IN ? AND outstanding_bhd > 0 AND due_date >= ? AND due_date < ?", openStatuses, scope.start, scope.end).
		Select("COALESCE(SUM(outstanding_bhd), 0)").
		Scan(&invoiceDueInWindow).Error

	var invoiceDueByWindowEnd float64
	_ = a.db.Model(&Invoice{}).
		Where("status IN ? AND outstanding_bhd > 0 AND due_date < ?", openStatuses, scope.end).
		Select("COALESCE(SUM(outstanding_bhd), 0)").
		Scan(&invoiceDueByWindowEnd).Error

	uninvoicedOrderExposure := 0.0
	pendingOrderCount := 0
	orderErrText := ""
	if scope.includeOrders {
		var err error
		uninvoicedOrderExposure, pendingOrderCount, err = a.calculateUninvoicedOrderExposure(scope.orderStart, scope.end)
		if err != nil {
			orderErrText = fmt.Sprintf(" Order exposure could not be calculated: %v", err)
		}
	}

	weightedPipeline := 0.0
	pipelineLines := []string{}
	if scope.includeOffers {
		weightedPipeline, pipelineLines = a.calculateWeightedButlerPipeline(scope.orderStart, scope.end)
	}

	totalAttention := invoiceDueByWindowEnd
	if scope.includeOrders {
		totalAttention += uninvoicedOrderExposure
	}
	if scope.includeOffers {
		totalAttention += weightedPipeline
	}

	orderLine := "- Confirmed uninvoiced orders were not included in this view."
	if scope.includeOrders {
		orderLine = fmt.Sprintf("- Confirmed uninvoiced orders from %s through %s: %d orders, %s BHD expected new AR once invoiced.%s",
			scope.orderStart.Format("2 Jan 2006"),
			scope.end.Add(-time.Nanosecond).Format("2 Jan 2006"),
			pendingOrderCount,
			formatBHD(uninvoicedOrderExposure),
			orderErrText,
		)
	}

	pipelineText := "- Weighted active offer/opportunity pipeline was not included in this view."
	if scope.includeOffers {
		if len(pipelineLines) == 0 {
			pipelineText = "- No weighted active offer/opportunity pipeline was found for this horizon."
		} else {
			pipelineText = strings.Join(pipelineLines, "\n")
		}
	}

	mode := "invoice-only"
	if scope.includeOrders && scope.includeOffers {
		mode = "issued invoices + confirmed orders + weighted active offers"
	} else if scope.includeOrders {
		mode = "issued invoices + confirmed orders"
	}

	response := fmt.Sprintf(`AR projection - %s

Basis
- Mode: %s.
- Window: %s to %s.
- Order exposure uses the fresh-start operating year from %s so confirmed 2026 orders are not ignored.

Issued invoice AR
- Current open invoice AR: %s BHD.
- Already overdue AR: %s BHD.
- Invoice AR due inside this window: %s BHD.
- Booked invoice AR due by window end, including overdue balances: %s BHD.

Confirmed order receivable path
%s

Weighted pipeline path
%s

Projection
- Receivable attention for this basis: %s BHD.
- This is an AR creation and collection focus number, not guaranteed cash collection.
- Confirmed orders should create receivables only after invoicing; cash timing then depends on payment terms and customer behavior.

Why this route exists
- Butler is using local SQL before the external model here.
- The earlier zero-collection style answer was mixing cash collection prediction with AR creation and was not enough for management decisions. Orders are now explicitly counted when you choose the confirmed-order basis.`, scope.label, mode, scope.start.Format("2 Jan 2006"), scope.end.Add(-time.Nanosecond).Format("2 Jan 2006"), scope.orderStart.Format("2 Jan 2006"), formatBHD(currentOpenAR), formatBHD(overdueAR), formatBHD(invoiceDueInWindow), formatBHD(invoiceDueByWindowEnd), orderLine, pipelineText, formatBHD(totalAttention))

	actions := []ButlerAction{
		butlerPromptAction("Invoice-only view", "Show AR projection for next month using issued invoices only.", "ar_projection", "invoices_only"),
		butlerPromptAction("Include orders", "Show AR projection for next month including issued invoices and confirmed uninvoiced orders.", "ar_projection", "invoices_confirmed_orders"),
		butlerPromptAction("Add weighted offers", "Show AR projection for next month including issued invoices, confirmed orders, and weighted active offers.", "ar_projection", "weighted_pipeline"),
		butlerPromptAction("Manager brief", "Create a manager AR brief for the next two months with confirmed orders, offer pipeline, and collection risk.", "ar_projection", "manager_brief"),
	}

	return response, actions, true
}

func butlerPromptAction(label, prompt, intentID, optionID string) ButlerAction {
	return butlerintent.PromptAction(label, prompt, intentID, optionID)
}

func parseButlerARProjectionScope(message string) butlerARProjectionScope {
	scope := butlerintent.ParseARProjectionScope(message)
	return butlerARProjectionScope{
		start:          scope.Start,
		end:            scope.End,
		label:          scope.Label,
		orderStart:     scope.OrderStart,
		includeOrders:  scope.IncludeOrders,
		includeOffers:  scope.IncludeOffers,
		invoiceOnly:    scope.InvoiceOnly,
		needsClarify:   scope.NeedsClarify,
		intentDetected: scope.IntentDetected,
	}
}

func (a *App) calculateWeightedButlerPipeline(start, end time.Time) (float64, []string) {
	if a == nil || a.db == nil {
		return 0, []string{}
	}

	rows := []butlerProjectionStageRow{}
	_ = a.db.Table("offers").
		Select("COALESCE(stage, 'Unstaged') AS stage, COUNT(*) AS count, COALESCE(SUM(total_value_bhd), 0) AS total_bhd").
		Where("deleted_at IS NULL").
		Where("quotation_date >= ? AND quotation_date < ?", start, end).
		Where("stage IN ?", []string{"RFQ", "Qualified", "Proposal", "Quoted", "Won"}).
		Group("COALESCE(stage, 'Unstaged')").
		Order("total_bhd DESC").
		Scan(&rows).Error

	var opportunityRows []butlerProjectionStageRow
	_ = a.db.Table("opportunities").
		Select("COALESCE(stage, 'Unstaged') AS stage, COUNT(*) AS count, COALESCE(SUM(revenue_bhd), 0) AS total_bhd").
		Where("deleted_at IS NULL").
		Where("(year = ? OR (offer_date >= ? AND offer_date < ?))", end.Year(), start, end).
		Where("stage IN ?", []string{"RFQ", "Qualified", "Proposal", "Quoted", "Won"}).
		Group("COALESCE(stage, 'Unstaged')").
		Order("total_bhd DESC").
		Scan(&opportunityRows).Error

	rows = append(rows, opportunityRows...)
	if len(rows) == 0 {
		return 0, []string{}
	}

	weighted := 0.0
	lines := []string{"- Weighted active pipeline, separated from confirmed orders:"}
	for _, row := range rows {
		if row.Count == 0 || row.TotalBHD <= 0 {
			continue
		}
		weight := butlerPipelineStageWeight(row.Stage)
		weightedValue := row.TotalBHD * weight
		weighted += weightedValue
		lines = append(lines, fmt.Sprintf("  - %s: %d records, %s BHD at %.0f%% = %s BHD",
			firstNonEmpty(strings.TrimSpace(row.Stage), "Unstaged"),
			row.Count,
			formatBHD(row.TotalBHD),
			weight*100,
			formatBHD(weightedValue),
		))
	}

	if weighted <= 0 {
		return 0, []string{}
	}
	lines = append(lines, fmt.Sprintf("- Weighted pipeline contribution: %s BHD.", formatBHD(weighted)))
	return round3(weighted), lines
}

func butlerPipelineStageWeight(stage string) float64 {
	return butlerintent.PipelineStageWeight(stage)
}

func isBroadCapabilitiesQuestion(q string) bool {
	return butlerintent.IsBroadCapabilitiesQuestion(q)
}

func isCapabilitySelectionPrompt(q string) bool {
	return butlerintent.IsCapabilitySelectionPrompt(q)
}

func normalizeButlerRouterText(message string) string {
	return butlerintent.NormalizeRouterText(message)
}

func butlerContainsWord(q, word string) bool {
	return butlerintent.ContainsWord(q, word)
}

func nextCalendarMonthWindow(now time.Time) (time.Time, time.Time, string) {
	return butlerintent.NextCalendarMonthWindow(now)
}

func beginningOfDay(value time.Time) time.Time {
	return butlerintent.BeginningOfDay(value)
}
