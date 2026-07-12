/**
 * Timeseries chart showcase page registry.
 * Import this in App.svelte and spread into the pages array.
 *
 * App.svelte addition:
 *   import { timeseriesChartPages } from './pages/charts/timeseries/registry.js';
 *   // spread: ...timeseriesChartPages into the pages array
 */

import type { Component } from 'svelte';
import SparklinePage from './SparklinePage.svelte';
import LineChartPage from './LineChartPage.svelte';
import AreaChartPage from './AreaChartPage.svelte';

export interface ChartShowcasePage {
  id: string;
  title: string;
  group: 'Charts';
  component: Component;
}

export const timeseriesChartPages: ChartShowcasePage[] = [
  {
    id: 'sparkline',
    title: 'Sparkline',
    group: 'Charts',
    component: SparklinePage as Component,
  },
  {
    id: 'line-chart',
    title: 'LineChart',
    group: 'Charts',
    component: LineChartPage as Component,
  },
  {
    id: 'area-chart',
    title: 'AreaChart',
    group: 'Charts',
    component: AreaChartPage as Component,
  },
];
