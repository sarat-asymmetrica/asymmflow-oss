<script lang="ts">
  import { devLog } from "$lib/utils/devLog";
  import type { ErrorFallbackComponent, ErrorHandler } from "$lib/types";
  /**
   * ERROR BOUNDARY COMPONENT
   *
   * Catches uncaught errors and displays graceful fallback UI.
   * Prevents entire app from crashing with white screen.
   *
   * Usage:
   *   <ErrorBoundary name="Sales Hub">
   *     <YourApp />
   *   </ErrorBoundary>
   */
  import { onMount, onDestroy } from "svelte";
  import WabiButton from "./ui/WabiButton.svelte";

  interface Props {
    fallback?: ErrorFallbackComponent;
    onError?: ErrorHandler | null;
    name?: string; // Name of the wrapped component for error context
    children?: import('svelte').Snippet;
  }

  let {
    fallback = null,
    onError = null,
    name = "Component",
    children
  }: Props = $props();

  let error: Error | null = $state(null);
  let errorInfo: { stack?: string } | null = $state(null);

  function handleError(event: ErrorEvent | PromiseRejectionEvent) {
    const err = "error" in event ? event.error : event.reason;

    // Capture the error and display fallback UI
    if (err instanceof Error) {
      error = err;
      errorInfo = { stack: err.stack };
    } else {
      error = new Error(String(err) || "An unexpected error occurred");
      errorInfo = { stack: "" };
    }

    devLog.error(`ErrorBoundary [${name}] caught:`, err);

    if (onError) {
      onError(err);
    }

    // Prevent default error handling (console spam)
    event.preventDefault();
  }

  function reload() {
    window.location.reload();
  }

  function reset() {
    error = null;
    errorInfo = null;
  }

  onMount(() => {
    window.addEventListener("error", handleError);
    window.addEventListener("unhandledrejection", handleError);
  });

  onDestroy(() => {
    window.removeEventListener("error", handleError);
    window.removeEventListener("unhandledrejection", handleError);
  });
</script>

{#if error}
  {#if fallback}
    {@const SvelteComponent = fallback}
    <SvelteComponent {error} {errorInfo} {reset} {reload} />
  {:else}
    <!-- Default Error UI -->
    <div class="error-boundary" data-testid="error-boundary">
      <div class="error-card">
        <div class="error-icon">!</div>
        <h2 class="error-title">Something went wrong in {name}</h2>
        <p class="error-message" data-testid="error-message">
          {error.message ||
            "An unexpected error occurred. The application encountered a problem."}
        </p>

        <details class="error-details">
          <summary class="details-toggle">Technical Details</summary>
          <div class="stack-trace">
            <pre>{errorInfo?.stack || "No stack trace available"}</pre>
          </div>
        </details>

        <div class="error-actions">
          <WabiButton variant="primary" on:click={reload}>
            Reload Application
          </WabiButton>
          <WabiButton variant="ghost" on:click={reset}>Try Again</WabiButton>
        </div>

        <p class="error-hint">
          If this problem persists, please contact support with the technical
          details above.
        </p>
      </div>
    </div>
  {/if}
{:else}
  {@render children?.()}
{/if}

<style>
  /* ============================================================
     ERROR BOUNDARY - WABI-SABI THEME
     φ-based spacing: 8, 13, 21, 34, 55px
     ============================================================ */

  .error-boundary {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    padding: 34px;
    background: var(--color-paper, #fdfbf7);
    background-image: url("data:image/svg+xml,%3Csvg width='100' height='100' viewBox='0 0 100 100' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='noise'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.8' numOctaves='3' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100' height='100' filter='url(%23noise)' opacity='0.04'/%3E%3C/svg%3E");
  }

  .error-card {
    max-width: 600px;
    width: 100%;
    padding: 34px;
    background: rgba(239, 68, 68, 0.03);
    border: 1px solid rgba(239, 68, 68, 0.15);
    border-radius: 8px;
    box-shadow: 0 13px 34px rgba(239, 68, 68, 0.05);
  }

  .error-icon {
    font-size: 48px;
    text-align: center;
    margin-bottom: 21px;
    opacity: 0.8;
    animation: pulse 2s ease-in-out infinite;
  }

  @keyframes pulse {
    0%,
    100% {
      opacity: 0.6;
    }
    50% {
      opacity: 1;
    }
  }

  .error-title {
    font-family: Georgia, serif;
    font-size: 24px;
    font-weight: normal;
    margin: 0 0 13px;
    text-align: center;
    color: #1c1c1c;
  }

  .error-message {
    font-family: Georgia, serif;
    font-size: 14px;
    line-height: 1.6;
    margin: 0 0 21px;
    color: #57534e;
    text-align: center;
  }

  .error-details {
    margin-bottom: 21px;
    background: rgba(0, 0, 0, 0.02);
    border: 1px solid rgba(0, 0, 0, 0.05);
    border-radius: 6px;
    padding: 13px;
  }

  .details-toggle {
    font-family: "Courier Prime", monospace;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 1px;
    cursor: pointer;
    color: #57534e;
    user-select: none;
  }

  .details-toggle:hover {
    color: #1c1c1c;
  }

  .stack-trace {
    margin-top: 13px;
  }

  .stack-trace pre {
    background: rgba(0, 0, 0, 0.03);
    padding: 13px;
    border-radius: 4px;
    font-family: "Courier Prime", monospace;
    font-size: 11px;
    overflow-x: auto;
    line-height: 1.5;
    color: #1c1c1c;
    margin: 0;
  }

  .error-actions {
    display: flex;
    gap: 13px;
    justify-content: center;
    margin-bottom: 21px;
  }

  .error-hint {
    font-family: Georgia, serif;
    font-size: 12px;
    font-style: italic;
    color: #888;
    text-align: center;
    margin: 0;
  }

  @media (max-width: 768px) {
    .error-boundary {
      padding: 21px;
    }

    .error-card {
      padding: 21px;
    }

    .error-title {
      font-size: 20px;
    }

    .error-actions {
      flex-direction: column;
    }
  }
</style>
