# Refactor from Hell: Operating Principles

## Core Truth
This is an EXPERIMENTAL branch. Jordan's production version runs independently.
There is ZERO risk here. Recovery is O(1) via git rollback.

## The Unchained Loop
```
1. READ the code (understand it)
2. PLAN the clean version (schema-first)
3. BUILD it clean (zero tech debt, no compromises)
4. TEST to find the edges
5. INTEGRATE cleanly
6. REPEAT down the chain

If mess up → git rollback → try again
```

## What DOES NOT Apply Here
- GitNexus impact analysis before edits (production safety — not needed)
- Blast radius warnings (WE are the blast, intentionally)
- Minimum viable diff (we want MAXIMUM viable improvement)
- Delicate extraction (chainsaw mode)
- Permission-seeking ("should I proceed?") → JUST DO IT
- Clause 21 / Give-up protocol → BURN AND REBUILD instead

## What DOES Apply
- CME Axioms 1-9 (composition, purity, symmetry, minimality, boundaries, inevitability, cost, locality, adequacy)
- Test suite as oracle (73 test files = the invariant)
- Schema-first design (types before implementations)
- Zero tech debt (if it's not clean, it doesn't exist in this branch)
- ELEGANCE_CHECK scoring (quality gates for each wave)

## Agent Mandate
- NO holding back — write the ideal version, not the safe version
- NO delicacy — rip things out, rebuild from scratch, be aggressive
- NO permission-seeking — if you can see a better shape, BUILD it
- MAXIMUM aggression with MAXIMUM learning per cycle
- Git is the safety net — 2 seconds to undo ANYTHING

## The Phoenix Clause (replaces "Give Up")
If a domain/service/file is too tangled to extract:
1. Verify test coverage exists for the behavior
2. Verify schema/contract defines the types
3. DELETE the implementation entirely
4. REBUILD from schema upward
5. Run tests to verify behavioral equivalence

Burning is not failure. It's recognition that code crystallized in the wrong shape.

## Mathematical Basis
- Recovery cost → 0 (git rollback = 2 seconds)
- Learning from failure → always positive
- Therefore: optimal strategy = maximum aggression + maximum learning per cycle
- Exploration dominates exploitation when rollback is free
- The most expensive thing is HESITATION, not mistakes

## Quality Standard
"Would this make a senior Google engineer weep with elegance?"
If no → iterate. If yes → ship it.
