<script lang="ts">
  /**
   * Status Badge Component with Smart Mapping
   * Automatically determines color from status text
   *
   * Status Mapping:
   * - Draft/New/Open → gray (default)
   * - Sent/Quoted/Processing/In Progress → blue (info)
   * - Won/Paid/Delivered/Approved/Complete → green (success)
   * - Lost/Overdue/Rejected/Cancelled → red (danger)
   * - Pending/Partial/Review → yellow (warning)
   */

  import Badge from './Badge.svelte';

  interface Props {
    status: string;
    size?: 'sm' | 'md';
  }

  let { status, size = 'md' }: Props = $props();

  type BadgeVariant = 'default' | 'success' | 'warning' | 'danger' | 'info';

  // Smart status mapping
  function getVariant(status: string): BadgeVariant {
    const normalized = status.toLowerCase().trim();

    // Success states
    if (
      normalized.includes('won') ||
      normalized.includes('paid') ||
      normalized.includes('delivered') ||
      normalized.includes('approved') ||
      normalized.includes('complete') ||
      normalized.includes('active') ||
      normalized.includes('success')
    ) {
      return 'success';
    }

    // Danger states
    if (
      normalized.includes('lost') ||
      normalized.includes('overdue') ||
      normalized.includes('rejected') ||
      normalized.includes('cancelled') ||
      normalized.includes('failed') ||
      normalized.includes('error')
    ) {
      return 'danger';
    }

    // Warning states
    if (
      normalized.includes('pending') ||
      normalized.includes('partial') ||
      normalized.includes('review') ||
      normalized.includes('hold') ||
      normalized.includes('awaiting')
    ) {
      return 'warning';
    }

    // Info states
    if (
      normalized.includes('sent') ||
      normalized.includes('quoted') ||
      normalized.includes('processing') ||
      normalized.includes('progress') ||
      normalized.includes('in review')
    ) {
      return 'info';
    }

    // Default states (Draft, New, Open, etc.)
    return 'default';
  }

  let variant = $derived(getVariant(status));
</script>

<Badge {variant} {size}>
  {status}
</Badge>
