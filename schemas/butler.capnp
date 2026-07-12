@0xd61eab82bab79d77;

using Go = import "/go.capnp";
using Common = import "common.capnp";
using CRM = import "crm.capnp";

$Go.package("butler");
$Go.import("ph_holdings_app/schemas/go/butler");

# Butler intelligence schema contracts.

enum ChatRole {
  user @0;
  assistant @1;
  system @2;
}

enum ActionStatus {
  none @0;
  pending @1;
  executed @2;
  failed @3;
  dismissed @4;
}

struct ButlerResponse {
  message @0 :Text;
  actions @1 :List(ButlerAction);
  confidence @2 :Float64;
  context @3 :List(Common.KeyValue);
  metadata @4 :ButlerResponseMetadata;
}

struct ButlerResponseMetadata {
  usedBackend @0 :Text;
  requestedModel @1 :Text;
  usedModel @2 :Text;
  fallbackReason @3 :Text;
  financeDataAccess @4 :Bool;
  contextMode @5 :Text;
  dataCoverage @6 :List(Text);
  entityResolution @7 :List(Common.KeyValue);
  generatedAt @8 :Text;
  error @9 :Text;
}

struct ButlerAction {
  type @0 :Text;
  target @1 :Text;
  label @2 :Text;
  data @3 :List(Common.KeyValue);
}

struct Intent {
  rawQuery @0 :Text;
  domain @1 :Text;
  entityName @2 :Text;
  personName @3 :Text;
  referenceKind @4 :Text;
  confidence @5 :Float64;
  keywords @6 :List(Text);
  needsScraper @7 :Bool;
  isComplex @8 :Bool;
}

struct ButlerResolvedEntity {
  entityType @0 :Text;
  entityId @1 :Text;
  displayName @2 :Text;
  confidence @3 :Float64;
  matchReason @4 :Text;
  relatedCustomerId @5 :Text;
  relatedCustomer @6 :Text;
  ambiguous @7 :Bool;
  alternatives @8 :List(Common.KeyValue);
}

struct PredictionRecord {
  base @0 :Common.Base;
  customerId @1 :Text;
  customerName @2 :Text;
  grade @3 :CRM.CustomerGrade;
  predictedDays @4 :Int64;
  confidence @5 :Float64;
  r1 @6 :Float64;
  r2 @7 :Float64;
  r3 @8 :Float64;
}

struct WinProbabilityPrediction {
  base @0 :Common.Base;
  offerId @1 :Text;
  predictedProbability @2 :Float64;
}

struct DiscountRecommendationRecord {
  base @0 :Common.Base;
  offerId @1 :Text;
  recommendedDiscount @2 :Float64;
}

struct CustomerSnapshot {
  base @0 :Common.Base;
  customerId @1 :Text;
  orderValue @2 :Float64;
}

struct ActualOutcome {
  base @0 :Common.Base;
  customerId @1 :Text;
  wasPaid @2 :Bool;
}

struct PaymentPredictionAccuracy {
  base @0 :Common.Base;
  customerId @1 :Text;
  wasAccurate @2 :Bool;
}

struct Conversation {
  base @0 :Common.Base;
  title @1 :Text;
  summary @2 :Text;
  isActive @3 :Bool;
  lastMsgAt @4 :Text;
}

struct ChatMessage {
  base @0 :Common.Base;
  conversationId @1 :Text;
  role @2 :ChatRole;
  content @3 :Text;
  tokensUsed @4 :Int64;
  messageType @5 :Text;
  actionType @6 :Text;
  actionTarget @7 :Text;
  actionLabel @8 :Text;
  actionData @9 :Text;
  actionStatus @10 :ActionStatus;
  actionMetadata @11 :Text;
}
