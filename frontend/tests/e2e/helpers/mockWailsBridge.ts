import type { Page } from '@playwright/test';

type MockWailsBridgeOptions = {
  butlerResponse?: string;
  notificationReadDelayMs?: number;
  collaborativeDataReadyAfterRefreshCount?: number;
  taskDetailsReadyAfterRefreshCount?: number;
  projectContextDelayMs?: number;
  // Real i18n messages (en.json) so the startup initI18n() call resolves and the
  // shell renders localized labels instead of raw keys. Without this the app hits
  // its startup error boundary (InfraService.GetTranslations is undefined in a
  // bare browser). Callers that don't set it get an empty map → raw keys render.
  translations?: Record<string, string>;
};

export async function installMockWailsBridge(page: Page, options: MockWailsBridgeOptions = {}) {
  await page.addInitScript((opts: MockWailsBridgeOptions) => {
    const CONVERSATIONS_KEY = '__ph_e2e_butler_conversations__';
    const MESSAGES_KEY = '__ph_e2e_butler_messages__';
    const ACTIVE_CONVERSATION_KEY = 'butler_active_conversation_id';
    const WORKFLOW_STATE_KEY = '__ph_e2e_workflow_state__';
    (window as any).__wailsFileDropCallbacks = [];
    (window as any).__wailsOCRCalls = [];
    (window as any).__wailsDocumentSaveCalls = [];

    const now = () => new Date().toISOString();
    const safeJson = (value: string | null, fallback: any) => {
      if (!value) return fallback;
      try {
        return JSON.parse(value);
      } catch {
        return fallback;
      }
    };
    const readConversations = () => safeJson(window.localStorage.getItem(CONVERSATIONS_KEY), []);
    const writeConversations = (value: any[]) => {
      window.localStorage.setItem(CONVERSATIONS_KEY, JSON.stringify(value));
    };
    const readMessages = () => safeJson(window.localStorage.getItem(MESSAGES_KEY), {});
    const writeMessages = (value: Record<string, any[]>) => {
      window.localStorage.setItem(MESSAGES_KEY, JSON.stringify(value));
    };
    const defaultWorkflowState = () => ({
      customers: [
        {
          id: 'cust-1',
          business_name: 'National Petroleum Co.',
          customer_id: 'C01011',
          customer_type: 'Government',
          payment_grade: 'A',
          city: 'Awali',
          industry: 'Energy',
          trn: '990000000000000',
          relation_years: 15,
          active_invoices: 4,
          outstanding_bhd: 1200,
          total_orders_value: 45000,
          total_orders_count: 12,
          avg_payment_days: 21,
          contacts: [
            {
              contact_name: 'Pat Morgan',
              job_title: 'Procurement Manager',
              email: 'pat.morgan@nationalpetroleum.example',
              phone: '+973-1700-0000',
              is_primary_contact: true,
            },
          ],
          notes: [
            {
              note_type: 'general',
              content: 'Existing relationship note for regression coverage.',
              created_at: now(),
            },
          ],
          recent_orders: [
            { order_number: 'ORD-2026-001', status: 'In Progress', created_at: now(), grand_total_bhd: 45000 },
          ],
          recent_invoices: [
            { invoice_number: 'INV-2026-001', status: 'Sent', created_at: now(), grand_total_bhd: 12000 },
          ],
          recent_rfqs: [
            { project: 'Refinery Expansion Phase 4', status: 'Quoted', created_at: now(), value: 25000 },
          ],
          rfqs_floated: 8,
          rfqs_won: 5,
          win_rate: 62.5,
        },
        {
          id: 'cust-2',
          business_name: 'BlueWave Marine',
          customer_id: 'C01012',
          customer_type: 'Corporate',
          payment_grade: 'B',
          city: 'Hidd',
          industry: 'Marine',
          trn: '990000000000002',
          relation_years: 8,
          active_invoices: 2,
          outstanding_bhd: 800,
          total_orders_value: 18000,
          total_orders_count: 6,
          avg_payment_days: 34,
          contacts: [],
          notes: [],
          recent_orders: [],
          recent_invoices: [],
          recent_rfqs: [],
          rfqs_floated: 3,
          rfqs_won: 1,
          win_rate: 33.3,
        },
        {
          id: 'cust-3',
          business_name: 'Pioneer Process Group',
          customer_id: 'C01013',
          customer_type: 'Corporate',
          payment_grade: 'C',
          city: 'Manama',
          industry: 'Industrial',
          trn: '990000000000003',
          relation_years: 3,
          active_invoices: 1,
          outstanding_bhd: 0,
          total_orders_value: 3463,
          total_orders_count: 2,
          avg_payment_days: 47,
          contacts: [],
          notes: [],
          recent_orders: [],
          recent_invoices: [],
          recent_rfqs: [],
          rfqs_floated: 2,
          rfqs_won: 1,
          win_rate: 50,
        },
      ],
      suppliers: [
        { id: 'sup-1', supplier_name: 'Rhine Instruments', supplier_code: 'SUP001' },
        { id: 'sup-2', supplier_name: 'Oxan Analytics', supplier_code: 'SUP002' },
      ],
      rfqs: [
        {
          id: 1,
          client: 'National Petroleum Co.',
          customer: 'National Petroleum Co.',
          project: 'Refinery Expansion Phase 4',
          project_name: 'Refinery Expansion Phase 4',
          status: 'New',
          stage: 'New',
          value: 25000,
          created_at: now(),
          rfq_number: 'RFQ-2026-001',
        },
      ],
      rfqComments: {},
      opportunityComments: {},
      pipeline: [
        {
          id: 'pipe-1',
          customer_name: 'Pioneer Process Group',
          customer: 'Pioneer Process Group',
          client: 'Pioneer Process Group',
          title: 'Gas Analyzer Package',
          folder_number: '2026-311',
          revenue_bhd: 3463,
          stage: 'Qualified',
          year: 2026,
          comment: 'Initial technical scope captured from OCR.',
          owner_notes: 'Prioritize management review before pricing.',
          product_details: JSON.stringify([
            {
              description: 'Gas analyzer transmitter',
              quantity: 2,
              unit_price: 540.758,
              total_price: 1081.516,
              equipment: 'Analyzer transmitter',
              specification: '4-20mA with HART',
            },
            {
              description: 'Sample conditioning panel',
              quantity: 1,
              unit_price: 1299.968,
              total_price: 1299.968,
              equipment: 'Conditioning panel',
              specification: 'SS316 enclosure',
            },
          ]),
        },
      ],
      offers: [
        {
          id: 'offer-1',
          offer_number: 'PHO-2026-001',
          customer_name: 'National Petroleum Co.',
          project_name: 'Refinery Expansion Phase 4',
          folder_number: '43-26',
          payment_terms: 'Net 30',
          delivery_terms: 'DAP Bahrain',
          customer_reference: 'RFQ-2026-001',
          attention_person: 'Pat Morgan',
          attention_company: 'National Petroleum Co.',
          attention_phone: '+973-1700-0000',
          issued_by: 'Jamie Wong',
          stage: 'Quoted',
          quotation_date: now(),
          validity_date: now(),
          total_value_bhd: 25000,
          created_at: now(),
          items: [{ description: 'Pressure transmitter', quantity: 2, unit_price: 1250, total_price: 2500 }],
        },
      ],
      orders: [
        {
          id: 'order-1',
          order_number: 'ORD-2026-001',
          customer_name: 'National Petroleum Co.',
          status: 'In Progress',
          total_amount: 45000,
          date: now(),
        },
      ],
      supplierInvoices: [
        {
          id: 'supp-inv-1',
          supplier_id: 'sup-1',
          supplier_name: 'Rhine Instruments',
          invoice_number: '6017300440',
          invoice_date: '2026-01-24',
          due_date: '2026-02-23',
          currency: 'BHD',
          exchange_rate: 1,
          subtotal_foreign: 5788.085,
          subtotal_bhd: 5788.085,
          vat_foreign: 578.809,
          vat_bhd: 578.809,
          total_foreign: 6366.894,
          total_bhd: 6366.894,
          status: 'Approved',
          payment_status: 'Unpaid',
          payment_method: '',
          payment_ref: '',
          match_status: 'Pending',
        },
      ],
      supplierPayments: [],
      customerInvoices: [
        {
          id: 'inv-1',
          invoice_number: 'INV-2026-001',
          customer_name: 'National Petroleum Co.',
          grand_total_bhd: 500,
          status: 'Sent',
        },
      ],
      bankAccounts: [
        {
          id: 'bank-1',
          bank_name: 'Demo Bank A',
          account_name: 'Demo Bank A Operating',
          account_number: '00DEMO0000000001',
          currency: 'BHD',
          is_active: true,
        },
      ],
      bankStatements: [
        {
          id: 'stmt-1',
          bank_account_id: 'bank-1',
          statement_number: 'DEMO-A-2026-03',
          period_start: '2026-03-01',
          period_end: '2026-03-31',
          opening_balance: 1000,
          closing_balance: 1500,
          total_debits: 6366.894,
          total_credits: 0,
          status: 'Imported',
        },
      ],
      bankStatementLines: [
        {
          id: 'line-1',
          bank_statement_id: 'stmt-1',
          line_number: 1,
          transaction_date: '2026-03-20',
          description: 'Supplier payment - Rhine Instruments',
          reference: 'TXN-1001',
          debit: 6366.894,
          credit: 0,
          balance: -5366.894,
          transaction_type: 'Supplier Payment',
          is_matched: false,
          match_type: 'Unmatched',
          match_confidence: 0,
          extracted_customer: '',
          extracted_invoices: '',
        },
      ],
      employeeProfiles: [
        {
          id: 'emp-admin',
          employee_code: 'EMP-001',
          full_name: 'Developer Admin',
          department: 'Management',
          job_title: 'Administrator',
          is_active: true,
        },
        {
          id: 'emp-jamie',
          employee_code: 'EMP-002',
          full_name: 'Jamie Wong',
          department: 'Operations',
          job_title: 'Project Coordinator',
          is_active: true,
        },
      ],
      collaborativeProjects: [
        {
          id: 'proj-1',
          name: 'Butler Rollout',
          project_type: 'internal',
          description: 'Regression project used for E2E task routing coverage.',
          status: 'active',
        },
      ],
      collaborativeProjectMembers: [
        {
          id: 'project-member-1',
          project_id: 'proj-1',
          employee_id: 'emp-jamie',
          employee_name: 'Jamie Wong',
          project_name: 'Butler Rollout',
          role: 'Member',
          allocation_percent: 100,
          is_active: true,
          joined_at: now(),
        },
      ],
      collaborativeTasks: [
        {
          id: 'task-1',
          title: 'Review salary expense forecast',
          description: 'Confirm this month\'s HR and payroll forecast.',
          task_type: 'internal',
          status: 'open',
          priority: 'high',
          project_id: 'proj-1',
          creator_employee_id: 'emp-admin',
          assignee_employee_id: 'emp-jamie',
          creator_name: 'Developer Admin',
          assignee_name: 'Jamie Wong',
          due_date: '2026-04-20T09:00:00.000Z',
        },
      ],
      collaborativeTaskComments: [
        {
          id: 'task-comment-1',
          task_id: 'task-1',
          employee_id: 'emp-admin',
          employee_name: 'Developer Admin',
          body: 'Please sanity-check the payroll assumptions before sign-off.',
          created_at: now(),
        },
      ],
      collaborativeTaskActivity: [
        {
          id: 'task-activity-1',
          task_id: 'task-1',
          employee_id: 'emp-admin',
          employee_name: 'Developer Admin',
          activity_type: 'created',
          detail: 'Task created for regression coverage.',
          metadata_json: '{}',
          created_at: now(),
        },
      ],
      notificationFeed: [
        {
          id: 'notif-task-1',
          notification_type: 'task',
          title: 'Task assigned',
          message: 'Review salary expense forecast has been assigned to you.',
          status: 'unread',
          source_type: 'task',
          source_id: 'task-1',
          action_route: 'work',
          action_payload: JSON.stringify({
            action: 'open_task',
            task_id: 'task-1',
            task_title: 'Review salary expense forecast',
            project_name: 'Butler Rollout',
            actor_name: 'Developer Admin',
          }),
          created_at: now(),
          read_at: null,
          employee_id: 'emp-jamie',
        },
      ],
      collaborativeWorkspaceRefreshCount: 0,
      purchaseOrders: [
        {
          id: 'po-1',
          po_number: 'PO-2026-007',
          order_id: 'order-1',
          supplier_id: 'sup-1',
          supplier_name: 'Rhine Instruments',
          po_date: '2026-03-28',
          expected_delivery: '2026-05-09',
          currency: 'EUR',
          exchange_rate: 0.41,
          subtotal_foreign: 6200,
          subtotal_bhd: 2542,
          vat_amount: 620,
          total_foreign: 6820,
          total_bhd: 2796.2,
          payment_terms: 'Net 30',
          status: 'Draft',
          items: [
            {
              id: 'po-item-1',
              purchase_order_id: 'po-1',
              description: 'Temperature transmitter',
              quantity: 4,
              unit_price_foreign: 850,
              total_foreign: 3400,
              total_bhd: 1394,
            },
            {
              id: 'po-item-2',
              purchase_order_id: 'po-1',
              description: 'Signal isolator',
              quantity: 4,
              unit_price_foreign: 700,
              total_foreign: 2800,
              total_bhd: 1148,
            },
          ],
        },
      ],
      costingSheets: [],
      nextRFQId: 2,
      nextOfferId: 2,
      nextPOId: 8,
    });
    const readWorkflowState = () => safeJson(window.localStorage.getItem(WORKFLOW_STATE_KEY), defaultWorkflowState());
    const writeWorkflowState = (value: any) => {
      window.localStorage.setItem(WORKFLOW_STATE_KEY, JSON.stringify(value));
    };
    const collaborativeDataReady = (state: any) =>
      Number(state.collaborativeWorkspaceRefreshCount || 0) >= Number(opts.collaborativeDataReadyAfterRefreshCount ?? 2);
    const taskDetailsReady = (state: any) =>
      Number(state.collaborativeWorkspaceRefreshCount || 0) >= Number(opts.taskDetailsReadyAfterRefreshCount ?? 3);
    const buildCustomerDashboard = (state: any) => {
      const customers = state.customers || [];
      const topCustomers = [...customers]
        .sort((a: any, b: any) => Number(b.total_orders_value || 0) - Number(a.total_orders_value || 0))
        .slice(0, 10)
        .map((customer: any) => ({
          id: customer.id,
          business_name: customer.business_name,
          total_revenue: Number(customer.total_orders_value || 0),
        }));

      const totalRevenue = customers.reduce((sum: number, customer: any) => sum + Number(customer.total_orders_value || 0), 0);
      const totalOutstanding = customers.reduce((sum: number, customer: any) => sum + Number(customer.outstanding_bhd || 0), 0);
      const overdueAmount = customers.reduce((sum: number, customer: any) => sum + Number(customer.overdue_bhd || 0), 0);
      const activeCustomers = customers.length;
      const top3Revenue = topCustomers.slice(0, 3).reduce((sum: number, customer: any) => sum + Number(customer.total_revenue || 0), 0);
      const top5Revenue = topCustomers.slice(0, 5).reduce((sum: number, customer: any) => sum + Number(customer.total_revenue || 0), 0);
      const top10Revenue = topCustomers.slice(0, 10).reduce((sum: number, customer: any) => sum + Number(customer.total_revenue || 0), 0);
      const countGrade = (grade: string) => customers.filter((customer: any) => customer.payment_grade === grade);
      const revenueFor = (grade: string) =>
        countGrade(grade).reduce((sum: number, customer: any) => sum + Number(customer.total_orders_value || 0), 0);

      return {
        total_customers: activeCustomers,
        active_customers: activeCustomers,
        total_revenue: totalRevenue,
        revenue_yoy: 12.4,
        total_outstanding: totalOutstanding,
        overdue_amount: overdueAmount,
        overdue_pct: totalOutstanding > 0 ? (overdueAmount / totalOutstanding) * 100 : 0,
        top_customers: topCustomers,
        top3_revenue_pct: totalRevenue > 0 ? (top3Revenue / totalRevenue) * 100 : 0,
        top5_revenue_pct: totalRevenue > 0 ? (top5Revenue / totalRevenue) * 100 : 0,
        top10_revenue_pct: totalRevenue > 0 ? (top10Revenue / totalRevenue) * 100 : 0,
        grade_a_count: countGrade('A').length,
        grade_b_count: countGrade('B').length,
        grade_c_count: countGrade('C').length,
        grade_d_count: countGrade('D').length,
        grade_a_revenue: revenueFor('A'),
        grade_b_revenue: revenueFor('B'),
        grade_c_revenue: revenueFor('C'),
        grade_d_revenue: revenueFor('D'),
        customers,
      };
    };
    const findCustomer = (state: any, customerId: string) =>
      state.customers.find((item: any) => String(item.id) === String(customerId) || String(item.customer_id) === String(customerId));
    const findPipelineOpportunity = (state: any, opportunityId: string) =>
      (state.pipeline || []).find((item: any) => String(item.id) === String(opportunityId));
    const findSupplierInvoice = (state: any, invoiceId: string) =>
      (state.supplierInvoices || []).find((item: any) => String(item.id) === String(invoiceId));
    const deriveSupplierPayments = (state: any) => {
      const explicitPayments = [...(state.supplierPayments || [])];
      const keyed = new Set(explicitPayments.map((item: any) => String(item.supplier_invoice_id || item.invoice_number || item.id)));

      (state.supplierInvoices || []).forEach((invoice: any) => {
        if (invoice.payment_status !== 'Paid') return;
        const key = String(invoice.id || invoice.invoice_number || '');
        if (keyed.has(key)) return;
        explicitPayments.push({
          id: `derived-${invoice.id}`,
          supplier_invoice_id: invoice.id,
          supplier_id: invoice.supplier_id,
          supplier_name: invoice.supplier_name,
          invoice_number: invoice.invoice_number,
          amount_foreign: Number(invoice.total_foreign || 0),
          amount_bhd: Number(invoice.total_bhd || 0),
          currency: invoice.currency || 'BHD',
          exchange_rate: Number(invoice.exchange_rate || 1),
          payment_date: invoice.payment_date || now(),
          payment_method: invoice.payment_method || 'Bank Transfer',
          reference: invoice.payment_ref || invoice.invoice_number || 'PAYMENT',
        });
      });

      return explicitPayments;
    };
    const buildCustomerProfile = (customer: any) => {
      if (!customer) return null;
      return {
        ...customer,
        address_line1: customer.address_line1 || 'Building 12, Road 45',
        country: customer.country || 'Bahrain',
        contacts: customer.contacts || [],
        notes: customer.notes || [],
        recent_orders: customer.recent_orders || [],
        recent_invoices: customer.recent_invoices || [],
        recent_rfqs: customer.recent_rfqs || [],
        rfqs_floated: customer.rfqs_floated || 0,
        rfqs_won: customer.rfqs_won || 0,
        win_rate: customer.win_rate || 0,
      };
    };

    const mockButlerResponse = opts.butlerResponse || 'Mock Butler response persisted.';
    const mockOCRResult = {
      documentType: 'BankStatement',
      document_type: 'BankStatement',
      confidence: 0.96,
      engine: 'mock-ocr',
      processingTimeMS: 41,
      processing_time_ms: 41,
      text: [
        'BANK STATEMENT',
        'Demo Bank B',
        'Opening Balance 1,000.000',
        '02/03/2026 Deposit 500.000 1,500.000',
        'Closing Balance 1,500.000',
      ].join('\n'),
      extractedData: {
        bank_name: 'Demo Bank B',
        account_number: '00DEMO0000000002',
        opening_balance: '1,000.000',
        closing_balance: '1,500.000',
        period_start: '2026-03-01',
        period_end: '2026-03-31',
        line_items: [],
      },
      extracted_data: {
        bank_name: 'Demo Bank B',
        account_number: '00DEMO0000000002',
        opening_balance: '1,000.000',
        closing_balance: '1,500.000',
        period_start: '2026-03-01',
        period_end: '2026-03-31',
        line_items: [],
      },
      line_items: [],
    };

    const mockButlerInsight = {
      summary: 'Mock Butler analysis: one credit transaction and a clean balance trail.',
      confidence: 0.97,
      detected_customer: 'Acme Instrumentation',
      detected_project: 'Bank reconciliation smoke test',
      extracted_items: [
        {
          date: '2026-03-02',
          description: 'Deposit',
          reference: 'DEP-001',
          debit: 0,
          credit: 500,
          balance: 1500,
        },
      ],
      metadata: {
        bank_name: 'Demo Bank B',
        account_number: '00DEMO0000000002',
        opening_balance: '1,000.000',
        closing_balance: '1,500.000',
        period_start: '2026-03-01',
        period_end: '2026-03-31',
      },
      document_type: 'BankStatement',
    };

    const createConversation = (conversationId: string, message: string) => {
      const conversations = readConversations();
      const conversation = conversations.find((item: any) => String(item.id) === String(conversationId));
      if (!conversation) {
        conversations.unshift({
          id: conversationId,
          title: message.slice(0, 42) || 'Butler Chat',
          updated_at: now(),
        });
      } else {
        conversation.title = conversation.title || message.slice(0, 42) || 'Butler Chat';
        conversation.updated_at = now();
      }
      writeConversations(conversations);

      const messages = readMessages();
      const thread = Array.isArray(messages[conversationId]) ? messages[conversationId] : [];
      thread.push({
        id: `msg-user-${Date.now()}`,
        role: 'user',
        content: message,
      });
      thread.push({
        id: `msg-bot-${Date.now()}`,
        role: 'assistant',
        content: mockButlerResponse,
      });
      messages[conversationId] = thread;
      writeMessages(messages);
      window.localStorage.setItem(ACTIVE_CONVERSATION_KEY, conversationId);
    };

    if (!window.localStorage.getItem(WORKFLOW_STATE_KEY)) {
      writeWorkflowState(defaultWorkflowState());
    }

    const defaultVoid = () => Promise.resolve();
    const defaultValue = (value: any) => Promise.resolve(value);
    const delayedValue = (value: any, delayMs = 0) => new Promise((resolve) => {
      if (delayMs <= 0) {
        resolve(value);
        return;
      }
      window.setTimeout(() => resolve(value), delayMs);
    });

    const app = new Proxy({}, {
      get(_, prop) {
        return (...args: any[]) => {
          switch (String(prop)) {
            case 'ValidateLicense':
              return defaultValue({
                valid: true,
                role: 'admin',
                display_name: 'Developer',
                permissions: ['*'],
              });
            case 'NeedsSetup':
            case 'NeedsLicenseActivation':
              return defaultValue(false);
            case 'CheckDeviceStatus':
              return defaultValue({
                status: 'approved',
                user_id: 'e2e-user',
                user_name: 'Developer',
                role_name: 'admin',
                permissions: ['*'],
              });
            case 'GetDBSyncStatus':
              return defaultValue({
                configured: false,
                online: false,
                sync_enabled: false,
                last_sync: null,
              });
            // People hub (Wave 11 A3): one synthetic employee so the directory
            // populates and the Employee Detail pane (Profile/Work/Access/
            // Compliance sub-tabs) is reachable in the sweep. Empty arrays for
            // the sibling lists keep the Promise.all load path from throwing.
            case 'ListEmployeeProfiles':
              return defaultValue([
                {
                  id: 'emp-1',
                  employee_code: 'EMP-001',
                  full_name: 'Jordan Avery',
                  preferred_name: 'Jordan',
                  email: 'jordan.avery@asymmflow.example',
                  phone: '+973-1700-0100',
                  department: 'Operations',
                  job_title: 'Operations Lead',
                  employment_status: 'Full-time',
                  manager_employee_id: '',
                  manager_name: '',
                  start_date: '2024-01-15T00:00:00Z',
                  end_date: null,
                  emergency_contact: 'Sam Avery +973-1700-0101',
                  notes: 'Synthetic profile for QA sweep.',
                  is_active: true,
                  archived_at: null,
                  archived_by: '',
                  archive_reason: '',
                  archive_request_id: '',
                },
              ]);
            case 'ListEmployeeAccessLinks':
            case 'ListLicenseKeys':
            case 'ListEmployeeContributionSummaries':
            case 'ListLoginUsers':
            case 'ListLoginRoles':
            case 'ListEmployeeComplianceDocuments':
              return defaultValue([]);
            case 'GetDashboardStats':
              return defaultValue({
                total_revenue: 12000,
                revenue_meta: 'Smoke test FY2026',
                month_growth: 1.8,
                activity_year: 2026,
                active_rfqs: readWorkflowState().rfqs.length + readWorkflowState().pipeline.length,
                active_orders: readWorkflowState().orders.length,
                outstanding_ar: 200,
                ar_days_overdue: 0,
                pending_invoices: 1,
                active_customers: readWorkflowState().customers.length,
                win_rate: 64.2,
              });
            case 'ListFollowUps':
              return defaultValue([]);
            case 'GetCRMCustomerDashboard':
            case 'GetCRMCustomerDashboardByYear':
              return defaultValue(buildCustomerDashboard(readWorkflowState()));
            case 'ListCustomers':
              return defaultValue(readWorkflowState().customers);
            case 'GetCustomerFullProfile': {
              const state = readWorkflowState();
              const customer = findCustomer(state, String(args[0] || ''));
              if (!customer) {
                return Promise.reject(new Error('Customer not found'));
              }
              return defaultValue(buildCustomerProfile(customer));
            }
            case 'AddCustomerNote': {
              const [customerId, noteType, content] = args;
              const state = readWorkflowState();
              const customer = findCustomer(state, String(customerId || ''));
              if (!customer) {
                return Promise.reject(new Error('Customer not found'));
              }
              customer.notes = customer.notes || [];
              customer.notes.unshift({
                note_type: String(noteType || 'general'),
                content: String(content || ''),
                created_at: now(),
              });
              writeWorkflowState(state);
              return defaultVoid();
            }
            case 'AddCustomerContact': {
              const [payload] = args;
              const state = readWorkflowState();
              const customer = findCustomer(state, String(payload?.customer_id || ''));
              if (!customer) {
                return Promise.reject(new Error('Customer not found'));
              }
              customer.contacts = customer.contacts || [];
              customer.contacts.unshift({
                contact_name: payload?.contact_name || 'Unnamed Contact',
                job_title: payload?.job_title || '',
                email: payload?.email || '',
                phone: payload?.phone || '',
                address: payload?.address || '',
                is_primary_contact: Boolean(payload?.is_primary_contact),
              });
              writeWorkflowState(state);
              return defaultVoid();
            }
            case 'UpdateCustomer': {
              const [payload] = args;
              const state = readWorkflowState();
              const customer = findCustomer(state, String(payload?.id || payload?.customer_id || ''));
              if (!customer) {
                return Promise.reject(new Error('Customer not found'));
              }
              Object.assign(customer, payload || {});
              writeWorkflowState(state);
              return defaultValue(customer);
            }
            case 'DeleteCustomer': {
              const state = readWorkflowState();
              state.customers = state.customers.filter((item: any) => String(item.id) !== String(args[0]));
              writeWorkflowState(state);
              return defaultVoid();
            }
            case 'ListSuppliers':
              return defaultValue(readWorkflowState().suppliers);
            case 'GetRFQs':
              return defaultValue(readWorkflowState().rfqs);
            case 'GetRFQ': {
              const state = readWorkflowState();
              return defaultValue(state.rfqs.find((item: any) => String(item.id) === String(args[0])) || null);
            }
            case 'CreateRFQ': {
              const [customer, project, value, notes] = args;
              const state = readWorkflowState();
              const rfq = {
                id: state.nextRFQId,
                client: String(customer),
                customer: String(customer),
                project: String(project),
                project_name: String(project),
                status: 'New',
                stage: 'New',
                value: Number(value) || 0,
                notes: String(notes || ''),
                created_at: now(),
                rfq_number: `RFQ-2026-${String(state.nextRFQId).padStart(3, '0')}`,
              };
              state.nextRFQId += 1;
              state.rfqs.unshift(rfq);
              writeWorkflowState(state);
              return defaultValue(rfq);
            }
            case 'GetPipelineOpportunities':
              return defaultValue(readWorkflowState().pipeline);
            case 'GetOpportunityLineItems': {
              const state = readWorkflowState();
              const opportunity = findPipelineOpportunity(state, String(args[0]));
              if (!opportunity?.product_details) {
                return defaultValue([]);
              }
              try {
                return defaultValue(JSON.parse(opportunity.product_details));
              } catch {
                return defaultValue([]);
              }
            }
            case 'CanResolveOpportunityConflicts':
              return defaultValue(true);
            case 'ListOpportunityEditConflicts':
              return defaultValue(readWorkflowState().opportunityConflicts || []);
            case 'ResolveOpportunityEditConflict': {
              const [conflictId, action] = args;
              const state = readWorkflowState();
              state.opportunityConflicts = (state.opportunityConflicts || []).map((conflict: any) => (
                String(conflict.id) === String(conflictId)
                  ? { ...conflict, status: action === 'apply' ? 'applied' : 'rejected', resolution_action: action }
                  : conflict
              ));
              writeWorkflowState(state);
              return defaultValue({ conflict: state.opportunityConflicts.find((item: any) => String(item.id) === String(conflictId)) });
            }
            case 'UpdateOpportunityStageWithVersion': {
              const [opportunityId, stage] = args;
              const state = readWorkflowState();
              const opportunity = findPipelineOpportunity(state, String(opportunityId));
              if (!opportunity) {
                return Promise.reject(new Error('Opportunity not found'));
              }
              opportunity.stage = String(stage || '').trim();
              opportunity.version = Number(opportunity.version || 1) + 1;
              writeWorkflowState(state);
              return defaultValue(opportunity);
            }
            case 'UpdateOpportunityDetails': {
              const [opportunityId, comment, ownerNotes] = args;
              const state = readWorkflowState();
              const opportunity = findPipelineOpportunity(state, String(opportunityId));
              if (!opportunity) {
                return Promise.reject(new Error('Opportunity not found'));
              }
              opportunity.comment = String(comment || '').trim();
              opportunity.owner_notes = String(ownerNotes || '').trim();
              writeWorkflowState(state);
              return defaultValue(opportunity);
            }
            case 'UpdateOpportunityDetailsWithVersion': {
              const [opportunityId, , comment, ownerNotes] = args;
              const state = readWorkflowState();
              const opportunity = findPipelineOpportunity(state, String(opportunityId));
              if (!opportunity) {
                return Promise.reject(new Error('Opportunity not found'));
              }
              opportunity.comment = String(comment || '').trim();
              opportunity.owner_notes = String(ownerNotes || '').trim();
              opportunity.version = Number(opportunity.version || 1) + 1;
              writeWorkflowState(state);
              return defaultValue(opportunity);
            }
            case 'UpdateRFQNotes': {
              const [rfqId, notes] = args;
              const state = readWorkflowState();
              const rfq = state.rfqs.find((item: any) => Number(item.id) === Number(rfqId));
              if (!rfq) {
                return Promise.reject(new Error('RFQ not found'));
              }
              rfq.notes = String(notes || '').trim();
              writeWorkflowState(state);
              return defaultValue(rfq);
            }
            case 'GetCurrentUserRole':
              return defaultValue('admin');
            case 'GetCurrentEmployeeContext':
              return defaultValue({
                employee_id: 'emp-jamie',
                employee_name: 'Jamie Wong',
                license_key: 'PH-MGR-JWONG1',
                license_role: 'manager',
                device_id: 'device-e2e',
                user_id: 'e2e-user',
                resolved_by: 'user',
                permissions: ['tasks:view', 'notifications:view', 'notifications:update'],
              });
            case 'ListEmployeeProfiles': {
              const state = readWorkflowState();
              return defaultValue(state.employeeProfiles || []);
            }
            case 'ListCollaborativeProjects': {
              const state = readWorkflowState();
              return defaultValue(collaborativeDataReady(state) ? (state.collaborativeProjects || []) : []);
            }
            case 'CreateCollaborativeProject': {
              const state = readWorkflowState();
              const payload = args[0] || {};
              const project = {
                id: `proj-created-${Date.now()}`,
                name: String(payload.name || '').trim(),
                project_type: String(payload.project_type || 'internal'),
                description: String(payload.description || ''),
                status: 'active',
                owner_employee_id: 'emp-jamie',
                owner_name: 'Jamie Wong',
                customer_id: payload.customer_id || undefined,
                opportunity_id: payload.opportunity_id || undefined,
                order_id: payload.order_id || undefined,
                customer_name: payload.customer_name || '',
                customer_poc_name: payload.customer_poc_name || '',
                customer_poc_email: payload.customer_poc_email || '',
                customer_poc_phone: payload.customer_poc_phone || '',
                created_at: now(),
                updated_at: now(),
              };
              state.collaborativeProjects = [project, ...(state.collaborativeProjects || [])];
              writeWorkflowState(state);
              return defaultValue(project);
            }
            case 'ListMyCollaborativeTasks': {
              const state = readWorkflowState();
              return defaultValue(collaborativeDataReady(state) ? (state.collaborativeTasks || []) : []);
            }
            case 'ListCollaborativeTeamTasks': {
              const state = readWorkflowState();
              return defaultValue(collaborativeDataReady(state) ? (state.collaborativeTasks || []) : []);
            }
            case 'ListCollaborativeProjectTasks': {
              const state = readWorkflowState();
              const projectID = String(args[0] || '');
              const tasks = collaborativeDataReady(state)
                ? (state.collaborativeTasks || []).filter((task: any) => String(task.project_id) === projectID)
                : [];
              return delayedValue(tasks, Number(opts.projectContextDelayMs || 0));
            }
            case 'ListCollaborativeProjectMembers': {
              const state = readWorkflowState();
              const projectID = String(args[0] || '');
              if (!collaborativeDataReady(state)) {
                return delayedValue([], Number(opts.projectContextDelayMs || 0));
              }
              const members = (state.collaborativeProjectMembers || []).filter((member: any) => String(member.project_id) === projectID);
              return delayedValue(members, Number(opts.projectContextDelayMs || 0));
            }
            case 'AddCollaborativeProjectMember': {
              const state = readWorkflowState();
              const [projectID, employeeID, role, allocationPercent] = args;
              const project = (state.collaborativeProjects || []).find((item: any) => String(item.id) === String(projectID));
              const employee = (state.employeeProfiles || []).find((item: any) => String(item.id) === String(employeeID));
              if (!project || !employee) {
                return Promise.reject(new Error('Project or employee not found'));
              }
              const allocation = Number(allocationPercent);
              const member = {
                id: `project-member-${Date.now()}`,
                project_id: project.id,
                employee_id: employee.id,
                employee_name: employee.full_name,
                project_name: project.name,
                role: String(role || 'Member'),
                allocation_percent: Number.isFinite(allocation) && allocation > 0 ? Math.min(allocation, 100) : 100,
                is_active: true,
                joined_at: now(),
              };
              state.collaborativeProjectMembers = [member, ...(state.collaborativeProjectMembers || [])];
              writeWorkflowState(state);
              return defaultValue(member);
            }
            case 'GetProjectTaskCounts': {
              const state = readWorkflowState();
              const counts: Record<string, number> = {};
              for (const task of state.collaborativeTasks || []) {
                const projectID = (task as any).project_id;
                if (!projectID) continue;
                counts[String(projectID)] = (counts[String(projectID)] || 0) + 1;
              }
              return defaultValue(counts);
            }
            case 'ListCollaborativeProjectActivity': {
              const state = readWorkflowState();
              if (!collaborativeDataReady(state)) {
                return delayedValue([], Number(opts.projectContextDelayMs || 0));
              }
              const activity = (state.collaborativeTaskActivity || []).filter((item: any) => {
                const task = (state.collaborativeTasks || []).find((candidate: any) => String(candidate.id) === String(item.task_id));
                return String(task?.project_id || '') === String(args[0] || '');
              });
              return delayedValue(activity, Number(opts.projectContextDelayMs || 0));
            }
            case 'GetCollaborativeTask': {
              const state = readWorkflowState();
              const taskID = String(args[0] || '');
              const task = (state.collaborativeTasks || []).find((item: any) => String(item.id) === taskID);
              if (!task || !taskDetailsReady(state)) {
                return Promise.reject(new Error('task not found'));
              }
              return defaultValue(task);
            }
            case 'ListCollaborativeTaskComments': {
              const state = readWorkflowState();
              if (!taskDetailsReady(state)) {
                return Promise.reject(new Error('task not found'));
              }
              return defaultValue((state.collaborativeTaskComments || []).filter((item: any) => String(item.task_id) === String(args[0] || '')));
            }
            case 'ListCollaborativeTaskActivity': {
              const state = readWorkflowState();
              if (!taskDetailsReady(state)) {
                return Promise.reject(new Error('task not found'));
              }
              return defaultValue((state.collaborativeTaskActivity || []).filter((item: any) => String(item.task_id) === String(args[0] || '')));
            }
            case 'ListNotificationFeed': {
              const state = readWorkflowState();
              const limit = Number(args[0] || 50);
              const unreadOnly = Boolean(args[1]);
              const feed = (state.notificationFeed || []).filter((item: any) => !unreadOnly || item.status !== 'read');
              return defaultValue(feed.slice(0, limit));
            }
            case 'GetUnreadNotificationsCount': {
              const state = readWorkflowState();
              return defaultValue((state.notificationFeed || []).filter((item: any) => item.status !== 'read').length);
            }
            case 'MarkNotificationAsRead': {
              const state = readWorkflowState();
              const notification = (state.notificationFeed || []).find((item: any) => String(item.id) === String(args[0] || ''));
              return new Promise((resolve) => {
                window.setTimeout(() => {
                  if (notification) {
                    notification.status = 'read';
                    notification.read_at = now();
                    writeWorkflowState(state);
                  }
                  resolve(undefined);
                }, Number(opts.notificationReadDelayMs || 0));
              });
            }
            case 'RefreshCollaborativeWorkspace': {
              const state = readWorkflowState();
              state.collaborativeWorkspaceRefreshCount = Number(state.collaborativeWorkspaceRefreshCount || 0) + 1;
              writeWorkflowState(state);
              return defaultVoid();
            }
            case 'GetSettings':
              return defaultValue({ defaultVatRate: 10, defaultMargin: 20 });
            case 'GetCostingSheets':
              return defaultValue(readWorkflowState().costingSheets);
            case 'ListConversations':
              return defaultValue(readConversations());
            case 'GetConversationMessages':
              return defaultValue(readMessages()[String(args[0])] || []);
            case 'ChatWithButlerPersistent': {
              const [conversationId, message] = args;
              const id = String(conversationId || window.localStorage.getItem(ACTIVE_CONVERSATION_KEY) || `conv-${Date.now()}`);
              createConversation(id, String(message || ''));
              return defaultValue({
                conversation_id: id,
                response: mockButlerResponse,
                confidence: 0.99,
                actions: [],
              });
            }
            case 'ChatWithButler':
              return defaultValue({
                message: mockButlerResponse,
                confidence: 0.99,
                actions: [],
              });
            case 'DeleteConversation':
              return defaultValue(undefined);
            case 'PurgeAllConversations':
              writeConversations([]);
              writeMessages({});
              window.localStorage.removeItem(ACTIVE_CONVERSATION_KEY);
              return defaultValue(undefined);
            case 'PickFile':
              return defaultValue('/tmp/ph-holdings-smoke/bank-statement.pdf');
            case 'ProcessDocumentWithOCR':
              (window as any).__wailsOCRCalls.push({ filePath: args[0], docType: args[1] });
              return defaultValue(mockOCRResult);
            case 'AnalyzeDocumentWithButler':
              return defaultValue(mockButlerInsight);
            case 'SaveDocumentToEntity':
              (window as any).__wailsDocumentSaveCalls.push({
                fileName: args[0],
                filePath: args[1],
                documentType: args[2],
                extractedText: args[3],
                confidence: args[4],
                processingTimeMs: args[5],
                engine: args[6],
                extractedDataJSON: args[7],
              });
              return defaultValue({ routed: true, entity_id: 'ocr-doc-1' });
            case 'SaveOCRDocument':
            case 'UpdateRFQStage':
            case 'GetPurchaseOrdersByOrder':
            case 'GetRFQTraceability':
              return defaultVoid();
            case 'GetRFQComments':
              return defaultValue(readWorkflowState().rfqComments?.[String(args[0])] || []);
            case 'GetOpportunityComments':
              return defaultValue(readWorkflowState().opportunityComments?.[String(args[0])] || []);
            case 'AddRFQComment': {
              const [rfqId, comment] = args;
              const state = readWorkflowState();
              const key = String(rfqId);
              state.rfqComments = state.rfqComments || {};
              state.rfqComments[key] = [
                ...(state.rfqComments[key] || []),
                { id: `rfq-comment-${Date.now()}`, comment, created_by: 'Developer', created_at: now() },
              ];
              writeWorkflowState(state);
              return defaultVoid();
            }
            case 'AddOpportunityComment': {
              const [opportunityId, comment] = args;
              const state = readWorkflowState();
              const key = String(opportunityId);
              state.opportunityComments = state.opportunityComments || {};
              state.opportunityComments[key] = [
                ...(state.opportunityComments[key] || []),
                { id: `opportunity-comment-${Date.now()}`, comment, created_by: 'Developer', created_at: now() },
              ];
              writeWorkflowState(state);
              return defaultVoid();
            }
            case 'CreateCostingSheet':
            case 'UpdateCostingSheet':
              return defaultVoid();
            case 'SaveCostingAsOffer': {
              const state = readWorkflowState();
              const payload = args[0] || {};
              const offer = {
                id: `offer-${state.nextOfferId}`,
                offer_number: `PHO-2026-${String(state.nextOfferId).padStart(3, '0')}`,
                customer_name: payload.customerName || payload.customer_name || 'National Petroleum Co.',
                project_name: payload.projectName || payload.project_name || 'Generated Offer',
                stage: 'Quoted',
                quotation_date: now(),
                validity_date: now(),
                total_value_bhd: Number(payload.grandTotal || payload.total_value_bhd || 0),
                created_at: now(),
                items: payload.items || payload.line_items || [],
              };
              state.nextOfferId += 1;
              state.offers.unshift(offer);
              writeWorkflowState(state);
              return defaultValue(offer);
            }
            case 'UpdateOfferFull': {
              const [offerId, payload] = args;
              const state = readWorkflowState();
              const offer = (state.offers || []).find((item: any) => String(item.id) === String(offerId));
              if (!offer) {
                return Promise.reject(new Error('Offer not found'));
              }
              const { items, ...headerPayload } = payload || {};
              Object.assign(offer, headerPayload);
              if (Array.isArray(items) && items.length > 0) {
                offer.items = items;
                offer.total_value_bhd = items.reduce((sum: number, item: any) => (
                  sum + (Number(item.total_price) || ((Number(item.quantity) || 0) * (Number(item.unit_price) || 0)))
                ), 0);
              }
              writeWorkflowState(state);
              return defaultValue(offer);
            }
            case 'GetAllOffers':
              return defaultValue(readWorkflowState().offers);
            case 'GetPendingFollowUps':
            case 'GetOffersWithNoItems':
              return defaultValue([]);
            case 'ListOrders':
              return defaultValue(readWorkflowState().orders);
            case 'GetSupplierInvoices':
              return defaultValue(readWorkflowState().supplierInvoices);
            case 'GetSupplierInvoiceByID': {
              const state = readWorkflowState();
              return defaultValue(findSupplierInvoice(state, String(args[0])) || null);
            }
            case 'GetSupplierPaymentsByInvoice': {
              const state = readWorkflowState();
              return defaultValue(deriveSupplierPayments(state).filter((item: any) => String(item.supplier_invoice_id) === String(args[0])));
            }
            case 'GetUnpaidSupplierInvoices': {
              const state = readWorkflowState();
              return defaultValue((state.supplierInvoices || []).filter((item: any) => item.payment_status !== 'Paid'));
            }
            case 'GetOverdueSupplierInvoices': {
              const state = readWorkflowState();
              return defaultValue((state.supplierInvoices || []).filter((item: any) => item.payment_status !== 'Paid'));
            }
            case 'GetAllSupplierPayments':
              return defaultValue(deriveSupplierPayments(readWorkflowState()));
            case 'UpdateSupplierInvoice': {
              const [payload] = args;
              const state = readWorkflowState();
              const invoice = findSupplierInvoice(state, String(payload?.id || ''));
              if (!invoice) {
                return Promise.reject(new Error('Supplier invoice not found'));
              }
              Object.assign(invoice, payload || {});
              writeWorkflowState(state);
              return defaultValue(invoice);
            }
            case 'MarkSupplierInvoicePaid': {
              const [invoiceId, paymentReference, paymentMethod] = args;
              const state = readWorkflowState();
              const invoice = findSupplierInvoice(state, String(invoiceId));
              if (!invoice) {
                return Promise.reject(new Error('Supplier invoice not found'));
              }
              invoice.payment_status = 'Paid';
              invoice.status = 'Paid';
              invoice.payment_ref = String(paymentReference || '');
              invoice.payment_method = String(paymentMethod || 'Bank Transfer');
              invoice.payment_date = now();
              writeWorkflowState(state);
              return defaultVoid();
            }
            case 'PerformThreeWayMatch':
              return defaultValue({ matched: true, reason: '' });
            case 'ApproveSupplierInvoice':
              return defaultVoid();
            case 'GetSupportedCurrencies':
              return defaultValue([{ code: 'BHD', name: 'Bahraini Dinar' }, { code: 'USD', name: 'US Dollar' }]);
            case 'GetInvoicesByStatus': {
              const state = readWorkflowState();
              return defaultValue((state.customerInvoices || []).filter((item: any) => String(item.status) === String(args[0])));
            }
            case 'GetActiveBankAccounts':
              return defaultValue((readWorkflowState().bankAccounts || []).filter((item: any) => item.is_active !== false));
            case 'GetAllBankAccounts':
              return defaultValue(readWorkflowState().bankAccounts);
            case 'GetCashPosition': {
              const state = readWorkflowState();
              const total = (state.bankAccounts || []).reduce((sum: number, _item: any) => sum + 1500, 0);
              return defaultValue({ total_bhd: total, by_account: { 'bank-1': 1500 }, as_of: now() });
            }
            case 'GetBankStatements': {
              const state = readWorkflowState();
              return defaultValue((state.bankStatements || []).filter((item: any) => String(item.bank_account_id) === String(args[0])));
            }
            case 'GetBankStatementLines': {
              const state = readWorkflowState();
              return defaultValue((state.bankStatementLines || []).filter((item: any) => String(item.bank_statement_id) === String(args[0])));
            }
            case 'ManualMatchLine': {
              const [lineId, entityType, entityId] = args;
              const state = readWorkflowState();
              const line = (state.bankStatementLines || []).find((item: any) => String(item.id) === String(lineId));
              if (!line) {
                return Promise.reject(new Error('Bank statement line not found'));
              }
              line.is_matched = true;
              line.match_type = 'Manual';
              line.match_confidence = 1;
              if (entityType === 'SUPPLIER_PAYMENT') {
                line.matched_payment_id = String(entityId);
              } else if (entityType === 'SUPPLIER_INVOICE') {
                line.matched_payment_ids = String(entityId);
              } else if (entityType === 'CUSTOMER_INVOICE') {
                line.matched_invoice_ids = String(entityId);
              }
              writeWorkflowState(state);
              return defaultVoid();
            }
            case 'GetPurchaseOrders':
              return defaultValue(readWorkflowState().purchaseOrders);
            case 'GetPurchaseOrderByID': {
              const state = readWorkflowState();
              return defaultValue(state.purchaseOrders.find((item: any) => item.id === args[0]) || state.purchaseOrders[0]);
            }
            case 'CreatePurchaseOrder': {
              const state = readWorkflowState();
              const payload = args[0] || {};
              const supplier = state.suppliers.find((item: any) => item.id === payload.supplier_id);
              const po = {
                id: `po-${state.nextPOId}`,
                po_number: `PO-2026-${String(state.nextPOId).padStart(3, '0')}`,
                supplier_name: supplier?.supplier_name || 'Unknown Supplier',
                payment_due_date: payload.expected_delivery,
                ...payload,
              };
              state.nextPOId += 1;
              state.purchaseOrders.unshift(po);
              writeWorkflowState(state);
              return defaultValue(po);
            }
            default:
              return defaultVoid();
          }
        };
      },
    });

    const runtime = new Proxy({}, {
      get(_, prop) {
        switch (String(prop)) {
          case 'EventsOn':
            return () => () => {};
          case 'OnFileDrop':
            return (callback: Function, useDropTarget = true) => {
              (window as any).__wailsFileDropCallbacks.push({ callback, useDropTarget });
              return () => {};
            };
          case 'EventsOff':
          case 'OnFileDropOff':
          case 'EventsOffAll':
          case 'BrowserOpenURL':
          case 'WindowReload':
          case 'WindowReloadApp':
          case 'LogInfo':
          case 'LogWarning':
          case 'LogError':
          case 'LogDebug':
          case 'LogTrace':
          case 'LogPrint':
            return () => {};
          default:
            return () => {};
        }
      },
    });

    // InfraService: the app's second bound Go service. The generated bindings
    // call window.go.main.InfraService.<Method>(...). Delegate every method to
    // the same App proxy (safe no-op defaults for anything unmocked), but serve
    // real translations for GetTranslations so startup i18n succeeds.
    const translations = (opts && (opts as any).translations) || {};
    const infra = new Proxy({}, {
      get(_, prop) {
        if (String(prop) === 'GetTranslations') {
          return () => defaultValue(translations);
        }
        if (String(prop) === 'GetAvailableLocales') {
          return () => defaultValue(['en', 'ar', 'hi', 'fr', 'es']);
        }
        // Everything else routes through the App proxy's generic dispatch.
        return (app as any)[prop];
      },
    });

    // The Go backend is split across several bound services (App, CRMService,
    // FinanceService, DocumentsService, ButlerService, SyncServiceBinding). The
    // generated bindings call window.go.main.<Service>.<Method>(...). Route every
    // service through the same App proxy (method-name dispatch + safe no-op
    // default) so a screen that calls any of them still renders its layout.
    window.go = {
      main: {
        App: app,
        InfraService: infra,
        CRMService: app,
        FinanceService: app,
        DocumentsService: app,
        ButlerService: app,
        SyncServiceBinding: app,
      },
    } as any;
    window.runtime = runtime as any;
  }, options);
}
