/* Data Quality bridge — self-contained: types + mock + real + switch.
 * `PreviewCustomerDataQuality` is a real, working fetch; `ReviewDataQuality
 * Issue` is real too on the old screen but K1-class mutations are gated at
 * K5 regardless (see build brief), so it's INTEG-gapped here, same
 * convention as credit-notes.ts's ApplyCreditNote. `GetDataQualityReview
 * History` is not called at all — the review-history table is a genuine
 * second panel the ledger archetype has no slot for, ledgered rather than
 * built (see DataQuality.parity.md). */
import { pick } from './runtime'
import { str } from './map'
import { PreviewCustomerDataQuality } from '$wails/go/main/App'

export interface DataQualityIssueRow {
  id: string
  severity: string
  issueType: string
  entityType: string
  entityId: string
  summary: string
  detail: string
  primaryAction: string
  reviewStatus: string
  reviewNote: string
  reviewedBy: string
  reviewedAt: string
}

/* ---- mock: adversarial + deterministic (see bridge/mock.ts) — every field
 * here is categorical/derived from `i`, so no LCG is needed for this one
 * (unlike the ledger bridges, which vary continuous money/date ranges). */
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))
const pad = (n: number, w: number): string => String(n).padStart(w, '0')

const ISSUE_TYPES = ['duplicate_customer', 'blank_customer_name', 'blank_opportunity_name', 'missing_customer_link', 'offer_missing_customer']
const SEVERITIES = ['critical', 'high', 'medium', 'low']
const ENTITY_TYPES: Record<string, string> = {
  duplicate_customer: 'customer',
  blank_customer_name: 'customer',
  blank_opportunity_name: 'opportunity',
  missing_customer_link: 'opportunity',
  offer_missing_customer: 'offer',
}
const PRIMARY_ACTIONS: Record<string, string> = {
  duplicate_customer: 'Merge duplicate customer records',
  blank_customer_name: 'Fill in the customer name',
  blank_opportunity_name: 'Fill in the opportunity name',
  missing_customer_link: 'Link the opportunity to a customer',
  offer_missing_customer: 'Link the offer to a customer',
}
const NAMES = [
  'Gulf Fabrication W.L.L.',
  'Manama Process Systems',
  'Al Dana Engineering Co.',
  'Interntional Establishment for Industrial & Petrochemical Instrumentation Services and General Trading (formerly Gulf Technical Calibration & Measurement Systems Company) W.L.L.',
  'المؤسسة الدولية لخدمات الأجهزة الصناعية والبتروكيماوية والتجارة العامة ذ.م.م',
  'Sitra Contracting',
  'X',
  'Bahrain Water Authority — Directorate of Operations & Maintenance, Section 7',
]

let cache: DataQualityIssueRow[] | null = null

function generate(): DataQualityIssueRow[] {
  const rows: DataQualityIssueRow[] = []
  const n = 86
  for (let i = 1; i <= n; i++) {
    // Adversarial seasoning at deterministic positions: an issue_type
    // outside the known vocabulary, a blank severity (old screen falls back
    // to the literal string "review"), and a blank entity name.
    const issueType = i % 71 === 0 ? 'unmapped_legacy_issue' : ISSUE_TYPES[i % ISSUE_TYPES.length]!
    const entityType = ENTITY_TYPES[issueType] ?? 'record'
    const severity = i % 37 === 0 ? '' : SEVERITIES[i % SEVERITIES.length]!
    const reviewed = i % 5 === 0
    const reviewStatus = !reviewed ? '' : i % 15 === 0 ? 'dismissed' : i % 10 === 0 ? 'resolved' : 'reviewed'
    const name = i % 67 === 0 ? '' : NAMES[i % NAMES.length]!
    const dupeCount = 2 + (i % 3)

    rows.push({
      id: `dq-${i}`,
      severity,
      issueType,
      entityType,
      entityId: `${entityType.slice(0, 3)}-${pad(i, 4)}`,
      summary:
        issueType === 'duplicate_customer'
          ? `Duplicate customer: ${name || '(unnamed)'} (×${dupeCount})`
          : issueType.startsWith('blank_')
            ? `Blank ${entityType} name on record ${pad(i, 4)}`
            : issueType === 'unmapped_legacy_issue'
              ? `Legacy data-quality flag on record ${pad(i, 4)} (no current handler)`
              : `Missing customer link on ${entityType} ${pad(i, 4)}`,
      detail:
        i % 23 === 0
          ? 'Cross-referenced against near-duplicate records sharing the same TRN and phone number — manual merge required, no automatic resolution available.'
          : `${entityType} record ${pad(i, 4)} needs review.`,
      primaryAction: PRIMARY_ACTIONS[issueType] ?? 'Review manually',
      reviewStatus,
      reviewNote: reviewed && i % 3 !== 0 ? 'Confirmed and corrected in source system.' : '',
      reviewedBy: reviewed ? `admin.${(i % 4) + 1}@example.bh` : '',
      reviewedAt: reviewed ? `2026-0${1 + (i % 6)}-${pad(1 + (i % 27), 2)}` : '',
    })
  }
  return rows
}

async function mockFetch(): Promise<DataQualityIssueRow[]> {
  cache ??= generate()
  await sleep(250)
  return [...cache]
}

async function mockReview(row: DataQualityIssueRow, action: string, note: string): Promise<void> {
  cache ??= generate()
  const r = cache.find((x) => x.id === row.id)
  if (r) {
    r.reviewStatus = action
    r.reviewNote = note
    r.reviewedBy = 'you@example.bh'
    r.reviewedAt = new Date().toISOString().slice(0, 10)
  }
  await sleep(150)
}

/* ---- real: fetch WIRED, review mutation is INTEG-gapped (honest throw) ---- */
function mapIssue(r: Record<string, unknown>): DataQualityIssueRow {
  return {
    id: str(r.id),
    severity: str(r.severity),
    issueType: str(r.issue_type),
    entityType: str(r.entity_type),
    entityId: str(r.entity_id),
    summary: str(r.summary),
    detail: str(r.detail),
    primaryAction: str(r.primary_action),
    reviewStatus: str(r.review_status),
    reviewNote: str(r.review_note),
    reviewedBy: str(r.reviewed_by),
    reviewedAt: str(r.reviewed_at),
  }
}

async function realFetch(): Promise<DataQualityIssueRow[]> {
  const rows = await PreviewCustomerDataQuality(300)
  return (rows ?? []).map((r) => mapIssue(r as unknown as Record<string, unknown>))
}

async function realReview(row: DataQualityIssueRow, action: string, note: string): Promise<void> {
  // ReviewDataQualityIssue is a real, working binding on the old screen —
  // but K1-class mutations are gated at K5 regardless (see build brief), so
  // this throws honestly rather than posting from an unreviewed screen.
  void row
  void action
  void note
  throw new Error('INTEG gap: ReviewDataQualityIssue — wires at K5')
}

/* ---- public switched API (descriptor imports THESE) ---- */
export const fetchDataQualityIssues = (): Promise<DataQualityIssueRow[]> => pick(realFetch, mockFetch)()
export const reviewDataQualityIssue = (row: DataQualityIssueRow, action: string, note: string): Promise<void> =>
  pick(realReview, mockReview)(row, action, note)
