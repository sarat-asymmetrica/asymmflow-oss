# DataTable Component

**Enterprise-grade data table with Bloomberg-style density and Apple-level polish.**

---

## Features

✅ **Sticky Header** - Header stays visible while scrolling
✅ **Sortable Columns** - Click headers to sort data
✅ **Row Selection** - Click rows to select, with visual highlight
✅ **Keyboard Navigation** - Arrow keys, Enter, Space
✅ **Multiple Column Types** - text, number, currency, date, status, actions
✅ **Loading States** - Elegant skeleton UI
✅ **Empty States** - Professional "no data" messaging
✅ **Compact Mode** - Denser layout for space-constrained UIs
✅ **Custom Formatters** - Full control over value display
✅ **Performance** - Handles 100+ rows smoothly
✅ **Accessibility** - Full ARIA support, keyboard navigation
✅ **Responsive** - Horizontal scroll on mobile

---

## Basic Usage

```svelte
<script>
  import DataTable from '$lib/components/ui/DataTable.svelte';
  import type { Column } from '$lib/components/ui/DataTable.types';

  const columns: Column[] = [
    { key: 'id', label: 'ID', sortable: true, width: '80px' },
    { key: 'name', label: 'Name', sortable: true },
    { key: 'amount', label: 'Amount', type: 'currency', sortable: true, align: 'right' },
    { key: 'status', label: 'Status', type: 'status' }
  ];

  const data = [
    { id: 1, name: 'ACME Corp', amount: 15750.500, status: 'approved' },
    { id: 2, name: 'TechStart', amount: 8900.250, status: 'pending' }
  ];

  function handleRowClick(row) {
    console.log('Row clicked:', row);
  }
</script>

<DataTable
  {columns}
  {data}
  onRowClick={handleRowClick}
  maxHeight="600px"
/>
```

---

## Column Configuration

### Column Properties

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `key` | `string` | **required** | Key/path to access value (supports nested like `customer.name`) |
| `label` | `string` | **required** | Display label for header |
| `align` | `'left' \| 'center' \| 'right'` | `'left'` | Text alignment (auto `right` for numbers) |
| `width` | `string` | auto | Fixed width (e.g., `'120px'`, `'15%'`) |
| `sortable` | `boolean` | `false` | Enable sorting |
| `type` | `ColumnType` | `'text'` | Column type for formatting |
| `render` | `(row) => string` | - | Custom render function (returns HTML) |
| `format` | `(value) => string` | - | Custom format function |

### Column Types

```typescript
type ColumnType = 'text' | 'number' | 'currency' | 'date' | 'status' | 'actions';
```

**Examples:**

```typescript
// Text (default)
{ key: 'name', label: 'Name', type: 'text' }

// Number (right-aligned, thousands separator)
{ key: 'quantity', label: 'Qty', type: 'number', align: 'right' }

// Currency (BHD with 3 decimals)
{ key: 'amount', label: 'Amount', type: 'currency' }

// Date (formatted as "Jan 15, 2026")
{ key: 'createdAt', label: 'Date', type: 'date' }

// Status (colored badge)
{ key: 'status', label: 'Status', type: 'status' }

// Actions (custom buttons/icons)
{
  key: 'actions',
  label: 'Actions',
  type: 'actions',
  render: (row) => `<button onclick="edit(${row.id})">Edit</button>`
}
```

---

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `columns` | `Column[]` | **required** | Column definitions |
| `data` | `any[]` | **required** | Data array |
| `loading` | `boolean` | `false` | Show loading skeleton |
| `emptyMessage` | `string` | `'No data available'` | Empty state message |
| `onRowClick` | `(row) => void` | - | Row click handler |
| `selectedId` | `string \| number` | - | ID of selected row |
| `stickyHeader` | `boolean` | `true` | Sticky header on scroll |
| `compact` | `boolean` | `false` | Compact mode (36px rows) |
| `maxHeight` | `string` | `'600px'` | Max height before scrolling |
| `showBorder` | `boolean` | `true` | Show table border |
| `keyField` | `string` | `'id'` | Field name for row keys |

---

## Events

### `on:sort`

Dispatched when column header is clicked (if `sortable: true`).

```typescript
interface SortEvent {
  key: string;
  direction: 'asc' | 'desc';
}
```

**Example:**

```svelte
<DataTable
  {columns}
  {data}
  on:sort={(e) => console.log('Sort:', e.detail)}
/>
```

### `on:rowClick`

Dispatched when a row is clicked.

```typescript
interface RowClickEvent {
  row: any;
  index: number;
}
```

**Example:**

```svelte
<DataTable
  {columns}
  {data}
  on:rowClick={(e) => console.log('Row:', e.detail.row)}
/>
```

---

## Advanced Examples

### Custom Formatter

```typescript
const columns = [
  {
    key: 'amount',
    label: 'Amount (Thousands)',
    align: 'right',
    format: (value) => `BHD ${(value / 1000).toFixed(1)}K`
  }
];
```

### Custom Render (HTML)

```typescript
const columns = [
  {
    key: 'actions',
    label: 'Actions',
    type: 'actions',
    render: (row) => `
      <div class="action-buttons">
        <button onclick="viewRow(${row.id})">View</button>
        <button onclick="editRow(${row.id})">Edit</button>
      </div>
    `
  }
];
```

### Nested Data Access

```typescript
const columns = [
  { key: 'customer.name', label: 'Customer' },
  { key: 'customer.contact.email', label: 'Email' }
];

const data = [
  {
    customer: {
      name: 'ACME Corp',
      contact: { email: 'info@acme.com' }
    }
  }
];
```

### Row Selection

```svelte
<script>
  let selectedId = undefined;

  function handleRowClick(row) {
    selectedId = row.id;
  }
</script>

<DataTable
  {columns}
  {data}
  {selectedId}
  onRowClick={handleRowClick}
/>
```

### Loading State

```svelte
<script>
  let loading = false;

  async function loadData() {
    loading = true;
    data = await fetchInvoices();
    loading = false;
  }
</script>

<DataTable {columns} {data} {loading} />
```

---

## Status Badges

Status column types automatically render colored badges. Supported statuses:

| Status | Color |
|--------|-------|
| `active`, `open`, `approved` | Green |
| `pending`, `draft` | Orange |
| `closed`, `rejected`, `cancelled` | Red |
| `inactive` | Gray |

**Usage:**

```typescript
const columns = [
  { key: 'status', label: 'Status', type: 'status' }
];

const data = [
  { id: 1, status: 'approved' },  // Green badge
  { id: 2, status: 'pending' },   // Orange badge
  { id: 3, status: 'rejected' }   // Red badge
];
```

---

## Keyboard Navigation

| Key | Action |
|-----|--------|
| `↓` Arrow Down | Move to next row |
| `↑` Arrow Up | Move to previous row |
| `Enter` / `Space` | Select current row |
| `Tab` | Navigate to sortable headers |

**Notes:**
- Rows are only keyboard-navigable when `onRowClick` is provided
- Sortable headers can be triggered with `Enter` or `Space`

---

## Styling & Theming

The component uses design tokens from the enterprise design system:

```css
/* Automatic support for: */
--table-row-height: 42px;  /* Bloomberg density */
--table-text-size: 14px;
--brand-indigo: #2F2DFF;
--interactive-hover: rgba(47, 45, 255, 0.04);  /* 4% tint on hover */
```

**Compact mode:**
- Row height: 36px
- Reduced padding: 8px

**Design Philosophy:**
- ❌ **NO zebra striping** (per design system)
- ✅ Hover = 4% indigo tint
- ✅ Selected = 8% indigo tint
- ✅ Subtle borders, professional aesthetic

---

## Accessibility

### ARIA Support

- `role="table"` on table element
- `role="columnheader"` on headers
- `role="row"` on rows
- `role="cell"` on cells
- `aria-selected` for selected rows
- `aria-label` for table description

### Keyboard Support

- Full keyboard navigation (arrow keys)
- Focus indicators (2px indigo outline)
- Skip-to-content support

### Screen Reader Support

- Semantic table structure
- Proper scope attributes
- Clear labels and descriptions

### Motion Preferences

```css
@media (prefers-reduced-motion: reduce) {
  /* All animations disabled */
}
```

---

## Performance

**Optimizations:**

✅ Virtual scrolling not needed (42px rows = ~14 visible at 600px height)
✅ CSS transforms for smooth scrolling
✅ Efficient sorting (native Array.sort)
✅ Minimal re-renders (reactive statements)

**Tested with:**
- 100+ rows: Smooth
- 500+ rows: Recommended to add pagination
- 1000+ rows: Use server-side pagination

---

## Browser Support

| Browser | Version | Support |
|---------|---------|---------|
| Chrome | 90+ | ✅ Full |
| Firefox | 88+ | ✅ Full |
| Safari | 14+ | ✅ Full |
| Edge | 90+ | ✅ Full |

**Features requiring modern browsers:**
- `position: sticky` for sticky headers
- CSS custom properties
- Array methods (sort, reduce)

---

## Troubleshooting

### Sticky header not working

Ensure parent container doesn't have `overflow: hidden`:

```svelte
<div style="overflow: visible;">
  <DataTable ... />
</div>
```

### Sorting not working

Ensure columns have `sortable: true`:

```typescript
{ key: 'name', label: 'Name', sortable: true }
```

### Currency not formatting correctly

Values must be numbers, not strings:

```typescript
// ✅ Correct
{ amount: 15750.500 }

// ❌ Wrong
{ amount: "15750.500" }
```

### Custom render HTML not showing

Use `render` function, not `format`:

```typescript
// ✅ Correct
{ render: (row) => `<button>${row.name}</button>` }

// ❌ Wrong
{ format: (row) => `<button>${row.name}</button>` }
```

---

## Related Components

- **Button** - For action columns
- **WabiBadge** - Alternative to built-in status badges
- **WabiSkeleton** - Alternative loading state
- **WabiEmptyState** - Alternative empty state

---

## Design System Compliance

✅ Uses design tokens from `design-tokens.css`
✅ Bloomberg-style density (42px rows)
✅ Apple-level polish (150ms transitions)
✅ No zebra striping (per style guide)
✅ 4% indigo hover tint
✅ Uppercase headers, 12px, 600 weight
✅ Full accessibility compliance

---

## Contributing

When modifying this component:

1. **Test all column types** (text, number, currency, date, status, actions)
2. **Test keyboard navigation** (arrow keys, Enter, Space)
3. **Test loading state** (skeleton should match columns)
4. **Test empty state** (proper messaging)
5. **Test sorting** (all sortable columns)
6. **Test selection** (visual highlight)
7. **Verify accessibility** (ARIA, keyboard, screen readers)

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2026-01-22 | Initial production release |

---

**Om Lokah Samastah Sukhino Bhavantu**
*May all users benefit from clear, accessible data tables.*
