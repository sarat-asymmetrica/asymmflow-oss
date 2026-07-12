<script lang="ts">
  /**
   * AmbientPage — showcase for AmbientField.
   *
   * Demonstrates the full intensity range and parallax depth.
   * Rebuilds as a "stage": large viewport, live controls, preset buttons.
   *
   * Constitution §0: defaults whisper. The Stage preset shows what ceremonies get.
   */

  import AmbientField from '@asymmflow/scenes/AmbientField.svelte';
  import { Button, Toggle } from '@asymmflow/ui';

  // ── Demo state ───────────────────────────────────────────────────────────
  let intensity = $state(0.35);
  let parallaxOn = $state(false);
  let fieldSeed = $state('dashboard');
  let density = $state(1.0);
  let paused = $state(false);

  // ── Presets ──────────────────────────────────────────────────────────────
  function applyProduction() {
    intensity = 0.35;
    parallaxOn = false;
  }

  function applyStage() {
    intensity = 0.85;
    parallaxOn = true;
  }
</script>

<div class="sections">
  <section>
    <h2 class="af-section-title">AmbientField — the whisper layer</h2>
    <p class="intro">
      Slow-drifting points on quaternion walk trajectories, connected by faint
      constellation lines when near. Colors come from CSS tokens — the field
      recolors itself when the theme changes. Under
      <code>prefers-reduced-motion</code>, a single static frame is rendered.
      Paused automatically when the tab is hidden.
    </p>
    <p class="intro">
      <strong>§0 stance:</strong> production defaults whisper. The intensity slider
      shows the full range. The Stage preset is for ceremonies — login screens,
      dashboards, first-run moments. It never appears inside forms or data tables.
    </p>
  </section>

  <!-- Stage: large demo viewport -->
  <section>
    <h3 class="af-label section-label">Live stage</h3>

    <div class="stage">
      <AmbientField {density} seed={fieldSeed} {paused} {intensity} parallax={parallaxOn} />
    </div>

    <!-- Controls -->
    <div class="controls">
      <!-- Intensity slider -->
      <div class="control-group">
        <label class="af-label" for="intensity-slider">Intensity</label>
        <input
          id="intensity-slider"
          type="range"
          min="0"
          max="1"
          step="0.05"
          bind:value={intensity}
          aria-label="Ambient field intensity"
          aria-valuemin={0}
          aria-valuemax={1}
          aria-valuenow={intensity}
        />
        <span class="af-numeric af-meta">{intensity.toFixed(2)}</span>
      </div>

      <!-- Parallax toggle -->
      <div class="control-group">
        <Toggle
          label="Parallax"
          bind:checked={parallaxOn}
          description="Pointer depth — reduced-motion disables"
        />
      </div>

      <!-- Seed input -->
      <div class="control-group">
        <label class="af-label" for="field-seed">Seed</label>
        <input
          id="field-seed"
          type="text"
          class="seed-input"
          bind:value={fieldSeed}
          placeholder="dashboard"
          maxlength="32"
          aria-label="Ambient field seed"
        />
      </div>

      <!-- Density slider -->
      <div class="control-group">
        <label class="af-label" for="density-slider">Density</label>
        <input
          id="density-slider"
          type="range"
          min="0.2"
          max="4"
          step="0.1"
          bind:value={density}
          aria-label="Field density"
        />
        <span class="af-numeric af-meta">{density.toFixed(1)}</span>
      </div>

      <!-- Pause toggle -->
      <div class="control-group">
        <Toggle
          label="Paused"
          bind:checked={paused}
        />
      </div>
    </div>

    <!-- Preset buttons -->
    <div class="presets">
      <span class="af-label preset-label">Presets</span>
      <Button variant="secondary" size="sm" onclick={applyProduction}>
        {#snippet children()}Production{/snippet}
      </Button>
      <Button variant="primary" size="sm" onclick={applyStage}>
        {#snippet children()}Stage{/snippet}
      </Button>
    </div>
  </section>

  <!-- Seed variations -->
  <section>
    <h3 class="af-label section-label">Seed variations — static frames</h3>
    <p class="intro-sm">
      Each seed produces a different initial arrangement. The topology is
      deterministic — same seed always renders identically.
    </p>
    <div class="seed-row">
      {#each ['sales', 'logistics', 'finance', 'hr'] as s}
        <div class="seed-cell">
          <div class="mini-field">
            <AmbientField seed={s} density={0.6} paused intensity={0.45} />
          </div>
          <span class="af-meta">{s}</span>
        </div>
      {/each}
    </div>
  </section>

  <!-- Prop docs -->
  <section>
    <h3 class="af-label section-label">Props</h3>
    <div class="prop-table">
      <div class="prop-row prop-row--header">
        <span class="af-label">Prop</span>
        <span class="af-label">Type</span>
        <span class="af-label">Default</span>
        <span class="af-label">Description</span>
      </div>

      <div class="prop-row">
        <code>density</code>
        <span class="af-meta">number</span>
        <code>1</code>
        <span class="af-meta">Point count factor. 1 = ~40 pts, max ~200.</span>
      </div>
      <div class="prop-row">
        <code>seed</code>
        <span class="af-meta">string</span>
        <code>'ambient'</code>
        <span class="af-meta">Deterministic seed — same string, same constellation.</span>
      </div>
      <div class="prop-row">
        <code>connectionDist</code>
        <span class="af-meta">number</span>
        <code>120</code>
        <span class="af-meta">Max px distance at which a line is drawn between two points.</span>
      </div>
      <div class="prop-row">
        <code>paused</code>
        <span class="af-meta">boolean</span>
        <code>false</code>
        <span class="af-meta">Freeze animation (useful for reduced-motion and screenshot tests).</span>
      </div>
      <div class="prop-row prop-row--highlight">
        <code>intensity</code>
        <span class="af-meta">number&nbsp;0–1</span>
        <code>0.35</code>
        <span class="af-meta">
          Master dial. Scales canvas opacity, particle opacity, dot radius, line
          width, line alpha, and color mix. At &gt;&nbsp;0.5, accent dots appear on
          every 5th particle. Default 0.35 = §0 whisper.
        </span>
      </div>
      <div class="prop-row prop-row--highlight">
        <code>parallax</code>
        <span class="af-meta">boolean</span>
        <code>false</code>
        <span class="af-meta">
          Pointer-driven depth shift. Each particle has a seeded depth factor;
          offset = (pointer − center) × depth × 0.03 × intensity, capped at 28px.
          Fully disabled under prefers-reduced-motion.
        </span>
      </div>
    </div>
  </section>

  <!-- Constitution note -->
  <section class="constitution-note">
    <p class="af-meta">
      <strong>Constitution §0:</strong> AmbientField lives at the
      <em>edges</em> only — behind login ceremonies, as a dashboard depth layer,
      in empty states. It never appears inside data tables, forms, or ledgers.
      The center is calm; the field exists to remind users the system is alive.
      Production defaults whisper. <code>intensity</code> is for ceremonies and demos.
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

  .intro-sm {
    font-size: var(--af-text-sm);
    color: var(--af-text-secondary);
    margin-top: 0;
    margin-bottom: var(--af-space-3);
    line-height: var(--af-leading-base);
  }

  .section-label {
    margin-bottom: var(--af-space-3);
  }

  code {
    font-family: monospace;
    font-size: 0.9em;
    background: var(--af-tint-medium);
    padding: 1px 4px;
    border-radius: 3px;
  }

  /* ── Stage ────────────────────────────────────────────────────────── */
  .stage {
    position: relative;
    height: 520px;
    background: var(--af-surface-sunken);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    overflow: hidden;
    margin-bottom: var(--af-space-3);
  }

  /* ── Controls ─────────────────────────────────────────────────────── */
  .controls {
    display: flex;
    align-items: flex-start;
    gap: var(--af-space-4);
    flex-wrap: wrap;
    margin-bottom: var(--af-space-3);
  }

  .control-group {
    display: flex;
    align-items: center;
    gap: var(--af-space-2);
  }

  .control-group input[type='range'] {
    accent-color: var(--af-accent);
    width: 120px;
  }

  .seed-input {
    height: 32px;
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    background: var(--af-surface);
    color: var(--af-text);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    padding: 0 var(--af-space-2);
    width: 120px;
    outline: none;
    transition: border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .seed-input:focus {
    border-color: var(--af-focus-ring);
    box-shadow: 0 0 0 3px var(--af-accent-tint);
  }

  /* ── Presets ──────────────────────────────────────────────────────── */
  .presets {
    display: flex;
    align-items: center;
    gap: var(--af-space-2);
    margin-bottom: var(--af-space-3);
  }

  .preset-label {
    color: var(--af-text-secondary);
    margin-inline-end: var(--af-space-1);
  }

  /* ── Seed row ─────────────────────────────────────────────────────── */
  .seed-row {
    display: flex;
    gap: var(--af-space-3);
    flex-wrap: wrap;
  }

  .seed-cell {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--af-space-2);
    flex: 1;
    min-width: 160px;
  }

  .mini-field {
    position: relative;
    width: 100%;
    height: 140px;
    background: var(--af-surface-raised);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    overflow: hidden;
  }

  /* ── Prop docs ────────────────────────────────────────────────────── */
  .prop-table {
    display: flex;
    flex-direction: column;
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    overflow: hidden;
  }

  .prop-row {
    display: grid;
    grid-template-columns: 140px 100px 100px 1fr;
    gap: var(--af-space-3);
    padding: var(--af-space-3) var(--af-space-4);
    border-bottom: 1px solid var(--af-border);
    align-items: baseline;
  }

  .prop-row:last-child {
    border-bottom: none;
  }

  .prop-row--header {
    background: var(--af-surface-sunken);
  }

  .prop-row--highlight {
    background: var(--af-accent-tint);
  }

  .prop-row .af-meta {
    color: var(--af-text-secondary);
    font-size: var(--af-text-sm);
    line-height: var(--af-leading-base);
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
