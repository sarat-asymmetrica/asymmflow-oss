<script lang="ts">
    let { sentiment = {
        emotion: "curious", // "frustrated" | "curious" | "confident" | "uncertain"
        positivity: 0.75,
        certainty: 0.80
    } } = $props();

    let emotion = $derived(sentiment.emotion || "neutral");
    let positivity = $derived(sentiment.positivity || 0.5);
    let certainty = $derived(sentiment.certainty || 0.5);

    // Emotion colors
    function getEmotionColor(emotion) {
        switch (emotion.toLowerCase()) {
            case 'frustrated': return '#ef4444'; // red
            case 'curious': return '#3b82f6'; // blue
            case 'confident': return '#15803d'; // green
            case 'uncertain': return '#fbbf24'; // yellow
            default: return '#57534e'; // gray
        }
    }

    // Emotion icon
    function getEmotionIcon(emotion) {
        return ''; // Removed emojis
    }

    // Capitalize emotion
    function capitalize(str) {
        return str.charAt(0).toUpperCase() + str.slice(1);
    }
</script>

<div
    class="sentiment-badge"
    style="--emotion-color: {getEmotionColor(emotion)}"
    title="Positivity: {(positivity * 100).toFixed(0)}% | Certainty: {(certainty * 100).toFixed(0)}%"
>
    <span class="emotion-icon">{getEmotionIcon(emotion)}</span>
    <span class="emotion-label">{capitalize(emotion)}</span>

    <!-- Hover tooltip -->
    <div class="sentiment-tooltip">
        <div class="tooltip-row">
            <span class="tooltip-label">Positivity</span>
            <span class="tooltip-value">{(positivity * 100).toFixed(0)}%</span>
        </div>
        <div class="tooltip-row">
            <span class="tooltip-label">Certainty</span>
            <span class="tooltip-value">{(certainty * 100).toFixed(0)}%</span>
        </div>
    </div>
</div>

<style>
    .sentiment-badge {
        position: relative;
        display: inline-flex;
        align-items: center;
        gap: 0.4rem;
        padding: 0.3rem 0.6rem;
        background: rgba(255, 255, 255, 0.5);
        border: 1px solid var(--emotion-color);
        border-radius: 16px;
        font-size: 0.75rem;
        cursor: help;
        transition: all 0.2s ease;
    }

    .sentiment-badge:hover {
        background: rgba(255, 255, 255, 0.9);
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
        transform: translateY(-1px);
    }

    .sentiment-badge:hover .sentiment-tooltip {
        opacity: 1;
        visibility: visible;
        transform: translateY(0);
    }

    .emotion-icon {
        font-size: 1rem;
        line-height: 1;
    }

    .emotion-label {
        font-family: var(--font-mono), 'Courier New', monospace;
        font-size: 0.65rem;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: var(--emotion-color);
        font-weight: 600;
    }

    /* Tooltip */
    .sentiment-tooltip {
        position: absolute;
        top: calc(100% + 8px);
        left: 50%;
        transform: translateX(-50%) translateY(-4px);
        background: var(--color-ink, #1c1c1c);
        color: var(--color-paper, #fdfbf7);
        padding: 0.5rem 0.75rem;
        border-radius: 6px;
        font-size: 0.7rem;
        white-space: nowrap;
        opacity: 0;
        visibility: hidden;
        transition: all 0.2s ease;
        pointer-events: none;
        z-index: 100;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
    }

    .sentiment-tooltip::before {
        content: '';
        position: absolute;
        bottom: 100%;
        left: 50%;
        transform: translateX(-50%);
        border: 5px solid transparent;
        border-bottom-color: var(--color-ink, #1c1c1c);
    }

    .tooltip-row {
        display: flex;
        justify-content: space-between;
        gap: 1rem;
        padding: 0.15rem 0;
    }

    .tooltip-label {
        font-family: var(--font-serif), Georgia, serif;
        opacity: 0.8;
    }

    .tooltip-value {
        font-family: var(--font-mono), 'Courier New', monospace;
        font-weight: 600;
    }
</style>
