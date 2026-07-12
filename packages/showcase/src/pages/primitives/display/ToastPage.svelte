<script lang="ts">
  import { toast, ToastContainer } from '@asymmflow/ui';
</script>

<!-- ToastContainer must be mounted once at the app shell level.
     In a real app it lives in App.svelte. Here it's mounted for demo isolation. -->
<ToastContainer />

<div class="sections">
  <section>
    <h2 class="af-section-title">Toast system</h2>
    <p class="intro">
      Fixed stack, bottom-right, <code>z-index: --af-z-toast</code>.
      Monochrome-first: the toast body is surface + border. Only the 3px
      inline-start accent stripe carries color — spent sparingly per §4c.
      Entrance R1 explore, exit R3 stabilize. Auto-dismiss with pause-on-hover.
      <code>warning</code> and <code>danger</code> use <code>role="alert"</code>
      (assertive); <code>success</code> and <code>info</code> use <code>role="status"</code>
      (polite).
    </p>
  </section>

  <!-- Fire buttons -->
  <section>
    <div class="af-label section-label">Trigger toasts</div>
    <div class="button-grid">
      <button
        class="trigger-btn trigger-btn--success"
        onclick={() => toast.success('Invoice INV-2026-0411 reconciled successfully.')}
      >
        Success
      </button>
      <button
        class="trigger-btn trigger-btn--info"
        onclick={() => toast.info('Sync in progress — 42 records updating.')}
      >
        Info
      </button>
      <button
        class="trigger-btn trigger-btn--warning"
        onclick={() => toast.warning('Payment BHD 12,450 is overdue by 3 days.')}
      >
        Warning
      </button>
      <button
        class="trigger-btn trigger-btn--danger"
        onclick={() => toast.danger('Failed to submit: authentication error. Please sign in again.')}
      >
        Danger
      </button>
    </div>
  </section>

  <!-- Rapid-fire -->
  <section>
    <div class="af-label section-label">Rapid-fire — stacked queue</div>
    <button
      class="trigger-btn"
      onclick={() => {
        toast.success('Vendor Gulf Equipment — approved');
        setTimeout(() => toast.info('3 line items updated'), 80);
        setTimeout(() => toast.warning('Approval deadline in 2 hours'), 160);
      }}
    >
      Fire 3 at once
    </button>
  </section>

  <!-- Persistent -->
  <section>
    <div class="af-label section-label">Persistent (duration=0) — dismiss manually</div>
    <div class="button-row">
      <button
        class="trigger-btn"
        onclick={() => toast.info('This toast will not auto-dismiss. Click × to close.', { duration: 0 })}
      >
        Persistent info
      </button>
      <button
        class="trigger-btn"
        onclick={() => toast.clear()}
      >
        Clear all
      </button>
    </div>
  </section>

  <!-- Design notes -->
  <section>
    <div class="demo-card notes-card">
      <span class="af-label">Design notes</span>
      <ul class="notes-list">
        <li>Hover a toast to pause its auto-dismiss timer.</li>
        <li>Mouse away → 1.2s grace then dismiss.</li>
        <li>The × close button is always present for manual control.</li>
        <li>Toasts stack newest on top (column-reverse ordering).</li>
        <li>Color accent is the 3px inline-start stripe only — no colored backgrounds.</li>
      </ul>
    </div>
  </section>
</div>

<style>
  .sections {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-5);
  }

  .intro {
    color: var(--af-text-secondary);
    font-size: var(--af-text-md);
    max-width: 64ch;
    margin-top: var(--af-space-2);
    margin-bottom: var(--af-space-4);
  }

  .section-label {
    margin-block-end: var(--af-space-3);
  }

  .button-grid {
    display: flex;
    flex-wrap: wrap;
    gap: var(--af-space-2);
  }

  .button-row {
    display: flex;
    gap: var(--af-space-2);
    flex-wrap: wrap;
  }

  /* Base trigger button */
  .trigger-btn {
    border: 1px solid var(--af-border-strong);
    background: var(--af-surface);
    color: var(--af-text);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-medium);
    padding: var(--af-space-2) var(--af-space-4);
    border-radius: var(--af-radius-sm);
    cursor: pointer;
    min-height: 44px;
    border-inline-start-width: 3px;
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .trigger-btn:hover {
    background: var(--af-surface-raised);
    box-shadow: var(--af-shadow-sm);
  }

  .trigger-btn--success { border-inline-start-color: var(--af-success); }
  .trigger-btn--info    { border-inline-start-color: var(--af-info); }
  .trigger-btn--warning { border-inline-start-color: var(--af-warning); }
  .trigger-btn--danger  { border-inline-start-color: var(--af-danger); }

  .demo-card {
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    padding: var(--af-card-padding);
  }

  .notes-card {
    background: var(--af-surface-raised);
  }

  .notes-list {
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
    margin-block-start: var(--af-space-3);
  }

  .notes-list li {
    font-size: var(--af-text-sm);
    color: var(--af-text-secondary);
    line-height: var(--af-leading-base);
    padding-inline-start: var(--af-space-3);
    border-inline-start: 2px solid var(--af-border);
  }
</style>
