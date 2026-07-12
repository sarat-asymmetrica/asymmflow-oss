// Package crm contains display-ready ViewModels for CRM screens.
package crm

import (
	vm "ph_holdings_app/internal/viewmodel"
	"ph_holdings_app/internal/viewmodel/shared"
)

// CustomerListVM is the display contract for the customer list screen.
type CustomerListVM struct {
	Table             shared.TableVM    `json:"table"`
	TotalCustomers    int               `json:"totalCustomers"`
	GradeDistribution []GradeBucketVM   `json:"gradeDistribution"`
	Actions           []vm.ActionButton `json:"actions"`
}

// GradeBucketVM summarizes customers by grade.
type GradeBucketVM struct {
	Grade string `json:"grade"`
	Count int    `json:"count"`
	Color string `json:"color"`
}

// CustomerDetailVM is the display contract for a customer profile.
type CustomerDetailVM struct {
	ID           string               `json:"id"`
	BusinessName string               `json:"businessName"`
	CustomerCode string               `json:"customerCode"`
	Status       shared.StatusBadgeVM `json:"status"`
	Grade        shared.StatusBadgeVM `json:"grade"`
	PrimaryEmail string               `json:"primaryEmail,omitempty"`
	PrimaryPhone string               `json:"primaryPhone,omitempty"`
	MobileNumber string               `json:"mobileNumber,omitempty"`
	Address      string               `json:"address,omitempty"`
	Contacts     []ContactVM          `json:"contacts"`
	RecentOrders []OrderSummaryVM     `json:"recentOrders"`
	ARAging      []ARAgingVM          `json:"arAging"`
	Notes        []NoteVM             `json:"notes"`
	Actions      []vm.ActionButton    `json:"actions"`
	Breadcrumbs  []vm.BreadcrumbItem  `json:"breadcrumbs"`
}

// Customer360VM is the display contract for the relationship graph view.
type Customer360VM struct {
	Customer        CustomerDetailVM `json:"customer"`
	Graph           GraphVM          `json:"graph"`
	RelationshipMap []RelationshipVM `json:"relationshipMap"`
}

// GraphVM contains graph visualization data.
type GraphVM struct {
	Nodes []GraphNodeVM `json:"nodes"`
	Edges []GraphEdgeVM `json:"edges"`
}

// GraphNodeVM is a display-ready graph node.
type GraphNodeVM struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"`
	Color string `json:"color,omitempty"`
}

// GraphEdgeVM is a display-ready graph edge.
type GraphEdgeVM struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label,omitempty"`
}

// RelationshipVM describes one customer relationship.
type RelationshipVM struct {
	Type        string `json:"type"`
	TargetID    string `json:"targetId"`
	TargetLabel string `json:"targetLabel"`
	Strength    string `json:"strength"`
	Detail      string `json:"detail,omitempty"`
}

// PipelineVM is the display contract for the opportunity pipeline screen.
type PipelineVM struct {
	Stages               []PipelineStageVM `json:"stages"`
	TotalPipelineValue   string            `json:"totalPipelineValue"`
	WinRate              string            `json:"winRate"`
	OpenOpportunityCount int               `json:"openOpportunityCount"`
	Actions              []vm.ActionButton `json:"actions"`
}

// PipelineSnapshotVM is a compact pipeline summary for dashboards.
type PipelineSnapshotVM struct {
	OpenCount     int    `json:"openCount"`
	WeightedValue string `json:"weightedValue"`
	TopStage      string `json:"topStage"`
	WinRate       string `json:"winRate"`
}

// PipelineStageVM summarizes opportunities in one stage.
type PipelineStageVM struct {
	Stage        string              `json:"stage"`
	Count        int                 `json:"count"`
	ValueDisplay string              `json:"valueDisplay"`
	Color        string              `json:"color"`
	Items        []OpportunityCardVM `json:"items,omitempty"`
}

// OpportunityCardVM is a display-ready opportunity card.
type OpportunityCardVM struct {
	ID           string               `json:"id"`
	FolderNumber string               `json:"folderNumber"`
	Title        string               `json:"title"`
	CustomerName string               `json:"customerName"`
	ValueDisplay string               `json:"valueDisplay"`
	ExpectedDate string               `json:"expectedDate,omitempty"`
	Status       shared.StatusBadgeVM `json:"status"`
}

// OfferDetailVM is the display contract for a quotation detail screen.
type OfferDetailVM struct {
	ID              string               `json:"id"`
	OfferNumber     string               `json:"offerNumber"`
	Revision        string               `json:"revision"`
	CustomerName    string               `json:"customerName"`
	QuotationDate   string               `json:"quotationDate"`
	ValidityDate    string               `json:"validityDate"`
	Status          shared.StatusBadgeVM `json:"status"`
	Items           []OfferItemVM        `json:"items"`
	MarginAnalysis  MarginAnalysisVM     `json:"marginAnalysis"`
	FollowUps       []FollowUpVM         `json:"followUps"`
	RevisionHistory []RevisionVM         `json:"revisionHistory"`
	Actions         []vm.ActionButton    `json:"actions"`
}

// OfferItemVM is a display-ready quotation line.
type OfferItemVM struct {
	ID           string `json:"id"`
	LineNumber   int    `json:"lineNumber"`
	ProductCode  string `json:"productCode"`
	Description  string `json:"description"`
	Quantity     string `json:"quantity"`
	UnitPrice    string `json:"unitPrice"`
	TotalDisplay string `json:"totalDisplay"`
	Margin       string `json:"margin"`
}

// MarginAnalysisVM displays quotation margin data.
type MarginAnalysisVM struct {
	TotalValue  string `json:"totalValue"`
	TotalCost   string `json:"totalCost"`
	MarginValue string `json:"marginValue"`
	MarginPct   string `json:"marginPct"`
	Color       string `json:"color"`
}

// FollowUpVM displays a follow-up action.
type FollowUpVM struct {
	ID      string               `json:"id"`
	DueDate string               `json:"dueDate"`
	Notes   string               `json:"notes"`
	Status  shared.StatusBadgeVM `json:"status"`
}

// RevisionVM displays a quotation revision.
type RevisionVM struct {
	Revision string `json:"revision"`
	Date     string `json:"date"`
	Summary  string `json:"summary"`
}

// OrderListVM is the display contract for the order list screen.
type OrderListVM struct {
	Table   shared.TableVM    `json:"table"`
	Actions []vm.ActionButton `json:"actions"`
}

// OrderDetailVM is the display contract for a customer order detail screen.
type OrderDetailVM struct {
	ID               string               `json:"id"`
	OrderNumber      string               `json:"orderNumber"`
	CustomerName     string               `json:"customerName"`
	OrderDate        string               `json:"orderDate"`
	RequiredDate     string               `json:"requiredDate,omitempty"`
	Status           shared.StatusBadgeVM `json:"status"`
	Items            []OrderItemVM        `json:"items"`
	DeliveryNotes    []DeliveryNoteVM     `json:"deliveryNotes"`
	Invoices         []DocumentLinkVM     `json:"invoices"`
	ShipmentTracking []ShipmentVM         `json:"shipmentTracking"`
	Actions          []vm.ActionButton    `json:"actions"`
}

// OrderSummaryVM is a compact order row used in CustomerDetailVM.
type OrderSummaryVM struct {
	ID           string               `json:"id"`
	OrderNumber  string               `json:"orderNumber"`
	OrderDate    string               `json:"orderDate"`
	Status       shared.StatusBadgeVM `json:"status"`
	TotalDisplay string               `json:"totalDisplay"`
}

// OrderItemVM is a display-ready order line.
type OrderItemVM struct {
	ID              string `json:"id"`
	LineNumber      int    `json:"lineNumber"`
	Description     string `json:"description"`
	Quantity        string `json:"quantity"`
	ShippedDisplay  string `json:"shippedDisplay"`
	InvoicedDisplay string `json:"invoicedDisplay"`
	TotalDisplay    string `json:"totalDisplay"`
}

// DeliveryNoteVM is a display-ready delivery note summary.
type DeliveryNoteVM struct {
	ID             string               `json:"id"`
	DNNumber       string               `json:"dnNumber"`
	DeliveryDate   string               `json:"deliveryDate"`
	Status         shared.StatusBadgeVM `json:"status"`
	DeliveredItems int                  `json:"deliveredItems"`
}

// DocumentLinkVM links related documents.
type DocumentLinkVM struct {
	ID     string `json:"id"`
	Number string `json:"number"`
	Type   string `json:"type"`
	Path   string `json:"path,omitempty"`
}

// ShipmentVM is a display-ready shipment row.
type ShipmentVM struct {
	ID             string               `json:"id"`
	CourierName    string               `json:"courierName"`
	TrackingNumber string               `json:"trackingNumber"`
	Status         shared.StatusBadgeVM `json:"status"`
	ShipmentDate   string               `json:"shipmentDate"`
	DeliveredDate  string               `json:"deliveredDate,omitempty"`
}

// SupplierDashboardVM is the display contract for supplier scorecards.
type SupplierDashboardVM struct {
	Scorecards      []SupplierScorecardVM `json:"scorecards"`
	LeadTimeMetrics []LeadTimeMetricVM    `json:"leadTimeMetrics"`
	TopIssues       []SupplierIssueVM     `json:"topIssues"`
	Actions         []vm.ActionButton     `json:"actions"`
}

// SupplierScorecardVM is a display-ready supplier scorecard.
type SupplierScorecardVM struct {
	ID           string `json:"id"`
	SupplierName string `json:"supplierName"`
	Rating       string `json:"rating"`
	LeadTime     string `json:"leadTime"`
	PaymentTerms string `json:"paymentTerms"`
	Country      string `json:"country"`
}

// LeadTimeMetricVM displays supplier lead-time performance.
type LeadTimeMetricVM struct {
	SupplierID   string `json:"supplierId"`
	SupplierName string `json:"supplierName"`
	AverageDays  int    `json:"averageDays"`
	Color        string `json:"color"`
}

// SupplierIssueVM displays one supplier issue.
type SupplierIssueVM struct {
	ID           string               `json:"id"`
	SupplierName string               `json:"supplierName"`
	Description  string               `json:"description"`
	Status       shared.StatusBadgeVM `json:"status"`
	CostDisplay  string               `json:"costDisplay,omitempty"`
}

// ContactVM is a display-ready contact row.
type ContactVM struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	JobTitle  string `json:"jobTitle,omitempty"`
	Email     string `json:"email,omitempty"`
	Phone     string `json:"phone,omitempty"`
	IsPrimary bool   `json:"isPrimary"`
}

// NoteVM is a display-ready CRM note.
type NoteVM struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content string `json:"content"`
	Date    string `json:"date"`
}

// ARAgingVM displays one AR bucket for a customer.
type ARAgingVM struct {
	Label         string `json:"label"`
	AmountDisplay string `json:"amountDisplay"`
	Color         string `json:"color"`
}
