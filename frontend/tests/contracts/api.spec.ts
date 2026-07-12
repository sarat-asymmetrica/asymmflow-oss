import { expect, test } from '@playwright/test';
import { installMockWailsBridge } from '../e2e/helpers/mockWailsBridge';

const BASE_URL = process.env.VITE_DEV_SERVER_URL || 'http://127.0.0.1:5173';

async function callApp<T>(page: any, method: string, ...args: any[]): Promise<T> {
  return await page.evaluate(
    async ({ method, args }) => {
      const app = (window as any).go?.main?.App;
      if (!app || typeof app[method] !== 'function') {
        throw new Error(`Missing app binding: ${method}`);
      }
      return await app[method](...args);
    },
    { method, args },
  );
}

test.describe('Wails app binding contracts', () => {
  test.beforeEach(async ({ page }) => {
    await installMockWailsBridge(page);
    await page.goto(BASE_URL);
  });

  test('ValidateLicense returns the current session contract', async ({ page }) => {
    const result = await callApp<any>(page, 'ValidateLicense');

    expect(result.valid).toBe(true);
    expect(result.role).toBe('admin');
    expect(result.display_name).toBe('Developer');
    expect(result.permissions).toContain('*');
  });

  test('GetDashboardStats reflects mocked workflow counts', async ({ page }) => {
    const stats = await callApp<any>(page, 'GetDashboardStats');

    expect(stats.activity_year).toBe(2026);
    expect(stats.total_revenue).toBe(12000);
    expect(stats.active_orders).toBe(1);
    expect(stats.active_customers).toBe(3);
    expect(stats.active_rfqs).toBe(2);
  });

  test('customer profile binding returns full customer context', async ({ page }) => {
    const customer = await callApp<any>(page, 'GetCustomerFullProfile', 'cust-1');

    expect(customer.business_name).toBe('NPC');
    expect(customer.payment_grade).toBe('A');
    expect(customer.contacts.length).toBeGreaterThan(0);
    expect(customer.notes.length).toBeGreaterThan(0);
    expect(customer.recent_orders[0].order_number).toBe('ORD-2026-001');
  });

  test('CreateRFQ appends a new RFQ to the workflow state', async ({ page }) => {
    const created = await callApp<any>(page, 'CreateRFQ', 'NPC', 'Contract smoke RFQ', 9800, 'created from contracts');
    const rfqs = await callApp<any[]>(page, 'GetRFQs', 50, 0);

    expect(created.rfq_number).toBe('RFQ-2026-002');
    expect(rfqs[0].project_name).toBe('Contract smoke RFQ');
    expect(rfqs[0].value).toBe(9800);
  });

  test('SaveCostingAsOffer appends a quoted offer with a generated number', async ({ page }) => {
    const offer = await callApp<any>(page, 'SaveCostingAsOffer', {
      customerName: 'NPC',
      projectName: 'Contract smoke offer',
      grandTotal: 1595,
      items: [{ description: 'Pressure transmitter', quantity: 1, unit_price: 1450 }],
    });
    const offers = await callApp<any[]>(page, 'GetAllOffers');

    expect(offer.offer_number).toBe('PHO-2026-002');
    expect(offers[0].project_name).toBe('Contract smoke offer');
    expect(offers[0].stage).toBe('Quoted');
  });

  test('CreatePurchaseOrder appends a supplier PO contract', async ({ page }) => {
    const po = await callApp<any>(page, 'CreatePurchaseOrder', {
      supplier_id: 'sup-1',
      expected_delivery: '2026-04-14',
      line_items: [{ description: 'Temperature transmitter', quantity: 3, unit_price: 850 }],
    });
    const purchaseOrders = await callApp<any[]>(page, 'GetPurchaseOrders');

    expect(po.po_number).toMatch(/^PO-2026-\d{3}$/);
    expect(purchaseOrders[0].po_number).toBe(po.po_number);
    expect(purchaseOrders[0].supplier_name).toBe('Rhine Instruments');
    expect(purchaseOrders[0].expected_delivery).toBe('2026-04-14');
  });

  test('Butler persistent chat contract stores thread history', async ({ page }) => {
    const reply = await callApp<any>(page, 'ChatWithButlerPersistent', 'conv-contract-1', 'Summarize today');
    const conversations = await callApp<any[]>(page, 'ListConversations');
    const messages = await callApp<any[]>(page, 'GetConversationMessages', 'conv-contract-1');

    expect(reply.conversation_id).toBe('conv-contract-1');
    expect(conversations[0].id).toBe('conv-contract-1');
    expect(messages).toHaveLength(2);
    expect(messages[0].role).toBe('user');
    expect(messages[1].role).toBe('assistant');
  });
});
