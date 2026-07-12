<script lang="ts">
    
    interface Props {
        /**
     * Wabi-Sabi Skeleton Loader
     * Rice-paper inspired loading placeholders
     */
        height?: string;
        width?: string;
        rounded?: boolean;
        className?: string;
    }

    let {
        height = '100px',
        width = '100%',
        rounded = true,
        className = ''
    }: Props = $props();
</script>

<div
    class="skeleton {className}"
    class:rounded
    style="height: {height}; width: {width};"
></div>

<style>
    .skeleton {
        background: linear-gradient(
            90deg,
            rgba(28, 28, 28, 0.03) 0%,
            rgba(28, 28, 28, 0.05) 50%,
            rgba(28, 28, 28, 0.03) 100%
        );
        background-size: 200% 100%;
        animation: shimmer 2s ease-in-out infinite;
        position: relative;
        overflow: hidden;
    }

    .skeleton.rounded {
        border-radius: var(--radius-md, 8px);
    }

    @keyframes shimmer {
        0% {
            background-position: 200% 0;
        }
        100% {
            background-position: -200% 0;
        }
    }

    /* Breathing effect for Wabi-Sabi aesthetic */
    .skeleton::after {
        content: '';
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(197, 160, 89, 0.02);
        animation: breathe var(--breath-calm, 2000ms) ease-in-out infinite;
    }

    @keyframes breathe {
        0%, 100% {
            opacity: 0;
        }
        50% {
            opacity: 1;
        }
    }
</style>
