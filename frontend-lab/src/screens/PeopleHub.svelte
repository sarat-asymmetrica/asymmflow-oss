<script lang="ts">
  /* PeopleHub — HR console (K4 operational hub), THE HIGHEST-PII screen in
   * the kernel. TabShell hosts four independent surfaces (Directory / Org /
   * Contributions / Payroll) behind a shared "Add Employee" composer in the
   * TabShell header. Directory's detail Card uses an INNER ViewSwitcher
   * (Profile / Work / Access / Compliance) — same employee dataset, different
   * view, not a nested TabShell. Payroll embeds the already-built Payroll
   * screen with `embedded presetEmployeeID`. All state/derivation/mutation
   * calls live in people-vm.svelte.ts (L5); this file only composes
   * primitives and renders (L1). See screens/parity/PeopleHub.parity.md. */
  import { onMount } from 'svelte'
  import PageShell from '$kernel/primitives/PageShell.svelte'
  import TabShell from '$kernel/primitives/TabShell.svelte'
  import ViewSwitcher from '$kernel/primitives/ViewSwitcher.svelte'
  import Stack from '$kernel/primitives/Stack.svelte'
  import Row from '$kernel/primitives/Row.svelte'
  import Grid from '$kernel/primitives/Grid.svelte'
  import Card from '$kernel/primitives/Card.svelte'
  import FormGrid from '$kernel/primitives/FormGrid.svelte'
  import DataTable from '$kernel/primitives/DataTable.svelte'
  import Badge from '$kernel/controls/Badge.svelte'
  import Button from '$kernel/controls/Button.svelte'
  import EmptyState from '$kernel/controls/EmptyState.svelte'
  import ConfirmDialog from '$kernel/controls/ConfirmDialog.svelte'
  import FilterChips from '$kernel/controls/FilterChips.svelte'
  import SearchInput from '$kernel/controls/SearchInput.svelte'
  import StatTileGrid from '$kernel/widgets/StatTileGrid.svelte'
  import RankedBarList from '$kernel/widgets/RankedBarList.svelte'
  import DistributionWidget from '$kernel/widgets/DistributionWidget.svelte'
  import CalloutWidget from '$kernel/widgets/CalloutWidget.svelte'
  import Payroll from './Payroll.svelte'
  import type { ColumnSpec, StatusSpec } from '$kernel/descriptor'
  import {
    PeopleHubViewModel,
    DOC_TYPE_LABELS,
    EMPLOYEE_STATUS_TONES,
    daysUntilExpiry,
    employeeStatusLabel,
    expiryTone,
    isArchivedEmployee,
    type DirectoryDetailTab,
    type PeopleTab,
  } from './people-vm.svelte'
  import type { EmployeeComplianceDocument, EmployeeProfile, ProjectAssignment } from '../bridge/people'

  const vm = new PeopleHubViewModel()
  onMount(() => void vm.load())

  const DETAIL_VIEWS: { key: DirectoryDetailTab; label: string }[] = [
    { key: 'profile', label: 'Profile' },
    { key: 'work', label: 'Work' },
    { key: 'access', label: 'Access' },
    { key: 'compliance', label: 'Compliance' },
  ]

  // FilterChips renders its own built-in "All" chip (selected='') — these two
  // are the explicit chips alongside it (mirrors payroll-vm's divisionFilter).
  const STATUS_FILTERS: { value: 'active' | 'archive'; label: string }[] = [
    { value: 'active', label: 'Active' },
    { value: 'archive', label: 'Archive' },
  ]

  const employeeStatus: StatusSpec<EmployeeProfile> = {
    value: (e) => employeeStatusLabel(e),
    tones: EMPLOYEE_STATUS_TONES,
  }

  const employeeColumns: ColumnSpec<EmployeeProfile>[] = [
    { key: 'fullName', label: 'Employee', content: 'name', value: (e) => e.fullName || '(no name on file)', grow: true, minWidth: 180 },
    { key: 'employeeCode', label: 'Code', content: 'code', value: (e) => e.employeeCode, minWidth: 100 },
    { key: 'department', label: 'Department', content: 'text', value: (e) => e.department, minWidth: 140 },
    { key: 'jobTitle', label: 'Job Title', content: 'text', value: (e) => e.jobTitle, minWidth: 150 },
    { key: 'status', label: 'Status', content: 'status', value: (e) => employeeStatusLabel(e), minWidth: 100 },
  ]

  const assignmentColumns: ColumnSpec<ProjectAssignment>[] = [
    { key: 'projectName', label: 'Project', content: 'name', value: (a) => a.projectName, grow: true, minWidth: 180 },
    { key: 'role', label: 'Role', content: 'text', value: (a) => a.role, minWidth: 100 },
    { key: 'allocationPercent', label: 'Allocation', content: 'quantity', value: (a) => a.allocationPercent, minWidth: 100 },
    { key: 'status', label: 'Status', content: 'text', value: (a) => (a.isActive === false ? 'Inactive' : 'Active'), minWidth: 90 },
  ]

  function docNumberDisplay(doc: EmployeeComplianceDocument): string {
    return vm.canViewUnmasked ? doc.docNumberMasked || '—' : '••••••'
  }
</script>

<PageShell title="People" subtitle="Employee directory, reporting structure, access mapping, and contribution insight.">
  {#if vm.error}
    <EmptyState message={`Could not load people hub: ${vm.error}`}>
      {#snippet actions()}
        <Button onclick={() => vm.load()}>Retry</Button>
      {/snippet}
    </EmptyState>
  {:else if vm.loading}
    <EmptyState message="Loading people hub…" />
  {:else}
    <TabShell
      activeKey={vm.activeTab}
      onSelect={(k) => vm.setTab(k as PeopleTab)}
      tabs={[
        { key: 'directory', label: 'Directory', badge: vm.activeEmployeeCount, content: directoryTab },
        { key: 'org', label: 'Org', badge: vm.orgGroups.length, content: orgTab },
        { key: 'contributions', label: 'Contributions', content: contributionsTab },
        { key: 'payroll', label: 'Payroll', visible: vm.canViewPayroll, content: payrollTab },
      ]}
      header={composerHeader}
    />
  {/if}
</PageShell>

{#snippet composerHeader()}
  <Card>
    <Stack gap="md">
      <Row justify="between" wrap>
        <span class="ph-section-label">Add Employee</span>
        <span class="ph-meta">{vm.employees.length} total · {vm.activeEmployeeCount} active · {vm.archivedEmployeeCount} archived</span>
      </Row>
      <FormGrid columns={3}>
        <label class="k-field">
          <span class="k-field-label">Full Name</span>
          <input class="k-input" bind:value={vm.createDraft.fullName} placeholder="Full name" />
        </label>
        <label class="k-field">
          <span class="k-field-label">Department</span>
          <input class="k-input" bind:value={vm.createDraft.department} placeholder="Department" />
        </label>
        <label class="k-field">
          <span class="k-field-label">Job Title</span>
          <input class="k-input" bind:value={vm.createDraft.jobTitle} placeholder="Job title" />
        </label>
        <label class="k-field">
          <span class="k-field-label">Email</span>
          <input class="k-input" type="email" bind:value={vm.createDraft.email} placeholder="Email" />
        </label>
        <label class="k-field">
          <span class="k-field-label">Phone</span>
          <input class="k-input" bind:value={vm.createDraft.phone} placeholder="Phone" />
        </label>
        <label class="k-field">
          <span class="k-field-label">Start Date</span>
          <input class="k-input" type="date" bind:value={vm.createDraft.startDate} />
        </label>
        <label class="k-field k-field-wide">
          <span class="k-field-label">Manager</span>
          <select class="k-input" bind:value={vm.createDraft.managerEmployeeId}>
            <option value="">No manager</option>
            {#each vm.employees as employee (employee.id)}
              <option value={employee.id}>{employee.fullName || '(no name on file)'}</option>
            {/each}
          </select>
        </label>
      </FormGrid>
      {#if vm.createError}
        <CalloutWidget items={[{ label: 'Create failed', text: vm.createError, tone: 'danger' }]} />
      {/if}
      <Row justify="end">
        <Button variant="primary" onclick={() => vm.createEmployeeFromDraft()} disabled={vm.creatingEmployee}>
          {vm.creatingEmployee ? 'Creating…' : 'Create Employee'}
        </Button>
      </Row>
    </Stack>
  </Card>
{/snippet}

{#snippet directoryTab()}
  <Grid min="380px" gap="lg">
    <Stack gap="md">
      <Card>
        <Stack gap="sm">
          <FilterChips label="Status" options={STATUS_FILTERS} bind:selected={vm.directoryStatusFilter} />
          <SearchInput bind:value={vm.directorySearch} placeholder="Search by name, code, department, or title" />
        </Stack>
      </Card>
      <Card padding="none">
        {#if vm.filteredEmployees.length === 0}
          <EmptyState message="No employee profiles match this search." />
        {:else}
          <DataTable
            columns={employeeColumns}
            rows={vm.filteredEmployees}
            id={(e) => e.id}
            status={employeeStatus}
            selectedId={vm.selectedEmployeeId || null}
            onSelect={(e) => vm.selectEmployee(e.id)}
          />
        {/if}
      </Card>
    </Stack>

    <Card>
      {#if !vm.selectedEmployee}
        <EmptyState message="Select an employee to manage their profile." />
      {:else}
        {@const employee = vm.selectedEmployee}
        <Stack gap="lg">
          <Row justify="between" wrap>
            <Stack gap="xs">
              <span class="ph-employee-name">{employee.fullName || '(no name on file)'}</span>
              <span class="ph-meta">{employee.employeeCode} · {employee.department || 'No department'}</span>
            </Stack>
            <Badge tone={EMPLOYEE_STATUS_TONES[employeeStatusLabel(employee)] ?? 'neutral'} label={employeeStatusLabel(employee)} />
          </Row>

          <ViewSwitcher views={DETAIL_VIEWS} activeKey={vm.detailTab} onSelect={(k) => vm.setDetailTab(k as DirectoryDetailTab)} />

          {#if vm.detailTab === 'profile'}
            <Stack gap="md">
              <FormGrid columns={2}>
                <label class="k-field">
                  <span class="k-field-label">Full Name</span>
                  <input class="k-input" bind:value={vm.profileDraft.fullName} placeholder="Full name" />
                </label>
                <label class="k-field">
                  <span class="k-field-label">Preferred Name</span>
                  <input class="k-input" bind:value={vm.profileDraft.preferredName} placeholder="Preferred name" />
                </label>
                <label class="k-field">
                  <span class="k-field-label">Email</span>
                  <input class="k-input" type="email" bind:value={vm.profileDraft.email} placeholder="Email" />
                </label>
                <label class="k-field">
                  <span class="k-field-label">Phone</span>
                  <input class="k-input" bind:value={vm.profileDraft.phone} placeholder="Phone" />
                </label>
                <label class="k-field k-field-wide">
                  <span class="k-field-label">Emergency Contact</span>
                  <input class="k-input" bind:value={vm.profileDraft.emergencyContact} placeholder="Emergency contact" />
                </label>
              </FormGrid>
              <label class="k-field">
                <span class="k-field-label">Notes</span>
                <textarea class="k-input k-input-area" bind:value={vm.profileDraft.notes} placeholder="Role notes, responsibilities, or context"></textarea>
              </label>
            </Stack>
          {:else if vm.detailTab === 'work'}
            <Stack gap="lg">
              <FormGrid columns={2}>
                <label class="k-field">
                  <span class="k-field-label">Department</span>
                  <input class="k-input" bind:value={vm.profileDraft.department} placeholder="Department" />
                </label>
                <label class="k-field">
                  <span class="k-field-label">Job Title</span>
                  <input class="k-input" bind:value={vm.profileDraft.jobTitle} placeholder="Job title" />
                </label>
                <label class="k-field">
                  <span class="k-field-label">Manager</span>
                  <select class="k-input" bind:value={vm.profileDraft.managerEmployeeId}>
                    <option value="">No manager</option>
                    {#each vm.employees.filter((candidate) => candidate.id !== employee.id) as candidate (candidate.id)}
                      <option value={candidate.id}>{candidate.fullName || '(no name on file)'}</option>
                    {/each}
                  </select>
                </label>
                <label class="k-field">
                  <span class="k-field-label">Status</span>
                  <select class="k-input" bind:value={vm.profileDraft.employmentStatus}>
                    <option value="active">Active</option>
                    <option value="on_leave">On Leave</option>
                    <option value="probation">Probation</option>
                    <option value="contract">Contract</option>
                  </select>
                </label>
                <label class="k-field">
                  <span class="k-field-label">Start Date</span>
                  <input class="k-input" type="date" bind:value={vm.profileDraft.startDate} />
                </label>
              </FormGrid>

              {#if vm.reportingChain.length > 0}
                <Stack gap="xs">
                  <span class="ph-section-label">Reporting Chain</span>
                  <span class="ph-meta">{vm.reportingChain.map((m) => m.fullName || '(no name on file)').join(' → ')}</span>
                </Stack>
              {/if}

              <StatTileGrid
                sections={[
                  {
                    title: 'Work Summary',
                    items: [
                      { label: 'Projects', value: vm.projectAssignments.filter((a) => a.isActive !== false).length },
                      { label: 'Licenses', value: vm.selectedAccessLinks.length },
                      { label: 'Manager', value: employee.managerName || 'None' },
                    ],
                  },
                ]}
              />

              {#if vm.canViewPayroll}
                <Row justify="end">
                  <Button onclick={() => (vm.activeTab = 'payroll')}>Set up payroll →</Button>
                </Row>
              {/if}

              {#if isArchivedEmployee(employee)}
                <Card>
                  <Stack gap="sm">
                    <span class="ph-section-label">Archived Employee</span>
                    <span class="ph-meta">
                      {employee.archiveReason || 'Employee is inactive. Historical work remains linked to this profile.'}
                      {employee.archivedAt ? ` Archived ${employee.archivedAt}.` : ''}
                    </span>
                    {#if vm.isAdmin}
                      <Row justify="end">
                        <Button onclick={() => vm.reactivateEmployee()} disabled={vm.savingProfile}>Reactivate</Button>
                      </Row>
                    {/if}
                  </Stack>
                </Card>
              {:else if vm.isAdmin}
                <Card>
                  <Stack gap="sm">
                    <span class="ph-section-label">Employee Archive</span>
                    <span class="ph-meta">
                      Archiving revokes login access and project memberships and routes to the Approvals queue. Linked tasks, offers,
                      orders, and invoices stay attached to the archived profile. This is the only way to deactivate — the Status field
                      above only tracks work state.
                    </span>
                    {#if vm.archiveError}
                      <CalloutWidget items={[{ label: 'Archive failed', text: vm.archiveError, tone: 'danger' }]} />
                    {/if}
                    <Row justify="end">
                      <Button variant="danger" onclick={() => vm.requestArchive()} disabled={vm.archiving}>
                        {vm.archiving ? 'Archiving…' : 'Archive Employee'}
                      </Button>
                    </Row>
                  </Stack>
                </Card>
              {/if}

              <Card padding="none">
                {#if vm.projectAssignments.length === 0}
                  <EmptyState message="No project assignments yet." />
                {:else}
                  <DataTable columns={assignmentColumns} rows={vm.projectAssignments} id={(a) => a.id} />
                {/if}
              </Card>
            </Stack>
          {:else if vm.detailTab === 'access'}
            <Stack gap="lg">
              <Card padding="none">
                {#if vm.selectedAccessLinks.length === 0}
                  <EmptyState message="No license linked yet. Link one below to grant app access." />
                {:else}
                  <Stack gap="sm">
                    {#each vm.selectedAccessLinks as link (link.id)}
                      {@const boundUser = vm.loginUsers.find((u) => u.id === link.userId)}
                      <Row justify="between" wrap>
                        <Stack gap="xs">
                          <span class="ph-mono">{link.licenseKey}</span>
                          <span class="ph-meta">
                            {link.deviceName || 'No device mapped'} · {link.accessStatus || 'active'}{link.isPrimary ? ' · Primary' : ''}
                          </span>
                        </Stack>
                        <Stack gap="xs" align="end">
                          {#if boundUser}
                            <span class="ph-mono">{boundUser.fullName || boundUser.username}</span>
                            <span class="ph-meta">@{boundUser.username} · {boundUser.roleName || 'No role'}</span>
                          {:else}
                            <span class="ph-meta">No login user bound</span>
                          {/if}
                        </Stack>
                      </Row>
                    {/each}
                  </Stack>
                {/if}
              </Card>

              {#if vm.accessError}
                <CalloutWidget items={[{ label: 'Access action failed', text: vm.accessError, tone: 'danger' }]} />
              {/if}

              {#if vm.isAdmin}
                <Card>
                  <Stack gap="sm">
                    <span class="ph-section-label">Issue License</span>
                    <FormGrid columns={2}>
                      <label class="k-field">
                        <span class="k-field-label">Role</span>
                        <select class="k-input" bind:value={vm.issueLicenseRole}>
                          {#each vm.issuableRoles as role (role)}
                            <option value={role}>{role}</option>
                          {/each}
                        </select>
                      </label>
                      <label class="k-field">
                        <span class="k-field-label">Notes</span>
                        <input class="k-input" bind:value={vm.issueLicenseNotes} placeholder="Notes (optional)" />
                      </label>
                    </FormGrid>
                    <Row justify="end">
                      <Button onclick={() => vm.issueLicense()} disabled={vm.issuingLicense}>
                        {vm.issuingLicense ? 'Issuing…' : 'Issue License'}
                      </Button>
                    </Row>
                  </Stack>
                </Card>
              {/if}

              <Card>
                <Stack gap="sm">
                  <span class="ph-section-label">Link a License</span>
                  <FormGrid columns={2}>
                    <label class="k-field k-field-wide">
                      <span class="k-field-label">Available License Keys</span>
                      <select class="k-input" bind:value={vm.selectedLicenseKey}>
                        <option value="">Select license key</option>
                        {#each vm.availableLicenseKeys as license (license.key)}
                          <option value={license.key}>{license.key} · {license.displayName || license.assignedTo || license.role}</option>
                        {/each}
                      </select>
                    </label>
                  </FormGrid>
                  <Row justify="end">
                    <Button onclick={() => vm.linkAccess()} disabled={!vm.selectedLicenseKey || vm.linkingAccess}>Link License</Button>
                  </Row>
                  {#if vm.isAdmin && vm.reassignableLicenseKeys.length > 0}
                    <FormGrid columns={2}>
                      <label class="k-field k-field-wide">
                        <span class="k-field-label">Reassign an Existing License</span>
                        <select class="k-input" bind:value={vm.reassignLicenseKey}>
                          <option value="">Select license to reassign</option>
                          {#each vm.reassignableLicenseKeys as license (license.key)}
                            <option value={license.key}>{license.key} · {license.displayName || license.assignedTo || license.role}</option>
                          {/each}
                        </select>
                      </label>
                    </FormGrid>
                    <Row justify="end">
                      <Button disabled={!vm.reassignLicenseKey} onclick={() => vm.reassignLicenseToSelected(vm.reassignLicenseKey)}>
                        Reassign to This Employee
                      </Button>
                    </Row>
                  {/if}
                </Stack>
              </Card>

              {#if vm.isAdmin}
                <Card>
                  <Stack gap="sm">
                    <span class="ph-section-label">Login User & Role</span>
                    {#if vm.selectedAccessLinks.length === 0}
                      <EmptyState message="Link a license above before granting a login." />
                    {:else}
                      <FormGrid columns={2}>
                        <label class="k-field k-field-wide">
                          <span class="k-field-label">Bind Existing User</span>
                          <select class="k-input" bind:value={vm.bindUserId}>
                            <option value="">Select existing user</option>
                            {#each vm.loginUsers as user (user.id)}
                              <option value={user.id}>{user.fullName || user.username} (@{user.username}{user.roleName ? ` · ${user.roleName}` : ''})</option>
                            {/each}
                          </select>
                        </label>
                      </FormGrid>
                      <Row justify="end">
                        <Button disabled={!vm.bindUserId || vm.bindingUser} onclick={() => vm.bindUser()}>Bind Login</Button>
                      </Row>
                      <Button onclick={() => (vm.showNewLoginForm = !vm.showNewLoginForm)}>
                        {vm.showNewLoginForm ? 'Cancel new login' : '+ Create new login user'}
                      </Button>
                      {#if vm.showNewLoginForm}
                        <FormGrid columns={3}>
                          <label class="k-field">
                            <span class="k-field-label">Username</span>
                            <input class="k-input" bind:value={vm.newLoginUsername} placeholder="Username" />
                          </label>
                          <label class="k-field">
                            <span class="k-field-label">Temporary Password</span>
                            <input class="k-input" type="password" bind:value={vm.newLoginPassword} placeholder="Temporary password" />
                          </label>
                          <label class="k-field">
                            <span class="k-field-label">Role</span>
                            <select class="k-input" bind:value={vm.newLoginRoleId}>
                              <option value="">Select role</option>
                              {#each vm.loginRoles as role (role.id)}
                                <option value={role.id}>{role.displayName || role.name}</option>
                              {/each}
                            </select>
                          </label>
                        </FormGrid>
                        <Row justify="end">
                          <Button disabled={vm.bindingUser} onclick={() => vm.createAndBindLoginUser()}>Create & Bind</Button>
                        </Row>
                      {/if}
                    {/if}
                  </Stack>
                </Card>
              {/if}
            </Stack>
          {:else}
            <Stack gap="lg">
              <Row justify="end">
                <Button onclick={() => (vm.canViewUnmasked = !vm.canViewUnmasked)}>
                  {vm.canViewUnmasked ? 'Mask Document Numbers' : 'Unmask Document Numbers'}
                </Button>
              </Row>
              <Card padding="none">
                {#if vm.complianceDocuments.length === 0}
                  <EmptyState message="No documents tracked yet. Add a CPR, passport, visa, or permit below." />
                {:else}
                  <Stack gap="sm">
                    {#each vm.complianceDocuments as doc (doc.id)}
                      {@const days = daysUntilExpiry(doc.expiresOn)}
                      <Row justify="between" wrap>
                        <Stack gap="xs">
                          <span class="ph-mono">{DOC_TYPE_LABELS[doc.docType] || doc.docType}{doc.permitSubtype ? ` · ${doc.permitSubtype}` : ''}</span>
                          <span class="ph-meta">{docNumberDisplay(doc)}</span>
                        </Stack>
                        <Stack gap="xs" align="end">
                          <span class="ph-meta">{doc.expiresOn || 'No expiry'}</span>
                          {#if days !== null}
                            <Badge tone={expiryTone(days)} label={days < 0 ? `Expired ${Math.abs(days)}d ago` : `${days}d left`} />
                          {/if}
                        </Stack>
                        <Row gap="xs">
                          <Button onclick={() => vm.editDocument(doc)}>Edit</Button>
                          <Button variant="danger" onclick={() => vm.removeDocument(doc.id)}>Delete</Button>
                        </Row>
                      </Row>
                    {/each}
                  </Stack>
                {/if}
              </Card>

              <Card>
                <Stack gap="sm">
                  <Row justify="between" wrap>
                    <span class="ph-section-label">{vm.editingDocumentId ? 'Edit Document' : 'Add Document'}</span>
                    {#if vm.editingDocumentId}
                      <Button onclick={() => vm.resetDocumentForm()}>Cancel edit</Button>
                    {/if}
                  </Row>
                  <FormGrid columns={2}>
                    <label class="k-field">
                      <span class="k-field-label">Type</span>
                      <select class="k-input" bind:value={vm.docType}>
                        <option value="cpr">CPR</option>
                        <option value="passport">Passport</option>
                        <option value="visa">Visa</option>
                        <option value="permit">Permit</option>
                      </select>
                    </label>
                    {#if vm.docType === 'permit'}
                      <label class="k-field">
                        <span class="k-field-label">Permit Subtype</span>
                        <input class="k-input" bind:value={vm.docPermitSubtype} placeholder="e.g. work permit" />
                      </label>
                    {/if}
                    <label class="k-field">
                      <span class="k-field-label">Document Number</span>
                      <input class="k-input" bind:value={vm.docNumber} placeholder="Document number" />
                    </label>
                    <label class="k-field">
                      <span class="k-field-label">Expires On</span>
                      <input class="k-input" type="date" bind:value={vm.docExpiresOn} />
                    </label>
                  </FormGrid>
                  <label class="k-field">
                    <span class="k-field-label">Notes</span>
                    <textarea class="k-input k-input-area" bind:value={vm.docNotes} placeholder="Optional notes"></textarea>
                  </label>
                  {#if vm.documentError}
                    <CalloutWidget items={[{ label: 'Save failed', text: vm.documentError, tone: 'danger' }]} />
                  {/if}
                  <Row justify="end">
                    <Button variant="primary" disabled={vm.savingDocument} onclick={() => vm.saveDocument()}>
                      {vm.savingDocument ? 'Saving…' : vm.editingDocumentId ? 'Update Document' : 'Add Document'}
                    </Button>
                  </Row>
                </Stack>
              </Card>
            </Stack>
          {/if}

          {#if vm.detailTab !== 'compliance'}
            {#if vm.profileError}
              <CalloutWidget items={[{ label: 'Save failed', text: vm.profileError, tone: 'danger' }]} />
            {/if}
            <Row justify="end">
              <Button variant="primary" disabled={vm.savingProfile} onclick={() => vm.saveProfile()}>
                {vm.savingProfile ? 'Saving…' : 'Save Profile'}
              </Button>
            </Row>
          {/if}
        </Stack>
      {/if}
    </Card>
  </Grid>
{/snippet}

{#snippet orgTab()}
  {#if vm.orgGroups.length === 0}
    <EmptyState message="No reporting structure yet." />
  {:else}
    <Grid min="320px" gap="lg">
      {#each vm.orgGroups as [managerLabel, reports] (managerLabel)}
        <Stack gap="sm">
          <Row justify="between" wrap>
            <span class="ph-section-label">{managerLabel}</span>
            <span class="ph-meta">{reports.length} reports</span>
          </Row>
          <Card padding="none">
            <DataTable
              columns={employeeColumns}
              rows={reports}
              id={(e) => e.id}
              status={employeeStatus}
              onSelect={(e) => {
                vm.activeTab = 'directory'
                void vm.selectEmployee(e.id)
              }}
            />
          </Card>
        </Stack>
      {/each}
    </Grid>
  {/if}
{/snippet}

{#snippet contributionsTab()}
  <Stack gap="lg">
    <StatTileGrid
      sections={[
        {
          title: 'Contributions',
          items: [
            { label: 'Employees Tracked', value: vm.contributionsSummary.tracked },
            { label: 'Avg Completion', value: `${vm.contributionsSummary.avgCompletion}%` },
            { label: 'Active Tasks', value: vm.contributionsSummary.totalActive },
            { label: 'Overdue Tasks', value: vm.contributionsSummary.totalOverdue, tone: vm.contributionsSummary.totalOverdue > 0 ? 'warning' : 'neutral' },
          ],
        },
      ]}
    />

    {#if vm.contributionsDistribution.length > 0}
      <Card>
        <Stack gap="sm">
          <span class="ph-section-label">Task Mix</span>
          <DistributionWidget segments={vm.contributionsDistribution} />
        </Stack>
      </Card>
    {/if}

    <Card>
      <Stack gap="sm">
        <span class="ph-section-label">Completion Rate by Employee</span>
        {#if vm.contributionsRanked.length === 0}
          <EmptyState message="No contribution history yet." />
        {:else}
          <RankedBarList rows={vm.contributionsRanked} unit="quantity" />
        {/if}
      </Stack>
    </Card>
  </Stack>
{/snippet}

{#snippet payrollTab()}
  <Payroll embedded presetEmployeeID={vm.selectedEmployeeId} />
{/snippet}

{#if vm.archiveConfirmOpen && vm.selectedEmployee}
  <ConfirmDialog
    title="Archive employee?"
    message={`This archives ${vm.selectedEmployee.fullName || vm.selectedEmployee.employeeCode} — login access and project memberships are revoked and the request routes to the Approvals queue.`}
    confirmLabel="Archive"
    reasonLabel="Reason"
    requireReason
    onConfirm={(reason) => vm.confirmArchive(reason || '')}
    onCancel={() => vm.cancelArchive()}
  />
{/if}

<style>
  /* Typography only (L1) — layout/spacing lives in primitives; native form
   * controls use the kernel-owned .k-field/.k-field-label/.k-input classes. */
  .ph-section-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-secondary);
  }
  .ph-employee-name {
    font-family: var(--font-display);
    font-size: var(--section-title-size);
    font-weight: var(--section-title-weight);
    overflow-wrap: break-word;
  }
  .ph-meta {
    font-size: var(--meta-size);
    color: var(--text-secondary);
    overflow-wrap: break-word;
  }
  .ph-mono {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    overflow-wrap: break-word;
  }
</style>
