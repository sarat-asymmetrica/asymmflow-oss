package main

// engine_bridge.go re-exports pkg/engines symbols into package main.
//
// These engines are implemented exactly ONCE, in ph_holdings_app/pkg/engines;
// the aliases below let the composition root (package main) refer to them by
// short name. This follows the bridge pattern already used by predictor.go,
// customer.go, batch.go, and payment_intelligence.go.
//
// All engines are implemented exactly once in pkg/engines and bridged here.
// Geometry, costing, and pdf/arabic/langpack engines are fully consolidated;
// root copies have been deleted. PDFGenerator.pdf is now accessed via Doc().
//
// Do NOT add implementation logic here — only aliases.

import engines "ph_holdings_app/pkg/engines"

// Engine types.
type (
	OfferScanner   = engines.OfferScanner
	OfferMetadata  = engines.OfferMetadata
	EHBasket       = engines.EHBasket
	EHParser       = engines.EHParser
	ParsedEHBasket = engines.ParsedEHBasket

	// Geometry bridge types.
	GeometryBridge   = engines.GeometryBridge
	InvoiceGeometry  = engines.InvoiceGeometry
	TenderGeometry   = engines.TenderGeometry
	TenderItem       = engines.TenderItem
	ComplianceData   = engines.ComplianceData
	InvoiceResult    = engines.InvoiceResult
	TenderResult     = engines.TenderResult
	Customer360      = engines.Customer360
	ComplianceResult = engines.ComplianceResult
	ERPEvent         = engines.ERPEvent
	RoutingResult    = engines.RoutingResult

	// Costing engine types.
	CostingEngine = engines.CostingEngine
	CostingSheet  = engines.CostingSheet
	CostingItem   = engines.CostingItem
	CustomerGrade = engines.CustomerGrade

	// PDF generator types.
	PDFGenerator         = engines.PDFGenerator
	InvoiceData          = engines.InvoiceData
	InvoiceItem          = engines.InvoiceItem
	TemplateLayoutConfig = engines.TemplateLayoutConfig
	TemplateZoneConfig   = engines.TemplateZoneConfig
	ColumnConfig         = engines.ColumnConfig
	ContentBox           = engines.ContentBox
	ArabicShaper         = engines.ArabicShaper
	LangPack             = engines.LangPack
	LangPackConfig       = engines.LangPackConfig
	NumberFormat         = engines.NumberFormat
)

// Costing engine grade constants.
const (
	GradeA = engines.GradeA
	GradeB = engines.GradeB
	GradeC = engines.GradeC
	GradeD = engines.GradeD
)

// Engine constructors (function-value aliases).
var (
	NewOfferScanner   = engines.NewOfferScanner
	NewEHParser       = engines.NewEHParser
	NewGeometryBridge = engines.NewGeometryBridge

	// Costing engine constructor.
	NewCostingEngine = engines.NewCostingEngine

	// PDF generator constructors.
	NewPDFGenerator          = engines.NewPDFGenerator
	NewPDFGeneratorWithZones = engines.NewPDFGeneratorWithZones
	DefaultContentBox        = engines.DefaultContentBox
	NewArabicShaper          = engines.NewArabicShaper
	NewLangPack              = engines.NewLangPack
	IsArabicText             = engines.IsArabicText
	IsArabicChar             = engines.IsArabicChar
	FormatArabicNumber       = engines.FormatArabicNumber
	ReverseArabicLine        = engines.ReverseArabicLine
)
