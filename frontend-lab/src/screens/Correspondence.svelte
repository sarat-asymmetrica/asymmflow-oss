<script lang="ts">
  /* Correspondence — the messenger's flat-inbox + thread console (Mission U2,
   * MESSENGER_UI_CAMPAIGN.md §2). Master-detail: LEFT a flat inbox sorted
   * expectation-first then recency; RIGHT the open room's thread (REPL-first
   * via ActivityFeed — message cards land at W-UI-2) plus a plain-input
   * composer with an expectation chip row and `/`-command support. Anchored
   * rooms (work) get an anchor chip + claim/release; social rooms/DMs render
   * with a lock glyph and NO claim UI at all (Design Constitution Art. II —
   * social-room authority belongs to its participants, never the org). All
   * state/derivation/mutation-calls live in correspondence-vm.svelte.ts (L5);
   * this file only composes primitives and renders (L1).
   *
   * LAW (Constitution Art. III/IV, binding on every render path here): never
   * render read/delivered/seen state, no unread badges, no typing indicators,
   * no sounds, no popups. The expectation tag — volunteered by the sender —
   * is the only attention signal, shown on the message and the inbox row. */
  import { onMount } from 'svelte'
  import PageShell from '$kernel/primitives/PageShell.svelte'
  import Grid from '$kernel/primitives/Grid.svelte'
  import Card from '$kernel/primitives/Card.svelte'
  import Stack from '$kernel/primitives/Stack.svelte'
  import Row from '$kernel/primitives/Row.svelte'
  import Toolbar from '$kernel/primitives/Toolbar.svelte'
  import ActivityFeed from '$kernel/widgets/ActivityFeed.svelte'
  import CalloutWidget from '$kernel/widgets/CalloutWidget.svelte'
  import Badge from '$kernel/controls/Badge.svelte'
  import Button from '$kernel/controls/Button.svelte'
  import EmptyState from '$kernel/controls/EmptyState.svelte'
  import SearchInput from '$kernel/controls/SearchInput.svelte'
  import FilterChips from '$kernel/controls/FilterChips.svelte'
  import type { Tone } from '$kernel/tones'
  import { formatDate } from '$kernel/format'
  import { CorrespondenceViewModel, EXPECTATION_OPTIONS } from './correspondence-vm.svelte'
  import type { ExpectationTag, RoomSummary } from '../bridge/mesh'

  const vm = new CorrespondenceViewModel()
  onMount(() => void vm.load())

  const TAG_TONE: Record<ExpectationTag, Tone> = { urgent: 'danger', today: 'warning', whenever: 'info', '': 'neutral' }

  function tagLabel(tag: ExpectationTag): string {
    return EXPECTATION_OPTIONS.find((o) => o.value === tag)?.label ?? tag
  }

  function rowClass(r: RoomSummary): string {
    const classes = ['co-row']
    if (r.roomKey === vm.selectedRoomKey) classes.push('co-row-selected')
    else if (r.topExpectation === 'urgent') classes.push('co-row-urgent')
    else if (r.topExpectation === 'today') classes.push('co-row-today')
    return classes.join(' ')
  }

  function messageTime(iso: string): string {
    if (!iso) return ''
    return `${formatDate(iso)} ${iso.slice(11, 16)}`
  }

  function onComposerKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      void vm.submit()
    }
  }
</script>

<PageShell title="Correspondence" subtitle="Every business object can hold a conversation — the RFQ pipeline, the open POs, the shipments in transit, and the people you work with, in one inbox.">
  {#if vm.error}
    <EmptyState message={`Could not load correspondence: ${vm.error}`}>
      {#snippet actions()}
        <Button onclick={() => vm.load()}>Retry</Button>
      {/snippet}
    </EmptyState>
  {:else if vm.loading}
    <EmptyState message="Loading correspondence…" />
  {:else}
    <Grid min="320px" gap="lg">
      <Card padding="none">
        <Stack gap="none">
          <div class="co-inbox-head">
            <Stack gap="sm">
              <SearchInput bind:value={vm.search} placeholder="Search correspondence…" />
              <FilterChips label="Show" options={vm.filterOptions} bind:selected={vm.filter} />
            </Stack>
          </div>

          {#if vm.visibleRooms.length === 0}
            <EmptyState message="Nothing matches." />
          {:else}
            <div class="co-inbox-list">
              {#each vm.visibleRooms as r (r.roomKey)}
                <button type="button" class={rowClass(r)} onclick={() => vm.selectRoom(r.roomKey)}>
                  <Row justify="between" align="start" gap="sm">
                    <Stack gap="xs">
                      <Row gap="xs" align="center">
                        {#if r.kind === 'social'}<span class="co-lock" aria-hidden="true">🔒</span>{/if}
                        <span class="co-title">{r.title}</span>
                      </Row>
                      <span class="co-preview">{r.lastPreview || 'No messages yet.'}</span>
                    </Stack>
                    {#if r.topExpectation}
                      <Badge tone={TAG_TONE[r.topExpectation]} label={tagLabel(r.topExpectation)} />
                    {/if}
                  </Row>
                </button>
              {/each}
            </div>
          {/if}
        </Stack>
      </Card>

      <Card>
        {#if !vm.selectedRoomSummary}
          <EmptyState message="Select a conversation to see its thread." />
        {:else}
          {@const room = vm.selectedRoomSummary}
          <Stack gap="lg">
            <Row justify="between" wrap>
              <Stack gap="xs">
                <Row gap="xs" align="center">
                  {#if room.kind === 'social'}<span class="co-lock" aria-hidden="true">🔒</span>{/if}
                  <span class="co-thread-title">{room.title}</span>
                </Row>
                <Row gap="xs" align="center" wrap>
                  {#if room.kind === 'anchored'}
                    <Badge tone="neutral" label={`${room.anchorType} · ${room.anchorId}`} />
                  {/if}
                  {#if vm.thread}
                    <span class="co-meta">{vm.thread.members.length} member{vm.thread.members.length === 1 ? '' : 's'}</span>
                  {/if}
                </Row>
              </Stack>

              {#if room.kind === 'anchored' && vm.thread}
                {#if vm.thread.claim}
                  <Row gap="xs" align="center" shrink={false}>
                    <span class="co-meta">Claimed by {vm.thread.claim.assignee}</span>
                    <Button onclick={() => vm.release()}>Release</Button>
                  </Row>
                {:else}
                  <Row gap="xs" align="center" shrink={false}>
                    <span class="co-meta">Unclaimed</span>
                    <Button onclick={() => vm.claim()}>Claim</Button>
                  </Row>
                {/if}
              {/if}
            </Row>

            {#if vm.thread && (vm.thread.skippedCount > 0 || vm.thread.rejectedCount > 0)}
              <span class="co-meta">{vm.thread.skippedCount} skipped · {vm.thread.rejectedCount} rejected</span>
            {/if}

            {#if vm.threadLoading}
              <EmptyState message="Loading thread…" />
            {:else if vm.thread}
              <ActivityFeed
                items={vm.thread.messages.map((m) => ({
                  title: m.body,
                  subtitle: `${m.actor}${m.expectation ? ' · ' + tagLabel(m.expectation) : ''}${m.attachment ? ' · 📎 ' + m.attachment.name : ''}`,
                  timestamp: messageTime(m.ts),
                }))}
                emptyMessage="No messages yet — say hello."
              />
            {/if}

            {#if vm.threadError}
              <CalloutWidget items={[{ label: 'Correspondence', text: vm.threadError, tone: 'warning' }]} />
            {/if}

            {#if vm.commandNotice}
              <span class="co-meta">{vm.commandNotice}</span>
            {/if}

            {#if vm.inviteCode}
              <label class="k-field">
                <span class="k-field-label">Invite code</span>
                <input
                  class="k-input"
                  readonly
                  value={vm.inviteCode}
                  onclick={(e) => (e.currentTarget as HTMLInputElement).select()}
                />
              </label>
            {/if}

            <Stack gap="sm">
              <Row gap="xs" wrap>
                {#each EXPECTATION_OPTIONS as opt (opt.value)}
                  <Button
                    variant={vm.composerExpectation === opt.value ? 'primary' : 'ghost'}
                    onclick={() => vm.setExpectation(opt.value)}
                  >{opt.label}</Button>
                {/each}
              </Row>

              <Row gap="sm" align="center">
                <input
                  class="k-input k-grow"
                  type="text"
                  placeholder="Message, or /expect, /claim, /release, /invite, /attach…"
                  bind:value={vm.composerText}
                  onkeydown={onComposerKeydown}
                  disabled={vm.posting}
                />
                <Button onclick={() => vm.toggleAttach()}>{vm.attaching ? 'Cancel Attach' : 'Attach'}</Button>
                <Button variant="primary" disabled={vm.posting || !vm.composerText.trim()} onclick={() => vm.submit()}>
                  {vm.posting ? 'Sending…' : 'Send'}
                </Button>
              </Row>

              {#if vm.attaching}
                <Row gap="sm" align="center">
                  <input
                    class="k-input k-grow"
                    type="text"
                    placeholder="Mock file pick — type a file name (e.g. Costing_Sheet_v4.xlsx)"
                    bind:value={vm.attachFileName}
                    disabled={vm.posting}
                  />
                  <Button variant="primary" disabled={vm.posting || !vm.attachFileName.trim()} onclick={() => vm.sendAttachment()}>
                    Send Attachment
                  </Button>
                </Row>
              {/if}
            </Stack>
          </Stack>
        {/if}
      </Card>
    </Grid>
  {/if}
</PageShell>

<style>
  /* Typography + skin only (L1) — layout lives in Grid/Row/Stack; the inbox
   * row's urgency/tint treatment is border-left/background skin on a token,
   * not structural CSS. */
  .co-inbox-head {
    padding: var(--card-padding);
  }
  .co-row {
    width: 100%;
    text-align: start;
    font-family: inherit;
    color: inherit;
    background: none;
    border: none;
    border-left: 3px solid transparent;
    border-radius: 0;
    padding: var(--k-space-sm) var(--card-padding);
    cursor: pointer;
    transition: background var(--motion-fast) var(--ease-standard);
  }
  .co-row:hover {
    background: var(--onyx-tint);
  }
  .co-row-selected {
    background: var(--onyx-tint);
  }
  .co-row-urgent {
    border-left-color: var(--k-tone-danger-fg);
  }
  .co-row-today {
    background: var(--k-tone-warning-bg);
  }
  .co-lock {
    color: var(--text-secondary);
  }
  .co-title {
    font-weight: 600;
    overflow-wrap: break-word;
  }
  .co-preview {
    font-size: var(--meta-size);
    color: var(--text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .co-thread-title {
    font-family: var(--font-display);
    font-size: var(--section-title-size);
    font-weight: var(--section-title-weight);
    overflow-wrap: break-word;
  }
  .co-meta {
    font-size: var(--meta-size);
    color: var(--text-secondary);
    overflow-wrap: break-word;
  }
</style>
