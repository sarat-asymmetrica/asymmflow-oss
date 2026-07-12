package pdf

import engines "ph_holdings_app/pkg/engines"

type InvoiceItem = engines.InvoiceItem
type InvoiceData = engines.InvoiceData
type TemplateZoneConfig = engines.TemplateZoneConfig
type ColumnConfig = engines.ColumnConfig
type TemplateLayoutConfig = engines.TemplateLayoutConfig
type ContentBox = engines.ContentBox
type PDFGenerator = engines.PDFGenerator

func DefaultContentBox() ContentBox {
	return engines.DefaultContentBox()
}

func NewPDFGenerator(letterheadPath string) (*PDFGenerator, error) {
	return engines.NewPDFGenerator(letterheadPath)
}

func NewPDFGeneratorWithZones(letterheadPath, zoneConfigPath string) (*PDFGenerator, error) {
	return engines.NewPDFGeneratorWithZones(letterheadPath, zoneConfigPath)
}
