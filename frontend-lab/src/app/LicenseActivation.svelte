<script lang="ts">
  /* License activation — the ONE live auth gate (K5 owner ruling: license-only).
   * Shown by the shell when no valid license is present. Enter a PH-XXX-YYYYYY
   * key; on success the shell sets the session + enters the app. Auth CHROME
   * (full-screen, pre-app) — owns its own centered layout like App.svelte's
   * shell, not a product screen. Uses the kernel k-input control for the field. */
  import Button from '$kernel/controls/Button.svelte'
  import { activateLicense, LICENSE_KEY_PATTERN, type AuthResult } from '../bridge/auth'
  import { getCompanyDisplayName } from '../stores/divisions.svelte'

  let { onActivated }: { onActivated: (result: AuthResult) => void } = $props()

  let key = $state('')
  let activating = $state(false)
  let error = $state('')

  // Auto-format as PH-<ROLE>-<SERIAL> while typing.
  function format(raw: string): string {
    const s = raw.toUpperCase().replace(/[^A-Z0-9]/g, '')
    const parts = [s.slice(0, 2)]
    if (s.length > 2) parts.push(s.slice(2, 5))
    if (s.length > 5) parts.push(s.slice(5, 11))
    return parts.filter(Boolean).join('-')
  }

  const valid = $derived(LICENSE_KEY_PATTERN.test(key))

  async function activate() {
    if (!valid || activating) return
    activating = true
    error = ''
    try {
      const result = await activateLicense(key)
      if (result.ok) onActivated(result)
      else error = result.message || 'Activation failed.'
    } catch (e) {
      error = e instanceof Error ? e.message : String(e)
    } finally {
      activating = false
    }
  }
</script>

<div class="k-auth">
  <div class="k-auth-card">
    <span class="k-auth-brand">{getCompanyDisplayName()}</span>
    <h1 class="k-auth-title">Activate this device</h1>
    <p class="k-auth-sub">Enter your license key to get started.</p>

    <input
      class="k-input k-auth-input"
      value={key}
      oninput={(e) => (key = format(e.currentTarget.value))}
      onkeydown={(e) => e.key === 'Enter' && activate()}
      placeholder="PH-XXX-YYYYYY"
      aria-label="License key"
      autocomplete="off"
      spellcheck="false"
    />

    {#if error}
      <div class="k-auth-error">{error}</div>
    {/if}

    <Button variant="primary" disabled={!valid || activating} onclick={activate}>
      {activating ? 'Activating…' : 'Activate'}
    </Button>
  </div>
</div>

<style>
  .k-auth {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    min-height: 0;
    padding: var(--k-space-lg);
    background: var(--bg-base);
  }
  .k-auth-card {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-sm);
    width: 100%;
    max-width: 360px;
    padding: var(--k-space-xl);
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius);
  }
  .k-auth-brand {
    font-family: var(--font-display);
    font-weight: 700;
    font-size: calc(13px * var(--ui-font-scale));
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--text-secondary);
  }
  .k-auth-title {
    font-family: var(--font-display);
    font-size: var(--page-title-size);
    font-weight: var(--page-title-weight);
  }
  .k-auth-sub {
    font-size: var(--meta-size);
    color: var(--text-secondary);
    margin-bottom: var(--k-space-sm);
  }
  .k-auth-input {
    font-family: var(--font-numeric);
    letter-spacing: 0.08em;
  }
  .k-auth-error {
    font-size: var(--meta-size);
    color: var(--k-tone-danger-fg);
    overflow-wrap: break-word;
  }
</style>
