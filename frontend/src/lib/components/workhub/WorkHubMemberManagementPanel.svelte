<script lang="ts">
  import WabiSpinner from "$lib/components/ui/WabiSpinner.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import type { EmployeeProfile, ProjectMember } from "$lib/api/collaboration";

  interface MemberDraft {
    role: string;
    allocation: number;
  }

  interface Props {
    projectMembers: ProjectMember[];
    availableProjectMembers: EmployeeProfile[];
    projectContextLoading: boolean;
    memberSelections: string[];
    newMemberRole: string;
    newMemberAllocation: number;
    savingMember: boolean;
    savingMemberId: string;
    highlightMemberStep: boolean;
    memberStepEl: HTMLElement | null;
    memberDraft: (member: ProjectMember) => MemberDraft;
    onToggleMemberSelection: (employeeID: string) => void;
    onAddMembers: () => void;
    onSetMemberDraftField: (employeeID: string, field: "role" | "allocation", value: string | number) => void;
    onSaveMember: (member: ProjectMember) => void;
  }

  let {
    projectMembers,
    availableProjectMembers,
    projectContextLoading,
    memberSelections = $bindable(),
    newMemberRole = $bindable(),
    newMemberAllocation = $bindable(),
    savingMember,
    savingMemberId,
    highlightMemberStep,
    memberStepEl = $bindable(null),
    memberDraft,
    onToggleMemberSelection,
    onAddMembers,
    onSetMemberDraftField,
    onSaveMember,
  }: Props = $props();
</script>

<div class="subpanel" class:highlight-step={highlightMemberStep} bind:this={memberStepEl}>
  <div class="section-head">
    <h3>Project Members</h3>
    <span class="count">{projectMembers.length}</span>
  </div>
  <p class="admin-hint">Membership drives who can be assigned to this project's tasks — add members here, then give each one a role and allocation below.</p>
  <div class="member-composer">
    <input bind:value={newMemberRole} placeholder="Default role for new members" />
    <input bind:value={newMemberAllocation} type="number" min="0" max="100" placeholder="Allocation %" />
    <Button variant="primary" on:click={onAddMembers} disabled={savingMember || memberSelections.length === 0}>
      {memberSelections.length > 0 ? `Add ${memberSelections.length} Member${memberSelections.length === 1 ? "" : "s"}` : "Add Members"}
    </Button>
  </div>
  {#if projectContextLoading}
    <div class="section-loading">
      <WabiSpinner size="md" tempo="calm" />
      <p>Loading members...</p>
    </div>
  {:else if availableProjectMembers.length > 0}
    <div class="member-picker">
      {#each availableProjectMembers as employee}
        <button
          type="button"
          class="member-pick"
          class:selected={memberSelections.includes(employee.id)}
          onclick={() => onToggleMemberSelection(employee.id)}
        >
          <strong>{employee.full_name}</strong>
          <span>{employee.department || employee.job_title || "Team member"}</span>
        </button>
      {/each}
    </div>
  {/if}
  {#if projectMembers.length === 0}
    <div class="empty compact">No members assigned yet.</div>
  {:else}
    <div class="member-roster">
      {#each projectMembers as member (member.id)}
        <div class="member-row">
          <strong>{member.employee_name || "Unknown employee"}</strong>
          <input
            value={memberDraft(member).role}
            oninput={(event) => onSetMemberDraftField(member.employee_id, "role", (event.currentTarget as HTMLInputElement).value)}
            placeholder="Role"
          />
          <input
            type="number"
            min="0"
            max="100"
            value={memberDraft(member).allocation}
            oninput={(event) => onSetMemberDraftField(member.employee_id, "allocation", Number((event.currentTarget as HTMLInputElement).value))}
            placeholder="Allocation %"
          />
          <Button
            variant="secondary"
            size="sm"
            on:click={() => onSaveMember(member)}
            disabled={savingMemberId === member.employee_id}
          >
            {savingMemberId === member.employee_id ? "Saving..." : "Save"}
          </Button>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  h3,
  p {
    margin: 0;
  }

  .section-head {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    align-items: center;
  }

  .count {
    color: var(--text-secondary);
  }

  input,
  button {
    font: inherit;
  }

  input {
    width: 100%;
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 10px 12px;
    background: var(--bg-base);
  }

  .subpanel {
    border-top: 1px solid var(--border);
    padding-top: 16px;
    display: grid;
    gap: 12px;
    align-content: start;
    grid-auto-rows: max-content;
  }

  .admin-hint {
    margin: 0;
    color: var(--text-secondary);
    font-size: 0.88rem;
    line-height: 1.5;
  }

  .member-composer {
    display: grid;
    grid-template-columns: 2fr 1fr auto;
    gap: 10px;
    align-items: center;
  }

  .highlight-step {
    animation: highlight-pulse 2s ease-in-out 2;
  }

  @keyframes highlight-pulse {
    0%, 100% { box-shadow: none; }
    50% { box-shadow: 0 0 0 2px var(--onyx); }
  }

  .member-picker {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 10px;
  }

  .member-pick {
    width: 100%;
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 12px;
    background: color-mix(in srgb, var(--bg-base) 92%, #dbeafe 8%);
    text-align: left;
    display: grid;
    gap: 4px;
  }

  .member-pick.selected {
    border-color: var(--onyx);
    box-shadow: 0 0 0 1px var(--onyx);
    background: color-mix(in srgb, var(--bg-base) 82%, #dbeafe 18%);
  }

  .member-pick span {
    color: var(--text-secondary);
    font-size: 13px;
  }

  .member-roster {
    display: grid;
    gap: 8px;
  }

  .member-row {
    display: grid;
    grid-template-columns: 2fr 1.5fr 1fr auto;
    gap: 8px;
    align-items: center;
    padding: 8px 10px;
    border: 1px solid var(--border);
    border-radius: 10px;
    background: var(--bg-base);
  }

  .member-row input {
    width: 100%;
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 6px 8px;
    background: var(--surface);
    font: inherit;
  }

  .section-loading {
    display: grid;
    place-items: center;
    gap: 10px;
    padding: 24px;
    color: var(--text-secondary);
    text-align: center;
  }

  .empty {
    padding: 24px 8px;
    color: var(--text-secondary);
    text-align: center;
  }

  .empty.compact {
    padding: 12px 8px;
  }
</style>
