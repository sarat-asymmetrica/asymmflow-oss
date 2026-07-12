<script lang="ts">
    /**
     * Pre-composed skeleton for card layouts
     */
    import Skeleton from './Skeleton.svelte';

    interface Props {
        variant?: string; // 'default', 'detailed', 'list'
    }

    let { variant = 'default' }: Props = $props();
</script>

{#if variant === 'list'}
    <!-- List item skeleton (compact) -->
    <div class="skeleton-list-item">
        <Skeleton height="20px" width="60%" />
        <Skeleton height="14px" width="40%" />
    </div>
{:else if variant === 'detailed'}
    <!-- Detailed card skeleton -->
    <div class="skeleton-card">
        <Skeleton height="24px" width="70%" />
        <Skeleton height="16px" width="90%" />
        <Skeleton height="16px" width="85%" />
        <div class="skeleton-row">
            <Skeleton height="14px" width="30%" />
            <Skeleton height="14px" width="25%" />
        </div>
    </div>
{:else}
    <!-- Default card skeleton -->
    <div class="skeleton-card">
        <Skeleton height="20px" width="60%" />
        <Skeleton height="14px" width="80%" />
        <Skeleton height="14px" width="40%" />
    </div>
{/if}

<style>
    .skeleton-card {
        display: flex;
        flex-direction: column;
        gap: var(--space-1, 13px);
        padding: var(--space-2, 21px);
        background: rgba(255, 255, 255, 0.4);
        border: var(--border-subtle, 1px solid rgba(0, 0, 0, 0.05));
        border-radius: var(--radius-lg, 13px);
    }

    .skeleton-list-item {
        display: flex;
        flex-direction: column;
        gap: var(--space-0, 8px);
        padding: var(--space-1, 13px);
    }

    .skeleton-row {
        display: flex;
        gap: var(--space-2, 21px);
        margin-top: var(--space-1, 13px);
    }
</style>
