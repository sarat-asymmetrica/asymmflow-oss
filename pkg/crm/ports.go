// Package crm defines the CRM domain ports.
package crm

type CustomerRepository interface {
	ListCustomers(limit, offset int) ([]CustomerMaster, error)
	GetCustomer(id string) (CustomerMaster, error)
	CreateCustomer(customer CustomerMaster) (*CustomerMaster, error)
	UpdateCustomer(customer CustomerMaster) (*CustomerMaster, error)
	DeleteCustomer(id string) error
	ListCustomerContacts(customerID string) ([]CustomerContact, error)
	AddCustomerContact(contact CustomerContact) (*CustomerContact, error)
}

type OfferRepository interface {
	GetAllOffers() ([]Offer, error)
	FindOfferByReference(reference string) (*Offer, error)
	UpdateOfferStatus(id uint, status string) error
	GetOpportunityLineItems(opportunityID string) ([]OfferItem, error)
	AddOfferNote(offerID, content string) (OfferNote, error)
	GetOfferNotes(offerID string) ([]OfferNote, error)
	DeleteOfferNote(noteID string) error
}

type OrderRepository interface {
	CreateOrder(orderNumber, customerName string, amount float64, orderDateStr, status string) (*Order, error)
	ListOrders(limit, offset int) ([]Order, error)
	GetOrder(orderID string) (Order, error)
	UpdateOrder(id string, order Order) (*Order, error)
	DeleteOrder(orderID string) error
	FilterOrders(customerQuery, dateFrom, dateTo, status string, limit, offset int) ([]Order, error)
}

type PipelineService interface {
	GetPipelineOpportunities(limit, offset int) ([]Opportunity, error)
	UpdateOpportunityStage(opportunityID, stage string) error
	UpdateOpportunityDetails(opportunityID, comment, ownerNotes string) (*Opportunity, error)
	MarkOfferWon(offerID, customerPO string) (*Order, error)
	MarkOfferLost(offerID, reason string) error
	ListFollowUps(limit int) ([]FollowUpTask, error)
	CreateFollowUp(task FollowUpTask) (FollowUpTask, error)
	CompleteFollowUp(id string) error
}

type ProcurementService interface {
	CreatePurchaseOrder(po PurchaseOrder) (PurchaseOrder, error)
	GetPurchaseOrders() ([]PurchaseOrder, error)
	GetPurchaseOrderByID(id string) (PurchaseOrder, error)
	UpdatePurchaseOrder(po PurchaseOrder) (PurchaseOrder, error)
	ApprovePurchaseOrder(id, approvedBy string) error
	CreatePOFromOrder(orderID, supplierID string, itemIDs []string) (PurchaseOrder, error)
	CreateGRN(grn GoodsReceivedNote) (GoodsReceivedNote, error)
	ReceiveAgainstPO(poID string, items []GRNItem) (GoodsReceivedNote, error)
}

type FulfillmentService interface {
	CreateDeliveryNote(dn DeliveryNote) (DeliveryNote, error)
	GetDeliveryNotes() ([]DeliveryNote, error)
	GetDeliveryNoteByID(id string) (DeliveryNote, error)
	UpdateDeliveryNote(dn DeliveryNote) (DeliveryNote, error)
	DispatchDeliveryNote(id, driverName, vehicleNumber string) error
	// ConfirmDeliveryNote returns a non-fatal warning string (empty when clean)
	// alongside the error: the DN itself is confirmed transactionally, but
	// downstream order-progression steps can fail independently and are
	// surfaced here rather than silently swallowed.
	ConfirmDeliveryNote(id, signedBy string) (string, error)
	RegisterSerials(productID string, serialNos []string) ([]SerialNumber, error)
	GetAvailableSerials(productID string) ([]SerialNumber, error)
}
