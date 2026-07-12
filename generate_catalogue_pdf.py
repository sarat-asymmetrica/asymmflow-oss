#!/usr/bin/env python3
"""Generate AsymmFlow Product Catalogue PDF — 4 pages, professional layout, India market."""

from fpdf import FPDF
import os

# Colors
DARK = (15, 23, 42)        # slate-900
ACCENT = (99, 102, 241)    # indigo-500
WHITE = (255, 255, 255)
LIGHT_BG = (241, 245, 249)  # slate-100
MID_TEXT = (71, 85, 105)    # slate-500
TABLE_HEAD = (30, 41, 59)   # slate-800
TABLE_ALT = (248, 250, 252) # slate-50
GREEN = (5, 150, 105)       # emerald-600
RED_ACCENT = (220, 38, 38)  # red-600

TOTAL_PAGES = 4
OUTPUT = os.path.join(os.path.dirname(__file__), "AsymmFlow_Product_Catalogue_2026.pdf")


class CataloguePDF(FPDF):
    def __init__(self):
        super().__init__("P", "mm", "A4")
        self.set_auto_page_break(auto=False)

    def _set_color(self, rgb):
        self.set_text_color(*rgb)

    def _fill(self, rgb):
        self.set_fill_color(*rgb)

    def _draw(self, rgb):
        self.set_draw_color(*rgb)

    # -- Reusable components --

    def accent_bar(self):
        """Top accent bar on every page."""
        self._fill(ACCENT)
        self.rect(0, 0, 210, 4, "F")

    def section_title(self, text, y=None):
        if y is not None:
            self.set_y(y)
        self._set_color(DARK)
        self.set_font("Helvetica", "B", 13)
        self.cell(0, 8, text, ln=True)
        self._draw(ACCENT)
        self.set_line_width(0.6)
        self.line(self.l_margin, self.get_y(), self.l_margin + 45, self.get_y())
        self.ln(3)

    def body_text(self, text, size=9, style=""):
        self._set_color(MID_TEXT)
        self.set_font("Helvetica", style, size)
        self.multi_cell(0, 4.5, text)
        self.ln(1)

    def bullet(self, label, desc, lw=38):
        self._set_color(DARK)
        self.set_font("Helvetica", "B", 8.5)
        self.cell(lw, 4.5, label)
        self._set_color(MID_TEXT)
        self.set_font("Helvetica", "", 8.5)
        self.multi_cell(0, 4.5, desc)
        self.ln(0.5)

    def stat_box(self, x, y, w, value, label):
        """Draw a stat card."""
        self._fill(LIGHT_BG)
        self._draw(LIGHT_BG)
        self.rect(x, y, w, 18, "DF")
        self._set_color(ACCENT)
        self.set_font("Helvetica", "B", 16)
        self.set_xy(x, y + 2)
        self.cell(w, 8, value, align="C")
        self._set_color(MID_TEXT)
        self.set_font("Helvetica", "", 7.5)
        self.set_xy(x, y + 10)
        self.cell(w, 5, label, align="C")

    def table(self, headers, rows, col_widths, header_bg=TABLE_HEAD,
              alt_bg=TABLE_ALT, font_size=7.5, row_height=5.5):
        """Generic table renderer."""
        # Header
        self._fill(header_bg)
        self._set_color(WHITE)
        self.set_font("Helvetica", "B", font_size)
        for i, h in enumerate(headers):
            self.cell(col_widths[i], row_height + 1, h, border=0, fill=True,
                      align="C" if i > 0 else "L")
        self.ln()

        # Rows
        for ri, row in enumerate(rows):
            if ri % 2 == 1:
                self._fill(alt_bg)
            else:
                self._fill(WHITE)
            fill = True
            self._set_color(DARK)
            self.set_font("Helvetica", "", font_size)
            for ci, cell_text in enumerate(row):
                align = "C" if ci > 0 else "L"
                style = ""
                color = DARK
                if cell_text == "Yes":
                    color = GREEN
                    style = "B"
                elif cell_text == "Full":
                    color = GREEN
                    style = "B"
                elif cell_text == "No":
                    color = RED_ACCENT
                elif cell_text == "Basic":
                    color = (180, 130, 30)
                elif cell_text.startswith("$") or cell_text.replace(".", "").isdigit():
                    style = "B" if ci == 1 else ""
                self._set_color(color)
                self.set_font("Helvetica", style, font_size)
                self.cell(col_widths[ci], row_height, cell_text, border=0,
                          fill=fill, align=align)
            self.ln()

    def page_footer(self, page_num):
        self.set_y(-12)
        self._draw((200, 210, 220))
        self.set_line_width(0.3)
        self.line(self.l_margin, self.get_y(), self.w - self.r_margin, self.get_y())
        self._set_color((160, 170, 185))
        self.set_font("Helvetica", "", 7)
        self.ln(2)
        self.cell(0, 4,
                  f"AsymmFlow Product Catalogue 2026  |  Asymmetrica  |  Page {page_num} of {TOTAL_PAGES}",
                  align="C")

    # =============================================
    #  PAGE 1 — Hero & Overview
    # =============================================

    def page1(self):
        self.add_page()
        self.accent_bar()

        # Title block
        self.set_y(12)
        self._set_color(DARK)
        self.set_font("Helvetica", "B", 28)
        self.cell(0, 12, "AsymmFlow", ln=True)
        self._set_color(ACCENT)
        self.set_font("Helvetica", "", 11)
        self.cell(0, 5, "Enterprise ERP for Trading & Distribution", ln=True)
        self.ln(3)

        # Tagline
        self._set_color(DARK)
        self.set_font("Helvetica", "B", 14)
        self.cell(0, 7, "Your Data. Your Machine. Your Rules.", ln=True)
        self.ln(2)

        # Intro
        self._set_color(MID_TEXT)
        self.set_font("Helvetica", "", 9)
        self.multi_cell(0, 4.5,
            "AsymmFlow is a sovereign, offline-first ERP system purpose-built for trading and "
            "distribution companies across India. It runs entirely on your hardware  - no cloud "
            "dependency, no recurring subscription, no vendor lock-in  - with optional multi-device "
            "sync when you need it."
            "\n\n"
            "Designed for companies serving government departments, public sector undertakings, and "
            "large industrial clients who demand airtight traceability, regulatory compliance, and the "
            "assurance that their business data never leaves their control."
        )
        self.ln(3)

        # Stat boxes
        bx = self.l_margin
        by = self.get_y()
        bw = 31
        gap = 4.6
        stats = [
            ("53", "Business Screens"),
            ("6", "Integrated Hubs"),
            ("70+", "Secured Functions"),
            ("5", "User Roles"),
            ("<35 MB", "Single-File Install"),
        ]
        for i, (val, label) in enumerate(stats):
            self.stat_box(bx + i * (bw + gap), by, bw, val, label)
        self.set_y(by + 24)

        # At a Glance
        self.section_title("At a Glance")
        glance_items = [
            ("Architecture",
             "Desktop application  - installs in seconds, no server infrastructure required"),
            ("Database",
             "Local-first storage with optional cloud sync for multi-location operations"),
            ("Compliance",
             "E-invoicing framework, configurable tax return export, document integrity verification"),
            ("AI",
             "Butler Intelligence  - natural language queries, OCR document processing, predictive analytics"),
            ("Security",
             "Role-based access control (5 roles), device-bound licensing, 70+ permission-guarded functions"),
            ("Deployment",
             "Single file  - Mac (32 MB) or Windows (35 MB). No installer, no dependencies, no IT overhead."),
        ]
        for label, desc in glance_items:
            self.bullet(label, desc, lw=25)
        self.ln(2)

        # Six Hubs
        self.section_title("Six Integrated Hubs")
        hubs = [
            ("Sales Hub",
             "Full opportunity pipeline, RFQ processing, costing sheets with margin enforcement, "
             "five quotation document types, PDF on letterhead, one-click costing-to-offer conversion, "
             "customer order lifecycle, win-rate analytics."),
            ("Finance Hub",
             "P&L and balance sheet dashboards, invoicing with integrity hashing, credit notes, "
             "e-invoicing, tax return export, bank reconciliation, book-bank reconciliation, cheque "
             "register, cash position, FX revaluation, general ledger, A/R aging, cash flow projection."),
            ("Operations Hub",
             "Multi-currency purchase orders with approval thresholds, GRN with QC inspection, delivery "
             "notes with serial allocation, end-to-end serial traceability PO to GRN to DN to Invoice, "
             "warranty tracking, 3-way matching, shipment management."),
            ("Relationships Hub",
             "360-degree customer and supplier profiles, KPI dashboards, contact management, supplier "
             "invoice verification, automated deduplication, configurable customer grading."),
            ("Intelligence Hub",
             "Butler AI: natural language queries across your dataset, OCR document analysis with "
             "drag-and-drop, conversation persistence, interactive data visualisations, predictive "
             "analytics, automated PDF report generation."),
            ("Administration",
             "License-based activation with device binding, user and role management, audit trail, "
             "cloud sync configuration, automated backup with integrity verification on every startup."),
        ]
        for label, desc in hubs:
            self._set_color(ACCENT)
            self.set_font("Helvetica", "B", 8.5)
            self.cell(0, 4.5, label, ln=True)
            self._set_color(MID_TEXT)
            self.set_font("Helvetica", "", 8)
            self.multi_cell(0, 4, desc)
            self.ln(1.5)

        self.page_footer(1)

    # =============================================
    #  PAGE 2 — Sales & Finance Capabilities
    # =============================================

    def page2(self):
        self.add_page()
        self.accent_bar()

        self.set_y(10)
        self._set_color(DARK)
        self.set_font("Helvetica", "B", 20)
        self.cell(0, 10, "Sales & Finance Capabilities", ln=True)
        self.ln(2)

        cw = [42, 138]

        # -- Sales Pipeline --
        self.section_title("Sales Pipeline")
        self.table(
            ["Capability", "Detail"],
            [
                ["Opportunity Mgmt",
                 "Full pipeline with configurable stages: Qualified, Proposal, Quoted, Won, Lost"],
                ["RFQ Pipeline",
                 "Multi-product line items, stage progression, win rate tracking, folder organisation"],
                ["Costing Sheets",
                 "Margin calculations with configurable minimum enforcement and advance payment rules"],
                ["5 Document Types",
                 "Quotation, Budgetary Quote, Estimate, Technical Proposal, Commercial Proposal"],
                ["PDF Generation",
                 "Company letterhead, annexure pages, serial listings, dynamic tax labeling"],
                ["Offer Conversion",
                 "One-click costing-to-offer preserving all terms, pricing, delivery terms, metadata"],
                ["Hidden Charges",
                 "Internal cost adjustments affecting margin only - never on customer documents"],
                ["Pipeline Analytics",
                 "Year-wise value tracking, stage distribution analysis, owner-level performance"],
                ["Customer Orders",
                 "Full order lifecycle - creation, status tracking, order-to-invoice pipeline"],
            ],
            cw, font_size=7.5, row_height=5
        )
        self.ln(4)

        # -- Finance & Compliance --
        self.section_title("Finance & Compliance")
        self.table(
            ["Capability", "Detail"],
            [
                ["Financial Dashboard",
                 "Revenue, gross profit, net profit, cash position with year-over-year indicators"],
                ["Balance Sheet",
                 "Current/non-current assets, liabilities, equity with visual bar charts"],
                ["14 Financial Ratios",
                 "Liquidity (Current, Quick, Cash), Solvency, Efficiency (DSO/DIO/DPO/CCC), ROA/ROE"],
                ["A/R Aging",
                 "Current, 30-60, 60-90, 90+ day buckets with overdue percentages"],
                ["Cash Flow Projection",
                 "Forward-looking projections based on receivables, payables, collection probability"],
                ["Customer Invoicing",
                 "Create from orders, PDF with serial numbers, status pipeline, integrity hashing"],
                ["E-Invoicing",
                 "Structured electronic invoice generation - adaptable to GST and intl standards"],
                ["Tax Return Export",
                 "Periodic CSV with configurable exclusion filters (cancels, voids, proforma, drafts)"],
                ["Credit Notes",
                 "Draft > Issued > Applied workflow with row-locked over-crediting prevention"],
                ["Payments",
                 "Record against invoices, partial payments, KPI dashboard, balance reconciliation"],
                ["Bank Reconciliation",
                 "PDF/CSV statement import, automated matching, manual matching, verification"],
                ["Book-Bank Recon",
                 "Side-by-side balance comparison, adjustments tracking, variance analysis"],
                ["Cheque Register",
                 "Full lifecycle: Issued > Presented > Cleared > Stale > Bounced"],
                ["Cash Position",
                 "Real-time balances across accounts, multi-currency, outstanding cheque tracking"],
                ["FX Revaluation",
                 "Exchange rate management, unrealised gain/loss, multi-currency exposure analysis"],
                ["General Ledger",
                 "Complete GL with journal entries, chart of accounts, transaction history"],
                ["Document Integrity",
                 "HMAC-SHA256 hashes on invoices and credit notes - tamper-evident verification"],
            ],
            cw, font_size=7.5, row_height=4.5
        )

        self.page_footer(2)

    # =============================================
    #  PAGE 3 — Operations, Intelligence & Security
    # =============================================

    def page3(self):
        self.add_page()
        self.accent_bar()

        self.set_y(10)
        self._set_color(DARK)
        self.set_font("Helvetica", "B", 20)
        self.cell(0, 10, "Operations, Intelligence & Security", ln=True)
        self.ln(2)

        cw = [42, 138]

        # -- Operations --
        self.section_title("Operations & Supply Chain")
        self.table(
            ["Capability", "Detail"],
            [
                ["Purchase Orders",
                 "Multi-currency, enforced state machine (Draft to Closed), configurable approval thresholds"],
                ["Goods Received Notes",
                 "Create from POs or standalone, QC status (Pending/Passed/Failed/Partial), serial capture"],
                ["Delivery Notes",
                 "Serial allocation with atomic availability check, enforced status guards"],
                ["Serial Traceability",
                 "Full lifecycle: PO > GRN > Inventory > DN > Delivery > Invoice > Customer"],
                ["Warranty Tracking",
                 "Auto-calculated end dates on delivery, calibration certificate attachment"],
                ["3-Way Matching",
                 "Supplier invoice verified against PO and GRN - visual match status indicator"],
                ["Shipment Mgmt",
                 "Pending, in-transit, delivered tracking with delivery confirmation workflow"],
            ],
            cw, font_size=7.5, row_height=4.5
        )
        self.ln(3)

        # -- CRM --
        self.section_title("CRM & Relationships")
        self.table(
            ["Capability", "Detail"],
            [
                ["Customer 360",
                 "Orders, invoices, contacts, notes, AI predictions - unified customer view"],
                ["Supplier 360",
                 "Purchase orders, invoices, payment history - complete supplier lifecycle"],
                ["KPI Dashboards",
                 "Year-over-year performance metrics for customers and suppliers, dynamic filtering"],
                ["Contact Management",
                 "Multiple contacts per entity with role designation and communication tracking"],
                ["Customer Grading",
                 "Configurable A/B/C/D grading with grade-based payment term enforcement"],
                ["Data Quality",
                 "Automated deduplication, name normalisation, variant matching, orphan identification"],
            ],
            cw, font_size=7.5, row_height=4.5
        )
        self.ln(3)

        # -- Intelligence --
        self.section_title("Intelligence & AI")
        self.table(
            ["Capability", "Detail"],
            [
                ["NL Queries",
                 "Ask business questions in plain English across your entire dataset"],
                ["OCR Processing",
                 "Drag-and-drop document analysis with entity extraction and line item recognition"],
                ["Conversation History",
                 "Persistent chat context - resume any prior business inquiry"],
                ["Data Visualisation",
                 "Interactive D3 graphs, entity relationship mapping, force-directed layouts"],
                ["Predictive Analytics",
                 "Win probability scoring on pipeline opportunities using historical patterns"],
                ["Report Generation",
                 "Automated PDF reports on company letterhead - one-click export"],
            ],
            cw, font_size=7.5, row_height=4.5
        )
        self.ln(3)

        # -- Security --
        self.section_title("Security & Access Control")
        self.table(
            ["Layer", "Detail"],
            [
                ["Authentication",
                 "Device-bound license keys with cryptographic generation, master key for admins"],
                ["Authorisation",
                 "5 roles (Admin/Manager/Sales/Operations/Staff), permissions on 70+ functions"],
                ["Data Protection",
                 "HMAC-SHA256 document integrity, hashed tokens, PBKDF2 key derivation"],
                ["Input Security",
                 "SQL injection, XSS, path traversal, and command injection prevention"],
                ["Business Controls",
                 "Credit limits, approval thresholds, serial allocation guards, state machines"],
                ["Audit & Recovery",
                 "Atomic backup every startup, integrity verification, session timeout, log rotation"],
            ],
            cw, font_size=7.5, row_height=4.5
        )
        self.ln(3)

        # -- Infrastructure bullets --
        self.section_title("Infrastructure")
        infra = [
            ("Offline-First",
             "Complete functionality without internet - no exceptions, no degradation"),
            ("Auto Backup",
             "Atomic database backup every startup, integrity check, last 7 retained"),
            ("Multi-Device Sync",
             "Bidirectional cloud sync every 10 minutes across all business tables"),
            ("Status Indicator",
             "Real-time sync: Green (connected), Yellow (syncing), Grey (offline)"),
            ("Graceful Shutdown",
             "All operations complete before close - zero data loss risk"),
            ("DB Performance",
             "Write-ahead logging, 20 MB cache, 256 MB memory-mapped I/O"),
            ("Log Management",
             "50 MB rotation, 5 archived logs, automatic credential sanitisation"),
        ]
        for label, desc in infra:
            self.bullet(label, desc, lw=28)

        self.page_footer(3)

    # =============================================
    #  PAGE 4 — Market Positioning
    # =============================================

    def page4(self):
        self.add_page()
        self.accent_bar()

        self.set_y(10)
        self._set_color(DARK)
        self.set_font("Helvetica", "B", 20)
        self.cell(0, 10, "Market Positioning", ln=True)
        self.ln(2)

        # -- Feature Comparison --
        self.section_title("Feature Comparison")
        cw5 = [48, 24, 28, 28, 28, 24]
        self.table(
            ["Capability", "Asymm", "SAP B1", "NetSuite", "Odoo", "Tally"],
            [
                ["Works fully offline",       "Yes", "Partial", "No",     "No",     "Yes"],
                ["No subscription required",   "Yes", "No",      "No",     "No",     "Yes"],
                ["Data on your machine",       "Yes", "Option",  "No",     "Option", "Yes"],
                ["E-invoicing support",        "Yes", "Add-on",  "Add-on", "Module", "Yes"],
                ["Tax return export",          "Yes", "Partner", "Config", "Module", "Yes"],
                ["AI natural language queries", "Yes", "Extra",   "Basic",  "Module", "No"],
                ["OCR document analysis",      "Yes", "No",      "No",     "Module", "No"],
                ["Serial traceability",        "Full", "Yes",    "Yes",    "Yes",    "Basic"],
                ["Document integrity hash",    "Yes", "No",      "No",     "No",     "No"],
                ["Multi-device sync",          "Yes", "Yes",     "Yes",    "Yes",    "Yes"],
                ["Costing with margin rules",  "Yes", "Manual",  "Manual", "Manual", "No"],
                ["3-way PO/GRN/Inv match",     "Yes", "Yes",     "Yes",    "Yes",    "No"],
                ["Single-file deployment",     "Yes", "No",      "No",     "No",     "No"],
                ["Implementation time",        "Days", "3-6 mo", "3-6 mo", "1-3 mo", "Weeks"],
            ],
            cw5, font_size=7, row_height=4.5
        )
        self.ln(3)

        # -- TCO Comparison --
        self.section_title("Total Cost of Ownership (10 Users, 3 Years)")
        # Note about currency
        self._set_color(MID_TEXT)
        self.set_font("Helvetica", "I", 7)
        self.cell(0, 4, "All figures in INR Lakhs", ln=True)
        self.ln(1)

        cw_tco = [38, 30, 34, 34, 30, 30]
        self.table(
            ["", "Asymm", "SAP B1", "NetSuite", "Odoo", "Tally"],
            [
                ["Year 1 (w/ setup)", "12.5",  "48.5",  "54.0",  "18.5",  "4.5"],
                ["Year 2",            "0",     "24.0",  "20.0",  "10.5",  "1.0"],
                ["Year 3",            "0",     "24.0",  "20.0",  "10.5",  "1.0"],
                ["3-Year Total",      "12.5",  "96.5",  "94.0",  "39.5",  "6.5"],
            ],
            cw_tco, font_size=7.5, row_height=5.5
        )
        self._set_color(MID_TEXT)
        self.set_font("Helvetica", "I", 6.5)
        self.ln(1)
        self.multi_cell(0, 3.5,
            "Estimates based on published 2025-2026 pricing. SAP/NetSuite include typical implementation "
            "costs. AsymmFlow is a one-time deployment. Tally lacks advanced capabilities. "
            "Actual costs vary by partner and configuration."
        )
        self.ln(3)

        # -- Positioning --
        self.section_title("Why AsymmFlow")

        positions = [
            ("vs. Enterprise ERPs (SAP / Oracle NetSuite)",
             "6-8x lower three-year cost of ownership. No implementation consultants. No mandatory "
             "cloud. Same core capabilities - invoicing, procurement, traceability, reconciliation, "
             "reporting - without the enterprise overhead."),
            ("vs. Mid-Market (Odoo)",
             "Comparable features at lower TCO. The decisive advantage is architecture: Odoo requires "
             "a server, a DBA, module configuration, and hosting. AsymmFlow is a single binary on "
             "any office laptop."),
            ("vs. Tally Prime",
             "Same price range, fundamentally different capability. Tally lacks serial traceability, "
             "AI queries, costing automation, and RBAC. AsymmFlow is what Tally would be if rebuilt "
             "for modern Indian distribution."),
        ]
        for label, desc in positions:
            self._set_color(ACCENT)
            self.set_font("Helvetica", "B", 8.5)
            self.cell(0, 5, label, ln=True)
            self._set_color(MID_TEXT)
            self.set_font("Helvetica", "", 7.5)
            self.multi_cell(0, 3.8, desc)
            self.ln(1.5)

        # -- Technical Specs --
        self.section_title("Technical Specifications")
        specs = [
            ("Platforms", "macOS (Apple Silicon + Intel), Windows 10/11 (x64)"),
            ("Binary Size", "32 MB (Mac), 35 MB (Windows)"),
            ("Memory", "~80 MB typical working set"),
            ("Storage", "~50 MB (app + 3 years of data)"),
            ("Network", "Optional  - sync needs internet, everything else works offline"),
            ("Updates", "Binary replacement  - no migration scripts, no downtime"),
        ]
        for label, desc in specs:
            self.bullet(label, desc, lw=22)

        # -- Bottom CTA --
        self.ln(2)
        cta_y = self.get_y()
        self._fill(DARK)
        self.rect(self.l_margin, cta_y, self.w - self.l_margin - self.r_margin, 16, "F")
        self._set_color(WHITE)
        self.set_font("Helvetica", "B", 9.5)
        self.set_xy(self.l_margin, cta_y + 3)
        self.cell(0, 5,
                  "For trading and distribution companies across India with 5-50 users:",
                  align="C", ln=True)
        self._set_color((180, 190, 255))
        self.set_font("Helvetica", "B", 9.5)
        self.cell(0, 5,
                  "Enterprise-grade ERP without enterprise-grade complexity or cost.",
                  align="C", ln=True)

        self.page_footer(4)


def main():
    pdf = CataloguePDF()
    pdf.set_margins(15, 10, 15)
    pdf.page1()
    pdf.page2()
    pdf.page3()
    pdf.page4()
    pdf.output(OUTPUT)
    print(f"PDF generated: {OUTPUT}")
    print(f"Size: {os.path.getsize(OUTPUT) / 1024:.0f} KB")


if __name__ == "__main__":
    main()
