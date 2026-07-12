# The Autonomous Sprint Methodology (ASM)

**Origin**: Developed during AsymmFlow "Refactor from Hell" (May 5-6, 2026)
**Authors**: the maintainer + Claude (Opus 4.6)
**Proven Results**: 7 waves, 42+ commits, 21K→2K LOC God Object decomposition, zero test regressions
**Calibrated Runtimes**: 13m (mechanical), 46m (moderate), 76m (complex)

---

## The Core Loop

```
PLAN (Claude)  →  SPEC (Handoff Doc)  →  EXECUTE (Codex)  →  REVIEW (Claude)  →  PLAN...
   5-15 min          10-20 min            30-90 min            5-10 min

Commander's role: STEER between loops. Fire Codex. Live life. Come back. Repeat.
```

**Total human attention per wave**: ~30 minutes
**Total autonomous compute per wave**: ~30-90 minutes
**Leverage ratio**: 1:3 to 1:6 (every minute of human attention produces 3-6 minutes of autonomous work)

---

## Why This Works (The Three Pillars)

### Pillar 1: Machine-Verifiable Gates

The spec must contain gates that Codex can verify WITHOUT human judgment:

```
GOOD GATES (machine-verifiable):
  ✅ go build ./... exits 0
  ✅ go test ./... -count=1 exits 0
  ✅ npm run build exits 0
  ✅ grep -c "func (a *App)" app.go returns < 20
  ✅ npx vitest run passes

BAD GATES (require human judgment):
  ❌ "code is clean"
  ❌ "looks correct"
  ❌ "follows best practices"
  ❌ "elegant enough"
```

If Codex can't run a command to check success, it can't autonomously verify its own work. Every ticket needs at least ONE machine gate.

### Pillar 2: Bounded Tickets with Spiral Exit

Each ticket must be:
- **Bounded**: clear start, clear end, clear deliverables
- **Independent-ish**: can succeed even if the previous ticket had partial results
- **Exit-able**: if stuck after 3 attempts → document, skip, move on

```
GOOD TICKET:
  "Move all Payment methods from app.go to pkg/finance/payment/service.go.
   Leave thin wrappers. Run go build after.
   If stuck after 3 attempts: skip with TODO comment, move to next ticket."

BAD TICKET:
  "Refactor the payment system to be more modular."
  (No clear deliverables, no machine gate, no exit condition)
```

### Pillar 3: Rolling Pipeline (Don't Estimate, FLOW)

Don't scope to time. Scope to an ORDERED LIST of tickets that flow into each other:

```
WRONG: "This should take about 45 minutes"
RIGHT: "Do tickets 1-8 in order. Don't stop between tickets.
        Stop when all are done OR a stop condition hits."
```

Codex finishes fast? It rolls into the next ticket. Codex hits a wall? Spiral exit, move on. The pipeline absorbs variance naturally.

---

## The Spec Template

Every Codex handoff document follows this structure:

```markdown
# Codex Autonomous Execution Spec — [Wave/Sprint Name]

**Date**: YYYY-MM-DD
**From**: Claude (Senior Architect) + the maintainer
**To**: Codex (GPT-5.5, Senior Architect)
**Run Target**: Autonomous until complete
**Previous Runs**: [Context from prior waves]
**Build Verification**: [exact commands that must pass after every ticket]

---

## 0. Who You Are
[Identity framing — "senior architect", not "code monkey"]
[Full permissions statement]
[Governance: which docs to read first]

## 1. What Already Exists (Starting Position)
[Table of what previous waves delivered — DO NOT REBUILD]
[Current metrics: LOC, method counts, test status]

## 2. Tickets
[Dependency graph in ASCII]

### Ticket N: [Title]
**Problem**: [what's wrong or missing]
**Deliverables**: [exact files/functions to create or modify]
**Pattern**: [code example of the target shape]
**Checklist**:
- [ ] [machine-verifiable gate]
- [ ] [machine-verifiable gate]
**Commit**: `prefix: message`

## 3. Quality Gates
[Commands to run after EVERY ticket]
[Special rules for complex tickets]
[Spiral exit rules]

## 4. Autonomy Contract
[Start order, stop conditions, commit rules]
[Priority if tradeoffs needed]

## 5. What NOT To Touch
[Explicit list of files/domains out of scope]

## 6. Expected Outcome
[End-state metrics]

## Sign-Off
[Motivational, clear, action-oriented]
```

---

## The AGENTS.md Template

Every repo that Codex works on should have `.codex/AGENTS.md`:

```markdown
# AGENTS.md — [Project Name]

## Identity
[Who Codex is in this project]
[What it can and cannot do]

## Project Context
[One paragraph: what is this, what state is it in]

## Critical Docs (Read First)
[Ordered list of docs to read before starting any work]

## Operating Rules
1. NEVER ask permission — execute if the task is clear
2. ALWAYS run [build command] after changes
3. ALWAYS commit working states
4. If stuck > 3 attempts → document and move on
5. No time estimates — just execute

## Tech Stack
[Table of current → target technology choices]

## Quality Gate
[Scoring formula or checklist]

## Commands You'll Frequently Use
[Build, test, lint, clean commands]

## What SUCCESS Looks Like
[Measurable end-state metrics]
```

---

## Scaling To Multiple Projects

### The Commander's Dashboard

```
PROJECT A: AsymmFlow Refactor     │ Wave 7 ACTIVE (Codex)     │ ETA: checking in
PROJECT B: Ananta v2              │ Wave 2 PLANNED (spec ready)│ Fire after A review
PROJECT C: Rythu Mitra            │ Wave 1 PLANNED             │ Spec needed
PROJECT D: Lean Proofs            │ IDLE                       │ Blocked on review
PROJECT E: asymm-kit              │ Wave 1 PLANNED             │ Spec needed
```

### Parallel Execution Pattern

With ChatGPT Pro's generous limits, you can run MULTIPLE Codex instances:

```
Terminal 1: codex exec "Read CODEX_WAVE7_HANDOFF.md..." (AsymmFlow)
Terminal 2: codex exec "Read CODEX_WAVE2_HANDOFF.md..." (Ananta v2)
Terminal 3: codex exec "Read CODEX_WAVE1_HANDOFF.md..." (Rythu Mitra)

Commander goes for tea ☕

Come back: Review all three. Claude writes next specs for each.
Fire all three again.
```

**Theoretical throughput**: 3 projects × 45 min avg × 8 cycles/day = 18 HOURS of autonomous compute per day across 3 projects, with ~4 hours of human attention.

### Cross-Project Methodology

```
STEP 1: For each project, create:
  ├── .codex/AGENTS.md (identity + rules)
  ├── .codex/config.toml (project settings)
  └── docs/MASTER_PLAN.md (wave plan)

STEP 2: Claude writes first wave spec for each project
  └── CODEX_WAVE{N}_HANDOFF.md

STEP 3: Commander fires all specs in parallel
  └── One terminal per project, or sequential if limits hit

STEP 4: Commander checks in periodically
  └── "Codex finished Project A Wave 3" → Claude reviews → writes Wave 4
  └── "Codex still running on Project B" → wait
  └── "Codex hit STOP on Project C" → Claude diagnoses → writes adjusted spec

STEP 5: Repeat until all projects reach their target state
```

---

## Pushing Duration Limits (How To Get 4-10h Runs)

### What KILLS Long Runs

| Problem | Why It Happens | Prevention |
|---------|---------------|-----------|
| Context drift | Agent forgets rules after 2h | Coherence checkpoints: "Re-read AGENTS.md at tickets 10, 20, 30" |
| Infinite loop on hard problem | No exit condition | Spiral exit: "3 failures → skip, document, move on" |
| Accumulating errors | Small mistakes compound | Machine gates after EVERY ticket, not just at the end |
| Disk/memory exhaustion | Large builds, test artifacts | Explicit cleanup: "rm -rf build/ between tickets" |
| Token budget exhaustion | Too much output/thinking | "Keep commit messages under 100 chars. Don't explain, just do." |
| Approval interruption | Commands need human approval | Use `codex exec` not `/goal`, or pre-approve command patterns |

### What ENABLES Long Runs

| Enabler | How To Implement |
|---------|-----------------|
| Self-contained spec | ALL context in one document. Agent never needs to ask. |
| Progressive commits | Commit after EVERY ticket. Work is never lost. |
| Machine gates | Build + test after every ticket = automatic course-correction. |
| Bounded tickets | Each ticket is 15-45 min. 20 tickets = 5-15h pipeline. |
| Coherence checkpoints | "At ticket 10, re-read AGENTS.md and the last 3 commit messages" |
| Stretch tickets | Last 3-5 tickets are "nice to have" — agent does them if budget remains |
| Clean exit | "After all tickets: write PROGRESS.md with metrics. Commit. Done." |

### The 10-Hour Spec Structure

```
TICKETS 1-5:   Foundation work (must complete)
TICKETS 6-10:  Core extraction/migration (must complete)
TICKETS 11-15: Integration + testing (should complete)
TICKETS 16-20: Polish + documentation (stretch goals)

COHERENCE CHECKPOINTS:
  After Ticket 5:  re-read AGENTS.md, verify build, report progress
  After Ticket 10: re-read AGENTS.md, full test suite, interim commit
  After Ticket 15: re-read AGENTS.md, verify metrics, report progress
  After Ticket 20: final verification, write progress doc

CLEANUP BETWEEN SECTIONS:
  After Ticket 10: "rm -rf build/ tmp/ .cache/ to free disk space"
```

---

## Methodology Metrics (What To Track)

### Per Wave

| Metric | How To Measure | Target |
|--------|---------------|--------|
| Duration | Codex reports elapsed time | Calibrate per project |
| Tickets completed | Count commits | All core tickets |
| Build status | Final `go build` / `npm run build` | GREEN |
| Test status | Final test suite | GREEN |
| LOC delta | `wc -l` before/after | Decreasing (refactor) or controlled growth (features) |
| Key metric | Project-specific (e.g., method count) | Moving toward target |

### Across Projects

| Metric | What It Tells You |
|--------|------------------|
| Waves completed / week | Throughput |
| Human attention hours / wave | Efficiency |
| Specs that needed revision | Spec quality (lower = better) |
| STOP conditions hit | Problem complexity |
| Average wave duration | Calibration for future planning |

---

## Project-Specific Adaptations

### For Refactoring Projects (AsymmFlow)
- Tickets = "extract X to Y, verify build + tests"
- Machine gates = `go build` + `go test`
- Key metric = methods on God Object, LOC, alias count
- Risk = test regressions from extraction

### For Greenfield Projects (new features, new repos)
- Tickets = "create X, write tests, verify"
- Machine gates = `go test` + `npm run build`
- Key metric = feature completeness, test coverage
- Risk = scope creep (bound tickets tightly)

### For Frontend Projects (asymm_studio, UI work)
- Tickets = "build component X, pass accessibility check"
- Machine gates = `vitest run` + `npm run verify`
- Key metric = component count, Lighthouse score
- Risk = visual quality (add screenshot comparison if possible)

### For Research/Proof Projects (Lean, math)
- Tickets = "prove lemma X, no sorry, no axiom"
- Machine gates = `lake build` exits 0, `grep -c sorry` = 0
- Key metric = sorry count, axiom count
- Risk = mathematical dead-ends (spiral exit is critical)

---

## The Philosophical Foundation

This methodology works because it respects THREE truths:

1. **AI is tireless but needs direction.** Codex will grind for hours, but without clear specs it grinds in circles. The spec is the COMPASS, not the map — it tells Codex WHERE to go, not every step to take.

2. **Humans are creative but tire quickly.** Commander + Claude design the wave in 30 focused minutes, then Codex executes for 60+ minutes. Human creativity is spent on STRATEGY, not mechanical execution.

3. **Strict constraints enable creative execution.** Machine gates, spiral exits, and bounded tickets are NOT limitations — they're the JAZZ KEY SIGNATURE. Within those constraints, Codex improvises freely. Without them, it produces noise.

The result: **a development methodology where human creativity and AI execution complement each other at their RESPECTIVE strengths, producing output that neither could achieve alone.**

---

## Quick Start (New Project)

```bash
# 1. Create project structure
mkdir -p .codex docs

# 2. Write AGENTS.md (use template above)
# 3. Write MASTER_PLAN.md (wave plan)
# 4. Write first CODEX_WAVE1_HANDOFF.md (use template above)

# 5. Fire!
codex exec --model gpt-5.5 "Read CODEX_WAVE1_HANDOFF.md in full. Execute all tickets. Commit after each. Do not stop until complete or STOP condition hit."

# 6. Go live your life 🍵
# 7. Come back, review, write next spec, repeat
```

---

Built with Love × Simplicity × Truth × Joy.
*The method is the message.* 🔥🐒
