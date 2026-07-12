//go:build manual

package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func defaultOneDriveImportRoot() string {
	if root := strings.TrimSpace(os.Getenv("ONEDRIVE_IMPORT_ROOT")); root != "" {
		return root
	}
	return "/Users/developer/Downloads/OneDrive_2026-03-31/Offers 2026"
}

func resolveOneDriveImportYear(rootPath string) int {
	if raw := strings.TrimSpace(os.Getenv("ONEDRIVE_IMPORT_YEAR")); raw != "" {
		if year, err := strconv.Atoi(raw); err == nil {
			if year < 100 {
				return 2000 + year
			}
			return year
		}
	}
	if inferred := inferYearFromPath(rootPath); inferred != 0 {
		return inferred
	}
	return 2026
}

func resolveOneDriveImportDBPath() string {
	if dbPath := strings.TrimSpace(os.Getenv("ONEDRIVE_IMPORT_DB")); dbPath != "" {
		return dbPath
	}
	return filepath.Join(".", "ph_holdings.db")
}
