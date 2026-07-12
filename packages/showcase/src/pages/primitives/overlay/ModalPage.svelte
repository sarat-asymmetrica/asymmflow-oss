<script lang="ts">
  import { Modal, ConfirmDialog } from '@asymmflow/ui';

  // ── Demo state ───────────────────────────────────────────────────────────
  let smOpen = $state(false);
  let mdOpen = $state(false);
  let lgOpen = $state(false);
  let fullOpen = $state(false);
  let formOpen = $state(false);
  let confirmOpen = $state(false);
  let confirmResult = $state<string | null>(null);

  // Form demo state
  let formName = $state('');
  let formRef = $state('');

  function handleFormSubmit() {
    formOpen = false;
  }
</script>

<div class="sections">
  <!-- ── Intro ──────────────────────────────────────────────────────────── -->
  <section>
    <h2 class="af-section-title">Modal</h2>
    <p class="intro">
      The canonical layered dialog. Portals to document.body, traps focus (§2.6),
      locks body scroll, and enters on the R1 explore curve (fade + 16px rise).
      Scrim click and Escape both close. Zero raw hex or millisecond literals —
      all motion from tokens.
    </p>
  </section>

  <!-- ── Size variants ─────────────────────────────────────────────────── -->
  <section>
    <h2 class="af-section-title">Sizes</h2>
    <div class="row">
      <button class="trigger-btn" onclick={() => (smOpen = true)}>sm — 400px</button>
      <button class="trigger-btn" onclick={() => (mdOpen = true)}>md — 560px (default)</button>
      <button class="trigger-btn" onclick={() => (lgOpen = true)}>lg — 840px</button>
      <button class="trigger-btn" onclick={() => (fullOpen = true)}>full — near-viewport</button>
    </div>

    <!-- sm -->
    <Modal bind:open={smOpen} title="Small modal" size="sm">
      {#snippet children()}
        <p class="body-copy">
          A compact confirmation or notice. Ideal for single-question dialogs where
          a ConfirmDialog isn't the right tone.
        </p>
      {/snippet}
      {#snippet footer()}
        <button class="action-btn action-btn--secondary" onclick={() => (smOpen = false)}>
          Dismiss
        </button>
      {/snippet}
    </Modal>

    <!-- md -->
    <Modal bind:open={mdOpen} title="Default modal" size="md">
      {#snippet children()}
        <p class="body-copy">
          The everyday workhorse. 560px accommodates a form, a detail view,
          or a rich notice without overwhelming the viewport.
        </p>
        <p class="body-copy" style="margin-top: var(--af-space-3);">
          Body scrolls independently; the header and footer are sticky.
        </p>
      {/snippet}
      {#snippet footer()}
        <button class="action-btn action-btn--secondary" onclick={() => (mdOpen = false)}>
          Cancel
        </button>
        <button class="action-btn action-btn--primary" onclick={() => (mdOpen = false)}>
          Confirm
        </button>
      {/snippet}
    </Modal>

    <!-- lg -->
    <Modal bind:open={lgOpen} title="Large modal" size="lg">
      {#snippet children()}
        <p class="body-copy">
          840px — use for complex data entry, multi-step flows, or side-by-side
          comparison layouts. Prefer a Drawer when the content is a detail view.
        </p>
        <div class="placeholder-grid">
          {#each Array(6) as _}
            <div class="placeholder-card">
              <div class="af-label">Field label</div>
              <div class="placeholder-value af-numeric">BHD 0.000</div>
            </div>
          {/each}
        </div>
      {/snippet}
      {#snippet footer()}
        <button class="action-btn action-btn--secondary" onclick={() => (lgOpen = false)}>
          Cancel
        </button>
        <button class="action-btn action-btn--primary" onclick={() => (lgOpen = false)}>
          Save
        </button>
      {/snippet}
    </Modal>

    <!-- full -->
    <Modal bind:open={fullOpen} title="Full-viewport modal" size="full">
      {#snippet children()}
        <p class="body-copy">
          Near-full-viewport. Useful for document editors, report builders, or
          any flow that needs maximum real estate without a full navigation change.
        </p>
      {/snippet}
      {#snippet footer()}
        <button class="action-btn action-btn--secondary" onclick={() => (fullOpen = false)}>
          Close
        </button>
      {/snippet}
    </Modal>
  </section>

  <!-- ── Form example ───────────────────────────────────────────────────── -->
  <section>
    <h2 class="af-section-title">With form</h2>
    <p class="intro">
      A realistic data-entry modal. The footer's Cancel / Save buttons are wired;
      the form uses af-label + af-numeric typography classes from base.css.
    </p>
    <div class="row">
      <button class="trigger-btn" onclick={() => (formOpen = true)}>New purchase order</button>
    </div>

    <Modal bind:open={formOpen} title="New purchase order" size="md">
      {#snippet children()}
        <form
          id="po-form"
          class="form-grid"
          onsubmit={(e) => {
            e.preventDefault();
            handleFormSubmit();
          }}
        >
          <label class="form-field">
            <span class="af-label">Vendor name</span>
            <input
              class="form-input"
              type="text"
              bind:value={formName}
              placeholder="e.g. Al Jazeera Trading"
              autocomplete="off"
            />
          </label>
          <label class="form-field">
            <span class="af-label">Reference no.</span>
            <input
              class="form-input af-numeric"
              type="text"
              bind:value={formRef}
              placeholder="PO-2026-XXXX"
            />
          </label>
          <label class="form-field form-field--full">
            <span class="af-label">Notes</span>
            <textarea class="form-input form-textarea" rows="3" placeholder="Optional notes…"></textarea>
          </label>
        </form>
      {/snippet}
      {#snippet footer()}
        <button
          class="action-btn action-btn--secondary"
          type="button"
          onclick={() => (formOpen = false)}
        >
          Cancel
        </button>
        <button
          class="action-btn action-btn--primary"
          type="submit"
          form="po-form"
        >
          Create order
        </button>
      {/snippet}
    </Modal>
  </section>

  <!-- ── ConfirmDialog — destructive example ────────────────────────────── -->
  <section>
    <h2 class="af-section-title">ConfirmDialog — destructive</h2>
    <p class="intro">
      Composition over Modal. Autofocus lands on the Cancel (safe) action.
      The destructive confirm renders in --af-danger. onConfirm / onCancel
      are synchronous callbacks; the parent owns the open state.
    </p>
    {#if confirmResult}
      <p class="result-notice">
        Last action: <strong>{confirmResult}</strong>
      </p>
    {/if}
    <div class="row">
      <button class="trigger-btn trigger-btn--danger" onclick={() => (confirmOpen = true)}>
        Delete invoice
      </button>
    </div>

    <ConfirmDialog
      bind:open={confirmOpen}
      message="Delete invoice INV-2026-0042?"
      description="This action cannot be undone. The invoice will be permanently removed from the ledger."
      confirmLabel="Delete"
      destructive={true}
      onConfirm={() => {
        confirmOpen = false;
        confirmResult = 'Confirmed — invoice deleted';
      }}
      onCancel={() => {
        confirmResult = 'Cancelled';
      }}
    />
  </section>
</div>

<style>
  .sections {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-6);
  }

  .intro {
    color: var(--af-text-secondary);
    font-size: var(--af-text-md);
    max-width: 64ch;
    margin-top: var(--af-space-2);
    margin-bottom: var(--af-space-4);
  }

  .row {
    display: flex;
    flex-wrap: wrap;
    gap: var(--af-space-3);
    margin-bottom: var(--af-space-3);
  }

  /* Demo trigger buttons */
  .trigger-btn {
    display: inline-flex;
    align-items: center;
    min-height: var(--af-control-height);
    padding: 0 var(--af-space-4);
    background: var(--af-inverse-surface);
    color: var(--af-text-inverse);
    border: 1px solid var(--af-inverse-surface);
    border-radius: var(--af-radius-sm);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-semibold);
    cursor: pointer;
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .trigger-btn:hover {
    background: color-mix(in srgb, var(--af-inverse-surface) 88%, transparent);
    box-shadow: var(--af-shadow-lift);
  }

  .trigger-btn:active {
    transform: scale(0.985);
  }

  .trigger-btn--danger {
    background: var(--af-danger);
    border-color: var(--af-danger);
  }

  .trigger-btn--danger:hover {
    background: color-mix(in srgb, var(--af-danger) 88%, transparent);
  }

  /* Modal content helpers */
  .body-copy {
    font-size: var(--af-text-md);
    line-height: var(--af-leading-base);
    color: var(--af-text-secondary);
  }

  .placeholder-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: var(--af-space-3);
    margin-top: var(--af-space-4);
  }

  .placeholder-card {
    background: var(--af-surface-raised);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    padding: var(--af-space-3);
    display: flex;
    flex-direction: column;
    gap: var(--af-space-1);
  }

  .placeholder-value {
    font-size: var(--af-text-lg);
    font-weight: var(--af-weight-semibold);
    color: var(--af-text);
  }

  /* Form */
  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--af-space-4);
  }

  .form-field {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-1);
  }

  .form-field--full {
    grid-column: 1 / -1;
  }

  .form-input {
    height: var(--af-control-height);
    padding: 0 var(--af-space-3);
    background: var(--af-surface);
    border: 1px solid var(--af-border-strong);
    border-radius: var(--af-radius-sm);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    color: var(--af-text);
    transition:
      border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .form-input:focus {
    border-color: var(--af-accent);
    box-shadow: 0 0 0 3px var(--af-accent-tint-strong);
    outline: none;
  }

  .form-textarea {
    height: auto;
    padding: var(--af-space-2) var(--af-space-3);
    resize: vertical;
  }

  /* Action buttons inside modals / footers */
  .action-btn {
    display: inline-flex;
    align-items: center;
    min-height: var(--af-control-height);
    padding: 0 var(--af-space-4);
    border-radius: var(--af-radius-sm);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-semibold);
    cursor: pointer;
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .action-btn:active { transform: scale(0.985); }

  .action-btn--primary {
    background: var(--af-inverse-surface);
    color: var(--af-text-inverse);
    border: 1px solid var(--af-inverse-surface);
  }

  .action-btn--primary:hover {
    background: color-mix(in srgb, var(--af-inverse-surface) 88%, transparent);
    box-shadow: var(--af-shadow-lift);
  }

  .action-btn--secondary {
    background: var(--af-surface);
    color: var(--af-text);
    border: 1px solid var(--af-border-strong);
  }

  .action-btn--secondary:hover {
    background: var(--af-surface-raised);
    box-shadow: var(--af-shadow-sm);
  }

  .result-notice {
    display: inline-flex;
    padding: var(--af-space-2) var(--af-space-3);
    background: var(--af-surface-raised);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    font-size: var(--af-text-sm);
    color: var(--af-text-secondary);
    margin-bottom: var(--af-space-3);
  }
</style>
