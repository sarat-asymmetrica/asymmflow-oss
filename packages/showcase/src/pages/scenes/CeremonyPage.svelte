<script lang="ts">
  /**
   * CeremonyPage — LoginCeremony embedded in a viewport-height card.
   *
   * Props exposed: company-name input, replay button.
   * Shows the arrival moment: AmbientField + GlyphMark + staggered card entrance.
   */

  import LoginCeremony from '@asymmflow/scenes/LoginCeremony.svelte';

  let companyName = $state('Acme Instrumentation');
  let inputValue = $state('Acme Instrumentation');
  let replayKey = $state(0);
  let lastSubmit = $state<{ email: string } | null>(null);

  function apply() {
    companyName = inputValue.trim() || 'AsymmFlow';
    replayKey += 1;
  }

  function handleSubmit(payload: { email: string; password: string }) {
    lastSubmit = { email: payload.email };
  }
</script>

<div class="sections">
  <section>
    <h2 class="af-section-title">LoginCeremony — the arrival moment</h2>
    <p class="intro">
      The login is a ceremony: AmbientField whispers behind, a GlyphMark anchors
      identity, and the card elements arrive in a staggered R1 · Explore entrance
      (≤700ms total). Under <code>prefers-reduced-motion</code>, all elements
      render instantly. The ambient field pauses when the tab is hidden.
    </p>
  </section>

  <!-- Company name control -->
  <section>
    <div class="controls">
      <label class="control-group">
        <span class="af-label">Company name</span>
        <input
          type="text"
          class="name-input"
          bind:value={inputValue}
          placeholder="Acme Instrumentation"
          maxlength="48"
          aria-label="Company name for ceremony demo"
        />
      </label>
      <button class="apply-btn" onclick={apply}>Apply &amp; replay</button>
    </div>
  </section>

  <!-- The ceremony itself — viewport-height feel -->
  {#key replayKey}
    <section>
      <div class="ceremony-stage">
        <LoginCeremony
          {companyName}
          subtitle="Back-office operations"
          onSubmit={handleSubmit}
        />
      </div>
    </section>
  {/key}

  {#if lastSubmit}
    <section class="submit-result">
      <p class="af-meta">
        Sign-in submitted for <strong>{lastSubmit.email}</strong>.
        (In production, onSubmit routes to your auth provider.)
      </p>
    </section>
  {/if}

  <!-- Notes -->
  <section class="constitution-note">
    <p class="af-meta">
      <strong>Constitution §0:</strong> The login ceremony is an
      <em>edge moment</em> — the only full-bleed composition in the product.
      Once past login, the ambient layer retreats to dashboard backgrounds only.
      The data surfaces (tables, forms, ledgers) are undecorated.
    </p>
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
    line-height: var(--af-leading-base);
  }

  code {
    font-family: monospace;
    font-size: 0.9em;
    background: var(--af-tint-medium);
    padding: 1px 4px;
    border-radius: 3px;
  }

  /* ── Controls ─────────────────────────────────────────────────────── */
  .controls {
    display: flex;
    align-items: flex-end;
    gap: var(--af-space-3);
    flex-wrap: wrap;
  }

  .control-group {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-1);
  }

  .name-input {
    height: var(--af-control-height);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    background: var(--af-surface);
    color: var(--af-text);
    font-family: var(--af-font-body);
    font-size: var(--af-text-md);
    padding: 0 var(--af-space-3);
    min-width: 220px;
    outline: none;
    transition: border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .name-input:focus {
    border-color: var(--af-focus-ring);
    box-shadow: 0 0 0 3px var(--af-accent-tint);
  }

  .apply-btn {
    height: var(--af-control-height);
    padding: 0 var(--af-space-4);
    background: var(--af-accent);
    color: var(--af-accent-contrast);
    border: none;
    border-radius: var(--af-radius-sm);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-semibold);
    cursor: pointer;
    transition: background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
    white-space: nowrap;
  }

  .apply-btn:hover {
    background: var(--af-accent-hover);
  }

  .apply-btn:focus-visible {
    outline: 3px solid var(--af-focus-ring);
    outline-offset: 2px;
  }

  /* ── Ceremony stage ───────────────────────────────────────────────── */
  .ceremony-stage {
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-lg);
    overflow: hidden;
    /* 80vh so it feels like a real login page without leaving the showcase */
    min-height: 80vh;
    position: relative;
  }

  /* ── Submit result ────────────────────────────────────────────────── */
  .submit-result {
    background: var(--af-success-tint);
    border: 1px solid var(--af-success);
    border-radius: var(--af-radius-md);
    padding: var(--af-space-3) var(--af-space-4);
    color: var(--af-success);
  }

  .submit-result p {
    margin: 0;
  }

  /* ── Constitution note ────────────────────────────────────────────── */
  .constitution-note {
    background: var(--af-surface-sunken);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    padding: var(--af-space-4);
  }

  .constitution-note p {
    margin: 0;
    color: var(--af-text-secondary);
    line-height: var(--af-leading-base);
  }
</style>
