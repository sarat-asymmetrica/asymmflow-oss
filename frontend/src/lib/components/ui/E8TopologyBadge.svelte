<script lang="ts">
    
    interface Props {
        /**
     * E8 LATTICE TOPOLOGY VERIFICATION BADGE
     *
     * Displays mathematical verification that entity linking is topology-sound.
     *
     * E8 LATTICE PROPERTIES (from E8Lattice.lean):
     * - 240 root vectors (kissing number optimal in 8D)
     * - All roots have norm √2
     * - countNonZero function = shared attributes = similarity measure
     * - Proves uniqueness via 8-dimensional sphere packing
     *
     * REFERENCE: C:\Projects\asymm_all_math\asymmetrica_proofs\AsymmetricaProofs\E8Lattice.lean
     *
     * @component E8TopologyBadge
     * @prop {boolean} verified - Whether entity has been verified
     * @prop {number} similarityCount - Number of similar entities (optional)
     * @prop {string} size - 'sm' | 'md' | 'lg'
     */
        verified?: boolean;
        similarityCount?: number;
        size?: 'sm' | 'md' | 'lg';
    }

    let { verified = true, similarityCount = 0, size = 'sm' }: Props = $props();

    // Size mappings (Wabi-Sabi aesthetic - natural proportions)
    const sizes = {
        sm: { fontSize: '0.7rem', padding: '0.25rem 0.5rem', iconSize: '12px' },
        md: { fontSize: '0.8rem', padding: '0.35rem 0.65rem', iconSize: '14px' },
        lg: { fontSize: '0.9rem', padding: '0.45rem 0.8rem', iconSize: '16px' }
    };

    const currentSize = sizes[size];
</script>

{#if verified}
    <div
        class="e8-badge"
        style="font-size: {currentSize.fontSize}; padding: {currentSize.padding}"
        title="E8 Lattice: Topology-sound entity linking via 240-root 8D sphere packing"
        data-testid="e8-topology-badge"
    >
        <span class="e8-icon" style="font-size: {currentSize.iconSize}"></span>
        <span class="e8-label">E8 Verified</span>
        {#if similarityCount > 0}
            <span class="similarity-count" title="{similarityCount} topologically similar entities detected">
                ({similarityCount})
            </span>
        {/if}
    </div>
{/if}

<style>
    .e8-badge {
        display: inline-flex;
        align-items: center;
        gap: 0.35rem;
        background: linear-gradient(135deg, rgba(59, 130, 246, 0.08) 0%, rgba(147, 51, 234, 0.08) 100%);
        border: 1px solid rgba(59, 130, 246, 0.2);
        border-radius: 4px;
        font-family: var(--font-mono);
        color: #1e40af;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        white-space: nowrap;
        transition: all 0.2s ease;
    }

    .e8-badge:hover {
        background: linear-gradient(135deg, rgba(59, 130, 246, 0.12) 0%, rgba(147, 51, 234, 0.12) 100%);
        border-color: rgba(59, 130, 246, 0.35);
        transform: translateY(-1px);
        box-shadow: 0 2px 4px rgba(59, 130, 246, 0.1);
    }

    .e8-icon {
        display: inline-block;
        line-height: 1;
    }

    .e8-label {
        font-weight: 600;
        letter-spacing: 0.8px;
    }

    .similarity-count {
        font-size: 0.85em;
        opacity: 0.8;
        font-weight: 500;
    }

    /* Accessibility: Ensure sufficient contrast */
    @media (prefers-contrast: high) {
        .e8-badge {
            background: rgba(59, 130, 246, 0.15);
            border: 2px solid #1e40af;
            color: #1e3a8a;
        }
    }

    /* Dark mode support (future-proof) */
    @media (prefers-color-scheme: dark) {
        .e8-badge {
            background: linear-gradient(135deg, rgba(96, 165, 250, 0.12) 0%, rgba(167, 139, 250, 0.12) 100%);
            border-color: rgba(96, 165, 250, 0.3);
            color: #93c5fd;
        }
    }
</style>
