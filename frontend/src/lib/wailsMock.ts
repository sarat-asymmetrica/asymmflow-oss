/**
 * DESIGN MODE - Wails Mock Layer
 * 
 * This file provides mock data for the Wails backend when running in design mode.
 * It's loaded early and sets up window.go to intercept all backend calls.
 * 
 * Usage: Import this at the top of main.ts before any other imports.
 */

const DESIGN_MODE = import.meta.env.VITE_DESIGN_MODE === 'true';

if (DESIGN_MODE) {
    console.log('DESIGN MODE: Initializing mock Wails layer');

    // ==============================================================
    // MOCK DATA - Representative sample data for design work
    // ==============================================================

    const mockDashboardStats = {
        TotalCustomers: 42,
        ActiveOrders: 15,
        PendingRFQs: 8,
        MonthlyRevenue: 125000,
        OpenInvoices: 23,
        OverdueInvoices: 3,
        TotalSuppliers: 12,
        ActiveOffers: 7,
        ConversionRate: 0.68,
        AvgOrderValue: 8500,
    };

    const mockSurvivalMetrics = {
        CashRunway: 180,
        BurnRate: 45000,
        RevenueGrowth: 0.12,
        CustomerChurn: 0.05,
        ARDays: 45,
        APDays: 30,
        GrossMargin: 0.35,
        NetMargin: 0.15,
        QuickRatio: 1.8,
        CurrentRatio: 2.5,
    };

    const mockAlertSummary = {
        Critical: 2,
        Warning: 5,
        Info: 12,
        TotalActive: 19,
    };

    const mockCustomers = [
        { ID: 1, Name: 'ADNOC Distribution', Code: 'ADNOC', Grade: 'A', Country: 'UAE', Email: 'procurement@adnoc.ae' },
        { ID: 2, Name: 'Dubai Municipality', Code: 'DM', Grade: 'A', Country: 'UAE', Email: 'supplies@dm.gov.ae' },
        { ID: 3, Name: 'Saudi Aramco', Code: 'ARAMCO', Grade: 'A+', Country: 'KSA', Email: 'vendor@aramco.com' },
        { ID: 4, Name: 'Qatar Petroleum', Code: 'QP', Grade: 'A', Country: 'Qatar', Email: 'procurement@qp.qa' },
        { ID: 5, Name: 'ENOC', Code: 'ENOC', Grade: 'B+', Country: 'UAE', Email: 'supplies@enoc.com' },
    ];

    const mockOrders = [
        { ID: 1, CustomerName: 'ADNOC Distribution', OrderNumber: 'ORD-2024-001', Status: 'In Progress', Total: 45000, Date: '2024-12-15' },
        { ID: 2, CustomerName: 'Dubai Municipality', OrderNumber: 'ORD-2024-002', Status: 'Pending', Total: 28000, Date: '2024-12-18' },
        { ID: 3, CustomerName: 'Saudi Aramco', OrderNumber: 'ORD-2024-003', Status: 'Delivered', Total: 125000, Date: '2024-12-10' },
    ];

    const mockOffers = [
        { ID: 1, CustomerName: 'ADNOC', Reference: 'OFF-2024-001', Status: 'Pending Review', Value: 55000, Date: '2024-12-20' },
        { ID: 2, CustomerName: 'ENOC', Reference: 'OFF-2024-002', Status: 'Sent', Value: 32000, Date: '2024-12-22' },
    ];

    const mockRFQs = [
        { ID: 1, CustomerName: 'Dubai Municipality', Reference: 'RFQ-2024-001', Status: 'New', Items: 5, Date: '2024-12-27' },
        { ID: 2, CustomerName: 'Qatar Petroleum', Reference: 'RFQ-2024-002', Status: 'In Review', Items: 12, Date: '2024-12-26' },
    ];

    const mockActiveAlerts = [
        { ID: 1, Type: 'critical', Message: 'Invoice #INV-2024-045 overdue by 15 days', Entity: 'ADNOC Distribution', CreatedAt: new Date().toISOString() },
        { ID: 2, Type: 'warning', Message: 'Low stock: Thermal Paste (5 units remaining)', Entity: 'Inventory', CreatedAt: new Date().toISOString() },
        { ID: 3, Type: 'info', Message: 'New RFQ received from Dubai Municipality', Entity: 'Inbox', CreatedAt: new Date().toISOString() },
    ];

    const mockUser = {
        ID: 1,
        Username: 'designer',
        Email: 'designer@asymmetrica.dev',
        Role: 'Admin',
        Permissions: ['read', 'write', 'admin'],
    };

    const mockSettings = {
        CompanyName: 'Acme Instrumentation Trading LLC',
        Currency: 'AED',
        Locale: 'en-AE',
        Theme: 'wabi-sabi',
        EmailNotifications: true,
    };

    // Mirrors overlay.BuiltinDefaults()'s division vocabulary (see
    // frontend/src/lib/divisions.svelte.ts's BUILTIN_DIVISION_REGISTRY) so
    // DESIGN_MODE exercises the same shape GetDivisionRegistry returns for real.
    const mockDivisionRegistry = {
        divisions: [
            { key: 'Acme Instrumentation', legalName: 'ACME INSTRUMENTATION W.L.L', aliases: [] },
            {
                key: 'Beacon Controls',
                legalName: 'BEACON CONTROLS W.L.L.',
                aliases: ['beacon controls wll', 'beacon controls w.l.l', 'beacon controls w.l.l.'],
            },
        ],
        defaultKey: 'Acme Instrumentation',
        companyDisplayName: 'Acme Instrumentation WLL',
    };

    const mockDashboardEvents = [
        { Type: 'order_created', Description: 'New order from ADNOC', Timestamp: new Date().toISOString() },
        { Type: 'payment_received', Description: 'Payment received: AED 45,000', Timestamp: new Date(Date.now() - 3600000).toISOString() },
        { Type: 'rfq_received', Description: 'RFQ from Dubai Municipality', Timestamp: new Date(Date.now() - 7200000).toISOString() },
    ];

    const mockFollowUps = [
        { ID: 1, Title: 'Follow up on ADNOC quotation', CustomerName: 'ADNOC Distribution', DueDate: '2024-12-30', Status: 'pending' },
        { ID: 2, Title: 'Send revised pricing to ENOC', CustomerName: 'ENOC', DueDate: '2024-12-29', Status: 'pending' },
    ];

    const mockAuthState = {
        IsAuthenticated: true,
        User: mockUser,
        Token: 'mock-token-for-design-mode',
    };

    // ==============================================================
    // CREATE MOCK WAILS GLOBAL OBJECT
    // ==============================================================

    // Helper to create async mock functions
    const asyncMock = <T>(data: T) => () => Promise.resolve(data);
    const voidMock = () => Promise.resolve();
    const emptyArrayMock = () => Promise.resolve([]);
    const emptyObjectMock = () => Promise.resolve({});

    // Create mock App object with all functions
    const mockApp: Record<string, (...args: unknown[]) => Promise<unknown>> = {
        // Dashboard & Stats
        GetDashboardStats: asyncMock(mockDashboardStats),
        GetSurvivalMetrics: asyncMock(mockSurvivalMetrics),
        GetSurvivalMetricsOptimized: asyncMock(mockSurvivalMetrics),
        GetAlertSummary: asyncMock(mockAlertSummary),
        GetActiveAlerts: asyncMock(mockActiveAlerts),
        GetActiveAlertsOptimized: asyncMock(mockActiveAlerts),
        GetDashboardEvents: asyncMock(mockDashboardEvents),
        GetStatistics: asyncMock(mockDashboardStats),

        // Auth & User
        GetAuthState: asyncMock(mockAuthState),
        GetCurrentUserStub: asyncMock(mockUser),
        GetUser: asyncMock(mockUser),
        GetUserPermissions: asyncMock(['read', 'write', 'admin']),
        HasPermission: asyncMock(true),
        ListUsers: asyncMock([mockUser]),
        ListRoles: asyncMock([{ ID: 1, Name: 'Admin', Permissions: ['all'] }]),
        StartLogin: asyncMock('mock-login-url'),
        Logout: voidMock,
        RefreshAuth: voidMock,

        // Settings
        GetSettings: asyncMock(mockSettings),
        GetDivisionRegistry: asyncMock(mockDivisionRegistry),
        UpdateSettings: voidMock,
        GetConfig: asyncMock(mockSettings),
        GetFolderPaths: asyncMock({ Documents: 'C:\\Documents', Inbox: 'C:\\Inbox' }),
        GetApplicationPaths: asyncMock({ Data: 'C:\\Data', Config: 'C:\\Config' }),

        // Setup
        NeedsSetup: asyncMock(false),
        CompleteSetup: voidMock,

        // Customers
        GetAllCustomers: asyncMock(mockCustomers),
        ListCustomers: asyncMock(mockCustomers),
        GetCustomersByGrade: asyncMock(mockCustomers),
        GetCustomer360: asyncMock({ Customer: mockCustomers[0], Orders: mockOrders, Stats: {} }),
        GetCustomer360Optimized: asyncMock({ Customer: mockCustomers[0], Orders: mockOrders, Stats: {} }),
        GetCustomer360View: asyncMock({ Customer: mockCustomers[0], Orders: mockOrders, Stats: {} }),
        GetCustomer360Geometry: asyncMock({ Customer: mockCustomers[0] }),
        GetCustomerOpportunities: emptyArrayMock,
        GetCustomerRecentOrders: asyncMock(mockOrders),
        GetCustomerHistory: emptyArrayMock,
        GetCustomerGraph: asyncMock({ Nodes: [], Edges: [] }),
        CreateCustomer: asyncMock(mockCustomers[0]),

        // Orders
        GetOrder: asyncMock(mockOrders[0]),
        ListOrders: asyncMock(mockOrders),
        GetAllOrdersOptimized: asyncMock(mockOrders),
        FilterOrders: asyncMock(mockOrders),
        GetOrderFulfillmentStatus: asyncMock({ Status: 'Pending', Shipped: 0, Total: 100 }),

        // Offers
        GetOffers: asyncMock(mockOffers),
        CreateOffer: asyncMock(mockOffers[0]),
        UpdateOfferStatus: voidMock,
        ConvertOfferToOrder: voidMock,

        // RFQs
        GetRFQs: asyncMock(mockRFQs),
        CreateRFQ: asyncMock(mockRFQs[0]),
        UpdateRFQStatus: voidMock,

        // Suppliers
        ListSuppliers: asyncMock([{ ID: 1, Name: 'Rhine Instruments', Code: 'EH', Country: 'Germany' }]),
        UpdateSupplierGoals: voidMock,

        // Follow-ups
        ListFollowUps: asyncMock(mockFollowUps),
        GetOverdueFollowUps: asyncMock(mockFollowUps),
        CreateFollowUp: asyncMock(mockFollowUps[0]),
        CompleteFollowUp: voidMock,
        UpdateFollowUp: voidMock,

        // Alerts
        DismissAlert: voidMock,
        AcknowledgeAlert: voidMock,
        ComputeAlerts: voidMock,

        // Reports
        GetReportData: asyncMock({ Title: 'Mock Report', Data: [] }),
        GenerateDashboardReport: asyncMock('mock-report-path'),
        ExportReport: asyncMock('mock-export-path'),

        // Inbox
        GetInboxDocuments: emptyArrayMock,
        GetInboxStats: asyncMock({ Total: 5, Unread: 2, Processed: 3 }),
        ProcessInboxDocument: asyncMock({ Success: true }),
        MarkInboxDocumentProcessed: voidMock,
        GetBusinessMemoryReviewQueue: asyncMock({ queueMetrics: [], candidates: [], selected: null, actions: [] }),
        RecordBusinessMemoryReviewDecision: asyncMock({ record: {}, queue: { queueMetrics: [], candidates: [], selected: null, actions: [] } }),
        GenerateBusinessMemoryContextPack: asyncMock({ candidate_id: 'mock', context_pack_toon: '' }),

        // Shipments & Delivery
        ListShipments: emptyArrayMock,
        CreateShipment: voidMock,
        UpdateShipment: voidMock,
        ConfirmDelivery: voidMock,
        RecordPartialShipment: voidMock,

        // Invoices
        GetAllInvoicesOptimized: emptyArrayMock,
        GetARAgingReport: asyncMock({ Total: 0, Buckets: [] }),
        GetAPAgingReport: asyncMock({ Total: 0, Buckets: [] }),
        GetReceivablesAging: asyncMock({ Total: 0, Buckets: [] }),

        // Inventory
        GetInventoryItems: emptyArrayMock,
        GetInventoryItem: asyncMock({ ID: 1, Name: 'Sample Item', Quantity: 100 }),
        GetLowStockItems: emptyArrayMock,
        CreateInventoryItem: asyncMock({ ID: 1 }),
        UpdateInventoryItem: voidMock,
        GetInventoryValuation: asyncMock({ Total: 50000 }),
        RecordStockMovement: asyncMock({ ID: 1 }),
        GetStockMovements: emptyArrayMock,

        // Warehouses
        GetWarehouses: asyncMock([{ ID: 1, Name: 'Main Warehouse', Location: 'Dubai' }]),
        CreateWarehouse: asyncMock({ ID: 1 }),

        // Costing
        GetCostingSheets: emptyArrayMock,
        CreateCostingSheet: asyncMock({ ID: 1 }),
        CalculateCosting: asyncMock({ Total: 0, Breakdown: [] }),
        ApproveCostingSheet: voidMock,
        RejectCostingSheet: voidMock,

        // Contracts
        GetContracts: emptyArrayMock,
        GetContract: asyncMock({ ID: 1, Title: 'Sample Contract' }),
        GetContractTemplates: emptyArrayMock,
        GetContractsByCustomer: emptyArrayMock,
        GenerateContract: asyncMock({ ID: 1 }),
        DownloadContract: asyncMock('mock-path'),

        // OCR & Documents
        GetOCRDocuments: emptyArrayMock,
        GetOCRDocumentByID: asyncMock({ ID: 1 }),
        GetOCRStats: emptyObjectMock,
        GetOCRProcessorStats: emptyObjectMock,
        GetOCRPipelineStats: emptyObjectMock,
        ClassifyDocument: asyncMock({ Type: 'invoice', Confidence: 0.95 }),
        GetClassificationStats: emptyObjectMock,

        // Entity Graph
        GetEntityGraph: asyncMock({ Nodes: [], Edges: [] }),
        GetGraphStats: asyncMock({ NodeCount: 0, EdgeCount: 0 }),
        SearchGraphEntities: emptyArrayMock,
        BuildEntityGraph: asyncMock({ NodesCreated: 0 }),
        RebuildEntityGraph: asyncMock({ NodesCreated: 0 }),
        ExportGraphJSON: asyncMock('{}'),

        // Jobs & Async
        GetRecentJobs: emptyArrayMock,
        GetJobStatus: asyncMock({ Status: 'completed', Progress: 100 }),
        CancelJob: voidMock,
        CleanupOldJobs: voidMock,
        InitializeJobQueue: voidMock,
        ShutdownJobQueue: voidMock,
        GenerateReportAsync: asyncMock(1),

        // File Watcher
        GetWatcherStatus: asyncMock({ Running: false, WatchedPaths: [] }),
        StartFileWatcher: voidMock,
        StopFileWatcher: voidMock,
        GetRecentEvents: emptyArrayMock,
        GetRecentSyncEvents: emptyArrayMock,
        ConfigureWatchPaths: voidMock,
        UpdateFolderPaths: voidMock,

        // Sync
        GetSyncStatus: asyncMock({ Connected: false, LastSync: null }),
        TriggerSync: voidMock,
        RetryFailedSyncs: asyncMock(0),
        ClearSyncHistory: voidMock,

        // System
        DetectGPU: asyncMock({ Name: 'Mock GPU', MemoryMB: 8192 }),
        DetectSystemInfo: asyncMock({ OS: 'Windows', CPU: 'Intel', RAM: 16384 }),
        DetectOffice: asyncMock({ Installed: true, Version: '365' }),
        DetectOneDrivePath: asyncMock('C:\\Users\\OneDrive'),
        GetToolsStatus: asyncMock({ Available: true }),
        RefreshToolsStatus: asyncMock({ Available: true }),
        GetToolInstallInstructions: asyncMock('Install instructions here'),

        // Misc
        Greet: asyncMock('Hello, Designer!'),
        TimeNow: asyncMock(new Date().toISOString()),
        BrowseFolder: asyncMock('C:\\Selected\\Folder'),
        ValidateFolder: asyncMock(true),
        ValidateCustomer: asyncMock({ Valid: true, Errors: [] }),

        // Pricing
        GetPricingRecommendation: asyncMock({ RecommendedPrice: 100, Discount: 0.1 }),
        GetOptimalDiscount: asyncMock({ Discount: 0.15, Reason: 'Volume' }),
        GetWinProbability: asyncMock({ Probability: 0.75 }),
        SimulateMargin: asyncMock({ Margin: 0.25, Profit: 5000 }),

        // Predictions
        PredictPayment: asyncMock({ PredictedDays: 30, Confidence: 0.85 }),
        BatchPredict: asyncMock({ Results: [] }),
        GetHistory: emptyArrayMock,
        ClearHistory: voidMock,

        // Quick Captures
        GetQuickCaptures: emptyArrayMock,
        CreateQuickCapture: asyncMock(1),
        UpdateQuickCapture: voidMock,
        DeleteQuickCapture: voidMock,
        QuickCaptureDocument: emptyObjectMock,

        // Archaeology
        StartArchaeologyScan: asyncMock('scan-123'),
        GetScanProgress: asyncMock({ Progress: 100, Status: 'complete' }),
        GetScanResult: asyncMock({ Files: [], Summary: {} }),
        CancelScan: voidMock,

        // Butler AI
        ChatWithButler: asyncMock({ Response: 'Hello! I am the Butler in design mode.' }),

        // Williams Metrics
        GetWilliamsMetrics: asyncMock({ Score: 0.75, Factors: [] }),
        CompareWilliamsLinear: emptyObjectMock,

        // Accounting
        GetChartOfAccounts: emptyArrayMock,
        CreateAccount: asyncMock({ ID: 1 }),
        UpdateAccount: voidMock,
        GetJournalEntries: emptyArrayMock,
        CreateJournalEntry: asyncMock({ ID: 1 }),
        PostJournalEntry: voidMock,
        GetBalanceSheet: asyncMock({ Assets: 0, Liabilities: 0, Equity: 0 }),
        GetProfitLoss: asyncMock({ Revenue: 0, Expenses: 0, NetIncome: 0 }),

        // VAT
        GetVATReturns: emptyArrayMock,
        GenerateVATReturn: asyncMock({ ID: 1 }),
        FileVATReturn: voidMock,

        // Audit
        GetAuditLogs: emptyArrayMock,

        // SSOT Import
        GetSSOTImportStatus: emptyObjectMock,
        ImportSSOTData: asyncMock({ Imported: 0 }),

        // Seed
        SeedDefaultRoles: voidMock,
        SeedDefaultChartOfAccounts: voidMock,
        SeedContractData: voidMock,

        // User Management
        CreateUser: asyncMock(mockUser),
        UpdateUser: voidMock,
        DeactivateUser: voidMock,
        ResetUserPassword: voidMock,
        GetRole: asyncMock({ ID: 1, Name: 'Admin' }),

        // Performance
        ApplyPerformanceIndexes: voidMock,

        // Email
        SendReportByEmail: voidMock,
        GetAccessToken: asyncMock('mock-access-token'),

        // API Keys
        SetAPIKeys: voidMock,
        TestAIConnection: voidMock,

        // Pipeline
        GetPipelineStatistics: emptyObjectMock,
        RouteEvent: asyncMock({ Routed: true }),
        GetRoutingHistory: emptyArrayMock,

        // Stock Adjustments
        CreateStockAdjustment: asyncMock({ ID: 1 }),
        ApproveStockAdjustment: voidMock,

        // Processing
        ProcessTender: asyncMock({ Success: true }),
        ProcessInvoice: asyncMock({ Success: true }),
        ProcessRFQToOrder: asyncMock({ Success: true }),
        ProcessOffersBatch: asyncMock({ Processed: 0 }),
        ProcessDocumentWithOCR: asyncMock({ Text: 'Mock OCR result' }),
        ProcessDocumentsBatch: emptyArrayMock,
        ExtractRFQDocument: asyncMock({ Text: '' }),
        ExtractInvoiceDocument: asyncMock({ Text: '' }),
        ExtractQuotationDocument: asyncMock({ Text: '' }),
        ProcessWithTesseract: asyncMock({ Text: '' }),
        ProcessWithGPU: asyncMock({ Text: '' }),
        ProcessWithGoFitz: asyncMock({ Text: '' }),
        ProcessWithFlorence2: asyncMock({ Text: '' }),

        // Simulation
        SimulateSurvivalGarden: emptyArrayMock,

        // Folder Structure
        CreateFolderStructure: asyncMock({ Created: true }),
        WatchInboxForTestFile: voidMock,
        RunInitialScan: asyncMock({ Files: 0 }),

        // Compliance
        CheckCompliance: asyncMock({ Compliant: true }),

        // Reports
        RegisterReportHandlers: voidMock,
        CreateReportDraft: asyncMock('draft-id'),
        GenerateCustomer360Report: asyncMock('report-path'),
        GeneratePredictionHistoryReport: asyncMock('report-path'),
        StoreCustomerGraph: voidMock,
        GetNodeRelationships: emptyArrayMock,
        ExportCustomerTemplate: asyncMock({}),
        GetPaymentHistory: emptyArrayMock,
        UpdateOrderStage: voidMock,
        UpdateOrderItemShipped: voidMock,
        UpdateOrderItemInvoiced: voidMock,
    };

    // Catch-all proxy for any functions we might have missed
    const appProxy = new Proxy(mockApp, {
        get(target, prop: string) {
            if (prop in target) {
                return target[prop];
            }
            // Return a generic void mock for any unmocked function
            console.warn(`DESIGN MODE: Unmocked function called: ${prop}`);
            return () => Promise.resolve(null);
        }
    });

    // Set up the window.go global object that Wails uses
    (window as any).go = {
        main: {
            App: appProxy,
            DocumentsService: appProxy,
            InfraService: appProxy,
            FinanceService: appProxy,
            CRMService: appProxy,
            ButlerService: appProxy,
            SyncService: appProxy,
        }
    };

    console.log('DESIGN MODE: Mock Wails layer initialized successfully');
}

export { };
