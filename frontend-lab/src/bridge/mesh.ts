/* Messenger bridge — the sidecar protocol v0 surface (MESSENGER_UI_CAMPAIGN.md
 * §1), mock-only for W-UI-1. The dev/DP4 transport (localhost TCP newline-JSON
 * today, sidecar stdio at DP4 — same frames) is Mission U1/W-UI-2 work; this
 * file exposes the exact method surface a `pick(realX, mockX)` swap will slot
 * into later, so the screen never has to change shape when the wire lands.
 *
 * The fold is the only rule engine (campaign §3). Nothing here re-implements
 * chat law — expectation vocabulary, claim rules, social-room skips are the
 * fold's job. The mock's "skip" behavior below exists only to give the UI
 * something real to render for that law, exactly as the real fold would
 * report it (skip/reject reasons surfaced verbatim, never invented client-side
 * beyond what a real host would say).
 *
 * Fixtures are synthetic (SYNTHETIC_IDENTITY.md canon — reuses the same
 * invented Gulf names already seeded in bridge/work.ts and bridge/
 * notifications.ts; no real people, no real companies). */

export type ExpectationTag = '' | 'whenever' | 'today' | 'urgent'
export type RoomKind = 'anchored' | 'social'

export interface Attachment {
  name: string
  size: number
  sha256: string
  ref: string
}

export interface MeshMessage {
  seq: number
  actor: string
  ts: string
  body: string
  expectation: ExpectationTag
  attachment: Attachment | null
}

export interface RoomSummary {
  roomKey: string
  title: string
  kind: RoomKind
  anchorType: string
  anchorId: string
  lastSeq: number
  lastTs: string
  lastPreview: string
  topExpectation: ExpectationTag
}

export interface RoomManifest {
  roomKey: string
  title: string
  kind: RoomKind
  anchorType: string
  anchorId: string
  createdAt: string
}

export interface RoomMember {
  actor: string
  devicePub: string
  role: string
}

export interface RoomClaim {
  assignee: string
  claimedAt: string
}

export interface RoomState {
  manifest: RoomManifest
  members: RoomMember[]
  claim: RoomClaim | null
  capEpoch: number
  messages: MeshMessage[]
  skippedCount: number
  rejectedCount: number
}

export interface HelloResult {
  devicePub: string
  actor: string
  version: string
}

export interface TranscriptBundle {
  bundleVersion: string
  roomKey: string
  exportedAt: string
  messages: MeshMessage[]
}

/* ---- mock: adversarial-lite + deterministic in-memory fold ----
 * Not adversarial-heavy (no 200-char monsters) — the protocol surface, room
 * taxonomy and expectation vocabulary are the thing under test this wave, not
 * overflow behavior (that's L1/primitives' job, already proven elsewhere). */

const ME: RoomMember = { actor: 'Aisha Al-Rumaihi', devicePub: 'dev:aisha-01', role: 'member' }
const MOHAMMED: RoomMember = { actor: 'Mohammed Bucheeri', devicePub: 'dev:mohammed-01', role: 'member' }
const FATIMA: RoomMember = { actor: 'Fatima Al-Zayani', devicePub: 'dev:fatima-01', role: 'member' }
const YUSUF: RoomMember = { actor: 'Yusuf Kanoo', devicePub: 'dev:yusuf-01', role: 'member' }
const KHALID: RoomMember = { actor: 'Khalid Al-Mannai', devicePub: 'dev:khalid-01', role: 'member' }
const NOORA: RoomMember = { actor: 'Noora Al-Sulaiti', devicePub: 'dev:noora-01', role: 'member' }

const VALID_TAGS = new Set<ExpectationTag>(['', 'whenever', 'today', 'urgent'])
const RANK: Record<ExpectationTag, number> = { urgent: 3, today: 2, whenever: 1, '': 0 }

/** Days-ago -> a fixed-anchor ISO timestamp so the mock stays deterministic
 * across runs (same idiom as bridge/notifications.ts TODAY anchor). */
const NOW = new Date('2026-07-18T09:00:00Z').getTime()
function ts(hoursAgo: number): string {
  return new Date(NOW - hoursAgo * 60 * 60 * 1000).toISOString()
}

interface MockRoom {
  manifest: RoomManifest
  members: RoomMember[]
  claim: RoomClaim | null
  capEpoch: number
  messages: MeshMessage[]
  skippedCount: number
  rejectedCount: number
}

let rooms: Map<string, MockRoom> | null = null

function seed(): Map<string, MockRoom> {
  const m = new Map<string, MockRoom>()

  m.set('room-po-2026-0417', {
    manifest: {
      roomKey: 'room-po-2026-0417',
      title: 'PO-2026-0417 — Gulf Fabrication W.L.L.',
      kind: 'anchored',
      anchorType: 'purchase_order',
      anchorId: 'po-2026-0417',
      createdAt: ts(96),
    },
    members: [ME, MOHAMMED],
    claim: { assignee: 'Mohammed Bucheeri', claimedAt: ts(20) },
    capEpoch: 1,
    skippedCount: 0,
    rejectedCount: 0,
    messages: [
      { seq: 1, actor: 'Aisha Al-Rumaihi', ts: ts(30), body: 'Supplier confirmed delivery for next Tuesday — can you check the LC amendment before then?', expectation: 'today', attachment: null },
      { seq: 2, actor: 'Mohammed Bucheeri', ts: ts(19), body: 'On it — LC amendment reviewed, sending to bank this afternoon.', expectation: '', attachment: null },
      { seq: 3, actor: 'Aisha Al-Rumaihi', ts: ts(2), body: 'Customer is pushing hard on this one, please treat as priority.', expectation: 'urgent', attachment: null },
    ],
  })

  m.set('room-proj-npc-retrofit', {
    manifest: {
      roomKey: 'room-proj-npc-retrofit',
      title: 'Project: NPC Retrofit Phase 2',
      kind: 'anchored',
      anchorType: 'project',
      anchorId: 'proj-npc-retrofit',
      createdAt: ts(200),
    },
    members: [ME, FATIMA, YUSUF],
    claim: null,
    capEpoch: 1,
    skippedCount: 0,
    rejectedCount: 0,
    messages: [
      { seq: 1, actor: 'Fatima Al-Zayani', ts: ts(6), body: 'Site survey scheduled for Thursday, need someone from ops to attend.', expectation: 'today', attachment: null },
      { seq: 2, actor: 'Yusuf Kanoo', ts: ts(5), body: 'I can go — will bring the calibration checklist.', expectation: '', attachment: null },
    ],
  })

  m.set('room-cust-manama-process', {
    manifest: {
      roomKey: 'room-cust-manama-process',
      title: 'Customer: Manama Process Systems',
      kind: 'anchored',
      anchorType: 'customer',
      anchorId: 'cust-manama-process',
      createdAt: ts(500),
    },
    members: [ME, NOORA],
    claim: null,
    capEpoch: 1,
    skippedCount: 1,
    rejectedCount: 0,
    messages: [
      {
        seq: 1,
        actor: 'Noora Al-Sulaiti',
        ts: ts(48),
        body: 'Attaching the revised costing sheet for their review.',
        expectation: 'whenever',
        attachment: { name: 'Costing_Sheet_v3.xlsx', size: 184320, sha256: '9f2c9c1a4e7b3d5f8a1c6e0b2d4f7a9c1e3b5d7f9a1c3e5b7d9f1a3c5e7b9d1f', ref: 'blob:9f2c9c1a4e7b' },
      },
      { seq: 2, actor: 'Aisha Al-Rumaihi', ts: ts(40), body: 'Thanks — will send to them this week.', expectation: '', attachment: null },
    ],
  })

  m.set('room-social-coffee-corner', {
    manifest: {
      roomKey: 'room-social-coffee-corner',
      title: 'Coffee Corner',
      kind: 'social',
      anchorType: '',
      anchorId: '',
      createdAt: ts(1000),
    },
    members: [ME, MOHAMMED, KHALID, NOORA],
    claim: null,
    capEpoch: 1,
    skippedCount: 0,
    rejectedCount: 0,
    messages: [
      { seq: 1, actor: 'Khalid Al-Mannai', ts: ts(3), body: 'Anyone up for lunch at the new place near the roundabout?', expectation: '', attachment: null },
      { seq: 2, actor: 'Mohammed Bucheeri', ts: ts(2.5), body: "I'm in, 1pm?", expectation: '', attachment: null },
    ],
  })

  m.set('room-dm-khalid', {
    manifest: {
      roomKey: 'room-dm-khalid',
      title: 'Direct: Khalid Al-Mannai',
      kind: 'social',
      anchorType: '',
      anchorId: '',
      createdAt: ts(300),
    },
    members: [ME, KHALID],
    claim: null,
    capEpoch: 1,
    skippedCount: 0,
    rejectedCount: 0,
    messages: [
      { seq: 1, actor: 'Khalid Al-Mannai', ts: ts(9), body: 'Quick one when you have a sec — no rush.', expectation: 'whenever', attachment: null },
      { seq: 2, actor: 'Aisha Al-Rumaihi', ts: ts(8), body: 'Sure, free after 3.', expectation: '', attachment: null },
    ],
  })

  return m
}

function store(): Map<string, MockRoom> {
  rooms ??= seed()
  return rooms
}

function getOrThrow(roomKey: string): MockRoom {
  const room = store().get(roomKey)
  if (!room) throw new Error(`No such room: ${roomKey}`)
  return room
}

function nextSeq(room: MockRoom): number {
  return (room.messages[room.messages.length - 1]?.seq ?? 0) + 1
}

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))

/** Highest-urgency tag across the room's own messages — the mock stand-in for
 * "highest-urgency expectation at/after my cursor" (v0 has no read cursors in
 * the client yet, so this over-approximates with the whole room; honest
 * simplification, not a fold behavior). */
function topExpectation(room: MockRoom): ExpectationTag {
  let best: ExpectationTag = ''
  for (const msg of room.messages) {
    if (RANK[msg.expectation] > RANK[best]) best = msg.expectation
  }
  return best
}

async function mockHello(): Promise<HelloResult> {
  await sleep(60)
  return { devicePub: ME.devicePub, actor: ME.actor, version: 'protocol-v0-mock' }
}

async function mockListRooms(): Promise<RoomSummary[]> {
  await sleep(120)
  return [...store().values()].map((room) => {
    const last = room.messages[room.messages.length - 1]
    return {
      roomKey: room.manifest.roomKey,
      title: room.manifest.title,
      kind: room.manifest.kind,
      anchorType: room.manifest.anchorType,
      anchorId: room.manifest.anchorId,
      lastSeq: last?.seq ?? 0,
      lastTs: last?.ts ?? room.manifest.createdAt,
      lastPreview: last?.body ?? '',
      topExpectation: topExpectation(room),
    }
  })
}

async function mockRoomState(roomKey: string): Promise<RoomState> {
  await sleep(100)
  const room = getOrThrow(roomKey)
  return {
    manifest: room.manifest,
    members: room.members,
    claim: room.claim,
    capEpoch: room.capEpoch,
    messages: room.messages,
    skippedCount: room.skippedCount,
    rejectedCount: room.rejectedCount,
  }
}

async function mockPost(roomKey: string, params: { body: string; expectation: ExpectationTag }): Promise<{ seq: number }> {
  await sleep(80)
  const room = getOrThrow(roomKey)
  if (!VALID_TAGS.has(params.expectation)) {
    room.skippedCount += 1
    throw new Error('unknown expectation tag')
  }
  if (!params.body.trim()) {
    room.skippedCount += 1
    throw new Error('post requires a body or an attachment')
  }
  const seq = nextSeq(room)
  room.messages.push({ seq, actor: ME.actor, ts: new Date().toISOString(), body: params.body.trim(), expectation: params.expectation, attachment: null })
  return { seq }
}

async function mockClaimRoom(roomKey: string, params: { assignee: string }): Promise<{ seq: number }> {
  await sleep(80)
  const room = getOrThrow(roomKey)
  if (room.manifest.kind === 'social') {
    room.skippedCount += 1
    throw new Error('claims are a work concept')
  }
  const seq = nextSeq(room)
  room.claim = { assignee: params.assignee || ME.actor, claimedAt: new Date().toISOString() }
  return { seq }
}

async function mockReleaseClaim(roomKey: string): Promise<{ seq: number }> {
  await sleep(80)
  const room = getOrThrow(roomKey)
  if (room.manifest.kind === 'social') {
    room.skippedCount += 1
    throw new Error('claims are a work concept')
  }
  const seq = nextSeq(room)
  room.claim = null
  return { seq }
}

async function mockAttach(
  roomKey: string,
  params: { filePath: string; body: string; expectation: ExpectationTag },
): Promise<{ seq: number; ref: string; sha256: string }> {
  await sleep(150)
  const room = getOrThrow(roomKey)
  if (!VALID_TAGS.has(params.expectation)) {
    room.skippedCount += 1
    throw new Error('unknown expectation tag')
  }
  const name = params.filePath.split(/[\\/]/).pop() || 'attachment'
  const seq = nextSeq(room)
  // Deterministic mock digest — a real host computes this over actual bytes.
  const sha256 = `mock${seq.toString(16).padStart(4, '0')}${name.length.toString(16).padStart(4, '0')}`.padEnd(64, '0')
  const ref = `blob:mock-${roomKey}-${seq}`
  room.messages.push({
    seq,
    actor: ME.actor,
    ts: new Date().toISOString(),
    body: params.body.trim(),
    expectation: params.expectation,
    attachment: { name, size: 1024 * (1 + (seq % 200)), sha256, ref },
  })
  return { seq, ref, sha256 }
}

async function mockFetchAttachment(roomKey: string, params: { ref: string; savePath: string }): Promise<{ path: string; sha256: string; verified: boolean }> {
  await sleep(150)
  const room = getOrThrow(roomKey)
  const msg = room.messages.find((m) => m.attachment?.ref === params.ref)
  if (!msg?.attachment) throw new Error(`No such attachment: ${params.ref}`)
  return { path: params.savePath, sha256: msg.attachment.sha256, verified: true }
}

let socialRoomCounter = 0

async function mockCreateSocialRoom(params: { title: string }): Promise<{ inviteCode: string }> {
  await sleep(100)
  socialRoomCounter += 1
  const roomKey = `room-social-${socialRoomCounter}-${Date.now().toString(36)}`
  const manifest: RoomManifest = {
    roomKey,
    title: params.title.trim() || 'Untitled room',
    kind: 'social',
    anchorType: '',
    anchorId: '',
    createdAt: new Date().toISOString(),
  }
  store().set(roomKey, { manifest, members: [ME], claim: null, capEpoch: 1, skippedCount: 0, rejectedCount: 0, messages: [] })
  return { inviteCode: `asymm-room2.${roomKey}.${Math.random().toString(36).slice(2, 10)}` }
}

async function mockOpenDmInvite(roomKey: string): Promise<{ inviteCode: string }> {
  await sleep(80)
  getOrThrow(roomKey)
  return { inviteCode: `asymm-room2.${roomKey}.${Math.random().toString(36).slice(2, 10)}` }
}

async function mockRedeemInvite(params: { inviteCode: string; actor: string }): Promise<{ roomKey: string }> {
  await sleep(120)
  const parts = params.inviteCode.split('.')
  const roomKey = parts[1]
  if (!roomKey || !store().has(roomKey)) throw new Error(`Invalid or unknown invite code: ${params.inviteCode}`)
  const room = getOrThrow(roomKey)
  if (!room.members.some((m) => m.actor === params.actor)) {
    room.members.push({ actor: params.actor, devicePub: `dev:${params.actor.toLowerCase().replace(/\s+/g, '-')}`, role: 'member' })
  }
  return { roomKey }
}

async function mockExportTranscript(roomKey: string): Promise<TranscriptBundle> {
  await sleep(150)
  const room = getOrThrow(roomKey)
  return { bundleVersion: 'asymm-transcript.v1', roomKey, exportedAt: new Date().toISOString(), messages: room.messages }
}

/* ---- public switched API (screen/viewmodel imports THESE) ----
 * v0: mock-only (see file header). Same shape `pick(realX, mockX)` will wrap
 * at W-UI-2 once the sidecar TCP client lands — no call-site change needed. */

export const hello = (): Promise<HelloResult> => mockHello()
export const listRooms = (): Promise<RoomSummary[]> => mockListRooms()
export const roomState = (roomKey: string): Promise<RoomState> => mockRoomState(roomKey)
export const post = (roomKey: string, params: { body: string; expectation: ExpectationTag }): Promise<{ seq: number }> => mockPost(roomKey, params)
export const claimRoom = (roomKey: string, params: { assignee: string }): Promise<{ seq: number }> => mockClaimRoom(roomKey, params)
export const releaseClaim = (roomKey: string): Promise<{ seq: number }> => mockReleaseClaim(roomKey)
export const attach = (roomKey: string, params: { filePath: string; body: string; expectation: ExpectationTag }): Promise<{ seq: number; ref: string; sha256: string }> =>
  mockAttach(roomKey, params)
export const fetchAttachment = (roomKey: string, params: { ref: string; savePath: string }): Promise<{ path: string; sha256: string; verified: boolean }> =>
  mockFetchAttachment(roomKey, params)
export const createSocialRoom = (params: { title: string }): Promise<{ inviteCode: string }> => mockCreateSocialRoom(params)
export const openDmInvite = (roomKey: string): Promise<{ inviteCode: string }> => mockOpenDmInvite(roomKey)
export const redeemInvite = (params: { inviteCode: string; actor: string }): Promise<{ roomKey: string }> => mockRedeemInvite(params)
export const exportTranscript = (roomKey: string): Promise<TranscriptBundle> => mockExportTranscript(roomKey)

// Re-exported so the VM can label posts as "me" without hardcoding the name.
export const MY_ACTOR = ME.actor
