<script lang="ts">
  /* Butler — bespoke-on-primitives (K4). Two-pane console: a conversation
   * sidebar (new/clear-all/quick actions/list with delete-arm) + a chat panel
   * (ChatTranscript + input bar) via Grid, per Butler build spec. All
   * arm/confirm/validation/dispatch logic lives in butler-vm.svelte.ts /
   * butler-actions.ts (L5); this file only composes primitives and renders
   * (L1). RETIRES IntelligenceHub — nav wiring is orchestrator-owned (K5).
   *
   * Known kernel gap (deferred to K5 app shell — see Butler.parity.md): no
   * primitive threads a bounded "fill remaining page height" down to a
   * scrollable child, so ChatTranscript's own internal auto-scroll can't
   * engage — the page scrolls instead. This is the viewport-height-chain the
   * real app shell owns at K5; the chat is fully functional meanwhile.
   * (The related bare-input-in-a-Row gap was fixed at the kernel: the chat
   * input uses `k-input k-grow` — `k-grow` is a kernel flex-grow utility so the
   * input absorbs the row beside the fixed Send button, no screen layout CSS.) */
  import { onDestroy, onMount } from 'svelte'
  import PageShell from '$kernel/primitives/PageShell.svelte'
  import Grid from '$kernel/primitives/Grid.svelte'
  import Card from '$kernel/primitives/Card.svelte'
  import Stack from '$kernel/primitives/Stack.svelte'
  import Row from '$kernel/primitives/Row.svelte'
  import ChatTranscript from '$kernel/primitives/ChatTranscript.svelte'
  import Button from '$kernel/controls/Button.svelte'
  import Badge from '$kernel/controls/Badge.svelte'
  import EmptyState from '$kernel/controls/EmptyState.svelte'
  import { ButlerViewModel } from './butler-vm.svelte'

  const vm = new ButlerViewModel()
  onMount(() => void vm.load())
  onDestroy(() => vm.dispose())

  function onInputKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') void vm.send()
  }
</script>

<PageShell title="Butler" subtitle="Ask about customers, suppliers, financials, and operations — or run a guided action.">
  {#if vm.error}
    <EmptyState message={`Could not load conversations: ${vm.error}`}>
      {#snippet actions()}
        <Button onclick={() => vm.load()}>Retry</Button>
      {/snippet}
    </EmptyState>
  {:else if vm.loading}
    <EmptyState message="Loading Butler…" />
  {:else}
    <Grid min="280px" gap="lg">
      <Card>
        <Stack gap="md">
          <Row gap="xs" wrap>
            <Button variant="primary" onclick={() => vm.startNewConversation()}>New Chat</Button>
            <Button variant="danger" onclick={() => vm.armClearAll()}>
              {vm.pendingClearAll ? 'Click again to confirm' : 'Clear All Chats'}
            </Button>
          </Row>

          <Stack gap="xs">
            <span class="b-section-label">Quick actions</span>
            <Row gap="xs" wrap>
              <Button disabled={vm.sending} onclick={() => vm.runWorkflowAction('create_offer_draft')}>Create offer draft</Button>
              <Button disabled={vm.sending} onclick={() => vm.runWorkflowAction('daily_briefing')}>Daily briefing</Button>
            </Row>
          </Stack>

          <Stack gap="xs">
            <span class="b-section-label">Conversations</span>
            {#if vm.conversations.length === 0}
              <span class="b-meta">No conversations yet.</span>
            {:else}
              <Stack gap="xs">
                {#each vm.conversations as conv (conv.id)}
                  <Row gap="xs" justify="between" align="center">
                    <Button
                      variant={vm.activeConversationId === conv.id ? 'primary' : 'ghost'}
                      onclick={() => vm.selectConversation(conv.id)}
                    >
                      {vm.conversationLabel(conv)}
                    </Button>
                    <Row gap="xs" align="center" shrink={false}>
                      <Badge tone={conv.isActive ? 'success' : 'neutral'} label={conv.isActive ? 'Active' : 'Idle'} />
                      <Button variant="danger" onclick={() => vm.armDeleteConversation(conv)}>
                        {vm.pendingDeleteKey === conv.id ? 'Confirm' : '×'}
                      </Button>
                    </Row>
                  </Row>
                {/each}
              </Stack>
            {/if}
          </Stack>
        </Stack>
      </Card>

      <Card>
        <Stack gap="md">
          {#if vm.activeConversation}
            {@const activeConv = vm.activeConversation}
            <Stack gap="xs">
              <span class="b-conv-title">{vm.conversationLabel(activeConv)}</span>
              {#if activeConv.summary}
                <span class="b-meta">{activeConv.summary}</span>
              {/if}
            </Stack>
          {/if}

          <ChatTranscript
            messages={vm.messages}
            armedChipId={vm.armedKey}
            onChipClick={(msgId, chip) => void vm.handleChipClick(msgId, chip)}
            loading={vm.sending || vm.loadingConversation}
          />

          <Row gap="sm" align="center">
            <input
              class="k-input k-grow"
              type="text"
              placeholder="Ask about financials, customers, suppliers…"
              bind:value={vm.userInput}
              onkeydown={onInputKeydown}
              disabled={vm.sending}
            />
            <Button variant="primary" disabled={vm.sending || !vm.userInput.trim()} onclick={() => vm.send()}>Send</Button>
          </Row>
        </Stack>
      </Card>
    </Grid>
  {/if}
</PageShell>

<style>
  /* Typography only (L1) — no layout/spacing rules here, that's Grid/Row/
   * Stack's job. Mirrors BookBankRecon.svelte's bbr-account / bbr-meta. */
  .b-section-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-secondary);
  }
  .b-conv-title {
    font-family: var(--font-display);
    font-size: var(--section-title-size);
    font-weight: var(--section-title-weight);
    overflow-wrap: break-word;
  }
  .b-meta {
    font-size: var(--meta-size);
    color: var(--text-secondary);
    overflow-wrap: break-word;
  }
</style>
