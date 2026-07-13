<script lang="ts">
  /**
   * DealTimeline - the deal-spine stepper (Article I - THE signature timeline).
   *
   * Owner-ratified direction (Wave 10 B3, 2026-07-13): UX simplicity above all.
   * A compact single-row horizontal stepper - small state-colored dots
   * connected by a thin rule, serial + date as quiet two-line labels beneath
   * each node, generous type size, zero ornamentation. Document detail (the
   * six-doc checklist) is a later coder's job (B5a) and lives BELOW or
   * on-demand - this component is only the host/spine, never the checklist.
   *
   * One backend call assembles the whole chain: GetDealTimeline(orderId).
   */
  import { onMount } from "svelte";
  import { createEventDispatcher } from "svelte";
  import { GetDealTimeline } from "../../../wailsjs/go/main/App";
  import type { main } from "../../../wailsjs/go/models";
  import { pendingOrderView } from "$lib/stores/navigation";

  interface Props {
    orderId: string;
  }

  let { orderId }: Props = $props();

  const dispatch = createEventDispatcher();

  let timeline: main.DealTimeline | null = $state(null);
  let loading = $state(false);
  let error = $state("");

  // Stages that have an established, existing deep-link mechanism elsewhere
  // in the app (Article II pattern #1: context-preserving handoffs). Stages
  // without one (Offer, Costing, Delivery) render as plain, non-interactive
  // nodes - we never invent new routing.
  const CLICKABLE_STAGES = new Set(["RFQ", "Order", "Invoice", "Paid"]);

  function formatDate(date: any): string {
    if (!date) return "";
    try {
      const d = new Date(date);
      if (isNaN(d.getTime()) || d.getFullYear() <= 1) return "";
      return d.toLocaleDateString("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric",
      });
    } catch {
      return "";
    }
  }

  async function load() {
    if (!orderId) return;
    loading = true;
    error = "";
    try {
      timeline = await GetDealTimeline(String(orderId));
    } catch (err: any) {
      error = err?.message || String(err) || "Failed to load deal timeline";
    } finally {
      loading = false;
    }
  }

  onMount(load);

  $effect(() => {
    if (orderId) load();
  });

  function navigateTo(target: Record<string, string>) {
    window.dispatchEvent(new CustomEvent("navigateToScreen", { detail: target }));
  }

  // Deep-link using the SAME mechanisms OrderDetail/CustomerDetailView/
  // OffersScreen already use to hand off between screens - never a new route.
  function openNode(node: main.DealTimelineNode) {
    if (!node.record_id || !CLICKABLE_STAGES.has(node.stage)) return;

    dispatch("nodeClick", { node });

    switch (node.stage) {
      case "RFQ":
        sessionStorage.setItem("asymmflow.pendingOpportunityId", String(node.record_id));
        navigateTo({ screen: "opportunities" });
        break;
      case "Order":
        pendingOrderView.request(String(node.record_id), node.serial || "");
        navigateTo({ screen: "opportunities", tab: "orders" });
        break;
      case "Invoice":
      case "Paid":
        // "Paid" node deep-links to the invoice that carries the settled state.
        if (node.record_type === "invoice") {
          sessionStorage.setItem(
            "asymmflow.pendingInvoiceFocus",
            JSON.stringify({ id: node.record_id, invoice_number: node.serial }),
          );
          navigateTo({ screen: "finance", tab: "invoices" });
        }
        break;
    }
  }

  function isClickable(node: main.DealTimelineNode): boolean {
    return !!node.record_id && CLICKABLE_STAGES.has(node.stage) && node.state !== "na";
  }
</script>

<div class="deal-timeline">
  {#if loading}
    <p class="muted">Loading deal timeline...</p>
  {:else if error}
    <p class="error-text">{error}</p>
  {:else if timeline && timeline.nodes?.length}
    <div class="stepper" role="list">
      {#each timeline.nodes as node, i (node.stage)}
        <div class="step" role="listitem">
          {#if i > 0}
            <span class="rule" class:rule-done={node.state === "done" || node.state === "current"}></span>
          {/if}
          <button
            type="button"
            class="node state-{node.state}"
            class:clickable={isClickable(node)}
            disabled={!isClickable(node)}
            onclick={() => openNode(node)}
            title={node.serial || node.stage}
          >
            <span class="dot" aria-hidden="true"></span>
            <span class="labels">
              <span class="stage-name">{node.stage}</span>
              <span class="serial">{node.serial || "—"}</span>
              <span class="date">{formatDate(node.date) || ""}</span>
              {#if node.count && node.count > 1}
                <span class="count">+{node.count - 1} more</span>
              {/if}
            </span>
          </button>
        </div>
      {/each}
    </div>
  {:else}
    <p class="muted">No timeline data.</p>
  {/if}
</div>

<style>
  .deal-timeline {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .muted {
    color: var(--text-muted);
    font-size: var(--table-text-size, 13px);
  }

  .error-text {
    color: var(--onyx);
    font-weight: 600;
    font-size: var(--table-text-size, 13px);
  }

  /* Single-row horizontal stepper. Never shrinks text to fit - it scrolls
     horizontally on narrow widths instead (owner-ratified, 2026-07-13). */
  .stepper {
    display: flex;
    align-items: flex-start;
    overflow-x: auto;
    padding: 4px 2px 8px;
    scrollbar-width: thin;
  }

  .step {
    display: flex;
    align-items: flex-start;
    flex: 0 0 auto;
  }

  .rule {
    width: 40px;
    height: 1px;
    background: var(--border);
    margin: 13px 4px 0;
    flex: 0 0 auto;
    transition: background var(--transition-fast);
  }

  .rule-done {
    background: var(--onyx);
  }

  .node {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    background: none;
    border: none;
    padding: 0 8px;
    min-width: 88px;
    cursor: default;
    font-family: var(--font-ui);
    text-align: center;
  }

  .node.clickable {
    cursor: pointer;
  }

  .node.clickable:hover .dot,
  .node.clickable:focus-visible .dot {
    box-shadow: 0 0 0 4px var(--onyx-tint);
  }

  .node:focus-visible {
    outline: 2px solid var(--onyx);
    outline-offset: 2px;
    border-radius: var(--border-radius-sm);
  }

  .dot {
    width: 14px;
    height: 14px;
    border-radius: 50%;
    box-sizing: border-box;
    transition:
      transform var(--motion-base) var(--ease-decelerate),
      box-shadow var(--transition-fast);
  }

  /* Monochrome semantic status - Onyx & Ether is a contrast system, not a
     color system, so "done"/"current"/"pending"/"na" are distinguished by
     fill/weight/opacity, never hue. This also sidesteps the flagged
     green-accent collision risk noted for a future deployment - there is no
     green here to collide with. */
  .state-done .dot {
    background: var(--onyx);
    border: 1.5px solid var(--onyx);
  }

  .state-current .dot {
    background: var(--canvas);
    border: 2px solid var(--onyx);
    transform: scale(1.15);
  }

  .state-pending .dot {
    background: transparent;
    border: 1.5px solid var(--steel);
  }

  .state-na .dot {
    background: transparent;
    border: 1.5px dashed var(--text-muted);
    opacity: 0.6;
  }

  .labels {
    display: flex;
    flex-direction: column;
    gap: 1px;
  }

  .stage-name {
    font-size: var(--label-size, 11px);
    font-weight: var(--label-weight, 600);
    letter-spacing: 0.03em;
    text-transform: uppercase;
    color: var(--text-secondary);
  }

  .state-done .stage-name,
  .state-current .stage-name {
    color: var(--text-primary);
  }

  .state-na .stage-name {
    opacity: 0.55;
  }

  .serial {
    font-size: var(--table-text-size, 13px);
    font-weight: 500;
    color: var(--text-primary);
    white-space: nowrap;
  }

  .state-na .serial,
  .state-pending .serial {
    color: var(--text-muted);
    font-weight: 400;
  }

  .date {
    font-size: var(--meta-size, 11px);
    color: var(--text-muted);
    white-space: nowrap;
  }

  .count {
    font-size: var(--meta-size, 11px);
    color: var(--text-secondary);
  }

  @media (prefers-reduced-motion: reduce) {
    .dot {
      transition: none;
    }
  }
</style>
