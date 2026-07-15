/* Butler viewmodel — L5's reactive half: conversation list/select/new/delete,
 * the chat transcript + send, and the arm/confirm/6s-timeout single-global-arm
 * gate for write-action chips. All domain logic (alias normalization, payload
 * validation, binding resolution) lives in butler-actions.ts (pure, no runes);
 * this file is state + orchestration only — same split as book-bank-recon.
 *
 * Named `butler-vm` (not `butler.svelte.ts`) so its stem never differs from
 * `Butler.svelte` by case only — collides under case-insensitive resolution
 * on Windows (see pricing-vm.svelte.ts's identical note). */

import type { Tone } from '$kernel/tones'
import {
  deleteButlerConversation,
  fetchButlerConversationMessages,
  fetchButlerConversations,
  purgeAllButlerConversations,
  sendButlerMessage,
  type ButlerActionPayload,
  type ButlerChatMessageRow,
  type ButlerConversationRow,
} from '../bridge/butler'
import {
  ACTION_CONFIRMATION_WINDOW_MS,
  actionExecutionKey,
  buildActionPreview,
  buildWorkflowPrompt,
  convTitle,
  describeAction,
  executeButlerAction,
  getActionDataObject,
  getActionWorkflowKey,
  hydrateActionsFromMessage,
  isWriteAction,
  normalizeActionType,
  resolveActionTarget,
  type ResolvedButlerAction,
} from './butler-actions'

/* ---- structurally compatible with ChatTranscript.svelte's exported types —
 * declared locally rather than imported cross-.svelte-file (its exports live
 * in the instance script, not a `<script module>` block, so they aren't a
 * reliable import target; TS structural typing makes this safe either way). ---- */
export type ButlerChatRole = 'user' | 'assistant'
export interface ButlerChatActionChip {
  id: string
  label: string
  tone?: Tone
  statusLabel?: string
  disabled?: boolean
}
export interface ButlerChatMessage {
  id: string
  role: ButlerChatRole
  text: string
  actionChips?: ButlerChatActionChip[]
  pending?: boolean
}

let localIdSeq = 0
function localId(): string {
  localIdSeq += 1
  return `local-${Date.now()}-${localIdSeq}`
}

function toResolvedAction(a: ButlerActionPayload): ResolvedButlerAction {
  return { type: a.type, target: a.target, label: a.label, data: a.data, requiresApproval: false, storedStatus: '', missingFields: [], invalidReason: '' }
}

export class ButlerViewModel {
  conversations = $state<ButlerConversationRow[]>([])
  loading = $state(true)
  error = $state<string | null>(null)

  activeConversationId = $state<string | null>(null)
  messages = $state<ButlerChatMessage[]>([])
  loadingConversation = $state(false)

  userInput = $state('')
  sending = $state(false)

  armedKey = $state<string | null>(null)
  private armTimer: ReturnType<typeof setTimeout> | null = null

  pendingDeleteKey = $state<string | null>(null)
  private pendingDeleteTimer: ReturnType<typeof setTimeout> | null = null
  pendingClearAll = $state(false)
  private pendingClearAllTimer: ReturnType<typeof setTimeout> | null = null

  /** Keyed by the SAME content-addressed key used for chip.id — many chips
   * (across messages, or the same message) can resolve to one entry. */
  private actionsByChipId = new Map<string, ResolvedButlerAction>()

  activeConversation = $derived.by(() => this.conversations.find((c) => c.id === this.activeConversationId) ?? null)

  async load(): Promise<void> {
    this.loading = true
    this.error = null
    try {
      this.conversations = await fetchButlerConversations()
      if (this.conversations.length > 0) {
        await this.selectConversation(this.conversations[0]!.id)
      } else {
        this.messages = [this.assistantMessage('Good day. I can answer questions about customers, suppliers, financials, and operations. I can also draft offers, log follow-ups, and prepare daily briefings on request.')]
      }
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
      this.conversations = []
    } finally {
      this.loading = false
    }
  }

  conversationLabel(c: ButlerConversationRow): string {
    return convTitle(c.title)
  }

  private chipFor(action: ResolvedButlerAction, seed: string): ButlerChatActionChip {
    const key = actionExecutionKey(action)
    this.actionsByChipId.set(key, action)
    const type = normalizeActionType(action.type)
    const described = describeAction(action)
    const dataStr = typeof action.data === 'string' ? action.data : ''
    const label = action.label.trim() || (dataStr || resolveActionTarget(action.target) || 'Action')
    const chip: ButlerChatActionChip = { id: key || `${seed}-noop`, label }
    chip.tone = described.tone
    chip.statusLabel = type === 'clarify' ? 'Choose' : described.statusLabel
    chip.disabled = described.disabled
    return chip
  }

  private mapRow(row: ButlerChatMessageRow): ButlerChatMessage {
    const role: ButlerChatRole = row.role === 'user' ? 'user' : 'assistant'
    const resolved = role === 'assistant' ? hydrateActionsFromMessage(row) : []
    const text = row.content.trim() === '' && resolved.length > 0 ? 'Action details available in this historical message.' : row.content
    const out: ButlerChatMessage = { id: row.id || localId(), role, text }
    if (resolved.length > 0) out.actionChips = resolved.map((a) => this.chipFor(a, row.id))
    return out
  }

  async selectConversation(id: string): Promise<void> {
    this.clearArm()
    this.activeConversationId = id
    this.loadingConversation = true
    this.actionsByChipId.clear()
    try {
      const rows = await fetchButlerConversationMessages(id)
      this.messages = rows.length > 0
        ? rows.map((r) => this.mapRow(r))
        : [this.assistantMessage('This conversation has no persisted messages yet.')]
    } catch (e) {
      this.messages = [this.assistantMessage(`Failed to load conversation history: ${e instanceof Error ? e.message : String(e)}`)]
    } finally {
      this.loadingConversation = false
    }
  }

  startNewConversation(): void {
    this.clearArm()
    this.activeConversationId = null
    this.actionsByChipId.clear()
    this.messages = [this.assistantMessage('Good day. How can I help you?')]
  }

  private assistantMessage(text: string, chips?: ButlerChatActionChip[]): ButlerChatMessage {
    const m: ButlerChatMessage = { id: localId(), role: 'assistant', text }
    if (chips && chips.length > 0) m.actionChips = chips
    return m
  }

  private pushAssistant(text: string): void {
    this.messages.push(this.assistantMessage(text))
  }

  async send(): Promise<void> {
    if (this.sending) return
    const text = this.userInput.trim()
    if (!text) return
    this.clearArm()
    this.userInput = ''
    this.messages.push({ id: localId(), role: 'user', text })
    this.sending = true
    try {
      const res = await sendButlerMessage(this.activeConversationId ?? '', text)
      if (res.conversationId && !this.activeConversationId) {
        this.activeConversationId = res.conversationId
        void this.refreshConversationList()
      }
      const resolvedActions = res.actions.map(toResolvedAction)
      const chips = resolvedActions.length > 0 ? resolvedActions.map((a) => this.chipFor(a, res.conversationId || 'send')) : undefined
      this.messages.push(this.assistantMessage(res.responseText, chips))
    } catch (e) {
      this.pushAssistant(`Butler could not complete this request: ${e instanceof Error ? e.message : String(e)}`)
    } finally {
      this.sending = false
    }
  }

  private async refreshConversationList(): Promise<void> {
    try {
      this.conversations = await fetchButlerConversations()
    } catch {
      // Non-blocking sidebar refresh — a failure here doesn't disrupt the chat.
    }
  }

  runWorkflowAction(workflow: string): void {
    this.userInput = buildWorkflowPrompt(workflow, {})
    void this.send()
  }

  /* ---- arm/confirm (write-action chips) ---- */

  private arm(key: string): void {
    this.clearArm()
    this.armedKey = key
    this.armTimer = setTimeout(() => {
      this.armedKey = null
      this.armTimer = null
    }, ACTION_CONFIRMATION_WINDOW_MS)
  }

  clearArm(): void {
    this.armedKey = null
    if (this.armTimer) {
      clearTimeout(this.armTimer)
      this.armTimer = null
    }
  }

  async handleChipClick(_messageId: string, chip: ButlerChatActionChip): Promise<void> {
    const action = this.actionsByChipId.get(chip.id)
    if (!action) return
    const type = normalizeActionType(action.type)
    const target = resolveActionTarget(action.target)

    if (type === 'clarify') {
      const data = getActionDataObject(action)
      const prompt = String(data.prompt ?? data.command ?? action.label ?? '').trim()
      if (!prompt) {
        this.pushAssistant('I need a prompt attached to this command choice. Please ask Butler to regenerate the options.')
        return
      }
      this.userInput = prompt
      await this.send()
      return
    }

    const described = describeAction(action)
    if (described.disabled) {
      this.pushAssistant(`This action is not ready: ${described.statusLabel.toLowerCase()}. ${described.state.reason || 'Please add missing details and ask Butler to regenerate this action.'}`)
      return
    }

    const key = actionExecutionKey(action)
    if (isWriteAction(action)) {
      if (this.armedKey !== key) {
        this.arm(key)
        this.pushAssistant(buildActionPreview(action))
        return
      }
      this.clearArm()
    }

    if (type === 'navigate') {
      this.pushAssistant(`Navigate to "${target || action.label}". Cross-screen navigation from Butler isn't wired in this kernel yet — a K5 follow-up.`)
      return
    }
    if (type === 'analyze' || type === 'fetch') {
      this.userInput = `Tell me more about ${target}`
      await this.send()
      return
    }
    if (type === 'approve' || type === 'reject' || type === 'update' || type === 'create') {
      const outcome = await executeButlerAction(action)
      this.pushAssistant(outcome.message)
      return
    }

    const workflowKey = getActionWorkflowKey(action)
    if (workflowKey) {
      this.runWorkflowAction(workflowKey)
      return
    }
    if (typeof action.data === 'string') {
      this.userInput = `Proceed with ${action.data}`
      await this.send()
    }
  }

  /* ---- conversation sidebar: delete-arm / clear-all-arm (3s double-click) ---- */

  armDeleteConversation(conv: ButlerConversationRow): void {
    if (this.pendingDeleteKey === conv.id) {
      if (this.pendingDeleteTimer) clearTimeout(this.pendingDeleteTimer)
      this.pendingDeleteKey = null
      void this.confirmDeleteConversation(conv)
      return
    }
    if (this.pendingDeleteTimer) clearTimeout(this.pendingDeleteTimer)
    this.pendingDeleteKey = conv.id
    this.pendingDeleteTimer = setTimeout(() => {
      this.pendingDeleteKey = null
    }, 3000)
  }

  private async confirmDeleteConversation(conv: ButlerConversationRow): Promise<void> {
    try {
      await deleteButlerConversation(conv.id)
      this.conversations = this.conversations.filter((c) => c.id !== conv.id)
      if (this.activeConversationId === conv.id) this.startNewConversation()
    } catch (e) {
      this.pushAssistant(`Delete failed: ${e instanceof Error ? e.message : String(e)}`)
    }
  }

  armClearAll(): void {
    if (this.pendingClearAll) {
      if (this.pendingClearAllTimer) clearTimeout(this.pendingClearAllTimer)
      this.pendingClearAll = false
      void this.confirmClearAll()
      return
    }
    this.pendingClearAll = true
    this.pendingClearAllTimer = setTimeout(() => {
      this.pendingClearAll = false
    }, 3000)
  }

  private async confirmClearAll(): Promise<void> {
    try {
      await purgeAllButlerConversations()
      this.conversations = []
      this.activeConversationId = null
      this.messages = [this.assistantMessage('All conversations have been cleared.')]
    } catch (e) {
      this.pushAssistant(`Purge failed: ${e instanceof Error ? e.message : String(e)}`)
    }
  }

  dispose(): void {
    if (this.armTimer) clearTimeout(this.armTimer)
    if (this.pendingDeleteTimer) clearTimeout(this.pendingDeleteTimer)
    if (this.pendingClearAllTimer) clearTimeout(this.pendingClearAllTimer)
  }
}
