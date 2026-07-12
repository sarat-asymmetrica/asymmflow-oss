<script lang="ts">
    /**
     * OpportunityCard - Intelligence-Enhanced
     * Shows payment risk, ABB warnings, and deal health
     */
    type Opportunity = {
        status?: keyof typeof statusColors;
        title?: string;
        project?: string;  // RFQData uses 'project' field
        customer?: string;
        client?: string;   // RFQData uses 'client' field
        value?: number;
        updatedAt?: string | number | Date | null;
        paymentGrade?: keyof typeof paymentGradeConfig;
        customer_payment_grade?: keyof typeof paymentGradeConfig;
        notes?: string | null;
        competitor?: string | null;
    };

    interface Props {
        opportunity?: Opportunity | null;
        selected?: boolean;
        loading?: boolean;
        error?: string;
    }

    let {
        opportunity = null,
        selected = false,
        loading = false,
        error = ''
    }: Props = $props();

    const currency = new Intl.NumberFormat('en-US', {
        style: 'currency',
        currency: 'BHD',
        maximumFractionDigits: 0,
    });

    const statusColors: Record<string, string> = {
        New: 'var(--color-ink)',
        Quoted: 'var(--color-gold)',
        Won: 'var(--color-safe)',
        Lost: 'var(--color-danger)',
    };

    // Payment grade intelligence from SSOT
    const paymentGradeConfig: Record<string, { color: string; label: string; hint: string; risk: string }> = {
        A: { color: '#15803d', label: 'A', hint: 'Pays within 45 days', risk: 'low' },
        B: { color: '#d97706', label: 'B', hint: 'Pays within 90 days', risk: 'medium' },
        C: { color: '#ea580c', label: 'C', hint: 'Pays when they feel like it', risk: 'high' },
        D: { color: '#ef4444', label: 'D', hint: 'Chase for 6+ months - REQUIRE 100% ADVANCE', risk: 'critical' },
    };



    
    function calculateDealHealth(opp: Opportunity | null) {
        if (!opp) return 'unknown';
        let score = 100;
        
        // Payment grade penalty
        if (paymentGrade === 'C') score -= 20;
        if (paymentGrade === 'D') score -= 40;
        
        // ABB competition penalty
        if (hasABBCompetition) score -= 30;
        
        // Value-based (small orders = PH advantage)
        if (opp.value && opp.value < 5000) score += 10; // Small order advantage
        
        if (score >= 80) return 'healthy';
        if (score >= 50) return 'caution';
        return 'risk';
    }
    // ABB competition detection (from notes/title)
    let hasABBCompetition = $derived((opportunity?.notes?.toLowerCase()?.includes('abb') ?? false) ||
                           (opportunity?.title?.toLowerCase()?.includes('abb') ?? false) ||
                           (opportunity?.competitor?.toLowerCase() === 'abb'));
    let paymentGrade = $derived((opportunity?.paymentGrade || opportunity?.customer_payment_grade || 'B') as keyof typeof paymentGradeConfig);
    let gradeConfig = $derived(paymentGradeConfig[paymentGrade] || paymentGradeConfig.B);
    // Deal health score (simple heuristic)
    let dealHealth = $derived(calculateDealHealth(opportunity));
</script>

<div class={`card ${selected ? 'selected' : ''}`}>
    {#if loading}
        <div class="skeleton title"></div>
        <div class="skeleton line"></div>
        <div class="skeleton line short"></div>
    {:else if error}
        <p class="error">{error}</p>
    {:else if opportunity}
        <!-- Intelligence badges row -->
        <div class="intel-row">
            <div class="status-row">
                <span class="status-dot" style={`background:${statusColors[opportunity.status ?? 'New'] || 'var(--color-ink)'}`}></span>
                <span class="status-label">{opportunity.status ?? 'New'}</span>
            </div>
            <div class="badges">
                <!-- Payment Grade Badge -->
                <span 
                    class="badge payment-grade" 
                    style={`background: ${gradeConfig.color}15; color: ${gradeConfig.color}; border-color: ${gradeConfig.color}40`}
                    title={gradeConfig.hint}
                >
                    {gradeConfig.label}
                </span>
                
                <!-- ABB Warning -->
                {#if hasABBCompetition}
                    <span class="badge abb-warning" title="ABB is competing - Focus on service value!">
                        ABB WARNING
                    </span>
                {/if}
            </div>
        </div>

        <h3>{opportunity.project ?? opportunity.title ?? 'Untitled Opportunity'}</h3>
        <p class="customer">{opportunity.client ?? opportunity.customer ?? 'Unknown Customer'}</p>

        <div class="meta">
            <span class="value">{currency.format(opportunity.value ?? 0)}</span>
            <span class="timestamp">Updated {opportunity.updatedAt ? new Date(opportunity.updatedAt).toLocaleDateString() : '—'}</span>
        </div>
        
        <!-- Deal health indicator -->
        <div class="health-bar {dealHealth}">
            <div class="health-fill"></div>
        </div>
    {:else}
        <p class="empty">No opportunity data.</p>
    {/if}
</div>

<style>
    .card {
        background: rgba(255, 255, 255, 0.5);
        border: 1px solid rgba(0, 0, 0, 0.08);
        padding: 1rem 1.25rem;
        cursor: pointer;
        transition: all 0.25s ease;
        display: flex;
        flex-direction: column;
        gap: 0.35rem;
    }

    .card:hover {
        background: rgba(255, 255, 255, 0.8);
        box-shadow: 0 10px 20px rgba(0, 0, 0, 0.04);
    }

    .card.selected {
        border-color: var(--color-ink);
        box-shadow: 0 8px 18px rgba(0, 0, 0, 0.06);
    }

    .status-row {
        display: flex;
        align-items: center;
        gap: 0.4rem;
        font-family: var(--font-mono, 'Courier New', monospace);
        letter-spacing: 1px;
        text-transform: uppercase;
        font-size: 0.7rem;
    }

    .status-dot {
        width: 10px;
        height: 10px;
        border-radius: 50%;
        display: inline-block;
        border: 1px solid rgba(0,0,0,0.1);
    }

    h3 {
        margin: 0;
        font-weight: 600;
        letter-spacing: -0.3px;
        color: var(--color-ink);
    }

    .customer {
        margin: 0;
        color: var(--color-ink-light);
        font-family: var(--font-serif, Georgia, serif);
    }

    .meta {
        display: flex;
        justify-content: space-between;
        align-items: center;
        font-family: var(--font-mono, 'Courier New', monospace);
        font-size: 0.75rem;
        color: var(--color-ink-light);
    }

    .value {
        color: var(--color-ink);
        letter-spacing: 1px;
    }

    .skeleton {
        background: rgba(0,0,0,0.05);
        border-radius: 4px;
        animation: pulse 1.6s ease-in-out infinite;
    }

    .skeleton.title { height: 18px; width: 70%; }
    .skeleton.line { height: 14px; width: 60%; }
    .skeleton.line.short { width: 40%; }

    @keyframes pulse {
        0% { opacity: 0.5; }
        50% { opacity: 0.9; }
        100% { opacity: 0.5; }
    }

    .error { color: var(--color-danger); margin: 0; }
    .empty { margin: 0; color: var(--color-ink-light); }

    /* Intelligence Layer Styles */
    .intel-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 0.25rem;
    }

    .badges {
        display: flex;
        gap: 0.35rem;
    }

    .badge {
        font-family: var(--font-mono, 'Courier New', monospace);
        font-size: 0.6rem;
        padding: 0.15rem 0.4rem;
        border-radius: 3px;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        border: 1px solid;
    }

    .badge.payment-grade {
        font-weight: bold;
    }

    .badge.abb-warning {
        background: rgba(239, 68, 68, 0.1);
        color: #ef4444;
        border-color: rgba(239, 68, 68, 0.3);
        animation: pulse-warning 2s ease-in-out infinite;
    }

    @keyframes pulse-warning {
        0%, 100% { opacity: 1; }
        50% { opacity: 0.7; }
    }

    /* Deal Health Bar */
    .health-bar {
        height: 3px;
        background: rgba(0, 0, 0, 0.08);
        border-radius: 2px;
        margin-top: 0.5rem;
        overflow: hidden;
    }

    .health-bar .health-fill {
        height: 100%;
        border-radius: 2px;
        transition: width 0.3s ease;
    }

    .health-bar.healthy .health-fill {
        width: 100%;
        background: #15803d;
    }

    .health-bar.caution .health-fill {
        width: 60%;
        background: #d97706;
    }

    .health-bar.risk .health-fill {
        width: 30%;
        background: #ef4444;
    }

    .health-bar.unknown .health-fill {
        width: 50%;
        background: #57534e;
    }
</style>
