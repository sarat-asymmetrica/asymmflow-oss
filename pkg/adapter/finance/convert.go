// Package finance converts finance GORM models to and from generated Proto messages.
package finance

import (
	"fmt"
	"strings"

	"ph_holdings_app/pkg/adapter"
	gormfinance "ph_holdings_app/pkg/finance"
	commonproto "ph_holdings_app/schemas/go/common"
	protofinance "ph_holdings_app/schemas/go/finance"

	capnp "capnproto.org/go/capnp/v3"
)

func newMessage() (*capnp.Message, *capnp.Segment, error) {
	return capnp.NewMessage(capnp.SingleSegment(nil))
}

func text(v string, err error) (string, error) {
	if err != nil {
		return "", err
	}
	return v, nil
}

func setBase(seg *capnp.Segment, setter func(commonproto.Base) error, base gormfinance.Base) error {
	pb, err := adapter.BaseToProto(seg, base)
	if err != nil {
		return err
	}
	return setter(pb)
}

func documentStatus(status string) commonproto.DocumentStatus {
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(status), " ", "")) {
	case "prepared":
		return commonproto.DocumentStatus_prepared
	case "sent":
		return commonproto.DocumentStatus_sent
	case "acknowledged":
		return commonproto.DocumentStatus_acknowledged
	case "issued":
		return commonproto.DocumentStatus_issued
	case "partiallypaid", "partialpaid", "partiallyreceived":
		return commonproto.DocumentStatus_partiallyPaid
	case "paid", "received":
		return commonproto.DocumentStatus_paid
	case "overdue":
		return commonproto.DocumentStatus_overdue
	case "void", "voided":
		return commonproto.DocumentStatus_voided
	case "cancelled", "canceled", "rejected", "dispute", "failed":
		return commonproto.DocumentStatus_cancelled
	case "closed", "reconciled", "verified", "matched", "approved", "converted":
		return commonproto.DocumentStatus_closed
	default:
		return commonproto.DocumentStatus_draft
	}
}

func statusText(status commonproto.DocumentStatus) string {
	switch status {
	case commonproto.DocumentStatus_prepared:
		return "Prepared"
	case commonproto.DocumentStatus_sent:
		return "Sent"
	case commonproto.DocumentStatus_acknowledged:
		return "Acknowledged"
	case commonproto.DocumentStatus_issued:
		return "Issued"
	case commonproto.DocumentStatus_partiallyPaid:
		return "PartiallyPaid"
	case commonproto.DocumentStatus_paid:
		return "Paid"
	case commonproto.DocumentStatus_overdue:
		return "Overdue"
	case commonproto.DocumentStatus_voided:
		return "Void"
	case commonproto.DocumentStatus_cancelled:
		return "Cancelled"
	case commonproto.DocumentStatus_closed:
		return "Closed"
	default:
		return "Draft"
	}
}

func currencyCode(currency string) commonproto.CurrencyCode {
	return commonproto.CurrencyCodeFromString(strings.ToLower(strings.TrimSpace(currency)))
}

func currencyText(currency commonproto.CurrencyCode) string {
	return strings.ToUpper(currency.String())
}

func paymentMethod(method string) protofinance.PaymentMethod {
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(method), " ", "")) {
	case "cash":
		return protofinance.PaymentMethod_cash
	case "cheque", "check":
		return protofinance.PaymentMethod_cheque
	case "banktransfer":
		return protofinance.PaymentMethod_bankTransfer
	case "creditcard":
		return protofinance.PaymentMethod_creditCard
	case "lc":
		return protofinance.PaymentMethod_lc
	case "pdc":
		return protofinance.PaymentMethod_pdc
	case "wiretransfer":
		return protofinance.PaymentMethod_wireTransfer
	default:
		return protofinance.PaymentMethod_other
	}
}

func paymentMethodText(method protofinance.PaymentMethod) string {
	switch method {
	case protofinance.PaymentMethod_cash:
		return "Cash"
	case protofinance.PaymentMethod_cheque:
		return "Cheque"
	case protofinance.PaymentMethod_bankTransfer:
		return "Bank Transfer"
	case protofinance.PaymentMethod_creditCard:
		return "Credit Card"
	case protofinance.PaymentMethod_lc:
		return "LC"
	case protofinance.PaymentMethod_pdc:
		return "PDC"
	case protofinance.PaymentMethod_wireTransfer:
		return "Wire Transfer"
	default:
		return "Other"
	}
}

// InvoiceToProto converts a GORM Invoice to a Proto Invoice message.
func InvoiceToProto(inv gormfinance.Invoice) (*protofinance.Invoice, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protofinance.NewInvoice(seg)
	if err != nil {
		return nil, err
	}
	if err := populateInvoice(seg, p, inv); err != nil {
		return nil, err
	}
	return &p, nil
}

func populateInvoice(seg *capnp.Segment, p protofinance.Invoice, inv gormfinance.Invoice) error {
	if err := setBase(seg, p.SetBase, inv.Base); err != nil {
		return err
	}
	setters := []error{
		p.SetInvoiceNumber(inv.InvoiceNumber),
		p.SetInvoiceDate(adapter.TimeToText(inv.InvoiceDate)),
		p.SetCustomerId(inv.CustomerID),
		p.SetCustomerName(inv.CustomerName),
		p.SetOrderId(inv.OrderID),
		p.SetCustomerPoNumber(inv.CustomerPONumber),
		p.SetDueDate(adapter.TimeToText(inv.DueDate)),
		p.SetUpdatedBy(inv.UpdatedBy),
		p.SetRfqId(inv.RfqID),
		p.SetQuoteId(inv.QuoteID),
		p.SetOfferId(inv.OfferID),
		p.SetOfferNumber(inv.OfferNumber),
		p.SetDeliveryNoteId(inv.DeliveryNoteID),
		p.SetDeliveryNoteNumber(inv.DeliveryNoteNumber),
		p.SetCustomerReference(inv.CustomerReference),
		p.SetAttentionPerson(inv.AttentionPerson),
		p.SetAttentionCompany(inv.AttentionCompany),
		p.SetAttentionPhone(inv.AttentionPhone),
		p.SetAttentionAddress(inv.AttentionAddress),
		p.SetDeliveryWeeks(inv.DeliveryWeeks),
		p.SetCountryOfOrigin(inv.CountryOfOrigin),
		p.SetIssuedBy(inv.IssuedBy),
		p.SetContactPhone(inv.ContactPhone),
		p.SetPaymentTerms(inv.PaymentTerms),
		p.SetDeliveryTerms(inv.DeliveryTerms),
		p.SetDivision(inv.Division),
		p.SetFieldVisibility(inv.FieldVisibility),
		p.SetDeliveryNoteRef(inv.DeliveryNoteRef),
		p.SetModeOfPayment(inv.ModeOfPayment),
		p.SetSuppliersRef(inv.SuppliersRef),
		p.SetOtherReferences(inv.OtherReferences),
		p.SetBuyersOrderNumber(inv.BuyersOrderNumber),
		p.SetBuyersOrderDate(adapter.TimePtrToText(inv.BuyersOrderDate)),
		p.SetDespatchDocumentNo(inv.DespatchDocumentNo),
		p.SetDeliveryNoteDate(adapter.TimePtrToText(inv.DeliveryNoteDate)),
		p.SetDespatchedThrough(inv.DespatchedThrough),
		p.SetDestination(inv.Destination),
		p.SetPlaceOfSupply(inv.PlaceOfSupply),
		p.SetTermsOfDelivery(inv.TermsOfDelivery),
		p.SetJournalEntryId(inv.JournalEntryID),
		p.SetInvoiceHash(inv.InvoiceHash),
	}
	for _, err := range setters {
		if err != nil {
			return err
		}
	}
	p.SetGrandTotalBhd(inv.GrandTotalBHD)
	p.SetStatus(documentStatus(inv.Status))
	p.SetOutstandingBhd(inv.OutstandingBHD)
	p.SetSubtotalBhd(inv.SubtotalBHD)
	p.SetTotalSupplierCostBhd(inv.TotalSupplierCostBHD)
	p.SetGrossMarginBhd(inv.GrossMarginBHD)
	p.SetGrossMarginPercent(inv.GrossMarginPercent)
	p.SetDiscountPercent(inv.DiscountPercent)
	p.SetVatBhd(inv.VATBHD)
	p.SetVatPercent(inv.VATPercent)
	if len(inv.Items) > 0 {
		items, err := protofinance.NewDBInvoiceItem_List(seg, int32(len(inv.Items)))
		if err != nil {
			return err
		}
		for i, item := range inv.Items {
			pi, err := protofinance.NewDBInvoiceItem(seg)
			if err != nil {
				return err
			}
			if err := populateDBInvoiceItem(seg, pi, item); err != nil {
				return err
			}
			if err := items.Set(i, pi); err != nil {
				return err
			}
		}
		return p.SetItems(items)
	}
	return nil
}

// InvoiceFromProto converts a Proto Invoice message to a GORM Invoice.
func InvoiceFromProto(p protofinance.Invoice) (gormfinance.Invoice, error) {
	base, err := p.Base()
	if err != nil {
		return gormfinance.Invoice{}, err
	}
	sharedBase, err := adapter.BaseFromProto(base)
	if err != nil {
		return gormfinance.Invoice{}, err
	}
	inv := gormfinance.Invoice{Base: sharedBase}
	if inv.InvoiceNumber, err = text(p.InvoiceNumber()); err != nil {
		return inv, err
	}
	if s, err := p.InvoiceDate(); err == nil {
		inv.InvoiceDate = adapter.TextToTime(s)
	}
	if inv.CustomerID, err = text(p.CustomerId()); err != nil {
		return inv, err
	}
	if inv.CustomerName, err = text(p.CustomerName()); err != nil {
		return inv, err
	}
	inv.OrderID, _ = p.OrderId()
	inv.CustomerPONumber, _ = p.CustomerPoNumber()
	inv.GrandTotalBHD = p.GrandTotalBhd()
	inv.Status = statusText(p.Status())
	inv.OutstandingBHD = p.OutstandingBhd()
	inv.SubtotalBHD = p.SubtotalBhd()
	if s, err := p.DueDate(); err == nil {
		inv.DueDate = adapter.TextToTime(s)
	}
	inv.UpdatedBy, _ = p.UpdatedBy()
	inv.RfqID, _ = p.RfqId()
	inv.QuoteID, _ = p.QuoteId()
	inv.OfferID, _ = p.OfferId()
	inv.OfferNumber, _ = p.OfferNumber()
	inv.DeliveryNoteID, _ = p.DeliveryNoteId()
	inv.DeliveryNoteNumber, _ = p.DeliveryNoteNumber()
	inv.TotalSupplierCostBHD = p.TotalSupplierCostBhd()
	inv.GrossMarginBHD = p.GrossMarginBhd()
	inv.GrossMarginPercent = p.GrossMarginPercent()
	inv.CustomerReference, _ = p.CustomerReference()
	inv.AttentionPerson, _ = p.AttentionPerson()
	inv.AttentionCompany, _ = p.AttentionCompany()
	inv.AttentionPhone, _ = p.AttentionPhone()
	inv.AttentionAddress, _ = p.AttentionAddress()
	inv.DeliveryWeeks, _ = p.DeliveryWeeks()
	inv.CountryOfOrigin, _ = p.CountryOfOrigin()
	inv.IssuedBy, _ = p.IssuedBy()
	inv.ContactPhone, _ = p.ContactPhone()
	inv.DiscountPercent = p.DiscountPercent()
	inv.PaymentTerms, _ = p.PaymentTerms()
	inv.DeliveryTerms, _ = p.DeliveryTerms()
	inv.Division, _ = p.Division()
	inv.FieldVisibility, _ = p.FieldVisibility()
	inv.DeliveryNoteRef, _ = p.DeliveryNoteRef()
	inv.ModeOfPayment, _ = p.ModeOfPayment()
	inv.SuppliersRef, _ = p.SuppliersRef()
	inv.OtherReferences, _ = p.OtherReferences()
	inv.BuyersOrderNumber, _ = p.BuyersOrderNumber()
	if s, err := p.BuyersOrderDate(); err == nil {
		inv.BuyersOrderDate = adapter.TextToTimePtr(s)
	}
	inv.DespatchDocumentNo, _ = p.DespatchDocumentNo()
	if s, err := p.DeliveryNoteDate(); err == nil {
		inv.DeliveryNoteDate = adapter.TextToTimePtr(s)
	}
	inv.DespatchedThrough, _ = p.DespatchedThrough()
	inv.Destination, _ = p.Destination()
	inv.PlaceOfSupply, _ = p.PlaceOfSupply()
	inv.TermsOfDelivery, _ = p.TermsOfDelivery()
	inv.VATBHD = p.VatBhd()
	inv.VATPercent = p.VatPercent()
	inv.JournalEntryID, _ = p.JournalEntryId()
	inv.InvoiceHash, _ = p.InvoiceHash()
	if p.HasItems() {
		items, err := p.Items()
		if err != nil {
			return inv, err
		}
		inv.Items = make([]gormfinance.DBInvoiceItem, items.Len())
		for i := 0; i < items.Len(); i++ {
			inv.Items[i], err = DBInvoiceItemFromProto(items.At(i))
			if err != nil {
				return inv, err
			}
		}
	}
	return inv, nil
}

func DBInvoiceItemToProto(item gormfinance.DBInvoiceItem) (*protofinance.DBInvoiceItem, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protofinance.NewDBInvoiceItem(seg)
	if err != nil {
		return nil, err
	}
	if err := populateDBInvoiceItem(seg, p, item); err != nil {
		return nil, err
	}
	return &p, nil
}

func populateDBInvoiceItem(seg *capnp.Segment, p protofinance.DBInvoiceItem, item gormfinance.DBInvoiceItem) error {
	if err := setBase(seg, p.SetBase, item.Base); err != nil {
		return err
	}
	setters := []error{
		p.SetInvoiceId(item.InvoiceID),
		p.SetDescription(item.Description),
		p.SetProductId(item.ProductID),
		p.SetProductCode(item.ProductCode),
		p.SetEquipment(item.Equipment),
		p.SetModel(item.Model),
		p.SetSpecification(item.Specification),
		p.SetDetailedDescription(item.DetailedDescription),
	}
	for _, err := range setters {
		if err != nil {
			return err
		}
	}
	p.SetLineNumber(int64(item.LineNumber))
	p.SetQuantity(item.Quantity)
	p.SetRate(item.Rate)
	p.SetTotalBhd(item.TotalBHD)
	p.SetCurrency(currencyCode(item.Currency))
	p.SetFob(item.FOB)
	p.SetFreight(item.Freight)
	p.SetTotalCost(item.TotalCost)
	p.SetMarginPercent(item.MarginPercent)
	p.SetTotalPrice(item.TotalPrice)
	return nil
}

func DBInvoiceItemFromProto(p protofinance.DBInvoiceItem) (gormfinance.DBInvoiceItem, error) {
	base, err := p.Base()
	if err != nil {
		return gormfinance.DBInvoiceItem{}, err
	}
	sharedBase, err := adapter.BaseFromProto(base)
	if err != nil {
		return gormfinance.DBInvoiceItem{}, err
	}
	item := gormfinance.DBInvoiceItem{Base: sharedBase}
	item.InvoiceID, _ = p.InvoiceId()
	item.LineNumber = int(p.LineNumber())
	item.Description, _ = p.Description()
	item.Quantity = p.Quantity()
	item.Rate = p.Rate()
	item.TotalBHD = p.TotalBhd()
	item.ProductID, _ = p.ProductId()
	item.ProductCode, _ = p.ProductCode()
	item.Equipment, _ = p.Equipment()
	item.Model, _ = p.Model()
	item.Specification, _ = p.Specification()
	item.DetailedDescription, _ = p.DetailedDescription()
	item.Currency = currencyText(p.Currency())
	item.FOB = p.Fob()
	item.Freight = p.Freight()
	item.TotalCost = p.TotalCost()
	item.MarginPercent = p.MarginPercent()
	item.TotalPrice = p.TotalPrice()
	return item, nil
}

func PaymentToProto(m gormfinance.Payment) (*protofinance.Payment, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protofinance.NewPayment(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetInvoiceId(m.InvoiceID), p.SetInvoiceNumber(m.InvoiceNumber), p.SetPaymentDate(adapter.TimeToText(m.PaymentDate)), p.SetIdempotencyKey(m.IdempotencyKey), p.SetJournalEntryId(m.JournalEntryID), p.SetBankAccountId(m.BankAccountID), p.SetReference(m.Reference), p.SetDivision(m.Division), p.SetUpdatedBy(m.UpdatedBy)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetAmountBhd(m.AmountBHD)
	p.SetPaymentMethod(paymentMethod(m.PaymentMethod))
	p.SetDaysToPayment(int64(m.DaysToPayment))
	return &p, nil
}

func PaymentFromProto(p protofinance.Payment) (gormfinance.Payment, error) {
	base, err := p.Base()
	if err != nil {
		return gormfinance.Payment{}, err
	}
	sharedBase, err := adapter.BaseFromProto(base)
	if err != nil {
		return gormfinance.Payment{}, err
	}
	m := gormfinance.Payment{Base: sharedBase}
	m.InvoiceID, _ = p.InvoiceId()
	m.InvoiceNumber, _ = p.InvoiceNumber()
	m.AmountBHD = p.AmountBhd()
	if s, err := p.PaymentDate(); err == nil {
		m.PaymentDate = adapter.TextToTime(s)
	}
	m.PaymentMethod = paymentMethodText(p.PaymentMethod())
	m.DaysToPayment = int(p.DaysToPayment())
	m.IdempotencyKey, _ = p.IdempotencyKey()
	m.JournalEntryID, _ = p.JournalEntryId()
	m.BankAccountID, _ = p.BankAccountId()
	m.Reference, _ = p.Reference()
	m.Division, _ = p.Division()
	m.UpdatedBy, _ = p.UpdatedBy()
	return m, nil
}

func CompanyBankAccountToProto(m gormfinance.CompanyBankAccount) (*protofinance.CompanyBankAccount, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protofinance.NewCompanyBankAccount(seg)
	if err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetId(m.ID), p.SetDivision(m.Division), p.SetBankName(m.BankName), p.SetAccountName(m.AccountName), p.SetAccountNumber(m.AccountNumber), p.SetIban(m.IBAN), p.SetSwiftBic(m.SwiftBIC), p.SetCreatedAt(adapter.TimeToText(m.CreatedAt)), p.SetUpdatedAt(adapter.TimeToText(m.UpdatedAt))} {
		if err != nil {
			return nil, err
		}
	}
	p.SetCurrency(currencyCode(m.Currency))
	p.SetIsActive(m.IsActive)
	p.SetDisplayOrder(int64(m.DisplayOrder))
	p.SetBookingRate(m.BookingRate)
	return &p, nil
}

func CompanyBankAccountFromProto(p protofinance.CompanyBankAccount) (gormfinance.CompanyBankAccount, error) {
	m := gormfinance.CompanyBankAccount{}
	m.ID, _ = p.Id()
	m.Division, _ = p.Division()
	m.BankName, _ = p.BankName()
	m.AccountName, _ = p.AccountName()
	m.AccountNumber, _ = p.AccountNumber()
	m.IBAN, _ = p.Iban()
	m.SwiftBIC, _ = p.SwiftBic()
	m.Currency = currencyText(p.Currency())
	m.IsActive = p.IsActive()
	m.DisplayOrder = int(p.DisplayOrder())
	m.BookingRate = p.BookingRate()
	if s, err := p.CreatedAt(); err == nil {
		m.CreatedAt = adapter.TextToTime(s)
	}
	if s, err := p.UpdatedAt(); err == nil {
		m.UpdatedAt = adapter.TextToTime(s)
	}
	return m, nil
}

func PurchaseOrderToProto(m gormfinance.PurchaseOrder) (*protofinance.PurchaseOrder, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protofinance.NewPurchaseOrder(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetOrderId(m.OrderID), p.SetRfqId(m.RfqID), p.SetSupplierId(m.SupplierID), p.SetSupplierName(m.SupplierName), p.SetPoNumber(m.PONumber), p.SetPoDate(adapter.TimeToText(m.PODate)), p.SetExpectedDelivery(adapter.TimeToText(m.ExpectedDelivery)), p.SetPaymentTerms(m.PaymentTerms), p.SetPaymentDueDate(adapter.TimeToText(m.PaymentDueDate)), p.SetApprovedBy(m.ApprovedBy), p.SetApprovedAt(adapter.TimePtrToText(m.ApprovedAt)), p.SetUpdatedBy(m.UpdatedBy), p.SetDivision(m.Division)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetCurrency(currencyCode(m.Currency))
	p.SetExchangeRate(m.ExchangeRate)
	p.SetSubtotalForeign(m.SubtotalForeign)
	p.SetSubtotalBhd(m.SubtotalBHD)
	p.SetVatAmount(m.VATAmount)
	p.SetTotalForeign(m.TotalForeign)
	p.SetTotalBhd(m.TotalBHD)
	p.SetStatus(documentStatus(m.Status))
	return &p, nil
}

func PurchaseOrderFromProto(p protofinance.PurchaseOrder) (gormfinance.PurchaseOrder, error) {
	base, err := p.Base()
	if err != nil {
		return gormfinance.PurchaseOrder{}, err
	}
	sharedBase, err := adapter.BaseFromProto(base)
	if err != nil {
		return gormfinance.PurchaseOrder{}, err
	}
	m := gormfinance.PurchaseOrder{Base: sharedBase}
	m.OrderID, _ = p.OrderId()
	m.RfqID, _ = p.RfqId()
	m.SupplierID, _ = p.SupplierId()
	m.SupplierName, _ = p.SupplierName()
	m.PONumber, _ = p.PoNumber()
	if s, err := p.PoDate(); err == nil {
		m.PODate = adapter.TextToTime(s)
	}
	if s, err := p.ExpectedDelivery(); err == nil {
		m.ExpectedDelivery = adapter.TextToTime(s)
	}
	m.Currency = currencyText(p.Currency())
	m.ExchangeRate = p.ExchangeRate()
	m.SubtotalForeign = p.SubtotalForeign()
	m.SubtotalBHD = p.SubtotalBhd()
	m.VATAmount = p.VatAmount()
	m.TotalForeign = p.TotalForeign()
	m.TotalBHD = p.TotalBhd()
	m.PaymentTerms, _ = p.PaymentTerms()
	if s, err := p.PaymentDueDate(); err == nil {
		m.PaymentDueDate = adapter.TextToTime(s)
	}
	m.Status = statusText(p.Status())
	m.ApprovedBy, _ = p.ApprovedBy()
	if s, err := p.ApprovedAt(); err == nil {
		m.ApprovedAt = adapter.TextToTimePtr(s)
	}
	m.UpdatedBy, _ = p.UpdatedBy()
	m.Division, _ = p.Division()
	return m, nil
}

func PurchaseOrderItemToProto(m gormfinance.PurchaseOrderItem) (*protofinance.PurchaseOrderItem, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protofinance.NewPurchaseOrderItem(seg)
	if err != nil {
		return nil, err
	}
	if err := populatePurchaseOrderItem(seg, p, m); err != nil {
		return nil, err
	}
	return &p, nil
}

func populatePurchaseOrderItem(seg *capnp.Segment, p protofinance.PurchaseOrderItem, m gormfinance.PurchaseOrderItem) error {
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return err
	}
	for _, err := range []error{p.SetPurchaseOrderId(m.PurchaseOrderID), p.SetOrderItemId(m.OrderItemID), p.SetProductId(m.ProductID), p.SetProductCode(m.ProductCode), p.SetSupplierPartNumber(m.SupplierPartNumber), p.SetDescription(m.Description)} {
		if err != nil {
			return err
		}
	}
	p.SetQuantity(m.Quantity)
	p.SetUnitPriceForeign(m.UnitPriceForeign)
	p.SetUnitPriceBhd(m.UnitPriceBHD)
	p.SetTotalForeign(m.TotalForeign)
	p.SetTotalBhd(m.TotalBHD)
	p.SetQuantityReceived(m.QuantityReceived)
	return nil
}

func PurchaseOrderItemFromProto(p protofinance.PurchaseOrderItem) (gormfinance.PurchaseOrderItem, error) {
	base, err := p.Base()
	if err != nil {
		return gormfinance.PurchaseOrderItem{}, err
	}
	sharedBase, err := adapter.BaseFromProto(base)
	if err != nil {
		return gormfinance.PurchaseOrderItem{}, err
	}
	m := gormfinance.PurchaseOrderItem{Base: sharedBase}
	m.PurchaseOrderID, _ = p.PurchaseOrderId()
	m.OrderItemID, _ = p.OrderItemId()
	m.ProductID, _ = p.ProductId()
	m.ProductCode, _ = p.ProductCode()
	m.SupplierPartNumber, _ = p.SupplierPartNumber()
	m.Description, _ = p.Description()
	m.Quantity = p.Quantity()
	m.UnitPriceForeign = p.UnitPriceForeign()
	m.UnitPriceBHD = p.UnitPriceBhd()
	m.TotalForeign = p.TotalForeign()
	m.TotalBHD = p.TotalBhd()
	m.QuantityReceived = p.QuantityReceived()
	return m, nil
}

func BankStatementToProto(m gormfinance.BankStatement) (*protofinance.BankStatement, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protofinance.NewBankStatement(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetBankAccountId(m.BankAccountID), p.SetStatementNumber(m.StatementNumber), p.SetStatementDate(adapter.TimeToText(m.StatementDate)), p.SetPeriodStart(adapter.TimeToText(m.PeriodStart)), p.SetPeriodEnd(adapter.TimeToText(m.PeriodEnd)), p.SetReconciledAt(adapter.TimePtrToText(m.ReconciledAt)), p.SetReconciledBy(m.ReconciledBy), p.SetImportedFrom(m.ImportedFrom), p.SetImportMethod(m.ImportMethod), p.SetNotes(m.Notes), p.SetDivision(m.Division)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetOpeningBalance(m.OpeningBalance)
	p.SetClosingBalance(m.ClosingBalance)
	p.SetCurrency(currencyCode(m.Currency))
	p.SetTotalDebits(m.TotalDebits)
	p.SetTotalCredits(m.TotalCredits)
	p.SetDebitCount(int64(m.DebitCount))
	p.SetCreditCount(int64(m.CreditCount))
	p.SetStatus(documentStatus(m.Status))
	p.SetOcrConfidence(m.OCRConfidence)
	p.SetBalanceVerified(m.BalanceVerified)
	p.SetDiscrepancyAmount(m.DiscrepancyAmount)
	return &p, nil
}

func BankStatementFromProto(p protofinance.BankStatement) (gormfinance.BankStatement, error) {
	base, err := p.Base()
	if err != nil {
		return gormfinance.BankStatement{}, err
	}
	sharedBase, err := adapter.BaseFromProto(base)
	if err != nil {
		return gormfinance.BankStatement{}, err
	}
	m := gormfinance.BankStatement{Base: sharedBase}
	m.BankAccountID, _ = p.BankAccountId()
	m.StatementNumber, _ = p.StatementNumber()
	if s, err := p.StatementDate(); err == nil {
		m.StatementDate = adapter.TextToTime(s)
	}
	m.OpeningBalance = p.OpeningBalance()
	m.ClosingBalance = p.ClosingBalance()
	m.Currency = currencyText(p.Currency())
	m.Status = statusText(p.Status())
	m.Division, _ = p.Division()
	return m, nil
}

func BankStatementLineToProto(m gormfinance.BankStatementLine) (*protofinance.BankStatementLine, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protofinance.NewBankStatementLine(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetBankStatementId(m.BankStatementID), p.SetTransactionDate(adapter.TimeToText(m.TransactionDate)), p.SetValueDate(adapter.TimeToText(m.ValueDate)), p.SetDescription(m.Description), p.SetReference(m.Reference), p.SetCategory(m.Category), p.SetSubCategory(m.SubCategory), p.SetExtractedCustomer(m.ExtractedCustomer), p.SetExtractedSupplier(m.ExtractedSupplier), p.SetExtractedInvoices(m.ExtractedInvoices), p.SetExtractedPoNumbers(m.ExtractedPONumbers), p.SetMatchedPaymentId(m.MatchedPaymentID), p.SetMatchedJournalId(m.MatchedJournalID), p.SetMatchedInvoiceIds(m.MatchedInvoiceIDs), p.SetMatchedExpenseId(stringPtrValue(m.MatchedExpenseID)), p.SetVerifiedBy(m.VerifiedBy), p.SetVerifiedAt(adapter.TimePtrToText(m.VerifiedAt)), p.SetNotes(m.Notes)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetLineNumber(int64(m.LineNumber))
	p.SetDebit(m.Debit)
	p.SetCredit(m.Credit)
	p.SetBalance(m.Balance)
	p.SetIsMatched(m.IsMatched)
	p.SetMatchConfidence(m.MatchConfidence)
	return &p, nil
}

func BankStatementLineFromProto(p protofinance.BankStatementLine) (gormfinance.BankStatementLine, error) {
	base, err := p.Base()
	if err != nil {
		return gormfinance.BankStatementLine{}, err
	}
	sharedBase, err := adapter.BaseFromProto(base)
	if err != nil {
		return gormfinance.BankStatementLine{}, err
	}
	m := gormfinance.BankStatementLine{Base: sharedBase}
	m.BankStatementID, _ = p.BankStatementId()
	m.LineNumber = int(p.LineNumber())
	if s, err := p.TransactionDate(); err == nil {
		m.TransactionDate = adapter.TextToTime(s)
	}
	m.Description, _ = p.Description()
	m.Reference, _ = p.Reference()
	m.Debit = p.Debit()
	m.Credit = p.Credit()
	m.Balance = p.Balance()
	return m, nil
}

func stringPtrValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// The following converters cover additional finance types used by screens and reports.
// They intentionally map stable identity/amount/status fields first; unmapped
// persistence-only fields remain in GORM until a consumer requires them.

func CreditNoteToProto(m gormfinance.CreditNote) (*protofinance.CreditNote, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protofinance.NewCreditNote(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetCreditNoteNumber(m.CreditNoteNumber), p.SetCreditNoteDate(adapter.TimeToText(m.CreditNoteDate)), p.SetInvoiceId(m.InvoiceID), p.SetInvoiceNumber(m.InvoiceNumber), p.SetCustomerId(m.CustomerID), p.SetCustomerName(m.CustomerName), p.SetReason(m.Reason), p.SetDivision(m.Division), p.SetAppliedAt(adapter.TimePtrToText(m.AppliedAt)), p.SetCreditNoteHash(m.CreditNoteHash)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetSubtotalBhd(m.SubtotalBHD)
	p.SetVatBhd(m.VATBHD)
	p.SetVatPercent(m.VATPercent)
	p.SetGrandTotalBhd(m.GrandTotalBHD)
	p.SetStatus(documentStatus(m.Status))
	return &p, nil
}

func CreditNoteFromProto(p protofinance.CreditNote) (gormfinance.CreditNote, error) {
	base, err := p.Base()
	if err != nil {
		return gormfinance.CreditNote{}, err
	}
	sharedBase, err := adapter.BaseFromProto(base)
	if err != nil {
		return gormfinance.CreditNote{}, err
	}
	m := gormfinance.CreditNote{Base: sharedBase}
	m.CreditNoteNumber, _ = p.CreditNoteNumber()
	m.InvoiceID, _ = p.InvoiceId()
	m.CustomerID, _ = p.CustomerId()
	m.GrandTotalBHD = p.GrandTotalBhd()
	m.Status = statusText(p.Status())
	return m, nil
}

func CreditNoteItemToProto(m gormfinance.CreditNoteItem) (*protofinance.CreditNoteItem, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protofinance.NewCreditNoteItem(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetCreditNoteId(m.CreditNoteID), p.SetDescription(m.Description)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetLineNumber(int64(m.LineNumber))
	p.SetQuantity(m.Quantity)
	p.SetRate(m.Rate)
	p.SetTotalBhd(m.TotalBHD)
	return &p, nil
}

func CreditNoteItemFromProto(p protofinance.CreditNoteItem) (gormfinance.CreditNoteItem, error) {
	m := gormfinance.CreditNoteItem{}
	m.CreditNoteID, _ = p.CreditNoteId()
	m.LineNumber = int(p.LineNumber())
	m.Description, _ = p.Description()
	m.Quantity = p.Quantity()
	m.Rate = p.Rate()
	m.TotalBHD = p.TotalBhd()
	return m, nil
}

func SupplierInvoiceToProto(m gormfinance.SupplierInvoice) (*protofinance.SupplierInvoice, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protofinance.NewSupplierInvoice(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetSupplierId(m.SupplierID), p.SetSupplierName(m.SupplierName), p.SetPurchaseOrderId(m.PurchaseOrderID), p.SetPoNumber(m.PONumber), p.SetGrnId(m.GRNID), p.SetOrderId(m.OrderID), p.SetInvoiceNumber(m.InvoiceNumber), p.SetInvoiceDate(adapter.TimeToText(m.InvoiceDate)), p.SetDueDate(adapter.TimeToText(m.DueDate)), p.SetApprovedBy(m.ApprovedBy), p.SetApprovedAt(adapter.TimePtrToText(m.ApprovedAt)), p.SetUpdatedBy(m.UpdatedBy), p.SetPaymentDate(adapter.TimePtrToText(m.PaymentDate)), p.SetPaymentRef(m.PaymentRef), p.SetOcrDocumentId(m.OCRDocumentID), p.SetDivision(m.Division), p.SetDiscrepancyReason(m.DiscrepancyReason), p.SetDisputeReason(m.DisputeReason), p.SetJournalEntryId(m.JournalEntryID)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetCurrency(currencyCode(m.Currency))
	p.SetExchangeRate(m.ExchangeRate)
	p.SetSubtotalForeign(m.SubtotalForeign)
	p.SetSubtotalBhd(m.SubtotalBHD)
	p.SetVatForeign(m.VATForeign)
	p.SetVatBhd(m.VATBHD)
	p.SetTotalForeign(m.TotalForeign)
	p.SetTotalBhd(m.TotalBHD)
	p.SetPoMatchOk(m.POMatchOK)
	p.SetGrnMatchOk(m.GRNMatchOK)
	p.SetStatus(documentStatus(m.Status))
	p.SetPaymentStatus(documentStatus(m.PaymentStatus))
	p.SetPaymentMethod(paymentMethod(m.PaymentMethod))
	p.SetOcrConfidence(m.OCRConfidence)
	return &p, nil
}

func SupplierInvoiceFromProto(p protofinance.SupplierInvoice) (gormfinance.SupplierInvoice, error) {
	base, err := p.Base()
	if err != nil {
		return gormfinance.SupplierInvoice{}, err
	}
	sharedBase, err := adapter.BaseFromProto(base)
	if err != nil {
		return gormfinance.SupplierInvoice{}, err
	}
	m := gormfinance.SupplierInvoice{Base: sharedBase}
	m.SupplierID, _ = p.SupplierId()
	m.SupplierName, _ = p.SupplierName()
	m.InvoiceNumber, _ = p.InvoiceNumber()
	m.TotalBHD = p.TotalBhd()
	m.Status = statusText(p.Status())
	return m, nil
}

func SupplierInvoiceItemToProto(m gormfinance.SupplierInvoiceItem) (*protofinance.SupplierInvoiceItem, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protofinance.NewSupplierInvoiceItem(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetSupplierInvoiceId(m.SupplierInvoiceID), p.SetDescription(m.Description)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetLineNumber(int64(m.LineNumber))
	p.SetQuantity(m.Quantity)
	p.SetUnitPrice(m.UnitPrice)
	p.SetTotalPrice(m.TotalPrice)
	p.SetCurrency(currencyCode(m.Currency))
	return &p, nil
}

func SupplierInvoiceItemFromProto(p protofinance.SupplierInvoiceItem) (gormfinance.SupplierInvoiceItem, error) {
	m := gormfinance.SupplierInvoiceItem{}
	m.SupplierInvoiceID, _ = p.SupplierInvoiceId()
	m.LineNumber = int(p.LineNumber())
	m.Description, _ = p.Description()
	m.Quantity = p.Quantity()
	m.UnitPrice = p.UnitPrice()
	m.TotalPrice = p.TotalPrice()
	m.Currency = currencyText(p.Currency())
	return m, nil
}

func SupplierPaymentToProto(m gormfinance.SupplierPayment) (*protofinance.SupplierPayment, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protofinance.NewSupplierPayment(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetSupplierInvoiceId(m.SupplierInvoiceID), p.SetSupplierId(m.SupplierID), p.SetPaymentDate(adapter.TimeToText(m.PaymentDate)), p.SetReference(m.Reference), p.SetNotes(m.Notes), p.SetJournalEntryId(m.JournalEntryID), p.SetBankAccountId(m.BankAccountID), p.SetUpdatedBy(m.UpdatedBy), p.SetDivision(m.Division), p.SetSupplierName(m.SupplierName), p.SetInvoiceNumber(m.InvoiceNumber)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetAmountForeign(m.AmountForeign)
	p.SetCurrency(currencyCode(m.Currency))
	p.SetExchangeRate(m.ExchangeRate)
	p.SetAmountBhd(m.AmountBHD)
	p.SetPaymentMethod(paymentMethod(m.PaymentMethod))
	return &p, nil
}

func SupplierPaymentFromProto(p protofinance.SupplierPayment) (gormfinance.SupplierPayment, error) {
	m := gormfinance.SupplierPayment{}
	m.SupplierInvoiceID, _ = p.SupplierInvoiceId()
	m.SupplierID, _ = p.SupplierId()
	m.AmountBHD = p.AmountBhd()
	m.PaymentMethod = paymentMethodText(p.PaymentMethod())
	m.Reference, _ = p.Reference()
	return m, nil
}

func ChartOfAccountToProto(m gormfinance.ChartOfAccount) (*protofinance.ChartOfAccount, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protofinance.NewChartOfAccount(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetAccountCode(m.AccountCode), p.SetAccountName(m.AccountName), p.SetParentAccountId(m.ParentAccountID), p.SetAccountGroup(m.AccountGroup)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetBalance(m.Balance)
	p.SetIsActive(m.IsActive)
	p.SetIsVatAccount(m.IsVATAccount)
	return &p, nil
}

func ChartOfAccountFromProto(p protofinance.ChartOfAccount) (gormfinance.ChartOfAccount, error) {
	m := gormfinance.ChartOfAccount{}
	m.AccountCode, _ = p.AccountCode()
	m.AccountName, _ = p.AccountName()
	m.Balance = p.Balance()
	m.IsActive = p.IsActive()
	return m, nil
}

func JournalEntryToProto(m gormfinance.JournalEntry) (*protofinance.JournalEntry, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protofinance.NewJournalEntry(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetEntryNumber(m.EntryNumber), p.SetEntryDate(adapter.TimeToText(m.EntryDate)), p.SetDescription(m.Description), p.SetPostedAt(adapter.TimePtrToText(m.PostedAt)), p.SetPostedBy(m.PostedBy), p.SetSourceType(m.SourceType), p.SetSourceId(m.SourceID), p.SetReversedById(m.ReversedByID), p.SetReversesId(m.ReversesID), p.SetUpdatedBy(m.UpdatedBy)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetDebitTotal(m.DebitTotal)
	p.SetCreditTotal(m.CreditTotal)
	p.SetIsPosted(m.IsPosted)
	p.SetFiscalYear(int64(m.FiscalYear))
	p.SetFiscalPeriod(int64(m.FiscalPeriod))
	p.SetIsAutoGenerated(m.IsAutoGenerated)
	return &p, nil
}

func JournalEntryFromProto(p protofinance.JournalEntry) (gormfinance.JournalEntry, error) {
	m := gormfinance.JournalEntry{}
	m.EntryNumber, _ = p.EntryNumber()
	m.Description, _ = p.Description()
	m.DebitTotal = p.DebitTotal()
	m.CreditTotal = p.CreditTotal()
	m.IsPosted = p.IsPosted()
	m.FiscalYear = int(p.FiscalYear())
	return m, nil
}

func JournalLineToProto(m gormfinance.JournalLine) (*protofinance.JournalLine, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protofinance.NewJournalLine(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetEntryId(m.EntryID), p.SetAccountId(m.AccountID), p.SetAccountName(m.AccountName), p.SetDescription(m.Description), p.SetUpdatedBy(m.UpdatedBy)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetDebit(m.Debit)
	p.SetCredit(m.Credit)
	return &p, nil
}

func JournalLineFromProto(p protofinance.JournalLine) (gormfinance.JournalLine, error) {
	m := gormfinance.JournalLine{}
	m.EntryID, _ = p.EntryId()
	m.AccountID, _ = p.AccountId()
	m.AccountName, _ = p.AccountName()
	m.Debit = p.Debit()
	m.Credit = p.Credit()
	return m, nil
}

func VATReturnToProto(m gormfinance.VATReturn) (*protofinance.VATReturn, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protofinance.NewVATReturn(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetReturnNumber(m.ReturnNumber), p.SetPeriodStart(adapter.TimeToText(m.PeriodStart)), p.SetPeriodEnd(adapter.TimeToText(m.PeriodEnd))} {
		if err != nil {
			return nil, err
		}
	}
	p.SetFiscalYear(int64(m.FiscalYear))
	p.SetQuarter(int64(m.Quarter))
	p.SetNetVat(m.NetVAT)
	p.SetStatus(documentStatus(m.Status))
	return &p, nil
}

func VATReturnFromProto(p protofinance.VATReturn) (gormfinance.VATReturn, error) {
	m := gormfinance.VATReturn{}
	m.ReturnNumber, _ = p.ReturnNumber()
	m.NetVAT = p.NetVat()
	m.Status = statusText(p.Status())
	return m, nil
}

func unsupportedGap(typeName, field string) error {
	return fmt.Errorf("%s.%s has no stable Wave 9 proto mapping", typeName, field)
}
