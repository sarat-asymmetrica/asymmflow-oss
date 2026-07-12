// Package sync contains the synchronization domain model.
package sync

import (
	"time"

	"gorm.io/gorm"
)

type Base struct {
	ID        string         `gorm:"primaryKey;size:36" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Version   int            `gorm:"default:1" json:"version"`
	CreatedBy string         `json:"created_by"`
}

type SyncStatus struct {
	Base
	FilePath string `gorm:"uniqueIndex;size:1000" json:"file_path"`
	Status   string `gorm:"index;size:50" json:"status"`

	LastSyncTime time.Time `json:"last_sync_time"`
}

type SyncRecord struct {
	Base
	SyncTable     string    `gorm:"index;size:100" json:"sync_table"`
	RecordID      string    `gorm:"index;size:36" json:"record_id"`
	SyncedAt      time.Time `gorm:"index;autoUpdateTime" json:"synced_at"`
	Direction     string    `gorm:"size:10;check:direction IN ('push','pull')" json:"direction"` // "push" or "pull"
	RemoteVersion int       `gorm:"check:remote_version >= 0" json:"remote_version"`
	LocalVersion  int       `gorm:"check:local_version >= 0" json:"local_version"`
	ConflictState string    `gorm:"size:20;check:conflict_state IN ('none','local_wins','remote_wins')" json:"conflict_state"` // "none", "local_wins", "remote_wins"
}

type TallyInvoiceImport struct {
	Base
	ImportBatch       string    `gorm:"index;size:36" json:"import_batch"`
	Year              int       `gorm:"index;check:year >= 2000 AND year <= 2100" json:"year"`
	InvoiceNumber     string    `gorm:"index;size:100" json:"invoice_number"`
	CustomerName      string    `gorm:"index;size:255" json:"customer_name"`
	MatchedCustomerID string    `gorm:"index;size:36" json:"matched_customer_id"`
	InvoiceDate       time.Time `gorm:"autoCreateTime:false" json:"invoice_date"`
	Amount            float64   `gorm:"check:amount >= 0" json:"amount"`
	Currency          string    `gorm:"size:3" json:"currency"`
	Status            string    `gorm:"index;size:50;check:status IN ('imported','matched','duplicate','error','pending')" json:"status"` // imported, matched, duplicate, error
	RawData           string    `gorm:"type:varchar(5000)" json:"raw_data"`
}

type TallyPurchaseImport struct {
	Base
	ImportBatch       string    `gorm:"index;size:36" json:"import_batch"`
	Year              int       `gorm:"index;check:year >= 2000 AND year <= 2100" json:"year"`
	InvoiceNumber     string    `gorm:"index;size:100" json:"invoice_number"`
	SupplierName      string    `gorm:"index;size:255" json:"supplier_name"`
	MatchedSupplierID string    `gorm:"index;size:36" json:"matched_supplier_id"`
	InvoiceDate       time.Time `gorm:"autoCreateTime:false" json:"invoice_date"`
	Amount            float64   `gorm:"check:amount >= 0" json:"amount"`
	Currency          string    `gorm:"size:3" json:"currency"`
	Status            string    `gorm:"index;size:50;check:status IN ('imported','matched','duplicate','error','pending')" json:"status"` // imported, matched, duplicate, error
	RawData           string    `gorm:"type:varchar(5000)" json:"raw_data"`
}
