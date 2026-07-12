
<script lang="ts">
    /**
     * KintsugiError - Gold-repaired error display component
     *
     * Embraces the Kintsugi (金継ぎ) philosophy:
     * "Errors are cracks in our application, made beautiful with gold"
     *
     * @component
     */

    

    

    

    
    interface Props {
        /** Error message to display */
        message?: string;
        /** Optional error code */
        code?: string | undefined;
        /** Error severity level */
        severity?: "error" | "warning" | "info";
        /** ARIA label for accessibility */
        ariaLabel?: string;
    }

    let {
        message = "Constraint Violation",
        code = undefined,
        severity = "error",
        ariaLabel = `${severity}: ${message}`
    }: Props = $props();

    // Severity-based styling
    const severityColors = {
        error: {
            bg: 'var(--danger-color, #fef2f2)',
            border: 'var(--danger-color, #fecaca)',
            text: 'var(--danger-color, #7f1d1d)',
            icon: ''
        },
        warning: {
            bg: 'color-mix(in srgb, var(--accent-color, #fef3c7) 30%, white)',
            border: 'var(--accent-color, #fde68a)',
            text: 'color-mix(in srgb, var(--text-color, #78350f) 80%, black)',
            icon: ''
        },
        info: {
            bg: 'color-mix(in srgb, var(--safe-color, #dbeafe) 30%, white)',
            border: 'var(--safe-color, #bfdbfe)',
            text: 'color-mix(in srgb, var(--text-color, #1e3a8a) 80%, black)',
            icon: ''
        }
    };

    let config = $derived(severityColors[severity]);
</script>

<div
    class="kintsugi-error"
    role="alert"
    aria-live="assertive"
    aria-label={ariaLabel}
    style="background: {config.bg}; border-color: {config.border}; color: {config.text};"
>
    <!-- Gold Kintsugi Crack Overlay -->
    <div class="kintsugi-crack" aria-hidden="true">
        <svg viewBox="0 0 100 100" preserveAspectRatio="none" class="crack-svg">
            <!-- Organic crack patterns (3 paths for complexity) -->
            <path
                d="M0,50 Q20,40 40,60 T80,40 T100,50"
                fill="none"
                stroke="var(--gold-color, gold)"
                stroke-width="2"
                class="crack-path"
            />
            <path
                d="M10,0 Q30,30 20,60 T40,100"
                fill="none"
                stroke="var(--gold-color, gold)"
                stroke-width="1.5"
                class="crack-path"
            />
            <path
                d="M80,0 Q70,40 85,70 T90,100"
                fill="none"
                stroke="var(--gold-color, gold)"
                stroke-width="1"
                class="crack-path"
            />
        </svg>
    </div>

    <!-- Error Content -->
    <div class="error-content">
        <span class="error-icon" aria-hidden="true">{config.icon}</span>
        <div class="error-details">
            {#if code}
                <span class="error-code">[{code}]</span>
            {/if}
            <span class="error-message">{message}</span>
        </div>
    </div>
</div>

<style>
    .kintsugi-error {
        position: relative;
        padding: 13px 21px; /* φ-based */
        border-radius: 8px;
        border: 1px solid;
        overflow: hidden;
        box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1),
                    0 2px 4px -1px rgba(0, 0, 0, 0.06);
        transition: all var(--transition-duration, 0.3s) ease;
    }

    .kintsugi-error:hover {
        box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1),
                    0 4px 6px -2px rgba(0, 0, 0, 0.05);
    }

    .kintsugi-crack {
        position: absolute;
        inset: 0;
        pointer-events: none;
        z-index: 1;
    }

    .crack-svg {
        width: 100%;
        height: 100%;
        opacity: 0.8;
    }

    .crack-path {
        filter: drop-shadow(0 0 2px rgba(255, 215, 0, 0.5));
        animation: crack-pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite;
    }

    @keyframes crack-pulse {
        0%, 100% {
            opacity: 0.8;
        }
        50% {
            opacity: 1;
        }
    }

    .error-content {
        position: relative;
        z-index: 10;
        display: flex;
        align-items: center;
        gap: 8px;
        font-family: 'Courier New', monospace;
        font-size: 0.875rem;
    }

    .error-icon {
        font-size: 1.25rem;
        flex-shrink: 0;
    }

    .error-details {
        display: flex;
        align-items: center;
        gap: 8px;
        flex-wrap: wrap;
    }

    .error-code {
        font-weight: 700;
        opacity: 0.7;
        font-size: 0.75rem;
    }

    .error-message {
        font-weight: 600;
    }

    /* Accessibility: Focus state */
    .kintsugi-error:focus-within {
        outline: 2px solid var(--accent-color, #c5a059);
        outline-offset: 2px;
    }
</style>
