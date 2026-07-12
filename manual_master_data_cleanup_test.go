//go:build manual

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManualApplyMasterDataCleanup(t *testing.T) {
	if os.Getenv("MASTER_DATA_CLEANUP_COMMIT") != "1" {
		t.Skip("set MASTER_DATA_CLEANUP_COMMIT=1 to apply low-risk master-data cleanup to the runtime database")
	}

	runtimePath := appDataDatabasePath()
	if runtimePath == "" || !fileExists(runtimePath) {
		t.Skip("runtime deployment database not present on this machine")
	}

	db := openDeploymentAuditTestDB(t, runtimePath)
	defer func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}()

	reportPath := filepath.Join("docs", "MASTER_DATA_CLEANUP_REVIEW_2026_04_08.md")
	auditBefore, err := buildMasterDataCleanupAudit(db)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(reportPath), 0755))
	require.NoError(t, os.WriteFile(reportPath, []byte(renderMasterDataCleanupReport(auditBefore)), 0644))

	result, _, err := ApplyLowRiskMasterDataCleanup(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec("PRAGMA wal_checkpoint(FULL)").Error)

	auditAfter, err := buildMasterDataCleanupAudit(db)
	require.NoError(t, err)

	t.Logf("master-data cleanup report: %s", reportPath)
	t.Logf("customer groups merged=%d records=%d", result.CustomerGroupsMerged, result.CustomerRecordsMerged)
	t.Logf("supplier groups merged=%d records=%d", result.SupplierGroupsMerged, result.SupplierRecordsMerged)
	t.Logf("remaining customer candidate groups=%d", len(auditAfter.CustomerCandidates))
	t.Logf("remaining supplier candidate groups=%d", len(auditAfter.SupplierCandidates))
}
