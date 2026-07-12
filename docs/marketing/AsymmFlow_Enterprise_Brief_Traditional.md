# AsymmFlow by Asymmetrica AI

## Enterprise Operations Platform

### Strategic Brief for Business Leaders

---

**Prepared by**: Asymmetrica AI
**Date**: February 2026
**Classification**: Business Confidential

---

## 1. Executive Summary

AsymmFlow is a sovereign, AI-augmented Enterprise Resource Planning platform developed by Asymmetrica AI. It unifies sales pipeline management, procurement, finance, CRM, and business intelligence into a single desktop application that operates offline-first and syncs to the cloud when connected.

The platform is designed for rapid industry adaptation. A complete, production-grade ERP deployment — from first line of code to team rollout — is measured in weeks, not the 12-18 month timelines typical of legacy ERP implementations. This is made possible by Asymmetrica AI's proprietary AI-led development methodology, where a founding engineer directs parallel AI agents to architect, build, test, and harden systems at a pace that traditional development teams cannot match.

**Current deployment**: A full-featured ERP serving an 8-person team at a process instrumentation trading company in Bahrain, managing 468+ customer invoices, 484 supplier invoices, 601 supplier payments, and real-time financial reporting against Demo Auditors audited figures.

**Expansion-ready**: The underlying platform architecture — modular service layer, configurable business rules engine, industry-agnostic data models — is engineered to extend into grain processing, manufacturing, distribution, and any sector requiring integrated operations management.

---

## 2. The Problem

Mid-market companies across trading, manufacturing, and processing industries share a common operational reality:

**Fragmented Systems**
- Accounting in Tally or QuickBooks
- Customer management in spreadsheets or basic CRM tools
- Sales pipeline tracked in email threads and WhatsApp groups
- Procurement managed through purchase order templates in Word or Excel
- No unified view of business health

**The Cost of Fragmentation**
- Manual data reconciliation across disconnected systems
- Duplicate data entry (order details entered in CRM, re-entered in accounting, re-entered in procurement)
- No real-time visibility into cash position, receivables, or pipeline
- Audit preparation requires weeks of manual compilation
- Compliance gaps — VAT calculations done manually, invoice sequencing errors, missing audit trails

**The Cloud ERP Dilemma**
- Enterprise solutions (SAP, Oracle NetSuite) carry implementation costs of $50,000-$500,000 and 12-18 month timelines
- Per-seat SaaS models (Salesforce, Zoho, Odoo) create recurring cost obligations that scale with headcount
- Cloud dependency means operations halt when internet is unavailable
- Data residency concerns — sensitive financial and customer data stored on third-party servers in foreign jurisdictions

---

## 3. The Solution: AsymmFlow Platform

AsymmFlow addresses these challenges through a unified platform with five integrated modules, a configurable business rules engine, and an embedded AI assistant.

### 3.1 Finance & Compliance Module

**Financial Reporting**
- Dynamic financial year selector with support for multiple fiscal years
- Integration of audited financial data (e.g., Demo Auditors FS2024) alongside live operational data
- Clear visual distinction between audited and unaudited figures (audit badges)
- 30+ financial ratios calculated automatically: gross margin, net margin, ROE, ROA, current ratio, DSO, DPO, cash conversion cycle
- Accounts receivable aging analysis (current, 30-60, 60-90, 90+ day buckets)

**Tax Compliance**
- VAT calculation built into every transaction layer (invoicing, costing, procurement)
- Currently configured for Bahrain's 10% VAT regime; adaptable to any jurisdiction's tax structure
- BHD 3-decimal currency precision maintained throughout (configurable per deployment)
- Invoice sequencing with uniqueness enforcement — no duplicate invoice numbers
- Full audit trail on all financial mutations (create, update, delete with timestamp and user attribution)

**Banking & Reconciliation**
- Bank statement reconciliation with transaction matching
- Book-side reconciliation for internal verification
- Cheque register management
- Foreign exchange revaluation for multi-currency operations

**Payments & Collections**
- Customer payment recording with automatic invoice reconciliation
- Supplier payment tracking with race condition protection (row-level locking)
- Overpayment prevention (backend validates amount against outstanding balance)
- Duplicate payment detection (cross-references amount, date, and reference)

### 3.2 Sales Pipeline Module

**End-to-End Workflow**
```
RFQ Received → Costing Sheet → Offer/Quotation → Order → Invoicing → Payment Collection
```

- Opportunity tracking from initial enquiry to close
- Costing engine with configurable product margins and customer-grade-based discount rules
- Automated quotation generation with PDF output on company letterhead
- Order management with line item detail (product, quantity, unit price, total, margin)
- Offer follow-up tracking and win/loss analysis

**Automated Business Rule Enforcement**
- 24 codified business invariants executed at every approval point
- Minimum margin thresholds (configurable per deployment; e.g., 8% floor)
- Customer grade determines payment terms, advance requirements, and discount eligibility
- Automated approval/caution/decline decisions on quotations based on risk profile
- Competitor-aware pricing rules (e.g., margin floors when specific competitors are bidding)

### 3.3 Procurement & Operations Module

**Purchase Order Management**
- Full PO lifecycle with enforced state machine: Draft → Approved → Sent → Received | Cancelled
- Financial field protection — amount, total, and rate fields locked after approval (prevents post-approval manipulation)
- Supplier name enrichment from master data

**Goods Received Notes (GRN)**
- Partial receiving support (track shipments that arrive in multiple batches)
- GRN-to-PO matching and variance reporting

**Delivery Management**
- Delivery note generation and tracking
- Warehouse location updates
- Integration with order fulfillment pipeline

### 3.4 CRM Module

**Customer Management**
- Customer 360 view: orders, invoices, payments, contacts, notes, and AI-generated payment grade in a single screen
- Payment behaviour grading (A through D) based on historical data analysis
- Credit limit enforcement with automatic blocking for overdue accounts
- Contact management with decision-maker identification

**Supplier Management**
- Supplier profiles with purchase history, invoice tracking, and payment records
- Supplier issue tracking and resolution
- Performance monitoring through KPI dashboards

### 3.5 Intelligence Module (Butler AI)

**Natural Language Business Queries**
- Users ask questions in plain English: "Who are our top 5 overdue customers?", "What's our win rate this quarter?", "Generate a risk report for NPC."
- Butler queries the live business database, aggregates relevant data across up to 13 sub-queries per domain, and returns structured answers with suggested actions
- Six business domains: Customer, Supplier, Financial, Operations, Risk, Market

**AI-Generated Reports**
- Six report types: Customer, Financial, Risk, Operations, Supplier, Executive
- Multi-page PDF output on company letterhead with cover page, data tables, and AI-generated analysis
- Insights are grounded in actual company data, not generic advice

**Document Intelligence**
- OCR-enabled document scanning (drag-and-drop from any screen)
- Automatic classification: Invoice, RFQ, Purchase Order, Bank Statement, Quotation, Delivery Note, Contract
- Field extraction: invoice numbers, dates, amounts, customer/supplier names
- Routing: classified documents are directed to the appropriate module automatically

**Access Controls on Intelligence**
- Financial and risk queries restricted to Manager and Admin roles
- Sales users receive redacted responses (sensitive fields like outstanding amounts and payment grades removed)
- Report generation rate-limited to prevent abuse

---

## 4. Data Sovereignty & Security Architecture

### 4.1 Offline-First Design

AsymmFlow operates on a sovereign data model:

| Layer | Technology | Purpose |
|-------|-----------|---------|
| Primary Database | SQLite (local) | All operations, always available, zero latency |
| Cloud Sync | Supabase PostgreSQL | Background backup every 10 minutes, multi-device access |
| Sync Status | Visual indicator | Green (synced), Yellow (syncing), Gray (offline) |

The application functions identically with or without internet connectivity. Cloud sync is a convenience layer, not a dependency. This architecture is particularly relevant for:
- Operations in regions with unreliable internet infrastructure
- Industries with data residency requirements
- Organizations that require operational continuity regardless of external conditions

### 4.2 Security Posture

AsymmFlow underwent a comprehensive security audit using six parallel AI security agents, producing 117 findings across authentication, injection vectors, data security, API validation, infrastructure, and business logic.

**Authentication & Licensing**
- Device-bound license keys generated with cryptographic randomness (`crypto/rand`)
- One license key activates one device (bound to hardware hash)
- Five role tiers: Admin, Manager, Sales, Operations, Staff
- Session timeout after 8 hours of inactivity

**Application Security**
- Role-based access control on all 260 API endpoints
- CSRF token protection (cryptographic, one-hour validity, single-use)
- Parameterized database queries (no SQL injection surface)
- Input sanitization: HTML escaping (XSS), PowerShell command injection prevention, SQL wildcard escaping
- Path traversal prevention on all file operations
- OAuth token storage as SHA-256 hashes (plaintext never persisted)
- 50 MB upload size limit to prevent resource exhaustion

**Financial Controls**
- Payment amounts validated against invoice outstanding balance
- PO financial fields immutable after approval
- Credit limit enforcement with row-level database locking
- Duplicate payment detection across amount, date, and reference fields

---

## 5. Industry Adaptability

AsymmFlow is not a single-purpose application. It is a platform architecture designed for rapid industry-specific deployment.

### 5.1 Current Deployment: Process Instrumentation Trading

**Acme Instrumentation WLL (Bahrain)**
- Industry: Process instrumentation distribution
- Suppliers: Rhine Instruments, Oxan Analytics, Helvetia Metering
- Customers: NPC, Gulf Smelting, NGA
- Users: 8 team members across 4 roles
- Data: 468 customer invoices, 484 supplier invoices, 601 payments, 5 years of financial history

### 5.2 Expansion Verticals

The platform's modular architecture and configurable business rules engine enable deployment across multiple industries:

**Grain Processing & Agricultural Trading**
- Commodity pricing with market-linked rates
- Lot tracking and quality grade management
- Warehouse and silo inventory with moisture/weight parameters
- Seasonal procurement cycles with futures contract tracking
- Export documentation and multi-jurisdiction compliance

**Manufacturing**
- Bill of Materials (BOM) management
- Production order scheduling and tracking
- Work-in-progress inventory valuation
- Quality control checkpoints with pass/fail gates
- Machine utilisation and downtime tracking

**Distribution & Wholesale**
- Multi-warehouse inventory management
- Route-based delivery optimization
- Minimum order quantities and volume pricing
- Consignment stock tracking
- Returns and credit note management

**Professional Services**
- Project-based billing and time tracking
- Resource allocation and utilisation
- Milestone-based invoicing
- Contract management with renewal tracking

### 5.3 What Makes Rapid Adaptation Possible

Each industry deployment leverages the same core platform:
- **88 data models** covering CRM, sales pipeline, finance, procurement, inventory, and system administration
- **31 backend service files** providing modular business logic
- **24 configurable business invariants** that encode industry-specific rules (payment terms, margin thresholds, discount policies, approval workflows)
- **51 frontend screens** covering all operational workflows
- **Butler AI** with domain-aware context injection that adapts to whatever data the system contains

Adding an industry vertical means configuring business rules, extending data models where needed, and adapting the UI — not rebuilding from scratch. The AI-led development methodology (detailed in Section 6) makes this a matter of weeks.

---

## 6. Development Model: AI-Led Engineering

Asymmetrica AI employs a development methodology that fundamentally changes the economics of custom ERP delivery.

### 6.1 The Model

```
Founding Engineer (Architect)
        │
        ├── Defines system architecture and business rules
        ├── Sets security requirements and compliance standards
        ├── Reviews and integrates all AI-generated code
        └── Makes judgment calls on trade-offs and priorities
        │
        ▼
Parallel AI Agents (Builders)
        │
        ├── Agent 1: Feature implementation
        ├── Agent 2: Security audit and hardening
        ├── Agent 3: Data reconciliation and migration
        ├── Agent 4: Testing and validation
        └── Agent N: Additional parallel workstreams as needed
```

### 6.2 How It Works in Practice

**Session-based development**: Each development session has a defined scope, produces a complete deliverable, and updates the system's living architecture document.

**The architecture document (`CLAUDE.md`)**: A comprehensive, continuously updated file that every AI agent reads before touching code. It contains architecture decisions, file purposes, business context, deployment history, and pending work. This is what enables any agent — whether working on security, UI, or data import — to understand the full system context without human re-explanation.

**Parallel execution**: Multiple agents work simultaneously on independent workstreams. During the Phase 17 security audit, six agents covering authentication, injection vectors, data security, API validation, infrastructure, and business logic produced 117 findings concurrently. All critical and high-priority fixes were implemented in the same session.

### 6.3 Delivery Economics

| Metric | Traditional ERP Implementation | AsymmFlow Deployment |
|--------|-------------------------------|---------------------|
| Implementation timeline | 12-18 months | Weeks |
| Team size required | 5-15 developers + consultants | 1 founding engineer + AI agents |
| Security audit | External firm, 2-4 weeks, $20,000-$100,000 | 6 parallel agents, same-day findings and fixes |
| Ongoing licensing | Per-seat annual fees | One-time deployment, $25/month cloud sync |
| Customisation cycle | Change request → scoping → scheduling → development → QA → release (weeks to months) | Define requirement → deploy agents → review → ship (days) |

### 6.4 Current System Scale

| Component | Metric |
|-----------|--------|
| Backend codebase | 108,000+ lines of Go |
| API endpoints | 260 |
| Service modules | 31 |
| Frontend screens | 51 |
| UI components | 104 |
| Database models | 88 |
| Business rules | 24 executable invariants |
| Security permissions | 60+ RBAC checks |

---

## 7. Total Cost of Ownership

### 7.1 AsymmFlow vs. Legacy ERP

| Cost Category | SAP Business One (8 users) | Salesforce (8 users) | AsymmFlow |
|--------------|---------------------------|---------------------|-----------|
| Implementation | $50,000 - $150,000 | $5,000 - $15,000 | Fraction of legacy cost |
| Annual licensing | $15,000 - $40,000 | $12,000 - $30,000 | $0 (no per-seat fees) |
| Cloud infrastructure | Included (vendor-hosted) | Included (vendor-hosted) | $25/month (optional sync) |
| Customisation | $200-$400/hour consulting | Limited without developer | Included in platform methodology |
| Data ownership | Vendor-hosted | Vendor-hosted | 100% client-owned, on-premise |
| Internet dependency | Required | Required | Optional (offline-first) |

### 7.2 What the Client Receives

- Compiled desktop application (Mac + Windows)
- Local database with full business data
- Personalised license keys for each team member
- Cloud sync configuration
- Company letterhead integrated into all PDF outputs
- Role-based access pre-configured
- Installation guide and deployment package

---

## 8. About Asymmetrica AI

Asymmetrica AI is a technology firm specialising in AI-led software development for enterprise operations. Our core thesis: the combination of experienced engineering architecture with parallel AI agent execution produces enterprise-grade software at a speed and cost structure that traditional development cannot match.

**Philosophy**: Research Sovereignty, Build-Test-Ship.

We build systems that our clients own — the data, the infrastructure, the operational continuity. No vendor lock-in, no cloud dependency for core operations, no per-seat rent-seeking.

**Contact**: [To be added]

---

*This document is confidential and intended for prospective clients and partners of Asymmetrica AI.*
