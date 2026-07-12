
<script lang="ts">
    /**
     * TearableCard - Constraint-violating draggable card component
     *
     * Features:
     * - Drag card within bounds
     * - Break constraint threshold (200px) triggers "tear"
     * - Torn card enters chaotic Lorenz-like orbit
     * - Physics-based spring return if released early
     *
     * @component
     */
    import { onMount, onDestroy } from 'svelte';
    import { gsap } from 'gsap';

    

    

    

    
    interface Props {
        /** Constraint violation threshold in pixels */
        tearThreshold?: number;
        /** ARIA label for accessibility */
        ariaLabel?: string;
        /** Title displayed on card */
        title?: string;
        /** Description text */
        description?: string;
    }

    let {
        tearThreshold = 200,
        ariaLabel = "Draggable card with constraint violation physics",
        title = "Tearable Component",
        description = "Drag me far away to break the constraints."
    }: Props = $props();

    let card: HTMLDivElement = $state();
    let container: HTMLDivElement = $state();
    let isTorn = $state(false);
    let orbitId: number;
    let angle = 0;

    // φ-based elastic spring for snap-back
    const PHI = 1.618;
    const ELASTIC_DURATION = 1 / PHI; // ≈ 0.618s

    function handleMouseDown(e: MouseEvent) {
        if (isTorn) return;

        const startX = e.clientX;
        const startY = e.clientY;
        const rect = card.getBoundingClientRect();

        function onMouseMove(e: MouseEvent) {
            const dx = e.clientX - startX;
            const dy = e.clientY - startY;

            card.style.transform = `translate(${dx}px, ${dy}px) rotate(${dx * 0.1}deg)`;

            // Constraint Violation Threshold - tear if dragged too far
            const distance = Math.hypot(dx, dy);
            if (distance > tearThreshold) {
                tearComponent();
                window.removeEventListener('mousemove', onMouseMove);
                window.removeEventListener('mouseup', onMouseUp);
            }
        }

        function onMouseUp() {
            window.removeEventListener('mousemove', onMouseMove);
            window.removeEventListener('mouseup', onMouseUp);
            if (!isTorn) {
                // Snap back with φ-based elastic spring
                gsap.to(card, {
                    x: 0,
                    y: 0,
                    rotation: 0,
                    duration: ELASTIC_DURATION,
                    ease: "elastic.out(1, 0.3)"
                });
            }
        }

        window.addEventListener('mousemove', onMouseMove);
        window.addEventListener('mouseup', onMouseUp);
    }

    function tearComponent() {
        isTorn = true;
        // Convert to fixed positioning to escape container
        const rect = card.getBoundingClientRect();
        card.style.position = 'fixed';
        card.style.left = `${rect.left}px`;
        card.style.top = `${rect.top}px`;
        card.style.width = `${rect.width}px`;
        card.style.zIndex = '9999';

        // Enter Controlled Chaos Orbit (Lorenz-like attractor)
        startOrbit();
    }

    function startOrbit() {
        // Strange Attractor Orbit around center of screen
        const centerX = window.innerWidth / 2;
        const centerY = window.innerHeight / 2;

        const animate = () => {
            if (!isTorn || !card) return;

            angle += 0.02;
            // Lorenz-like or Lissajous curve for chaotic but bounded motion
            const r = 200 + Math.sin(angle * 3) * 50;
            const x = centerX + Math.cos(angle) * r - card.offsetWidth / 2;
            const y = centerY + Math.sin(angle * 1.5) * r - card.offsetHeight / 2;

            card.style.left = `${x}px`;
            card.style.top = `${y}px`;
            card.style.transform = `rotate(${angle * 50}deg) scale(${1 + Math.sin(angle) * 0.1})`;

            orbitId = requestAnimationFrame(animate);
        };
        animate();
    }

    function handleKeydown(e: KeyboardEvent) {
        // Escape key resets the card
        if (e.key === 'Escape' && isTorn) {
            resetCard();
        }
    }

    function resetCard() {
        if (typeof window !== 'undefined') {
            cancelAnimationFrame(orbitId);
        }
        isTorn = false;
        card.style.position = '';
        card.style.left = '';
        card.style.top = '';
        card.style.width = '';
        card.style.zIndex = '';
        gsap.to(card, {
            x: 0,
            y: 0,
            rotation: 0,
            scale: 1,
            duration: ELASTIC_DURATION,
            ease: "elastic.out(1, 0.3)"
        });
    }

    onMount(() => {
        if (typeof window !== 'undefined') {
            window.addEventListener('keydown', handleKeydown);
        }
    });

    onDestroy(() => {
        if (typeof window !== 'undefined') {
            cancelAnimationFrame(orbitId);
            window.removeEventListener('keydown', handleKeydown);
        }
    });
</script>

<div
    bind:this={container}
    class="tearable-container"
    role="region"
    aria-label="Constraint visualization area"
>
    {#if !isTorn}
        <div class="constraint-label">
            Constraint Anchor
        </div>
    {/if}

    <div
        bind:this={card}
        class="tearable-card"
        onmousedown={handleMouseDown}
        role="button"
        tabindex="0"
        aria-label={ariaLabel}
    >
        <h3 class="card-title">{title}</h3>
        <p class="card-description">{description}</p>
        <div class="card-footer">
            Physics: Constraint Violation
        </div>
        {#if isTorn}
            <div class="chaos-badge">CHAOTIC ORBIT</div>
        {/if}
    </div>
</div>

<style>
    .tearable-container {
        position: relative;
        width: 100%;
        height: 16rem; /* 256px */
        border: 2px dashed color-mix(in srgb, var(--text-color, #d1d5db) 30%, transparent);
        border-radius: 8px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: color-mix(in srgb, var(--bg-color, #f9fafb) 95%, transparent);
    }

    .constraint-label {
        position: absolute;
        top: 8px;
        left: 8px;
        font-family: 'Courier New', monospace;
        font-size: 0.75rem;
        color: color-mix(in srgb, var(--text-color, #9ca3af) 50%, transparent);
        text-transform: uppercase;
        letter-spacing: 0.05em;
        pointer-events: none;
    }

    .tearable-card {
        position: relative;
        background: var(--bg-color, #ffffff);
        padding: 21px; /* φ-based: 13 × φ ≈ 21 */
        box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1),
                    0 4px 6px -4px rgba(0, 0, 0, 0.1);
        border-radius: 8px;
        cursor: grab;
        user-select: none;
        touch-action: none;
        max-width: 320px;
        transition: box-shadow var(--transition-duration, 0.3s) ease;
    }

    .tearable-card:hover {
        box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1),
                    0 8px 10px -6px rgba(0, 0, 0, 0.1);
    }

    .tearable-card:active {
        cursor: grabbing;
    }

    .tearable-card:focus-visible {
        outline: 2px solid var(--accent-color, #c5a059);
        outline-offset: 2px;
    }

    .card-title {
        font-weight: 700;
        font-size: 1.125rem;
        margin-bottom: 8px;
        color: var(--text-color, #111827);
    }

    .card-description {
        font-size: 0.875rem;
        color: color-mix(in srgb, var(--text-color, #6b7280) 70%, transparent);
        margin-bottom: 13px; /* φ-based */
    }

    .card-footer {
        margin-top: 13px; /* φ-based */
        font-size: 0.75rem;
        color: var(--accent-color, #6366f1);
        font-family: 'Courier New', monospace;
    }

    .chaos-badge {
        position: absolute;
        top: 8px;
        right: 8px;
        padding: 4px 8px;
        background: var(--danger-color, #ef4444);
        color: white;
        font-size: 0.625rem;
        font-weight: 700;
        border-radius: 4px;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        animation: pulse 1s cubic-bezier(0.4, 0, 0.6, 1) infinite;
    }

    @keyframes pulse {
        0%, 100% {
            opacity: 1;
        }
        50% {
            opacity: 0.5;
        }
    }
</style>
