@0xd0b8835cc94b27a5;

using Go = import "/go.capnp";
using Common = import "common.capnp";

$Go.package("syncschema");
$Go.import("ph_holdings_app/schemas/go/syncschema");

# File watcher, database sync, and Tally import schema contracts.

enum FileSyncStatus {
  queued @0;
  processing @1;
  synced @2;
  failed @3;
  conflict @4;
  skippedLarge @5;
}

enum SyncDirection {
  push @0;
  pull @1;
}

enum ConflictState {
  none @0;
  localWins @1;
  remoteWins @2;
}

enum TallyImportStatus {
  pending @0;
  imported @1;
  matched @2;
  duplicate @3;
  error @4;
}

struct FileWatchEvent {
  base @0 :Common.Base;
  filePath @1 :Text;
  eventType @2 :Text;
}

struct FileSyncState {
  path @0 :Text;
  eventType @1 :Text;
  status @2 :FileSyncStatus;
  lastModified @3 :Text;
  lastSynced @4 :Text;
  remoteHash @5 :Text;
  localHash @6 :Text;
  retryCount @7 :Int64;
  lastError @8 :Text;
  metadata @9 :List(Common.KeyValue);
}

struct SyncStatus {
  base @0 :Common.Base;
  filePath @1 :Text;
  status @2 :FileSyncStatus;
  lastSyncTime @3 :Text;
}

struct SyncRecord {
  base @0 :Common.Base;
  syncTable @1 :Text;
  recordId @2 :Text;
  syncedAt @3 :Text;
  direction @4 :SyncDirection;
  remoteVersion @5 :Int64;
  localVersion @6 :Int64;
  conflictState @7 :ConflictState;
}

struct TallyInvoiceImport {
  base @0 :Common.Base;
  importBatch @1 :Text;
  year @2 :Int64;
  invoiceNumber @3 :Text;
  customerName @4 :Text;
  matchedCustomerId @5 :Text;
  invoiceDate @6 :Text;
  amount @7 :Float64;
  currency @8 :Common.CurrencyCode;
  status @9 :TallyImportStatus;
  rawData @10 :Text;
}

struct TallyPurchaseImport {
  base @0 :Common.Base;
  importBatch @1 :Text;
  year @2 :Int64;
  invoiceNumber @3 :Text;
  supplierName @4 :Text;
  matchedSupplierId @5 :Text;
  invoiceDate @6 :Text;
  amount @7 :Float64;
  currency @8 :Common.CurrencyCode;
  status @9 :TallyImportStatus;
  rawData @10 :Text;
}

struct SyncRunStatus {
  lastSyncStatus @0 :Text;
  lastSyncTime @1 :Text;
  cloudSyncEnabled @2 :Bool;
  isRunning @3 :Bool;
  pendingPushCount @4 :Int64;
  pendingPullCount @5 :Int64;
  failedCount @6 :Int64;
}
