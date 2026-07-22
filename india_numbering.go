package main

import "ph_holdings_app/pkg/documents/numbering"

// India Spec-01 B4 (R-A3-3): India numbering series are per-GSTIN per-FY.
// Spec.Prefix carries the sequence KEY, which must embed the GSTIN so two
// divisions sharing one PAN (Meridian Mumbai vs Bengaluru) never collide on
// the same counter row; Template renders the SHORT number without a system
// prefix — Rule 46 numbers read as a clean fiscal-year series, not a system
// tag. FYStartMonth is threaded from the caller (activeOverlay.
// FYStartMonthOrDefault(), April for a mounted India plane per G9). Every
// number these specs produce is validated with numbering.
// ValidateGSTSeriesNumber before it is persisted — house refuse-to-generate
// doctrine, never a silent Rule-46 violation.

// indiaInvoiceNumberSpec mints India tax-invoice numbers, e.g.
// "INV/26-27/007" for GSTIN 27AABCM0472E1ZT issued in FY 2026-27.
func indiaInvoiceNumberSpec(gstin string, fyStartMonth int) numbering.Spec {
	return numbering.Spec{
		Prefix:       "ININV:" + gstin,
		Template:     "INV/{fy}/{seq}",
		Pad:          3,
		FYStartMonth: fyStartMonth,
	}
}

// indiaBillOfSupplyNumberSpec is indiaInvoiceNumberSpec's twin for a
// composition division's Bill of Supply — its own series, since a Bill of
// Supply is a genuine second document type (G6), never sharing a counter
// with the tax-invoice series.
func indiaBillOfSupplyNumberSpec(gstin string, fyStartMonth int) numbering.Spec {
	return numbering.Spec{
		Prefix:       "INBOS:" + gstin,
		Template:     "BOS/{fy}/{seq}",
		Pad:          3,
		FYStartMonth: fyStartMonth,
	}
}

// indiaCreditNoteNumberSpec is the credit-note twin of indiaInvoiceNumberSpec.
func indiaCreditNoteNumberSpec(gstin string, fyStartMonth int) numbering.Spec {
	return numbering.Spec{
		Prefix:       "INCN:" + gstin,
		Template:     "CN/{fy}/{seq}",
		Pad:          3,
		FYStartMonth: fyStartMonth,
	}
}
