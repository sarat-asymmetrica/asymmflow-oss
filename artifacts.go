package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// ARTIFACT A - WORKSPACE INDEX
// ═══════════════════════════════════════════════════════════════════════════

// ArtifactA_FileNode represents a tree node in the workspace index
type ArtifactA_FileNode struct {
	Path       string                `json:"path"`
	Type       string                `json:"type"`                // "file" or "folder"
	Extension  string                `json:"extension,omitempty"` // e.g., ".pdf", ".xlsx"
	SizeBytes  int64                 `json:"size_bytes"`
	ModifiedAt time.Time             `json:"modified_at"`
	Hash       string                `json:"hash,omitempty"` // SHA256 for files only
	Parent     string                `json:"parent"`
	Children   []*ArtifactA_FileNode `json:"children,omitempty"`
}

// ArtifactA_WorkspaceIndex represents the complete workspace structure
type ArtifactA_WorkspaceIndex struct {
	RootPath     string              `json:"root_path"`
	ScanTime     time.Time           `json:"scan_time"`
	TotalFiles   int                 `json:"total_files"`
	TotalFolders int                 `json:"total_folders"`
	TotalBytes   int64               `json:"total_bytes"`
	Root         *ArtifactA_FileNode `json:"root"`
	Extensions   map[string]int      `json:"extensions"` // Count by extension
}

// GenerateArtifactA_WorkspaceIndex creates a workspace index from a folder or ZIP file
// Rules:
// - ZIP vs folder must generate IDENTICAL index (except paths)
// - No inference, No AI, Cheap, fast, deterministic
func GenerateArtifactA_WorkspaceIndex(sourcePath string, isZIP bool) (*ArtifactA_WorkspaceIndex, error) {
	index := &ArtifactA_WorkspaceIndex{
		RootPath:   sourcePath,
		ScanTime:   time.Now(),
		Extensions: make(map[string]int),
	}

	if isZIP {
		return artifactA_generateIndexFromZIP(sourcePath, index)
	}

	return artifactA_generateIndexFromFolder(sourcePath, index)
}

// artifactA_generateIndexFromFolder walks a filesystem and builds the index
func artifactA_generateIndexFromFolder(rootPath string, index *ArtifactA_WorkspaceIndex) (*ArtifactA_WorkspaceIndex, error) {
	rootInfo, err := os.Stat(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat root path: %w", err)
	}

	// Create root node
	root := &ArtifactA_FileNode{
		Path:       rootPath,
		Type:       "folder",
		ModifiedAt: rootInfo.ModTime(),
		Parent:     "",
	}

	index.Root = root

	// Walk the directory tree
	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip root itself
		if path == rootPath {
			return nil
		}

		// Create node for this file/folder
		node := &ArtifactA_FileNode{
			Path:       path,
			ModifiedAt: info.ModTime(),
			SizeBytes:  info.Size(),
			Parent:     filepath.Dir(path),
		}

		if info.IsDir() {
			node.Type = "folder"
			index.TotalFolders++
		} else {
			node.Type = "file"
			node.Extension = strings.ToLower(filepath.Ext(path))

			// Count extension
			if node.Extension != "" {
				index.Extensions[node.Extension]++
			}

			// Calculate hash for files
			hash, err := artifactA_calculateFileHash(path)
			if err == nil {
				node.Hash = hash
			}

			index.TotalFiles++
			index.TotalBytes += info.Size()
		}

		// Attach to tree (find parent and add as child)
		artifactA_attachNodeToTree(root, node)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return index, nil
}

// artifactA_generateIndexFromZIP extracts ZIP structure and builds the index
func artifactA_generateIndexFromZIP(zipPath string, index *ArtifactA_WorkspaceIndex) (*ArtifactA_WorkspaceIndex, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP: %w", err)
	}
	defer reader.Close()

	// Create root node
	root := &ArtifactA_FileNode{
		Path:       zipPath,
		Type:       "folder",
		ModifiedAt: time.Now(),
		Parent:     "",
	}

	index.Root = root

	// Track folders we've created
	folders := make(map[string]*ArtifactA_FileNode)
	folders[""] = root

	// Process each file in ZIP
	for _, file := range reader.File {
		// Create folder nodes for all parent directories
		dir := filepath.Dir(file.Name)
		if dir != "." && dir != "" {
			artifactA_ensureFolderPath(dir, folders, root, index)
		}

		sizeBytes, err := artifactA_zipEntrySizeBytes(file)
		if err != nil {
			return nil, err
		}

		// Create file/folder node
		node := &ArtifactA_FileNode{
			Path:       file.Name,
			ModifiedAt: file.Modified,
			SizeBytes:  sizeBytes,
			Parent:     dir,
		}

		if file.FileInfo().IsDir() {
			node.Type = "folder"
			folders[file.Name] = node
			index.TotalFolders++
		} else {
			node.Type = "file"
			node.Extension = strings.ToLower(filepath.Ext(file.Name))

			// Count extension
			if node.Extension != "" {
				index.Extensions[node.Extension]++
			}

			// Calculate hash from ZIP entry
			hash, err := artifactA_calculateZIPEntryHash(file)
			if err == nil {
				node.Hash = hash
			}

			index.TotalFiles++
			index.TotalBytes += sizeBytes
		}

		// Attach to parent
		parent := folders[dir]
		if parent != nil {
			parent.Children = append(parent.Children, node)
		}

		if node.Type == "folder" {
			folders[file.Name] = node
		}
	}

	return index, nil
}

// artifactA_ensureFolderPath creates all intermediate folder nodes for a path
func artifactA_zipEntrySizeBytes(file *zip.File) (int64, error) {
	if file.UncompressedSize64 > math.MaxInt64 {
		return 0, fmt.Errorf("zip entry %q exceeds supported size limit", file.Name)
	}
	return int64(file.UncompressedSize64), nil
}

func artifactA_ensureFolderPath(path string, folders map[string]*ArtifactA_FileNode, root *ArtifactA_FileNode, index *ArtifactA_WorkspaceIndex) {
	if _, exists := folders[path]; exists {
		return
	}

	parts := strings.Split(filepath.ToSlash(path), "/")
	currentPath := ""

	for _, part := range parts {
		if currentPath != "" {
			currentPath += "/"
		}
		currentPath += part

		if _, exists := folders[currentPath]; !exists {
			parentPath := filepath.Dir(currentPath)
			if parentPath == "." {
				parentPath = ""
			}

			node := &ArtifactA_FileNode{
				Path:       currentPath,
				Type:       "folder",
				ModifiedAt: time.Now(),
				Parent:     parentPath,
			}

			parent := folders[parentPath]
			if parent != nil {
				parent.Children = append(parent.Children, node)
			}

			folders[currentPath] = node
			index.TotalFolders++
		}
	}
}

// artifactA_attachNodeToTree finds the parent node and attaches the child
func artifactA_attachNodeToTree(root *ArtifactA_FileNode, node *ArtifactA_FileNode) {
	// Simple recursive search
	var attach func(*ArtifactA_FileNode) bool
	attach = func(parent *ArtifactA_FileNode) bool {
		if parent.Path == node.Parent {
			parent.Children = append(parent.Children, node)
			return true
		}
		for _, child := range parent.Children {
			if attach(child) {
				return true
			}
		}
		return false
	}

	attach(root)
}

// artifactA_calculateFileHash computes SHA256 hash of a file
func artifactA_calculateFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// artifactA_calculateZIPEntryHash computes SHA256 hash of a ZIP entry
func artifactA_calculateZIPEntryHash(file *zip.File) (string, error) {
	rc, err := file.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, rc); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// ═══════════════════════════════════════════════════════════════════════════
// ARTIFACT B - EVIDENCE EXTRACTS
// ═══════════════════════════════════════════════════════════════════════════

// NumericField represents an extracted numeric value from a document
type ArtifactB_NumericField struct {
	Label      string  `json:"label"`      // e.g., "Total Amount", "Invoice Number"
	Value      string  `json:"value"`      // Raw extracted value
	Confidence float64 `json:"confidence"` // 0.0 - 1.0
	Type       string  `json:"type"`       // "amount", "date", "number", "id"
}

// ArtifactB_QualityMetrics represents the five-dimension quality assessment
type ArtifactB_QualityMetrics struct {
	Clarity            float64 `json:"clarity"`             // Text clarity (0.0 - 1.0)
	Layout             float64 `json:"layout"`              // Layout preservation (0.0 - 1.0)
	LanguageConfidence float64 `json:"language_confidence"` // Language detection confidence
	ScanNoise          float64 `json:"scan_noise"`          // Scan artifacts/noise level
	TableIntegrity     float64 `json:"table_integrity"`     // Table structure preservation
}

// ArtifactB_ProvenanceInfo tracks how the document was processed
type ArtifactB_ProvenanceInfo struct {
	Engine  string `json:"engine"`  // "ACE", "Tesseract", etc.
	Backend string `json:"backend"` // "GPU", "CPU", "AIMLAPI"
	TimeMS  int64  `json:"time_ms"` // Processing time in milliseconds
}

// EvidenceExtract represents OCR output for a single document
// Rules:
// - Never overwrite
// - Never "improve"
// - Never collapse versions
// - Low quality is a FEATURE, not a bug
type ArtifactB_EvidenceExtract struct {
	SourcePath    string                   `json:"source_path"`
	Pages         int                      `json:"pages"`
	Languages     []string                 `json:"languages"`
	LayoutTokens  bool                     `json:"layout_tokens"`
	ExtractedText string                   `json:"extracted_text"`
	NumericFields []ArtifactB_NumericField `json:"numeric_fields"`
	Quality       ArtifactB_QualityMetrics `json:"quality"`
	Provenance    ArtifactB_ProvenanceInfo `json:"provenance"`
}

// GenerateArtifactB_EvidenceExtract transforms ACE OCR response into Evidence format
// This is called AFTER ACE OCR has processed a document
func GenerateArtifactB_EvidenceExtract(sourcePath string, rawText string, pages int, languages []string, processingTimeMS int64) *ArtifactB_EvidenceExtract {
	// Detect backend based on processing time heuristic
	// GPU processing typically < 500ms per page, CPU > 2000ms per page
	backend := "CPU"
	if pages > 0 {
		msPerPage := processingTimeMS / int64(pages)
		if msPerPage < 500 {
			backend = "GPU"
		}
	}

	extract := &ArtifactB_EvidenceExtract{
		SourcePath:    sourcePath,
		Pages:         pages,
		Languages:     languages,
		LayoutTokens:  true, // ACE preserves layout
		ExtractedText: rawText,
		Provenance: ArtifactB_ProvenanceInfo{
			Engine:  "ACE",
			Backend: backend,
			TimeMS:  processingTimeMS,
		},
	}

	// Extract numeric fields from text
	extract.NumericFields = artifactB_extractNumericFields(rawText)

	// Calculate quality metrics (heuristic-based)
	extract.Quality = artifactB_calculateQualityMetrics(rawText, pages)

	return extract
}

// artifactB_extractNumericFields uses regex to find amounts, dates, and IDs
func artifactB_extractNumericFields(text string) []ArtifactB_NumericField {
	fields := []ArtifactB_NumericField{}

	// Pattern for amounts (e.g., "Total: 1,234.56 BHD")
	amountPattern := regexp.MustCompile(`(?i)(total|amount|price|cost|value)[\s:]*([0-9,]+\.?[0-9]*)\s*(BHD|USD|EUR|SAR)?`)
	amountMatches := amountPattern.FindAllStringSubmatch(text, -1)
	for _, match := range amountMatches {
		if len(match) >= 3 {
			fields = append(fields, ArtifactB_NumericField{
				Label:      strings.TrimSpace(match[1]),
				Value:      strings.TrimSpace(match[2]),
				Confidence: 0.8, // Heuristic confidence
				Type:       "amount",
			})
		}
	}

	// Pattern for dates (e.g., "Date: 15/12/2024")
	datePattern := regexp.MustCompile(`(?i)(date|dated)[\s:]*([0-9]{1,2}[/-][0-9]{1,2}[/-][0-9]{2,4})`)
	dateMatches := datePattern.FindAllStringSubmatch(text, -1)
	for _, match := range dateMatches {
		if len(match) >= 3 {
			fields = append(fields, ArtifactB_NumericField{
				Label:      strings.TrimSpace(match[1]),
				Value:      strings.TrimSpace(match[2]),
				Confidence: 0.75,
				Type:       "date",
			})
		}
	}

	// Pattern for invoice numbers (e.g., "Invoice #12345")
	invoicePattern := regexp.MustCompile(`(?i)(invoice|inv|receipt|ref)[\s#:]*([A-Z0-9-]+)`)
	invoiceMatches := invoicePattern.FindAllStringSubmatch(text, -1)
	for _, match := range invoiceMatches {
		if len(match) >= 3 {
			fields = append(fields, ArtifactB_NumericField{
				Label:      strings.TrimSpace(match[1]),
				Value:      strings.TrimSpace(match[2]),
				Confidence: 0.7,
				Type:       "id",
			})
		}
	}

	return fields
}

// artifactB_calculateQualityMetrics estimates quality from text characteristics
func artifactB_calculateQualityMetrics(text string, pages int) ArtifactB_QualityMetrics {
	metrics := ArtifactB_QualityMetrics{}

	// Clarity: ratio of alphanumeric chars to total chars
	alphanumeric := 0
	total := len(text)
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ' {
			alphanumeric++
		}
	}
	if total > 0 {
		metrics.Clarity = float64(alphanumeric) / float64(total)
	}

	// Layout: presence of whitespace structure (heuristic)
	lines := strings.Split(text, "\n")
	nonEmptyLines := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines++
		}
	}
	if len(lines) > 0 {
		metrics.Layout = float64(nonEmptyLines) / float64(len(lines))
	}

	// Language confidence: assume high if text is readable
	metrics.LanguageConfidence = metrics.Clarity * 0.95

	// Scan noise: inverse of clarity (more garbage = more noise)
	metrics.ScanNoise = 1.0 - metrics.Clarity

	// Table integrity: detect table-like structures (tabs, aligned columns)
	tabCount := strings.Count(text, "\t")
	columnPattern := regexp.MustCompile(`\s{3,}`) // 3+ spaces
	columnCount := len(columnPattern.FindAllString(text, -1))
	tableScore := float64(tabCount+columnCount) / float64(len(lines)+1)
	if tableScore > 1.0 {
		tableScore = 1.0
	}
	metrics.TableIntegrity = tableScore

	return metrics
}

// ═══════════════════════════════════════════════════════════════════════════
// ARTIFACT C - ARCHAEOLOGY REPORT (Human-Readable)
// ═══════════════════════════════════════════════════════════════════════════

// ArtifactC_OverviewSection summarizes the workspace at a high level
type ArtifactC_OverviewSection struct {
	TotalFiles      int       `json:"total_files"`
	TotalFolders    int       `json:"total_folders"`
	TotalSizeBytes  int64     `json:"total_size_bytes"`
	TotalSizeMB     float64   `json:"total_size_mb"`
	OldestFile      time.Time `json:"oldest_file"`
	NewestFile      time.Time `json:"newest_file"`
	FormatsDetected []string  `json:"formats_detected"`
}

// ArtifactC_ClusterInfo describes a group of related documents (weak inference)
type ArtifactC_ClusterInfo struct {
	ClusterType string   `json:"cluster_type"` // "folder", "name_pattern", "revision"
	Description string   `json:"description"`
	FilePaths   []string `json:"file_paths"`
	Confidence  float64  `json:"confidence"` // 0.0 - 1.0, always < 0.8 for weak inference
}

// ArtifactC_QualitySummary categorizes documents by quality level
type ArtifactC_QualitySummary struct {
	HighConfidence   int `json:"high_confidence"`   // clarity >= 0.8
	MediumConfidence int `json:"medium_confidence"` // 0.5 <= clarity < 0.8
	LowConfidence    int `json:"low_confidence"`    // 0.2 <= clarity < 0.5
	Unreadable       int `json:"unreadable"`        // clarity < 0.2
}

// ArtifactC_LanguageFormatSection describes language and format distribution
type ArtifactC_LanguageFormatSection struct {
	English      int `json:"english"`
	Arabic       int `json:"arabic"`
	Mixed        int `json:"mixed"`
	ScannedPDFs  int `json:"scanned_pdfs"`
	NativePDFs   int `json:"native_pdfs"`
	ExcelFiles   int `json:"excel_files"`
	WordFiles    int `json:"word_files"`
	OtherFormats int `json:"other_formats"`
}

// ArtifactC_UncertaintyItem describes an explicit uncertainty or conflict
type ArtifactC_UncertaintyItem struct {
	Type         string   `json:"type"` // "conflicting_totals", "duplicates", "clarification_needed"
	Description  string   `json:"description"`
	AffectedDocs []string `json:"affected_docs"`
}

// Type aliases for backward compatibility
type UncertaintyItem = ArtifactC_UncertaintyItem
type ClusterInfo = ArtifactC_ClusterInfo
type NumericField = ArtifactB_NumericField
type ArchaeologyReport = ArtifactC_ArchaeologyReport

// ArtifactC_ArchaeologyReport is the final human-readable report
// Tone rule: Describe. Do not judge. Do not fix.
type ArtifactC_ArchaeologyReport struct {
	GeneratedAt       time.Time                       `json:"generated_at"`
	WorkspaceOverview ArtifactC_OverviewSection       `json:"workspace_overview"`
	DetectedClusters  []ClusterInfo                   `json:"detected_clusters"`
	QualitySummary    ArtifactC_QualitySummary        `json:"quality_summary"`
	LanguagesFormats  ArtifactC_LanguageFormatSection `json:"languages_formats"`
	Uncertainties     []UncertaintyItem               `json:"uncertainties"`
	MarkdownContent   string                          `json:"markdown_content"`
}

// GenerateArchaeologyReport creates a human-readable report from index and extracts
func GenerateArtifactC_ArchaeologyReport(index *ArtifactA_WorkspaceIndex, extracts []*ArtifactB_EvidenceExtract) *ArtifactC_ArchaeologyReport {
	report := &ArchaeologyReport{
		GeneratedAt: time.Now(),
	}

	// 1. Workspace Overview
	report.WorkspaceOverview = artifactC_generateOverview(index)

	// 2. Detected Clusters (weak inference)
	report.DetectedClusters = artifactC_detectClusters(index, extracts)

	// 3. Quality Summary
	report.QualitySummary = artifactC_summarizeQuality(extracts)

	// 4. Languages & Formats
	report.LanguagesFormats = artifactC_analyzeLanguagesFormats(index, extracts)

	// 5. Uncertainties
	report.Uncertainties = artifactC_detectUncertainties(extracts)

	// 6. Render to Markdown
	report.MarkdownContent = artifactC_renderMarkdown(report)

	return report
}

// artifactC_generateOverview creates the workspace overview section
func artifactC_generateOverview(index *ArtifactA_WorkspaceIndex) ArtifactC_OverviewSection {
	overview := ArtifactC_OverviewSection{
		TotalFiles:     index.TotalFiles,
		TotalFolders:   index.TotalFolders,
		TotalSizeBytes: index.TotalBytes,
		TotalSizeMB:    float64(index.TotalBytes) / (1024 * 1024),
	}

	// Extract formats from extensions
	formats := make([]string, 0, len(index.Extensions))
	for ext := range index.Extensions {
		formats = append(formats, ext)
	}
	sort.Strings(formats)
	overview.FormatsDetected = formats

	// Find oldest and newest files (traverse tree)
	var oldest, newest time.Time
	var traverse func(*ArtifactA_FileNode)
	traverse = func(node *ArtifactA_FileNode) {
		if node.Type == "file" {
			if oldest.IsZero() || node.ModifiedAt.Before(oldest) {
				oldest = node.ModifiedAt
			}
			if newest.IsZero() || node.ModifiedAt.After(newest) {
				newest = node.ModifiedAt
			}
		}
		for _, child := range node.Children {
			traverse(child)
		}
	}

	if index.Root != nil {
		traverse(index.Root)
	}

	overview.OldestFile = oldest
	overview.NewestFile = newest

	return overview
}

// artifactC_detectClusters finds related documents using weak heuristics
func artifactC_detectClusters(index *ArtifactA_WorkspaceIndex, extracts []*ArtifactB_EvidenceExtract) []ArtifactC_ClusterInfo {
	clusters := []ArtifactC_ClusterInfo{}

	// Cluster 1: Folders with similar names
	folderGroups := make(map[string][]string)
	var collectFolders func(*ArtifactA_FileNode)
	collectFolders = func(node *ArtifactA_FileNode) {
		if node.Type == "folder" {
			baseName := strings.ToLower(filepath.Base(node.Path))
			folderGroups[baseName] = append(folderGroups[baseName], node.Path)
		}
		for _, child := range node.Children {
			collectFolders(child)
		}
	}

	if index.Root != nil {
		collectFolders(index.Root)
	}

	for baseName, paths := range folderGroups {
		if len(paths) > 1 {
			clusters = append(clusters, ArtifactC_ClusterInfo{
				ClusterType: "folder",
				Description: fmt.Sprintf("Folders with similar names: '%s'", baseName),
				FilePaths:   paths,
				Confidence:  0.6, // Weak inference
			})
		}
	}

	// Cluster 2: Files with shared numeric patterns (e.g., invoice numbers)
	numberPattern := regexp.MustCompile(`[0-9]{3,}`)
	numberGroups := make(map[string][]string)

	for _, extract := range extracts {
		numbers := numberPattern.FindAllString(extract.SourcePath, -1)
		for _, num := range numbers {
			numberGroups[num] = append(numberGroups[num], extract.SourcePath)
		}
	}

	for num, paths := range numberGroups {
		if len(paths) > 1 {
			clusters = append(clusters, ArtifactC_ClusterInfo{
				ClusterType: "name_pattern",
				Description: fmt.Sprintf("Documents sharing number pattern: '%s'", num),
				FilePaths:   paths,
				Confidence:  0.5, // Very weak inference
			})
		}
	}

	// Cluster 3: Potential revisions (same name, different folders or dates)
	// (Simplified: just look for "v1", "v2", "rev", etc. in filenames)
	revisionPattern := regexp.MustCompile(`(?i)(v[0-9]|rev[0-9]|draft)`)
	revisionGroups := make(map[string][]string)

	for _, extract := range extracts {
		if revisionPattern.MatchString(extract.SourcePath) {
			baseName := filepath.Base(extract.SourcePath)
			cleanName := revisionPattern.ReplaceAllString(baseName, "")
			revisionGroups[cleanName] = append(revisionGroups[cleanName], extract.SourcePath)
		}
	}

	for baseName, paths := range revisionGroups {
		if len(paths) > 1 {
			clusters = append(clusters, ArtifactC_ClusterInfo{
				ClusterType: "revision",
				Description: fmt.Sprintf("Possible revisions of: '%s'", baseName),
				FilePaths:   paths,
				Confidence:  0.4, // Very weak inference
			})
		}
	}

	return clusters
}

// artifactC_summarizeQuality categorizes documents by quality level
func artifactC_summarizeQuality(extracts []*ArtifactB_EvidenceExtract) ArtifactC_QualitySummary {
	summary := ArtifactC_QualitySummary{}

	for _, extract := range extracts {
		clarity := extract.Quality.Clarity

		if clarity >= 0.8 {
			summary.HighConfidence++
		} else if clarity >= 0.5 {
			summary.MediumConfidence++
		} else if clarity >= 0.2 {
			summary.LowConfidence++
		} else {
			summary.Unreadable++
		}
	}

	return summary
}

// artifactC_analyzeLanguagesFormats analyzes language and format distribution
func artifactC_analyzeLanguagesFormats(index *ArtifactA_WorkspaceIndex, extracts []*ArtifactB_EvidenceExtract) ArtifactC_LanguageFormatSection {
	section := ArtifactC_LanguageFormatSection{}

	// Analyze languages from extracts
	for _, extract := range extracts {
		hasEnglish := false
		hasArabic := false

		for _, lang := range extract.Languages {
			if strings.Contains(strings.ToLower(lang), "en") {
				hasEnglish = true
			}
			if strings.Contains(strings.ToLower(lang), "ar") {
				hasArabic = true
			}
		}

		if hasEnglish && hasArabic {
			section.Mixed++
		} else if hasEnglish {
			section.English++
		} else if hasArabic {
			section.Arabic++
		}
	}

	// Analyze formats from index
	for ext, count := range index.Extensions {
		switch ext {
		case ".pdf":
			// Heuristic: assume scanned if we have evidence extract with low clarity
			scanned := 0
			native := 0
			for _, extract := range extracts {
				if strings.HasSuffix(strings.ToLower(extract.SourcePath), ".pdf") {
					if extract.Quality.Clarity < 0.7 {
						scanned++
					} else {
						native++
					}
				}
			}
			section.ScannedPDFs = scanned
			section.NativePDFs = native
		case ".xlsx", ".xls":
			section.ExcelFiles += count
		case ".docx", ".doc":
			section.WordFiles += count
		default:
			section.OtherFormats += count
		}
	}

	return section
}

// artifactC_detectUncertainties identifies conflicts and duplicates
func artifactC_detectUncertainties(extracts []*ArtifactB_EvidenceExtract) []ArtifactC_UncertaintyItem {
	uncertainties := []ArtifactC_UncertaintyItem{}

	// Detect duplicates by hash (if available in future)
	// For now, detect potential duplicates by similar amounts in numeric fields
	amountMap := make(map[string][]string)

	for _, extract := range extracts {
		for _, field := range extract.NumericFields {
			if field.Type == "amount" {
				amountMap[field.Value] = append(amountMap[field.Value], extract.SourcePath)
			}
		}
	}

	for amount, paths := range amountMap {
		if len(paths) > 1 {
			uncertainties = append(uncertainties, ArtifactC_UncertaintyItem{
				Type:         "duplicates",
				Description:  fmt.Sprintf("Multiple documents with same amount: %s", amount),
				AffectedDocs: paths,
			})
		}
	}

	// Detect low-confidence extracts that need manual review
	lowConfidenceDocs := []string{}
	for _, extract := range extracts {
		if extract.Quality.Clarity < 0.3 {
			lowConfidenceDocs = append(lowConfidenceDocs, extract.SourcePath)
		}
	}

	if len(lowConfidenceDocs) > 0 {
		uncertainties = append(uncertainties, ArtifactC_UncertaintyItem{
			Type:         "clarification_needed",
			Description:  "Documents with very low quality - manual review recommended",
			AffectedDocs: lowConfidenceDocs,
		})
	}

	return uncertainties
}

// artifactC_renderMarkdown generates the final Markdown report
func artifactC_renderMarkdown(report *ArtifactC_ArchaeologyReport) string {
	var md strings.Builder

	md.WriteString("# Workspace Archaeology Report\n\n")
	md.WriteString(fmt.Sprintf("**Generated:** %s\n\n", report.GeneratedAt.Format("2006-01-02 15:04:05")))
	md.WriteString("---\n\n")

	// Section 1: Workspace Overview
	md.WriteString("## 1. Workspace Overview\n\n")
	ov := report.WorkspaceOverview
	md.WriteString(fmt.Sprintf("- **Total Files:** %d\n", ov.TotalFiles))
	md.WriteString(fmt.Sprintf("- **Total Folders:** %d\n", ov.TotalFolders))
	md.WriteString(fmt.Sprintf("- **Total Size:** %.2f MB (%d bytes)\n", ov.TotalSizeMB, ov.TotalSizeBytes))
	if !ov.OldestFile.IsZero() && !ov.NewestFile.IsZero() {
		md.WriteString(fmt.Sprintf("- **Date Range:** %s to %s\n",
			ov.OldestFile.Format("2006-01-02"),
			ov.NewestFile.Format("2006-01-02")))
	}
	md.WriteString(fmt.Sprintf("- **Formats Detected:** %s\n\n", strings.Join(ov.FormatsDetected, ", ")))

	// Section 2: Detected Clusters
	md.WriteString("## 2. Detected Clusters\n\n")
	if len(report.DetectedClusters) == 0 {
		md.WriteString("*No significant clusters detected.*\n\n")
	} else {
		for i, cluster := range report.DetectedClusters {
			md.WriteString(fmt.Sprintf("### Cluster %d: %s\n\n", i+1, cluster.Description))
			md.WriteString(fmt.Sprintf("**Type:** %s | **Confidence:** %.0f%%\n\n", cluster.ClusterType, cluster.Confidence*100))
			md.WriteString("**Files:**\n\n")
			for _, path := range cluster.FilePaths {
				md.WriteString(fmt.Sprintf("- `%s`\n", path))
			}
			md.WriteString("\n")
		}
	}

	// Section 3: Document Quality Summary
	md.WriteString("## 3. Document Quality Summary\n\n")
	qs := report.QualitySummary
	md.WriteString(fmt.Sprintf("- **High Confidence:** %d documents (clarity ≥ 80%%)\n", qs.HighConfidence))
	md.WriteString(fmt.Sprintf("- **Medium Confidence:** %d documents (clarity 50-80%%)\n", qs.MediumConfidence))
	md.WriteString(fmt.Sprintf("- **Low Confidence:** %d documents (clarity 20-50%%)\n", qs.LowConfidence))
	md.WriteString(fmt.Sprintf("- **Unreadable:** %d documents (clarity < 20%%)\n\n", qs.Unreadable))

	// Section 4: Languages & Formats
	md.WriteString("## 4. Languages & Formats\n\n")
	lf := report.LanguagesFormats
	md.WriteString("### Languages\n\n")
	md.WriteString(fmt.Sprintf("- **English:** %d documents\n", lf.English))
	md.WriteString(fmt.Sprintf("- **Arabic:** %d documents\n", lf.Arabic))
	md.WriteString(fmt.Sprintf("- **Mixed:** %d documents\n\n", lf.Mixed))
	md.WriteString("### Formats\n\n")
	md.WriteString(fmt.Sprintf("- **Scanned PDFs:** %d\n", lf.ScannedPDFs))
	md.WriteString(fmt.Sprintf("- **Native PDFs:** %d\n", lf.NativePDFs))
	md.WriteString(fmt.Sprintf("- **Excel Files:** %d\n", lf.ExcelFiles))
	md.WriteString(fmt.Sprintf("- **Word Documents:** %d\n", lf.WordFiles))
	md.WriteString(fmt.Sprintf("- **Other Formats:** %d\n\n", lf.OtherFormats))

	// Section 5: Uncertainties
	md.WriteString("## 5. Explicit Uncertainties\n\n")
	if len(report.Uncertainties) == 0 {
		md.WriteString("*No significant uncertainties detected. All documents appear consistent.*\n\n")
	} else {
		for i, unc := range report.Uncertainties {
			md.WriteString(fmt.Sprintf("### Uncertainty %d: %s\n\n", i+1, unc.Type))
			md.WriteString(fmt.Sprintf("**Description:** %s\n\n", unc.Description))
			if len(unc.AffectedDocs) > 0 {
				md.WriteString("**Affected Documents:**\n\n")
				for _, doc := range unc.AffectedDocs {
					md.WriteString(fmt.Sprintf("- `%s`\n", doc))
				}
				md.WriteString("\n")
			}
		}
	}

	md.WriteString("---\n\n")
	md.WriteString("*This report describes the workspace as observed. No judgment. No fixes applied.*\n")

	return md.String()
}

// ═══════════════════════════════════════════════════════════════════════════
// UTILITY FUNCTIONS
// ═══════════════════════════════════════════════════════════════════════════

// SaveWorkspaceIndex saves the index to a JSON file
func SaveArtifactA_WorkspaceIndex(index *ArtifactA_WorkspaceIndex, outputPath string) error {
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	return os.WriteFile(outputPath, data, 0644)
}

// SaveEvidenceExtract appends an evidence extract to a JSON Lines file
// Never overwrites, always appends (append-only evidence store)
func SaveArtifactB_EvidenceExtract(extract *ArtifactB_EvidenceExtract, outputPath string) error {
	data, err := json.Marshal(extract)
	if err != nil {
		return fmt.Errorf("failed to marshal extract: %w", err)
	}

	f, err := os.OpenFile(outputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open evidence file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write extract: %w", err)
	}

	if _, err := f.WriteString("\n"); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

// SaveArchaeologyReport saves the report as Markdown
func SaveArtifactC_ArchaeologyReport(report *ArtifactC_ArchaeologyReport, outputPath string) error {
	return os.WriteFile(outputPath, []byte(report.MarkdownContent), 0644)
}
