/**
 * @asymmflow/charts — data visualization for the calm center.
 *
 * Headless d3 math (scales/shapes), token-driven SVG rendering, geodesic
 * value transitions. Series colors come from --af-chart-1..8 — a theme
 * change re-skins every chart live.
 *
 * Constitution: packages/DESIGN_CONSTITUTION.md.
 */

// Core stage
export { default as ChartFrame } from './ChartFrame.svelte';
export type { ChartContext, ChartMargin } from './ChartFrame.svelte';
export { default as Axis } from './Axis.svelte';
export { default as Legend } from './Legend.svelte';
export type { LegendItem } from './Legend.svelte';
export { default as ChartTooltip } from './ChartTooltip.svelte';

// Timeseries family
export { default as Sparkline } from './timeseries/Sparkline.svelte';
export { default as LineChart } from './timeseries/LineChart.svelte';
export { default as AreaChart } from './timeseries/AreaChart.svelte';

// Categorical family
export { default as BarChart } from './categorical/BarChart.svelte';
export { default as DonutChart } from './categorical/DonutChart.svelte';

// ERP showpieces
export { default as CashflowBridge } from './erp/CashflowBridge.svelte';
export { default as AgingHeatmap } from './erp/AgingHeatmap.svelte';
export { default as PipelineFunnel } from './erp/PipelineFunnel.svelte';

// Headless math
export * from './scales.js';
export * from './format.js';
export { seriesToken, seriesColor, readChartPalette, CHART_SERIES_COUNT } from './palette.js';
export { createValuesTween } from './valuesTween.js';
export type { ValuesTween, TransitionOptions } from './valuesTween.js';
