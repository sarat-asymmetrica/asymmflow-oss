// Package main - MSG Parser Service
// σ: ASYMM-MSG-PARSER | ρ: msg_parser.go | γ: v1 | κ: Extract RFQ data from Outlook .msg files
// Author: Agent Ramanujan (maintainer + AI pair)
// Date: February 5, 2026
//
// Purpose: Parse Outlook .msg files to extract RFQ information from emails
//   - Email metadata (subject, from, to, date, body)
//   - RFQ reference numbers, customer names, due dates
//   - Product requirements from body text
//   - Attachment listings
//
// Library: github.com/willthrom/outlook-msg-parser
//   - Handles MAPI properties for complete .msg parsing
//   - Supports various .msg versions and formats
//   - Extracts headers, body, attachments, recipients
//
// Philosophy: EXTRACT TRUTH FROM EMAIL. NO GUESSING. STRUCTURED DATA.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	msgparser "github.com/willthrom/outlook-msg-parser"
	"github.com/willthrom/outlook-msg-parser/models"
)

// ═══════════════════════════════════════════════════════════════════════════
// MSG PARSER SERVICE
// ═══════════════════════════════════════════════════════════════════════════

// MSGParserService handles parsing of Outlook .msg files
type MSGParserService struct {
	// Configuration
	debug bool
}

// NewMSGParserService creates a new MSG parser service
func NewMSGParserService() *MSGParserService {
	return &MSGParserService{
		debug: false, // Set to true for verbose logging
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// DATA STRUCTURES
// ═══════════════════════════════════════════════════════════════════════════

// ParsedRFQEmail represents parsed RFQ email data from .msg file
type ParsedRFQEmail struct {
	// File metadata
	FilePath    string    `json:"file_path"`     // Full path to .msg file
	FileName    string    `json:"file_name"`     // Just the filename
	FileSize    int64     `json:"file_size"`     // Size in bytes
	FileModTime time.Time `json:"file_mod_time"` // Last modified timestamp

	// Contextual data (from folder structure)
	OfferNumber   string `json:"offer_number"`   // Extracted from parent folder name
	FolderContext string `json:"folder_context"` // Full folder path context

	// Email metadata
	Subject      string    `json:"subject"`       // Email subject line
	From         string    `json:"from"`          // Sender email address
	FromName     string    `json:"from_name"`     // Sender display name
	To           []string  `json:"to"`            // Recipient email addresses
	ToNames      []string  `json:"to_names"`      // Recipient display names
	CC           []string  `json:"cc"`            // CC recipients
	DateSent     time.Time `json:"date_sent"`     // When email was sent
	DateReceived time.Time `json:"date_received"` // When email was received

	// Email body
	BodyText string `json:"body_text"` // Plain text body
	BodyHTML string `json:"body_html"` // HTML body (if available)

	// Extracted RFQ information
	RFQReference string     `json:"rfq_reference"` // RFQ number (e.g., "RFQ-2025-001")
	CustomerName string     `json:"customer_name"` // Customer/Company name
	DueDate      *time.Time `json:"due_date"`      // Due date if mentioned
	ProjectName  string     `json:"project_name"`  // Project name if mentioned

	// Product/item extraction
	ExtractedItems []string `json:"extracted_items"` // Product items/part numbers mentioned

	// Attachments
	Attachments []AttachmentInfo `json:"attachments"` // List of attachments

	// Parsing metadata
	ParsedAt     time.Time `json:"parsed_at"`     // When this parsing occurred
	ParseSuccess bool      `json:"parse_success"` // Whether parsing succeeded
	ParseError   string    `json:"parse_error"`   // Error message if parsing failed
}

// AttachmentInfo represents an email attachment
type AttachmentInfo struct {
	FileName string `json:"file_name"` // Attachment filename
}

// BatchParseResult represents results from batch parsing multiple .msg files
type BatchParseResult struct {
	BasePath    string           `json:"base_path"`    // Root path scanned
	TotalFiles  int              `json:"total_files"`  // Total .msg files found
	ParsedFiles int              `json:"parsed_files"` // Successfully parsed
	FailedFiles int              `json:"failed_files"` // Failed to parse
	Emails      []ParsedRFQEmail `json:"emails"`       // All parsed emails
	Duration    time.Duration    `json:"duration"`     // Time taken
	ParsedAt    time.Time        `json:"parsed_at"`    // When batch parse occurred
}

// ═══════════════════════════════════════════════════════════════════════════
// CORE PARSING FUNCTIONS
// ═══════════════════════════════════════════════════════════════════════════

// ParseRFQEmail parses a single .msg file and extracts RFQ data
func (s *MSGParserService) ParseRFQEmail(msgPath string) (*ParsedRFQEmail, error) {
	startTime := time.Now()

	// Initialize result
	result := &ParsedRFQEmail{
		FilePath:     msgPath,
		FileName:     filepath.Base(msgPath),
		ParsedAt:     startTime,
		ParseSuccess: false,
	}

	// Get file metadata
	fileInfo, err := os.Stat(msgPath)
	if err != nil {
		result.ParseError = fmt.Sprintf("Failed to stat file: %v", err)
		return result, err
	}
	result.FileSize = fileInfo.Size()
	result.FileModTime = fileInfo.ModTime()

	// Extract context from folder structure
	s.extractFolderContext(result, msgPath)

	// Open and parse .msg file
	if s.debug {
		log.Printf("📧 Parsing MSG file: %s", msgPath)
	}

	msg, err := msgparser.ParseMsgFile(msgPath)
	if err != nil {
		result.ParseError = fmt.Sprintf("Failed to parse MSG file: %v", err)
		return result, err
	}

	// Extract email metadata
	result.Subject = msg.Subject
	result.From = msg.FromEmail
	result.FromName = msg.FromName

	// Get recipients
	result.To = s.extractRecipients(msg.To)
	result.ToNames = s.extractRecipientNames(msg.ToDisplay)
	result.CC = s.extractRecipients(msg.CC)

	// Get dates
	result.DateSent = msg.ClientSubmitTime
	result.DateReceived = msg.Date

	// Get body content (prefer plain text, fallback to HTML or converted HTML)
	result.BodyText = msg.BodyPlainText
	if msg.BodyHTML != "" {
		result.BodyHTML = msg.BodyHTML
	} else if msg.ConvertedBodyHTML != "" {
		result.BodyHTML = msg.ConvertedBodyHTML
	}

	// Extract attachments
	result.Attachments = s.extractAttachments(msg)

	// Parse RFQ-specific information from subject and body
	s.extractRFQInformation(result)

	// Extract product items/part numbers
	s.extractProductItems(result)

	// Mark as successful
	result.ParseSuccess = true

	if s.debug {
		log.Printf("✅ Parsed MSG in %v: Subject='%s', From='%s'",
			time.Since(startTime), result.Subject, result.From)
	}

	return result, nil
}

// BatchParseRFQEmails scans a directory recursively for .msg files and parses them all
func (s *MSGParserService) BatchParseRFQEmails(basePath string) (*BatchParseResult, error) {
	startTime := time.Now()

	result := &BatchParseResult{
		BasePath: basePath,
		Emails:   []ParsedRFQEmail{},
		ParsedAt: startTime,
	}

	// Find all .msg files
	msgFiles, err := s.findMSGFiles(basePath)
	if err != nil {
		return result, fmt.Errorf("failed to scan directory: %w", err)
	}

	result.TotalFiles = len(msgFiles)

	if s.debug {
		log.Printf("📂 Found %d .msg files in %s", len(msgFiles), basePath)
	}

	// Parse each file
	for i, msgFile := range msgFiles {
		if s.debug {
			log.Printf("[%d/%d] Parsing: %s", i+1, len(msgFiles), msgFile)
		}

		parsed, err := s.ParseRFQEmail(msgFile)
		if err != nil {
			result.FailedFiles++
			if s.debug {
				log.Printf("❌ Failed to parse %s: %v", msgFile, err)
			}
		} else {
			result.ParsedFiles++
		}

		// Add to results even if parsing failed (to capture error info)
		result.Emails = append(result.Emails, *parsed)
	}

	result.Duration = time.Since(startTime)

	if s.debug {
		log.Printf("✅ Batch parse complete: %d/%d succeeded in %v",
			result.ParsedFiles, result.TotalFiles, result.Duration)
	}

	return result, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// EXTRACTION HELPERS
// ═══════════════════════════════════════════════════════════════════════════

// extractFolderContext extracts offer number and context from file path
func (s *MSGParserService) extractFolderContext(result *ParsedRFQEmail, msgPath string) {
	// Get directory path
	dirPath := filepath.Dir(msgPath)
	result.FolderContext = dirPath

	// Try to extract offer number from parent folder name
	// Pattern: "1-26 - Rhine Instruments - NPC - 8 inch MID" -> "1-26"
	parentDir := filepath.Base(dirPath)

	// Try pattern: "NUMBER-NUMBER - ..."
	offerPattern := regexp.MustCompile(`^(\d+-\d+)`)
	if matches := offerPattern.FindStringSubmatch(parentDir); len(matches) > 1 {
		result.OfferNumber = matches[1]
	}

	// Try to extract customer name from folder structure
	// Pattern: "1-26 - Rhine Instruments - NPC - ..."
	parts := strings.Split(parentDir, " - ")
	if len(parts) >= 3 {
		// Second part is usually supplier, third is customer
		if result.CustomerName == "" {
			result.CustomerName = strings.TrimSpace(parts[2])
		}
	}
}

// extractRecipients extracts email addresses from recipient string
func (s *MSGParserService) extractRecipients(recipients string) []string {
	if recipients == "" {
		return []string{}
	}

	// Split by semicolon or comma
	parts := regexp.MustCompile(`[;,]`).Split(recipients, -1)

	var emails []string
	emailPattern := regexp.MustCompile(`([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`)

	for _, part := range parts {
		// Extract email from format: "Name <email@domain.com>"
		if matches := emailPattern.FindStringSubmatch(part); len(matches) > 1 {
			emails = append(emails, matches[1])
		}
	}

	return emails
}

// extractRecipientNames extracts display names from recipient string
func (s *MSGParserService) extractRecipientNames(recipients string) []string {
	if recipients == "" {
		return []string{}
	}

	// Split by semicolon or comma
	parts := regexp.MustCompile(`[;,]`).Split(recipients, -1)

	var names []string
	// Pattern to extract name from: "Display Name <email@domain.com>"
	namePattern := regexp.MustCompile(`^([^<]+)<`)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if matches := namePattern.FindStringSubmatch(part); len(matches) > 1 {
			names = append(names, strings.TrimSpace(matches[1]))
		} else if part != "" {
			// If no angle brackets, use whole string
			names = append(names, part)
		}
	}

	return names
}

// extractAttachments extracts attachment information from MSG
func (s *MSGParserService) extractAttachments(msg *models.Message) []AttachmentInfo {
	var attachments []AttachmentInfo

	// Iterate through attachments
	for _, attachment := range msg.Attachments {
		info := AttachmentInfo{
			FileName: attachment.Name,
		}

		attachments = append(attachments, info)
	}

	return attachments
}

// extractRFQInformation extracts RFQ-specific data from subject and body
func (s *MSGParserService) extractRFQInformation(result *ParsedRFQEmail) {
	// Combine subject and body for pattern matching
	combinedText := result.Subject + "\n" + result.BodyText

	// Extract RFQ reference numbers (various patterns)
	rfqPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)RFQ[:\s#-]*([A-Z0-9-]+)`),            // RFQ-2025-001, RFQ: 12345
		regexp.MustCompile(`(?i)Request\s+for\s+Quote[:\s#-]*(\S+)`), // Request for Quote: RFQ001
		regexp.MustCompile(`(?i)Quotation\s+Request[:\s#-]*(\S+)`),   // Quotation Request #12345
		regexp.MustCompile(`(?i)Quote\s+Request[:\s#-]*(\S+)`),       // Quote Request: ABC123
	}

	for _, pattern := range rfqPatterns {
		if matches := pattern.FindStringSubmatch(combinedText); len(matches) > 1 {
			result.RFQReference = matches[1]
			break // Use first match
		}
	}

	// Extract customer/company names (look for common patterns)
	if result.CustomerName == "" {
		customerPatterns := []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:for|from|company)[:\s]+([A-Z][A-Za-z\s&.]+(?:Ltd|LLC|Inc|Corporation|WLL|Co))`),
			regexp.MustCompile(`(?i)Customer[:\s]+([A-Z][A-Za-z\s&.]+)`),
			regexp.MustCompile(`(?i)Client[:\s]+([A-Z][A-Za-z\s&.]+)`),
		}

		for _, pattern := range customerPatterns {
			if matches := pattern.FindStringSubmatch(combinedText); len(matches) > 1 {
				result.CustomerName = strings.TrimSpace(matches[1])
				break
			}
		}
	}

	// Extract project name
	projectPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)Project[:\s]+([^\n]{5,50})`),
		regexp.MustCompile(`(?i)for\s+the\s+([A-Z][^\n]{10,50})\s+project`),
	}

	for _, pattern := range projectPatterns {
		if matches := pattern.FindStringSubmatch(combinedText); len(matches) > 1 {
			result.ProjectName = strings.TrimSpace(matches[1])
			break
		}
	}

	// Extract due date
	dueDatePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:due|deadline|submit\s+by)[:\s]+(\d{1,2}[/-]\d{1,2}[/-]\d{2,4})`),
		regexp.MustCompile(`(?i)(?:due|deadline|submit\s+by)[:\s]+(\d{1,2}\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{2,4})`),
		regexp.MustCompile(`(?i)by\s+(\d{1,2}[/-]\d{1,2}[/-]\d{2,4})`),
	}

	for _, pattern := range dueDatePatterns {
		if matches := pattern.FindStringSubmatch(combinedText); len(matches) > 1 {
			// Try to parse the date
			dateStr := matches[1]
			if parsedDate := s.parseFlexibleDate(dateStr); parsedDate != nil {
				result.DueDate = parsedDate
				break
			}
		}
	}
}

// extractProductItems extracts product part numbers and item descriptions
func (s *MSGParserService) extractProductItems(result *ParsedRFQEmail) {
	var items []string

	// Look for part numbers (various formats)
	partNumberPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\b([A-Z0-9]{3,}-[A-Z0-9]{2,})\b`),   // ABC123-DE45
		regexp.MustCompile(`\bP/N[:\s]+([A-Z0-9-]+)`),           // P/N: ABC-123
		regexp.MustCompile(`\bPart\s+Number[:\s]+([A-Z0-9-]+)`), // Part Number: 12345
		regexp.MustCompile(`\bModel[:\s]+([A-Z0-9-]+)`),         // Model: XYZ-789
		regexp.MustCompile(`\b([A-Z]{2,}\d{3,}[A-Z0-9-]*)\b`),   // ABC12345, XY789-Z
	}

	combinedText := result.Subject + "\n" + result.BodyText

	// Extract unique part numbers
	seenItems := make(map[string]bool)

	for _, pattern := range partNumberPatterns {
		matches := pattern.FindAllStringSubmatch(combinedText, -1)
		for _, match := range matches {
			if len(match) > 1 {
				item := strings.TrimSpace(match[1])
				if !seenItems[item] && len(item) >= 4 {
					items = append(items, item)
					seenItems[item] = true
				}
			}
		}
	}

	result.ExtractedItems = items
}

// parseFlexibleDate attempts to parse various date formats
func (s *MSGParserService) parseFlexibleDate(dateStr string) *time.Time {
	formats := []string{
		"02/01/2006",      // DD/MM/YYYY
		"01/02/2006",      // MM/DD/YYYY
		"2006-01-02",      // YYYY-MM-DD
		"02-01-2006",      // DD-MM-YYYY
		"01-02-2006",      // MM-DD-YYYY
		"2 January 2006",  // D Month YYYY
		"2 Jan 2006",      // D Mon YYYY
		"January 2, 2006", // Month D, YYYY
		"Jan 2, 2006",     // Mon D, YYYY
	}

	for _, format := range formats {
		if parsed, err := time.Parse(format, dateStr); err == nil {
			return &parsed
		}
	}

	return nil
}

// findMSGFiles recursively finds all .msg files in a directory
func (s *MSGParserService) findMSGFiles(basePath string) ([]string, error) {
	var msgFiles []string

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if file has .msg extension (case-insensitive)
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".msg") {
			msgFiles = append(msgFiles, path)
		}

		return nil
	})

	return msgFiles, err
}

// ═══════════════════════════════════════════════════════════════════════════
// WAILS BINDINGS
// ═══════════════════════════════════════════════════════════════════════════

// ParseMSGFile parses a single .msg file (Wails binding)
func (a *App) ParseMSGFile(msgPath string) (*ParsedRFQEmail, error) {
	if a.msgParser == nil {
		a.msgParser = NewMSGParserService()
	}

	return a.msgParser.ParseRFQEmail(msgPath)
}

// BatchParseMSGFiles batch parses all .msg files in a directory (Wails binding)
func (a *App) BatchParseMSGFiles(basePath string) (*BatchParseResult, error) {
	if a.msgParser == nil {
		a.msgParser = NewMSGParserService()
	}

	return a.msgParser.BatchParseRFQEmails(basePath)
}

// ParseMSGFileToJSON parses .msg file and returns JSON string (for easy frontend consumption)
func (a *App) ParseMSGFileToJSON(msgPath string) (string, error) {
	parsed, err := a.ParseMSGFile(msgPath)
	if err != nil {
		return "", err
	}

	jsonData, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(jsonData), nil
}

// SaveParsedEmailAsRFQ converts a ParsedRFQEmail to an RFQ record in database
func (a *App) SaveParsedEmailAsRFQ(parsedEmail *ParsedRFQEmail) (uint, error) {
	if err := a.requirePermission("offers:create"); err != nil {
		return 0, err
	}
	// Create RFQ record from parsed email
	rfq := RFQData{
		RFQNumber:      parsedEmail.RFQReference,
		Client:         parsedEmail.CustomerName,
		Project:        parsedEmail.ProjectName,
		Notes:          parsedEmail.BodyText,
		Status:         "pending",
		Stage:          "RFQ Received",
		DocumentHash:   "", // TODO: Calculate hash of email content
		ProductDetails: "", // TODO: Convert ExtractedItems to JSON
		SourceDocPath:  parsedEmail.FilePath,
		CreatedAt:      parsedEmail.DateReceived,
	}

	// Set offer number if extracted from folder
	if parsedEmail.OfferNumber != "" {
		rfq.RFQNumber = parsedEmail.OfferNumber
	}

	// Save to database
	if err := a.db.Create(&rfq).Error; err != nil {
		return 0, fmt.Errorf("failed to save RFQ: %w", err)
	}

	// Create initial comment with email details
	comment := RFQComment{
		RFQID:     rfq.ID,
		Comment:   fmt.Sprintf("📧 Created from email: %s\nFrom: %s (%s)\nDate: %s", parsedEmail.Subject, parsedEmail.FromName, parsedEmail.From, parsedEmail.DateSent.Format("2006-01-02 15:04")),
		CreatedBy: "system",
		CreatedAt: time.Now(),
	}

	if err := a.db.Create(&comment).Error; err != nil {
		log.Printf("⚠️ Failed to create RFQ comment: %v", err)
	}

	return rfq.ID, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// UTILITY FUNCTIONS
// ═══════════════════════════════════════════════════════════════════════════

// GetMSGFileInfo returns basic info about a .msg file without full parsing
func (a *App) GetMSGFileInfo(msgPath string) (map[string]any, error) {
	msg, err := msgparser.ParseMsgFile(msgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MSG file: %w", err)
	}

	info := map[string]any{
		"subject":          msg.Subject,
		"from":             msg.FromEmail,
		"from_name":        msg.FromName,
		"date_sent":        msg.ClientSubmitTime,
		"attachment_count": len(msg.Attachments),
		"has_body":         msg.BodyPlainText != "",
		"has_html":         msg.BodyHTML != "" || msg.ConvertedBodyHTML != "",
	}

	return info, nil
}

// ListMSGFilesInDirectory lists all .msg files in a directory (no parsing)
func (a *App) ListMSGFilesInDirectory(dirPath string) ([]map[string]any, error) {
	if a.msgParser == nil {
		a.msgParser = NewMSGParserService()
	}

	msgFiles, err := a.msgParser.findMSGFiles(dirPath)
	if err != nil {
		return nil, err
	}

	var results []map[string]any
	for _, msgFile := range msgFiles {
		fileInfo, err := os.Stat(msgFile)
		if err != nil {
			continue
		}

		results = append(results, map[string]any{
			"path":     msgFile,
			"name":     filepath.Base(msgFile),
			"size":     fileInfo.Size(),
			"mod_time": fileInfo.ModTime(),
		})
	}

	return results, nil
}
