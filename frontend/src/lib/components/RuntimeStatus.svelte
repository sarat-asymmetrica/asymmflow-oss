<script lang="ts">
    import { onMount, onDestroy } from 'svelte';
    import { checkRuntimeHealth, type RuntimeHealth } from '../api/runtime';

    let health: RuntimeHealth = $state({ status: 'unknown' });
    let checking = true;
    let intervalId: ReturnType<typeof setInterval>;

    async function check() {
        checking = true;
        health = await checkRuntimeHealth();
        checking = false;
    }

    onMount(() => {
        check();
        // Check every 30 seconds
        intervalId = setInterval(check, 30000);
    });

    onDestroy(() => {
        if (intervalId) clearInterval(intervalId);
    });

    let statusColor = $derived(health.status === 'healthy' 
        ? 'var(--color-safe)' 
        : health.status === 'unhealthy' 
            ? 'var(--color-danger)' 
            : 'var(--color-ink-light)');

    let statusText = $derived(health.status === 'healthy'
        ? 'Runtime Connected'
        : health.status === 'unhealthy'
            ? 'Runtime Error'
            : 'Runtime Offline');
</script>

<div class="runtime-status" title={health.error || statusText}>
    <span class="status-dot" style="background-color: {statusColor}"></span>
    <span class="status-text">{statusText}</span>
    {#if health.status === 'healthy' && health.kernelCount}
        <span class="kernel-count">{health.kernelCount} kernels</span>
    {/if}
</div>

<style>
    .runtime-status {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.4rem 0.75rem;
        background: rgba(0, 0, 0, 0.03);
        border-radius: 4px;
        font-family: var(--font-mono);
        font-size: 0.7rem;
        color: var(--color-ink-light);
    }

    .status-dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        animation: pulse 2s ease-in-out infinite;
    }

    .status-text {
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .kernel-count {
        opacity: 0.6;
        margin-left: 0.25rem;
    }

    @keyframes pulse {
        0%, 100% { opacity: 1; }
        50% { opacity: 0.5; }
    }
</style>
