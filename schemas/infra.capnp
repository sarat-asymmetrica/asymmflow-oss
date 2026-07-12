@0xc198fb847f509fc5;

using Go = import "/go.capnp";
using Common = import "common.capnp";

$Go.package("infra");
$Go.import("ph_holdings_app/schemas/go/infra");

# Shared infrastructure schema contracts.

enum DeviceStatus {
  firstSetup @0;
  pending @1;
  approved @2;
  blocked @3;
  revoked @4;
}

enum JobStatus {
  pending @0;
  running @1;
  completed @2;
  failed @3;
  cancelled @4;
}

struct Setting {
  base @0 :Common.Base;
  key @1 :Text;
  value @2 :Text;
  category @3 :Text;
  description @4 :Text;
  isEncrypted @5 :Bool;
}

struct Role {
  base @0 :Common.Base;
  name @1 :Text;
  displayName @2 :Text;
  description @3 :Text;
  permissions @4 :Text;
  isActive @5 :Bool;
  isSystem @6 :Bool;
}

struct User {
  base @0 :Common.Base;
  username @1 :Text;
  email @2 :Text;
  passwordHash @3 :Text;
  roleId @4 :Text;
  fullName @5 :Text;
  displayName @6 :Text;
  department @7 :Text;
  jobTitle @8 :Text;
  isActive @9 :Bool;
  lastLoginAt @10 :Text;
  passwordChangedAt @11 :Text;
  mustChangePassword @12 :Bool;
  roleName @13 :Text;
  role @14 :Role;
}

struct Device {
  base @0 :Common.Base;
  machineId @1 :Text;
  deviceName @2 :Text;
  osInfo @3 :Text;
  firstSeenAt @4 :Text;
  lastSeenAt @5 :Text;
  status @6 :DeviceStatus;
  approvedBy @7 :Text;
  approvedAt @8 :Text;
  isAdminDevice @9 :Bool;
  notes @10 :Text;
  approverName @11 :Text;
}

struct DeviceUser {
  base @0 :Common.Base;
  deviceId @1 :Text;
  userId @2 :Text;
  isPrimary @3 :Bool;
  user @4 :User;
  device @5 :Device;
}

struct UserSession {
  base @0 :Common.Base;
  userId @1 :Text;
  token @2 :Text;
  refreshToken @3 :Text;
  accessTokenExpiry @4 :Text;
  refreshTokenExpiry @5 :Text;
  lastActivityAt @6 :Text;
  isActive @7 :Bool;
  invalidatedAt @8 :Text;
  invalidatedReason @9 :Text;
}

struct Alert {
  base @0 :Common.Base;
  alertType @1 :Text;
  severity @2 :Common.RiskLevel;
  title @3 :Text;
  message @4 :Text;
  isActive @5 :Bool;
  isAcknowledged @6 :Bool;
}

struct AuditLog {
  base @0 :Common.Base;
  userId @1 :Text;
  action @2 :Text;
  resource @3 :Text;
}

struct Job {
  base @0 :Common.Base;
  type @1 :Text;
  status @2 :JobStatus;
  input @3 :Text;
  output @4 :Text;
  error @5 :Text;
  progress @6 :Int64;
  startedAt @7 :Text;
  completedAt @8 :Text;
  attempts @9 :Int64;
  maxAttempts @10 :Int64;
}

struct BackupPolicy {
  autoBackupEnabled @0 :Bool;
  frequencyDays @1 :Int64;
  lastBackupAt @2 :Text;
  lastBackupPath @3 :Text;
  nextBackupDueAt @4 :Text;
  dueNow @5 :Bool;
}
