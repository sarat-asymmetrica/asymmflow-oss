import { Page, Locator } from '@playwright/test';

/**
 * Page Object: Customer 360 View
 * Encapsulates all interactions with the Customer360 screen
 *
 * Usage:
 *   const page360 = new Customer360Page(page);
 *   await page360.goto('C01011');
 *   await page360.verifyGrade('Grade A');
 */
export class Customer360Page {
  readonly page: Page;

  // Locators
  readonly customerNameLocator: Locator;
  readonly customerGradeLocator: Locator;
  readonly healthScoreLocator: Locator;
  readonly churnRiskLocator: Locator;
  readonly productPriceLocator: (productId: string) => Locator;
  readonly orderHistoryTable: Locator;
  readonly confidenceMeterLocator: Locator;
  readonly sentimentBadgeLocator: Locator;
  readonly flowIndicatorLocator: Locator;

  constructor(page: Page) {
    this.page = page;

    // Semantic locators (data-testid preferred for reliability)
    this.customerNameLocator = page.locator('[data-testid="customer-name"]');
    this.customerGradeLocator = page.locator('[data-testid="customer-grade"]');
    this.healthScoreLocator = page.locator('[data-testid="health-score"]');
    this.churnRiskLocator = page.locator('[data-testid="churn-risk"]');
    this.orderHistoryTable = page.locator('[data-testid="order-history-table"]');
    this.confidenceMeterLocator = page.locator('[data-testid="confidence-meter"]');
    this.sentimentBadgeLocator = page.locator('[data-testid="sentiment-badge"]');
    this.flowIndicatorLocator = page.locator('[data-testid="flow-indicator"]');
  }

  /**
   * Navigate to Customer360 for a specific customer
   * @param customerId e.g., "C01011" (Northstar Trading)
   */
  async goto(customerId: string) {
    await this.page.goto(`/customer/${customerId}`);
    // Wait for main content to load (consciousness data collection)
    await this.customerNameLocator.waitFor({ state: 'visible', timeout: 5000 });
  }

  /**
   * Get customer name (assertion helper)
   */
  async getCustomerName(): Promise<string> {
    return this.customerNameLocator.textContent() || '';
  }

  /**
   * Verify customer grade (Grade A, B, C, D)
   * @param expectedGrade e.g., "Grade A"
   */
  async verifyGrade(expectedGrade: string): Promise<boolean> {
    const gradeText = await this.customerGradeLocator.textContent();
    return gradeText?.includes(expectedGrade) || false;
  }

  /**
   * Get health score (0-100)
   * Used for customer analysis
   */
  async getHealthScore(): Promise<number> {
    const scoreText = await this.healthScoreLocator.textContent();
    const match = scoreText?.match(/\d+/);
    return match ? parseInt(match[0]) : 0;
  }

  /**
   * Get churn risk percentage (0-100)
   * Used for customer retention analysis
   */
  async getChurnRisk(): Promise<number> {
    const riskText = await this.churnRiskLocator.textContent();
    const match = riskText?.match(/\d+/);
    return match ? parseInt(match[0]) : 0;
  }

  /**
   * Get product price (formatted)
   * @param productId e.g., "LUB001"
   * @returns price as number, e.g., 132.00
   */
  productPriceLocator(productId: string): Locator {
    return this.page.locator(`[data-testid="product-${productId}-price"]`);
  }

  async getProductPrice(productId: string): Promise<number> {
    const priceText = await this.productPriceLocator(productId).textContent();
    const match = priceText?.match(/[\d.]+/);
    return match ? parseFloat(match[0]) : 0;
  }

  /**
   * Verify product pricing (including discounts)
   * @param productId e.g., "LUB001"
   * @param expectedPrice e.g., 132.00 (for Northstar Trading Grade A with 12% discount)
   */
  async verifyProductPrice(productId: string, expectedPrice: number): Promise<boolean> {
    const actualPrice = await this.getProductPrice(productId);
    // Allow 0.01 tolerance for floating point
    return Math.abs(actualPrice - expectedPrice) < 0.01;
  }

  /**
   * Get confidence level (0.0-1.0 from Consciousness Engine)
   * Used for SHM validation
   */
  async getConfidenceLevel(): Promise<number> {
    const confidenceAttr = await this.confidenceMeterLocator.getAttribute('data-value');
    return confidenceAttr ? parseFloat(confidenceAttr) : 0;
  }

  /**
   * Verify UX smoothness (FPS validation)
   * Requires PerformanceObserver data collection
   */
  async getAverageFPS(): Promise<number> {
    const fps = await this.page.evaluate(() => {
      return (window as any).__sonicTelemetry?.averageFPS || 60;
    });
    return fps;
  }

  /**
   * Verify CLS (Cumulative Layout Shift) < 0.1
   * Nielsen threshold for visual stability
   */
  async getCumulativeLayoutShift(): Promise<number> {
    const cls = await this.page.evaluate(() => {
      return (window as any).__sonicTelemetry?.cumulativeLayoutShift || 0;
    });
    return cls;
  }

  /**
   * Get sentiment from Consciousness Engine
   * e.g., "confident", "curious", "frustrated"
   */
  async getSentiment(): Promise<string> {
    const sentiment = await this.sentimentBadgeLocator.getAttribute('data-sentiment');
    return sentiment || '';
  }

  /**
   * Get flow state indicator
   * e.g., "in-flow", "distracted", "frustrated"
   */
  async getFlowState(): Promise<string> {
    const flowState = await this.flowIndicatorLocator.getAttribute('data-flow-state');
    return flowState || '';
  }

  /**
   * Search for product (e.g., "Marine Engine Lubricant")
   */
  async searchProduct(productName: string): Promise<void> {
    const searchInput = this.page.locator('[data-testid="product-search"]');
    await searchInput.fill(productName);
    await this.page.waitForLoadState('networkidle');
  }

  /**
   * Get order history count
   */
  async getOrderHistoryCount(): Promise<number> {
    const rows = await this.orderHistoryTable.locator('tbody tr').count();
    return rows;
  }

  /**
   * Scroll to ensure visibility (tests for layout stability)
   */
  async scrollToElement(testId: string): Promise<void> {
    const element = this.page.locator(`[data-testid="${testId}"]`);
    await element.scrollIntoViewIfNeeded();
  }

  /**
   * Verify page title (accessibility & SEO)
   */
  async verifyPageTitle(expectedTitle: string): Promise<boolean> {
    const title = await this.page.title();
    return title.includes(expectedTitle);
  }

  /**
   * Take screenshot for visual regression testing
   */
  async takeScreenshot(name: string): Promise<Buffer> {
    return await this.page.screenshot({ path: `test-results/screenshots/${name}.png` });
  }
}
