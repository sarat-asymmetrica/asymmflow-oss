
<script lang="ts">
    import { run } from 'svelte/legacy';

    /**
     * Terminal - Cyberpunk-style terminal component
     *
     * Features theme-aware styling that adapts between:
     * - Wabi-Sabi (rice paper, sumi ink, minimal)
     * - Cyberpunk (dark background, neon scanlines, grid)
     *
     * @component
     */
    import { currentThemeQuaternion, THEME_QUATERNIONS } from "../core/theme";

    

    
    interface Props {
        /** ARIA label for accessibility */
        ariaLabel?: string;
        /** Terminal version label */
        version?: string;
        children?: import('svelte').Snippet;
    }

    let { ariaLabel = "Terminal display panel", version = "TERMINAL_V1.0", children }: Props = $props();

    // Determine if we are in Wabi-Sabi theme via quaternion dot product
    let isWabiSabi = $state(false);

    run(() => {
        const dist = $currentThemeQuaternion.dot(THEME_QUATERNIONS.WABISABI);
        // If dot product is close to 1 (or -1), we are near Wabi-Sabi quaternion
        isWabiSabi = Math.abs(dist) > 0.9;
    });
</script>

<div
    class="terminal-container"
    class:wabi-sabi={isWabiSabi}
    class:cyberpunk={!isWabiSabi}
    role="region"
    aria-label={ariaLabel}
>
    <!-- Scanline overlay (maintained in all themes) -->
    <div class="scanline" aria-hidden="true"></div>

    <!-- Content -->
    <div class="terminal-content">
        <!-- Header -->
        <div class="terminal-header">
            <div class="header-left">
                <span class="header-version">{version}</span>
            </div>
            <div class="header-right">
                <div class="status-indicator pulse"></div>
                <div class="status-indicator dim"></div>
                <div class="status-indicator faint"></div>
            </div>
        </div>

        <!-- Data Grid -->
        <div class="terminal-grid">
            <div class="grid-column">
                <div class="data-block">
                    <div class="data-label">
                        Memory_Heap
                    </div>
                    <div class="data-value">
                        0x7F_FF_A2
                    </div>
                </div>
            </div>

            <div class="grid-column border-left">
                <div class="data-block">
                    <div class="data-label">
                        PROCESS_LIST
                    </div>
                    <div class="process-list">
                        <div class="process-item">
                            <span>daemon.sys</span>
                            <span class="process-status">RUNNING</span>
                        </div>
                        <div class="process-item">
                            <span>kernel.ink</span>
                            <span class="process-status">SLEEP</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- Sumi Ink Effect for Wabi Sabi -->
        {#if isWabiSabi}
            <div class="sumi-ink" aria-hidden="true"></div>
        {/if}

        <!-- Slot for custom content -->
        <div class="terminal-slot">
            {@render children?.()}
        </div>
    </div>
</div>

<style>
    .terminal-container {
        position: relative;
        width: 100%;
        min-height: 16rem;
        overflow: hidden;
        border: 1px solid;
        border-radius: 4px;
        transition: all var(--transition-duration, 0.5s) ease;
    }

    /* Wabi-Sabi styling */
    .terminal-container.wabi-sabi {
        background: color-mix(in srgb, var(--bg-color, #fafaf9) 100%, transparent);
        border-color: color-mix(in srgb, var(--text-color, #d6d3d1) 30%, transparent);
        color: color-mix(in srgb, var(--text-color, #292524) 80%, transparent);
    }

    /* Cyberpunk styling */
    .terminal-container.cyberpunk {
        background: color-mix(in srgb, var(--bg-color, #111827) 100%, transparent);
        border-color: color-mix(in srgb, var(--text-color, #1f2937) 80%, transparent);
        color: color-mix(in srgb, var(--text-color, #9ca3af) 100%, transparent);
    }

    .scanline {
        position: absolute;
        inset: 0;
        pointer-events: none;
        opacity: 0.1;
        background: linear-gradient(transparent 50%, rgba(0,0,0,0.5) 50%);
        background-size: 100% 4px;
        z-index: 1;
    }

    .terminal-content {
        position: relative;
        z-index: 10;
        padding: 21px; /* φ-based: 13 × φ ≈ 21 */
        font-family: 'Courier New', monospace;
        font-size: 0.75rem;
    }

    .terminal-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 34px; /* φ-based: 21 × φ ≈ 34 */
        opacity: 0.6;
    }

    .header-version {
        text-transform: uppercase;
        letter-spacing: 0.05em;
    }

    .header-right {
        display: flex;
        gap: 4px;
    }

    .status-indicator {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        background: currentColor;
    }

    .status-indicator.pulse {
        opacity: 0.5;
        animation: pulse-dot 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
    }

    .status-indicator.dim {
        opacity: 0.3;
    }

    .status-indicator.faint {
        opacity: 0.1;
    }

    @keyframes pulse-dot {
        0%, 100% {
            opacity: 0.5;
        }
        50% {
            opacity: 1;
        }
    }

    .terminal-grid {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 34px; /* φ-based */
    }

    .grid-column {
        display: flex;
        flex-direction: column;
        gap: 13px; /* φ-based */
    }

    .grid-column.border-left {
        border-left: 1px solid;
        padding-left: 21px; /* φ-based */
    }

    .wabi-sabi .grid-column.border-left {
        border-color: color-mix(in srgb, var(--text-color, #d6d3d1) 30%, transparent);
    }

    .cyberpunk .grid-column.border-left {
        border-color: color-mix(in srgb, var(--text-color, #1f2937) 80%, transparent);
    }

    .data-label {
        margin-bottom: 4px;
        text-transform: uppercase;
        letter-spacing: 0.1em;
        opacity: 0.5;
        font-size: 0.625rem;
    }

    .data-value {
        font-size: 1.25rem;
        font-weight: 700;
        opacity: 0.9;
    }

    .process-list {
        display: flex;
        flex-direction: column;
        gap: 4px;
        opacity: 0.8;
    }

    .process-item {
        display: flex;
        justify-content: space-between;
        font-size: 0.75rem;
    }

    .process-status {
        opacity: 0.5;
    }

    .sumi-ink {
        position: absolute;
        bottom: 0;
        right: 0;
        width: 8rem;
        height: 8rem;
        background-image: url('data:image/svg+xml;utf8,<svg viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg"><path d="M50 50 Q 60 20 80 40 T 90 90" fill="none" stroke="black" stroke-width="20" filter="url(%23blur)"/><defs><filter id="blur"><feGaussianBlur in="SourceGraphic" stdDeviation="5"/></filter></defs></svg>');
        background-size: contain;
        background-repeat: no-repeat;
        opacity: 0.2;
        pointer-events: none;
        mix-blend-mode: multiply;
    }

    .terminal-slot {
        margin-top: 21px; /* φ-based */
    }

    /* Accessibility: Focus state */
    .terminal-container:focus-within {
        outline: 2px solid var(--accent-color, #c5a059);
        outline-offset: 2px;
    }
</style>
