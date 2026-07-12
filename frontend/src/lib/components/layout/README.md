# Layout Components - Enterprise Design System

**Version:** 1.0
**Philosophy:** Apple-level polish × Bloomberg-level data density

These layout components provide the structural foundation for building enterprise ERP/CRM screens with consistent spacing, navigation, and interaction patterns.

---

## Components Overview

| Component | Purpose | Use Case |
|-----------|---------|----------|
| `PageLayout` | Full page wrapper | Dashboard, Settings, Reports |
| `ModuleLayout` | Module with header + tabs | Sales Hub, Operations, Finance |
| `SplitView` | List + Detail layout | Customers, Products, Suppliers |
| `Modal` | Overlay dialogs | Forms, confirmations, details |
| `Sidebar` | Navigation sidebar | Main app navigation |

---

## 1. PageLayout

**Full page wrapper with title section and content area.**

### Usage

```svelte
<script>
  import { PageLayout, Button } from '$lib/components';
</script>

<PageLayout title="Dashboard" subtitle="Your business at a glance">
  <svelte:fragment slot="header-actions">
    <Button variant="primary">Create Report</Button>
  </svelte:fragment>

  <!-- Your page content here -->
  <div class="grid grid-4">
    <!-- KPI Cards, Charts, etc. -->
  </div>
</PageLayout>
```

### Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `title` | `string` | `''` | Page title |
| `subtitle` | `string` | `''` | Page subtitle/description |

### Slots

- `header-actions` - Action buttons in header (e.g., "Create", "Filter")
- `default` - Page content

### Design Tokens

- Header height: `var(--header-height)` = 56px
- Padding: `var(--page-padding)` = 16px
- Border: `1px solid var(--border)`

---

## 2. ModuleLayout

**Standard module layout with header, tabs, and content area. Perfect for hub screens.**

### Usage

```svelte
<script>
  import { ModuleLayout, Button } from '$lib/components';

  let activeTab = 'inbox';
  const tabs = [
    { id: 'inbox', label: 'Inbox', count: 12 },
    { id: 'opportunities', label: 'Opportunities', count: 8 },
    { id: 'offers', label: 'Offers', count: 5 },
  ];

  function handleTabChange(event) {
    activeTab = event.detail;
  }
</script>

<ModuleLayout
  title="Sales Hub"
  {tabs}
  {activeTab}
  on:tabChange={handleTabChange}
>
  <svelte:fragment slot="header-actions">
    <Button variant="secondary">Filter</Button>
    <Button variant="primary">New RFQ</Button>
  </svelte:fragment>

  <!-- Tab content -->
  {#if activeTab === 'inbox'}
    <InboxContent />
  {:else if activeTab === 'opportunities'}
    <OpportunitiesContent />
  {/if}
</ModuleLayout>
```

### Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `title` | `string` | Required | Module title |
| `tabs` | `Array<{id, label, count?}>` | `[]` | Tab configuration |
| `activeTab` | `string` | `''` | Currently active tab ID |

### Events

- `tabChange` - Emitted when tab changes (detail: tab ID)

### Slots

- `header-actions` - Action buttons in header
- `default` - Content area (changes with active tab)

### Design Tokens

- Header height: `var(--header-height)` = 56px
- Tab height: `var(--tab-height)` = 36px
- Padding: `var(--page-padding)` = 16px

### Accessibility

- Tab buttons have `aria-current="page"` when active
- Focus states visible with indigo outline
- Keyboard navigation supported

---

## 3. SplitView

**Two-column layout with list on left and detail on right. Commonly used for master-detail views.**

### Usage

```svelte
<script>
  import { SplitView, Card } from '$lib/components';

  let selectedCustomer = null;
  const customers = [
    { id: 1, name: 'Acme Corp' },
    { id: 2, name: 'TechStart Inc' },
  ];
</script>

<SplitView listWidth="400px" minListWidth="300px">
  <svelte:fragment slot="list">
    <!-- Customer list -->
    {#each customers as customer}
      <button
        class="list-item"
        on:click={() => selectedCustomer = customer}
      >
        {customer.name}
      </button>
    {/each}
  </svelte:fragment>

  <svelte:fragment slot="detail">
    {#if selectedCustomer}
      <Card title={selectedCustomer.name}>
        <!-- Customer details -->
      </Card>
    {/if}
  </svelte:fragment>
</SplitView>
```

### Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `listWidth` | `string` | `'400px'` | Width of list column |
| `minListWidth` | `string` | `'300px'` | Minimum width of list |
| `resizable` | `boolean` | `false` | Enable resize (future) |

### Slots

- `list` - Left side list/navigation
- `detail` - Right side detail view (shows empty state if not provided)

### Design Tokens

- Border: `1px solid var(--border)`
- Empty state text: `var(--text-muted)`

### Scrolling

Both list and detail areas are independently scrollable. Custom scrollbar styling matches design system.

---

## 4. Modal

**Overlay dialog with backdrop, keyboard support, and focus trapping.**

### Usage

```svelte
<script>
  import { Modal, Button } from '$lib/components';

  let showModal = false;
</script>

<button on:click={() => showModal = true}>Open Modal</button>

<Modal
  bind:open={showModal}
  title="Confirm Action"
  size="md"
  closable={true}
  on:close={() => console.log('Modal closed')}
>
  <p>Are you sure you want to proceed?</p>

  <svelte:fragment slot="footer">
    <Button variant="secondary" on:click={() => showModal = false}>
      Cancel
    </Button>
    <Button variant="primary" on:click={() => {
      // Handle confirmation
      showModal = false;
    }}>
      Confirm
    </Button>
  </svelte:fragment>
</Modal>
```

### Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `open` | `boolean` | `false` | Modal visibility (bind this!) |
| `title` | `string` | `''` | Modal title |
| `size` | `'sm' \| 'md' \| 'lg' \| 'full'` | `'md'` | Modal size |
| `closable` | `boolean` | `true` | Can close via backdrop/escape |

### Events

- `close` - Emitted when modal is closed

### Slots

- `header` - Custom header (overrides title)
- `default` - Modal content
- `footer` - Footer actions (typically buttons)

### Sizes

- `sm` - 400px width
- `md` - 600px width
- `lg` - 900px width
- `full` - calc(100vw - 32px) × calc(100vh - 32px)

### Accessibility Features

- **Focus trap** - Tab cycles within modal
- **Escape key** - Closes modal (if closable)
- **Focus restoration** - Returns focus to trigger element
- **ARIA attributes** - `role="dialog"`, `aria-modal="true"`
- **Body scroll lock** - Prevents background scrolling

### Design Tokens

- Backdrop: `rgba(0, 0, 0, 0.5)`
- Border radius: `var(--border-radius-lg)` = 12px
- Shadow: `var(--shadow-lg)`
- Transitions: `var(--transition-base)` = 200ms

---

## 5. Sidebar

**Collapsible navigation sidebar with nested items.**

### Usage

```svelte
<script>
  import { Sidebar } from '$lib/components/layout';
  import { createEventDispatcher } from 'svelte';

  let currentScreen = 'dashboard';
  let collapsed = false;

  const dispatch = createEventDispatcher();

  function handleNavigate(event) {
    currentScreen = event.detail.screen;
    // Handle navigation
  }
</script>

<Sidebar
  {currentScreen}
  {collapsed}
  on:navigate={handleNavigate}
/>
```

### Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `currentScreen` | `string` | `'dashboard'` | Active screen ID |
| `collapsed` | `boolean` | `false` | Collapsed state |

### Events

- `navigate` - Emitted when nav item clicked (detail: `{ screen }`)

### Built-in Navigation Items

The sidebar includes these navigation items by default:

1. **Dashboard** (icon: home)
2. **Sales Hub** (icon: briefcase)
   - RFQs
   - Offers
   - Orders
3. **Operations** (icon: truck)
   - Purchase Orders
   - Goods Receipt
   - Supplier Invoices
   - Delivery Notes
4. **Finance** (icon: dollar)
   - Invoices
   - Payments
   - Accounting
5. **Customers** (icon: users)
6. **Products** (icon: box)
7. **Settings** (icon: settings)

### States

- **Expanded** - 240px width, shows labels
- **Collapsed** - 60px width, icons only with tooltips

### Accessibility

- All nav items are `<button>` elements
- `aria-current="page"` on active item
- Tooltips on hover when collapsed
- Focus-visible outline

### Design Tokens

- Width expanded: 240px
- Width collapsed: 60px
- Active background: `var(--brand-indigo)`
- Hover background: `var(--brand-indigo-tint)`
- Transition: `var(--transition-base)` = 200ms

---

## Design System Integration

All layout components use design tokens from `design-tokens.css`:

### Spacing
```css
--page-padding: 16px
--card-padding: 12px
--section-spacing: 16px
--grid-gap: 12px
```

### Heights
```css
--header-height: 56px
--tab-height: 36px
--kpi-card-height: 130px
--sidebar-width: 220px
```

### Colors
```css
--surface: #FFFFFF (light) / #121526 (dark)
--border: #E6E8F0 (light) / #23264A (dark)
--brand-indigo: #2F2DFF
--text-primary: #0E1020 (light) / #F2F3FF (dark)
```

### Transitions
```css
--transition-fast: 150ms
--transition-base: 200ms
--easing-smooth: cubic-bezier(0.4, 0.0, 0.2, 1)
```

---

## Common Patterns

### Pattern 1: Dashboard

```svelte
<PageLayout title="Dashboard" subtitle="Business overview">
  <div class="grid grid-4">
    <KPICard label="Revenue" value="$2.4M" />
    <KPICard label="Orders" value="342" />
    <KPICard label="Customers" value="128" />
    <KPICard label="Pending" value="23" accent />
  </div>

  <div class="grid grid-2">
    <Card title="Revenue Chart">...</Card>
    <Card title="Top Products">...</Card>
  </div>
</PageLayout>
```

### Pattern 2: Module with Tabs

```svelte
<ModuleLayout title="Sales Hub" {tabs} {activeTab} on:tabChange={handleTabChange}>
  <Table data={filteredData} />
</ModuleLayout>
```

### Pattern 3: Master-Detail

```svelte
<SplitView>
  <svelte:fragment slot="list">
    <CustomerList bind:selected={selectedCustomer} />
  </svelte:fragment>
  <svelte:fragment slot="detail">
    {#if selectedCustomer}
      <Customer360 customer={selectedCustomer} />
    {/if}
  </svelte:fragment>
</SplitView>
```

### Pattern 4: Confirmation Modal

```svelte
<Modal bind:open={showConfirm} title="Delete Customer?" size="sm">
  <p>This action cannot be undone.</p>
  <svelte:fragment slot="footer">
    <Button variant="secondary" on:click={() => showConfirm = false}>Cancel</Button>
    <Button variant="primary" on:click={handleDelete}>Delete</Button>
  </svelte:fragment>
</Modal>
```

---

## Testing

See `LayoutExample.svelte` for a comprehensive demonstration of all layout components.

Run the example:
```bash
# Add route in App.svelte to render LayoutExample
```

---

## Quality Checklist

Before using layout components in production:

- [ ] Design tokens imported (`design-tokens.css`)
- [ ] Accessibility tested (keyboard nav, screen reader)
- [ ] Dark mode verified
- [ ] Responsive behavior checked
- [ ] Focus states visible
- [ ] Transitions smooth (60fps)
- [ ] No console errors
- [ ] TypeScript types correct

---

## File Structure

```
frontend/src/lib/components/layout/
├── PageLayout.svelte       # Full page wrapper
├── ModuleLayout.svelte     # Module with header + tabs
├── SplitView.svelte        # List + Detail layout
├── Modal.svelte            # Overlay dialog
├── Sidebar.svelte          # Navigation sidebar
├── index.ts                # Export barrel
├── LayoutExample.svelte    # Comprehensive example
└── README.md               # This file
```

---

**Om Lokah Samastah Sukhino Bhavantu**
*May all developers benefit from consistent, accessible layouts.*
