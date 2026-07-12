<script>
    import { onMount, onDestroy } from "svelte";

    let cursorX = $state(0);
    let cursorY = $state(0);
    let targetX = 0;
    let targetY = 0;
    let visible = $state(false);
    let mode = $state("default"); // "default" | "text" | "magnetic"
    let magneticScale = $state(1);
    let magneticWidth = $state(6);
    let magneticHeight = $state(6);
    let magneticRadius = $state(50); // percent

    let animFrame;
    let magnetTarget = null; // The element we're magnetically attached to

    const LERP_FACTOR = 0.15; // Smooth follow (lower = more lag)
    const MAGNETIC_RANGE = 80; // px distance to start magnetic pull

    function lerp(a, b, t) {
        return a + (b - a) * t;
    }

    function handleMouseMove(e) {
        targetX = e.clientX;
        targetY = e.clientY;
        visible = true;

        const target = e.target;

        // Detect text elements for I-beam mode
        const isText = target.closest('p, span, h1, h2, h3, h4, h5, h6, label, .text-content, td');
        const isInput = target.closest('input, textarea, [contenteditable]');

        const isInteractive = target.closest(
            'button, a, [role="button"], .nav-link, .tab, .card, .btn, .quick-link'
        );

        if (isInput) {
            mode = "text";
            magnetTarget = null;
            magneticWidth = 2;
            magneticHeight = 20;
            magneticRadius = 2;
            magneticScale = 1;
        } else if (isText && !isInteractive) {
            mode = "text";
            magnetTarget = null;
            magneticWidth = 2;
            magneticHeight = 18;
            magneticRadius = 2;
            magneticScale = 1;
        } else if (isInteractive) {
            mode = "default";
            magnetTarget = null;
            magneticWidth = 6;
            magneticHeight = 6;
            magneticRadius = 50;
            magneticScale = 1;
        } else {
            mode = "default";
            magnetTarget = null;
            magneticWidth = 6;
            magneticHeight = 6;
            magneticRadius = 50;
            magneticScale = 1;
        }
    }

    function handleMouseLeave() {
        visible = false;
    }

    function animate() {
        // Smooth follow for default/text mode
        cursorX = lerp(cursorX, targetX, LERP_FACTOR);
        cursorY = lerp(cursorY, targetY, LERP_FACTOR);

        animFrame = requestAnimationFrame(animate);
    }

    function handleMouseDown() {
        magneticScale = 0.9;
        setTimeout(() => { magneticScale = 1; }, 150);
    }

    onMount(() => {
        document.addEventListener("mousemove", handleMouseMove);
        document.addEventListener("mouseleave", handleMouseLeave);
        document.addEventListener("mousedown", handleMouseDown);
        animFrame = requestAnimationFrame(animate);
    });

    onDestroy(() => {
        document.removeEventListener("mousemove", handleMouseMove);
        document.removeEventListener("mouseleave", handleMouseLeave);
        document.removeEventListener("mousedown", handleMouseDown);
        if (animFrame) cancelAnimationFrame(animFrame);
    });
</script>

<div
    class="cursor-follower"
    class:visible
    class:mode-text={mode === "text"}
    class:mode-magnetic={mode === "magnetic"}
    style="
        left: {cursorX}px;
        top: {cursorY}px;
        width: {magneticWidth}px;
        height: {magneticHeight}px;
        border-radius: {magneticRadius}px;
        transform: translate(-50%, -50%) scale({magneticScale});
    "
></div>

<style>
    .cursor-follower {
        position: fixed;
        pointer-events: none;
        z-index: 9999;
        background: #000;
        opacity: 0;
        transition:
            width 0.25s cubic-bezier(0.34, 1.56, 0.64, 1),
            height 0.25s cubic-bezier(0.34, 1.56, 0.64, 1),
            border-radius 0.2s ease,
            opacity 0.15s ease,
            background 0.2s ease;
        will-change: left, top, width, height;
    }

    .cursor-follower.visible {
        opacity: 1;
    }

    /* Default: small black dot */
    .cursor-follower:not(.mode-text):not(.mode-magnetic) {
        background: #000;
    }

    /* Text mode: thin I-beam line */
    .cursor-follower.mode-text {
        background: #1D1D1F;
        opacity: 0.7;
    }

</style>
