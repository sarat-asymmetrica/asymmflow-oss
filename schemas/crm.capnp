@0xdcc25c48df300d3c;

using Go = import "/go.capnp";
using Common = import "common.capnp";
using Finance = import "finance.capnp";

$Go.package("crm");
$Go.import("ph_holdings_app/schemas/go/crm");

# CRM, sales pipeline, fulfillment, and inventory schema contracts.
# PurchaseOrder and PurchaseOrderItem remain finance-owned and are not duplicated here.

enum CustomerGrade {
  unknown @0;
  a @1;
  b @2;
  c @3;
  d @4;
}

enum OpportunityStage {
  lead @0;
  qualified @1;
  proposal @2;
  quoted @3;
  negotiation @4;
  won @5;
  lost @6;
  expired @7;
}

enum DeliveryStatus {
  prepared @0;
  dispatched @1;
  inTransit @2;
  delivered @3;
  signed @4;
  cancelled @5;
}

enum StockDirection {
  inbound @0;
  outbound @1;
  adjustment @2;
}

struct CustomerMaster {
  base @0 :Common.Base;
  customerId @1 :Text;
  customerCode @2 :Text;
  customerType @3 :Text;
  businessName @4 :Text;
  shortCode @5 :Text;
  tradingName @6 :Text;
  crNumber @7 :Text;
  status @8 :Common.RecordStatus;
  primaryPhone @9 :Text;
  primaryEmail @10 :Text;
  website @11 :Text;
  addressLine1 @12 :Text;
  city @13 :Text;
  country @14 :Text;
  trn @15 :Text;
  mobileNumber @16 :Text;
  industry @17 :Text;
  relationYears @18 :Int64;
  paymentGrade @19 :CustomerGrade;
  customerGrade @20 :CustomerGrade;
  paymentTermsDays @21 :Int64;
  avgPaymentDays @22 :Float64;
  disputeCount @23 :Int64;
  totalOrdersValue @24 :Float64;
  totalOrdersCount @25 :Int64;
  avgOrderValue @26 :Float64;
  lastOrderDate @27 :Text;
  arRiskTier @28 :Common.RiskLevel;
  outstandingBhd @29 :Float64;
  overdueDays @30 :Int64;
  creditLimitBhd @31 :Float64;
  isCreditBlocked @32 :Bool;
  requiresPrepayment @33 :Bool;
  hasAbbCompetition @34 :Bool;
  isEmergencyOnly @35 :Bool;
}

struct CustomerContact {
  base @0 :Common.Base;
  customerId @1 :Text;
  contactName @2 :Text;
  jobTitle @3 :Text;
  email @4 :Text;
  phone @5 :Text;
  address @6 :Text;
  isPrimaryContact @7 :Bool;
}

struct SupplierContact {
  base @0 :Common.Base;
  supplierId @1 :Text;
  contactName @2 :Text;
  jobTitle @3 :Text;
  email @4 :Text;
  phone @5 :Text;
  address @6 :Text;
  isPrimaryContact @7 :Bool;
}

struct SupplierMaster {
  base @0 :Common.Base;
  supplierCode @1 :Text;
  supplierName @2 :Text;
  country @3 :Text;
  leadTimeDays @4 :Int64;
  taxId @5 :Text;
  supplierType @6 :Text;
  brandsHandled @7 :Text;
  productTypes @8 :Text;
  primaryContact @9 :Text;
  email @10 :Text;
  phone @11 :Text;
  address @12 :Text;
  bankName @13 :Text;
  accountNumber @14 :Text;
  iban @15 :Text;
  swiftCode @16 :Text;
  paymentTerms @17 :Text;
  rating @18 :Int64;
  notes @19 :Text;
}

struct EntityNote {
  base @0 :Common.Base;
  entityType @1 :Text;
  entityId @2 :Text;
  noteType @3 :Text;
  content @4 :Text;
}

struct SupplierIssue {
  base @0 :Common.Base;
  supplierId @1 :Text;
  orderRef @2 :Text;
  description @3 :Text;
  status @4 :Common.DocumentStatus;
  resolution @5 :Text;
  costBhd @6 :Float64;
  resolvedAt @7 :Text;
}

struct ProductMaster {
  base @0 :Common.Base;
  productCode @1 :Text;
  productName @2 :Text;
  productCategory @3 :Text;
  supplierId @4 :Text;
  supplierCode @5 :Text;
  standardCostBhd @6 :Float64;
  standardPriceBhd @7 :Float64;
  description @8 :Text;
  isActive @9 :Bool;
  stockQuantity @10 :Int64;
  sku @11 :Text;
  partNumber @12 :Text;
  hsCode @13 :Text;
  unitOfMeasure @14 :Text;
  datasheetUrl @15 :Text;
  specifications @16 :Text;
  requiresSerialTracking @17 :Bool;
}

struct Offer {
  base @0 :Common.Base;
  offerNumber @1 :Text;
  revisionNumber @2 :Int64;
  rfqId @3 :Text;
  customerId @4 :Text;
  customerName @5 :Text;
  quotationDate @6 :Text;
  validityDate @7 :Text;
  totalValueBhd @8 :Float64;
  estimatedMargin @9 :Float64;
  stage @10 :OpportunityStage;
  hasAbbCompetition @11 :Bool;
  lostReason @12 :Text;
  paymentTerms @13 :Text;
  deliveryTerms @14 :Text;
  deliveryWeeks @15 :Text;
  countryOfOrigin @16 :Text;
  issuedBy @17 :Text;
  contactPhone @18 :Text;
  customerReference @19 :Text;
  attentionPerson @20 :Text;
  attentionCompany @21 :Text;
  attentionPhone @22 :Text;
  attentionAddress @23 :Text;
  discountPercent @24 :Float64;
  quoteType @25 :Text;
  vatRate @26 :Float64;
  division @27 :Text;
  termsAndConditions @28 :Text;
  subject @29 :Text;
  body @30 :Text;
  cocCoo @31 :Text;
  testCertificate @32 :Text;
  installation @33 :Text;
  commissioning @34 :Text;
  testing @35 :Text;
  folderNumber @36 :Text;
  projectName @37 :Text;
  items @38 :List(OfferItem);
}

struct OfferItem {
  base @0 :Common.Base;
  offerId @1 :Text;
  lineNumber @2 :Int64;
  productId @3 :Text;
  productCode @4 :Text;
  model @5 :Text;
  description @6 :Text;
  quantity @7 :Float64;
  unitPriceBhd @8 :Float64;
  longCode @9 :Text;
  equipment @10 :Text;
  specification @11 :Text;
  detailedDescription @12 :Text;
  currency @13 :Common.CurrencyCode;
  fob @14 :Float64;
  freight @15 :Float64;
  totalCost @16 :Float64;
  marginPercent @17 :Float64;
  totalPrice @18 :Float64;
  exchangeRate @19 :Float64;
  fobBhd @20 :Float64;
  freightBhd @21 :Float64;
  insurance @22 :Float64;
  customsPercent @23 :Float64;
  customsBhd @24 :Float64;
  handlingPercent @25 :Float64;
  handlingBhd @26 :Float64;
  financePercent @27 :Float64;
  financeBhd @28 :Float64;
  otherCosts @29 :Float64;
  userPrice @30 :Float64;
  userPriceSet @31 :Bool;
}

struct Opportunity {
  base @0 :Common.Base;
  folderNumber @1 :Text;
  offerId @2 :Text;
  customerId @3 :Text;
  customerName @4 :Text;
  customerGrade @5 :CustomerGrade;
  salesperson @6 :Text;
  division @7 :Text;
  year @8 :Int64;
  oppNumber @9 :Int64;
  folderName @10 :Text;
  title @11 :Text;
  ehRef @12 :Text;
  source @13 :Text;
  comment @14 :Text;
  ownerNotes @15 :Text;
  productDetails @16 :Text;
  offerDate @17 :Text;
  orderDate @18 :Text;
  expectedDate @19 :Text;
  closedDate @20 :Text;
  deliveryTerms @21 :Text;
  paymentTerms @22 :Text;
  revenueBhd @23 :Float64;
  costBhd @24 :Float64;
  profitBhd @25 :Float64;
  spocStatus @26 :Text;
  wipStatus @27 :Text;
  stage @28 :OpportunityStage;
  regime @29 :Int64;
  confidence @30 :Float64;
  r1 @31 :Float64;
  r2 @32 :Float64;
  r3 @33 :Float64;
  hasAbbCompetition @34 :Bool;
  productType @35 :Text;
  wonReason @36 :Text;
  lostReason @37 :Text;
}

struct Order {
  base @0 :Common.Base;
  orderNumber @1 :Text;
  customerPoNumber @2 :Text;
  customerId @3 :Text;
  customerName @4 :Text;
  orderDate @5 :Text;
  requiredDate @6 :Text;
  totalValueBhd @7 :Float64;
  grandTotalBhd @8 :Float64;
  status @9 :Common.DocumentStatus;
  updatedBy @10 :Text;
  paymentTerms @11 :Text;
  deliveryTerms @12 :Text;
  offerId @13 :Text;
  offerNumber @14 :Text;
  rfqId @15 :Text;
  customerReference @16 :Text;
  attentionPerson @17 :Text;
  attentionCompany @18 :Text;
  attentionPhone @19 :Text;
  attentionAddress @20 :Text;
  deliveryWeeks @21 :Text;
  countryOfOrigin @22 :Text;
  issuedBy @23 :Text;
  contactPhone @24 :Text;
  discountPercent @25 :Float64;
  division @26 :Text;
  items @27 :List(OrderItem);
}

struct OrderItem {
  base @0 :Common.Base;
  orderId @1 :Text;
  lineNumber @2 :Int64;
  productId @3 :Text;
  productCode @4 :Text;
  description @5 :Text;
  quantity @6 :Float64;
  unitPriceBhd @7 :Float64;
  quantityShipped @8 :Float64;
  quantityInvoiced @9 :Float64;
  equipment @10 :Text;
  model @11 :Text;
  specification @12 :Text;
  detailedDescription @13 :Text;
  currency @14 :Common.CurrencyCode;
  fob @15 :Float64;
  freight @16 :Float64;
  totalCost @17 :Float64;
  marginPercent @18 :Float64;
  totalPrice @19 :Float64;
}

struct Shipment {
  base @0 :Common.Base;
  orderId @1 :Text;
  orderNumber @2 :Text;
  status @3 :DeliveryStatus;
  shipmentDate @4 :Text;
  deliveredDate @5 :Text;
  courierName @6 :Text;
  trackingNumber @7 :Text;
}

struct PostSaleNote {
  base @0 :Common.Base;
  orderId @1 :Text;
  orderNumber @2 :Text;
  noteType @3 :Text;
  description @4 :Text;
  costBhd @5 :Float64;
  noteDate @6 :Text;
  resolvedAt @7 :Text;
  resolution @8 :Text;
}

struct DeliveryNote {
  base @0 :Common.Base;
  orderId @1 :Text;
  customerId @2 :Text;
  dnNumber @3 :Text;
  deliveryDate @4 :Text;
  deliveryAddress @5 :Text;
  contactPerson @6 :Text;
  contactPhone @7 :Text;
  driverName @8 :Text;
  vehicleNumber @9 :Text;
  transportMethod @10 :Text;
  status @11 :DeliveryStatus;
  updatedBy @12 :Text;
  signedBy @13 :Text;
  signedAt @14 :Text;
  signatureImage @15 :Text;
  isPartialDelivery @16 :Bool;
  deliverySequence @17 :Int64;
  totalDeliveries @18 :Int64;
  items @19 :List(DeliveryNoteItem);
}

struct DeliveryNoteItem {
  base @0 :Common.Base;
  deliveryNoteId @1 :Text;
  orderItemId @2 :Text;
  productId @3 :Text;
  productCode @4 :Text;
  description @5 :Text;
  quantityOrdered @6 :Float64;
  quantityDelivered @7 :Float64;
  quantityRemaining @8 :Float64;
}

struct DBCostingSheet {
  base @0 :Common.Base;
  costingNumber @1 :Text;
  customerId @2 :Text;
  customerName @3 :Text;
  costingDate @4 :Text;
  validUntil @5 :Text;
  subtotalBhd @6 :Float64;
  totalMarginBhd @7 :Float64;
  shippingCostBhd @8 :Float64;
  customsDutyBhd @9 :Float64;
  clearanceCostBhd @10 :Float64;
  handlingCostBhd @11 :Float64;
  additionalCostsBhd @12 :Float64;
  grandTotalBhd @13 :Float64;
  status @14 :Common.DocumentStatus;
  convertedToOfferId @15 :Text;
  items @16 :List(DBCostingItem);
  additionalCosts @17 :List(DBCostingAdditionalCost);
}

struct DBCostingItem {
  base @0 :Common.Base;
  costingSheetId @1 :Text;
  lineNumber @2 :Int64;
  productId @3 :Text;
  productType @4 :Text;
  description @5 :Text;
  quantity @6 :Float64;
  unitCostBhd @7 :Float64;
  marginPercent @8 :Float64;
  unitPriceBhd @9 :Float64;
  lineTotalBhd @10 :Float64;
}

struct DBCostingAdditionalCost {
  base @0 :Common.Base;
  costingSheetId @1 :Text;
  description @2 :Text;
  amountBhd @3 :Float64;
}

struct SerialNumber {
  base @0 :Common.Base;
  productId @1 :Text;
  productCode @2 :Text;
  serialNo @3 :Text;
  lotNumber @4 :Text;
  status @5 :Common.DocumentStatus;
  poId @6 :Text;
  poNumber @7 :Text;
  grnItemId @8 :Text;
  grnNumber @9 :Text;
  dnItemId @10 :Text;
  dnNumber @11 :Text;
  invoiceId @12 :Text;
  invoiceNumber @13 :Text;
  customerId @14 :Text;
  customerName @15 :Text;
  receivedDate @16 :Text;
  shippedDate @17 :Text;
  warrantyStartDate @18 :Text;
  warrantyEndDate @19 :Text;
  warrantyMonths @20 :Int64;
  calibrationDate @21 :Text;
  calibrationDueDate @22 :Text;
  calibrationCertPath @23 :Text;
  notes @24 :Text;
}

struct InventoryItem {
  base @0 :Common.Base;
  productId @1 :Text;
  productCode @2 :Text;
  warehouseId @3 :Text;
  quantityOnHand @4 :Float64;
  quantityReserved @5 :Float64;
  quantityAvailable @6 :Float64;
  unitCost @7 :Float64;
  stockStatus @8 :Text;
  isActive @9 :Bool;
  reorderPoint @10 :Float64;
  minimumStock @11 :Float64;
  maximumStock @12 :Float64;
  totalValue @13 :Float64;
  lastPurchaseCost @14 :Float64;
  lastMovementAt @15 :Text;
}

struct StockMovement {
  base @0 :Common.Base;
  inventoryItemId @1 :Text;
  movementType @2 :Text;
  movementNumber @3 :Text;
  quantity @4 :Float64;
  direction @5 :StockDirection;
  balanceBefore @6 :Float64;
  balanceAfter @7 :Float64;
  movementDate @8 :Text;
  unitCost @9 :Float64;
  totalValue @10 :Float64;
}

struct StockAdjustment {
  base @0 :Common.Base;
  inventoryItemId @1 :Text;
  adjustmentDate @2 :Text;
  adjustmentType @3 :Text;
  reason @4 :Text;
  variance @5 :Float64;
  systemQuantity @6 :Float64;
  physicalQuantity @7 :Float64;
  unitCost @8 :Float64;
  valueImpact @9 :Float64;
  notes @10 :Text;
  status @11 :Common.ApprovalStatus;
  adjustmentNumber @12 :Text;
  approvedBy @13 :Text;
  approvedAt @14 :Text;
}

struct Warehouse {
  base @0 :Common.Base;
  code @1 :Text;
  name @2 :Text;
  isActive @3 :Bool;
}

struct CostingHistory {
  base @0 :Common.Base;
  productId @1 :Text;
  costBhd @2 :Float64;
}

struct CostingLineItemData {
  id @0 :Text;
  costingSheetId @1 :Int64;
  productNumber @2 :Int64;
  equipment @3 :Text;
  model @4 :Text;
  specification @5 :Text;
  supplier @6 :Text;
  quantity @7 :Float64;
  fobEur @8 :Float64;
  exchangeRate @9 :Float64;
  totalCostBhd @10 :Float64;
  markupPercent @11 :Float64;
  sellingPriceBhd @12 :Float64;
  totalSuggestedBhd @13 :Float64;
  createdAt @14 :Text;
  updatedAt @15 :Text;
}

struct GradeChange {
  base @0 :Common.Base;
  customerId @1 :Text;
  newGrade @2 :CustomerGrade;
}

struct FollowUpTask {
  base @0 :Common.Base;
  customerId @1 :Text;
  title @2 :Text;
  description @3 :Text;
  dueDate @4 :Text;
  status @5 :Common.DocumentStatus;
  priority @6 :Common.Priority;
  type @7 :Text;
  amount @8 :Float64;
  contact @9 :Text;
  notes @10 :Text;
  completedAt @11 :Text;
}

struct GoodsReceivedNote {
  base @0 :Common.Base;
  purchaseOrderId @1 :Text;
  grnNumber @2 :Text;
  receivedDate @3 :Text;
  receivedBy @4 :Text;
  warehouseId @5 :Text;
  supplierDnNumber @6 :Text;
  qcStatus @7 :Common.ApprovalStatus;
  qcNotes @8 :Text;
  qcDate @9 :Text;
  qcBy @10 :Text;
  updatedBy @11 :Text;
  items @12 :List(GRNItem);
}

struct GRNItem {
  base @0 :Common.Base;
  grnId @1 :Text;
  poItemId @2 :Text;
  productId @3 :Text;
  quantityOrdered @4 :Float64;
  quantityReceived @5 :Float64;
  quantityAccepted @6 :Float64;
  quantityRejected @7 :Float64;
  rejectionReason @8 :Text;
}

struct OfferFollowUp {
  base @0 :Common.Base;
  offerId @1 :Text;
  followUpDate @2 :Text;
  notes @3 :Text;
  status @4 :Common.DocumentStatus;
  completedAt @5 :Text;
  completedBy @6 :Text;
}

struct OfferNote {
  base @0 :Common.Base;
  offerId @1 :Text;
  noteDate @2 :Text;
  content @3 :Text;
}

struct CustomerProfile {
  customer @0 :CustomerMaster;
  contacts @1 :List(CustomerContact);
  notes @2 :List(EntityNote);
  openTasks @3 :List(FollowUpTask);
  arAging @4 :Finance.ARAgingBucket;
}

struct SupplierProfile {
  supplier @0 :SupplierMaster;
  contacts @1 :List(SupplierContact);
  notes @2 :List(EntityNote);
  issues @3 :List(SupplierIssue);
}

struct PipelineSnapshot {
  asOfDate @0 :Text;
  opportunities @1 :List(Opportunity);
  totalRevenueBhd @2 :Float64;
  totalProfitBhd @3 :Float64;
  weightedRevenueBhd @4 :Float64;
  wonCount @5 :Int64;
  lostCount @6 :Int64;
  openCount @7 :Int64;
}
