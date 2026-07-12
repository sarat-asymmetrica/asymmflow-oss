@0xb7ac2215649a020c;

using Go = import "/go.capnp";
$Go.package("common");
$Go.import("ph_holdings_app/schemas/go/common");

# Shared schema primitives for AsymmFlow contracts.

enum RecordStatus {
  active @0;
  inactive @1;
  archived @2;
  deleted @3;
  draft @4;
}

enum ApprovalStatus {
  pending @0;
  approved @1;
  rejected @2;
  cancelled @3;
  notRequired @4;
}

enum CurrencyCode {
  bhd @0;
  usd @1;
  eur @2;
  gbp @3;
  sar @4;
  aed @5;
  qar @6;
  omr @7;
  kwd @8;
  inr @9;
}

enum DocumentStatus {
  draft @0;
  prepared @1;
  sent @2;
  acknowledged @3;
  issued @4;
  partiallyPaid @5;
  paid @6;
  overdue @7;
  voided @8;
  cancelled @9;
  closed @10;
}

enum SyncState {
  localOnly @0;
  pendingPush @1;
  pendingPull @2;
  synced @3;
  conflicted @4;
  failed @5;
}

enum RiskLevel {
  unknown @0;
  low @1;
  medium @2;
  high @3;
  critical @4;
}

enum Priority {
  low @0;
  normal @1;
  high @2;
  urgent @3;
}

enum Direction {
  inbound @0;
  outbound @1;
}

struct Base {
  id @0 :Text;
  createdAt @1 :Text;
  updatedAt @2 :Text;
  deletedAt @3 :Text;
  createdBy @4 :Text;
  updatedBy @5 :Text;
  status @6 :RecordStatus;
  syncState @7 :SyncState;
}

struct Money {
  amount @0 :Float64;
  currency @1 :CurrencyCode;
}

struct Percentage {
  value @0 :Float64;
}

struct DateRange {
  from @0 :Text;
  to @1 :Text;
}

struct Address {
  line1 @0 :Text;
  line2 @1 :Text;
  city @2 :Text;
  region @3 :Text;
  postalCode @4 :Text;
  country @5 :Text;
}

struct ContactInfo {
  name @0 :Text;
  email @1 :Text;
  phone @2 :Text;
  mobile @3 :Text;
  designation @4 :Text;
}

struct AttachmentRef {
  id @0 :Text;
  fileName @1 :Text;
  mimeType @2 :Text;
  path @3 :Text;
  sha256 @4 :Text;
  sizeBytes @5 :UInt64;
}

struct KeyValue {
  key @0 :Text;
  value @1 :Text;
}

struct ValidationIssue {
  field @0 :Text;
  code @1 :Text;
  message @2 :Text;
  severity @3 :RiskLevel;
}

struct PageRequest {
  page @0 :UInt32;
  pageSize @1 :UInt32;
  sortBy @2 :Text;
  descending @3 :Bool;
}

struct PageInfo {
  page @0 :UInt32;
  pageSize @1 :UInt32;
  totalItems @2 :UInt64;
  totalPages @3 :UInt32;
}
