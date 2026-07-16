/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import path from 'path'

export default defineConfig({
  plugins: [svelte()],
  resolve: {
    alias: {
      $kernel: path.resolve('./src/kernel'),
      $screens: path.resolve('./src/screens'),
      // One-source law: the semantic token layer (tokens/fonts/sounds) lives
      // here since the K6 flip; $tokens is the only import path for it.
      $tokens: path.resolve('./src/assets'),
      // Generated Wails bindings (same generation the old frontend uses).
      $wails: path.resolve('./wailsjs'),
    },
  },
  server: {
    port: 5175,
    fs: {
      // Repo-root allowance kept for wails dev serving from the repo root.
      allow: ['..'],
    },
  },
  test: {
    environment: 'node',
    include: ['tests/**/*.test.ts'],
  },
})
