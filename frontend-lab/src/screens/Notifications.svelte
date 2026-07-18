<script lang="ts">
  /* Notifications — bespoke-on-primitives (K4). Day-grouped feed merging
   * task-assignment notices with delete/employee-archive approval review
   * cards (same two request kinds Approvals Queue lists). Row display reuses
   * the ActivityFeed widget (dot/title/subtitle/timestamp, click = mark
   * read); review cards add an inline Approve/Reject action row on top —
   * Approve via ConfirmDialog, Reject via FormModal (reason capture),
   * mirroring the ActionHost escalation pattern. See
   * screens/parity/Notifications.parity.md. */
  import { onMount } from 'svelte'
  import PageShell from '$kernel/primitives/PageShell.svelte'
  import Stack from '$kernel/primitives/Stack.svelte'
  import Row from '$kernel/primitives/Row.svelte'
  import Card from '$kernel/primitives/Card.svelte'
  import Toolbar from '$kernel/primitives/Toolbar.svelte'
  import FilterChips from '$kernel/controls/FilterChips.svelte'
  import Badge from '$kernel/controls/Badge.svelte'
  import Button from '$kernel/controls/Button.svelte'
  import EmptyState from '$kernel/controls/EmptyState.svelte'
  import ConfirmDialog from '$kernel/controls/ConfirmDialog.svelte'
  import FormModal from '$kernel/archetypes/FormModal.svelte'
  import ActivityFeed from '$kernel/widgets/ActivityFeed.svelte'
  import type { ActivityItem, NavIntent } from '$kernel/hub'
  import type { FormSpec } from '$kernel/form'
  import type { Tone } from '$kernel/tones'
  import { NotificationsViewModel } from './notifications-vm.svelte'
  import type { NotificationRow } from '../bridge/notifications'

  const vm = new NotificationsViewModel()
  onMount(() => void vm.load())

  // FilterChips drives a single 'unread' toggle chip; anything else ('') is "All".
  let viewFilter = $state('')
  $effect(() => {
    vm.unreadOnly = viewFilter === 'unread'
  })

  const KIND_LABEL: Record<NotificationRow['kind'], string> = {
    task: 'Task',
    'delete-approval': 'Delete',
    'archive-approval': 'Archive',
  }
  const STATUS_TONES: Record<string, Tone> = {
    pending: 'warning',
    approved: 'success',
    rejected: 'danger',
  }

  let confirmApprove = $state<NotificationRow | null>(null)
  let rejectTarget = $state<NotificationRow | null>(null)

  const rejectForm: FormSpec<{ reason: string }> = {
    title: 'Reject Request',
    submitLabel: 'Reject',
    initial: () => ({ reason: '' }),
    fields: [
      {
        key: 'reason',
        label: 'Reason',
        kind: 'textarea',
        required: true,
        placeholder: 'Why is this request being rejected?',
      },
    ],
    submit: async (draft, row) => {
      await vm.reject(row as NotificationRow, draft.reason)
    },
  }

  function toActivityItem(row: NotificationRow): ActivityItem {
    return {
      title: row.title,
      subtitle: row.reason ? `${row.subtitle} — ${row.reason}` : row.subtitle,
      timestamp: row.time,
      tone: row.tone,
      // Only plain, unread task items are clickable — that click is this
      // screen's "mark read" gesture, routed through the nav-intent slot
      // ActivityFeed already exposes for row interaction.
      nav: row.kind === 'task' && !row.read ? { key: `notif:${row.id}` } : undefined,
    }
  }

  function onActivityNav(intent: NavIntent) {
    const id = intent.key.slice('notif:'.length)
    const row = vm.rows.find((r) => r.id === id)
    if (row) void vm.markRead(row)
  }
</script>

<PageShell
  title="Notifications"
  subtitle="Task assignments and approval requests, grouped by day."
>
  {#snippet toolbar()}
    <Toolbar>
      <FilterChips
        label="View"
        options={[{ value: 'unread', label: `Unread (${vm.unreadCount})` }]}
        bind:selected={viewFilter}
      />
    </Toolbar>
  {/snippet}

  {#if vm.loading}
    <EmptyState message="Loading notifications…" />
  {:else if vm.error}
    <EmptyState message={`Could not load notifications: ${vm.error}`}>
      {#snippet actions()}
        <Button onclick={() => vm.load()}>Retry</Button>
      {/snippet}
    </EmptyState>
  {:else if vm.grouped.length === 0}
    <EmptyState message={vm.unreadOnly ? 'No unread notifications.' : 'Nothing here yet.'} />
  {:else}
    <Stack gap="lg">
      {#each vm.grouped as group (group.date)}
        <Stack gap="sm">
          <Row justify="between">
            <span class="n-day-label">{group.label}</span>
            <Badge tone="neutral" label={`${group.items.length}`} />
          </Row>
          <Stack gap="xs">
            {#each group.items as row (row.id)}
              <Card padding="md">
                <Stack gap="sm">
                  <Row gap="sm" wrap>
                    <Badge tone={row.kind === 'task' ? 'info' : row.tone} label={KIND_LABEL[row.kind]} />
                    {#if row.reviewStatus}
                      <Badge tone={STATUS_TONES[row.reviewStatus] ?? 'neutral'} label={row.reviewStatus} />
                    {/if}
                    {#if !row.read}
                      <Badge tone="info" label="Unread" />
                    {/if}
                  </Row>
                  <ActivityFeed items={[toActivityItem(row)]} navigate={onActivityNav} />
                  {#if row.reviewStatus === 'pending'}
                    <Row gap="sm" justify="end">
                      <Button onclick={() => (rejectTarget = row)}>Reject</Button>
                      <Button variant="primary" onclick={() => (confirmApprove = row)}>Approve</Button>
                    </Row>
                  {/if}
                </Stack>
              </Card>
            {/each}
          </Stack>
        </Stack>
      {/each}
    </Stack>
  {/if}
</PageShell>

{#if confirmApprove}
  <ConfirmDialog
    title="Approve Request"
    message={`Approve this ${confirmApprove.kind === 'archive-approval' ? 'archive' : 'delete'} request — ${confirmApprove.title}?`}
    confirmLabel="Approve"
    danger={false}
    onConfirm={async () => {
      const row = confirmApprove!
      confirmApprove = null
      await vm.approve(row)
    }}
    onCancel={() => (confirmApprove = null)}
  />
{/if}

{#if rejectTarget}
  <FormModal
    spec={rejectForm}
    row={rejectTarget}
    onDone={() => (rejectTarget = null)}
    onCancel={() => (rejectTarget = null)}
  />
{/if}

<style>
  /* Typography only (L1) — no layout/spacing rules here, that's Row/Stack's
   * job. Mirrors StatTileGrid's k-stat-title section-label treatment. */
  .n-day-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-secondary);
  }
</style>
