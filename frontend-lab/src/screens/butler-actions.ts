/* Butler action domain logic — ported faithfully from the old
 * ButlerScreen.svelte (2960 lines): LLM-output alias/target normalization,
 * payload validation -> chip runtime state, the arm/confirm hot-zone key,
 * and the write-action dispatch that resolves which of the 23 real Go
 * bindings an action maps to. Pure TypeScript (L5) — no Svelte, no DOM;
 * butler-vm.svelte.ts is the only caller. This is the "AI proposes, human
 * arms+confirms, deterministic backend executes" boundary: everything below
 * VALIDATES and ROUTES; only executeButlerAction ever calls the bridge, and
 * the bridge's write seam always throws (see bridge/butler.ts header) — the
 * boundary itself is the K5 gap, not a bug here.
 *
 * DROPPED from the old screen: the insights/butler:event feed (populated via
 * an EventsOn subscription, never rendered anywhere — dead code) and the
 * duplicate/unreachable switch cases in the old resolveActionTarget. */

import {
  executeButlerActionBinding,
  fetchCustomerLookup,
  fetchSupplierLookup,
  resolveCustomerName,
  resolveSupplierName,
  type ButlerChatMessageRow,
} from '../bridge/butler'
import type { Tone } from '$kernel/tones'

export const ACTION_CONFIRMATION_WINDOW_MS = 6000

/* ---- hydrated action shape ----
 * The wire JSON (action_metadata) is loose (see bridge/butler.ts's
 * RawActionJSON comment); this is what it narrows to for rendering + dispatch. */
export interface ResolvedButlerAction {
  type: string
  target: string
  label: string
  data: unknown
  requiresApproval: boolean
  /** Explicitly stored status, if the backend already computed one; '' means
   * "compute it live from data" (getActionRuntimeState's default path). */
  storedStatus: string
  missingFields: string[]
  invalidReason: string
}

export type ActionExecutionStatus = 'ready_for_execution' | 'needs_input' | 'needs_approval' | 'invalid_payload'

export interface ActionRuntimeState {
  status: ActionExecutionStatus
  missing: string[]
  reason: string
  mode: string
  target: string
}

/* ---- LLM-output alias normalization ---- */

const ACTION_TYPE_ALIAS: Record<string, string> = {
  create_offer: 'create',
  createoffer: 'create',
  create_offer_draft: 'create',
  create_quotation: 'create',
  createoffer_draft: 'create',
  createfollowup: 'create',
  create_follow_up: 'create',
  create_followup_task: 'create',
  create_stock_adjustment: 'create',
  create_stockadjustment: 'create',
  create_stock: 'create',
  createorder: 'create',
  create_order: 'create',
  createfollowuptask: 'create',
  daily_briefing: 'daily_briefing',
}

const ACTION_TARGET_ALIAS: Record<string, string> = {
  offer_draft: 'offer',
  offerdraft: 'offer',
  quotation: 'offer',
  quote: 'offer',
  opportunities: 'opportunity',
  customercontact: 'customer_contact',
  'customer contact': 'customer_contact',
  customer_contacts: 'customer_contact',
  suppliercontact: 'supplier_contact',
  'supplier contact': 'supplier_contact',
  supplier_contacts: 'supplier_contact',
  followup: 'follow_up',
  followup_task: 'follow_up',
  follow_up_task: 'follow_up',
  'follow-up task': 'follow_up',
  'follow up task': 'follow_up',
  followuptask: 'follow_up',
  stockadjustment: 'stock_adjustment',
  'stock-adjustment': 'stock_adjustment',
  stock_adjustments: 'stock_adjustment',
  stockadjust: 'stock_adjustment',
  'daily briefing': 'daily_briefing',
}

export function normalizeActionType(value: unknown): string {
  const normalized = String(value ?? '')
    .trim()
    .toLowerCase()
    .replace(/\s+/g, '_')
    .replace(/-/g, '_')
  return ACTION_TYPE_ALIAS[normalized] ?? normalized
}

export function resolveActionTarget(rawTarget: unknown): string {
  const target = String(rawTarget ?? '').trim().toLowerCase()
  if (ACTION_TARGET_ALIAS[target]) return ACTION_TARGET_ALIAS[target]!
  switch (target) {
    case 'offer_draft':
    case 'offerdraft':
      return 'offer_draft'
    case 'offer':
    case 'offers':
    case 'quotation':
    case 'quote':
    case 'quotations':
      return 'offer'
    case 'po':
    case 'purchaseorder':
    case 'purchase_orders':
    case 'purchase-order':
      return 'purchase_order'
    case 'invoic':
    case 'invoice':
    case 'invoices':
      return 'invoice'
    case 'follow-up':
    case 'followup':
    case 'followup_task':
    case 'follow_up_task':
    case 'follow up task':
    case 'follow-up task':
    case 'task':
    case 'tasks':
      return 'follow_up'
    case 'rfq':
      return 'rfq'
    case 'costingsheet':
    case 'costings':
    case 'costingsheets':
    case 'costing sheet':
      return 'costing_sheet'
    case 'supplierinvoice':
    case 'supplier_invoice':
      return 'supplier_invoice'
    case 'stock_adjustment':
    case 'stockadjustment':
    case 'stock-adjustment':
    case 'stock adjustment':
      return 'stock_adjustment'
    case 'order':
    case 'orders':
      return 'order'
    case 'opportunity':
    case 'opportunities':
      return 'opportunity'
    case 'customer':
    case 'customers':
      return 'customer'
    case 'customer_contact':
    case 'customer contacts':
    case 'customer contact':
      return 'customer_contact'
    case 'supplier':
    case 'suppliers':
      return 'supplier'
    case 'supplier_contact':
    case 'supplier contacts':
    case 'supplier contact':
      return 'supplier_contact'
    case 'contact':
    case 'contacts':
      return 'contact'
    case 'finance':
      return 'finance'
    case 'operations':
      return 'operations'
    case 'dashboard':
    case 'home':
      return 'dashboard'
    default:
      return target || ''
  }
}

function getActionWorkflowMode(type: string, target: string): string {
  if (type === 'approve' || type === 'reject') return type
  if (type === 'update') return 'update'
  if (type === 'create') {
    if (target === 'follow_up') return 'create_follow_up'
    if (target === 'offer' || target === 'quotation') return 'create_offer_draft'
    if (target === 'order') return 'create_order'
    if (target === 'opportunity') return 'create_opportunity'
    if (target === 'customer_contact') return 'create_customer_contact'
    if (target === 'supplier_contact') return 'create_supplier_contact'
    if (target === 'stock_adjustment') return 'create_stock_adjustment'
    return 'create'
  }
  return type || 'workflow'
}

export function getActionWorkflowKey(action: { type: unknown; target: unknown }): string {
  const type = normalizeActionType(action.type)
  const target = resolveActionTarget(action.target)
  if (type === 'daily_briefing') return 'daily_briefing'
  if (type === 'approve' || type === 'reject') return `${type}_action`
  if (type === 'update') return `update_${target}`
  if (type === 'create') {
    if (target === 'follow_up') return 'create_follow_up'
    if (target === 'offer') return 'create_offer_draft'
    if (target === 'order') return 'create_order'
    if (target === 'opportunity') return 'create_opportunity'
    if (target === 'customer_contact') return 'create_customer_contact'
    if (target === 'supplier_contact') return 'create_supplier_contact'
    if (target === 'contact') return 'create_contact'
    if (target === 'stock_adjustment') return 'create_stock_adjustment'
    return 'create'
  }
  if (type === 'navigate' || type === 'open') return `open_${target || 'screen'}`
  if (type === 'analyze' || type === 'fetch') return type
  return type
}

/* ---- value coercion ---- */

function normalizeToPlainText(value: unknown): string {
  if (value === null || value === undefined) return ''
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') return String(value)
  return ''
}

export function parseNumeric(value: unknown, fallback = 0): number {
  const raw = Number(normalizeToPlainText(value))
  return Number.isFinite(raw) ? raw : fallback
}

export function toActionIdValue(value: unknown): string {
  const raw = String(value ?? '').trim()
  if (!raw) return ''
  if (/^\d+$/.test(raw)) return raw
  const parsed = Number(raw)
  return Number.isFinite(parsed) ? String(parsed) : raw
}

export function toNumericId(value: unknown): number | null {
  const n = Number(toActionIdValue(value))
  return Number.isFinite(n) ? n : null
}

function toStringArray(raw: unknown): string[] {
  if (!raw) return []
  if (Array.isArray(raw)) return raw.map((v) => String(v ?? '').trim()).filter(Boolean)
  if (typeof raw === 'string') return raw.split(',').map((v) => v.trim()).filter(Boolean)
  return []
}

function toBoolean(value: unknown): boolean {
  if (typeof value === 'boolean') return value
  if (typeof value === 'number') return value === 1
  if (typeof value === 'string') {
    const raw = value.trim().toLowerCase()
    return raw === '1' || raw === 'true' || raw === 'yes'
  }
  return false
}

/* ---- action data access ---- */

export function getActionDataObject(action: { data: unknown }): Record<string, unknown> {
  return typeof action.data === 'object' && action.data !== null ? (action.data as Record<string, unknown>) : {}
}

export function summarizeActionData(data: unknown): string {
  if (!data) return ''
  if (typeof data === 'string') return data.trim()
  if (typeof data !== 'object') return String(data).trim()

  const obj = data as Record<string, unknown>
  const pick = (...keys: string[]): string => {
    for (const key of keys) {
      const value = obj[key]
      if (value !== undefined && value !== null && String(value).trim()) return String(value).trim()
    }
    return ''
  }
  const pieces: string[] = []
  const customer = pick('customer', 'customer_name', 'customerName', 'company', 'account')
  const opportunity = pick('opportunity', 'opportunity_name', 'opportunityName', 'title', 'subject')
  const date = pick('date', 'briefing_date', 'day', 'for_date', 'period', 'range')
  const id = pick('id', 'offer_id', 'offerId', 'customer_id', 'customerId')
  if (customer) pieces.push(customer)
  if (opportunity) pieces.push(opportunity)
  if (date) pieces.push(date)
  if (id) pieces.push(`ID ${id}`)

  const consumed = new Set([
    'customer', 'customer_name', 'customerName', 'company', 'account',
    'opportunity', 'opportunity_name', 'opportunityName', 'title', 'subject',
    'date', 'briefing_date', 'day', 'for_date', 'period', 'range',
    'id', 'offer_id', 'offerId', 'customer_id', 'customerId',
  ])
  for (const [key, value] of Object.entries(obj)) {
    if (value === undefined || value === null || typeof value !== 'string') continue
    const trimmed = value.trim()
    if (!trimmed || consumed.has(key)) continue
    pieces.push(`${key}: ${trimmed}`)
  }
  return pieces.join(', ')
}

export function actionLabelOrFallback(action: { label: unknown }, fallback = 'Action'): string {
  const label = String(action.label ?? '').trim()
  return label || fallback
}

export function convTitle(title: string): string {
  return title.trim() || 'Untitled'
}

/* ---- status constraints (legal state machine per target) ---- */

const STATUS_CONSTRAINTS: Record<string, string[]> = {
  opportunity: ['New', 'Qualified', 'Proposal', 'Quoted', 'Won', 'Lost', 'On Hold'],
  purchase_order: ['Draft', 'Pending Approval', 'Approved', 'Sent', 'Acknowledged', 'Partially Received', 'Received', 'Closed', 'Cancelled'],
  order: ['Draft', 'Confirmed', 'Processing', 'InProgress', 'Shipped', 'PartiallyDelivered', 'FullyDelivered', 'Delivered', 'Invoiced', 'Complete', 'Cancelled'],
  rfq: ['RFQ Received', 'Offer Sent', 'Follow-up/Eval', 'PO/LOI Received', 'Order Placed', 'In Process', 'Delivered', 'Closed (Payment)', 'Closed (Lost)'],
  costing_sheet: ['draft', 'pending_approval', 'approved', 'rejected'],
  offer: ['draft', 'quoted', 'sent', 'accepted', 'rejected', 'won', 'lost'],
  quotation: ['draft', 'quoted', 'sent', 'accepted', 'rejected', 'won', 'lost'],
  follow_up: ['pending', 'in_progress', 'completed', 'cancelled', 'overdue'],
  stock_adjustment: ['pending', 'approved', 'rejected'],
}

function normalizeStatusValue(status: unknown): string {
  return String(status ?? '').trim()
}

function isStatusAllowedForTarget(target: string, status: string): boolean {
  const normalizedStatus = normalizeStatusValue(status).toLowerCase().replace(/[-\s]+/g, '')
  if (!normalizedStatus) return true
  const allowed = STATUS_CONSTRAINTS[target.toLowerCase()]
  if (!allowed || allowed.length === 0) return true
  return allowed.some((c) => c.toLowerCase().replace(/[-\s]+/g, '') === normalizedStatus)
}

function parseStatusPayload(action: { data: unknown }): string {
  const data = getActionDataObject(action)
  return toActionIdValue(
    data.status ?? data.stage ?? data.new_status ?? data.new_stage ?? data.target_status ?? data.stage_to ?? data.approved_to ?? data.to,
  )
}

/* ---- payload validation -> the source of truth for chip readiness ---- */

export interface ActionValidation {
  type: string
  target: string
  id: string
  statusValue: string
  reason: string
  missing: string[]
  allowedStatuses: string[]
  data: Record<string, unknown>
}

export function validateActionPayload(action: { type: unknown; target: unknown; data: unknown }, mode: string): ActionValidation {
  const target = resolveActionTarget(action.target).toLowerCase()
  const data = getActionDataObject(action)
  const id =
    toActionIdValue(
      data.entity_id ?? data.id ?? data.offer_id ?? data.order_id ?? data.purchase_order_id ??
        data.supplier_invoice_id ?? data.costing_sheet_id ?? data.stock_adjustment_id,
    ) || ''
  const statusValue = parseStatusPayload(action)
  const actionType = String(action.type ?? '').toLowerCase()
  const reason = String(data.reason ?? data.note ?? data.notes ?? data.rejection_reason ?? '').trim()
  const missing: string[] = []
  const allowedStatuses = STATUS_CONSTRAINTS[target] ?? []

  if (mode === 'approve') {
    if (!id) missing.push('entity id')
  }

  if (mode === 'update') {
    if (!id) missing.push('entity id')
    if (target === 'opportunity') {
      const comment = String(data.comment ?? data.notes ?? data.description ?? '').trim()
      const ownerNotes = String(data.owner_notes ?? data.ownerNotes ?? '').trim()
      if (!statusValue && !comment && !ownerNotes) {
        missing.push('stage/status or comment/owner_notes')
      } else if (statusValue && !isStatusAllowedForTarget(target, statusValue)) {
        missing.push(`valid status/stage for ${target}${allowedStatuses.length ? ` Expected: ${allowedStatuses.join(', ')}` : ''}`)
      }
    } else if (!statusValue) {
      missing.push('status/stage')
    } else if (!isStatusAllowedForTarget(target, statusValue)) {
      missing.push(`valid status/stage for ${target}${allowedStatuses.length ? ` Expected: ${allowedStatuses.join(', ')}` : ''}`)
    }
  }

  const lineItemCount = Array.isArray(data.line_items)
    ? data.line_items.length
    : Array.isArray(data.lineItems)
      ? data.lineItems.length
      : Array.isArray(data.items)
        ? data.items.length
        : 0
  const amount = parseNumeric(data.grand_total ?? data.total ?? data.total_amount ?? data.amount ?? data.amount_bhd ?? data.value, 0)

  const requireCustomer = () => {
    if (
      !toActionIdValue(data.customer_id ?? data.customerId ?? data.customer_id_text ?? data.customerId_text) &&
      !toActionIdValue(data.customer_name ?? data.customerName ?? data.customer)
    ) {
      missing.push('customer')
    }
  }

  if (mode === 'create' && target === 'offer') {
    requireCustomer()
    if (!lineItemCount && (!Number.isFinite(amount) || amount <= 0)) missing.push('amount')
  } else if (mode === 'create' && target === 'follow_up') {
    requireCustomer()
    if (!toActionIdValue(data.title ?? data.subject)) missing.push('follow-up title')
  } else if (mode === 'create' && target === 'order') {
    if (!toActionIdValue(data.order_number ?? data.orderNumber ?? data.reference)) missing.push('order number')
    if (!toActionIdValue(data.customer_name ?? data.customer ?? data.customer_id ?? data.customerId ?? data.customer_id_text)) missing.push('customer')
    const amountValue = parseNumeric(data.amount ?? data.total_amount ?? data.totalAmount ?? data.amount_bhd ?? data.value, 0)
    if (!Number.isFinite(amountValue) || amountValue <= 0) missing.push('amount')
  } else if (mode === 'create' && target === 'opportunity') {
    if (!toActionIdValue(data.customer_name ?? data.customer ?? data.customer_id ?? data.customerId ?? data.customer_id_text)) missing.push('customer')
    if (!toActionIdValue(data.title ?? data.project ?? data.opportunity_name ?? data.name)) missing.push('project/title')
  } else if (mode === 'create' && (target === 'customer_contact' || target === 'contact')) {
    if (
      !toActionIdValue(data.customer_name ?? data.customer ?? data.customer_id ?? data.customerId ?? data.customer_id_text) &&
      !toActionIdValue(data.supplier_name ?? data.supplier ?? data.supplier_id ?? data.supplierId ?? data.supplier_id_text)
    ) {
      missing.push('customer or supplier')
    }
    if (!toActionIdValue(data.contact_name ?? data.name ?? data.person ?? data.primary_contact)) missing.push('contact name')
  } else if (mode === 'create' && target === 'supplier_contact') {
    if (!toActionIdValue(data.supplier_name ?? data.supplier ?? data.supplier_id ?? data.supplierId ?? data.supplier_id_text)) missing.push('supplier')
    if (!toActionIdValue(data.contact_name ?? data.name ?? data.person ?? data.primary_contact)) missing.push('contact name')
  } else if (mode === 'create' && target === 'stock_adjustment') {
    const inventoryItem = toActionIdValue(data.inventory_item_id ?? data.item_id ?? data.inventoryItemId ?? data.itemId ?? data.item_code ?? data.inventoryItem)
    const stockReason = String(data.reason ?? data.notes ?? '').trim()
    const variance = parseNumeric(data.variance, NaN)
    const systemQuantity = parseNumeric(data.system_quantity, NaN)
    const physicalQuantity = parseNumeric(data.physical_quantity, NaN)
    if (!inventoryItem) missing.push('inventory item id')
    if (!stockReason) missing.push('reason')
    if (!Number.isFinite(variance) && (!Number.isFinite(systemQuantity) || !Number.isFinite(physicalQuantity))) {
      missing.push('variance/system_quantity/physical_quantity')
    }
  } else if (mode === 'create' && !target) {
    missing.push('target')
  }

  return { type: actionType, target, id, statusValue, reason, missing, allowedStatuses, data }
}

function getActionRuntimeState(action: ResolvedButlerAction): ActionRuntimeState {
  const type = normalizeActionType(action.type)
  const target = resolveActionTarget(action.target)
  const mode = getActionWorkflowMode(type, target)

  if (
    action.storedStatus &&
    (['ready_for_execution', 'pending_execution', 'needs_input', 'needs_approval', 'invalid_payload'] as string[]).includes(
      action.storedStatus,
    )
  ) {
    return { status: action.storedStatus as ActionExecutionStatus, missing: action.missingFields, reason: action.invalidReason, mode, target }
  }

  const validation = validateActionPayload(action, mode)
  let status: ActionExecutionStatus = 'ready_for_execution'
  if (action.requiresApproval) status = 'needs_approval'
  if (validation.missing.length > 0) status = 'needs_input'

  return {
    status,
    missing: action.missingFields.length > 0 ? action.missingFields : validation.missing,
    reason: action.invalidReason || (validation.missing.length > 0 ? `Missing: ${validation.missing.join(', ')}` : ''),
    mode,
    target,
  }
}

export function actionChipStatusLabel(status: ActionExecutionStatus | string): string {
  switch (status) {
    case 'pending_execution':
    case 'ready_for_execution':
      return 'Ready'
    case 'needs_input':
      return 'Needs input'
    case 'needs_approval':
      return 'Needs approval'
    case 'invalid_payload':
      return 'Invalid'
    default:
      return 'Review'
  }
}

export function chipToneForStatus(status: ActionExecutionStatus | string): Tone {
  switch (status) {
    case 'ready_for_execution':
    case 'pending_execution':
      return 'success'
    case 'needs_approval':
      return 'warning'
    case 'needs_input':
    case 'invalid_payload':
      return 'danger'
    default:
      return 'neutral'
  }
}

/** Public: runtime state + presentation, one call for the VM's chip mapper. */
export function describeAction(action: ResolvedButlerAction): { state: ActionRuntimeState; tone: Tone; statusLabel: string; disabled: boolean } {
  const type = normalizeActionType(action.type)
  if (type === 'clarify') {
    return { state: { status: 'ready_for_execution', missing: [], reason: '', mode: 'clarify', target: '' }, tone: 'info', statusLabel: 'Choose', disabled: false }
  }
  const state = getActionRuntimeState(action)
  return {
    state,
    tone: chipToneForStatus(state.status),
    statusLabel: actionChipStatusLabel(state.status),
    disabled: state.status === 'needs_input' || state.status === 'invalid_payload',
  }
}

/* ---- hydration from raw message rows ---- */

function parseJsonSafe(raw: string): unknown {
  if (!raw || !raw.trim()) return null
  try {
    return JSON.parse(raw)
  } catch {
    return null
  }
}

function normalizeStoredAction(raw: unknown): ResolvedButlerAction | null {
  if (!raw || typeof raw !== 'object') return null
  const a = raw as Record<string, unknown>
  const type = String(a.type ?? a.Type ?? '').toLowerCase().trim()
  if (!type) return null

  const rawData = a.data ?? a.parameters ?? a.payload ?? a.rawData
  const parsedData =
    typeof rawData === 'string'
      ? parseJsonSafe(rawData) ?? {}
      : rawData && typeof rawData === 'object'
        ? rawData
        : rawData ?? {}
  const data =
    parsedData && typeof parsedData === 'object' && 'payload' in (parsedData as Record<string, unknown>)
      ? (parsedData as Record<string, unknown>).payload
      : parsedData

  return {
    type: normalizeActionType(type),
    target: resolveActionTarget(a.target ?? a.Target ?? a.entity ?? a.Entity ?? a.entity_type ?? a.EntityType ?? a.target_type ?? a.targetType),
    label: String(a.label ?? a.action_label ?? a.Label ?? a.ActionLabel ?? a.name ?? a.Name ?? '').trim(),
    data: data ?? {},
    requiresApproval: toBoolean(a.requires_approval ?? a.requiresApproval),
    storedStatus: String(a.execution_status ?? a.executionStatus ?? a.status ?? '').trim().toLowerCase(),
    missingFields: toStringArray(a.missing_fields ?? a.missingFields),
    invalidReason: String(a.invalid_reason ?? a.invalidReason ?? a.reason ?? '').trim(),
  }
}

/** Priority order mirrors the old screen: structured action_metadata first,
 * then a JSON action_data list/object, then the legacy singular action_*
 * columns as a last resort (conv-13 in the mock exercises this fallback). */
export function hydrateActionsFromMessage(row: ButlerChatMessageRow): ResolvedButlerAction[] {
  if (row.messageType !== 'assistant_actionable') return []

  const metadataParsed = parseJsonSafe(row.actionMetadata) as Record<string, unknown> | null
  const metadataActions = Array.isArray(metadataParsed?.actions) ? metadataParsed!.actions : []
  if (metadataActions.length > 0) {
    const normalized = metadataActions.map(normalizeStoredAction).filter((a): a is ResolvedButlerAction => a !== null)
    if (normalized.length > 0) return normalized
  }

  const dataParsed = parseJsonSafe(row.actionData)
  const list = Array.isArray(dataParsed) ? dataParsed : Array.isArray((dataParsed as Record<string, unknown>)?.actions) ? (dataParsed as Record<string, unknown>).actions : dataParsed && typeof dataParsed === 'object' ? [dataParsed] : []
  const normalizedList = (list as unknown[]).map(normalizeStoredAction).filter((a): a is ResolvedButlerAction => a !== null)
  if (normalizedList.length > 0) return normalizedList

  if (!row.actionType) return []
  return [
    {
      type: normalizeActionType(row.actionType),
      target: resolveActionTarget(row.actionTarget),
      label: row.actionLabel.trim() || `Action: ${row.actionType}`,
      data: toActionIdValue(row.actionData),
      requiresApproval: false,
      storedStatus: '',
      missingFields: [],
      invalidReason: '',
    },
  ]
}

/* ---- arm/confirm hot-zone ---- */

export function isWriteAction(action: { type: unknown }): boolean {
  const type = normalizeActionType(action.type)
  return type === 'create' || type === 'update' || type === 'approve' || type === 'reject'
}

/** Single global arm slot key — content-addressed, not chip-instance-addressed,
 * so arming any chip that resolves to the SAME logical action also arms every
 * other rendering of it (matches the old screen's pendingActionKey). */
export function actionExecutionKey(action: { type: unknown; target: unknown; label: unknown; data: unknown }): string {
  return JSON.stringify({
    type: normalizeActionType(action.type),
    target: resolveActionTarget(action.target),
    label: actionLabelOrFallback(action, 'Action'),
    data: getActionDataObject(action),
  })
}

export function buildActionPreview(action: ResolvedButlerAction): string {
  const state = getActionRuntimeState(action)
  const type = normalizeActionType(action.type)
  const target = resolveActionTarget(action.target)
  const summary = summarizeActionData(getActionDataObject(action))
  const intro =
    type === 'create'
      ? "I'm ready to create this record."
      : type === 'update'
        ? "I'm ready to update this record."
        : type === 'approve'
          ? "I'm ready to approve this action."
          : type === 'reject'
            ? "I'm ready to reject this action."
            : "I'm ready to run this Butler action."
  const details = summary ? ` Details: ${summary}.` : ''
  const missing = state.missing.length ? ` Missing: ${state.missing.join(', ')}.` : ''
  return `${intro} ${actionLabelOrFallback(action, target || 'Action')}.${details}${missing} Click the same action again to confirm.`
}

export function buildWorkflowPrompt(workflow: string, action: { data?: unknown } = {}): string {
  const summary = summarizeActionData(action.data)
  const withSummary = (yes: string, no: string) => (summary ? yes : no)

  switch (workflow) {
    case 'create_offer_draft':
      return withSummary(
        `Create an offer draft for ${summary}. Include recommended line items, commercial terms, and flag any missing details before finalizing.`,
        'Create an offer draft. Use the current Butler context and ask for any missing customer, product, quantity, or pricing details before finalizing.',
      )
    case 'create_follow_up':
      return withSummary(`Create this follow-up from Butler data: ${summary}.`, 'Create a follow-up task with a clear customer, title, and due date.')
    case 'create_order':
      return withSummary(
        `Create an order for ${summary}. Validate required fields and confirmation details before saving.`,
        'Create an order from Butler data with required order number, customer, and amount.',
      )
    case 'create_opportunity':
      return withSummary(
        `Create an opportunity for ${summary}. Ask for any missing customer, project, or reference details before finalizing.`,
        'Create an opportunity from Butler data with a customer and project/title. Ask for missing details before finalizing.',
      )
    case 'create_customer_contact':
      return withSummary(`Create a customer contact for ${summary}. Confirm the customer and contact details before saving.`, 'Create a customer contact from Butler data. I need the customer and contact details.')
    case 'create_supplier_contact':
    case 'create_contact':
      return withSummary(`Create a contact for ${summary}. Confirm the parent company and contact details before saving.`, 'Create a supplier or customer contact from Butler data. I need the parent company and contact details.')
    case 'create_stock_adjustment':
      return withSummary(`Create a stock adjustment for ${summary}.`, 'Create a stock adjustment action from Butler data. I need inventory item, reason, and quantity variance.')
    case 'daily_briefing':
      return withSummary(`Generate the daily briefing for ${summary}. Summarize priorities, risks, and next actions in a concise briefing format.`, "Generate today's daily briefing. Summarize priorities, risks, and next actions in a concise briefing format.")
    default:
      return summary ? `Proceed with ${workflow} for ${summary}.` : `Proceed with ${workflow.replace(/_/g, ' ')}.`
  }
}

/* ---- customer/supplier identity resolution (name<->id, TTL-cached in the bridge) ---- */

function normalizeLookupText(value: string): string {
  return value
    .toLowerCase()
    .replace(/\b(ltd|llc|inc|co|pvt|ltd\.|inc\.|co\.)\b/g, '')
    .replace(/[^a-z0-9 ]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

async function resolveIdFromHint(hint: string, lookup: { id: string; name: string }[]): Promise<string> {
  const direct = toActionIdValue(hint)
  if (direct && direct === hint.trim() && !/\s/.test(direct)) return direct
  const normalizedHint = normalizeLookupText(hint)
  if (!normalizedHint) return ''
  const exact = lookup.find((item) => normalizeLookupText(item.name) === normalizedHint)
  if (exact) return exact.id
  const partial = lookup.find((item) => {
    const n = normalizeLookupText(item.name)
    return n.includes(normalizedHint) || normalizedHint.includes(n)
  })
  return partial?.id ?? ''
}

export async function resolveActionCustomerIdentity(action: { data: unknown }): Promise<{ customerId: string; customerName: string }> {
  const data = getActionDataObject(action)
  const idHint = toActionIdValue(data.customer_id ?? data.customerId ?? data.customer_id_text ?? data.customerId_text ?? data.customer ?? data.customer_name ?? data.customerName)
  const nameHint = normalizeToPlainText(data.customer_name ?? data.customerName ?? data.customer ?? data.contact ?? data.client)

  let customerId = ''
  if (idHint) customerId = idHint
  else if (nameHint) customerId = await resolveIdFromHint(nameHint, await fetchCustomerLookup())

  let customerName = nameHint
  if (!customerName && customerId) customerName = await resolveCustomerName(customerId)

  return { customerId: toActionIdValue(customerId), customerName }
}

export async function resolveActionSupplierIdentity(action: { data: unknown }): Promise<{ supplierId: string; supplierName: string }> {
  const data = getActionDataObject(action)
  const idHint = toActionIdValue(data.supplier_id ?? data.supplierId ?? data.supplier_id_text ?? data.supplierId_text ?? data.supplier ?? data.supplier_name ?? data.supplierName)
  const nameHint = normalizeToPlainText(data.supplier_name ?? data.supplierName ?? data.supplier ?? data.vendor)

  let supplierId = ''
  if (idHint) supplierId = idHint
  else if (nameHint) supplierId = await resolveIdFromHint(nameHint, await fetchSupplierLookup())

  let supplierName = nameHint
  if (!supplierName && supplierId) supplierName = await resolveSupplierName(supplierId)

  return { supplierId: toActionIdValue(supplierId), supplierName }
}

/* ---- execution: resolve which of the 23 bindings, then call the always-throwing seam ----
 * PRESERVED VERBATIM refuse-over-guess guards from the old screen:
 *  - MarkOfferWon refuses if customer_po is missing (never substitutes a literal).
 *  - stock_adjustment "update" supports ONLY the approval transition; every
 *    other status is explicitly refused, not silently attempted. */

export interface ExecutionOutcome {
  message: string
}

interface Refusal {
  refusal: string
}
interface Resolved {
  bindingName: string
  verb: string
}

async function resolveCreateAction(action: ResolvedButlerAction): Promise<Refusal | Resolved> {
  const target = resolveActionTarget(action.target)
  const data = getActionDataObject(action)

  if (target === 'follow_up') {
    const v = validateActionPayload(action, 'create_follow_up')
    if (v.missing.length) return { refusal: `I can create this follow-up only with: ${v.missing.join(', ')}.` }
    return { bindingName: 'CreateFollowUp', verb: 'create the follow-up' }
  }
  if (target === 'offer' || target === 'offer_draft' || target === 'quotation') {
    const v = validateActionPayload(action, 'create_offer_draft')
    if (v.missing.length) return { refusal: `I can create this offer draft only with: ${v.missing.join(', ')}.` }
    return { bindingName: 'CreateOfferDraftFromButler', verb: 'create the offer draft' }
  }
  if (target === 'order' || target === 'orders') {
    const identity = await resolveActionCustomerIdentity(action)
    const orderNumber = toActionIdValue(data.order_number ?? data.orderNumber ?? data.reference ?? data.order_no).trim()
    const amount = parseNumeric(data.amount ?? data.total_amount ?? data.totalAmount ?? data.amount_bhd, Number.NaN)
    if (!orderNumber || Number.isNaN(amount) || amount <= 0 || (!identity.customerName && !identity.customerId)) {
      return { refusal: 'I can create this order only with an order number, amount and customer (name or id). Please add missing fields and run the action again.' }
    }
    return { bindingName: 'CreateOrder', verb: 'create that order' }
  }
  if (target === 'opportunity' || target === 'opportunities') {
    const identity = await resolveActionCustomerIdentity(action)
    const customerName = identity.customerName || String(data.customer_name ?? data.customer ?? '').trim()
    const project = String(data.project ?? data.title ?? data.opportunity_name ?? data.name ?? '').trim()
    if (!customerName || !project) {
      return { refusal: 'I can create this opportunity only with a customer and project/title. Please provide the missing details.' }
    }
    // Old screen calls CheckDuplicateOpportunity first, then creates via CreateRFQ
    // (legacy naming — opportunities ARE RFQs pre-offer). Both are INTEG-gapped;
    // CreateRFQ is the binding actually named per Butler.parity.md.
    return { bindingName: 'CreateRFQ', verb: 'create that opportunity' }
  }
  if (target === 'customer_contact' || target === 'contact') {
    const hasSupplierHint = toActionIdValue(data.supplier_id ?? data.supplierId ?? data.supplier_name ?? data.supplier ?? data.vendor)
    if (target === 'contact' && hasSupplierHint) {
      const identity = await resolveActionSupplierIdentity(action)
      if (!identity.supplierId && !identity.supplierName) return { refusal: 'I need the supplier record before I can add this contact. Please specify the supplier.' }
      if (!toActionIdValue(data.contact_name ?? data.name ?? data.person ?? data.primary_contact)) return { refusal: 'I need the contact name before I can create this supplier contact.' }
      return { bindingName: 'AddSupplierContact', verb: 'create the supplier contact' }
    }
    const identity = await resolveActionCustomerIdentity(action)
    if (!identity.customerId && !identity.customerName) return { refusal: 'I need the customer record before I can add this contact. Please specify the customer.' }
    if (!toActionIdValue(data.contact_name ?? data.name ?? data.person ?? data.primary_contact)) return { refusal: 'I need the contact name before I can create this customer contact.' }
    return { bindingName: 'AddCustomerContact', verb: 'create the customer contact' }
  }
  if (target === 'supplier_contact') {
    const identity = await resolveActionSupplierIdentity(action)
    if (!identity.supplierId && !identity.supplierName) return { refusal: 'I need the supplier record before I can add this contact. Please specify the supplier.' }
    if (!toActionIdValue(data.contact_name ?? data.name ?? data.person ?? data.primary_contact)) return { refusal: 'I need the contact name before I can create this supplier contact.' }
    return { bindingName: 'AddSupplierContact', verb: 'create the supplier contact' }
  }
  if (target === 'stock_adjustment') {
    const v = validateActionPayload(action, 'create_stock_adjustment')
    if (v.missing.length) return { refusal: `I can create this stock adjustment only with: ${v.missing.join(', ')}.` }
    return { bindingName: 'CreateStockAdjustment', verb: 'create the stock adjustment' }
  }
  if (target === 'customer' || target === 'customers') {
    if (!String(data.business_name ?? data.businessName ?? data.customer_name ?? data.name ?? '').trim()) {
      return { refusal: 'I need a business name to create a customer. Please provide the company name.' }
    }
    return { bindingName: 'CreateCustomerFromButler', verb: 'create the customer' }
  }
  if (target === 'supplier' || target === 'suppliers') {
    if (!String(data.supplier_name ?? data.supplierName ?? data.name ?? '').trim()) {
      return { refusal: 'I need a supplier name to create a supplier. Please provide the company name.' }
    }
    return { bindingName: 'CreateSupplierFromButler', verb: 'create the supplier' }
  }
  return { refusal: 'I can create offer drafts, follow-ups, orders, opportunities, contacts, stock adjustments, customers, and suppliers from Butler actions. Please refine this action with the required details.' }
}

async function resolveUpdateAction(action: ResolvedButlerAction): Promise<Refusal | Resolved> {
  const target = resolveActionTarget(action.target)
  const updateData = getActionDataObject(action)
  const id = toActionIdValue(updateData.id ?? updateData.entity_id)
  const statusValue = parseStatusPayload(action)
  if (!id) return { refusal: `This update action needs an entity id.` }

  if (target === 'purchase_order') {
    if (!statusValue) return { refusal: `I need a status or stage value to update purchase_order ${id}.` }
    return { bindingName: 'UpdatePOStatus', verb: 'update that purchase order' }
  }
  if (target === 'opportunity') {
    const data = getActionDataObject(action)
    const comment = String(data.comment ?? data.notes ?? data.description ?? '').trim()
    const ownerNotes = String(data.owner_notes ?? data.ownerNotes ?? '').trim()
    if (!statusValue && !comment && !ownerNotes) return { refusal: `I need a stage/status or note update to modify opportunity ${id}.` }
    return { bindingName: statusValue ? 'UpdateOpportunityStage + UpdateOpportunityDetails' : 'UpdateOpportunityDetails', verb: 'update that opportunity' }
  }
  if (target === 'order') {
    if (!statusValue) return { refusal: `I need a status or stage value to update order ${id}.` }
    if (toNumericId(id) === null) return { refusal: 'Order id must be numeric.' }
    return { bindingName: 'UpdateOrderStage', verb: 'update that order' }
  }
  if (target === 'rfq') {
    if (!statusValue) return { refusal: `I need a status or stage value to update rfq ${id}.` }
    if (toNumericId(id) === null) return { refusal: 'RFQ id must be numeric.' }
    return { bindingName: 'UpdateRFQStage', verb: 'update that RFQ' }
  }
  if (target === 'costing_sheet') {
    if (!statusValue) return { refusal: `I need a status or stage value to update costing_sheet ${id}.` }
    if (toNumericId(id) === null) return { refusal: 'Costing sheet id must be numeric.' }
    return { bindingName: 'UpdateCostingSheet', verb: 'update that costing sheet' }
  }
  if (target === 'offer' || target === 'quotation') {
    if (!statusValue) return { refusal: `I need a status or stage value to update ${target} ${id}.` }
    if (toNumericId(id) === null) return { refusal: 'Offer id must be numeric.' }
    return { bindingName: 'UpdateOfferStatus', verb: `update that ${target}` }
  }
  if (target === 'stock_adjustment') {
    if (toNumericId(id) === null) return { refusal: 'Stock adjustment id must be numeric.' }
    // PRESERVED VERBATIM: stock-adjustment "update" supports ONLY approval —
    // every other status transition is explicitly refused, not attempted.
    if (statusValue.toLowerCase() !== 'approved') {
      return { refusal: `Stock adjustment update only supports approval action for now. This status change to "${statusValue}" is not supported.` }
    }
    return { bindingName: 'ApproveStockAdjustment', verb: 'approve that stock adjustment' }
  }
  return { refusal: `No update execution path is configured for target '${target}'.` }
}

async function resolveApprovalAction(action: ResolvedButlerAction): Promise<Refusal | Resolved> {
  const actionType = normalizeActionType(action.type)
  const v = validateActionPayload(action, 'approve')
  const target = v.target
  const id = v.id
  if (!id) return { refusal: `This ${actionType} action is missing entity id for ${target}.` }
  if (v.missing.length > 0) return { refusal: `I need: ${v.missing.join(', ')} to execute this ${actionType} action.` }

  if (target === 'purchase_order') {
    return actionType === 'approve'
      ? { bindingName: 'ApprovePurchaseOrder', verb: 'approve that purchase order' }
      : { bindingName: 'UpdatePOStatus', verb: 'update that purchase order' }
  }
  if (target === 'order') {
    return { bindingName: 'UpdateOrderStage', verb: `${actionType} that order` }
  }
  if (target === 'offer') {
    if (actionType === 'approve') {
      // PRESERVED VERBATIM: MarkOfferWon's 2nd arg is the customer PO number
      // (persisted onto the Order, part of its idempotency key) — NOT an
      // approver name. Refuse rather than write a placeholder literal.
      const customerPO = String(v.data.customer_po ?? v.data.customerPO ?? '').trim()
      if (!customerPO) return { refusal: `I need the customer PO number to mark offer #${id} as won.` }
      return { bindingName: 'MarkOfferWon', verb: 'mark that offer as won' }
    }
    return { bindingName: 'MarkOfferLost', verb: 'mark that offer as lost' }
  }
  if (target === 'supplier_invoice') {
    return actionType === 'approve'
      ? { bindingName: 'ApproveSupplierInvoice', verb: 'approve that supplier invoice' }
      : { bindingName: 'DisputeSupplierInvoice', verb: 'dispute that supplier invoice' }
  }
  if (target === 'rfq') {
    return { bindingName: 'UpdateRFQStage', verb: `${actionType} that RFQ` }
  }
  if (target === 'stock_adjustment') {
    if (actionType !== 'approve') return { refusal: 'Stock adjustment rejection is not supported yet. Re-map to a supported status update.' }
    if (toNumericId(id) === null) return { refusal: 'I need a valid stock adjustment id to approve.' }
    return { bindingName: 'ApproveStockAdjustment', verb: 'approve that stock adjustment' }
  }
  if (target === 'costing_sheet') {
    if (toNumericId(id) === null) return { refusal: 'Invalid costing sheet id.' }
    return actionType === 'approve'
      ? { bindingName: 'ApproveCostingSheet', verb: 'approve that costing sheet' }
      : { bindingName: 'RejectCostingSheet', verb: 'reject that costing sheet' }
  }
  return { refusal: `This Butler action is recognized, but the execution path is not mapped for target '${target}'.` }
}

/** The single dispatcher seam: resolves which of the 23 write bindings an
 * armed+confirmed action maps to (running every refuse-over-guess guard along
 * the way), then calls the bridge's always-throwing executor. Never called
 * for navigate/analyze/fetch/clarify — those are single-click, non-armed,
 * and handled directly by the VM. */
export async function executeButlerAction(action: ResolvedButlerAction): Promise<ExecutionOutcome> {
  const type = normalizeActionType(action.type)
  let resolved: Refusal | Resolved
  if (type === 'create') resolved = await resolveCreateAction(action)
  else if (type === 'update') resolved = await resolveUpdateAction(action)
  else if (type === 'approve' || type === 'reject') resolved = await resolveApprovalAction(action)
  else return { message: `I don't know how to execute a '${type}' action yet.` }

  if ('refusal' in resolved) return { message: resolved.refusal }

  try {
    await executeButlerActionBinding(resolved.bindingName)
    return { message: `Done — ${resolved.verb}.` }
  } catch (err) {
    return { message: `I couldn't ${resolved.verb}: ${err instanceof Error ? err.message : String(err)}` }
  }
}
