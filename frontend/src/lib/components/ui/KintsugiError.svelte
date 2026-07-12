<script lang="ts">
    interface Props {
        message?: string;
        show?: boolean;
    }

    let { message = "Error", show = false }: Props = $props();
</script>

{#if show}
<div class="bg-red-900/10 border-l-4 border-[var(--gold-color)] p-4 rounded-r shadow-sm my-4 backdrop-blur-sm relative overflow-hidden group">
    <div class="flex items-start">
        <div class="flex-shrink-0">
             <!-- Kintsugi Crack Icon -->
            <svg class="h-5 w-5 text-[var(--gold-color)]" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
            </svg>
        </div>
        <div class="ml-3">
            <h3 class="text-sm font-medium text-[var(--gold-color)] font-serif">Kintsugi Repair Required</h3>
            <div class="mt-2 text-sm text-[var(--text-color)] opacity-80">
                <p>{message}</p>
            </div>
        </div>
    </div>

    <!-- Gold Shimmer Effect -->
    <div class="absolute inset-0 -translate-x-full group-hover:animate-[shimmer_1.5s_infinite] bg-gradient-to-r from-transparent via-[var(--gold-color)]/20 to-transparent skew-x-12"></div>
</div>
{/if}

<style>
    @keyframes shimmer {
        100% {
            transform: translateX(100%);
        }
    }
</style>
