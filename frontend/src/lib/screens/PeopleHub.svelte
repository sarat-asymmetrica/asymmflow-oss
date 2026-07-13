<script lang="ts">
  import { onDestroy, onMount, tick } from "svelte";
  import { EventsOff, EventsOn } from "../../../wailsjs/runtime/runtime";
  import { GenerateLicenseKey } from "../../../wailsjs/go/main/App";
  import { toast } from "../stores/toasts";
  import { brand } from "$lib/brand";
  import { permissions as permissionsStore, currentUser } from "$lib/stores/authContext";
  import PayrollScreen from "./PayrollScreen.svelte";
  import {
    createEmployeeAccessLink,
    createEmployeeDocument,
    createEmployeeProfile,
    createLoginUser,
    deleteEmployeeDocument,
    getCurrentEmployeeContext,
    listEmployeeAccessLinks,
    listEmployeeDocuments,
    listEmployeeContributionSummaries,
    listEmployeeProfiles,
    listEmployeeProjectAssignments,
    listLicenseKeys,
    listLoginRoles,
    listLoginUsers,
    reassignEmployeeLicenseAccess,
    reassignEmployeeManager,
    refreshCollaborativeWorkspace,
    requestEmployeeArchive,
    setEmployeeEmploymentState,
    updateEmployeeDocument,
    updateEmployeeProfile,
    type EmployeeAccessLink,
    type EmployeeComplianceDocument,
    type EmployeeContributionSummary,
    type EmployeeProfile,
    type LicenseKeySummary,
    type LoginUserSummary,
    type ProjectMember,
    type RoleSummary,
  } from "$lib/api/collaboration";

  type PeopleTab = "directory" | "org" | "contributions" | "payroll";
  type EmployeeDetailTab = "profile" | "work" | "access" | "compliance";
  type PayrollCompany = "Acme Instrumentation" | "Beacon Controls";
  const PEOPLE_HUB_CACHE_TTL_MS = 30_000;

  interface Props {
    params?: { tab?: string; payrollEmployeeID?: string };
  }

  let { params = {} }: Props = $props();

  type PeopleHubSnapshot = {
    employees: EmployeeProfile[];
    accessLinks: EmployeeAccessLink[];
    licenseKeys: LicenseKeySummary[];
    contributions: EmployeeContributionSummary[];
    projectAssignments: ProjectMember[];
    loginUsers: LoginUserSummary[];
    loginRoles: RoleSummary[];
    selectedEmployeeID: string;
    selectedLicenseKey: string;
    activeTab: PeopleTab;
    savedAt: number;
  };

  let peopleHubSnapshot: PeopleHubSnapshot | null = null;

  let loading = $state(true);
  let saving = $state(false);
  let activeTab: PeopleTab = $state("directory");
  let employeeDetailTab: EmployeeDetailTab = $state("profile");

  let employees: EmployeeProfile[] = $state([]);
  let accessLinks: EmployeeAccessLink[] = $state([]);
  let licenseKeys: LicenseKeySummary[] = $state([]);
  let contributions: EmployeeContributionSummary[] = $state([]);
  let projectAssignments: ProjectMember[] = $state([]);
  let loginUsers: LoginUserSummary[] = $state([]);
  let loginRoles: RoleSummary[] = $state([]);

  let selectedEmployeeID = $state("");
  let directorySearch = $state("");
  let directoryStatusFilter: "active" | "archive" | "all" = $state("active");
  let currentRole = $state("");
  let archiveReason = $state("");
  let archiveSubmitting = $state(false);

  let createFullName = $state("");
  let createDepartment = $state("");
  let createJobTitle = $state("");
  let createEmail = $state("");
  let createEmailError = $state("");
  let createPhone = $state("");
  let createStartDate = $state("");
  let createManagerEmployeeID = $state("");
  let selectedLicenseKey = $state("");

  let fullName = $state("");
  let preferredName = $state("");
  let email = $state("");
  let phone = $state("");
  let department = $state("");
  let jobTitle = $state("");
  let managerEmployeeID = $state("");
  let employmentStatus = $state("active");
  let startDate = $state("");
  let emergencyContact = $state("");
  let notes = $state("");

  // B1a Access tab: bind an existing login user, or create a new one, for the
  // selected employee's primary license link.
  let selectedBindUserID = $state("");
  let reassignLicenseKey = $state("");
  let showNewLoginForm = $state(false);
  let newLoginUsername = $state("");
  let newLoginPassword = $state("");
  let newLoginRoleID = $state("");
  let bindingUser = $state(false);

  // C2: issue a brand-new license key from the Access tab (GenerateLicenseKey
  // is already wailsjs-bound and server-gated on licenses:manage — this just
  // gives it a caller so an admin can issue + bind without leaving the app).
  const ISSUABLE_LICENSE_ROLES = ["admin", "manager", "sales", "operations", "staff"];
  let issueLicenseRole = $state("staff");
  let issueLicenseNotes = $state("");
  let issuingLicense = $state(false);

  // B2: Payroll lives in People too — gated on payroll:view, with a preselect
  // deep-link from the employee record ("Set up payroll").
  let payrollCompany: PayrollCompany = $state(brand.defaultDivision as PayrollCompany);
  let payrollPresetEmployeeID = $state("");
  let lastAppliedPeopleRouteKey = $state("");

  let detailPanelEl: HTMLElement | null = $state(null);

  let assignmentLoadToken = 0;
  let loadRequestSeq = 0;

  // Wave 9.8 B4: employee compliance documents (visa / CPR / passport / permit).
  let complianceDocuments: EmployeeComplianceDocument[] = $state([]);
  let complianceLoadToken = 0;
  let complianceSaving = $state(false);
  let editingDocumentID = $state("");
  let docType = $state("cpr");
  let docPermitSubtype = $state("");
  let docNumber = $state("");
  let docExpiresOn = $state("");
  let docNotes = $state("");

  const DOC_TYPE_LABELS: Record<string, string> = {
    cpr: "CPR",
    passport: "Passport",
    visa: "Visa",
    permit: "Permit",
  };

  function daysUntil(value: string | null): number | null {
    if (!value) return null;
    const target = new Date(value);
    if (Number.isNaN(target.getTime())) return null;
    return Math.ceil((target.getTime() - Date.now()) / (1000 * 60 * 60 * 24));
  }

  function resetDocumentForm() {
    editingDocumentID = "";
    docType = "cpr";
    docPermitSubtype = "";
    docNumber = "";
    docExpiresOn = "";
    docNotes = "";
  }

  function editDocument(doc: EmployeeComplianceDocument) {
    editingDocumentID = doc.id;
    docType = doc.doc_type;
    docPermitSubtype = doc.permit_subtype || "";
    docNumber = doc.doc_number || "";
    docExpiresOn = toDateInput(doc.expires_on || "");
    docNotes = doc.notes || "";
  }

  async function loadComplianceDocuments(employeeID = selectedEmployeeID) {
    if (!employeeID) {
      complianceDocuments = [];
      return;
    }
    const token = ++complianceLoadToken;
    try {
      const docs = await listEmployeeDocuments(employeeID);
      if (token === complianceLoadToken) complianceDocuments = docs;
    } catch (err) {
      if (token === complianceLoadToken) complianceDocuments = [];
      toast.warning(`Could not load compliance documents: ${String(err)}`);
    }
  }

  async function handleSaveDocument() {
    if (!selectedEmployeeID) return;
    if (!docNumber.trim()) {
      toast.warning("Document number is required");
      return;
    }
    complianceSaving = true;
    try {
      if (editingDocumentID) {
        await updateEmployeeDocument(
          editingDocumentID,
          {
            doc_type: docType,
            permit_subtype: docType === "permit" ? docPermitSubtype.trim() : "",
            expires_on: toISODate(docExpiresOn) ?? null,
            notes: docNotes.trim(),
          },
          docNumber.trim(),
        );
        toast.success("Document updated");
      } else {
        await createEmployeeDocument({
          employee_id: selectedEmployeeID,
          doc_type: docType,
          permit_subtype: docType === "permit" ? docPermitSubtype.trim() : "",
          expires_on: toISODate(docExpiresOn) ?? null,
          notes: docNotes.trim(),
          doc_number: docNumber.trim(),
        });
        toast.success("Document added");
      }
      resetDocumentForm();
      await loadComplianceDocuments();
    } catch (err) {
      toast.danger(`Could not save document: ${String(err)}`);
    } finally {
      complianceSaving = false;
    }
  }

  async function handleDeleteDocument(documentID: string) {
    try {
      await deleteEmployeeDocument(documentID);
      if (editingDocumentID === documentID) resetDocumentForm();
      toast.success("Document removed");
      await loadComplianceDocuments();
    } catch (err) {
      toast.danger(`Could not remove document: ${String(err)}`);
    }
  }

  function toDateInput(value?: string) {
    if (!value) return "";
    return value.slice(0, 10);
  }

  function formatShortDate(value?: string) {
    if (!value) return "";
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return "";
    return date.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
  }

  function isArchivedEmployee(employee: EmployeeProfile) {
    return employee.is_active === false || (employee.employment_status || "").toLowerCase() === "archived";
  }

  function employeeStatusLabel(employee: EmployeeProfile) {
    if ((employee.employment_status || "").toLowerCase() === "archived") return "Archived";
    if (employee.is_active === false) return "Inactive";
    return employee.employment_status || "Active";
  }

  function toISODate(value: string) {
    const trimmed = value.trim();
    return trimmed ? `${trimmed}T00:00:00Z` : undefined;
  }

  function isValidEmail(value: string) {
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);
  }

  function applyEmployeeToForm(employee: EmployeeProfile | null) {
    fullName = employee?.full_name || "";
    preferredName = employee?.preferred_name || "";
    email = employee?.email || "";
    phone = employee?.phone || "";
    department = employee?.department || "";
    jobTitle = employee?.job_title || "";
    managerEmployeeID = employee?.manager_employee_id || "";
    // Vocabulary is work-state only (active/on_leave/probation/contract) —
    // "inactive" is no longer a selectable status; deactivation is Archive
    // only (B1e). Legacy records already flipped to archived carry
    // employment_status "archived", handled separately by isArchivedEmployee.
    employmentStatus = employee?.employment_status || "active";
    startDate = toDateInput(employee?.start_date);
    emergencyContact = employee?.emergency_contact || "";
    notes = employee?.notes || "";
  }

  function hasLoadedPeopleData() {
    return Boolean(employees.length || accessLinks.length || licenseKeys.length || contributions.length);
  }

  function restoreSnapshot() {
    if (!peopleHubSnapshot) return false;
    employees = peopleHubSnapshot.employees;
    accessLinks = peopleHubSnapshot.accessLinks;
    licenseKeys = peopleHubSnapshot.licenseKeys;
    contributions = peopleHubSnapshot.contributions;
    projectAssignments = peopleHubSnapshot.projectAssignments;
    loginUsers = peopleHubSnapshot.loginUsers;
    loginRoles = peopleHubSnapshot.loginRoles;
    selectedEmployeeID = peopleHubSnapshot.selectedEmployeeID;
    selectedLicenseKey = peopleHubSnapshot.selectedLicenseKey;
    activeTab = peopleHubSnapshot.activeTab;
    applyEmployeeToForm(employees.find((employee) => employee.id === selectedEmployeeID) || null);
    loading = false;
    return true;
  }

  function saveSnapshot() {
    peopleHubSnapshot = {
      employees: [...employees],
      accessLinks: [...accessLinks],
      licenseKeys: [...licenseKeys],
      contributions: [...contributions],
      projectAssignments: [...projectAssignments],
      loginUsers: [...loginUsers],
      loginRoles: [...loginRoles],
      selectedEmployeeID,
      selectedLicenseKey,
      activeTab,
      savedAt: Date.now(),
    };
  }

  function isSnapshotStale() {
    return !peopleHubSnapshot || Date.now() - peopleHubSnapshot.savedAt > PEOPLE_HUB_CACHE_TTL_MS;
  }

  async function load(options: { refreshRemote?: boolean; silent?: boolean } = {}) {
    const { refreshRemote = false, silent = false } = options;
    const requestSeq = ++loadRequestSeq;
    const shouldShowLoading = !silent || !hasLoadedPeopleData();
    if (shouldShowLoading) {
      loading = true;
    }
    try {
      if (refreshRemote) {
        await refreshCollaborativeWorkspace().catch(() => undefined);
      }
      const [employeeRows, linkRows, licenseRows, contributionRows, userRows, roleRows] = await Promise.all([
        listEmployeeProfiles(false),
        listEmployeeAccessLinks(),
        listLicenseKeys(),
        listEmployeeContributionSummaries(),
        // Access tab (B1a): loading these here (rather than only from
        // UserManagementScreen) is what makes the employee record the single
        // home for "who is this person / what can they do". Non-admins are
        // denied server-side (users:view) — swallow that quietly, the Access
        // composer is admin-gated in the template anyway.
        listLoginUsers().catch(() => []),
        listLoginRoles().catch(() => []),
      ]);

      if (requestSeq !== loadRequestSeq) {
        return;
      }

      employees = employeeRows;
      accessLinks = linkRows;
      licenseKeys = licenseRows;
      contributions = contributionRows;
      loginUsers = userRows;
      loginRoles = roleRows;
      if (!newLoginRoleID && loginRoles.length > 0) {
        newLoginRoleID = loginRoles[0].id;
      }

      if (!selectedEmployeeID && employees.length > 0) {
        selectedEmployeeID = employees[0].id;
      } else if (selectedEmployeeID && !employees.some((employee) => employee.id === selectedEmployeeID)) {
        selectedEmployeeID = employees[0]?.id || "";
      }

      if (!selectedLicenseKey && availableLicenseKeys.length > 0) {
        selectedLicenseKey = availableLicenseKeys[0].key;
      }

      await loadSelectedEmployeeData(selectedEmployeeID);
      saveSnapshot();
    } catch (err) {
      if (!silent || !hasLoadedPeopleData()) {
        toast.danger(`Failed to load people hub: ${String(err)}`);
      }
    } finally {
      if (requestSeq === loadRequestSeq && shouldShowLoading) {
        loading = false;
      }
    }
  }

  async function hydratePeopleHub() {
    const restored = restoreSnapshot();
    if (!restored) {
      await load();
      await load({ refreshRemote: true, silent: true });
      return;
    }
    if (hasLoadedPeopleData()) {
      await load({ refreshRemote: isSnapshotStale(), silent: true });
      return;
    }
    await load({ refreshRemote: true, silent: true });
  }

  async function loadSelectedEmployeeData(employeeID = selectedEmployeeID) {
    const selected = employees.find((employee) => employee.id === employeeID) || null;
    applyEmployeeToForm(selected);

    if (!selected?.id) {
      projectAssignments = [];
      complianceDocuments = [];
      resetDocumentForm();
      return;
    }

    resetDocumentForm();
    void loadComplianceDocuments(selected.id);

    const token = ++assignmentLoadToken;
    try {
      const assignments = await listEmployeeProjectAssignments(selected.id);
      if (token === assignmentLoadToken) {
        projectAssignments = assignments;
      }
    } catch (err) {
      if (token === assignmentLoadToken) {
        projectAssignments = [];
      }
      toast.warning(`Could not load project assignments: ${String(err)}`);
    }
  }

  function validateCreateEmail() {
    const trimmed = createEmail.trim();
    createEmailError = trimmed && !isValidEmail(trimmed) ? "Enter a valid email address" : "";
    return !createEmailError;
  }

  async function handleCreate() {
    if (!createFullName.trim()) {
      toast.warning("Employee name is required");
      return;
    }
    if (!validateCreateEmail()) {
      toast.warning(createEmailError);
      return;
    }

    try {
      const employee = await createEmployeeProfile({
        full_name: createFullName.trim(),
        department: createDepartment.trim(),
        job_title: createJobTitle.trim(),
        email: createEmail.trim(),
        phone: createPhone.trim(),
        start_date: toISODate(createStartDate),
        manager_employee_id: createManagerEmployeeID || undefined,
      });
      createFullName = "";
      createDepartment = "";
      createJobTitle = "";
      createEmail = "";
      createEmailError = "";
      createPhone = "";
      createStartDate = "";
      createManagerEmployeeID = "";
      selectedEmployeeID = employee.id;
      // B1c: onboarding is one continuous flow — land the operator on the new
      // profile (Directory / Profile tab) instead of stranding them below the
      // fold on the composer they just submitted.
      activeTab = "directory";
      employeeDetailTab = "profile";
      await load();
      toast.success("Employee profile created — add contact & start details below.");
      await tick();
      detailPanelEl?.scrollIntoView({ behavior: "smooth", block: "start" });
    } catch (err) {
      toast.danger(`Failed to create employee profile: ${String(err)}`);
    }
  }

  async function handleSaveProfile() {
    if (!selectedEmployee?.id) {
      toast.warning("Choose an employee first");
      return;
    }
    if (!fullName.trim()) {
      toast.warning("Employee name is required");
      return;
    }

    saving = true;
    try {
      await updateEmployeeProfile({
        id: selectedEmployee.id,
        employee_code: selectedEmployee.employee_code,
        full_name: fullName.trim(),
        preferred_name: preferredName.trim(),
        email: email.trim(),
        phone: phone.trim(),
        department: department.trim(),
        job_title: jobTitle.trim(),
        employment_status: employmentStatus,
        start_date: toISODate(startDate),
        emergency_contact: emergencyContact.trim(),
        notes: notes.trim(),
        is_active: selectedEmployee.is_active !== false,
      });

      if ((selectedEmployee.manager_employee_id || "") !== managerEmployeeID) {
        await reassignEmployeeManager(selectedEmployee.id, managerEmployeeID);
      }

      await load();
      toast.success("Employee profile updated");
    } catch (err) {
      toast.danger(`Failed to update employee profile: ${String(err)}`);
    } finally {
      saving = false;
    }
  }

  async function handleEmploymentToggle() {
    if (!selectedEmployee?.id) {
      toast.warning("Choose an employee first");
      return;
    }
    if (!isAdmin) {
      toast.warning("Only admin can reactivate employee profiles");
      return;
    }
    if (selectedEmployee.is_active !== false) {
      toast.warning("Use employee archive for removals");
      return;
    }

    try {
      await setEmployeeEmploymentState(selectedEmployee.id, true, "active");
      employmentStatus = "active";
      await load();
      toast.success("Employee reactivated");
      toast.info(
        "Login access and project memberships were revoked on archive and must be re-granted manually (Access tab / project rosters).",
        0
      );
    } catch (err) {
      toast.danger(`Failed to update employee status: ${String(err)}`);
    }
  }

  async function handleArchiveRequest() {
    if (!selectedEmployee?.id) {
      toast.warning("Choose an employee first");
      return;
    }
    if (!isAdmin) {
      toast.warning("Only admin can archive employees");
      return;
    }
    if (selectedEmployee.is_active === false) {
      toast.warning("This employee is already archived or inactive");
      return;
    }
    if (!archiveReason.trim()) {
      toast.warning("Archive reason is required");
      return;
    }

    archiveSubmitting = true;
    try {
      await requestEmployeeArchive(selectedEmployee.id, archiveReason.trim());
      toast.success("Archive submitted for approval — an admin must approve it in the Approvals queue");
      archiveReason = "";
      await load({ refreshRemote: true });
    } catch (err) {
      toast.danger(`Failed to submit archive request: ${String(err)}`);
    } finally {
      archiveSubmitting = false;
    }
  }

  async function handleLinkAccess() {
    if (!selectedEmployeeID) {
      toast.warning("Choose an employee first");
      return;
    }
    if (!selectedLicenseKey) {
      toast.warning("Choose a license key to link");
      return;
    }

    try {
      await createEmployeeAccessLink({
        employee_id: selectedEmployeeID,
        license_key: selectedLicenseKey,
        access_status: "active",
      });
      await load();
      selectedLicenseKey = availableLicenseKeys[0]?.key || "";
      toast.success("Employee linked to license");
    } catch (err) {
      toast.danger(`Failed to link employee access: ${String(err)}`);
    }
  }

  async function handleReassignLicense(licenseKey: string) {
    if (!selectedEmployeeID || !licenseKey) return;
    try {
      await reassignEmployeeLicenseAccess(selectedEmployeeID, licenseKey, true);
      reassignLicenseKey = "";
      await load();
      toast.success("License reassigned to this employee");
    } catch (err) {
      toast.danger(`Failed to reassign license: ${String(err)}`);
    }
  }

  // B1a: bind an existing login User to the employee's primary license link.
  // Reuses createEmployeeAccessLink — passing the same employee_id + existing
  // license_key updates that link's user_id in place instead of creating a
  // duplicate (see CreateEmployeeAccessLink's existing-link branch).
  async function handleBindUser() {
    if (!selectedEmployee?.id) {
      toast.warning("Choose an employee first");
      return;
    }
    const primaryLink = selectedAccessLinks.find((link) => link.is_primary) || selectedAccessLinks[0];
    if (!primaryLink) {
      toast.warning("Link a license before binding a login user");
      return;
    }
    if (!selectedBindUserID) {
      toast.warning("Choose a user to bind");
      return;
    }

    bindingUser = true;
    try {
      await createEmployeeAccessLink({
        employee_id: selectedEmployee.id,
        license_key: primaryLink.license_key,
        user_id: selectedBindUserID,
        access_status: primaryLink.access_status || "active",
      });
      await load();
      toast.success("Login user linked to employee");
      selectedBindUserID = "";
    } catch (err) {
      toast.danger(`Failed to bind login user: ${String(err)}`);
    } finally {
      bindingUser = false;
    }
  }

  // B1a: "if binding a login User to an employee needs a create/update-user
  // affordance, wire it via the existing CreateUser/UpdateUser" — creates the
  // account (permission-gated users:create server-side) then binds it the
  // same way handleBindUser does.
  async function handleCreateLoginUser() {
    if (!selectedEmployee?.id) {
      toast.warning("Choose an employee first");
      return;
    }
    if (!newLoginUsername.trim() || !newLoginPassword.trim() || !newLoginRoleID) {
      toast.warning("Username, password, and role are required");
      return;
    }

    bindingUser = true;
    try {
      const user = await createLoginUser({
        username: newLoginUsername.trim(),
        email: selectedEmployee.email || "",
        password: newLoginPassword,
        full_name: selectedEmployee.full_name,
        department: selectedEmployee.department || "",
        job_title: selectedEmployee.job_title || "",
        role_id: newLoginRoleID,
      });
      newLoginUsername = "";
      newLoginPassword = "";
      showNewLoginForm = false;

      const primaryLink = selectedAccessLinks.find((link) => link.is_primary) || selectedAccessLinks[0];
      if (primaryLink) {
        await createEmployeeAccessLink({
          employee_id: selectedEmployee.id,
          license_key: primaryLink.license_key,
          user_id: user.id,
          access_status: primaryLink.access_status || "active",
        });
        toast.success("Login user created and linked");
      } else {
        toast.success("Login user created — link a license above to finish granting access");
      }
      await load();
    } catch (err) {
      toast.danger(`Failed to create login user: ${String(err)}`);
    } finally {
      bindingUser = false;
    }
  }

  // C2: issue a new license key (server-gated on licenses:manage — same gate
  // as Reassign above, which is why this reuses the isAdmin UI gate too),
  // then hand it straight to the existing "Link a License" flow above by
  // preselecting it once the list refreshes.
  async function handleIssueLicense() {
    issuingLicense = true;
    try {
      const createdBy = $currentUser?.username || $currentUser?.full_name || "admin";
      const newKey = await GenerateLicenseKey(issueLicenseRole, issueLicenseNotes.trim(), createdBy);
      issueLicenseNotes = "";
      await load();
      selectedLicenseKey = newKey;
      toast.success(`License issued: ${newKey} — select it below to link.`);
    } catch (err) {
      toast.danger(`Failed to issue license: ${String(err)}`);
    } finally {
      issuingLicense = false;
    }
  }

  function handleSetupPayroll() {
    if (!selectedEmployee?.id) {
      toast.warning("Choose an employee first");
      return;
    }
    // Pattern #1 handoff: navigate into the People "Payroll" tab, preselecting
    // this employee's compensation profile. Goes through the same
    // navigateToScreen event every other cross-screen deep link uses.
    window.dispatchEvent(new CustomEvent("navigateToScreen", {
      detail: { screen: "people", tab: "payroll", payrollEmployeeID: selectedEmployee.id },
    }));
  }

  function openUsersAndAccess() {
    window.dispatchEvent(new CustomEvent("navigateToScreen", { detail: { screen: "usermanagement" } }));
  }

  function selectEmployee(employeeID: string) {
    selectedEmployeeID = employeeID;
    employeeDetailTab = "profile";
    loadSelectedEmployeeData(employeeID);
  }

  let linkedLicenseKeys = $derived(new Set(accessLinks.map((link) => link.license_key)));
  let availableLicenseKeys = $derived(licenseKeys.filter((license) => !linkedLicenseKeys.has(license.key)));
  let selectedEmployee = $derived(employees.find((employee) => employee.id === selectedEmployeeID) || null);
  let isAdmin = $derived(["admin", "administrator", "developer"].includes(currentRole.toLowerCase()));
  let activeEmployeeCount = $derived(employees.filter((employee) => !isArchivedEmployee(employee)).length);
  let archivedEmployeeCount = $derived(employees.filter(isArchivedEmployee).length);
  let filteredEmployees = $derived(employees.filter((employee) => {
    if (directoryStatusFilter === "active" && isArchivedEmployee(employee)) return false;
    if (directoryStatusFilter === "archive" && !isArchivedEmployee(employee)) return false;
    const search = directorySearch.trim().toLowerCase();
    if (!search) return true;
    const haystack = [
      employee.full_name,
      employee.employee_code,
      employee.department,
      employee.job_title,
      employee.manager_name,
    ]
      .filter(Boolean)
      .join(" ")
      .toLowerCase();
    return haystack.includes(search);
  }));
  let selectedAccessLinks = $derived(accessLinks.filter((link) => link.employee_id === selectedEmployeeID));
  let reassignableLicenseKeys = $derived(licenseKeys.filter((license) => linkedLicenseKeys.has(license.key) && !selectedAccessLinks.some((link) => link.license_key === license.key)));
  let sortedContributions = $derived([...contributions].sort((left, right) => {
    if ((right.completed_task_count || 0) !== (left.completed_task_count || 0)) {
      return (right.completed_task_count || 0) - (left.completed_task_count || 0);
    }
    return (right.active_task_count || 0) - (left.active_task_count || 0);
  }));
  let orgGroups = $derived(employees.reduce((groups, employee) => {
    const managerLabel = employee.manager_name || "Leadership / Unassigned";
    if (!groups[managerLabel]) {
      groups[managerLabel] = [];
    }
    groups[managerLabel].push(employee);
    return groups;
  }, {} as Record<string, EmployeeProfile[]>));
  let orgGroupEntries = $derived(Object.entries(orgGroups).sort((left, right) => left[0].localeCompare(right[0])));

  // Client-side visibility only — every write this gates (payroll runs,
  // users:*) is enforced server-side regardless of what this hides/shows.
  let permissionList = $derived(Array.isArray($permissionsStore) ? $permissionsStore : []);
  function hasPermission(permission: string) {
    if (permissionList.includes("*")) return true;
    if (permissionList.includes(permission)) return true;
    const [resource] = permission.split(":");
    return permissionList.includes(`${resource}:*`);
  }
  let hasPayrollAccess = $derived(hasPermission("payroll:view"));

  function userByID(userID?: string) {
    if (!userID) return null;
    return loginUsers.find((user) => user.id === userID) || null;
  }
  function roleByID(roleID?: string) {
    if (!roleID) return null;
    return loginRoles.find((role) => role.id === roleID) || null;
  }
  function accessLinkLoginLabel(link: EmployeeAccessLink) {
    const user = userByID(link.user_id);
    if (!user) return null;
    const role = roleByID(user.role_id);
    return { user, roleLabel: user.role_name || role?.display_name || role?.name || "No role" };
  }

  // B2 deep-link target: "Set up payroll" on the employee record navigates
  // here via navigateToScreen({screen:"people", tab:"payroll",
  // payrollEmployeeID}) — mirrors FinanceHub's params.tab pattern.
  $effect(() => {
    const nextKey = JSON.stringify({ tab: params?.tab || "", employeeID: params?.payrollEmployeeID || "", payrollOk: hasPayrollAccess });
    if (nextKey !== lastAppliedPeopleRouteKey) {
      lastAppliedPeopleRouteKey = nextKey;
      if (params?.tab === "payroll" && hasPayrollAccess) {
        activeTab = "payroll";
      }
      if (params?.payrollEmployeeID) {
        payrollPresetEmployeeID = params.payrollEmployeeID;
      }
    }
  });

  onMount(() => {
    getCurrentEmployeeContext().then((ctx) => {
      currentRole = ctx?.license_role || "";
    }).catch(() => undefined);
    void hydratePeopleHub();
    EventsOn("employees:updated", () => void load({ silent: true }));
    EventsOn("tasks:updated", () => void load({ silent: true }));
    EventsOn("projects:updated", () => void load({ silent: true }));
  });

  onDestroy(() => {
    EventsOff("employees:updated");
    EventsOff("tasks:updated");
    EventsOff("projects:updated");
  });
</script>

<div class="page">
  <header class="header">
    <div>
      <h1>People.</h1>
      <p class="subtitle">Employee directory, reporting structure, access mapping, and contribution insight.</p>
    </div>
    <div class="header-actions">
      {#if isAdmin}
        <button class="ghost" onclick={openUsersAndAccess}>Users & Access →</button>
      {/if}
      <div class="tabs">
        <button class:active={activeTab === "directory"} onclick={() => (activeTab = "directory")}>Directory</button>
        <button class:active={activeTab === "org"} onclick={() => (activeTab = "org")}>Org</button>
        <button class:active={activeTab === "contributions"} onclick={() => (activeTab = "contributions")}>Contributions</button>
        {#if hasPayrollAccess}
          <button class:active={activeTab === "payroll"} onclick={() => (activeTab = "payroll")}>Payroll</button>
        {/if}
      </div>
    </div>
  </header>

  <section class="composer">
    <div class="panel-head">
      <h2>Add Employee</h2>
      <span>{employees.length} total</span>
    </div>
    <div class="grid three">
      <input bind:value={createFullName} placeholder="Full name" />
      <input bind:value={createDepartment} placeholder="Department" />
      <input bind:value={createJobTitle} placeholder="Job title" />
    </div>
    <div class="grid three">
      <label class="inline-field">
        <input bind:value={createEmail} type="email" placeholder="Email" onblur={validateCreateEmail} />
        {#if createEmailError}<small class="error-text">{createEmailError}</small>{/if}
      </label>
      <input bind:value={createPhone} placeholder="Phone" />
      <input bind:value={createStartDate} type="date" placeholder="Start date" />
    </div>
    <div class="grid three">
      <select bind:value={createManagerEmployeeID}>
        <option value="">No manager</option>
        {#each employees as employee}
          <option value={employee.id}>{employee.full_name}</option>
        {/each}
      </select>
    </div>
    <div class="actions">
      <button onclick={handleCreate}>Create Employee</button>
    </div>
  </section>

  {#if activeTab === "directory"}
    <section class="content-grid directory-layout">
      <article class="panel">
        <div class="panel-head">
          <h2>Employee Directory</h2>
          <span>{activeEmployeeCount} active · {archivedEmployeeCount} archived</span>
        </div>
        <div class="filter-pills">
          <button class:active={directoryStatusFilter === "active"} onclick={() => (directoryStatusFilter = "active")}>Active</button>
          <button class:active={directoryStatusFilter === "archive"} onclick={() => (directoryStatusFilter = "archive")}>Archive</button>
          <button class:active={directoryStatusFilter === "all"} onclick={() => (directoryStatusFilter = "all")}>All</button>
        </div>
        <input bind:value={directorySearch} placeholder="Search by name, code, department, or title" />
        {#if loading}
          <div class="empty">Loading employees...</div>
        {:else if filteredEmployees.length === 0}
          <div class="empty">No employee profiles match this search.</div>
        {:else}
          <div class="list">
            {#each filteredEmployees as employee}
              <button
                class:selected={employee.id === selectedEmployeeID}
                class:archived={isArchivedEmployee(employee)}
                class="list-row selectable"
                onclick={() => selectEmployee(employee.id)}
              >
                <div>
                  <strong>{employee.full_name}</strong>
                  <div class="meta">{employee.employee_code} • {employee.department || "No department"}</div>
                </div>
                <div class="meta right">
                  <div>{employee.job_title || "No title"}</div>
                  <div>{isArchivedEmployee(employee) ? employeeStatusLabel(employee) : employee.manager_name || employee.employment_status || "Active"}</div>
                </div>
              </button>
            {/each}
          </div>
        {/if}
      </article>

      <article class="panel detail-panel" bind:this={detailPanelEl}>
        <div class="panel-head">
          <h2>Employee Detail</h2>
          {#if selectedEmployee}
            <span class:inactive={isArchivedEmployee(selectedEmployee)}>{employeeStatusLabel(selectedEmployee)}</span>
          {/if}
        </div>

        {#if !selectedEmployee}
          <div class="empty">Select an employee to manage their profile.</div>
        {:else}
          <!-- B1d: job-shaped profile — Profile / Work / Access. Sales metrics
               (if any exist for this person) live in the Contributions tab
               above, never in this HR editor. -->
          <div class="detail-subtabs">
            <button class:active={employeeDetailTab === "profile"} onclick={() => (employeeDetailTab = "profile")}>Profile</button>
            <button class:active={employeeDetailTab === "work"} onclick={() => (employeeDetailTab = "work")}>Work</button>
            <button class:active={employeeDetailTab === "access"} onclick={() => (employeeDetailTab = "access")}>Access</button>
            <button class:active={employeeDetailTab === "compliance"} onclick={() => (employeeDetailTab = "compliance")}>Compliance</button>
          </div>

          {#if employeeDetailTab === "profile"}
            <div class="grid two">
              <label>
                <span>Full Name</span>
                <input bind:value={fullName} placeholder="Full name" />
              </label>
              <label>
                <span>Preferred Name</span>
                <input bind:value={preferredName} placeholder="Preferred name" />
              </label>
              <label>
                <span>Email</span>
                <input bind:value={email} type="email" placeholder="Email" />
              </label>
              <label>
                <span>Phone</span>
                <input bind:value={phone} placeholder="Phone" />
              </label>
              <label>
                <span>Emergency Contact</span>
                <input bind:value={emergencyContact} placeholder="Emergency contact" />
              </label>
            </div>
            <label class="stack">
              <span>Notes</span>
              <textarea bind:value={notes} placeholder="Role notes, responsibilities, or context"></textarea>
            </label>
          {/if}

          {#if employeeDetailTab === "work"}
            <div class="grid two">
              <label>
                <span>Department</span>
                <input bind:value={department} placeholder="Department" />
              </label>
              <label>
                <span>Job Title</span>
                <input bind:value={jobTitle} placeholder="Job title" />
              </label>
              <label>
                <span>Manager</span>
                <select bind:value={managerEmployeeID}>
                  <option value="">No manager</option>
                  {#each employees.filter((candidate) => candidate.id !== selectedEmployee.id) as employee}
                    <option value={employee.id}>{employee.full_name}</option>
                  {/each}
                </select>
              </label>
              <label>
                <span>Status</span>
                <select bind:value={employmentStatus}>
                  <option value="active">Active</option>
                  <option value="on_leave">On Leave</option>
                  <option value="probation">Probation</option>
                  <option value="contract">Contract</option>
                </select>
              </label>
              <label>
                <span>Start Date</span>
                <input bind:value={startDate} type="date" />
              </label>
            </div>

            <div class="stat-grid">
              <div class="stat-card">
                <span>Projects</span>
                <strong>{projectAssignments.filter((assignment) => assignment.is_active !== false).length}</strong>
              </div>
              <div class="stat-card">
                <span>Licenses</span>
                <strong>{selectedAccessLinks.length}</strong>
              </div>
              <div class="stat-card">
                <span>Manager</span>
                <strong>{selectedEmployee.manager_name || "None"}</strong>
              </div>
            </div>

            {#if hasPayrollAccess}
              <div class="actions">
                <button class="secondary" onclick={handleSetupPayroll}>Set up payroll →</button>
              </div>
            {/if}

            {#if isAdmin && selectedEmployee.is_active === false}
              <div class="actions">
                <button class="secondary" onclick={handleEmploymentToggle}>Reactivate</button>
              </div>
            {/if}

            {#if isAdmin && selectedEmployee.is_active !== false}
              <div class="archive-panel">
                <div>
                  <strong>Employee Archive</strong>
                  <p>Any admin can archive this employee. Linked tasks, offers, orders, invoices, and audit history stay attached to the archived profile. Archive is the only way to deactivate — the Status field above only tracks work state.</p>
                </div>
                <textarea bind:value={archiveReason} placeholder="Reason for archive"></textarea>
                <div class="actions">
                  <button class="danger" disabled={archiveSubmitting} onclick={handleArchiveRequest}>
                    {archiveSubmitting ? "Archiving..." : "Archive Employee"}
                  </button>
                </div>
              </div>
            {:else if isArchivedEmployee(selectedEmployee)}
              <div class="archive-panel archived-panel">
                <div>
                  <strong>Archived Employee</strong>
                  <p>
                    {selectedEmployee.archive_reason || "Employee is inactive. Historical work remains linked to this profile."}
                    {selectedEmployee.archived_at ? ` Archived ${formatShortDate(selectedEmployee.archived_at)}.` : ""}
                  </p>
                </div>
              </div>
            {/if}

            <div class="subsection">
              <div class="panel-head compact">
                <h3>Project Assignments</h3>
                <span>{projectAssignments.length} records</span>
              </div>
              {#if projectAssignments.length === 0}
                <div class="empty small">No project assignments yet.</div>
              {:else}
                <div class="list compact-list">
                  {#each projectAssignments as assignment}
                    <div class="list-row">
                      <div>
                        <strong>{assignment.project_name || "Untitled project"}</strong>
                        <div class="meta">{assignment.role || "Member"}</div>
                      </div>
                      <div class="meta right">
                        <div>{assignment.is_active === false ? "Inactive" : "Active"}</div>
                        <div>{assignment.joined_at ? toDateInput(assignment.joined_at) : "No join date"}</div>
                      </div>
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
          {/if}

          {#if employeeDetailTab === "access"}
            <!-- B1a: the single home for "who is this person and what can
                 they do" — license link(s), the login user bound to it, and
                 that user's role, plus grant/change controls right here. -->
            <div class="subsection">
              <div class="panel-head compact">
                <h3>Access & Login</h3>
                <span>{selectedAccessLinks.length} license link(s)</span>
              </div>
              {#if selectedAccessLinks.length === 0}
                <div class="empty small">No license linked yet. Link one below to grant app access.</div>
              {:else}
                <div class="list compact-list">
                  {#each selectedAccessLinks as link}
                    {@const loginInfo = accessLinkLoginLabel(link)}
                    <div class="list-row access-row">
                      <div>
                        <strong>{link.license_key}</strong>
                        <div class="meta">{link.device_name || "No device mapped"} · {link.access_status || "active"}{link.is_primary ? " · Primary" : ""}</div>
                      </div>
                      <div class="meta right">
                        {#if loginInfo}
                          <div><strong>{loginInfo.user.full_name || loginInfo.user.username}</strong></div>
                          <div>@{loginInfo.user.username} · {loginInfo.roleLabel}</div>
                        {:else}
                          <div>No login user bound</div>
                        {/if}
                      </div>
                    </div>
                  {/each}
                </div>
              {/if}
            </div>

            {#if isAdmin}
              <div class="subsection composer-inline">
                <div class="panel-head compact">
                  <h3>Issue License</h3>
                  <span>New keys land in the picker below</span>
                </div>
                <div class="grid two">
                  <select bind:value={issueLicenseRole}>
                    {#each ISSUABLE_LICENSE_ROLES as role}
                      <option value={role}>{role}</option>
                    {/each}
                  </select>
                  <input bind:value={issueLicenseNotes} placeholder="Notes (optional)" />
                </div>
                <div class="actions">
                  <button disabled={issuingLicense} onclick={handleIssueLicense}>
                    {issuingLicense ? "Issuing..." : "Issue License"}
                  </button>
                </div>
              </div>
            {/if}

            <div class="subsection composer-inline">
              <div class="panel-head compact">
                <h3>Link a License</h3>
              </div>
              <div class="grid two">
                <select bind:value={selectedLicenseKey}>
                  <option value="">Select license key</option>
                  {#each availableLicenseKeys as license}
                    <option value={license.key}>{license.key} • {license.display_name || license.assigned_to || license.role}</option>
                  {/each}
                </select>
                <button onclick={handleLinkAccess} disabled={!selectedLicenseKey}>Link License</button>
              </div>
              {#if availableLicenseKeys.length === 0}
                <div class="empty small">All visible license keys are already linked to an employee.</div>
              {/if}
              {#if isAdmin && reassignableLicenseKeys.length > 0}
                <div class="grid two">
                  <select bind:value={reassignLicenseKey}>
                    <option value="">Reassign an existing license…</option>
                    {#each reassignableLicenseKeys as license}
                      <option value={license.key}>{license.key} • {license.display_name || license.assigned_to || license.role}</option>
                    {/each}
                  </select>
                  <button disabled={!reassignLicenseKey} onclick={() => handleReassignLicense(reassignLicenseKey)}>Reassign to This Employee</button>
                </div>
              {/if}
            </div>

            {#if isAdmin}
              <div class="subsection composer-inline">
                <div class="panel-head compact">
                  <h3>Login User & Role</h3>
                </div>
                {#if selectedAccessLinks.length === 0}
                  <div class="empty small">Link a license above before granting a login.</div>
                {:else}
                  <div class="grid two">
                    <select bind:value={selectedBindUserID}>
                      <option value="">Select existing user</option>
                      {#each loginUsers as user}
                        <option value={user.id}>{user.full_name || user.username} (@{user.username}{user.role_name ? ` • ${user.role_name}` : ""})</option>
                      {/each}
                    </select>
                    <button disabled={!selectedBindUserID || bindingUser} onclick={handleBindUser}>Bind Login</button>
                  </div>
                  <button class="ghost" onclick={() => (showNewLoginForm = !showNewLoginForm)}>
                    {showNewLoginForm ? "Cancel new login" : "+ Create new login user"}
                  </button>
                  {#if showNewLoginForm}
                    <div class="grid three">
                      <input bind:value={newLoginUsername} placeholder="Username" />
                      <input bind:value={newLoginPassword} type="password" placeholder="Temporary password" />
                      <select bind:value={newLoginRoleID}>
                        <option value="">Select role</option>
                        {#each loginRoles as role}
                          <option value={role.id}>{role.display_name || role.name}</option>
                        {/each}
                      </select>
                    </div>
                    <div class="actions">
                      <button disabled={bindingUser} onclick={handleCreateLoginUser}>Create & Bind</button>
                    </div>
                  {/if}
                {/if}
              </div>
            {/if}
          {/if}

          {#if employeeDetailTab === "compliance"}
            <!-- B4: identity/permit document expiry tracking. Document numbers
                 are PII — encrypted at rest server-side (FieldCrypto); the list
                 shows a masked number, the editor the full value (HR-gated).
                 Documents expiring within 60 days raise a notification via the
                 existing Article V notifications home. -->
            <div class="subsection">
              <div class="panel-head compact">
                <h3>Compliance Documents</h3>
                <span>{complianceDocuments.length} document(s)</span>
              </div>
              {#if complianceDocuments.length === 0}
                <div class="empty small">No documents tracked yet. Add a CPR, passport, visa, or permit below.</div>
              {:else}
                <div class="list compact-list">
                  {#each complianceDocuments as doc}
                    {@const days = daysUntil(doc.expires_on)}
                    <div class="list-row">
                      <div>
                        <strong>{DOC_TYPE_LABELS[doc.doc_type] || doc.doc_type}{doc.permit_subtype ? ` · ${doc.permit_subtype}` : ""}</strong>
                        <div class="meta">{doc.doc_number_masked || "—"}</div>
                      </div>
                      <div class="meta right">
                        <div>{doc.expires_on ? formatShortDate(doc.expires_on) : "No expiry"}</div>
                        {#if days !== null}
                          <div class:expiry-warn={days <= 60} class:expiry-past={days < 0}>
                            {days < 0 ? `Expired ${Math.abs(days)}d ago` : `${days}d left`}
                          </div>
                        {/if}
                      </div>
                      <div class="actions inline">
                        <button class="ghost" onclick={() => editDocument(doc)}>Edit</button>
                        <button class="ghost danger" onclick={() => handleDeleteDocument(doc.id)}>Delete</button>
                      </div>
                    </div>
                  {/each}
                </div>
              {/if}
            </div>

            <div class="subsection composer-inline">
              <div class="panel-head compact">
                <h3>{editingDocumentID ? "Edit Document" : "Add Document"}</h3>
                {#if editingDocumentID}
                  <button class="ghost" onclick={resetDocumentForm}>Cancel edit</button>
                {/if}
              </div>
              <div class="grid two">
                <label>
                  <span>Type</span>
                  <select bind:value={docType}>
                    <option value="cpr">CPR</option>
                    <option value="passport">Passport</option>
                    <option value="visa">Visa</option>
                    <option value="permit">Permit</option>
                  </select>
                </label>
                {#if docType === "permit"}
                  <label>
                    <span>Permit Subtype</span>
                    <input bind:value={docPermitSubtype} placeholder="e.g. work permit" />
                  </label>
                {/if}
                <label>
                  <span>Document Number</span>
                  <input bind:value={docNumber} placeholder="Document number" />
                </label>
                <label>
                  <span>Expires On</span>
                  <input bind:value={docExpiresOn} type="date" />
                </label>
              </div>
              <label class="stack">
                <span>Notes</span>
                <textarea bind:value={docNotes} placeholder="Optional notes"></textarea>
              </label>
              <div class="actions">
                <button disabled={complianceSaving} onclick={handleSaveDocument}>
                  {complianceSaving ? "Saving..." : editingDocumentID ? "Update Document" : "Add Document"}
                </button>
              </div>
            </div>
          {/if}

          {#if employeeDetailTab !== "compliance"}
            <div class="actions detail-save">
              <button disabled={saving} onclick={handleSaveProfile}>Save Profile</button>
            </div>
          {/if}
        {/if}
      </article>
    </section>
  {/if}

  {#if activeTab === "org"}
    <section class="panel">
      <div class="panel-head">
        <h2>Org / Reporting</h2>
        <span>{orgGroupEntries.length} reporting groups</span>
      </div>
      {#if loading}
        <div class="empty">Loading reporting structure...</div>
      {:else if orgGroupEntries.length === 0}
        <div class="empty">No reporting structure yet.</div>
      {:else}
        <div class="org-grid">
          {#each orgGroupEntries as [managerLabel, reports]}
            <div class="org-card">
              <div class="org-head">
                <strong>{managerLabel}</strong>
                <span>{reports.length} reports</span>
              </div>
              <div class="list compact-list">
                {#each reports as employee}
                  <button class="list-row selectable" onclick={() => { activeTab = "directory"; selectEmployee(employee.id); }}>
                    <div>
                      <strong>{employee.full_name}</strong>
                      <div class="meta">{employee.department || "No department"}</div>
                    </div>
                    <div class="meta right">
                      <div>{employee.job_title || "No title"}</div>
                      <div>{employee.is_active === false ? "Inactive" : "Active"}</div>
                    </div>
                  </button>
                {/each}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </section>
  {/if}

  {#if activeTab === "contributions"}
    <section class="panel">
      <div class="panel-head">
        <h2>Assignments & Contribution</h2>
        <span>{sortedContributions.length} employees tracked</span>
      </div>
      {#if loading}
        <div class="empty">Loading contribution metrics...</div>
      {:else if sortedContributions.length === 0}
        <div class="empty">No contribution history yet.</div>
      {:else}
        <div class="contribution-grid">
          {#each sortedContributions as summary}
            <button class="contribution-card" onclick={() => { activeTab = "directory"; selectEmployee(summary.employee_id); }}>
              <div class="card-head">
                <div>
                  <strong>{summary.employee_name}</strong>
                  <div class="meta">
                    {summary.employee_code} • {summary.department || "No department"}
                  </div>
                </div>
                <span class:inactive={summary.is_active === false}>
                  {summary.is_active === false ? "Inactive" : "Active"}
                </span>
              </div>

              <div class="metric-row">
                <div>
                  <span>Active Tasks</span>
                  <strong>{summary.active_task_count}</strong>
                </div>
                <div>
                  <span>Completed</span>
                  <strong>{summary.completed_task_count}</strong>
                </div>
                <div>
                  <span>Projects</span>
                  <strong>{summary.active_project_count}</strong>
                </div>
              </div>

              <div class="metric-row secondary-row">
                <div>
                  <span>Blocked</span>
                  <strong>{summary.blocked_task_count}</strong>
                </div>
                <div>
                  <span>Overdue</span>
                  <strong>{summary.overdue_task_count}</strong>
                </div>
                <div>
                  <span>Completion</span>
                  <strong>{summary.completion_rate.toFixed(0)}%</strong>
                </div>
              </div>

              <div class="meta strip">
                {summary.manager_name || "No manager"} • {summary.primary_device_name || "No device linked"}
              </div>
            </button>
          {/each}
        </div>
      {/if}
    </section>
  {/if}

  {#if activeTab === "payroll" && hasPayrollAccess}
    <!-- B2: Payroll lives with People — an HR/payroll-permissioned user reaches
         it here without needing finance:view. Same PayrollScreen the Finance
         Hub mounts (workspace mode); "Set up payroll" on an employee record
         deep-links here via payrollPresetEmployeeID. -->
    <section class="panel payroll-panel">
      <div class="panel-head">
        <h2>Payroll</h2>
        <div class="company-toggle">
          <button class:active={payrollCompany === "Acme Instrumentation"} onclick={() => (payrollCompany = "Acme Instrumentation")}>Acme Instrumentation</button>
          <button class:active={payrollCompany === "Beacon Controls"} onclick={() => (payrollCompany = "Beacon Controls")}>Beacon Controls</button>
        </div>
      </div>
      <PayrollScreen embedded mode="workspace" company={payrollCompany} presetEmployeeID={payrollPresetEmployeeID} />
    </section>
  {/if}
</div>

<style>
  .page {
    padding: 24px;
    display: grid;
    gap: 20px;
  }

  .header,
  .panel-head,
  .card-head,
  .org-head,
  .actions {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  h1,
  h2,
  h3,
  p {
    margin: 0;
  }

  .subtitle,
  .meta,
  label span,
  .metric-row span {
    color: var(--text-secondary);
  }

  .tabs,
  .actions {
    display: flex;
    gap: 10px;
    flex-wrap: wrap;
  }

  .composer,
  .panel,
  .org-card,
  .contribution-card,
  .stat-card {
    border: 1px solid var(--border);
    border-radius: 16px;
    background: var(--surface);
  }

  .composer,
  .panel {
    padding: 18px;
  }

  .grid,
  .content-grid,
  .stat-grid,
  .org-grid,
  .contribution-grid {
    display: grid;
    gap: 12px;
  }

  .grid {
    margin-top: 12px;
  }

  .grid.two {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .grid.three {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .directory-layout {
    grid-template-columns: minmax(320px, 0.9fr) minmax(420px, 1.1fr);
  }

  .list {
    display: grid;
    gap: 10px;
    margin-top: 14px;
  }

  .list-row {
    width: 100%;
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 12px 14px;
    background: color-mix(in srgb, var(--surface) 92%, var(--accent-primary) 8%);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    text-align: left;
  }

  .actions.inline {
    display: flex;
    gap: 6px;
    flex: 0 0 auto;
  }

  .expiry-warn {
    color: var(--warning, #b45309);
    font-weight: 600;
  }

  .expiry-past {
    color: var(--danger, #b91c1c);
    font-weight: 700;
  }

  .selectable {
    cursor: pointer;
  }

  .selectable.selected,
  .selectable:hover,
  .contribution-card:hover,
  .tabs button.active {
    border-color: color-mix(in srgb, var(--accent-primary) 62%, var(--border) 38%);
    background: color-mix(in srgb, var(--surface) 76%, var(--accent-primary) 24%);
  }

  .detail-panel,
  .org-card,
  .contribution-card {
    display: grid;
    gap: 14px;
  }

  label {
    display: grid;
    gap: 6px;
    font-size: 0.92rem;
  }

  .stack {
    margin-top: 14px;
  }

  input,
  select,
  textarea,
  button {
    font: inherit;
  }

  input,
  select,
  textarea {
    width: 100%;
    border-radius: 12px;
    border: 1px solid var(--border);
    background: color-mix(in srgb, var(--surface) 94%, white 6%);
    color: var(--text-primary);
    padding: 12px 14px;
    box-sizing: border-box;
  }

  textarea {
    min-height: 110px;
    resize: vertical;
  }

  button {
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 10px 14px;
    background: color-mix(in srgb, var(--surface) 84%, var(--accent-primary) 16%);
    color: var(--text-primary);
  }

  button.secondary {
    background: color-mix(in srgb, var(--surface) 88%, #d97706 12%);
  }

  button.danger {
    border-color: color-mix(in srgb, #dc2626 40%, var(--border) 60%);
    background: color-mix(in srgb, var(--surface) 84%, #dc2626 16%);
  }

  .filter-pills {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
    margin: 12px 0;
  }

  .filter-pills button {
    border-radius: 999px;
    padding: 8px 12px;
    background: color-mix(in srgb, var(--surface) 92%, var(--accent-primary) 8%);
  }

  .filter-pills button.active {
    border-color: color-mix(in srgb, var(--accent-primary) 62%, var(--border) 38%);
    background: color-mix(in srgb, var(--surface) 76%, var(--accent-primary) 24%);
  }

  .list-row.archived {
    background: color-mix(in srgb, var(--surface) 86%, #92400e 14%);
  }

  .archive-panel {
    border: 1px solid color-mix(in srgb, #dc2626 24%, var(--border) 76%);
    border-radius: 14px;
    padding: 14px;
    display: grid;
    gap: 12px;
    background: color-mix(in srgb, var(--surface) 90%, #fee2e2 10%);
    margin-top: 12px;
  }

  .archive-panel p {
    margin-top: 6px;
    color: var(--text-secondary);
    line-height: 1.5;
  }

  .archived-panel {
    border-color: color-mix(in srgb, #92400e 30%, var(--border) 70%);
    background: color-mix(in srgb, var(--surface) 88%, #fef3c7 12%);
  }

  .stat-card {
    padding: 14px;
  }

  .stat-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
    margin-top: 12px;
  }

  .stat-card span,
  .metric-row span {
    display: block;
    font-size: 0.82rem;
  }

  .stat-card strong,
  .metric-row strong {
    display: block;
    margin-top: 6px;
    font-size: 1.1rem;
  }

  .subsection {
    display: grid;
    gap: 10px;
    margin-top: 10px;
  }

  .compact {
    align-items: end;
  }

  .compact-list {
    margin-top: 0;
  }

  .empty {
    border: 1px dashed var(--border);
    border-radius: 14px;
    padding: 20px;
    color: var(--text-secondary);
    margin-top: 14px;
  }

  .empty.small {
    margin-top: 0;
    padding: 14px;
  }

  .org-grid,
  .contribution-grid {
    margin-top: 16px;
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  }

  .org-card,
  .contribution-card {
    padding: 16px;
  }

  .metric-row {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 10px;
  }

  .secondary-row {
    padding-top: 4px;
    border-top: 1px solid var(--border);
  }

  .strip {
    padding-top: 8px;
    border-top: 1px solid var(--border);
  }

  .inactive {
    color: #b45309;
  }

  .right {
    text-align: right;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 14px;
    flex-wrap: wrap;
  }

  button.ghost {
    background: transparent;
    border-color: color-mix(in srgb, var(--accent-primary) 40%, var(--border) 60%);
  }

  .detail-subtabs {
    display: flex;
    align-items: center; /* Wave 11 A3: never let a pill stretch to container height (200px-oval guard) */
    gap: 8px;
    flex-wrap: wrap;
    margin-bottom: 4px;
  }

  .detail-subtabs button {
    border-radius: 999px;
    padding: 8px 14px;
    line-height: 1.2; /* compact, height driven by content+padding, not inherited leading */
    white-space: nowrap;
    border: 1px solid var(--border);
    background: color-mix(in srgb, var(--surface) 92%, var(--accent-primary) 8%);
    cursor: pointer;
  }

  .detail-subtabs button.active {
    border-color: color-mix(in srgb, var(--accent-primary) 62%, var(--border) 38%);
    background: color-mix(in srgb, var(--surface) 76%, var(--accent-primary) 24%);
  }

  .detail-save {
    margin-top: 6px;
    padding-top: 14px;
    border-top: 1px solid var(--border);
  }

  .composer-inline {
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 14px;
    background: color-mix(in srgb, var(--surface) 94%, var(--accent-primary) 6%);
  }

  .access-row .meta.right div:first-child strong {
    color: var(--text-primary);
  }

  .inline-field {
    display: grid;
    gap: 4px;
  }

  .error-text {
    color: #dc2626;
    font-size: 0.8rem;
  }

  .company-toggle {
    display: flex;
    border: 1px solid var(--border);
    border-radius: 10px;
    overflow: hidden;
  }

  .company-toggle button {
    border: none;
    border-radius: 0;
    background: transparent;
  }

  .company-toggle button.active {
    background: color-mix(in srgb, var(--surface) 76%, var(--accent-primary) 24%);
  }

  .payroll-panel {
    display: grid;
    gap: 16px;
  }

  @media (max-width: 1100px) {
    .directory-layout,
    .grid.two,
    .grid.three,
    .stat-grid {
      grid-template-columns: 1fr;
    }
  }

  @media (max-width: 720px) {
    .page {
      padding: 16px;
    }

    .header,
    .panel-head,
    .card-head,
    .org-head,
    .actions {
      align-items: stretch;
      flex-direction: column;
    }

    .metric-row {
      grid-template-columns: 1fr;
    }

    .header-actions {
      align-items: stretch;
      flex-direction: column;
    }
  }
</style>
