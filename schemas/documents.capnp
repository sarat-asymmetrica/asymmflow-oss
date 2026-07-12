@0xb3ef4a342087e75d;

using Go = import "/go.capnp";
using Common = import "common.capnp";

$Go.package("documents");
$Go.import("ph_holdings_app/schemas/go/documents");

# Document generation, OCR, classification, and orchestration contracts.

enum DocumentType {
  unknown @0;
  rfq @1;
  quote @2;
  purchaseOrder @3;
  invoice @4;
  supplierInvoice @5;
  deliveryNote @6;
  bankStatement @7;
  contract @8;
  receipt @9;
  other @10;
}

enum OCRTier {
  local @0;
  aimlapi @1;
  consensus @2;
}

enum SourceType {
  file @0;
  url @1;
  bytes @2;
  zip @3;
  directory @4;
}

struct CompanyInfo {
  name @0 :Text;
  legalName @1 :Text;
  address @2 :Text;
  phone @3 :Text;
  email @4 :Text;
  website @5 :Text;
  taxNumber @6 :Text;
  commercialReg @7 :Text;
}

struct BrandingConfig {
  division @0 :Text;
  logoPath @1 :Text;
  letterheadPath @2 :Text;
  primaryColor @3 :Text;
  secondaryColor @4 :Text;
  footerText @5 :Text;
  documentPrefix @6 :Text;
  defaultCurrency @7 :Common.CurrencyCode;
}

struct FileWatchEvent {
  base @0 :Common.Base;
  filePath @1 :Text;
  eventType @2 :Text;
}

struct BankStatementFile {
  base @0 :Common.Base;
  bankStatementId @1 :Text;
  fileName @2 :Text;
  fileType @3 :Text;
  fileSize @4 :Int64;
  fileHash @5 :Text;
  storagePath @6 :Text;
  isStored @7 :Bool;
  ocrEngine @8 :Text;
  ocrConfidence @9 :Float64;
  ocrProcessedAt @10 :Text;
}

struct OCRField {
  key @0 :Text;
  value @1 :Text;
  confidence @2 :Float64;
  pageNumber @3 :Int64;
}

struct OCRPage {
  pageNumber @0 :Int64;
  text @1 :Text;
  confidence @2 :Float64;
  width @3 :Int64;
  height @4 :Int64;
}

struct OCRError {
  stage @0 :Text;
  message @1 :Text;
  timestamp @2 :Text;
  fatal @3 :Bool;
}

struct OCRRequest {
  sourcePath @0 :Text;
  sourceType @1 :SourceType;
  fileName @2 :Text;
  documentType @3 :DocumentType;
  language @4 :Text;
  countryCode @5 :Text;
  enableGpu @6 :Bool;
  enablePreprocessing @7 :Bool;
  enableTranslation @8 :Bool;
  targetLanguage @9 :Text;
  tier @10 :OCRTier;
  fallbackToAimlapi @11 :Bool;
  checkpointDir @12 :Text;
  resumeFromCheckpoint @13 :Bool;
}

struct OCRResult {
  text @0 :Text;
  fields @1 :List(OCRField);
  pages @2 :List(OCRPage);
  confidence @3 :Float64;
  documentType @4 :DocumentType;
  detectedLanguage @5 :Text;
  pageCount @6 :Int64;
  processingTimeMs @7 :Int64;
  tier @8 :OCRTier;
  gpuUsed @9 :Bool;
  estimatedCostUsd @10 :Float64;
  errors @11 :List(OCRError);
  warnings @12 :List(Text);
}

struct ClassificationResult {
  documentType @0 :DocumentType;
  confidence @1 :Float64;
  method @2 :Text;
  routeTo @3 :Text;
  suggestedAction @4 :Text;
  keywordsFound @5 :List(Text);
  explanation @6 :Text;
}

# Business Memory Intake durable/cross-module contracts.

enum BusinessMemorySourceKind {
  message @0;
  email @1;
  pdf @2;
  scan @3;
  screenshot @4;
  excel @5;
  folder @6;
  inboxRecord @7;
  other @8;
}

enum BusinessMemoryReviewStatus {
  new @0;
  needsReview @1;
  corrected @2;
  linked @3;
  rejected @4;
  archived @5;
}

enum BusinessMemoryFieldStatus {
  extracted @0;
  missing @1;
  inferred @2;
  needsConfirmation @3;
  corrected @4;
}

enum BusinessMemoryReviewDecision {
  acceptProposal @0;
  needsInput @1;
  correctField @2;
  rejectCandidate @3;
  archive @4;
}

struct BusinessMemorySourceRef {
  id @0 :Text;
  label @1 :Text;
  path @2 :Text;
  kind @3 :BusinessMemorySourceKind;
  processedAt @4 :Text;
}

struct BusinessMemoryClassification {
  type @0 :Text;
  method @1 :Text;
  routeTo @2 :Text;
  reason @3 :Text;
  keywords @4 :List(Text);
  confidence @5 :Float64;
}

struct BusinessMemoryExtractedField {
  name @0 :Text;
  label @1 :Text;
  value @2 :Text;
  status @3 :BusinessMemoryFieldStatus;
  confidence @4 :Float64;
  source @5 :Text;
}

struct BusinessMemorySuggestedLink {
  id @0 :Text;
  label @1 :Text;
  reason @2 :Text;
  businessObjectType @3 :Text;
  requiredDeterministicService @4 :Text;
}

struct BusinessMemoryAuditRef {
  type @0 :Text;
  sourceId @1 :Text;
  summary @2 :Text;
  timestamp @3 :Text;
}

struct BusinessMemoryCandidate {
  id @0 :Text;
  source @1 :BusinessMemorySourceRef;
  sourceKind @2 :BusinessMemorySourceKind;
  businessObjectType @3 :Text;
  classification @4 :BusinessMemoryClassification;
  extractedFields @5 :List(BusinessMemoryExtractedField);
  suggestedLinks @6 :List(BusinessMemorySuggestedLink);
  reviewStatus @7 :BusinessMemoryReviewStatus;
  auditRefs @8 :List(BusinessMemoryAuditRef);
  confidence @9 :Float64;
  warnings @10 :List(Text);
}

struct BusinessMemoryContextPack {
  candidateId @0 :Text;
  sourceSummary @1 :Text;
  sourceKind @2 :BusinessMemorySourceKind;
  businessObjectType @3 :Text;
  classification @4 :BusinessMemoryClassification;
  extractedFields @5 :List(BusinessMemoryExtractedField);
  missingFields @6 :List(Text);
  suggestedDeterministicServiceTargets @7 :List(Text);
  reviewStatus @8 :BusinessMemoryReviewStatus;
  warnings @9 :List(Text);
  auditRefs @10 :List(BusinessMemoryAuditRef);
  allowedAgentActions @11 :List(Text);
  forbiddenAgentActions @12 :List(Text);
}

struct BusinessMemoryReviewRecord {
  id @0 :Text;
  candidateId @1 :Text;
  sourceId @2 :Text;
  decision @3 :BusinessMemoryReviewDecision;
  reviewStatus @4 :BusinessMemoryReviewStatus;
  proposedDeterministicService @5 :Text;
  actor @6 :Text;
  reason @7 :Text;
  correlationId @8 :Text;
  createdAt @9 :Text;
}

struct BusinessMemoryCandidateBatch {
  candidates @0 :List(BusinessMemoryCandidate);
}

struct BusinessMemoryContextPackBatch {
  contextPacks @0 :List(BusinessMemoryContextPack);
}

struct BusinessMemoryReviewRecordBatch {
  records @0 :List(BusinessMemoryReviewRecord);
}

struct DocumentAnalysisRequest {
  text @0 :Text;
  documentType @1 :DocumentType;
  metadata @2 :List(Common.KeyValue);
  maxTextLength @3 :Int64;
}

struct RFQLineItem {
  description @0 :Text;
  quantity @1 :Int64;
  unit @2 :Text;
  partNumber @3 :Text;
  notes @4 :Text;
}

struct ButlerOCRInsight {
  summary @0 :Text;
  extractedItems @1 :List(RFQLineItem);
  detectedCustomer @2 :Text;
  detectedProject @3 :Text;
  requiredDeadline @4 :Text;
  confidence @5 :Float64;
  suggestedActions @6 :List(Text);
  documentType @7 :DocumentType;
  extractedMetadata @8 :List(Common.KeyValue);
}

struct OCRBatchRequest {
  requests @0 :List(OCRRequest);
  maxConcurrency @1 :Int64;
}

struct OCRBatchResult {
  results @0 :List(OCRResult);
  totalTimeMs @1 :Int64;
  successCount @2 :Int64;
  failureCount @3 :Int64;
  averageConfidence @4 :Float64;
}

struct OCRProgress {
  completed @0 :Int64;
  total @1 :Int64;
  currentFile @2 :Text;
  percentage @3 :Float64;
  estimatedEtaMs @4 :Int64;
}

struct DocumentPipelineRequest {
  ocr @0 :OCRRequest;
  classify @1 :Bool;
  analyzeWithButler @2 :Bool;
  persistFile @3 :Bool;
}

struct DocumentPipelineResult {
  file @0 :BankStatementFile;
  ocr @1 :OCRResult;
  classification @2 :ClassificationResult;
  insight @3 :ButlerOCRInsight;
  issues @4 :List(Common.ValidationIssue);
}
