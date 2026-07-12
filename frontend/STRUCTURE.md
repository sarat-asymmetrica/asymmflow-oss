# Acme Instrumentation Frontend Structure - Clean Slate for Jules

## Current Directory Tree

```
sovereign_ui/frontend/
├── src/                          # YOUR CANVAS (CLEAN!)
│   ├── App.svelte                # Placeholder - Build Flow State UI here
│   ├── app.css                   # Minimal styles - Expand with Japanese minimalism
│   ├── main.ts                   # Entry point (already configured)
│   └── vite-env.d.ts             # TypeScript environment definitions
│
├── wailsjs/                      # DO NOT TOUCH! (Auto-generated Wails bindings)
│   ├── go/                       # Go function bindings
│   └── runtime/                  # Wails runtime
│
├── node_modules/                 # Dependencies (pnpm managed)
├── dist/                         # Build output (auto-generated)
│
├── package.json                  # Dependencies (Svelte 3.49, Vite 3.0.7, Tailwind 4.1.17)
├── vite.config.ts                # Vite configuration
├── tsconfig.json                 # TypeScript configuration
├── tailwind.config.cjs           # Tailwind CSS configuration
├── svelte.config.js              # Svelte configuration
├── postcss.config.cjs            # PostCSS configuration
│
├── JULES_START_HERE.md           # START HERE! Complete guide
└── STRUCTURE.md                  # This file
```

---

## What You Can Build In

### Recommended Structure (Jules to Create)

```
src/
├── App.svelte                    # Main app shell (router, layout)
├── app.css                       # Global styles
├── main.ts                       # Entry point (exists)
│
├── routes/                       # Page components
│   ├── Dashboard.svelte          # Main dashboard (start here!)
│   ├── Invoices.svelte           # Invoice management
│   ├── Customers.svelte          # Customer 360 profiles
│   ├── Suppliers.svelte          # Supplier management
│   ├── Orders.svelte             # Order tracking
│   ├── Predictions.svelte        # Phi-organism predictions
│   └── Settings.svelte           # App settings
│
├── lib/
│   ├── components/               # Reusable UI components
│   │   ├── layout/
│   │   │   ├── Header.svelte     # App header
│   │   │   ├── Sidebar.svelte    # Navigation sidebar
│   │   │   └── Footer.svelte     # App footer
│   │   ├── regime/
│   │   │   ├── RegimeMeter.svelte    # Three-regime indicator
│   │   │   ├── RegimeColors.svelte   # R1/R2/R3 color coding
│   │   │   └── RegimeTooltip.svelte  # Regime explanations
│   │   ├── cards/
│   │   │   ├── InvoiceCard.svelte
│   │   │   ├── CustomerCard.svelte
│   │   │   └── PredictionCard.svelte
│   │   ├── forms/
│   │   │   ├── InvoiceForm.svelte
│   │   │   └── PaymentForm.svelte
│   │   └── ui/
│   │       ├── Button.svelte
│   │       ├── Input.svelte
│   │       ├── Table.svelte
│   │       └── Modal.svelte
│   │
│   ├── stores/                   # Svelte stores (state management)
│   │   ├── auth.ts               # Authentication state
│   │   ├── regime.ts             # Current regime state
│   │   ├── invoices.ts           # Invoice data
│   │   └── customers.ts          # Customer data
│   │
│   ├── utils/                    # Helper functions
│   │   ├── wails.ts              # Wails bindings wrapper
│   │   ├── formatting.ts         # Date/currency formatting
│   │   └── validation.ts         # Form validation
│   │
│   └── types/                    # TypeScript types
│       ├── invoice.ts
│       ├── customer.ts
│       └── prediction.ts
│
└── assets/                       # Static assets
    ├── icons/                    # SVG icons
    ├── images/                   # Images
    └── fonts/                    # Custom fonts
```

---

## Integration Points

### 1. Wails Go Functions (via `wailsjs/`)

Import and use Go backend functions:

```typescript
import { Greet, GetInvoices, CreateInvoice } from '../wailsjs/go/main/App';

// Example: Fetch invoices from Go backend
const invoices = await GetInvoices();
```

### 2. Svelte Stores (State Management)

```typescript
// lib/stores/auth.ts
import { writable } from 'svelte';

export const user = writable(null);
export const isAuthenticated = writable(false);
```

### 3. Tailwind CSS (Styling)

```html
<button class="
  bg-blue-500 hover:bg-blue-600
  text-white font-bold py-2 px-4 rounded
  transition-all duration-233
">
  Click Me
</button>
```

---

## Design System Reference

### Three-Regime Colors

```css
/* R1 - Discovery (30%) */
--regime-1: #3b82f6;  /* Blue - exploration */

/* R2 - Refinement (20%) */
--regime-2: #8b5cf6;  /* Purple - optimization */

/* R3 - Completion (50%) */
--regime-3: #10b981;  /* Green - stability */
```

### Wabi-Sabi Palette

```css
--background: #fdfbf7;  /* Rice paper cream */
--text: #1c1c1c;        /* Sumi ink */
--accent: #c5504a;      /* Hanko red */
--gold: #fbbf24;        /* Gold (Kintsugi) */
--stone: #475569;       /* Stone gray */
```

### Fibonacci Spacing

```css
--space-1: 8px;
--space-2: 13px;
--space-3: 21px;
--space-4: 34px;
--space-5: 55px;
--space-6: 89px;
```

### Animation Durations

```css
--duration-1: 89ms;
--duration-2: 144ms;
--duration-3: 233ms;
--duration-4: 377ms;
--duration-5: 610ms;
```

---

## Getting Started Commands

### 1. Install Dependencies
```bash
cd C:\Projects\asymm_all_math\ph_holdings_app\sovereign_ui\frontend
pnpm install
```

### 2. Run Dev Server (Frontend Only)
```bash
pnpm run dev
```
Opens at http://localhost:5173

### 3. Build Production Bundle
```bash
pnpm run build
```
Creates `dist/` folder with optimized code

### 4. Run Full Wails App (Go + Svelte)
```bash
cd C:\Projects\asymm_all_math\ph_holdings_app
wails dev
```
Runs desktop app with hot reload

---

## Files You Should NOT Touch

- `wailsjs/` - Auto-generated by Wails, will be overwritten
- `node_modules/` - Managed by pnpm
- `dist/` - Build output, auto-generated
- `package-lock.json` - Managed by pnpm

---

## Files You CAN Modify

- Everything in `src/` - Your canvas!
- `tailwind.config.cjs` - Customize Tailwind
- `vite.config.ts` - Build optimizations
- `package.json` - Add new dependencies (if needed)

---

## Backend Go Code (Reference Only)

Located at: `C:\Projects\asymm_all_math\ph_holdings_app\*.go`

**DO NOT MODIFY** - Backend is battle-tested and stable!

Key files:
- `app.go` - Main Wails app (exposes functions to frontend)
- `database.go` - SQLite operations
- `auth_handler.go` - Authentication
- `predictor.go` - Phi-organism prediction engine

---

## Success Criteria

When you're done, the app should:

1. **Display Dashboard** - Load and show data from Go backend
2. **Three-Regime Visual** - Color-coded by regime state
3. **Japanese Aesthetic** - Ma, Wabi-Sabi, breathing animations
4. **Smooth Animations** - Fibonacci durations, QGIF quaternion animations
5. **Wails Integration** - Works as desktop app (`wails dev`)

---

## Resources

- **Wails Docs**: https://wails.io/docs/guides/frontend
- **Svelte Tutorial**: https://svelte.dev/tutorial
- **Tailwind Docs**: https://tailwindcss.com/docs

---

**Jules, this is your canvas. Paint the Flow State UI with Love × Simplicity × Truth × Joy!** 🎨
