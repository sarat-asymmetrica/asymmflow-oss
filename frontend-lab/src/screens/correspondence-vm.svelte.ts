/* Correspondence viewmodel — L5's reactive half for the flat-inbox +
 * REPL-thread master-detail (Mission U2, MESSENGER_UI_CAMPAIGN.md §2). Owns:
 * the room list load + search/filter/sort, the open thread's state, the
 * composer (plain text + expectation chip + `/`-commands + mock attach), and
 * claim/release/invite mutations. Never re-implements chat law (Article III/
 * VI) — every skip/reject reason rendered here is the string mesh.ts's mock
 * fold threw, passed through verbatim, exactly as a real host's response
 * would be. No layout in this file (L1) — the screen renders this state on
 * primitives only. */

import {
  attach as bridgeAttach,
  claimRoom as bridgeClaimRoom,
  listRooms,
  MY_ACTOR,
  openDmInvite,
  post as bridgePost,
  releaseClaim as bridgeReleaseClaim,
  roomState as bridgeRoomState,
  type ExpectationTag,
  type RoomState,
  type RoomSummary,
} from '../bridge/mesh'

export type InboxFilter = '' | 'rooms' | 'people'

const RANK: Record<ExpectationTag, number> = { urgent: 3, today: 2, whenever: 1, '': 0 }

export const EXPECTATION_OPTIONS: { value: ExpectationTag; label: string }[] = [
  { value: '', label: 'No tag' },
  { value: 'whenever', label: '🍃 Whenever' },
  { value: 'today', label: '🌤 Today' },
  { value: 'urgent', label: '🔥 Urgent' },
]

const KNOWN_COMMANDS = new Set(['expect', 'claim', 'release', 'invite', 'attach'])

function matchesSearch(room: RoomSummary, search: string): boolean {
  if (!search.trim()) return true
  return room.title.toLowerCase().includes(search.trim().toLowerCase())
}

export class CorrespondenceViewModel {
  loading = $state(true)
  error = $state<string | null>(null)
  rooms = $state<RoomSummary[]>([])

  search = $state('')
  filter = $state<InboxFilter>('')

  selectedRoomKey = $state('')
  thread = $state<RoomState | null>(null)
  threadLoading = $state(false)
  threadError = $state<string | null>(null)

  composerText = $state('')
  // Default is `whenever`, per Constitution Art. III §3 (gate ruling: the
  // constitution's stated default wins over the mission brief's literal '' —
  // "no tag" stays selectable, it just isn't the default posture).
  composerExpectation = $state<ExpectationTag>('whenever')
  posting = $state(false)

  attaching = $state(false)
  attachFileName = $state('')

  inviteCode = $state('')
  commandNotice = $state('')

  filterOptions = [
    { value: 'rooms', label: 'Rooms' },
    { value: 'people', label: 'People' },
  ]

  visibleRooms = $derived.by(() => {
    const filtered = this.rooms
      .filter((r) => matchesSearch(r, this.search))
      .filter((r) => {
        if (this.filter === 'rooms') return r.kind === 'anchored'
        if (this.filter === 'people') return r.kind === 'social'
        return true
      })
    return [...filtered].sort((a, b) => {
      const rankDiff = RANK[b.topExpectation] - RANK[a.topExpectation]
      if (rankDiff !== 0) return rankDiff
      return b.lastTs.localeCompare(a.lastTs)
    })
  })

  selectedRoomSummary = $derived(this.rooms.find((r) => r.roomKey === this.selectedRoomKey) ?? null)

  async load(): Promise<void> {
    this.loading = true
    this.error = null
    try {
      this.rooms = await listRooms()
      if (!this.selectedRoomKey && this.rooms.length > 0) {
        await this.selectRoom(this.rooms[0]!.roomKey)
      }
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e)
    } finally {
      this.loading = false
    }
  }

  async selectRoom(roomKey: string): Promise<void> {
    this.selectedRoomKey = roomKey
    this.threadError = null
    this.inviteCode = ''
    this.commandNotice = ''
    this.attaching = false
    this.composerExpectation = 'whenever'
    await this.loadThread()
  }

  async loadThread(): Promise<void> {
    if (!this.selectedRoomKey) return
    this.threadLoading = true
    try {
      this.thread = await bridgeRoomState(this.selectedRoomKey)
    } catch (e) {
      this.threadError = e instanceof Error ? e.message : String(e)
      this.thread = null
    } finally {
      this.threadLoading = false
    }
  }

  /** Refresh both the open thread and the inbox's list-level summary (a post
   * changes lastPreview/lastTs/topExpectation for the row too). */
  private async refreshAfterMutation(): Promise<void> {
    await Promise.all([this.loadThread(), this.load()])
  }

  setExpectation(tag: ExpectationTag): void {
    this.composerExpectation = tag
  }

  toggleAttach(): void {
    this.attaching = !this.attaching
    if (!this.attaching) this.attachFileName = ''
  }

  /** The single composer submit path — plain text posts a message; a leading
   * `/` routes to a command. Mirrors a REPL, not a form: one input, one
   * dispatch point (L2 — one path for composer intent). */
  async submit(): Promise<void> {
    const raw = this.composerText.trim()
    if (!raw || !this.selectedRoomKey) return
    if (raw.startsWith('/')) {
      await this.runCommand(raw.slice(1))
      return
    }
    await this.sendMessage(raw)
  }

  private async sendMessage(body: string): Promise<void> {
    this.posting = true
    this.threadError = null
    try {
      await bridgePost(this.selectedRoomKey, { body, expectation: this.composerExpectation })
      this.composerText = ''
      await this.refreshAfterMutation()
    } catch (e) {
      // The fold's own skip reason, surfaced verbatim — never re-worded.
      this.threadError = e instanceof Error ? e.message : String(e)
    } finally {
      this.posting = false
    }
  }

  private async runCommand(rest: string): Promise<void> {
    const tokens = rest.trim().split(/\s+/)
    const cmd = (tokens[0] ?? '').toLowerCase()
    this.threadError = null
    this.commandNotice = ''

    if (!KNOWN_COMMANDS.has(cmd)) {
      this.threadError = `Unknown command: /${cmd}`
      return
    }

    if (cmd === 'expect') {
      const tag = (tokens[1] ?? '').toLowerCase() as ExpectationTag
      if (!EXPECTATION_OPTIONS.some((o) => o.value === tag)) {
        this.threadError = `Unknown expectation tag: "${tokens[1] ?? ''}" (use whenever | today | urgent, or nothing to clear).`
        return
      }
      this.composerExpectation = tag
      this.commandNotice = tag ? `Expectation set to "${tag}" for your next message.` : 'Expectation cleared for your next message.'
      this.composerText = ''
      return
    }

    if (cmd === 'claim') {
      await this.claim()
      this.composerText = ''
      return
    }

    if (cmd === 'release') {
      await this.release()
      this.composerText = ''
      return
    }

    if (cmd === 'invite') {
      await this.invite()
      this.composerText = ''
      return
    }

    if (cmd === 'attach') {
      this.attaching = true
      this.composerText = ''
      return
    }
  }

  async claim(): Promise<void> {
    if (!this.selectedRoomKey) return
    this.threadError = null
    try {
      await bridgeClaimRoom(this.selectedRoomKey, { assignee: MY_ACTOR })
      await this.refreshAfterMutation()
    } catch (e) {
      this.threadError = e instanceof Error ? e.message : String(e)
    }
  }

  async release(): Promise<void> {
    if (!this.selectedRoomKey) return
    this.threadError = null
    try {
      await bridgeReleaseClaim(this.selectedRoomKey)
      await this.refreshAfterMutation()
    } catch (e) {
      this.threadError = e instanceof Error ? e.message : String(e)
    }
  }

  async invite(): Promise<void> {
    if (!this.selectedRoomKey) return
    this.threadError = null
    try {
      const { inviteCode } = await openDmInvite(this.selectedRoomKey)
      this.inviteCode = inviteCode
      this.commandNotice = 'Invite code generated — copy the line below.'
    } catch (e) {
      this.threadError = e instanceof Error ? e.message : String(e)
    }
  }

  async sendAttachment(): Promise<void> {
    if (!this.selectedRoomKey || !this.attachFileName.trim()) return
    this.posting = true
    this.threadError = null
    try {
      await bridgeAttach(this.selectedRoomKey, {
        filePath: this.attachFileName.trim(),
        body: this.composerText.trim(),
        expectation: this.composerExpectation,
      })
      this.composerText = ''
      this.attachFileName = ''
      this.attaching = false
      await this.refreshAfterMutation()
    } catch (e) {
      this.threadError = e instanceof Error ? e.message : String(e)
    } finally {
      this.posting = false
    }
  }
}
