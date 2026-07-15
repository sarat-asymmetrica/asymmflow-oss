<script lang="ts">
  /* ChatTranscript — the role-bubble conversation feed for the Butler AI chat.
   * A CONTROLLED primitive: it renders `messages` + an `armedChipId`, and
   * forwards every action-chip click to `onChipClick`. It owns NO arm/confirm
   * state — the consequential-action gate (single global arm slot + timeout)
   * lives in the viewmodel, because the AI-authority boundary (an armed write
   * action must be human-confirmed) is business logic, not rendering. The
   * primitive only needs to know which chip is currently armed (to flip its
   * label to Confirm) and to forward clicks. Assistant text renders through the
   * escape-first kernel markdown renderer (untrusted LLM output → {@html}).
   * Owns its layout (L1). */
  import { renderMarkdown as defaultRender, escapeHtml } from '../markdown'
  import type { Tone } from '../tones'

  export type ChatRole = 'user' | 'assistant'

  export interface ChatActionChip {
    id: string
    label: string
    /** Chip tone for the ready/needs-approval/etc. visual state. */
    tone?: Tone
    /** Human-readable state label (e.g. "Ready", "Needs input"). */
    statusLabel?: string
    /** Not clickable (needs_input / invalid_payload). */
    disabled?: boolean
  }

  export interface ChatMessage {
    id: string
    role: ChatRole
    text: string
    actionChips?: ChatActionChip[]
    /** True = render a typing/pending bubble instead of text. */
    pending?: boolean
  }

  let {
    messages,
    armedChipId = null,
    onChipClick,
    renderMarkdown = defaultRender,
    loading = false,
    emptyMessage = 'Ask the Butler anything to get started.',
  }: {
    messages: ChatMessage[]
    armedChipId?: string | null
    onChipClick: (messageId: string, chip: ChatActionChip) => void
    renderMarkdown?: (text: string) => string
    loading?: boolean
    emptyMessage?: string
  } = $props()

  let feed = $state<HTMLDivElement | null>(null)

  // Autoscroll to the newest message as the feed grows / a pending bubble appears.
  $effect(() => {
    // touch reactive deps so the effect re-runs on new content
    void messages.length
    void loading
    if (feed) feed.scrollTop = feed.scrollHeight
  })
</script>

<div class="k-chat" bind:this={feed}>
  {#if messages.length === 0 && !loading}
    <div class="k-chat-empty">{emptyMessage}</div>
  {/if}

  {#each messages as msg (msg.id)}
    <div class="k-chat-turn k-chat-{msg.role}">
      <div class="k-chat-bubble">
        {#if msg.pending}
          <span class="k-chat-typing" aria-label="Butler is typing">
            <span class="k-chat-dot"></span><span class="k-chat-dot"></span><span class="k-chat-dot"></span>
          </span>
        {:else if msg.role === 'assistant'}
          <!-- Untrusted LLM output — escape-first renderer guarantees no live markup. -->
          <div class="k-chat-md">{@html renderMarkdown(msg.text)}</div>
        {:else}
          <div class="k-chat-user-text">{@html escapeHtml(msg.text)}</div>
        {/if}
      </div>

      {#if msg.actionChips?.length}
        <div class="k-chat-chips">
          {#each msg.actionChips as chip (chip.id)}
            {@const armed = armedChipId === chip.id}
            <button
              class="k-chat-chip"
              class:armed
              disabled={chip.disabled}
              style:background={armed
                ? 'var(--k-tone-info-bg)'
                : chip.tone
                  ? `var(--k-tone-${chip.tone}-bg)`
                  : undefined}
              style:color={armed
                ? 'var(--k-tone-info-fg)'
                : chip.tone
                  ? `var(--k-tone-${chip.tone}-fg)`
                  : undefined}
              onclick={() => onChipClick(msg.id, chip)}
            >
              <span class="k-chat-chip-label">{armed ? 'Confirm' : chip.label}</span>
              {#if !armed && chip.statusLabel}
                <span class="k-chat-chip-status">{chip.statusLabel}</span>
              {/if}
            </button>
          {/each}
        </div>
      {/if}
    </div>
  {/each}

  {#if loading}
    <div class="k-chat-turn k-chat-assistant">
      <div class="k-chat-bubble">
        <span class="k-chat-typing" aria-label="Butler is typing">
          <span class="k-chat-dot"></span><span class="k-chat-dot"></span><span class="k-chat-dot"></span>
        </span>
      </div>
    </div>
  {/if}
</div>

<style>
  .k-chat {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-md);
    min-width: 0;
    overflow-y: auto;
    height: 100%;
    padding: var(--k-space-xs);
  }
  .k-chat-empty {
    font-size: calc(13px * var(--ui-font-scale));
    color: var(--text-secondary);
    text-align: center;
    padding: var(--k-space-xl) 0;
  }
  .k-chat-turn {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-xs);
    min-width: 0;
    max-width: 92%;
  }
  .k-chat-user {
    align-self: flex-end;
    align-items: flex-end;
  }
  .k-chat-assistant {
    align-self: flex-start;
    align-items: flex-start;
  }
  .k-chat-bubble {
    min-width: 0;
    max-width: 100%;
    padding: var(--k-space-sm) var(--k-space-md);
    border-radius: var(--border-radius);
    font-size: calc(13px * var(--ui-font-scale));
    line-height: 1.5;
    overflow-wrap: anywhere;
  }
  .k-chat-user .k-chat-bubble {
    background: var(--onyx);
    color: var(--surface);
  }
  .k-chat-assistant .k-chat-bubble {
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    color: var(--text-primary);
  }
  .k-chat-user-text {
    white-space: pre-wrap;
  }
  /* Scoped markdown typography for assistant bubbles. :global because the HTML
   * is injected via {@html} (not compiled by Svelte, so scoped classes don't
   * attach) — namespaced under .k-chat-md so it never leaks. */
  .k-chat-md :global(.k-md-h) {
    margin: var(--k-space-sm) 0 var(--k-space-xs);
    font-size: calc(13px * var(--ui-font-scale));
    font-weight: 700;
  }
  .k-chat-md :global(.k-md-p) {
    margin: 0 0 var(--k-space-xs);
  }
  .k-chat-md :global(.k-md-ul),
  .k-chat-md :global(.k-md-ol) {
    margin: 0 0 var(--k-space-xs);
    padding-left: var(--k-space-lg);
  }
  .k-chat-md :global(code) {
    font-family: var(--font-numeric);
    font-size: 0.92em;
    background: var(--onyx-tint);
    padding: 1px 4px;
    border-radius: var(--border-radius-sm);
  }
  .k-chat-md :global(.k-md-table) {
    border-collapse: collapse;
    margin: var(--k-space-xs) 0;
    font-size: calc(12px * var(--ui-font-scale));
    display: block;
    overflow-x: auto;
    max-width: 100%;
  }
  .k-chat-md :global(.k-md-table th),
  .k-chat-md :global(.k-md-table td) {
    border: var(--border-width) solid var(--border);
    padding: 4px 8px;
    text-align: left;
    white-space: nowrap;
  }
  .k-chat-chips {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-xs);
    min-width: 0;
  }
  .k-chat-chip {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-xs);
    font-family: var(--font-ui);
    font-size: calc(12px * var(--ui-font-scale));
    font-weight: 600;
    padding: 5px 12px;
    border-radius: var(--border-radius-pill);
    border: var(--border-width) solid var(--border);
    background: var(--onyx-tint);
    color: var(--text-secondary);
    cursor: pointer;
    transition: border-color var(--motion-fast) var(--ease-standard);
  }
  .k-chat-chip:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .k-chat-chip.armed {
    border-color: var(--k-tone-info-fg);
  }
  .k-chat-chip-status {
    font-weight: 500;
    opacity: 0.8;
  }
  .k-chat-typing {
    display: inline-flex;
    gap: 4px;
    align-items: center;
  }
  .k-chat-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--text-muted);
    animation: k-chat-pulse 1.2s infinite ease-in-out;
  }
  .k-chat-dot:nth-child(2) {
    animation-delay: 0.2s;
  }
  .k-chat-dot:nth-child(3) {
    animation-delay: 0.4s;
  }
  @keyframes k-chat-pulse {
    0%,
    60%,
    100% {
      opacity: 0.3;
    }
    30% {
      opacity: 1;
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .k-chat-dot {
      animation: none;
    }
  }
</style>
