/**
 * Data primitive showcase page registry.
 * Import this in App.svelte and spread into the pages array.
 *
 * App.svelte addition:
 *   import { dataPages } from './pages/primitives/data/registry.js';
 *   // then spread: ...dataPages  into the pages array with group: 'Primitives'
 */

import type { Component } from 'svelte';
import DataTablePage from './DataTablePage.svelte';
import PaginationPage from './PaginationPage.svelte';

export interface ShowcasePage {
  id: string;
  title: string;
  group: 'Foundations' | 'Primitives' | 'Patterns' | 'Scenes';
  component: Component;
}

export const dataPages: ShowcasePage[] = [
  {
    id: 'data-table',
    title: 'DataTable',
    group: 'Primitives',
    component: DataTablePage,
  },
  {
    id: 'pagination',
    title: 'Pagination',
    group: 'Primitives',
    component: PaginationPage,
  },
];
