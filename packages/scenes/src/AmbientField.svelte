<script lang="ts">
  /**
   * AmbientField — canvas-2D ambient background for dashboards and ceremonies.
   *
   * N slow-drifting points, each following a quaternion walk trajectory,
   * connected by faint lines when near (constellation feel).
   * Colors read from computed CSS custom properties — never hardcoded.
   *
   * Constitution §0: this component lives at the EDGES only (login, dashboards).
   * Production defaults whisper. `intensity` is for ceremonies and demo moments.
   *
   * Props:
   *   density        - Point count factor (1 = ~40 points, max ~200). Default: 1.
   *   seed           - Seed for initial positions (default 'ambient'). Makes it
   *                    deterministic for screenshots and SSR static frames.
   *   connectionDist - Max distance (px) to draw a connection line. Default: 120.
   *   paused         - Manually pause animation. Default: false.
   *   intensity      - 0..1 master dial. Default 0.35 = calm whisper per §0.
   *                    Scales canvas opacity, particle opacity, dot radius, line
   *                    width, line alpha, and color. At > 0.5, accent dots appear.
   *                    Use high values (0.75–1.0) only for ceremonies and demos.
   *   parallax       - When true (and not prefers-reduced-motion): pointer position
   *                    shifts particles by depth, adding 3-D weight. Default false.
   */

  import { onMount } from 'svelte';
  import { seededRng, rngRange } from './rng.js';
  import { fromAxisAngle, normalize, multiply, type Quat, QUAT_IDENTITY } from '@asymmflow/motion';

  interface Props {
    density?: number;
    seed?: string;
    connectionDist?: number;
    paused?: boolean;
    intensity?: number;
    parallax?: boolean;
    class?: string;
  }

  let {
    density = 1,
    seed = 'ambient',
    connectionDist = 120,
    paused = false,
    intensity = 0.35,
    parallax = false,
    class: className = '',
  }: Props = $props();

  let canvasEl: HTMLCanvasElement | undefined = $state();
  let running = $state(false);

  // Reduced motion: static frame only, parallax fully disabled
  const prefersReduced =
    typeof window !== 'undefined'
      ? window.matchMedia('(prefers-reduced-motion: reduce)').matches
      : false;

  // ── Particle system ──────────────────────────────────────────────────────

  interface Particle {
    x: number;
    y: number;
    // velocity (px/frame at 30fps)
    vx: number;
    vy: number;
    // orientation on S³ — used to modulate velocity direction over time
    q: Quat;
    // rotation axis for the velocity walk
    ax: number;
    ay: number;
    az: number;
    // base opacity — clamped below §3 max
    opacity: number;
    // depth factor for parallax [0.2, 1.0]
    depth: number;
  }

  let particles: Particle[] = [];
  let lastFrame = 0;
  let raf = 0;

  function initParticles(w: number, h: number) {
    const rng = seededRng(seed);
    const count = Math.round(Math.max(12, Math.min(200, density * 40)));
    particles = [];
    for (let i = 0; i < count; i++) {
      const ax = rngRange(rng, -1, 1);
      const ay = rngRange(rng, -1, 1);
      const az = rngRange(rng, -1, 1);
      particles.push({
        x: rngRange(rng, 0, w),
        y: rngRange(rng, 0, h),
        vx: rngRange(rng, -0.4, 0.4),
        vy: rngRange(rng, -0.4, 0.4),
        q: { ...QUAT_IDENTITY },
        ax,
        ay,
        az,
        // At intensity=0.35 this gives [0.15, 0.4]; at 1 the lerp in draw reaches [0.35, 0.75]
        opacity: rngRange(rng, 0.15, 0.4),
        depth: rngRange(rng, 0.2, 1.0),
      });
    }
  }

  interface Colors {
    fg: string;
    fgStrong: string;
    accent: string;
  }

  function readColors(canvas: HTMLCanvasElement): Colors {
    const cs = getComputedStyle(canvas);
    const border = cs.getPropertyValue('--af-border').trim() || 'rgba(180,200,180,0.5)';
    const borderStrong = cs.getPropertyValue('--af-border-strong').trim() || 'rgba(42,117,50,0.28)';
    const accent = cs.getPropertyValue('--af-accent').trim() || 'rgba(42, 117, 50, 0.10)';
    return { fg: border, fgStrong: borderStrong, accent };
  }

  // Throttle to ~30fps (33ms between frames) to be ambient, not demanding.
  const FRAME_INTERVAL = 33;

  /** Clamp-safe linear interpolation. */
  function lerp(a: number, b: number, t: number): number {
    return a + (b - a) * Math.max(0, Math.min(1, t));
  }

  // Parallax state — smoothed pointer offset (never mutates particle positions)
  let pointerTargetX = 0;
  let pointerTargetY = 0;
  let pointerX = 0;
  let pointerY = 0;
  const PARALLAX_EASE = 0.06;
  const PARALLAX_SCALE = 0.03;
  const PARALLAX_CAP = 28; // px

  function draw(now: number, canvas: HTMLCanvasElement) {
    if (now - lastFrame < FRAME_INTERVAL) {
      raf = requestAnimationFrame((t) => draw(t, canvas));
      return;
    }
    lastFrame = now;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    const w = canvas.width / window.devicePixelRatio;
    const h = canvas.height / window.devicePixelRatio;

    ctx.clearRect(0, 0, canvas.width, canvas.height);

    const { fg, fgStrong, accent } = readColors(canvas);

    // Smooth parallax pointer toward target
    if (parallax && !prefersReduced) {
      pointerX += (pointerTargetX - pointerX) * PARALLAX_EASE;
      pointerY += (pointerTargetY - pointerY) * PARALLAX_EASE;
    }

    const centerX = w / 2;
    const centerY = h / 2;

    // Intensity-derived scales
    const dotRadius = lerp(1.3, 2.6, intensity);
    const lineWidth = lerp(0.8, 1.4, intensity);
    const lineAlphaMult = lerp(0.7, 1.0, intensity);
    // Opacity range: at intensity=0, min=0.15, max=0.4; at intensity=1, min=0.35, max=0.75
    const opacityMin = lerp(0.15, 0.35, intensity);
    const opacityMax = lerp(0.40, 0.75, intensity);
    const useAccent = intensity > 0.5;

    // Color mixing: lerp between fg and fgStrong by intensity
    // We blend by drawing in fg always, then overlay fgStrong at intensity fraction
    // Simple approach: pick one color per intensity level (canvas can't mix CSS var strings)
    const dotColor = intensity >= 0.5 ? fgStrong : fg;
    const lineColor = intensity >= 0.5 ? fgStrong : fg;

    // Update particle physics
    for (let i = 0; i < particles.length; i++) {
      const p = particles[i];

      if (!prefersReduced) {
        const rot = fromAxisAngle(p.ax, p.ay, p.az, 0.003);
        p.q = normalize(multiply(p.q, rot));

        p.vx += p.q.x * 0.008;
        p.vy += p.q.y * 0.008;

        const speed = Math.hypot(p.vx, p.vy);
        if (speed > 0.55) {
          p.vx = (p.vx / speed) * 0.55;
          p.vy = (p.vy / speed) * 0.55;
        }

        p.x = ((p.x + p.vx + w) % w);
        p.y = ((p.y + p.vy + h) % h);
      }
    }

    // Compute parallax offset per particle at draw time (never mutate positions)
    const getDrawX = (p: Particle): number => {
      if (!parallax || prefersReduced) return p.x;
      const raw = (pointerX - centerX) * p.depth * PARALLAX_SCALE * intensity;
      return p.x + Math.max(-PARALLAX_CAP, Math.min(PARALLAX_CAP, raw));
    };
    const getDrawY = (p: Particle): number => {
      if (!parallax || prefersReduced) return p.y;
      const raw = (pointerY - centerY) * p.depth * PARALLAX_SCALE * intensity;
      return p.y + Math.max(-PARALLAX_CAP, Math.min(PARALLAX_CAP, raw));
    };

    // Draw connections
    const distSq = connectionDist * connectionDist;
    ctx.lineWidth = lineWidth;
    ctx.strokeStyle = lineColor;
    for (let i = 0; i < particles.length; i++) {
      for (let j = i + 1; j < particles.length; j++) {
        const dx = particles[i].x - particles[j].x;
        const dy = particles[i].y - particles[j].y;
        const d2 = dx * dx + dy * dy;
        if (d2 < distSq) {
          const proximity = 1 - d2 / distSq;
          // Map stored opacity to the intensity-driven range
          const opI = lerp(opacityMin, opacityMax, (particles[i].opacity - 0.15) / 0.25);
          const opJ = lerp(opacityMin, opacityMax, (particles[j].opacity - 0.15) / 0.25);
          const alpha = proximity * Math.min(opI, opJ) * lineAlphaMult;
          ctx.globalAlpha = Math.max(0, Math.min(1, alpha));
          ctx.beginPath();
          ctx.moveTo(getDrawX(particles[i]), getDrawY(particles[i]));
          ctx.lineTo(getDrawX(particles[j]), getDrawY(particles[j]));
          ctx.stroke();
        }
      }
    }

    // Draw dots
    for (let i = 0; i < particles.length; i++) {
      const p = particles[i];
      const mappedOpacity = lerp(opacityMin, opacityMax, (p.opacity - 0.15) / 0.25);
      const isAccentDot = useAccent && i % 5 === 0;

      ctx.globalAlpha = Math.max(0, Math.min(1, mappedOpacity));
      ctx.fillStyle = isAccentDot ? accent : dotColor;
      ctx.beginPath();
      ctx.arc(getDrawX(p), getDrawY(p), dotRadius, 0, Math.PI * 2);
      ctx.fill();
    }

    ctx.globalAlpha = 1;

    if (!paused && !prefersReduced) {
      raf = requestAnimationFrame((t) => draw(t, canvas));
      running = true;
    } else {
      running = false;
    }
  }

  function startLoop(canvas: HTMLCanvasElement) {
    cancelAnimationFrame(raf);
    if (prefersReduced) {
      draw(performance.now(), canvas);
      return;
    }
    running = true;
    raf = requestAnimationFrame((t) => draw(t, canvas));
  }

  function stopLoop() {
    cancelAnimationFrame(raf);
    running = false;
  }

  // ── Resize handling ──────────────────────────────────────────────────────

  let ro: ResizeObserver | null = null;

  onMount(() => {
    const canvas = canvasEl;
    if (!canvas) return;

    const handleResize = () => {
      const rect = canvas.getBoundingClientRect();
      canvas.width = rect.width * window.devicePixelRatio;
      canvas.height = rect.height * window.devicePixelRatio;
      const ctx = canvas.getContext('2d');
      if (ctx) ctx.scale(window.devicePixelRatio, window.devicePixelRatio);
      initParticles(rect.width, rect.height);
      if (!running) startLoop(canvas);
    };

    ro = new ResizeObserver(handleResize);
    ro.observe(canvas);
    handleResize();

    const visHandler = () => {
      if (document.hidden) {
        stopLoop();
      } else if (!paused && !prefersReduced) {
        startLoop(canvas);
      }
    };
    document.addEventListener('visibilitychange', visHandler);

    // Parallax pointer listener — attached/removed based on prop
    const onPointerMove = (e: PointerEvent) => {
      if (!canvas) return;
      const rect = canvas.getBoundingClientRect();
      pointerTargetX = e.clientX - rect.left;
      pointerTargetY = e.clientY - rect.top;
    };

    // We always add the listener but the draw loop ignores it when parallax=false
    window.addEventListener('pointermove', onPointerMove, { passive: true });

    return () => {
      stopLoop();
      ro?.disconnect();
      document.removeEventListener('visibilitychange', visHandler);
      window.removeEventListener('pointermove', onPointerMove);
    };
  });

  // React to paused prop changes
  $effect(() => {
    const canvas = canvasEl;
    if (!canvas) return;
    if (paused) {
      stopLoop();
    } else if (!running && !prefersReduced) {
      startLoop(canvas);
    }
  });

  // React to intensity/parallax changes: draw an immediate frame to reflect
  // the new values without re-seeding. The loop already reads intensity each frame.
  $effect(() => {
    // Touch these to register the dependency
    void intensity;
    void parallax;
    const canvas = canvasEl;
    if (!canvas || !running) return;
    // The running loop will pick them up next frame automatically.
    // If paused (static frame), redraw now.
    if (paused || prefersReduced) {
      draw(performance.now(), canvas);
    }
  });

  // Canvas opacity = lerp(0.4, 0.9, intensity) — applied via inline style
  const canvasOpacity = $derived(lerp(0.4, 0.9, intensity));
</script>

<!--
  Full-bleed canvas. The parent sizes it — AmbientField fills whatever space it gets.
  Position absolute/fixed is the parent's responsibility.
-->
<canvas
  bind:this={canvasEl}
  class="ambient-field {className}"
  aria-hidden="true"
  style:opacity={canvasOpacity}
></canvas>

<style>
  .ambient-field {
    display: block;
    width: 100%;
    height: 100%;
    pointer-events: none;
    /* opacity is set via inline style driven by the intensity prop */
  }
</style>
