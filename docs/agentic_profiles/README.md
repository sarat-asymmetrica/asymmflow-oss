# Agentic profiles

Operating manuals for running AI agents against this codebase, written by the
model that built Waves 1–2 (Claude Fable 5). Each profile answers the same
question honestly: **what does THIS class of model need to be told, checked
on, and forbidden from, to produce work of the quality this repo demands?**

The profiles encode the lessons from the decision journals
([FABLE_WAVE1_DECISIONS.md](../FABLE_WAVE1_DECISIONS.md),
[FABLE_WAVE2_DECISIONS.md](../FABLE_WAVE2_DECISIONS.md)) — especially the
`[Mirror]` annotations, which were written for exactly this purpose.

| Profile | Use for |
| --- | --- |
| [opus-4-8.md](opus-4-8.md) | Deep implementation work: new engines, compliance modules, cross-layer refactors |
| [sonnet-5.md](sonnet-5.md) | Scoped, well-specified tasks: call-site rewires, test authoring, doc upkeep, bug fixes with a repro |
| [orchestrator.md](orchestrator.md) | A coordinator model decomposing a wave-sized goal across worker agents |

Non-negotiables for EVERY agent, any model (from CLAUDE.md, restated because
agents skim):

1. No secrets in source; no real client data (synthetic canon only).
2. No CGO. ncruces SQLite stays.
3. AI-authority boundary: agents inspect/draft/recommend; only deterministic
   services approve/post/persist/delete — in the product AND in the process
   (an agent session must not "approve" its own financial-semantics change).
4. Financial semantics (rounding, posting order, tax) are stop-and-ask.
5. Green at every checkpoint: `go build ./...` + tests.
6. Kernel purity: no domain vocabulary in `pkg/kernel`.
