/**
 * Categorical chart showcase page registry.
 *
 * The orchestrator spreads categoricalChartPages into the App.svelte pages
 * array under the 'Charts' group.
 */

import type { Component } from 'svelte';
import BarChartPage from './BarChartPage.svelte';
import DonutChartPage from './DonutChartPage.svelte';

interface ChartShowcasePage {
  id: string;
  title: string;
  group: 'Charts';
  component: Component;
}

export const categoricalChartPages: ChartShowcasePage[] = [
  {
    id: 'bar-chart',
    title: 'BarChart',
    group: 'Charts',
    component: BarChartPage,
  },
  {
    id: 'donut-chart',
    title: 'DonutChart',
    group: 'Charts',
    component: DonutChartPage,
  },
];
