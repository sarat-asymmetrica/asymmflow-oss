/**
 * Scenes showcase page registry.
 *
 * Add this import to App.svelte and spread scenePages into the pages array:
 *
 *   import { scenePages } from './pages/scenes/registry';
 *   // in pages array:
 *   ...scenePages,
 */

import type { Component } from 'svelte';
import GlyphPage from './GlyphPage.svelte';
import AmbientPage from './AmbientPage.svelte';
import ThemeForgePage from './ThemeForgePage.svelte';
import CeremonyPage from './CeremonyPage.svelte';

export interface ShowcasePage {
  id: string;
  title: string;
  group: 'Foundations' | 'Primitives' | 'Patterns' | 'Scenes';
  component: Component;
}

export const scenePages: ShowcasePage[] = [
  {
    id: 'glyph-mark',
    title: 'GlyphMark',
    group: 'Scenes',
    component: GlyphPage as Component,
  },
  {
    id: 'ambient-field',
    title: 'AmbientField',
    group: 'Scenes',
    component: AmbientPage as Component,
  },
  {
    id: 'theme-forge',
    title: 'Theme Forge',
    group: 'Scenes',
    component: ThemeForgePage as Component,
  },
  {
    id: 'login-ceremony',
    title: 'Login Ceremony',
    group: 'Scenes',
    component: CeremonyPage as Component,
  },
];
