import type { CollaborativeProject, CollaborativeTask, EmployeeProfile, ProjectMember } from "$lib/api/collaboration";

export function formatDate(value?: string) {
  if (!value) return "No date";
  return new Date(value).toLocaleString();
}

export function projectNameFor(task: CollaborativeTask, projects: CollaborativeProject[]) {
  return projects.find((project) => project.id === task.project_id)?.name || "None";
}

export function normalizedBoardStatus(task: CollaborativeTask) {
  const status = String(task.status || "open").trim().toLowerCase().replace(/[\s-]+/g, "_");
  if (["completed", "archived", "done"].includes(status)) return "completed";
  if (status === "blocked") return "blocked";
  if (["in_progress", "started", "active_progress"].includes(status)) return "in_progress";
  if (["open", "pending", "new", "active", ""].includes(status)) return "open";
  return "open";
}

export function dueLabel(task: CollaborativeTask) {
  if (!task.due_date) return "No due date";
  const due = new Date(task.due_date);
  if (Number.isNaN(due.getTime())) return "No due date";
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const dueDay = new Date(due);
  dueDay.setHours(0, 0, 0, 0);
  if (dueDay.getTime() < today.getTime() && !["completed", "archived"].includes(task.status || "")) {
    return `Overdue • ${due.toLocaleDateString()}`;
  }
  return `Due ${due.toLocaleDateString()}`;
}

export function isTaskOverdue(task: CollaborativeTask) {
  return Boolean(task.due_date && new Date(task.due_date) < new Date() && !["completed", "archived"].includes(task.status || ""));
}

export function upsertTaskRow(rows: CollaborativeTask[], task: CollaborativeTask) {
  const existingIndex = rows.findIndex((row) => row.id === task.id);
  if (existingIndex >= 0) {
    const next = [...rows];
    next[existingIndex] = task;
    return next;
  }
  return [task, ...rows];
}

export function upsertProjectRow(rows: CollaborativeProject[], project: CollaborativeProject) {
  const existingIndex = rows.findIndex((row) => row.id === project.id);
  if (existingIndex >= 0) {
    const next = [...rows];
    next[existingIndex] = project;
    return next;
  }
  return [project, ...rows];
}

// B3.1: membership drives assignee dropdowns on project-scoped work. When a
// project is chosen, scope to that project's members; with no project
// context, fall back to all employees so cross-context creation still works.
export function assigneeOptionsFor(
  members: ProjectMember[],
  loadedForProjectId: string,
  projectId: string,
  employees: EmployeeProfile[],
) {
  if (projectId && loadedForProjectId === projectId && members.length > 0) {
    return members.map((member) => ({
      id: member.employee_id,
      label: member.employee_name || "Unknown",
      sub: member.role || "",
    }));
  }
  return employees.map((employee) => ({ id: employee.id, label: employee.full_name, sub: employee.department || "" }));
}
