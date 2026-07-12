import {defineConfig} from 'vite'
import {svelte} from '@sveltejs/vite-plugin-svelte'
import path from 'path'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [svelte()],
  resolve: {
    alias: {
      $lib: path.resolve('./src/lib')
    }
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          const normalized = id.split(path.sep).join('/');

          if (normalized.includes('/node_modules/')) {
            if (normalized.includes('/chart.js/') || normalized.includes('/svelte-chartjs/')) {
              return 'vendor-charts';
            }
            if (normalized.includes('/d3/')) {
              return 'vendor-d3';
            }
            if (
              normalized.includes('/three/') ||
              normalized.includes('/gsap/') ||
              normalized.includes('/lenis/') ||
              normalized.includes('/@studio-freight/lenis/') ||
              normalized.includes('/@types/three/')
            ) {
              return 'vendor-motion-3d';
            }
            if (normalized.includes('/svelte/') || normalized.includes('/tslib/')) {
              return 'vendor-svelte';
            }
            return 'vendor-misc';
          }
        },
      },
    },
  },
})
