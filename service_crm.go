package main

import (
	"ph_holdings_app/pkg/data"
	"ph_holdings_app/pkg/engines"
	"ph_holdings_app/pkg/graph"
)

// CRMService exposes domain-specific Wails bindings by delegating to App.
type CRMService struct {
	app *App
}

func NewCRMService(app *App) *CRMService {
	return &CRMService{app: app}
}

// --- P1_SALES_PIPELINE_FIXES.go ---

func (s *CRMService) AutoExpireOffers() error {
	return s.app.AutoExpireOffers()
}

func (s *CRMService) CreateCostingSheetVersion(originalID string, updatedBy string) (*DBCostingSheet, error) {
	return s.app.CreateCostingSheetVersion(originalID, updatedBy)
}

func (s *CRMService) GetLowMarginOffers(threshold float64) ([]Offer, error) {
	return s.app.GetLowMarginOffers(threshold)
}

func (s *CRMService) GetRFQTraceability(orderID string) (*PipelineTraceability, error) {
	return s.app.GetRFQTraceability(orderID)
}

func (s *CRMService) LogMarginAlert(entityType string, entityID string, alert *MarginAlert) error {
	return s.app.LogMarginAlert(entityType, entityID, alert)
}

func (s *CRMService) UpdateCostingSheetWithVersionCheck(sheetID string, updates map[string]any, updatedBy string) error {
	return s.app.UpdateCostingSheetWithVersionCheck(sheetID, updates, updatedBy)
}

// --- app_crm_surface.go ---

func (s *CRMService) AddCustomerNote(customerID string, noteType string, content string) error {
	return s.app.AddCustomerNote(customerID, noteType, content)
}

func (s *CRMService) AddSupplierIssue(supplierID string, orderRef string, description string, costBHD float64) error {
	return s.app.AddSupplierIssue(supplierID, orderRef, description, costBHD)
}

func (s *CRMService) AddSupplierNote(supplierID string, noteType string, content string) error {
	return s.app.AddSupplierNote(supplierID, noteType, content)
}

func (s *CRMService) GetAllCustomers() ([]CustomerMaster, error) {
	return s.app.GetAllCustomers()
}

func (s *CRMService) GetCustomer360View(customerID string) (Customer360Data, error) {
	return s.app.GetCustomer360View(customerID)
}

func (s *CRMService) GetCustomerFullProfile(customerID string) (CustomerFullProfile, error) {
	return s.app.GetCustomerFullProfile(customerID)
}

func (s *CRMService) GetCustomerOpportunities(customerID string) []OpportunitySummary {
	return s.app.GetCustomerOpportunities(customerID)
}

func (s *CRMService) GetCustomerRecentOrders(customerID string, limit int) []OrderSummary {
	return s.app.GetCustomerRecentOrders(customerID, limit)
}

func (s *CRMService) GetCustomersByGrade(grade string) ([]CustomerMaster, error) {
	return s.app.GetCustomersByGrade(grade)
}

func (s *CRMService) GetPaymentHistory(customerID string, limit int) []PaymentHistoryEntry {
	return s.app.GetPaymentHistory(customerID, limit)
}

func (s *CRMService) GetReceivablesAging(customerID string) ReceivablesAgingSummary {
	return s.app.GetReceivablesAging(customerID)
}

func (s *CRMService) GetSupplierFullProfile(supplierID string) (SupplierFullProfile, error) {
	return s.app.GetSupplierFullProfile(supplierID)
}

func (s *CRMService) ResolveSupplierIssue(issueID string, resolution string) error {
	return s.app.ResolveSupplierIssue(issueID, resolution)
}

// --- app_graph_contract_surface.go ---

func (s *CRMService) BuildEntityGraph() (*graph.BuildStats, error) {
	return s.app.BuildEntityGraph()
}

func (s *CRMService) CompareWilliamsLinear(totalItems int) map[string]any {
	return s.app.CompareWilliamsLinear(totalItems)
}

func (s *CRMService) DownloadContract(contractID uint) (string, error) {
	return s.app.DownloadContract(contractID)
}

func (s *CRMService) ExportGraphJSON() (string, error) {
	return s.app.ExportGraphJSON()
}

func (s *CRMService) GenerateContract(customerID string, templateName, contractType string, valueBHD float64, orderID string) (*Contract, error) {
	return s.app.GenerateContract(customerID, templateName, contractType, valueBHD, orderID)
}

func (s *CRMService) GetContract(contractID uint) (*Contract, error) {
	return s.app.GetContract(contractID)
}

func (s *CRMService) GetContractTemplates() ([]ContractTemplate, error) {
	return s.app.GetContractTemplates()
}

func (s *CRMService) GetContracts(limit, offset int) ([]Contract, error) {
	return s.app.GetContracts(limit, offset)
}

func (s *CRMService) GetContractsByCustomer(customerID uint) ([]Contract, error) {
	return s.app.GetContractsByCustomer(customerID)
}

func (s *CRMService) GetCustomerGraph(customerID string, depth int) (*graph.GraphData, error) {
	return s.app.GetCustomerGraph(customerID, depth)
}

func (s *CRMService) GetEntityGraph(nodeType string, limit int) (*graph.GraphData, error) {
	return s.app.GetEntityGraph(nodeType, limit)
}

func (s *CRMService) GetGraphStats() (*graph.GraphStats, error) {
	return s.app.GetGraphStats()
}

func (s *CRMService) GetNodeRelationships(nodeID uint) ([]graph.GraphEdge, error) {
	return s.app.GetNodeRelationships(nodeID)
}

func (s *CRMService) GetSSOTImportStatus() map[string]any {
	return s.app.GetSSOTImportStatus()
}

func (s *CRMService) GetWilliamsMetrics(totalItems int) engines.WilliamsMetrics {
	return s.app.GetWilliamsMetrics(totalItems)
}

func (s *CRMService) ImportSSOTData() (*data.ImportResult, error) {
	return s.app.ImportSSOTData()
}

func (s *CRMService) RebuildEntityGraph() (*graph.BuildStats, error) {
	return s.app.RebuildEntityGraph()
}

func (s *CRMService) SearchGraphEntities(query string, limit int) ([]graph.GraphNode, error) {
	return s.app.SearchGraphEntities(query, limit)
}

func (s *CRMService) SeedContractData() error {
	return s.app.SeedContractData()
}

// --- app_order_customer_surface.go ---

func (s *CRMService) AddCustomerContact(contact CustomerContact) (*CustomerContact, error) {
	return s.app.AddCustomerContact(contact)
}

func (s *CRMService) AddSupplierContact(contact SupplierContact) (*SupplierContact, error) {
	return s.app.AddSupplierContact(contact)
}

func (s *CRMService) BackfillBusinessCustomerIDs() (map[string]any, error) {
	return s.app.BackfillBusinessCustomerIDs()
}

func (s *CRMService) CompleteFollowUp(id string) error {
	return s.app.CompleteFollowUp(id)
}

func (s *CRMService) ConfirmDelivery(shipmentID string) error {
	return s.app.ConfirmDelivery(shipmentID)
}

func (s *CRMService) CreateCustomer(customer CustomerMaster) (*CustomerMaster, error) {
	return s.app.CreateCustomer(customer)
}

func (s *CRMService) CreateFollowUp(task FollowUpTask) (FollowUpTask, error) {
	return s.app.CreateFollowUp(task)
}

func (s *CRMService) CreateOrder(orderNumber string, customerName string, amount float64, orderDateStr string, status string) (*Order, error) {
	return s.app.CreateOrder(orderNumber, customerName, amount, orderDateStr, status)
}

func (s *CRMService) CreateOrderWithItems(order Order, items []OrderItem) (*Order, error) {
	return s.app.CreateOrderWithItems(order, items)
}

func (s *CRMService) CreateReportDraft(reportType string, to string, params map[string]any) (string, error) {
	return s.app.CreateReportDraft(reportType, to, params)
}

func (s *CRMService) CreateShipment(orderIds []string, trackingNumber string, courier string, estimatedDelivery string, notes string) error {
	return s.app.CreateShipment(orderIds, trackingNumber, courier, estimatedDelivery, notes)
}

func (s *CRMService) CreateSupplier(supplier SupplierMaster) (*SupplierMaster, error) {
	return s.app.CreateSupplier(supplier)
}

func (s *CRMService) DeleteCustomer(id string) error {
	return s.app.DeleteCustomer(id)
}

func (s *CRMService) DeleteCustomerContact(contactID string) error {
	return s.app.DeleteCustomerContact(contactID)
}

func (s *CRMService) DeleteOrder(orderID string) error {
	return s.app.DeleteOrder(orderID)
}

func (s *CRMService) DeleteSupplier(id string) error {
	return s.app.DeleteSupplier(id)
}

func (s *CRMService) DeleteSupplierContact(contactID string) error {
	return s.app.DeleteSupplierContact(contactID)
}

func (s *CRMService) FilterOrders(customerQuery string, dateFrom string, dateTo string, status string, limit int, offset int) ([]Order, error) {
	return s.app.FilterOrders(customerQuery, dateFrom, dateTo, status, limit, offset)
}

func (s *CRMService) GetCRMSupplierDashboard() CRMSupplierDashboard {
	return s.app.GetCRMSupplierDashboard()
}

func (s *CRMService) GetCRMSupplierDashboardByYear(year int) CRMSupplierDashboard {
	return s.app.GetCRMSupplierDashboardByYear(year)
}

func (s *CRMService) GetCustomer(id string) (CustomerMaster, error) {
	return s.app.GetCustomer(id)
}

func (s *CRMService) GetCustomer360(customerID string) (Customer360Data, error) {
	return s.app.GetCustomer360(customerID)
}

func (s *CRMService) GetCustomer360Graph(customerID string) (Customer360Graph, error) {
	return s.app.GetCustomer360Graph(customerID)
}

func (s *CRMService) GetOrder(orderID string) (Order, error) {
	return s.app.GetOrder(orderID)
}

func (s *CRMService) GetOrderFulfillmentStatus(orderID string) (*FulfillmentStatus, error) {
	return s.app.GetOrderFulfillmentStatus(orderID)
}

func (s *CRMService) GetOverdueFollowUps() ([]FollowUpTask, error) {
	return s.app.GetOverdueFollowUps()
}

func (s *CRMService) GetSupplier(id string) (SupplierMaster, error) {
	return s.app.GetSupplier(id)
}

func (s *CRMService) ListCustomerContacts(customerID string) ([]CustomerContact, error) {
	return s.app.ListCustomerContacts(customerID)
}

func (s *CRMService) ListCustomers(limit int, offset int) ([]CustomerMaster, error) {
	return s.app.ListCustomers(limit, offset)
}

func (s *CRMService) ListFollowUps(limit int) ([]FollowUpTask, error) {
	return s.app.ListFollowUps(limit)
}

func (s *CRMService) ListOrders(limit int, offset int) ([]Order, error) {
	return s.app.ListOrders(limit, offset)
}

func (s *CRMService) ListShipments() ([]map[string]any, error) {
	return s.app.ListShipments()
}

func (s *CRMService) ListSupplierContacts(supplierID string) ([]SupplierContact, error) {
	return s.app.ListSupplierContacts(supplierID)
}

func (s *CRMService) ListSuppliers(limit int, offset int) ([]SupplierMaster, error) {
	return s.app.ListSuppliers(limit, offset)
}

func (s *CRMService) QuickMarkOrderDelivered(orderID string) (string, error) {
	return s.app.QuickMarkOrderDelivered(orderID)
}

func (s *CRMService) RecordPartialShipment(orderID string, shipments map[string]float64, trackingNumber, courier, notes string) error {
	return s.app.RecordPartialShipment(orderID, shipments, trackingNumber, courier, notes)
}

func (s *CRMService) SeedCustomerDatabase() error {
	return s.app.SeedCustomerDatabase()
}

func (s *CRMService) SendReportByEmail(reportType string, to string, params map[string]any) error {
	return s.app.SendReportByEmail(reportType, to, params)
}

func (s *CRMService) SimulateSurvivalGarden(cashRunway float64, monthlyBurn float64, expenses []map[string]any, months int) ([]map[string]any, error) {
	return s.app.SimulateSurvivalGarden(cashRunway, monthlyBurn, expenses, months)
}

func (s *CRMService) UpdateCustomer(customer CustomerMaster) (*CustomerMaster, error) {
	return s.app.UpdateCustomer(customer)
}

func (s *CRMService) UpdateCustomerContact(contact CustomerContact) (*CustomerContact, error) {
	return s.app.UpdateCustomerContact(contact)
}

func (s *CRMService) UpdateFollowUp(id uint, task FollowUpTask) error {
	return s.app.UpdateFollowUp(id, task)
}

func (s *CRMService) UpdateOrder(id string, order Order) (*Order, error) {
	return s.app.UpdateOrder(id, order)
}

func (s *CRMService) UpdateOrderItemInvoiced(itemID uint, quantityInvoiced float64) error {
	return s.app.UpdateOrderItemInvoiced(itemID, quantityInvoiced)
}

func (s *CRMService) UpdateOrderItemShipped(itemID uint, quantityShipped float64) error {
	return s.app.UpdateOrderItemShipped(itemID, quantityShipped)
}

func (s *CRMService) UpdateOrderStage(orderID string, stage string) error {
	return s.app.UpdateOrderStage(orderID, stage)
}

func (s *CRMService) UpdateShipment(shipmentID string, status, notes string) error {
	return s.app.UpdateShipment(shipmentID, status, notes)
}

func (s *CRMService) UpdateSupplier(supplier SupplierMaster) (*SupplierMaster, error) {
	return s.app.UpdateSupplier(supplier)
}

func (s *CRMService) UpdateSupplierContact(contact SupplierContact) (*SupplierContact, error) {
	return s.app.UpdateSupplierContact(contact)
}

func (s *CRMService) UpdateSupplierGoals(supplierID uint, targetAmount float64, currentAmount float64) error {
	return s.app.UpdateSupplierGoals(supplierID, targetAmount, currentAmount)
}

// --- app_sales_pipeline.go ---

func (s *CRMService) AddOfferNote(offerID string, content string) (OfferNote, error) {
	return s.app.AddOfferNote(offerID, content)
}

func (s *CRMService) AddOpportunityComment(opportunityID, comment string) (*OpportunityComment, error) {
	return s.app.AddOpportunityComment(opportunityID, comment)
}

func (s *CRMService) AddRFQComment(rfqID uint, comment, createdBy string) (*RFQComment, error) {
	return s.app.AddRFQComment(rfqID, comment, createdBy)
}

func (s *CRMService) ApproveCostingSheet(id uint, approvedBy string) error {
	return s.app.ApproveCostingSheet(id, approvedBy)
}

func (s *CRMService) BackfillWonOfferItemsFromOpportunityProductDetails() (map[string]any, error) {
	return s.app.BackfillWonOfferItemsFromOpportunityProductDetails()
}

func (s *CRMService) CheckDuplicateOpportunity(reference, customer, project string) (*Opportunity, bool, error) {
	return s.app.CheckDuplicateOpportunity(reference, customer, project)
}

func (s *CRMService) CheckDuplicateRFQ(customer, project, documentHash string) (*RFQData, bool, error) {
	return s.app.CheckDuplicateRFQ(customer, project, documentHash)
}

func (s *CRMService) CloneCostingAsNewRevision(sourceCostingID uint, preparedBy string) (*CostingSheetData, error) {
	return s.app.CloneCostingAsNewRevision(sourceCostingID, preparedBy)
}

func (s *CRMService) ConvertOfferToOrder(offerID uint) error {
	return s.app.ConvertOfferToOrder(offerID)
}

func (s *CRMService) CreateCostingSheet(rfqID uint, itemsJSON string, createdBy string) (*CostingSheetData, error) {
	return s.app.CreateCostingSheet(rfqID, itemsJSON, createdBy)
}

func (s *CRMService) CreateOffer(costingID string) (*OfferData, error) {
	return s.app.CreateOffer(costingID)
}

func (s *CRMService) CreateRFQ(client, project string, value float64, notes string, productDetails string) (*RFQData, error) {
	return s.app.CreateRFQ(client, project, value, notes, productDetails)
}

func (s *CRMService) CreateRFQWithReference(client, project, reference string, value float64, notes string) (*RFQData, error) {
	return s.app.CreateRFQWithReference(client, project, reference, value, notes)
}

func (s *CRMService) DeleteCostingSheet(id uint) error {
	return s.app.DeleteCostingSheet(id)
}

func (s *CRMService) DeleteOffer(offerID string) error {
	return s.app.DeleteOffer(offerID)
}

func (s *CRMService) DeleteOfferNote(noteID string) error {
	return s.app.DeleteOfferNote(noteID)
}

func (s *CRMService) DeleteOpportunity(opportunityID string) error {
	return s.app.DeleteOpportunity(opportunityID)
}

func (s *CRMService) DeleteOpportunityComment(commentID string) error {
	return s.app.DeleteOpportunityComment(commentID)
}

func (s *CRMService) DeleteRFQ(id uint) error {
	return s.app.DeleteRFQ(id)
}

func (s *CRMService) DeleteRFQWithCascade(id uint, cascade bool) (*DeleteCascadeResult, error) {
	return s.app.DeleteRFQWithCascade(id, cascade)
}

func (s *CRMService) FindOfferByReference(reference string) (*Offer, error) {
	return s.app.FindOfferByReference(reference)
}

func (s *CRMService) GetActiveCostingByRFQ(rfqID uint) (*CostingSheetData, error) {
	return s.app.GetActiveCostingByRFQ(rfqID)
}

func (s *CRMService) GetAllOffers() ([]Offer, error) {
	return s.app.GetAllOffers()
}

func (s *CRMService) GetCostingSheet(id uint) (CostingSheetData, error) {
	return s.app.GetCostingSheet(id)
}

func (s *CRMService) GetCostingSheets(limit int) ([]CostingSheetData, error) {
	return s.app.GetCostingSheets(limit)
}

func (s *CRMService) GetCostingsByRFQ(rfqID uint) ([]CostingSheetData, error) {
	return s.app.GetCostingsByRFQ(rfqID)
}

func (s *CRMService) GetOffer(id uint) (OfferData, error) {
	return s.app.GetOffer(id)
}

func (s *CRMService) GetOfferNotes(offerID string) ([]OfferNote, error) {
	return s.app.GetOfferNotes(offerID)
}

func (s *CRMService) GetOffers(limit int) ([]OfferData, error) {
	return s.app.GetOffers(limit)
}

func (s *CRMService) GetOffersWithNoItems() ([]map[string]any, error) {
	return s.app.GetOffersWithNoItems()
}

func (s *CRMService) GetOpportunityComments(opportunityID string) ([]OpportunityComment, error) {
	return s.app.GetOpportunityComments(opportunityID)
}

func (s *CRMService) GetOpportunityLineItems(opportunityID string) ([]OfferItem, error) {
	return s.app.GetOpportunityLineItems(opportunityID)
}

func (s *CRMService) GetOrdersWithNoItems() ([]map[string]any, error) {
	return s.app.GetOrdersWithNoItems()
}

func (s *CRMService) GetPipelineOpportunities(limit int, offset int) ([]Opportunity, error) {
	return s.app.GetPipelineOpportunities(limit, offset)
}

func (s *CRMService) GetRFQ(id uint) (RFQData, error) {
	return s.app.GetRFQ(id)
}

func (s *CRMService) GetRFQComments(rfqID uint) ([]RFQComment, error) {
	return s.app.GetRFQComments(rfqID)
}

func (s *CRMService) GetRFQs(limit int, offset int) ([]RFQData, error) {
	return s.app.GetRFQs(limit, offset)
}

func (s *CRMService) MarkOfferLost(offerID string, reason string) error {
	return s.app.MarkOfferLost(offerID, reason)
}

func (s *CRMService) MarkOfferWon(offerID string, customerPO string) (*Order, error) {
	return s.app.MarkOfferWon(offerID, customerPO)
}

func (s *CRMService) RejectCostingSheet(id uint, rejectedBy string) error {
	return s.app.RejectCostingSheet(id, rejectedBy)
}

func (s *CRMService) SaveCostingAsOffer(data CostingExportData) (*Offer, error) {
	return s.app.SaveCostingAsOffer(data)
}

func (s *CRMService) SetActiveCostingRevision(costingID uint) error {
	return s.app.SetActiveCostingRevision(costingID)
}

func (s *CRMService) UpdateCostingSheet(id uint, data CostingSheetData) (*CostingSheetData, error) {
	return s.app.UpdateCostingSheet(id, data)
}

func (s *CRMService) UpdateOfferFull(offerID string, data OfferUpdateData) (*Offer, error) {
	return s.app.UpdateOfferFull(offerID, data)
}

func (s *CRMService) UpdateOfferStatus(id uint, status string) error {
	return s.app.UpdateOfferStatus(id, status)
}

func (s *CRMService) UpdateOpportunityDetails(opportunityID, comment, ownerNotes string) (*Opportunity, error) {
	return s.app.UpdateOpportunityDetails(opportunityID, comment, ownerNotes)
}

func (s *CRMService) UpdateOpportunityStage(opportunityID, stage string) error {
	return s.app.UpdateOpportunityStage(opportunityID, stage)
}

func (s *CRMService) UpdateRFQ(id uint, updates RFQUpdateRequest) (*RFQData, error) {
	return s.app.UpdateRFQ(id, updates)
}

func (s *CRMService) UpdateRFQNotes(id uint, notes string) (*RFQData, error) {
	return s.app.UpdateRFQNotes(id, notes)
}

func (s *CRMService) UpdateRFQStage(rfqID uint, stage string) error {
	return s.app.UpdateRFQStage(rfqID, stage)
}

func (s *CRMService) UpdateRFQStatus(id uint, status string) error {
	return s.app.UpdateRFQStatus(id, status)
}

// --- delivery_note_service.go ---

// ConfirmDeliveryNote returns a non-fatal warning string (empty when clean)
// alongside the error — see App.ConfirmDeliveryNote (Inv4).
func (s *CRMService) ConfirmDeliveryNote(id string, signedBy string) (string, error) {
	return s.app.ConfirmDeliveryNote(id, signedBy)
}

func (s *CRMService) CreateDNFromOrder(orderID string, items []DeliveryNoteItem) (DeliveryNote, error) {
	return s.app.CreateDNFromOrder(orderID, items)
}

func (s *CRMService) CreateDNWithSerials(orderID string, items []DNItemInputWithSerials, header DeliveryNoteHeaderInput) (DeliveryNote, error) {
	return s.app.CreateDNWithSerials(orderID, items, header)
}

func (s *CRMService) CreateDeliveryNote(dn DeliveryNote) (DeliveryNote, error) {
	return s.app.CreateDeliveryNote(dn)
}

func (s *CRMService) CreateDeliveryNoteWithItems(orderID string, items []DeliveryNoteItemInput) (DeliveryNote, error) {
	return s.app.CreateDeliveryNoteWithItems(orderID, items)
}

func (s *CRMService) DeleteDeliveryNote(id string) error {
	return s.app.DeleteDeliveryNote(id)
}

func (s *CRMService) DispatchDeliveryNote(id string, driverName string, vehicleNumber string) error {
	return s.app.DispatchDeliveryNote(id, driverName, vehicleNumber)
}

func (s *CRMService) GenerateDNNumber() (string, error) {
	return s.app.GenerateDNNumber()
}

func (s *CRMService) GenerateDeliveryNotePDF(id string) (string, error) {
	return s.app.GenerateDeliveryNotePDF(id)
}

func (s *CRMService) GetDeliveriesByArea(area string) ([]DeliveryPlanningItem, error) {
	return s.app.GetDeliveriesByArea(area)
}

func (s *CRMService) GetDeliveryAreaSummary() (map[string]int, error) {
	return s.app.GetDeliveryAreaSummary()
}

func (s *CRMService) GetDeliveryNoteByID(id string) (DeliveryNote, error) {
	return s.app.GetDeliveryNoteByID(id)
}

func (s *CRMService) GetDeliveryNotes() ([]DeliveryNote, error) {
	return s.app.GetDeliveryNotes()
}

func (s *CRMService) GetDeliveryNotesByCustomer(customerID string) ([]DeliveryNote, error) {
	return s.app.GetDeliveryNotesByCustomer(customerID)
}

func (s *CRMService) GetDeliveryNotesByOrder(orderID string) ([]DeliveryNote, error) {
	return s.app.GetDeliveryNotesByOrder(orderID)
}

func (s *CRMService) GetOrderDeliveryStatus(orderID string) (map[string]float64, error) {
	return s.app.GetOrderDeliveryStatus(orderID)
}

func (s *CRMService) GetOrderDeliveryStatusBatch(orderIDs []string) (map[string]map[string]float64, error) {
	return s.app.GetOrderDeliveryStatusBatch(orderIDs)
}

func (s *CRMService) GetOrderFulfillmentDetail(orderID string) (OrderFulfillment, error) {
	return s.app.GetOrderFulfillmentDetail(orderID)
}

func (s *CRMService) GetPendingDeliveries() ([]DeliveryPlanningItem, error) {
	return s.app.GetPendingDeliveries()
}

func (s *CRMService) UpdateDeliveryNote(dn DeliveryNote) (DeliveryNote, error) {
	return s.app.UpdateDeliveryNote(dn)
}

// --- grn_service.go ---

func (s *CRMService) CompleteGRN(id string) error {
	return s.app.CompleteGRN(id)
}

func (s *CRMService) CreateGRN(grn GoodsReceivedNote) (GoodsReceivedNote, error) {
	return s.app.CreateGRN(grn)
}

func (s *CRMService) DeleteGRN(id string) error {
	return s.app.DeleteGRN(id)
}

func (s *CRMService) GenerateGRNNumber() (string, error) {
	return s.app.GenerateGRNNumber()
}

func (s *CRMService) GetGRN(grnID string) (*GRNResponse, error) {
	return s.app.GetGRN(grnID)
}

func (s *CRMService) GetGRNByID(id string) (GoodsReceivedNote, error) {
	return s.app.GetGRNByID(id)
}

func (s *CRMService) GetGRNs() ([]GoodsReceivedNote, error) {
	return s.app.GetGRNs()
}

func (s *CRMService) GetGRNsByPO(purchaseOrderID string) ([]GoodsReceivedNote, error) {
	return s.app.GetGRNsByPO(purchaseOrderID)
}

func (s *CRMService) ListGRNs(limit int, offset int, qcStatus string) ([]GRNResponse, error) {
	return s.app.ListGRNs(limit, offset, qcStatus)
}

func (s *CRMService) RaiseGRNDiscrepancy(grnID string, itemID string, reason string, discrepancyType string, rejectedQty float64) error {
	return s.app.RaiseGRNDiscrepancy(grnID, itemID, reason, discrepancyType, rejectedQty)
}

func (s *CRMService) ReceiveAgainstPO(poID string, items []GRNItem) (GoodsReceivedNote, error) {
	return s.app.ReceiveAgainstPO(poID, items)
}

func (s *CRMService) ReceiveAgainstPOWithSerials(poID string, items []GRNItemWithSerials) (GoodsReceivedNote, error) {
	return s.app.ReceiveAgainstPOWithSerials(poID, items)
}

func (s *CRMService) UpdateGRN(grn GoodsReceivedNote) (GoodsReceivedNote, error) {
	return s.app.UpdateGRN(grn)
}

func (s *CRMService) UpdateGRNQCStatus(id string, status string, notes string, qcBy string) error {
	return s.app.UpdateGRNQCStatus(id, status, notes, qcBy)
}

// --- inventory_service.go ---

func (s *CRMService) GetInventoryAlertsLowStock(threshold float64) ([]InventoryAlert, error) {
	return s.app.GetInventoryAlertsLowStock(threshold)
}

func (s *CRMService) GetInventoryAlertsSlowMoving(days int) ([]InventoryAlert, error) {
	return s.app.GetInventoryAlertsSlowMoving(days)
}

func (s *CRMService) GetInventoryAlertsSummary() (map[string]int, error) {
	return s.app.GetInventoryAlertsSummary()
}

func (s *CRMService) GetReorderSuggestions() ([]ReorderSuggestion, error) {
	return s.app.GetReorderSuggestions()
}

// --- offer_followup_service.go ---

func (s *CRMService) AddOfferFollowUp(offerID, followUpDate, notes string) (*OfferFollowUp, error) {
	return s.app.AddOfferFollowUp(offerID, followUpDate, notes)
}

func (s *CRMService) CancelOfferFollowUp(followUpID string) error {
	return s.app.CancelOfferFollowUp(followUpID)
}

func (s *CRMService) CompleteOfferFollowUp(followUpID string) error {
	return s.app.CompleteOfferFollowUp(followUpID)
}

func (s *CRMService) GetOfferFollowUps(offerID string) ([]OfferFollowUp, error) {
	return s.app.GetOfferFollowUps(offerID)
}

func (s *CRMService) GetOverdueOfferFollowUps() ([]OfferFollowUp, error) {
	return s.app.GetOverdueOfferFollowUps()
}

func (s *CRMService) GetPendingFollowUps() ([]OfferFollowUp, error) {
	return s.app.GetPendingFollowUps()
}

// --- opportunity_conflict_service.go ---

func (s *CRMService) CanResolveOpportunityConflicts() bool {
	return s.app.CanResolveOpportunityConflicts()
}

func (s *CRMService) EnsureOpportunityConflictFoundation() error {
	return s.app.EnsureOpportunityConflictFoundation()
}

func (s *CRMService) ListOpportunityEditConflicts(status string, limit int) ([]OpportunityEditConflict, error) {
	return s.app.ListOpportunityEditConflicts(status, limit)
}

func (s *CRMService) ResolveOpportunityEditConflict(conflictID, action, note string) (*OpportunityConflictResolutionResult, error) {
	return s.app.ResolveOpportunityEditConflict(conflictID, action, note)
}

func (s *CRMService) UpdateOpportunityDetailsWithVersion(opportunityID string, expectedVersion int, comment, ownerNotes string) (*Opportunity, error) {
	return s.app.UpdateOpportunityDetailsWithVersion(opportunityID, expectedVersion, comment, ownerNotes)
}

func (s *CRMService) UpdateOpportunityStageWithVersion(opportunityID, stage string, expectedVersion int) (*Opportunity, error) {
	return s.app.UpdateOpportunityStageWithVersion(opportunityID, stage, expectedVersion)
}

// --- p2_sales_ux_enhancements.go ---

func (s *CRMService) BulkUpdateOfferStage(offerIDs []string, stage string) error {
	return s.app.BulkUpdateOfferStage(offerIDs, stage)
}

func (s *CRMService) GetCustomerOrderHistorySummary(customerID string) (CustomerOrderHistorySummary, error) {
	return s.app.GetCustomerOrderHistorySummary(customerID)
}

func (s *CRMService) GetOfferRevisionHistory(offerID string) ([]OfferRevision, error) {
	return s.app.GetOfferRevisionHistory(offerID)
}

func (s *CRMService) GetOverdueRFQs() ([]OpportunityDueData, error) {
	return s.app.GetOverdueRFQs()
}

func (s *CRMService) GetRFQsDueSoon(days int) ([]OpportunityDueData, error) {
	return s.app.GetRFQsDueSoon(days)
}

func (s *CRMService) SearchOffers(query string) ([]OfferSearchResult, error) {
	return s.app.SearchOffers(query)
}

func (s *CRMService) SearchOrders(query string) ([]OrderSearchResult, error) {
	return s.app.SearchOrders(query)
}

// --- pipeline_handlers.go ---

func (s *CRMService) CheckCompliance(data ComplianceData) (*ComplianceResult, error) {
	return s.app.CheckCompliance(data)
}

func (s *CRMService) GetCustomer360Geometry(customerID string) (*Customer360, error) {
	return s.app.GetCustomer360Geometry(customerID)
}

func (s *CRMService) GetPipelineStatistics() map[string]any {
	return s.app.GetPipelineStatistics()
}

func (s *CRMService) GetRoutingHistory(limit int) []RoutingResult {
	return s.app.GetRoutingHistory(limit)
}

func (s *CRMService) ProcessInvoice(invoice InvoiceGeometry) (*InvoiceResult, error) {
	return s.app.ProcessInvoice(invoice)
}

func (s *CRMService) ProcessRFQToOrder(tender TenderGeometry, customer Customer) (*CompleteFlowResult, error) {
	return s.app.ProcessRFQToOrder(tender, customer)
}

func (s *CRMService) ProcessTender(tender TenderGeometry) (*TenderResult, error) {
	return s.app.ProcessTender(tender)
}

func (s *CRMService) RouteEvent(event ERPEvent) (*RoutingResult, error) {
	return s.app.RouteEvent(event)
}

// --- product_service.go ---

func (s *CRMService) GetProductByCode(code string) (*ProductMaster, error) {
	return s.app.GetProductByCode(code)
}

func (s *CRMService) SearchProducts(query string) ([]ProductMaster, error) {
	return s.app.SearchProducts(query)
}

func (s *CRMService) SeedProductDatabase() error {
	return s.app.SeedProductDatabase()
}

// --- purchase_order_service.go ---

func (s *CRMService) AmendPurchaseOrder(poID string, amendments POAmendment) error {
	return s.app.AmendPurchaseOrder(poID, amendments)
}

func (s *CRMService) ApprovePurchaseOrder(id string, approvedBy string) error {
	return s.app.ApprovePurchaseOrder(id, approvedBy)
}

func (s *CRMService) CreatePOFromOrder(orderID string, supplierID string, itemIDs []string) (PurchaseOrder, error) {
	return s.app.CreatePOFromOrder(orderID, supplierID, itemIDs)
}

func (s *CRMService) CreatePurchaseOrder(po PurchaseOrder) (PurchaseOrder, error) {
	return s.app.CreatePurchaseOrder(po)
}

func (s *CRMService) DeletePurchaseOrder(id string) error {
	return s.app.DeletePurchaseOrder(id)
}

func (s *CRMService) GeneratePONumber() (string, error) {
	return s.app.GeneratePONumber()
}

func (s *CRMService) GetPOAmendmentHistory(poID string) ([]POAmendment, error) {
	return s.app.GetPOAmendmentHistory(poID)
}

func (s *CRMService) GetPurchaseOrderByID(id string) (PurchaseOrder, error) {
	return s.app.GetPurchaseOrderByID(id)
}

func (s *CRMService) GetPurchaseOrders() ([]PurchaseOrder, error) {
	return s.app.GetPurchaseOrders()
}

func (s *CRMService) GetPurchaseOrdersByOrder(orderID string) ([]PurchaseOrder, error) {
	return s.app.GetPurchaseOrdersByOrder(orderID)
}

func (s *CRMService) GetPurchaseOrdersBySupplier(supplierID string) ([]PurchaseOrder, error) {
	return s.app.GetPurchaseOrdersBySupplier(supplierID)
}

func (s *CRMService) SendPurchaseOrder(id string) error {
	return s.app.SendPurchaseOrder(id)
}

func (s *CRMService) UpdatePOStatus(id string, status string) error {
	return s.app.UpdatePOStatus(id, status)
}

func (s *CRMService) UpdatePurchaseOrder(po PurchaseOrder) (PurchaseOrder, error) {
	return s.app.UpdatePurchaseOrder(po)
}

// --- serial_number_service.go ---

func (s *CRMService) AttachCalibrationCert(serialID, certPath string) error {
	return s.app.AttachCalibrationCert(serialID, certPath)
}

func (s *CRMService) GetAvailableSerials(productID string) ([]SerialNumber, error) {
	return s.app.GetAvailableSerials(productID)
}

func (s *CRMService) GetSerialByNumber(serialNo string) (SerialNumber, error) {
	return s.app.GetSerialByNumber(serialNo)
}

func (s *CRMService) GetSerialsByCustomer(customerID string) ([]SerialNumber, error) {
	return s.app.GetSerialsByCustomer(customerID)
}

func (s *CRMService) GetSerialsByProduct(productID string) ([]SerialNumber, error) {
	return s.app.GetSerialsByProduct(productID)
}

func (s *CRMService) GetSerialsForInvoiceItem(invoiceID, productID string) ([]SerialNumber, error) {
	return s.app.GetSerialsForInvoiceItem(invoiceID, productID)
}

func (s *CRMService) GetRecentlyDeliveredSerials(limit int) ([]SerialNumber, error) {
	return s.app.GetRecentlyDeliveredSerials(limit)
}

func (s *CRMService) RegisterSerials(productID string, serialNos []string) ([]SerialNumber, error) {
	return s.app.RegisterSerials(productID, serialNos)
}

func (s *CRMService) SearchSerials(query string, limit int) ([]SerialNumber, error) {
	return s.app.SearchSerials(query, limit)
}

func (s *CRMService) UpdateSerialWarranty(serialID string, warrantyMonths int) error {
	return s.app.UpdateSerialWarranty(serialID, warrantyMonths)
}
