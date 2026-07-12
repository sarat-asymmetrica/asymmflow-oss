# AGENTS.md — AsymmFlow

This file mirrors [`CLAUDE.md`](CLAUDE.md). See it for the full guidance: architecture,
the kernel → engine → overlay layer model, non-negotiable invariants, and security posture.

**Two rules that matter most for any agent touching this repo:**

1. **No secrets in source, ever.** Credentials load from env / in-app settings.
2. **No real client data.** All sample/test/demo data follows
   [`SYNTHETIC_IDENTITY.md`](SYNTHETIC_IDENTITY.md) — fictional company, tax IDs, bank
   details, people, and financials. Never reintroduce real values.

**Build → Test → Ship. Measure, don't estimate.**
