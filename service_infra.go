package main

import (
	"time"

	"ph_holdings_app/pkg/i18n"
	"ph_holdings_app/pkg/infra/release"
)

// InfraService exposes domain-specific Wails bindings by delegating to App.
type InfraService struct {
	app *App
}

func NewInfraService(app *App) *InfraService {
	return &InfraService{app: app}
}

// --- app.go ---

func (s *InfraService) GetApplicationPaths() *ApplicationPaths {
	return s.app.GetApplicationPaths()
}

func (s *InfraService) GetCSRFToken() string {
	return s.app.GetCSRFToken()
}

func (s *InfraService) TimeNow() time.Time {
	return s.app.TimeNow()
}

func (s *InfraService) GetBuildInfo() release.BuildInfo {
	return release.Current()
}

func (s *InfraService) GetTranslations(locale string) map[string]string {
	messages, err := i18n.LoadEmbedded(i18n.Locale(locale))
	if err != nil {
		messages, _ = i18n.LoadEmbedded(i18n.EN)
	}
	return messages
}

func (s *InfraService) GetAvailableLocales() []string {
	locales := i18n.AvailableLocales()
	out := make([]string, 0, len(locales))
	for _, locale := range locales {
		out = append(out, string(locale))
	}
	return out
}

func (s *InfraService) ValidateCSRFToken(token string) bool {
	return s.app.ValidateCSRFToken(token)
}

// --- app_auth_rbac.go ---

func (s *InfraService) CheckPermissionByRole(roleName, permission string) bool {
	return s.app.CheckPermissionByRole(roleName, permission)
}

func (s *InfraService) CreateUser(username, email, password, fullName, department, jobTitle string, roleID string) (*User, error) {
	return s.app.CreateUser(username, email, password, fullName, department, jobTitle, roleID)
}

func (s *InfraService) DeactivateUser(userID string) error {
	return s.app.DeactivateUser(userID)
}

func (s *InfraService) GetAuditLogs(limit int, resource string, action string) ([]AuditLog, error) {
	return s.app.GetAuditLogs(limit, resource, action)
}

func (s *InfraService) GetCurrentUserRole() string {
	return s.app.GetCurrentUserRole()
}

func (s *InfraService) GetCurrentUserStub() (*User, error) {
	return s.app.GetCurrentUserStub()
}

func (s *InfraService) GetRole(roleID uint) (*Role, error) {
	return s.app.GetRole(roleID)
}

func (s *InfraService) GetRolePermissionsList(roleName string) ([]string, error) {
	return s.app.GetRolePermissionsList(roleName)
}

func (s *InfraService) GetUser(userID string) (*User, error) {
	return s.app.GetUser(userID)
}

func (s *InfraService) GetUserPermissions(userID string) ([]string, error) {
	return s.app.GetUserPermissions(userID)
}

func (s *InfraService) HasPermission(userID string, permission string) bool {
	return s.app.HasPermission(userID, permission)
}

func (s *InfraService) ListRoles() ([]Role, error) {
	return s.app.ListRoles()
}

func (s *InfraService) ListUsers() ([]User, error) {
	return s.app.ListUsers()
}

func (s *InfraService) ResetUserPassword(userID string, newPassword string) error {
	return s.app.ResetUserPassword(userID, newPassword)
}

func (s *InfraService) SeedDefaultRoles() error {
	return s.app.SeedDefaultRoles()
}

func (s *InfraService) UpdateUser(userID string, fullName, email, department, jobTitle string, roleID string, isActive bool) error {
	return s.app.UpdateUser(userID, fullName, email, department, jobTitle, roleID, isActive)
}

// --- app_costing_exports_surface.go ---

func (s *InfraService) CalculateCosting(req CostingRequest) (*CostingResult, error) {
	return s.app.CalculateCosting(req)
}

func (s *InfraService) CreateCustomerFromButler(req ButlerCustomerRequest) (*CustomerMaster, error) {
	return s.app.CreateCustomerFromButler(req)
}

func (s *InfraService) CreateOfferDraftFromButler(req ButlerOfferDraftRequest) (*Offer, error) {
	return s.app.CreateOfferDraftFromButler(req)
}

func (s *InfraService) CreateSupplierFromButler(req ButlerSupplierRequest) (*SupplierMaster, error) {
	return s.app.CreateSupplierFromButler(req)
}

func (s *InfraService) ExportCostingToExcel(data CostingExportData) (string, error) {
	return s.app.ExportCostingToExcel(data)
}

func (s *InfraService) ExportCostingToPDF(data CostingExportData) (string, error) {
	return s.app.ExportCostingToPDF(data)
}

func (s *InfraService) OpenExportedFile(filePath string) error {
	return s.app.OpenExportedFile(filePath)
}

// --- app_dashboard_datafix_surface.go ---

func (s *InfraService) BackfillOfferItemCostBreakdown() (map[string]any, error) {
	return s.app.BackfillOfferItemCostBreakdown()
}

func (s *InfraService) BackfillRFQDocumentTracking() (int, error) {
	return s.app.BackfillRFQDocumentTracking()
}

func (s *InfraService) FixDatabaseDates() (map[string]int, error) {
	return s.app.FixDatabaseDates()
}

func (s *InfraService) FixPurchaseOrderSupplierNames() (int, error) {
	return s.app.FixPurchaseOrderSupplierNames()
}

func (s *InfraService) GetCRMCustomerDashboard() CRMCustomerDashboard {
	return s.app.GetCRMCustomerDashboard()
}

func (s *InfraService) GetCRMCustomerDashboardByYear(year int) CRMCustomerDashboard {
	return s.app.GetCRMCustomerDashboardByYear(year)
}

func (s *InfraService) GetFinancialDashboard() FinancialDashboard {
	return s.app.GetFinancialDashboard()
}

func (s *InfraService) GetFinancialDashboardForYear(year int) (FinancialDashboard, error) {
	return s.app.GetFinancialDashboardForYear(year)
}

func (s *InfraService) RecalculateInvoiceItemCosts() (map[string]any, error) {
	return s.app.RecalculateInvoiceItemCosts()
}

func (s *InfraService) RunAllDataFixes() (map[string]any, error) {
	return s.app.RunAllDataFixes()
}

// --- app_watcher.go ---

func (s *InfraService) ClearSyncHistory() error {
	return s.app.ClearSyncHistory()
}

func (s *InfraService) ConfigureWatchPaths(rfqPath, ehXMLPath, offerPath, invoicePath string) error {
	return s.app.ConfigureWatchPaths(rfqPath, ehXMLPath, offerPath, invoicePath)
}

func (s *InfraService) GetRecentEvents(limit int) []*FileSyncState {
	return s.app.GetRecentEvents(limit)
}

func (s *InfraService) GetRecentSyncEvents(limit int) []FileWatchEvent {
	return s.app.GetRecentSyncEvents(limit)
}

func (s *InfraService) GetSyncStatus() map[string]any {
	return s.app.GetSyncStatus()
}

func (s *InfraService) GetWatcherStatus() map[string]any {
	return s.app.GetWatcherStatus()
}

func (s *InfraService) RetryFailedSyncs() (int, error) {
	return s.app.RetryFailedSyncs()
}

func (s *InfraService) StartFileWatcher() error {
	return s.app.StartFileWatcher()
}

func (s *InfraService) StopFileWatcher() error {
	return s.app.StopFileWatcher()
}

func (s *InfraService) TriggerSync(filePath string) error {
	return s.app.TriggerSync(filePath)
}

// --- archaeologist.go ---

func (s *InfraService) CancelScan(scanID string) error {
	return s.app.CancelScan(scanID)
}

func (s *InfraService) GetScanProgress(scanID string) (ScanProgress, error) {
	return s.app.GetScanProgress(scanID)
}

func (s *InfraService) GetScanResult(scanID string) (ScanResult, error) {
	return s.app.GetScanResult(scanID)
}

func (s *InfraService) StartArchaeologyScan(sourcePath string, isZIP bool, outputDir string) (string, error) {
	return s.app.StartArchaeologyScan(sourcePath, isZIP, outputDir)
}

// --- auth_handler.go ---

func (s *InfraService) GetAccessToken() (string, error) {
	return s.app.GetAccessToken()
}

func (s *InfraService) GetAuthState() *AuthState {
	return s.app.GetAuthState()
}

func (s *InfraService) Logout() error {
	return s.app.Logout()
}

func (s *InfraService) RefreshAuth() error {
	return s.app.RefreshAuth()
}

func (s *InfraService) StartLogin() (string, error) {
	return s.app.StartLogin()
}

// --- auth_session.go ---

func (s *InfraService) GetAccessTokenWithValidation() (string, error) {
	return s.app.GetAccessTokenWithValidation()
}

func (s *InfraService) LogoutWithSession() error {
	return s.app.LogoutWithSession()
}

func (s *InfraService) RefreshAuthWithSession(session *UserSession) error {
	return s.app.RefreshAuthWithSession(session)
}

// --- book_bank_reconciliation_service.go ---

func (s *InfraService) AutoMatchChequesToStatement(bankAccountID, statementID string) (int, error) {
	return s.app.AutoMatchChequesToStatement(bankAccountID, statementID)
}

func (s *InfraService) AutoMatchDepositsToStatement(bankAccountID, statementID string) (int, error) {
	return s.app.AutoMatchDepositsToStatement(bankAccountID, statementID)
}

func (s *InfraService) ClearDepositInTransit(depositID, matchedLineID string, clearedDate time.Time) error {
	return s.app.ClearDepositInTransit(depositID, matchedLineID, clearedDate)
}

func (s *InfraService) CreateBookBankReconciliation(bankAccountID string, reconciliationDate time.Time, bankStatementBalance, bookBalance, depositsInTransit, outstandingCheques float64) (*BookBankReconciliation, error) {
	return s.app.CreateBookBankReconciliation(bankAccountID, reconciliationDate, bankStatementBalance, bookBalance, depositsInTransit, outstandingCheques)
}

func (s *InfraService) UpdateBookBankReconciliationAdjustments(reconID string, depositsInTransit, outstandingCheques float64) error {
	return s.app.UpdateBookBankReconciliationAdjustments(reconID, depositsInTransit, outstandingCheques)
}

func (s *InfraService) CreateDepositInTransit(bankAccountID string, depositDate time.Time, amount float64, slipNo, description, sourceType string, customerID *string, invoiceIDs []string) (*DepositInTransit, error) {
	return s.app.CreateDepositInTransit(bankAccountID, depositDate, amount, slipNo, description, sourceType, customerID, invoiceIDs)
}

func (s *InfraService) FinalizeBookBankReconciliation(reconID, user string) error {
	return s.app.FinalizeBookBankReconciliation(reconID, user)
}

func (s *InfraService) GetBookBankReconciliation(reconID string) (*BookBankReconciliation, error) {
	return s.app.GetBookBankReconciliation(reconID)
}

func (s *InfraService) GetBookBankReconciliationReport(reconID string) (*BookBankReconciliationReport, error) {
	return s.app.GetBookBankReconciliationReport(reconID)
}

func (s *InfraService) GetBookBankReconciliations(bankAccountID string) ([]BookBankReconciliation, error) {
	return s.app.GetBookBankReconciliations(bankAccountID)
}

func (s *InfraService) GetDepositsInTransit(bankAccountID string) (*DepositsInTransitResult, error) {
	return s.app.GetDepositsInTransit(bankAccountID)
}

func (s *InfraService) GetLatestBookBankReconciliation(bankAccountID string) (*BookBankReconciliation, error) {
	return s.app.GetLatestBookBankReconciliation(bankAccountID)
}

func (s *InfraService) GetReconciliationStatusSummary() ([]ReconciliationStatusSummary, error) {
	return s.app.GetReconciliationStatusSummary()
}

func (s *InfraService) GetReconciliationVariances(reconID string) ([]VarianceItem, error) {
	return s.app.GetReconciliationVariances(reconID)
}

func (s *InfraService) ReturnDeposit(depositID, reason string) error {
	return s.app.ReturnDeposit(depositID, reason)
}

func (s *InfraService) UpdateBookBankReconciliation(reconID string, bankCharges, interest, nsfCheques, bankErrors, bookErrors float64, notes string) (*BookBankReconciliation, error) {
	return s.app.UpdateBookBankReconciliation(reconID, bankCharges, interest, nsfCheques, bankErrors, bookErrors, notes)
}

// --- dashboard_stats_v2.go ---

func (s *InfraService) GetDashboardStatsV2() (DashboardStatsV2, error) {
	return s.app.GetDashboardStatsV2()
}

// --- data_reconciliation.go ---

func (s *InfraService) AddManualCustomerMapping(extractedName string, customerID string) error {
	return s.app.AddManualCustomerMapping(extractedName, customerID)
}

func (s *InfraService) ApplyReconciliation(resultJSON string) error {
	return s.app.ApplyReconciliation(resultJSON)
}

func (s *InfraService) ExportReconciliationReport(result *ReconciliationResult, outputPath string) error {
	return s.app.ExportReconciliationReport(result, outputPath)
}

func (s *InfraService) GetCustomerNameMapping() (map[string]string, error) {
	return s.app.GetCustomerNameMapping()
}

func (s *InfraService) GetDiscrepanciesByType(resultJSON string, discType string, minSeverity string) ([]DataDiscrepancy, error) {
	return s.app.GetDiscrepanciesByType(resultJSON, discType, minSeverity)
}

func (s *InfraService) ReconcileOfferData(basePath string) (*ReconciliationResult, error) {
	return s.app.ReconcileOfferData(basePath)
}

// --- database.go ---

func (s *InfraService) BackupDatabase() (string, error) {
	return s.app.BackupDatabase()
}

func (s *InfraService) GetBackupInfo() map[string]any {
	return s.app.GetBackupInfo()
}

func (s *InfraService) GetBackupPolicy() (BackupPolicy, error) {
	return s.app.GetBackupPolicy()
}

func (s *InfraService) RunIntegrityCheck() string {
	return s.app.RunIntegrityCheck()
}

func (s *InfraService) RunScheduledBackupIfDue(reason string) map[string]any {
	return s.app.RunScheduledBackupIfDue(reason)
}

func (s *InfraService) SaveBackupPolicy(autoEnabled bool, frequencyDays int) (BackupPolicy, error) {
	return s.app.SaveBackupPolicy(autoEnabled, frequencyDays)
}

func (s *InfraService) TriggerBackup() map[string]any {
	return s.app.TriggerBackup()
}

// --- delete_approval_service.go ---

func (s *InfraService) RequestDeleteApproval(entityType, entityID, entityLabel, reason string) (*DeleteApprovalRequest, error) {
	return s.app.RequestDeleteApproval(entityType, entityID, entityLabel, reason)
}

func (s *InfraService) ReviewDeleteApprovalRequest(requestID, decision, notes string) (*DeleteApprovalRequest, error) {
	return s.app.ReviewDeleteApprovalRequest(requestID, decision, notes)
}

// --- deployment_audit.go ---

func (s *InfraService) GetDeploymentDataAudit() (DeploymentDataAudit, error) {
	return s.app.GetDeploymentDataAudit()
}

// --- device_service.go ---

func (s *InfraService) ApproveDevice(deviceID, roleID, username, password, fullName, email string) error {
	return s.app.ApproveDevice(deviceID, roleID, username, password, fullName, email)
}

func (s *InfraService) BlockDevice(deviceID string) error {
	return s.app.BlockDevice(deviceID)
}

func (s *InfraService) CheckDeviceStatus() (*DeviceRegistrationResult, error) {
	return s.app.CheckDeviceStatus()
}

func (s *InfraService) GetCurrentDeviceInfo() (*Device, error) {
	return s.app.GetCurrentDeviceInfo()
}

func (s *InfraService) GetDeviceUsers(deviceID string) ([]DeviceUser, error) {
	return s.app.GetDeviceUsers(deviceID)
}

func (s *InfraService) ListAllDevices() ([]Device, error) {
	return s.app.ListAllDevices()
}

func (s *InfraService) ListPendingDevices() ([]Device, error) {
	return s.app.ListPendingDevices()
}

func (s *InfraService) LoginDevice(username, password string) (*DeviceRegistrationResult, error) {
	return s.app.LoginDevice(username, password)
}

func (s *InfraService) RegisterDevice() (*DeviceRegistrationResult, error) {
	return s.app.RegisterDevice()
}

func (s *InfraService) SetupAdminAccount(username, password, fullName, email string) (*User, error) {
	return s.app.SetupAdminAccount(username, password, fullName, email)
}

func (s *InfraService) UnblockDevice(deviceID string) error {
	return s.app.UnblockDevice(deviceID)
}

// --- import_2026_data.go ---

func (s *InfraService) Import2026BusinessData(offersPath string) map[string]any {
	return s.app.Import2026BusinessData(offersPath)
}

func (s *InfraService) SeedAdditionalBankAccounts() error {
	return s.app.SeedAdditionalBankAccounts()
}

// --- job_handlers.go ---

func (s *InfraService) CancelJob(jobID string) error {
	return s.app.CancelJob(jobID)
}

func (s *InfraService) CleanupOldJobs(retentionDays int) error {
	return s.app.CleanupOldJobs(retentionDays)
}

func (s *InfraService) GenerateReportAsync(category, format, startDate, endDate string) (string, error) {
	return s.app.GenerateReportAsync(category, format, startDate, endDate)
}

func (s *InfraService) GetJobStatus(jobID string) (*JobStatusResponse, error) {
	return s.app.GetJobStatus(jobID)
}

func (s *InfraService) GetRecentJobs(limit int) ([]JobSummary, error) {
	return s.app.GetRecentJobs(limit)
}

func (s *InfraService) InitializeJobQueue() error {
	return s.app.InitializeJobQueue()
}

func (s *InfraService) ShutdownJobQueue() {
	s.app.ShutdownJobQueue()
}

// --- license_service.go ---

func (s *InfraService) ActivateLicense(key string) (LicenseActivationResult, error) {
	return s.app.ActivateLicense(key)
}

func (s *InfraService) ApplyDeploymentLicenseActivationFlush() error {
	return s.app.ApplyDeploymentLicenseActivationFlush()
}

func (s *InfraService) CheckFirstInstall() bool {
	return s.app.CheckFirstInstall()
}

func (s *InfraService) EnsureLicenseTableExists() error {
	return s.app.EnsureLicenseTableExists()
}

func (s *InfraService) GenerateBatchLicenseKeys(role string, count int, notes, createdBy string) ([]string, error) {
	return s.app.GenerateBatchLicenseKeys(role, count, notes, createdBy)
}

func (s *InfraService) GenerateLicenseKey(role, notes, createdBy string) (string, error) {
	return s.app.GenerateLicenseKey(role, notes, createdBy)
}

func (s *InfraService) GetLicenseRole() string {
	return s.app.GetLicenseRole()
}

func (s *InfraService) HasLicensePermission(permission string) bool {
	return s.app.HasLicensePermission(permission)
}

func (s *InfraService) ListLicenseKeys() ([]LicenseKey, error) {
	return s.app.ListLicenseKeys()
}

func (s *InfraService) NeedsLicenseActivation() (bool, error) {
	return s.app.NeedsLicenseActivation()
}

func (s *InfraService) RevokeLicense(key string) error {
	return s.app.RevokeLicense(key)
}

func (s *InfraService) SeedEmployeeKeys() error {
	return s.app.SeedEmployeeKeys()
}

func (s *InfraService) SeedLicenseKeys() error {
	return s.app.SeedLicenseKeys()
}

func (s *InfraService) UpdateLicenseDisplayName(key, displayName string) (LicenseKey, error) {
	return s.app.UpdateLicenseDisplayName(key, displayName)
}

func (s *InfraService) ValidateLicense() (LicenseValidationResult, error) {
	return s.app.ValidateLicense()
}

// --- master_data_cleanup.go ---

func (s *InfraService) GetMasterDataCleanupAudit() (*MasterDataCleanupAudit, error) {
	return s.app.GetMasterDataCleanupAudit()
}

func (s *InfraService) WriteMasterDataCleanupReport(outputPath string) (string, error) {
	return s.app.WriteMasterDataCleanupReport(outputPath)
}

// --- onedrive_import_service.go ---

func (s *InfraService) ConfirmOneDriveDeal(localID string, customerID string) (map[string]any, error) {
	return s.app.ConfirmOneDriveDeal(localID, customerID)
}

func (s *InfraService) ImportOneDriveDeals(deals []DiscoveredDeal) ([]OneDriveImportResult, error) {
	return s.app.ImportOneDriveDeals(deals)
}

func (s *InfraService) ScanOneDrivePaths(paths []string) (OneDriveScanResult, error) {
	return s.app.ScanOneDrivePaths(paths)
}

func (s *InfraService) ValidateOneDrivePath(path string) (map[string]any, error) {
	return s.app.ValidateOneDrivePath(path)
}

// --- pagination.go ---

func (s *InfraService) ListCustomersPaginated(page, pageSize int) (PaginationResult, error) {
	return s.app.ListCustomersPaginated(page, pageSize)
}

func (s *InfraService) ListInvoicesPaginated(page, pageSize int, status string) (PaginationResult, error) {
	return s.app.ListInvoicesPaginated(page, pageSize, status)
}

func (s *InfraService) ListOffersPaginated(page, pageSize int, stage string) (PaginationResult, error) {
	return s.app.ListOffersPaginated(page, pageSize, stage)
}

func (s *InfraService) ListOrdersPaginated(page, pageSize int, status string) (PaginationResult, error) {
	return s.app.ListOrdersPaginated(page, pageSize, status)
}

func (s *InfraService) ListProductsPaginated(page, pageSize int, category string, activeOnly bool) (PaginationResult, error) {
	return s.app.ListProductsPaginated(page, pageSize, category, activeOnly)
}

func (s *InfraService) ListPurchaseOrdersPaginated(page, pageSize int, status string) (PaginationResult, error) {
	return s.app.ListPurchaseOrdersPaginated(page, pageSize, status)
}

func (s *InfraService) ListSuppliersPaginated(page, pageSize int) (PaginationResult, error) {
	return s.app.ListSuppliersPaginated(page, pageSize)
}

// --- performance_optimizations.go ---

func (s *InfraService) GetCollectionPerformance() (map[string]any, error) {
	return s.app.GetCollectionPerformance()
}

func (s *InfraService) GetSurvivalMetricsOptimized() (SurvivalMetrics, error) {
	return s.app.GetSurvivalMetricsOptimized()
}

// --- phase7_rollout.go ---

func (s *InfraService) EnsurePhase7Rollout() error {
	return s.app.EnsurePhase7Rollout()
}

func (s *InfraService) ExportPilotSignoffReport() (PilotExportResult, error) {
	return s.app.ExportPilotSignoffReport()
}

func (s *InfraService) ExportPilotSupportBundle() (PilotSupportBundleResult, error) {
	return s.app.ExportPilotSupportBundle()
}

func (s *InfraService) GetPhase7RolloutStatus() Phase7RolloutStatus {
	return s.app.GetPhase7RolloutStatus()
}

func (s *InfraService) GetPilotDeploymentChecklist() ([]PilotChecklistItem, error) {
	return s.app.GetPilotDeploymentChecklist()
}

func (s *InfraService) GetPilotReadinessSummary() PilotReadinessSummary {
	return s.app.GetPilotReadinessSummary()
}

func (s *InfraService) ListCollaborativePendingOperations(status string, limit int) ([]CollaborativePendingOperation, error) {
	return s.app.ListCollaborativePendingOperations(status, limit)
}

func (s *InfraService) ListPilotReadinessRows(onlyIssues bool) ([]PilotReadinessRow, error) {
	return s.app.ListPilotReadinessRows(onlyIssues)
}

func (s *InfraService) RerunPhase7FollowUpBackfill() (Phase7ActionResult, error) {
	return s.app.RerunPhase7FollowUpBackfill()
}

func (s *InfraService) RetryCollaborativePendingOperation(operationID string) error {
	return s.app.RetryCollaborativePendingOperation(operationID)
}

func (s *InfraService) RetryCollaborativePendingOperations(status string, limit int) (Phase7ActionResult, error) {
	return s.app.RetryCollaborativePendingOperations(status, limit)
}

func (s *InfraService) TriggerCollaborativeSyncNow() error {
	return s.app.TriggerCollaborativeSyncNow()
}

func (s *InfraService) UpdatePilotDeploymentChecklistItem(itemID string, completed bool, notes string) ([]PilotChecklistItem, error) {
	return s.app.UpdatePilotDeploymentChecklistItem(itemID, completed, notes)
}

// --- query_optimizations.go ---

func (s *InfraService) GetCacheStats() map[string]any {
	return s.app.GetCacheStats()
}

func (s *InfraService) GetDashboardStatsOptimized() (map[string]any, error) {
	return s.app.GetDashboardStatsOptimized()
}

func (s *InfraService) GetInvoiceWithItems(invoiceID string) (*Invoice, error) {
	return s.app.GetInvoiceWithItems(invoiceID)
}

func (s *InfraService) GetOrderWithItems(orderID string) (*Order, error) {
	return s.app.GetOrderWithItems(orderID)
}

func (s *InfraService) InvalidateCache(pattern string) {
	s.app.InvalidateCache(pattern)
}

func (s *InfraService) ListCustomersOptimized(limit int) ([]CustomerMaster, error) {
	return s.app.ListCustomersOptimized(limit)
}

func (s *InfraService) ListProductsOptimized(limit int, activeOnly bool) ([]ProductMaster, error) {
	return s.app.ListProductsOptimized(limit, activeOnly)
}

func (s *InfraService) ListSuppliersOptimized(limit int) ([]SupplierMaster, error) {
	return s.app.ListSuppliersOptimized(limit)
}

// --- report_generators.go ---

func (s *InfraService) RegisterReportHandlers() {
	s.app.RegisterReportHandlers()
}

// --- reports.go ---

func (s *InfraService) ExportReport(reportType string, format string, dataJSON string) (string, error) {
	return s.app.ExportReport(reportType, format, dataJSON)
}

func (s *InfraService) GenerateCustomer360Report(customerID string) (string, error) {
	return s.app.GenerateCustomer360Report(customerID)
}

func (s *InfraService) GenerateDashboardReport() (string, error) {
	return s.app.GenerateDashboardReport()
}

func (s *InfraService) GeneratePredictionHistoryReport(limit int) (string, error) {
	return s.app.GeneratePredictionHistoryReport(limit)
}

func (s *InfraService) GetReportData(reportType string, dateRange string) (ReportData, error) {
	return s.app.GetReportData(reportType, dateRange)
}

// --- runtime_handlers.go ---

func (s *InfraService) GetInboxDocuments(status string) ([]InboxDocument, error) {
	return s.app.GetInboxDocuments(status)
}

func (s *InfraService) GetInboxStats() (*InboxStats, error) {
	return s.app.GetInboxStats()
}

func (s *InfraService) GetPricingRecommendation(customer string, historicalData map[string]any) (*PricingRecommendation, error) {
	return s.app.GetPricingRecommendation(customer, historicalData)
}

func (s *InfraService) MarkInboxDocumentProcessed(documentID string, action string) error {
	return s.app.MarkInboxDocumentProcessed(documentID, action)
}

func (s *InfraService) ProcessInboxDocument(filePath string, rawText string) (*InboxProcessResult, error) {
	return s.app.ProcessInboxDocument(filePath, rawText)
}

func (s *InfraService) SimulateMargin(customer string, proposedMargin float64) (*MarginSimulation, error) {
	return s.app.SimulateMargin(customer, proposedMargin)
}

func (s *InfraService) StoreCustomerGraph(customerID string, businessName string, properties map[string]any) error {
	return s.app.StoreCustomerGraph(customerID, businessName, properties)
}

// --- user_activity_monitoring.go ---

func (s *InfraService) CanViewUserActivityMonitoring() bool {
	return s.app.CanViewUserActivityMonitoring()
}

func (s *InfraService) EndUserActivitySession(sessionID string) error {
	return s.app.EndUserActivitySession(sessionID)
}

func (s *InfraService) EnsureUserActivityMonitoringFoundation() error {
	return s.app.EnsureUserActivityMonitoringFoundation()
}

func (s *InfraService) GetWeeklyUserActivityReport(weekStart string) (UserActivityWeeklyReport, error) {
	return s.app.GetWeeklyUserActivityReport(weekStart)
}

func (s *InfraService) RecordUserActivityBatch(events []UserActivityEventInput) error {
	return s.app.RecordUserActivityBatch(events)
}

func (s *InfraService) RecordUserActivityHeartbeat(input UserActivityHeartbeatInput) error {
	return s.app.RecordUserActivityHeartbeat(input)
}

func (s *InfraService) StartUserActivitySession(source string) (UserActivitySession, error) {
	return s.app.StartUserActivitySession(source)
}
