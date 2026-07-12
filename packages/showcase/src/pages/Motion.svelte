<script lang="ts">
  import { MOTION_REGIMES, MOTION_STAGGER_MS } from '@asymmflow/tokens';

  // Replay keys — bumping re-mounts the demo block, re-triggering its animation.
  let exploreKey = $state(0);
  let stabilizeKey = $state(0);
  let staggerKey = $state(0);

  let stabilizeVisible = $state(true);

  const regimes = [
    {
      name: 'R1 · Explore',
      use: 'entrances, reveals, arrivals',
      token: 'explore',
      detail: `${MOTION_REGIMES.explore.duration}ms · decelerate — fast in, soft landing`,
    },
    {
      name: 'R2 · Optimize',
      use: 'hover, press, toggle, focus',
      token: 'optimize',
      detail: `${MOTION_REGIMES.optimize.duration}ms · tight symmetric`,
    },
    {
      name: 'R3 · Stabilize',
      use: 'exits, settles, confirmations',
      token: 'stabilize',
      detail: `${MOTION_REGIMES.stabilize.duration}ms · accelerate-out`,
    },
  ];
</script>

<div class="sections">
  <section>
    <h2 class="af-section-title">The three-regime motion policy</h2>
    <p class="intro">
      Every duration and easing in the system belongs to a regime — the same 30/20/50
      taxonomy the backend engines run on. Entrances explore, micro-interactions optimize,
      exits stabilize. Opacity and transform only; nothing exceeds 700ms without a
      ceremony justification; <code>prefers-reduced-motion</code> collapses everything.
    </p>

    <div class="regime-table card">
      {#each regimes as r}
        <div class="regime-row">
          <span class="regime-name">{r.name}</span>
          <span class="regime-use">{r.use}</span>
          <code class="af-meta">--af-motion-{r.token}-*</code>
          <span class="af-meta">{r.detail}</span>
        </div>
      {/each}
    </div>
  </section>

  <section>
    <h2 class="af-section-title">R1 · Explore — entrance</h2>
    <p class="intro">Fade + 16px rise on the explore curve. The arrival of new content.</p>
    <div class="demo-stage card">
      {#key exploreKey}
        <div class="demo-card explore-demo">
          <span class="af-label">New invoice</span>
          <span class="af-numeric demo-value">BHD 12,450.000</span>
        </div>
      {/key}
      <button class="replay" onclick={() => exploreKey++}>Replay</button>
    </div>
  </section>

  <section>
    <h2 class="af-section-title">R2 · Optimize — micro-interaction</h2>
    <p class="intro">
      Hover and press the card below. 140ms — felt as instant, never as animation.
    </p>
    <div class="demo-stage card">
      <button class="demo-card optimize-demo">
        <span class="af-label">Hover · press me</span>
        <span class="demo-value">The Lift + tint, optimize curve</span>
      </button>
    </div>
  </section>

  <section>
    <h2 class="af-section-title">R3 · Stabilize — exit &amp; settle</h2>
    <p class="intro">Dismissal accelerates out — the interface settles, it never lingers.</p>
    <div class="demo-stage card">
      {#key stabilizeKey}
        <div class="demo-card stabilize-demo" class:leaving={!stabilizeVisible}>
          <span class="af-label">Reconciled</span>
          <span class="demo-value">42 transactions matched</span>
        </div>
      {/key}
      <button
        class="replay"
        onclick={() => {
          if (stabilizeVisible) {
            stabilizeVisible = false;
          } else {
            stabilizeKey++;
            stabilizeVisible = true;
          }
        }}
      >
        {stabilizeVisible ? 'Dismiss' : 'Bring back'}
      </button>
    </div>
  </section>

  <section>
    <h2 class="af-section-title">Stagger — {MOTION_STAGGER_MS}ms sibling rhythm</h2>
    <p class="intro">
      Siblings arrive in sequence, not as a wall. The stagger interval is a token,
      so the rhythm is identical across every list, grid, and table in the system.
    </p>
    <div class="demo-stage card">
      {#key staggerKey}
        <div class="stagger-grid">
          {#each Array(6) as _, i}
            <div class="demo-card stagger-item" style:animation-delay="{i * MOTION_STAGGER_MS}ms">
              <span class="af-label">Row {i + 1}</span>
            </div>
          {/each}
        </div>
      {/key}
      <button class="replay" onclick={() => staggerKey++}>Replay</button>
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

  .regime-table {
    display: flex;
    flex-direction: column;
  }

  .regime-row {
    display: grid;
    grid-template-columns: minmax(110px, 130px) minmax(0, 1fr) minmax(0, 180px) minmax(0, 1fr);
    gap: var(--af-space-3);
    align-items: baseline;
    padding: var(--af-space-3) 0;
    border-bottom: 1px solid var(--af-border);
  }

  /* Let every cell shrink past its content so the token codes never spill. */
  .regime-row > * {
    min-width: 0;
  }

  .regime-row code {
    overflow-wrap: anywhere;
  }

  .regime-row:last-child {
    border-bottom: none;
  }

  .regime-name {
    font-weight: var(--af-weight-semibold);
    font-size: var(--af-text-sm);
  }

  .regime-use {
    font-size: var(--af-text-sm);
    color: var(--af-text-secondary);
  }

  .demo-stage {
    display: flex;
    align-items: center;
    gap: var(--af-space-4);
    min-height: 120px;
    background: var(--af-surface-raised);
  }

  .demo-card {
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    padding: var(--af-space-3) var(--af-space-4);
    display: flex;
    flex-direction: column;
    gap: var(--af-space-1);
    text-align: left;
  }

  .demo-value {
    font-size: var(--af-text-lg);
    font-weight: var(--af-weight-semibold);
  }

  .replay {
    margin-left: auto;
    border: 1px solid var(--af-border);
    background: var(--af-surface);
    color: var(--af-text);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-medium);
    padding: var(--af-space-2) var(--af-space-3);
    border-radius: var(--af-radius-sm);
    cursor: pointer;
    transition:
      border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .replay:hover {
    border-color: var(--af-border-strong);
    background: var(--af-surface-raised);
  }

  /* R1 — entrance */
  @keyframes af-explore-in {
    from {
      opacity: 0;
      transform: translateY(16px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .explore-demo {
    animation: af-explore-in var(--af-motion-explore-duration) var(--af-motion-explore-ease) both;
  }

  /* R2 — micro-interaction */
  .optimize-demo {
    cursor: pointer;
    transition:
      box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      transform var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .optimize-demo:hover {
    box-shadow: var(--af-shadow-lift);
    background: var(--af-surface-raised);
  }

  .optimize-demo:active {
    transform: scale(0.985);
    background: var(--af-tint);
  }

  /* R3 — exit */
  @keyframes af-stabilize-out {
    to {
      opacity: 0;
      transform: translateY(8px) scale(0.98);
    }
  }

  .stabilize-demo.leaving {
    animation: af-stabilize-out var(--af-motion-stabilize-duration) var(--af-motion-stabilize-ease) both;
  }

  /* Stagger */
  .stagger-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(120px, 1fr));
    gap: var(--af-space-3);
    flex: 1;
  }

  .stagger-item {
    animation: af-explore-in var(--af-motion-explore-duration) var(--af-motion-explore-ease) both;
  }
</style>
