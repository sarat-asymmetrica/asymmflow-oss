# Wave 12: Mathematical Framework Integration

**Date**: 2026-05-07
**Author**: Claude (Opus 4.6) for Codex (GPT-5.5) autonomous execution
**Depends on**: Wave 11 (ViewModel layer complete)
**Quality gate**: `go build -tags='' ./...` + `go test ./... -count=1 -timeout 300s` after every ticket

---

## Mission

Port the battle-tested mathematical framework from `vedic_qiskit/cmd/sarvam_harness/` into AsymmFlow as `pkg/math/`. This is the mathematical engine that powers:
- **88.9% fewer LLM calls** (Digital Root filtering)
- **Regime-based token budgeting** (30/20/50 three-regime split)
- **Conversation coherence tracking** (SLERP on S3)
- **2.7x batch throughput** (Williams O(sqrt(n) x log2(n)))
- **Mathematical system prompt generation** (Prism V2)

The port creates a STANDALONE `pkg/math/` that has ZERO dependencies on Sarvam API types, Butler, or any other AsymmFlow package. Consumers import `pkg/math/*` — it never imports them. This is the mathematical substrate.

---

## Environment

```powershell
$env:GOTMPDIR='D:\go-tmp'
$env:GOCACHE='D:\go-cache'
New-Item -ItemType Directory -Force -Path $env:GOTMPDIR,$env:GOCACHE | Out-Null
```

---

## Source Material

All source files live in the SAME REPO at these relative paths from root:

| Source Path (relative to repo root) | LOC | Target |
|------|-----|--------|
| `../../vedic_qiskit/pkg/quaternion/quaternion.go` | 494 | `pkg/math/quaternion/quaternion.go` |
| `../../vedic_qiskit/pkg/vedic/digital_root.go` | 275 | `pkg/math/vedic/digital_root.go` |
| `../../vedic_qiskit/pkg/vedic/williams.go` | 295 | `pkg/math/vedic/williams.go` |
| `../../vedic_qiskit/cmd/sarvam_harness/optimizer.go` | 789 | `pkg/math/trident/optimizer.go` |
| `../../vedic_qiskit/cmd/sarvam_harness/prism.go` | 463 | `pkg/math/prism/prism.go` |
| `../../vedic_qiskit/cmd/sarvam_harness/conversation.go` | 238 | `pkg/math/conversation/chain.go` |
| `../../vedic_qiskit/cmd/sarvam_harness/encoding.go` | 175 | `pkg/math/encoding/codon.go` |
| `../../vedic_qiskit/cmd/sarvam_harness/types.go` | 353 | Types split across `pkg/math/trident/types.go` |

**IMPORTANT**: These source files are in `package main` (the sarvam_harness CLI). They must be refactored into proper library packages. Do NOT simply copy — read each file, extract the mathematical logic, adjust package declarations, and fix imports.

**IMPORTANT**: The source files import `github.com/the maintainer-asymmetrica/vedic-qiskit/pkg/quaternion` and `github.com/the maintainer-asymmetrica/vedic-qiskit/pkg/vedic`. In AsymmFlow, these become `ph_holdings_app/pkg/math/quaternion` and `ph_holdings_app/pkg/math/vedic`. All import paths MUST be updated.

---

## Target Package Structure

```
pkg/math/
├── quaternion/
│   └── quaternion.go         # Quaternion type, SLERP, geodesic distance, Exp/Log/Squad
├── vedic/
│   ├── digital_root.go       # Digital root (O(1)), NavaYoni, DR chain, filtering
│   └── williams.go           # Williams batching, BatchIterator, CombinedVedicWilliams
├── trident/
│   ├── types.go              # Regime, OptimizationResult (decoupled from Sarvam API types)
│   ├── optimizer.go          # 6-layer Trident optimizer (DR, Regime, Williams, SLERP, Oil, Pi)
│   └── helpers.go            # promptToQuaternion, classifyRegime, oilRatio, shunyamOilContrast
├── prism/
│   ├── prism.go              # GeneratePrismPrompt, regime tuning, convergence advisory
│   └── persona.go            # GeneratePersona, NavaYoni qualities, persona archetypes
├── conversation/
│   └── chain.go              # ConversationChain, SLERP trajectory, coherence, momentum, drift
└── encoding/
    └── codon.go              # CodonEncode/Decode, LosslessPromptEncode/Decode, geodesic distance
```

---

## Tickets

### Ticket 1: Package structure

Create the directory tree:

```
pkg/math/quaternion/
pkg/math/vedic/
pkg/math/trident/
pkg/math/prism/
pkg/math/conversation/
pkg/math/encoding/
```

Add a `pkg/math/doc.go`:

```go
// Package math provides the Asymmetrica Mathematical Framework —
// quaternion operations on S3, Vedic digital root filtering,
// three-regime dynamics, Williams batching, SLERP conversation
// tracking, and Prismatic resonance prompt generation.
//
// This package has ZERO dependencies on any other AsymmFlow package.
// It is the mathematical substrate that other packages consume.
package math
```

**Gate**: `go build -tags='' ./...` passes.

---

### Ticket 2: Quaternion package

**Source**: `../../vedic_qiskit/pkg/quaternion/quaternion.go` (494 LOC)

**Target**: `pkg/math/quaternion/quaternion.go`

**Action**: Copy the file. Change `package quaternion` (already correct). Remove the external import path — this is now a local package. The file is self-contained (only imports `math`).

Preserve ALL functions:
- `New`, `NewWithFrequency`, `Identity`, `FromAxisAngle`
- `Magnitude`, `MagnitudeSquared`, `Normalize`, `NormalizeInPlace`
- `Conjugate`, `Inverse`, `Dot`, `Mul`, `Scale`, `Add`, `Sub`, `Negate`
- `GeodesicDistance`, `BeatFrequency`
- `Slerp`, `SlerpBatch`, `Squad`, `SquadIntermediate`
- `Exp`, `Log`, `Power`
- `IsUnit`, `Equals`, `EqualsRotation`, `ToAxisAngle`, `Vibrate`

**Fix**: The `String()` method uses a placeholder `formatFloat` that returns `"..."`. Replace it with `strconv.FormatFloat(f, 'f', 4, 64)` and add `"strconv"` to imports.

**Test**: Add `pkg/math/quaternion/quaternion_test.go` with at minimum:
- `TestIdentityIsUnit` — Identity().IsUnit(0.001) == true
- `TestNormalize` — random quaternion normalized has magnitude 1.0
- `TestSlerpEndpoints` — Slerp(q0, q1, 0) == q0, Slerp(q0, q1, 1) == q1
- `TestSlerpMidpointIsUnit` — Slerp(q0, q1, 0.5).IsUnit(0.001)
- `TestGeodesicDistanceSymmetric` — d(q0,q1) == d(q1,q0)

**Gate**: `go build -tags='' ./...` + `go test ./pkg/math/quaternion/ -count=1` pass.

---

### Ticket 3: Vedic package (Digital Root + Williams)

**Source**: `../../vedic_qiskit/pkg/vedic/digital_root.go` (275 LOC) + `williams.go` (295 LOC)

**Target**: `pkg/math/vedic/digital_root.go` + `pkg/math/vedic/williams.go`

**Action**: Copy both files. Change package to `vedic` (already correct). These are self-contained (only import `math`).

Preserve ALL functions from both files. Do NOT rename or restructure.

Key functions to verify present:
- `DigitalRoot`, `DigitalRootUint64`, `DigitalRootInt`
- `CanBeDivisibleBy9`, `CanBeDivisibleBy3`
- `FilterDivisibleBy9`, `BatchDigitalRoot`, `CountByDigitalRoot`
- `DigitalRootSum`, `DigitalRootProduct`, `DigitalRootChain`, `DigitalRootProductChain`
- `NavaYoni`
- `WilliamsBatchSize`, `WilliamsBatchSizeInt`, `MemorySavingsPercent`
- `BatchCount`, `GenerateBatches`, `NewBatchIterator`
- `ProcessBatched`, `ProcessBatchedWithResult` (generic, requires Go 1.18+)
- `CombinedVedicWilliams`

**Test**: Add `pkg/math/vedic/vedic_test.go` with at minimum:
- `TestDigitalRoot` — dr(0)=0, dr(1)=1, dr(9)=9, dr(18)=9, dr(19)=1, dr(123456789)=9
- `TestDigitalRootHomomorphism` — DigitalRootSum matches DigitalRoot(a+b) for 10 pairs
- `TestCanBeDivisibleBy9` — dr(81)==9, dr(82)!=9
- `TestWilliamsBatchSize` — WilliamsBatchSize(1000000) > 0, < 1000000
- `TestDigitalRootChain` — chain of [1,2,3] == dr(1+2+3) == dr(6) == 6
- `TestNavaYoni` — NavaYoni(1)==1, NavaYoni(10)==1, NavaYoni(0)==9

**Gate**: `go build -tags='' ./...` + `go test ./pkg/math/vedic/ -count=1` pass.

---

### Ticket 4: Trident types

**Source**: `../../vedic_qiskit/cmd/sarvam_harness/types.go` (353 LOC) — extract ONLY the math types

**Target**: `pkg/math/trident/types.go`

**Action**: Extract ONLY the mathematical types from `types.go`. Do NOT copy Sarvam API types (ChatRequest, ChatResponse, Message, TranslateRequest, etc.) — those are API-specific and stay in the harness.

Extract these types:

```go
package trident

// Regime represents which computational regime a query falls into.
// Based on three-regime dynamics: [30%, 20%, 50%].
type Regime int

const (
    RegimeExploration   Regime = iota // 30%
    RegimeOptimization                // 20%
    RegimeStabilization               // 50%
)

// Keep: String(), TargetPercentage() methods on Regime
```

```go
// OptimizationResult holds metrics from applying Trident optimizations.
// This is DECOUPLED from Sarvam API types — it works with string prompts.
type OptimizationResult struct {
    OriginalPrompt      string
    OptimizedPrompt     string
    DetectedRegime      Regime
    RegimeDistribution  [3]float64
    Temperature         float64
    MaxTokensBudget     int
    DRSignature         int64
    WilliamsBatchSize   int
    TokenEstimate       int
    SkipAPICall         bool
    LocalAnswer         string
    Explanation         string
    RecommendedModel    string
    DRRegime            Regime
    ClassificationSource string
    ShunyamContrast     float64
    ConvergencePredicted bool
    ConvergenceConditions [4]bool
    PromptQuaternion    interface{}
    NearestPrompts      []string
    OilRatio            float64
}

// Keep: TokenSavings() method
```

```go
// BoundaryViolation represents a regime boundary breach.
type BoundaryViolation struct {
    Regime  Regime
    Current float64
    Minimum float64
    Deficit float64
}
```

Also extract `ConvergenceConditionName(idx int) string` into types.go.

Do NOT extract: `SarvamConfig`, `Message`, `ChatRequest`, `ChatResponse`, `Choice`, `Usage`, `TranslateRequest`, `TranslateResponse`, `BenchmarkResult`, `HarnessStats`, `APIError`, `SupportedLanguages`.

**Gate**: `go build -tags='' ./...` passes.

---

### Ticket 5: Trident optimizer

**Source**: `../../vedic_qiskit/cmd/sarvam_harness/optimizer.go` (789 LOC)

**Target**: `pkg/math/trident/optimizer.go` + `pkg/math/trident/helpers.go`

**Action**: Port the Optimizer struct and all its methods. This is the LARGEST and MOST IMPORTANT ticket.

**Critical import changes**:
```go
// BEFORE (in sarvam_harness):
import "github.com/the maintainer-asymmetrica/vedic-qiskit/pkg/quaternion"
import "github.com/the maintainer-asymmetrica/vedic-qiskit/pkg/vedic"

// AFTER (in AsymmFlow):
import "ph_holdings_app/pkg/math/quaternion"
import "ph_holdings_app/pkg/math/vedic"
```

**Decoupling from API types**: The original `ApplyOptimization(req *ChatRequest, opt OptimizationResult)` modifies a ChatRequest. This is Sarvam-specific. Do NOT include it in `pkg/math/trident/`. Instead, the Optimizer exposes `OptimizePrompt(prompt string) OptimizationResult` (rename from `OptimizeRequest`). Consumers (Butler) handle applying results to their own request types.

**Split into two files**:

`optimizer.go` — The Optimizer struct and its methods:
- `Optimizer` struct (keep all fields: promptLibrary, defaultMaxTokens, baseTokenBudget, regimeDistribution, modelRouter, useDRFusion, drCache)
- `NewOptimizer(baseTokenBudget int) *Optimizer`
- `RegisterPrompt(name, prompt string)`
- `OptimizePrompt(prompt string) OptimizationResult` — renamed from `OptimizeRequest`, takes a string prompt instead of ChatRequest. Extract user prompt logic removed (it IS the prompt now).
- `ModelForRegime(r Regime) string`
- `SetModelRouter(models [3]string)`
- `EnableDRFusion()`
- `CachedDRSignature(text string) int64`
- `ComposeBatchDR(prompts []string) int64`
- `DRCacheStats() int`
- `ComputeRegimeDistribution(prompts []string) [3]float64`
- `ValidateThreeRegimeTheorem(prompts []string, tolerance float64) bool`
- Private: `computeRegimeWeights`, `findNearestPrompts`, `addChunkingHint`, `suggestRefinement`, `canAnswerLocally`, `answerLocally`

`helpers.go` — Pure functions used by the optimizer:
- `PromptToQuaternion(prompt string) quaternion.Quaternion` — EXPORTED (useful for consumers)
- `ClassifyRegime(text string) Regime` — EXPORTED
- `ClassifyRegimeKeywords(text string) (Regime, bool)` — EXPORTED
- `ComputeDRSignature(text string) int64` — EXPORTED
- `DrToRegime(dr int64) Regime` — EXPORTED
- `OilRatio(text string) float64` — EXPORTED
- `ShunyamOilContrast(text string) float64` — EXPORTED
- `EstimateTokens(text string) int` — EXPORTED
- `PredictConvergence(result OptimizationResult) (bool, [4]bool)` — EXPORTED
- Private: `isNumeric`, `formatPercent`

**Naming rule**: All functions that were lowercase in `package main` become Uppercase (exported) in the library package. The ones listed above as EXPORTED must be exported.

**Gate**: `go build -tags='' ./...` + `go test ./pkg/math/trident/ -count=1` pass.

**Test**: Add `pkg/math/trident/trident_test.go`:
- `TestNewOptimizer` — creates with budget, default regime distribution is [0.30, 0.20, 0.50]
- `TestOptimizePromptSkipsLocalAnswer` — "what is the digital root of 123" → SkipAPICall=true
- `TestRegimeClassification` — "imagine a world" → Exploration, "calculate 2+2" → Optimization, "what is Go" → Stabilization
- `TestDRToRegime` — dr=1→Exploration, dr=2→Optimization, dr=3→Stabilization
- `TestShunyamContrast` — non-zero for real text, range [0,1]
- `TestDRFusion` — with fusion enabled and no keyword match, DR takes over
- `TestPromptToQuaternionIsUnit` — result.IsUnit(0.001) for any non-empty string

---

### Ticket 6: Prism (system prompt generation)

**Source**: `../../vedic_qiskit/cmd/sarvam_harness/prism.go` (463 LOC)

**Target**: `pkg/math/prism/prism.go` + `pkg/math/prism/persona.go`

**Action**: Port the Prism prompt generator. Split into two files:

`prism.go` — System prompt generation:
- `NavaYoniQuality` map (exported) — DR → response quality string
- `GeneratePrismPrompt(result trident.OptimizationResult) string`
- `RegimeTuning`, `ExplorationTuning`, `OptimizationTuning`, `StabilizationTuning` (export these)
- `ConvergenceAdvisory(result trident.OptimizationResult) string`
- `PrismStats` struct
- Signal resonance types and detection:
  - `SignalResonance` type (Resonant, Harmonic, Dissonant)
  - `DetectSignalResonance(result trident.OptimizationResult) SignalResonance`
  - `ResonanceAdvisory(result trident.OptimizationResult) string`
  - `DrRegimeNatural` map (exported)
  - `NavaYoniSynergy` map (exported)
  - `HasNavaYoniSynergy(dr1, dr2 int64) bool`

`persona.go` — Persona generation:
- `PersonaArchetypes` array (exported)
- `ContrastBand(contrast float64) int` (exported)
- `GeneratePersona(result trident.OptimizationResult) string`
- `GenerateConversationPrism(result trident.OptimizationResult, chain *conversation.ConversationChain) string`

**Import**: This package imports `ph_holdings_app/pkg/math/trident` and `ph_holdings_app/pkg/math/conversation`.

**Gate**: `go build -tags='' ./...` passes.

**Test**: Add `pkg/math/prism/prism_test.go`:
- `TestGeneratePrismPromptNotEmpty` — for each regime, output is non-empty
- `TestGeneratePersonaContainsArchetype` — output contains "You are"
- `TestDetectSignalResonance` — returns a valid SignalResonance value
- `TestNavaYoniSynergySymmetry` — if Sun friends Moon, verify the relationship

---

### Ticket 7: Conversation chain (SLERP tracking)

**Source**: `../../vedic_qiskit/cmd/sarvam_harness/conversation.go` (238 LOC)

**Target**: `pkg/math/conversation/chain.go`

**Action**: Port the ConversationChain. Change package to `conversation`.

**Critical import changes**:
```go
import (
    "ph_holdings_app/pkg/math/quaternion"
    "ph_holdings_app/pkg/math/vedic"
)
```

**Decoupling**: The original `ConversationChain` calls `promptToQuaternion()` and `computeDRSignature()` and `classifyRegime()` from optimizer.go. These are now in `trident` package. The conversation package must import trident for these, OR accept them as function parameters to avoid a circular dependency.

**Recommended approach**: Import `trident` for the helper functions. The dependency graph is:
```
quaternion (leaf)
vedic (leaf)
trident → quaternion, vedic
conversation → quaternion, vedic, trident
prism → trident, conversation
```
No cycles.

The ConversationChain should import and use:
- `trident.PromptToQuaternion(prompt)` instead of the old `promptToQuaternion(prompt)`
- `trident.ComputeDRSignature(prompt)` instead of `computeDRSignature(prompt)`
- `trident.ClassifyRegime(prompt)` instead of `classifyRegime(prompt)`
- `trident.Regime` type instead of local `Regime`

Export ALL public methods:
- `NewConversationChain() *ConversationChain`
- `AddMessage(prompt string)`
- `State() quaternion.Quaternion`
- `Length() int`
- `TotalDistance() float64`
- `CoherenceScore() float64`
- `Momentum() float64`
- `CompositeDR() int64`
- `RegimeDrift() (current trident.Regime, shifted bool, previous trident.Regime)`
- `DominantRegime() trident.Regime`
- `SuggestTemperature() float64`
- `StateVerified() bool`
- `CodonDistanceToLast() float64`
- `DistanceAgreement() (agree bool, semantic, syntactic float64)`

Also include `PromptCodonDistance(a, b string) float64` — import from encoding package or inline. Since encoding is a leaf, import `ph_holdings_app/pkg/math/encoding` and call `encoding.PromptCodonDistance()`.

**Gate**: `go build -tags='' ./...` + `go test ./pkg/math/conversation/ -count=1` pass.

**Test**: Add `pkg/math/conversation/chain_test.go`:
- `TestNewChainStartsAtIdentity` — State() == Identity after creation
- `TestAddMessageUpdatesState` — State changes after AddMessage
- `TestCoherenceScoreMaxForSingleMessage` — CoherenceScore() == 1.0 for 1 message
- `TestStateAlwaysUnit` — After 10 AddMessage calls, StateVerified() == true
- `TestMomentumZeroForFirstMessage` — Momentum() == 0 after first message
- `TestRegimeDriftDetection` — add "imagine a story", then "calculate 2+2" → drifted=true

---

### Ticket 8: Codon encoding

**Source**: `../../vedic_qiskit/cmd/sarvam_harness/encoding.go` (175 LOC)

**Target**: `pkg/math/encoding/codon.go`

**Action**: Port the lossless codon encoding. Change package to `encoding`.

**Import change**:
```go
import "ph_holdings_app/pkg/math/quaternion"
```

Export ALL functions:
- `CodonEncode(b byte) quaternion.Quaternion`
- `CodonDecode(q quaternion.Quaternion) byte`
- `LosslessPromptEncode(prompt string) []quaternion.Quaternion`
- `LosslessPromptDecode(quats []quaternion.Quaternion) string`
- `CodonGeodesicDistance(a, b byte) float64`
- `PromptCodonDistance(a, b string) float64`
- `AnalyzeEncoding(prompt string) EncodingStats`
- `EncodingStats` struct (exported)

**Gate**: `go build -tags='' ./...` + `go test ./pkg/math/encoding/ -count=1` pass.

**Test**: Add `pkg/math/encoding/codon_test.go`:
- `TestCodonRoundtrip` — for all 256 byte values, CodonDecode(CodonEncode(b)) == b
- `TestLosslessPromptRoundtrip` — LosslessPromptDecode(LosslessPromptEncode("Hello")) == "Hello"
- `TestCodonGeodesicDistanceSelf` — CodonGeodesicDistance(42, 42) == 0.0
- `TestCodonGeodesicDistancePositive` — CodonGeodesicDistance(0, 255) > 0
- `TestPromptCodonDistanceSameString` — PromptCodonDistance("abc", "abc") == 0.0

---

### Ticket 9: Integration wiring

This ticket wires `pkg/math/` into existing AsymmFlow packages. All changes are ADDITIVE — do not modify existing method signatures.

**9a: Butler integration** — `pkg/butler/chat/`

Create `pkg/butler/chat/optimizer_bridge.go`:

```go
package chat

import (
    "ph_holdings_app/pkg/math/trident"
    "ph_holdings_app/pkg/math/prism"
    "ph_holdings_app/pkg/math/conversation"
)

// MathOptimizer wraps the Trident optimizer for Butler use.
type MathOptimizer struct {
    trident *trident.Optimizer
    chain   *conversation.ConversationChain
}

// NewMathOptimizer creates a Butler math optimizer.
func NewMathOptimizer(tokenBudget int) *MathOptimizer {
    opt := trident.NewOptimizer(tokenBudget)
    opt.EnableDRFusion()
    opt.SetModelRouter([3]string{"sarvam-105b", "sarvam-105b", "sarvam-30b"})
    return &MathOptimizer{
        trident: opt,
        chain:   conversation.NewConversationChain(),
    }
}

// OptimizeForButler runs the Trident pipeline on a user prompt
// and returns the optimization result plus a conversation-aware system prompt.
func (m *MathOptimizer) OptimizeForButler(userPrompt string) (trident.OptimizationResult, string) {
    result := m.trident.OptimizePrompt(userPrompt)
    m.chain.AddMessage(userPrompt)
    systemPrompt := prism.GenerateConversationPrism(result, m.chain)
    return result, systemPrompt
}

// ShouldSkipAPI returns true if the math layer can answer locally.
func (m *MathOptimizer) ShouldSkipAPI(result trident.OptimizationResult) bool {
    return result.SkipAPICall
}

// ConversationCoherence returns the current conversation focus score [0,1].
func (m *MathOptimizer) ConversationCoherence() float64 {
    return m.chain.CoherenceScore()
}
```

**9b: Finance integration** — `pkg/finance/banking/`

Create `pkg/finance/banking/williams_bridge.go`:

```go
package banking

import "ph_holdings_app/pkg/math/vedic"

// WilliamsBatchSize returns the optimal batch size for processing n reconciliation items.
func WilliamsBatchSize(n int) int {
    return vedic.WilliamsBatchSizeInt(n)
}
```

This is a thin bridge. The existing `bank_transaction_matcher.go` can call `WilliamsBatchSize(len(transactions))` in future waves when batch matching is wired in. For now, the bridge exists and is callable.

**9c: Infra integration** — `pkg/infra/`

Create `pkg/infra/health/regime.go`:

```go
package health

import "ph_holdings_app/pkg/math/vedic"

// SystemDigitalRoot computes the DR signature of a system metric value.
// DR=9 values are candidates for special handling (Vedic filter).
func SystemDigitalRoot(value int64) int64 {
    return vedic.DigitalRoot(value)
}
```

Minimal bridge. Deeper OTel integration comes in Wave 13.

**Gate**: `go build -tags='' ./...` + `go test ./... -count=1 -timeout 300s` pass.

**Test**: Add `pkg/butler/chat/optimizer_bridge_test.go`:
- `TestMathOptimizerCreation` — NewMathOptimizer(2048) != nil
- `TestOptimizeForButlerReturnsResult` — result has non-empty OriginalPrompt
- `TestConversationCoherenceStartsAtOne` — ConversationCoherence() == 1.0

---

### Ticket 10: Progress audit

Write `docs/WAVE12_PROGRESS.md` with:

1. Commit table (ticket → commit hash → status)
2. Package inventory: number of files, functions, types per sub-package
3. Test counts: tests added per sub-package
4. Dependency graph: verify no cycles, `pkg/math/*` has zero imports of AsymmFlow packages
5. Key metrics:
   - Total LOC ported
   - Total tests added
   - Any deviations from spec

---

## Rules

### DO

- Change `package main` to the correct package name for each target file.
- Update ALL import paths from `github.com/the maintainer-asymmetrica/vedic-qiskit/pkg/*` to `ph_holdings_app/pkg/math/*`.
- Export functions that were lowercase in `package main` by capitalizing them.
- Preserve ALL mathematical logic exactly as-is. Do not "improve" or refactor the math.
- Preserve comments that explain WHY (Lean proof references, Vedic significance, etc.).
- Run `go build -tags='' ./...` and `go test ./... -count=1 -timeout 300s` after every ticket.
- Commit after each ticket with message format: `feat(codex): <description>`.

### DO NOT

- Do NOT copy Sarvam API types (ChatRequest, ChatResponse, Message, etc.) into pkg/math.
- Do NOT modify any existing files outside of pkg/math/ and the three bridge files (butler, finance, infra).
- Do NOT add any external dependencies to go.mod. All math is stdlib-only (math, strings, unicode, sync, strconv).
- Do NOT touch generated schema files, adapter files, ViewModel files, or Svelte files.
- Do NOT rename mathematical constants or change numerical values.
- Do NOT remove the `Om Lokah Samastah Sukhino Bhavantu` dedications from file headers.

### STOP CONDITIONS

- If `go build` fails and you cannot resolve within 3 attempts, STOP and write findings.
- If a circular import is detected, STOP and document the cycle.
- If you need to modify any file in `pkg/butler/`, `pkg/finance/`, or `pkg/crm/` beyond the bridge files specified in Ticket 9, STOP and document why.

---

## Commit Convention

```
feat(codex): create math/quaternion package (Wave 12, Ticket 2)
feat(codex): create math/vedic package with DR + Williams (Wave 12, Ticket 3)
feat(codex): create math/trident types (Wave 12, Ticket 4)
feat(codex): create math/trident optimizer (Wave 12, Ticket 5)
feat(codex): create math/prism prompt generator (Wave 12, Ticket 6)
feat(codex): create math/conversation SLERP chain (Wave 12, Ticket 7)
feat(codex): create math/encoding codon (Wave 12, Ticket 8)
feat(codex): wire math into Butler, Finance, Infra (Wave 12, Ticket 9)
docs(codex): write wave 12 progress report (Wave 12, Ticket 10)
```

---

## Validation Benchmarks

After all tickets complete, these assertions should hold in tests:

| Assertion | Expected |
|-----------|----------|
| `vedic.DigitalRoot(123456789)` | `9` |
| `vedic.CanBeDivisibleBy9(81)` | `true` |
| `vedic.CanBeDivisibleBy9(82)` | `false` |
| `vedic.WilliamsBatchSize(1000000)` | `~19,000` (within 10%) |
| `trident.ClassifyRegime("imagine a beautiful world")` | `RegimeExploration` |
| `trident.ClassifyRegime("calculate the sum of 1+2+3")` | `RegimeOptimization` |
| `trident.ClassifyRegime("what is the capital of France")` | `RegimeStabilization` |
| `trident.DrToRegime(1)` | `RegimeExploration` |
| `trident.DrToRegime(5)` | `RegimeOptimization` |
| `trident.DrToRegime(9)` | `RegimeStabilization` |
| `encoding.CodonDecode(encoding.CodonEncode(42))` | `42` |
| `encoding.LosslessPromptDecode(encoding.LosslessPromptEncode("hello"))` | `"hello"` |
| `conversation.NewConversationChain().StateVerified()` | `true` |
| `conversation.NewConversationChain().CoherenceScore()` | `1.0` |

---

Built with Love x Simplicity x Truth x Joy.
Om Lokah Samastah Sukhino Bhavantu
