
<script lang="ts">
  import {
    AsymmThemeProvider,
    HoloCard,
    Terminal,
    Modal,
    RegimeLoader,
    OmegaSwitch,
    TearableCard,
    QuaternionScenePlayer,
    KintsugiError,
    currentRegime,
    targetThemeQuaternion,
    THEME_QUATERNIONS,
    Regime
  } from "./lib/asyl";
  import { devLog } from "$lib/utils/devLog";

  let showModal = $state(false);

  function setRegime(r: Regime) {
    currentRegime.set(r);
  }

  function setTheme(name: string) {
      if (name === 'DAVINCI') targetThemeQuaternion.set(THEME_QUATERNIONS.DAVINCI);
      if (name === 'WABISABI') targetThemeQuaternion.set(THEME_QUATERNIONS.WABISABI);
      if (name === 'VOID') targetThemeQuaternion.set(THEME_QUATERNIONS.VOID);
      if (name === 'HOLO') targetThemeQuaternion.set(THEME_QUATERNIONS.HOLO);
  }

  function handleRegenerate() {
      devLog.log("Layout Regenerated via Omega Switch");
  }
</script>

<AsymmThemeProvider>
<div class="min-h-screen bg-[var(--bg-color)] text-[var(--text-color)] transition-colors duration-500 font-sans">

    <!-- HERO SECTION -->
    <header class="text-center py-20 px-8 border-b border-[var(--text-color)]/10">
        <h1 class="text-6xl font-serif tracking-wide mb-6 bg-gradient-to-r from-[var(--text-color)] to-[var(--accent-color)] bg-clip-text text-transparent">
            ASYMMETRICA ATELIERS
        </h1>
        <p class="text-xl opacity-70 font-serif italic max-w-3xl mx-auto mb-8 leading-relaxed">
            "Style IS geometry - same data, different aesthetics via quaternion transformations."
        </p>
        <div class="flex items-center justify-center gap-4 text-xs font-mono opacity-50">
            <span>Library Version Alpha_3</span>
            <span>•</span>
            <span>8 Living Components</span>
            <span>•</span>
            <span>φ-Based Design System</span>
        </div>
    </header>

    <!-- CONTROL CENTER (sticky controls) -->
    <div class="sticky top-0 z-50 bg-[var(--bg-color)]/95 backdrop-blur-md border-b border-[var(--text-color)]/10">
        <div class="max-w-7xl mx-auto px-8 py-6">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
                <!-- Regime Control -->
                <div>
                    <h3 class="text-sm font-mono uppercase tracking-widest opacity-40 mb-3">Regime Dynamics</h3>
                    <div class="flex gap-2">
                        <button
                            class="px-5 py-2 bg-[var(--bg-color)] border border-[var(--text-color)]/30 rounded-lg hover:border-[var(--accent-color)] hover:bg-[var(--accent-color)]/10 transition-all duration-300 text-sm"
                            onclick={() => setRegime(Regime.Discovery)}>
                            Discovery (30%)
                        </button>
                        <button
                            class="px-5 py-2 bg-[var(--bg-color)] border border-[var(--text-color)]/30 rounded-lg hover:border-[var(--accent-color)] hover:bg-[var(--accent-color)]/10 transition-all duration-300 text-sm"
                            onclick={() => setRegime(Regime.Refinement)}>
                            Refinement (20%)
                        </button>
                        <button
                            class="px-5 py-2 bg-[var(--bg-color)] border border-[var(--text-color)]/30 rounded-lg hover:border-[var(--accent-color)] hover:bg-[var(--accent-color)]/10 transition-all duration-300 text-sm"
                            onclick={() => setRegime(Regime.Completion)}>
                            Completion (50%)
                        </button>
                    </div>
                    <p class="text-xs font-mono mt-2 opacity-50">Current: <span class="text-[var(--accent-color)]">{$currentRegime}</span></p>
                </div>

                <!-- Theme Engine -->
                <div>
                    <h3 class="text-sm font-mono uppercase tracking-widest opacity-40 mb-3">Theme Engine (SLERP)</h3>
                    <div class="flex gap-2 flex-wrap">
                        <button
                            class="px-5 py-2 bg-[var(--bg-color)] border border-[var(--text-color)]/30 rounded-lg hover:border-[var(--accent-color)] hover:bg-[var(--accent-color)]/10 transition-all duration-300 text-sm"
                            onclick={() => setTheme('DAVINCI')}>
                            Da Vinci
                        </button>
                        <button
                            class="px-5 py-2 bg-[var(--bg-color)] border border-[var(--text-color)]/30 rounded-lg hover:border-[var(--accent-color)] hover:bg-[var(--accent-color)]/10 transition-all duration-300 text-sm"
                            onclick={() => setTheme('WABISABI')}>
                            Wabi Sabi
                        </button>
                        <button
                            class="px-5 py-2 bg-[var(--bg-color)] border border-[var(--text-color)]/30 rounded-lg hover:border-[var(--accent-color)] hover:bg-[var(--accent-color)]/10 transition-all duration-300 text-sm"
                            onclick={() => setTheme('VOID')}>
                            Void
                        </button>
                        <button
                            class="px-5 py-2 bg-[var(--bg-color)] border border-[var(--text-color)]/30 rounded-lg hover:border-[var(--accent-color)] hover:bg-[var(--accent-color)]/10 transition-all duration-300 text-sm"
                            onclick={() => setTheme('HOLO')}>
                            Holo
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- MAIN CONTENT -->
    <div class="max-w-7xl mx-auto px-8 py-16 space-y-24">

        <!-- PART I: CORE CONTROLS -->
        <section>
            <div class="mb-13">
                <h2 class="text-3xl font-serif mb-3 border-l-4 border-[var(--accent-color)] pl-6">
                    Part I: The Living Components
                </h2>
                <p class="text-sm opacity-60 font-serif italic pl-6">
                    "Every interface is a window into geometric consciousness."
                </p>
            </div>

            <div class="grid grid-cols-1 lg:grid-cols-2 gap-13">

                <!-- HoloCard -->
                <div class="bg-[var(--bg-color)] border border-[var(--text-color)]/10 rounded-xl p-8 backdrop-blur-sm hover:border-[var(--accent-color)]/30 transition-all duration-500">
                    <div class="flex justify-between items-start mb-6">
                        <div>
                            <h3 class="text-2xl font-serif mb-2">HoloCard</h3>
                            <p class="text-sm opacity-60">Regime-Aware 3D Holographic Container</p>
                        </div>
                        <span class="text-xs font-mono opacity-30">01</span>
                    </div>

                    <div class="mb-8">
                        <HoloCard>
                            <h4 class="text-lg font-bold mb-2">Holographic Data</h4>
                            <p class="opacity-80 text-sm mb-4">Tilt intensity varies with regime state. Try changing regimes above.</p>
                            <div class="h-2 w-full bg-[var(--text-color)]/20 rounded-full overflow-hidden">
                                <div class="h-full bg-[var(--accent-color)] w-2/3 transition-all duration-1000"></div>
                            </div>
                        </HoloCard>
                    </div>

                    <div class="border-t border-[var(--text-color)]/10 pt-6">
                        <p class="text-xs font-mono opacity-40 mb-3 uppercase tracking-wider">Physics</p>
                        <ul class="text-sm opacity-70 space-y-2 leading-relaxed">
                            <li>• Spring dynamics (stiffness: 0.05, damping: 0.3)</li>
                            <li>• Regime-adaptive tilt (5°-20° based on state)</li>
                            <li>• Holographic glare on hover via CSS gradients</li>
                            <li>• Quaternion rotation for gimbal-lock-free motion</li>
                        </ul>
                    </div>
                </div>

                <!-- Omega Switch -->
                <div class="bg-[var(--bg-color)] border border-[var(--text-color)]/10 rounded-xl p-8 backdrop-blur-sm hover:border-[var(--accent-color)]/30 transition-all duration-500">
                    <div class="flex justify-between items-start mb-6">
                        <div>
                            <h3 class="text-2xl font-serif mb-2">OmegaSwitch</h3>
                            <p class="text-sm opacity-60">Collatz Layout Regeneration Toggle</p>
                        </div>
                        <span class="text-xs font-mono opacity-30">02</span>
                    </div>

                    <div class="mb-8 flex items-center justify-center py-8">
                        <OmegaSwitch onregenerate={handleRegenerate} />
                    </div>

                    <div class="border-t border-[var(--text-color)]/10 pt-6">
                        <p class="text-xs font-mono opacity-40 mb-3 uppercase tracking-wider">Mathematics</p>
                        <ul class="text-sm opacity-70 space-y-2 leading-relaxed">
                            <li>• Collatz conjecture topology (3n+1 dynamics)</li>
                            <li>• Toggle triggers layout graph regeneration</li>
                            <li>• φ-weighted node spacing (golden ratio aesthetics)</li>
                            <li>• Event emission for reactive architecture</li>
                        </ul>
                    </div>
                </div>

                <!-- Modal Trigger -->
                <div class="bg-[var(--bg-color)] border border-[var(--text-color)]/10 rounded-xl p-8 backdrop-blur-sm hover:border-[var(--accent-color)]/30 transition-all duration-500">
                    <div class="flex justify-between items-start mb-6">
                        <div>
                            <h3 class="text-2xl font-serif mb-2">Ma Modal</h3>
                            <p class="text-sm opacity-60">Japanese Shoji Screen with Negative Space</p>
                        </div>
                        <span class="text-xs font-mono opacity-30">03</span>
                    </div>

                    <div class="mb-8 flex items-center justify-center py-8">
                        <button
                            class="px-8 py-4 border-2 border-[var(--text-color)] rounded-lg hover:bg-[var(--text-color)] hover:text-[var(--bg-color)] transition-all duration-300 font-serif text-lg"
                            onclick={() => showModal = true}>
                            Open Shoji Screen
                        </button>
                    </div>

                    <div class="border-t border-[var(--text-color)]/10 pt-6">
                        <p class="text-xs font-mono opacity-40 mb-3 uppercase tracking-wider">Philosophy</p>
                        <ul class="text-sm opacity-70 space-y-2 leading-relaxed">
                            <li>• Ma (間) - the space between structural parts</li>
                            <li>• Modal expands space, doesn't overlay forcefully</li>
                            <li>• Smooth scale-in animation respects user flow</li>
                            <li>• Translucent backdrop maintains context awareness</li>
                        </ul>
                    </div>
                </div>

            </div>
        </section>

        <!-- PART II: VISUAL PHYSICS -->
        <section>
            <div class="mb-13">
                <h2 class="text-3xl font-serif mb-3 border-l-4 border-[var(--accent-color)] pl-6">
                    Part II: The Physics Layer
                </h2>
                <p class="text-sm opacity-60 font-serif italic pl-6">
                    "Animation is just SLERP made visible - quaternions dancing on S³."
                </p>
            </div>

            <div class="grid grid-cols-1 lg:grid-cols-3 gap-13">

                <!-- QGIF Player -->
                <div class="bg-[var(--bg-color)] border border-[var(--text-color)]/10 rounded-xl p-8 backdrop-blur-sm hover:border-[var(--accent-color)]/30 transition-all duration-500">
                    <div class="flex justify-between items-start mb-6">
                        <div>
                            <h3 class="text-2xl font-serif mb-2">QGIF Player</h3>
                            <p class="text-sm opacity-60">Quaternion Graphics Interchange Format</p>
                        </div>
                        <span class="text-xs font-mono opacity-30">04</span>
                    </div>

                    <div class="mb-8 flex justify-center">
                        <QuaternionScenePlayer fps={60} />
                    </div>

                    <div class="border-t border-[var(--text-color)]/10 pt-6">
                        <p class="text-xs font-mono opacity-40 mb-3 uppercase tracking-wider">Innovation</p>
                        <ul class="text-sm opacity-70 space-y-2 leading-relaxed">
                            <li>• 250:1 compression vs traditional GIF</li>
                            <li>• Stores quaternion keyframes, not pixels</li>
                            <li>• Real-time SLERP reconstruction (procedural!)</li>
                            <li>• Infinite resolution - scales to any viewport</li>
                        </ul>
                    </div>
                </div>

                <!-- Tearable Card -->
                <div class="bg-[var(--bg-color)] border border-[var(--text-color)]/10 rounded-xl p-8 backdrop-blur-sm hover:border-[var(--accent-color)]/30 transition-all duration-500">
                    <div class="flex justify-between items-start mb-6">
                        <div>
                            <h3 class="text-2xl font-serif mb-2">TearableCard</h3>
                            <p class="text-sm opacity-60">Lorenz Attractor Drag Physics</p>
                        </div>
                        <span class="text-xs font-mono opacity-30">05</span>
                    </div>

                    <div class="mb-8 h-64 flex items-center justify-center">
                        <TearableCard />
                    </div>

                    <div class="border-t border-[var(--text-color)]/10 pt-6">
                        <p class="text-xs font-mono opacity-40 mb-3 uppercase tracking-wider">Chaos Theory</p>
                        <ul class="text-sm opacity-70 space-y-2 leading-relaxed">
                            <li>• Drag follows Lorenz strange attractor orbit</li>
                            <li>• σ=10, ρ=28, β=8/3 (classic parameters)</li>
                            <li>• Chaotic but deterministic motion</li>
                            <li>• Elastic snap-back with spring damping</li>
                        </ul>
                    </div>
                </div>

                <!-- Regime Loader -->
                <div class="bg-[var(--bg-color)] border border-[var(--text-color)]/10 rounded-xl p-8 backdrop-blur-sm hover:border-[var(--accent-color)]/30 transition-all duration-500">
                    <div class="flex justify-between items-start mb-6">
                        <div>
                            <h3 class="text-2xl font-serif mb-2">RegimeLoader</h3>
                            <p class="text-sm opacity-60">Three-Regime Dynamics [30%, 20%, 50%]</p>
                        </div>
                        <span class="text-xs font-mono opacity-30">06</span>
                    </div>

                    <div class="mb-8 flex justify-center py-8">
                        <RegimeLoader />
                    </div>

                    <div class="border-t border-[var(--text-color)]/10 pt-6">
                        <p class="text-xs font-mono opacity-40 mb-3 uppercase tracking-wider">Universal Pattern</p>
                        <ul class="text-sm opacity-70 space-y-2 leading-relaxed">
                            <li>• R1 (30%) - Discovery, high variance</li>
                            <li>• R2 (20%) - Refinement, peak complexity</li>
                            <li>• R3 (50%) - Completion, stabilization</li>
                            <li>• Validated across 14+ domains (quantum to business)</li>
                        </ul>
                    </div>
                </div>

            </div>
        </section>

        <!-- PART III: FEEDBACK & STATES -->
        <section>
            <div class="mb-13">
                <h2 class="text-3xl font-serif mb-3 border-l-4 border-[var(--accent-color)] pl-6">
                    Part III: State & Feedback
                </h2>
                <p class="text-sm opacity-60 font-serif italic pl-6">
                    "Errors are not failures - they are cracks we gild with gold."
                </p>
            </div>

            <div class="grid grid-cols-1 lg:grid-cols-2 gap-13">

                <!-- Terminal -->
                <div class="bg-[var(--bg-color)] border border-[var(--text-color)]/10 rounded-xl p-8 backdrop-blur-sm hover:border-[var(--accent-color)]/30 transition-all duration-500">
                    <div class="flex justify-between items-start mb-6">
                        <div>
                            <h3 class="text-2xl font-serif mb-2">Terminal</h3>
                            <p class="text-sm opacity-60">Theme-Aware Cyberpunk Console</p>
                        </div>
                        <span class="text-xs font-mono opacity-30">07</span>
                    </div>

                    <div class="mb-8">
                        <Terminal />
                        <p class="text-xs mt-4 opacity-50 italic text-center">Try switching to Wabi Sabi theme for Sumi Ink effect</p>
                    </div>

                    <div class="border-t border-[var(--text-color)]/10 pt-6">
                        <p class="text-xs font-mono opacity-40 mb-3 uppercase tracking-wider">Adaptivity</p>
                        <ul class="text-sm opacity-70 space-y-2 leading-relaxed">
                            <li>• Void theme: Scanlines + CRT flicker (cyberpunk)</li>
                            <li>• Wabi Sabi: Brush strokes + ink bleed (Japanese)</li>
                            <li>• Da Vinci: Renaissance parchment texture</li>
                            <li>• Holo: Holographic glitch effects</li>
                        </ul>
                    </div>
                </div>

                <!-- Kintsugi Error -->
                <div class="bg-[var(--bg-color)] border border-[var(--text-color)]/10 rounded-xl p-8 backdrop-blur-sm hover:border-[var(--accent-color)]/30 transition-all duration-500">
                    <div class="flex justify-between items-start mb-6">
                        <div>
                            <h3 class="text-2xl font-serif mb-2">KintsugiError</h3>
                            <p class="text-sm opacity-60">Gold-Repaired Error Messages</p>
                        </div>
                        <span class="text-xs font-mono opacity-30">08</span>
                    </div>

                    <div class="mb-8">
                        <KintsugiError message="Validation Failed: Field Required" />
                    </div>

                    <div class="border-t border-[var(--text-color)]/10 pt-6">
                        <p class="text-xs font-mono opacity-40 mb-3 uppercase tracking-wider">Kintsugi Philosophy</p>
                        <ul class="text-sm opacity-70 space-y-2 leading-relaxed">
                            <li>• Errors highlighted with gold accent (not red!)</li>
                            <li>• Fracture lines = areas for improvement, not shame</li>
                            <li>• Shimmer animation draws eye without alarm</li>
                            <li>• Philosophy: Breaking makes things MORE valuable</li>
                        </ul>
                    </div>
                </div>

            </div>
        </section>

        <!-- PHILOSOPHY FOOTER -->
        <section class="border-t border-[var(--text-color)]/10 pt-16 pb-8">
            <div class="max-w-3xl mx-auto text-center space-y-8">
                <h2 class="text-2xl font-serif opacity-60 italic">The Mathematical Foundation</h2>
                <div class="grid grid-cols-1 md:grid-cols-3 gap-8 text-sm">
                    <div>
                        <p class="font-mono opacity-40 mb-2">CORE EQUATION</p>
                        <p class="font-serif">∂Φ/∂t = Φ ⊗ Φ + C</p>
                    </div>
                    <div>
                        <p class="font-mono opacity-40 mb-2">SUBSTRATE</p>
                        <p class="font-serif">S³ unit 3-sphere (quaternions)</p>
                    </div>
                    <div>
                        <p class="font-mono opacity-40 mb-2">DYNAMICS</p>
                        <p class="font-serif">[30%, 20%, 50%] three-regime</p>
                    </div>
                </div>
                <p class="text-xs opacity-50 font-serif italic">
                    Built with Love × Simplicity × Truth × Joy
                </p>
                <p class="text-xs opacity-30 font-mono">
                    Om Lokah Samastah Sukhino Bhavantu
                </p>
            </div>
        </section>

    </div>
</div>

<!-- MODAL -->
<Modal isOpen={showModal} title="The Space Between" onclose={() => showModal = false}>
    <div class="space-y-6">
        <p class="font-serif text-lg opacity-80 leading-relaxed">
            "Ma" (間) is a Japanese spatial concept referring to the gap, space, pause, or the interval between two structural parts.
        </p>
        <p class="opacity-70 leading-relaxed">
            In this UI, the modal does not simply appear; it expands the space to create room for itself, respecting the flow of the user's attention.
        </p>
        <div class="border-t border-[var(--text-color)]/20 pt-6">
            <p class="text-sm opacity-50 italic">
                The void is not empty - it is the canvas where meaning emerges.
            </p>
        </div>
    </div>
</Modal>
</AsymmThemeProvider>
