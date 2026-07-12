<script lang="ts">
    let { warning = {
        severity: "medium", // "low" | "medium" | "high" | "critical"
        message: "Unusual pattern detected in customer behavior",
        details: "Customer payment velocity has decreased by 40% in last 2 weeks",
        suggestedActions: [
            "Contact customer for check-in",
            "Review recent orders for quality issues"
        ]
    } } = $props();

    let severity = $derived(warning.severity || "low");
    let message = $derived(warning.message || "");
    let details = $derived(warning.details || "");
    let suggestedActions = $derived(warning.suggestedActions || []);

    let expanded = $state(false);

    // Severity colors and icons
    function getSeverityColor(sev) {
        switch (sev.toLowerCase()) {
            case 'critical': return '#dc2626'; // dark red
            case 'high': return '#ef4444'; // red
            case 'medium': return '#fbbf24'; // yellow
            case 'low': return '#3b82f6'; // blue
            default: return '#57534e'; // gray
        }
    }

    function getSeverityIcon(sev) {
        return ''; // Removed emojis
    }

    function getSeverityLabel(sev) {
        return sev.toUpperCase();
    }
</script>

<div
    class="black-swan-alert"
    class:critical={severity === 'critical'}
    class:high={severity === 'high'}
    class:medium={severity === 'medium'}
    class:low={severity === 'low'}
    style="--severity-color: {getSeverityColor(severity)}"
>
    <div
        class="alert-header"
        role="button"
        tabindex="0"
        onclick={() => expanded = !expanded}
        onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), expanded = !expanded)}
        aria-expanded={expanded}
        aria-controls="alert-details"
    >
        <div class="header-left">
            <span class="severity-icon">{getSeverityIcon(severity)}</span>
            <span class="severity-label">{getSeverityLabel(severity)}</span>
            <span class="alert-message">{message}</span>
        </div>
        <button class="expand-btn" aria-label={expanded ? 'Collapse' : 'Expand'} tabindex="-1">
            {expanded ? '▼' : '▶'}
        </button>
    </div>

    {#if expanded}
        <div class="alert-body" id="alert-details">
            <!-- Details -->
            {#if details}
                <div class="details-section">
                    <div class="section-header">Details</div>
                    <p class="details-text">{details}</p>
                </div>
            {/if}

            <!-- Suggested Actions -->
            {#if suggestedActions.length > 0}
                <div class="actions-section">
                    <div class="section-header">Suggested Actions</div>
                    <ul class="actions-list">
                        {#each suggestedActions as action}
                            <li class="action-item">{action}</li>
                        {/each}
                    </ul>
                </div>
            {/if}
        </div>
    {/if}
</div>

<style>
    .black-swan-alert {
        background: rgba(255, 255, 255, 0.9);
        border-left: 4px solid var(--severity-color);
        border-radius: 6px;
        margin: 0.75rem 0;
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
        overflow: hidden;
        transition: all 0.2s ease;
    }

    .black-swan-alert:hover {
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    }

    /* Severity-specific backgrounds */
    .black-swan-alert.critical {
        background: linear-gradient(to right, rgba(220, 38, 38, 0.08), rgba(255, 255, 255, 0.9));
    }

    .black-swan-alert.high {
        background: linear-gradient(to right, rgba(239, 68, 68, 0.06), rgba(255, 255, 255, 0.9));
    }

    .black-swan-alert.medium {
        background: linear-gradient(to right, rgba(251, 191, 36, 0.06), rgba(255, 255, 255, 0.9));
    }

    .black-swan-alert.low {
        background: linear-gradient(to right, rgba(59, 130, 246, 0.06), rgba(255, 255, 255, 0.9));
    }

    /* Header */
    .alert-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 0.75rem 1rem;
        cursor: pointer;
        user-select: none;
    }

    .alert-header:hover {
        background: rgba(0, 0, 0, 0.02);
    }

    .header-left {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        flex: 1;
    }

    .severity-icon {
        font-size: 1.1rem;
        line-height: 1;
    }

    .severity-label {
        font-family: var(--font-mono), 'Courier New', monospace;
        font-size: 0.65rem;
        text-transform: uppercase;
        letter-spacing: 1px;
        color: var(--severity-color);
        font-weight: 700;
        padding: 0.2rem 0.4rem;
        background: rgba(0, 0, 0, 0.05);
        border-radius: 3px;
    }

    .alert-message {
        font-family: var(--font-serif), Georgia, serif;
        font-size: 0.85rem;
        color: var(--color-ink, #1c1c1c);
        font-weight: 500;
    }

    .expand-btn {
        background: none;
        border: none;
        color: var(--severity-color);
        font-size: 0.7rem;
        cursor: pointer;
        padding: 0.25rem 0.5rem;
        border-radius: 3px;
        transition: background 0.2s ease;
    }

    .expand-btn:hover {
        background: rgba(0, 0, 0, 0.05);
    }

    /* Body */
    .alert-body {
        padding: 0 1rem 0.75rem;
        border-top: 1px solid rgba(0, 0, 0, 0.08);
        animation: slideDown 0.2s ease;
    }

    @keyframes slideDown {
        from {
            opacity: 0;
            transform: translateY(-8px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }

    .details-section,
    .actions-section {
        margin-top: 0.75rem;
    }

    .section-header {
        font-family: var(--font-mono), 'Courier New', monospace;
        font-size: 0.65rem;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: var(--color-ink-light, #57534e);
        margin-bottom: 0.4rem;
        font-weight: 600;
    }

    .details-text {
        font-family: var(--font-serif), Georgia, serif;
        font-size: 0.8rem;
        color: var(--color-ink, #1c1c1c);
        line-height: 1.5;
        margin: 0;
        padding: 0.5rem;
        background: rgba(0, 0, 0, 0.02);
        border-radius: 4px;
        border-left: 2px solid var(--severity-color);
    }

    .actions-list {
        list-style: none;
        padding: 0;
        margin: 0;
    }

    .action-item {
        font-family: var(--font-serif), Georgia, serif;
        font-size: 0.8rem;
        color: var(--color-ink, #1c1c1c);
        padding: 0.4rem 0.5rem;
        padding-left: 1.5rem;
        position: relative;
        background: rgba(0, 0, 0, 0.02);
        border-radius: 4px;
        margin-bottom: 0.3rem;
        transition: background 0.2s ease;
    }

    .action-item:hover {
        background: rgba(0, 0, 0, 0.04);
    }

    .action-item::before {
        content: '-';
        position: absolute;
        left: 0.5rem;
        color: var(--severity-color);
        font-weight: bold;
    }

    .action-item:last-child {
        margin-bottom: 0;
    }
</style>
