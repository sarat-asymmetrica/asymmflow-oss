package payroll

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"ph_holdings_app/pkg/approvals"
	"ph_holdings_app/pkg/finance"
	"ph_holdings_app/pkg/kernel/actor"
	"ph_holdings_app/pkg/kernel/approval"
	"ph_holdings_app/pkg/overlay"
)

// EmployeeRef is the slice of the host's employee record payroll needs.
type EmployeeRef struct {
	ID       string
	FullName string
	JobTitle string
	IsActive bool
}

// DirectoryPort resolves employees. Implemented by the host — the employee
// directory lives with the collaboration hub (W4-D9: ports, not
// relocation).
type DirectoryPort interface {
	Employees(ids []string) map[string]EmployeeRef
}

// IdentityPort answers who is driving the current session.
type IdentityPort interface {
	// UserID is the raw authenticated user id (CreatedBy fields).
	UserID() string
	// ActorID prefers the employee context, then the user id, then "system"
	// (approval/posting attribution).
	ActorID() string
	// DisplayName describes the approver for the kernel actor.
	DisplayName() string
}

// EventsPort publishes UI events (payroll:updated etc.).
type EventsPort interface {
	Emit(name string, payload map[string]any)
}

// ExpenseBridgePort mirrors a posted/paid run into the host's expense
// ledger. The expense service (numbering, approvals, its own events) is a
// host cluster; payroll only triggers the sync.
type ExpenseBridgePort interface {
	SyncRunExpense(tx *gorm.DB, run *Run) error
	EmitExpenseUpdated(runID string)
}

// Service is the payroll domain service.
type Service struct {
	db        *gorm.DB
	identity  IdentityPort
	directory DirectoryPort
	events    EventsPort
	expenses  ExpenseBridgePort
}

func New(db *gorm.DB, identity IdentityPort, directory DirectoryPort, events EventsPort, expenses ExpenseBridgePort) *Service {
	return &Service{db: db, identity: identity, directory: directory, events: events, expenses: expenses}
}

func normalizeDivision(name string) string {
	return overlay.Active().NormalizeDivisionName(name)
}

func (s *Service) emit(name string, payload map[string]any) {
	if s.events != nil {
		s.events.Emit(name, payload)
	}
}

// ListProfiles lists compensation profiles, decorated with employee names.
func (s *Service) ListProfiles(activeOnly bool) ([]CompensationProfile, error) {
	query := s.db.Order("updated_at DESC")
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	var profiles []CompensationProfile
	if err := query.Find(&profiles).Error; err != nil {
		return nil, fmt.Errorf("failed to list compensation profiles: %w", err)
	}
	return s.decorateProfiles(profiles), nil
}

// UpsertProfile creates or updates an employee's compensation profile.
func (s *Service) UpsertProfile(profile CompensationProfile) (CompensationProfile, error) {
	profile.EmployeeID = strings.TrimSpace(profile.EmployeeID)
	if profile.EmployeeID == "" {
		return CompensationProfile{}, fmt.Errorf("employee is required")
	}

	if _, ok := s.directory.Employees([]string{profile.EmployeeID})[profile.EmployeeID]; !ok {
		return CompensationProfile{}, fmt.Errorf("employee not found")
	}

	profile.PayFrequency = strings.ToLower(strings.TrimSpace(profile.PayFrequency))
	if profile.PayFrequency == "" {
		profile.PayFrequency = "monthly"
	}
	profile.Division = normalizeDivision(profile.Division)
	profile.Currency = strings.ToUpper(strings.TrimSpace(profile.Currency))
	if profile.Currency == "" {
		profile.Currency = "BHD"
	}

	profile.BaseSalary = clampAmount(profile.BaseSalary)
	profile.HousingAllowance = clampAmount(profile.HousingAllowance)
	profile.TransportAllowance = clampAmount(profile.TransportAllowance)
	profile.OtherAllowance = clampAmount(profile.OtherAllowance)
	profile.StandardDeduction = clampAmount(profile.StandardDeduction)
	profile.TaxDeduction = clampAmount(profile.TaxDeduction)
	profile.EmployerCost = clampAmount(profile.EmployerCost)
	profile.Notes = strings.TrimSpace(profile.Notes)

	var existing CompensationProfile
	err := s.db.Where("employee_id = ?", profile.EmployeeID).First(&existing).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return CompensationProfile{}, fmt.Errorf("failed to lookup compensation profile: %w", err)
	}

	if err == nil {
		existingDivision := strings.TrimSpace(existing.Division)
		incomingDivision := strings.TrimSpace(profile.Division)
		if existingDivision != "" && incomingDivision != "" && existingDivision != incomingDivision {
			return CompensationProfile{}, fmt.Errorf("compensation profile for this employee already exists under division %q; refusing to overwrite it with division %q — edit the profile under its own division, or archive it first", existing.Division, profile.Division)
		}
	}

	now := time.Now()
	if err == gorm.ErrRecordNotFound {
		profile.CreatedBy = s.identity.UserID()
		if !profile.IsActive {
			profile.IsActive = true
		}
		if err := s.db.Create(&profile).Error; err != nil {
			return CompensationProfile{}, fmt.Errorf("failed to create compensation profile: %w", err)
		}
		s.emit("payroll:updated", map[string]any{"entity": "compensation_profile", "action": "create", "id": profile.ID})
		return s.decorateProfile(profile), nil
	}

	updates := map[string]any{
		"pay_frequency":       profile.PayFrequency,
		"division":            profile.Division,
		"currency":            profile.Currency,
		"base_salary":         profile.BaseSalary,
		"housing_allowance":   profile.HousingAllowance,
		"transport_allowance": profile.TransportAllowance,
		"other_allowance":     profile.OtherAllowance,
		"standard_deduction":  profile.StandardDeduction,
		"tax_deduction":       profile.TaxDeduction,
		"employer_cost":       profile.EmployerCost,
		"effective_from":      profile.EffectiveFrom,
		"effective_to":        profile.EffectiveTo,
		"is_active":           profile.IsActive,
		"notes":               profile.Notes,
		"updated_at":          now,
	}
	if err := s.db.Model(&existing).Updates(updates).Error; err != nil {
		return CompensationProfile{}, fmt.Errorf("failed to update compensation profile: %w", err)
	}

	existing.PayFrequency = profile.PayFrequency
	existing.Division = profile.Division
	existing.Currency = profile.Currency
	existing.BaseSalary = profile.BaseSalary
	existing.HousingAllowance = profile.HousingAllowance
	existing.TransportAllowance = profile.TransportAllowance
	existing.OtherAllowance = profile.OtherAllowance
	existing.StandardDeduction = profile.StandardDeduction
	existing.TaxDeduction = profile.TaxDeduction
	existing.EmployerCost = profile.EmployerCost
	existing.EffectiveFrom = profile.EffectiveFrom
	existing.EffectiveTo = profile.EffectiveTo
	existing.IsActive = profile.IsActive
	existing.Notes = profile.Notes

	s.emit("payroll:updated", map[string]any{"entity": "compensation_profile", "action": "update", "id": existing.ID})
	return s.decorateProfile(existing), nil
}

// ListPeriods lists payroll periods, optionally including closed ones.
func (s *Service) ListPeriods(includeClosed bool) ([]Period, error) {
	query := s.db.Order("period_start DESC")
	if !includeClosed {
		query = query.Where("status != ?", "closed")
	}

	var periods []Period
	if err := query.Find(&periods).Error; err != nil {
		return nil, fmt.Errorf("failed to list payroll periods: %w", err)
	}
	return periods, nil
}

// CreatePeriod creates a payroll period.
func (s *Service) CreatePeriod(period Period) (Period, error) {
	if period.PeriodStart.IsZero() || period.PeriodEnd.IsZero() {
		return Period{}, fmt.Errorf("period start and end are required")
	}
	if period.PeriodEnd.Before(period.PeriodStart) {
		return Period{}, fmt.Errorf("period end must be on or after period start")
	}

	period.Name = strings.TrimSpace(period.Name)
	period.Division = normalizeDivision(period.Division)
	if period.Name == "" {
		period.Name = fmt.Sprintf("%s Payroll - %s", period.PeriodStart.Format("Jan 2006"), period.Division)
	}
	if period.PaymentDate == nil {
		paymentDate := period.PeriodEnd
		period.PaymentDate = &paymentDate
	}
	period.Status = strings.ToLower(strings.TrimSpace(period.Status))
	if period.Status == "" {
		period.Status = "open"
	}
	period.Notes = strings.TrimSpace(period.Notes)
	period.CreatedBy = s.identity.UserID()

	var count int64
	if err := s.db.Model(&Period{}).Where("name = ?", period.Name).Count(&count).Error; err == nil && count > 0 {
		return Period{}, fmt.Errorf("payroll period %q already exists", period.Name)
	}

	if err := s.db.Create(&period).Error; err != nil {
		return Period{}, fmt.Errorf("failed to create payroll period: %w", err)
	}
	s.emit("payroll:updated", map[string]any{"entity": "payroll_period", "action": "create", "id": period.ID})
	return period, nil
}

// ListRuns lists payroll runs, optionally filtered by period.
func (s *Service) ListRuns(payrollPeriodID string) ([]Run, error) {
	query := s.db.Order("created_at DESC")
	if strings.TrimSpace(payrollPeriodID) != "" {
		query = query.Where("payroll_period_id = ?", payrollPeriodID)
	}

	var runs []Run
	if err := query.Find(&runs).Error; err != nil {
		return nil, fmt.Errorf("failed to list payroll runs: %w", err)
	}
	return s.decorateRuns(runs), nil
}

// GetRun loads a run with items, components, and payouts.
func (s *Service) GetRun(runID string) (Run, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return Run{}, fmt.Errorf("payroll run id is required")
	}

	var run Run
	if err := s.db.First(&run, "id = ?", runID).Error; err != nil {
		return Run{}, fmt.Errorf("payroll run not found: %w", err)
	}

	var items []RunItem
	if err := s.db.Where("payroll_run_id = ?", run.ID).Order("employee_name_snapshot ASC").Find(&items).Error; err != nil {
		return Run{}, fmt.Errorf("failed to load payroll run items: %w", err)
	}

	itemIDs := make([]string, 0, len(items))
	for _, item := range items {
		itemIDs = append(itemIDs, item.ID)
	}

	var components []Component
	if len(itemIDs) > 0 {
		if err := s.db.Where("payroll_run_item_id IN ?", itemIDs).Order("component_type ASC, name ASC").Find(&components).Error; err != nil {
			return Run{}, fmt.Errorf("failed to load payroll components: %w", err)
		}
	}

	componentMap := map[string][]Component{}
	for _, component := range components {
		componentMap[component.PayrollRunItemID] = append(componentMap[component.PayrollRunItemID], component)
	}

	payouts, err := s.ListPayouts(run.ID)
	if err != nil {
		return Run{}, err
	}
	payoutMap := map[string]Payout{}
	for _, payout := range payouts {
		payoutMap[payout.PayrollRunItemID] = payout
	}

	items = s.decorateRunItems(items)
	for i := range items {
		items[i].Components = componentMap[items[i].ID]
		if payout, ok := payoutMap[items[i].ID]; ok {
			items[i].PayoutID = payout.ID
			items[i].PayoutStatus = payout.Status
			items[i].PayoutPaidAt = payout.PaidAt
		}
	}

	run = s.decorateRun(run)
	run.Items = items
	run.Payouts = payouts
	return run, nil
}

// GenerateRun generates (or regenerates a draft) payroll run for a period.
func (s *Service) GenerateRun(payrollPeriodID string) (Run, error) {
	payrollPeriodID = strings.TrimSpace(payrollPeriodID)
	if payrollPeriodID == "" {
		return Run{}, fmt.Errorf("payroll period is required")
	}

	var period Period
	if err := s.db.First(&period, "id = ?", payrollPeriodID).Error; err != nil {
		return Run{}, fmt.Errorf("payroll period not found: %w", err)
	}

	var profiles []CompensationProfile
	if err := s.db.Where(
		"is_active = ? AND division = ? AND (effective_from IS NULL OR effective_from <= ?) AND (effective_to IS NULL OR effective_to >= ?)",
		true, normalizeDivision(period.Division), period.PeriodEnd, period.PeriodStart,
	).Find(&profiles).Error; err != nil {
		return Run{}, fmt.Errorf("failed to load compensation profiles: %w", err)
	}
	if len(profiles) == 0 {
		return Run{}, fmt.Errorf("no active compensation profiles available for this payroll period")
	}

	employeeIDs := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		employeeIDs = append(employeeIDs, profile.EmployeeID)
	}
	employees := s.directory.Employees(employeeIDs)

	var run Run
	existingErr := s.db.Where("payroll_period_id = ?", payrollPeriodID).Order("created_at DESC").First(&run).Error
	if existingErr != nil && existingErr != gorm.ErrRecordNotFound {
		return Run{}, fmt.Errorf("failed to lookup payroll run: %w", existingErr)
	}
	if existingErr == nil && run.Status != "draft" {
		return Run{}, fmt.Errorf("payroll run already exists for this period and is %s", run.Status)
	}

	now := time.Now()
	items := make([]RunItem, 0, len(profiles))
	components := make([]Component, 0, len(profiles)*6)
	payouts := make([]Payout, 0, len(profiles))

	var grossTotal, deductionsTotal, netTotal, employerCostTotal float64
	totalEmployees := 0
	paymentDate := period.PeriodEnd
	if period.PaymentDate != nil {
		paymentDate = *period.PaymentDate
	}

	// Deductions exceeding gross would make the accrual journal unbalanced
	// (the net clamp absorbs the difference on the debit side only), so the
	// whole run is refused — no partial generation, no silent skipping.
	var negativeNet []string

	for _, profile := range profiles {
		employee, ok := employees[profile.EmployeeID]
		if !ok || !employee.IsActive {
			continue
		}

		allowances := profile.HousingAllowance + profile.TransportAllowance + profile.OtherAllowance
		deductions := profile.StandardDeduction + profile.TaxDeduction
		gross := profile.BaseSalary + allowances
		net := gross - deductions
		if net < 0 {
			negativeNet = append(negativeNet, fmt.Sprintf(
				"%s: deductions %.3f exceed gross %.3f", employee.FullName, deductions, gross))
			continue
		}
		employerCost := clampAmount(profile.EmployerCost)

		item := RunItem{
			Base:                  Base{ID: uuid.New().String(), CreatedBy: s.identity.UserID(), CreatedAt: now, UpdatedAt: now},
			PayrollRunID:          run.ID,
			EmployeeID:            profile.EmployeeID,
			CompensationProfileID: &profile.ID,
			EmployeeNameSnapshot:  employee.FullName,
			JobTitleSnapshot:      employee.JobTitle,
			BaseSalary:            profile.BaseSalary,
			AllowancesTotal:       allowances,
			DeductionsTotal:       deductions,
			EmployerCostTotal:     employerCost,
			GrossPay:              gross,
			NetPay:                net,
			Status:                "draft",
		}
		item.Components = buildComponents(item.ID, profile, now, s.identity.UserID())
		items = append(items, item)
		components = append(components, item.Components...)
		payouts = append(payouts, Payout{
			Base:             Base{ID: uuid.New().String(), CreatedBy: s.identity.UserID(), CreatedAt: now, UpdatedAt: now},
			PayrollRunID:     run.ID,
			PayrollRunItemID: item.ID,
			EmployeeID:       item.EmployeeID,
			Division:         normalizeDivision(period.Division),
			ScheduledAt:      &paymentDate,
			Amount:           net,
			Currency:         profile.Currency,
			Status:           "scheduled",
		})

		grossTotal += gross
		deductionsTotal += deductions
		netTotal += net
		employerCostTotal += employerCost
		totalEmployees++
	}

	if len(negativeNet) > 0 {
		return Run{}, fmt.Errorf("payroll run refused — deductions exceed gross pay for: %s; correct the compensation profile(s) and regenerate", strings.Join(negativeNet, "; "))
	}
	if totalEmployees == 0 {
		return Run{}, fmt.Errorf("no active employees available for payroll generation")
	}

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if existingErr == gorm.ErrRecordNotFound {
		run = Run{
			Base:              Base{CreatedBy: s.identity.UserID(), CreatedAt: now, UpdatedAt: now},
			RunNumber:         generateRunNumber(period, now),
			PayrollPeriodID:   period.ID,
			Division:          normalizeDivision(period.Division),
			Status:            "draft",
			GeneratedAt:       &now,
			TotalEmployees:    totalEmployees,
			GrossTotal:        grossTotal,
			DeductionsTotal:   deductionsTotal,
			NetTotal:          netTotal,
			EmployerCostTotal: employerCostTotal,
			Currency:          "BHD",
		}
		if err := tx.Create(&run).Error; err != nil {
			tx.Rollback()
			return Run{}, fmt.Errorf("failed to create payroll run: %w", err)
		}
	} else {
		var existingItems []RunItem
		if err := tx.Where("payroll_run_id = ?", run.ID).Find(&existingItems).Error; err != nil {
			tx.Rollback()
			return Run{}, fmt.Errorf("failed to load existing payroll items: %w", err)
		}
		if len(existingItems) > 0 {
			oldItemIDs := make([]string, 0, len(existingItems))
			for _, item := range existingItems {
				oldItemIDs = append(oldItemIDs, item.ID)
			}
			if err := tx.Where("payroll_run_item_id IN ?", oldItemIDs).Delete(&Component{}).Error; err != nil {
				tx.Rollback()
				return Run{}, fmt.Errorf("failed to clear payroll components: %w", err)
			}
		}
		if err := tx.Where("payroll_run_id = ?", run.ID).Delete(&Payout{}).Error; err != nil {
			tx.Rollback()
			return Run{}, fmt.Errorf("failed to clear payroll payouts: %w", err)
		}
		if err := tx.Where("payroll_run_id = ?", run.ID).Delete(&RunItem{}).Error; err != nil {
			tx.Rollback()
			return Run{}, fmt.Errorf("failed to clear payroll items: %w", err)
		}
		run.Status = "draft"
		run.Division = normalizeDivision(period.Division)
		run.GeneratedAt = &now
		run.ApprovedAt = nil
		run.ApprovedBy = ""
		run.PostedAt = nil
		run.PostedBy = ""
		run.PaidAt = nil
		run.PaymentReference = ""
		run.BankAccountID = nil
		run.JournalEntryID = nil
		run.PayoutJournalEntryID = nil
		run.TotalEmployees = totalEmployees
		run.GrossTotal = grossTotal
		run.DeductionsTotal = deductionsTotal
		run.NetTotal = netTotal
		run.EmployerCostTotal = employerCostTotal
		run.Currency = "BHD"
		if err := tx.Model(&run).Updates(map[string]any{
			"status":                  run.Status,
			"division":                run.Division,
			"generated_at":            run.GeneratedAt,
			"approved_at":             nil,
			"approved_by":             "",
			"posted_at":               nil,
			"posted_by":               "",
			"paid_at":                 nil,
			"payment_reference":       "",
			"bank_account_id":         nil,
			"journal_entry_id":        nil,
			"payout_journal_entry_id": nil,
			"total_employees":         run.TotalEmployees,
			"gross_total":             run.GrossTotal,
			"deductions_total":        run.DeductionsTotal,
			"net_total":               run.NetTotal,
			"employer_cost_total":     run.EmployerCostTotal,
			"currency":                run.Currency,
			"updated_at":              now,
		}).Error; err != nil {
			tx.Rollback()
			return Run{}, fmt.Errorf("failed to update payroll run: %w", err)
		}
	}

	for i := range items {
		items[i].PayrollRunID = run.ID
	}
	for i := range payouts {
		payouts[i].PayrollRunID = run.ID
	}

	if err := tx.Create(&items).Error; err != nil {
		tx.Rollback()
		return Run{}, fmt.Errorf("failed to create payroll items: %w", err)
	}
	if len(components) > 0 {
		if err := tx.Create(&components).Error; err != nil {
			tx.Rollback()
			return Run{}, fmt.Errorf("failed to create payroll components: %w", err)
		}
	}
	if len(payouts) > 0 {
		if err := tx.Create(&payouts).Error; err != nil {
			tx.Rollback()
			return Run{}, fmt.Errorf("failed to create payroll payouts: %w", err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		return Run{}, fmt.Errorf("failed to commit payroll run: %w", err)
	}

	s.emit("payroll:updated", map[string]any{"entity": "payroll_run", "action": "generate", "id": run.ID})
	return s.GetRun(run.ID)
}

// ApproveRun approves a draft run through the kernel approval gate (W4
// A.3): RBAC (checked by the host before delegating) remains the
// human-authority source; the kernel gate adds the transition table and
// the AI-authority boundary — an agent-minted actor can never pass.
func (s *Service) ApproveRun(runID, notes string) (Run, error) {
	var run Run
	if err := s.db.First(&run, "id = ?", strings.TrimSpace(runID)).Error; err != nil {
		return Run{}, fmt.Errorf("payroll run not found: %w", err)
	}
	if run.Status != "draft" {
		return Run{}, fmt.Errorf("only draft payroll runs can be approved")
	}

	by, err := s.approvalActor()
	if err != nil {
		return Run{}, fmt.Errorf("payroll approver identity: %w", err)
	}
	if err := GateRunApproval(run, notes, by); err != nil {
		return Run{}, err
	}

	now := time.Now()
	notes = strings.TrimSpace(notes)
	if err := s.db.Model(&run).Updates(map[string]any{
		"status":      "approved",
		"approved_at": &now,
		"approved_by": s.identity.ActorID(),
		"notes":       mergeNotes(run.Notes, notes),
		"updated_at":  now,
	}).Error; err != nil {
		return Run{}, fmt.Errorf("failed to approve payroll run: %w", err)
	}
	// Mission I (I-16): item-status flip must not fail silently after the run
	// itself has been approved — a half-approved run corrupts payout gating.
	if err := s.db.Model(&RunItem{}).Where("payroll_run_id = ?", run.ID).Updates(map[string]any{
		"status":     "approved",
		"updated_at": now,
	}).Error; err != nil {
		return Run{}, fmt.Errorf("payroll run approved but item statuses failed to update: %w", err)
	}

	s.emit("payroll:updated", map[string]any{"entity": "payroll_run", "action": "approve", "id": run.ID})
	return s.GetRun(run.ID)
}

// PostRun posts an approved run: accrual journal, status flip, expense
// mirror.
func (s *Service) PostRun(runID string) (Run, error) {
	run, err := s.GetRun(strings.TrimSpace(runID))
	if err != nil {
		return Run{}, err
	}
	if run.Status == "posted" || run.Status == "paid" {
		return run, nil
	}
	if run.Status != "approved" {
		return Run{}, fmt.Errorf("payroll run must be approved before posting")
	}

	journalID, err := s.postAccrualJournal(&run)
	if err != nil {
		return Run{}, err
	}

	now := time.Now()
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Model(&Run{}).Where("id = ?", run.ID).Updates(map[string]any{
		"status":           "posted",
		"posted_at":        &now,
		"posted_by":        s.identity.ActorID(),
		"journal_entry_id": journalID,
		"updated_at":       now,
	}).Error; err != nil {
		tx.Rollback()
		return Run{}, fmt.Errorf("failed to mark payroll run posted: %w", err)
	}
	if err := tx.Model(&RunItem{}).Where("payroll_run_id = ?", run.ID).Updates(map[string]any{
		"status":     "posted",
		"updated_at": now,
	}).Error; err != nil {
		tx.Rollback()
		return Run{}, fmt.Errorf("failed to update payroll run items: %w", err)
	}

	run.Status = "posted"
	run.PostedAt = &now
	run.PostedBy = s.identity.ActorID()
	run.JournalEntryID = &journalID
	if err := s.expenses.SyncRunExpense(tx, &run); err != nil {
		tx.Rollback()
		return Run{}, err
	}
	if err := tx.Commit().Error; err != nil {
		return Run{}, fmt.Errorf("failed to commit payroll posting: %w", err)
	}

	s.emit("payroll:updated", map[string]any{"entity": "payroll_run", "action": "post", "id": run.ID})
	s.expenses.EmitExpenseUpdated(run.ID)
	return s.GetRun(run.ID)
}

// MarkPaid marks a run paid: payout journal, payout rows, expense mirror.
func (s *Service) MarkPaid(runID, paidAtISO, paymentReference, bankAccountID string) (Run, error) {
	run, err := s.GetRun(strings.TrimSpace(runID))
	if err != nil {
		return Run{}, err
	}
	if run.Status == "paid" {
		return run, nil
	}
	if run.Status != "posted" {
		run, err = s.PostRun(run.ID)
		if err != nil {
			return Run{}, err
		}
	}

	paidAt, err := parseOptionalISODateTime(paidAtISO)
	if err != nil {
		return Run{}, err
	}
	if paidAt == nil {
		now := time.Now()
		paidAt = &now
	}

	payoutJournalID, err := s.postPayoutJournal(&run, paymentReference)
	if err != nil {
		return Run{}, err
	}

	var bankAccountRef *string
	if trimmed := strings.TrimSpace(bankAccountID); trimmed != "" {
		bankDivision := s.resolveBankAccountDivision(trimmed)
		runDivision := normalizeDivision(run.Division)
		if bankDivision != runDivision {
			return Run{}, fmt.Errorf("payroll run belongs to %s but bank account belongs to %s", runDivision, bankDivision)
		}
		bankAccountRef = &trimmed
	}

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Model(&Payout{}).Where("payroll_run_id = ?", run.ID).Updates(map[string]any{
		"status":            "paid",
		"paid_at":           paidAt,
		"payment_reference": strings.TrimSpace(paymentReference),
		"bank_account_id":   bankAccountRef,
		"division":          normalizeDivision(run.Division),
		"updated_at":        time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		return Run{}, fmt.Errorf("failed to mark payroll payouts paid: %w", err)
	}

	if err := tx.Model(&Run{}).Where("id = ?", run.ID).Updates(map[string]any{
		"status":                  "paid",
		"paid_at":                 paidAt,
		"payment_reference":       strings.TrimSpace(paymentReference),
		"bank_account_id":         bankAccountRef,
		"division":                normalizeDivision(run.Division),
		"payout_journal_entry_id": payoutJournalID,
		"updated_at":              time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		return Run{}, fmt.Errorf("failed to update payroll run payment status: %w", err)
	}
	if err := tx.Model(&RunItem{}).Where("payroll_run_id = ?", run.ID).Updates(map[string]any{
		"status":     "paid",
		"updated_at": time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		return Run{}, fmt.Errorf("failed to update payroll run items: %w", err)
	}

	run.Status = "paid"
	run.PaidAt = paidAt
	run.PaymentReference = strings.TrimSpace(paymentReference)
	run.BankAccountID = bankAccountRef
	if payoutJournalID != "" {
		run.PayoutJournalEntryID = &payoutJournalID
	} else {
		run.PayoutJournalEntryID = nil
	}
	if err := s.expenses.SyncRunExpense(tx, &run); err != nil {
		tx.Rollback()
		return Run{}, err
	}

	if err := tx.Commit().Error; err != nil {
		return Run{}, fmt.Errorf("failed to commit payroll payment: %w", err)
	}

	s.emit("payroll:updated", map[string]any{"entity": "payroll_run", "action": "paid", "id": run.ID})
	s.expenses.EmitExpenseUpdated(run.ID)
	return s.GetRun(run.ID)
}

// ListPayouts lists payouts, optionally scoped to a run.
func (s *Service) ListPayouts(payrollRunID string) ([]Payout, error) {
	query := s.db.Order("scheduled_at DESC, created_at DESC")
	if strings.TrimSpace(payrollRunID) != "" {
		query = query.Where("payroll_run_id = ?", payrollRunID)
	}

	var payouts []Payout
	if err := query.Find(&payouts).Error; err != nil {
		return nil, fmt.Errorf("failed to list payroll payouts: %w", err)
	}
	return s.decoratePayouts(payouts), nil
}

// ListUnreconciledPayouts lists paid payouts with no bank statement line.
func (s *Service) ListUnreconciledPayouts() ([]Payout, error) {
	var payouts []Payout
	if err := s.db.Where("status = ? AND bank_statement_line_id IS NULL", "paid").
		Order("paid_at DESC, created_at DESC").
		Find(&payouts).Error; err != nil {
		return nil, fmt.Errorf("failed to list unreconciled payroll payouts: %w", err)
	}
	return s.decoratePayouts(payouts), nil
}

// Dashboard summarizes payroll state for the dashboard.
func (s *Service) Dashboard() (DashboardSummary, error) {
	summary := DashboardSummary{}
	var activeProfiles int64
	var openPeriods int64
	var draftRuns int64
	var approvedUnpaidRuns int64
	_ = s.db.Model(&CompensationProfile{}).Where("is_active = ?", true).Count(&activeProfiles).Error
	_ = s.db.Model(&Period{}).Where("status = ?", "open").Count(&openPeriods).Error
	_ = s.db.Model(&Run{}).Where("status = ?", "draft").Count(&draftRuns).Error
	_ = s.db.Model(&Run{}).Where("status IN ?", []string{"approved", "posted"}).Count(&approvedUnpaidRuns).Error
	summary.ActiveProfiles = int(activeProfiles)
	summary.OpenPeriods = int(openPeriods)
	summary.DraftRuns = int(draftRuns)
	summary.ApprovedUnpaidRuns = int(approvedUnpaidRuns)

	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	type aggregate struct {
		Total float64
	}
	var mtd aggregate
	_ = s.db.Model(&Run{}).Select("COALESCE(SUM(net_total), 0) AS total").Where("status = ? AND paid_at >= ?", "paid", startOfMonth).Scan(&mtd).Error
	summary.MonthToDateNetPayroll = mtd.Total

	var liabilities aggregate
	_ = s.db.Model(&Run{}).Select("COALESCE(SUM(net_total + deductions_total + employer_cost_total), 0) AS total").Where("status IN ?", []string{"approved", "posted"}).Scan(&liabilities).Error
	summary.UpcomingPayrollLiability = liabilities.Total

	return summary, nil
}

func (s *Service) postAccrualJournal(run *Run) (string, error) {
	salaryExpense, err := s.ensureAccount("6000", "Salaries & Wages", "Expense")
	if err != nil {
		return "", err
	}
	overheadExpense, err := s.ensureAccount("6050", "Payroll Overheads", "Expense")
	if err != nil {
		return "", err
	}
	payrollPayable, err := s.ensureAccount("2210", "Payroll Payable", "Liability")
	if err != nil {
		return "", err
	}
	deductionPayable, err := s.ensureAccount("2211", "Payroll Deductions Payable", "Liability")
	if err != nil {
		return "", err
	}
	employerPayable, err := s.ensureAccount("2212", "Payroll Employer Liabilities", "Liability")
	if err != nil {
		return "", err
	}

	now := time.Now()
	entryDate := now
	var period Period
	if err := s.db.First(&period, "id = ?", run.PayrollPeriodID).Error; err == nil {
		entryDate = period.PeriodEnd
	}

	lines := []finance.JournalLine{
		s.newJournalLine(salaryExpense, run.GrossTotal, 0, "Payroll gross accrual", now),
		s.newJournalLine(payrollPayable, 0, run.NetTotal, "Payroll net payable", now),
	}
	if run.EmployerCostTotal > 0 {
		lines = append(lines,
			s.newJournalLine(overheadExpense, run.EmployerCostTotal, 0, "Payroll employer cost accrual", now),
			s.newJournalLine(employerPayable, 0, run.EmployerCostTotal, "Payroll employer liabilities", now),
		)
	}
	if run.DeductionsTotal > 0 {
		lines = append(lines, s.newJournalLine(deductionPayable, 0, run.DeductionsTotal, "Payroll deductions withheld", now))
	}

	journal := finance.JournalEntry{
		Base:            Base{ID: uuid.New().String(), CreatedBy: s.identity.UserID(), CreatedAt: now, UpdatedAt: now},
		EntryNumber:     fmt.Sprintf("PAY-JE-%d-%04d", now.Year(), now.UnixNano()%10000),
		EntryDate:       entryDate,
		Description:     fmt.Sprintf("Payroll accrual %s", run.RunNumber),
		DebitTotal:      run.GrossTotal + run.EmployerCostTotal,
		CreditTotal:     run.NetTotal + run.DeductionsTotal + run.EmployerCostTotal,
		IsPosted:        true,
		PostedAt:        &now,
		PostedBy:        s.identity.ActorID(),
		FiscalYear:      entryDate.Year(),
		FiscalPeriod:    int(entryDate.Month()),
		SourceType:      "payroll_run",
		SourceID:        run.ID,
		IsAutoGenerated: true,
	}

	return s.persistPostedJournal(journal, lines)
}

func (s *Service) postPayoutJournal(run *Run, paymentReference string) (string, error) {
	if run.NetTotal <= 0 {
		return "", nil
	}

	payrollPayable, err := s.ensureAccount("2210", "Payroll Payable", "Liability")
	if err != nil {
		return "", err
	}
	cashAccount, err := s.ensureAccount("1000", "Cash", "Asset")
	if err != nil {
		return "", err
	}

	now := time.Now()
	lines := []finance.JournalLine{
		s.newJournalLine(payrollPayable, run.NetTotal, 0, "Payroll payout clearing", now),
		s.newJournalLine(cashAccount, 0, run.NetTotal, "Payroll cash disbursement", now),
	}
	journal := finance.JournalEntry{
		Base:            Base{ID: uuid.New().String(), CreatedBy: s.identity.UserID(), CreatedAt: now, UpdatedAt: now},
		EntryNumber:     fmt.Sprintf("PAY-OUT-%d-%04d", now.Year(), now.UnixNano()%10000),
		EntryDate:       now,
		Description:     fmt.Sprintf("Payroll payout %s %s", run.RunNumber, strings.TrimSpace(paymentReference)),
		DebitTotal:      run.NetTotal,
		CreditTotal:     run.NetTotal,
		IsPosted:        true,
		PostedAt:        &now,
		PostedBy:        s.identity.ActorID(),
		FiscalYear:      now.Year(),
		FiscalPeriod:    int(now.Month()),
		SourceType:      "payroll_payout",
		SourceID:        run.ID,
		IsAutoGenerated: true,
	}

	return s.persistPostedJournal(journal, lines)
}

// ensureAccount finds or creates a supporting GL account. (The expense
// service keeps its own root copy; both operate on finance.ChartOfAccount.)
func (s *Service) ensureAccount(code, name, accountType string) (finance.ChartOfAccount, error) {
	var account finance.ChartOfAccount
	if err := s.db.Where("account_code = ?", code).First(&account).Error; err == nil {
		return account, nil
	}
	account = finance.ChartOfAccount{
		Base:        Base{CreatedBy: "system"},
		AccountCode: code,
		AccountName: name,
		AccountType: accountType,
		IsActive:    true,
	}
	if err := s.db.Create(&account).Error; err != nil {
		return finance.ChartOfAccount{}, fmt.Errorf("failed to ensure account %s: %w", code, err)
	}
	return account, nil
}

func (s *Service) persistPostedJournal(journal finance.JournalEntry, lines []finance.JournalLine) (string, error) {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(&journal).Error; err != nil {
		tx.Rollback()
		return "", fmt.Errorf("failed to create payroll journal entry: %w", err)
	}
	for i := range lines {
		lines[i].EntryID = journal.ID
	}
	if err := tx.Create(&lines).Error; err != nil {
		tx.Rollback()
		return "", fmt.Errorf("failed to create payroll journal lines: %w", err)
	}
	for _, line := range lines {
		var account finance.ChartOfAccount
		if err := tx.First(&account, "id = ?", line.AccountID).Error; err != nil {
			tx.Rollback()
			return "", fmt.Errorf("failed to load payroll posting account: %w", err)
		}
		change := line.Credit - line.Debit
		if account.AccountType == "Asset" || account.AccountType == "Expense" {
			change = line.Debit - line.Credit
		}
		if err := tx.Model(&account).Update("balance", gorm.Expr("balance + ?", change)).Error; err != nil {
			tx.Rollback()
			return "", fmt.Errorf("failed to update payroll account balance: %w", err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		return "", fmt.Errorf("failed to commit payroll journal: %w", err)
	}
	return journal.ID, nil
}

func (s *Service) resolveBankAccountDivision(bankAccountID string) string {
	if s.db == nil || strings.TrimSpace(bankAccountID) == "" {
		return overlay.Active().DefaultDivision()
	}

	var account finance.CompanyBankAccount
	if err := s.db.Select("division").First(&account, "id = ?", bankAccountID).Error; err == nil {
		return normalizeDivision(account.Division)
	}
	return overlay.Active().DefaultDivision()
}

func (s *Service) decorateProfiles(profiles []CompensationProfile) []CompensationProfile {
	if len(profiles) == 0 {
		return profiles
	}
	employeeIDs := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		employeeIDs = append(employeeIDs, profile.EmployeeID)
	}
	employees := s.directory.Employees(employeeIDs)
	for i := range profiles {
		if employee, ok := employees[profiles[i].EmployeeID]; ok {
			profiles[i].EmployeeName = employee.FullName
			profiles[i].JobTitle = employee.JobTitle
		}
	}
	return profiles
}

func (s *Service) decorateProfile(profile CompensationProfile) CompensationProfile {
	rows := s.decorateProfiles([]CompensationProfile{profile})
	if len(rows) == 0 {
		return profile
	}
	return rows[0]
}

func (s *Service) decorateRuns(runs []Run) []Run {
	if len(runs) == 0 {
		return runs
	}
	periodIDs := make([]string, 0, len(runs))
	for _, run := range runs {
		periodIDs = append(periodIDs, run.PayrollPeriodID)
	}
	periodNames := map[string]string{}
	var periods []Period
	if err := s.db.Select("id", "name").Where("id IN ?", periodIDs).Find(&periods).Error; err == nil {
		for _, period := range periods {
			periodNames[period.ID] = period.Name
		}
	}
	for i := range runs {
		runs[i].PeriodName = periodNames[runs[i].PayrollPeriodID]
	}
	return runs
}

func (s *Service) decorateRun(run Run) Run {
	rows := s.decorateRuns([]Run{run})
	if len(rows) == 0 {
		return run
	}
	return rows[0]
}

func (s *Service) decorateRunItems(items []RunItem) []RunItem {
	if len(items) == 0 {
		return items
	}
	employeeIDs := make([]string, 0, len(items))
	for _, item := range items {
		employeeIDs = append(employeeIDs, item.EmployeeID)
	}
	employees := s.directory.Employees(employeeIDs)
	for i := range items {
		if employee, ok := employees[items[i].EmployeeID]; ok {
			items[i].EmployeeName = employee.FullName
		}
		if items[i].EmployeeName == "" {
			items[i].EmployeeName = items[i].EmployeeNameSnapshot
		}
	}
	return items
}

func (s *Service) decoratePayouts(payouts []Payout) []Payout {
	if len(payouts) == 0 {
		return payouts
	}
	employeeIDs := make([]string, 0, len(payouts))
	runIDs := make([]string, 0, len(payouts))
	for _, payout := range payouts {
		employeeIDs = append(employeeIDs, payout.EmployeeID)
		runIDs = append(runIDs, payout.PayrollRunID)
	}
	employees := s.directory.Employees(employeeIDs)
	runNumbers := map[string]string{}
	var runs []Run
	if err := s.db.Select("id", "run_number").Where("id IN ?", runIDs).Find(&runs).Error; err == nil {
		for _, run := range runs {
			runNumbers[run.ID] = run.RunNumber
		}
	}
	for i := range payouts {
		if employee, ok := employees[payouts[i].EmployeeID]; ok {
			payouts[i].EmployeeName = employee.FullName
		}
		payouts[i].RunNumber = runNumbers[payouts[i].PayrollRunID]
	}
	return payouts
}

func buildComponents(itemID string, profile CompensationProfile, now time.Time, actorID string) []Component {
	components := make([]Component, 0, 6)
	appendComponent := func(componentType, code, name string, amount float64) {
		if amount <= 0 {
			return
		}
		components = append(components, Component{
			Base:             Base{ID: uuid.New().String(), CreatedBy: actorID, CreatedAt: now, UpdatedAt: now},
			PayrollRunItemID: itemID,
			ComponentType:    componentType,
			Code:             code,
			Name:             name,
			Amount:           amount,
		})
	}
	appendComponent("earning", "BASE", "Base Salary", profile.BaseSalary)
	appendComponent("earning", "HOUSING", "Housing Allowance", profile.HousingAllowance)
	appendComponent("earning", "TRANSPORT", "Transport Allowance", profile.TransportAllowance)
	appendComponent("earning", "OTHER", "Other Allowance", profile.OtherAllowance)
	appendComponent("deduction", "STANDARD", "Standard Deduction", profile.StandardDeduction)
	appendComponent("deduction", "TAX", "Tax Deduction", profile.TaxDeduction)
	appendComponent("employer_cost", "EMPLOYER", "Employer Cost", profile.EmployerCost)
	return components
}

func (s *Service) newJournalLine(account finance.ChartOfAccount, debit, credit float64, description string, now time.Time) finance.JournalLine {
	return finance.JournalLine{
		Base:        Base{ID: uuid.New().String(), CreatedBy: s.identity.UserID(), CreatedAt: now, UpdatedAt: now},
		AccountID:   account.ID,
		AccountName: account.AccountName,
		Debit:       debit,
		Credit:      credit,
		Description: description,
	}
}

func generateRunNumber(period Period, now time.Time) string {
	return fmt.Sprintf("PAY-%d-%02d-%d", period.PeriodStart.Year(), int(period.PeriodStart.Month()), now.Unix()%10000)
}

func clampAmount(value float64) float64 {
	if value < 0 {
		return 0
	}
	return value
}

func mergeNotes(existing, next string) string {
	existing = strings.TrimSpace(existing)
	next = strings.TrimSpace(next)
	if existing == "" {
		return next
	}
	if next == "" {
		return existing
	}
	return existing + "\n" + next
}

func parseOptionalISODateTime(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	layouts := []string{time.RFC3339, "2006-01-02"}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return &parsed, nil
		}
	}
	return nil, fmt.Errorf("invalid date format")
}

// approvalActor mints the kernel actor for a payroll approval. The host
// has already passed its RBAC approve check before delegating, so the
// operator carries approve authority; agent code paths must mint their
// own TypeAgent actors, which the kernel refuses approve power at
// construction AND pkg/approvals refuses at the transition.
func (s *Service) approvalActor() (actor.Actor, error) {
	return actor.New(actor.Input{
		ID:          s.identity.ActorID(),
		DisplayName: s.identity.DisplayName(),
		Type:        actor.TypeOperator,
		Authority:   actor.AuthorityApprove,
	})
}

// GateRunApproval runs the kernel approval gate for a payroll-run
// approval. Pure (no persistence): callers keep writing the same status
// strings they always did; the gate only decides legality.
func GateRunApproval(run Run, notes string, by actor.Actor) error {
	from, err := approvals.DecisionFromStatus(run.Status)
	if err != nil {
		return fmt.Errorf("payroll run %s has %w", run.ID, err)
	}
	if _, err := approvals.Transition(run.ID, "payroll_run", from, approval.DecisionApproved, by, strings.TrimSpace(notes), time.Now().UTC()); err != nil {
		return err
	}
	return nil
}
