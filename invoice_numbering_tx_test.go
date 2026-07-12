package main

// Wave 8 P1: invoice numbering was moved inside CreateInvoiceWithOptions's
// transaction so a rollback releases the reserved number instead of leaving a
// sequence gap (deployed-PH parity). This pins that transactional property at
// the generateInvoiceNumberWithTx primitive.

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGenerateInvoiceNumberWithTx_RollbackReleasesNumber(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Invoice{}))

	// Reserve a number inside a transaction that then fails. Because the number
	// is drawn on the same tx, the rollback must un-reserve it (no gap).
	sentinel := errors.New("force rollback")
	var reserved string
	err := app.db.Transaction(func(tx *gorm.DB) error {
		n, e := app.generateInvoiceNumberWithTx(tx)
		if e != nil {
			return e
		}
		reserved = n
		return sentinel
	})
	require.ErrorIs(t, err, sentinel)
	require.NotEmpty(t, reserved)

	// The next successful allocation reuses the rolled-back number — proof the
	// numbering shared the invoice's transaction rather than its own.
	next, err := app.GenerateInvoiceNumber()
	require.NoError(t, err)
	require.Equal(t, reserved, next,
		"a rolled-back invoice tx must release its reserved number (no sequence gap)")
}

func TestGenerateInvoiceNumberWithTx_CommitAdvancesSequence(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Invoice{}))

	var first string
	require.NoError(t, app.db.Transaction(func(tx *gorm.DB) error {
		n, e := app.generateInvoiceNumberWithTx(tx)
		first = n
		return e
	}))
	require.NotEmpty(t, first)

	// A committed reservation must advance the sequence: the next number differs.
	second, err := app.GenerateInvoiceNumber()
	require.NoError(t, err)
	require.NotEqual(t, first, second,
		"a committed invoice number must not be reissued")
	require.Greater(t, second, first)
}
