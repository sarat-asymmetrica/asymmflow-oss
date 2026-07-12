# Collaboration Protocol: Claude + GPT-5.5 Codex

## Architecture

```
Commander (the maintainer) — Vision, steering, domain knowledge
       |
       v
Claude (Opus 4.6, 1M ctx) — Architecture, planning, review, scoring
       |
       v
GPT-5.5 (Codex CLI, /goal) — Autonomous execution, mechanical grinding
       |
       v
Git — Safety net, audit trail, rollback
```

## Role Division

### Claude Handles (HIGH-LEVEL)
- Architecture decisions and domain boundary design
- Schema design (the creative 20%)
- CME scoring and quality gates
- Code review of Codex output
- Course-correction when Codex drifts
- Unique domain logic specification
- Wave planning and sequencing
- Commander communication

### GPT-5.5 Codex Handles (EXECUTION)
- Running Wave 0 audit scripts
- Mechanical file moves and renames
- Struct extraction from God Object
- Import path rewiring
- Test suite maintenance (keeping green)
- Dependency migration (mattn -> ncruces)
- Boilerplate generation when alchemy engines don't fit
- Repetitive pattern application across files

## Communication Pattern

### Claude -> Codex (via `codex exec` or `/goal`)
```bash
# One-shot task
codex exec "Extract all bank-related functions from app.go into pkg/finance/banking/service.go. Keep function signatures identical. Run go build after."

# Long-horizon goal
codex
> /goal "Run all Phase 1-3 scripts from docs/WAVE0_AUDIT.md, save outputs to docs/audit_results/, then compile docs/ARCHAEOLOGICAL_REPORT.md summarizing findings"
```

### Codex -> Claude (via git commits + output files)
- Codex commits its work with prefix `refactor(codex):`
- Claude reviews via `git log` and `git diff`
- Output files in `docs/audit_results/` for Claude to analyze

## Goal Templates (Pre-Written for Each Wave)

### Wave 0: Archaeological Audit
```
/goal "Execute the complete archaeological audit defined in docs/WAVE0_AUDIT.md.
Run all Phase 1-7 scripts. Save raw outputs to docs/audit_results/.
Then synthesize findings into docs/ARCHAEOLOGICAL_REPORT.md following the
template at the bottom of WAVE0_AUDIT.md. Do NOT modify source code in this wave."
```

### Wave 1: Schema Extraction
```
/goal "Extract all 90 struct definitions from database.go and the ~70 additional
structs from app.go. Group them into 6 domain files per docs/TARGET_ARCHITECTURE.md:
schemas/finance.go, schemas/crm.go, schemas/documents.go, schemas/butler.go,
schemas/sync.go, schemas/infra.go. Each file should contain ONLY type definitions
(no methods, no logic). Verify with go build."
```

### Wave 2: God Object Decomposition (per domain)
```
/goal "Extract all finance-related methods from app.go into pkg/finance/.
Target files listed in docs/TARGET_ARCHITECTURE.md under pkg/finance.
Each method keeps its exact signature but receives dependencies via struct
fields instead of *App. After extraction, app.go should have ~50 fewer methods.
Run go build ./... after each batch of 10 methods."
```

### Wave 3: SQLite Migration
```
/goal "Replace all imports of github.com/mattn/go-sqlite3 with
github.com/ncruces/go-sqlite3. Update go.mod. Fix any API differences.
Run go build ./... and go test ./... -count=1 to verify."
```

## Quality Checkpoints

After each Codex `/goal` completes, Claude runs:

1. **Build check:** `go build ./...`
2. **Test check:** `go test ./... -count=1`
3. **Diff review:** `git diff --stat` (scope check)
4. **CME score:** Per-domain ELEGANCE_CHECK from CME_SCORING_GATES.md
5. **Method count:** `grep -c "func (a *App)" app.go` (should be decreasing)

If score < threshold: Claude specifies corrections, Codex executes them.

## Conflict Resolution

- If Codex makes an architectural decision Claude disagrees with: Claude wins (architect > executor)
- If Codex finds a better implementation path: Codex proceeds (executor > architect for tactics)
- If both are stuck: Commander decides
- If code is too tangled: Phoenix Clause (both agree to burn and rebuild)

## Parallelism Opportunities

With `multi_agent = true`, Codex can spawn multiple agents:
- Agent 1: Extract pkg/finance while Agent 2: Extract pkg/crm
- Agent 1: Run audit scripts while Agent 2: Set up Taskfile
- Agent 1: Backend migration while Agent 2: Frontend Svelte 5 runes

Claude coordinates which agents run in parallel based on dependency graph.

## Session Management

```bash
# Start interactive session with goal
codex
> /goal "..."

# Resume interrupted session
codex resume --last

# Fork a session (try alternative approach)
codex fork --last

# Review what Codex did
git log --oneline --author="codex" -20
```

## Anti-Patterns to Avoid

- DON'T: Have Claude and Codex edit the same file simultaneously
- DON'T: Let Codex make architecture decisions (it should follow TARGET_ARCHITECTURE.md)
- DON'T: Skip the quality checkpoint between goals
- DON'T: Let a /goal run without clear success criteria
- DO: Give Codex bounded, testable objectives
- DO: Review Codex commits before starting next wave
- DO: Use Phoenix Clause early if Codex is struggling (> 3 attempts)
