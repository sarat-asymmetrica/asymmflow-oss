<script lang="ts">
  /**
   * GlyphPage — showcase for GlyphMark, the generative identity mark.
   *
   * Proves:
   *   1. Deterministic variety: 12 seeds → 12 distinct-but-coherent marks
   *   2. Same seed = same mark, always
   *   3. Live seed input: type any string, watch the mark form
   *   4. Draw-on replay
   */

  import GlyphMark from '@asymmflow/scenes/GlyphMark.svelte';

  // Reference seeds — real company/project names that should each produce
  // a unique and recognizable sigil.
  const referenceSeed = [
    'Acme Instrumentation',
    'AsymmFlow',
    'Rythu Mitra',
    'EthioCare',
    'Ananta',
    'Asymmetrica',
    'darch',
    'Menubar',
    'LaunchTable',
    'VedicDoc',
    'Sarvam',
    'Betanet',
  ];

  let customSeed = $state('');
  let replayKey = $state(0);

  function triggerReplay() {
    replayKey += 1;
  }

  const previewSeed = $derived(customSeed.trim() || 'Your Company');
</script>

<div class="sections">
  <section>
    <h2 class="af-section-title">GlyphMark — generative identity</h2>
    <p class="intro">
      Each seed string produces a deterministic quaternion walk on S³, projected to a
      smooth stroke path inside a rounded-square tile. Same seed, same mark, forever.
      The marks look like signatures or sigils — not blobs. Motion is earned: a
      stroke-dasharray reveal at R1 · Explore speed (400ms). Under
      <code>prefers-reduced-motion</code>, the path renders instantly.
    </p>
  </section>

  <!-- Grid of reference seeds -->
  <section>
    <h3 class="af-label section-label">Reference marks — 12 seeds</h3>
    <div class="glyph-grid">
      {#each referenceSeed as seed}
        <figure class="glyph-figure">
          <GlyphMark {seed} size={72} />
          <figcaption class="af-meta seed-caption">{seed}</figcaption>
        </figure>
      {/each}
    </div>
  </section>

  <!-- Live seed input -->
  <section>
    <h3 class="af-label section-label">Live seed — type any string</h3>
    <div class="live-demo card">
      <div class="live-input-row">
        <label for="seed-input" class="af-label">Seed</label>
        <input
          id="seed-input"
          class="seed-input"
          type="text"
          placeholder="Your Company"
          bind:value={customSeed}
          maxlength="64"
          aria-label="Seed for live GlyphMark preview"
        />
        <button class="replay-btn" onclick={triggerReplay} aria-label="Replay draw animation">
          Replay
        </button>
      </div>

      <div class="live-mark">
        {#key `${previewSeed}-${replayKey}`}
          <GlyphMark seed={previewSeed} size={120} />
        {/key}
        <div class="live-label">
          <span class="af-text-xl seed-name">{previewSeed}</span>
          <span class="af-meta">Same seed, same mark — always.</span>
        </div>
      </div>
    </div>
  </section>

  <!-- Sizes -->
  <section>
    <h3 class="af-label section-label">Size variants</h3>
    <div class="size-row card">
      {#each [24, 32, 48, 64, 96] as sz}
        <div class="size-item">
          <GlyphMark seed="AsymmFlow" size={sz} animate={false} />
          <span class="af-meta">{sz}px</span>
        </div>
      {/each}
    </div>
  </section>

  <!-- Constitution note -->
  <section class="constitution-note">
    <p class="af-meta">
      <strong>Constitution §5:</strong> Generative identity lives at the
      <em>edges</em> — login ceremonies, nav brand marks, empty states.
      Not in table cells, not in form labels. The center is quiet; the
      edges earn expression.
    </p>
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
    line-height: var(--af-leading-base);
  }

  .section-label {
    margin-bottom: var(--af-space-3);
  }

  /* ── Grid ─────────────────────────────────────────────────────────── */
  .glyph-grid {
    display: flex;
    flex-wrap: wrap;
    gap: var(--af-space-4);
  }

  .glyph-figure {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--af-space-2);
    margin: 0;
  }

  .seed-caption {
    text-align: center;
    max-width: 80px;
    word-break: break-word;
  }

  /* ── Live demo ────────────────────────────────────────────────────── */
  .card {
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    padding: var(--af-card-padding);
  }

  .live-demo {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-4);
  }

  .live-input-row {
    display: flex;
    align-items: center;
    gap: var(--af-space-3);
    flex-wrap: wrap;
  }

  .seed-input {
    height: var(--af-control-height);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    background: var(--af-surface-raised);
    color: var(--af-text);
    font-family: var(--af-font-body);
    font-size: var(--af-text-md);
    padding: 0 var(--af-space-3);
    flex: 1;
    min-width: 200px;
    outline: none;
    transition:
      border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .seed-input:focus {
    border-color: var(--af-focus-ring);
    box-shadow: 0 0 0 3px var(--af-accent-tint);
  }

  .replay-btn {
    height: var(--af-control-height);
    padding: 0 var(--af-space-4);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    background: var(--af-surface);
    color: var(--af-text);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-medium);
    cursor: pointer;
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
    white-space: nowrap;
  }

  .replay-btn:hover {
    background: var(--af-surface-raised);
    border-color: var(--af-border-strong);
  }

  .replay-btn:focus-visible {
    outline: 3px solid var(--af-focus-ring);
    outline-offset: 2px;
  }

  .live-mark {
    display: flex;
    align-items: center;
    gap: var(--af-space-5);
  }

  .live-label {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
  }

  .seed-name {
    font-family: var(--af-font-display);
    font-size: var(--af-text-2xl);
    font-weight: var(--af-weight-bold);
    letter-spacing: var(--af-title-tracking);
    color: var(--af-text);
  }

  /* ── Size row ─────────────────────────────────────────────────────── */
  .size-row {
    display: flex;
    align-items: flex-end;
    gap: var(--af-space-4);
    flex-wrap: wrap;
  }

  .size-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--af-space-2);
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

  code {
    font-family: monospace;
    font-size: 0.9em;
    background: var(--af-tint-medium);
    padding: 1px 4px;
    border-radius: 3px;
  }
</style>
