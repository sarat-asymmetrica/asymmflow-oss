
<script lang="ts">
    /**
     * OmegaSwitch - Quaternion-aware toggle/regenerate button
     *
     * Features two animation modes:
     * 1. Lightning Flash (20% probability) - Mirzakhani Saddle Jump optimization
     * 2. Origami Fold (80% probability) - Collatz path visualization
     *
     * Dispatches 'regenerate' event on completion.
     *
     * @component
     */
    import { createEventDispatcher } from 'svelte';
    import { gsap } from 'gsap';

    /** Event interface for type-safe dispatch */
    interface OmegaSwitchEvents {
        regenerate: void;
    }

    const dispatch = createEventDispatcher<OmegaSwitchEvents>();

    

    

    

    
    interface Props {
        /** Button label text */
        label?: string;
        /** Shows Ω symbol */
        showOmega?: boolean;
        /** Disables button interaction */
        disabled?: boolean;
        /** ARIA label for accessibility */
        ariaLabel?: string;
        onregenerate?: () => void;
    }

    let {
        label = "Regenerate Layout",
        showOmega = true,
        disabled = false,
        ariaLabel = "Regenerate layout with quaternion transformation",
        onregenerate
    }: Props = $props();

    let container: HTMLDivElement = $state();
    let isFolding = $state(false);

    // φ-based animation durations
    const PHI = 1.618;
    const DURATION_SHORT = 1 / (PHI * PHI * PHI); // ≈ 0.236s
    const DURATION_MEDIUM = 1 / (PHI * PHI);      // ≈ 0.382s
    const DURATION_LONG = 1 / PHI;                // ≈ 0.618s

    /**
     * Triggers the fold/regenerate animation
     * Implements Collatz-inspired folding with Mirzakhani optimization
     */
    function triggerFold() {
        if (isFolding || disabled) return;
        isFolding = true;

        // Mirzakhani Saddle Jump optimization (20% probability)
        // When geodesic path is "too long", we quantum jump across the saddle
        const isOptimized = Math.random() > 0.8;

        if (isOptimized) {
            // Lightning Flash - instant saddle crossing
            gsap.to(container, {
                opacity: 0,
                scale: 1.2,
                duration: DURATION_SHORT,
                ease: "power4.in",
                onComplete: () => {
                    dispatch('regenerate');
                    onregenerate?.();
                    gsap.set(container, { scale: 0.8 });
                    gsap.to(container, {
                        opacity: 1,
                        scale: 1,
                        duration: DURATION_MEDIUM,
                        ease: "elastic.out(1, 0.5)",
                        onComplete: () => { isFolding = false; }
                    });
                }
            });
            return;
        }

        // Standard Origami Fold - Collatz path visualization
        const tl = gsap.timeline({
            onComplete: () => {
                dispatch('regenerate');
                onregenerate?.();
                // Unfold with φ-based spring
                gsap.to(container, {
                    rotationX: 0,
                    rotationY: 0,
                    scale: 1,
                    duration: DURATION_LONG,
                    ease: "back.out(1.7)",
                    onComplete: () => { isFolding = false; }
                });
            }
        });

        tl.to(container, {
            rotationX: 75,
            rotationY: 15,
            scale: 0.5,
            duration: DURATION_MEDIUM,
            ease: "power2.inOut",
            transformOrigin: "center bottom"
        }).to(container, {
            rotationX: 180, // Fold flat
            duration: DURATION_SHORT,
            ease: "power2.in"
        });
    }

    function handleKeydown(e: KeyboardEvent) {
        if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            triggerFold();
        }
    }
</script>

<div class="omega-container inline-block" bind:this={container}>
    <button
        class="omega-switch"
        onclick={triggerFold}
        onkeydown={handleKeydown}
        {disabled}
        aria-label={ariaLabel}
        aria-busy={isFolding}
        type="button"
    >
        {#if showOmega}
            <span class="omega-symbol" aria-hidden="true">Ω</span>
        {/if}
        <span>{label}</span>
    </button>
</div>

<style>
    .omega-container {
        perspective: 1000px;
    }

    .omega-switch {
        /* φ-based padding: 21px ≈ 13 × φ, 34px ≈ 21 × φ */
        padding: 13px 21px;
        background: var(--accent-color, #4f46e5);
        color: var(--bg-color, #ffffff);
        font-weight: 600;
        border-radius: 8px;
        border: none;
        box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1),
                    0 2px 4px -2px rgba(0, 0, 0, 0.1);
        transform-style: preserve-3d;
        backface-visibility: hidden;
        cursor: pointer;
        display: flex;
        align-items: center;
        gap: 8px;
        transition: background calc(var(--transition-duration, 0.3s)) ease,
                    transform calc(var(--transition-duration, 0.3s)) ease;
    }

    .omega-switch:hover:not(:disabled) {
        background: color-mix(in srgb, var(--accent-color, #4f46e5) 80%, black);
        transform: translateY(-2px);
    }

    .omega-switch:active:not(:disabled) {
        transform: translateY(0);
    }

    .omega-switch:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .omega-switch:focus-visible {
        outline: 2px solid var(--accent-color, #4f46e5);
        outline-offset: 2px;
    }

    .omega-symbol {
        font-size: 1.2em;
        font-weight: bold;
    }
</style>
