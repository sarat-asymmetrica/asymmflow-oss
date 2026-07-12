// Package expense contains the concrete expense service implementation.
package expense

import "gorm.io/gorm"

type Handlers[CategoryModel any, VendorModel any, EntryModel any, SummaryModel any, RecurringModel any, BankCandidateModel any] struct {
	EnsureFoundation             func() error
	ListCategories               func(activeOnly bool) ([]CategoryModel, error)
	CreateCategory               func(category CategoryModel) (CategoryModel, error)
	ListVendors                  func(activeOnly bool) ([]VendorModel, error)
	CreateVendor                 func(vendor VendorModel) (VendorModel, error)
	CreateEntry                  func(entry EntryModel) (EntryModel, error)
	ListEntries                  func(status string, includePaid bool) ([]EntryModel, error)
	ListDashboardSummary         func() (SummaryModel, error)
	SubmitEntry                  func(entryID string) (EntryModel, error)
	ApproveEntry                 func(entryID, notes string) (EntryModel, error)
	RejectEntry                  func(entryID, reason string) (EntryModel, error)
	PostEntry                    func(entryID string) (EntryModel, error)
	MarkEntryPaid                func(entryID, paidAtISO, paymentReference, bankAccountID, paymentMethod string) (EntryModel, error)
	ListRecurring                func(activeOnly bool) ([]RecurringModel, error)
	CreateRecurring              func(item RecurringModel) (RecurringModel, error)
	DeleteRecurring              func(recurringID string) error
	GenerateRecurring            func(cutoffISO string) ([]EntryModel, error)
	ListBankCandidates           func(includeLinked bool) ([]BankCandidateModel, error)
	CreateEntryFromBankCandidate func(bankExpenseID, categoryID string) (EntryModel, error)
}

type Service[CategoryModel any, VendorModel any, EntryModel any, SummaryModel any, RecurringModel any, BankCandidateModel any] struct {
	db       *gorm.DB
	handlers Handlers[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel]
}

func New[CategoryModel any, VendorModel any, EntryModel any, SummaryModel any, RecurringModel any, BankCandidateModel any](db *gorm.DB, handlers Handlers[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel]) *Service[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel] {
	return &Service[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel]{db: db, handlers: handlers}
}

func (s *Service[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel]) EnsureFoundation() error {
	return s.handlers.EnsureFoundation()
}

func (s *Service[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel]) ListCategories(activeOnly bool) ([]CategoryModel, error) {
	return s.handlers.ListCategories(activeOnly)
}

func (s *Service[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel]) CreateCategory(category CategoryModel) (CategoryModel, error) {
	return s.handlers.CreateCategory(category)
}

func (s *Service[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel]) ListVendors(activeOnly bool) ([]VendorModel, error) {
	return s.handlers.ListVendors(activeOnly)
}

func (s *Service[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel]) CreateVendor(vendor VendorModel) (VendorModel, error) {
	return s.handlers.CreateVendor(vendor)
}

func (s *Service[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel]) CreateEntry(entry EntryModel) (EntryModel, error) {
	return s.handlers.CreateEntry(entry)
}

func (s *Service[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel]) ListEntries(status string, includePaid bool) ([]EntryModel, error) {
	return s.handlers.ListEntries(status, includePaid)
}

func (s *Service[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel]) ListDashboardSummary() (SummaryModel, error) {
	return s.handlers.ListDashboardSummary()
}

func (s *Service[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel]) SubmitEntry(entryID string) (EntryModel, error) {
	return s.handlers.SubmitEntry(entryID)
}

func (s *Service[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel]) ApproveEntry(entryID, notes string) (EntryModel, error) {
	return s.handlers.ApproveEntry(entryID, notes)
}

func (s *Service[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel]) RejectEntry(entryID, reason string) (EntryModel, error) {
	return s.handlers.RejectEntry(entryID, reason)
}

func (s *Service[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel]) PostEntry(entryID string) (EntryModel, error) {
	return s.handlers.PostEntry(entryID)
}

func (s *Service[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel]) MarkEntryPaid(entryID, paidAtISO, paymentReference, bankAccountID, paymentMethod string) (EntryModel, error) {
	return s.handlers.MarkEntryPaid(entryID, paidAtISO, paymentReference, bankAccountID, paymentMethod)
}

func (s *Service[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel]) ListRecurring(activeOnly bool) ([]RecurringModel, error) {
	return s.handlers.ListRecurring(activeOnly)
}

func (s *Service[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel]) CreateRecurring(item RecurringModel) (RecurringModel, error) {
	return s.handlers.CreateRecurring(item)
}

func (s *Service[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel]) DeleteRecurring(recurringID string) error {
	return s.handlers.DeleteRecurring(recurringID)
}

func (s *Service[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel]) GenerateRecurring(cutoffISO string) ([]EntryModel, error) {
	return s.handlers.GenerateRecurring(cutoffISO)
}

func (s *Service[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel]) ListBankCandidates(includeLinked bool) ([]BankCandidateModel, error) {
	return s.handlers.ListBankCandidates(includeLinked)
}

func (s *Service[CategoryModel, VendorModel, EntryModel, SummaryModel, RecurringModel, BankCandidateModel]) CreateEntryFromBankCandidate(bankExpenseID, categoryID string) (EntryModel, error) {
	return s.handlers.CreateEntryFromBankCandidate(bankExpenseID, categoryID)
}
