// Package documents contains the document domain model.
package documents

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

type FileWatchEvent struct {
	Base
	FilePath  string `gorm:"index;size:1000" json:"file_path"`
	EventType string `json:"event_type"`
}

type BankStatementFile struct {
	Base
	BankStatementID string     `gorm:"index;size:36" json:"bank_statement_id"`
	FileName        string     `gorm:"size:255" json:"file_name"`
	FileType        string     `gorm:"size:10" json:"file_type"` // PDF, CSV, XLS
	FileSize        int64      `json:"file_size"`
	FileHash        string     `gorm:"size:64" json:"file_hash"` // SHA-256
	StoragePath     string     `gorm:"size:500" json:"storage_path"`
	IsStored        bool       `gorm:"default:false" json:"is_stored"`
	OCREngine       string     `gorm:"size:30" json:"ocr_engine"`
	OCRConfidence   float64    `json:"ocr_confidence"`
	OCRProcessedAt  *time.Time `json:"ocr_processed_at"`
}
