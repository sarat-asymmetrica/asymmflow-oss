<script lang="ts">
  /* WorkHub — task/project collaboration console (K4 operational hub).
   * TabShell with 4 tabs: My Work / Team Board (task lists close to the
   * ledger shape, kept bespoke-on-primitives for the shared VM) / Projects
   * (bespoke master-detail: list + create/edit + member roster + activity
   * feed) / Approvals (embeds the built ApprovalsQueue ledger archetype).
   * All state/derivation/mutation-calls live in work-vm.svelte.ts (L5); this
   * file only composes primitives and renders (L1). See
   * screens/parity/WorkHub.parity.md. */
  import { onMount } from 'svelte'
  import PageShell from '$kernel/primitives/PageShell.svelte'
  import TabShell from '$kernel/primitives/TabShell.svelte'
  import Toolbar from '$kernel/primitives/Toolbar.svelte'
  import Stack from '$kernel/primitives/Stack.svelte'
  import Row from '$kernel/primitives/Row.svelte'
  import Grid from '$kernel/primitives/Grid.svelte'
  import Card from '$kernel/primitives/Card.svelte'
  import FormGrid from '$kernel/primitives/FormGrid.svelte'
  import DataTable from '$kernel/primitives/DataTable.svelte'
  import Modal from '$kernel/primitives/Modal.svelte'
  import Badge from '$kernel/controls/Badge.svelte'
  import Button from '$kernel/controls/Button.svelte'
  import EmptyState from '$kernel/controls/EmptyState.svelte'
  import ConfirmDialog from '$kernel/controls/ConfirmDialog.svelte'
  import SearchInput from '$kernel/controls/SearchInput.svelte'
  import FilterChips from '$kernel/controls/FilterChips.svelte'
  import StatTileGrid from '$kernel/widgets/StatTileGrid.svelte'
  import DistributionWidget from '$kernel/widgets/DistributionWidget.svelte'
  import ActivityFeed from '$kernel/widgets/ActivityFeed.svelte'
  import CalloutWidget from '$kernel/widgets/CalloutWidget.svelte'
  import DocumentLedger from '$kernel/archetypes/DocumentLedger.svelte'
  import type { ColumnSpec, StatusSpec } from '$kernel/descriptor'
  import { approvalsDescriptor } from './approvals.descriptor'
  import {
    WorkViewModel,
    TASK_STATUS_TONES,
    PROJECT_STATUS_TONES,
    taskDueTone,
    type WorkTab,
  } from './work-vm.svelte'
  import type { Project, ProjectMember, TaskItem } from '../bridge/work'

  const vm = new WorkViewModel()
  onMount(() => void vm.load())

  const PRIORITY_LABEL: Record<string, string> = { low: 'Low', medium: 'Medium', high: 'High', urgent: 'Urgent' }

  function taskColumns(includeAssignee: boolean): ColumnSpec<TaskItem>[] {
    const cols: ColumnSpec<TaskItem>[] = [
      { key: 'title', label: 'Task', content: 'name', value: (t) => t.title, grow: true, minWidth: 220 },
      { key: 'project', label: 'Project', content: 'text', value: (t) => vm.projectName(t.projectId), minWidth: 160 },
    ]
    if (includeAssignee) {
      cols.push({ key: 'assignee', label: 'Assignee', content: 'name', value: (t) => t.assigneeName || 'Unassigned', minWidth: 150 })
    }
    cols.push(
      { key: 'priority', label: 'Priority', content: 'text', value: (t) => PRIORITY_LABEL[t.priority] || t.priority || '—', minWidth: 90 },
      { key: 'status', label: 'Status', content: 'status', value: (t) => t.status, minWidth: 110 },
      { key: 'due', label: 'Due', content: 'date', value: (t) => t.dueDate, tone: (t) => taskDueTone(t), minWidth: 110 },
    )
    return cols
  }

  const myWorkColumns = taskColumns(false)
  const teamBoardColumns = taskColumns(true)
  const projectTaskColumns = taskColumns(true)

  const taskStatus: StatusSpec<TaskItem> = { value: (t) => t.status, tones: TASK_STATUS_TONES }
  const projectStatus: StatusSpec<Project> = { value: (p) => p.status, tones: PROJECT_STATUS_TONES }
  const memberStatus: StatusSpec<ProjectMember> = {
    value: (m) => (m.isActive ? 'Active' : 'Inactive'),
    tones: { Active: 'success', Inactive: 'neutral' },
  }

  const projectColumns: ColumnSpec<Project>[] = [
    { key: 'name', label: 'Project', content: 'name', value: (p) => p.name, grow: true, minWidth: 200 },
    { key: 'type', label: 'Type', content: 'text', value: (p) => p.projectType || '—', minWidth: 90 },
    { key: 'tasks', label: 'Tasks', content: 'quantity', value: (p) => vm.projectTaskCounts[p.id] ?? 0, minWidth: 70 },
    { key: 'status', label: 'Status', content: 'status', value: (p) => p.status, minWidth: 110 },
  ]

  const memberColumns: ColumnSpec<ProjectMember>[] = [
    { key: 'employee', label: 'Employee', content: 'name', value: (m) => m.employeeName || m.employeeId, grow: true, minWidth: 180 },
    { key: 'role', label: 'Role', content: 'text', value: (m) => m.role || '—', minWidth: 120 },
    { key: 'allocation', label: 'Allocation', content: 'text', value: (m) => `${m.allocationPercent}%`, minWidth: 90 },
    { key: 'active', label: 'Membership', content: 'status', value: (m) => (m.isActive ? 'Active' : 'Inactive'), minWidth: 100 },
  ]

  const STATUS_QUICK_MARKS = ['todo', 'in_progress', 'completed']
</script>

{#snippet taskComposerForm()}
  <Card>
    <Stack gap="md">
      <span class="wh-section-label">New Task</span>
      <FormGrid columns={3}>
        <label class="k-field k-field-wide">
          <span class="k-field-label">Title</span>
          <input class="k-input" bind:value={vm.composerTitle} placeholder="What needs doing?" />
        </label>
        <label class="k-field">
          <span class="k-field-label">Priority</span>
          <select class="k-input" bind:value={vm.composerPriority}>
            <option value="low">Low</option>
            <option value="medium">Medium</option>
            <option value="high">High</option>
            <option value="urgent">Urgent</option>
          </select>
        </label>
        <label class="k-field">
          <span class="k-field-label">Due Date</span>
          <input class="k-input" type="date" bind:value={vm.composerDueDate} />
        </label>
        <label class="k-field">
          <span class="k-field-label">Project</span>
          <select class="k-input" bind:value={vm.composerProjectId}>
            <option value="">No project</option>
            {#each vm.projects as p (p.id)}
              <option value={p.id}>{p.name || '(untitled project)'}</option>
            {/each}
          </select>
        </label>
        <label class="k-field">
          <span class="k-field-label">Assignee</span>
          <select class="k-input" bind:value={vm.composerAssigneeId}>
            <option value="">{vm.currentEmployee?.employeeName || 'Me'}</option>
            {#each vm.employees as e (e.id)}
              <option value={e.id}>{e.name || '(no name on file)'}</option>
            {/each}
          </select>
        </label>
        <label class="k-field k-field-wide">
          <span class="k-field-label">Description</span>
          <textarea class="k-input k-input-area" rows="2" bind:value={vm.composerDescription}></textarea>
        </label>
      </FormGrid>
      {#if vm.composerError}
        <CalloutWidget items={[{ label: 'Create failed', text: vm.composerError, tone: 'danger' }]} />
      {/if}
      <Row justify="end">
        <Button variant="primary" onclick={() => vm.submitComposer()} disabled={vm.creatingTask}>
          {vm.creatingTask ? 'Creating…' : 'Add Task'}
        </Button>
      </Row>
    </Stack>
  </Card>
{/snippet}

{#snippet myWorkTab()}
  <Stack gap="lg">
    {@render taskComposerForm()}

    <Toolbar>
      <SearchInput bind:value={vm.mySearch} placeholder="Search my tasks…" />
      {#snippet trailing()}
        <Row gap="xs" align="center">
          <input type="checkbox" bind:checked={vm.myShowCompleted} />
          <span class="wh-meta">Show completed</span>
        </Row>
      {/snippet}
    </Toolbar>

    <Card padding="none">
      {#if vm.visibleMyTasks.length === 0}
        <EmptyState message={vm.myTasks.length === 0 ? 'No tasks assigned to you yet.' : 'Nothing matches the current search.'} />
      {:else}
        <DataTable
          columns={myWorkColumns}
          rows={vm.visibleMyTasks}
          id={(t) => t.id}
          status={taskStatus}
          selectedId={vm.selectedTaskId || null}
          onSelect={(t) => vm.openTask(t.id)}
        />
      {/if}
    </Card>
  </Stack>
{/snippet}

{#snippet teamBoardTab()}
  <Stack gap="lg">
    {#if vm.teamStatusDistribution.length}
      <Card>
        <Stack gap="sm">
          <span class="wh-section-label">Tasks by Status</span>
          <DistributionWidget segments={vm.teamStatusDistribution} />
        </Stack>
      </Card>
    {/if}

    <Toolbar>
      <SearchInput bind:value={vm.teamSearch} placeholder="Search team tasks…" />
      <FilterChips label="Status" options={vm.teamStatusOptions} bind:selected={vm.teamStatusFilter} />
      <FilterChips label="Assignee" options={vm.teamAssigneeOptions} bind:selected={vm.teamAssigneeFilter} />
      {#snippet trailing()}
        <Row gap="xs" align="center">
          <input type="checkbox" bind:checked={vm.teamShowCompleted} />
          <span class="wh-meta">Show completed</span>
        </Row>
      {/snippet}
    </Toolbar>

    <Card padding="none">
      {#if vm.visibleTeamTasks.length === 0}
        <EmptyState message={vm.teamTasks.length === 0 ? 'No team tasks yet.' : 'Nothing matches the current search and filters.'} />
      {:else}
        <DataTable
          columns={teamBoardColumns}
          rows={vm.visibleTeamTasks}
          id={(t) => t.id}
          status={taskStatus}
          selectedId={vm.selectedTaskId || null}
          onSelect={(t) => vm.openTask(t.id)}
        />
      {/if}
    </Card>
  </Stack>
{/snippet}

{#snippet projectsTab()}
  <Grid min="380px" gap="lg">
    <Stack gap="lg">
      <Card>
        <Stack gap="md">
          <span class="wh-section-label">New Project</span>
          <FormGrid columns={2}>
            <label class="k-field k-field-wide">
              <span class="k-field-label">Name</span>
              <input class="k-input" bind:value={vm.projectDraft.name} placeholder="Project name" />
            </label>
            <label class="k-field">
              <span class="k-field-label">Type</span>
              <select class="k-input" bind:value={vm.projectDraft.projectType}>
                <option value="internal">Internal</option>
                <option value="customer">Customer</option>
              </select>
            </label>
            <label class="k-field k-field-wide">
              <span class="k-field-label">Description</span>
              <textarea class="k-input k-input-area" rows="2" bind:value={vm.projectDraft.description}></textarea>
            </label>
            {#if vm.projectDraft.projectType === 'customer'}
              <label class="k-field">
                <span class="k-field-label">Customer</span>
                <input class="k-input" bind:value={vm.projectDraft.customerName} />
              </label>
              <label class="k-field">
                <span class="k-field-label">POC Name</span>
                <input class="k-input" bind:value={vm.projectDraft.customerPocName} />
              </label>
              <label class="k-field">
                <span class="k-field-label">POC Email</span>
                <input class="k-input" bind:value={vm.projectDraft.customerPocEmail} />
              </label>
              <label class="k-field">
                <span class="k-field-label">POC Phone</span>
                <input class="k-input" bind:value={vm.projectDraft.customerPocPhone} />
              </label>
            {/if}
          </FormGrid>
          {#if vm.projectComposerError}
            <CalloutWidget items={[{ label: 'Create failed', text: vm.projectComposerError, tone: 'danger' }]} />
          {/if}
          <Row justify="end">
            <Button variant="primary" onclick={() => vm.submitProjectComposer()} disabled={vm.savingProject}>
              {vm.savingProject ? 'Creating…' : 'Create Project'}
            </Button>
          </Row>
        </Stack>
      </Card>

      <Stack gap="sm">
        <Row justify="between" wrap>
          <span class="wh-section-label">Projects</span>
          <Button onclick={() => vm.toggleShowArchived()}>
            {vm.showArchivedProjects ? 'Show Active' : 'Show Archived'}
          </Button>
        </Row>
        <Card padding="none">
          {#if vm.loadingArchivedProjects}
            <EmptyState message="Loading archived projects…" />
          {:else if vm.visibleProjects.length === 0}
            <EmptyState message={vm.showArchivedProjects ? 'No archived, shelved, or deleted projects.' : 'No projects yet — create one above.'} />
          {:else}
            <DataTable
              columns={projectColumns}
              rows={vm.visibleProjects}
              id={(p) => p.id}
              status={projectStatus}
              selectedId={vm.selectedProjectId || null}
              onSelect={(p) => vm.selectProject(p.id)}
            />
          {/if}
        </Card>
      </Stack>
    </Stack>

    <Card>
      {#if !vm.selectedProject}
        <EmptyState message="Select a project to view its detail." />
      {:else}
        {@const p = vm.selectedProject}
        <Stack gap="lg">
          <Row justify="between" wrap>
            <Stack gap="xs">
              <span class="wh-project-title">{p.name || '—'}</span>
              <span class="wh-meta">{p.projectType === 'customer' ? p.customerName || 'Customer project' : 'Internal project'}</span>
            </Stack>
            <Badge tone={PROJECT_STATUS_TONES[p.status] ?? 'neutral'} label={p.status} />
          </Row>

          {#if p.description}
            <span class="wh-description">{p.description}</span>
          {/if}

          {#if p.projectType === 'customer'}
            <Stack gap="xs">
              <span class="wh-section-label">Customer Contact</span>
              <span class="wh-meta">{p.customerPocName || '—'} · {p.customerPocEmail || 'no email on file'} · {p.customerPocPhone || 'no phone on file'}</span>
            </Stack>
          {/if}

          <StatTileGrid
            sections={[
              {
                title: 'Project',
                items: [
                  { label: 'Open Tasks', value: vm.projectStats.open },
                  { label: 'Blocked', value: vm.projectStats.blocked, tone: vm.projectStats.blocked ? 'danger' : 'neutral' },
                  { label: 'Completed', value: vm.projectStats.completed },
                  { label: 'Members', value: vm.projectStats.members },
                ],
              },
            ]}
          />

          {#if vm.projectContextError}
            <CalloutWidget items={[{ label: 'Load failed', text: vm.projectContextError, tone: 'danger' }]} />
          {/if}

          {#if vm.editingProject}
            <Stack gap="md">
              <span class="wh-section-label">Edit Project</span>
              <FormGrid columns={2}>
                <label class="k-field k-field-wide">
                  <span class="k-field-label">Name</span>
                  <input class="k-input" bind:value={vm.projectEditDraft.name} />
                </label>
                <label class="k-field">
                  <span class="k-field-label">Type</span>
                  <select class="k-input" bind:value={vm.projectEditDraft.projectType}>
                    <option value="internal">Internal</option>
                    <option value="customer">Customer</option>
                  </select>
                </label>
                <label class="k-field k-field-wide">
                  <span class="k-field-label">Description</span>
                  <textarea class="k-input k-input-area" rows="2" bind:value={vm.projectEditDraft.description}></textarea>
                </label>
              </FormGrid>
              {#if vm.projectAdminError}
                <CalloutWidget items={[{ label: 'Save failed', text: vm.projectAdminError, tone: 'danger' }]} />
              {/if}
              <Row gap="sm" justify="end">
                <Button onclick={() => vm.cancelProjectEdit()}>Cancel</Button>
                <Button variant="primary" onclick={() => vm.saveProjectEdit()} disabled={vm.savingProjectAdmin}>
                  {vm.savingProjectAdmin ? 'Saving…' : 'Save Changes'}
                </Button>
              </Row>
            </Stack>
          {:else}
            <Stack gap="sm">
              <Row gap="sm" wrap>
                {#if vm.canManageProjects}
                  <Button onclick={() => vm.startProjectEdit()}>Edit</Button>
                {/if}
                {#if p.status === 'archived' || p.status === 'shelved' || p.status === 'deleted'}
                  {#if vm.canManageProjects}
                    <Button onclick={() => vm.restoreProject()} disabled={vm.savingProjectAdmin}>Restore to Active</Button>
                  {/if}
                {:else if vm.canManageProjects}
                  <Button onclick={() => vm.requestProjectAdmin('archive')}>Archive</Button>
                  <Button onclick={() => vm.requestProjectAdmin('shelve')}>Shelve</Button>
                {/if}
                {#if vm.canDeleteProject}
                  <Button variant="danger" onclick={() => vm.requestProjectAdmin('delete')}>Delete</Button>
                {/if}
              </Row>
              {#if vm.projectAdminError}
                <CalloutWidget items={[{ label: 'Action failed', text: vm.projectAdminError, tone: 'danger' }]} />
              {/if}
            </Stack>
          {/if}

          {@render taskComposerForm()}

          <Stack gap="sm">
            <span class="wh-section-label">Members</span>
            {#if vm.projectMembers.length === 0}
              <EmptyState message="No members on this project yet." />
            {:else}
              <Card padding="none">
                <DataTable
                  columns={memberColumns}
                  rows={vm.projectMembers}
                  id={(m) => m.employeeId}
                  status={memberStatus}
                  selectedId={vm.editingMemberId || null}
                  onSelect={(m) => vm.beginEditMember(m)}
                />
              </Card>
            {/if}

            {#if vm.editingMemberId}
              {@const editing = vm.projectMembers.find((m) => m.employeeId === vm.editingMemberId)}
              {#if editing}
                {@const draft = vm.memberDraft(editing)}
                <Card>
                  <Stack gap="sm">
                    <span class="wh-section-label">Edit {editing.employeeName || editing.employeeId}</span>
                    <FormGrid columns={2}>
                      <label class="k-field">
                        <span class="k-field-label">Role</span>
                        <input
                          class="k-input"
                          value={draft.role}
                          oninput={(e) => vm.setMemberDraftField(editing.employeeId, 'role', (e.currentTarget as HTMLInputElement).value)}
                        />
                      </label>
                      <label class="k-field">
                        <span class="k-field-label">Allocation %</span>
                        <input
                          class="k-input"
                          type="number"
                          value={draft.allocation}
                          oninput={(e) => vm.setMemberDraftField(editing.employeeId, 'allocation', Number((e.currentTarget as HTMLInputElement).value))}
                        />
                      </label>
                    </FormGrid>
                    {#if vm.memberEditError}
                      <CalloutWidget items={[{ label: 'Update failed', text: vm.memberEditError, tone: 'danger' }]} />
                    {/if}
                    <Row gap="sm" justify="end">
                      <Button onclick={() => vm.cancelEditMember()}>Cancel</Button>
                      <Button variant="primary" onclick={() => vm.requestSaveMember(editing)} disabled={vm.savingMemberId === editing.employeeId}>
                        {vm.savingMemberId === editing.employeeId ? 'Saving…' : 'Save Member'}
                      </Button>
                    </Row>
                  </Stack>
                </Card>
              {/if}
            {/if}

            {#if vm.canManageProjects}
              <Card>
                <Stack gap="sm">
                  <span class="wh-meta">Add Members</span>
                  {#if vm.availableEmployeesForMembers.length === 0}
                    <span class="wh-meta">Every active employee is already on this project.</span>
                  {:else}
                    <Stack gap="xs">
                      {#each vm.availableEmployeesForMembers as e (e.id)}
                        <label class="k-field k-field-row">
                          <input type="checkbox" checked={vm.memberSelections.includes(e.id)} onchange={() => vm.toggleMemberSelection(e.id)} />
                          <span class="wh-meta">{e.name || '(no name on file)'}{e.isActive ? '' : ' — inactive'}</span>
                        </label>
                      {/each}
                    </Stack>
                    <FormGrid columns={2}>
                      <label class="k-field">
                        <span class="k-field-label">Role</span>
                        <input class="k-input" bind:value={vm.newMemberRole} />
                      </label>
                      <label class="k-field">
                        <span class="k-field-label">Allocation %</span>
                        <input class="k-input" type="number" bind:value={vm.newMemberAllocation} />
                      </label>
                    </FormGrid>
                    {#if vm.memberAddError}
                      <CalloutWidget items={[{ label: 'Add failed', text: vm.memberAddError, tone: 'danger' }]} />
                    {/if}
                    <Row justify="end">
                      <Button variant="primary" onclick={() => vm.requestAddMembers()} disabled={vm.savingMembers || vm.memberSelections.length === 0}>
                        {vm.savingMembers ? 'Adding…' : `Add ${vm.memberSelections.length || ''} Member${vm.memberSelections.length === 1 ? '' : 's'}`}
                      </Button>
                    </Row>
                  {/if}
                </Stack>
              </Card>
            {/if}
          </Stack>

          <Stack gap="sm">
            <span class="wh-section-label">Tasks</span>
            <Card padding="none">
              {#if vm.projectTasks.length === 0}
                <EmptyState message="No tasks on this project yet." />
              {:else}
                <DataTable
                  columns={projectTaskColumns}
                  rows={vm.projectTasks}
                  id={(t) => t.id}
                  status={taskStatus}
                  selectedId={vm.selectedTaskId || null}
                  onSelect={(t) => vm.openTask(t.id)}
                />
              {/if}
            </Card>
          </Stack>

          <Stack gap="sm">
            <span class="wh-section-label">Recent Activity</span>
            <ActivityFeed
              items={vm.projectActivity.map((a) => ({ title: a.detail || a.activityType, subtitle: a.employeeName, timestamp: a.createdAt }))}
              emptyMessage="No activity recorded yet."
            />
          </Stack>
        </Stack>
      {/if}
    </Card>
  </Grid>
{/snippet}

{#snippet approvalsTab()}
  <DocumentLedger descriptor={approvalsDescriptor} embedded />
{/snippet}

<PageShell title="Work" subtitle="Projects, team workload, and task history in one collaborative workspace.">
  {#if vm.error}
    <EmptyState message={`Could not load the work hub: ${vm.error}`}>
      {#snippet actions()}
        <Button onclick={() => vm.load()}>Retry</Button>
      {/snippet}
    </EmptyState>
  {:else if vm.loading}
    <EmptyState message="Loading work hub…" />
  {:else}
    <TabShell
      activeKey={vm.activeTab}
      onSelect={(k) => (vm.activeTab = k as WorkTab)}
      tabs={[
        { key: 'my_work', label: 'My Work', ...(vm.myOpenCount ? { badge: vm.myOpenCount } : {}), content: myWorkTab },
        {
          key: 'team_board',
          label: 'Team Board',
          ...(vm.teamOpenCount ? { badge: vm.teamOpenCount } : {}),
          ...(vm.teamBlockedCount ? { badgeTone: 'danger' as const } : {}),
          content: teamBoardTab,
        },
        { key: 'projects', label: 'Projects', ...(vm.projects.length ? { badge: vm.projects.length } : {}), content: projectsTab },
        { key: 'approvals', label: 'Approvals', content: approvalsTab },
      ]}
    >
      {#snippet header()}
        {#if vm.currentEmployee}
          <Row justify="between" wrap>
            <Stack gap="xs">
              <span class="wh-identity-label">Active Employee</span>
              <span class="wh-identity-name">{vm.currentEmployee.employeeName || '—'}</span>
            </Stack>
            <StatTileGrid
              sections={[
                {
                  items: [
                    { label: 'My Open', value: vm.myOpenCount },
                    { label: 'Team Open', value: vm.teamOpenCount },
                    { label: 'Blocked', value: vm.teamBlockedCount, tone: vm.teamBlockedCount ? 'danger' : 'neutral' },
                    { label: 'Active Projects', value: vm.projects.length },
                  ],
                },
              ]}
            />
          </Row>
        {/if}
      {/snippet}
    </TabShell>
  {/if}
</PageShell>

{#if vm.taskModalOpen}
  <Modal title={vm.selectedTask?.title || 'Task'} onClose={() => vm.closeTaskModal()}>
    {#if vm.taskDetailLoading}
      <EmptyState message="Loading task…" />
    {:else if !vm.selectedTask}
      <EmptyState message={vm.taskDetailError ? `Could not load task: ${vm.taskDetailError}` : 'Task not found.'} />
    {:else}
      <Stack gap="lg">
        <FormGrid columns={2}>
          <label class="k-field k-field-wide">
            <span class="k-field-label">Title</span>
            <input class="k-input" bind:value={vm.draftTitle} />
          </label>
          <label class="k-field">
            <span class="k-field-label">Priority</span>
            <select class="k-input" bind:value={vm.draftPriority}>
              <option value="low">Low</option>
              <option value="medium">Medium</option>
              <option value="high">High</option>
              <option value="urgent">Urgent</option>
            </select>
          </label>
          <Stack gap="xs">
            <span class="k-field-label">Project</span>
            <span class="wh-meta">{vm.projectName(vm.selectedTask.projectId)}</span>
          </Stack>
          <label class="k-field k-field-wide">
            <span class="k-field-label">Description</span>
            <textarea class="k-input k-input-area" rows="3" bind:value={vm.draftDescription}></textarea>
          </label>
        </FormGrid>
        <Row justify="end">
          <Button variant="primary" onclick={() => vm.saveTaskDetails()} disabled={vm.savingTaskDetails}>
            {vm.savingTaskDetails ? 'Saving…' : 'Save Details'}
          </Button>
        </Row>

        <FormGrid columns={2}>
          <label class="k-field">
            <span class="k-field-label">Assignee</span>
            <select class="k-input" bind:value={vm.draftAssigneeId}>
              <option value="">Unassigned</option>
              {#each vm.employees as e (e.id)}
                <option value={e.id}>{e.name || '(no name on file)'}</option>
              {/each}
            </select>
          </label>
          <Row gap="sm" align="end">
            <Button onclick={() => vm.reassign()} disabled={vm.savingAssignment}>
              {vm.savingAssignment ? 'Saving…' : 'Reassign'}
            </Button>
          </Row>
          <label class="k-field">
            <span class="k-field-label">Due Date</span>
            <input class="k-input" type="date" bind:value={vm.draftDueDate} />
          </label>
          <Row gap="sm" align="end">
            <Button onclick={() => vm.saveDueDate()} disabled={vm.savingDueDate}>
              {vm.savingDueDate ? 'Saving…' : 'Update Due Date'}
            </Button>
          </Row>
        </FormGrid>

        <Row gap="sm" wrap align="center">
          <span class="wh-meta">Status</span>
          <Badge tone={TASK_STATUS_TONES[vm.selectedTask.status] ?? 'neutral'} label={vm.selectedTask.status} />
          {#each STATUS_QUICK_MARKS as s (s)}
            {#if vm.selectedTask.status !== s}
              <Button onclick={() => vm.setStatus(s)}>Mark {s.replace('_', ' ')}</Button>
            {/if}
          {/each}
        </Row>

        <Stack gap="xs">
          <span class="k-field-label">Blocked Reason</span>
          <textarea class="k-input k-input-area" rows="2" bind:value={vm.draftBlockedReason} placeholder="Why is this task stuck?"></textarea>
          <Row justify="end">
            <Button onclick={() => vm.block()}>Mark Blocked</Button>
          </Row>
        </Stack>

        {#if vm.taskDetailError}
          <CalloutWidget items={[{ label: 'Action failed', text: vm.taskDetailError, tone: 'danger' }]} />
        {/if}

        <Stack gap="sm">
          <span class="wh-section-label">Comments</span>
          <ActivityFeed
            items={vm.taskComments.map((c) => ({ title: c.body, subtitle: c.employeeName, timestamp: c.createdAt }))}
            emptyMessage="No comments yet."
          />
          <Row gap="sm">
            <input class="k-input k-grow" bind:value={vm.commentDraft} placeholder="Add a comment…" />
            <Button onclick={() => vm.addComment()} disabled={vm.savingComment || !vm.commentDraft.trim()}>
              {vm.savingComment ? 'Posting…' : 'Post'}
            </Button>
          </Row>
        </Stack>

        <Stack gap="sm">
          <span class="wh-section-label">Activity</span>
          <ActivityFeed
            items={vm.taskActivity.map((a) => ({ title: a.detail || a.activityType, subtitle: a.employeeName, timestamp: a.createdAt }))}
            emptyMessage="No activity recorded."
          />
        </Stack>

        <Row justify="end">
          <Button variant="danger" onclick={() => vm.requestDeleteTask()}>Delete Task</Button>
        </Row>
      </Stack>
    {/if}
  </Modal>
{/if}

{#if vm.taskDeleteConfirmOpen}
  <ConfirmDialog
    title="Delete task?"
    message={`Delete "${vm.selectedTask?.title || 'this task'}"? This removes it from the work board and cannot be undone.`}
    confirmLabel="Delete Task"
    danger
    onConfirm={() => vm.confirmDeleteTask()}
    onCancel={() => vm.cancelDeleteTask()}
  />
{/if}

{#if vm.projectAdminConfirm}
  {@const action = vm.projectAdminConfirm.action}
  <ConfirmDialog
    title={action === 'archive' ? 'Archive project?' : action === 'shelve' ? 'Shelve project?' : 'Delete project?'}
    message={action === 'delete'
      ? `Delete "${vm.selectedProject?.name || 'this project'}"? This cannot be undone. It has ${vm.projectTasks.length} task(s) and ${vm.projectMembers.length} member(s) attached — they stay linked to the deleted project's history.`
      : action === 'archive'
        ? `Archive "${vm.selectedProject?.name || 'this project'}"? This closes a finished project.`
        : `Shelve "${vm.selectedProject?.name || 'this project'}"? It stays findable under Archived and can be restored later.`}
    confirmLabel={action === 'archive' ? 'Archive' : action === 'shelve' ? 'Shelve' : 'Delete Project'}
    danger={action === 'delete'}
    reasonLabel="Reason (recorded in the audit trail)"
    requireReason
    onConfirm={(reason) => vm.confirmProjectAdmin(reason || '')}
    onCancel={() => vm.cancelProjectAdmin()}
  />
{/if}

{#if vm.memberAddWarning}
  <ConfirmDialog
    title="Over 100% allocation"
    message={`Adding ${vm.newMemberAllocation}% here would push these members over 100% total allocation: ${vm.memberAddWarning.join(', ')}. Save anyway?`}
    confirmLabel="Save Anyway"
    danger={false}
    onConfirm={() => vm.confirmMemberAddOverAllocation()}
    onCancel={() => vm.cancelMemberAddOverAllocation()}
  />
{/if}

{#if vm.memberEditWarning}
  <ConfirmDialog
    title="Over 100% allocation"
    message={vm.memberEditWarning}
    confirmLabel="Save Anyway"
    danger={false}
    onConfirm={() => vm.confirmMemberEditOverAllocation()}
    onCancel={() => vm.cancelMemberEditOverAllocation()}
  />
{/if}

<style>
  /* Typography only (L1) — layout/spacing lives in primitives; native form
   * controls use the kernel-owned .k-field/.k-field-label/.k-input classes
   * (single-source in styles/kernel.css), not per-screen CSS. */
  .wh-identity-label,
  .wh-meta {
    font-size: var(--meta-size);
    color: var(--text-secondary);
    overflow-wrap: break-word;
  }
  .wh-identity-name,
  .wh-project-title {
    font-family: var(--font-display);
    font-size: var(--section-title-size);
    font-weight: var(--section-title-weight);
    overflow-wrap: break-word;
  }
  .wh-section-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-secondary);
  }
  .wh-description {
    color: var(--text-primary);
    font-size: calc(13px * var(--ui-font-scale));
    overflow-wrap: break-word;
  }
</style>
