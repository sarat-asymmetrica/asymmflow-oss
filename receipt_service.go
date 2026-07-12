package main

// Wave 8 P3 slice 2 (Bucket B): customer receipt-allocation sub-model.
// Ported from the frozen PH reference (receipt_service.go). Payments already
// work on the substrate; this adds the customer-facing receipt header that can
// be applied to invoices (funding invoice Payment rows) or held on-account and
// allocated later. Each allocation creates a Payment + a CustomerReceiptAllocation
// link and advances the receipt's applied/unapplied balance.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CustomerReceipt is the receipt header used for both invoice-applied receipts
// and on-account customer advances/deposits that can be allocated later.
type CustomerReceipt struct {
	Base
	ReceiptNumber      string    `gorm:"uniqueIndex;size:50" json:"receipt_number"`
	CustomerID         string    `gorm:"index;size:36" json:"customer_id"`
	CustomerName       string    `gorm:"index;size:255" json:"customer_name"`
	Division           string    `gorm:"size:100;default:'PH Trading'" json:"division"`
	ReceiptDate        time.Time `gorm:"index;autoCreateTime:false" json:"receipt_date"`
	AmountBHD          float64   `gorm:"type:decimal(15,3);not null;default:0;check:amount_bhd >= 0" json:"amount_bhd"`
	AppliedAmountBHD   float64   `gorm:"type:decimal(15,3);not null;default:0;check:applied_amount_bhd >= 0" json:"applied_amount_bhd"`
	UnappliedAmountBHD float64   `gorm:"type:decimal(15,3);not null;default:0;check:unapplied_amount_bhd >= 0" json:"unapplied_amount_bhd"`
	PaymentMethod      string    `gorm:"size:50" json:"payment_method"`
	Reference          string    `gorm:"size:100" json:"reference"`
	Status             string    `gorm:"index;size:30;default:'OnAccount'" json:"status"` // OnAccount, PartiallyApplied, Applied, Reversed
	Notes              string    `gorm:"type:text" json:"notes"`
	UpdatedBy          string    `json:"updated_by"`
}

func (CustomerReceipt) TableName() string { return "customer_receipts" }

// CustomerReceiptAllocation links a receipt to the invoice payment row it funded.
type CustomerReceiptAllocation struct {
	Base
	ReceiptID          string    `gorm:"index;size:36" json:"receipt_id"`
	InvoiceID          string    `gorm:"index;size:36" json:"invoice_id"`
	InvoiceNumber      string    `gorm:"index;size:50" json:"invoice_number"`
	PaymentID          string    `gorm:"index;size:36" json:"payment_id"`
	AllocatedAmountBHD float64   `gorm:"type:decimal(15,3);not null;default:0;check:allocated_amount_bhd >= 0" json:"allocated_amount_bhd"`
	AllocationDate     time.Time `gorm:"index;autoCreateTime:false" json:"allocation_date"`
	Status             string    `gorm:"index;size:20;default:'Applied'" json:"status"` // Applied, Reversed
	UpdatedBy          string    `json:"updated_by"`
}

func (CustomerReceiptAllocation) TableName() string { return "customer_receipt_allocations" }

type CustomerReceiptInput struct {
	CustomerID    string  `json:"customer_id"`
	CustomerName  string  `json:"customer_name"`
	InvoiceID     string  `json:"invoice_id"`
	AmountBHD     float64 `json:"amount_bhd"`
	ReceiptDate   string  `json:"receipt_date"`
	PaymentMethod string  `json:"payment_method"`
	Reference     string  `json:"reference"`
	Division      string  `json:"division"`
	Notes         string  `json:"notes"`
}

func (a *App) CreateCustomerReceipt(input CustomerReceiptInput) (*CustomerReceipt, error) {
	if err := a.requirePermission("payments:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	amount := roundBHD(input.AmountBHD)
	if amount <= 0 {
		return nil, fmt.Errorf("receipt amount must be greater than zero")
	}

	receiptDate, err := parseCustomerReceiptDate(input.ReceiptDate)
	if err != nil {
		return nil, err
	}

	method := normalizeCustomerReceiptMethod(input.PaymentMethod)
	reference := strings.TrimSpace(input.Reference)
	if (method == "Bank Transfer" || method == "Wire Transfer") && reference == "" {
		return nil, fmt.Errorf("receipt reference is required for bank transfers")
	}

	var created CustomerReceipt
	err = a.db.Transaction(func(tx *gorm.DB) error {
		customerID := strings.TrimSpace(input.CustomerID)
		customerName := strings.TrimSpace(input.CustomerName)
		division := normalizeDivisionName(input.Division)
		if division == "" {
			division = "PH Trading"
		}

		var invoice Invoice
		if strings.TrimSpace(input.InvoiceID) != "" {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&invoice, "id = ?", strings.TrimSpace(input.InvoiceID)).Error; err != nil {
				if retryErr := tx.First(&invoice, "id = ?", strings.TrimSpace(input.InvoiceID)).Error; retryErr != nil {
					return fmt.Errorf("invoice not found: %w", err)
				}
			}
			if !canRecordCustomerInvoicePayment(invoice, time.Now()) {
				return fmt.Errorf("cannot apply receipt to invoice with status %q", invoice.Status)
			}
			if amount > invoice.OutstandingBHD+0.001 {
				return fmt.Errorf("receipt amount %.3f BHD exceeds open invoice balance %.3f BHD", amount, invoice.OutstandingBHD)
			}
			customerID = invoice.CustomerID
			customerName = invoice.CustomerName
			if invoice.Division != "" {
				division = normalizeDivisionName(invoice.Division)
			}
		}

		if customerID == "" {
			return fmt.Errorf("customer is required for on-account receipts")
		}
		if customerName == "" {
			var customer CustomerMaster
			if err := tx.First(&customer, "id = ? OR customer_id = ?", customerID, customerID).Error; err == nil {
				customerID = customer.ID
				customerName = customer.BusinessName
			}
		}
		if customerName == "" {
			return fmt.Errorf("customer name is required")
		}

		receiptNumber, err := generateCustomerReceiptNumberTx(tx, receiptDate.Year())
		if err != nil {
			return err
		}

		created = CustomerReceipt{
			ReceiptNumber:      receiptNumber,
			CustomerID:         customerID,
			CustomerName:       customerName,
			Division:           division,
			ReceiptDate:        receiptDate,
			AmountBHD:          amount,
			AppliedAmountBHD:   0,
			UnappliedAmountBHD: amount,
			PaymentMethod:      method,
			Reference:          reference,
			Status:             "OnAccount",
			Notes:              strings.TrimSpace(input.Notes),
			UpdatedBy:          a.getCurrentUserID(),
		}

		if err := tx.Create(&created).Error; err != nil {
			return fmt.Errorf("failed to create customer receipt: %w", err)
		}

		if invoice.ID != "" {
			if _, err := a.applyCustomerReceiptToInvoiceTx(tx, &created, &invoice, amount, receiptDate, method, reference); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &created, nil
}

func (a *App) ApplyCustomerReceiptToInvoice(receiptID, invoiceID string, amount float64) (*CustomerReceiptAllocation, error) {
	if err := a.requirePermission("payments:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var allocation CustomerReceiptAllocation
	err := a.db.Transaction(func(tx *gorm.DB) error {
		var receipt CustomerReceipt
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&receipt, "id = ?", strings.TrimSpace(receiptID)).Error; err != nil {
			if retryErr := tx.First(&receipt, "id = ?", strings.TrimSpace(receiptID)).Error; retryErr != nil {
				return fmt.Errorf("receipt not found: %w", err)
			}
		}
		if receipt.Status == "Reversed" {
			return fmt.Errorf("reversed receipts cannot be allocated")
		}

		var invoice Invoice
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&invoice, "id = ?", strings.TrimSpace(invoiceID)).Error; err != nil {
			if retryErr := tx.First(&invoice, "id = ?", strings.TrimSpace(invoiceID)).Error; retryErr != nil {
				return fmt.Errorf("invoice not found: %w", err)
			}
		}
		if normalizeDivisionName(invoice.Division) != normalizeDivisionName(receipt.Division) {
			return fmt.Errorf("cannot allocate %s receipt to %s invoice", normalizeDivisionName(receipt.Division), normalizeDivisionName(invoice.Division))
		}
		if invoice.CustomerID != receipt.CustomerID {
			return fmt.Errorf("receipt customer does not match invoice customer")
		}

		allocationAmount := roundBHD(amount)
		if allocationAmount <= 0 {
			allocationAmount = math.Min(receipt.UnappliedAmountBHD, invoice.OutstandingBHD)
			allocationAmount = roundBHD(allocationAmount)
		}
		if allocationAmount <= 0 {
			return fmt.Errorf("no unapplied receipt balance or invoice balance is available")
		}

		createdAllocation, err := a.applyCustomerReceiptToInvoiceTx(tx, &receipt, &invoice, allocationAmount, receipt.ReceiptDate, receipt.PaymentMethod, receipt.Reference)
		if err != nil {
			return err
		}
		allocation = *createdAllocation
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &allocation, nil
}

func (a *App) ListCustomerReceipts(limit, offset int) ([]CustomerReceipt, error) {
	if err := a.requirePermission("payments:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if limit <= 0 {
		limit = 200
	} else if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	var receipts []CustomerReceipt
	if err := a.db.Order("receipt_date DESC, created_at DESC").Limit(limit).Offset(offset).Find(&receipts).Error; err != nil {
		return nil, fmt.Errorf("failed to list customer receipts: %w", err)
	}
	return receipts, nil
}

func (a *App) GetCustomerReceiptAllocations(receiptID string) ([]CustomerReceiptAllocation, error) {
	if err := a.requirePermission("payments:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var allocations []CustomerReceiptAllocation
	if err := a.db.Where("receipt_id = ?", strings.TrimSpace(receiptID)).
		Order("allocation_date ASC, created_at ASC").
		Find(&allocations).Error; err != nil {
		return nil, fmt.Errorf("failed to list receipt allocations: %w", err)
	}
	return allocations, nil
}

// ReverseCustomerReceipt reverses a receipt that has NOT been applied to any
// invoice (AppliedAmountBHD == 0). Applied/posted receipts carry a Payment +
// allocation trail and invoice-state changes that reversal here does not
// unwind — that case is stop-and-report, not a judgment call, and is
// rejected with a clear error instead of being silently allowed.
func (a *App) ReverseCustomerReceipt(receiptID string, reason string) (*CustomerReceipt, error) {
	if err := a.requirePermission("payments:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	reason = strings.TrimSpace(reason)

	var reversed CustomerReceipt
	err := a.db.Transaction(func(tx *gorm.DB) error {
		var receipt CustomerReceipt
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&receipt, "id = ?", strings.TrimSpace(receiptID)).Error; err != nil {
			if retryErr := tx.First(&receipt, "id = ?", strings.TrimSpace(receiptID)).Error; retryErr != nil {
				return fmt.Errorf("receipt not found: %w", err)
			}
		}
		if receipt.Status == "Reversed" {
			return fmt.Errorf("receipt is already reversed")
		}
		if receipt.AppliedAmountBHD > 0.001 {
			return fmt.Errorf("cannot reverse a receipt with applied allocations — reverse the applications first (not supported here)")
		}

		note := "Reversed"
		if reason != "" {
			note = fmt.Sprintf("Reversed: %s", reason)
		}
		notes := strings.TrimSpace(receipt.Notes)
		if notes != "" {
			notes = notes + " | " + note
		} else {
			notes = note
		}

		receipt.Status = "Reversed"
		receipt.UnappliedAmountBHD = 0
		receipt.Notes = notes
		receipt.UpdatedBy = a.getCurrentUserID()

		if err := tx.Model(&receipt).Updates(map[string]any{
			"status":               receipt.Status,
			"unapplied_amount_bhd": receipt.UnappliedAmountBHD,
			"notes":                receipt.Notes,
			"updated_by":           receipt.UpdatedBy,
			"updated_at":           time.Now(),
		}).Error; err != nil {
			return fmt.Errorf("failed to reverse receipt: %w", err)
		}

		reversed = receipt
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &reversed, nil
}

func (a *App) applyCustomerReceiptToInvoiceTx(tx *gorm.DB, receipt *CustomerReceipt, invoice *Invoice, amount float64, paymentDate time.Time, method, reference string) (*CustomerReceiptAllocation, error) {
	if receipt == nil || invoice == nil {
		return nil, fmt.Errorf("receipt and invoice are required")
	}
	amount = roundBHD(amount)
	if amount <= 0 {
		return nil, fmt.Errorf("allocation amount must be greater than zero")
	}
	if amount > receipt.UnappliedAmountBHD+0.001 {
		return nil, fmt.Errorf("allocation %.3f BHD exceeds unapplied receipt balance %.3f BHD", amount, receipt.UnappliedAmountBHD)
	}
	if amount > invoice.OutstandingBHD+0.001 {
		return nil, fmt.Errorf("allocation %.3f BHD exceeds open invoice balance %.3f BHD", amount, invoice.OutstandingBHD)
	}
	if !canRecordCustomerInvoicePayment(*invoice, time.Now()) {
		return nil, fmt.Errorf("cannot apply receipt to invoice with status %q", invoice.Status)
	}

	daysToPayment := int(paymentDate.Sub(invoice.InvoiceDate).Hours() / 24)
	if daysToPayment < 0 {
		daysToPayment = 0
	}

	receiptID := receipt.ID
	idempotencyInput := fmt.Sprintf("%s|%s|%.3f|%s|%s", invoice.ID, receipt.ID, amount, paymentDate.Format("2006-01-02"), reference)
	idempotencyHash := sha256.Sum256([]byte(idempotencyInput))
	payment := Payment{
		InvoiceID:      invoice.ID,
		InvoiceNumber:  invoice.InvoiceNumber,
		AmountBHD:      amount,
		PaymentDate:    paymentDate,
		PaymentMethod:  normalizeCustomerReceiptMethod(method),
		Reference:      reference,
		DaysToPayment:  daysToPayment,
		IdempotencyKey: hex.EncodeToString(idempotencyHash[:]),
		ReceiptID:      &receiptID,
		Division:       normalizeDivisionName(invoice.Division),
	}
	if payment.Division == "" {
		payment.Division = normalizeDivisionName(receipt.Division)
	}
	if err := tx.Create(&payment).Error; err != nil {
		return nil, fmt.Errorf("failed to create invoice payment from receipt: %w", err)
	}

	invoice.OutstandingBHD = roundBHD(invoice.OutstandingBHD - amount)
	if invoice.OutstandingBHD < 0 {
		invoice.OutstandingBHD = 0
	}
	if _, err := a.applyCustomerInvoicePaymentState(tx, invoice); err != nil {
		return nil, fmt.Errorf("failed to update invoice payment state: %w", err)
	}

	allocation := CustomerReceiptAllocation{
		ReceiptID:          receipt.ID,
		InvoiceID:          invoice.ID,
		InvoiceNumber:      invoice.InvoiceNumber,
		PaymentID:          payment.ID,
		AllocatedAmountBHD: amount,
		AllocationDate:     paymentDate,
		Status:             "Applied",
		UpdatedBy:          a.getCurrentUserID(),
	}
	if err := tx.Create(&allocation).Error; err != nil {
		return nil, fmt.Errorf("failed to create receipt allocation: %w", err)
	}

	receipt.AppliedAmountBHD = roundBHD(receipt.AppliedAmountBHD + amount)
	receipt.UnappliedAmountBHD = roundBHD(receipt.AmountBHD - receipt.AppliedAmountBHD)
	if receipt.UnappliedAmountBHD <= 0.001 {
		receipt.UnappliedAmountBHD = 0
		receipt.Status = "Applied"
	} else {
		receipt.Status = "PartiallyApplied"
	}
	receipt.UpdatedBy = a.getCurrentUserID()
	if err := tx.Model(receipt).Updates(map[string]any{
		"applied_amount_bhd":   receipt.AppliedAmountBHD,
		"unapplied_amount_bhd": receipt.UnappliedAmountBHD,
		"status":               receipt.Status,
		"updated_by":           receipt.UpdatedBy,
		"updated_at":           time.Now(),
	}).Error; err != nil {
		return nil, fmt.Errorf("failed to update receipt balance: %w", err)
	}

	return &allocation, nil
}

func generateCustomerReceiptNumberTx(tx *gorm.DB, year int) (string, error) {
	prefix := fmt.Sprintf("RCT-%d-", year)
	var count int64
	if err := tx.Model(&CustomerReceipt{}).Where("receipt_number LIKE ?", prefix+"%").Count(&count).Error; err != nil {
		return "", fmt.Errorf("failed to generate receipt number: %w", err)
	}
	return fmt.Sprintf("%s%04d", prefix, count+1), nil
}

func parseCustomerReceiptDate(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), nil
	}
	if parsed, err := time.Parse("2006-01-02", raw); err == nil {
		return parsed, nil
	}
	if parsed := parseDate(raw); !parsed.IsZero() {
		return parsed, nil
	}
	return time.Time{}, fmt.Errorf("invalid receipt date")
}

func normalizeCustomerReceiptMethod(method string) string {
	switch strings.TrimSpace(method) {
	case "Cash", "Cheque", "Bank Transfer", "Credit Card", "LC", "PDC", "Other":
		return strings.TrimSpace(method)
	case "Wire Transfer", "NEFT", "Online":
		return "Bank Transfer"
	case "Card":
		return "Credit Card"
	default:
		return "Other"
	}
}
