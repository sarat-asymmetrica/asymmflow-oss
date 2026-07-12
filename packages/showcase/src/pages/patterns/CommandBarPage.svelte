<script lang="ts">
  /**
   * CommandBarPage — showcase for CommandBar (Ctrl+K palette).
   *
   * 12 realistic ERP commands spanning 4 groups.
   * Shows the Ctrl+K shortcut hint prominently.
   * Demonstrates recent-command memory (run 2–3 commands, reopen).
   */

  import { CommandBar } from '@asymmflow/patterns';
  import type { Command } from '@asymmflow/patterns';

  // ─── Commands — 12 realistic ERP operations ──────────────────────────────────

  const commands: Command[] = [
    // Invoicing
    {
      id: 'new-invoice',
      label: 'New Invoice',
      group: 'Invoicing',
      hint: 'Create a new sales invoice',
      action: () => { lastAction = 'New Invoice'; },
    },
    {
      id: 'find-invoice',
      label: 'Find Invoice',
      group: 'Invoicing',
      hint: 'Search by number or customer',
      action: () => { lastAction = 'Find Invoice'; },
    },
    {
      id: 'bank-reconcile',
      label: 'Bank Reconciliation',
      group: 'Invoicing',
      hint: 'Match payments to bank statements',
      action: () => { lastAction = 'Bank Reconciliation'; },
    },
    {
      id: 'send-reminder',
      label: 'Send Payment Reminder',
      group: 'Invoicing',
      hint: 'Bulk-send overdue notices',
      action: () => { lastAction = 'Send Payment Reminder'; },
    },

    // Customers
    {
      id: 'find-customer',
      label: 'Find Customer',
      group: 'Customers',
      hint: 'Search by name, CR, or type',
      action: () => { lastAction = 'Find Customer'; },
    },
    {
      id: 'new-customer',
      label: 'New Customer',
      group: 'Customers',
      hint: 'Register a new Gulf trading counterparty',
      action: () => { lastAction = 'New Customer'; },
    },
    {
      id: 'grade-review',
      label: 'Payment Grade Review',
      group: 'Customers',
      hint: 'Update A/B/C/D risk grades',
      action: () => { lastAction = 'Payment Grade Review'; },
    },

    // Reports
    {
      id: 'ar-aging',
      label: 'AR Aging Report',
      group: 'Reports',
      hint: '30 / 60 / 90-day buckets',
      action: () => { lastAction = 'AR Aging Report'; },
    },
    {
      id: 'revenue-export',
      label: 'Revenue Export (BHD)',
      group: 'Reports',
      hint: 'CSV — by customer, by month',
      action: () => { lastAction = 'Revenue Export'; },
    },
    {
      id: 'statement',
      label: 'Customer Statement',
      group: 'Reports',
      hint: 'Generate and email account statement',
      action: () => { lastAction = 'Customer Statement'; },
    },

    // System
    {
      id: 'settings',
      label: 'System Settings',
      group: 'System',
      hint: 'Users, currency, fiscal year',
      action: () => { lastAction = 'System Settings'; },
    },
    {
      id: 'import-csv',
      label: 'Import from CSV',
      group: 'System',
      hint: 'Bulk-upload customers or invoices',
      action: () => { lastAction = 'Import CSV'; },
    },
  ];

  let open = $state(false);
  let lastAction = $state<string | null>(null);
</script>

<div class="sections">

  <!-- ===== SECTION 1: Trigger + hint ===== -->
  <section>
    <h2 class="af-section-title">Command Bar</h2>
    <p class="intro">
      The Ctrl+K palette — frosted glass, instant, grouped by ERP domain.
      Arrow keys navigate; Enter runs; Esc closes. Recent commands bubble
      to the top on the next open (marked with a green dot).
    </p>

    <div class="trigger-row">
      <button class="open-btn" onclick={() => (open = true)}>
        <!-- Terminal-style icon, no emoji -->
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
          <path d="M2.5 4.5L5.5 7L2.5 9.5M7 9.5h4.5" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
        Open Command Bar
      </button>

      <!-- Ctrl+K shortcut badge -->
      <div class="shortcut-hint" aria-label="Keyboard shortcut: Control + K">
        <kbd>Ctrl</kbd>
        <span class="shortcut-plus" aria-hidden="true">+</span>
        <kbd>K</kbd>
        <span class="shortcut-label">opens anywhere</span>
      </div>
    </div>

    {#if lastAction}
      <p class="feedback af-meta">
        Last command run: <strong>{lastAction}</strong>
      </p>
    {/if}
  </section>

  <!-- ===== SECTION 2: Prop reference ===== -->
  <section>
    <h2 class="af-section-title">Props</h2>
    <div class="prop-table-wrap">
      <table class="prop-table">
        <thead>
          <tr>
            <th class="af-label">Prop</th>
            <th class="af-label">Type</th>
            <th class="af-label">Default</th>
            <th class="af-label">Description</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td><code>open</code></td>
            <td><code>boolean</code></td>
            <td><code>false</code></td>
            <td>Bindable open state.</td>
          </tr>
          <tr>
            <td><code>commands</code></td>
            <td><code>Command[]</code></td>
            <td><code>[]</code></td>
            <td>All available commands with id, label, group, hint, icon, action.</td>
          </tr>
          <tr>
            <td><code>registerGlobalShortcut</code></td>
            <td><code>boolean</code></td>
            <td><code>true</code></td>
            <td>Wire Ctrl+K / Cmd+K globally.</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>

</div>

<!-- CommandBar — lives outside sections so portal can hoist it correctly -->
<CommandBar bind:open {commands} />

<style>
  .sections {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-6);
  }

  .intro {
    color: var(--af-text-secondary);
    font-size: var(--af-text-md);
    max-width: 72ch;
    margin-top: var(--af-space-2);
    margin-bottom: var(--af-space-4);
  }

  /* ── Trigger row ─────────────────────────────────────────────────────────── */
  .trigger-row {
    display: flex;
    align-items: center;
    gap: var(--af-space-4);
    flex-wrap: wrap;
  }

  .open-btn {
    display: inline-flex;
    align-items: center;
    gap: var(--af-space-2);
    padding: 0 var(--af-space-4);
    height: var(--af-control-height);
    background: var(--af-inverse-surface);
    color: var(--af-text-inverse);
    border: none;
    border-radius: var(--af-radius-sm);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-semibold);
    cursor: pointer;
    transition:
      opacity var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .open-btn:hover {
    opacity: 0.9;
    box-shadow: var(--af-shadow-lift);
  }

  .open-btn:focus-visible {
    outline: 2px solid var(--af-focus-ring);
    outline-offset: 2px;
  }

  .shortcut-hint {
    display: inline-flex;
    align-items: center;
    gap: var(--af-space-1);
  }

  .shortcut-plus {
    color: var(--af-text-muted);
    font-size: var(--af-text-xs);
  }

  .shortcut-label {
    font-size: var(--af-text-xs);
    color: var(--af-text-muted);
    margin-left: var(--af-space-1);
  }

  kbd {
    font-family: var(--af-font-body);
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-semibold);
    color: var(--af-text-secondary);
    background: var(--af-surface);
    border: 1px solid var(--af-border-strong);
    border-radius: 4px;
    padding: 2px var(--af-space-2);
    line-height: 1.4;
    box-shadow: 0 1px 0 var(--af-border-strong);
  }

  /* ── Feedback ────────────────────────────────────────────────────────────── */
  .feedback {
    margin-top: var(--af-space-3);
    padding: var(--af-space-2) var(--af-space-3);
    background: var(--af-surface-raised);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    font-size: var(--af-text-xs);
    color: var(--af-text-muted);
  }

  .feedback strong {
    color: var(--af-text);
    font-weight: var(--af-weight-semibold);
  }

  /* ── Prop table ──────────────────────────────────────────────────────────── */
  .prop-table-wrap {
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    overflow: hidden;
  }

  .prop-table {
    width: 100%;
    border-collapse: collapse;
    font-size: var(--af-text-sm);
    font-family: var(--af-font-body);
  }

  .prop-table th {
    padding: var(--af-space-2) var(--af-space-3);
    font-family: var(--af-font-body);
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-semibold);
    text-transform: uppercase;
    letter-spacing: var(--af-label-tracking);
    color: var(--af-text-secondary);
    text-align: left;
    border-bottom: 1px solid var(--af-border);
    background: var(--af-glass-bg);
    white-space: nowrap;
  }

  .prop-table td {
    padding: var(--af-space-2) var(--af-space-3);
    border-bottom: 1px solid var(--af-border);
    color: var(--af-text);
    font-size: var(--af-text-sm);
    vertical-align: top;
  }

  .prop-table tr:last-child td {
    border-bottom: none;
  }

  code {
    font-family: 'Fira Code', 'Cascadia Code', 'Courier New', monospace;
    font-size: 0.88em;
    background: var(--af-surface-sunken);
    padding: 1px 5px;
    border-radius: 3px;
    color: var(--af-accent-pressed);
  }

  @media (prefers-reduced-motion: reduce) {
    .open-btn {
      transition: none;
    }
  }
</style>
