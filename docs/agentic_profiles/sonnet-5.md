# Agentic profile: Claude Sonnet 5 (scoped implementation worker)

**Role fit.** Sonnet-class models execute well-specified tasks fast and
cheaply. The quality of their output on this repo is almost entirely a
function of task framing: given a crisp spec with acceptance checks they are
excellent; given "improve X" they will produce something plausible that
violates a constraint they had no way to weigh. So: use Sonnet for call-site
rewires, test authoring against an existing pattern, bug fixes WITH a repro,
data/doc upkeep — and do the framing work below every single time.

## Task template (fill all five, no exceptions)

1. **Exact scope**: files it may touch, files it must not.
2. **Pattern to follow**: point at a concrete exemplar IN THIS REPO
   (e.g. "rewire like `customer_invoice_service.go`'s delegation to
   `pkg/documents/numbering`, byte-identical output format").
3. **Acceptance commands**: the literal `go test ./...` (or narrower)
   invocations that must pass, plus any behavioral pin ("number format
   INV-{date}-{seq} unchanged — add a test that locks it").
4. **Stop conditions**: "if the change requires touching rounding, tax math,
   posting order, sequence formats, or anything in pkg/kernel — STOP and
   report, do not adapt." Sonnet honors explicit stop conditions well and
   infers them poorly.
5. **The trap list** (below) pasted verbatim.

## The trap list (paste into every Sonnet task)

- SQLite: SELECT-FOR-UPDATE is a no-op; transactions that read-then-write
  deadlock under concurrency. If you must touch a transaction, keep its
  first statement a write (see `pkg/documents/numbering.NextInTx`).
- Windows CI: `t.TempDir()` cleanup fails unless the GORM `sql.DB` pool is
  closed in `t.Cleanup`.
- Never seed tests with fixed calendar dates + relative-period semantics
  ("this quarter"); derive from `time.Now().UTC()`.
- Money is int64 minor units; do not introduce float arithmetic on amounts.
- No CGO, no new heavyweight deps without asking.
- Demo/test data must come from the synthetic canon (SYNTHETIC_IDENTITY.md);
  never invent realistic-looking company names, VAT numbers, or IBANs.
- PowerShell 5.1 mangles quoted commit messages — commit with
  `git commit -F <file>`.
- Root-level `package main` files are legacy call sites, not architectural
  precedent. Copy patterns from `pkg/`, not from root.

## Review contract (the orchestrator's/human's side)

Sonnet output on this repo needs a **seam check**, not a line-by-line read:

- Did it stay inside the declared file scope? (diff the file list first)
- Do the pinned-behavior tests exist and assert the OLD format/values?
- Any new `time.Date(`, `float64` on money paths, `math/rand` in tests,
  raw SQL string concatenation? Grep for these four; they cover most
  regressions this class of model introduces here.
- If the task was a rewire: is the old implementation actually deleted or
  delegating (no silent fork)?

## Anti-fit (route to Opus/Fable instead)

- Designing a new engine seam or public API.
- Anything ZATCA-crypto or C14N adjacent.
- Diagnosing failures with no repro.
- Multi-layer changes where the spec cannot enumerate the file scope.
