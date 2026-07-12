/**
 * Display primitives registry.
 *
 * Import this in App.svelte and spread into the pages array under group: 'Primitives'.
 *
 * Usage in App.svelte:
 *   import { displayPages } from './pages/primitives/display/registry.js';
 *
 *   const pages: Page[] = [
 *     { id: 'foundations', title: 'Tokens',  group: 'Foundations', component: Foundations },
 *     { id: 'motion',      title: 'Motion',  group: 'Foundations', component: Motion },
 *     ...displayPages,
 *   ];
 */

import type { Component } from 'svelte';

import CardPage    from './CardPage.svelte';
import BadgePage   from './BadgePage.svelte';
import KPIPage     from './KPIPage.svelte';
import TabsPage    from './TabsPage.svelte';
import FeedbackPage from './FeedbackPage.svelte';
import ToastPage   from './ToastPage.svelte';

export interface ShowcasePage {
  id: string;
  title: string;
  group: 'Foundations' | 'Primitives' | 'Patterns' | 'Scenes';
  component: Component;
}

export const displayPages: ShowcasePage[] = [
  { id: 'card',     title: 'Card',             group: 'Primitives', component: CardPage },
  { id: 'badge',    title: 'Badge & Status',   group: 'Primitives', component: BadgePage },
  { id: 'kpi',      title: 'KPI Card',         group: 'Primitives', component: KPIPage },
  { id: 'tabs',     title: 'Tabs',             group: 'Primitives', component: TabsPage },
  { id: 'feedback', title: 'Spinner / Skeleton / Empty', group: 'Primitives', component: FeedbackPage },
  { id: 'toast',    title: 'Toast',            group: 'Primitives', component: ToastPage },
];
