package main

// Wave 8 Bucket D — sales/procurement ports: CreatePOsFromOrder (per-supplier
// split), PreviewOrderDeleteCascade (+ TOCTOU-safe DeleteOrder refactor),
// tender-folder preview/import, GetPreparedByOptions — plus the P5-2 decision:
// MarkOfferWon no longer auto-creates a junk EUR/0-rate Draft PO.

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCreatePOsFromOrder_SplitsPerSupplier(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&ProductMaster{}, &Order{}, &OrderItem{}, &PurchaseOrder{}, &PurchaseOrderItem{}, &InvoiceSequence{}))

	beta := SupplierMaster{SupplierCode: "BETA", SupplierName: "Beta Flow Systems"}
	oxan := SupplierMaster{SupplierCode: "SRVX", SupplierName: "Oxan Analytics"}
	require.NoError(t, a.db.Create(&beta).Error)
	require.NoError(t, a.db.Create(&oxan).Error)

	analyzer := ProductMaster{ProductCode: "SVX-2200", ProductName: "Oxan 2200", SupplierID: oxan.ID, StandardCostBHD: 100}
	meter := ProductMaster{ProductCode: "BETA-FM-1", ProductName: "Beta Flow Meter", SupplierID: beta.ID, StandardCostBHD: 50}
	require.NoError(t, a.db.Create(&analyzer).Error)
	require.NoError(t, a.db.Create(&meter).Error)

	order := Order{Base: Base{ID: "pos-o1"}, OrderNumber: "ORD-26-6001", CustomerName: "Nimbus Controls", Status: "Confirmed"}
	require.NoError(t, a.db.Create(&order).Error)
	items := []OrderItem{
		{Base: Base{ID: "pos-i1"}, OrderID: "pos-o1", ProductID: analyzer.ID, ProductCode: "SVX-2200", Description: "O2 analyzer", Quantity: 2, UnitPrice: 300},
		{Base: Base{ID: "pos-i2"}, OrderID: "pos-o1", ProductID: analyzer.ID, ProductCode: "SVX-2200", Description: "O2 analyzer spare", Quantity: 1, UnitPrice: 300},
		{Base: Base{ID: "pos-i3"}, OrderID: "pos-o1", ProductID: meter.ID, ProductCode: "BETA-FM-1", Description: "Flow meter", Quantity: 3, UnitPrice: 150},
	}
	require.NoError(t, a.db.Create(&items).Error)

	created, err := a.CreatePOsFromOrder("pos-o1", nil)
	require.NoError(t, err)
	require.Len(t, created, 2, "one PO per inferred supplier")

	// Deterministic supplier-name ordering: Beta before Oxan.
	require.Equal(t, beta.ID, created[0].SupplierID)
	require.Len(t, created[0].Items, 1)
	require.Equal(t, oxan.ID, created[1].SupplierID)
	require.Len(t, created[1].Items, 2)
	for _, po := range created {
		require.Equal(t, "Draft", po.Status)
		require.Equal(t, "pos-o1", po.OrderID)
	}
}

func TestCreatePOsFromOrder_UnresolvableItemErrors(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&ProductMaster{}, &Order{}, &OrderItem{}))

	order := Order{Base: Base{ID: "pos-o2"}, OrderNumber: "ORD-26-6002", CustomerName: "Atlas Traders", Status: "Confirmed"}
	require.NoError(t, a.db.Create(&order).Error)
	require.NoError(t, a.db.Create(&OrderItem{
		Base: Base{ID: "pos-i9"}, OrderID: "pos-o2", Description: "Mystery part with no supplier signal", Quantity: 1,
	}).Error)

	_, err := a.CreatePOsFromOrder("pos-o2", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Unable to determine supplier")
}

func TestMarkOfferWon_NoAutoDraftPO(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&Offer{}, &OfferItem{}, &Order{}, &OrderItem{}, &PurchaseOrder{}, &PurchaseOrderItem{}, &RFQData{}, &Opportunity{}))

	offer := Offer{
		Base: Base{ID: "won-of1"}, OfferNumber: "OFF-26-001", CustomerID: "cust-1",
		CustomerName: "Nimbus Controls", Stage: "Quoted", TotalValueBHD: 750,
	}
	require.NoError(t, a.db.Create(&offer).Error)
	require.NoError(t, a.db.Create(&OfferItem{
		Base: Base{ID: "won-oi1"}, OfferID: "won-of1", LineNumber: 1,
		Description: "Analyzer", Quantity: 1, UnitPrice: 750,
	}).Error)

	order, err := a.MarkOfferWon("won-of1", "CPO-9001")
	require.NoError(t, err)
	require.NotNil(t, order)
	require.Equal(t, "CPO-9001", order.CustomerPONumber)

	// P5-2: winning an offer creates the ORDER only. PO creation is the
	// deliberate CreatePOsFromOrder flow — no junk EUR/0-rate draft.
	var poCount int64
	require.NoError(t, a.db.Model(&PurchaseOrder{}).Count(&poCount).Error)
	require.Zero(t, poCount, "MarkOfferWon must not auto-create a purchase order")
}

func TestPreviewOrderDeleteCascade_BlocksOnPayments(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&Order{}, &OrderItem{}, &Invoice{}, &DBInvoiceItem{}, &Payment{}, &PurchaseOrder{}, &PurchaseOrderItem{}, &DeliveryNote{}, &DeliveryNoteItem{}))

	require.NoError(t, a.db.Create(&Order{Base: Base{ID: "del-o1"}, OrderNumber: "ORD-26-7001", CustomerName: "Alpha Trading Co", Status: "Confirmed"}).Error)
	require.NoError(t, a.db.Create(&OrderItem{Base: Base{ID: "del-i1"}, OrderID: "del-o1", Description: "Analyzer", Quantity: 1}).Error)
	require.NoError(t, a.db.Create(&Invoice{
		Base: Base{ID: "del-inv1"}, InvoiceNumber: "DEL-INV-1", OrderID: "del-o1", CustomerID: "c1",
		InvoiceDate: time.Now(), Status: "Sent", GrandTotalBHD: 100, OutstandingBHD: 50,
	}).Error)
	require.NoError(t, a.db.Create(&Payment{
		Base: Base{ID: "del-p1"}, InvoiceID: "del-inv1", AmountBHD: 50,
		PaymentDate: time.Now(), PaymentMethod: "Cash", IdempotencyKey: "del-pay-1",
	}).Error)

	preview, err := a.PreviewOrderDeleteCascade("del-o1")
	require.NoError(t, err)
	require.Equal(t, true, preview["blocked"])
	require.Equal(t, 1, preview["invoice_count"])
	require.EqualValues(t, 1, preview["payment_count"])
	require.EqualValues(t, 1, preview["order_item_count"])
	require.Contains(t, preview["block_reason"], "payment")

	// The delete itself refuses while payments exist.
	err = a.DeleteOrder("del-o1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "payment")

	// Unknown order errors cleanly.
	_, err = a.PreviewOrderDeleteCascade("missing")
	require.ErrorContains(t, err, "Order does not exist")
}

func TestDeleteOrder_CascadesLinkedRecords(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&Order{}, &OrderItem{}, &Invoice{}, &DBInvoiceItem{}, &Payment{}, &PurchaseOrder{}, &PurchaseOrderItem{}, &DeliveryNote{}, &DeliveryNoteItem{}))

	require.NoError(t, a.db.Create(&Order{Base: Base{ID: "del-o2"}, OrderNumber: "ORD-26-7002", CustomerName: "Beta Industrial", Status: "Confirmed"}).Error)
	require.NoError(t, a.db.Create(&OrderItem{Base: Base{ID: "del2-i1"}, OrderID: "del-o2", Description: "Meter", Quantity: 1}).Error)
	// Legacy-linked invoice: order_id carries the ORDER NUMBER, not the ID —
	// the widened link condition must still find (and delete) it.
	require.NoError(t, a.db.Create(&Invoice{
		Base: Base{ID: "del2-inv1"}, InvoiceNumber: "DEL2-INV-1", OrderID: "ORD-26-7002", CustomerID: "c2",
		InvoiceDate: time.Now(), Status: "Sent", GrandTotalBHD: 100, OutstandingBHD: 100,
	}).Error)
	require.NoError(t, a.db.Create(&PurchaseOrder{Base: Base{ID: "del2-po1"}, OrderID: "del-o2", PONumber: "DEL2-PO-1", Status: "Draft"}).Error)
	require.NoError(t, a.db.Create(&DeliveryNote{Base: Base{ID: "del2-dn1"}, OrderID: "del-o2", DNNumber: "DEL2-DN-1"}).Error)

	require.NoError(t, a.DeleteOrder("del-o2"))

	var orderCount, invoiceCount, poCount, dnCount int64
	a.db.Model(&Order{}).Where("id = ?", "del-o2").Count(&orderCount)
	a.db.Model(&Invoice{}).Where("id = ?", "del2-inv1").Count(&invoiceCount)
	a.db.Model(&PurchaseOrder{}).Where("id = ?", "del2-po1").Count(&poCount)
	a.db.Model(&DeliveryNote{}).Where("id = ?", "del2-dn1").Count(&dnCount)
	require.Zero(t, orderCount)
	require.Zero(t, invoiceCount, "number-linked legacy invoice must cascade too")
	require.Zero(t, poCount)
	require.Zero(t, dnCount)
}

func TestTenderFolders_PreviewAndImport(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&RFQData{}, &Opportunity{}))

	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "12 Substation upgrade"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(root, "13.1 Water plant expansion"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(root, "no leading number"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "14 not a directory.txt"), []byte("x"), 0o644))

	// T-12 already exists as an RFQ — must flag Existing and be skipped on import.
	require.NoError(t, a.db.Create(&RFQData{RFQNumber: "T-12", Client: "Tender", Project: "Substation upgrade", Status: "Tender", Stage: "RFQ Received"}).Error)

	previews, err := a.PreviewTenderFolders(root)
	require.NoError(t, err)
	require.Len(t, previews, 2, "non-matching folder names and plain files are ignored")
	require.Equal(t, "T-12", previews[0].WorkflowKey)
	require.True(t, previews[0].Existing)
	require.Equal(t, "T-13", previews[1].WorkflowKey)
	require.Equal(t, "Water plant expansion", previews[1].Title)
	require.False(t, previews[1].Existing)

	imported, err := a.ImportTenderFolders(root)
	require.NoError(t, err)
	require.Len(t, imported, 1)
	require.Equal(t, "T-13", imported[0].WorkflowKey)

	var rfq RFQData
	require.NoError(t, a.db.First(&rfq, "rfq_number = ?", "T-13").Error)
	require.Equal(t, "Water plant expansion", rfq.Project)
	require.Equal(t, "Tender", rfq.Status)
	require.Equal(t, "RFQ Received", rfq.Stage)

	// Idempotent: a second import finds everything existing.
	imported, err = a.ImportTenderFolders(root)
	require.NoError(t, err)
	require.Empty(t, imported)

	// Blank path rejected.
	_, err = a.PreviewTenderFolders("   ")
	require.Error(t, err)
}

func TestGetPreparedByOptions_SyntheticSources(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&Employee{}, &LicenseKey{}))

	require.NoError(t, a.db.Create(&Employee{
		Base: Base{ID: "emp-1"}, EmployeeCode: "E-001", FullName: "Jordan Smith", PreferredName: "Jordan",
	}).Error)
	require.NoError(t, a.db.Create(&LicenseKey{
		Key: "XX-ADM-TEST01", Role: "admin", DisplayName: "Casey",
	}).Error)

	options, err := a.GetPreparedByOptions()
	require.NoError(t, err)
	require.Contains(t, options, "Jordan Smith")
	require.Contains(t, options, "Jordan")
	require.Contains(t, options, "Casey")
	for _, name := range defaultOfferSignatureNames() {
		require.Contains(t, options, name, "overlay signature identities must seed the picker")
	}
	for _, opt := range options {
		require.NotEmpty(t, opt)
	}
	// No hardcoded real-people names: the seed list is exactly the overlay
	// blocks — anything else must come from the DB rows created above.
	seeded := map[string]bool{"Jordan Smith": true, "Jordan": true, "Casey": true}
	for _, name := range defaultOfferSignatureNames() {
		seeded[name] = true
	}
	for _, opt := range options {
		require.True(t, seeded[opt], "unexpected hardcoded prepared-by option %q", opt)
	}
}
