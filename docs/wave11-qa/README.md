# Wave 11 QA sweep — screenshot index

Full-page renders of every primary screen, captured by the standing mirror
(`QA_SWEEP.md`). Synthetic identity only. Regenerate with:

```bash
cd frontend && npx playwright test tests/e2e/wave11-sweep.spec.ts --project=chromium
```

## Screens (each captured at `1440/` and `1100/` widths)

| Screen | id | Wave 11 status |
|---|---|---|
| Dashboard | `dashboard` | ✅ composed (styling baseline) |
| Opportunities | `opportunities` | ✅ composed |
| Operations | `operations` | ✅ composed |
| Finance | `finance` | ✅ composed |
| Accounting | `accounting` | ✅ **B2 fixed** — was flat/unstyled (undefined phi tokens) |
| Reports | `reports` | ✅ **B2 fixed** — was flat/unstyled (undefined phi tokens) |
| Work | `work` | ✅ composed |
| People | `people` | ✅ **B3 fixed** — detail sub-tabs were transparent/mis-styled |
| Notifications | `notifications` | ✅ composed |
| Relationships | `relationships` | ✅ composed (CustomersScreen — migration reference) |
| Intelligence | `intelligence` | ✅ composed |
| Settings | `settings` | ✅ composed (was phi-token dependent) |
| UserManagement | `usermanagement` | ✅ composed (deep-link) |
| Deployment | `deployment` | ✅ composed (deep-link) |

`debug/` holds targeted before/after probes (e.g. `people-detail.png` — the
Employee Detail sub-tab strip that A3 fixed).

See `../../FABLE_WAVE11_SPEC_REPORT.md` for the Defect Ledger and root-cause
narratives.
