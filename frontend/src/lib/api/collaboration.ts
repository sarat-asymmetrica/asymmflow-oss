import { AddCollaborativeProjectMember, AddCollaborativeTaskComment, ArchiveCollaborativeProject, CreateCollaborativeProject, CreateCollaborativeTask, DeleteCollaborativeProject, DeleteCollaborativeTask, CreateEmployeeAccessLink, CreateEmployeeProfile, GetCollaborativeTask, GetCurrentEmployeeContext, GetProjectTaskCounts, GetUnreadNotificationsCount, ListEmployeeContributionSummaries, ListCollaborativeProjects, ListCollaborativeProjectMembers, ListCollaborativeProjectActivity, ListCollaborativeProjectTasks, ListCollaborativeTaskActivity, ListCollaborativeTaskComments, ListCollaborativeTeamTasks, ListEmployeeAccessLinks, ListEmployeeProjectAssignments, ListEmployeeProfiles, ListMyCollaborativeTasks, ListNotificationFeed, MarkNotificationAsRead, ReassignCollaborativeTask, ReassignEmployeeLicenseAccess, ReassignEmployeeManager, RefreshCollaborativeWorkspace, SetEmployeeEmploymentState, ShelveCollaborativeProject, UpdateCollaborativeProject, UpdateCollaborativeTaskDueDate, UpdateCollaborativeTask, UpdateCollaborativeTaskStatus, UpdateEmployeeProfile } from "../../../wailsjs/go/main/SyncServiceBinding";
import { RequestEmployeeArchive, ReviewEmployeeArchiveRequest, CreateEmployeeDocument, UpdateEmployeeDocument, DeleteEmployeeDocument, ListEmployeeDocuments, GetEmployeeAllocationSummary } from "../../../wailsjs/go/main/App";
import { CreateUser, ListLicenseKeys, ListRoles, ListUsers, ReviewDeleteApprovalRequest, UpdateUser } from "../../../wailsjs/go/main/InfraService";
import { deletion, infra, main } from "../../../wailsjs/go/models";
import { buildWailsInput, normalizeWailsDateTime } from "$lib/utils/wailsInterop";

export interface EmployeeProfile {
  id: string;
  employee_code: string;
  full_name: string;
  preferred_name?: string;
  email?: string;
  phone?: string;
  department?: string;
  job_title?: string;
  employment_status?: string;
  manager_employee_id?: string;
  manager_name?: string;
  start_date?: string;
  end_date?: string;
  emergency_contact?: string;
  notes?: string;
  is_active?: boolean;
  archived_at?: string;
  archived_by?: string;
  archive_reason?: string;
  archive_request_id?: string;
}

export interface EmployeeArchiveApproval {
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
}

export interface EmployeeAccessLink {
  id: string;
  employee_id: string;
  employee_name?: string;
  license_key: string;
  user_id?: string;
  device_id?: string;
  device_name?: string;
  access_status?: string;
  is_primary?: boolean;
}

export interface CurrentEmployeeContext {
  employee_id: string;
  employee_name: string;
  license_key?: string;
  license_role?: string;
  device_id?: string;
  user_id?: string;
  resolved_by?: string;
  permissions?: string[];
}

export interface CollaborativeProject {
  id: string;
  name: string;
  project_type?: string;
  description?: string;
  status?: string;
  customer_id?: string;
  opportunity_id?: string;
  order_id?: string;
  customer_name?: string;
  end_user_name?: string;
  opportunity_key?: string;
  customer_poc_name?: string;
  customer_poc_email?: string;
  customer_poc_phone?: string;
  starts_on?: string;
  ends_on?: string;
}

export interface ProjectMember {
  id: string;
  project_id: string;
  employee_id: string;
  employee_name?: string;
  project_name?: string;
  role?: string;
  allocation_percent?: number;
  is_active?: boolean;
  joined_at?: string;
  left_at?: string;
}

// Wave 9.8 B3: read-only precheck for the allocation-capacity WARN. The
// server computes other_projects_total (never trust a client-side sum) —
// the UI only ever displays what comes back here.
export interface AllocationProjectLine {
  project_id: string;
  project_name?: string;
  allocation_percent: number;
}

export interface AllocationSummary {
  employee_id: string;
  other_projects_total: number;
  projects: AllocationProjectLine[];
}

export interface CollaborativeTask {
  id: string;
  title: string;
  description?: string;
  task_type?: string;
  status?: string;
  blocked_reason?: string;
  priority?: string;
  due_date?: string;
  customer_id?: string;
  opportunity_id?: string;
  order_id?: string;
  project_id?: string;
  creator_employee_id?: string;
  assignee_employee_id?: string;
  creator_name?: string;
  assignee_name?: string;
  completed_at?: string;
  started_at?: string;
  last_comment_at?: string;
}

export interface TaskCommentItem {
  id: string;
  task_id: string;
  employee_id: string;
  employee_name?: string;
  body: string;
  created_at?: string;
}

export interface TaskActivityItem {
  id: string;
  task_id: string;
  employee_id: string;
  employee_name?: string;
  activity_type?: string;
  detail: string;
  metadata_json?: string;
  created_at?: string;
}

export interface NotificationItem {
  id: string;
  notification_type?: string;
  title: string;
  message: string;
  status?: string;
  source_type?: string;
  source_id?: string;
  action_route?: string;
  action_payload?: string;
  read_at?: string;
  created_at?: string;
}

export interface LicenseKeySummary {
  id: string;
  key: string;
  role: string;
  device_id?: string;
  assigned_to?: string;
  display_name?: string;
  status?: string;
}

export interface LoginUserSummary {
  id: string;
  username: string;
  email?: string;
  full_name?: string;
  department?: string;
  job_title?: string;
  role_id?: string;
  role_name?: string;
  is_active?: boolean;
}

export interface RoleSummary {
  id: string;
  name: string;
  display_name?: string;
}

export interface EmployeeContributionSummary {
  employee_id: string;
  employee_code: string;
  employee_name: string;
  department?: string;
  job_title?: string;
  manager_employee_id?: string;
  manager_name?: string;
  employment_status?: string;
  is_active?: boolean;
  active_project_count: number;
  active_task_count: number;
  completed_task_count: number;
  blocked_task_count: number;
  overdue_task_count: number;
  completion_rate: number;
  primary_license_key?: string;
  primary_device_name?: string;
}

const isDesktop = () => Boolean((window as any)?.go?.main?.App);
const COLLABORATIVE_REFRESH_INTERVAL_MS = 15_000;
const COLLABORATIVE_REFRESH_TIMEOUT_MS = 5_000;

let collaborativeRefreshInFlight: Promise<void> | null = null;
let lastCollaborativeRefreshAt = 0;

function withTimeout<T>(promise: Promise<T>, timeoutMs: number, label: string): Promise<T> {
  let timeoutID: ReturnType<typeof setTimeout> | undefined;
  const timeout = new Promise<T>((_, reject) => {
    timeoutID = setTimeout(() => reject(new Error(`${label} timed out`)), timeoutMs);
  });
  return Promise.race([promise, timeout]).finally(() => {
    if (timeoutID) clearTimeout(timeoutID);
  });
}

function toEmployeeProfile(employee: main.Employee): EmployeeProfile {
  return {
    id: employee.id,
    employee_code: employee.employee_code,
    full_name: employee.full_name,
    preferred_name: employee.preferred_name,
    email: employee.email,
    phone: employee.phone,
    department: employee.department,
    job_title: employee.job_title,
    employment_status: employee.employment_status,
    manager_employee_id: employee.manager_employee_id,
    manager_name: employee.manager_name,
    start_date: normalizeWailsDateTime(employee.start_date),
    end_date: normalizeWailsDateTime(employee.end_date),
    emergency_contact: employee.emergency_contact,
    notes: employee.notes,
    is_active: employee.is_active,
    archived_at: normalizeWailsDateTime(employee.archived_at),
    archived_by: employee.archived_by,
    archive_reason: employee.archive_reason,
    archive_request_id: employee.archive_request_id,
  };
}

function toEmployeeArchiveApproval(request: main.EmployeeArchiveRequest): EmployeeArchiveApproval {
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
  };
}

function toEmployeeAccessLink(link: main.EmployeeAccessLink): EmployeeAccessLink {
  return {
    id: link.id,
    employee_id: link.employee_id,
    employee_name: link.employee_name,
    license_key: link.license_key,
    user_id: link.user_id,
    device_id: link.device_id,
    device_name: link.device_name,
    access_status: link.access_status,
    is_primary: link.is_primary,
  };
}

function toProject(project: main.Project): CollaborativeProject {
  return {
    id: project.id,
    name: project.name,
    project_type: project.project_type,
    description: project.description,
    status: project.status,
    customer_id: project.customer_id,
    opportunity_id: project.opportunity_id,
    order_id: project.order_id,
    customer_name: project.customer_name,
    end_user_name: project.end_user_name,
    opportunity_key: project.opportunity_key,
    customer_poc_name: project.customer_poc_name,
    customer_poc_email: project.customer_poc_email,
    customer_poc_phone: project.customer_poc_phone,
    starts_on: normalizeWailsDateTime(project.starts_on),
    ends_on: normalizeWailsDateTime(project.ends_on),
  };
}

function toProjectMember(member: main.ProjectMember): ProjectMember {
  return {
    id: member.id,
    project_id: member.project_id,
    employee_id: member.employee_id,
    employee_name: member.employee_name,
    project_name: member.project_name,
    role: member.role,
    allocation_percent: member.allocation_percent,
    is_active: member.is_active,
    joined_at: normalizeWailsDateTime(member.joined_at),
    left_at: normalizeWailsDateTime(member.left_at),
  };
}

function toTask(task: main.TaskItem): CollaborativeTask {
  return {
    id: task.id,
    title: task.title,
    description: task.description,
    task_type: task.task_type,
    status: task.status,
    blocked_reason: task.blocked_reason,
    priority: task.priority,
    due_date: normalizeWailsDateTime(task.due_date),
    customer_id: task.customer_id,
    opportunity_id: task.opportunity_id,
    order_id: task.order_id,
    project_id: task.project_id,
    creator_employee_id: task.creator_employee_id,
    assignee_employee_id: task.assignee_employee_id,
    creator_name: task.creator_name,
    assignee_name: task.assignee_name,
    completed_at: normalizeWailsDateTime(task.completed_at),
    started_at: normalizeWailsDateTime(task.started_at),
    last_comment_at: normalizeWailsDateTime(task.last_comment_at),
  };
}

function toTaskComment(comment: main.TaskComment): TaskCommentItem {
  return {
    id: comment.id,
    task_id: comment.task_id,
    employee_id: comment.employee_id,
    employee_name: comment.employee_name,
    body: comment.body,
    created_at: normalizeWailsDateTime(comment.created_at),
  };
}

function toTaskActivity(activity: main.TaskActivity): TaskActivityItem {
  return {
    id: activity.id,
    task_id: activity.task_id,
    employee_id: activity.employee_id,
    employee_name: activity.employee_name,
    activity_type: activity.activity_type,
    detail: activity.detail,
    metadata_json: activity.metadata_json,
    created_at: normalizeWailsDateTime(activity.created_at),
  };
}

function toNotification(notification: main.Notification): NotificationItem {
  return {
    id: notification.id,
    notification_type: notification.notification_type,
    title: notification.title,
    message: notification.message,
    status: notification.status,
    source_type: notification.source_type,
    source_id: notification.source_id,
    action_route: notification.action_route,
    action_payload: notification.action_payload,
    read_at: normalizeWailsDateTime(notification.read_at),
    created_at: normalizeWailsDateTime(notification.created_at),
  };
}

function toLoginUserSummary(user: infra.User): LoginUserSummary {
  return {
    id: user.id,
    username: user.username,
    email: user.email,
    full_name: user.full_name || user.display_name,
    department: user.department,
    job_title: user.job_title,
    role_id: user.role_id,
    role_name: user.role_name || user.role?.display_name || user.role?.name,
    is_active: user.is_active,
  };
}

function toRoleSummary(role: infra.Role): RoleSummary {
  return {
    id: role.id,
    name: role.name,
    display_name: role.display_name,
  };
}

function toLicenseKeySummary(key: main.LicenseKey): LicenseKeySummary {
  return {
    id: String(key.id),
    key: key.key,
    role: key.role,
    device_id: key.device_hash || undefined,
    assigned_to: key.display_name || undefined,
    display_name: key.display_name || undefined,
    status: key.activated ? "Activated" : "Available",
  };
}

export async function getCurrentEmployeeContext(): Promise<CurrentEmployeeContext | null> {
  if (!isDesktop()) return null;
  try {
    return await GetCurrentEmployeeContext() as CurrentEmployeeContext;
  } catch {
    return null;
  }
}

export async function listEmployeeProfiles(activeOnly = true): Promise<EmployeeProfile[]> {
  if (!isDesktop()) return [];
  return (await ListEmployeeProfiles(activeOnly)).map(toEmployeeProfile);
}

export async function createEmployeeProfile(employee: Partial<EmployeeProfile>): Promise<EmployeeProfile> {
  return toEmployeeProfile(await CreateEmployeeProfile(buildWailsInput(main.Employee, employee as Record<string, any>)));
}

export async function updateEmployeeProfile(employee: Partial<EmployeeProfile>): Promise<EmployeeProfile> {
  return toEmployeeProfile(await UpdateEmployeeProfile(buildWailsInput(main.Employee, employee as Record<string, any>)));
}

export async function setEmployeeEmploymentState(
  employeeID: string,
  isActive: boolean,
  employmentStatus: string,
): Promise<EmployeeProfile> {
  return toEmployeeProfile(await SetEmployeeEmploymentState(employeeID, isActive, employmentStatus));
}

export async function requestEmployeeArchive(employeeID: string, reason: string): Promise<EmployeeArchiveApproval> {
  return toEmployeeArchiveApproval(await RequestEmployeeArchive(employeeID, reason));
}

export async function reviewEmployeeArchiveRequest(
  requestID: string,
  decision: "approve" | "reject",
  notes = "",
): Promise<EmployeeArchiveApproval> {
  return toEmployeeArchiveApproval(await ReviewEmployeeArchiveRequest(requestID, decision, notes));
}

export async function reassignEmployeeManager(employeeID: string, managerEmployeeID: string): Promise<EmployeeProfile> {
  return toEmployeeProfile(await ReassignEmployeeManager(employeeID, managerEmployeeID));
}

// Wave 9.8 B4: employee compliance documents (visa / CPR / passport / permit).
// The Go DTO (EmployeeDocumentDTO) returns the DECRYPTED doc_number plus a
// masked convenience; the raw ciphertext column is never exposed.
export interface EmployeeComplianceDocument {
  id: string;
  employee_id: string;
  doc_type: string;
  permit_subtype?: string;
  doc_number: string;
  doc_number_masked: string;
  expires_on: string | null;
  notes?: string;
  notified_at?: string | null;
  created_at: string;
  updated_at: string;
}

function toEmployeeComplianceDocument(doc: any): EmployeeComplianceDocument {
  return {
    id: doc.id,
    employee_id: doc.employee_id,
    doc_type: doc.doc_type,
    permit_subtype: doc.permit_subtype ?? "",
    doc_number: doc.doc_number ?? "",
    doc_number_masked: doc.doc_number_masked ?? "",
    expires_on: normalizeWailsDateTime(doc.expires_on) ?? null,
    notes: doc.notes ?? "",
    notified_at: normalizeWailsDateTime(doc.notified_at) ?? null,
    created_at: normalizeWailsDateTime(doc.created_at) ?? "",
    updated_at: normalizeWailsDateTime(doc.updated_at) ?? "",
  };
}

export async function listEmployeeDocuments(employeeID: string): Promise<EmployeeComplianceDocument[]> {
  if (!isDesktop()) return [];
  return (await ListEmployeeDocuments(employeeID)).map(toEmployeeComplianceDocument);
}

export async function createEmployeeDocument(input: {
  employee_id: string;
  doc_type: string;
  permit_subtype?: string;
  expires_on?: string | null;
  notes?: string;
  doc_number: string;
}): Promise<EmployeeComplianceDocument> {
  const model = buildWailsInput(main.EmployeeDocument, {
    employee_id: input.employee_id,
    doc_type: input.doc_type,
    permit_subtype: input.permit_subtype ?? "",
    expires_on: input.expires_on ?? null,
    notes: input.notes ?? "",
  } as Record<string, any>);
  return toEmployeeComplianceDocument(await CreateEmployeeDocument(model, input.doc_number));
}

export async function updateEmployeeDocument(
  documentID: string,
  updates: {
    doc_type?: string;
    permit_subtype?: string;
    expires_on?: string | null;
    notes?: string;
  },
  docNumber = "",
): Promise<EmployeeComplianceDocument> {
  const model = buildWailsInput(main.EmployeeDocument, {
    doc_type: updates.doc_type ?? "",
    permit_subtype: updates.permit_subtype ?? "",
    expires_on: updates.expires_on ?? null,
    notes: updates.notes ?? "",
  } as Record<string, any>);
  return toEmployeeComplianceDocument(await UpdateEmployeeDocument(documentID, model, docNumber));
}

export async function deleteEmployeeDocument(documentID: string): Promise<void> {
  await DeleteEmployeeDocument(documentID);
}

export async function listEmployeeAccessLinks(): Promise<EmployeeAccessLink[]> {
  if (!isDesktop()) return [];
  return (await ListEmployeeAccessLinks()).map(toEmployeeAccessLink);
}

export async function listEmployeeContributionSummaries(): Promise<EmployeeContributionSummary[]> {
  if (!isDesktop()) return [];
  return await ListEmployeeContributionSummaries() as EmployeeContributionSummary[];
}

export async function listEmployeeProjectAssignments(employeeID: string): Promise<ProjectMember[]> {
  if (!isDesktop()) return [];
  return (await ListEmployeeProjectAssignments(employeeID)).map(toProjectMember);
}

export async function createEmployeeAccessLink(link: Partial<EmployeeAccessLink>): Promise<EmployeeAccessLink> {
  return toEmployeeAccessLink(await CreateEmployeeAccessLink(buildWailsInput(main.EmployeeAccessLink, link as Record<string, any>)));
}

export async function listLicenseKeys(): Promise<LicenseKeySummary[]> {
  if (!isDesktop()) return [];
  return (await ListLicenseKeys()).map(toLicenseKeySummary);
}

export async function reassignEmployeeLicenseAccess(
  employeeID: string,
  licenseKey: string,
  syncDisplayName = false,
): Promise<EmployeeAccessLink> {
  return toEmployeeAccessLink(await ReassignEmployeeLicenseAccess(employeeID, licenseKey, syncDisplayName));
}

// Access-tab helpers (B1a): the Employee record is the single home for "who is
// this person and what can they do" — these wrap the existing User/Role bound
// methods (same ones UserManagementScreen uses) so PeopleHub can show/grant
// login access without a second screen. Server-side RBAC (users:view/create/
// update) is unchanged; this is read/compose wiring only.
export async function listLoginUsers(): Promise<LoginUserSummary[]> {
  if (!isDesktop()) return [];
  return (await ListUsers()).map(toLoginUserSummary);
}

export async function listLoginRoles(): Promise<RoleSummary[]> {
  if (!isDesktop()) return [];
  return (await ListRoles()).map(toRoleSummary);
}

export async function createLoginUser(input: {
  username: string;
  email: string;
  password: string;
  full_name: string;
  department?: string;
  job_title?: string;
  role_id: string;
}): Promise<LoginUserSummary> {
  return toLoginUserSummary(
    await CreateUser(
      input.username,
      input.email,
      input.password,
      input.full_name,
      input.department || "",
      input.job_title || "",
      input.role_id,
    ),
  );
}

export async function updateLoginUser(input: {
  id: string;
  full_name: string;
  email: string;
  department?: string;
  job_title?: string;
  role_id: string;
  is_active: boolean;
}): Promise<void> {
  await UpdateUser(
    input.id,
    input.full_name,
    input.email,
    input.department || "",
    input.job_title || "",
    input.role_id,
    input.is_active,
  );
}

export async function listProjects(activeOnly = true): Promise<CollaborativeProject[]> {
  if (!isDesktop()) return [];
  return (await ListCollaborativeProjects(activeOnly)).map(toProject);
}

export async function createProject(project: Partial<CollaborativeProject>): Promise<CollaborativeProject> {
  return toProject(await CreateCollaborativeProject(buildWailsInput(main.Project, project as Record<string, any>)));
}

export async function updateProject(projectID: string, updates: Partial<CollaborativeProject>): Promise<CollaborativeProject> {
  return toProject(await UpdateCollaborativeProject(projectID, updates as Record<string, any>));
}

export async function archiveProject(projectID: string, reason: string): Promise<CollaborativeProject> {
  return toProject(await ArchiveCollaborativeProject(projectID, reason));
}

export async function shelveProject(projectID: string, reason: string): Promise<CollaborativeProject> {
  return toProject(await ShelveCollaborativeProject(projectID, reason));
}

export async function deleteProject(projectID: string, reason: string): Promise<void> {
  await DeleteCollaborativeProject(projectID, reason);
}

export async function addProjectMember(
  projectID: string,
  employeeID: string,
  role: string,
  allocationPercent = 100,
): Promise<ProjectMember> {
  return toProjectMember(await AddCollaborativeProjectMember(projectID, employeeID, role, allocationPercent));
}

export async function listProjectMembers(projectID: string): Promise<ProjectMember[]> {
  if (!isDesktop()) return [];
  return (await ListCollaborativeProjectMembers(projectID)).map(toProjectMember);
}

// Wave 9.8 B3: read-only allocation-capacity precheck. Call before saving a
// member's allocation_percent; the returned other_projects_total already
// excludes excludeProjectID (the membership being edited) — the caller
// compares (other_projects_total + newAllocation) > 100 to decide whether to
// WARN. This never blocks the save itself.
export async function getEmployeeAllocationSummary(
  employeeID: string,
  excludeProjectID = "",
): Promise<AllocationSummary> {
  if (!isDesktop()) return { employee_id: employeeID, other_projects_total: 0, projects: [] };
  const summary = await GetEmployeeAllocationSummary(employeeID, excludeProjectID);
  return {
    employee_id: summary.employee_id,
    other_projects_total: summary.other_projects_total,
    projects: (summary.projects || []).map((p) => ({
      project_id: p.project_id,
      project_name: p.project_name,
      allocation_percent: p.allocation_percent,
    })),
  };
}

// Wave 9.4 B3.3: lightweight per-project task counts for list-row badges —
// avoids loading every task body just to count them.
export async function getProjectTaskCounts(): Promise<Record<string, number>> {
  if (!isDesktop()) return {};
  return (await GetProjectTaskCounts()) || {};
}

export async function listProjectActivity(projectID: string): Promise<TaskActivityItem[]> {
  if (!isDesktop()) return [];
  return (await ListCollaborativeProjectActivity(projectID)).map(toTaskActivity);
}

export async function listMyTasks(includeCompleted = false): Promise<CollaborativeTask[]> {
  if (!isDesktop()) return [];
  return (await ListMyCollaborativeTasks(includeCompleted)).map(toTask);
}

export async function listTeamTasks(includeCompleted = false): Promise<CollaborativeTask[]> {
  if (!isDesktop()) return [];
  return (await ListCollaborativeTeamTasks(includeCompleted)).map(toTask);
}

export async function listProjectTasks(projectID: string, includeCompleted = false): Promise<CollaborativeTask[]> {
  if (!isDesktop()) return [];
  return (await ListCollaborativeProjectTasks(projectID, includeCompleted)).map(toTask);
}

export async function getTask(taskID: string): Promise<CollaborativeTask | null> {
  if (!isDesktop()) return null;
  return toTask(await GetCollaborativeTask(taskID));
}

export async function createTask(task: Partial<CollaborativeTask>): Promise<CollaborativeTask> {
  return toTask(await CreateCollaborativeTask(buildWailsInput(main.TaskItem, task as Record<string, any>)));
}

export async function updateTask(task: Partial<CollaborativeTask>): Promise<CollaborativeTask> {
  return toTask(await UpdateCollaborativeTask(buildWailsInput(main.TaskItem, task as Record<string, any>)));
}

export async function updateTaskStatus(taskID: string, status: string, note = ""): Promise<void> {
  await UpdateCollaborativeTaskStatus(taskID, status, note);
}

export async function reassignTask(taskID: string, assigneeEmployeeID: string): Promise<void> {
  await ReassignCollaborativeTask(taskID, assigneeEmployeeID);
}

export async function updateTaskDueDate(taskID: string, dueDateISO: string): Promise<void> {
  await UpdateCollaborativeTaskDueDate(taskID, dueDateISO);
}

export async function addTaskComment(taskID: string, body: string): Promise<Record<string, any>> {
  return await AddCollaborativeTaskComment(taskID, body);
}

export async function deleteTask(taskID: string): Promise<void> {
  await DeleteCollaborativeTask(taskID);
}

export async function listTaskComments(taskID: string): Promise<TaskCommentItem[]> {
  if (!isDesktop()) return [];
  return (await ListCollaborativeTaskComments(taskID)).map(toTaskComment);
}

export async function listTaskActivity(taskID: string): Promise<TaskActivityItem[]> {
  if (!isDesktop()) return [];
  return (await ListCollaborativeTaskActivity(taskID)).map(toTaskActivity);
}

export async function listNotifications(limit = 50, unreadOnly = false): Promise<NotificationItem[]> {
  if (!isDesktop()) return [];
  return (await ListNotificationFeed(limit, unreadOnly)).map(toNotification);
}

export async function getUnreadNotificationsCount(): Promise<number> {
  if (!isDesktop()) return 0;
  try {
    return await GetUnreadNotificationsCount();
  } catch {
    return 0;
  }
}

export async function markNotificationAsRead(notificationID: string): Promise<void> {
  await MarkNotificationAsRead(notificationID);
}

export async function reviewDeleteApprovalRequest(
  requestID: string,
  decision: "approve" | "reject",
  notes = "",
): Promise<deletion.Request> {
  return await ReviewDeleteApprovalRequest(requestID, decision, notes);
}

export async function refreshCollaborativeWorkspace(
  options: { force?: boolean; minIntervalMs?: number } = {},
): Promise<void> {
  if (!isDesktop()) return;
  const {
    force = false,
    minIntervalMs = COLLABORATIVE_REFRESH_INTERVAL_MS,
  } = options;
  const now = Date.now();

  if (!force && collaborativeRefreshInFlight) {
    return await collaborativeRefreshInFlight;
  }

  if (!force && lastCollaborativeRefreshAt > 0 && now - lastCollaborativeRefreshAt < minIntervalMs) {
    return;
  }

  const refreshPromise = RefreshCollaborativeWorkspace()
    .then(() => {
      lastCollaborativeRefreshAt = Date.now();
    });

  collaborativeRefreshInFlight = withTimeout(refreshPromise, COLLABORATIVE_REFRESH_TIMEOUT_MS, "Collaborative refresh")
    .finally(() => {
      collaborativeRefreshInFlight = null;
    });

  return await collaborativeRefreshInFlight;
}
