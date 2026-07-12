<script lang="ts">
  /**
   * ThemeForgePage — THE design engine demo.
   *
   * Proves: themes are data, the engine generates them, the system re-skins itself.
   *
   *   - Text input for seed + light/dark toggle
   *   - generateTheme() → applyTheme(document.documentElement) LIVE
   *   - The entire showcase re-themes instantly
   *   - Generated palette swatches with hex values + contrast ratios
   *   - 'Reset to Onyx & Ether' button
   */

  import { generateTheme, contrastRatio } from '@asymmflow/scenes';
  import { applyTheme, clearTheme, onyxEther } from '@asymmflow/tokens';
  import GlyphMark from '@asymmflow/scenes/GlyphMark.svelte';

  let seed = $state('');
  let mode = $state<'light' | 'dark'>('light');
  let currentTheme = $state(onyxEther);
  let isForged = $state(false);

  const liveSeed = $derived(seed.trim() || 'AsymmFlow');

  // Re-generate whenever seed or mode changes (but don't apply — only on demand)
  const previewTheme = $derived(generateTheme(liveSeed, { mode }));

  function forge() {
    currentTheme = previewTheme;
    isForged = true;
    applyTheme(currentTheme);
  }

  function reset() {
    clearTheme();
    currentTheme = onyxEther;
    isForged = false;
    seed = '';
    mode = 'light';
  }

  // Color swatch rows to display
  interface SwatchGroup {
    label: string;
    swatches: Array<{
      token: string;
      label: string;
      contrastOn?: string; // bg token to check contrast against
    }>;
  }

  const swatchGroups = $derived<SwatchGroup[]>([
    {
      label: 'Surfaces',
      swatches: [
        { token: 'bg', label: 'bg' },
        { token: 'surface', label: 'surface' },
        { token: 'surface-raised', label: 'surface-raised' },
        { token: 'surface-sunken', label: 'surface-sunken' },
      ],
    },
    {
      label: 'Text',
      swatches: [
        { token: 'text', label: 'text', contrastOn: 'bg' },
        { token: 'text-secondary', label: 'text-secondary', contrastOn: 'bg' },
        { token: 'text-muted', label: 'text-muted', contrastOn: 'bg' },
        { token: 'text-inverse', label: 'text-inverse', contrastOn: 'inverse-surface' },
      ],
    },
    {
      label: 'Accent',
      swatches: [
        { token: 'accent', label: 'accent', contrastOn: 'surface' },
        { token: 'accent-hover', label: 'accent-hover', contrastOn: 'surface' },
        { token: 'accent-pressed', label: 'accent-pressed', contrastOn: 'surface' },
        { token: 'accent-contrast', label: 'accent-contrast', contrastOn: 'accent' },
      ],
    },
    {
      label: 'Status',
      swatches: [
        { token: 'success', label: 'success', contrastOn: 'surface' },
        { token: 'warning', label: 'warning', contrastOn: 'surface' },
        { token: 'danger', label: 'danger', contrastOn: 'surface' },
        { token: 'info', label: 'info', contrastOn: 'surface' },
      ],
    },
    {
      label: 'Borders & Tints',
      swatches: [
        { token: 'border', label: 'border' },
        { token: 'border-strong', label: 'border-strong' },
        { token: 'focus-ring', label: 'focus-ring' },
        { token: 'inverse-surface', label: 'inverse-surface' },
      ],
    },
  ]);

  function contrastLabel(ratio: number): string {
    if (ratio >= 7.0) return 'AAA';
    if (ratio >= 4.5) return 'AA';
    if (ratio >= 3.0) return 'AA Large';
    return 'Fail';
  }

  function contrastClass(ratio: number): string {
    if (ratio >= 4.5) return 'cr-pass';
    if (ratio >= 3.0) return 'cr-large';
    return 'cr-fail';
  }
</script>

<div class="sections">
  <section>
    <h2 class="af-section-title">Theme Forge — the design engine</h2>
    <p class="intro">
      Seed string → deterministic hue → OKLCH-derived neutral ramp → complete
      <code>Theme</code> object — validated, applied, live. Every property advances
      the same philosophy: one confident accent, tinted neutrals (never pure gray),
      AA contrast guaranteed on the accent, status colors harmonized to the seed hue.
      The entire showcase re-skins itself instantly because themes are just data.
    </p>
  </section>

  <!-- Forge controls -->
  <section>
    <div class="forge-card">
      <div class="forge-controls">
        <div class="input-group">
          <label class="af-label" for="forge-seed">Seed</label>
          <input
            id="forge-seed"
            class="forge-input"
            type="text"
            placeholder="AsymmFlow"
            bind:value={seed}
            maxlength="64"
            aria-label="Theme seed string"
          />
        </div>

        <div class="mode-toggle" role="group" aria-label="Color mode">
          <button
            class="mode-btn"
            class:on={mode === 'light'}
            onclick={() => (mode = 'light')}
          >Light</button>
          <button
            class="mode-btn"
            class:on={mode === 'dark'}
            onclick={() => (mode = 'dark')}
          >Dark</button>
        </div>

        <button class="forge-btn" onclick={forge}>
          Apply theme
        </button>

        {#if isForged}
          <button class="reset-btn" onclick={reset}>
            Reset to Onyx &amp; Ether
          </button>
        {/if}
      </div>

      <!-- Glyph preview of the seed -->
      <div class="seed-preview">
        {#key liveSeed}
          <GlyphMark seed={liveSeed} size={64} />
        {/key}
        <div>
          <div class="af-text-lg preview-seed">{liveSeed}</div>
          <div class="af-meta">
            Theme: <code>{previewTheme.name}</code>
          </div>
        </div>
      </div>
    </div>
  </section>

  <!-- Palette swatches -->
  <section>
    <h3 class="af-label section-label">Generated palette</h3>
    <div class="swatch-groups">
      {#each swatchGroups as group}
        <div class="swatch-group">
          <div class="af-label swatch-group-label">{group.label}</div>
          <div class="swatch-row">
            {#each group.swatches as swatch}
              {@const value = previewTheme.tokens[swatch.token as keyof typeof previewTheme.tokens] ?? ''}
              {@const bgValue = swatch.contrastOn
                ? previewTheme.tokens[swatch.contrastOn as keyof typeof previewTheme.tokens] ?? '#ffffff'
                : null}
              {@const cr = bgValue && value && !value.startsWith('rgba') && !value.startsWith('rgb')
                ? contrastRatio(value, bgValue)
                : null}
              <div class="swatch-item">
                <div
                  class="swatch-chip"
                  style:background={value}
                  title={value}
                  role="presentation"
                ></div>
                <div class="swatch-info">
                  <span class="af-label swatch-token">--af-{swatch.label}</span>
                  <span class="af-meta swatch-value">{value.length > 22 ? value.slice(0, 20) + '…' : value}</span>
                  {#if cr !== null}
                    <span class="contrast-badge {contrastClass(cr)}">
                      {cr.toFixed(1)}:1 {contrastLabel(cr)}
                    </span>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        </div>
      {/each}
    </div>
  </section>

  <!-- Key contrast ratio callouts -->
  <section>
    <h3 class="af-label section-label">Key contrast ratios</h3>
    <div class="contrast-table">
      {#each [
        { fg: 'accent', bg: 'surface', label: 'Accent on surface' },
        { fg: 'text', bg: 'bg', label: 'Text on bg' },
        { fg: 'text-secondary', bg: 'bg', label: 'Text-secondary on bg' },
        { fg: 'accent-contrast', bg: 'accent', label: 'accent-contrast on accent' },
      ] as row}
        {@const fgVal = previewTheme.tokens[row.fg as keyof typeof previewTheme.tokens] ?? ''}
        {@const bgVal = previewTheme.tokens[row.bg as keyof typeof previewTheme.tokens] ?? ''}
        {@const cr = fgVal && bgVal && !fgVal.startsWith('rgba') && !bgVal.startsWith('rgba')
          ? contrastRatio(fgVal, bgVal)
          : 0}
        <div class="contrast-row">
          <span class="af-text-sm contrast-label-text">{row.label}</span>
          <div class="contrast-preview" style:background={bgVal} style:color={fgVal}>
            Aa
          </div>
          <span class="contrast-badge contrast-badge--large {contrastClass(cr)}">
            {cr.toFixed(1)}:1 {contrastLabel(cr)}
          </span>
        </div>
      {/each}
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
    line-height: var(--af-leading-base);
  }

  code {
    font-family: monospace;
    font-size: 0.9em;
    background: var(--af-tint-medium);
    padding: 1px 4px;
    border-radius: 3px;
  }

  .section-label {
    margin-bottom: var(--af-space-3);
  }

  /* ── Forge card ───────────────────────────────────────────────────── */
  .forge-card {
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    padding: var(--af-card-padding);
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--af-space-5);
    flex-wrap: wrap;
  }

  .forge-controls {
    display: flex;
    align-items: center;
    gap: var(--af-space-3);
    flex-wrap: wrap;
    flex: 1;
  }

  .input-group {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-1);
  }

  .forge-input {
    height: var(--af-control-height);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    background: var(--af-surface-raised);
    color: var(--af-text);
    font-family: var(--af-font-body);
    font-size: var(--af-text-md);
    padding: 0 var(--af-space-3);
    min-width: 200px;
    outline: none;
    transition: border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .forge-input:focus {
    border-color: var(--af-focus-ring);
    box-shadow: 0 0 0 3px var(--af-accent-tint);
  }

  .mode-toggle {
    display: flex;
  }

  .mode-btn {
    border: 1px solid var(--af-border);
    background: var(--af-surface);
    color: var(--af-text-secondary);
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-medium);
    padding: 6px 12px;
    cursor: pointer;
    height: var(--af-control-height);
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .mode-btn:first-child {
    border-radius: var(--af-radius-sm) 0 0 var(--af-radius-sm);
  }

  .mode-btn:last-child {
    border-radius: 0 var(--af-radius-sm) var(--af-radius-sm) 0;
    margin-left: -1px;
  }

  .mode-btn.on {
    background: var(--af-inverse-surface);
    border-color: var(--af-inverse-surface);
    color: var(--af-text-inverse);
  }

  .forge-btn {
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
  }

  .forge-btn:hover {
    background: var(--af-accent-hover);
  }

  .forge-btn:focus-visible {
    outline: 3px solid var(--af-focus-ring);
    outline-offset: 2px;
  }

  .reset-btn {
    height: var(--af-control-height);
    padding: 0 var(--af-space-4);
    background: transparent;
    color: var(--af-text-secondary);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-medium);
    cursor: pointer;
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .reset-btn:hover {
    background: var(--af-danger-tint);
    color: var(--af-danger);
    border-color: var(--af-danger);
  }

  .reset-btn:focus-visible {
    outline: 3px solid var(--af-focus-ring);
    outline-offset: 2px;
  }

  .seed-preview {
    display: flex;
    align-items: center;
    gap: var(--af-space-3);
  }

  .preview-seed {
    font-family: var(--af-font-display);
    font-weight: var(--af-weight-bold);
    letter-spacing: var(--af-title-tracking);
    color: var(--af-text);
  }

  /* ── Swatches ─────────────────────────────────────────────────────── */
  .swatch-groups {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-4);
  }

  .swatch-group {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
  }

  .swatch-group-label {
    color: var(--af-text-secondary);
  }

  .swatch-row {
    display: flex;
    gap: var(--af-space-3);
    flex-wrap: wrap;
  }

  .swatch-item {
    display: flex;
    align-items: center;
    gap: var(--af-space-2);
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    padding: var(--af-space-2) var(--af-space-3);
    min-width: 200px;
  }

  .swatch-chip {
    width: 32px;
    height: 32px;
    border-radius: var(--af-radius-sm);
    flex-shrink: 0;
    border: 1px solid var(--af-border);
  }

  .swatch-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .swatch-token {
    color: var(--af-text);
    font-size: 10px;
  }

  .swatch-value {
    font-family: monospace;
    font-size: 10px;
    color: var(--af-text-muted);
  }

  /* ── Contrast badges ──────────────────────────────────────────────── */
  .contrast-badge {
    display: inline-block;
    font-size: 9px;
    font-weight: var(--af-weight-semibold);
    padding: 1px 5px;
    border-radius: var(--af-radius-pill);
    letter-spacing: 0.03em;
  }

  .contrast-badge--large {
    font-size: 11px;
    padding: 2px 8px;
  }

  .cr-pass {
    background: var(--af-success-tint);
    color: var(--af-success);
  }

  .cr-large {
    background: var(--af-warning-tint);
    color: var(--af-warning);
  }

  .cr-fail {
    background: var(--af-danger-tint);
    color: var(--af-danger);
  }

  /* ── Contrast table ───────────────────────────────────────────────── */
  .contrast-table {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    padding: var(--af-card-padding);
  }

  .contrast-row {
    display: flex;
    align-items: center;
    gap: var(--af-space-4);
  }

  .contrast-label-text {
    flex: 1;
    color: var(--af-text-secondary);
  }

  .contrast-preview {
    width: 48px;
    height: 32px;
    border-radius: var(--af-radius-sm);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: var(--af-text-lg);
    font-weight: var(--af-weight-bold);
    border: 1px solid var(--af-border);
    flex-shrink: 0;
  }
</style>
