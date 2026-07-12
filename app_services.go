package main

import (
	"time"

	butlerdomain "ph_holdings_app/pkg/butler"
	crmcontract "ph_holdings_app/pkg/crm/contract"
	crmfulfillment "ph_holdings_app/pkg/crm/fulfillment"
	crmprocurement "ph_holdings_app/pkg/crm/procurement"
	financebanking "ph_holdings_app/pkg/finance/banking"
	financecheque "ph_holdings_app/pkg/finance/cheque"
	financeexpense "ph_holdings_app/pkg/finance/expense"
	financefx "ph_holdings_app/pkg/finance/fx"
	financepayment "ph_holdings_app/pkg/finance/payment"
	financepayroll "ph_holdings_app/pkg/finance/payroll"
	infraassets "ph_holdings_app/pkg/infra/assets"
	infradeletion "ph_holdings_app/pkg/infra/deletion"
	infradevice "ph_holdings_app/pkg/infra/device"
	infralicense "ph_holdings_app/pkg/infra/license"
)

// AppServices groups domain services that App exposes through Wails wrappers.
// The DB is available before startup wires these services; some services also
// receive lightweight dependencies as their extraction tickets need them.
type AppServices struct {
	payment       *financepayment.Service[Payment, SupplierPayment]
	expense       *financeexpense.Service[ExpenseCategory, ExpenseVendor, ExpenseEntry, ExpenseDashboardSummary, RecurringExpense, BankExpenseEntry]
	banking       *financebanking.Service[BookBankReconciliation, BankReconciliationMatchResult, AllocationInput, BankStatement, BankStatementLine, BalanceContinuityReportData, StatementHash, BankReconciliationAuditLog]
	fulfillment   *crmfulfillment.Service[DeliveryNote]
	serials       *crmfulfillment.Serials
	cheques       *financecheque.Service
	fx            *financefx.Service
	assets        *infraassets.Service
	device        *infradevice.Service
	payroll       *financepayroll.Service
	procurement   *crmprocurement.Service[PurchaseOrder, GoodsReceivedNote, GRNItem]
	contract      *crmcontract.Service
	license       *infralicense.Service[LicenseKey, LicenseActivationResult, LicenseValidationResult]
	deletion      *infradeletion.Service
	butlerContext butlerdomain.ButlerAppContext
}

func (a *App) initServices() {
	a.services = AppServices{
		payment: financepayment.New(a.db, a.cache, financepayment.Handlers[Payment, SupplierPayment]{
			RecordPayment: func(invoiceID string, amount float64, method string, dateStr string, reference string) (*Payment, error) {
				return recordPayment(a, invoiceID, amount, method, dateStr, reference)
			},
			GetPaymentsByInvoice:    func(invoiceID string) ([]Payment, error) { return getPaymentsByInvoice(a, invoiceID) },
			GetAllPayments:          func(limit, offset int) ([]Payment, error) { return getAllPayments(a, limit, offset) },
			GetPayment:              func(id string) (Payment, error) { return getPayment(a, id) },
			UpdatePayment:           func(id string, payment Payment) (*Payment, error) { return updatePayment(a, id, payment) },
			CheckOrderCompletion:    func(orderID string) error { return checkOrderCompletion(a, orderID) },
			ProgressOrderOnDelivery: func(orderID string) error { return progressOrderOnDelivery(a, orderID) },
			ProgressOrderOnInvoice:  func(orderID string) error { return progressOrderOnInvoice(a, orderID) },
			RecordSupplierPayment: func(invoiceID string, amount float64, currency, method, date, reference string, exchangeRate float64) (*SupplierPayment, error) {
				return recordSupplierPayment(a, invoiceID, amount, currency, method, date, reference, exchangeRate)
			},
			GetSupplierPaymentsByInvoice: func(invoiceID string) ([]SupplierPayment, error) { return getSupplierPaymentsByInvoice(a, invoiceID) },
			GetAllSupplierPayments:       func() ([]SupplierPayment, error) { return getAllSupplierPayments(a) },
			GetSupplierPaymentsSummary:   func() (map[string]any, error) { return getSupplierPaymentsSummary(a) },
			GetSupplierPayment:           func(id string) (SupplierPayment, error) { return getSupplierPayment(a, id) },
			UpdateSupplierPayment: func(id string, payment SupplierPayment) (*SupplierPayment, error) {
				return updateSupplierPayment(a, id, payment)
			},
		}),
		expense: financeexpense.New(a.db, financeexpense.Handlers[ExpenseCategory, ExpenseVendor, ExpenseEntry, ExpenseDashboardSummary, RecurringExpense, BankExpenseEntry]{
			EnsureFoundation: func() error { return ensureExpenseFoundation(a) },
			ListCategories:   func(activeOnly bool) ([]ExpenseCategory, error) { return listExpenseCategories(a, activeOnly) },
			CreateCategory:   func(category ExpenseCategory) (ExpenseCategory, error) { return createExpenseCategory(a, category) },
			ListVendors:      func(activeOnly bool) ([]ExpenseVendor, error) { return listExpenseVendors(a, activeOnly) },
			CreateVendor:     func(vendor ExpenseVendor) (ExpenseVendor, error) { return createExpenseVendor(a, vendor) },
			CreateEntry:      func(entry ExpenseEntry) (ExpenseEntry, error) { return createExpenseEntry(a, entry) },
			ListEntries: func(status string, includePaid bool) ([]ExpenseEntry, error) {
				return listExpenseEntries(a, status, includePaid)
			},
			ListDashboardSummary: func() (ExpenseDashboardSummary, error) { return listExpenseDashboardSummary(a) },
			SubmitEntry:          func(entryID string) (ExpenseEntry, error) { return submitExpenseEntry(a, entryID) },
			ApproveEntry:         func(entryID, notes string) (ExpenseEntry, error) { return approveExpenseEntry(a, entryID, notes) },
			RejectEntry:          func(entryID, reason string) (ExpenseEntry, error) { return rejectExpenseEntry(a, entryID, reason) },
			PostEntry:            func(entryID string) (ExpenseEntry, error) { return postExpenseEntry(a, entryID) },
			MarkEntryPaid: func(entryID, paidAtISO, paymentReference, bankAccountID, paymentMethod string) (ExpenseEntry, error) {
				return markExpenseEntryPaid(a, entryID, paidAtISO, paymentReference, bankAccountID, paymentMethod)
			},
			ListRecurring:     func(activeOnly bool) ([]RecurringExpense, error) { return listRecurringExpenses(a, activeOnly) },
			CreateRecurring:   func(item RecurringExpense) (RecurringExpense, error) { return createRecurringExpense(a, item) },
			DeleteRecurring:   func(recurringID string) error { return deleteRecurringExpense(a, recurringID) },
			GenerateRecurring: func(cutoffISO string) ([]ExpenseEntry, error) { return generateRecurringExpenses(a, cutoffISO) },
			ListBankCandidates: func(includeLinked bool) ([]BankExpenseEntry, error) {
				return listBankExpenseCandidates(a, includeLinked)
			},
			CreateEntryFromBankCandidate: func(bankExpenseID, categoryID string) (ExpenseEntry, error) {
				return createExpenseFromBankCandidate(a, bankExpenseID, categoryID)
			},
		}),
		banking: financebanking.New(a.db, financebanking.Handlers[BookBankReconciliation, BankReconciliationMatchResult, AllocationInput, BankStatement, BankStatementLine, BalanceContinuityReportData, StatementHash, BankReconciliationAuditLog]{
			RequirePermission:        func(permission string) error { return a.requirePermission(permission) },
			GetCashPosition:          func() (map[string]any, error) { return getCashPosition(a) },
			GetCashPositionByAccount: func(bankAccountID string) (float64, error) { return getCashPositionByAccount(a, bankAccountID) },
			CreateBookBankReconciliation: func(bankAccountID string, reconciliationDate time.Time, bankStatementBalance, bookBalance, depositsInTransit, outstandingCheques float64) (*BookBankReconciliation, error) {
				return createBookBankReconciliation(a, bankAccountID, reconciliationDate, bankStatementBalance, bookBalance, depositsInTransit, outstandingCheques)
			},
		}, financebanking.Ports{
			Auth:           appBankingAuthPort{app: a},
			Audit:          appBankingAuditPort{app: a},
			Division:       appBankingDivisionPort{app: a},
			DeleteApproval: appBankingDeleteApprovalPort{app: a},
		}),
		fulfillment: crmfulfillment.New(a.db, crmfulfillment.Handlers[DeliveryNote]{
			CreateDeliveryNote:  func(dn DeliveryNote) (DeliveryNote, error) { return createDeliveryNote(a, dn) },
			GetDeliveryNotes:    func() ([]DeliveryNote, error) { return getDeliveryNotes(a) },
			GetDeliveryNoteByID: func(id string) (DeliveryNote, error) { return getDeliveryNoteByID(a, id) },
			UpdateDeliveryNote:  func(dn DeliveryNote) (DeliveryNote, error) { return updateDeliveryNote(a, dn) },
			DispatchDeliveryNote: func(id, driverName, vehicleNumber string) error {
				return dispatchDeliveryNote(a, id, driverName, vehicleNumber)
			},
			ConfirmDeliveryNote: func(id, signedBy string) (string, error) { return confirmDeliveryNote(a, id, signedBy) },
		}),
		serials: crmfulfillment.NewSerials(a.db),
		cheques: financecheque.New(a.db),
		fx:      financefx.New(a.db),
		assets:  infraassets.New(a.db, a.getAssetCacheDir()),
		device:  infradevice.New(a.db),
		payroll: financepayroll.New(a.db, appPayrollIdentity{app: a}, appPayrollDirectory{app: a}, appPayrollEvents{app: a}, appPayrollExpenseBridge{app: a}),
		procurement: crmprocurement.New(a.db, crmprocurement.Handlers[PurchaseOrder, GoodsReceivedNote, GRNItem]{
			CreatePurchaseOrder:  func(po PurchaseOrder) (PurchaseOrder, error) { return createPurchaseOrder(a, po) },
			GetPurchaseOrders:    func() ([]PurchaseOrder, error) { return getPurchaseOrders(a) },
			GetPurchaseOrderByID: func(id string) (PurchaseOrder, error) { return getPurchaseOrderByID(a, id) },
			UpdatePurchaseOrder:  func(po PurchaseOrder) (PurchaseOrder, error) { return updatePurchaseOrder(a, po) },
			ApprovePurchaseOrder: func(id, approvedBy string) error { return approvePurchaseOrder(a, id, approvedBy) },
			CreatePOFromOrder: func(orderID, supplierID string, itemIDs []string) (PurchaseOrder, error) {
				return createPOFromOrder(a, orderID, supplierID, itemIDs)
			},
			CreateGRN:        func(grn GoodsReceivedNote) (GoodsReceivedNote, error) { return createGRN(a, grn) },
			ReceiveAgainstPO: func(poID string, items []GRNItem) (GoodsReceivedNote, error) { return receiveAgainstPO(a, poID, items) },
		}),
		contract: crmcontract.New(a.db),
		license: infralicense.New(a.db, infralicense.Handlers[LicenseKey, LicenseActivationResult, LicenseValidationResult]{
			GenerateLicenseKey: func(role, notes, createdBy string) (string, error) {
				return generateLicenseKey(a, role, notes, createdBy)
			},
			GenerateBatchLicenseKeys: func(role string, count int, notes, createdBy string) ([]string, error) {
				return generateBatchLicenseKeys(a, role, count, notes, createdBy)
			},
			ActivateLicense:      func(key string) (LicenseActivationResult, error) { return activateLicense(a, key) },
			ValidateLicense:      func() (LicenseValidationResult, error) { return validateLicense(a) },
			GetLicenseRole:       func() string { return getLicenseRole(a) },
			HasLicensePermission: func(permission string) bool { return hasLicensePermission(a, permission) },
			ListLicenseKeys:      func() ([]LicenseKey, error) { return listLicenseKeys(a) },
			UpdateLicenseDisplayName: func(key, displayName string) (LicenseKey, error) {
				return updateLicenseDisplayName(a, key, displayName)
			},
			RevokeLicense:                         func(key string) error { return revokeLicense(a, key) },
			EnsureLicenseTableExists:              func() error { return ensureLicenseTableExists(a) },
			SeedLicenseKeys:                       func() error { return seedLicenseKeys(a) },
			ApplyDeploymentLicenseActivationFlush: func() error { return applyDeploymentLicenseActivationFlush(a) },
			SeedEmployeeKeys:                      func() error { return seedEmployeeKeys(a) },
			CheckFirstInstall:                     func() bool { return checkFirstInstall(a) },
			NeedsLicenseActivation:                func() (bool, error) { return needsLicenseActivation(a) },
		}),
		deletion: infradeletion.New(a.db,
			appDeletionIdentityPort{app: a},
			appDeletionNotifierPort{app: a},
			appDeletionExecutorPort{app: a},
		),
		butlerContext: appButlerContext{app: a},
	}
}

func (a *App) deletionService() *infradeletion.Service {
	if a.services.deletion == nil {
		a.initServices()
	}
	return a.services.deletion
}

func (a *App) paymentService() *financepayment.Service[Payment, SupplierPayment] {
	if a.services.payment == nil {
		a.initServices()
	}
	return a.services.payment
}

func (a *App) expenseService() *financeexpense.Service[ExpenseCategory, ExpenseVendor, ExpenseEntry, ExpenseDashboardSummary, RecurringExpense, BankExpenseEntry] {
	if a.services.expense == nil {
		a.initServices()
	}
	return a.services.expense
}

func (a *App) bankingService() *financebanking.Service[BookBankReconciliation, BankReconciliationMatchResult, AllocationInput, BankStatement, BankStatementLine, BalanceContinuityReportData, StatementHash, BankReconciliationAuditLog] {
	if a.services.banking == nil {
		a.initServices()
	}
	return a.services.banking
}

func (a *App) fulfillmentService() *crmfulfillment.Service[DeliveryNote] {
	if a.services.fulfillment == nil {
		a.initServices()
	}
	return a.services.fulfillment
}

func (a *App) serialService() *crmfulfillment.Serials {
	if a.services.serials == nil {
		a.initServices()
	}
	return a.services.serials
}

func (a *App) chequeService() *financecheque.Service {
	if a.services.cheques == nil {
		a.initServices()
	}
	return a.services.cheques
}

func (a *App) fxService() *financefx.Service {
	if a.services.fx == nil {
		a.initServices()
	}
	return a.services.fx
}

func (a *App) assetService() *infraassets.Service {
	if a.services.assets == nil {
		a.initServices()
	}
	return a.services.assets
}

func (a *App) deviceService() *infradevice.Service {
	if a.services.device == nil {
		a.initServices()
	}
	return a.services.device
}

func (a *App) payrollService() *financepayroll.Service {
	if a.services.payroll == nil {
		a.initServices()
	}
	return a.services.payroll
}

func (a *App) procurementService() *crmprocurement.Service[PurchaseOrder, GoodsReceivedNote, GRNItem] {
	if a.services.procurement == nil {
		a.initServices()
	}
	return a.services.procurement
}

func (a *App) licenseService() *infralicense.Service[LicenseKey, LicenseActivationResult, LicenseValidationResult] {
	if a.services.license == nil {
		a.initServices()
	}
	return a.services.license
}
