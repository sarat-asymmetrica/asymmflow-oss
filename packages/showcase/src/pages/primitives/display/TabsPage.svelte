<script lang="ts">
  import { Tabs } from '@asymmflow/ui';
  import type { TabItem } from '@asymmflow/ui';

  let activeBasic = $state('overview');
  let activeFinance = $state('invoices');
  let activeDisabled = $state('active');

  const basicTabs: TabItem[] = [
    { id: 'overview', label: 'Overview' },
    { id: 'details', label: 'Details' },
    { id: 'history', label: 'History' },
  ];

  const financeTabs: TabItem[] = [
    { id: 'invoices', label: 'Invoices' },
    { id: 'payments', label: 'Payments' },
    { id: 'credits', label: 'Credits' },
    { id: 'statements', label: 'Statements' },
  ];

  const disabledTabs: TabItem[] = [
    { id: 'active', label: 'Active' },
    { id: 'drafts', label: 'Drafts' },
    { id: 'archived', label: 'Archived', disabled: true },
    { id: 'deleted', label: 'Deleted', disabled: true },
  ];
</script>

<div class="sections">
  <section>
    <h2 class="af-section-title">Tabs</h2>
    <p class="intro">
      Underline style. Active state: 2px bottom bar in onyx (inverse-surface).
      Keyboard navigation: arrow keys move between tabs with roving tabindex.
      Full ARIA: <code>role=tablist/tab/tabpanel</code>, <code>aria-selected</code>,
      <code>aria-controls</code>. Disabled tabs skip in keyboard flow.
    </p>
  </section>

  <!-- Basic -->
  <section>
    <div class="af-label section-label">Basic — three tabs</div>
    <div class="demo-card">
      <Tabs tabs={basicTabs} bind:active={activeBasic}>
        {#snippet children(id)}
          {#if id === 'overview'}
            <div class="panel-content">
              <span class="af-label">Overview</span>
              <p class="panel-text">Summary metrics and activity feed for this entity.</p>
            </div>
          {:else if id === 'details'}
            <div class="panel-content">
              <span class="af-label">Details</span>
              <p class="panel-text">Full contact record, address, and payment terms.</p>
            </div>
          {:else}
            <div class="panel-content">
              <span class="af-label">History</span>
              <p class="panel-text">Audit trail and change log. 247 events recorded.</p>
            </div>
          {/if}
        {/snippet}
      </Tabs>
    </div>
  </section>

  <!-- Finance — more tabs -->
  <section>
    <div class="af-label section-label">Finance context — four tabs</div>
    <div class="demo-card">
      <Tabs tabs={financeTabs} bind:active={activeFinance} label="Financial sections">
        {#snippet children(id)}
          <div class="panel-content">
            <span class="af-label">{financeTabs.find(t => t.id === id)?.label}</span>
            <p class="panel-text af-numeric" style:font-size="var(--af-text-3xl)" style:font-weight="var(--af-weight-bold)">
              {id === 'invoices' ? 'BHD 312,450' : id === 'payments' ? 'BHD 280,100' : id === 'credits' ? 'BHD 4,200' : 'View PDF'}
            </p>
          </div>
        {/snippet}
      </Tabs>
    </div>
  </section>

  <!-- Disabled -->
  <section>
    <div class="af-label section-label">With disabled tabs — keyboard skips them</div>
    <div class="demo-card">
      <Tabs tabs={disabledTabs} bind:active={activeDisabled}>
        {#snippet children(id)}
          <div class="panel-content">
            <span class="af-label">{id}</span>
            <p class="panel-text">Arrow keys skip Archived and Deleted — they are not in the tab order.</p>
          </div>
        {/snippet}
      </Tabs>
    </div>
  </section>

  <!-- Keyboard instructions -->
  <section>
    <div class="demo-card info-card">
      <span class="af-label">Keyboard navigation</span>
      <ul class="key-list">
        <li><kbd>Arrow Left / Right</kbd> — move between tabs</li>
        <li><kbd>Home</kbd> — jump to first enabled tab</li>
        <li><kbd>End</kbd> — jump to last enabled tab</li>
        <li><kbd>Tab</kbd> — leave the tablist (panel is next in tab order)</li>
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

  .demo-card {
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    padding: var(--af-card-padding);
  }

  .panel-content {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
    min-height: 80px;
  }

  .panel-text {
    font-size: var(--af-text-sm);
    color: var(--af-text-secondary);
    line-height: var(--af-leading-base);
    margin: 0;
  }

  .info-card {
    background: var(--af-surface-raised);
  }

  .key-list {
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
    margin-block-start: var(--af-space-3);
  }

  .key-list li {
    font-size: var(--af-text-sm);
    color: var(--af-text-secondary);
  }

  kbd {
    font-family: var(--af-font-body);
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-semibold);
    background: var(--af-surface);
    border: 1px solid var(--af-border-strong);
    border-radius: var(--af-radius-sm);
    padding: 2px 6px;
    color: var(--af-text);
  }
</style>
