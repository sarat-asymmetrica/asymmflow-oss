// Package documents converts document models to and from generated Proto messages.
package documents

import (
	"strings"

	"ph_holdings_app/pkg/adapter"
	gormdocuments "ph_holdings_app/pkg/documents"
	"ph_holdings_app/pkg/documents/classifier"
	commonproto "ph_holdings_app/schemas/go/common"
	protodocuments "ph_holdings_app/schemas/go/documents"

	capnp "capnproto.org/go/capnp/v3"
)

func newMessage() (*capnp.Message, *capnp.Segment, error) {
	return capnp.NewMessage(capnp.SingleSegment(nil))
}

func documentBaseToProto(seg *capnp.Segment, base gormdocuments.Base) (commonproto.Base, error) {
	p, err := commonproto.NewBase(seg)
	if err != nil {
		return commonproto.Base{}, err
	}
	for _, err := range []error{p.SetId(base.ID), p.SetCreatedAt(adapter.TimeToText(base.CreatedAt)), p.SetUpdatedAt(adapter.TimeToText(base.UpdatedAt)), p.SetDeletedAt(adapter.DeletedAtToText(base.DeletedAt)), p.SetCreatedBy(base.CreatedBy), p.SetUpdatedBy("")} {
		if err != nil {
			return commonproto.Base{}, err
		}
	}
	p.SetStatus(commonproto.RecordStatus_active)
	p.SetSyncState(commonproto.SyncState_synced)
	return p, nil
}

func setBase(seg *capnp.Segment, setter func(commonproto.Base) error, base gormdocuments.Base) error {
	pb, err := documentBaseToProto(seg, base)
	if err != nil {
		return err
	}
	return setter(pb)
}

func currencyCode(currency string) commonproto.CurrencyCode {
	return commonproto.CurrencyCodeFromString(strings.ToLower(strings.TrimSpace(currency)))
}

func currencyText(currency commonproto.CurrencyCode) string {
	return strings.ToUpper(currency.String())
}

func documentType(kind string) protodocuments.DocumentType {
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(kind), " ", "_")) {
	case "rfq":
		return protodocuments.DocumentType_rfq
	case "quote", "quotation", "offer":
		return protodocuments.DocumentType_quote
	case "purchase_order", "po":
		return protodocuments.DocumentType_purchaseOrder
	case "invoice":
		return protodocuments.DocumentType_invoice
	case "supplier_invoice":
		return protodocuments.DocumentType_supplierInvoice
	case "delivery_note":
		return protodocuments.DocumentType_deliveryNote
	case "bank_statement":
		return protodocuments.DocumentType_bankStatement
	case "contract":
		return protodocuments.DocumentType_contract
	case "receipt":
		return protodocuments.DocumentType_receipt
	case "other":
		return protodocuments.DocumentType_other
	default:
		return protodocuments.DocumentType_unknown
	}
}

func documentTypeText(kind protodocuments.DocumentType) string {
	switch kind {
	case protodocuments.DocumentType_rfq:
		return "RFQ"
	case protodocuments.DocumentType_quote:
		return "Quotation"
	case protodocuments.DocumentType_purchaseOrder:
		return "PurchaseOrder"
	case protodocuments.DocumentType_invoice:
		return "Invoice"
	case protodocuments.DocumentType_supplierInvoice:
		return "SupplierInvoice"
	case protodocuments.DocumentType_deliveryNote:
		return "DeliveryNote"
	case protodocuments.DocumentType_bankStatement:
		return "BankStatement"
	case protodocuments.DocumentType_contract:
		return "Contract"
	case protodocuments.DocumentType_receipt:
		return "Receipt"
	case protodocuments.DocumentType_other:
		return "Other"
	default:
		return "Unknown"
	}
}

func textList(seg *capnp.Segment, values []string) (capnp.TextList, error) {
	l, err := capnp.NewTextList(seg, int32(len(values)))
	if err != nil {
		return capnp.TextList{}, err
	}
	for i, value := range values {
		if err := l.Set(i, value); err != nil {
			return capnp.TextList{}, err
		}
	}
	return l, nil
}

func CompanyInfoToProto(m gormdocuments.CompanyInfo) (*protodocuments.CompanyInfo, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protodocuments.NewCompanyInfo(seg)
	if err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetName(m.Name), p.SetLegalName(m.LegalName), p.SetAddress(m.Address), p.SetPhone(m.Phone), p.SetEmail(m.Email), p.SetWebsite(m.Website), p.SetTaxNumber(m.TaxNumber), p.SetCommercialReg(m.CommercialReg)} {
		if err != nil {
			return nil, err
		}
	}
	return &p, nil
}

func CompanyInfoFromProto(p protodocuments.CompanyInfo) (gormdocuments.CompanyInfo, error) {
	m := gormdocuments.CompanyInfo{}
	m.Name, _ = p.Name()
	m.LegalName, _ = p.LegalName()
	m.Address, _ = p.Address()
	m.Phone, _ = p.Phone()
	m.Email, _ = p.Email()
	m.Website, _ = p.Website()
	m.TaxNumber, _ = p.TaxNumber()
	m.CommercialReg, _ = p.CommercialReg()
	return m, nil
}

func BrandingConfigToProto(m gormdocuments.BrandingConfig) (*protodocuments.BrandingConfig, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protodocuments.NewBrandingConfig(seg)
	if err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetDivision(m.Division), p.SetLogoPath(m.LogoPath), p.SetLetterheadPath(m.LetterheadPath), p.SetPrimaryColor(m.PrimaryColor), p.SetSecondaryColor(m.SecondaryColor), p.SetFooterText(m.FooterText), p.SetDocumentPrefix(m.DocumentPrefix)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetDefaultCurrency(currencyCode(m.DefaultCurrency))
	return &p, nil
}

func BrandingConfigFromProto(p protodocuments.BrandingConfig) (gormdocuments.BrandingConfig, error) {
	m := gormdocuments.BrandingConfig{}
	m.Division, _ = p.Division()
	m.LogoPath, _ = p.LogoPath()
	m.LetterheadPath, _ = p.LetterheadPath()
	m.PrimaryColor, _ = p.PrimaryColor()
	m.SecondaryColor, _ = p.SecondaryColor()
	m.FooterText, _ = p.FooterText()
	m.DocumentPrefix, _ = p.DocumentPrefix()
	m.DefaultCurrency = currencyText(p.DefaultCurrency())
	return m, nil
}

func FileWatchEventToProto(m gormdocuments.FileWatchEvent) (*protodocuments.FileWatchEvent, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protodocuments.NewFileWatchEvent(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetFilePath(m.FilePath), p.SetEventType(m.EventType)} {
		if err != nil {
			return nil, err
		}
	}
	return &p, nil
}

func FileWatchEventFromProto(p protodocuments.FileWatchEvent) (gormdocuments.FileWatchEvent, error) {
	m := gormdocuments.FileWatchEvent{}
	m.FilePath, _ = p.FilePath()
	m.EventType, _ = p.EventType()
	return m, nil
}

func BankStatementFileToProto(m gormdocuments.BankStatementFile) (*protodocuments.BankStatementFile, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protodocuments.NewBankStatementFile(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetBankStatementId(m.BankStatementID), p.SetFileName(m.FileName), p.SetFileType(m.FileType), p.SetFileHash(m.FileHash), p.SetStoragePath(m.StoragePath), p.SetOcrEngine(m.OCREngine), p.SetOcrProcessedAt(adapter.TimePtrToText(m.OCRProcessedAt))} {
		if err != nil {
			return nil, err
		}
	}
	p.SetFileSize(m.FileSize)
	p.SetIsStored(m.IsStored)
	p.SetOcrConfidence(m.OCRConfidence)
	return &p, nil
}

func BankStatementFileFromProto(p protodocuments.BankStatementFile) (gormdocuments.BankStatementFile, error) {
	m := gormdocuments.BankStatementFile{}
	m.BankStatementID, _ = p.BankStatementId()
	m.FileName, _ = p.FileName()
	m.FileType, _ = p.FileType()
	m.FileSize = p.FileSize()
	m.FileHash, _ = p.FileHash()
	m.StoragePath, _ = p.StoragePath()
	m.IsStored = p.IsStored()
	m.OCREngine, _ = p.OcrEngine()
	m.OCRConfidence = p.OcrConfidence()
	if s, err := p.OcrProcessedAt(); err == nil {
		m.OCRProcessedAt = adapter.TextToTimePtr(s)
	}
	return m, nil
}

func ClassificationResultToProto(m classifier.ClassificationResult) (*protodocuments.ClassificationResult, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protodocuments.NewClassificationResult(seg)
	if err != nil {
		return nil, err
	}
	p.SetDocumentType(documentType(m.DocumentType))
	p.SetConfidence(m.Confidence)
	for _, err := range []error{p.SetMethod(m.Method), p.SetRouteTo(m.RouteTo), p.SetSuggestedAction(m.SuggestedAction), p.SetExplanation(m.Explanation)} {
		if err != nil {
			return nil, err
		}
	}
	keywords, err := textList(seg, m.KeywordsFound)
	if err != nil {
		return nil, err
	}
	if err := p.SetKeywordsFound(keywords); err != nil {
		return nil, err
	}
	return &p, nil
}

func ClassificationResultFromProto(p protodocuments.ClassificationResult) (classifier.ClassificationResult, error) {
	m := classifier.ClassificationResult{DocumentType: documentTypeText(p.DocumentType()), Confidence: p.Confidence()}
	m.Method, _ = p.Method()
	m.RouteTo, _ = p.RouteTo()
	m.SuggestedAction, _ = p.SuggestedAction()
	m.Explanation, _ = p.Explanation()
	return m, nil
}
