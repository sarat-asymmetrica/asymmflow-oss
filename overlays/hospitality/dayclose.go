package hospitality

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ph_holdings_app/pkg/finance/settlement"
	"ph_holdings_app/pkg/kernel/actor"
	"ph_holdings_app/pkg/kernel/money"
)

// OpenItems reports what would block a day close right now: the vertical's
// definition of "open" handed to the generic settlement engine.
func (s *Service) OpenItems() (settlement.OpenItems, error) {
	open := settlement.OpenItems{}

	var sessions int64
	if err := s.db.Model(&OrderSession{}).Where("status = ?", SessionOpen).Count(&sessions).Error; err != nil {
		return nil, err
	}
	open["open sessions"] = int(sessions)

	var tickets int64
	if err := s.db.Model(&Ticket{}).
		Where("status IN ?", []string{TicketQueued, TicketPreparing, TicketReady}).
		Count(&tickets).Error; err != nil {
		return nil, err
	}
	open["live kitchen tickets"] = int(tickets)

	var unpaid int64
	if err := s.db.Model(&Invoice{}).Where("status = ?", InvoiceIssued).Count(&unpaid).Error; err != nil {
		return nil, err
	}
	open["unpaid invoices"] = int(unpaid)

	return open, nil
}

// ExpectedTenders sums the day's captured payments per tender method — the
// system-expected side of the reconciliation.
func (s *Service) ExpectedTenders(businessDate string) ([]settlement.TenderLine, error) {
	type row struct {
		Method string
		Total  int64
	}
	var rows []row
	if err := s.db.Model(&Payment{}).
		Select("method, SUM(amount_halalas) AS total").
		Where("business_date = ?", businessDate).
		Group("method").Order("method").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	lines := make([]settlement.TenderLine, 0, len(rows))
	for _, r := range rows {
		lines = append(lines, settlement.TenderLine{Method: r.Method, Expected: sar(r.Total)})
	}
	return lines, nil
}

// CloseDay reconciles and closes a business date.
//
// The heavy lifting is pkg/finance/settlement: Compute reconciles expected vs
// counted per tender, Close gates on open items, approve authority (kernel —
// agents can never close a day) and the variance-requires-note rule. This
// method adds the vertical's two contributions: the manager PIN gate and the
// once-per-date persistence (UNIQUE business_date).
func (s *Service) CloseDay(businessDate string, declared []settlement.Declaration, by actor.Actor, pin, note string) (*DayClose, error) {
	businessDay, err := time.Parse("2006-01-02", businessDate)
	if err != nil {
		return nil, fmt.Errorf("hospitality: business date must be YYYY-MM-DD: %w", err)
	}
	if err := s.VerifyManagerPIN(pin); err != nil {
		return nil, err
	}

	var existing int64
	if err := s.db.Model(&DayClose{}).Where("business_date = ?", businessDate).Count(&existing).Error; err != nil {
		return nil, err
	}
	if existing > 0 {
		return nil, fmt.Errorf("hospitality: business date %s is already closed", businessDate)
	}

	expected, err := s.ExpectedTenders(businessDate)
	if err != nil {
		return nil, err
	}
	if len(expected) == 0 {
		return nil, errors.New("hospitality: no payments captured for this business date")
	}

	summary, err := settlement.Compute(expected, declared)
	if err != nil {
		return nil, err
	}
	open, err := s.OpenItems()
	if err != nil {
		return nil, err
	}
	record, err := settlement.Close(businessDay, summary, open, by, note, s.now())
	if err != nil {
		return nil, err
	}

	type tenderJSON struct {
		Method   string `json:"method"`
		Expected int64  `json:"expected_halalas"`
		Counted  int64  `json:"counted_halalas"`
		Variance int64  `json:"variance_halalas"`
		Declared bool   `json:"declared"`
	}
	tenders := make([]tenderJSON, 0, len(record.Summary.Tenders))
	for _, t := range record.Summary.Tenders {
		tenders = append(tenders, tenderJSON{
			Method:   t.Method,
			Expected: t.Expected.Minor(),
			Counted:  t.Counted.Minor(),
			Variance: t.Variance.Minor(),
			Declared: t.Declared,
		})
	}
	tendersBlob, _ := json.Marshal(tenders)

	stored := DayClose{
		BusinessDate:    businessDate,
		ExpectedHalalas: record.Summary.TotalExpected.Minor(),
		CountedHalalas:  record.Summary.TotalCounted.Minor(),
		VarianceHalalas: record.Summary.TotalVariance.Minor(),
		TendersJSON:     string(tendersBlob),
		ClosedByID:      record.ClosedBy.ID,
		ClosedByName:    record.ClosedBy.DisplayName,
		Note:            record.Note,
		ClosedAt:        record.ClosedAt,
	}
	if err := s.db.Create(&stored).Error; err != nil {
		return nil, err
	}
	return &stored, nil
}

// DayTotal is a convenience for demos/tests: the day's expected total as money.
func (s *Service) DayTotal(businessDate string) (money.Amount, error) {
	lines, err := s.ExpectedTenders(businessDate)
	if err != nil {
		return money.Amount{}, err
	}
	total := sar(0)
	for _, l := range lines {
		total, _ = total.Add(l.Expected)
	}
	return total, nil
}
