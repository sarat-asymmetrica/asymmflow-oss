<script lang="ts">
  import { Input, Textarea, CurrencyInput } from '@asymmflow/ui';

  let textVal = $state('');
  let emailVal = $state('the maintainer@asymmetrica.ai');
  let passwordVal = $state('correct-horse-battery');
  let searchVal = $state('');
  let taVal = $state('Line one.\nLine two.');
  let taAutoVal = $state('Auto-resize expands as you type more lines here.');
  let amountBhd = $state(12450.500);
  let amountUsd = $state(0);
</script>

<div class="sections">

  <section>
    <h2 class="af-section-title">Input</h2>
    <p class="intro">
      The wrapper element carries the border and focus ring — the <code>&lt;input&gt;</code>
      itself is chrome-free. Prefix and suffix Snippets support inline adornments without
      breaking the unified border treatment.
    </p>
  </section>

  <!-- States -->
  <section>
    <h2 class="af-section-title">States</h2>
    <div class="card state-grid">
      <div class="demo-cell">
        <span class="af-label">default</span>
        <Input bind:value={textVal} placeholder="Vendor name" />
      </div>
      <div class="demo-cell">
        <span class="af-label">with value</span>
        <Input bind:value={emailVal} type="email" />
      </div>
      <div class="demo-cell">
        <span class="af-label">password</span>
        <Input type="password" bind:value={passwordVal} />
      </div>
      <div class="demo-cell">
        <span class="af-label">search</span>
        <Input type="search" bind:value={searchVal} placeholder="Search invoices…" />
      </div>
      <div class="demo-cell">
        <span class="af-label">disabled</span>
        <Input value="Acme Instrumentation" disabled />
      </div>
      <div class="demo-cell">
        <span class="af-label">readonly</span>
        <Input value="INV-2024-00421" readonly />
      </div>
      <div class="demo-cell">
        <span class="af-label">invalid</span>
        <Input value="not-an-email" type="email" invalid aria-describedby="email-err" />
        <span id="email-err" class="af-meta" style="color: var(--af-danger);">Enter a valid email address.</span>
      </div>
    </div>
  </section>

  <!-- Adornments -->
  <section>
    <h2 class="af-section-title">Prefix &amp; suffix adornments</h2>
    <div class="card state-grid">
      <div class="demo-cell">
        <span class="af-label">prefix icon</span>
        <Input placeholder="Search transactions…">
          {#snippet prefix()}
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
              <circle cx="6" cy="6" r="4.5" stroke="currentColor" stroke-width="1.4"/>
              <path d="M9.5 9.5L13 13" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/>
            </svg>
          {/snippet}
        </Input>
      </div>
      <div class="demo-cell">
        <span class="af-label">suffix unit</span>
        <Input value="2500" type="text">
          {#snippet suffix()}
            <span class="af-label" style="color: var(--af-text-muted);">KG</span>
          {/snippet}
        </Input>
      </div>
      <div class="demo-cell">
        <span class="af-label">prefix + suffix</span>
        <Input value="0.00" type="text">
          {#snippet prefix()}
            <span class="af-label" style="color: var(--af-text-muted);">$</span>
          {/snippet}
          {#snippet suffix()}
            <span class="af-label" style="color: var(--af-text-muted);">USD</span>
          {/snippet}
        </Input>
      </div>
    </div>
  </section>

  <!-- Textarea -->
  <section>
    <h2 class="af-section-title">Textarea</h2>
    <p class="intro">Same state model as Input. <code>autoResize</code> expands vertically; manual resize is removed.</p>
    <div class="card state-grid">
      <div class="demo-cell">
        <span class="af-label">default (3 rows)</span>
        <Textarea bind:value={taVal} placeholder="Enter remarks…" />
      </div>
      <div class="demo-cell">
        <span class="af-label">auto-resize</span>
        <Textarea bind:value={taAutoVal} autoResize />
      </div>
      <div class="demo-cell">
        <span class="af-label">disabled</span>
        <Textarea value="Terms accepted by client on 2024-06-01." disabled />
      </div>
      <div class="demo-cell">
        <span class="af-label">invalid</span>
        <Textarea value="" invalid aria-describedby="ta-err" placeholder="Description required" />
        <span id="ta-err" class="af-meta" style="color: var(--af-danger);">Description is required for BHD transactions.</span>
      </div>
    </div>
  </section>

  <!-- CurrencyInput -->
  <section>
    <h2 class="af-section-title">CurrencyInput</h2>
    <p class="intro">
      Right-aligned with <code>.af-numeric</code> (tabular numerals always).
      BHD uses 3 decimals — the component defaults to this. Formats on blur,
      raw while editing. The currency code is an UPPERCASE label, not a prefix icon.
    </p>
    <div class="card state-grid">
      <div class="demo-cell">
        <span class="af-label">BHD · 3 decimals</span>
        <CurrencyInput bind:value={amountBhd} currency="BHD" decimals={3} />
        <span class="af-meta">value: {amountBhd}</span>
      </div>
      <div class="demo-cell">
        <span class="af-label">USD · 2 decimals</span>
        <CurrencyInput bind:value={amountUsd} currency="USD" decimals={2} />
        <span class="af-meta">value: {amountUsd}</span>
      </div>
      <div class="demo-cell">
        <span class="af-label">disabled</span>
        <CurrencyInput value={6250.000} currency="BHD" disabled />
      </div>
      <div class="demo-cell">
        <span class="af-label">invalid</span>
        <CurrencyInput value={-1} currency="BHD" invalid min={0} aria-describedby="cur-err" />
        <span id="cur-err" class="af-meta" style="color: var(--af-danger);">Amount must be 0 or greater.</span>
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

  .state-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
    gap: var(--af-space-4);
  }

  .demo-cell {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
  }
</style>
