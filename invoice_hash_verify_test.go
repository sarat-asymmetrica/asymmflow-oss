package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// PH convergence 1-HMAC (PH MON-003): stored invoice hashes are verifiable,
// blank hashes are reported (not treated as valid), and tampering flips Valid.
func TestVerifyInvoiceHash(t *testing.T) {
	a := setupTestApp(t)

	inv := Invoice{InvoiceNumber: "INV-26-7001", Status: "Draft", InvoiceDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), GrandTotalBHD: 315, VATBHD: 28.636}
	require.NoError(t, a.db.Create(&inv).Error)

	blank, err := a.VerifyInvoiceHash(inv.ID)
	require.NoError(t, err)
	require.False(t, blank.HasHash)
	require.False(t, blank.Valid, "a blank hash must never verify as valid")

	good := computeDocumentHMAC(inv.InvoiceNumber, inv.InvoiceDate.Format("2006-01-02"), inv.GrandTotalBHD, inv.VATBHD)
	require.NoError(t, a.db.Model(&Invoice{}).Where("id = ?", inv.ID).Update("invoice_hash", good).Error)

	ok, err := a.VerifyInvoiceHash(inv.ID)
	require.NoError(t, err)
	require.True(t, ok.HasHash)
	require.True(t, ok.Valid)

	// Tamper with the stored total — recomputed hash must no longer match.
	require.NoError(t, a.db.Model(&Invoice{}).Where("id = ?", inv.ID).Update("grand_total_bhd", 999).Error)
	tampered, err := a.VerifyInvoiceHash(inv.ID)
	require.NoError(t, err)
	require.True(t, tampered.HasHash)
	require.False(t, tampered.Valid, "tampering must be detected")
}
