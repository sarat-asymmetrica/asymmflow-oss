/* Butler bridge module — self-contained: types + mock + real + switch.
 * FETCH is real (ListConversations/GetConversationMessages on ButlerService,
 * ListCustomers/GetCustomer/ListSuppliers/GetSupplier on CRMService for the
 * name<->id lookup cache the action-payload resolver needs). Every MUTATION
 * is INTEG-gapped by design (Butler.parity.md):
 *  - sendButlerMessage: the old screen's 3-tier fallback (persistent chat ->
 *    fresh-persistent retry -> legacy ChatWithButler) collapses to ONE INTEG
 *    throw naming ChatWithButlerPersistent when a live Wails runtime is
 *    present; the mock path still returns a canned acknowledgement so the
 *    chat feels alive in the lab (mirrors the old screen's `!window.go`
 *    branch, which already behaved this way).
 *  - deleteButlerConversation / purgeAllButlerConversations: real throws
 *    naming DeleteConversation/PurgeAllConversations; mock actually mutates
 *    the seeded cache so the sidebar's delete-arm/clear-all-arm hot zones are
 *    demonstrably functional in the lab.
 *  - executeButlerActionBinding: the ONE generic thrower behind ALL 23
 *    write-action bindings (CreateOfferDraftFromButler, CreateFollowUp, ...
 *    see butler-actions.ts's resolveBindingName). Unlike the mutations above,
 *    these are NEVER simulated — even in mock mode — because "the AI proposed
 *    a write, a human armed+confirmed it" is exactly the boundary K5 wiring
 *    must cross for real; faking success here would hide that boundary. */
import { pick } from './runtime'
import { goDate, str } from './map'
import { ListConversations, GetConversationMessages } from '$wails/go/main/ButlerService'
import { ListCustomers, GetCustomer, ListSuppliers, GetSupplier } from '$wails/go/main/CRMService'

export interface ButlerConversationRow {
  id: string
  title: string
  summary: string
  isActive: boolean
  lastMsgAt: string
}

/** Mirrors butler.ChatMessage's raw shape — action hydration (parsing
 * action_metadata / action_data / the legacy singular action_* fields) is
 * domain logic that lives in butler-actions.ts, identically for mock + real. */
export interface ButlerChatMessageRow {
  id: string
  role: string
  content: string
  messageType: string
  actionType: string
  actionTarget: string
  actionLabel: string
  actionData: string
  actionMetadata: string
}

export interface ButlerActionPayload {
  type: string
  target: string
  label: string
  data: unknown
}

export interface ButlerChatResult {
  responseText: string
  conversationId: string
  actions: ButlerActionPayload[]
  confidence: number
}

export interface LookupEntry {
  id: string
  name: string
}

/* ---- mock: adversarial + deterministic (see bridge/mock.ts pattern) ---- */
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))
function lcg(seed: number): () => number {
  let s = seed >>> 0
  return () => {
    s = (s * 1664525 + 1013904223) >>> 0
    return s / 0xffffffff
  }
}
const pad = (n: number, w: number): string => String(n).padStart(w, '0')

/** The raw action_metadata wire shape is untyped JSON (`data any` on the Go
 * struct) — the real backend can and does ride extra ad-hoc keys
 * (requires_approval, execution_status, invalid_reason, missing_fields) on
 * top of the strict {type,target,label,data} ButlerAction fields. Loose here
 * on purpose; butler-actions.ts's hydrateActionsFromMessage is what narrows
 * this into the typed ResolvedButlerAction the VM renders. */
type RawActionJSON = Record<string, unknown>

function action(type: string, target: string, label: string, data: unknown, extra: RawActionJSON = {}): RawActionJSON {
  return { type, target, label, data, ...extra }
}

/** Encodes actions exactly like the real backend's action_metadata column
 * (`{"actions":[...]}`) so hydration in butler-actions.ts never branches on
 * mock-vs-real. */
function withActions(actions: RawActionJSON[]): string {
  return JSON.stringify({ actions })
}

const CUSTOMER_NAMES = [
  'Bahrain Precision Instruments W.L.L.',
  'Gulf Valve & Actuator Trading Co.',
  'Al Manar Industrial Supplies',
  'Manama Energy Services',
  'International Establishment for Process Control Equipment, Calibration Services, Spare Parts Distribution, Instrumentation Repair, Field Service Contracting and General Engineering Trading (formerly Gulf Technical Instrumentation Holding Company) W.L.L.',
  'شركة الخليج للتوريدات الصناعية والمعايرة ذ.م.م',
  'Sitra Metal Works',
  '',
]

const SUPPLIER_NAMES = [
  'Northbridge Flow Controls Ltd',
  'Al Dana Engineering Supplies',
  'Qalali Industrial Trading',
  'Riffa Calibration Services',
  '',
]

let customerLookupSeed: LookupEntry[] | null = null
let supplierLookupSeed: LookupEntry[] | null = null

function seedCustomerLookup(): LookupEntry[] {
  customerLookupSeed ??= CUSTOMER_NAMES.filter(Boolean).map((name, i) => ({ id: `cust-${i + 1}`, name }))
  return customerLookupSeed
}
function seedSupplierLookup(): LookupEntry[] {
  supplierLookupSeed ??= SUPPLIER_NAMES.filter(Boolean).map((name, i) => ({ id: `supp-${i + 1}`, name }))
  return supplierLookupSeed
}

/* ---- conversation seeds: hand-authored monsters (adversarial coverage) +
 * a small LCG-generated filler tail to reach the spec's 15-20 range. ---- */
interface ConvSeed {
  id: string
  title: string
  summary: string
  isActive: boolean
  lastMsgAt: string
  messages: ButlerChatMessageRow[]
}

function msg(id: string, role: string, content: string, extra: Partial<ButlerChatMessageRow> = {}): ButlerChatMessageRow {
  return {
    id,
    role,
    content,
    messageType: extra.messageType ?? (extra.actionMetadata || extra.actionType ? 'assistant_actionable' : role === 'user' ? 'user_message' : 'assistant_text'),
    actionType: extra.actionType ?? '',
    actionTarget: extra.actionTarget ?? '',
    actionLabel: extra.actionLabel ?? '',
    actionData: extra.actionData ?? '',
    actionMetadata: extra.actionMetadata ?? '',
  }
}

function buildSeeds(): ConvSeed[] {
  const seeds: ConvSeed[] = []

  // 1 — normal short conversation, one READY create-offer chip.
  seeds.push({
    id: 'conv-1',
    title: 'Q3 offer follow-ups',
    summary: 'Follow-ups on pending offers for Bahrain Precision Instruments.',
    isActive: true,
    lastMsgAt: '2026-07-14',
    messages: [
      msg('conv-1-m1', 'user', 'Draft an offer for Bahrain Precision Instruments, 4 flow meters.'),
      msg(
        'conv-1-m2',
        'assistant',
        'Here is a draft offer summary. Confirm to create it.',
        {
          actionMetadata: withActions([
            action('create', 'offer', 'Create offer draft', {
              customer_name: 'Bahrain Precision Instruments W.L.L.',
              customer_id: 'cust-1',
              line_items: [{ equipment: 'Flow meter', quantity: 4, unit_price_bhd: 210.5 }],
            }),
          ]),
        },
      ),
    ],
  })

  // 2 — 200-char title, one NEEDS_INPUT chip (missing customer + amount).
  seeds.push({
    id: 'conv-2',
    title:
      'International Establishment for Process Control Equipment, Calibration Services, Spare Parts Distribution, Instrumentation Repair, Field Service Contracting and General Engineering Trading — quarterly review thread'.padEnd(
        210,
        ' ·',
      ),
    summary: '',
    isActive: false,
    lastMsgAt: '2025-11-02',
    messages: [
      msg('conv-2-m1', 'user', 'Create a follow-up task.'),
      msg('conv-2-m2', 'assistant', 'I need a customer and a follow-up title before I can create this.', {
        actionMetadata: withActions([action('create', 'follow_up', 'Create follow-up', {})]),
      }),
    ],
  })

  // 3 — empty title (renders "Untitled"), NEEDS_APPROVAL chip (PO approve).
  seeds.push({
    id: 'conv-3',
    title: '',
    summary: 'Purchase order approvals pending.',
    isActive: true,
    lastMsgAt: '2026-01-30',
    messages: [
      msg('conv-3-m1', 'user', 'Approve PO-2201.'),
      msg('conv-3-m2', 'assistant', "This purchase order exceeds the auto-approval threshold and needs sign-off.", {
        actionMetadata: withActions([
          action(
            'approve',
            'purchase_order',
            'Approve PO-2201',
            { id: '2201', purchase_order_id: '2201' },
            { requires_approval: true },
          ),
        ]),
      }),
    ],
  })

  // 4 — the scroll-stress conversation: 42 alternating messages.
  {
    const messages: ButlerChatMessageRow[] = []
    for (let i = 1; i <= 42; i++) {
      const role = i % 2 === 1 ? 'user' : 'assistant'
      messages.push(
        msg(
          `conv-4-m${i}`,
          role,
          role === 'user' ? `Status update on shipment ${i}?` : `Shipment ${i} is on schedule, ETA in ${i % 10} days.`,
        ),
      )
    }
    seeds.push({
      id: 'conv-4',
      title: 'Daily shipment status thread',
      summary: 'Recurring shipment status Q&A.',
      isActive: true,
      lastMsgAt: '2026-07-10',
      messages,
    })
  }

  // 5 — empty conversation (0 messages).
  seeds.push({
    id: 'conv-5',
    title: 'New conversation',
    summary: '',
    isActive: false,
    lastMsgAt: '',
    messages: [],
  })

  // 6 — huge multi-block markdown response (headings + bullets + numbered + table).
  seeds.push({
    id: 'conv-6',
    title: 'Daily briefing — 2026-07-14',
    summary: 'Full daily briefing with priorities, risks, and a pipeline table.',
    isActive: true,
    lastMsgAt: '2026-07-14',
    messages: [
      msg('conv-6-m1', 'user', 'Give me the daily briefing.'),
      msg(
        'conv-6-m2',
        'assistant',
        [
          '# Daily Briefing',
          '',
          '## Priorities',
          '- Follow up on 3 offers awaiting customer response',
          '- Chase overdue supplier invoice from Northbridge Flow Controls Ltd',
          '- Review stock adjustment pending approval on item FM-4410',
          '',
          '## Risks',
          '1. Two purchase orders are past their expected delivery date',
          '2. One customer contact record is missing a phone number',
          '3. FX exposure on the USD-denominated HSBC account has widened',
          '',
          '### Pipeline snapshot',
          '| Stage | Count | Value (BHD) |',
          '|---|---|---|',
          '| RFQ Received | 5 | 42,300.500 |',
          '| Offer Sent | 3 | 18,750.000 |',
          '| Order Placed | 2 | 61,000.000 |',
          '',
          'Let me know if you want detail on any of the above.',
        ].join('\n'),
      ),
    ],
  })

  // 7 — malformed/ragged markdown table (uneven column counts).
  seeds.push({
    id: 'conv-7',
    title: 'Supplier scorecard export',
    summary: '',
    isActive: true,
    lastMsgAt: '2026-06-20',
    messages: [
      msg('conv-7-m1', 'user', 'Summarize supplier scorecards.'),
      msg(
        'conv-7-m2',
        'assistant',
        [
          '| Supplier | On-time % | Notes |',
          '|---|---|---|',
          '| Northbridge Flow Controls Ltd | 92 |',
          '| Al Dana Engineering Supplies | 78 | Late twice this quarter | extra cell |',
          '| Qalali Industrial Trading |',
        ].join('\n'),
      ),
    ],
  })

  // 8 — 4-chip message (chip wrap test), mixed states.
  seeds.push({
    id: 'conv-8',
    title: 'Month-end action queue',
    summary: '',
    isActive: true,
    lastMsgAt: '2026-07-01',
    messages: [
      msg('conv-8-m1', 'user', 'What needs my attention this month-end?'),
      msg('conv-8-m2', 'assistant', 'Here are four items that need a decision.', {
        actionMetadata: withActions([
          action('approve', 'purchase_order', 'Approve PO-1187', { id: '1187' }),
          action('reject', 'supplier_invoice', 'Dispute SI-2093', { id: '2093', reason: 'Quantity mismatch' }),
          action('create', 'stock_adjustment', 'Log stock adjustment', {}),
          action(
            'update',
            'opportunity',
            'Update opportunity stage',
            { id: '77', status: 'Vaporized' },
            { execution_status: 'invalid_payload', invalid_reason: "Unrecognized stage 'Vaporized' for opportunity" },
          ),
        ]),
      }),
    ],
  })

  // 9 — a chip whose data is a bare string (not an object).
  seeds.push({
    id: 'conv-9',
    title: 'Quick note',
    summary: '',
    isActive: false,
    lastMsgAt: '2026-05-15',
    messages: [
      msg('conv-9-m1', 'user', 'Proceed with the reorder.'),
      msg('conv-9-m2', 'assistant', 'Proceed with the reorder as discussed?', {
        actionMetadata: withActions([action('create', 'order', 'Proceed', 'reorder-batch-14')]),
      }),
    ],
  })

  // 10 — invalid_payload chip (stock adjustment missing everything).
  seeds.push({
    id: 'conv-10',
    title: 'Inventory count discrepancy',
    summary: '',
    isActive: true,
    lastMsgAt: '2026-04-11',
    messages: [
      msg('conv-10-m1', 'user', 'Physical count for FM-4410 came in short.'),
      msg('conv-10-m2', 'assistant', "This stock adjustment isn't ready yet — I'm missing the required fields.", {
        actionMetadata: withActions([action('create', 'stock_adjustment', 'Create stock adjustment', { notes: 'short count' })]),
      }),
    ],
  })

  // 11 — clarify-type chip (single-click, not armed).
  seeds.push({
    id: 'conv-11',
    title: 'Ambiguous request',
    summary: '',
    isActive: true,
    lastMsgAt: '2026-03-22',
    messages: [
      msg('conv-11-m1', 'user', 'Handle the Al Manar thing.'),
      msg('conv-11-m2', 'assistant', 'Did you mean the open RFQ or the overdue invoice for Al Manar Industrial Supplies?', {
        actionMetadata: withActions([
          action('clarify', 'rfq', 'Open RFQ', { prompt: 'Tell me more about the open RFQ for Al Manar Industrial Supplies' }),
          action('clarify', 'invoice', 'Overdue invoice', { prompt: 'Tell me more about the overdue invoice for Al Manar Industrial Supplies' }),
        ]),
      }),
    ],
  })

  // 12 — amounts huge / negative / zero on order-create chips.
  seeds.push({
    id: 'conv-12',
    title: 'Order amount edge cases',
    summary: '',
    isActive: true,
    lastMsgAt: '2026-02-18',
    messages: [
      msg('conv-12-m1', 'user', 'Create these three orders.'),
      msg('conv-12-m2', 'assistant', 'Three draft orders — review before confirming.', {
        actionMetadata: withActions([
          action('create', 'order', 'Create order (huge)', {
            order_number: 'ORD-9001',
            customer_name: 'Manama Energy Services',
            amount: 987654321098.765,
          }),
          action('create', 'order', 'Create order (negative)', {
            order_number: 'ORD-9002',
            customer_name: 'Manama Energy Services',
            amount: -450,
          }),
          action('create', 'order', 'Create order (zero)', {
            order_number: 'ORD-9003',
            customer_name: 'Manama Energy Services',
            amount: 0,
          }),
        ]),
      }),
    ],
  })

  // 13 — legacy singular action_* fields only (no action_metadata) — exercises
  // hydrateActionsFromMessage's fallback path.
  seeds.push({
    id: 'conv-13',
    title: 'Legacy action record',
    summary: '',
    isActive: false,
    lastMsgAt: '2025-09-04',
    messages: [
      msg('conv-13-m1', 'user', 'Approve the costing sheet.'),
      msg('conv-13-m2', 'assistant', "I'm ready to approve costing sheet #55.", {
        messageType: 'assistant_actionable',
        actionType: 'approve',
        actionTarget: 'costing_sheet',
        actionLabel: 'Approve costing sheet',
        actionData: JSON.stringify({ id: '55' }),
      }),
    ],
  })

  // 14 — RTL customer name + navigate-type chip (single-click).
  seeds.push({
    id: 'conv-14',
    title: 'شركة الخليج — عرض أسعار',
    summary: '',
    isActive: true,
    lastMsgAt: '2026-06-01',
    messages: [
      msg('conv-14-m1', 'user', 'Open the customer record.'),
      msg('conv-14-m2', 'assistant', 'Here is the customer record.', {
        actionMetadata: withActions([action('navigate', 'customer', 'Open customer', { customer_id: 'cust-6' })]),
      }),
    ],
  })

  // 15-18 — LCG filler: short, action-free Q&A conversations for volume.
  const rand = lcg(20260714 ^ 0x8a7e)
  const topics = [
    'exchange rate today',
    'overdue receivables',
    'supplier lead times',
    'open RFQs this week',
    'VAT return status',
    'warehouse stock levels',
  ]
  for (let i = 0; i < 6; i++) {
    const topic = topics[i % topics.length]!
    seeds.push({
      id: `conv-fill-${i + 1}`,
      title: `Question about ${topic}`,
      summary: '',
      isActive: rand() > 0.5,
      lastMsgAt: `2026-0${1 + (i % 9)}-1${i % 9}`,
      messages: [
        msg(`conv-fill-${i + 1}-m1`, 'user', `What's the latest on ${topic}?`),
        msg(`conv-fill-${i + 1}-m2`, 'assistant', `Here is the latest on ${topic}: everything looks normal as of today.`),
      ],
    })
  }

  return seeds
}

let seedCache: ConvSeed[] | null = null
function seeds(): ConvSeed[] {
  seedCache ??= buildSeeds()
  return seedCache
}

async function mockFetchConversations(): Promise<ButlerConversationRow[]> {
  await sleep(200)
  return seeds().map((s) => ({ id: s.id, title: s.title, summary: s.summary, isActive: s.isActive, lastMsgAt: s.lastMsgAt }))
}

async function mockFetchConversationMessages(id: string): Promise<ButlerChatMessageRow[]> {
  await sleep(150)
  const found = seeds().find((s) => s.id === id)
  return found ? [...found.messages] : []
}

async function mockFetchCustomerLookup(): Promise<LookupEntry[]> {
  await sleep(80)
  return [...seedCustomerLookup()]
}

async function mockFetchSupplierLookup(): Promise<LookupEntry[]> {
  await sleep(80)
  return [...seedSupplierLookup()]
}

async function mockResolveCustomerName(id: string): Promise<string> {
  await sleep(60)
  return seedCustomerLookup().find((c) => c.id === id)?.name ?? ''
}

async function mockResolveSupplierName(id: string): Promise<string> {
  await sleep(60)
  return seedSupplierLookup().find((s) => s.id === id)?.name ?? ''
}

async function mockSendMessage(conversationId: string, _text: string): Promise<ButlerChatResult> {
  await sleep(500)
  return {
    responseText: 'I have noted your request. Is there anything else?',
    conversationId: conversationId || `conv-new-${pad(Math.floor(Math.random() * 9999), 4)}`,
    actions: [],
    confidence: 0.85,
  }
}

async function mockDeleteConversation(id: string): Promise<void> {
  await sleep(120)
  const list = seeds()
  const idx = list.findIndex((c) => c.id === id)
  if (idx >= 0) list.splice(idx, 1)
}

async function mockPurgeAllConversations(): Promise<void> {
  await sleep(150)
  seeds().length = 0
}

/* ---- real: fetch is wired; mutations are honest INTEG-gap throws ---- */

async function realFetchConversations(): Promise<ButlerConversationRow[]> {
  const rows = await ListConversations()
  return (rows ?? []).map((c) => ({
    id: str(c.id),
    title: str(c.title),
    summary: str(c.summary),
    isActive: Boolean(c.is_active),
    lastMsgAt: goDate(c.last_msg_at),
  }))
}

async function realFetchConversationMessages(id: string): Promise<ButlerChatMessageRow[]> {
  const rows = await GetConversationMessages(id)
  return (rows ?? []).map((m) => ({
    id: str(m.id),
    role: str(m.role),
    content: str(m.content),
    messageType: str(m.message_type),
    actionType: str(m.action_type),
    actionTarget: str(m.action_target),
    actionLabel: str(m.action_label),
    actionData: str(m.action_data),
    actionMetadata: str(m.action_metadata),
  }))
}

let realCustomerLookupCache: LookupEntry[] = []
let realCustomerLookupAt = 0
let realSupplierLookupCache: LookupEntry[] = []
let realSupplierLookupAt = 0
const LOOKUP_TTL_MS = 5 * 60 * 1000

async function realFetchCustomerLookup(): Promise<LookupEntry[]> {
  const now = Date.now()
  if (realCustomerLookupCache.length > 0 && now - realCustomerLookupAt < LOOKUP_TTL_MS) return realCustomerLookupCache
  const rows = await ListCustomers(500, 0)
  realCustomerLookupCache = (rows ?? [])
    .map((c) => ({ id: str(c.id), name: str(c.business_name) }))
    .filter((c) => c.id && c.name)
  realCustomerLookupAt = now
  return realCustomerLookupCache
}

async function realFetchSupplierLookup(): Promise<LookupEntry[]> {
  const now = Date.now()
  if (realSupplierLookupCache.length > 0 && now - realSupplierLookupAt < LOOKUP_TTL_MS) return realSupplierLookupCache
  const rows = await ListSuppliers(500, 0)
  realSupplierLookupCache = (rows ?? [])
    .map((s) => ({ id: str(s.id), name: str(s.supplier_name) }))
    .filter((s) => s.id && s.name)
  realSupplierLookupAt = now
  return realSupplierLookupCache
}

async function realResolveCustomerName(id: string): Promise<string> {
  try {
    const c = await GetCustomer(id)
    return str(c.business_name)
  } catch {
    return ''
  }
}

async function realResolveSupplierName(id: string): Promise<string> {
  try {
    const s = await GetSupplier(id)
    return str(s.supplier_name)
  } catch {
    return ''
  }
}

async function realSendMessage(_conversationId: string, _text: string): Promise<ButlerChatResult> {
  throw new Error('INTEG gap: ChatWithButlerPersistent — wires at K5')
}

async function realDeleteConversation(_id: string): Promise<void> {
  throw new Error('INTEG gap: DeleteConversation — wires at K5')
}

async function realPurgeAllConversations(): Promise<void> {
  throw new Error('INTEG gap: PurgeAllConversations — wires at K5')
}

/** The single seam behind all 23 write-action bindings (CreateOfferDraftFromButler,
 * CreateFollowUp, CreateOrder, CreateRFQ, AddCustomerContact, AddSupplierContact,
 * CreateStockAdjustment, CreateCustomerFromButler, CreateSupplierFromButler,
 * ApprovePurchaseOrder, UpdatePOStatus, UpdateOpportunityStage, UpdateOpportunityDetails,
 * UpdateOrderStage, UpdateRFQStage, UpdateCostingSheet, UpdateOfferStatus,
 * ApproveStockAdjustment, MarkOfferWon, MarkOfferLost, ApproveSupplierInvoice,
 * DisputeSupplierInvoice, ApproveCostingSheet, RejectCostingSheet). butler-actions.ts
 * resolves which binding an armed+confirmed action maps to and calls this with its
 * name; it always throws — deliberately never simulated, mock or real (see file header). */
export async function executeButlerActionBinding(bindingName: string): Promise<never> {
  throw new Error(`INTEG gap: ${bindingName} — wires at K5`)
}

/* ---- public switched API (butler-actions.ts / butler-vm.svelte.ts import THESE) ---- */
export const fetchButlerConversations = (): Promise<ButlerConversationRow[]> =>
  pick(realFetchConversations, mockFetchConversations)()
export const fetchButlerConversationMessages = (id: string): Promise<ButlerChatMessageRow[]> =>
  pick(realFetchConversationMessages, mockFetchConversationMessages)(id)
export const fetchCustomerLookup = (): Promise<LookupEntry[]> => pick(realFetchCustomerLookup, mockFetchCustomerLookup)()
export const fetchSupplierLookup = (): Promise<LookupEntry[]> => pick(realFetchSupplierLookup, mockFetchSupplierLookup)()
export const resolveCustomerName = (id: string): Promise<string> => pick(realResolveCustomerName, mockResolveCustomerName)(id)
export const resolveSupplierName = (id: string): Promise<string> => pick(realResolveSupplierName, mockResolveSupplierName)(id)
export const sendButlerMessage = (conversationId: string, text: string): Promise<ButlerChatResult> =>
  pick(realSendMessage, mockSendMessage)(conversationId, text)
export const deleteButlerConversation = (id: string): Promise<void> => pick(realDeleteConversation, mockDeleteConversation)(id)
export const purgeAllButlerConversations = (): Promise<void> => pick(realPurgeAllConversations, mockPurgeAllConversations)()
