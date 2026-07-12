package main

import (
	"time"

	documents "ph_holdings_app/internal/viewmodel/documents"
)

// DocumentsService exposes domain-specific Wails bindings by delegating to App.
type DocumentsService struct {
	app *App
}

func NewDocumentsService(app *App) *DocumentsService {
	return &DocumentsService{app: app}
}

// --- app_setup_documents_surface.go ---

func (s *DocumentsService) BrowseFolder() (string, error) {
	return s.app.BrowseFolder()
}

func (s *DocumentsService) CompleteSetup() error {
	return s.app.CompleteSetup()
}

func (s *DocumentsService) CreateCustomerFolder(basePath string, company string, customerName string) (string, error) {
	return s.app.CreateCustomerFolder(basePath, company, customerName)
}

func (s *DocumentsService) CreateFolderStructure(basePath string, companyName string) (FolderStructureResult, error) {
	return s.app.CreateFolderStructure(basePath, companyName)
}

func (s *DocumentsService) CreateQuickCapture(title, content, tags, priority string) (uint, error) {
	return s.app.CreateQuickCapture(title, content, tags, priority)
}

func (s *DocumentsService) CreateSupplierFolder(basePath string, company string, supplierName string) (string, error) {
	return s.app.CreateSupplierFolder(basePath, company, supplierName)
}

func (s *DocumentsService) DeleteQuickCapture(id string) error {
	return s.app.DeleteQuickCapture(id)
}

func (s *DocumentsService) DetectGPU() (GPUInfo, error) {
	return s.app.DetectGPU()
}

func (s *DocumentsService) DetectOffice() (OfficeInfo, error) {
	return s.app.DetectOffice()
}

func (s *DocumentsService) DetectOneDrivePath() (string, error) {
	return s.app.DetectOneDrivePath()
}

func (s *DocumentsService) DetectSystemInfo() (SystemInfo, error) {
	return s.app.DetectSystemInfo()
}

func (s *DocumentsService) ExportEncryptionBackup() (map[string]string, error) {
	return s.app.ExportEncryptionBackup()
}

func (s *DocumentsService) ExportBusinessMemoryReviewBundle(candidateID string) (*BusinessMemoryReviewExportResult, error) {
	return s.app.ExportBusinessMemoryReviewBundle(candidateID)
}

func (s *DocumentsService) ExtractInvoiceDocument(filePath string) *OCRResult {
	return s.app.ExtractInvoiceDocument(filePath)
}

func (s *DocumentsService) ExtractQuotationDocument(filePath string) *OCRResult {
	return s.app.ExtractQuotationDocument(filePath)
}

func (s *DocumentsService) ExtractRFQDocument(filePath string) *OCRResult {
	return s.app.ExtractRFQDocument(filePath)
}

func (s *DocumentsService) GenerateSeedData(opportunitiesFile, outputPath string) (map[string]any, error) {
	return s.app.GenerateSeedData(opportunitiesFile, outputPath)
}

func (s *DocumentsService) GenerateBusinessMemoryContextPack(candidateID string) (*BusinessMemoryContextPackResult, error) {
	return s.app.GenerateBusinessMemoryContextPack(candidateID)
}

func (s *DocumentsService) GetActiveCurrencyRates() ([]CurrencyExchangeRate, error) {
	return s.app.GetActiveCurrencyRates()
}

func (s *DocumentsService) GetBusinessMemoryReviewQueue(selectedID string) (documents.IntakeReviewVM, error) {
	return s.app.GetBusinessMemoryReviewQueue(selectedID)
}

func (s *DocumentsService) GetBusinessVATRate() float64 {
	return s.app.GetBusinessVATRate()
}

func (s *DocumentsService) GetCurrencyRateHistory(currencyCode string) ([]CurrencyExchangeRate, error) {
	return s.app.GetCurrencyRateHistory(currencyCode)
}

func (s *DocumentsService) GetCurrentExchangeRate(currencyCode string) (float64, error) {
	return s.app.GetCurrentExchangeRate(currencyCode)
}

func (s *DocumentsService) GetExchangeRate(currencyCode string, asOfDate time.Time) (float64, error) {
	return s.app.GetExchangeRate(currencyCode, asOfDate)
}

func (s *DocumentsService) GetFolderPaths() (map[string]string, error) {
	return s.app.GetFolderPaths()
}

func (s *DocumentsService) GetOCRDocumentByID(id string) (*OCRDocument, error) {
	return s.app.GetOCRDocumentByID(id)
}

func (s *DocumentsService) GetOCRDocuments(limit int) ([]OCRDocument, error) {
	return s.app.GetOCRDocuments(limit)
}

func (s *DocumentsService) GetOCRDocumentsByType(docType string, limit int) ([]OCRDocument, error) {
	return s.app.GetOCRDocumentsByType(docType, limit)
}

func (s *DocumentsService) GetOCRPipelineStats() (map[string]any, error) {
	return s.app.GetOCRPipelineStats()
}

func (s *DocumentsService) GetOCRProcessorStats() (map[string]any, error) {
	return s.app.GetOCRProcessorStats()
}

func (s *DocumentsService) GetOCRStats() (map[string]any, error) {
	return s.app.GetOCRStats()
}

func (s *DocumentsService) GetQuickCaptures(limit int) ([]QuickCapture, error) {
	return s.app.GetQuickCaptures(limit)
}

func (s *DocumentsService) GetSettings() (map[string]any, error) {
	return s.app.GetSettings()
}

func (s *DocumentsService) GetSupportedCurrencies() ([]map[string]string, error) {
	return s.app.GetSupportedCurrencies()
}

func (s *DocumentsService) ImportBankStatementWithDialog(bankAccountID string) (*BankStatement, error) {
	return s.app.ImportBankStatementWithDialog(bankAccountID)
}

// Wave 9.3 B1d: statement-import preview (parse -> review -> confirm) siblings of
// ImportBankStatementWithDialog, exposed on DocumentsService for binding consistency.
func (s *DocumentsService) PreviewBankStatementImportWithDialog(bankAccountID string) (*BankStatement, error) {
	return s.app.PreviewBankStatementImportWithDialog(bankAccountID)
}

func (s *DocumentsService) ConfirmBankStatementImport(previewID string) (*BankStatement, error) {
	return s.app.ConfirmBankStatementImport(previewID)
}

func (s *DocumentsService) DiscardBankStatementImportPreview(previewID string) {
	s.app.DiscardBankStatementImportPreview(previewID)
}

func (s *DocumentsService) ImportEncryptionBackup(masterKeyHex, saltHex string) error {
	return s.app.ImportEncryptionBackup(masterKeyHex, saltHex)
}

func (s *DocumentsService) NeedsSetup() bool {
	return s.app.NeedsSetup()
}

func (s *DocumentsService) PickCSVFile(title string) (string, error) {
	return s.app.PickCSVFile(title)
}

func (s *DocumentsService) PickFile(title string) (string, error) {
	return s.app.PickFile(title)
}

func (s *DocumentsService) ProcessDocumentWithOCR(filePath string, docType string) (*OCRResult, error) {
	return s.app.ProcessDocumentWithOCR(filePath, docType)
}

func (s *DocumentsService) RecordBusinessMemoryReviewDecision(req BusinessMemoryReviewDecisionRequest) (*BusinessMemoryReviewResult, error) {
	return s.app.RecordBusinessMemoryReviewDecision(req)
}

func (s *DocumentsService) ProcessDocumentsBatch(filePaths []string, docType string) []*OCRResult {
	return s.app.ProcessDocumentsBatch(filePaths, docType)
}

func (s *DocumentsService) ProcessOffersBatch(offersFolder string) (*BatchOfferResult, error) {
	return s.app.ProcessOffersBatch(offersFolder)
}

func (s *DocumentsService) ProcessWithFlorence2(filePath string) *OCRResult {
	return s.app.ProcessWithFlorence2(filePath)
}

func (s *DocumentsService) ProcessWithGPU(filePath string) *OCRResult {
	return s.app.ProcessWithGPU(filePath)
}

func (s *DocumentsService) ProcessWithGoFitz(filePath string) *OCRResult {
	return s.app.ProcessWithGoFitz(filePath)
}

func (s *DocumentsService) ProcessWithTesseract(filePath string) *OCRResult {
	return s.app.ProcessWithTesseract(filePath)
}

func (s *DocumentsService) QuickCaptureDocument(filePath string) (map[string]any, error) {
	return s.app.QuickCaptureDocument(filePath)
}

func (s *DocumentsService) QuickCaptureDocumentFromBase64(base64Data string, fileName string) (map[string]any, error) {
	return s.app.QuickCaptureDocumentFromBase64(base64Data, fileName)
}

func (s *DocumentsService) RotateEncryptionKey() error {
	return s.app.RotateEncryptionKey()
}

func (s *DocumentsService) RunBatchOCR(basePath, opportunitiesFile, outputDir string) (map[string]any, error) {
	return s.app.RunBatchOCR(basePath, opportunitiesFile, outputDir)
}

func (s *DocumentsService) RunInitialScan() (InitialScanResult, error) {
	return s.app.RunInitialScan()
}

func (s *DocumentsService) SaveDocumentToEntity(fileName, filePath, documentType, extractedText string, confidence float64, processingTimeMS int64, engine string, extractedDataJSON string) (map[string]any, error) {
	return s.app.SaveDocumentToEntity(fileName, filePath, documentType, extractedText, confidence, processingTimeMS, engine, extractedDataJSON)
}

func (s *DocumentsService) SaveOCRDocument(fileName, filePath, documentType, extractedText string, confidence float64, processingTimeMS int64, engine string, extractedDataJSON string) (*OCRDocument, error) {
	return s.app.SaveOCRDocument(fileName, filePath, documentType, extractedText, confidence, processingTimeMS, engine, extractedDataJSON)
}

func (s *DocumentsService) SeedBankDemoData() error {
	return s.app.SeedBankDemoData()
}

func (s *DocumentsService) SeedDatabaseFromOpportunities(opportunitiesFile string) (map[string]any, error) {
	return s.app.SeedDatabaseFromOpportunities(opportunitiesFile)
}

func (s *DocumentsService) SeedDefaultExchangeRates() error {
	return s.app.SeedDefaultExchangeRates()
}

func (s *DocumentsService) SetAPIKeys(apiKeys map[string]string) error {
	return s.app.SetAPIKeys(apiKeys)
}

func (s *DocumentsService) SetExchangeRate(currencyCode string, rate float64, effectiveFrom time.Time, notes string) error {
	return s.app.SetExchangeRate(currencyCode, rate, effectiveFrom, notes)
}

func (s *DocumentsService) TestAIConnection(provider string, apiKey string) error {
	return s.app.TestAIConnection(provider, apiKey)
}

func (s *DocumentsService) UpdateFolderPaths(paths map[string]string) error {
	return s.app.UpdateFolderPaths(paths)
}

func (s *DocumentsService) UpdateQuickCapture(id uint, title, content, tags, priority, status string) error {
	return s.app.UpdateQuickCapture(id, title, content, tags, priority, status)
}

func (s *DocumentsService) UpdateSettings(settings map[string]any) error {
	return s.app.UpdateSettings(settings)
}

func (s *DocumentsService) ValidateFolder(path string) (bool, error) {
	return s.app.ValidateFolder(path)
}

func (s *DocumentsService) WatchInboxForTestFile(inboxPath string) error {
	return s.app.WatchInboxForTestFile(inboxPath)
}

// --- assets_service.go ---

func (s *DocumentsService) DeleteAsset(name string) error {
	return s.app.DeleteAsset(name)
}

func (s *DocumentsService) EnsureAssetsTable() error {
	return s.app.EnsureAssetsTable()
}

func (s *DocumentsService) GetAsset(name string) ([]byte, error) {
	return s.app.GetAsset(name)
}

func (s *DocumentsService) GetAssetToFile(name string) (string, error) {
	return s.app.GetAssetToFile(name)
}

func (s *DocumentsService) GetLetterheadPath() string {
	return s.app.GetLetterheadPath()
}

func (s *DocumentsService) HasAsset(name string) bool {
	return s.app.HasAsset(name)
}

func (s *DocumentsService) InitializeDefaultAssets() {
	s.app.InitializeDefaultAssets()
}

func (s *DocumentsService) ListAssets() ([]AssetInfo, error) {
	return s.app.ListAssets()
}

func (s *DocumentsService) UploadAsset(name, description, filePath string) error {
	return s.app.UploadAsset(name, description, filePath)
}

// --- document_classifier.go ---

func (s *DocumentsService) AIClassifyDocumentType(text string, filename string) *ClassificationResult {
	return s.app.AIClassifyDocumentType(text, filename)
}

func (s *DocumentsService) ClassifyDocument(text string, filename string) *ClassificationResult {
	return s.app.ClassifyDocument(text, filename)
}

func (s *DocumentsService) ClassifyFilesystemDocuments(basePath string) (*FilesystemClassificationSummary, error) {
	return s.app.ClassifyFilesystemDocuments(basePath)
}

func (s *DocumentsService) GetClassificationStats() (map[string]any, error) {
	return s.app.GetClassificationStats()
}

func (s *DocumentsService) GetFilesystemClassificationStats(basePath string) (map[string]any, error) {
	return s.app.GetFilesystemClassificationStats(basePath)
}

func (s *DocumentsService) GetFilesystemDocumentsByCustomer(basePath, customerName string) ([]FilesystemDocClassification, error) {
	return s.app.GetFilesystemDocumentsByCustomer(basePath, customerName)
}

func (s *DocumentsService) GetFilesystemDocumentsByOfferNumber(basePath, offerNumber string) ([]FilesystemDocClassification, error) {
	return s.app.GetFilesystemDocumentsByOfferNumber(basePath, offerNumber)
}

func (s *DocumentsService) GetFilesystemDocumentsByType(basePath, docType string) ([]FilesystemDocClassification, error) {
	return s.app.GetFilesystemDocumentsByType(basePath, docType)
}

// --- excel_costing_parser.go ---

func (s *DocumentsService) BatchImportCostingSheets(basePath string) (*ExcelBatchImportResult, error) {
	return s.app.BatchImportCostingSheets(basePath)
}

func (s *DocumentsService) ImportCostingToOpportunity(data *ExcelCostingData) error {
	return s.app.ImportCostingToOpportunity(data)
}

func (s *DocumentsService) ParseCostingSheetFile(filePath string) (*ExcelCostingData, error) {
	return s.app.ParseCostingSheetFile(filePath)
}

func (s *DocumentsService) ParseCostingSheetWithDialog() (*ExcelCostingData, error) {
	return s.app.ParseCostingSheetWithDialog()
}

func (s *DocumentsService) PickExcelFile() (string, error) {
	return s.app.PickExcelFile()
}

func (s *DocumentsService) ScanOfferFolders(basePath string) (*ExcelBatchImportResult, error) {
	return s.app.ScanOfferFolders(basePath)
}

// --- excel_template_generator.go ---

func (s *DocumentsService) GenerateDataImportTemplate() (string, error) {
	return s.app.GenerateDataImportTemplate()
}

// --- invoice_pdf_service.go ---

func (s *DocumentsService) GenerateInvoicePDF(invoiceID string) (string, error) {
	return s.app.GenerateInvoicePDF(invoiceID)
}

func (s *DocumentsService) GenerateSupplierInvoicePDF(invoiceID string) (string, error) {
	return s.app.GenerateSupplierInvoicePDF(invoiceID)
}

// --- msg_parser.go ---

func (s *DocumentsService) BatchParseMSGFiles(basePath string) (*BatchParseResult, error) {
	return s.app.BatchParseMSGFiles(basePath)
}

func (s *DocumentsService) GetMSGFileInfo(msgPath string) (map[string]any, error) {
	return s.app.GetMSGFileInfo(msgPath)
}

func (s *DocumentsService) ListMSGFilesInDirectory(dirPath string) ([]map[string]any, error) {
	return s.app.ListMSGFilesInDirectory(dirPath)
}

func (s *DocumentsService) ParseMSGFile(msgPath string) (*ParsedRFQEmail, error) {
	return s.app.ParseMSGFile(msgPath)
}

func (s *DocumentsService) ParseMSGFileToJSON(msgPath string) (string, error) {
	return s.app.ParseMSGFileToJSON(msgPath)
}

func (s *DocumentsService) SaveParsedEmailAsRFQ(parsedEmail *ParsedRFQEmail) (uint, error) {
	return s.app.SaveParsedEmailAsRFQ(parsedEmail)
}

// --- offer_pdf_service.go ---

func (s *DocumentsService) GenerateOfferPDF(offerID string) (string, error) {
	return s.app.GenerateOfferPDF(offerID)
}

// --- purchase_order_pdf_service.go ---

func (s *DocumentsService) GeneratePurchaseOrderPDF(poID string) (string, error) {
	return s.app.GeneratePurchaseOrderPDF(poID)
}

// --- report_storage.go ---

func (s *DocumentsService) DeleteRemoteReport(remotePath string) error {
	return s.app.DeleteRemoteReport(remotePath)
}

func (s *DocumentsService) DownloadReport(remotePath string) (string, error) {
	return s.app.DownloadReport(remotePath)
}

func (s *DocumentsService) ListRemoteReports() ([]ReportMetadata, error) {
	return s.app.ListRemoteReports()
}

func (s *DocumentsService) UploadReportToStorage(localPath string) (string, error) {
	return s.app.UploadReportToStorage(localPath)
}
