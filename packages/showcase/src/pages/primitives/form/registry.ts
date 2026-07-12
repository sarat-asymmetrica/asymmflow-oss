/**
 * Form primitive showcase page registry.
 * Import this in App.svelte and spread into the pages array.
 *
 * App.svelte addition:
 *   import { formPages } from './pages/primitives/form/registry.js';
 *   // then spread: ...formPages  into the pages array (group is already set to 'Primitives')
 */

import type { Component } from 'svelte';
import ButtonPage from './ButtonPage.svelte';
import InputPage from './InputPage.svelte';
import SelectionPage from './SelectionPage.svelte';
import FormGroupPage from './FormGroupPage.svelte';

export interface ShowcasePage {
  id: string;
  title: string;
  group: 'Foundations' | 'Primitives' | 'Patterns' | 'Scenes';
  component: Component;
}

export const formPages: ShowcasePage[] = [
  { id: 'button', title: 'Button', group: 'Primitives', component: ButtonPage },
  { id: 'input', title: 'Input · Textarea · Currency', group: 'Primitives', component: InputPage },
  { id: 'selection', title: 'Select · Checkbox · Toggle', group: 'Primitives', component: SelectionPage },
  { id: 'form-group', title: 'FormGroup', group: 'Primitives', component: FormGroupPage },
];
