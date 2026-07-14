// ============================================================================
// ASSETS SERVICE - Database-backed binary assets (letterhead, logos, etc.)
//
// PURPOSE: Store assets in database so they sync via Supabase automatically
// No need to include asset folders in deployment packages
//
// Wave 5 A.1: the asset store lives in pkg/infra/assets. These delegates
// keep the Wails binding surface, the RBAC guards, and the host-owned
// concerns (application-path discovery, letterhead filesystem fallback).
// ============================================================================

package main

import (
	"log"
	"os"
	"path/filepath"

	infraassets "ph_holdings_app/pkg/infra/assets"
)

// Asset represents a binary asset stored in the database. The alias keeps
// the table shape, JSON contract, and model registry at the root.
type Asset = infraassets.Asset

// AssetInfo is returned when listing assets (without the data)
type AssetInfo = infraassets.Info

// Known asset names
const (
	AssetLetterhead    = infraassets.Letterhead
	AssetLetterheadAHS = infraassets.LetterheadAHS
	AssetLogo          = infraassets.Logo
	AssetStamp         = infraassets.Stamp
)

// EnsureAssetsTable creates the assets table if it doesn't exist
func (a *App) EnsureAssetsTable() error {
	return a.assetService().EnsureTable()
}

// UploadAsset stores a binary file as a base64-encoded asset in the database
func (a *App) UploadAsset(name, description, filePath string) error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	return a.assetService().UpsertFromFile(name, description, filePath)
}

// getAssetInternal retrieves an asset without permission check (for internal use like PDF generation)
func (a *App) getAssetInternal(name string) ([]byte, error) {
	return a.assetService().Get(name)
}

// GetAsset retrieves an asset from the database and returns the raw bytes (Wails-bound, requires permission)
func (a *App) GetAsset(name string) ([]byte, error) {
	if err := a.requirePermission("settings:view"); err != nil {
		return nil, err
	}
	return a.assetService().Get(name)
}

// GetAssetToFile retrieves an asset and saves it to a file, returning the path
// This is useful for PDF generation which needs a file path (internal, no permission check)
func (a *App) GetAssetToFile(name string) (string, error) {
	return a.assetService().GetToFile(name)
}

// ListAssets returns info about all stored assets (without the data)
func (a *App) ListAssets() ([]AssetInfo, error) {
	if err := a.requirePermission("settings:view"); err != nil {
		return nil, err
	}
	return a.assetService().List()
}

// DeleteAsset removes an asset from the database
func (a *App) DeleteAsset(name string) error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	return a.assetService().Delete(name)
}

// HasAsset checks if an asset exists in the database
func (a *App) HasAsset(name string) bool {
	return a.assetService().Has(name)
}

// InitializeDefaultAssets uploads default assets if they don't exist
// Called during app startup
func (a *App) InitializeDefaultAssets() {
	a.EnsureAssetsTable()

	// Iterate the registry rather than hand-duplicating one call per division
	// (the two literal calls this replaced could silently drift from
	// DivisionProfile.LetterheadFile/LetterheadAssetName — exactly the class
	// of bug the registry exists to prevent). Byte-identical for the
	// synthetic overlay: LetterheadAssetName/LetterheadFile reproduce
	// "letterhead"/"Acme Instrumentation Letterhead.png" and
	// "letterhead_ahs"/"Beacon Controls Letterhead.jpg" exactly.
	for _, div := range activeOverlay.Divisions {
		a.ensureDefaultLetterheadAsset(div.LetterheadAssetName, div.LetterheadFile, div.Key+" letterhead template for PDF generation")
	}
}

// ensureDefaultLetterheadAsset builds the host's candidate artwork paths
// (application paths, executable-relative, cwd) and hands seeding to the
// asset service.
func (a *App) ensureDefaultLetterheadAsset(assetName, letterheadFile, description string) {
	candidatePaths := []string{}

	// Check various locations - GetApplicationPaths may return nil if no license is activated
	paths := a.getAppPaths()
	if paths != nil {
		candidatePaths = append(candidatePaths, filepath.Join(paths.ProjectRoot, "data/ssot", letterheadFile))
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidatePaths = append(candidatePaths,
			filepath.Join(exeDir, "..", "..", "data/ssot", letterheadFile),
			filepath.Join(exeDir, "..", "Resources", "data", letterheadFile),
			filepath.Join(exeDir, "data", letterheadFile),
			filepath.Join(exeDir, "data/ssot", letterheadFile),
		)
	}

	if cwd, err := os.Getwd(); err == nil {
		candidatePaths = append(candidatePaths, filepath.Join(cwd, "data/ssot", letterheadFile))
	}

	a.assetService().EnsureDefaultLetterhead(assetName, letterheadFile, description, candidatePaths)
}

// GetLetterheadPath returns the path to the letterhead image
// First checks database cache, then falls back to file system
func (a *App) GetLetterheadPath() string {
	if a.HasAsset(AssetLetterhead) {
		cachePath, err := a.GetAssetToFile(AssetLetterhead)
		if err == nil {
			return cachePath
		}
		log.Printf("Warning: Failed to extract letterhead from DB: %v", err)
	}

	// Fall back to file system search
	return a.letterheadImagePath()
}

// getAssetCacheDir returns the directory for cached asset files
func (a *App) getAssetCacheDir() string {
	// Use AppData on Windows, or temp directory
	if appData := os.Getenv("LOCALAPPDATA"); appData != "" {
		return filepath.Join(appData, "PHHoldings", "asset_cache")
	}
	return filepath.Join(os.TempDir(), "ph_holdings_assets")
}

// Helper wrappers kept for root callers/tests; canonical implementations
// live in pkg/infra/assets.

func getMimeType(ext string) string {
	return infraassets.MimeTypeForExtension(ext)
}

func getExtensionFromMime(mimeType string) string {
	return infraassets.ExtensionForMimeType(mimeType)
}
