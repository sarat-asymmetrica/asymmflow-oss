/* OneDrive Import bridge module — self-contained: types + mock + real +
 * switch (same shape as bridge/serial-trace.ts). Wraps the Go service in
 * onedrive_import_service.go (+ DetectOneDrivePath in
 * app_setup_documents_surface.go): walk locally-synced OneDrive deal
 * folders, fuzzy-match customers, and import deal folders into
 * opportunities/offers. Per the K5 brief, ALL FOUR real calls are
 * INTEG-gapped — this screen runs entirely on the mock until a future
 * owner-gated INTEG pass wires `wailsjs/go/main/App`.
 *
 * Go json field mapping (snake_case -> camelCase TS):
 *   DiscoveredDeal:        local_id -> localId, folder_path -> folderPath,
 *     folder_name -> folderName, final_path -> finalPath, root_path -> rootPath,
 *     customer_matches -> customerMatches, files -> files,
 *     instrument_type -> instrumentType, year_hint -> yearHint,
 *     status -> status, error_msg -> errorMsg,
 *     confirmed_customer_id -> confirmedCustomerId (base field; ReviewDeal
 *     below redeclares it as UI-owned, always-present state),
 *     imported_offer_id -> importedOfferId
 *   CustomerMatchResult:    customer_id -> customerId, business_name -> businessName,
 *     short_code -> shortCode, score -> score, match_reason -> matchReason
 *   DiscoveredFile:         file_name -> fileName, file_path -> filePath,
 *     file_type -> fileType, extension -> extension, size_bytes -> sizeBytes,
 *     mod_time -> modTime
 *   OneDriveScanResult:     deals -> deals, total_folders -> totalFolders,
 *     total_files -> totalFiles, scan_paths -> scanPaths, scanned_at -> scannedAt,
 *     errors -> errors
 *   OneDriveImportResult:   deal_local_id -> dealLocalId, success -> success,
 *     offer_id -> offerId, message -> message,
 *     costing_sheets_imported -> costingSheetsImported, pdfs_queued -> pdfsQueued
 *   ValidateOneDrivePath returns a raw `map[string]any` on the Go side
 *     (valid/estimated_deals/path/error) -> ValidatePathResult below.
 *
 * ConfirmOneDriveDeal is dead per the brief — not modeled here.
 */
import { pick } from './runtime'
import { num, str } from './map'
import {
  DetectOneDrivePath,
  ImportOneDriveDeals,
  ScanOneDrivePaths,
  ValidateOneDrivePath,
} from '$wails/go/main/App'
import type { main } from '$wails/go/models'

export interface CustomerMatch {
  customerId: string
  businessName: string
  shortCode: string
  score: number
  matchReason: string
}

export interface DiscoveredFile {
  fileName: string
  filePath: string
  fileType: string
  extension: string
  sizeBytes: number
  modTime: string
}

/** Mirrors Go's DiscoveredDeal exactly (server-shape). The screen works with
 * ReviewDeal (below), which adds the UI-owned selection state. */
export interface DiscoveredDeal {
  localId: string
  folderPath: string
  folderName: string
  finalPath: string
  rootPath: string
  customerMatches: CustomerMatch[]
  files: DiscoveredFile[]
  instrumentType: string
  yearHint: string
  status: string
  errorMsg?: string
  confirmedCustomerId?: string
  importedOfferId?: string
}

/** A scanned deal enriched with UI state. `selected` = the include checkbox;
 * `confirmedCustomerId` is always a string here ('' = skip/unmatched), unlike
 * the base DiscoveredDeal's optional field, so cell components can bind to it
 * directly without an undefined check. */
export type ReviewDeal = Omit<DiscoveredDeal, 'confirmedCustomerId'> & {
  selected: boolean
  confirmedCustomerId: string
}

export interface OneDriveScanResult {
  deals: ReviewDeal[]
  totalFolders: number
  totalFiles: number
  scanPaths: string[]
  scannedAt: string
  errors: string[]
}

export interface OneDriveImportResult {
  dealLocalId: string
  success: boolean
  offerId?: string
  message: string
  costingSheetsImported: number
  pdfsQueued: number
}

export interface ValidatePathResult {
  valid: boolean
  estimatedDeals?: number
  path?: string
  error?: string
}

/* ---- mock: adversarial + deterministic (see bridge/serial-trace.ts for the
 * pattern this mirrors) ---- */
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))
function lcg(seed: number): () => number {
  let s = seed >>> 0
  return () => {
    s = (s * 1664525 + 1013904223) >>> 0
    return s / 0xffffffff
  }
}
const pad = (n: number, w: number): string => String(n).padStart(w, '0')

const CUSTOMERS = [
  { id: 'cust-001', businessName: 'Gulf Fabrication W.L.L.', shortCode: 'GFAB' },
  { id: 'cust-002', businessName: 'Manama Process Systems', shortCode: 'MPS' },
  { id: 'cust-003', businessName: 'Al Dana Engineering Co.', shortCode: 'ADE' },
  { id: 'cust-004', businessName: 'Northgrid Industrial Holdings', shortCode: 'NGIH' },
  { id: 'cust-005', businessName: 'Sitra Contracting', shortCode: 'SITRA' },
  { id: 'cust-006', businessName: 'مؤسسة الخليج الدولية للأجهزة الصناعية', shortCode: 'GII' },
  { id: 'cust-007', businessName: 'National Petroleum Upstream Ltd.', shortCode: 'NPU' },
] as const

function match(customerIdx: number, score: number, matchReason: string): CustomerMatch {
  const c = CUSTOMERS[customerIdx]!
  return { customerId: c.id, businessName: c.businessName, shortCode: c.shortCode, score, matchReason }
}

const FILE_KINDS: { type: string; ext: string }[] = [
  { type: 'rfq', ext: '.pdf' },
  { type: 'rfq', ext: '.msg' },
  { type: 'quotation', ext: '.pdf' },
  { type: 'order', ext: '.pdf' },
  { type: 'document', ext: '.pdf' },
  { type: 'document', ext: '.docx' },
  { type: 'email', ext: '.msg' },
  { type: 'unknown', ext: '.tmp' },
]

function makeFiles(rand: () => number, count: number, dealIdx: number, costingSheets: number): DiscoveredFile[] {
  const files: DiscoveredFile[] = []
  for (let i = 0; i < count; i++) {
    const isCosting = i < costingSheets
    const kind = isCosting ? { type: 'costing_sheet', ext: '.xlsx' } : FILE_KINDS[Math.floor(rand() * FILE_KINDS.length)]!
    const size = 1024 + Math.floor(rand() * 4_500_000)
    const monthIdx = Math.floor(rand() * 24)
    const year = 2023 + Math.floor(monthIdx / 12)
    const month = (monthIdx % 12) + 1
    const day = 1 + Math.floor(rand() * 27)
    files.push({
      fileName: `file-${dealIdx}-${i + 1}${kind.ext}`,
      filePath: `deal-${dealIdx}\\FINAL\\file-${dealIdx}-${i + 1}${kind.ext}`,
      fileType: kind.type,
      extension: kind.ext,
      sizeBytes: size,
      modTime: `${year}-${pad(month, 2)}-${pad(day, 2)}T00:00:00Z`,
    })
  }
  return files
}

interface DealSpec {
  folderName: string
  instrumentType: string
  yearHint: string
  matches: CustomerMatch[]
  fileCount: number
  costingSheets: number
}

/** 18 deal specs covering the adversarial surface the K5 brief calls out:
 * 0-match deals, a single clear match, 2-3 near-tie ambiguous matches, a
 * 200-char folder name, an empty/whitespace folder name, RTL text, a huge
 * file count and a zero-file deal, an UNKNOWN (empty) instrument type, and
 * several empty year hints. */
const DEAL_SPECS: DealSpec[] = [
  {
    folderName: 'NORTHGRID LIT Q3 2025',
    instrumentType: 'Level (LIT)',
    yearHint: '2025',
    matches: [match(3, 0.95, 'shortcode')],
    fileCount: 7,
    costingSheets: 1,
  },
  {
    folderName: 'RI-37-26 NATIONALPETROLEUM UPSTREAM TIT 2026',
    instrumentType: 'Temperature (TIT)',
    yearHint: '2026',
    matches: [match(6, 0.88, 'token_overlap')],
    fileCount: 9,
    costingSheets: 2,
  },
  {
    // Adversary: an unbroken 200-char folder name.
    folderName: 'LEGACY ARCHIVE PROJECT DEAL FOLDER '.repeat(6).slice(0, 200),
    instrumentType: '',
    yearHint: '',
    matches: [],
    fileCount: 4,
    costingSheets: 0,
  },
  {
    // Adversary: empty/whitespace-only folder name.
    folderName: '   ',
    instrumentType: '',
    yearHint: '',
    matches: [],
    fileCount: 2,
    costingSheets: 0,
  },
  {
    // Adversary: RTL text.
    folderName: 'مشروع الغاز الطبيعي 2024',
    instrumentType: 'Gas (GIT)',
    yearHint: '2024',
    matches: [match(5, 0.79, 'token_overlap')],
    fileCount: 5,
    costingSheets: 1,
  },
  {
    // Adversary: 3-way near-tie ambiguous match.
    folderName: 'AMBIGUOUS DEAL ABC CORP 2025',
    instrumentType: '',
    yearHint: '2025',
    matches: [match(0, 0.55, 'token_overlap'), match(1, 0.52, 'token_overlap'), match(2, 0.5, 'token_overlap')],
    fileCount: 6,
    costingSheets: 1,
  },
  {
    // Adversary: 2-way near-tie ambiguous match.
    folderName: 'SPARES ORDER XYZ 2023',
    instrumentType: 'Spares',
    yearHint: '2023',
    matches: [match(4, 0.45, 'prefix'), match(2, 0.42, 'prefix')],
    fileCount: 5,
    costingSheets: 0,
  },
  {
    // Adversary: huge file count.
    folderName: 'HUGE ARCHIVE PROJECT CONSOLIDATED 2024',
    instrumentType: 'Service',
    yearHint: '2024',
    matches: [match(1, 0.68, 'token_overlap')],
    fileCount: 480,
    costingSheets: 3,
  },
  {
    // Adversary: zero-file deal.
    folderName: 'EMPTY FOLDER STUB',
    instrumentType: '',
    yearHint: '',
    matches: [],
    fileCount: 0,
    costingSheets: 0,
  },
  {
    folderName: 'VALVE REPLACEMENT PROGRAM',
    instrumentType: 'Valves',
    yearHint: '',
    matches: [match(2, 0.72, 'acronym')],
    fileCount: 6,
    costingSheets: 1,
  },
  {
    // Adversary: UNKNOWN instrument type (no keyword match) + no customer match.
    folderName: 'UNKNOWN TYPE MISC 2025',
    instrumentType: '',
    yearHint: '2025',
    matches: [],
    fileCount: 3,
    costingSheets: 0,
  },
  {
    folderName: 'SERVICE CONTRACT RENEWAL',
    instrumentType: 'Service',
    yearHint: '',
    matches: [match(0, 0.48, 'token_overlap'), match(6, 0.46, 'token_overlap')],
    fileCount: 4,
    costingSheets: 0,
  },
  {
    folderName: 'AL DANA MAINTENANCE PIT 2024',
    instrumentType: 'Pressure (PIT)',
    yearHint: '2024',
    matches: [match(2, 0.91, 'shortcode')],
    fileCount: 8,
    costingSheets: 2,
  },
  {
    folderName: 'SITRA FIT UPGRADE 2026',
    instrumentType: 'Flow (FIT)',
    yearHint: '2026',
    matches: [match(4, 0.85, 'shortcode')],
    fileCount: 6,
    costingSheets: 1,
  },
  {
    folderName: 'MPS AIT CALIBRATION 2023',
    instrumentType: 'Analytical (AIT)',
    yearHint: '2023',
    matches: [match(1, 0.8, 'shortcode')],
    fileCount: 5,
    costingSheets: 1,
  },
  {
    folderName: 'GFAB SPARE PARTS BATCH 2025',
    instrumentType: 'Spare Parts',
    yearHint: '2025',
    matches: [match(0, 0.93, 'shortcode')],
    fileCount: 10,
    costingSheets: 2,
  },
  {
    // Adversary: has a costing sheet but zero customer matches — cannot be
    // imported until a user manually confirms a customer (there is none to
    // pick from), so it stays excluded from canAdvance by construction.
    folderName: 'ORPHAN DEAL NO MATCH 2027',
    instrumentType: 'Level (LIT)',
    yearHint: '2027',
    matches: [],
    fileCount: 4,
    costingSheets: 1,
  },
  {
    // Adversary: mixed RTL + LTR text, 2-way near-tie.
    folderName: 'GULF INTERNATIONAL PROJECT مشروع 2025',
    instrumentType: 'Gas (GIT)',
    yearHint: '2025',
    matches: [match(0, 0.58, 'token_overlap'), match(5, 0.55, 'token_overlap')],
    fileCount: 7,
    costingSheets: 1,
  },
]

/** Deterministic, index-keyed import failures — independent of which subset
 * of deals a user chooses to import, so the "1-2 deterministic failures"
 * behavior always shows up whenever these two deals are included. */
const FAILING_DEAL_LOCAL_IDS: Record<string, string> = {
  'deal-6': 'match confidence too low: no candidate scored above the 0.6 auto-confirm threshold (synthetic demo failure)',
  'deal-16': 'duplicate opportunity: an offer already exists for this deal folder (synthetic demo failure)',
}

function generateDeals(rootPath: string): ReviewDeal[] {
  const rand = lcg(20260715)
  return DEAL_SPECS.map((spec, i) => {
    const idx = i + 1
    const localId = `deal-${idx}`
    const folderPath = `${rootPath}\\${spec.folderName}`
    const files = makeFiles(rand, spec.fileCount, idx, spec.costingSheets)
    const sortedMatches = [...spec.matches].sort((a, b) => b.score - a.score)
    return {
      localId,
      folderPath,
      folderName: spec.folderName,
      finalPath: `${folderPath}\\FINAL`,
      rootPath,
      customerMatches: sortedMatches,
      files,
      instrumentType: spec.instrumentType,
      yearHint: spec.yearHint,
      status: 'pending',
      errorMsg: '',
      importedOfferId: '',
      // Default selection: included iff >=1 match, defaulting to the
      // highest-score candidate (matches sorted descending above).
      selected: sortedMatches.length > 0,
      confirmedCustomerId: sortedMatches[0]?.customerId ?? '',
    }
  })
}

async function mockDetectOneDrivePath(): Promise<string> {
  await sleep(200)
  return 'C:\\Users\\Operator\\OneDrive - Synthetic Instruments Co\\Deals'
}

async function mockValidateOneDrivePath(path: string): Promise<ValidatePathResult> {
  await sleep(160)
  const trimmed = path.trim()
  if (!trimmed) return { valid: false, error: 'path is empty' }
  // "Looks like a folder": has a path separator or a drive letter — mirrors
  // the shape of real Windows/OneDrive paths without needing a filesystem.
  const looksLikeFolder = /[\\/]/.test(trimmed) || /^[a-zA-Z]:/.test(trimmed)
  if (!looksLikeFolder) return { valid: false, error: 'path is not a directory' }
  // Deterministic estimate from the path length (stands in for the real
  // call's bounded filepath.Walk folder count).
  const estimatedDeals = 1 + (trimmed.length % 24)
  return { valid: true, estimatedDeals, path: trimmed }
}

async function mockScanOneDrivePaths(paths: string[]): Promise<OneDriveScanResult> {
  await sleep(260)
  const rootPath = paths.find((p) => p.trim()) ?? 'C:\\Users\\Operator\\OneDrive - Synthetic Instruments Co\\Deals'
  // Fresh each scan: deterministic seed → identical data, but new objects so a
  // Start-Over re-scan restores default selections instead of reusing the
  // in-place mutations the include/customer cells made on the prior pass.
  const deals = generateDeals(rootPath)
  const totalFiles = deals.reduce((sum, d) => sum + d.files.length, 0)
  return {
    deals,
    totalFolders: deals.length + 6, // + folders walked that weren't deal folders
    totalFiles,
    scanPaths: paths,
    scannedAt: new Date().toISOString(),
    errors: [
      `permission denied: ${rootPath}\\LOCKED_ARCHIVE_2022`,
      `skipped unreadable folder: ${rootPath}\\~$TEMP_LOCK.tmp`,
    ],
  }
}

async function mockImportOneDriveDeals(deals: ReviewDeal[]): Promise<OneDriveImportResult[]> {
  await sleep(320)
  return deals.map((deal) => {
    if (!deal.confirmedCustomerId) {
      return {
        dealLocalId: deal.localId,
        success: false,
        message: 'skipped: no customer confirmed',
        costingSheetsImported: 0,
        pdfsQueued: 0,
      }
    }
    const failureMessage = FAILING_DEAL_LOCAL_IDS[deal.localId]
    if (failureMessage) {
      return {
        dealLocalId: deal.localId,
        success: false,
        message: failureMessage,
        costingSheetsImported: 0,
        pdfsQueued: 0,
      }
    }
    const costingSheetsImported = deal.files.filter((f) => f.fileType === 'costing_sheet').length
    const pdfsQueued = deal.files.filter((f) => f.extension === '.pdf').length
    return {
      dealLocalId: deal.localId,
      success: true,
      offerId: `OFR-2026-${pad(Number(deal.localId.replace('deal-', '')), 4)}`,
      message: `Offer created from "${deal.folderName.trim() || deal.localId}"`,
      costingSheetsImported,
      pdfsQueued,
    }
  })
}

/* ---- real: all four bindings WIRED to `$wails/go/main/App`, using the
 * snake_case↔camelCase field mapping documented at the top of this file. The
 * Go json field names are verified against the model classes in
 * wailsjs/go/models.ts (CustomerMatchResult, DiscoveredFile, DiscoveredDeal,
 * OneDriveScanResult, OneDriveImportResult). ---- */

// DiscoveredDeal[] element (server → ReviewDeal), reused by realScan.
function mapCustomerMatch(raw: unknown): CustomerMatch {
  const r = raw as Record<string, unknown>
  return {
    customerId: str(r.customer_id),
    businessName: str(r.business_name),
    shortCode: str(r.short_code),
    score: num(r.score),
    matchReason: str(r.match_reason),
  }
}

function mapDiscoveredFile(raw: unknown): DiscoveredFile {
  const r = raw as Record<string, unknown>
  return {
    fileName: str(r.file_name),
    filePath: str(r.file_path),
    fileType: str(r.file_type),
    extension: str(r.extension),
    sizeBytes: num(r.size_bytes),
    // mod_time is a Go time.Time — keep the raw RFC3339 string (no date-only
    // truncation; the file list may show the time component).
    modTime: str(r.mod_time),
  }
}

/** Server DiscoveredDeal → ReviewDeal, adding the UI-owned selection state with
 * the SAME defaults the mock's generateDeals uses: included iff ≥1 match,
 * defaulting the confirmed customer to the top-scored candidate. */
function mapScannedDeal(raw: unknown): ReviewDeal {
  const r = raw as Record<string, unknown>
  const matches = ((r.customer_matches as unknown[]) ?? [])
    .map(mapCustomerMatch)
    .sort((a, b) => b.score - a.score)
  return {
    localId: str(r.local_id),
    folderPath: str(r.folder_path),
    folderName: str(r.folder_name),
    finalPath: str(r.final_path),
    rootPath: str(r.root_path),
    customerMatches: matches,
    files: ((r.files as unknown[]) ?? []).map(mapDiscoveredFile),
    instrumentType: str(r.instrument_type),
    yearHint: str(r.year_hint),
    status: str(r.status) || 'pending',
    errorMsg: str(r.error_msg),
    importedOfferId: str(r.imported_offer_id),
    selected: matches.length > 0,
    confirmedCustomerId: str(r.confirmed_customer_id) || matches[0]?.customerId || '',
  }
}

/** ReviewDeal → Go DiscoveredDeal (camelCase → snake_case json). The importer
 * (onedrive_import_service.go:1454) reads confirmed_customer_id (gates creation:
 * an empty value makes the server SKIP the deal — never a wrong offer — and an
 * unresolvable id returns a "customer lookup failed" result, still no bad
 * document), plus local_id, folder_name, folder_path, root_path and
 * files[].file_type/file_path/extension; the remaining fields are sent
 * faithfully. mod_time defaults to Go zero-time when blank so time.Time
 * unmarshaling can't fail on an empty string. */
function toDiscoveredDeal(deal: ReviewDeal): main.DiscoveredDeal {
  return {
    local_id: deal.localId,
    folder_path: deal.folderPath,
    folder_name: deal.folderName,
    final_path: deal.finalPath,
    root_path: deal.rootPath,
    customer_matches: deal.customerMatches.map((m) => ({
      customer_id: m.customerId,
      business_name: m.businessName,
      short_code: m.shortCode,
      score: m.score,
      match_reason: m.matchReason,
    })),
    files: deal.files.map((f) => ({
      file_name: f.fileName,
      file_path: f.filePath,
      file_type: f.fileType,
      extension: f.extension,
      size_bytes: f.sizeBytes,
      mod_time: f.modTime || '0001-01-01T00:00:00Z',
    })),
    instrument_type: deal.instrumentType,
    year_hint: deal.yearHint,
    status: deal.status,
    error_msg: deal.errorMsg ?? '',
    confirmed_customer_id: deal.confirmedCustomerId,
    imported_offer_id: deal.importedOfferId ?? '',
  } as unknown as main.DiscoveredDeal
}

function mapImportResult(raw: unknown): OneDriveImportResult {
  const r = raw as Record<string, unknown>
  // Optional keys added conditionally (exactOptionalPropertyTypes: true — never
  // assign `undefined` to an optional prop).
  const result: OneDriveImportResult = {
    dealLocalId: str(r.deal_local_id),
    success: !!r.success,
    message: str(r.message),
    costingSheetsImported: num(r.costing_sheets_imported),
    pdfsQueued: num(r.pdfs_queued),
  }
  if (r.offer_id != null) result.offerId = str(r.offer_id)
  return result
}

async function realDetectOneDrivePath(): Promise<string> {
  // DetectOneDrivePath() → string (App.d.ts:433). Advisory prefill; the VM
  // swallows any error, so a bare await is fine.
  return (await DetectOneDrivePath()) ?? ''
}

async function realValidateOneDrivePath(path: string): Promise<ValidatePathResult> {
  // ValidateOneDrivePath(path) → map[string]any {valid, estimated_deals, path,
  // error} (App.d.ts:1875).
  const raw = (await ValidateOneDrivePath(path)) as Record<string, unknown>
  // Optional keys added conditionally (exactOptionalPropertyTypes: true).
  const result: ValidatePathResult = { valid: !!raw.valid }
  if (raw.estimated_deals != null) result.estimatedDeals = num(raw.estimated_deals)
  if (raw.path != null) result.path = str(raw.path)
  if (raw.error != null) result.error = str(raw.error)
  return result
}

async function realScanOneDrivePaths(paths: string[]): Promise<OneDriveScanResult> {
  // ScanOneDrivePaths(paths) → OneDriveScanResult (App.d.ts:1637). Read-only walk.
  const raw = (await ScanOneDrivePaths(paths)) as unknown as Record<string, unknown>
  return {
    deals: ((raw.deals as unknown[]) ?? []).map(mapScannedDeal),
    totalFolders: num(raw.total_folders),
    totalFiles: num(raw.total_files),
    scanPaths: ((raw.scan_paths as unknown[]) ?? []).map(str),
    scannedAt: str(raw.scanned_at),
    errors: ((raw.errors as unknown[]) ?? []).map(str),
  }
}

async function realImportOneDriveDeals(deals: ReviewDeal[]): Promise<OneDriveImportResult[]> {
  // ImportOneDriveDeals([]DiscoveredDeal) → []OneDriveImportResult (App.d.ts:1205).
  // Creates offers server-side, but only for deals whose confirmed_customer_id
  // resolves to a live customer (finance:create gated); everything else is
  // skipped/errored with a result row, never a wrong document.
  const payload = deals.map(toDiscoveredDeal)
  const results = await ImportOneDriveDeals(payload)
  return (results ?? []).map((r) => mapImportResult(r as unknown))
}

/* ---- public switched API (viewmodel imports THESE) ---- */
export const detectOneDrivePath = (): Promise<string> => pick(realDetectOneDrivePath, mockDetectOneDrivePath)()
export const validateOneDrivePath = (path: string): Promise<ValidatePathResult> =>
  pick(realValidateOneDrivePath, mockValidateOneDrivePath)(path)
export const scanOneDrivePaths = (paths: string[]): Promise<OneDriveScanResult> =>
  pick(realScanOneDrivePaths, mockScanOneDrivePaths)(paths)
export const importOneDriveDeals = (deals: ReviewDeal[]): Promise<OneDriveImportResult[]> =>
  pick(realImportOneDriveDeals, mockImportOneDriveDeals)(deals)
