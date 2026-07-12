package payment

import "gorm.io/gorm"

type Handlers[PaymentModel any, SupplierPaymentModel any] struct {
	RecordPayment                func(invoiceID string, amount float64, method string, dateStr string, reference string) (*PaymentModel, error)
	GetPaymentsByInvoice         func(invoiceID string) ([]PaymentModel, error)
	GetAllPayments               func(limit, offset int) ([]PaymentModel, error)
	GetPayment                   func(id string) (PaymentModel, error)
	UpdatePayment                func(id string, payment PaymentModel) (*PaymentModel, error)
	CheckOrderCompletion         func(orderID string) error
	ProgressOrderOnDelivery      func(orderID string) error
	ProgressOrderOnInvoice       func(orderID string) error
	RecordSupplierPayment        func(invoiceID string, amount float64, currency, method, date, reference string, exchangeRate float64) (*SupplierPaymentModel, error)
	GetSupplierPaymentsByInvoice func(invoiceID string) ([]SupplierPaymentModel, error)
	GetAllSupplierPayments       func() ([]SupplierPaymentModel, error)
	GetSupplierPaymentsSummary   func() (map[string]any, error)
	GetSupplierPayment           func(id string) (SupplierPaymentModel, error)
	UpdateSupplierPayment        func(id string, payment SupplierPaymentModel) (*SupplierPaymentModel, error)
}

type Service[PaymentModel any, SupplierPaymentModel any] struct {
	db       *gorm.DB
	cache    any
	handlers Handlers[PaymentModel, SupplierPaymentModel]
}

func New[PaymentModel any, SupplierPaymentModel any](db *gorm.DB, cache any, handlers Handlers[PaymentModel, SupplierPaymentModel]) *Service[PaymentModel, SupplierPaymentModel] {
	return &Service[PaymentModel, SupplierPaymentModel]{db: db, cache: cache, handlers: handlers}
}

func (s *Service[PaymentModel, SupplierPaymentModel]) RecordPayment(invoiceID string, amount float64, method string, dateStr string, reference string) (*PaymentModel, error) {
	return s.handlers.RecordPayment(invoiceID, amount, method, dateStr, reference)
}

func (s *Service[PaymentModel, SupplierPaymentModel]) GetPaymentsByInvoice(invoiceID string) ([]PaymentModel, error) {
	return s.handlers.GetPaymentsByInvoice(invoiceID)
}

func (s *Service[PaymentModel, SupplierPaymentModel]) GetAllPayments(limit, offset int) ([]PaymentModel, error) {
	return s.handlers.GetAllPayments(limit, offset)
}

func (s *Service[PaymentModel, SupplierPaymentModel]) GetPayment(id string) (PaymentModel, error) {
	return s.handlers.GetPayment(id)
}

func (s *Service[PaymentModel, SupplierPaymentModel]) UpdatePayment(id string, payment PaymentModel) (*PaymentModel, error) {
	return s.handlers.UpdatePayment(id, payment)
}

func (s *Service[PaymentModel, SupplierPaymentModel]) CheckOrderCompletion(orderID string) error {
	return s.handlers.CheckOrderCompletion(orderID)
}

func (s *Service[PaymentModel, SupplierPaymentModel]) ProgressOrderOnDelivery(orderID string) error {
	return s.handlers.ProgressOrderOnDelivery(orderID)
}

func (s *Service[PaymentModel, SupplierPaymentModel]) ProgressOrderOnInvoice(orderID string) error {
	return s.handlers.ProgressOrderOnInvoice(orderID)
}

func (s *Service[PaymentModel, SupplierPaymentModel]) RecordSupplierPayment(invoiceID string, amount float64, currency, method, date, reference string, exchangeRate float64) (*SupplierPaymentModel, error) {
	return s.handlers.RecordSupplierPayment(invoiceID, amount, currency, method, date, reference, exchangeRate)
}

func (s *Service[PaymentModel, SupplierPaymentModel]) GetSupplierPaymentsByInvoice(invoiceID string) ([]SupplierPaymentModel, error) {
	return s.handlers.GetSupplierPaymentsByInvoice(invoiceID)
}

func (s *Service[PaymentModel, SupplierPaymentModel]) GetAllSupplierPayments() ([]SupplierPaymentModel, error) {
	return s.handlers.GetAllSupplierPayments()
}

func (s *Service[PaymentModel, SupplierPaymentModel]) GetSupplierPaymentsSummary() (map[string]any, error) {
	return s.handlers.GetSupplierPaymentsSummary()
}

func (s *Service[PaymentModel, SupplierPaymentModel]) GetSupplierPayment(id string) (SupplierPaymentModel, error) {
	return s.handlers.GetSupplierPayment(id)
}

func (s *Service[PaymentModel, SupplierPaymentModel]) UpdateSupplierPayment(id string, payment SupplierPaymentModel) (*SupplierPaymentModel, error) {
	return s.handlers.UpdateSupplierPayment(id, payment)
}
