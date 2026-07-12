// Package documents defines the document domain ports.
package documents

import (
	"context"
	"io"
)

type StoragePort interface {
	SaveFile(path string, data []byte) error
	ReadFile(path string) ([]byte, error)
	FileExists(path string) bool
}

type ConfigPort interface {
	GetExportDir() string
	GetCompanyInfo() CompanyInfo
	GetDivisionBranding(division string) BrandingConfig
}

type FinanceDataPort interface {
	GetInvoiceForPDF(id string) (any, error)
	GetOfferForPDF(id string) (any, error)
	GetPurchaseOrderForPDF(id string) (any, error)
}

type CompanyInfo struct {
	Name          string `json:"name"`
	LegalName     string `json:"legal_name"`
	Address       string `json:"address"`
	Phone         string `json:"phone"`
	Email         string `json:"email"`
	Website       string `json:"website"`
	TaxNumber     string `json:"tax_number"`
	CommercialReg string `json:"commercial_reg"`
}

type BrandingConfig struct {
	Division        string `json:"division"`
	LogoPath        string `json:"logo_path"`
	LetterheadPath  string `json:"letterhead_path"`
	PrimaryColor    string `json:"primary_color"`
	SecondaryColor  string `json:"secondary_color"`
	FooterText      string `json:"footer_text"`
	DocumentPrefix  string `json:"document_prefix"`
	DefaultCurrency string `json:"default_currency"`
}

type PDFGenerator interface {
	GenerateInvoicePDF(ctx context.Context, invoiceID string) (string, error)
	GenerateOfferPDF(ctx context.Context, offerID string) (string, error)
	GeneratePurchaseOrderPDF(ctx context.Context, purchaseOrderID string) (string, error)
	GenerateDeliveryNotePDF(ctx context.Context, deliveryNoteID string) (string, error)
}

type OCRService interface {
	ProcessDocument(ctx context.Context, filePath, documentType string) (map[string]any, error)
	ProcessDocumentBytes(ctx context.Context, data []byte, fileName, documentType string) (map[string]any, error)
	ProcessBatch(ctx context.Context, filePaths []string, documentType string) ([]map[string]any, error)
	GetStats(ctx context.Context) (map[string]any, error)
}

type DocumentClassifier interface {
	Classify(ctx context.Context, filePath string) (map[string]any, error)
	ClassifyText(ctx context.Context, fileName, content string) (map[string]any, error)
	RouteToEntity(ctx context.Context, classification map[string]any) (map[string]any, error)
}

type ExcelParser interface {
	ParseCosting(ctx context.Context, reader io.Reader) (map[string]any, error)
	ParseBankStatement(ctx context.Context, reader io.Reader) ([]map[string]any, error)
	ExportCosting(ctx context.Context, payload map[string]any) (string, error)
	GenerateTemplate(ctx context.Context, templateName string) (string, error)
}
