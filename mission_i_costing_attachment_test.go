package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"
	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/stretchr/testify/require"
)

// Mission I item I-25: costing_sheet_attachments model + attachment service +
// technical-datasheet bundling into the exported quotation PDF.

func writeTinyPDFForAttachmentTest(t *testing.T, path string, text string) {
	t.Helper()
	pdf := gofpdf.New("P", "pt", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "", 12)
	pdf.Text(40, 80, text)
	require.NoError(t, pdf.OutputFileAndClose(path))
}

func TestCostingSheetAttachmentStoresAndListsTechnicalDatasheets(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&CostingSheetAttachment{}))

	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "Techno-Commercial Offer 91-26.pdf")
	pdfBytes := []byte("%PDF-1.4\nsample datasheet\n%%EOF")
	require.NoError(t, os.WriteFile(pdfPath, pdfBytes, 0600))

	attached, err := app.AttachCostingSheetFile("costing-91-26", "91-26", "National Petroleum Co.", pdfPath, "Technical datasheet")
	require.NoError(t, err)
	require.NotNil(t, attached)
	require.Equal(t, "Techno-Commercial Offer 91-26.pdf", attached.FileName)
	require.Equal(t, "pdf", attached.FileExt)
	require.Equal(t, int64(len(pdfBytes)), attached.FileSize)
	require.NotEmpty(t, attached.FileHash)
	require.Equal(t, costingAttachmentStorageDatabase, attached.StorageMode)
	require.Empty(t, attached.LocalPath)
	require.Equal(t, "test-admin", attached.UploadedBy)

	list, err := app.ListCostingSheetAttachments("costing-91-26")
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, attached.ID, list[0].ID)
	require.Equal(t, "application/pdf", list[0].MimeType)
}

func TestCostingSheetAttachmentCreatesMissingTableOnAttach(t *testing.T) {
	app := setupTestApp(t)
	if app.db.Migrator().HasTable(&CostingSheetAttachment{}) {
		require.NoError(t, app.db.Migrator().DropTable(&CostingSheetAttachment{}))
	}
	require.False(t, app.db.Migrator().HasTable(&CostingSheetAttachment{}))

	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "datasheet.pdf")
	require.NoError(t, os.WriteFile(pdfPath, []byte("%PDF-1.4\nself healing table\n%%EOF"), 0600))

	attached, err := app.AttachCostingSheetFile("costing-self-heal", "91-26", "Rhine Instruments", pdfPath, "")
	require.NoError(t, err)
	require.NotNil(t, attached)
	require.True(t, app.db.Migrator().HasTable(&CostingSheetAttachment{}))
}

func TestCostingSheetAttachmentRejectsUnsupportedFiles(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&CostingSheetAttachment{}))

	dir := t.TempDir()
	exePath := filepath.Join(dir, "datasheet.exe")
	require.NoError(t, os.WriteFile(exePath, []byte("not a datasheet"), 0600))

	_, err := app.AttachCostingSheetFile("costing-91-26", "91-26", "National Petroleum Co.", exePath, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported attachment type")
}

func TestCostingSheetAttachmentStoresOversizedPDFLocally(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&CostingSheetAttachment{}))

	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "Large Datasheet.pdf")
	file, err := os.Create(pdfPath)
	require.NoError(t, err)
	_, err = file.WriteString("%PDF-1.4\n")
	require.NoError(t, err)
	require.NoError(t, file.Truncate(maxCostingAttachmentBytes+1024))
	require.NoError(t, file.Close())

	attached, err := app.AttachCostingSheetFile("costing-91-26", "91-26", "National Petroleum Co.", pdfPath, "Large local datasheet")
	require.NoError(t, err)
	require.NotNil(t, attached)
	require.Equal(t, costingAttachmentStorageLocalFile, attached.StorageMode)
	require.NotEmpty(t, attached.LocalPath)
	require.Contains(t, attached.LocalPath, "AsymmFlow Exports")
	require.Contains(t, attached.LocalPath, "Technical_Datasheets")
	require.FileExists(t, attached.LocalPath)
	require.Greater(t, attached.FileSize, int64(maxCostingAttachmentBytes))
	require.NotEmpty(t, attached.FileHash)

	var row CostingSheetAttachment
	require.NoError(t, app.db.First(&row, "id = ?", attached.ID).Error)
	require.Empty(t, row.ContentBase64)
}

func TestCostingSheetAttachmentRejectsOversizedImages(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&CostingSheetAttachment{}))

	dir := t.TempDir()
	imagePath := filepath.Join(dir, "huge.png")
	file, err := os.Create(imagePath)
	require.NoError(t, err)
	require.NoError(t, file.Truncate(maxCostingAttachmentBytes+1))
	require.NoError(t, file.Close())

	_, err = app.AttachCostingSheetFile("costing-91-26", "91-26", "National Petroleum Co.", imagePath, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "oversized PDFs are stored locally")
}

func TestCostingSheetAttachmentDeleteSoftRemovesFromList(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&CostingSheetAttachment{}))

	dir := t.TempDir()
	imagePath := filepath.Join(dir, "probe.png")
	require.NoError(t, os.WriteFile(imagePath, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, 0600))

	attached, err := app.AttachCostingSheetFile("costing-probe", "PROBE", "Probe Customer", imagePath, "")
	require.NoError(t, err)
	require.NotNil(t, attached)

	require.NoError(t, app.DeleteCostingSheetAttachment(attached.ID))
	list, err := app.ListCostingSheetAttachments("costing-probe")
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestCostingSheetAttachmentRequiresPermission(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&CostingSheetAttachment{}))
	app.currentUser = &User{
		Base:     Base{ID: "limited-user"},
		Username: "limited-sales",
		RoleName: "sales",
		Role: Role{
			Name:        "sales",
			DisplayName: "Sales",
			Permissions: `["customers:view"]`,
		},
	}

	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "datasheet.pdf")
	require.NoError(t, os.WriteFile(pdfPath, []byte("%PDF-1.4\ndenied\n%%EOF"), 0600))

	attached, err := app.AttachCostingSheetFile("costing-denied", "91-26", "National Petroleum Co.", pdfPath, "")
	require.Error(t, err)
	require.Nil(t, attached)
	require.Contains(t, err.Error(), "offers:create")
}

func TestCostingPDFBundleAppendsLocalDatasheetPDF(t *testing.T) {
	app := setupTestApp(t)
	dir := t.TempDir()
	basePath := filepath.Join(dir, "offer.pdf")
	datasheetPath := filepath.Join(dir, "datasheet.pdf")
	writeTinyPDFForAttachmentTest(t, basePath, "Offer")
	writeTinyPDFForAttachmentTest(t, datasheetPath, "Datasheet")

	baseInfo, err := os.Stat(basePath)
	require.NoError(t, err)
	basePages, err := pdfcpuapi.PageCountFile(basePath)
	require.NoError(t, err)

	err = app.appendCostingPDFDatasheets(basePath, []CostingSheetAttachmentSummary{{
		FileName:    "datasheet.pdf",
		FileExt:     "pdf",
		StorageMode: costingAttachmentStorageLocalFile,
		LocalPath:   datasheetPath,
	}})
	require.NoError(t, err)

	mergedInfo, err := os.Stat(basePath)
	require.NoError(t, err)
	require.Greater(t, mergedInfo.Size(), baseInfo.Size())

	mergedPages, err := pdfcpuapi.PageCountFile(basePath)
	require.NoError(t, err)
	require.Equal(t, basePages+1, mergedPages, "bundling should append the datasheet page after the offer")
}

func TestCostingPDFBundleFromDatabaseStoredDatasheet(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&CostingSheetAttachment{}))

	dir := t.TempDir()
	datasheetPath := filepath.Join(dir, "db-datasheet.pdf")
	writeTinyPDFForAttachmentTest(t, datasheetPath, "DB Datasheet")

	attached, err := app.AttachCostingSheetFile("costing-db", "91-26", "National Petroleum Co.", datasheetPath, "")
	require.NoError(t, err)
	require.Equal(t, costingAttachmentStorageDatabase, attached.StorageMode)

	basePath := filepath.Join(dir, "offer.pdf")
	writeTinyPDFForAttachmentTest(t, basePath, "Offer")
	basePages, err := pdfcpuapi.PageCountFile(basePath)
	require.NoError(t, err)

	summaries, err := app.ListCostingSheetAttachments("costing-db")
	require.NoError(t, err)
	require.NoError(t, app.appendCostingPDFDatasheets(basePath, summaries))

	mergedPages, err := pdfcpuapi.PageCountFile(basePath)
	require.NoError(t, err)
	require.Equal(t, basePages+1, mergedPages)
}

func TestCostingPDFExportBundlesLocalPDFDatasheet(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	app := setupTestApp(t)
	dir := t.TempDir()
	datasheetPath := filepath.Join(dir, "datasheet.pdf")
	writeTinyPDFForAttachmentTest(t, datasheetPath, "Datasheet")

	filePath, err := app.exportCostingToPDF(CostingExportData{
		Division:      "Acme Instrumentation",
		Date:          "2026-05-12",
		PreparedBy:    "Sam Rivera",
		CustomerName:  "Bundle Customer",
		ContactPerson: "Procurement",
		CostingId:     "91-26",
		QuoteType:     "Quotation",
		VatRate:       10,
		LineItems: []CostingExportLineItem{{
			SlNo:           1,
			Equipment:      "Instrument Junction Box",
			Model:          "JB-8150",
			Quantity:       1,
			SuggestedPrice: 100,
			TotalPrice:     100,
		}},
		Subtotal:   100,
		NetAmount:  100,
		VAT:        10,
		GrandTotal: 110,
		Attachments: []CostingSheetAttachmentSummary{{
			FileName:    "datasheet.pdf",
			FileExt:     "pdf",
			StorageMode: costingAttachmentStorageLocalFile,
			LocalPath:   datasheetPath,
			FileSize:    1024,
		}},
	}, "Quotation")
	require.NoError(t, err)
	require.FileExists(t, filePath)

	pageCount, err := pdfcpuapi.PageCountFile(filePath)
	require.NoError(t, err)
	require.Greater(t, pageCount, 1, "exported quotation should carry the bundled datasheet page")
}

func TestCostingPDFExportPreservesUnmergeableDatasheetSeparately(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	app := setupTestApp(t)
	dir := t.TempDir()
	datasheetPath := filepath.Join(dir, "vendor-protected-datasheet.pdf")
	require.NoError(t, os.WriteFile(datasheetPath, []byte("%PDF-1.7\nthis is intentionally malformed\n%%EOF"), 0600))

	filePath, err := app.exportCostingToPDF(CostingExportData{
		Division:      "Acme Instrumentation",
		Date:          "2026-05-14",
		PreparedBy:    "Sam Rivera",
		CustomerName:  "Fallback Customer",
		ContactPerson: "Procurement",
		CostingId:     "FALLBACK-01",
		QuoteType:     "Quotation",
		VatRate:       10,
		LineItems: []CostingExportLineItem{{
			SlNo:           1,
			Equipment:      "Flow meter",
			Model:          "FM-100",
			Quantity:       1,
			SuggestedPrice: 100,
			TotalPrice:     100,
		}},
		Subtotal:   100,
		NetAmount:  100,
		VAT:        10,
		GrandTotal: 110,
		Attachments: []CostingSheetAttachmentSummary{{
			FileName:    "vendor-protected-datasheet.pdf",
			FileExt:     "pdf",
			StorageMode: costingAttachmentStorageLocalFile,
			LocalPath:   datasheetPath,
			FileSize:    1024,
		}},
	}, "Quotation")
	require.NoError(t, err)
	require.FileExists(t, filePath)

	sidecarFolder := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + "_datasheets"
	require.DirExists(t, sidecarFolder)
	require.FileExists(t, filepath.Join(sidecarFolder, "vendor-protected-datasheet.pdf"))
	require.FileExists(t, filepath.Join(sidecarFolder, "README.txt"))
}

func TestOfferPDFBundlesDatasheetByAttachmentScope(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&CostingSheetAttachment{}))

	dir := t.TempDir()
	datasheetPath := filepath.Join(dir, "offer-datasheet.pdf")
	writeTinyPDFForAttachmentTest(t, datasheetPath, "Offer Datasheet")

	scope := "offer-scope-91-26"
	_, err := app.AttachCostingSheetFile(scope, "91-26", "Scoped Customer", datasheetPath, "")
	require.NoError(t, err)

	offer := &Offer{
		Base:              Base{ID: uuid.New().String()},
		OfferNumber:       "OFR-BUNDLE-1",
		CustomerName:      "Scoped Customer",
		Stage:             "Quoted",
		QuoteType:         "Quotation",
		VatRate:           10,
		Division:          "Acme Instrumentation",
		AttachmentScopeID: scope,
		Items: []OfferItem{{
			Base:       Base{ID: uuid.New().String()},
			Equipment:  "Instrument Junction Box",
			Model:      "JB-8150",
			Quantity:   1,
			UnitPrice:  100,
			TotalPrice: 100,
			LineNumber: 1,
		}},
	}
	require.NoError(t, app.db.Create(offer).Error)

	filePath, err := app.GenerateOfferPDF(offer.ID)
	require.NoError(t, err)
	require.FileExists(t, filePath)

	pageCount, err := pdfcpuapi.PageCountFile(filePath)
	require.NoError(t, err)
	require.Greater(t, pageCount, 1, "offer PDF should bundle the scoped datasheet page")
}
