/* AllocationMatchPanel schema + pure math — the "spread a target amount across
 * several candidate documents" shape. Two consumers: bank-statement-line
 * matching (match one cash line to N invoices/expenses/payments) and AR/AP
 * allocation (apply one receipt across N invoices). The panel renders + plumbs
 * edits; this module owns the arithmetic (unit-tested), and the consuming VM
 * owns what "confirm" DOES (CreateSplitAllocation vs a loop of
 * ApplyCustomerReceiptToInvoice) — the panel judges nothing (financial
 * semantics stay with the backend). */

export interface MatchCandidate {
  id: string
  /** Entity type discriminator (CUSTOMER_INVOICE, EXPENSE, …) — drives the
   * type filter + which backend call the caller's confirm resolves to. */
  type: string
  label: string
  /** The candidate's open/outstanding amount — the allocation ceiling. */
  amount: number
}

export interface AllocationDraft {
  /** Stable key `${type}:${id}`. */
  key: string
  candidateId: string
  candidateType: string
  label: string
  /** User-editable applied amount. amount < maxAmount = partial application;
   * the panel doesn't forbid it (allowPartial), the backend decides legality. */
  amount: number
  maxAmount: number
}

export function allocationKey(type: string, id: string): string {
  return `${type}:${id}`
}

/** Sum of applied amounts across the plan. */
export function totalAllocated(allocations: AllocationDraft[]): number {
  return allocations.reduce((s, a) => s + (Number(a.amount) || 0), 0)
}

/** Target minus allocated. Positive = under-allocated, negative = over. */
export function remainingToAllocate(target: number, allocations: AllocationDraft[]): number {
  return target - totalAllocated(allocations)
}

/** Balanced when the remainder is within BHD 3dp tolerance of zero. */
export function isFullyAllocated(
  target: number,
  allocations: AllocationDraft[],
  tolerance = 0.001,
): boolean {
  return Math.abs(remainingToAllocate(target, allocations)) < tolerance
}

/** Default applied amount when adding a candidate: as much of the remainder as
 * the candidate can absorb, never negative, never over its own ceiling. */
export function defaultAllocationAmount(
  remaining: number,
  candidateMax: number,
): number {
  return Math.max(0, Math.min(remaining, candidateMax))
}
