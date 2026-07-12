# Enterprise Design System Components

**Philosophy:** Apple-level polish × Bloomberg-level data density
**Version:** 1.0
**Date:** January 22, 2026

---

## Overview

This document details the three core components of the Enterprise ERP/CRM Design System:
1. **Button** - Primary interaction component with multiple variants and states
2. **Card** - Content container with flexible layouts
3. **KPICard** - Specialized component for key performance indicators

All components follow the design tokens defined in `design-tokens.css` and adhere to WCAG 2.1 AA accessibility standards.

---

## Button Component

### Import
```typescript
import { Button } from '$lib/components/ui';
```

### Props
| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `variant` | `'primary' \| 'secondary' \| 'ghost' \| 'danger'` | `'primary'` | Visual style variant |
| `size` | `'sm' \| 'md' \| 'lg'` | `'md'` | Size of the button |
| `type` | `'button' \| 'submit' \| 'reset'` | `'button'` | HTML button type |
| `disabled` | `boolean` | `false` | Disabled state |
| `loading` | `boolean` | `false` | Loading state with spinner |
| `fullWidth` | `boolean` | `false` | Full width button |
| `aria-label` | `string` | `undefined` | Accessibility label |

### Usage Examples

#### Basic Variants
```svelte
<Button variant="primary" on:click={handleSubmit}>
  Save Changes
</Button>

<Button variant="secondary" on:click={handleCancel}>
  Cancel
</Button>

<Button variant="ghost" on:click={handleReset}>
  Reset
</Button>

<Button variant="danger" on:click={handleDelete}>
  Delete Record
</Button>
```

#### Sizes
```svelte
<Button size="sm">Small Button</Button>
<Button size="md">Medium Button</Button>
<Button size="lg">Large Button</Button>
```

#### States
```svelte
<!-- Disabled -->
<Button disabled>Cannot Click</Button>

<!-- Loading -->
<Button loading on:click={handleAsync}>
  {loading ? 'Processing...' : 'Submit'}
</Button>

<!-- Full Width -->
<Button fullWidth>Full Width Action</Button>
```

#### With Icons (using slots)
```svelte
<Button variant="primary">
  <svg width="16" height="16" viewBox="0 0 16 16">...</svg>
  Save Changes
</Button>
```

### Design Specifications

**Primary Button:**
- Background: `var(--brand-indigo)` (#2F2DFF)
- Text: White
- Hover: Indigo glow shadow
- Active: Darker indigo press state

**Secondary Button:**
- Background: Transparent
- Border: 1px solid `var(--border)`
- Hover: Elevated background + subtle shadow

**Ghost Button:**
- Background: Transparent
- Hover: 4% indigo tint background

**Danger Button:**
- Background: Red (#DC2626)
- Text: White
- Hover: Red glow shadow

### Accessibility
- All buttons include proper `type` attribute
- Disabled buttons have `aria-disabled` and `pointer-events: none`
- Focus visible states with 2px indigo outline
- Loading state announces with spinner role

---

## Card Component

### Import
```typescript
import { Card } from '$lib/components/ui';
```

### Props
| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `variant` | `'default' \| 'elevated' \| 'accent'` | `'default'` | Visual style variant |
| `padding` | `'sm' \| 'md' \| 'lg'` | `'md'` | Internal padding |
| `hoverable` | `boolean` | `false` | Enables hover effects and cursor pointer |
| `aria-label` | `string` | `undefined` | Accessibility label |

### Usage Examples

#### Basic Card
```svelte
<Card variant="default" padding="md">
  <h3 class="label">Card Title</h3>
  <p class="text-primary">Card content goes here.</p>
  <p class="meta">Additional metadata</p>
</Card>
```

#### Elevated Card
```svelte
<Card variant="elevated" padding="lg">
  <h2 class="section-title">Important Section</h2>
  <div class="content">
    <!-- Elevated cards have higher shadow for emphasis -->
  </div>
</Card>
```

#### Accent Card (with 3px indigo border)
```svelte
<Card variant="accent" padding="md">
  <h3 class="label">Featured Content</h3>
  <p class="text-primary">This card has a 3px indigo left border.</p>
</Card>
```

#### Hoverable Interactive Card
```svelte
<Card variant="default" padding="md" hoverable on:click={handleCardClick}>
  <h3 class="label">Click Me</h3>
  <p class="text-primary">This card responds to hover and click.</p>
</Card>
```

#### Different Padding Sizes
```svelte
<!-- Small padding (8px) -->
<Card padding="sm">Compact card</Card>

<!-- Medium padding (12px) - default -->
<Card padding="md">Standard card</Card>

<!-- Large padding (14px) -->
<Card padding="lg">Spacious card</Card>
```

### Design Specifications

**Default Card:**
- Background: `var(--surface)` (white in light mode)
- Shadow: `var(--shadow-sm)` (subtle 3px shadow)
- Border-radius: `var(--border-radius)` (10px)

**Elevated Card:**
- Background: `var(--surface-elevated)`
- Shadow: `var(--shadow-md)` (medium 12px shadow)

**Accent Card:**
- Border-left: 3px solid `var(--brand-indigo)`
- All other properties same as default

**Hoverable Card:**
- Cursor: pointer
- Hover: Elevates shadow from sm to md
- Focus-visible: 2px indigo outline

### Accessibility
- Hoverable cards have `role="button"` and `tabindex="0"`
- Keyboard navigation supported (Enter/Space to activate)
- Focus visible states with indigo outline

---

## KPICard Component

### Import
```typescript
import { KPICard } from '$lib/components/ui';
```

### Props
| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `label` | `string` | (required) | KPI label/title (uppercase) |
| `value` | `string` | (required) | KPI value (large, bold) |
| `meta` | `string` | `''` | Metadata text (small, muted) |
| `trend` | `'up' \| 'down' \| 'neutral'` | `'neutral'` | Trend indicator |
| `accent` | `boolean` | `false` | Add 3px indigo left border |

### Usage Examples

#### Basic KPI Card
```svelte
<KPICard
  label="Total Revenue"
  value="$2,450,000"
  meta="YTD Performance"
/>
```

#### With Trend Indicators
```svelte
<!-- Upward trend (green) -->
<KPICard
  label="Conversion Rate"
  value="68.2%"
  meta="Last 30 Days"
  trend="up"
/>

<!-- Downward trend (red) -->
<KPICard
  label="Cash Runway"
  value="18 Months"
  meta="Based on Burn Rate"
  trend="down"
/>
```

#### With Accent Border
```svelte
<KPICard
  label="Active Opportunities"
  value="34"
  meta="Pipeline Count"
  accent
/>
```

#### Using Footer Slot
```svelte
<KPICard
  label="Monthly Recurring Revenue"
  value="$125,000"
  meta="Last updated: 2 hours ago"
>
  <div slot="footer" class="kpi-actions">
    <Button size="sm" variant="ghost">View Details</Button>
  </div>
</KPICard>
```

### Design Specifications

**Layout:**
- Height: Fixed at `var(--kpi-card-height)` (130px)
- Padding: `var(--card-padding)` (12px)
- Display: Flexbox column
- Gap: 8px between elements

**Typography:**
- **Label**: 12px, 600 weight, uppercase, secondary color, 0.05em letter-spacing
- **Value**: 32px, 700 weight, primary color, line-height 1.2
- **Meta**: 11px, 400 weight, muted color
- **Trend**: 12px, 600 weight, green (#10B981) or red (#EF4444)

**Trend Icons:**
- Up arrow: Green, 16×16px SVG
- Down arrow: Red, 16×16px SVG
- Position: Inline with value, aligned to baseline

### Grid Layout for Dashboards

KPI Cards are designed to fit in a 4-column grid on desktop:

```svelte
<div class="kpi-grid">
  <KPICard label="Revenue" value="$2.45M" meta="YTD" trend="up" />
  <KPICard label="Opportunities" value="34" meta="Active" />
  <KPICard label="Conversion" value="68.2%" meta="Last 30d" trend="up" />
  <KPICard label="Cash Runway" value="18 Mo" meta="Current" trend="down" />
</div>

<style>
  .kpi-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: var(--grid-gap); /* 12px */
  }
</style>
```

### Accessibility
- Semantic HTML structure
- Color is not the only indicator (trend includes icon)
- Proper contrast ratios (WCAG AA compliant)
- Can be made focusable if interactive

---

## Common Patterns

## Product Operator Components

These components capture reusable MVVM product surfaces proven by Cashflow Evidence. They render display-ready state and emit operator intent through callbacks; backend/domain services remain authoritative for persistence, posting, approval, export, and audit.

### Import

```typescript
import {
  KpiStatusStrip,
  EvidenceSourceList,
  ActionProposalCard,
} from '$lib/components/ui';
```

### KpiStatusStrip

Use for compact command-center metrics where operators need to scan state, amount, and supporting context.

```svelte
<KpiStatusStrip
  items={[
    { label: 'Attention', value: 'BHD 42.8K', meta: 'BHD 18.2K overdue', status: 'review' },
    { label: 'Posting', value: '3 missing', meta: '2 drafts', status: 'review' },
    { label: 'Evidence Pack', value: '17 items', meta: '5 follow-ups', status: 'ready' },
  ]}
/>
```

### EvidenceSourceList

Use for provenance, completeness, confidence, and missing-evidence surfaces.

```svelte
<EvidenceSourceList
  sources={[
    { label: 'Receivables', required: 14, present: 9, missing: 5, confidence: 0.64, status: 'review', priority: 'high' },
    { label: 'Invoice Links', required: 11, present: 11, missing: 0, confidence: 1, status: 'ready', priority: 'low' },
  ]}
/>
```

### ActionProposalCard

Use for advisory actions that point to deterministic services. The component may show review controls, but the caller owns the handler and must route real mutations through backend authority.

```svelte
<ActionProposalCard
  proposal={proposal}
  reviewLabel={proposal.required_deterministic_service}
  hasReview={Boolean(review)}
  onApprove={() => reviewProposal(proposal, 'approved')}
  onNeedsInput={() => reviewProposal(proposal, 'needs_input')}
  onReject={() => reviewProposal(proposal, 'rejected')}
/>
```

Good module fits:

- Cashflow Evidence command centers.
- Business Memory Intake review queues.
- Compliance readiness and filing evidence.
- Inventory or asset evidence ledgers.
- Sync/support-bundle readiness surfaces.

### Dashboard KPI Section
```svelte
<script>
  import { KPICard } from '$lib/components/ui';

  const kpis = [
    { label: 'Total Revenue', value: '$2,450,000', meta: 'YTD Performance', trend: 'up' },
    { label: 'Active Opportunities', value: '34', meta: 'Pipeline Count', trend: 'neutral' },
    { label: 'Conversion Rate', value: '68.2%', meta: 'Last 30 Days', trend: 'up' },
    { label: 'Cash Runway', value: '18 Months', meta: 'Based on Burn Rate', trend: 'down' },
  ];
</script>

<section class="dashboard-kpis">
  <h2 class="section-title">Key Metrics</h2>
  <div class="kpi-grid">
    {#each kpis as kpi}
      <KPICard {...kpi} />
    {/each}
  </div>
</section>

<style>
  .dashboard-kpis {
    margin-bottom: var(--section-spacing);
  }

  .kpi-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: var(--grid-gap);
  }
</style>
```

### Action Panel with Cards
```svelte
<script>
  import { Card, Button } from '$lib/components/ui';

  function handleAction() {
    console.log('Action triggered');
  }
</script>

<div class="action-panel">
  <Card variant="accent" padding="lg">
    <h3 class="label">Quick Actions</h3>
    <p class="text-secondary">Perform common tasks quickly.</p>

    <div class="button-group">
      <Button variant="primary" on:click={handleAction}>
        Create Invoice
      </Button>
      <Button variant="secondary" on:click={handleAction}>
        New Customer
      </Button>
      <Button variant="ghost" on:click={handleAction}>
        View Reports
      </Button>
    </div>
  </Card>
</div>

<style>
  .button-group {
    display: flex;
    gap: 12px;
    margin-top: 16px;
  }
</style>
```

---

## Design Tokens Reference

All components use the following CSS custom properties from `design-tokens.css`:

### Colors
- `--brand-indigo`: #2F2DFF (primary brand color)
- `--brand-indigo-hover`: #2624D9
- `--brand-indigo-pressed`: #1E1CB3
- `--brand-indigo-tint`: rgba(47, 45, 255, 0.04) (4% overlay)
- `--text-primary`: #0E1020 (dark text)
- `--text-secondary`: #5B5F7A (medium text)
- `--text-muted`: #8C90A8 (light text)
- `--surface`: #FFFFFF (card background)
- `--border`: #E6E8F0 (borders)

### Spacing
- `--card-padding`: 12px
- `--card-padding-lg`: 14px
- `--grid-gap`: 12px
- `--section-spacing`: 16px
- `--kpi-card-height`: 130px

### Typography
- `--label-size`: 12px (KPI labels, small text)
- `--label-weight`: 500
- `--meta-size`: 11px (metadata text)
- `--section-title-size`: 16px

### Borders & Radius
- `--border-radius`: 10px
- `--border-radius-sm`: 8px (buttons)
- `--border-radius-lg`: 12px

### Shadows
- `--shadow-sm`: 0 1px 3px rgba(0, 0, 0, 0.05)
- `--shadow-md`: 0 4px 12px rgba(0, 0, 0, 0.08)
- `--shadow-indigo`: 0 4px 12px rgba(47, 45, 255, 0.24)

### Transitions
- `--transition-fast`: 150ms cubic-bezier(0.4, 0.0, 0.2, 1)
- `--transition-base`: 200ms cubic-bezier(0.4, 0.0, 0.2, 1)

---

## Testing

A comprehensive showcase component is available for visual testing:

```svelte
import DesignSystemShowcase from '$lib/components/ui/DesignSystemShowcase.svelte';
```

This component demonstrates:
- All button variants and states
- All card variants
- KPI cards with different configurations
- Design tokens reference
- Interactive examples

---

## Accessibility Checklist

All components meet the following standards:

- [ ] WCAG 2.1 AA contrast ratios
- [ ] Keyboard navigation support
- [ ] Focus visible states (2px indigo outline)
- [ ] Proper ARIA labels and roles
- [ ] Screen reader announcements for state changes
- [ ] Disabled states properly communicated
- [ ] Interactive elements have appropriate cursor styles

---

## Future Enhancements

Components planned for next iteration:
- Input (text, email, number variants)
- Select (dropdown with search)
- Textarea (auto-resize)
- Toggle (on/off switch)
- DatePicker (calendar interface)
- FormGroup (label + input + error wrapper)

---

**Om Lokah Samastah Sukhino Bhavantu**
*May all users benefit from calm, capable, premium software.*
