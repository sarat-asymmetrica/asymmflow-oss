// Package procurement contains the concrete purchase order and GRN service implementation.
package procurement

import "gorm.io/gorm"

type Handlers[PurchaseOrderModel any, GoodsReceivedNoteModel any, GRNItemModel any] struct {
	CreatePurchaseOrder  func(po PurchaseOrderModel) (PurchaseOrderModel, error)
	GetPurchaseOrders    func() ([]PurchaseOrderModel, error)
	GetPurchaseOrderByID func(id string) (PurchaseOrderModel, error)
	UpdatePurchaseOrder  func(po PurchaseOrderModel) (PurchaseOrderModel, error)
	ApprovePurchaseOrder func(id, approvedBy string) error
	CreatePOFromOrder    func(orderID, supplierID string, itemIDs []string) (PurchaseOrderModel, error)
	CreateGRN            func(grn GoodsReceivedNoteModel) (GoodsReceivedNoteModel, error)
	ReceiveAgainstPO     func(poID string, items []GRNItemModel) (GoodsReceivedNoteModel, error)
}

type Service[PurchaseOrderModel any, GoodsReceivedNoteModel any, GRNItemModel any] struct {
	db       *gorm.DB
	handlers Handlers[PurchaseOrderModel, GoodsReceivedNoteModel, GRNItemModel]
}

func New[PurchaseOrderModel any, GoodsReceivedNoteModel any, GRNItemModel any](db *gorm.DB, handlers Handlers[PurchaseOrderModel, GoodsReceivedNoteModel, GRNItemModel]) *Service[PurchaseOrderModel, GoodsReceivedNoteModel, GRNItemModel] {
	return &Service[PurchaseOrderModel, GoodsReceivedNoteModel, GRNItemModel]{db: db, handlers: handlers}
}

func (s *Service[PurchaseOrderModel, GoodsReceivedNoteModel, GRNItemModel]) CreatePurchaseOrder(po PurchaseOrderModel) (PurchaseOrderModel, error) {
	return s.handlers.CreatePurchaseOrder(po)
}

func (s *Service[PurchaseOrderModel, GoodsReceivedNoteModel, GRNItemModel]) GetPurchaseOrders() ([]PurchaseOrderModel, error) {
	return s.handlers.GetPurchaseOrders()
}

func (s *Service[PurchaseOrderModel, GoodsReceivedNoteModel, GRNItemModel]) GetPurchaseOrderByID(id string) (PurchaseOrderModel, error) {
	return s.handlers.GetPurchaseOrderByID(id)
}

func (s *Service[PurchaseOrderModel, GoodsReceivedNoteModel, GRNItemModel]) UpdatePurchaseOrder(po PurchaseOrderModel) (PurchaseOrderModel, error) {
	return s.handlers.UpdatePurchaseOrder(po)
}

func (s *Service[PurchaseOrderModel, GoodsReceivedNoteModel, GRNItemModel]) ApprovePurchaseOrder(id, approvedBy string) error {
	return s.handlers.ApprovePurchaseOrder(id, approvedBy)
}

func (s *Service[PurchaseOrderModel, GoodsReceivedNoteModel, GRNItemModel]) CreatePOFromOrder(orderID, supplierID string, itemIDs []string) (PurchaseOrderModel, error) {
	return s.handlers.CreatePOFromOrder(orderID, supplierID, itemIDs)
}

func (s *Service[PurchaseOrderModel, GoodsReceivedNoteModel, GRNItemModel]) CreateGRN(grn GoodsReceivedNoteModel) (GoodsReceivedNoteModel, error) {
	return s.handlers.CreateGRN(grn)
}

func (s *Service[PurchaseOrderModel, GoodsReceivedNoteModel, GRNItemModel]) ReceiveAgainstPO(poID string, items []GRNItemModel) (GoodsReceivedNoteModel, error) {
	return s.handlers.ReceiveAgainstPO(poID, items)
}
