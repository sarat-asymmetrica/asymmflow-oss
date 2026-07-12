//go:build manual

package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type deploymentEmployeeKey struct {
	DisplayName string
	Role        string
	Key         string
	Activated   bool
}

type deploymentRoleKeySpec struct {
	Role        string
	DisplayName string
	Key         string
	Notes       string
}

func resolveDeploymentSource(candidates ...string) (string, error) {
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no deployment source found in candidates: %v", candidates)
}

func copySanitizedDeploymentDB(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return err
	}
	cleanupSQLiteSidecars(dst)

	db, err := gorm.Open(sqlite.Open(dst), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return err
	}

	if err := sanitizeDeploymentDatabase(db); err != nil {
		return err
	}
	if err := ensureDeploymentLetterheadAssets(db); err != nil {
		return err
	}

	if err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)").Error; err != nil {
		return err
	}
	if err := db.Exec("PRAGMA journal_mode=DELETE").Error; err != nil {
		return err
	}
	cleanupSQLiteSidecars(dst)
	return nil
}

func sanitizeDeploymentDatabase(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	if db.Migrator().HasTable("license_keys") {
		if err := db.Exec(`
			UPDATE license_keys
			SET activated = 0,
			    activated_at = NULL,
			    device_hash = ''
			WHERE activated = 1
		`).Error; err != nil {
			return err
		}

		if err := db.Exec(`
			DELETE FROM license_keys
			WHERE role = 'developer'
			   OR key LIKE 'PH-DEV-%'
		`).Error; err != nil {
			return err
		}
	}

	if err := scrubCleanSlateWorkflowData(db); err != nil {
		return err
	}

	if db.Migrator().HasTable("expense_entries") {
		if db.Migrator().HasTable("expense_approvals") {
			if err := db.Exec(`
				DELETE FROM expense_approvals
				WHERE expense_entry_id IN (
					SELECT id FROM expense_entries
					WHERE source_type = 'payroll' OR cost_center = 'Payroll'
				)
			`).Error; err != nil {
				return err
			}
		}
		if db.Migrator().HasTable("expense_allocations") {
			if err := db.Exec(`
				DELETE FROM expense_allocations
				WHERE expense_entry_id IN (
					SELECT id FROM expense_entries
					WHERE source_type = 'payroll' OR cost_center = 'Payroll'
				)
			`).Error; err != nil {
				return err
			}
		}
		if db.Migrator().HasTable("expense_attachments") {
			if err := db.Exec(`
				DELETE FROM expense_attachments
				WHERE expense_entry_id IN (
					SELECT id FROM expense_entries
					WHERE source_type = 'payroll' OR cost_center = 'Payroll'
				)
			`).Error; err != nil {
				return err
			}
		}
		if err := db.Exec(`
			DELETE FROM expense_entries
			WHERE source_type = 'payroll' OR cost_center = 'Payroll'
		`).Error; err != nil {
			return err
		}
	}

	if db.Migrator().HasTable("journal_entries") {
		if db.Migrator().HasTable("journal_lines") {
			if err := db.Exec(`
				DELETE FROM journal_lines
				WHERE entry_id IN (
					SELECT id FROM journal_entries
					WHERE source_type IN ('payroll_run', 'payroll_payout')
				)
			`).Error; err != nil {
				return err
			}
		}
		if err := db.Exec(`
			DELETE FROM journal_entries
			WHERE source_type IN ('payroll_run', 'payroll_payout')
		`).Error; err != nil {
			return err
		}
	}

	for _, table := range []string{
		"payroll_components",
		"payroll_run_items",
		"payroll_payouts",
		"payroll_runs",
		"payroll_periods",
		"employee_compensation_profiles",
	} {
		if db.Migrator().HasTable(table) {
			if err := db.Exec("DELETE FROM " + table).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func scrubCleanSlateWorkflowData(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	if db.Migrator().HasTable("notification_receipts") && db.Migrator().HasTable("notifications") {
		if err := db.Exec(`
			DELETE FROM notification_receipts
			WHERE notification_id IN (
				SELECT id FROM notifications
				WHERE source_type = 'task'
				   OR notification_type LIKE 'task%'
			)
		`).Error; err != nil {
			return err
		}
	}
	if db.Migrator().HasTable("notifications") {
		if err := db.Exec(`
			DELETE FROM notifications
			WHERE source_type = 'task'
			   OR notification_type LIKE 'task%'
		`).Error; err != nil {
			return err
		}
	}

	for _, table := range []string{
		"task_activity",
		"task_comments",
		"task_items",
		"followup_tasks",
	} {
		if db.Migrator().HasTable(table) {
			if err := db.Exec("DELETE FROM " + table).Error; err != nil {
				return err
			}
		}
	}

	if db.Migrator().HasTable("bank_line_payment_allocations") {
		if err := db.Exec("DELETE FROM bank_line_payment_allocations").Error; err != nil {
			return err
		}
	}
	if db.Migrator().HasTable("bank_statement_lines") {
		if err := db.Exec(`
			UPDATE bank_statement_lines
			SET is_matched = 0,
			    matched_payment_id = '',
			    matched_invoice_ids = '',
			    match_type = 'Unmatched',
			    match_confidence = 0,
			    verified_by = '',
			    verified_at = NULL
		`).Error; err != nil {
			return err
		}
	}

	for _, table := range []string{
		"supplier_payments",
		"supplier_invoice_items",
		"supplier_invoices",
		"grn_items",
		"goods_received_notes",
		"purchase_order_items",
		"purchase_orders",
		"payments",
	} {
		if db.Migrator().HasTable(table) {
			if err := db.Exec("DELETE FROM " + table).Error; err != nil {
				return err
			}
		}
	}
	if db.Migrator().HasTable("invoices") {
		if err := db.Exec(`
			UPDATE invoices
			SET outstanding_bhd = grand_total_bhd,
			    status = CASE
			        WHEN status IN ('Paid', 'PartiallyPaid', 'Overdue') THEN 'Sent'
			        ELSE status
			    END
		`).Error; err != nil {
			return err
		}
	}
	if db.Migrator().HasTable("serial_numbers") {
		if err := db.Exec(`
			UPDATE serial_numbers
			SET status = CASE
			        WHEN COALESCE(dn_number, '') = '' AND COALESCE(invoice_number, '') = '' THEN 'Available'
			        ELSE status
			    END,
			    po_id = '',
			    po_number = '',
			    grn_item_id = '',
			    grn_number = '',
			    received_date = NULL
			WHERE COALESCE(po_id, '') <> ''
			   OR COALESCE(po_number, '') <> ''
			   OR COALESCE(grn_item_id, '') <> ''
			   OR COALESCE(grn_number, '') <> ''
		`).Error; err != nil {
			return err
		}
	}

	if db.Migrator().HasTable("collaborative_pending_operations") {
		if err := db.Exec(`
			DELETE FROM collaborative_pending_operations
			WHERE entity_type IN ('task', 'task_activity', 'task_comment')
		`).Error; err != nil {
			return err
		}
	}

	if db.Migrator().HasTable("serial_numbers") {
		if err := db.Exec(`
			UPDATE serial_numbers
			SET status = 'Available',
			    dn_item_id = '',
			    dn_number = '',
			    shipped_date = NULL,
			    warranty_start_date = NULL,
			    warranty_end_date = NULL
			WHERE COALESCE(dn_number, '') LIKE 'DN-ORD-PH25/%'
		`).Error; err != nil {
			return err
		}
	}

	if db.Migrator().HasTable("delivery_notes") {
		dnScrubWhere := "dn_number LIKE 'DN-ORD-PH25/%'"
		if db.Migrator().HasTable("orders") {
			dnScrubWhere += ` OR order_id IN (
				SELECT id FROM orders
				WHERE strftime('%Y', order_date) = '2025'
			)`
		}
		if db.Migrator().HasTable("delivery_note_items") {
			if err := db.Exec(`
				DELETE FROM delivery_note_items
				WHERE delivery_note_id IN (
					SELECT id FROM delivery_notes WHERE ` + dnScrubWhere + `
				)
			`).Error; err != nil {
				return err
			}
		}
		if err := db.Exec("DELETE FROM delivery_notes WHERE " + dnScrubWhere).Error; err != nil {
			return err
		}
	}

	return nil
}

func TestSanitizeDeploymentDatabaseScrubsTasksAndGeneratedPH25DeliveryNotes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:sanitize-clean-slate?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&LicenseKey{},
		&TaskItem{},
		&TaskComment{},
		&TaskActivity{},
		&FollowUpTask{},
		&Notification{},
		&NotificationReceipt{},
		&CollaborativePendingOperation{},
		&Order{},
		&Invoice{},
		&Payment{},
		&PurchaseOrder{},
		&PurchaseOrderItem{},
		&GoodsReceivedNote{},
		&GRNItem{},
		&SupplierInvoice{},
		&SupplierInvoiceItem{},
		&SupplierPayment{},
		&BankStatementLine{},
		&BankLinePaymentAllocation{},
		&DeliveryNote{},
		&DeliveryNoteItem{},
		&SerialNumber{},
	); err != nil {
		t.Fatalf("failed to migrate clean-slate tables: %v", err)
	}

	order := Order{
		Base:        Base{ID: "order-2025"},
		OrderNumber: "ORD-INV-2025-0001",
		OrderDate:   time.Date(2025, 1, 12, 0, 0, 0, 0, time.UTC),
	}
	dn := DeliveryNote{
		Base:         Base{ID: "dn-2025"},
		OrderID:      order.ID,
		DNNumber:     "DN-ORD-INV-2025-0001",
		DeliveryDate: time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC),
		Status:       "Delivered",
	}
	requireCreate := func(value any) {
		if err := db.Create(value).Error; err != nil {
			t.Fatalf("failed to seed %T: %v", value, err)
		}
	}
	requireCreate(&order)
	requireCreate(&dn)
	requireCreate(&DeliveryNoteItem{Base: Base{ID: "dn-item-2025"}, DeliveryNoteID: dn.ID, QuantityDelivered: 1})
	requireCreate(&SerialNumber{Base: Base{ID: "serial-2025"}, SerialNo: "SER-2025", Status: "Delivered", DNItemID: "dn-item-2025", DNNumber: dn.DNNumber})
	requireCreate(&TaskItem{Base: Base{ID: "task-1"}, Title: "Do not package"})
	requireCreate(&TaskComment{Base: Base{ID: "task-comment-1"}, TaskID: "task-1", Body: "comment"})
	requireCreate(&TaskActivity{Base: Base{ID: "task-activity-1"}, TaskID: "task-1", ActivityType: "create"})
	requireCreate(&FollowUpTask{Base: Base{ID: "followup-task-1"}, Title: "Legacy followup", Status: "pending", Priority: "medium"})
	requireCreate(&Notification{Base: Base{ID: "notification-task-1"}, SourceType: "task", SourceID: "task-1", NotificationType: "task"})
	requireCreate(&NotificationReceipt{Base: Base{ID: "receipt-task-1"}, NotificationID: "notification-task-1"})
	requireCreate(&CollaborativePendingOperation{Base: Base{ID: "op-task-1"}, EntityType: "task", EntityID: "task-1", Operation: "create"})
	requireCreate(&Invoice{Base: Base{ID: "invoice-1"}, InvoiceNumber: "INV-1", GrandTotalBHD: 100, OutstandingBHD: 0, Status: "Paid"})
	requireCreate(&Payment{Base: Base{ID: "payment-1"}, InvoiceID: "invoice-1", InvoiceNumber: "INV-1", AmountBHD: 100, PaymentDate: time.Now(), PaymentMethod: "Bank Transfer"})
	requireCreate(&PurchaseOrder{Base: Base{ID: "po-1"}, OrderID: order.ID, PONumber: "PO-1", PODate: time.Now(), SupplierID: "supplier-1", SupplierName: "Supplier"})
	requireCreate(&PurchaseOrderItem{Base: Base{ID: "po-item-1"}, PurchaseOrderID: "po-1", Description: "PO item", Quantity: 1})
	requireCreate(&GoodsReceivedNote{Base: Base{ID: "grn-1"}, PurchaseOrderID: "po-1", GRNNumber: "GRN-1", ReceivedDate: time.Now()})
	requireCreate(&GRNItem{Base: Base{ID: "grn-item-1"}, GRNID: "grn-1", POItemID: "po-item-1", QuantityOrdered: 1, QuantityReceived: 1, QuantityAccepted: 1})
	requireCreate(&SupplierInvoice{Base: Base{ID: "supplier-invoice-1"}, SupplierID: "supplier-1", PurchaseOrderID: "po-1", GRNID: "grn-1", InvoiceNumber: "SI-1", InvoiceDate: time.Now(), Status: "Paid", PaymentStatus: "Paid"})
	requireCreate(&SupplierInvoiceItem{Base: Base{ID: "supplier-invoice-item-1"}, SupplierInvoiceID: "supplier-invoice-1", Description: "Supplier item", Quantity: 1})
	requireCreate(&SupplierPayment{Base: Base{ID: "supplier-payment-1"}, SupplierInvoiceID: "supplier-invoice-1", SupplierID: "supplier-1", AmountBHD: 100, PaymentDate: time.Now(), PaymentMethod: "Bank Transfer"})
	requireCreate(&BankStatementLine{Base: Base{ID: "bank-line-1"}, IsMatched: true, MatchedPaymentID: "payment-1", MatchedInvoiceIDs: `["invoice-1"]`, MatchType: "Manual", MatchConfidence: 1, VerifiedBy: "tester"})
	requireCreate(&BankLinePaymentAllocation{Base: Base{ID: "bank-allocation-1"}, BankStatementLineID: "bank-line-1", SupplierInvoiceID: stringPtr("supplier-invoice-1"), AllocatedAmount: 100})
	requireCreate(&SerialNumber{Base: Base{ID: "serial-po-1"}, SerialNo: "SER-PO-1", Status: "Received", POID: "po-1", PONumber: "PO-1", GRNItemID: "grn-item-1", GRNNumber: "GRN-1"})

	if err := sanitizeDeploymentDatabase(db); err != nil {
		t.Fatalf("sanitize deployment database failed: %v", err)
	}

	assertTableCount := func(table string, expected int64) {
		var count int64
		if err := db.Table(table).Count(&count).Error; err != nil {
			t.Fatalf("failed to count %s: %v", table, err)
		}
		if count != expected {
			t.Fatalf("expected %s count %d, got %d", table, expected, count)
		}
	}
	assertTableCount("task_items", 0)
	assertTableCount("task_comments", 0)
	assertTableCount("task_activity", 0)
	assertTableCount("followup_tasks", 0)
	assertTableCount("notifications", 0)
	assertTableCount("notification_receipts", 0)
	assertTableCount("collaborative_pending_operations", 0)
	assertTableCount("payments", 0)
	assertTableCount("supplier_payments", 0)
	assertTableCount("supplier_invoice_items", 0)
	assertTableCount("supplier_invoices", 0)
	assertTableCount("grn_items", 0)
	assertTableCount("goods_received_notes", 0)
	assertTableCount("purchase_order_items", 0)
	assertTableCount("purchase_orders", 0)
	assertTableCount("bank_line_payment_allocations", 0)
	assertTableCount("delivery_note_items", 0)
	assertTableCount("delivery_notes", 0)

	var serial SerialNumber
	if err := db.First(&serial, "id = ?", "serial-2025").Error; err != nil {
		t.Fatalf("failed to load scrubbed serial: %v", err)
	}
	if serial.Status != "Available" || serial.DNItemID != "" || serial.DNNumber != "" {
		t.Fatalf("serial was not released from scrubbed DN: status=%s dn_item=%s dn=%s", serial.Status, serial.DNItemID, serial.DNNumber)
	}

	var invoice Invoice
	if err := db.First(&invoice, "id = ?", "invoice-1").Error; err != nil {
		t.Fatalf("failed to load reset invoice: %v", err)
	}
	if invoice.Status != "Sent" || invoice.OutstandingBHD != invoice.GrandTotalBHD {
		t.Fatalf("invoice payment state was not reset: status=%s outstanding=%.3f total=%.3f", invoice.Status, invoice.OutstandingBHD, invoice.GrandTotalBHD)
	}

	var bankLine BankStatementLine
	if err := db.First(&bankLine, "id = ?", "bank-line-1").Error; err != nil {
		t.Fatalf("failed to load reset bank line: %v", err)
	}
	if bankLine.IsMatched || bankLine.MatchedPaymentID != "" || bankLine.MatchedInvoiceIDs != "" || bankLine.MatchType != "Unmatched" {
		t.Fatalf("bank line payment match was not reset: matched=%t payment=%s invoices=%s type=%s", bankLine.IsMatched, bankLine.MatchedPaymentID, bankLine.MatchedInvoiceIDs, bankLine.MatchType)
	}

	var poSerial SerialNumber
	if err := db.First(&poSerial, "id = ?", "serial-po-1").Error; err != nil {
		t.Fatalf("failed to load scrubbed PO serial: %v", err)
	}
	if poSerial.POID != "" || poSerial.PONumber != "" || poSerial.GRNItemID != "" || poSerial.GRNNumber != "" || poSerial.Status != "Available" {
		t.Fatalf("serial was not released from scrubbed procurement data: status=%s po=%s grn=%s", poSerial.Status, poSerial.POID, poSerial.GRNItemID)
	}
}

func cleanupSQLiteSidecars(dbPath string) {
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")
}

func ensureDeploymentLetterheadAssets(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	assets := []struct {
		Name        string
		FileName    string
		Description string
	}{
		{
			Name:        AssetLetterhead,
			FileName:    "Acme Instrumentation Letterhead.png",
			Description: "Primary letterhead template for PDF generation",
		},
		{
			Name:        AssetLetterheadAHS,
			FileName:    "Beacon Controls Letterhead.jpg",
			Description: "Secondary letterhead template for PDF generation",
		},
	}

	for _, assetSpec := range assets {
		var count int64
		if err := db.Model(&Asset{}).Where("name = ?", assetSpec.Name).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}

		sourcePath, err := resolveDeploymentSource(
			filepath.Join("data/ssot", assetSpec.FileName),
			filepath.Join("deploy_package", "data", assetSpec.FileName),
			filepath.Join("build", "bin", "data", assetSpec.FileName),
		)
		if err != nil {
			return fmt.Errorf("failed to resolve %s: %w", assetSpec.Name, err)
		}

		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return err
		}

		encoded := base64.StdEncoding.EncodeToString(data)
		record := Asset{
			ID:          assetSpec.Name,
			Name:        assetSpec.Name,
			Description: assetSpec.Description,
			MimeType:    getMimeType(filepath.Ext(sourcePath)),
			Data:        encoded,
			Size:        int64(len(data)),
		}

		if err := db.Where("name = ?", assetSpec.Name).Assign(record).FirstOrCreate(&record).Error; err != nil {
			return err
		}
	}

	return nil
}

func copyDeploymentDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}

func sanitizeAppBundle(appPath string, deploymentEnv []byte, dbSrc string) error {
	resourcesDir := filepath.Join(appPath, "Contents", "Resources")
	// The app may have been launched locally before packaging. Do not ship
	// machine-local field crypto salt inside the client bundle.
	_ = os.Remove(filepath.Join(appPath, "Contents", "MacOS", ".field_crypto_salt"))
	if err := os.MkdirAll(filepath.Join(resourcesDir, "data"), 0755); err != nil {
		return err
	}
	if err := syncDeploymentAppIcon(resourcesDir); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(resourcesDir, ".env"), deploymentEnv, 0644); err != nil {
		return err
	}
	dbDst := filepath.Join(resourcesDir, "data", "ph_holdings.db")
	if err := copySanitizedDeploymentDB(dbSrc, dbDst); err != nil {
		return err
	}
	for _, assetFile := range []string{
		"Acme Instrumentation Letterhead.png",
		"Beacon Controls Letterhead.jpg",
	} {
		src, err := resolveDeploymentSource(
			filepath.Join("data/ssot", assetFile),
			filepath.Join("deploy_package", "data", assetFile),
		)
		if err != nil {
			return err
		}
		if err := copyDeploymentFile(src, filepath.Join(resourcesDir, "data", assetFile)); err != nil {
			return err
		}
	}
	cleanupSQLiteSidecars(dbDst)
	return nil
}

func syncDeploymentAppIcon(resourcesDir string) error {
	iconSrc, err := resolveDeploymentSource(
		filepath.Join("build", "iconfile.icns"),
		filepath.Join("build", "darwin", "iconfile.icns"),
	)
	if err != nil {
		return nil
	}
	iconDst := filepath.Join(resourcesDir, "iconfile.icns")
	return copyDeploymentFile(iconSrc, iconDst)
}

func TestPrepareDeploymentPackage(t *testing.T) {
	t.Skip("Skipping manual deployment package test: build/bin/AsymmFlow.app is not present in this experimental repo")

	dbPath := filepath.Join(".", "ph_holdings.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	app := &App{
		db:                     db,
		cache:                  NewCache(),
		startupImporting:       true,
		startupImportStartTime: time.Now(),
		currentUserID:          "manual-deploy-package",
		currentUser:            &User{Base: Base{ID: "manual-deploy-package"}, Username: "manual-deploy-package"},
	}
	t.Cleanup(app.cache.Stop)

	if err := app.ensureCriticalDeploymentFoundations(); err != nil {
		t.Fatalf("failed to ensure deployment database foundations: %v", err)
	}

	if os.Getenv("FLUSH_EMPLOYEE_KEYS") == "1" {
		if err := resetDeploymentLicensePool(app, db); err != nil {
			t.Fatalf("failed to rebuild deployment license pool: %v", err)
		}
	}

	if err := ensureDeploymentEmployeeKeys(db); err != nil {
		t.Fatalf("failed to ensure deployment employee keys: %v", err)
	}

	var employeeKeys []deploymentEmployeeKey
	if err := db.Raw(`
		SELECT display_name, role, key, activated
		FROM license_keys
		WHERE display_name IS NOT NULL AND display_name != '' AND activated = 0
		ORDER BY
			CASE role
				WHEN 'admin' THEN 1
				WHEN 'manager' THEN 2
				WHEN 'sales' THEN 3
				WHEN 'operations' THEN 4
				WHEN 'staff' THEN 5
				ELSE 99
			END,
			display_name
	`).Scan(&employeeKeys).Error; err != nil {
		t.Fatalf("failed to load employee keys: %v", err)
	}

	if len(employeeKeys) == 0 {
		t.Fatalf("no employee keys available to package")
	}

	_ = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")

	stamp := os.Getenv("DEPLOY_STAMP")
	if strings.TrimSpace(stamp) == "" {
		stamp = time.Now().Format("2006_01_02_150405")
	}
	packageDir := filepath.Join("deploy_package", "AsymmFlow_Deploy_"+stamp)
	if err := os.RemoveAll(packageDir); err != nil {
		t.Fatalf("failed to reset package dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(packageDir, "data"), 0755); err != nil {
		t.Fatalf("failed to create package data dir: %v", err)
	}

	// Refresh top-level deploy_package bundle as well.
	if err := os.MkdirAll(filepath.Join("deploy_package", "data"), 0755); err != nil {
		t.Fatalf("failed to create top-level package data dir: %v", err)
	}

	macAppBundle := filepath.Join("build", "bin", "AsymmFlow.app")
	if _, err := os.Stat(macAppBundle); err != nil {
		t.Fatalf("failed to locate mac app bundle: %v", err)
	}

	winBinary, err := resolveDeploymentSource(
		filepath.Join("build", "bin", "AsymmFlow.exe"),
		filepath.Join("deploy_package", "AsymmFlow.exe"),
	)
	if err != nil {
		t.Fatalf("failed to resolve windows binary: %v", err)
	}
	envSrc := filepath.Join("deploy_package", ".env")
	dbSrc := filepath.Join(".", "ph_holdings.db")
	templateImage := filepath.Join("deploy_package", "data", "Acme Instrumentation Letterhead.png")
	ahsTemplateImage := filepath.Join("data/ssot", "Beacon Controls Letterhead.jpg")

	copyTargets := []struct {
		src string
		dst string
	}{
		{winBinary, filepath.Join(packageDir, "AsymmFlow.exe")},
		{winBinary, filepath.Join("deploy_package", "AsymmFlow.exe")},
		{envSrc, filepath.Join(packageDir, ".env")},
		{envSrc, filepath.Join("deploy_package", ".env")},
		{templateImage, filepath.Join(packageDir, "data", "Acme Instrumentation Letterhead.png")},
		{ahsTemplateImage, filepath.Join(packageDir, "data", "Beacon Controls Letterhead.jpg")},
		{templateImage, filepath.Join("deploy_package", "data", "Acme Instrumentation Letterhead.png")},
		{ahsTemplateImage, filepath.Join("deploy_package", "data", "Beacon Controls Letterhead.jpg")},
		{filepath.Join("docs", "DEPLOYMENT_SIGNOFF_CHECKLIST_2026_04_08.md"), filepath.Join(packageDir, "DEPLOYMENT_SIGNOFF_CHECKLIST_2026_04_08.md")},
		{filepath.Join("docs", "DEPLOYMENT_SIGNOFF_CHECKLIST_2026_04_08.pdf"), filepath.Join(packageDir, "DEPLOYMENT_SIGNOFF_CHECKLIST_2026_04_08.pdf")},
		{filepath.Join("docs", "MENTOR_2025_ONEDRIVE_IMPORT_GUIDE_2026_04_10.md"), filepath.Join(packageDir, "MENTOR_2025_ONEDRIVE_IMPORT_GUIDE_2026_04_10.md")},
		{filepath.Join("docs", "DEPLOYMENT_SIGNOFF_CHECKLIST_2026_04_08.md"), filepath.Join("deploy_package", "DEPLOYMENT_SIGNOFF_CHECKLIST_2026_04_08.md")},
		{filepath.Join("docs", "DEPLOYMENT_SIGNOFF_CHECKLIST_2026_04_08.pdf"), filepath.Join("deploy_package", "DEPLOYMENT_SIGNOFF_CHECKLIST_2026_04_08.pdf")},
		{filepath.Join("docs", "MENTOR_2025_ONEDRIVE_IMPORT_GUIDE_2026_04_10.md"), filepath.Join("deploy_package", "MENTOR_2025_ONEDRIVE_IMPORT_GUIDE_2026_04_10.md")},
	}

	for _, target := range copyTargets {
		if err := copyDeploymentFile(target.src, target.dst); err != nil {
			t.Fatalf("failed to copy %s -> %s: %v", target.src, target.dst, err)
		}
	}
	webView2Bootstrapper := filepath.Join("build", "windows", "installer", "tmp", "MicrosoftEdgeWebview2Setup.exe")
	if _, err := os.Stat(webView2Bootstrapper); err == nil {
		for _, target := range []string{
			filepath.Join(packageDir, "MicrosoftEdgeWebview2Setup.exe"),
			filepath.Join("deploy_package", "MicrosoftEdgeWebview2Setup.exe"),
		} {
			if err := copyDeploymentFile(webView2Bootstrapper, target); err != nil {
				t.Fatalf("failed to copy WebView2 bootstrapper to %s: %v", target, err)
			}
		}
	}

	deploymentEnv, err := buildDeploymentEnvText(envSrc, stamp)
	if err != nil {
		t.Fatalf("failed to build deployment env: %v", err)
	}

	// Sanitize the built app bundle in place before packaging so direct local testing
	// and packaged distribution both use the same clean license state.
	if err := sanitizeAppBundle(macAppBundle, deploymentEnv, dbSrc); err != nil {
		t.Fatalf("failed to sanitize mac app bundle: %v", err)
	}
	for _, target := range []string{
		filepath.Join(packageDir, ".env"),
		filepath.Join("deploy_package", ".env"),
	} {
		if err := os.WriteFile(target, deploymentEnv, 0644); err != nil {
			t.Fatalf("failed to write deployment env %s: %v", target, err)
		}
	}

	for _, target := range []string{
		filepath.Join(packageDir, "AsymmFlow.app"),
		filepath.Join("deploy_package", "AsymmFlow.app"),
	} {
		if err := os.RemoveAll(target); err != nil {
			t.Fatalf("failed to reset app bundle target %s: %v", target, err)
		}
		if err := copyDeploymentDir(macAppBundle, target); err != nil {
			t.Fatalf("failed to copy app bundle to %s: %v", target, err)
		}
		if err := sanitizeAppBundle(target, deploymentEnv, dbSrc); err != nil {
			t.Fatalf("failed to sanitize copied app bundle %s: %v", target, err)
		}
	}

	for _, dbTarget := range []string{
		filepath.Join(packageDir, "data", "ph_holdings.db"),
		filepath.Join(packageDir, "ph_holdings.db"),
		filepath.Join("deploy_package", "data", "ph_holdings.db"),
		filepath.Join("deploy_package", "ph_holdings.db"),
	} {
		if err := copySanitizedDeploymentDB(dbSrc, dbTarget); err != nil {
			t.Fatalf("failed to prepare sanitized deployment DB %s: %v", dbTarget, err)
		}
	}

	generatedAt := time.Now()
	licenseText := buildLicenseKeysText(employeeKeys, generatedAt)
	installGuide := buildInstallGuideText(generatedAt)

	for _, target := range []string{
		filepath.Join(packageDir, "LICENSE_KEYS.txt"),
	} {
		if err := os.WriteFile(target, []byte(licenseText), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", target, err)
		}
	}
	if err := os.WriteFile(filepath.Join("deploy_package", "LICENSE_KEYS.txt"), []byte(buildLicenseKeysPlaceholderText(generatedAt)), 0644); err != nil {
		t.Fatalf("failed to write top-level license placeholder: %v", err)
	}

	for _, target := range []string{
		filepath.Join(packageDir, "INSTALL_GUIDE.txt"),
		filepath.Join("deploy_package", "INSTALL_GUIDE.txt"),
	} {
		if err := os.WriteFile(target, []byte(installGuide), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", target, err)
		}
	}
	if err := os.WriteFile(filepath.Join(packageDir, "README_START_HERE.txt"), []byte(buildStartHereText(stamp)), 0644); err != nil {
		t.Fatalf("failed to write start-here readme: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packageDir, "RUN_INSTALLED_ASYMMFLOW_DEBUG.bat"), []byte(buildInstalledLaunchDebugScriptText()), 0644); err != nil {
		t.Fatalf("failed to write launch debug script: %v", err)
	}

	if os.Getenv("BUILD_WINDOWS_INSTALLER") == "1" {
		installerPath, err := buildPHWindowsInstaller(packageDir, stamp)
		if err != nil {
			t.Fatalf("failed to build Windows installer: %v", err)
		}
		t.Logf("installer=%s", installerPath)
	}

	t.Logf("package_dir=%s employee_keys=%d", packageDir, len(employeeKeys))
}

func TestManualVerifyDeploymentPackageLicenseActivation(t *testing.T) {
	packageDir := strings.TrimSpace(os.Getenv("VERIFY_DEPLOYMENT_PACKAGE"))
	if packageDir == "" {
		t.Skip("set VERIFY_DEPLOYMENT_PACKAGE to a prepared deployment package directory")
	}

	dbSrc := filepath.Join(packageDir, "data", "ph_holdings.db")
	if _, err := os.Stat(dbSrc); err != nil {
		t.Fatalf("deployment database missing: %v", err)
	}

	sourceDB, err := gorm.Open(sqlite.Open(dbSrc), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open source deployment database: %v", err)
	}

	roleOrder := []string{"admin", "manager", "sales", "operations", "staff"}
	for _, role := range roleOrder {
		var key string
		if err := sourceDB.Raw(`
			SELECT key
			FROM license_keys
			WHERE role = ?
			  AND activated = 0
			  AND LOWER(COALESCE(display_name, '')) LIKE '%test%'
			ORDER BY id
			LIMIT 1
		`, role).Scan(&key).Error; err != nil {
			t.Fatalf("failed to load %s test key: %v", role, err)
		}
		if strings.TrimSpace(key) == "" {
			t.Fatalf("missing unactivated test key for role %s", role)
		}

		tempDB := filepath.Join(t.TempDir(), role+"_ph_holdings.db")
		if err := copyDeploymentFile(dbSrc, tempDB); err != nil {
			t.Fatalf("failed to copy DB for role %s: %v", role, err)
		}
		db, err := gorm.Open(sqlite.Open(tempDB), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			t.Fatalf("failed to open temp DB for role %s: %v", role, err)
		}

		app := &App{db: db, cache: NewCache()}
		t.Cleanup(app.cache.Stop)
		needsActivation, err := app.NeedsLicenseActivation()
		if err != nil {
			t.Fatalf("initial NeedsLicenseActivation failed for role %s: %v", role, err)
		}
		if !needsActivation {
			t.Fatalf("fresh package DB should need activation for role %s", role)
		}

		activation, err := app.ActivateLicense(key)
		if err != nil {
			t.Fatalf("ActivateLicense errored for role %s: %v", role, err)
		}
		if !activation.Success || activation.Role != role {
			t.Fatalf("ActivateLicense failed for role %s: success=%v activatedRole=%s message=%s", role, activation.Success, activation.Role, activation.Message)
		}

		validation, err := app.ValidateLicense()
		if err != nil {
			t.Fatalf("ValidateLicense errored for role %s: %v", role, err)
		}
		if !validation.Valid || validation.Role != role {
			t.Fatalf("ValidateLicense failed for role %s: valid=%v validatedRole=%s", role, validation.Valid, validation.Role)
		}
		needsActivation, err = app.NeedsLicenseActivation()
		if err != nil {
			t.Fatalf("post-activation NeedsLicenseActivation failed for role %s: %v", role, err)
		}
		if needsActivation {
			t.Fatalf("activated package DB still asks for key for role %s", role)
		}
	}

	tempDB := filepath.Join(t.TempDir(), "admin_generate_ph_holdings.db")
	if err := copyDeploymentFile(dbSrc, tempDB); err != nil {
		t.Fatalf("failed to copy DB for admin key generation: %v", err)
	}
	adminDB, err := gorm.Open(sqlite.Open(tempDB), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open admin temp DB: %v", err)
	}
	var adminKey string
	if err := adminDB.Raw(`
		SELECT key
		FROM license_keys
		WHERE role = 'admin'
		  AND activated = 0
		  AND LOWER(COALESCE(display_name, '')) LIKE '%test%'
		ORDER BY id
		LIMIT 1
	`).Scan(&adminKey).Error; err != nil {
		t.Fatalf("failed to load admin test key: %v", err)
	}
	adminApp := &App{db: adminDB, cache: NewCache()}
	t.Cleanup(adminApp.cache.Stop)
	if activation, err := adminApp.ActivateLicense(adminKey); err != nil || !activation.Success {
		t.Fatalf("admin activation failed before key generation: success=%v err=%v", activation.Success, err)
	}
	generatedKey, err := adminApp.GenerateLicenseKey("staff", "Verification new employee key", "admin-verification")
	if err != nil {
		t.Fatalf("admin could not generate additional license key: %v", err)
	}
	if !strings.HasPrefix(generatedKey, "PH-STF-") {
		t.Fatalf("generated staff key has unexpected prefix: %s", generatedKey)
	}
}

func copyDeploymentFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0755)
}

func deploymentVerificationHoldSyncTables() string {
	return strings.Join([]string{
		"payments",
		"supplier_invoices",
		"supplier_invoice_items",
		"supplier_payments",
		"purchase_orders",
		"purchase_order_items",
		"goods_received_notes",
		"grn_items",
		"task_items",
		"task_comments",
		"task_activity",
		"followup_tasks",
		"notifications",
		"notification_receipts",
		"bank_line_payment_allocations",
		"payroll_components",
		"payroll_run_items",
		"payroll_payouts",
		"payroll_runs",
		"payroll_periods",
		"employee_compensation_profiles",
	}, ",")
}

func buildDeploymentEnvText(src, stamp string) ([]byte, error) {
	data, err := os.ReadFile(src)
	if err != nil {
		return nil, err
	}

	text := string(data)
	lines := strings.Split(text, "\n")
	filtered := make([]string, 0, len(lines)+3)
	existingKeys := make(map[string]bool, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "PH_DB_PATH=") ||
			strings.HasPrefix(trimmed, "DATABASE_PATH=") ||
			strings.HasPrefix(trimmed, "ASYMMFLOW_DB_RESEED_STAMP=") ||
			strings.HasPrefix(trimmed, "ASYMMFLOW_SYNC_EXCLUDE_TABLES=") ||
			strings.HasPrefix(trimmed, "ASYMMFLOW_FLUSH_LICENSE_ON_RESEED=") ||
			strings.HasPrefix(trimmed, "ASYMMFLOW_LICENSE_FLUSH_STAMP=") {
			continue
		}
		if key, _, ok := strings.Cut(trimmed, "="); ok && key != "" && !strings.HasPrefix(key, "#") {
			existingKeys[key] = true
		}
		filtered = append(filtered, line)
	}
	text = strings.TrimRight(strings.Join(filtered, "\n"), "\n")
	if text != "" {
		text += "\n"
	}
	appendEnvIfPresent := func(key string) {
		if existingKeys[key] {
			return
		}
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			text += fmt.Sprintf("%s=%s\n", key, value)
			existingKeys[key] = true
		}
	}
	appendEnvIfPresent("ASYMM_AIML_API_KEY")
	appendEnvIfPresent("AIML_API_KEY")
	appendEnvIfPresent("ASYMM_AIML_MODEL")
	appendEnvIfPresent("AIML_MODEL")
	if !existingKeys["ASYMM_AIML_MODEL"] && !existingKeys["AIML_MODEL"] {
		text += fmt.Sprintf("AIML_MODEL=%s\n", getAIMLModelID())
		existingKeys["AIML_MODEL"] = true
	}
	appendEnvIfPresent("MISTRAL_API_KEY")
	if strings.TrimSpace(stamp) != "" {
		text += fmt.Sprintf("ASYMMFLOW_DB_RESEED_STAMP=%s\n", strings.TrimSpace(stamp))
	}
	text += fmt.Sprintf("ASYMMFLOW_SYNC_EXCLUDE_TABLES=%s\n", deploymentVerificationHoldSyncTables())
	text += "ASYMMFLOW_FLUSH_LICENSE_ON_RESEED=false\n"
	if stamp := strings.TrimSpace(os.Getenv("DEPLOY_STAMP")); stamp != "" && strings.EqualFold(strings.TrimSpace(os.Getenv("ASYMMFLOW_FORCE_LICENSE_REACTIVATION")), "true") {
		text += fmt.Sprintf("ASYMMFLOW_LICENSE_FLUSH_STAMP=%s\n", stamp)
	}
	return []byte(text), nil
}

func buildPHWindowsInstaller(packageDir, stamp string) (string, error) {
	if _, err := exec.LookPath("makensis"); err != nil {
		return "", fmt.Errorf("makensis not found: %w", err)
	}

	installerWorkDir := filepath.Join("build", "windows", "installer")
	if _, err := os.Stat(filepath.Join(installerWorkDir, "wails_tools.nsh")); err != nil {
		return "", fmt.Errorf("Wails NSIS support files not found; run wails build -platform windows/amd64 -nsis first: %w", err)
	}

	installerName := fmt.Sprintf("AsymmFlow_PH_Setup_%s.exe", stamp)
	installerPath := filepath.Join(packageDir, installerName)
	nsiPath := filepath.Join(installerWorkDir, "project_ph_delivery.nsi")

	nsisPath := func(path string) (string, error) {
		rel, err := filepath.Rel(installerWorkDir, path)
		if err != nil {
			return "", err
		}
		return strings.ReplaceAll(rel, string(os.PathSeparator), `\`), nil
	}
	requiredFiles := []string{
		filepath.Join(packageDir, ".env"),
		filepath.Join(packageDir, "data", "ph_holdings.db"),
		filepath.Join(packageDir, "data", "Acme Instrumentation Letterhead.png"),
		filepath.Join(packageDir, "data", "Beacon Controls Letterhead.jpg"),
		filepath.Join(packageDir, "LICENSE_KEYS.txt"),
		filepath.Join(packageDir, "INSTALL_GUIDE.txt"),
		filepath.Join(packageDir, "README_START_HERE.txt"),
		filepath.Join(packageDir, "RUN_INSTALLED_ASYMMFLOW_DEBUG.bat"),
		filepath.Join(packageDir, "DEPLOYMENT_SIGNOFF_CHECKLIST_2026_04_08.pdf"),
		filepath.Join(packageDir, "MENTOR_2025_ONEDRIVE_IMPORT_GUIDE_2026_04_10.md"),
		filepath.Join(installerWorkDir, "tmp", "MicrosoftEdgeWebview2Setup.exe"),
	}
	for _, file := range requiredFiles {
		if _, err := os.Stat(file); err != nil {
			return "", fmt.Errorf("installer source missing %s: %w", file, err)
		}
	}

	outFile, err := nsisPath(installerPath)
	if err != nil {
		return "", err
	}
	envFile, err := nsisPath(filepath.Join(packageDir, ".env"))
	if err != nil {
		return "", err
	}
	dbFile, err := nsisPath(filepath.Join(packageDir, "data", "ph_holdings.db"))
	if err != nil {
		return "", err
	}
	phLetterhead, err := nsisPath(filepath.Join(packageDir, "data", "Acme Instrumentation Letterhead.png"))
	if err != nil {
		return "", err
	}
	ahsLetterhead, err := nsisPath(filepath.Join(packageDir, "data", "Beacon Controls Letterhead.jpg"))
	if err != nil {
		return "", err
	}
	licenseKeys, err := nsisPath(filepath.Join(packageDir, "LICENSE_KEYS.txt"))
	if err != nil {
		return "", err
	}
	installGuide, err := nsisPath(filepath.Join(packageDir, "INSTALL_GUIDE.txt"))
	if err != nil {
		return "", err
	}
	startHere, err := nsisPath(filepath.Join(packageDir, "README_START_HERE.txt"))
	if err != nil {
		return "", err
	}
	launchDebugScript, err := nsisPath(filepath.Join(packageDir, "RUN_INSTALLED_ASYMMFLOW_DEBUG.bat"))
	if err != nil {
		return "", err
	}
	signoffPDF, err := nsisPath(filepath.Join(packageDir, "DEPLOYMENT_SIGNOFF_CHECKLIST_2026_04_08.pdf"))
	if err != nil {
		return "", err
	}
	mentorGuide, err := nsisPath(filepath.Join(packageDir, "MENTOR_2025_ONEDRIVE_IMPORT_GUIDE_2026_04_10.md"))
	if err != nil {
		return "", err
	}
	webView2Setup, err := nsisPath(filepath.Join(installerWorkDir, "tmp", "MicrosoftEdgeWebview2Setup.exe"))
	if err != nil {
		return "", err
	}

	script := fmt.Sprintf(`Unicode true

!include "wails_tools.nsh"

VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} PH Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

ManifestDPIAware true

!include "MUI.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXECUTABLE}"
!define MUI_FINISHPAGE_RUN_TEXT "Open AsymmFlow now"
!define MUI_ABORTWARNING

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Name "${INFO_PRODUCTNAME}"
OutFile "%s"
InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
ShowInstDetails show

Function .onInit
   !insertmacro wails.checkArchitecture
FunctionEnd

Section
    !insertmacro wails.setShellContext
    !insertmacro wails.webview2runtime

    SetOutPath $INSTDIR
    !insertmacro wails.files
    File "/oname=.env" "%s"
    File "/oname=LICENSE_KEYS.txt" "%s"
    File "/oname=INSTALL_GUIDE.txt" "%s"
    File "/oname=README_START_HERE.txt" "%s"
    File "/oname=RUN_INSTALLED_ASYMMFLOW_DEBUG.bat" "%s"
    File "/oname=MicrosoftEdgeWebview2Setup.exe" "%s"
    File "/oname=DEPLOYMENT_SIGNOFF_CHECKLIST_2026_04_08.pdf" "%s"
    File "/oname=MENTOR_2025_ONEDRIVE_IMPORT_GUIDE_2026_04_10.md" "%s"

    SetOutPath "$INSTDIR\data"
    File "/oname=ph_holdings.db" "%s"
    File "/oname=Acme Instrumentation Letterhead.png" "%s"
    File "/oname=Beacon Controls Letterhead.jpg" "%s"

    SetOutPath $INSTDIR
    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols
    !insertmacro wails.writeUninstaller
SectionEnd

Section "uninstall"
    !insertmacro wails.setShellContext

    RMDir /r "$AppData\${INFO_PRODUCTNAME}"
    RMDir /r $INSTDIR

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols
    !insertmacro wails.deleteUninstaller
SectionEnd
`, outFile, envFile, licenseKeys, installGuide, startHere, launchDebugScript, webView2Setup, signoffPDF, mentorGuide, dbFile, phLetterhead, ahsLetterhead)

	if err := os.WriteFile(nsiPath, []byte(script), 0644); err != nil {
		return "", err
	}

	cmd := exec.Command("makensis", "-DARG_WAILS_AMD64_BINARY=..\\..\\bin\\AsymmFlow.exe", filepath.Base(nsiPath))
	cmd.Dir = installerWorkDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w\n%s", err, string(output))
	}
	return installerPath, nil
}

func buildStartHereText(stamp string) string {
	return fmt.Sprintf(`ASYMMFLOW WINDOWS INSTALLER - START HERE
=======================================

Use this package for the final training reseed test:

  AsymmFlow_Deploy_%s

Windows install:

  1. Close AsymmFlow.
  2. Run the Windows uninstaller if AsymmFlow is already installed.
  3. Double-click:
       AsymmFlow_PH_Setup_%s.exe
  4. Leave "Open AsymmFlow now" selected on the final installer page.
  5. Enter the Admin or Sales license key from LICENSE_KEYS.txt.

If the app does not open:

  1. Double-click RUN_INSTALLED_ASYMMFLOW_DEBUG.bat.
  2. If Windows asks about WebView2, run MicrosoftEdgeWebview2Setup.exe from this package or from the installed AsymmFlow folder.
  3. Send these two files/outputs to the developer:
       %%TEMP%%\asymmflow_startup.log
       %%APPDATA%%\AsymmFlow\logs\startup.log
       %%APPDATA%%\AsymmFlow\logs\app_debug.log

What this package fixes:

  - It includes a proper Windows installer.
  - It auto-launches AsymmFlow from the installer finish page.
  - It carries a one-time database reseed stamp.
  - If an older user database exists, AsymmFlow backs it up and replaces it with the clean packaged seed on first launch.
  - License activation is preserved during the reseed unless a license reset stamp is explicitly included.
  - Task data, supplier invoices, supplier payments, supplier POs, GRNs, and customer payments are removed from the packaged database.
  - Those verification-pending tables are held out of cloud sync, so Supabase cannot pull them back onto the developer PC during training.

Do not test older FINAL_TRAINING_INSTALLER packages for this issue.
`, stamp, stamp)
}

func buildInstalledLaunchDebugScriptText() string {
	return `@echo off
setlocal
set "APPDIR=%ProgramFiles%\AsymmFlow\AsymmFlow"
if not exist "%APPDIR%\AsymmFlow.exe" set "APPDIR=%ProgramFiles(x86)%\AsymmFlow\AsymmFlow"
if not exist "%APPDIR%\AsymmFlow.exe" set "APPDIR=%~dp0"

echo AsymmFlow launch diagnostic
echo ===========================
echo App directory: %APPDIR%
echo Temp log: %TEMP%\asymmflow_startup.log
echo Console log: %TEMP%\asymmflow_console_output.log
echo App data: %APPDATA%\AsymmFlow
echo.

if not exist "%APPDIR%\AsymmFlow.exe" (
  echo ERROR: AsymmFlow.exe was not found.
  pause
  exit /b 1
)

cd /d "%APPDIR%"
echo Starting AsymmFlow directly so any startup error stays visible...
"%APPDIR%\AsymmFlow.exe" > "%TEMP%\asymmflow_console_output.log" 2>&1
set "EXITCODE=%ERRORLEVEL%"
echo.
echo AsymmFlow process exit code: %EXITCODE%

echo.
echo Console output:
if exist "%TEMP%\asymmflow_console_output.log" (
  type "%TEMP%\asymmflow_console_output.log"
) else (
  echo Not found.
)

echo.
echo Temp startup diagnostic:
if exist "%TEMP%\asymmflow_startup.log" (
  type "%TEMP%\asymmflow_startup.log"
) else (
  echo Not found.
)

echo.
echo AppData startup diagnostic:
if exist "%APPDATA%\AsymmFlow\logs\startup.log" (
  type "%APPDATA%\AsymmFlow\logs\startup.log"
) else (
  echo Not found.
)

echo.
echo App debug log:
if exist "%APPDATA%\AsymmFlow\logs\app_debug.log" (
  type "%APPDATA%\AsymmFlow\logs\app_debug.log"
) else (
  echo Not found.
)

echo.
echo Installed folder:
dir "%APPDIR%" /b

echo.
echo App-data folder:
if exist "%APPDATA%\AsymmFlow" (
  dir "%APPDATA%\AsymmFlow" /b
) else (
  echo Not found.
)

echo.
echo If AsymmFlow still did not open, run MicrosoftEdgeWebview2Setup.exe from the installed folder.
pause
`
}

func buildLicenseKeysText(keys []deploymentEmployeeKey, generatedAt time.Time) string {
	var b strings.Builder
	b.WriteString("===============================================================================\n")
	b.WriteString("                    ASYMMFLOW ERP - LICENSE KEYS\n")
	b.WriteString(fmt.Sprintf("                    Generated: %s\n", generatedAt.Format("2006-01-02")))
	b.WriteString("===============================================================================\n\n")
	b.WriteString("INSTRUCTIONS:\n")
	b.WriteString("  1. Copy the deploy_package folder to your PC\n")
	b.WriteString("  2. Run AsymmFlow.exe (Windows) or open AsymmFlow.app (Mac)\n")
	b.WriteString("  3. Enter YOUR license key when prompted\n")
	b.WriteString("  4. Each key works on ONE PC only - do not share keys\n\n")

	roleTitles := []struct {
		role  string
		title string
		desc  string
	}{
		{"admin", "ADMIN", "Full Access - Dashboard, Finance, Operations, CRM, Intelligence"},
		{"manager", "MANAGER", "Dashboard, Finance, Operations, CRM, Intelligence"},
		{"sales", "SALES", "Dashboard, Opportunities, CRM, Intelligence"},
		{"operations", "OPERATIONS", "Dashboard, Operations Hub, Intelligence"},
		{"staff", "STAFF", "Dashboard, Work, Notifications, Intelligence"},
	}

	for _, rt := range roleTitles {
		roleKeys := make([]deploymentEmployeeKey, 0, len(keys))
		for _, key := range keys {
			if key.Role == rt.role {
				roleKeys = append(roleKeys, key)
			}
		}
		if len(roleKeys) == 0 {
			continue
		}

		b.WriteString("-------------------------------------------------------------------------------\n")
		b.WriteString(fmt.Sprintf("%s (%s)\n", rt.title, rt.desc))
		b.WriteString("-------------------------------------------------------------------------------\n\n")
		for _, key := range roleKeys {
			b.WriteString(fmt.Sprintf("  %-16s %s\n", key.DisplayName, key.Key))
		}
		b.WriteString("\n")
	}

	b.WriteString("===============================================================================\n")
	b.WriteString("                    ROLE ACCESS MATRIX\n")
	b.WriteString("===============================================================================\n\n")
	b.WriteString("  Screen            Admin   Manager   Sales   Operations   Staff\n")
	b.WriteString("  -------           -----   -------   -----   ----------   -----\n")
	b.WriteString("  Dashboard           Y       Y         Y        Y           Y\n")
	b.WriteString("  Opportunities       Y       Y         Y        -           -\n")
	b.WriteString("  Operations          Y       Y         -        Y           -\n")
	b.WriteString("  Finance Hub         Y       Y         -        -           -\n")
	b.WriteString("  Work                Y       Y         Y        Y           Y\n")
	b.WriteString("  Notifications       Y       Y         Y        Y           Y\n")
	b.WriteString("  Relationships       Y       Y         Y        -           -\n")
	b.WriteString("  Intelligence        Y       Y         Y        Y           Y\n\n")
	b.WriteString("===============================================================================\n")
	b.WriteString("  SUPPORT: Contact your SPOC for key issues or new key requests\n")
	b.WriteString("  CLOUD SYNC: Auto-syncs approved tables every 10 minutes when online\n")
	b.WriteString("===============================================================================\n")
	return b.String()
}

func buildLicenseKeysPlaceholderText(generatedAt time.Time) string {
	return fmt.Sprintf(`===============================================================================
                    ASYMMFLOW ERP - LICENSE KEYS
                    Generated: %s
===============================================================================

This tracked top-level file is intentionally a placeholder.

The employee license keys are generated only inside the timestamped deployment
package produced by TestPrepareDeploymentPackage:

  deploy_package/AsymmFlow_Deploy_<stamp>/LICENSE_KEYS.txt

Do not commit generated employee keys, packaged .env files, packaged databases,
or historical deployment package snapshots back to source control.
===============================================================================
`, generatedAt.Format("2006-01-02"))
}

func buildInstallGuideText(generatedAt time.Time) string {
	return fmt.Sprintf(`===============================================================================
                ASYMMFLOW ERP - INSTALLATION GUIDE
                Version: %s | Deployment Package
===============================================================================

QUICK START (3 Steps)
---------------------

  1. WINDOWS: run the installer if present
     - Double-click AsymmFlow_PH_Setup_*.exe
     - Finish the installer, then open AsymmFlow from the Desktop or Start Menu
     - Keep LICENSE_KEYS.txt available for the first activation prompt

  2. PORTABLE / MAC: copy this entire folder to your PC as-is
     - Windows: C:\AsymmFlow\
     - Mac: ~/AsymmFlow/
     - Do NOT move AsymmFlow.exe or AsymmFlow.app out of this folder
     - Do NOT separate the app from the data/ folder

  3. RUN the application if you did not use the Windows installer
     - Windows: Double-click AsymmFlow.exe
     - Mac: Open AsymmFlow.app

  4. ENTER your license key when prompted
     - See LICENSE_KEYS.txt for your personal employee key
     - Key activates on first use and binds to your PC

That's it! The app will open to the Dashboard.


IMPORTANT DATABASE RULE
-----------------------

  - The packaged data/ph_holdings.db is a first-run seed database.
  - On first launch, AsymmFlow copies it to the machine app-data folder and uses that persistent copy.
  - This full training installer carries a one-time reseed stamp. If an older app-data database exists, it is backed up and replaced from the clean packaged seed.
  - License activation is preserved during that reseed unless the package explicitly includes a license reset stamp.
  - Pending-verification workflow/procurement/payment tables are held out of cloud sync until SPOC sign-off.
  - On normal future upgrades, replace the app/package; do not delete the app-data folder unless you intend to reset local data.


ADMIN DATA SAFETY
-----------------

  - Only Admin keys can complete record deletion.
  - Sales, Operations, Staff, and Manager delete attempts are converted into admin approval requests.
  - Admin users review delete requests from the Notifications screen and can Approve Delete or Reject.
  - Admin users can create a database backup from Settings > Supabase Sync > Database Backups.
  - Auto Backup is enabled by default on a weekly schedule; Admin can change the frequency from 1 to 30 days.
  - Backup files are stored next to the app-data database in the backups/ folder and the last 7 are retained.


FOLDER STRUCTURE
----------------

  deploy_package/
  ├── AsymmFlow_PH_Setup_*.exe
  │                         <- Windows installer, use this first on Windows
  ├── AsymmFlow.app          <- The application (Mac)
  ├── AsymmFlow.exe          <- The application (Windows)
  ├── .env                   <- Cloud sync configuration
  ├── data/
  │   ├── ph_holdings.db     <- Database with all company data
  │   └── Acme Instrumentation Letterhead.png
  ├── LICENSE_KEYS.txt       <- Named employee license keys
  ├── DEPLOYMENT_SIGNOFF_CHECKLIST_2026_04_08.md
  ├── DEPLOYMENT_SIGNOFF_CHECKLIST_2026_04_08.pdf
  ├── MENTOR_2025_ONEDRIVE_IMPORT_GUIDE_2026_04_10.md
  └── INSTALL_GUIDE.txt      <- This file


WINDOWS REQUIREMENTS
--------------------

  - WebView2 Runtime: Pre-installed on Windows 10 21H2+ and Windows 11
  - No other dependencies needed


CLOUD SYNC
----------

  - Data syncs automatically every 10 minutes
  - Green dot = Connected | Yellow = Connecting | Gray = Offline
  - App works fully offline - changes sync when reconnected
  - All PCs share the same data through Supabase cloud


TROUBLESHOOTING
---------------

  Problem                     Solution
  -------                     --------
  App won't start (Windows)   Install WebView2 Runtime
  App won't start             Make sure .env file is in same folder
  "Invalid license key"       Check LICENSE_KEYS.txt for correct key
  "Already activated"         Each key works on ONE PC only
  Database locked             Close other app instances
  Data not syncing            Check internet connection, wait 10 min
  Empty dashboard             Make sure data/ph_holdings.db exists for first launch
  Wrong or old database       Confirm the app-data database is the intended live database
  Wrong or old database       Replace the whole package folder, not just the app binary

  For all issues: Contact your SPOC
===============================================================================
  Acme Instrumentation W.L.L | Powered by Asymmetrica Mathematical Organism
===============================================================================
`, generatedAt.Format("2006-01-02"))
}

func resetDeploymentLicensePool(app *App, db *gorm.DB) error {
	if app == nil || db == nil {
		return fmt.Errorf("deployment license pool requires initialized app/db")
	}
	if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&LicenseKey{}).Error; err != nil {
		return err
	}
	return ensureDeploymentEmployeeKeys(db)
}

func ensureDeploymentEmployeeKeys(db *gorm.DB) error {
	for _, candidate := range deploymentEmployeeKeySpecs() {
		canonicalKey := strings.ToUpper(strings.TrimSpace(candidate.Key))
		if canonicalKey != "" {
			var byKey LicenseKey
			if err := db.Where("key = ?", canonicalKey).First(&byKey).Error; err == nil {
				if err := db.Model(&byKey).Updates(map[string]any{
					"role":         candidate.Role,
					"display_name": candidate.DisplayName,
					"activated":    false,
					"activated_at": nil,
					"device_hash":  "",
					"notes":        candidate.Notes,
					"created_by":   "deployment-package",
				}).Error; err != nil {
					return err
				}
				continue
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			var byName LicenseKey
			if err := db.Where("role = ? AND display_name = ?", candidate.Role, candidate.DisplayName).
				Order("id ASC").
				First(&byName).Error; err == nil {
				if err := db.Model(&byName).Updates(map[string]any{
					"key":          canonicalKey,
					"activated":    false,
					"activated_at": nil,
					"device_hash":  "",
					"notes":        candidate.Notes,
					"created_by":   "deployment-package",
				}).Error; err != nil {
					return err
				}
				continue
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			record := LicenseKey{
				Key:         canonicalKey,
				Role:        candidate.Role,
				DisplayName: candidate.DisplayName,
				Activated:   false,
				Notes:       candidate.Notes,
				CreatedBy:   "deployment-package",
			}
			if err := db.Create(&record).Error; err != nil {
				return err
			}
			continue
		}

		var existing LicenseKey
		if err := db.Where("role = ? AND display_name = ?", candidate.Role, candidate.DisplayName).First(&existing).Error; err == nil {
			if err := db.Model(&existing).Updates(map[string]any{
				"activated":    false,
				"activated_at": nil,
				"device_hash":  "",
				"notes":        candidate.Notes,
				"created_by":   "deployment-package",
			}).Error; err != nil {
				return err
			}
			continue
		}

		prefix, ok := rolePrefixes[candidate.Role]
		if !ok {
			return fmt.Errorf("no license prefix configured for role %s", candidate.Role)
		}
		randomBytes := make([]byte, 3)
		if _, err := rand.Read(randomBytes); err != nil {
			return err
		}
		record := LicenseKey{
			Key:         fmt.Sprintf("PH-%s-%s", prefix, strings.ToUpper(hex.EncodeToString(randomBytes))),
			Role:        candidate.Role,
			DisplayName: candidate.DisplayName,
			Activated:   false,
			Notes:       candidate.Notes,
			CreatedBy:   "deployment-package",
		}
		if err := db.Create(&record).Error; err != nil {
			return err
		}
	}

	return nil
}

func deploymentEmployeeKeySpecs() []deploymentRoleKeySpec {
	source := phTradingNamedLicenseSpecs()
	specs := make([]deploymentRoleKeySpec, 0, len(source))
	for _, item := range source {
		specs = append(specs, deploymentRoleKeySpec{
			Role:        item.Role,
			DisplayName: item.DisplayName,
			Key:         item.Key,
			Notes:       item.Notes,
		})
	}
	return specs
}
