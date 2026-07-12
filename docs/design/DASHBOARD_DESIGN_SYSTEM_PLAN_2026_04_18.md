# Dashboard Design System Plan - 2026-04-18

## Direction

The customer reference is not asking for a marketing dashboard. It is a decision dashboard: calm background, sparse top metrics, compact operating context, soft severity color, and almost no navigation chrome inside the page.

The core design idea should be:

- one quiet shell color for the app canvas
- white translucent panels with 8px radius
- four primary numbers only
- color used only for business state: green collected, red credit risk, amber overdue, blue pipeline
- cards as information surfaces, not menu buttons
- no generic KPI zoo on the landing dashboard

## Dashboard Changes Applied

- Removed dashboard action buttons for overdue invoices, open opportunities, and active orders.
- Removed KPI cards for open opportunities, active orders, active customers, and win rate.
- Removed the OCR browse/drop strip from the landing dashboard.
- Replaced the top area with four primary metrics:
  - Revenue
  - Outstanding
  - Pipeline
  - Collection Rate
- Added backend values for `pipeline_value_bhd` and `collection_rate` so the new dashboard is not guessing.
- Reframed the main content as operating focus, alerts, collections, recent activity, and cash attention.

## How We Reach The Reference System

### Pass 1 - Dashboard Tokenization

Create shared design tokens for:

- shell background: `#edf3f7`
- panel background: white with subtle transparency
- panel border: cool grey-blue
- text primary: dark blue-grey
- text muted: medium blue-grey
- severity red, amber, green, blue
- radius: 6px for inner rows, 8px for panels
- shadows: one very soft elevation only

This is easy. It can be done without changing app logic.

### Pass 2 - Component Kit

Extract the dashboard patterns into reusable components:

- `MetricTile`
- `DecisionPanel`
- `AlertRow`
- `ActivityRow`
- `ProgressBar`
- `CashStrip`

This is medium effort because the current app already has `Card`, `KPICard`, and other components with a different visual language. We should avoid breaking the rest of the app by creating the new kit beside the old one, then migrating page by page.

### Pass 3 - Data Contracts

The reference dashboard needs richer operating data than the old dashboard had:

- top overdue customers
- invoice ageing by customer
- collection pressure by limit breach
- recent activity from invoices, orders, quotations, payments, and tasks
- cash forecast by 30/60/90 day windows

This is the real work. The UI is straightforward, but the dashboard becomes valuable only if these are honest backend queries. We should add one purpose-built endpoint, probably `GetExecutiveDashboard()`, instead of overloading `GetDashboardStats()`.

### Pass 4 - Page-By-Page Migration

Use the dashboard as the north star, then migrate:

- Finance Hub
- Opportunities
- Customer Orders
- Operations
- CRM customer/supplier dashboards
- Work Hub

Each page should get one clear job, fewer buttons, and the same shell/panel/row system.

## Difficulty

The visual stage is easy to reach. The reference look is mostly disciplined spacing, restrained typography, and better hierarchy.

The business stage is moderate. To make the dashboard truly like the customer reference, we need richer backend summaries for overdue customers, alerts, recent activity, and cash forecast. That is not hard, but it needs careful accounting rules so the numbers do not become decorative.

## Recommended Next Step

Build `GetExecutiveDashboard()` and replace the remaining placeholder-derived blocks with real rows:

- top 3 overdue customers
- top 4 alerts
- last 5 activity events
- 30/60/90 day receivable forecast

After that, we can turn the dashboard components into a reusable visual system for the rest of the app.
