// Package sync converts synchronization models to and from generated Proto messages.
package sync

import (
	"strings"
	"time"

	"ph_holdings_app/pkg/adapter"
	gormdocuments "ph_holdings_app/pkg/documents"
	gormsync "ph_holdings_app/pkg/sync"
	commonproto "ph_holdings_app/schemas/go/common"
	protosync "ph_holdings_app/schemas/go/syncschema"

	capnp "capnproto.org/go/capnp/v3"
	"gorm.io/gorm"
)

func newMessage() (*capnp.Message, *capnp.Segment, error) {
	return capnp.NewMessage(capnp.SingleSegment(nil))
}

func text(v string, err error) (string, error) {
	if err != nil {
		return "", err
	}
	return v, nil
}

func syncBaseToProto(seg *capnp.Segment, base gormsync.Base) (commonproto.Base, error) {
	return baseToProto(seg, base.ID, base.CreatedAt, base.UpdatedAt, base.DeletedAt, base.CreatedBy)
}

func documentBaseToProto(seg *capnp.Segment, base gormdocuments.Base) (commonproto.Base, error) {
	return baseToProto(seg, base.ID, base.CreatedAt, base.UpdatedAt, base.DeletedAt, base.CreatedBy)
}

func baseToProto(seg *capnp.Segment, id string, createdAt time.Time, updatedAt time.Time, deletedAt gorm.DeletedAt, createdBy string) (commonproto.Base, error) {
	p, err := commonproto.NewBase(seg)
	if err != nil {
		return commonproto.Base{}, err
	}
	if err := p.SetId(id); err != nil {
		return commonproto.Base{}, err
	}
	if err := p.SetCreatedAt(adapter.TimeToText(createdAt)); err != nil {
		return commonproto.Base{}, err
	}
	if err := p.SetUpdatedAt(adapter.TimeToText(updatedAt)); err != nil {
		return commonproto.Base{}, err
	}
	if err := p.SetDeletedAt(adapter.DeletedAtToText(deletedAt)); err != nil {
		return commonproto.Base{}, err
	}
	if err := p.SetCreatedBy(createdBy); err != nil {
		return commonproto.Base{}, err
	}
	if err := p.SetUpdatedBy(""); err != nil {
		return commonproto.Base{}, err
	}
	p.SetStatus(commonproto.RecordStatus_active)
	p.SetSyncState(commonproto.SyncState_synced)
	return p, nil
}

func syncBaseFromProto(p commonproto.Base) (gormsync.Base, error) {
	id, err := text(p.Id())
	if err != nil {
		return gormsync.Base{}, err
	}
	createdAt, err := text(p.CreatedAt())
	if err != nil {
		return gormsync.Base{}, err
	}
	updatedAt, err := text(p.UpdatedAt())
	if err != nil {
		return gormsync.Base{}, err
	}
	deletedAt, err := text(p.DeletedAt())
	if err != nil {
		return gormsync.Base{}, err
	}
	createdBy, err := text(p.CreatedBy())
	if err != nil {
		return gormsync.Base{}, err
	}
	return gormsync.Base{
		ID:        id,
		CreatedAt: adapter.TextToTime(createdAt),
		UpdatedAt: adapter.TextToTime(updatedAt),
		DeletedAt: adapter.TextToDeletedAt(deletedAt),
		CreatedBy: createdBy,
	}, nil
}

func documentBaseFromProto(p commonproto.Base) (gormdocuments.Base, error) {
	base, err := syncBaseFromProto(p)
	if err != nil {
		return gormdocuments.Base{}, err
	}
	return gormdocuments.Base{
		ID:        base.ID,
		CreatedAt: base.CreatedAt,
		UpdatedAt: base.UpdatedAt,
		DeletedAt: base.DeletedAt,
		CreatedBy: base.CreatedBy,
	}, nil
}

func normalized(v string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(v), " ", ""))
}

func fileSyncStatus(status string) protosync.FileSyncStatus {
	switch normalized(status) {
	case "processing", "inprogress":
		return protosync.FileSyncStatus_processing
	case "synced", "success", "complete", "completed":
		return protosync.FileSyncStatus_synced
	case "failed", "error":
		return protosync.FileSyncStatus_failed
	case "conflict":
		return protosync.FileSyncStatus_conflict
	case "skippedlarge", "skipped_large":
		return protosync.FileSyncStatus_skippedLarge
	default:
		return protosync.FileSyncStatus_queued
	}
}

func fileSyncStatusText(status protosync.FileSyncStatus) string {
	switch status {
	case protosync.FileSyncStatus_processing:
		return "processing"
	case protosync.FileSyncStatus_synced:
		return "synced"
	case protosync.FileSyncStatus_failed:
		return "failed"
	case protosync.FileSyncStatus_conflict:
		return "conflict"
	case protosync.FileSyncStatus_skippedLarge:
		return "skipped_large"
	default:
		return "queued"
	}
}

func syncDirection(direction string) protosync.SyncDirection {
	if normalized(direction) == "pull" {
		return protosync.SyncDirection_pull
	}
	return protosync.SyncDirection_push
}

func conflictState(state string) protosync.ConflictState {
	switch normalized(state) {
	case "localwins", "local_wins":
		return protosync.ConflictState_localWins
	case "remotewins", "remote_wins":
		return protosync.ConflictState_remoteWins
	default:
		return protosync.ConflictState_none
	}
}

func conflictStateText(state protosync.ConflictState) string {
	switch state {
	case protosync.ConflictState_localWins:
		return "local_wins"
	case protosync.ConflictState_remoteWins:
		return "remote_wins"
	default:
		return "none"
	}
}

func tallyStatus(status string) protosync.TallyImportStatus {
	switch normalized(status) {
	case "imported":
		return protosync.TallyImportStatus_imported
	case "matched":
		return protosync.TallyImportStatus_matched
	case "duplicate":
		return protosync.TallyImportStatus_duplicate
	case "error", "failed":
		return protosync.TallyImportStatus_error
	default:
		return protosync.TallyImportStatus_pending
	}
}

func currencyCode(currency string) commonproto.CurrencyCode {
	return commonproto.CurrencyCodeFromString(strings.ToLower(strings.TrimSpace(currency)))
}

func currencyText(currency commonproto.CurrencyCode) string {
	return strings.ToUpper(currency.String())
}

// FileWatchEventToProto converts a document watcher event to the sync Proto event.
func FileWatchEventToProto(event gormdocuments.FileWatchEvent) (*protosync.FileWatchEvent, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protosync.NewRootFileWatchEvent(seg)
	if err != nil {
		return nil, err
	}
	base, err := documentBaseToProto(seg, event.Base)
	if err != nil {
		return nil, err
	}
	if err := p.SetBase(base); err != nil {
		return nil, err
	}
	if err := p.SetFilePath(event.FilePath); err != nil {
		return nil, err
	}
	if err := p.SetEventType(event.EventType); err != nil {
		return nil, err
	}
	return &p, nil
}

// FileWatchEventFromProto converts the sync Proto event back to a document watcher event.
func FileWatchEventFromProto(p protosync.FileWatchEvent) (gormdocuments.FileWatchEvent, error) {
	base, err := p.Base()
	if err != nil {
		return gormdocuments.FileWatchEvent{}, err
	}
	modelBase, err := documentBaseFromProto(base)
	if err != nil {
		return gormdocuments.FileWatchEvent{}, err
	}
	filePath, err := text(p.FilePath())
	if err != nil {
		return gormdocuments.FileWatchEvent{}, err
	}
	eventType, err := text(p.EventType())
	if err != nil {
		return gormdocuments.FileWatchEvent{}, err
	}
	return gormdocuments.FileWatchEvent{
		Base:      modelBase,
		FilePath:  filePath,
		EventType: eventType,
	}, nil
}

// SyncStatusToProto converts a GORM SyncStatus to a Proto SyncStatus.
func SyncStatusToProto(status gormsync.SyncStatus) (*protosync.SyncStatus, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protosync.NewRootSyncStatus(seg)
	if err != nil {
		return nil, err
	}
	base, err := syncBaseToProto(seg, status.Base)
	if err != nil {
		return nil, err
	}
	if err := p.SetBase(base); err != nil {
		return nil, err
	}
	if err := p.SetFilePath(status.FilePath); err != nil {
		return nil, err
	}
	p.SetStatus(fileSyncStatus(status.Status))
	if err := p.SetLastSyncTime(adapter.TimeToText(status.LastSyncTime)); err != nil {
		return nil, err
	}
	return &p, nil
}

// SyncStatusFromProto converts a Proto SyncStatus back to a GORM SyncStatus.
func SyncStatusFromProto(p protosync.SyncStatus) (gormsync.SyncStatus, error) {
	base, err := p.Base()
	if err != nil {
		return gormsync.SyncStatus{}, err
	}
	modelBase, err := syncBaseFromProto(base)
	if err != nil {
		return gormsync.SyncStatus{}, err
	}
	filePath, err := text(p.FilePath())
	if err != nil {
		return gormsync.SyncStatus{}, err
	}
	lastSyncTime, err := text(p.LastSyncTime())
	if err != nil {
		return gormsync.SyncStatus{}, err
	}
	return gormsync.SyncStatus{
		Base:         modelBase,
		FilePath:     filePath,
		Status:       fileSyncStatusText(p.Status()),
		LastSyncTime: adapter.TextToTime(lastSyncTime),
	}, nil
}

// SyncRecordToProto converts a GORM SyncRecord to a Proto SyncRecord.
func SyncRecordToProto(record gormsync.SyncRecord) (*protosync.SyncRecord, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protosync.NewRootSyncRecord(seg)
	if err != nil {
		return nil, err
	}
	base, err := syncBaseToProto(seg, record.Base)
	if err != nil {
		return nil, err
	}
	if err := p.SetBase(base); err != nil {
		return nil, err
	}
	if err := p.SetSyncTable(record.SyncTable); err != nil {
		return nil, err
	}
	if err := p.SetRecordId(record.RecordID); err != nil {
		return nil, err
	}
	if err := p.SetSyncedAt(adapter.TimeToText(record.SyncedAt)); err != nil {
		return nil, err
	}
	p.SetDirection(syncDirection(record.Direction))
	p.SetRemoteVersion(adapter.IntToInt64(record.RemoteVersion))
	p.SetLocalVersion(adapter.IntToInt64(record.LocalVersion))
	p.SetConflictState(conflictState(record.ConflictState))
	return &p, nil
}

// SyncRecordFromProto converts a Proto SyncRecord back to a GORM SyncRecord.
func SyncRecordFromProto(p protosync.SyncRecord) (gormsync.SyncRecord, error) {
	base, err := p.Base()
	if err != nil {
		return gormsync.SyncRecord{}, err
	}
	modelBase, err := syncBaseFromProto(base)
	if err != nil {
		return gormsync.SyncRecord{}, err
	}
	syncTable, err := text(p.SyncTable())
	if err != nil {
		return gormsync.SyncRecord{}, err
	}
	recordID, err := text(p.RecordId())
	if err != nil {
		return gormsync.SyncRecord{}, err
	}
	syncedAt, err := text(p.SyncedAt())
	if err != nil {
		return gormsync.SyncRecord{}, err
	}
	return gormsync.SyncRecord{
		Base:          modelBase,
		SyncTable:     syncTable,
		RecordID:      recordID,
		SyncedAt:      adapter.TextToTime(syncedAt),
		Direction:     p.Direction().String(),
		RemoteVersion: adapter.Int64ToInt(p.RemoteVersion()),
		LocalVersion:  adapter.Int64ToInt(p.LocalVersion()),
		ConflictState: conflictStateText(p.ConflictState()),
	}, nil
}

// TallyInvoiceImportToProto converts a GORM TallyInvoiceImport to Proto.
func TallyInvoiceImportToProto(row gormsync.TallyInvoiceImport) (*protosync.TallyInvoiceImport, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protosync.NewRootTallyInvoiceImport(seg)
	if err != nil {
		return nil, err
	}
	base, err := syncBaseToProto(seg, row.Base)
	if err != nil {
		return nil, err
	}
	if err := p.SetBase(base); err != nil {
		return nil, err
	}
	if err := p.SetImportBatch(row.ImportBatch); err != nil {
		return nil, err
	}
	p.SetYear(adapter.IntToInt64(row.Year))
	if err := p.SetInvoiceNumber(row.InvoiceNumber); err != nil {
		return nil, err
	}
	if err := p.SetCustomerName(row.CustomerName); err != nil {
		return nil, err
	}
	if err := p.SetMatchedCustomerId(row.MatchedCustomerID); err != nil {
		return nil, err
	}
	if err := p.SetInvoiceDate(adapter.TimeToText(row.InvoiceDate)); err != nil {
		return nil, err
	}
	p.SetAmount(row.Amount)
	p.SetCurrency(currencyCode(row.Currency))
	p.SetStatus(tallyStatus(row.Status))
	if err := p.SetRawData(row.RawData); err != nil {
		return nil, err
	}
	return &p, nil
}

// TallyInvoiceImportFromProto converts a Proto TallyInvoiceImport back to GORM.
func TallyInvoiceImportFromProto(p protosync.TallyInvoiceImport) (gormsync.TallyInvoiceImport, error) {
	base, err := p.Base()
	if err != nil {
		return gormsync.TallyInvoiceImport{}, err
	}
	modelBase, err := syncBaseFromProto(base)
	if err != nil {
		return gormsync.TallyInvoiceImport{}, err
	}
	importBatch, err := text(p.ImportBatch())
	if err != nil {
		return gormsync.TallyInvoiceImport{}, err
	}
	invoiceNumber, err := text(p.InvoiceNumber())
	if err != nil {
		return gormsync.TallyInvoiceImport{}, err
	}
	customerName, err := text(p.CustomerName())
	if err != nil {
		return gormsync.TallyInvoiceImport{}, err
	}
	matchedCustomerID, err := text(p.MatchedCustomerId())
	if err != nil {
		return gormsync.TallyInvoiceImport{}, err
	}
	invoiceDate, err := text(p.InvoiceDate())
	if err != nil {
		return gormsync.TallyInvoiceImport{}, err
	}
	rawData, err := text(p.RawData())
	if err != nil {
		return gormsync.TallyInvoiceImport{}, err
	}
	return gormsync.TallyInvoiceImport{
		Base:              modelBase,
		ImportBatch:       importBatch,
		Year:              adapter.Int64ToInt(p.Year()),
		InvoiceNumber:     invoiceNumber,
		CustomerName:      customerName,
		MatchedCustomerID: matchedCustomerID,
		InvoiceDate:       adapter.TextToTime(invoiceDate),
		Amount:            p.Amount(),
		Currency:          currencyText(p.Currency()),
		Status:            p.Status().String(),
		RawData:           rawData,
	}, nil
}

// TallyPurchaseImportToProto converts a GORM TallyPurchaseImport to Proto.
func TallyPurchaseImportToProto(row gormsync.TallyPurchaseImport) (*protosync.TallyPurchaseImport, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protosync.NewRootTallyPurchaseImport(seg)
	if err != nil {
		return nil, err
	}
	base, err := syncBaseToProto(seg, row.Base)
	if err != nil {
		return nil, err
	}
	if err := p.SetBase(base); err != nil {
		return nil, err
	}
	if err := p.SetImportBatch(row.ImportBatch); err != nil {
		return nil, err
	}
	p.SetYear(adapter.IntToInt64(row.Year))
	if err := p.SetInvoiceNumber(row.InvoiceNumber); err != nil {
		return nil, err
	}
	if err := p.SetSupplierName(row.SupplierName); err != nil {
		return nil, err
	}
	if err := p.SetMatchedSupplierId(row.MatchedSupplierID); err != nil {
		return nil, err
	}
	if err := p.SetInvoiceDate(adapter.TimeToText(row.InvoiceDate)); err != nil {
		return nil, err
	}
	p.SetAmount(row.Amount)
	p.SetCurrency(currencyCode(row.Currency))
	p.SetStatus(tallyStatus(row.Status))
	if err := p.SetRawData(row.RawData); err != nil {
		return nil, err
	}
	return &p, nil
}

// TallyPurchaseImportFromProto converts a Proto TallyPurchaseImport back to GORM.
func TallyPurchaseImportFromProto(p protosync.TallyPurchaseImport) (gormsync.TallyPurchaseImport, error) {
	base, err := p.Base()
	if err != nil {
		return gormsync.TallyPurchaseImport{}, err
	}
	modelBase, err := syncBaseFromProto(base)
	if err != nil {
		return gormsync.TallyPurchaseImport{}, err
	}
	importBatch, err := text(p.ImportBatch())
	if err != nil {
		return gormsync.TallyPurchaseImport{}, err
	}
	invoiceNumber, err := text(p.InvoiceNumber())
	if err != nil {
		return gormsync.TallyPurchaseImport{}, err
	}
	supplierName, err := text(p.SupplierName())
	if err != nil {
		return gormsync.TallyPurchaseImport{}, err
	}
	matchedSupplierID, err := text(p.MatchedSupplierId())
	if err != nil {
		return gormsync.TallyPurchaseImport{}, err
	}
	invoiceDate, err := text(p.InvoiceDate())
	if err != nil {
		return gormsync.TallyPurchaseImport{}, err
	}
	rawData, err := text(p.RawData())
	if err != nil {
		return gormsync.TallyPurchaseImport{}, err
	}
	return gormsync.TallyPurchaseImport{
		Base:              modelBase,
		ImportBatch:       importBatch,
		Year:              adapter.Int64ToInt(p.Year()),
		InvoiceNumber:     invoiceNumber,
		SupplierName:      supplierName,
		MatchedSupplierID: matchedSupplierID,
		InvoiceDate:       adapter.TextToTime(invoiceDate),
		Amount:            p.Amount(),
		Currency:          currencyText(p.Currency()),
		Status:            p.Status().String(),
		RawData:           rawData,
	}, nil
}
