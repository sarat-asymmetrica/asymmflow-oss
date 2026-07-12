import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright E2E Configuration
 * ACE Engine Frontend Test Suite
 *
 * Framework: Three-Regime QA (Stabilization 100%, Optimization 85%, Exploration 70%)
 * Target: Acme Instrumentation Deployment (National Day → Nigeria Launch)
 */

export default defineConfig({
  testDir: './tests',

  // Timeout for individual tests (30 seconds = Nielsen flow threshold)
  timeout: 30 * 1000,

  // Test retry strategy (Harmonic backoff: 1×, 2×, 4×, 8×, 16×)
  retries: process.env.CI ? 2 : 0,

  // Workers (parallel execution)
  workers: process.env.CI ? 1 : 4,

  // Reporter configuration (detailed + HTML)
  reporter: [
    ['html'],
    ['json', { outputFile: 'test-results/results.json' }],
    ['junit', { outputFile: 'test-results/results.xml' }],
    ['dot'],
  ],

  // Web server configuration (start dev server before tests)
  webServer: {
    command: 'npm run dev -- --host 127.0.0.1',
    url: 'http://127.0.0.1:5173',
    reuseExistingServer: !process.env.CI,
    timeout: 120 * 1000,
  },

  // Global test configuration
  use: {
    // Base URL for all tests
    baseURL: 'http://127.0.0.1:5173',

    // Screenshot on failure (helps with debugging)
    screenshot: 'only-on-failure',

    // Video on failure (UX Sonar data collection)
    video: 'retain-on-failure',

    // Trace on first retry (debugging support)
    trace: 'on-first-retry',

    // Slow down interactions (100ms = Nielsen threshold for perceivable delay)
    slowMo: process.env.SLOW_MO ? 100 : 0,
  },

  // Projects (browser matrix)
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },

    // Mobile testing (responsive design validation)
    {
      name: 'Mobile Chrome',
      use: { ...devices['Pixel 5'] },
    },
    {
      name: 'Mobile Safari',
      use: { ...devices['iPhone 12'] },
    },
  ],

  // Output directory
  outputDir: 'test-results/',
});
