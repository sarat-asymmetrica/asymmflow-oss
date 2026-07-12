// Wave 9.4 B5c: read-only aggregation for the persistent approvals queue
// (Design Constitution Article V.2 — the Task class persists until *done*
// and lives in a work queue with a clear owner; Article V.4 — reading a
// notification may never strand an actionable item).
//
// These wrap the two new backend list methods (pkg/infra/deletion +
// employee_archive_service.go). Both are admin/reviewer-gated server-side
// and return an empty array for non-reviewer sessions rather than an error,
// so callers can poll them unconditionally.
//
// Review mutations are intentionally NOT duplicated here — NotificationsScreen
// and ApprovalsQueueScreen both call the existing server-resolved-reviewer
// review functions exported by `$lib/api/collaboration`
// (reviewDeleteApprovalRequest / reviewEmployeeArchiveRequest) so there is a
// single source of truth for the approve/reject transition.

import { ListDeleteApprovalRequests, ListEmployeeArchiveRequests } from "../../../wailsjs/go/main/App";
import { normalizeWailsDateTime } from "$lib/utils/wailsInterop";

export interface DeleteApprovalItem {
  id: string;
  entity_type: string;
  entity_id: string;
  entity_label: string;
  requested_by: string;
  requested_by_name?: string;
  requested_role?: string;
  reason?: string;
  status?: string;
  reviewed_by?: string;
  reviewed_by_name?: string;
  reviewed_at?: string;
  review_notes?: string;
  created_at?: string;
}

export interface EmployeeArchiveApprovalItem {
  id: string;
  employee_id: string;
  employee_name: string;
  requested_by: string;
  requested_by_name?: string;
  reason?: string;
  status?: string;
  required_approvals?: number;
  first_approved_by?: string;
  first_approved_by_name?: string;
  first_approved_at?: string;
  second_approved_by?: string;
  second_approved_by_name?: string;
  second_approved_at?: string;
  rejected_by?: string;
  rejected_by_name?: string;
  rejected_at?: string;
  review_notes?: string;
  created_at?: string;
}

const isDesktop = () => Boolean((window as any)?.go?.main?.App);

function toDeleteApprovalItem(request: any): DeleteApprovalItem {
  return {
    id: request.id,
    entity_type: request.entity_type,
    entity_id: request.entity_id,
    entity_label: request.entity_label,
    requested_by: request.requested_by,
    requested_by_name: request.requested_by_name,
    requested_role: request.requested_role,
    reason: request.reason,
    status: request.status,
    reviewed_by: request.reviewed_by,
    reviewed_by_name: request.reviewed_by_name,
    reviewed_at: normalizeWailsDateTime(request.reviewed_at),
    review_notes: request.review_notes,
    created_at: normalizeWailsDateTime(request.created_at),
  };
}

function toEmployeeArchiveApprovalItem(request: any): EmployeeArchiveApprovalItem {
  return {
    id: request.id,
    employee_id: request.employee_id,
    employee_name: request.employee_name,
    requested_by: request.requested_by,
    requested_by_name: request.requested_by_name,
    reason: request.reason,
    status: request.status,
    required_approvals: request.required_approvals,
    first_approved_by: request.first_approved_by,
    first_approved_by_name: request.first_approved_by_name,
    first_approved_at: normalizeWailsDateTime(request.first_approved_at),
    second_approved_by: request.second_approved_by,
    second_approved_by_name: request.second_approved_by_name,
    second_approved_at: normalizeWailsDateTime(request.second_approved_at),
    rejected_by: request.rejected_by,
    rejected_by_name: request.rejected_by_name,
    rejected_at: normalizeWailsDateTime(request.rejected_at),
    review_notes: request.review_notes,
    created_at: normalizeWailsDateTime(request.created_at),
  };
}

/**
 * List delete-approval requests by status (default "pending"). Returns an
 * empty array for non-reviewer sessions and when running outside the
 * desktop shell — never throws for authorization reasons.
 */
export async function listDeleteApprovals(status = "pending"): Promise<DeleteApprovalItem[]> {
  if (!isDesktop()) return [];
  const rows = await ListDeleteApprovalRequests(status);
  return (rows || []).map(toDeleteApprovalItem);
}

/**
 * List employee-archive requests by status (default "pending"). Returns an
 * empty array for non-reviewer sessions and when running outside the
 * desktop shell — never throws for authorization reasons.
 */
export async function listEmployeeArchiveApprovals(status = "pending"): Promise<EmployeeArchiveApprovalItem[]> {
  if (!isDesktop()) return [];
  const rows = await ListEmployeeArchiveRequests(status);
  return (rows || []).map(toEmployeeArchiveApprovalItem);
}
