package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ReportMetadata represents metadata for a report stored in Supabase
type ReportMetadata struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	MimeType  string    `json:"mime_type"`
}

// validateSupabaseURL validates that the URL points to a legitimate Supabase domain
func validateSupabaseURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	// Must be HTTPS
	if parsed.Scheme != "https" {
		return fmt.Errorf("supabase URL must use HTTPS")
	}
	// Must end with supabase.co or supabase.in (official domains)
	host := strings.ToLower(parsed.Host)
	if !strings.HasSuffix(host, ".supabase.co") && !strings.HasSuffix(host, ".supabase.in") {
		return fmt.Errorf("URL must point to a Supabase domain (*.supabase.co)")
	}
	return nil
}

// supabaseListResponse represents the response from Supabase list API
type supabaseListResponse struct {
	Name         string    `json:"name"`
	ID           string    `json:"id"`
	UpdatedAt    time.Time `json:"updated_at"`
	CreatedAt    time.Time `json:"created_at"`
	LastAccessAt time.Time `json:"last_accessed_at"`
	Metadata     struct {
		ETag           string `json:"eTag"`
		Size           int64  `json:"size"`
		Mimetype       string `json:"mimetype"`
		CacheControl   string `json:"cacheControl"`
		LastModified   string `json:"lastModified"`
		ContentLength  int64  `json:"contentLength"`
		HTTPStatusCode int    `json:"httpStatusCode"`
	} `json:"metadata"`
}

// UploadReportToStorage uploads a local report file to Supabase Storage
// Returns the remote URL on success
func (a *App) UploadReportToStorage(localPath string) (string, error) {
	if err := a.requirePermission("reports:view"); err != nil {
		return "", err
	}

	// Check if Supabase is configured
	if !a.config.Supabase.Enabled {
		return "", fmt.Errorf("supabase storage not enabled in configuration")
	}

	if a.config.Supabase.URL == "" || a.config.Supabase.ServiceKey == "" {
		return "", fmt.Errorf("supabase URL or service key not configured")
	}

	// Validate Supabase URL to prevent SSRF
	if err := validateSupabaseURL(a.config.Supabase.URL); err != nil {
		return "", err
	}

	// Validate local path is within exports directory (prevent path traversal)
	absPath, err := filepath.Abs(localPath)
	if err != nil {
		return "", fmt.Errorf("invalid file path: %w", err)
	}
	// Ensure it's within the exports directory
	if !strings.Contains(absPath, "exports") {
		return "", fmt.Errorf("file must be within exports directory")
	}

	bucket := a.config.Supabase.StorageBucket
	if bucket == "" {
		bucket = "reports"
	}

	log.Printf("📤 Uploading report to Supabase Storage: %s", localPath)

	// Read the file
	fileData, err := os.ReadFile(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Extract filename and create remote path
	filename := filepath.Base(localPath)
	remotePath := fmt.Sprintf("intelligence/%s", filename)

	// Construct the upload URL
	uploadURL := fmt.Sprintf("%s/storage/v1/object/%s/%s",
		strings.TrimSuffix(a.config.Supabase.URL, "/"),
		bucket,
		remotePath,
	)

	// Create HTTP request
	req, err := http.NewRequest("POST", uploadURL, bytes.NewReader(fileData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+a.config.Supabase.ServiceKey)
	req.Header.Set("Content-Type", "application/pdf")

	// Execute request
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for error details
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Construct the public URL
	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s",
		strings.TrimSuffix(a.config.Supabase.URL, "/"),
		bucket,
		remotePath,
	)

	log.Printf("✅ Report uploaded successfully: %s", publicURL)
	return publicURL, nil
}

// ListRemoteReports lists all reports stored in Supabase Storage
func (a *App) ListRemoteReports() ([]ReportMetadata, error) {
	if err := a.requirePermission("reports:view"); err != nil {
		return nil, err
	}

	// Check if Supabase is configured
	if !a.config.Supabase.Enabled {
		return nil, fmt.Errorf("supabase storage not enabled in configuration")
	}

	if a.config.Supabase.URL == "" || a.config.Supabase.AnonKey == "" {
		return nil, fmt.Errorf("supabase URL or anon key not configured")
	}

	// Validate Supabase URL to prevent SSRF
	if err := validateSupabaseURL(a.config.Supabase.URL); err != nil {
		return nil, err
	}

	bucket := a.config.Supabase.StorageBucket
	if bucket == "" {
		bucket = "reports"
	}

	log.Printf("📋 Listing reports from Supabase Storage bucket: %s", bucket)

	// Construct the list URL
	listURL := fmt.Sprintf("%s/storage/v1/object/list/%s",
		strings.TrimSuffix(a.config.Supabase.URL, "/"),
		bucket,
	)

	// Create request body with prefix filter
	requestBody := map[string]any{
		"prefix": "intelligence/",
		"limit":  1000,
		"offset": 0,
	}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", listURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers - use service key for listing to get full metadata
	req.Header.Set("Authorization", "Bearer "+a.config.Supabase.ServiceKey)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var supabaseFiles []supabaseListResponse
	if err := json.Unmarshal(body, &supabaseFiles); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to ReportMetadata
	reports := make([]ReportMetadata, 0, len(supabaseFiles))
	for _, file := range supabaseFiles {
		// Skip directories
		if file.Metadata.Size == 0 && file.Name == ".emptyFolderPlaceholder" {
			continue
		}

		reports = append(reports, ReportMetadata{
			Name:      file.Name,
			Path:      fmt.Sprintf("intelligence/%s", file.Name),
			Size:      file.Metadata.Size,
			CreatedAt: file.CreatedAt,
			MimeType:  file.Metadata.Mimetype,
		})
	}

	log.Printf("✅ Found %d reports in Supabase Storage", len(reports))
	return reports, nil
}

// DownloadReport downloads a report from Supabase Storage to local exports directory
// Returns the local file path on success
func (a *App) DownloadReport(remotePath string) (string, error) {
	if err := a.requirePermission("reports:view"); err != nil {
		return "", err
	}

	// Validate remote path to prevent path traversal
	cleanPath := filepath.Clean(remotePath)
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("invalid report path: path traversal detected")
	}

	// Check if Supabase is configured
	if !a.config.Supabase.Enabled {
		return "", fmt.Errorf("supabase storage not enabled in configuration")
	}

	if a.config.Supabase.URL == "" || a.config.Supabase.AnonKey == "" {
		return "", fmt.Errorf("supabase URL or anon key not configured")
	}

	// Validate Supabase URL to prevent SSRF
	if err := validateSupabaseURL(a.config.Supabase.URL); err != nil {
		return "", err
	}

	bucket := a.config.Supabase.StorageBucket
	if bucket == "" {
		bucket = "reports"
	}

	log.Printf("📥 Downloading report from Supabase: %s", remotePath)

	// Construct download URL
	downloadURL := fmt.Sprintf("%s/storage/v1/object/%s/%s",
		strings.TrimSuffix(a.config.Supabase.URL, "/"),
		bucket,
		remotePath,
	)

	// Create HTTP request
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+a.config.Supabase.AnonKey)

	// Execute request
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("download failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Extract filename from remote path
	filename := filepath.Base(remotePath)

	// Create local directory if it doesn't exist
	localDir := "exports/intelligence"
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create local directory: %w", err)
	}

	// Create local file path
	localPath := filepath.Join(localDir, filename)

	// Create local file
	outFile, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to create local file: %w", err)
	}
	defer outFile.Close()

	// Copy data to file with size limit (100MB max)
	maxSize := int64(100 * 1024 * 1024) // 100MB max
	limitedReader := io.LimitReader(resp.Body, maxSize)
	bytesWritten, err := io.Copy(outFile, limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	if bytesWritten >= maxSize {
		os.Remove(localPath) // Clean up partial download
		return "", fmt.Errorf("report file exceeds maximum size of 100MB")
	}

	log.Printf("✅ Report downloaded successfully: %s (%d bytes)", localPath, bytesWritten)
	return localPath, nil
}

// DeleteRemoteReport deletes a report from Supabase Storage
func (a *App) DeleteRemoteReport(remotePath string) error {
	if err := a.requirePermission("settings:manage"); err != nil {
		return err
	}

	// Validate remote path to prevent path traversal
	cleanPath := filepath.Clean(remotePath)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid report path: path traversal detected")
	}

	// Check if Supabase is configured
	if !a.config.Supabase.Enabled {
		return fmt.Errorf("supabase storage not enabled in configuration")
	}

	if a.config.Supabase.URL == "" || a.config.Supabase.ServiceKey == "" {
		return fmt.Errorf("supabase URL or service key not configured")
	}

	// Validate Supabase URL to prevent SSRF
	if err := validateSupabaseURL(a.config.Supabase.URL); err != nil {
		return err
	}

	bucket := a.config.Supabase.StorageBucket
	if bucket == "" {
		bucket = "reports"
	}

	log.Printf("🗑️  Deleting report from Supabase: %s", remotePath)

	// Construct delete URL
	deleteURL := fmt.Sprintf("%s/storage/v1/object/%s/%s",
		strings.TrimSuffix(a.config.Supabase.URL, "/"),
		bucket,
		remotePath,
	)

	// Create HTTP request
	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers - use service key for delete operations
	req.Header.Set("Authorization", "Bearer "+a.config.Supabase.ServiceKey)

	// Execute request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	defer resp.Body.Close()

	// Read response for error details
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete failed with status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("✅ Report deleted successfully: %s", remotePath)
	return nil
}
