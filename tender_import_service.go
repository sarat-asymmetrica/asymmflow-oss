package main

// Wave 8 Bucket D — tender-folder ingestion, ported from deployed PH. A tender
// folder tree ("<number> <title>" directories) previews into workflow rows and
// imports as RFQData records keyed "T-<number>", skipping ones that already
// exist as an RFQ or opportunity.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type TenderFolderPreview struct {
	FolderPath  string `json:"folder_path"`
	FolderName  string `json:"folder_name"`
	WorkflowKey string `json:"workflow_key"`
	Title       string `json:"title"`
	Existing    bool   `json:"existing"`
	Status      string `json:"status"`
}

var tenderFolderPattern = regexp.MustCompile(`^\s*(\d+)(?:\.\d+)?\s+(.+?)\s*$`)

func ParseTenderFolderName(folderName string) (TenderFolderPreview, bool) {
	matches := tenderFolderPattern.FindStringSubmatch(strings.TrimSpace(folderName))
	if len(matches) != 3 {
		return TenderFolderPreview{}, false
	}
	number, err := strconv.Atoi(matches[1])
	if err != nil || number <= 0 {
		return TenderFolderPreview{}, false
	}
	title := strings.TrimSpace(matches[2])
	if title == "" {
		return TenderFolderPreview{}, false
	}
	return TenderFolderPreview{
		FolderName:  strings.TrimSpace(folderName),
		WorkflowKey: fmt.Sprintf("T-%d", number),
		Title:       title,
		Status:      "RFQ Received / Tender",
	}, true
}

func (a *App) PreviewTenderFolders(rootPath string) ([]TenderFolderPreview, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	rootPath = strings.TrimSpace(rootPath)
	if rootPath == "" {
		return nil, fmt.Errorf("tender folder path is required")
	}

	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tender folder: %w", err)
	}

	previews := make([]TenderFolderPreview, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		preview, ok := ParseTenderFolderName(entry.Name())
		if !ok {
			continue
		}
		preview.FolderPath = filepath.Join(rootPath, entry.Name())
		var count int64
		_ = a.db.Model(&RFQData{}).Where("rfq_number = ? OR rfq_ref = ?", preview.WorkflowKey, preview.WorkflowKey).Count(&count).Error
		if count == 0 {
			_ = a.db.Model(&Opportunity{}).Where("folder_number = ?", preview.WorkflowKey).Count(&count).Error
		}
		preview.Existing = count > 0
		previews = append(previews, preview)
	}

	sort.Slice(previews, func(i, j int) bool {
		return previews[i].WorkflowKey < previews[j].WorkflowKey
	})
	return previews, nil
}

func (a *App) ImportTenderFolders(rootPath string) ([]TenderFolderPreview, error) {
	if err := a.requirePermission("offers:create"); err != nil {
		return nil, err
	}
	previews, err := a.PreviewTenderFolders(rootPath)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	imported := make([]TenderFolderPreview, 0)
	for _, preview := range previews {
		if preview.Existing {
			continue
		}
		rfq := RFQData{
			RFQNumber:     preview.WorkflowKey,
			RFQRef:        preview.WorkflowKey,
			Client:        "Tender",
			Project:       preview.Title,
			Value:         0,
			Notes:         "Imported from tender folder: " + preview.FolderPath,
			Status:        "Tender",
			Stage:         "RFQ Received",
			SourceDocPath: preview.FolderPath,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := a.db.Create(&rfq).Error; err != nil {
			return imported, fmt.Errorf("failed to import tender %s: %w", preview.WorkflowKey, err)
		}
		preview.Existing = true
		imported = append(imported, preview)
	}

	if len(imported) > 0 && a.ctx != nil {
		runtime.EventsEmit(a.ctx, "opportunities:updated", map[string]any{"source": "tender_import", "count": len(imported)})
	}
	return imported, nil
}
