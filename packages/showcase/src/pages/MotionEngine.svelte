<script lang="ts">
  import {
    createGeodesicTween,
    slerpState,
    fromAxisAngle,
    multiply,
    slerp,
    toCssRotate,
    QUAT_IDENTITY,
    type Quat,
    type MotionRegime,
  } from '@asymmflow/motion';

  // ── Demo 1: interruptible geodesic chase ─────────────────────────────
  let chase = $state({ x: 40, y: 40, scale: 1 });
  let trail = $state<{ x: number; y: number }[]>([]);
  let straightTrail = $state<{ x: number; y: number }[]>([]);
  let straight = $state({ x: 40, y: 40 });

  const chaseTween = createGeodesicTween({ x: 40, y: 40, scale: 1 }, (s) => {
    chase = { ...s } as typeof chase;
    trail = [...trail.slice(-40), { x: s.x, y: s.y }];
  });

  // A plain straight-line tween for comparison (same easing, same duration).
  let straightRaf = 0;
  function straightTo(tx: number, ty: number) {
    cancelAnimationFrame(straightRaf);
    const from = { ...straight };
    const start = performance.now();
    const dur = 400;
    const ease = (x: number) => 1 - Math.pow(1 - x, 3);
    function frame(now: number) {
      const t = Math.min(1, (now - start) / dur);
      const e = ease(t);
      straight = { x: from.x + (tx - from.x) * e, y: from.y + (ty - from.y) * e };
      straightTrail = [...straightTrail.slice(-40), { ...straight }];
      if (t < 1) straightRaf = requestAnimationFrame(frame);
    }
    straightRaf = requestAnimationFrame(frame);
  }

  function onPlaygroundClick(e: MouseEvent) {
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const x = e.clientX - rect.left - 28;
    const y = e.clientY - rect.top - 28;
    chaseTween.to({ x, y, scale: 0.9 }, { regime: 'explore', onComplete: () => chaseTween.to({ scale: 1 }, { regime: 'stabilize' }) });
    straightTo(x, y);
  }

  // ── Demo 2: quaternion orientation slerp ─────────────────────────────
  let orientation = $state('rotate3d(0, 0, 1, 0deg)');
  let fromQ: Quat = QUAT_IDENTITY;
  let toQ: Quat = QUAT_IDENTITY;

  const orientTween = createGeodesicTween({ t: 0 }, (s) => {
    orientation = toCssRotate(slerp(fromQ, toQ, s.t));
  });

  const orientations: { label: string; q: Quat }[] = [
    { label: 'Front', q: QUAT_IDENTITY },
    { label: 'Iso', q: multiply(fromAxisAngle(1, 0, 0, -0.45), fromAxisAngle(0, 1, 0, 0.62)) },
    { label: 'Side', q: fromAxisAngle(0, 1, 0, 1.2) },
    { label: 'Tipped', q: multiply(fromAxisAngle(0, 0, 1, 0.35), fromAxisAngle(1, 0, 0, 0.5)) },
  ];

  function goOrientation(q: Quat) {
    fromQ = slerp(fromQ, toQ, orientTween.current.t);
    toQ = q;
    orientTween.jump({ t: 0 });
    orientTween.to({ t: 1 }, { regime: 'explore' });
  }

  // ── Demo 3: regime comparison ────────────────────────────────────────
  const regimes: MotionRegime[] = ['explore', 'optimize', 'stabilize', 'spring'];
  let lanes = $state<Record<string, number>>({ explore: 0, optimize: 0, stabilize: 0, spring: 0 });
  let lanesRight = $state(false);
  let trackW = $state(260);
  const laneTweens = regimes.map((r) =>
    createGeodesicTween({ x: 0 }, (s) => {
      lanes = { ...lanes, [r]: s.x };
    }),
  );

  function raceLanes() {
    lanesRight = !lanesRight;
    const x = lanesRight ? 100 : 0;
    regimes.forEach((r, i) => laneTweens[i].to({ x }, { regime: r }));
  }

  // Static curve preview of the geodesic vs linear path (drawn once).
  const arcPreview = $derived.by(() => {
    const pts: string[] = [];
    for (let i = 0; i <= 24; i++) {
      const s = slerpState({ x: 10, y: 90 }, { x: 150, y: 20 }, i / 24);
      pts.push(`${s.x},${s.y}`);
    }
    return pts.join(' ');
  });
</script>

<div class="sections">
  <section>
    <h2 class="af-section-title">The geodesic engine — AsymmFlow never snaps</h2>
    <p class="intro">
      A UI state is one point on a hypersphere; transitions travel the geodesic between
      points, so every property advances under a single shared parameter. Interrupting a
      transition starts a new geodesic <em>from wherever it currently is</em> — click the
      playground rapidly and watch: the dark card flows through redirects along subtly
      curved paths, properties always coherent. The ghost card is a conventional
      straight-line tween for comparison.
    </p>

    <div
      class="playground card"
      onclick={onPlaygroundClick}
      role="presentation"
    >
      <svg class="trails" aria-hidden="true">
        {#each straightTrail as p, i}
          <circle cx={p.x + 28} cy={p.y + 28} r="1.5" class="trail-dot trail-dot--straight" style:opacity={i / straightTrail.length * 0.5} />
        {/each}
        {#each trail as p, i}
          <circle cx={p.x + 28} cy={p.y + 28} r="1.5" class="trail-dot" style:opacity={i / trail.length * 0.8} />
        {/each}
      </svg>
      <div
        class="chaser chaser--ghost"
        style:transform="translate({straight.x}px, {straight.y}px)"
        aria-hidden="true"
      ></div>
      <div
        class="chaser"
        style:transform="translate({chase.x}px, {chase.y}px) scale({chase.scale})"
      >
        <span class="af-label chaser-label">PO-2026</span>
      </div>
      <span class="af-meta hint">click anywhere — repeatedly</span>
    </div>
  </section>

  <section>
    <h2 class="af-section-title">Quaternion orientation — SLERP on S³</h2>
    <p class="intro">
      True 4D: orientations interpolate along the shortest arc on the unit 3-sphere at
      constant angular velocity — no gimbal artifacts, no detours. This is the engine the
      scenes and glyph layers run on; here it drives a CSS 3D surface.
    </p>
    <div class="card orient-stage">
      <div class="orient-scene">
        <div class="orient-card" style:transform={orientation}>
          <div class="orient-face">
            <span class="af-label">Invoice</span>
            <span class="af-numeric orient-value">BHD 12,450.000</span>
            <span class="af-meta">Gulf Equipment Trading WLL</span>
          </div>
        </div>
      </div>
      <div class="orient-controls">
        {#each orientations as o}
          <button class="seg-btn" onclick={() => goOrientation(o.q)}>{o.label}</button>
        {/each}
      </div>
    </div>
  </section>

  <section>
    <h2 class="af-section-title">Regimes, raced</h2>
    <p class="intro">
      The same movement under each regime's duration and easing. One taxonomy, four
      temperaments — explore arrives soft, optimize is felt not seen, stabilize settles,
      spring is for earned moments only.
    </p>
    <div class="card lanes">
      {#each regimes as r}
        <div class="lane">
          <span class="af-label lane-label">{r}</span>
          <div class="lane-track" bind:clientWidth={trackW}>
            <div
              class="lane-dot"
              style:transform="translateX({(lanes[r] / 100) * Math.max(0, trackW - 28)}px)"
            ></div>
          </div>
        </div>
      {/each}
      <button class="seg-btn race-btn" onclick={raceLanes}>Race</button>
    </div>
  </section>

  <section>
    <h2 class="af-section-title">Why paths curve</h2>
    <p class="intro">
      Direction interpolates on the sphere while magnitude interpolates linearly — the
      projected path bows gently instead of cutting a straight chord. Subtle, organic,
      and mathematically inevitable.
    </p>
    <div class="card curve-card">
      <svg viewBox="0 0 160 110" class="curve-svg" aria-hidden="true">
        <line x1="10" y1="90" x2="150" y2="20" class="curve-chord" />
        <polyline points={arcPreview} class="curve-arc" />
        <circle cx="10" cy="90" r="3" class="curve-end" />
        <circle cx="150" cy="20" r="3" class="curve-end" />
      </svg>
      <div class="curve-legend">
        <span class="af-meta"><span class="legend-swatch legend-swatch--chord"></span> straight chord (conventional tween)</span>
        <span class="af-meta"><span class="legend-swatch legend-swatch--arc"></span> geodesic (AsymmFlow)</span>
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

  /* Demo 1 */
  .playground {
    position: relative;
    height: 320px;
    overflow: hidden;
    cursor: crosshair;
    background: var(--af-surface-raised);
  }

  .trails {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    pointer-events: none;
  }

  .trail-dot {
    fill: var(--af-accent);
  }

  .trail-dot--straight {
    fill: var(--af-text-muted);
  }

  .chaser {
    position: absolute;
    top: 0;
    left: 0;
    width: 56px;
    height: 56px;
    border-radius: var(--af-radius-md);
    background: var(--af-inverse-surface);
    color: var(--af-text-inverse);
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: var(--af-shadow-lift);
    will-change: transform;
    pointer-events: none;
  }

  .chaser--ghost {
    background: transparent;
    border: 1px dashed var(--af-border-strong);
    box-shadow: none;
  }

  .chaser-label {
    color: var(--af-text-inverse);
    font-size: 9px;
  }

  .hint {
    position: absolute;
    bottom: var(--af-space-2);
    right: var(--af-space-3);
  }

  /* Demo 2 */
  .orient-stage {
    display: flex;
    align-items: center;
    gap: var(--af-space-5);
    flex-wrap: wrap;
  }

  .orient-scene {
    perspective: 900px;
    width: 260px;
    height: 180px;
    display: grid;
    place-items: center;
  }

  .orient-card {
    transform-style: preserve-3d;
    will-change: transform;
  }

  .orient-face {
    width: 220px;
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    background: var(--af-surface);
    box-shadow: var(--af-shadow-overlay);
    padding: var(--af-space-4);
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
  }

  .orient-value {
    font-size: var(--af-text-xl);
    font-weight: var(--af-weight-semibold);
  }

  .orient-controls {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
  }

  .seg-btn {
    border: 1px solid var(--af-border);
    background: var(--af-surface);
    color: var(--af-text);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-medium);
    padding: var(--af-space-2) var(--af-space-4);
    border-radius: var(--af-radius-sm);
    cursor: pointer;
    transition:
      border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .seg-btn:hover {
    border-color: var(--af-border-strong);
    background: var(--af-surface-raised);
  }

  /* Demo 3 */
  .lanes {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-3);
  }

  .lane {
    display: grid;
    grid-template-columns: 90px 1fr;
    align-items: center;
    gap: var(--af-space-3);
  }

  .lane-track {
    position: relative;
    height: 28px;
    background: var(--af-surface-sunken);
    border-radius: var(--af-radius-pill);
  }

  .lane-dot {
    position: absolute;
    top: 4px;
    left: 4px;
    width: 20px;
    height: 20px;
    border-radius: 50%;
    background: var(--af-inverse-surface);
    will-change: transform;
  }

  .race-btn {
    align-self: flex-start;
  }

  /* Demo 4 */
  .curve-card {
    display: flex;
    align-items: center;
    gap: var(--af-space-5);
    flex-wrap: wrap;
  }

  .curve-svg {
    width: 320px;
    max-width: 100%;
  }

  .curve-chord {
    stroke: var(--af-text-muted);
    stroke-width: 1;
    stroke-dasharray: 4 4;
  }

  .curve-arc {
    fill: none;
    stroke: var(--af-accent);
    stroke-width: 2;
    stroke-linecap: round;
  }

  .curve-end {
    fill: var(--af-text);
  }

  .curve-legend {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
  }

  .legend-swatch {
    display: inline-block;
    width: 18px;
    height: 3px;
    border-radius: 2px;
    vertical-align: middle;
    margin-inline-end: 6px;
  }

  .legend-swatch--chord {
    background: var(--af-text-muted);
  }

  .legend-swatch--arc {
    background: var(--af-accent);
  }
</style>
