# DataTable Quick Reference Card

**5-Minute Guide to Using the Enterprise DataTable Component**

---

## 🚀 Quick Start

```svelte
<script>
  import { DataTable } from '$lib/components/ui';
  import type { Column } from '$lib/components/ui';

  const columns: Column[] = [
    { key: 'id', label: 'ID', sortable: true },
    { key: 'name', label: 'Name', sortable: true },
    { key: 'amount', label: 'Amount', type: 'currency', sortable: true }
  ];

  const data = [
    { id: 1, name: 'ACME', amount: 15750.500 }
  ];
</script>

<DataTable {columns} {data} />
```

---

## 📋 Column Types Cheatsheet

```typescript
// Text (default)
{ key: 'name', label: 'Name' }

// Number (right-aligned, 1,234)
{ key: 'quantity', label: 'Qty', type: 'number' }

// Currency (15,750.500 BHD)
{ key: 'amount', label: 'Amount', type: 'currency' }

// Date (Jan 15, 2026)
{ key: 'created', label: 'Date', type: 'date' }

// Status (colored badge)
{ key: 'status', label: 'Status', type: 'status' }

// Actions (custom HTML)
{
  key: 'actions',
  label: 'Actions',
  type: 'actions',
  render: (row) => `<button>Edit</button>`
}
```

---

## 🎛️ Common Props

```svelte
<!-- Sortable table -->
<DataTable {columns} {data} />

<!-- With row selection -->
<DataTable
  {columns}
  {data}
  selectedId={selectedId}
  onRowClick={(row) => selectedId = row.id}
/>

<!-- With loading state -->
<DataTable {columns} {data} loading={isLoading} />

<!-- Compact mode (36px rows) -->
<DataTable {columns} {data} compact={true} />

<!-- Custom height -->
<DataTable {columns} {data} maxHeight="400px" />

<!-- Empty state message -->
<DataTable
  {columns}
  data={[]}
  emptyMessage="No invoices found"
/>
```

---

## 🎨 Status Badge Colors

| Status | Color |
|--------|-------|
| `active`, `open`, `approved` | 🟢 Green |
| `pending`, `draft` | 🟠 Orange |
| `closed`, `rejected`, `cancelled` | 🔴 Red |
| `inactive` | ⚪ Gray |

---

## 🔧 Custom Formatters

```typescript
// Custom format function
{
  key: 'amount',
  label: 'Amount (K)',
  format: (val) => `${(val / 1000).toFixed(1)}K`
}

// Custom render (HTML)
{
  key: 'actions',
  label: 'Actions',
  render: (row) => `
    <button onclick="edit(${row.id})">Edit</button>
    <button onclick="delete(${row.id})">Delete</button>
  `
}

// Nested data access
{
  key: 'customer.name',
  label: 'Customer'
}
```

---

## ⌨️ Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↓` | Next row |
| `↑` | Previous row |
| `Enter` | Select row |
| `Space` | Select row |
| `Tab` | Navigate headers |

---

## 📡 Events

```svelte
<DataTable
  {columns}
  {data}
  on:sort={(e) => console.log(e.detail)}
  on:rowClick={(e) => console.log(e.detail)}
/>
```

**Event Payloads:**

```typescript
// on:sort
{ key: 'name', direction: 'asc' | 'desc' }

// on:rowClick
{ row: any, index: number }
```

---

## 💡 Common Patterns

### Invoices Table

```typescript
const columns: Column[] = [
  { key: 'number', label: 'Invoice #', sortable: true },
  { key: 'customer', label: 'Customer', sortable: true },
  { key: 'date', label: 'Date', type: 'date', sortable: true },
  { key: 'amount', label: 'Amount', type: 'currency', sortable: true },
  { key: 'status', label: 'Status', type: 'status' }
];
```

### Customers Table

```typescript
const columns: Column[] = [
  { key: 'name', label: 'Name', sortable: true },
  { key: 'email', label: 'Email', sortable: true },
  { key: 'totalOrders', label: 'Orders', type: 'number', sortable: true },
  { key: 'totalSpent', label: 'Total', type: 'currency', sortable: true },
  { key: 'status', label: 'Status', type: 'status' }
];
```

### Products Table

```typescript
const columns: Column[] = [
  { key: 'sku', label: 'SKU', sortable: true },
  { key: 'name', label: 'Product', sortable: true },
  { key: 'stock', label: 'Stock', type: 'number', sortable: true },
  { key: 'price', label: 'Price', type: 'currency', sortable: true },
  {
    key: 'actions',
    label: 'Actions',
    type: 'actions',
    render: (row) => `
      <button onclick="editProduct(${row.id})">Edit</button>
    `
  }
];
```

---

## 🐛 Troubleshooting

### "Sticky header not working"
✅ Parent container needs `overflow: visible`

### "Sorting doesn't work"
✅ Add `sortable: true` to column definition

### "Currency shows as 15750.5 instead of 15,750.500"
✅ Ensure value is a number, not a string

### "Custom HTML not rendering"
✅ Use `render`, not `format`
✅ HTML is injected with `@html` directive

---

## 📚 Full Documentation

For complete API reference, examples, and advanced usage:

📖 See `DataTable.README.md`

---

## ✅ Quick Checklist

Before deploying a table:

- [ ] All columns have `key` and `label`
- [ ] Currency/number columns use correct `type`
- [ ] Sortable columns have `sortable: true`
- [ ] Row selection has `onRowClick` handler
- [ ] Empty state has custom `emptyMessage`
- [ ] Loading state uses `loading` prop
- [ ] Max height set appropriately
- [ ] Tested with 100+ rows
- [ ] Keyboard navigation works
- [ ] Accessibility verified

---

**Ready to ship!** 🚀

*Philosophy: Bloomberg density × Apple polish*
