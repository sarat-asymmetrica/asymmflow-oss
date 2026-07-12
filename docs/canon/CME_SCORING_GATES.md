# CME Scoring Gates: Refactor from Hell

## Elegance Formula (Multiplicative, Bounded [0, 1])

```
Score = (Adequacy x Symmetry x Inevitability x Locality) - (Complexity + HiddenCost)
```

Zero on ANY positive axis zeros the whole thing.
A well-calibrated 0.72 > an inflated 0.95.

## Refactor-Specific Variant

```
Refactor_Score = (Symmetry_Gain x Locality_Gain x Inevitability) - (Regression_Risk + Migration_Complexity)
```

- Symmetry_Gain: how much duplication collapsed
- Locality_Gain: how much more isolated each domain became
- Inevitability: does the boundary feel correct?
- Regression_Risk: probability of breaking tests
- Migration_Complexity: how many files changed (lower = better)

## Wave Gate Criteria

| Gate | Score | Condition |
|------|-------|-----------|
| Schema Design (Wave 1 exit) | >= 0.72 | Types cover all 837 frontend functions |
| Generated Code (Wave 2-3 exit) | >= 0.68 | Generated output compiles, tests pass |
| Domain Logic (Wave 4 exit) | >= 0.80 | Hand-written logic is pure, testable, minimal |
| Integration (Wave 5 exit) | >= 0.75 | External systems wired, TOON/Pretext working |
| Final (Wave 6 exit) | >= 0.85 | Full system, multi-window, all tests green |

## Per-Domain ELEGANCE_CHECK Template

```
### ELEGANCE_CHECK — pkg/<domain>

- Adequacy:       X.XX  — [types admit all required operations?]
- Symmetry:       X.XX  — [duplicates collapsed? one pattern per concept?]
- Inevitability:  X.XX  — [boundary drawn correctly? "couldn't be otherwise"?]
- Locality:       X.XX  — [readable in isolation? max 2 external imports?]
- Hidden cost:    [coupling, goroutine leaks, global state, DB in pure functions?]
- Strongest objection: [what would a skeptic say about this boundary?]
- Generated %:    XX%   — [how much came from alchemy engines vs hand-written?]
- Final score:    X.XX  |  [SHIP / ITERATE / BURN-AND-REBUILD]
```

## The Inevitability Test (Domain Boundary Validation)

All must pass before a domain extraction is considered complete:

### 1. Naming Test
- [ ] Domain nameable in 2 words (no "and" or "or")
- [ ] Name is a NOUN, not a verb

### 2. Single Owner Test
- [ ] One person could own this domain entirely
- [ ] 80%+ of changes don't require coordination with other domains

### 3. Data Ownership Test
- [ ] Domain owns its primary tables
- [ ] Other domains access data through interfaces, not direct table queries

### 4. Business Language Test
- [ ] Acme Instrumentation staff use consistent vocabulary for this area
- [ ] Boundary aligns with how the BUSINESS thinks

### 5. Change Frequency Test
- [ ] Files in this domain co-change (high internal coupling)
- [ ] Files rarely change when OTHER domains change

### 6. Failure Isolation Test
- [ ] Domain bug doesn't crash the whole app
- [ ] Error boundary contains failures

### 7. "Couldn't Have Been Otherwise" Test
- [ ] No reasonable alternative grouping exists
- [ ] OR: documented why THIS grouping is superior

## Coupling Metrics (Track Per Wave)

| Metric | How to Measure | Target |
|--------|---------------|--------|
| Fan-In | Packages importing this domain | Stable/decreasing |
| Fan-Out | Packages this domain imports | <= 10 |
| Methods on *App | `grep -c "func (a \*App)" app.go` | Decreasing (331 → < 50) |
| Cyclomatic Complexity | `gocyclo` per function | <= 15 |
| Lines per Function | awk measurement | <= 80 (hard cap: 120) |
| Total Domain LOC | wc -l | Decreasing vs original |
| Generated vs Hand-Written | file header markers | >= 60% generated |

## Anti-Patterns (with CME Axiom Violations)

| Anti-Pattern | Axiom Violated | Signal | Fix |
|-------------|---------------|--------|-----|
| God Interface (50+ methods) | Minimality (4) | Replacing God Object with God Interface | Max 7-10 methods per interface |
| Hidden State Smuggling | Ref. Transparency (2) | Moving method but state stays in old location | Audit every a.db, a.cache reference |
| Adapter Explosion | Composition (1) | Dozens of adapter types | Direct DI over adapter layers |
| RBAC Infection | Locality (8) | Every method starts with requirePermission() | RBAC as middleware at Wails layer |
| Database Coupling | Ref. Transparency (2) | Domain logic calls a.db.Where().Find() | Repository interface injection |
| Refactor Tourism | Boundary Honesty (5) | Fixing unrelated things while extracting | One domain per wave |
| Premature Split | Locality (8) | 20 packages from day 1 | Start with internal/, promote later |
| Big Bang | Cost Awareness (7) | Extract everything at once | Each commit leaves tests green |

## Three-Regime Budget (Per Wave)

| Regime | Allocation | Activities |
|--------|-----------|-----------|
| R1 (Exploration) | 30% | Read existing code, trace workflows, identify unique logic |
| R2 (Optimization) | 20% | Design schemas, draw interfaces, run alchemy generators |
| R3 (Stabilization) | 50% | Wire code, run tests, fix breakages, polish |

**Critical rule:** Never skip R2. The 20% schema design is the hardest, most valuable work. Jumping from R1 to R3 causes leaky abstractions and repeated rework.

## Dead Code Criteria

Delete when 2 of 3 signals agree:
1. **Git history**: No commits in 3+ months
2. **Test coverage**: 0% AND no test references
3. **Call graph**: Zero callers (fan-in = 0)

Additional delete signals:
- File has `//go:build ignore`
- File prefixed with `manual_`, `demo_`, `example_`
- Function added in a "Phase X" but never wired to frontend

## The Phoenix Response

When a domain scores < 0.55:
1. Extract the unique algorithms (the 20% that's irreplaceable)
2. DELETE the rest
3. Re-run alchemy generators with refined schema
4. Wire unique algorithms into generated scaffolding
5. Re-score

There is no "give up." There is only "burn and rebuild better."

## Operational Rules

- NEVER hold back — write the ideal version
- NEVER be delicate — rip things out aggressively
- NEVER seek permission — if you see a better shape, BUILD it
- ALWAYS run tests after changes — they're the invariant
- ALWAYS use git as safety net — 2 seconds to undo
- ALWAYS score honestly — 0.72 with concerns > inflated 0.95
