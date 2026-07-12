# Agentic profile: Claude Opus 4.8 (deep implementation worker)

**Role fit.** Opus-class models carry enough reasoning depth to do what Waves
1–2 required: hold a layer model in mind while writing code, notice when a
pattern being extracted is itself broken, and design seams rather than just
fill them. Give Opus the missions that create new structure — engines,
compliance modules, verticals. What it does NOT bring for free is this repo's
specific hard-won knowledge; that is what this profile injects.

## System-prompt ingredients (paste-ready)

Beyond CLAUDE.md, tell Opus explicitly:

- **The layer model is load-bearing, with one known trap:** the substrate in
  `pkg/` is clean, but ~140 root-level `package main` files predate it and
  ignore it. Never take architectural precedent from root files; take it from
  `pkg/kernel`, `pkg/overlay`, `pkg/compliance`, `pkg/finance/settlement`.
  Root files are call sites to rewire, not styles to copy.
- **SQLite is the real database, not a test stand-in.** Locking clauses like
  SELECT-FOR-UPDATE are silent no-ops; a read→write lock upgrade inside a
  transaction deadlocks under concurrency. Any read-modify-write transaction
  must issue its WRITE first (see `pkg/documents/numbering.NextInTx`). Write
  a concurrency test (20 goroutines, file-backed DB, WAL + busy_timeout DSN
  pragmas in ncruces `?_pragma=` syntax) for anything that allocates or
  increments.
- **Windows is a CI platform:** close GORM's `sql.DB` pool in `t.Cleanup` or
  `t.TempDir()` teardown fails; never rely on `:memory:` sharing across
  connections with the ncruces driver.
- **Time bombs:** never seed tests with fixed calendar dates plus "this
  quarter/year" semantics. Derive from `time.Now().UTC()` — and check the
  derivation itself survives period boundaries (a `-3 days` offset breaks in
  the first days of a quarter).
- **Money is integer minor units** (`pkg/kernel/money`); floats may exist
  only at a rendering/XML boundary that owns its own rounding rules, and each
  jurisdiction's rounding is that module's law (SAR: 2dp at each aggregation
  point; BHD: 3dp).
- **Go stdlib crypto refuses secp256k1.** For ZATCA work: decred
  `dcrec/secp256k1/v4`, manual ASN.1 (and `encoding/asn1` struct tags do not
  compose with `RawValue` for context-specific elements — walk elements
  manually). X.509 name strings must be UTF8String, not PrintableString.

## Working agreements

- **One coherent commit per capability**, message explains the why; decision
  log entry (`docs/FABLE_WAVE*_DECISIONS.md`) for anything a future reader
  would ask "why is it like this?" about.
- **Extraction = audit.** When promoting root code into an engine, test the
  engine HARDER than the original was tested. When the original fails your
  test, fix it in the engine and record the divergence — never silently.
- **Boot, don't just build.** After wiring a second consumer of any engine,
  run an end-to-end scenario immediately. Both Wave 2 composition bugs
  (PrintableString rejecting "é", VAT base-vs-total in the event payload)
  were invisible statically and surfaced in the first minute of running.
- **Financial semantics: stop-and-ask, with a written question.** If a task
  touches rounding, posting order, sequence formats, or tax treatment,
  produce the exact before/after numbers and wait for a human yes.
- **Reference codebases are read-only.** Re-implement the *shape* of a
  proven pattern; never copy code or import their vocabulary. Private study
  notes stay OUT of this public repo.

## Verification gate before claiming done

```
go build ./...        # needs frontend/dist present
go vet ./<touched>/...
go test ./<touched>/... then go test ./...
# if the work is a composition/vertical: RUN its binary, exit code 0
```

## Failure modes to watch (observed in Opus-class output)

- Confidently "simplifying" a guard whose reason lives two layers away —
  require the caller-comprehension pass before edits.
- Writing the happy-path test and declaring victory — demand at least one
  test per *documented refusal* (authority denied, lockout, variance without
  note, illegal state transition).
- Scope-creep refactors mid-task. The residue list exists so good ideas
  don't derail the current mission; append, don't pursue.
