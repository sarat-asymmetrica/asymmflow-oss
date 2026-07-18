<script lang="ts">
  import PageShell from '$kernel/primitives/PageShell.svelte'
  import Stack from '$kernel/primitives/Stack.svelte'
  import Row from '$kernel/primitives/Row.svelte'
  import Card from '$kernel/primitives/Card.svelte'
  import Grid from '$kernel/primitives/Grid.svelte'
  import Scroll from '$kernel/primitives/Scroll.svelte'
  import { formatDate, formatMoney } from '$kernel/format'

  // Adversarial content is the DEFAULT demo data (KERNEL.md layout doctrine).
  const monsterName =
    'Interntional Establishment for Industrial & Petrochemical Instrumentation Services and General Trading (formerly Gulf Technical Calibration & Measurement Systems Company) W.L.L.'
  const rtlName = 'المؤسسة الدولية لخدمات الأجهزة الصناعية والبتروكيماوية والتجارة العامة ذ.م.م'
  const monsterAmount = 123456789012.345
  const unbrokenToken = 'INV-2026-Q3-0000000000000000000000000000000000000042'

  const cards = [
    { title: 'Ordinary card', body: 'Normal content, nothing scary.', amount: 1234.5 },
    { title: monsterName, body: 'A 170+ char legal name must wrap, never blow out.', amount: monsterAmount },
    { title: rtlName, body: 'RTL Arabic must render and wrap correctly.', amount: 42 },
    { title: unbrokenToken, body: 'Unbroken tokens must break, not overflow.', amount: 0.001 },
    { title: '', body: 'Empty title — the layout must not collapse.', amount: 0 },
  ]
</script>

<PageShell
  title="Kernel Lab — primitives under adversarial load"
  subtitle="If anything on this page overflows its container, a primitive has a bug."
>
  {#snippet actions()}
    <Row gap="sm">
      <button class="demo-btn">Action</button>
      <button class="demo-btn">{monsterName.slice(0, 40)}…</button>
    </Row>
  {/snippet}

  <Stack gap="lg">
    <Card padding="lg">
      <Stack gap="sm">
        <h2 class="demo-section-title">Format kernel (L2 — defined once)</h2>
        <Row gap="lg" wrap>
          <span>{formatDate(new Date())}</span>
          <span>{formatDate(null)}</span>
          <span class="demo-numeric">{formatMoney(monsterAmount)}</span>
          <span class="demo-numeric">{formatMoney(0.001)}</span>
          <span class="demo-numeric">{formatMoney(1500, 'USD')}</span>
        </Row>
      </Stack>
    </Card>

    <Grid min="260px" gap="md">
      {#each cards as card}
        <Card>
          <Stack gap="xs">
            <h3 class="demo-card-title">{card.title || '—'}</h3>
            <p class="demo-card-body">{card.body}</p>
            <span class="demo-numeric">{formatMoney(card.amount)}</span>
          </Stack>
        </Card>
      {/each}
    </Grid>

    <Card padding="none">
      <Scroll axis="x">
        <div class="demo-wide">
          A deliberately 3000px-wide strip — it must scroll inside this declared
          Scroll region and never widen the page. ({unbrokenToken})
        </div>
      </Scroll>
    </Card>
  </Stack>
</PageShell>

<style>
  /* Demo-only chrome. Screens in the real system won't have style blocks
   * like this (L1); the lab harness is kernel territory, so it may. */
  .demo-btn {
    font: inherit;
    padding: 6px 14px;
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius-sm);
    background: var(--surface);
    cursor: pointer;
    max-width: 320px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .demo-section-title {
    font-size: var(--section-title-size);
    font-weight: var(--section-title-weight);
  }
  .demo-card-title {
    font-size: var(--section-title-size);
    font-weight: var(--section-title-weight);
    overflow-wrap: anywhere;
  }
  .demo-card-body {
    color: var(--text-secondary);
    font-size: var(--table-text-size);
  }
  .demo-numeric {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
  }
  .demo-wide {
    width: 3000px;
    padding: var(--card-padding);
    white-space: nowrap;
  }
</style>
