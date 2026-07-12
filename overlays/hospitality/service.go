package hospitality

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"

	"ph_holdings_app/pkg/compliance/saudi"
	"ph_holdings_app/pkg/documents/numbering"
	"ph_holdings_app/pkg/infra/auth"
	"ph_holdings_app/pkg/infra/events"
	"ph_holdings_app/pkg/kernel/actor"
	"ph_holdings_app/pkg/overlay"
)

// Numbering specs. They share the trading substrate's invoice_sequences table
// (keyed by prefix), so a hospitality deployment gets the same concurrency
// discipline the trading documents get — for free.
var (
	invoiceNumberSpec    = numbering.Spec{Prefix: "SINV", Template: "SINV-{date}-{seq}", Pad: 4}
	ticketNumberSpec     = numbering.Spec{Prefix: "KOT", Template: "KOT-{date}-{seq}", Pad: 3}
	creditNoteNumberSpec = numbering.Spec{Prefix: "SCRN", Template: "SCRN-{date}-{seq}", Pad: 4}
)

// Settings keys for the manager-PIN engine state.
const (
	settingManagerPINHash = "manager_pin_hash"
	settingManagerPINLock = "manager_pin_lock"
)

// Service is the hospitality domain service: the ONLY layer that knows both
// the vertical's vocabulary (sessions, tickets, tenders) and the substrate's
// engines. It owns a GORM handle, the deployment overlay, the event bus, and
// the ZATCA signing material.
type Service struct {
	db      *gorm.DB
	ov      *overlay.CompanyOverlay
	bus     events.Bus
	numbers *numbering.Engine
	signKey *saudi.KeyPair
	cert    *saudi.Certificate
	now     func() time.Time
}

// NewService wires the service and migrates the vertical's schema (plus the
// shared numbering table). The signing key/cert pair is the EGS unit identity;
// for offline/demo deployments use saudi.GenerateKeyPair +
// saudi.NewSelfSignedCertificate, for production the CSID materials.
func NewService(db *gorm.DB, ov *overlay.CompanyOverlay, bus events.Bus, signKey *saudi.KeyPair, cert *saudi.Certificate) (*Service, error) {
	if db == nil {
		return nil, errors.New("hospitality: nil database")
	}
	if ov == nil {
		return nil, errors.New("hospitality: nil overlay")
	}
	if signKey == nil || cert == nil {
		return nil, errors.New("hospitality: signing key and certificate are required (ZATCA e-invoicing is not optional in SA)")
	}
	if err := db.AutoMigrate(
		&MenuItem{}, &DiningTable{}, &OrderSession{}, &OrderLine{},
		&Ticket{}, &TicketLine{}, &Invoice{}, &Payment{}, &CreditNote{}, &CreditNoteLine{},
		&DayClose{}, &Setting{}, &PrintJob{}, &numbering.Sequence{},
	); err != nil {
		return nil, fmt.Errorf("hospitality: migrate: %w", err)
	}
	// W4 C.1: pre-partial-refund databases carry a UNIQUE index on
	// hosp_credit_notes.original_invoice_id ("one full refund per invoice").
	// AutoMigrate never drops indexes, so retire it explicitly; its
	// replacement (plain idx_hosp_cn_original_invoice) is created above.
	if err := db.Exec("DROP INDEX IF EXISTS idx_hosp_credit_notes_original_invoice_id").Error; err != nil {
		return nil, fmt.Errorf("hospitality: migrate (retire legacy credit-note index): %w", err)
	}
	return &Service{
		db:      db,
		ov:      ov,
		bus:     bus,
		numbers: numbering.New(db),
		signKey: signKey,
		cert:    cert,
		now:     func() time.Time { return time.Now().UTC() },
	}, nil
}

// SetClock overrides the service clock (tests and deterministic demos).
func (s *Service) SetClock(now func() time.Time) { s.now = now }

// ---- Manager PIN (pkg/infra/auth engine; the service persists the state) ----

// SetManagerPIN hashes and stores the manager PIN, resetting any lockout.
func (s *Service) SetManagerPIN(pin string) error {
	h, err := auth.HashPIN(pin)
	if err != nil {
		return err
	}
	if err := s.putSetting(settingManagerPINHash, h.Encode()); err != nil {
		return err
	}
	return s.putSetting(settingManagerPINLock, "")
}

// VerifyManagerPIN checks the PIN and PERSISTS the updated lockout state —
// the engine's contract is that lockout only protects if the caller stores
// what Verify returns.
func (s *Service) VerifyManagerPIN(pin string) error {
	encoded, err := s.getSetting(settingManagerPINHash)
	if err != nil || encoded == "" {
		return errors.New("hospitality: no manager PIN configured")
	}
	h, err := auth.ParsePINHash(encoded)
	if err != nil {
		return err
	}
	var state auth.LockState
	if raw, _ := s.getSetting(settingManagerPINLock); raw != "" {
		_ = json.Unmarshal([]byte(raw), &state)
	}
	newState, verr := auth.Verify(h, pin, state, s.now())
	stateJSON, _ := json.Marshal(newState)
	if err := s.putSetting(settingManagerPINLock, string(stateJSON)); err != nil {
		return err
	}
	return verr
}

func (s *Service) putSetting(key, value string) error {
	return s.db.Save(&Setting{Key: key, Value: value}).Error
}

func (s *Service) getSetting(key string) (string, error) {
	var row Setting
	err := s.db.First(&row, "key = ?", key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	return row.Value, err
}

// ---- Order sessions ----

// OpenSession opens a session on a table. One open session per table.
func (s *Service) OpenSession(tableCode string) (*OrderSession, error) {
	var table DiningTable
	if err := s.db.First(&table, "code = ?", strings.TrimSpace(tableCode)).Error; err != nil {
		return nil, fmt.Errorf("hospitality: unknown table %q: %w", tableCode, err)
	}
	var openCount int64
	if err := s.db.Model(&OrderSession{}).
		Where("table_id = ? AND status = ?", table.ID, SessionOpen).
		Count(&openCount).Error; err != nil {
		return nil, err
	}
	if openCount > 0 {
		return nil, fmt.Errorf("hospitality: table %s already has an open session", table.Code)
	}
	session := OrderSession{TableID: table.ID, Status: SessionOpen, OpenedAt: s.now()}
	if err := s.db.Create(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// AddLine adds a menu item to an open session, snapshotting name/price/rate.
func (s *Service) AddLine(sessionID uint, itemName string, qty float64) (*OrderLine, error) {
	if qty <= 0 {
		return nil, errors.New("hospitality: quantity must be positive")
	}
	session, err := s.openSession(sessionID)
	if err != nil {
		return nil, err
	}
	var item MenuItem
	if err := s.db.First(&item, "name = ? AND active = ?", strings.TrimSpace(itemName), true).Error; err != nil {
		return nil, fmt.Errorf("hospitality: no active menu item %q: %w", itemName, err)
	}
	line := OrderLine{
		SessionID:        session.ID,
		MenuItemID:       item.ID,
		Name:             item.Name,
		Qty:              qty,
		UnitPriceHalalas: item.UnitPriceHalalas,
		TaxRate:          item.TaxRate,
		Status:           LinePending,
	}
	if err := s.db.Create(&line).Error; err != nil {
		return nil, err
	}
	return &line, nil
}

// SendKOT dispatches all pending lines of a session to the kitchen as one
// numbered ticket. The ticket number and the line-state flip commit atomically
// with the number allocation (numbering.NextInTx inside the same transaction).
func (s *Service) SendKOT(sessionID uint) (*Ticket, error) {
	if _, err := s.openSession(sessionID); err != nil {
		return nil, err
	}
	var ticket Ticket
	err := s.db.Transaction(func(tx *gorm.DB) error {
		number, err := numbering.NextInTx(tx, ticketNumberSpec, s.now())
		if err != nil {
			return err
		}
		var pending []OrderLine
		if err := tx.Where("session_id = ? AND status = ?", sessionID, LinePending).Find(&pending).Error; err != nil {
			return err
		}
		if len(pending) == 0 {
			return errors.New("hospitality: no pending lines to send")
		}
		ticket = Ticket{SessionID: sessionID, Number: number, Status: TicketQueued, CreatedAt: s.now(), UpdatedAt: s.now()}
		if err := tx.Create(&ticket).Error; err != nil {
			return err
		}
		for _, line := range pending {
			if err := tx.Create(&TicketLine{TicketID: ticket.ID, OrderLineID: line.ID, Name: line.Name, Qty: line.Qty}).Error; err != nil {
				return err
			}
		}
		// Spool the kitchen copy (W5 C.2): job and ticket commit together.
		if err := s.enqueuePrintTx(tx, PrintKindKitchenTicket, "kitchen", number); err != nil {
			return err
		}
		return tx.Model(&OrderLine{}).
			Where("session_id = ? AND status = ?", sessionID, LinePending).
			Update("status", LineSent).Error
	})
	if err != nil {
		return nil, err
	}
	return &ticket, nil
}

// ticketTransitions is the kitchen state machine. Cancellation is only
// possible before the food is ready; after that the cost is real and the
// resolution is a manager void on the LINE, not a ticket cancel.
var ticketTransitions = map[string][]string{
	TicketQueued:    {TicketPreparing, TicketCancelled},
	TicketPreparing: {TicketReady, TicketCancelled},
	TicketReady:     {TicketServed},
}

// AdvanceTicket moves a ticket along the kitchen state machine.
func (s *Service) AdvanceTicket(ticketID uint, to string) (*Ticket, error) {
	var ticket Ticket
	if err := s.db.First(&ticket, ticketID).Error; err != nil {
		return nil, fmt.Errorf("hospitality: ticket %d: %w", ticketID, err)
	}
	if !slices.Contains(ticketTransitions[ticket.Status], to) {
		return nil, fmt.Errorf("hospitality: illegal ticket transition %s → %s", ticket.Status, to)
	}
	ticket.Status = to
	ticket.UpdatedAt = s.now()
	if err := s.db.Save(&ticket).Error; err != nil {
		return nil, err
	}
	return &ticket, nil
}

// VoidLine voids an order line. Two gates, deliberately layered:
//   - the acting operator must hold approve authority (kernel actor —
//     an AI agent can NEVER void, whatever it claims);
//   - the manager PIN must verify (possession factor at the till).
//
// A reason is mandatory; voids are the classic hospitality fraud channel.
func (s *Service) VoidLine(sessionID, lineID uint, by actor.Actor, pin, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return errors.New("hospitality: void requires a reason")
	}
	if !by.CanApprove() {
		return fmt.Errorf("hospitality: actor %q (type %s) lacks authority to void a line", by.ID, by.Type)
	}
	if err := s.VerifyManagerPIN(pin); err != nil {
		return err
	}
	if _, err := s.openSession(sessionID); err != nil {
		return err
	}
	var line OrderLine
	if err := s.db.First(&line, "id = ? AND session_id = ?", lineID, sessionID).Error; err != nil {
		return fmt.Errorf("hospitality: line %d: %w", lineID, err)
	}
	if line.Status == LineVoided {
		return errors.New("hospitality: line already voided")
	}
	line.Status = LineVoided
	line.VoidReason = strings.TrimSpace(reason)
	line.VoidedByID = by.ID
	return s.db.Save(&line).Error
}

func (s *Service) openSession(sessionID uint) (*OrderSession, error) {
	var session OrderSession
	if err := s.db.First(&session, sessionID).Error; err != nil {
		return nil, fmt.Errorf("hospitality: session %d: %w", sessionID, err)
	}
	if session.Status != SessionOpen {
		return nil, fmt.Errorf("hospitality: session %d is %s, not open", sessionID, session.Status)
	}
	return &session, nil
}
