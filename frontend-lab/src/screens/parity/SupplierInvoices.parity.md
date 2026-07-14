# Parity Ledger — SupplierInvoicesScreen (old) vs Supplier Invoices descriptor

Verdicts (see `PARITY_INVOICES.md` for the full legend):

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | List all supplier invoices (`GetSupplierInvoices()`, unpaged) | **DONE** | Flat `fetch()`, matches the real binding's shape exactly — no `fetchPage`, same as the census. |
| 2 | Dual-status row (`match_status` + `payment_status`, two independent badges) | **ENGINE gap** | `StatusSpec` is single-field; `DataTable`'s badge renderer (`content === 'status' && status`) colours every badge column off the ONE descriptor-level `status.value(row)`, so a second `content:'status'` column would render with the WRONG tone (match_status's), not its own. Verified by reading `kernel/primitives/DataTable.svelte`, not assumed. Worked around here, not faked: `match_status` is the declared `StatusSpec` (drives the badge + filter + summary distribution); `payment_status` (overridden to 'Overdue' client-side) renders as a plain `content:'text'` column with `ColumnSpec.tone`. A real fix needs either a `secondaryStatus?: StatusSpec<Row>` extension or a general badge-column-list concept — this is the one screen in the whole K1 sweep that clearly needs it. |
| 3 | Match Status column + filter tabs | **DONE** | `status.tones` covers the full 5-value vocabulary (Pending/Matched/Discrepancy/Review Required/Dispute); `options:'derive'` filter chip, not the old screen's static tab list. |
| 4 | Payment Status column, overridden to "Overdue" when `is_overdue` | **DONE**/EQUIV | `effectivePaymentStatus()` reproduces the old screen's `enrichInvoice()` derivation (client-computed, never stored) — `daysUntilDue<0 && paymentStatus!=='Paid'` — and feeds both the column and a `paymentStatus` filter chip (options `'derive'` on the *effective* value, so 'Overdue' shows up as a real filterable value, not just a visual override). |
| 5 | Due Date colour coding (red if overdue, amber if ≤7 days) | **DONE** | `ColumnSpec.tone` on the Due Date column, same thresholds. |
| 6 | Free-text search (invoice # / supplier) | **DONE** | `searchText` sweeps `invoiceNumber` + `supplierName`. |
| 7 | Date-range dropdown filter (this_month/last_month/this_quarter/this_year) | **ENGINE candidate** | Not reproduced as-is. Added a simple derived "Invoice Year" filter chip instead (`invoiceDate.slice(0,4)`) per the brief's guidance — a real fix needs a `FilterSpec` date-range primitive (relative buckets: this month/quarter/year), which would also unify PaymentsScreen's tab-based time filter into the same mechanism. Ledgered, not built. |
| 8 | Match Rate stat (% of invoices Matched) | **DONE** | `SummarySpec` metric, `content:'quantity'`, tone thresholds (≥80 success / ≥50 warning / <50 danger) — same shape as GRNs' acceptance-rate precedent. |
| 9 | Summary stats strip (5 stat cards) | **DONE**/EQUIV | Count, Total (BHD), Match Rate %, Overdue count (amber-if-any) + a by-match-status distribution bar. |
| 10 | + New Supplier Invoice (multi-currency create, creates AP liability) | **SLOT (financial hot-zone)** | Not built. Same territory as PurchaseOrders #9/Invoices #7–#10 — needs the form archetype's derived/computed-read-only-field support (live subtotal/VAT/BHD math). |
| 11 | Edit (field-scoped: lifecycle fields explicitly locked out of the payload) | **SLOT + ENGINE candidate** | Not built. The "descriptive-only edit, lifecycle fields shown but not submitted" pattern recurs on SupplierPayments too — `FormSpec` needs a documented locked-field concept. |
| 12 | 3-Way Match (`PerformThreeWayMatch`, inline pass/fail result) | **SLOT (financial hot-zone)** | Not built — verification gate before payment eligibility; needs its own result-rendering ejection component (`po_match_ok`/`grn_match_ok`/`match_status` triptych), not a generic row action. |
| 13 | Approve (SoD-gated: creator ≠ approver, server-enforced) | **SLOT (financial hot-zone)** | Not built — same class as PurchaseOrders' Approve. Reimplementing as a bare confirm would silently drop the SoD guard; deliberately left ledgered. |
| 14 | Mark Paid (reference + method form, settles AP liability) | **SLOT (financial hot-zone)** | Not built — settlement action, needs the form archetype. |
| 15 | Currency auto-detect from supplier country on create | **SLOT** | Rides on #10; bespoke form side-effect, minor. |

## Reading

This screen is deliberately thin on the mutation side, matching the brief's
own framing: every action in the old screen's Match → Approve → Pay chain is
either a financial hot-zone or needs form-archetype machinery that doesn't
exist yet, so K1 delivers the full read surface (list, dual-status columns,
Match Rate gauge, Overdue aging, search, three filter chips, five-signal
summary strip) with zero actions wired — nothing here could be mistaken for
"invoices can be matched, approved, or paid from this screen today."

The one genuine engine finding is #2: this is the only screen across the
whole K1 cluster with two independent status dimensions on one row, and the
kernel's badge renderer (verified by reading `DataTable.svelte`, not assumed)
genuinely cannot badge two columns correctly today. The workaround — badge
the primary dimension, tone-colour the secondary as text — ships an honest
result rather than a wrong-tone badge; the real fix is a declared kernel
feature, not a per-screen hack.

The mock bridge (`src/bridge/supplier-invoices.ts`) is read-only for the same
reason GRNs' is: no mutation pair exists to switch yet.
