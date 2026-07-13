<script lang="ts">
  /**
   * DealDocumentChecklist - the six-document closing set as a checklist
   * (Wave 10 B5a, Article I.6, DESIGN_CONSTITUTION.md).
   *
   * The set: offer / order confirmation / delivery note / invoice /
   * statement entry / receipt. Mounted directly below DealTimeline (the seam
   * B3 left) - this is the document detail that timeline deliberately keeps
   * OUT of its nodes. Read-only: one call to GetDealTimeline(orderId), whose
   * `documents` field carries the six rows already assembled server-side
   * (invoice_traceability.go: buildDealDocumentChecklist). Missing documents
   * render as an honest "— not yet" row, never fabricated.
   *
   * When every row is present AND the invoice is Paid, the set renders a
   * quiet complete state - the settle moment. B4's sound already lives on
   * the PAID posting elsewhere; this only gets the one quiet `.motion-settle`
   * transition, nothing more (no extra celebration, per spec).
   */
  import { onMount } from "svelte";
  import { GetDealTimeline } from "../../../wailsjs/go/main/App";
  import type { main } from "../../../wailsjs/go/models";
  import { pendingOrderView } from "$lib/stores/navigation";

  interface Props {
    orderId: string;
  }

  let { orderId }: Props = $props();

  let timeline: main.DealTimeline | null = $state(null);
  let loading = $state(false);
  let error = $state("");

  // Same deep-link discipline as DealTimeline.svelte: only record types that
  // already have an established nav target elsewhere in the app are
  // clickable. Delivery notes, offers, and payment/receipt records have no
  // existing route to hand off to, so those rows render as plain,
  // non-interactive text - we never invent new routing.
  const CLICKABLE_RECORD_TYPES = new Set(["order", "invoice"]);

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
      error = err?.message || String(err) || "Failed to load document checklist";
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

  // Reuses the exact same handoff mechanisms DealTimeline uses for its
  // "Order" and "Invoice" nodes - never a new route.
  function openDoc(doc: main.DealDocumentStatus) {
    if (!isClickable(doc)) return;
    switch (doc.record_type) {
      case "order":
        pendingOrderView.request(String(doc.record_id), doc.serial || "");
        navigateTo({ screen: "opportunities", tab: "orders" });
        break;
      case "invoice":
        sessionStorage.setItem(
          "asymmflow.pendingInvoiceFocus",
          JSON.stringify({ id: doc.record_id, invoice_number: doc.serial }),
        );
        navigateTo({ screen: "finance", tab: "invoices" });
        break;
    }
  }

  function isClickable(doc: main.DealDocumentStatus): boolean {
    return !!doc.present && !!doc.record_id && CLICKABLE_RECORD_TYPES.has(doc.record_type || "");
  }

  // The quiet complete state: every one of the six documents present AND the
  // invoice fully settled (the same Paid signal the DealTimeline spine
  // already derives on its "Paid" node - read here, never recomputed).
  let allPresent = $derived(
    !!timeline?.documents?.length && timeline.documents.every((d) => d.present),
  );
  let isPaid = $derived(
    !!timeline?.nodes?.some((n) => n.stage === "Paid" && n.state === "done"),
  );
  let isComplete = $derived(allPresent && isPaid);
</script>

<div class="deal-documents">
  {#if loading}
    <p class="muted">Loading document checklist...</p>
  {:else if error}
    <p class="error-text">{error}</p>
  {:else if timeline && timeline.documents?.length}
    <ul class="checklist" class:complete={isComplete} class:motion-settle={isComplete} role="list">
      {#each timeline.documents as doc (doc.document)}
        <li class="row" role="listitem">
          <span class="mark" class:present={doc.present} aria-hidden="true"></span>
          <span class="doc-name">{doc.document}</span>
          {#if doc.present}
            <button
              type="button"
              class="doc-value"
              class:clickable={isClickable(doc)}
              disabled={!isClickable(doc)}
              onclick={() => openDoc(doc)}
            >
              <span class="serial">{doc.serial || "—"}</span>
              {#if formatDate(doc.date)}
                <span class="date">{formatDate(doc.date)}</span>
              {/if}
            </button>
          {:else}
            <span class="doc-value missing">— not yet</span>
          {/if}
        </li>
      {/each}
    </ul>
    {#if isComplete}
      <p class="complete-note">Full document set on file — settled.</p>
    {/if}
  {:else}
    <p class="muted">No document data.</p>
  {/if}
</div>

<style>
  .deal-documents {
    display: flex;
    flex-direction: column;
    gap: 6px;
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

  .checklist {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 3px 0;
    border-bottom: 1px solid var(--border, rgba(0, 0, 0, 0.06));
  }

  .row:last-child {
    border-bottom: none;
  }

  .mark {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex: 0 0 auto;
    background: transparent;
    border: 1.5px solid var(--steel);
  }

  .mark.present {
    background: var(--onyx);
    border-color: var(--onyx);
  }

  .doc-name {
    flex: 0 0 auto;
    min-width: 130px;
    font-size: var(--label-size, 11px);
    font-weight: var(--label-weight, 600);
    letter-spacing: 0.02em;
    text-transform: uppercase;
    color: var(--text-secondary);
  }

  .doc-value {
    display: flex;
    align-items: baseline;
    gap: 8px;
    flex: 1 1 auto;
    background: none;
    border: none;
    padding: 0;
    margin: 0;
    font-family: var(--font-ui);
    text-align: left;
    cursor: default;
  }

  .doc-value.clickable {
    cursor: pointer;
  }

  .doc-value.clickable:hover .serial,
  .doc-value.clickable:focus-visible .serial {
    text-decoration: underline;
  }

  .doc-value:focus-visible {
    outline: 2px solid var(--onyx);
    outline-offset: 2px;
    border-radius: var(--border-radius-sm);
  }

  .serial {
    font-size: var(--table-text-size, 13px);
    font-weight: 500;
    color: var(--text-primary);
  }

  .date {
    font-size: var(--meta-size, 11px);
    color: var(--text-muted);
  }

  .missing {
    font-size: var(--table-text-size, 13px);
    color: var(--text-muted);
    font-style: italic;
  }

  .complete-note {
    margin: 4px 0 0;
    font-size: var(--meta-size, 11px);
    color: var(--text-secondary);
    letter-spacing: 0.01em;
  }

  /* .motion-settle's reduced-motion handling is global (design-tokens.css,
     Wave 10 B2) - nothing to duplicate here. */
</style>
