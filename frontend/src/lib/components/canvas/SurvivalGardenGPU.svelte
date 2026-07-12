<script lang="ts">
  import { onMount } from "svelte";
  import { SimulateSurvivalGarden } from "../../../../wailsjs/go/main/App";
  import TimeMachine from "../TimeMachine.svelte";
  import { devLog } from "$lib/utils/devLog";

  interface Props {
    runwayMonths?: number;
    burnRate?: number;
  }

  let { runwayMonths = 6.2, burnRate = 11000 }: Props = $props();
  export const flowRate = 0.0; // External reference only

  // Sample expenses (in production, these come from backend data)
  const expenses = [
    { name: "Salaries", amount: 5000, weight: 0.45 },
    { name: "Rent", amount: 2500, weight: 0.23 },
    { name: "Operations", amount: 1800, weight: 0.16 },
    { name: "Marketing", amount: 1200, weight: 0.11 },
    { name: "Misc", amount: 500, weight: 0.05 }
  ];

  let canvas: HTMLCanvasElement = $state();
  let ctx: CanvasRenderingContext2D;
  let width: number;
  let height: number;
  let animationId: number;

  // GPU simulation states
  let gardenStates: any[] = [];
  let currentTimeMonth = $state(0); // Current position in time machine
  let maxMonths = 24; // Simulate 24 months into future

  // Animation state
  let waveOffset = 0;
  let particlePositions: Array<{x: number, y: number, velocity: number}> = [];

  // Current displayed state (interpolated from GPU simulation)
  let currentState = $state({
    waterLevel: 0.6,
    stoneHeights: [0.1, 0.15, 0.2, 0.25, 0.3],
    particleCount: 0,
    regime: 3,
    temperature: 0.0,
    turbulence: 0.2
  });


  function getWaterColor(temperature: number): string {
    // Temperature: 0.0 (cold blue) → 1.0 (hot red)
    if (temperature < 0.3) return "#15803d"; // Safe (green-blue)
    if (temperature < 0.7) return "#fbbf24"; // Warning (yellow)
    return "#ef4444"; // Danger (red)
  }

  function handleTimeChange(event: CustomEvent) {
    currentTimeMonth = event.detail.month;
    updateCurrentState();
  }

  function updateCurrentState() {
    if (gardenStates.length === 0) return;

    // Linear interpolation between states
    const lowMonth = Math.floor(currentTimeMonth);
    const highMonth = Math.ceil(currentTimeMonth);
    const t = currentTimeMonth - lowMonth;

    const low = gardenStates[Math.min(lowMonth, gardenStates.length - 1)];
    const high = gardenStates[Math.min(highMonth, gardenStates.length - 1)];

    if (!low || !high) return;

    currentState = {
      waterLevel: low.waterLevel * (1 - t) + high.waterLevel * t,
      stoneHeights: low.stoneHeights.map((h: number, i: number) =>
        h * (1 - t) + (high.stoneHeights[i] || h) * t
      ),
      particleCount: Math.round(
        low.particleCount * (1 - t) + high.particleCount * t
      ),
      regime: high.regime,
      temperature: low.temperature * (1 - t) + high.temperature * t,
      turbulence: low.turbulence * (1 - t) + high.turbulence * t
    };

    // Update particles based on new state
    updateParticles();
  }

  function updateParticles() {
    const targetCount = currentState.particleCount;
    const currentCount = particlePositions.length;

    if (currentCount < targetCount) {
      // Add particles
      for (let i = 0; i < targetCount - currentCount; i++) {
        particlePositions.push({
          x: Math.random() * width,
          y: height * (1 - currentState.waterLevel),
          velocity: Math.random() * 2 + 1
        });
      }
    } else if (currentCount > targetCount) {
      // Remove particles
      particlePositions = particlePositions.slice(0, targetCount);
    }
  }

  function resize() {
    if (!canvas) return;
    width = canvas.offsetWidth;
    height = canvas.offsetHeight;
    canvas.width = width;
    canvas.height = height;
  }

  function draw() {
    if (!ctx) return;
    ctx.clearRect(0, 0, width, height);

    // Draw Water with GPU-computed level
    ctx.fillStyle = waterColor;
    ctx.globalAlpha = 0.6;

    ctx.beginPath();
    ctx.moveTo(0, height);

    // Wave simulation with GPU turbulence
    const amplitude = 5 + currentState.turbulence * 10;
    for (let x = 0; x <= width; x += 10) {
      const y =
        height * (1 - currentState.waterLevel) +
        Math.sin(x * 0.02 + waveOffset) * amplitude;
      ctx.lineTo(x, y);
    }

    ctx.lineTo(width, height);
    ctx.closePath();
    ctx.fill();

    // Draw Stones (Expenses) - emerge from water as it drops
    currentState.stoneHeights.forEach((stoneHeight, i) => {
      const xPos = ((i + 1) / (currentState.stoneHeights.length + 1)) * width;
      const baseY = height; // Bottom of canvas
      const visibleHeight = stoneHeight * height * 0.5; // Max 50% of canvas height

      // Stone is positioned at bottom, grows upward
      const stoneTop = baseY - visibleHeight;

      ctx.fillStyle = "#475569"; // Stone gray
      ctx.globalAlpha = 1;

      // Draw stone
      ctx.beginPath();
      ctx.ellipse(
        xPos,
        stoneTop,
        20 + stoneHeight * 30, // Width based on height
        visibleHeight * 0.3, // Height
        0,
        0,
        Math.PI * 2
      );
      ctx.fill();

      // Label (expense name)
      if (visibleHeight > 20) {
        ctx.fillStyle = "#fafafa";
        ctx.font = "10px monospace";
        ctx.textAlign = "center";
        ctx.fillText(expenses[i]?.name || "", xPos, stoneTop - 10);
      }
    });

    // Draw Steam Particles (when water evaporating)
    ctx.fillStyle = "#ffffff";
    ctx.globalAlpha = 0.3;

    particlePositions.forEach((particle) => {
      ctx.beginPath();
      ctx.arc(particle.x, particle.y, 2, 0, Math.PI * 2);
      ctx.fill();

      // Update particle position (rise up)
      particle.y -= particle.velocity;

      // Reset if particle reaches top
      if (particle.y < 0) {
        particle.y = height * (1 - currentState.waterLevel);
        particle.x = Math.random() * width;
      }
    });

    waveOffset += 0.05;
    animationId = requestAnimationFrame(draw);
  }

  onMount(() => {
    ctx = canvas.getContext("2d")!;
    resize();
    window.addEventListener("resize", resize);

    void (async () => {
      // Run GPU-accelerated simulation via Asymmetrica.Runtime
      try {
        devLog.log('Starting GPU-accelerated simulation...');
        const startTime = performance.now();

        gardenStates = await SimulateSurvivalGarden(
          runwayMonths,
          burnRate,
          expenses,
          maxMonths
        );

        const elapsed = performance.now() - startTime;
        devLog.log(`GPU simulation completed in ${elapsed.toFixed(1)}ms`);
        devLog.log(`Generated ${gardenStates.length} garden states`);

        updateCurrentState();
      } catch (error) {
        devLog.warn("GPU simulation failed, using CPU fallback:", error);

        // Fallback: create default states (CPU-based)
        gardenStates = Array.from({ length: maxMonths + 1 }, (_, i) => ({
          waterLevel: Math.max(0, (runwayMonths - i) / runwayMonths),
          stoneHeights: [0.1, 0.15, 0.2, 0.25, 0.3],
          particleCount: 0,
          regime: 3,
          temperature: 0,
          turbulence: 0.2
        }));

        devLog.log('CPU fallback simulation completed');
      }

      draw();
    })();

    return () => {
      window.removeEventListener("resize", resize);
      cancelAnimationFrame(animationId);
    };
  });
  let waterColor = $derived(getWaterColor(currentState.temperature));
</script>

<div class="component-card h-full flex flex-col relative overflow-hidden">
  <div class="card-title flex justify-between items-center">
    <div>
      <span>Survival Garden</span>
      <span class="text-[10px] ml-2 opacity-50">GPU-ACCELERATED</span>
    </div>
    <!-- Regime Indicator -->
    <div class="flex gap-2 text-[10px]">
      {#if currentState.regime === 3}
        <span class="flex items-center gap-1"
          ><span class="w-2 h-2 rounded-full bg-green-600"></span> SAFE</span
        >
      {:else if currentState.regime === 2}
        <span class="flex items-center gap-1"
          ><span class="w-2 h-2 rounded-full bg-yellow-500"></span> WARNING</span
        >
      {:else}
        <span class="flex items-center gap-1"
          ><span class="w-2 h-2 rounded-full bg-red-500"></span> DANGER</span
        >
      {/if}
    </div>
  </div>

  <!-- Canvas Visualization -->
  <div class="relative w-full flex-grow min-h-[200px] mb-4">
    <canvas bind:this={canvas} class="w-full h-full"></canvas>
  </div>

  <!-- Metrics -->
  <div class="flex justify-between items-end">
    <div>
      <span class="metric-big" style="color: {waterColor}">
        {(runwayMonths - currentTimeMonth).toFixed(1)}
      </span>
      <span class="metric-label">Months at T+{currentTimeMonth.toFixed(1)}</span>
    </div>
    <div class="text-right">
      <span class="metric-label block">Burn Rate</span>
      <span class="font-mono font-bold text-lg"
        >{burnRate.toLocaleString()} BHD</span
      >
    </div>
  </div>

  <!-- THE TIME MACHINE -->
  <TimeMachine months={maxMonths} currentMonth={currentTimeMonth} on:timeChange={handleTimeChange} />

  <!-- Future Changed Indicator -->
  {#if currentTimeMonth > 0}
    <div class="text-center text-[10px] mt-2" style="color: var(--color-gold)">
      VIEWING FUTURE: T+{currentTimeMonth.toFixed(1)} MONTHS
    </div>
  {/if}
</div>
