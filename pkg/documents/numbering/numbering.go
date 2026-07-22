// Package numbering is the generic document-number engine: sequential,
// gap-conscious, per-prefix/per-year counters with pluggable formats.
//
// Promoted from four near-identical implementations in package main
// (GenerateInvoiceNumber / GenerateCreditNoteNumber / GeneratePONumber /
// GenerateDNNumber) as Wave 2 Mission A engine-promotion work. The engine
// preserves their exact concurrency discipline: a transaction with a
// row-locked read-modify-write on the sequence table, plus an optional
// first-of-period seed callback so deployments migrating from ad-hoc
// numbering don't restart at 1.
//
// The sequence table is the pre-existing `invoice_sequences`
// (prefix, year, last_number) — the name is historical; it stores counters
// for every document type, keyed by prefix.
package numbering

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Sequence mirrors the historical invoice_sequences table (one counter per
// prefix+year). It intentionally matches pkg/finance.InvoiceSequence's schema
// so both bind to the same rows.
type Sequence struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Prefix     string    `gorm:"size:10;uniqueIndex:idx_invoice_sequence_prefix_year" json:"prefix"`
	Year       int       `gorm:"uniqueIndex:idx_invoice_sequence_prefix_year" json:"year"`
	LastNumber int       `gorm:"not null;default:0" json:"last_number"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (Sequence) TableName() string { return "invoice_sequences" }

// Spec describes one document-number scheme.
type Spec struct {
	// Prefix keys the counter row (e.g. "INV", "CN", "PO", "DN"). Counters
	// reset per calendar year via the (prefix, year) unique index.
	Prefix string

	// Template renders the number. Placeholders:
	//   {prefix}  the Prefix verbatim
	//   {year}    4-digit CALENDAR year of now   {yy}  2-digit calendar year
	//   {fy}      fiscal-year label "YY-YY" (e.g. "26-27"), per FYStartMonth
	//   {date}    YYYYMMDD issue date
	//   {seq}     the sequence number, zero-padded to Pad digits
	// Examples: "INV-{date}-{seq}" → INV-20260703-0007
	//           "PO-{year}-{seq}"  → PO-2026-0012
	//           "INV/{fy}/{seq}"   → INV/26-27/0007 (FYStartMonth 4, issued Jun 2026)
	// {year}/{yy} always mean the calendar year of now, regardless of
	// FYStartMonth — only {fy} and the counter's reset cadence follow it.
	Template string

	// Pad is the zero-pad width for {seq}. Zero defaults to 4.
	Pad int

	// FYStartMonth selects the counter's reset cadence. 0 or 1 means
	// calendar year (Jan 1 reset) — every existing GCC spec, left unset,
	// behaves exactly as before. 2-12 means a fiscal year starting on the
	// 1st of that month; India uses 4 (April-March, per Rule 46's "unique
	// per FY" numbering requirement). See FiscalYearFor.
	FYStartMonth int

	// Seed, when non-nil, initializes a first-of-year counter from existing
	// data (e.g. COUNT of documents already numbered under a legacy scheme)
	// so migrated deployments continue rather than restart. It runs inside
	// the same transaction that creates the counter row.
	Seed func(tx *gorm.DB, year int) (int64, error)
}

// FiscalYearFor returns the year in which the fiscal year containing now
// started. startMonth <= 1 means calendar year, so the result is just
// now.Year() — every existing GCC spec's behavior, unchanged. startMonth
// 2-12 means a fiscal year starting on the 1st of that month: a date before
// that month belongs to the fiscal year that started the PREVIOUS calendar
// year (e.g. startMonth 4: 2026-03-31 → 2025, 2026-04-01 → 2026).
func FiscalYearFor(now time.Time, startMonth int) int {
	if startMonth <= 1 {
		return now.Year()
	}
	if int(now.Month()) >= startMonth {
		return now.Year()
	}
	return now.Year() - 1
}

// Engine issues document numbers against a GORM database.
type Engine struct {
	db *gorm.DB
}

// New creates a numbering engine. AutoMigrate of the Sequence table is the
// caller's responsibility (the trading app already migrates the same table
// via pkg/finance.InvoiceSequence).
func New(db *gorm.DB) *Engine {
	return &Engine{db: db}
}

// Next issues the next number for the spec at time now, in its own
// transaction.
func (e *Engine) Next(spec Spec, now time.Time) (string, error) {
	if e == nil || e.db == nil {
		return "", errors.New("numbering: engine has no database")
	}
	var out string
	err := e.db.Transaction(func(tx *gorm.DB) error {
		n, err := NextInTx(tx, spec, now)
		out = n
		return err
	})
	return out, err
}

// NextInTx issues the next number inside an existing transaction, so a
// document insert and its number allocation can commit atomically.
//
// Concurrency: the FIRST statement is the increment UPDATE, so the
// transaction acquires its write lock immediately instead of upgrading a
// read lock later — the upgrade path is what produced "database is locked"
// deadlocks under concurrent allocation on SQLite. Writers then serialize
// on busy_timeout. (The four package-main implementations this replaces
// used SELECT-FOR-UPDATE-then-save, which row-locks on Postgres but
// devolves to the fragile read→write upgrade on SQLite.)
func NextInTx(tx *gorm.DB, spec Spec, now time.Time) (string, error) {
	if tx == nil {
		return "", errors.New("numbering: nil transaction")
	}
	prefix := strings.TrimSpace(spec.Prefix)
	if prefix == "" {
		return "", errors.New("numbering: spec.Prefix is required")
	}
	if strings.TrimSpace(spec.Template) == "" {
		return "", errors.New("numbering: spec.Template is required")
	}

	year := FiscalYearFor(now, spec.FYStartMonth)
	res := tx.Model(&Sequence{}).
		Where("prefix = ? AND year = ?", prefix, year).
		Updates(map[string]any{
			"last_number": gorm.Expr("last_number + 1"),
			"updated_at":  now,
		})
	if res.Error != nil {
		return "", fmt.Errorf("numbering: increment sequence %s/%d: %w", prefix, year, res.Error)
	}

	if res.RowsAffected == 0 {
		// First allocation for this prefix+year: seed (optionally) and create.
		// A concurrent first-allocation loses the UNIQUE(prefix,year) race and
		// errors; on SQLite the write lock taken by the UPDATE above already
		// serializes writers, so the race is Postgres-only and rare.
		var start int64
		if spec.Seed != nil {
			var err error
			start, err = spec.Seed(tx, year)
			if err != nil {
				return "", fmt.Errorf("numbering: seed sequence %s/%d: %w", prefix, year, err)
			}
		}
		seq := Sequence{Prefix: prefix, Year: year, LastNumber: int(start) + 1, UpdatedAt: now}
		if err := tx.Create(&seq).Error; err != nil {
			return "", fmt.Errorf("numbering: create sequence %s/%d: %w", prefix, year, err)
		}
		return Render(spec, now, seq.LastNumber), nil
	}

	var seq Sequence
	if err := tx.Where("prefix = ? AND year = ?", prefix, year).First(&seq).Error; err != nil {
		return "", fmt.Errorf("numbering: read sequence %s/%d after increment: %w", prefix, year, err)
	}
	return Render(spec, now, seq.LastNumber), nil
}

// Render expands the spec's template for a given time and sequence value.
// Exposed so previews can show "the next number would be ..." without
// consuming one.
func Render(spec Spec, now time.Time, seqNumber int) string {
	pad := spec.Pad
	if pad <= 0 {
		pad = 4
	}
	fyStart := FiscalYearFor(now, spec.FYStartMonth)
	r := strings.NewReplacer(
		"{prefix}", spec.Prefix,
		"{year}", fmt.Sprintf("%d", now.Year()),
		"{yy}", fmt.Sprintf("%02d", now.Year()%100),
		"{fy}", fmt.Sprintf("%02d-%02d", fyStart%100, (fyStart+1)%100),
		"{date}", now.Format("20060102"),
		"{seq}", fmt.Sprintf("%0*d", pad, seqNumber),
	)
	return r.Replace(spec.Template)
}

// ValidateGSTSeriesNumber checks a rendered document number against GST
// Rule 46: at most 16 characters, restricted to alphanumerics plus "/" and
// "-". Numbering itself stays GST-agnostic (it also serves GCC specs with
// no such constraint); callers emitting India invoice/credit-note numbers
// validate through this before persisting.
func ValidateGSTSeriesNumber(number string) error {
	if number == "" {
		return errors.New("numbering: invoice number is empty")
	}
	if len(number) > 16 {
		return fmt.Errorf("numbering: %q is %d characters, Rule 46 allows at most 16", number, len(number))
	}
	for _, c := range number {
		if !isRule46Char(c) {
			return fmt.Errorf("numbering: %q contains %q, Rule 46 allows only letters, digits, \"/\", and \"-\"", number, c)
		}
	}
	return nil
}

func isRule46Char(c rune) bool {
	switch {
	case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		return true
	case c == '/' || c == '-':
		return true
	default:
		return false
	}
}
