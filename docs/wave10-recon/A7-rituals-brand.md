# A7 — Empty States + Document Rituals + Brand Slot (feeds B5)

## 1. Empty states
~26 distinct empty-state strings across `frontend/src/lib/screens`. **~12 generic** ("No X found") on the LIST views (RFQ/Offers/Orders/Delivery Notes/Invoices/Customers/Suppliers/Purchase Orders/Opportunities) plus `DataTable.svelte` default `emptyMessage='No data available'`. **~14 already operator-voiced** (mostly modal/dead-end states, e.g. InvoicesScreen "No unfulfilled orders available. Every order is already Complete or Invoiced…").
- **Highest-leverage B5b target: the list-view empty states** (first thing a new user sees).
- `WabiEmptyState.svelte` exists but is UNUSED dead code — natural home for the new copy (component-shaped: title + body + optional one action link, no illustration/mascot).
- No Statement/Receipt list screens exist.

Suggested rewrites (operator language, domain nouns, one optional action):
- RFQs: "No RFQs yet — enquiries you log land here."
- Offers: "No offers yet — quotes you send appear here."
- Orders: "No orders yet — won offers become orders here."
- Delivery Notes: "No delivery notes yet — dispatches you record show here."
- Invoices: "No invoices yet — bill a completed order to start."
- Customers/Suppliers: "No customers yet — add one to begin quoting."
(Coder to align exact wording with existing tone; keep short.)

## 2. Document-ritual visibility
**No document-set/checklist UI today.** Closest: `OrdersScreen.svelte` Order Detail modal shows a static "Traceability Chain" (RFQ/Enquiry → Offer → Order — STOPS at Order) + a separate "Linked Invoices" table. DN, Statement, Receipt never appear. `CustomersScreen.svelte` 360 has no document timeline (just stat fields). Greenfield for B5a; the Traceability Chain is a usable seam but B3's new timeline + one-call assembly (see A3) is the real home for the six-doc checklist.

## 3. Brand-slot inventory
| Touchpoint | File | State | B5c action |
|---|---|---|---|
| Sidebar header | `EnterpriseSidebar.svelte:164-165` | **HARDCODED** literal `PH`/`Trading` (STALE client name — also a synthetic-invariant issue) | replace with tokenized brand slot; default synthetic "AsymmFlow" |
| Login/lock | `LoginScreen.svelte:95-97` | **HARDCODED** duplicate `PH`/`Holdings`, no shared component | consume same brand slot |
| PDF headers | `company_branding.go` + `pkg/overlay`, tested by `ahs_branding_smoke_test.go` | **CONFIG-DRIVEN** (gold standard) — division-aware CompanyOverlay | already rebrand-ready; document the override in DEPLOYMENT_BRANDING.md |
| Accent color | `design-tokens.css:12-14 --brand-indigo:#2F2DFF` | **TOKENIZED** single var | this is the accent token a deployment overrides |
| App name/title | `wails.json "name":"AsymmFlow"` | CONFIG | document |
| Desktop icon | `build/appicon.png` (132KB) + `build/windows/icon.ico` (21.6KB) — both confirmed present | static build assets | icon swap = build-asset step, document in DEPLOYMENT_BRANDING.md |

**Bottom line for B5c:** backend PDF + desktop shell already config/token-driven. The REAL gap = the Svelte chrome: sidebar + login hardcode a stale "PH Trading"/"PH Holdings" wordmark in TWO places with no shared component. B5c must:
1. Create ONE brand-slot source (wordmark text + accent token + optional mark), default synthetic "AsymmFlow".
2. Consume it in sidebar header + login/lock (delete the hardcoded literals — also fixes the synthetic-identity leak).
3. Confirm PDF header path reads the same config concept (or document how it maps).
4. Prove the override path with a **gitignored throwaway** local override file (green accent + non-synthetic wordmark) — never commit/screenshot non-synthetic values.
5. Write `docs/DEPLOYMENT_BRANDING.md`: every slot + its file/token + the 2-step recipe (one config/token override + one icon build-asset swap).
Tone: lightly personal — accent + wordmark + icon only. Do NOT restyle components/charts/semantic status colors. Check the green accent stays distinguishable from the semantic success token where they co-occur (B3 timeline).
