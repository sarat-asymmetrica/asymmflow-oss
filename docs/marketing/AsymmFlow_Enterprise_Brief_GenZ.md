# AsymmFlow by Asymmetrica AI

## Ship Fast. Own Your Data. Run Offline.

### For Builders Who Don't Wait 18 Months for an ERP

---

**From**: Asymmetrica AI
**Date**: February 2026

---

## 1. What This Is

AsymmFlow is a desktop ERP with an embedded AI brain. It manages your entire business — sales pipeline, procurement, finance, CRM, and reporting — in a single app that runs offline, syncs to the cloud when it feels like it, and doesn't charge you per seat.

One engineer built it using parallel AI agents. 108,000 lines of production Go. 51 screens. 260 API endpoints. Deployed to a live team in weeks, not quarters.

It's not a demo. It's running a real trading operation in Bahrain right now — 468 customer invoices, 484 supplier invoices, 601 payments, audited financials, 8 users across 4 roles.

And it's built to replicate. Trading was first. Grain processing, manufacturing, distribution — same platform, new configuration, same speed.

---

## 2. The Problem (You Already Know This)

Your operations are duct-taped together:

- **Tally** for accounting
- **Excel** for everything Tally can't do
- **WhatsApp** for approvals
- **Email** for document exchange
- **Someone's memory** for customer payment history

You can't answer "who owes us money and how overdue are they?" without opening 4 files and doing mental math.

**Cloud ERPs** want $15,000-$40,000/year in licensing for 8 people, take 12-18 months to implement, and your data lives on someone else's server in a jurisdiction you didn't pick.

**The real cost**: every hour your team spends on manual reconciliation, duplicate data entry, and "let me check and get back to you" is an hour they're not closing deals or managing operations.

---

## 3. What AsymmFlow Actually Does

### The Full Workflow

```
Customer sends RFQ
    ↓
AI classifies the document (Invoice? RFQ? PO? Bank statement?)
    ↓
Opportunity created in pipeline
    ↓
Costing sheet auto-generated
  → Product margins applied (configurable per product line)
  → Customer grade checked (A/B/C/D based on payment history)
  → Discount rules enforced (Grade A gets up to 7%, Grade D gets 0%)
  → Margin floor validated (below 8%? flagged automatically)
  → Approval decision: APPROVE / CAUTION / DECLINE
    ↓
Quotation PDF generated on your letterhead
    ↓
Customer accepts → Order created
    ↓
Purchase Order sent to supplier (state machine: Draft → Approved → Sent → Received)
  → Financial fields LOCKED after approval (no post-approval tampering)
    ↓
Goods arrive → GRN with partial receiving (shipment in 3 batches? tracked.)
    ↓
Delivery note generated → shipped to customer
    ↓
Invoice generated (VAT calculated, sequenced, on letterhead)
    ↓
Payment received → auto-reconciled against invoice
  → AR aging updated in real time
  → Customer grade recalculated
    ↓
Finance Hub shows your P&L, cash position, DSO, and 30+ ratios — live.
```

Every step has an audit trail. Every financial mutation is logged with timestamp and user. Every API endpoint is permission-checked.

### Module Breakdown

**Finance**
- Dynamic year selector (FY2023, 2024, 2025 — as many years as you have data)
- Audited data integrated alongside live operational data (green badge = audited, orange = unaudited)
- 30+ financial ratios: margins, ROE, ROA, current ratio, DSO, cash conversion cycle
- AR aging in real time (current, 30-60, 60-90, 90+ days)
- Bank reconciliation, cheque register, FX revaluation
- VAT built into every transaction layer (currently 10% Bahrain; configurable)

**Sales Pipeline**
- RFQ → Costing → Offer → Order → Invoice → Payment
- 24 business rules enforced automatically (margins, discounts, payment terms, competitor-aware pricing)
- Quotation PDFs with terms and conditions pages
- Win/loss tracking and pipeline analytics

**Procurement**
- PO lifecycle with enforced state machine (no skipping steps, no going backwards from terminal states)
- Supplier invoice processing and payment tracking
- GRN with partial receiving
- Delivery tracking with warehouse updates

**CRM**
- Customer 360: every order, invoice, payment, contact, and note in one screen
- AI-assigned payment grades (A-D) based on 79 behavioural signals
- Credit limit enforcement (overdue customer tries to place an order? blocked automatically)
- Supplier profiles with purchase history and issue tracking

**Intelligence (Butler AI)**
- Ask your database questions in English
- "Who are our top overdue customers?" → structured answer with data, not a chatbot hallucination
- "Generate a risk report" → multi-page PDF on your letterhead with cover page, data tables, AI analysis
- Drag-drop a document → AI classifies it, extracts fields, routes it to the right screen
- RBAC-aware: sales team can't ask Butler about financial data. Manager+ only.

---

## 4. The AI Layer Is Not a Gimmick

Butler AI doesn't just wrap an LLM around your data. Here's what happens when you ask a question:

```
Your question: "How is NPC doing?"
    ↓
Intent classification: Customer domain detected
    ↓
13 database queries executed:
  → Customer master record (grade, credit status)
  → Yearly revenue breakdown
  → Last 10 invoices with status
  → Recent payments and days-to-pay
  → Active contacts and decision makers
  → Open orders with line item detail
  → CRM notes history
  → Pending follow-ups
  → Active offers in pipeline
  → Portfolio context (how they compare to other customers)
    ↓
All context packaged and sent to Mistral LLM
    ↓
Response: structured analysis + suggested actions
  → "Navigate to NPC's 360 view"
  → "Create follow-up task for overdue invoice"
  → "Flag: 3 invoices past 90 days"
```

**Payment Prediction**: Each customer is encoded into a 79-dimensional state vector covering payment history, relationship strength, order patterns, and risk signals. Normalised to a unit sphere. Classified into three regimes. Output: a grade (A-D), predicted payment days (45-180), confidence score, and a recommended action. This feeds directly into the costing engine — no human has to manually assess "should we give this customer credit?"

**Document OCR**: Drag a PDF onto any screen. The AI classifies it (Invoice? RFQ? Bank statement? Supplier invoice?) with confidence scoring. Extracts key fields. Routes it to the correct module. No manual data entry for standard documents.

---

## 5. The Tech Stack (And Why These Choices)

| Choice | What We Use | What We Avoided | Why |
|--------|------------|-----------------|-----|
| Framework | **Wails** (Go + WebView2) | Electron | 35 MB binary vs 200 MB. Uses OS-native renderer, not bundled Chromium. |
| Backend | **Go** | Node.js, Python | Real concurrency (goroutines), single compiled binary, type safety. PDF generation, OCR, AI calls, and DB sync all run concurrently. |
| Frontend | **Svelte** | React | Compiles to direct DOM manipulation. No virtual DOM diffing. 51 screens built in a fraction of the time. |
| Database | **SQLite** (primary) | PostgreSQL, MySQL | Sub-millisecond local queries. No server process. Works offline. File-based — backup is literally copying a file. |
| Cloud sync | **Supabase** PostgreSQL | Firebase, AWS | $25/month. Background sync every 10 min. If it goes down, zero impact on operations. |
| AI | **Mistral** | OpenAI, Claude API | Cost-effective tiered model selection. Self-hostable future for full sovereignty. |
| ORM | **GORM** | Raw SQL | 88 models with relationships, soft deletes, auto-migration. Same models work with SQLite and PostgreSQL. |

**The result**: A 35 MB binary that runs on Windows and Mac. Double-click to launch. No Docker, no npm install, no server configuration, no "it works on my machine." One binary, one database file, one `.env` for cloud config.

---

## 6. Security (Red-Teamed, Not Hand-Waved)

We ran a security audit with **6 parallel AI agents** covering:
1. Authentication & Sessions
2. SQL/Command Injection
3. Data Security & Crypto
4. API & Input Validation
5. Infrastructure & Config
6. Business Logic & Race Conditions

**Result**: 117 raw findings. 74 unique after dedup. All critical (P0) and high-priority (P1) fixes shipped same day.

**What's in place**:
- RBAC on all 260 API endpoints (60+ distinct permissions)
- Device-bound license keys (`crypto/rand`, not deterministic hashing)
- CSRF tokens (cryptographic, 1-hour expiry, single-use)
- OAuth tokens stored as SHA-256 hashes (never plaintext)
- Parameterised queries everywhere (zero SQL injection surface)
- XSS escaping on all rendered data
- Command injection sanitisation (PowerShell, SQLite CLI)
- Path traversal prevention on file operations
- Payment race condition protection (row-level locking inside transactions)
- PO financial fields immutable post-approval
- Credit limit enforcement with database-level blocking
- 50 MB upload limit against resource exhaustion

This isn't a checkbox exercise. These are fixes for specific attack vectors identified during the audit.

---

## 7. Tax & Compliance

- **VAT**: Built into every transaction layer (invoicing, costing, procurement). Currently 10% for Bahrain. Configurable per deployment.
- **Currency precision**: BHD with 3 decimal places maintained throughout. Adaptable to any currency.
- **Invoice sequencing**: Enforced uniqueness. No duplicates. No gaps without explanation.
- **Audit trail**: Every financial operation logged — who, what, when. Immutable.
- **Audited vs live data**: Visually separated. Green badge = audited (e.g., Demo Auditors verified). Orange = live/unaudited. No ambiguity about what's verified and what's calculated.

---

## 8. How We Build: AI-Led Development

This is the part that changes the economics of everything.

### The Methodology

```
One founding engineer (the architect)
    +
N parallel AI agents (the builders)
    +
CLAUDE.md (the system constitution)
    =
108,000 lines of production code, shipped in weeks
```

**The founding engineer** defines architecture, sets business rules, reviews all output, and makes trade-off decisions (what to ship now vs defer, which security fixes are P0 vs P2, deployment sequencing).

**AI agents** execute in parallel. Feature implementation, security auditing, data migration, testing — all running concurrently. Phase 17 security audit: 6 agents, 117 findings, critical fixes same session. Phase 18 deployment: data reconciliation, Supabase migration, cross-compilation, and license generation — all parallel.

**CLAUDE.md** is a 500+ line living architecture document that every agent reads before touching code. It contains file purposes, architectural decisions, business context, deployment history, and pending work. This is what makes it possible for any agent to understand the full system without the engineer re-explaining context every time.

### What This Means for New Deployments

A new industry vertical (grain processing, manufacturing, distribution) doesn't start from zero. It starts from:
- 88 data models covering CRM, pipeline, finance, procurement, inventory
- 31 backend services with modular business logic
- 51 frontend screens covering all operational workflows
- 24 configurable business invariants
- A security posture that's already been red-teamed

New deployment = configure business rules + extend models where needed + adapt the UI. Not rebuild. The AI agents handle the implementation. The engineer handles the architecture.

**Timeline for a new vertical**: Weeks. Not months. Not years.

---

## 9. Industry Expansion

### Currently Live: Trading & Distribution
- Process instrumentation (Bahrain)
- Full pipeline: RFQ → Quote → Order → PO → GRN → Delivery → Invoice → Payment

### Ready to Deploy: Grain Processing & Agriculture
- Commodity pricing with market-linked rates
- Lot tracking with quality grading (moisture, weight, impurities)
- Warehouse and silo inventory management
- Seasonal procurement cycle support
- Export documentation and multi-jurisdiction compliance

### Ready to Deploy: Manufacturing
- Bill of Materials management
- Production order scheduling and tracking
- Work-in-progress inventory valuation
- Quality control checkpoints
- Machine utilisation tracking

### Ready to Deploy: Professional Services
- Project-based billing and time tracking
- Resource allocation and utilisation monitoring
- Milestone-based invoicing
- Contract lifecycle management

### What "Ready to Deploy" Means

The core platform handles 80% of any ERP use case out of the box: CRM, pipeline, finance, procurement, invoicing, payments, reporting, RBAC, AI assistant. The remaining 20% is industry-specific configuration — business rules, custom fields, specialised workflows. That 20% is what the AI agents build in weeks.

---

## 10. Cost Reality

| | SAP Business One (8 users) | Salesforce (8 users) | AsymmFlow |
|---|---|---|---|
| Implementation | $50,000 - $150,000 | $5,000 - $15,000 | Fraction of legacy |
| Annual licensing | $15,000 - $40,000 | $12,000 - $30,000 | $0 per-seat |
| Cloud infra | Bundled (vendor-hosted) | Bundled (vendor-hosted) | $25/month (optional) |
| Customisation | $200-$400/hr consulting | Limited | Built into methodology |
| Time to deploy | 12-18 months | 2-6 months | Weeks |
| Data ownership | Vendor | Vendor | You. 100%. On your machine. |
| Offline capability | No | No | Full functionality |

---

## 11. About Asymmetrica AI

We build enterprise software at AI speed.

One founding engineer. Parallel AI agents. Production-grade systems in weeks.

We don't sell seats. We don't host your data. We don't lock you in. We build systems you own — the code, the data, the infrastructure, the operational independence.

**Current scale**: 108K+ LOC production system, 8-user deployment, live financial data, red-teamed security.

**What we're building next**: The same platform, adapted for every industry that's currently overpaying for ERPs that take too long to implement and don't work offline.

**Philosophy**: Research Sovereignty, Build-Test-Ship.

**Contact**: [To be added]

---

*Built by Asymmetrica AI. Confidential.*
