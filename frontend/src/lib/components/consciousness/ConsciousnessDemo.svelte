<script>
    import ConfidenceMeter from './ConfidenceMeter.svelte';
    import FlowIndicator from './FlowIndicator.svelte';
    import SentimentBadge from './SentimentBadge.svelte';
    import BlackSwanAlert from './BlackSwanAlert.svelte';

    // Demo data
    const demoConfidence = {
        score: 0.87,
        breakdown: {
            completeness: 0.92,
            reliability: 0.85,
            relevance: 0.88,
            consistency: 0.82,
            historical: 0.85
        }
    };

    const demoFlowState = {
        state: "In Flow",
        challenge: 0.75,
        skill: 0.80,
        recommendations: []
    };

    const demoSentiment = {
        emotion: "confident",
        positivity: 0.85,
        certainty: 0.90
    };

    const demoWarning = {
        severity: "medium",
        message: "Unusual payment pattern detected",
        details: "Customer EC-401 payment velocity decreased by 45% over last 14 days",
        suggestedActions: [
            "Contact customer for check-in call",
            "Review recent orders for quality issues",
            "Analyze market conditions in their sector"
        ]
    };
</script>

<div class="demo-container">
    <h2>Phase 2 Consciousness Components Demo</h2>

    <div class="demo-section">
        <h3>1. Confidence Meter</h3>
        <ConfidenceMeter confidence={demoConfidence} />
    </div>

    <div class="demo-section">
        <h3>2. Flow Indicator</h3>
        <FlowIndicator flowState={demoFlowState} />
    </div>

    <div class="demo-section">
        <h3>3. Sentiment Badge</h3>
        <div class="badges-row">
            <SentimentBadge sentiment={{ emotion: "frustrated", positivity: 0.3, certainty: 0.7 }} />
            <SentimentBadge sentiment={{ emotion: "curious", positivity: 0.6, certainty: 0.5 }} />
            <SentimentBadge sentiment={{ emotion: "confident", positivity: 0.9, certainty: 0.85 }} />
            <SentimentBadge sentiment={{ emotion: "uncertain", positivity: 0.5, certainty: 0.3 }} />
        </div>
    </div>

    <div class="demo-section">
        <h3>4. Black Swan Alert</h3>
        <BlackSwanAlert warning={demoWarning} />
        <BlackSwanAlert warning={{ ...demoWarning, severity: "critical", message: "Critical: Payment default imminent" }} />
        <BlackSwanAlert warning={{ ...demoWarning, severity: "low", message: "Info: Minor variance in delivery schedule" }} />
    </div>
</div>

<style>
    .demo-container {
        padding: 2rem;
        max-width: 1200px;
        margin: 0 auto;
    }

    h2 {
        font-family: var(--font-serif), Georgia, serif;
        font-size: 1.8rem;
        margin-bottom: 2rem;
        color: var(--color-ink, #1c1c1c);
    }

    h3 {
        font-family: var(--font-mono), 'Courier New', monospace;
        font-size: 0.75rem;
        text-transform: uppercase;
        letter-spacing: 1px;
        color: var(--color-ink-light, #57534e);
        margin-bottom: 1rem;
    }

    .demo-section {
        margin-bottom: 3rem;
        padding: 1.5rem;
        background: rgba(255, 255, 255, 0.5);
        border: 1px solid rgba(0, 0, 0, 0.1);
        border-radius: 8px;
    }

    .badges-row {
        display: flex;
        flex-wrap: wrap;
        gap: 1rem;
    }
</style>
