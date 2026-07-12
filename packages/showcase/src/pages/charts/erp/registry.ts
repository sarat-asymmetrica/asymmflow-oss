/**
 * ERP chart showcase page registry.
 *
 * Import this in the showcase App.svelte and spread into the pages array:
 *   import { erpChartPages } from './pages/charts/erp/registry.js';
 *   // then spread into your pages array
 */

import type { Component } from 'svelte';
import CashflowBridgePage from './CashflowBridgePage.svelte';
import AgingHeatmapPage from './AgingHeatmapPage.svelte';
import PipelineFunnelPage from './PipelineFunnelPage.svelte';

interface ShowcasePage {
  id: string;
  title: string;
  group: 'Charts';
  component: Component;
}

export const erpChartPages: ShowcasePage[] = [
  {
    id: 'cashflow-bridge',
    title: 'CashflowBridge',
    group: 'Charts',
    component: CashflowBridgePage as Component,
  },
  {
    id: 'aging-heatmap',
    title: 'AgingHeatmap',
    group: 'Charts',
    component: AgingHeatmapPage as Component,
  },
  {
    id: 'pipeline-funnel',
    title: 'PipelineFunnel',
    group: 'Charts',
    component: PipelineFunnelPage as Component,
  },
];
