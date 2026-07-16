/* Bank Accounts bridge module — self-contained: types + mock + real + switch
 * (pricing.ts/suppliers.ts pattern). K4 SettingsScreen split: this is the
 * "Bank Accounts" tab standalone (see screens/parity/Settings.parity.md).
 *
 * Real bindings confirmed on FinanceService (also mirrored on App):
 * GetAllBankAccounts / CreateBankAccount / UpdateBankAccount / DeleteBankAccount
 * (wailsjs/go/main/FinanceService.d.ts), model finance.CompanyBankAccount
 * (wailsjs/go/models.ts:4412).
 *
 * R1.4 — PLAINTEXT CONTRACT (verified against the Go impl, 2026-07-16):
 * CompanyBankAccount IBAN/SWIFT/account_number are stored PLAINTEXT by
 * deliberate design. Field encryption was REMOVED — migrateBankAccountEncryption
 * strips any leftover ciphertext back to plaintext, and TestBankAccount_Update-
 * PreservesEncryption asserts the IBAN reads back as literal plaintext. So
 * Create/Update send plaintext fields straight over the binding; there is NO
 * server-side FieldCrypto to feed and NO client-side pre-encryption (that would
 * corrupt the record). UpdateBankAccount server-whitelists editable columns.
 * Synthetic-only data (SYNTHETIC_IDENTITY.md). */

import { pick } from './runtime'
import { goDate, num, str } from './map'
import {
  CreateBankAccount,
  DeleteBankAccount,
  GetAllBankAccounts,
  UpdateBankAccount,
} from '$wails/go/main/FinanceService'
import type { finance } from '$wails/go/models'

export interface BankAccountRow {
  id: string
  name: string // account_name
  bankName: string
  accountNumber: string
  currency: string
  iban: string
  swiftCode: string
  isActive: boolean
  status: string // derived: 'Active' | 'Inactive'
  bookingRate: number
  updatedAt: string
}

export interface BankAccountDraft {
  name: string
  bankName: string
  accountNumber: string
  currency: string
  iban: string
  swiftCode: string
  status: string
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

const BANKS = ['Ahli United Bank', 'National Bank of Bahrain', 'BBK', 'Standard Chartered', 'HSBC Bahrain', '']
const CURRENCIES = ['BHD', 'USD', 'EUR', 'GBP', 'SAR']
const ACCOUNT_LABELS = [
  'Main Operating Account',
  'Payroll Account',
  'USD Trade Account',
  'EUR Settlement Account',
  'Petty Cash Reserve',
  'International Establishment for Process Control Equipment, Calibration Services and General Engineering Trading — Escrow Account',
  'X',
  'GBP Reserve',
]

let cache: BankAccountRow[] | null = null

function generate(): BankAccountRow[] {
  const rand = lcg(20260714 ^ 0xba2c)
  const rows: BankAccountRow[] = []
  const n = 22
  for (let i = 1; i <= n; i++) {
    const currency = CURRENCIES[i % CURRENCIES.length]!
    const isActive = i % 7 !== 0 // most accounts stay active
    const name = ACCOUNT_LABELS[i % ACCOUNT_LABELS.length]!
    rows.push({
      id: `bank-${i}`,
      name,
      bankName: BANKS[i % BANKS.length]!,
      accountNumber: i % 31 === 0 ? '' : pad(Math.floor(rand() * 1e10), 12),
      currency,
      iban: i % 41 === 0 ? `BH67AUBB${pad(i, 40)}` /* unbroken monster token */ : `BH${pad(i, 2)}AUBB${pad(Math.floor(rand() * 1e12), 14)}`,
      swiftCode: i % 17 === 0 ? '' : 'AUBBBHBM',
      isActive,
      status: isActive ? 'Active' : 'Inactive',
      bookingRate: currency === 'BHD' ? 1 : Math.round((0.1 + rand() * 1.2) * 10000) / 10000,
      updatedAt: `2026-0${1 + (i % 7)}-${pad(1 + (i % 27), 2)}`,
    })
  }
  return rows
}

async function mockFetchAll(): Promise<BankAccountRow[]> {
  cache ??= generate()
  await sleep(220)
  return [...cache]
}

async function mockCreate(draft: BankAccountDraft): Promise<void> {
  cache ??= generate()
  cache.unshift({
    id: `bank-new-${cache.length + 1}`,
    name: draft.name,
    bankName: draft.bankName,
    accountNumber: draft.accountNumber,
    currency: draft.currency,
    iban: draft.iban,
    swiftCode: draft.swiftCode,
    isActive: draft.status !== 'Inactive',
    status: draft.status || 'Active',
    bookingRate: draft.currency === 'BHD' ? 1 : 0,
    updatedAt: new Date().toISOString().slice(0, 10),
  })
  await sleep(150)
}

async function mockUpdate(id: string, draft: BankAccountDraft): Promise<void> {
  cache ??= generate()
  const row = cache.find((r) => r.id === id)
  if (row) {
    row.name = draft.name
    row.bankName = draft.bankName
    row.accountNumber = draft.accountNumber
    row.currency = draft.currency
    row.iban = draft.iban
    row.swiftCode = draft.swiftCode
    row.isActive = draft.status !== 'Inactive'
    row.status = draft.status || 'Active'
    row.updatedAt = new Date().toISOString().slice(0, 10)
  }
  await sleep(150)
}

async function mockDelete(id: string): Promise<void> {
  cache ??= generate()
  cache = cache.filter((r) => r.id !== id)
  await sleep(120)
}

/* ---- real: list-fetch WIRED, mutations INTEG-gapped (see file header) ---- */
function mapBankAccount(r: Record<string, unknown>): BankAccountRow {
  const isActive = Boolean(r.is_active)
  return {
    id: str(r.id),
    name: str(r.account_name),
    bankName: str(r.bank_name),
    accountNumber: str(r.account_number),
    currency: str(r.currency) || 'BHD',
    iban: str(r.iban),
    swiftCode: str(r.swift_bic),
    isActive,
    status: isActive ? 'Active' : 'Inactive',
    bookingRate: num(r.booking_rate),
    updatedAt: goDate(r.updated_at),
  }
}

async function realFetchAll(): Promise<BankAccountRow[]> {
  const rows = await GetAllBankAccounts()
  return (rows ?? []).map((r) => mapBankAccount(r as unknown as Record<string, unknown>))
}

async function realCreate(draft: BankAccountDraft): Promise<void> {
  // Plaintext contract (see file header): assemble a finance.CompanyBankAccount
  // and hand it over as-is. The server sets id/timestamps, forces is_active,
  // defaults currency→BHD and division→default, and derives display_order — so
  // we send only the operator-entered fields. IBAN/SWIFT go plaintext (never
  // pre-encrypted). The structural cast satisfies the generated model type
  // (its time.Time stubs are server-populated; map.ts precedent).
  const account = {
    bank_name: draft.bankName,
    account_name: draft.name,
    account_number: draft.accountNumber,
    iban: draft.iban,
    swift_bic: draft.swiftCode,
    currency: draft.currency,
    is_active: draft.status !== 'Inactive',
  } as unknown as finance.CompanyBankAccount
  await CreateBankAccount(account)
}

async function realUpdate(id: string, draft: BankAccountDraft): Promise<void> {
  // Plaintext patch of the server-whitelisted editable columns (Mission I-12:
  // UpdateBankAccount drops any non-editable key). Snake_case keys match the
  // GORM column names the whitelist checks. IBAN/SWIFT plaintext, as stored.
  const patch: Record<string, unknown> = {
    bank_name: draft.bankName,
    account_name: draft.name,
    account_number: draft.accountNumber,
    iban: draft.iban,
    swift_bic: draft.swiftCode,
    currency: draft.currency,
    is_active: draft.status !== 'Inactive',
  }
  await UpdateBankAccount(id, patch)
}

async function realDelete(id: string): Promise<void> {
  // DeleteBankAccount(id) — plain id delete (no draft/encryption surface). The
  // server refuses if the account carries statement/reconciliation history; the
  // descriptor surfaces that thrown error honestly.
  await DeleteBankAccount(id)
}

/* ---- public switched API (descriptor imports THESE) ---- */
export const fetchBankAccounts = (): Promise<BankAccountRow[]> => pick(realFetchAll, mockFetchAll)()
export const createBankAccount = (d: BankAccountDraft): Promise<void> => pick(realCreate, mockCreate)(d)
export const updateBankAccount = (id: string, d: BankAccountDraft): Promise<void> =>
  pick(realUpdate, mockUpdate)(id, d)
export const deleteBankAccount = (id: string): Promise<void> => pick(realDelete, mockDelete)(id)
export const currencyOptions = (): { value: string; label: string }[] =>
  CURRENCIES.map((c) => ({ value: c, label: c }))
