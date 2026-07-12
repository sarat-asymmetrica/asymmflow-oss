import type { Component } from 'svelte';
import ModalPage from './ModalPage.svelte';
import DrawerPage from './DrawerPage.svelte';
import MenuPage from './MenuPage.svelte';

export interface ShowcasePage {
  id: string;
  title: string;
  group: 'Foundations' | 'Primitives' | 'Patterns' | 'Scenes';
  component: Component;
}

export const overlayPages: ShowcasePage[] = [
  { id: 'overlay-modal',   title: 'Modal',    group: 'Primitives', component: ModalPage },
  { id: 'overlay-drawer',  title: 'Drawer',   group: 'Primitives', component: DrawerPage },
  { id: 'overlay-menus',   title: 'Menus',    group: 'Primitives', component: MenuPage },
];
