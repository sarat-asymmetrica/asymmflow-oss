/**
 * @asymmflow/patterns — composed UX workflows.
 * Imports only from: @asymmflow/ui, @asymmflow/motion, @asymmflow/tokens.
 * Constitution: packages/DESIGN_CONSTITUTION.md
 */

// ── Workflows ────────────────────────────────────────────────────────────────
export { default as CommandBar } from './CommandBar.svelte';
export type { CommandBarProps, Command } from './CommandBar.svelte';

export { default as DataShell } from './DataShell.svelte';
export type { DataShellProps } from './DataShell.svelte';

export { default as PageHeader } from './PageHeader.svelte';
export type { PageHeaderProps, BreadcrumbItem } from './PageHeader.svelte';

export { default as SplitDetail } from './SplitDetail.svelte';
export type { SplitDetailProps } from './SplitDetail.svelte';
