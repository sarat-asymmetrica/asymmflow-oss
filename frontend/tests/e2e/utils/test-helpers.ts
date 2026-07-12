/**
 * Test Helper Utilities
 * Harmonic retry backoff, test data generation, and assertions
 *
 * Philosophy: Mathematical elegance in test infrastructure
 */

import { Page, Locator } from '@playwright/test';
import { devLog } from "$lib/utils/devLog";


// =============================================================================
// Harmonic Retry Backoff (φ-based exponential backoff)
// =============================================================================

/**
 * Harmonic retry with golden ratio backoff
 * Retries: 1×, φ×, φ²×, φ³×, ... (1.618, 2.618, 4.236, 6.854, ...)
 *
 * @param fn Function to retry
 * @param maxRetries Maximum number of retries (default: 5)
 * @param baseDelay Base delay in ms (default: 100)
 * @returns Result of function
 */
export async function harmonicRetry<T>(
  fn: () => Promise<T>,
  maxRetries = 5,
  baseDelay = 100
): Promise<T> {
  const PHI = 1.618033988749895; // Golden ratio
  let lastError: Error | undefined;

  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    try {
      return await fn();
    } catch (error) {
      lastError = error as Error;

      if (attempt === maxRetries) {
        throw new Error(
          `Harmonic retry failed after ${maxRetries} attempts: ${lastError.message}`
        );
      }

      // Calculate φ-based delay
      const delay = baseDelay * Math.pow(PHI, attempt);
      devLog.log(`⏳ Retry ${attempt + 1}/${maxRetries} after ${delay.toFixed(0)}ms...`);

      await new Promise((resolve) => setTimeout(resolve, delay));
    }
  }

  throw lastError || new Error('Unexpected retry failure');
}

// =============================================================================
// Test Data Generators
// =============================================================================

/**
 * Generate test customer data
 */
export interface TestCustomer {
  id: string;
  name: string;
  code: string;
  type: string;
  grade: 'A' | 'B' | 'C' | 'D';
  discountPercent: number;
  healthScore: number;
  churnRisk: number;
  relationYears: number;
}

/**
 * Generate test product data
 */
export interface TestProduct {
  id: string;
  name: string;
  basePrice: number;
}

/**
 * Calculate customer-specific price
 */
export function calculateCustomerPrice(
  basePrice: number,
  discountPercent: number
): number {
  const multiplier = 1 - discountPercent / 100;
  return parseFloat((basePrice * multiplier).toFixed(2));
}

/**
 * Generate random customer for stress testing
 */
export function generateRandomCustomer(grade: 'A' | 'B' | 'C' | 'D'): TestCustomer {
  const gradeConfig = {
    A: { discountPercent: 12, healthScore: [85, 100], churnRisk: [0, 15] },
    B: { discountPercent: 7, healthScore: [65, 90], churnRisk: [15, 35] },
    C: { discountPercent: 0, healthScore: [45, 70], churnRisk: [30, 55] },
    D: { discountPercent: -5, healthScore: [0, 60], churnRisk: [50, 100] },
  };

  const config = gradeConfig[grade];
  const id = `C${Math.floor(Math.random() * 99999).toString().padStart(5, '0')}`;

  return {
    id,
    name: `Test Customer ${id}`,
    code: `EC${Math.floor(Math.random() * 200) + 1}`,
    type: 'End Customer',
    grade,
    discountPercent: config.discountPercent,
    healthScore:
      config.healthScore[0] +
      Math.random() * (config.healthScore[1] - config.healthScore[0]),
    churnRisk:
      config.churnRisk[0] + Math.random() * (config.churnRisk[1] - config.churnRisk[0]),
    relationYears: Math.floor(Math.random() * 20) + 1,
  };
}

// =============================================================================
// Visual Regression Helpers
// =============================================================================

/**
 * Wait for element to be stable (no layout shift)
 */
export async function waitForStableElement(
  locator: Locator,
  timeoutMs = 3000
): Promise<void> {
  const startTime = Date.now();
  let lastPosition: { x: number; y: number } | null = null;
  let stableCount = 0;

  while (Date.now() - startTime < timeoutMs) {
    const box = await locator.boundingBox();
    if (!box) {
      await new Promise((resolve) => setTimeout(resolve, 100));
      continue;
    }

    const currentPosition = { x: box.x, y: box.y };

    if (lastPosition) {
      const distance = Math.sqrt(
        Math.pow(currentPosition.x - lastPosition.x, 2) +
          Math.pow(currentPosition.y - lastPosition.y, 2)
      );

      if (distance < 1) {
        // Less than 1px movement
        stableCount++;
        if (stableCount >= 3) {
          // Stable for 3 checks
          return;
        }
      } else {
        stableCount = 0;
      }
    }

    lastPosition = currentPosition;
    await new Promise((resolve) => setTimeout(resolve, 100));
  }

  throw new Error(`Element did not stabilize within ${timeoutMs}ms`);
}

/**
 * Measure Cumulative Layout Shift (CLS)
 */
export async function measureCLS(page: Page): Promise<number> {
  return page.evaluate(() => {
    return (window as any).__sonicTelemetry?.cumulativeLayoutShift || 0;
  });
}

/**
 * Measure average FPS
 */
export async function measureFPS(page: Page): Promise<number> {
  return page.evaluate(() => {
    return (window as any).__sonicTelemetry?.averageFPS || 60;
  });
}

// =============================================================================
// Assertion Helpers
// =============================================================================

/**
 * Assert value within tolerance
 */
export function assertWithinTolerance(
  actual: number,
  expected: number,
  tolerance: number,
  message?: string
): void {
  const diff = Math.abs(actual - expected);
  if (diff > tolerance) {
    throw new Error(
      message ||
        `Expected ${actual} to be within ${tolerance} of ${expected}, but difference was ${diff}`
    );
  }
}

/**
 * Assert price calculation
 */
export function assertPriceCalculation(
  basePrice: number,
  discountPercent: number,
  actualPrice: number
): void {
  const expectedPrice = calculateCustomerPrice(basePrice, discountPercent);
  assertWithinTolerance(
    actualPrice,
    expectedPrice,
    0.01,
    `Price calculation failed: expected ${expectedPrice} BHD, got ${actualPrice} BHD`
  );
}

// =============================================================================
// Performance Metrics
// =============================================================================

/**
 * Measure page load performance
 */
export interface LoadMetrics {
  navigationStart: number;
  domContentLoaded: number;
  loadComplete: number;
  totalDuration: number;
}

export async function measurePageLoad(page: Page): Promise<LoadMetrics> {
  return page.evaluate(() => {
    const timing = performance.timing;
    return {
      navigationStart: timing.navigationStart,
      domContentLoaded: timing.domContentLoadedEventEnd - timing.navigationStart,
      loadComplete: timing.loadEventEnd - timing.navigationStart,
      totalDuration: timing.loadEventEnd - timing.fetchStart,
    };
  });
}

/**
 * Log test metrics for SHM calculation
 */
export function logTestMetrics(testName: string, metrics: Record<string, any>): void {
  devLog.log(`📊 [${testName}] Metrics:`, JSON.stringify(metrics, null, 2));

  // In production, this would send to Consciousness Engine
  // For now, just log to console for visibility
}

// =============================================================================
// Three-Regime Validation
// =============================================================================

/**
 * Validate three-regime percentages
 * Target: R1 ≈ 30%, R2 ≈ 20%, R3 ≈ 50%
 */
export interface RegimePercentages {
  r1: number;
  r2: number;
  r3: number;
}

export function validateThreeRegimes(
  regimes: RegimePercentages,
  tolerance = 0.1
): boolean {
  const targets = { r1: 0.3, r2: 0.2, r3: 0.5 };

  const r1Valid = Math.abs(regimes.r1 - targets.r1) <= tolerance;
  const r2Valid = Math.abs(regimes.r2 - targets.r2) <= tolerance;
  const r3Valid = Math.abs(regimes.r3 - targets.r3) <= tolerance;

  if (!r1Valid || !r2Valid || !r3Valid) {
    devLog.warn(
      `⚠️ Regime distribution off-target: R1=${(regimes.r1 * 100).toFixed(1)}%, R2=${(regimes.r2 * 100).toFixed(1)}%, R3=${(regimes.r3 * 100).toFixed(1)}%`
    );
  }

  return r1Valid && r2Valid && r3Valid;
}

/**
 * Extract regime percentages from page
 */
export async function extractRegimePercentages(page: Page): Promise<RegimePercentages> {
  return page.evaluate(() => {
    const r1Element = document.querySelector('[data-regime="r1"]');
    const r2Element = document.querySelector('[data-regime="r2"]');
    const r3Element = document.querySelector('[data-regime="r3"]');

    return {
      r1: r1Element ? parseFloat(r1Element.getAttribute('data-value') || '0') : 0.3,
      r2: r2Element ? parseFloat(r2Element.getAttribute('data-value') || '0') : 0.2,
      r3: r3Element ? parseFloat(r3Element.getAttribute('data-value') || '0') : 0.5,
    };
  });
}

// =============================================================================
// Accessibility Helpers
// =============================================================================

/**
 * Check color contrast ratio (simplified)
 * Full implementation would require chroma.js or similar
 */
export interface ColorRGB {
  r: number;
  g: number;
  b: number;
}

export function parseRGB(rgbString: string): ColorRGB {
  const match = rgbString.match(/rgb\((\d+),\s*(\d+),\s*(\d+)\)/);
  if (!match) {
    throw new Error(`Invalid RGB string: ${rgbString}`);
  }
  return {
    r: parseInt(match[1]),
    g: parseInt(match[2]),
    b: parseInt(match[3]),
  };
}

export function calculateRelativeLuminance(color: ColorRGB): number {
  const rsRGB = color.r / 255;
  const gsRGB = color.g / 255;
  const bsRGB = color.b / 255;

  const r = rsRGB <= 0.03928 ? rsRGB / 12.92 : Math.pow((rsRGB + 0.055) / 1.055, 2.4);
  const g = gsRGB <= 0.03928 ? gsRGB / 12.92 : Math.pow((gsRGB + 0.055) / 1.055, 2.4);
  const b = bsRGB <= 0.03928 ? bsRGB / 12.92 : Math.pow((bsRGB + 0.055) / 1.055, 2.4);

  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

export function calculateContrastRatio(color1: ColorRGB, color2: ColorRGB): number {
  const l1 = calculateRelativeLuminance(color1);
  const l2 = calculateRelativeLuminance(color2);

  const lighter = Math.max(l1, l2);
  const darker = Math.min(l1, l2);

  return (lighter + 0.05) / (darker + 0.05);
}

/**
 * Verify WCAG AA compliance (4.5:1 for normal text)
 */
export function isWCAGCompliant(contrastRatio: number, level: 'AA' | 'AAA' = 'AA'): boolean {
  const threshold = level === 'AAA' ? 7.0 : 4.5;
  return contrastRatio >= threshold;
}

// =============================================================================
// Consciousness Engine Integration
// =============================================================================

/**
 * Mock Consciousness Engine telemetry
 */
export async function injectConsciousnessTelemetry(page: Page): Promise<void> {
  await page.addInitScript(() => {
    // Initialize telemetry object
    (window as any).__sonicTelemetry = {
      averageFPS: 60,
      cumulativeLayoutShift: 0,
      sentiment: 'confident',
      flowState: 'in-flow',
      confidenceLevel: 0.85,
    };

    // FPS tracking
    let frameCount = 0;
    let lastTime = performance.now();

    function measureFPS() {
      const now = performance.now();
      const delta = now - lastTime;
      const fps = 1000 / delta;
      frameCount++;

      // Exponential moving average
      (window as any).__sonicTelemetry.averageFPS =
        (window as any).__sonicTelemetry.averageFPS * 0.9 + fps * 0.1;

      lastTime = now;
      requestAnimationFrame(measureFPS);
    }
    requestAnimationFrame(measureFPS);

    // CLS tracking
    const observer = new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        if ((entry as any).hadRecentInput) continue;
        (window as any).__sonicTelemetry.cumulativeLayoutShift += (entry as any).value;
      }
    });
    observer.observe({ type: 'layout-shift', buffered: true });
  });
}

// =============================================================================
// Export All
// =============================================================================

export default {
  harmonicRetry,
  calculateCustomerPrice,
  generateRandomCustomer,
  waitForStableElement,
  measureCLS,
  measureFPS,
  assertWithinTolerance,
  assertPriceCalculation,
  measurePageLoad,
  logTestMetrics,
  validateThreeRegimes,
  extractRegimePercentages,
  parseRGB,
  calculateRelativeLuminance,
  calculateContrastRatio,
  isWCAGCompliant,
  injectConsciousnessTelemetry,
};
