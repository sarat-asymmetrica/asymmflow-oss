import { vitePreprocess } from '@sveltejs/vite-plugin-svelte'

export default {
  // Using vitePreprocess for TypeScript 5.x compatibility
  // Modern alternative to deprecated svelte-preprocess
  preprocess: vitePreprocess()
}
