<script lang="ts">
  import { Button } from '@asymmflow/ui';

  let loadingPrimary = $state(false);
  let loadingSecondary = $state(false);

  function simulateLoad(set: (v: boolean) => void) {
    set(true);
    setTimeout(() => set(false), 2200);
  }
</script>

<div class="sections">

  <section>
    <h2 class="af-section-title">Button</h2>
    <p class="intro">
      Four variants, two sizes. Primary uses <code>--af-inverse-surface</code> — executive
      monochrome, not the accent. Accent is a scarce resource (§4c). Danger is a ghost
      variant: dangerous actions should feel weightless until committed.
    </p>
  </section>

  <!-- Variants -->
  <section>
    <h2 class="af-section-title">Variants</h2>
    <div class="card row-demo">
      <div class="demo-cell">
        <span class="af-label">primary</span>
        <Button variant="primary">Save changes</Button>
      </div>
      <div class="demo-cell">
        <span class="af-label">secondary</span>
        <Button variant="secondary">Export</Button>
      </div>
      <div class="demo-cell">
        <span class="af-label">ghost</span>
        <Button variant="ghost">Cancel</Button>
      </div>
      <div class="demo-cell">
        <span class="af-label">danger</span>
        <Button variant="danger">Delete entry</Button>
      </div>
    </div>
  </section>

  <!-- Sizes -->
  <section>
    <h2 class="af-section-title">Sizes</h2>
    <div class="card row-demo">
      <div class="demo-cell">
        <span class="af-label">md (default)</span>
        <Button size="md">Reconcile batch</Button>
      </div>
      <div class="demo-cell">
        <span class="af-label">sm</span>
        <Button size="sm">Reconcile batch</Button>
      </div>
    </div>
  </section>

  <!-- With icon -->
  <section>
    <h2 class="af-section-title">Leading icon</h2>
    <p class="intro">Pass an icon via the <code>icon</code> snippet. The button preserves gap via <code>--af-space-2</code>.</p>
    <div class="card row-demo">
      <div class="demo-cell">
        <span class="af-label">primary + icon</span>
        <Button variant="primary">
          {#snippet icon()}
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
              <path d="M7 1v12M1 7h12" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/>
            </svg>
          {/snippet}
          New invoice
        </Button>
      </div>
      <div class="demo-cell">
        <span class="af-label">secondary + icon</span>
        <Button variant="secondary" size="sm">
          {#snippet icon()}
            <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
              <path d="M1 6l4 4 6-8" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          {/snippet}
          Approve
        </Button>
      </div>
    </div>
  </section>

  <!-- Loading -->
  <section>
    <h2 class="af-section-title">Loading state</h2>
    <p class="intro">
      The spinner replaces leading content while preserving button width.
      <code>aria-busy</code> is set automatically.
    </p>
    <div class="card row-demo">
      <div class="demo-cell">
        <span class="af-label">primary loading</span>
        <Button
          variant="primary"
          loading={loadingPrimary}
          onclick={() => simulateLoad((v) => (loadingPrimary = v))}
        >
          Save changes
        </Button>
      </div>
      <div class="demo-cell">
        <span class="af-label">secondary loading</span>
        <Button
          variant="secondary"
          loading={loadingSecondary}
          onclick={() => simulateLoad((v) => (loadingSecondary = v))}
        >
          Export
        </Button>
      </div>
    </div>
  </section>

  <!-- Disabled -->
  <section>
    <h2 class="af-section-title">Disabled state</h2>
    <div class="card row-demo">
      <div class="demo-cell">
        <span class="af-label">primary disabled</span>
        <Button variant="primary" disabled>Save changes</Button>
      </div>
      <div class="demo-cell">
        <span class="af-label">secondary disabled</span>
        <Button variant="secondary" disabled>Export</Button>
      </div>
      <div class="demo-cell">
        <span class="af-label">ghost disabled</span>
        <Button variant="ghost" disabled>Cancel</Button>
      </div>
      <div class="demo-cell">
        <span class="af-label">danger disabled</span>
        <Button variant="danger" disabled>Delete</Button>
      </div>
    </div>
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

  .card {
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    padding: var(--af-card-padding);
  }

  .row-demo {
    display: flex;
    flex-wrap: wrap;
    gap: var(--af-space-5);
    align-items: flex-end;
  }

  .demo-cell {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
  }
</style>
