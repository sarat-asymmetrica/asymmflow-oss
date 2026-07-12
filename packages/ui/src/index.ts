/**
 * @asymmflow/ui — the calm center.
 * Svelte 5 primitives, token-driven, zero app coupling.
 * Constitution: packages/DESIGN_CONSTITUTION.md
 */

// ── Form ────────────────────────────────────────────────────────────────
export { default as Button } from './form/Button.svelte';
export type { ButtonProps } from './form/Button.svelte';
export { default as Input } from './form/Input.svelte';
export type { InputProps } from './form/Input.svelte';
export { default as Textarea } from './form/Textarea.svelte';
export type { TextareaProps } from './form/Textarea.svelte';
export { default as Select } from './form/Select.svelte';
export type { SelectProps, SelectOption } from './form/Select.svelte';
export { default as Checkbox } from './form/Checkbox.svelte';
export type { CheckboxProps } from './form/Checkbox.svelte';
export { default as Toggle } from './form/Toggle.svelte';
export type { ToggleProps } from './form/Toggle.svelte';
export { default as CurrencyInput } from './form/CurrencyInput.svelte';
export type { CurrencyInputProps } from './form/CurrencyInput.svelte';
export { default as FormGroup } from './form/FormGroup.svelte';
export type { FormGroupProps } from './form/FormGroup.svelte';

// ── Display ─────────────────────────────────────────────────────────────
export { default as Card } from './display/Card.svelte';
export type { CardProps } from './display/Card.svelte';
export { default as Badge } from './display/Badge.svelte';
export type { BadgeProps } from './display/Badge.svelte';
export { default as StatusBadge } from './display/StatusBadge.svelte';
export type { StatusBadgeProps, StatusKind, StatusEmphasis } from './display/StatusBadge.svelte';
export { default as KPICard } from './display/KPICard.svelte';
export type { KPICardProps } from './display/KPICard.svelte';
export { default as Tabs } from './display/Tabs.svelte';
export type { TabsProps, TabItem } from './display/Tabs.svelte';
export { default as Spinner } from './display/Spinner.svelte';
export type { SpinnerProps } from './display/Spinner.svelte';
export { default as Skeleton } from './display/Skeleton.svelte';
export type { SkeletonProps } from './display/Skeleton.svelte';
export { default as EmptyState } from './display/EmptyState.svelte';
export type { EmptyStateProps } from './display/EmptyState.svelte';
export { default as ToastContainer } from './display/ToastContainer.svelte';
export { toast } from './display/toast.svelte.js';
export type { ToastItem, ToastSeverity, ToastOptions } from './display/toast.svelte.js';

// ── Data ────────────────────────────────────────────────────────────────
export { default as DataTable } from './data/DataTable.svelte';
export { default as DataToolbar } from './data/DataToolbar.svelte';
export { default as Pagination } from './data/Pagination.svelte';
export type {
  Column,
  SortState,
  SortDirection,
  Alignment,
  CellContext,
  DataTableProps,
} from './data/DataTable.types.js';

// ── Overlay ─────────────────────────────────────────────────────────────
export { default as Modal } from './overlay/Modal.svelte';
export type { ModalProps } from './overlay/Modal.svelte';
export { default as Drawer } from './overlay/Drawer.svelte';
export type { DrawerProps } from './overlay/Drawer.svelte';
export { default as Dropdown } from './overlay/Dropdown.svelte';
export type { DropdownProps, DropdownItem } from './overlay/Dropdown.svelte';
export { default as Tooltip } from './overlay/Tooltip.svelte';
export type { TooltipProps } from './overlay/Tooltip.svelte';
export { default as ConfirmDialog } from './overlay/ConfirmDialog.svelte';
export type { ConfirmDialogProps } from './overlay/ConfirmDialog.svelte';

// ── Actions ─────────────────────────────────────────────────────────────
export { focusTrap } from './actions/focusTrap.js';
export type { FocusTrapOptions } from './actions/focusTrap.js';
export { clickOutside } from './actions/clickOutside.js';
export { portal } from './overlay/portal.js';
