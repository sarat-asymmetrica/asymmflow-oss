// Package cheque owns the cheque lifecycle: cheque-book registers,
// issuance (transactional number allocation), clearance tracking, and
// stale/bounced/cancelled handling.
//
// Wave 5 A.1: a W4-D1 peel — the logic moved inward from the root
// cheque_register_service.go. The models already live in pkg/finance, so
// the service needs only the database; RBAC guards stay with the host's
// thin delegates.
package cheque

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"ph_holdings_app/pkg/finance"
)

// Service is the cheque register + lifecycle service.
type Service struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Service { return &Service{db: db} }

// CreateRegister creates a new cheque book register, refusing ranges that
// overlap an ACTIVE register on the same account.
func (s *Service) CreateRegister(bankAccountID, chequeBookNo string, startNum, endNum int) (*finance.ChequeRegister, error) {
	if startNum >= endNum {
		return nil, fmt.Errorf("start number must be less than end number")
	}

	var existing finance.ChequeRegister
	err := s.db.Where("bank_account_id = ? AND status = ? AND ((start_number <= ? AND end_number >= ?) OR (start_number <= ? AND end_number >= ?))",
		bankAccountID, "ACTIVE", startNum, startNum, endNum, endNum).First(&existing).Error
	if err == nil {
		return nil, fmt.Errorf("overlapping cheque register exists: %s", existing.ChequeBookNo)
	}

	register := &finance.ChequeRegister{
		BankAccountID: bankAccountID,
		ChequeBookNo:  chequeBookNo,
		StartNumber:   startNum,
		EndNumber:     endNum,
		CurrentNumber: startNum,
		Status:        "ACTIVE",
		IssuedDate:    time.Now(),
	}

	if err := s.db.Create(register).Error; err != nil {
		return nil, fmt.Errorf("failed to create register: %w", err)
	}

	log.Printf("📗 Cheque register created: %s (#%d - #%d)", chequeBookNo, startNum, endNum)
	return register, nil
}

// Registers retrieves all registers for a bank account.
func (s *Service) Registers(bankAccountID string) ([]finance.ChequeRegister, error) {
	var registers []finance.ChequeRegister
	err := s.db.Where("bank_account_id = ?", bankAccountID).
		Order("issued_date DESC").
		Find(&registers).Error
	return registers, err
}

// ActiveRegister gets the active register for a bank account.
func (s *Service) ActiveRegister(bankAccountID string) (*finance.ChequeRegister, error) {
	var register finance.ChequeRegister
	err := s.db.Where("bank_account_id = ? AND status = ?", bankAccountID, "ACTIVE").
		First(&register).Error
	if err != nil {
		return nil, fmt.Errorf("no active cheque register found")
	}
	return &register, nil
}

// NextNumber returns the next available cheque number.
func (s *Service) NextNumber(bankAccountID string) (string, error) {
	register, err := s.ActiveRegister(bankAccountID)
	if err != nil {
		return "", err
	}
	if register.CurrentNumber > register.EndNumber {
		return "", fmt.Errorf("cheque book exhausted - please create a new register")
	}
	return fmt.Sprintf("%06d", register.CurrentNumber), nil
}

// Exhaust marks a register as exhausted.
func (s *Service) Exhaust(registerID string) error {
	now := time.Now()
	return s.db.Model(&finance.ChequeRegister{}).
		Where("id = ?", registerID).
		Updates(map[string]any{
			"status":         "EXHAUSTED",
			"exhausted_date": now,
		}).Error
}

// Issue issues a new cheque and records it as outstanding. Number
// allocation, creation, and register increment run in one transaction.
func (s *Service) Issue(bankAccountID string, amount float64, payeeName, payeeType string, supplierID *string, purpose string) (*finance.OutstandingCheque, error) {
	var cheque *finance.OutstandingCheque
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var register finance.ChequeRegister
		if err := tx.Where("bank_account_id = ? AND status = ?", bankAccountID, "ACTIVE").
			First(&register).Error; err != nil {
			return fmt.Errorf("no active cheque register found for bank account %s", bankAccountID)
		}

		if register.CurrentNumber > register.EndNumber {
			return fmt.Errorf("cheque book exhausted - please create a new register")
		}

		chequeNumber := fmt.Sprintf("%06d", register.CurrentNumber)

		cheque = &finance.OutstandingCheque{
			BankAccountID: bankAccountID,
			ChequeNumber:  chequeNumber,
			Amount:        amount,
			Currency:      "BHD",
			IssuedDate:    time.Now(),
			PayeeName:     payeeName,
			PayeeType:     payeeType,
			SupplierID:    supplierID,
			Purpose:       purpose,
			Status:        "ISSUED",
		}

		if err := tx.Create(cheque).Error; err != nil {
			return fmt.Errorf("failed to issue cheque: %w", err)
		}

		newNumber := register.CurrentNumber + 1
		if err := tx.Model(&register).Update("current_number", newNumber).Error; err != nil {
			return fmt.Errorf("failed to increment cheque number: %w", err)
		}

		if newNumber > register.EndNumber {
			now := time.Now()
			if err := tx.Model(&finance.ChequeRegister{}).
				Where("id = ?", register.ID).
				Updates(map[string]any{
					"status":         "EXHAUSTED",
					"exhausted_date": now,
				}).Error; err != nil {
				log.Printf("⚠️ Failed to mark register as exhausted: %v", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	log.Printf("📝 Cheque issued: #%s for %.3f BHD to %s", cheque.ChequeNumber, amount, payeeName)
	return cheque, nil
}

// MarkPresented marks an ISSUED cheque as presented.
func (s *Service) MarkPresented(chequeNumber string) error {
	result := s.db.Model(&finance.OutstandingCheque{}).
		Where("cheque_number = ? AND status = ?", chequeNumber, "ISSUED").
		Update("status", "PRESENTED")
	if result.RowsAffected == 0 {
		return fmt.Errorf("cheque not found or not in ISSUED status")
	}
	log.Printf("📥 Cheque presented: #%s", chequeNumber)
	return nil
}

// MarkCleared marks a cheque as cleared and links the bank statement line.
func (s *Service) MarkCleared(chequeNumber, bankStatementLineID string, clearedDate time.Time) error {
	result := s.db.Model(&finance.OutstandingCheque{}).
		Where("cheque_number = ? AND status IN (?, ?)", chequeNumber, "ISSUED", "PRESENTED").
		Updates(map[string]any{
			"status":          "CLEARED",
			"cleared_date":    clearedDate,
			"matched_line_id": bankStatementLineID,
		})
	if result.RowsAffected == 0 {
		return fmt.Errorf("cheque not found or already cleared")
	}
	log.Printf("✅ Cheque cleared: #%s on %s", chequeNumber, clearedDate.Format("2006-01-02"))
	return nil
}

// MarkStale marks a cheque as stale (>6 months).
func (s *Service) MarkStale(chequeNumber string) error {
	now := time.Now()
	result := s.db.Model(&finance.OutstandingCheque{}).
		Where("cheque_number = ? AND status IN (?, ?)", chequeNumber, "ISSUED", "PRESENTED").
		Updates(map[string]any{
			"status":     "STALE",
			"is_stale":   true,
			"stale_date": now,
		})
	if result.RowsAffected == 0 {
		return fmt.Errorf("cheque not found or cannot be marked stale")
	}
	log.Printf("⏳ Cheque marked stale: #%s", chequeNumber)
	return nil
}

// MarkBounced marks a cheque as bounced.
func (s *Service) MarkBounced(chequeNumber, reason string) error {
	result := s.db.Model(&finance.OutstandingCheque{}).
		Where("cheque_number = ? AND status IN (?, ?)", chequeNumber, "ISSUED", "PRESENTED").
		Updates(map[string]any{
			"status":  "BOUNCED",
			"purpose": reason, // Append reason to purpose
		})
	if result.RowsAffected == 0 {
		return fmt.Errorf("cheque not found or cannot be marked bounced")
	}
	log.Printf("❌ Cheque bounced: #%s - %s", chequeNumber, reason)
	return nil
}

// Cancel cancels an ISSUED cheque.
func (s *Service) Cancel(chequeNumber, reason string) error {
	result := s.db.Model(&finance.OutstandingCheque{}).
		Where("cheque_number = ? AND status = ?", chequeNumber, "ISSUED").
		Updates(map[string]any{
			"status":  "CANCELLED",
			"purpose": reason,
		})
	if result.RowsAffected == 0 {
		return fmt.Errorf("cheque not found or not in ISSUED status")
	}
	log.Printf("🚫 Cheque cancelled: #%s - %s", chequeNumber, reason)
	return nil
}

// Reissue issues a new cheque to replace a stale/cancelled one.
func (s *Service) Reissue(oldChequeNumber, bankAccountID string) (*finance.OutstandingCheque, error) {
	var oldCheque finance.OutstandingCheque
	if err := s.db.Where("cheque_number = ?", oldChequeNumber).First(&oldCheque).Error; err != nil {
		return nil, fmt.Errorf("original cheque not found: %w", err)
	}

	if oldCheque.Status != "STALE" && oldCheque.Status != "CANCELLED" {
		return nil, fmt.Errorf("only stale or cancelled cheques can be reissued")
	}

	newCheque, err := s.Issue(bankAccountID, oldCheque.Amount, oldCheque.PayeeName, oldCheque.PayeeType, oldCheque.SupplierID, "Reissue of #"+oldChequeNumber)
	if err != nil {
		return nil, err
	}

	s.db.Model(&oldCheque).Update("reissued_as", newCheque.ChequeNumber)

	log.Printf("🔄 Cheque reissued: #%s → #%s", oldChequeNumber, newCheque.ChequeNumber)
	return newCheque, nil
}

// OutstandingResult bundles outstanding cheques with their total value.
// Wails v2's bound-method marshaling only handles OutputCount 1 or 2 (see
// internal/binding/boundMethod.go) — a 3-value Go return silently marshals
// to null on the JS side. Bundling into a struct + error keeps the binding
// a clean 2-value return.
type OutstandingResult struct {
	Cheques []finance.OutstandingCheque `json:"cheques"`
	Total   float64                     `json:"total"`
}

// Outstanding returns all outstanding (not cleared) cheques and their total.
func (s *Service) Outstanding(bankAccountID string) (*OutstandingResult, error) {
	var cheques []finance.OutstandingCheque
	query := s.db.Where("status IN (?, ?)", "ISSUED", "PRESENTED")
	if bankAccountID != "" {
		query = query.Where("bank_account_id = ?", bankAccountID)
	}

	if err := query.Order("issued_date DESC").Find(&cheques).Error; err != nil {
		return nil, err
	}

	var total float64
	for _, c := range cheques {
		total += c.Amount
	}
	return &OutstandingResult{Cheques: cheques, Total: total}, nil
}

// ByNumber retrieves a cheque by its number.
func (s *Service) ByNumber(chequeNumber string) (*finance.OutstandingCheque, error) {
	var cheque finance.OutstandingCheque
	if err := s.db.Where("cheque_number = ?", chequeNumber).First(&cheque).Error; err != nil {
		return nil, fmt.Errorf("cheque not found: %w", err)
	}
	return &cheque, nil
}

// ByStatus retrieves cheques by status.
func (s *Service) ByStatus(bankAccountID, status string) ([]finance.OutstandingCheque, error) {
	var cheques []finance.OutstandingCheque
	query := s.db.Where("status = ?", status)
	if bankAccountID != "" {
		query = query.Where("bank_account_id = ?", bankAccountID)
	}
	err := query.Order("issued_date DESC").Find(&cheques).Error
	return cheques, err
}

// StaleCheques returns cheques >6 months old and not cleared.
func (s *Service) StaleCheques(bankAccountID string) ([]finance.OutstandingCheque, error) {
	sixMonthsAgo := time.Now().AddDate(0, -6, 0)

	var cheques []finance.OutstandingCheque
	query := s.db.Where("status IN (?, ?) AND issued_date < ?", "ISSUED", "PRESENTED", sixMonthsAgo)
	if bankAccountID != "" {
		query = query.Where("bank_account_id = ?", bankAccountID)
	}
	err := query.Order("issued_date ASC").Find(&cheques).Error
	return cheques, err
}

// Report generates a cheque register report for a date range.
func (s *Service) Report(bankAccountID string, startDate, endDate time.Time) (map[string]any, error) {
	var cheques []finance.OutstandingCheque
	s.db.Where("bank_account_id = ? AND issued_date BETWEEN ? AND ?",
		bankAccountID, startDate, endDate).
		Order("issued_date ASC").
		Find(&cheques)

	statusCounts := make(map[string]int)
	statusAmounts := make(map[string]float64)
	for _, c := range cheques {
		statusCounts[c.Status]++
		statusAmounts[c.Status] += c.Amount
	}

	var totalAmount float64
	for _, c := range cheques {
		totalAmount += c.Amount
	}

	outstanding, _ := s.Outstanding(bankAccountID)
	outstandingCount := 0
	var outstandingTotal float64
	if outstanding != nil {
		outstandingCount = len(outstanding.Cheques)
		outstandingTotal = outstanding.Total
	}

	return map[string]any{
		"bank_account_id":   bankAccountID,
		"start_date":        startDate,
		"end_date":          endDate,
		"total_cheques":     len(cheques),
		"total_amount":      totalAmount,
		"status_counts":     statusCounts,
		"status_amounts":    statusAmounts,
		"outstanding_count": outstandingCount,
		"outstanding_total": outstandingTotal,
		"cheques":           cheques,
	}, nil
}
