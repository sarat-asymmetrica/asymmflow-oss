<script lang="ts">
  /**
   * LoginCeremony — the arrival moment.
   *
   * Full-bleed composition: AmbientField behind, GlyphMark above, calm login card
   * centered with staggered R1 entrance (≤700ms total). Constitution §0 and §4e.
   *
   * Props:
   *   companyName  - The organization name. Seeds the GlyphMark and headline.
   *   onSubmit     - Called with { email, password } on form submit.
   *   subtitle     - Optional short tagline below the company name.
   */

  import AmbientField from './AmbientField.svelte';
  import GlyphMark from './GlyphMark.svelte';

  interface SubmitPayload {
    email: string;
    password: string;
  }

  interface Props {
    companyName: string;
    onSubmit?: (payload: SubmitPayload) => void;
    subtitle?: string;
  }

  let { companyName, onSubmit, subtitle }: Props = $props();

  let email = $state('');
  let password = $state('');
  let loading = $state(false);
  let errorMsg = $state('');

  function handleSubmit(e: Event) {
    e.preventDefault();
    if (!email.trim() || !password) {
      errorMsg = 'Please enter your email and password.';
      return;
    }
    errorMsg = '';
    loading = true;
    onSubmit?.({ email: email.trim(), password });
  }

  // Reduced motion: skip entrance animation
  const prefersReduced =
    typeof window !== 'undefined'
      ? window.matchMedia('(prefers-reduced-motion: reduce)').matches
      : false;
</script>

<div class="ceremony">
  <!-- Ambient background — lives at the edge, behind everything -->
  <div class="ambient-layer" aria-hidden="true">
    <AmbientField seed={companyName} density={0.8} />
  </div>

  <!-- Centered login card -->
  <main class="card-wrapper">
    <div class="card" class:no-animate={prefersReduced}>
      <!-- Identity mark + company name (stagger item 1) -->
      <div class="identity" style:--lc-index={0}>
        <GlyphMark seed={companyName} size={52} animate={!prefersReduced} />
        <div class="brand-text">
          <h1 class="company-name">{companyName}</h1>
          {#if subtitle}
            <p class="subtitle">{subtitle}</p>
          {/if}
        </div>
      </div>

      <!-- Divider (stagger item 2) -->
      <div class="divider" style:--lc-index={1} aria-hidden="true"></div>

      <!-- Headline (stagger item 3) -->
      <p class="headline" style:--lc-index={2}>Sign in to continue</p>

      <!-- Form (stagger item 4) -->
      <form
        class="form"
        style:--lc-index={3}
        onsubmit={handleSubmit}
        novalidate
        aria-label="Sign in form"
      >
        <div class="field">
          <label class="field-label" for="lc-email">Email</label>
          <input
            id="lc-email"
            class="field-input"
            type="email"
            autocomplete="username"
            bind:value={email}
            placeholder="name@company.com"
            required
            aria-required="true"
            disabled={loading}
          />
        </div>

        <div class="field">
          <label class="field-label" for="lc-password">Password</label>
          <input
            id="lc-password"
            class="field-input"
            type="password"
            autocomplete="current-password"
            bind:value={password}
            placeholder="••••••••"
            required
            aria-required="true"
            disabled={loading}
          />
        </div>

        {#if errorMsg}
          <p class="error-msg" role="alert">{errorMsg}</p>
        {/if}

        <button class="submit-btn" type="submit" disabled={loading}>
          {#if loading}
            <span class="spinner" aria-hidden="true"></span>
            Signing in…
          {:else}
            Sign in
          {/if}
        </button>
      </form>
    </div>
  </main>
</div>

<style>
  /* ── Layout ──────────────────────────────────────────────────────────── */
  .ceremony {
    position: relative;
    min-height: 100vh;
    width: 100%;
    background: var(--af-bg);
    display: grid;
    place-items: center;
    padding: var(--af-space-4);
    box-sizing: border-box;
  }

  .ambient-layer {
    position: absolute;
    inset: 0;
    /* Canvas is a bg edge element — behind card via z-index */
    z-index: 0;
    overflow: hidden;
  }

  .card-wrapper {
    position: relative;
    /* Above the ambient canvas (which sits at 0, below the base rung). */
    z-index: var(--af-z-base);
    width: 100%;
    max-width: 400px;
  }

  /* ── Card ────────────────────────────────────────────────────────────── */
  .card {
    background: var(--af-glass-bg);
    border: 1px solid var(--af-glass-border);
    backdrop-filter: var(--af-glass-blur);
    -webkit-backdrop-filter: var(--af-glass-blur);
    border-radius: var(--af-radius-lg);
    padding: var(--af-space-6);
    box-shadow: var(--af-shadow-overlay);
    display: flex;
    flex-direction: column;
    gap: var(--af-space-4);
  }

  /* Staggered entrance: each child gets the R1 explore treatment, delayed by
     its --lc-index times the stagger token (no raw ms in markup, §4b/§4e). */
  .card > * {
    animation: ceremony-in var(--af-motion-explore-duration)
      var(--af-motion-explore-ease) both;
    animation-delay: calc(var(--lc-index, 0) * var(--af-motion-stagger));
  }

  .card.no-animate > * {
    animation: none;
  }

  @keyframes ceremony-in {
    from {
      opacity: 0;
      transform: translateY(12px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  /* ── Identity ────────────────────────────────────────────────────────── */
  .identity {
    display: flex;
    align-items: center;
    gap: var(--af-space-3);
  }

  .brand-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .company-name {
    margin: 0;
    font-family: var(--af-font-display);
    font-size: var(--af-text-xl);
    font-weight: var(--af-weight-bold);
    letter-spacing: var(--af-title-tracking);
    color: var(--af-text);
    line-height: var(--af-leading-tight);
  }

  .subtitle {
    margin: 0;
    font-size: var(--af-text-xs);
    color: var(--af-text-muted);
    font-weight: var(--af-weight-medium);
  }

  /* ── Divider ─────────────────────────────────────────────────────────── */
  .divider {
    height: 1px;
    background: var(--af-border);
  }

  /* ── Headline ────────────────────────────────────────────────────────── */
  .headline {
    margin: 0;
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-medium);
    color: var(--af-text-secondary);
    letter-spacing: 0.01em;
  }

  /* ── Form ────────────────────────────────────────────────────────────── */
  .form {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-3);
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-1);
  }

  .field-label {
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-semibold);
    text-transform: uppercase;
    letter-spacing: var(--af-label-tracking);
    color: var(--af-text-secondary);
  }

  .field-input {
    height: var(--af-control-height);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    background: var(--af-surface);
    color: var(--af-text);
    font-family: var(--af-font-body);
    font-size: var(--af-text-md);
    padding: 0 var(--af-space-3);
    box-sizing: border-box;
    width: 100%;
    outline: none;
    transition:
      border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .field-input::placeholder {
    color: var(--af-text-muted);
  }

  .field-input:focus {
    border-color: var(--af-focus-ring);
    box-shadow: 0 0 0 3px var(--af-accent-tint);
  }

  .field-input:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  /* ── Error message ───────────────────────────────────────────────────── */
  .error-msg {
    margin: 0;
    font-size: var(--af-text-sm);
    color: var(--af-danger);
    font-weight: var(--af-weight-medium);
  }

  /* ── Submit button ───────────────────────────────────────────────────── */
  .submit-btn {
    height: var(--af-control-height);
    background: var(--af-accent);
    color: var(--af-accent-contrast);
    border: none;
    border-radius: var(--af-radius-sm);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-semibold);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--af-space-2);
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
    margin-top: var(--af-space-1);
  }

  .submit-btn:hover:not(:disabled) {
    background: var(--af-accent-hover);
    box-shadow: var(--af-shadow-lift);
  }

  .submit-btn:active:not(:disabled) {
    background: var(--af-accent-pressed);
  }

  .submit-btn:focus-visible {
    outline: 3px solid var(--af-focus-ring);
    outline-offset: 2px;
  }

  .submit-btn:disabled {
    opacity: 0.65;
    cursor: not-allowed;
  }

  /* ── Spinner ─────────────────────────────────────────────────────────── */
  .spinner {
    display: inline-block;
    width: 14px;
    height: 14px;
    border: 2px solid transparent;
    border-top-color: currentColor;
    border-radius: 50%;
    animation: spin var(--af-motion-spin) linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .spinner {
      animation: none;
      opacity: 0.7;
    }
  }
</style>
