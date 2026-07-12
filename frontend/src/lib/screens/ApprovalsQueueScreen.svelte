<script lang="ts">
  // Wave 9.4 B5c: the persistent approvals queue (Design Constitution
  // Article V.2 — the Task class persists until *done*, reading never
  // dismisses it, and it lives in a work queue with a clear owner).
  //
  // Notifications keep announcing pending delete/employee-archive approvals
  // (unchanged); this screen is their durable home — it always reflects the
  // real pending state of every request an admin/reviewer can act on,
  // independent of whether any notification about it has been read.
  //
  // Mounted standalone (its own screen) or embedded (e.g. as a WorkHub
  // "Approvals" tab) via the optional `embedded` prop.

  import { onMount, onDestroy } from "svelte";
  import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime";
  import { toast } from "$lib/stores/toasts";
  import { confirm } from "$lib/stores/confirm";
  import PageLayout from "$lib/components/layout/PageLayout.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import WabiSpinner from "$lib/components/ui/WabiSpinner.svelte";
  import { getCurrentEmployeeContext, reviewDeleteApprovalRequest, reviewEmployeeArchiveRequest } from "$lib/api/collaboration";
  import { listDeleteApprovals, listEmployeeArchiveApprovals, type DeleteApprovalItem, type EmployeeArchiveApprovalItem } from "$lib/api/approvals";

  interface Props {
    embedded?: boolean;
  }
  let { embedded = false }: Props = $props();

  type ApprovalKind = "delete" | "employee_archive";

  interface ApprovalRow {
    kind: ApprovalKind;
    id: string;
    title: string;
    subtitle: string;
    owner: string;
    reason: string;
    createdAt?: string;
    consequenceWeight: number;
  }

  // Handoff key read by NotificationsScreen (a screen this coder also owns)
  // to scroll a queue row's source notification into view — the "opens the
  // item" half of the deep-link requirement.
  const pendingApprovalStorageKey = "asymmflow.pendingApprovalSourceId";

  let loading = $state(true);
  let loadError = $state("");
  let currentRole = $state("");
  let isAdmin = $state(false);
  let deleteApprovals: DeleteApprovalItem[] = $state([]);
  let archiveApprovals: EmployeeArchiveApprovalItem[] = $state([]);
  let reviewingID = $state("");

  function relativeAge(createdAt?: string): string {
    if (!createdAt) return "Unknown age";
    const created = new Date(createdAt);
    if (Number.isNaN(created.getTime())) return "Unknown age";
    const diffMs = Math.max(0, Date.now() - created.getTime());
    const minutes = Math.floor(diffMs / 60_000);
    if (minutes < 1) return "Just now";
    if (minutes < 60) return `${minutes}m ago`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours}h ago`;
    const days = Math.floor(hours / 24);
    if (days < 30) return `${days}d ago`;
    const months = Math.floor(days / 30);
    return `${months}mo ago`;
  }

  function buildRows(): ApprovalRow[] {
    const deleteRows: ApprovalRow[] = deleteApprovals.map((item) => ({
      kind: "delete" as const,
      id: item.id,
      title: item.entity_label || `${item.entity_type || "record"} ${item.entity_id || ""}`.trim(),
      subtitle: `Delete · ${item.entity_type || "record"}`,
      owner: item.requested_by_name || "Unknown requester",
      reason: item.reason || "",
      createdAt: item.created_at,
      // Deletes are consequential and irreversible once approved — they lead the queue.
      consequenceWeight: 2,
    }));
    const archiveRows: ApprovalRow[] = archiveApprovals.map((item) => ({
      kind: "employee_archive" as const,
      id: item.id,
      title: item.employee_name || "Employee archive request",
      subtitle: "Employee archive",
      owner: item.requested_by_name || "Unknown requester",
      reason: item.reason || "",
      createdAt: item.created_at,
      consequenceWeight: 1,
    }));

    return [...deleteRows, ...archiveRows].sort((a, b) => {
      if (a.consequenceWeight !== b.consequenceWeight) return b.consequenceWeight - a.consequenceWeight;
      const aTime = a.createdAt ? new Date(a.createdAt).getTime() : 0;
      const bTime = b.createdAt ? new Date(b.createdAt).getTime() : 0;
      return aTime - bTime; // oldest first within the same consequence tier
    });
  }

  let rows = $derived(buildRows());

  async function load() {
    loading = rows.length === 0;
    loadError = "";
    try {
      const ctx = await getCurrentEmployeeContext();
      currentRole = ctx?.license_role || "";
      isAdmin = ["admin", "administrator", "developer"].includes(currentRole.toLowerCase());

      if (!isAdmin) {
        deleteApprovals = [];
        archiveApprovals = [];
        return;
      }

      const [deleteRows, archiveRows] = await Promise.all([
        listDeleteApprovals("pending"),
        listEmployeeArchiveApprovals("pending"),
      ]);
      deleteApprovals = deleteRows;
      archiveApprovals = archiveRows;
    } catch (err) {
      loadError = String(err);
      toast.danger(`Failed to load approvals queue: ${String(err)}`);
    } finally {
      loading = false;
    }
  }

  async function approve(row: ApprovalRow) {
    reviewingID = row.id;
    try {
      if (row.kind === "delete") {
        await reviewDeleteApprovalRequest(row.id, "approve", "");
        toast.success("Delete approved and completed");
      } else {
        await reviewEmployeeArchiveRequest(row.id, "approve", "");
        toast.success("Employee archived");
      }
      await load();
    } catch (err) {
      toast.danger(`Approval failed: ${String(err)}`);
    } finally {
      reviewingID = "";
    }
  }

  async function reject(row: ApprovalRow) {
    const isDelete = row.kind === "delete";
    const r = await confirm.askForReason({
      title: isDelete ? "Reject Delete Request" : "Reject Employee Archive",
      message: `This ${isDelete ? "delete" : "employee archive"} request will be rejected. Provide a reason.`,
      reasonLabel: "Reason for rejection",
      reasonRequired: true,
      variant: "danger",
    });
    if (!r.confirmed) return;

    reviewingID = row.id;
    try {
      if (isDelete) {
        await reviewDeleteApprovalRequest(row.id, "reject", r.reason);
        toast.success("Delete request rejected");
      } else {
        await reviewEmployeeArchiveRequest(row.id, "reject", r.reason);
        toast.success("Employee archive request rejected");
      }
      await load();
    } catch (err) {
      toast.danger(`Rejection failed: ${String(err)}`);
    } finally {
      reviewingID = "";
    }
  }

  // Deep-link: hand off to NotificationsScreen, which reads this key on
  // mount, scrolls the matching card into view, and clears it.
  function openInNotifications(row: ApprovalRow) {
    sessionStorage.setItem(pendingApprovalStorageKey, row.id);
    window.dispatchEvent(new CustomEvent("navigateToScreen", { detail: { screen: "notifications" } }));
  }

  onMount(() => {
    void load();
    EventsOn("notifications:new", () => void load());
    EventsOn("notifications:updated", () => void load());
  });

  onDestroy(() => {
    EventsOff("notifications:new", "notifications:updated");
  });
</script>

<PageLayout title="Approvals." subtitle="Everything awaiting your review — persists until resolved, not until read." {embedded}>
  <div class="queue">
    {#if loading}
      <div class="empty">
        <WabiSpinner size="md" tempo="calm" />
        <p>Loading the approvals queue...</p>
      </div>
    {:else if !isAdmin}
      <div class="empty">
        <p>This queue is limited to admin reviewers.</p>
        <p class="hint">Ask an admin if something you requested needs a decision.</p>
      </div>
    {:else if loadError}
      <div class="empty">
        <p>Could not load the approvals queue.</p>
        <p class="hint">{loadError}</p>
        <Button variant="secondary" size="sm" on:click={() => load()}>Retry</Button>
      </div>
    {:else if rows.length === 0}
      <div class="empty">
        <p>Nothing awaits your approval.</p>
      </div>
    {:else}
      <div class="rows">
        {#each rows as row (row.id)}
          <article class="row {row.kind}">
            <div class="row-main">
              <div class="row-heading">
                <span class="pill {row.kind}">{row.subtitle}</span>
                <span class="pill owner">Requested by {row.owner}</span>
                <span class="pill age">{relativeAge(row.createdAt)}</span>
              </div>
              <h3>{row.title}</h3>
              {#if row.reason}
                <p class="reason">{row.reason}</p>
              {/if}
            </div>
            <div class="row-actions">
              <Button
                variant="secondary"
                size="sm"
                on:click={() => openInNotifications(row)}
              >
                Open
              </Button>
              <Button
                variant="danger"
                size="sm"
                on:click={() => approve(row)}
                disabled={reviewingID === row.id}
              >
                {row.kind === "delete" ? "Approve Delete" : "Approve Archive"}
              </Button>
              <Button
                variant="ghost"
                size="sm"
                on:click={() => reject(row)}
                disabled={reviewingID === row.id}
              >
                Reject
              </Button>
            </div>
          </article>
        {/each}
      </div>
    {/if}
  </div>
</PageLayout>

<style>
  .queue {
    display: grid;
    gap: 16px;
  }

  .rows {
    display: grid;
    gap: 12px;
  }

  .row {
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 14px 16px;
    background: var(--surface);
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 16px;
    flex-wrap: wrap;
  }

  .row.delete {
    border-color: rgba(185, 28, 28, 0.28);
  }

  .row.employee_archive {
    border-color: rgba(146, 64, 14, 0.32);
  }

  .row-main {
    display: grid;
    gap: 6px;
    min-width: 240px;
    flex: 1;
  }

  .row-heading {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  .row h3 {
    margin: 0;
    color: var(--text-primary);
  }

  .reason {
    margin: 0;
    color: var(--text-secondary);
    font-size: 13px;
  }

  .pill {
    border-radius: 999px;
    padding: 4px 10px;
    font-size: 11px;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    background: rgba(148, 163, 184, 0.14);
    color: var(--text-secondary);
  }

  .pill.delete {
    background: rgba(254, 226, 226, 0.7);
    color: var(--text-danger);
  }

  .pill.employee_archive {
    background: rgba(254, 243, 199, 0.7);
    color: var(--color-warning, #92400e);
  }

  .row-actions {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  .empty {
    padding: 32px 16px;
    text-align: center;
    color: var(--text-secondary);
    display: grid;
    gap: 8px;
    justify-items: center;
  }

  .hint {
    font-size: 13px;
    opacity: 0.8;
  }
</style>
