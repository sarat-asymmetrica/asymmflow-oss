package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"github.com/jung-kurt/gofpdf/contrib/gofpdi"
	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"
	pdfcpumodel "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// appendCostingPDFDatasheets merges any PDF datasheets (in listed order) after
// the freshly written offer/costing PDF at outputPath, replacing it in place.
// pdfcpu is the primary merger; gofpdi is the legacy fallback; if both fail the
// datasheets are copied to a sidecar folder so nothing is silently lost.
func (a *App) appendCostingPDFDatasheets(outputPath string, attachments []CostingSheetAttachmentSummary) error {
	if strings.TrimSpace(outputPath) == "" || len(attachments) == 0 {
		return nil
	}

	sources, cleanup, err := a.resolveCostingPDFDatasheetSources(attachments)
	defer cleanup()
	if err != nil {
		return err
	}
	if len(sources) == 0 {
		return nil
	}

	allSources := append([]string{outputPath}, sources...)
	tmpPath := outputPath + ".bundle.tmp.pdf"
	_ = os.Remove(tmpPath)

	pdfcpuErr := mergeCostingPDFsWithPDFCPU(allSources, tmpPath)
	if pdfcpuErr == nil {
		if err := replacePDFWithBundle(outputPath, tmpPath); err != nil {
			return err
		}
		log.Printf("✅ Appended %d technical PDF datasheet(s) to %s", len(sources), outputPath)
		return nil
	} else {
		log.Printf("⚠️ pdfcpu PDF datasheet merge failed, falling back to legacy importer: %v", pdfcpuErr)
	}

	legacyErr := mergeCostingPDFsWithLegacyImporter(allSources, tmpPath)
	if legacyErr != nil {
		_ = os.Remove(tmpPath)
		folder, preserveErr := preserveUnmergedCostingPDFDatasheets(outputPath, sources, legacyErr)
		if preserveErr != nil {
			return fmt.Errorf("failed to append PDF datasheets with pdfcpu (%v) and legacy importer (%v); additionally failed to preserve separate datasheets: %w", pdfcpuErr, legacyErr, preserveErr)
		}
		log.Printf("⚠️ Generated offer PDF without embedded technical datasheets; copied unmerged datasheet(s) to %s. pdfcpu: %v; legacy: %v", folder, pdfcpuErr, legacyErr)
		return nil
	}
	if err := replacePDFWithBundle(outputPath, tmpPath); err != nil {
		return err
	}

	log.Printf("✅ Appended %d technical PDF datasheet(s) to %s", len(sources), outputPath)
	return nil
}

func mergeCostingPDFsWithPDFCPU(sources []string, tmpPath string) error {
	if len(sources) == 0 {
		return nil
	}
	_ = os.Remove(tmpPath)
	conf := pdfcpumodel.NewDefaultConfiguration()
	conf.ValidationMode = pdfcpumodel.ValidationRelaxed
	conf.CreateBookmarks = false
	conf.Offline = true
	if err := pdfcpuapi.MergeCreateFile(sources, tmpPath, false, conf); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to merge PDF datasheets with pdfcpu: %w", err)
	}
	return nil
}

func mergeCostingPDFsWithLegacyImporter(allSources []string, tmpPath string) error {
	_ = os.Remove(tmpPath)
	bundled := gofpdf.New("P", "pt", "A4", "")
	importer := gofpdi.NewImporter()
	for _, source := range allSources {
		if err := importPDFPagesIntoBundle(bundled, importer, source); err != nil {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("failed to append PDF datasheet %s: %w", filepath.Base(source), err)
		}
	}

	if err := bundled.OutputFileAndClose(tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to write bundled offer PDF: %w", err)
	}
	return nil
}

func replacePDFWithBundle(outputPath, tmpPath string) error {
	if err := os.Remove(outputPath); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to replace offer PDF with bundled version: %w", err)
	}
	if err := os.Rename(tmpPath, outputPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize bundled offer PDF: %w", err)
	}
	return nil
}

func preserveUnmergedCostingPDFDatasheets(outputPath string, sources []string, mergeErr error) (string, error) {
	if len(sources) == 0 {
		return "", nil
	}
	baseName := strings.TrimSuffix(filepath.Base(outputPath), filepath.Ext(outputPath))
	if strings.TrimSpace(baseName) == "" {
		baseName = "offer"
	}
	folder := filepath.Join(filepath.Dir(outputPath), sanitizeFileName(baseName)+"_datasheets")
	if err := os.MkdirAll(folder, 0755); err != nil {
		return "", err
	}

	for index, source := range sources {
		info, err := os.Stat(source)
		if err != nil || info.IsDir() {
			continue
		}
		targetName := sanitizeFileName(filepath.Base(source))
		if targetName == "" || targetName == "." {
			targetName = fmt.Sprintf("datasheet_%02d.pdf", index+1)
		}
		targetPath := filepath.Join(folder, targetName)
		if _, err := os.Stat(targetPath); err == nil {
			ext := filepath.Ext(targetName)
			stem := strings.TrimSuffix(targetName, ext)
			targetPath = filepath.Join(folder, fmt.Sprintf("%s_%02d%s", stem, index+1, ext))
		}
		if err := copyCostingPDFBundleFile(source, targetPath); err != nil {
			return "", err
		}
	}

	readme := fmt.Sprintf(
		"Technical datasheets were not embedded into the offer PDF because the PDF merger could not read one or more source files.\n\nOffer PDF: %s\nReason: %v\n\nPrint the PDFs in this folder after the offer PDF.\n",
		filepath.Base(outputPath),
		mergeErr,
	)
	if err := os.WriteFile(filepath.Join(folder, "README.txt"), []byte(readme), 0600); err != nil {
		return "", err
	}
	return folder, nil
}

func copyCostingPDFBundleFile(sourcePath, targetPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer target.Close()

	_, err = io.Copy(target, source)
	return err
}

func (a *App) resolveCostingPDFDatasheetSources(attachments []CostingSheetAttachmentSummary) ([]string, func(), error) {
	cleanupPaths := []string{}
	cleanup := func() {
		for _, path := range cleanupPaths {
			_ = os.Remove(path)
		}
	}

	sources := []string{}
	for _, attachment := range attachments {
		if strings.ToLower(strings.TrimPrefix(attachment.FileExt, ".")) != "pdf" {
			continue
		}

		storageMode := strings.TrimSpace(attachment.StorageMode)
		if storageMode == "" {
			storageMode = costingAttachmentStorageDatabase
		}

		if storageMode == costingAttachmentStorageLocalFile {
			localPath := strings.TrimSpace(attachment.LocalPath)
			if localPath == "" {
				return nil, cleanup, fmt.Errorf("local PDF attachment %s has no file path", attachment.FileName)
			}
			if _, err := os.Stat(localPath); err != nil {
				return nil, cleanup, fmt.Errorf("local PDF attachment %s is not available on this device: %w", attachment.FileName, err)
			}
			sources = append(sources, localPath)
			continue
		}

		if strings.TrimSpace(attachment.ID) == "" || a.db == nil {
			continue
		}
		var row CostingSheetAttachment
		if err := a.db.First(&row, "id = ?", attachment.ID).Error; err != nil {
			return nil, cleanup, fmt.Errorf("failed to load PDF attachment %s: %w", attachment.FileName, err)
		}
		content, err := base64.StdEncoding.DecodeString(row.ContentBase64)
		if err != nil {
			return nil, cleanup, fmt.Errorf("PDF attachment %s content is corrupted: %w", attachment.FileName, err)
		}
		if len(content) == 0 {
			continue
		}
		tmpPath := filepath.Join(os.TempDir(), "asymmflow-costing-pdf-"+sanitizeFileName(firstNonEmptyString(row.ID, row.FileName))+".pdf")
		if err := os.WriteFile(tmpPath, content, 0600); err != nil {
			return nil, cleanup, fmt.Errorf("failed to prepare PDF attachment %s: %w", attachment.FileName, err)
		}
		cleanupPaths = append(cleanupPaths, tmpPath)
		sources = append(sources, tmpPath)
	}

	return sources, cleanup, nil
}

func importPDFPagesIntoBundle(bundle *gofpdf.Fpdf, importer *gofpdi.Importer, sourcePath string) (err error) {
	// Open the file ourselves so we control the handle lifetime.  The gofpdi
	// library's SetSourceFile opens files internally and never closes them
	// (PdfReader has no Close method), which leaks handles on Windows and
	// prevents the caller from removing the source file.  Using the stream-
	// based import path lets us defer-close the handle reliably.
	f, openErr := os.Open(sourcePath)
	if openErr != nil {
		return fmt.Errorf("failed to open PDF source %s: %w", filepath.Base(sourcePath), openErr)
	}
	defer f.Close()

	var rs io.ReadSeeker = f

	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("PDF import failed: %v", recovered)
		}
	}()
	if importer == nil {
		importer = gofpdi.NewImporter()
	}
	templateID := importer.ImportPageFromStream(bundle, &rs, 1, "/MediaBox")
	pageSizes := importer.GetPageSizes()
	pageCount := len(pageSizes)
	if pageCount == 0 {
		return fmt.Errorf("PDF has no readable pages")
	}

	for page := 1; page <= pageCount; page++ {
		if page > 1 {
			templateID = importer.ImportPageFromStream(bundle, &rs, page, "/MediaBox")
		}
		mediaBox := pageSizes[page]["/MediaBox"]
		width := mediaBox["w"]
		height := mediaBox["h"]
		if width <= 0 || height <= 0 {
			width = 595
			height = 842
		}
		orientation := "P"
		if width > height {
			orientation = "L"
		}
		bundle.AddPageFormat(orientation, gofpdf.SizeType{Wd: width, Ht: height})
		importer.UseImportedTemplate(bundle, templateID, 0, 0, width, height)
		if err := bundle.Error(); err != nil {
			return err
		}
	}
	return bundle.Error()
}
