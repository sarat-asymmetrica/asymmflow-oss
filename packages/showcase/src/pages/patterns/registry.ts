/**
 * Patterns showcase page registry.
 *
 * Wire into App.svelte:
 *   import { patternPages } from './pages/patterns/registry';
 *   // then spread: ...patternPages  into the pages array
 *
 * The ShowcasePage type is defined (and re-exported) by the data registry.
 */

import type { ShowcasePage } from '../primitives/data/registry';
import CommandBarPage from './CommandBarPage.svelte';
import DataShellPage from './DataShellPage.svelte';
import CustomersProofPage from './CustomersProofPage.svelte';

export const patternPages: ShowcasePage[] = [
  {
    id: 'command-bar',
    title: 'CommandBar',
    group: 'Patterns',
    component: CommandBarPage,
  },
  {
    id: 'data-shell',
    title: 'DataShell',
    group: 'Patterns',
    component: DataShellPage,
  },
  {
    id: 'customers-proof',
    title: 'Customers (Proof)',
    group: 'Patterns',
    component: CustomersProofPage,
  },
];
