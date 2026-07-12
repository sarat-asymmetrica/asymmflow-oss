<script lang="ts">
    import { run, self } from 'svelte/legacy';

    import { onDestroy, onMount } from "svelte";
    import { get } from "svelte/store";
    import { toast } from "$lib/stores/toasts";
    import { confirm } from '$lib/stores/confirm';
    import { permissions, currentUser } from '$lib/stores/authContext';
    let permissionList = $derived(Array.isArray($permissions) ? $permissions : []);
    let canView =
        $derived(permissionList.includes('*') ||
        permissionList.includes('costing:read') ||
        permissionList.includes('costing:*') ||
        permissionList.includes('offers:view') ||
        permissionList.includes('offers:create') ||
        permissionList.includes('rfq:view') ||
        permissionList.includes('orders:view'));
    import WabiSpinner from "../components/ui/WabiSpinner.svelte";
    import Button from "../components/ui/Button.svelte";
    import Card from "../components/ui/Card.svelte";
    import LineItemsEditor from "../components/ui/LineItemsEditor.svelte";
    import { createEventDispatcher } from "svelte";
    import {
        ListCustomers, GetPreparedByOptions } from "../../../wailsjs/go/main/App";
import { ListSuppliers, GetCostingSheets, GetRFQs, GetPipelineOpportunities, GetOpportunityLineItems, SaveCostingAsOffer, CreateCostingSheet, UpdateCostingSheet, GetCostingsByRFQ, SetActiveCostingRevision, CloneCostingAsNewRevision } from "../../../wailsjs/go/main/CRMService";
import { ExportCostingToPDF, ExportCostingToExcel, OpenExportedFile } from "../../../wailsjs/go/main/InfraService";
import { GetSettings } from "../../../wailsjs/go/main/DocumentsService";

    const dispatch = createEventDispatcher();
    const COSTING_DRAFT_STORAGE_KEY = 'asymmflow.costingSheet.unsavedDraft.v2';
    const COSTING_DRAFT_RESTORE_PAUSED_KEY = 'asymmflow.costingSheet.restorePaused.v1';

    
    interface Props {
        // Props
        embedded?: boolean;
        active?: boolean;
    }

    let { embedded = false, active = true }: Props = $props();

    // State
    let loading = $state(true);
    let hasLoadedInitialData = $state(false);
    let customers: any[] = $state([]);
    let costingSheets: any[] = $state([]);
    let opportunities: any[] = $state([]);
    let previewOpportunities: any[] = $state([]);
    let selectedOpportunityId: string = $state('');
    let selectedOpportunity: any = $state(null);
    let showStartFreshConfirm = $state(false); // confirm before discarding work/link on "Start Fresh"
    let sourceOfferId = '';
    let sourceOfferNumber = '';
    let showCostingForm = $state(false);
    let calculating = false;
    let saving = false;
    let exporting = $state(false);

    // Settings-driven defaults (loaded from Settings screen)
    let vatRate = $state(10);      // Percentage, loaded from settings
    let defaultMargin = 20; // Percentage, loaded from settings
    const defaultFreightPercent = 9;

    // Revision tracking state (Feature D)
    let rfqCostings: any[] = $state([]);
    let selectedRevision: any = $state(null);
    let loadingRevisions = false;
    let isDirty = $state(false); // Track unsaved changes
    let showAdvancedHeader = $state(false); // Toggle for compliance/certificate fields
    let suppressDraftPersistence = false;
    let lastPersistedDraftJSON = '';
    let hasStoredDraft = $state(false);
    let showBackWarning = $state(false);

    function parseYear(value: any): number {
        const numericYear = Number(value);
        if (Number.isFinite(numericYear) && numericYear >= 2000 && numericYear <= 2100) {
            return numericYear;
        }

        if (value) {
            const date = new Date(value);
            if (!Number.isNaN(date.getTime())) {
                return date.getFullYear();
            }
        }

        return new Date().getFullYear();
    }

    function normalizePipelineOpportunity(opp: any) {
        return {
            ...opp,
            client: opp.customer_name,
            customer: opp.customer_name,
            project: opp.title || opp.folder_name || opp.folder_number,
            project_name: opp.title || opp.folder_name || opp.folder_number,
            value: Number(opp.revenue_bhd) || 0,
            status: opp.stage || "New",
            stage: opp.stage || "New",
            rfq_number: opp.folder_number,
            year: parseYear(opp.year),
            _source: "pipeline",
        };
    }

    function normalizeRFQ(rfq: any) {
        return {
            ...rfq,
            client: rfq.client || rfq.customer || "Unknown",
            customer: rfq.client || rfq.customer || "Unknown",
            project: rfq.project || rfq.project_name || "Untitled",
            project_name: rfq.project || rfq.project_name || "Untitled",
            value: Number(rfq.value) || 0,
            status: rfq.status || rfq.stage || "New",
            stage: rfq.stage || rfq.status || "New",
            year: parseYear(rfq.created_at),
            _source: "rfq",
        };
    }

    function parseOpportunitySeedItems(productDetails?: string): any[] {
        if (!productDetails) return [];
        try {
            const parsed = JSON.parse(productDetails);
            const items = Array.isArray(parsed) ? parsed : (parsed && typeof parsed === 'object' ? [parsed] : []);
            return items.filter((item: any) => !isSyntheticSeedItem(item));
        } catch {
            return [];
        }
    }

    function isSyntheticSeedItem(item: any): boolean {
        const values = [
            item?.description,
            item?.equipment,
            item?.name,
            item?.part_number,
            item?.model,
            item?.product_code,
        ];
        return values.some((value) => /^line item\s+\d+\s*-?$/i.test(String(value || '').trim()));
    }

    function mapSeedItemsToCosting(seedItems: any[]): LineItem[] {
        // Wave 9.5 B3a: import ALL seed lines — the 10-line cap is UI-only
        // (Items persists as an opaque JSON blob with no backend limit), so
        // silently dropping items 11+ here was real, invisible data loss.
        const mapped = seedItems.map((seedItem: any) => {
            const item = createNewLineItem();
            const quantity = Number(seedItem.quantity) || 1;
            const explicitBHDUnitPrice = Number(seedItem.unit_price_bhd) || 0;
            const totalPrice = Number(seedItem.total_price) || 0;
            const foreignUnitPrice = Number(seedItem.unit_price) || (totalPrice > 0 ? totalPrice / quantity : 0);
            const currency = explicitBHDUnitPrice > 0 ? 'BHD' : String(seedItem.currency || 'BHD').trim().toUpperCase();
            const currencyRate = currencyOptions.find((option) => option.code === currency)?.rate || 1.0;

            item.equipment = seedItem.description || seedItem.equipment || seedItem.name || '';
            item.model = seedItem.part_number || seedItem.model || seedItem.product_code || seedItem.sku || '';
            item.longCode = seedItem.long_code || '';
            item.specification = seedItem.specification || '';
            item.detailedDescription = seedItem.detailed_description || '';
            item.quantity = quantity;
            item.currency = currencyOptions.some((option) => option.code === currency) ? currency : 'BHD';
            item.exchangeRate = item.currency === 'BHD' ? 1.0 : currencyRate;
            item.fobForeign = explicitBHDUnitPrice || foreignUnitPrice || 0;
            return item;
        }).filter((item: LineItem) => item.equipment || item.model || item.detailedDescription);

        return mapped.length > 0 ? mapped : [createNewLineItem()];
    }

    function isRFQOpportunity(opp: any): boolean {
        return Boolean(opp && opp._source === "rfq" && Number.isFinite(Number(opp.id)));
    }

    function getSelectedRFQId(): number {
        return isRFQOpportunity(selectedOpportunity) ? Number(selectedOpportunity.id) : 0;
    }

    function getOpportunityDisplayReference(opp: any): string {
        return opp?.rfq_ref || opp?.eh_ref || opp?.folder_number || opp?.rfq_number || "Manual";
    }

    function getOpportunityUserReference(opp: any): string {
        return String(opp?.rfq_ref || opp?.eh_ref || opp?.customer_reference || opp?.rfqReference || opp?.folder_number || opp?.rfq_number || '').trim();
    }

    function getOpportunityFolderReference(opp: any): string {
        return String(opp?.folder_number || (opp?._source === 'rfq' ? opp?.rfq_number : '') || '').trim();
    }

    function getSelectedOpportunityRecordId(): string {
        return selectedOpportunity?._source === 'pipeline' ? String(selectedOpportunity.id || '') : '';
    }

    function getOpportunityValue(opp: any): number {
        return Number(opp?.value ?? opp?.revenue_bhd ?? 0) || 0;
    }

    function formatOpportunityValue(opp: any): string {
        const value = getOpportunityValue(opp);
        if (value <= 0) {
            return 'Value pending';
        }
        return new Intl.NumberFormat('en-BH', {
            style: 'currency',
            currency: 'BHD',
        }).format(value);
    }

    // ========================================
    // HEADER SECTION - Matches Excel "Costing Sheet" header
    // ========================================
    const defaultDeliveryTermsForDivision = (division: string) =>
        division === 'Beacon Controls' ? 'DAP Bahrain at your store or Beacon Controls' : 'DAP Bahrain at your store or Acme Instrumentation';

    const normaliseDeliveryTermsForDivision = (deliveryTerms: string, division: string) => {
        const trimmed = (deliveryTerms || '').trim();
        if (!trimmed || trimmed === 'DAP Bahrain at your store or Acme Instrumentation' || trimmed === 'DAP Bahrain at your store or Beacon Controls') {
            return defaultDeliveryTermsForDivision(division);
        }
        return trimmed;
    };

    function createDefaultHeader(division = 'Acme Instrumentation') {
        // Wave 9.3 B2: default Prepared By to the current operator (Article
        // III.4) instead of leaving it blank — the picker still lets them
        // choose someone else (e.g. preparing on a colleague's behalf).
        const operator = get(currentUser);
        return {
            division,
            date: new Date().toISOString().split('T')[0],
            preparedBy: operator?.full_name || operator?.username || '',
            customerId: '',
            customerName: '',
            contactPerson: '',
            rfqReference: '',
            folderNumber: '',
            costingId: '',
            subject: '',
            quoteType: 'Quotation',
            estDelivery: '5-7 weeks',
            deliveryTerms: defaultDeliveryTermsForDivision(division),
            paymentTerms: '30 days from Date of Delivery',
            orderType: 'General',
            countryOfOrigin: 'DE',
            cocCoo: 'No',
            testCertificate: 'Additional charges applicable as per OEM terms',
            installation: 'No',
            commissioning: 'No',
            testing: 'No',
            // VAT/Tax compliance fields (Bahrain NBR)
            placeOfSupply: 'Kingdom of Bahrain',
            taxCategory: 'Standard',
            customerTRN: '',
        };
    }

    let header = $state(createDefaultHeader());

    const defaultQuotationBody = `We thank you for the opportunity and are pleased to submit our techno-commercial offer for your review.

Please find our pricing and scope below. We trust the proposal meets your requirement and look forward to your valued order.`;
    let quotationBody = $state(defaultQuotationBody);

    // Division options (sister companies)
    const divisionOptions = ['Acme Instrumentation', 'Beacon Controls'];
    let lastAppliedDivision = header.division;
    // Wave 9.3 B2: "Prepared By" options come from GetPreparedByOptions()
    // (configured signature identities + employee names — synthetic canon by
    // default, real identities via the sovereign overlay), not a hardcoded
    // synthetic staff list. Loaded in loadData().
    let preparedByOptions: string[] = $state([]);

    // Ensures the current operator is always selectable, even if the
    // backend's seed list doesn't happen to include their exact display name.
    function mergeCurrentOperatorIntoPreparedByOptions(options: string[]): string[] {
        const operator = get(currentUser);
        const operatorName = (operator?.full_name || operator?.username || '').trim();
        if (operatorName && !options.includes(operatorName)) {
            return [operatorName, ...options];
        }
        return options;
    }

    // Wave 9.3 B2: Prepared By is a real, user-chosen identity — never a
    // "System" ghost fallback. Block the save/revision action instead of
    // sending a fake name when it's genuinely empty.
    function resolvePreparedByOrBlock(actionLabel: string): string {
        const preparedBy = (header.preparedBy || '').trim();
        if (!preparedBy) {
            toast.warning(`Select who prepared this costing before ${actionLabel}.`);
            return '';
        }
        return preparedBy;
    }

    // Dropdown options from Excel "Code" sheet
    const deliveryOptions = [
        '3-5 weeks', '4-6 weeks', '5-7 weeks', '7-9 weeks', '9-11 weeks',
        '12-14 weeks', '16-18 weeks', 'On request'
    ];
    const deliveryTermsOptions = [
        'DAP Bahrain at your store or Acme Instrumentation',
        'DAP Bahrain at your store or Beacon Controls',
        'Ex-Works (EXW)',
        'Free Carrier (FCA)',
        'Delivered Duty Paid (DDP)',
    ];
    const paymentTermsOptions = [
        '100% Advance Payment with PO',
        '100% Payment Against Delivery',
        '30 days from Date of Delivery',
        '60 days from Date of Delivery',
        'Letter of Credit (LC)',
        'Project Stage Payments',
        '50% Advance + 50% Against Delivery',
    ];
    const orderTypeOptions = ['General', 'Spareparts', 'Items + Spareparts'];
    const countryOptions = [
        { code: 'DE', name: 'Germany' },
        { code: 'CH', name: 'Switzerland' },
        { code: 'FR', name: 'France' },
        { code: 'UK', name: 'United Kingdom (GB)' },
        { code: 'US', name: 'United States of America' },
        { code: 'SL', name: 'Slovenia' },
        { code: 'GR', name: 'Greece' },
        { code: 'IT', name: 'Italy' },
    ];
    let supplierOptions: { code: string; name: string; statement: string }[] = [
        { code: 'EH', name: 'Rhine Instruments', statement: '(Authorized Rhine Instruments Agency in Bahrain)' },
        { code: 'SM', name: 'Oxan Analytics', statement: '(Authorized Oxan Analytics Agency in Bahrain)' },
        { code: 'GI', name: 'Summit Gauges', statement: '(Authorized Summit Gauges Agency in Bahrain)' },
        { code: 'LG', name: 'Helvetia Metering', statement: '(Authorized Helvetia Metering Agency in Bahrain)' },
        { code: 'IS', name: 'Vertex Meters', statement: '(Authorized Vertex Meters Agency in Bahrain)' },
    ];
    const currencyOptions = [
        { code: 'BHD', rate: 1.0 },
        { code: 'EUR', rate: 0.45 },
        { code: 'USD', rate: 0.376 },
        { code: 'GBP', rate: 0.52 },
        { code: 'CHF', rate: 0.43 },
    ];
    const certificateOptions = ['Yes', 'No', 'Additional charges applicable as per OEM terms'];

    // ========================================
    // TERMS AND CONDITIONS (printed on separate page)
    // ========================================
    let termsAndConditions = $state(`1. QUOTATION VALIDITY
This quotation is valid for thirty (30) days from the date of issue.

2. PRICES
All prices are in Bahraini Dinars (BHD) unless otherwise stated. Prices are exclusive of VAT (${vatRate}%) which will be added to the invoice.

3. PAYMENT TERMS
As per the payment terms specified in this quotation. Late payments may incur interest charges.

4. DELIVERY
Delivery times are estimates and subject to manufacturer's confirmation. Acme Instrumentation shall not be liable for delays beyond our control.

5. WARRANTY
All products carry the manufacturer's standard warranty. Extended warranty options are available upon request.

6. INSTALLATION & COMMISSIONING
Installation and commissioning services are available at additional cost unless included in the quotation.

7. FORCE MAJEURE
Acme Instrumentation shall not be liable for failure to perform due to causes beyond reasonable control.

8. GOVERNING LAW
This quotation is governed by the laws of the Kingdom of Bahrain.`);

    // ========================================
    // LINE ITEMS - Up to 10 products per Excel template
    // ========================================
    interface LineItem {
        // Core item fields (supplier removed per user request - invoices link to suppliers instead)
        equipment: string;
        model: string;
        serialNumber: string;
        longCode: string;  // Long supplier order/configuration code (e.g. Rhine Instruments instrumentation codes)
        specification: string;
        // NEW: Wide description field for detailed instrumentation specs (prints under line item)
        detailedDescription: string;
        currency: string;
        quantity: number;
        fobForeign: number;
        freightPercent: number;
        freightForeign: number;
        // User-editable selling price (overrides calculated suggested price)
        userPrice: number;
        userPriceSet: boolean;
        // Calculated fields
        exchangeRate: number;
        fobBHD: number;
        freightBHD: number;
        cf: number;
        insurance: number;
        customsPercent: number;
        customsBHD: number;
        landedCost: number;
        handlingPercent: number;
        handlingBHD: number;
        financePercent: number;
        financeBHD: number;
        otherCosts: number;
        totalCost: number;
        // Markup percentage applied on cost: SellPrice = Cost * (1 + Markup/100)
        marginPercent: number;
        marginBHD: number;
        sellingPrice: number;
        suggestedPriceUnit: number;
        totalSuggestedPrice: number;
    }

    let lineItems: LineItem[] = $state([createNewLineItem()]);

    function createNewLineItem(): LineItem {
        return {
            equipment: '',
            model: '',
            serialNumber: '', // Serial number for traceability (GRN → DN → Invoice)
            longCode: '',  // Long supplier order/configuration code
            specification: '',
            detailedDescription: '', // Wide field for detailed instrumentation specs
            currency: 'BHD',
            quantity: 1,
            fobForeign: 0,
            freightPercent: defaultFreightPercent,
            freightForeign: 0,
            userPrice: 0,
            userPriceSet: false,
            exchangeRate: 1.0,
            fobBHD: 0,
            freightBHD: 0,
            cf: 0,
            insurance: 0,
            customsPercent: 5,
            customsBHD: 0,
            landedCost: 0,
            handlingPercent: 4,
            handlingBHD: 0,
            financePercent: 1,
            financeBHD: 0,
            otherCosts: 0,
            totalCost: 0,
            marginPercent: defaultMargin, // From Settings (default 20%)
            marginBHD: 0,
            sellingPrice: 0,
            suggestedPriceUnit: 0,
            totalSuggestedPrice: 0,
        };
    }

    // Wave 9.5 B3a: 10 was a UI-only artifact of the Excel template, not a
    // backend limit (Items persists as an opaque JSON blob). Raised well
    // beyond realistic costings; still enforced (and surfaced, never silent)
    // so the sheet can't grow unbounded by accident.
    const MAX_LINE_ITEMS = 100;

    function addLineItem() {
        if (lineItems.length < MAX_LINE_ITEMS) {
            lineItems = [...lineItems, createNewLineItem()];
            calculateAll();
        } else {
            toast.warning(`Maximum ${MAX_LINE_ITEMS} line items allowed`);
        }
    }

    function removeLineItem(index: number) {
        if (lineItems.length > 1) {
            lineItems = lineItems.filter((_, i) => i !== index);
            calculateAll();
        }
    }

    function toFiniteNumber(value: unknown, fallback = 0): number {
        if (value === null || value === undefined || value === '') return fallback;
        const numeric = typeof value === 'number' ? value : Number(value);
        return Number.isFinite(numeric) ? numeric : fallback;
    }

    function nonNegativeNumber(value: unknown, fallback = 0): number {
        return Math.max(0, toFiniteNumber(value, fallback));
    }

    // Issue #7: Copy first item's costs to all subsequent items
    function copyFirstItemCostsToAll() {
        if (lineItems.length <= 1) {
            toast.info('Only one item - nothing to copy to');
            return;
        }

        const first = lineItems[0];
        for (let i = 1; i < lineItems.length; i++) {
            lineItems[i].customsPercent = first.customsPercent;
            lineItems[i].handlingPercent = first.handlingPercent;
            lineItems[i].financePercent = first.financePercent;
            lineItems[i].marginPercent = first.marginPercent;
        }

        // Force reactive update
        lineItems = [...lineItems];
        calculateAll();

        toast.success(`Copied costs to ${lineItems.length - 1} items`);
    }

    // Issue #12: Filter out empty line items for validation/export
    function getValidLineItems(): LineItem[] {
        return lineItems.filter(item =>
            item.equipment?.trim() || nonNegativeNumber(item.fobForeign) > 0
        );
    }

    // ========================================
    // CALCULATIONS - Matches Excel formulas
    // ========================================
    function calculateLineItem(item: LineItem): LineItem {
        // Get exchange rate from currency
        const curr = currencyOptions.find(c => c.code === item.currency);
        item.exchangeRate = toFiniteNumber(item.exchangeRate, curr?.rate || 0.45);
        if (!(item.exchangeRate > 0)) item.exchangeRate = curr?.rate || 0.45;
        item.quantity = Math.max(1, toFiniteNumber(item.quantity, 1));
        item.fobForeign = nonNegativeNumber(item.fobForeign);
        item.insurance = nonNegativeNumber(item.insurance);
        item.customsPercent = nonNegativeNumber(item.customsPercent, 5);
        item.handlingPercent = nonNegativeNumber(item.handlingPercent, 4);
        item.financePercent = nonNegativeNumber(item.financePercent, 1);
        item.otherCosts = nonNegativeNumber(item.otherCosts);
        item.userPrice = nonNegativeNumber(item.userPrice);

        // FOB and Freight in BHD
        item.fobBHD = item.fobForeign * item.exchangeRate;
        item.freightPercent = nonNegativeNumber(item.freightPercent);
        item.freightForeign = item.fobForeign * (item.freightPercent / 100);
        item.freightBHD = item.freightForeign * item.exchangeRate;

        // C&F = FOB + Freight
        item.cf = item.fobBHD + item.freightBHD;

        // Customs
        item.customsBHD = item.cf * (item.customsPercent / 100);

        // Landed Cost = C&F + Insurance + Customs
        item.landedCost = item.cf + item.insurance + item.customsBHD;

        // Handling
        item.handlingBHD = item.landedCost * (item.handlingPercent / 100);

        // Finance Charges
        item.financeBHD = item.landedCost * (item.financePercent / 100);

        // Total Cost
        item.totalCost = item.landedCost + item.handlingBHD + item.financeBHD + item.otherCosts;

        // MARKUP CALCULATION
        item.marginPercent = nonNegativeNumber(item.marginPercent);
        item.sellingPrice = item.totalCost * (1 + item.marginPercent / 100);

        // Profit amount from markup for display
        item.marginBHD = item.sellingPrice - item.totalCost;

        // Suggested Price (rounded to nearest whole number)
        item.suggestedPriceUnit = Math.ceil(item.sellingPrice);

        // If user set a custom price, use that; otherwise use suggested
        const effectivePrice = item.userPriceSet && item.userPrice > 0
            ? item.userPrice
            : item.suggestedPriceUnit;

        // Total Suggested Price
        item.totalSuggestedPrice = effectivePrice * item.quantity;

        return item;
    }

    function recalculateLineItems() {
        lineItems = lineItems.map(item => calculateLineItem(item));
        lineItems = [...lineItems]; // Trigger reactivity
    }

    function calculateAll() {
        recalculateLineItems();
        markDirty();
    }

    function buildCostingDraftPayload() {
        return {
            schemaVersion: 2,
            savedAt: new Date().toISOString(),
            showCostingForm,
            selectedOpportunityId,
            selectedOpportunity,
            sourceOfferId,
            sourceOfferNumber,
            selectedRevisionId: selectedRevision?.id || null,
            currentCostingId,
            header: { ...header },
            lineItems: lineItems.map((item) => ({ ...item })),
            discount,
            hiddenCharges,
            vatRate,
            quotationBody,
            termsAndConditions,
        };
    }

    function persistCostingDraft() {
        if (suppressDraftPersistence || !showCostingForm || typeof localStorage === 'undefined') {
            return;
        }
        try {
            const draftJSON = JSON.stringify(buildCostingDraftPayload());
            if (draftJSON !== lastPersistedDraftJSON) {
                localStorage.setItem(COSTING_DRAFT_STORAGE_KEY, draftJSON);
                lastPersistedDraftJSON = draftJSON;
            }
            hasStoredDraft = true;
        } catch (err) {
            console.warn('Failed to persist costing draft:', err);
        }
    }

    function clearCostingDraft() {
        if (typeof localStorage !== 'undefined') {
            localStorage.removeItem(COSTING_DRAFT_STORAGE_KEY);
        }
        if (typeof sessionStorage !== 'undefined') {
            sessionStorage.removeItem(COSTING_DRAFT_RESTORE_PAUSED_KEY);
        }
        lastPersistedDraftJSON = '';
        hasStoredDraft = false;
    }

    function refreshStoredDraftState() {
        hasStoredDraft = typeof localStorage !== 'undefined' && Boolean(localStorage.getItem(COSTING_DRAFT_STORAGE_KEY));
    }

    function pauseDraftRestoreForSession() {
        if (typeof sessionStorage !== 'undefined') {
            sessionStorage.setItem(COSTING_DRAFT_RESTORE_PAUSED_KEY, '1');
        }
    }

    function resumeDraftRestoreForSession() {
        if (typeof sessionStorage !== 'undefined') {
            sessionStorage.removeItem(COSTING_DRAFT_RESTORE_PAUSED_KEY);
        }
    }

    function isDraftRestorePausedForSession() {
        return typeof sessionStorage !== 'undefined' && sessionStorage.getItem(COSTING_DRAFT_RESTORE_PAUSED_KEY) === '1';
    }

    function markDirty() {
        if (suppressDraftPersistence) return;
        isDirty = true;
        persistCostingDraft();
    }

    function restoreLocalCostingDraft(payload: any) {
        resumeDraftRestoreForSession();
        suppressDraftPersistence = true;
        try {
            const draftHeader = payload?.header || {};
            const draftDivision = draftHeader.division || header.division || 'Acme Instrumentation';
            header = {
                ...createDefaultHeader(draftDivision),
                ...draftHeader,
                deliveryTerms: normaliseDeliveryTermsForDivision(draftHeader.deliveryTerms || '', draftDivision),
            };
            selectedOpportunityId = String(payload?.selectedOpportunityId || '');
            selectedOpportunity = opportunities.find((opp: any) => String(opp.id) === selectedOpportunityId) || payload?.selectedOpportunity || null;
            sourceOfferId = String(payload?.sourceOfferId || '');
            sourceOfferNumber = String(payload?.sourceOfferNumber || '');
            selectedRevision = null;
            currentCostingId = payload?.currentCostingId || null;
            rfqCostings = [];
            lineItems = Array.isArray(payload?.lineItems) && payload.lineItems.length > 0
                ? payload.lineItems.map((item: any, index: number) => normalisePersistedLineItem(item, index))
                : [createNewLineItem()];
            discount = Number(payload?.discount ?? 0) || 0;
            hiddenCharges = Number(payload?.hiddenCharges ?? 0) || 0;
            vatRate = toFiniteNumber(payload?.vatRate, vatRate);
            quotationBody = payload?.quotationBody || defaultQuotationBody;
            termsAndConditions = payload?.termsAndConditions || termsAndConditions;
            showCostingForm = true;
            recalculateLineItems();
        } finally {
            suppressDraftPersistence = false;
        }
        isDirty = true;
        persistCostingDraft();
        toast.info('Restored an unsaved costing draft from this device.');
    }

    function maybeRestoreLocalCostingDraft() {
        if (showCostingForm || typeof localStorage === 'undefined') return;
        const draftJSON = localStorage.getItem(COSTING_DRAFT_STORAGE_KEY);
        hasStoredDraft = Boolean(draftJSON);
        if (!draftJSON) return;
        if (isDraftRestorePausedForSession()) return;
        try {
            const payload = JSON.parse(draftJSON);
            if (payload?.schemaVersion) {
                lastPersistedDraftJSON = draftJSON;
                restoreLocalCostingDraft(payload);
            }
        } catch (err) {
            console.warn('Discarding unreadable costing draft:', err);
            clearCostingDraft();
        }
    }

    function resumeStoredCostingDraft() {
        if (typeof localStorage === 'undefined') return;
        const draftJSON = localStorage.getItem(COSTING_DRAFT_STORAGE_KEY);
        if (!draftJSON) {
            refreshStoredDraftState();
            toast.warning('No saved costing draft found on this device.');
            return;
        }
        try {
            const payload = JSON.parse(draftJSON);
            if (!payload?.schemaVersion) {
                throw new Error('Draft format is not supported');
            }
            lastPersistedDraftJSON = draftJSON;
            restoreLocalCostingDraft(payload);
        } catch (err) {
            console.warn('Discarding unreadable costing draft:', err);
            clearCostingDraft();
            isDirty = false;
            toast.warning('That saved costing draft could not be reopened and was discarded.');
        }
    }

    function discardStoredCostingDraft() {
        clearCostingDraft();
        isDirty = false;
        toast.success('Saved costing draft discarded.');
    }

    function handleBeforeUnload(event: BeforeUnloadEvent) {
        if (!isDirty) return;
        persistCostingDraft();
        event.preventDefault();
        event.returnValue = 'You have unsaved costing changes. Save them or discard the draft before closing.';
    }

    // Summary calculations
    let subtotal = $derived(lineItems.reduce((sum, item) => sum + toFiniteNumber(item.totalSuggestedPrice), 0));
    let discount = $state(0);
    let hiddenCharges = $state(0);
    let effectiveDiscount = $derived(nonNegativeNumber(discount));
    let effectiveHiddenCharges = $derived(nonNegativeNumber(hiddenCharges));
    let effectiveVatRate = $derived(Math.min(100, nonNegativeNumber(vatRate, 10)));
    let netAmount = $derived(Math.max(0, subtotal - effectiveDiscount));
    let vat = $derived(netAmount * (effectiveVatRate / 100));
    let grandTotal = $derived(netAmount + vat);
    let totalCost = $derived(lineItems.reduce((sum, item) => sum + (toFiniteNumber(item.totalCost) * Math.max(1, toFiniteNumber(item.quantity, 1))), 0) + effectiveHiddenCharges);
    let profit = $derived(netAmount - totalCost);
    let profitPercent = $derived(netAmount > 0 ? (profit / netAmount) * 100 : 0);

    // Keep T&C VAT label in sync when user changes VAT rate
    run(() => {
        termsAndConditions = termsAndConditions.replace(/VAT \(\d+\.?\d*%\)/, `VAT (${effectiveVatRate}%)`);
    });

    // ========================================
    // DATA LOADING
    // ========================================
    async function loadData() {
        loading = true;
        try {
            const [c, s, rfqs, pipeline, settingsResult, preparedByResult] = await Promise.all([
                ListCustomers(500, 0),
                GetCostingSheets(20).catch(() => []),
                GetRFQs(200, 0).catch(() => []),
                GetPipelineOpportunities(500, 0).catch(() => []),
                GetSettings().catch(() => null),
                GetPreparedByOptions().catch(() => []),
            ]);
            customers = c || [];
            costingSheets = s || [];
            preparedByOptions = mergeCurrentOperatorIntoPreparedByOptions(preparedByResult || []);
            const pipelineNormalized = (pipeline || []).map(normalizePipelineOpportunity);
            const liveRfqs = (rfqs || [])
                .filter((rfq: any) => String(rfq?.rfq_number || '').trim())
                .map(normalizeRFQ);

            const rfqFolders = new Set(
                liveRfqs
                    .map((rfq: any) => String(rfq.rfq_number || '').trim())
                    .filter(Boolean)
            );
            const uniquePipeline = pipelineNormalized.filter(
                (opp: any) => !rfqFolders.has(String(opp.folder_number || '').trim())
            );

            opportunities = [...uniquePipeline, ...liveRfqs].sort((a: any, b: any) => {
                const aTime = new Date(a.updated_at || a.created_at || 0).getTime();
                const bTime = new Date(b.updated_at || b.created_at || 0).getTime();
                return bTime - aTime;
            });
            previewOpportunities = [...opportunities]
                .sort((a: any, b: any) => {
                    const aHasValue = getOpportunityValue(a) > 0 ? 1 : 0;
                    const bHasValue = getOpportunityValue(b) > 0 ? 1 : 0;
                    if (aHasValue !== bHasValue) {
                        return bHasValue - aHasValue;
                    }
                    const aTime = new Date(a.updated_at || a.created_at || 0).getTime();
                    const bTime = new Date(b.updated_at || b.created_at || 0).getTime();
                    return bTime - aTime;
                })
                .slice(0, 6);

            // Apply settings-driven defaults
            if (settingsResult?.business) {
                if (typeof settingsResult.business.vat_rate === 'number') {
                    vatRate = settingsResult.business.vat_rate;
                }
                if (typeof settingsResult.business.default_margin === 'number') {
                    defaultMargin = settingsResult.business.default_margin;
                }
                // Update terms with actual VAT rate from settings
                termsAndConditions = termsAndConditions.replace(/VAT \(\d+%\)/, `VAT (${vatRate}%)`);
            }

            // Load suppliers from database
            try {
                const dbSuppliers = await ListSuppliers(1000, 0);
                if (dbSuppliers && dbSuppliers.length > 0) {
                    supplierOptions = dbSuppliers.map((s: any) => ({
                        code: s.supplier_code || s.id?.slice(0, 4)?.toUpperCase() || 'XX',
                        name: s.supplier_name || 'Unknown',
                        statement: s.category ? `(${s.category})` : ''
                    }));
                }
            } catch (e) {
                console.warn('Using default supplier options:', e);
            }

            let restoredLaunchPayload = false;
            const pendingCostingOpportunity = sessionStorage.getItem('asymmflow.pendingCostingOpportunity');
            if (pendingCostingOpportunity) {
                try {
                    const pending = JSON.parse(pendingCostingOpportunity);
                    const match = findOpportunityFromPendingLaunch(pending);
                    if (match) {
                        restoredLaunchPayload = true;
                        selectedOpportunityId = String(match.id);
                        await handleOpportunitySelect();
                    } else {
                        toast.warning('Could not reopen that opportunity in costing. Please select it from the list.');
                    }
                } finally {
                    sessionStorage.removeItem('asymmflow.pendingCostingOpportunity');
                }
            }

            const pendingCostingOffer = sessionStorage.getItem('asymmflow.pendingCostingOffer');
            if (pendingCostingOffer) {
                try {
                    const pending = JSON.parse(pendingCostingOffer);
                    restoredLaunchPayload = true;
                    restoreOfferCostingPayload(pending);
                } finally {
                    sessionStorage.removeItem('asymmflow.pendingCostingOffer');
                }
            }

            if (!restoredLaunchPayload && active) {
                maybeRestoreLocalCostingDraft();
            } else {
                refreshStoredDraftState();
            }
        } catch (err) {
            console.error('Failed to load data:', err);
            toast.danger("Failed to load data");
        } finally {
            loading = false;
        }
    }

    async function handleOpportunitySelect() {
        if (!selectedOpportunityId) {
            showCostingForm = false;
            selectedOpportunity = null;
            sourceOfferId = '';
            sourceOfferNumber = '';
            rfqCostings = [];
            selectedRevision = null;
            currentCostingId = null;
            return;
        }

        resumeDraftRestoreForSession();
        selectedOpportunity = opportunities.find(o => String(o.id) === selectedOpportunityId);
        if (selectedOpportunity) {
            sourceOfferId = '';
            sourceOfferNumber = '';
            let restoredRevision = false;
            if (isRFQOpportunity(selectedOpportunity)) {
                restoredRevision = await loadRevisionsForRFQ(Number(selectedOpportunity.id));
            } else {
                rfqCostings = [];
                selectedRevision = null;
                currentCostingId = null;
            }

            // Pre-fill the commercial header from the selected opportunity, including
            // the user-facing reference so it survives the costing -> offer flow.
            const customerName = selectedOpportunity.client || selectedOpportunity.customer || '';
            const projectName = selectedOpportunity.project || selectedOpportunity.project_name || '';
            const opportunityReference = getOpportunityUserReference(selectedOpportunity);
            const opportunityFolder = getOpportunityFolderReference(selectedOpportunity);
            const opportunitySubject = selectedOpportunity.title || selectedOpportunity.project_name || selectedOpportunity.project || '';

            // Find matching customer in customers list
            const matchedCustomer = findMatchingCustomerByName(customerName);

            if (!restoredRevision) {
                const nextDivision = selectedOpportunity.division || header.division || 'Acme Instrumentation';
                header = {
                    ...header,
                    customerName: matchedCustomer?.business_name || customerName,
                    customerId: matchedCustomer?.id || '',
                    contactPerson: matchedCustomer?.contact_person || matchedCustomer?.primary_contact || '',
                    rfqReference: opportunityReference,
                    folderNumber: opportunityFolder,
                    costingId: opportunityReference || opportunityFolder,
                    subject: opportunitySubject,
                    division: nextDivision,
                    deliveryTerms: normaliseDeliveryTermsForDivision(header.deliveryTerms, nextDivision),
                };
            }

            // ========================================
            // AUTO-FILL LINE ITEMS from structured opportunity data
            // ========================================
            if (!restoredRevision) {
                const seedItems = parseOpportunitySeedItems(selectedOpportunity.product_details);

                if (seedItems.length > 0) {
                    lineItems = mapSeedItemsToCosting(seedItems);
                    calculateAll();
                    toast.success(`Loaded ${lineItems.length} line items from opportunity`);
                } else if (selectedOpportunity._source === 'pipeline') {
                    try {
                        const linkedItems = await GetOpportunityLineItems(String(selectedOpportunity.id));
                        if (Array.isArray(linkedItems) && linkedItems.length > 0) {
                            lineItems = mapSeedItemsToCosting(linkedItems);
                            calculateAll();
                            toast.success(`Loaded ${lineItems.length} line items from linked opportunity data`);
                        } else {
                            lineItems = [createNewLineItem()];
                            toast.info('No structured line items were captured for this opportunity yet.');
                        }
                    } catch (e) {
                        console.warn('Failed to load structured opportunity line items:', e);
                        lineItems = [createNewLineItem()];
                    }
                } else {
                    lineItems = [createNewLineItem()];
                }
            }

            showCostingForm = true;
            if (!restoredRevision) {
                markDirty();
            }
            toast.success(`Loaded opportunity: ${projectName}`);
        }
    }

    // ========================================
    // REVISION MANAGEMENT (Feature D)
    // ========================================
    async function loadRevisionsForRFQ(rfqId: number): Promise<boolean> {
        loadingRevisions = true;
        try {
            rfqCostings = await GetCostingsByRFQ(rfqId) || [];

            // Auto-select the active revision if exists
            const active = rfqCostings.find(c => c.is_active);
            if (active) {
                selectedRevision = active;
                const payload = JSON.parse(active.items || '[]');
                restoreCostingPayload(active, payload);
                toast.info(`Loaded active revision ${active.revision_number}`);
                return true;
            } else {
                // No existing revisions - start fresh
                selectedRevision = null;
                currentCostingId = null;
                lineItems = [createNewLineItem()];
                return false;
            }
        } catch (err) {
            console.error('Failed to load revisions:', err);
            rfqCostings = [];
            return false;
        } finally {
            loadingRevisions = false;
        }
    }

    async function selectRevision(rev: any) {
        selectedRevision = rev;

        // Load the revision's items into the form
        try {
            const payload = JSON.parse(rev.items || '[]');
            restoreCostingPayload(rev, payload);
            toast.success(`Loaded revision ${rev.revision_number}`);
        } catch (err) {
            toast.danger('Failed to load revision: ' + String(err));
        }
    }

    // Wave 9.5 B3b: this is the standalone "Save Costing" action too (see the
    // header-bar button) — same CreateCostingSheet-for-first-save /
    // CloneCostingAsNewRevision-thereafter logic that already backed "+ New
    // Revision"; only the reachability changed, not the save/revision logic.
    let savingRevision = $state(false);

    async function handleCreateNewRevision() {
        if (savingRevision) return; // Prevent double-click
        if (!selectedOpportunity) return;
        if (!isRFQOpportunity(selectedOpportunity)) {
            toast.info('Revision history is available for RFQ records only.');
            return;
        }

        const preparedBy = resolvePreparedByOrBlock('creating a revision');
        if (!preparedBy) return;

        savingRevision = true;
        try {
            // Clone current revision or create fresh
            const sourceId = selectedRevision?.id;
            let newCosting;

            if (sourceId) {
                newCosting = await CloneCostingAsNewRevision(sourceId, preparedBy);
            } else {
                // Create from current form data
                const itemsJSON = JSON.stringify(buildExportData());
                newCosting = await CreateCostingSheet(Number(selectedOpportunity.id), itemsJSON, preparedBy);
            }

            toast.success(`Created Revision ${newCosting.revision_number}`);
            await loadRevisionsForRFQ(Number(selectedOpportunity.id));
        } catch (err) {
            toast.danger('Failed to create revision: ' + String(err));
        } finally {
            savingRevision = false;
        }
    }

    async function handleSetActiveRevision(costingId: number) {
        if (!isRFQOpportunity(selectedOpportunity)) {
            toast.info('Revision history is available for RFQ records only.');
            return;
        }
        try {
            await SetActiveCostingRevision(costingId);
            toast.success('Revision set as current');
            await loadRevisionsForRFQ(Number(selectedOpportunity.id));
        } catch (err) {
            toast.danger('Failed to set active revision: ' + String(err));
        }
    }

    function formatDate(dateStr: string): string {
        const d = new Date(dateStr);
        return d.toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' });
    }

    function handleCustomerChange() {
        const customer = customers.find(c => c.id === header.customerId);
        if (customer) {
            header = {
                ...header,
                customerName: customer.business_name || customer.name || '',
                contactPerson: customer.contact_person || customer.primary_contact || '',
            };
            markDirty();
        }
    }

    function normalizeCustomerName(value: string) {
        return (value || '').trim().toLowerCase().replace(/\s+/g, ' ');
    }

    function normalizePartyLookupName(value: string) {
        return normalizeCustomerName(value)
            .replace(/&/g, ' and ')
            .replace(/[^a-z0-9]+/g, ' ')
            .replace(/\b(wll|llc|bsc|ltd|limited|company|co|with|liability)\b/g, ' ')
            .replace(/\s+/g, ' ')
            .trim();
    }

    function namesRepresentSameParty(a: string, b: string) {
        const left = normalizePartyLookupName(a);
        const right = normalizePartyLookupName(b);
        if (!left || !right) return false;
        if (left === right) return true;
        return left.length >= 8 && right.length >= 8 && (left.includes(right) || right.includes(left));
    }

    function findMatchingCustomerByName(customerName: string) {
        const normalizedName = normalizePartyLookupName(customerName);
        if (!normalizedName) return null;
        return customers.find((c: any) => {
            const names = [
                c.business_name,
                c.name,
                c.customer_name,
                c.customer_id,
                c.customer_code,
            ];
            return names.some((candidate) => namesRepresentSameParty(String(candidate || ''), customerName));
        }) || null;
    }

    function findOpportunityFromPendingLaunch(pending: any) {
        const pendingId = String(pending?.id || '').trim();
        const pendingReference = String(pending?.rfq_ref || pending?.eh_ref || pending?.folder_number || pending?.rfq_number || '').trim();
        const pendingCustomer = String(pending?.customer_name || pending?.client || pending?.customer || '').trim();
        // Ordered passes (highest confidence first). A single combined predicate
        // could short-circuit to an early same-customer row before a later exact
        // id/ref match was ever considered (the list is recency-sorted), which is
        // what linked costings to the WRONG opportunity. Each pass scans the WHOLE
        // list so an exact match always beats a customer-name guess.

        // 1. Exact opportunity record id.
        if (pendingId) {
            const byId = opportunities.find((opp: any) => String(opp?.id || '').trim() === pendingId);
            if (byId) return byId;
        }

        // 2. Exact user-facing reference / folder number.
        if (pendingReference) {
            const byReference = opportunities.find((opp: any) => {
                const oppReference = getOpportunityUserReference(opp);
                return Boolean(oppReference) && oppReference === pendingReference;
            });
            if (byReference) return byReference;
        }

        // 3. Last resort: same customer (name guess — may match the wrong row).
        if (pendingCustomer) {
            const byCustomer = opportunities.find((opp: any) =>
                namesRepresentSameParty(opp?.client || opp?.customer || opp?.customer_name || '', pendingCustomer)
            );
            if (byCustomer) return byCustomer;
        }

        return null;
    }

    function resolveCustomerForCostingPayload(customerId: string, customerName: string) {
        const byId = customerId ? customers.find((c: any) => c.id === customerId) : null;
        if (byId) {
            return {
                customerId: byId.id,
                customerName: byId.business_name || customerName || '',
            };
        }

        const byName = findMatchingCustomerByName(customerName);

        return {
            customerId: byName?.id || customerId || '',
            customerName: byName?.business_name || customerName || '',
        };
    }

    // D3: unlink a costing that auto-linked to the wrong opportunity. The link
    // lives purely in this frontend state (selectedOpportunity drives both the
    // "Linked Opportunity" display and the ids carried into the saved offer),
    // so clearing it fully unlinks the costing, leaving header + lines intact.
    function disconnectFromOpportunity() {
        if (!selectedOpportunity && !selectedOpportunityId) return;
        selectedOpportunityId = '';
        selectedOpportunity = null;
        markDirty();
        toast.info('Costing disconnected from opportunity. It is now unlinked — link another opportunity or save it as-is.');
    }

    // "Start Fresh" wipes the header, all line items and any opportunity link.
    // Confirm first when there is something to lose (unsaved edits or a linked
    // opportunity); a pristine, unlinked costing starts fresh without friction.
    function requestStartFresh() {
        if (isDirty || selectedOpportunity || selectedOpportunityId) {
            showStartFreshConfirm = true;
            return;
        }
        startNewCosting();
    }

    function confirmStartFresh() {
        showStartFreshConfirm = false;
        startNewCosting();
    }

    function startNewCosting() {
        // Start a blank costing without opportunity
        resumeDraftRestoreForSession();
        selectedOpportunityId = '';
        selectedOpportunity = null;
        sourceOfferId = '';
        sourceOfferNumber = '';
        showCostingForm = true;
        // Reset header
        header = createDefaultHeader();
        lineItems = [createNewLineItem()];
        quotationBody = defaultQuotationBody;
        isDirty = true;
        persistCostingDraft();
    }

    function returnToCostingList() {
        if (isDirty) {
            persistCostingDraft();
            toast.info('Costing draft saved on this device. Resume it from the Costing page when needed.');
        }
        showBackWarning = false;
        showCostingForm = false;
        selectedOpportunityId = '';
        selectedOpportunity = null;
        selectedRevision = null;
        currentCostingId = null;
        rfqCostings = [];
        refreshStoredDraftState();
        if (hasStoredDraft) {
            pauseDraftRestoreForSession();
        }
        dispatch('navigate', { screen: 'opportunities', tab: 'costing' });
    }

    function handleBackToOpportunities() {
        if (isDirty) {
            persistCostingDraft();
            showBackWarning = true;
            return;
        }
        returnToCostingList();
    }

    function handleCurrencyChange(index: number) {
        const item = lineItems[index];
        const curr = currencyOptions.find(c => c.code === item.currency);
        if (curr) {
            item.exchangeRate = curr.rate;
            calculateLineItem(item);
            lineItems = [...lineItems];
        }
    }

    // Wave 9.5 B4: extracted verbatim from the inline oninput handler on the
    // "Manual Unit Price" field so LineItemsEditor (presentation) can call it
    // without duplicating the userPriceSet / calculateAll logic (math stays here).
    function handleUserPriceInput(item: LineItem) {
        item.userPriceSet = nonNegativeNumber(item.userPrice) > 0;
        calculateAll();
    }

    // Format currency
    function formatBHD(val: unknown): string {
        return toFiniteNumber(val).toLocaleString('en-US', {
            minimumFractionDigits: 3,
            maximumFractionDigits: 3
        }) + ' BHD';
    }

    function formatPercent(val: unknown): string {
        return toFiniteNumber(val).toFixed(0) + '%';
    }

    function formatNumber(val: unknown, decimals = 0): string {
        return toFiniteNumber(val).toLocaleString('en-US', {
            minimumFractionDigits: decimals,
            maximumFractionDigits: decimals
        });
    }

    function normalisePersistedLineItem(item: any, index: number) {
        const rawMargin = typeof item.markupPercent === 'number'
            ? item.markupPercent
            : (typeof item.marginPercent === 'number' ? item.marginPercent : (item.margin_percent || 0));
        const marginPercent = rawMargin > 0 && rawMargin <= 1 ? rawMargin * 100 : rawMargin;
        const quantity = Number(item.quantity || 1) || 1;
        const userPrice = Number(item.userPrice ?? item.user_price ?? item.selling_price ?? 0) || 0;
        const suggestedPrice = Number(item.suggestedPrice ?? item.suggested_price ?? userPrice ?? 0) || 0;
        const fobForeign = Number(item.fob ?? item.unit_cost ?? 0) || 0;
        const freightPercent = Number(item.freightPercent ?? item.freight_percent ?? 0) || (
            fobForeign > 0 ? ((Number(item.freight ?? 0) || 0) / fobForeign) * 100 : defaultFreightPercent
        );

        return {
            ...createNewLineItem(),
            slNo: item.slNo || item.sl_no || index + 1,
            equipment: item.equipment || item.description || '',
            model: item.model || item.product_code || item.product_id || '',
            serialNumber: item.serialNumber || item.serial_number || '',
            longCode: item.longCode || item.long_code || '',
            specification: item.specification || '',
            detailedDescription: item.detailedDescription || item.detailed_description || '',
            currency: item.currency || 'BHD',
            quantity,
            fobForeign,
            freightPercent,
            freightForeign: Number(item.freight ?? 0) || 0,
            totalCost: Number(item.totalCost ?? item.total_cost ?? 0) || 0,
            marginPercent,
            suggestedPriceUnit: suggestedPrice || Number(item.suggestedPriceUnit ?? 0) || 0,
            totalSuggestedPrice: Number(item.totalPrice ?? item.total_price ?? (suggestedPrice * quantity)) || 0,
            exchangeRate: Number(item.exchangeRate ?? item.exchange_rate ?? 1) || 1,
            fobBHD: Number(item.fobBHD ?? item.fob_bhd ?? 0) || 0,
            freightBHD: Number(item.freightBHD ?? item.freight_bhd ?? 0) || 0,
            insurance: Number(item.insurance ?? 0) || 0,
            customsPercent: Number(item.customsPercent ?? item.customs_percent ?? 5) || 0,
            customsBHD: Number(item.customsBHD ?? item.customs_bhd ?? 0) || 0,
            handlingPercent: Number(item.handlingPercent ?? item.handling_percent ?? 4) || 0,
            handlingBHD: Number(item.handlingBHD ?? item.handling_bhd ?? 0) || 0,
            financePercent: Number(item.financePercent ?? item.finance_percent ?? 1) || 0,
            financeBHD: Number(item.financeBHD ?? item.finance_bhd ?? 0) || 0,
            otherCosts: Number(item.otherCosts ?? item.other_costs ?? 0) || 0,
            userPrice,
            userPriceSet: Boolean(item.userPriceSet ?? item.user_price_set ?? userPrice > 0),
        };
    }

    function restoreCostingPayload(revision: any, parsed: any) {
        const payload = Array.isArray(parsed) ? { lineItems: parsed } : (parsed || {});
        const persistedLineItems = Array.isArray(payload.lineItems) ? payload.lineItems : [];
        sourceOfferId = String(payload.offerId || payload.offer_id || '');
        sourceOfferNumber = String(payload.offerNumber || payload.offer_number || '');
        const resolvedCustomer = resolveCustomerForCostingPayload(
            payload.customerId || payload.customer_id || header.customerId,
            payload.customerName || payload.customer_name || header.customerName,
        );

        lineItems = persistedLineItems.length > 0
            ? persistedLineItems.map((item: any, index: number) => normalisePersistedLineItem(item, index))
            : [createNewLineItem()];

        header = {
            ...header,
            division: payload.division || header.division,
            date: payload.date || header.date,
            preparedBy: payload.preparedBy || revision.created_by || header.preparedBy,
            customerName: resolvedCustomer.customerName || header.customerName,
            contactPerson: payload.contactPerson || header.contactPerson,
            customerId: resolvedCustomer.customerId || header.customerId,
            rfqReference: payload.rfqReference || header.rfqReference,
            folderNumber: payload.folderNumber || header.folderNumber,
            costingId: payload.costingId || header.costingId,
            subject: payload.subject || header.subject,
            estDelivery: payload.estDelivery || header.estDelivery,
            deliveryTerms: normaliseDeliveryTermsForDivision(payload.deliveryTerms || header.deliveryTerms, payload.division || header.division),
            paymentTerms: payload.paymentTerms || header.paymentTerms,
            orderType: payload.orderType || header.orderType,
            countryOfOrigin: payload.countryOfOrigin || header.countryOfOrigin,
            cocCoo: payload.cocCoo || header.cocCoo,
            testCertificate: payload.testCertificate || header.testCertificate,
            installation: payload.installation || header.installation,
            commissioning: payload.commissioning || header.commissioning,
            testing: payload.testing || header.testing,
            placeOfSupply: payload.placeOfSupply || header.placeOfSupply,
            taxCategory: payload.taxCategory || header.taxCategory,
            customerTRN: payload.customerTRN || header.customerTRN,
            quoteType: payload.quoteType || header.quoteType,
        };

        discount = Number(payload.discount ?? 0) || 0;
        hiddenCharges = Number(payload.hiddenCharges ?? 0) || 0;
        vatRate = toFiniteNumber(payload.vatRate, vatRate);
        quotationBody = payload.body || defaultQuotationBody;
        termsAndConditions = payload.termsAndConditions || termsAndConditions;
        currentCostingId = revision.id || null;
        recalculateLineItems();
        isDirty = false;
    }

    function restoreOfferCostingPayload(payload: any) {
        selectedOpportunityId = '';
        selectedOpportunity = null;
        rfqCostings = [];
        selectedRevision = null;
        currentCostingId = null;
        restoreCostingPayload({ id: null, created_by: payload.preparedBy || '' }, payload);
        showCostingForm = true;
        markDirty();
        toast.success(`Loaded ${payload.offerNumber || 'offer'} into the costing sheet`);
    }

    // ========================================
    // EXPORT FUNCTIONS
    // ========================================
    function buildExportData() {
        recalculateLineItems();
        const opportunityReference = getOpportunityUserReference(selectedOpportunity);
        const rfqReference = header.rfqReference || opportunityReference;
        const folderNumber = header.folderNumber || getOpportunityFolderReference(selectedOpportunity);
        const costingId = header.costingId || rfqReference || folderNumber;

        return {
            division: header.division,
            source: sourceOfferId ? 'offer' : '',
            offerId: sourceOfferId,
            offerNumber: sourceOfferNumber,
            date: header.date,
            preparedBy: header.preparedBy,
            customerId: header.customerId,
            customerName: header.customerName,
            contactPerson: header.contactPerson,
            rfqReference,
            folderNumber,
            costingId,
            subject: header.subject,
            estDelivery: header.estDelivery,
            deliveryTerms: normaliseDeliveryTermsForDivision(header.deliveryTerms, header.division),
            paymentTerms: header.paymentTerms,
            orderType: header.orderType,
            countryOfOrigin: header.countryOfOrigin,
            cocCoo: header.cocCoo,
            testCertificate: header.testCertificate,
            installation: header.installation,
            commissioning: header.commissioning,
            testing: header.testing,
            lineItems: getValidLineItems().map((item, i) => {
                const quantity = Math.max(1, toFiniteNumber(item.quantity, 1));
                const userPrice = nonNegativeNumber(item.userPrice);
                const suggestedPriceUnit = nonNegativeNumber(item.suggestedPriceUnit);
                const effectivePrice = item.userPriceSet && userPrice > 0
                    ? userPrice
                    : suggestedPriceUnit;
                return {
                    slNo: i + 1,
                    // supplier removed - invoices link to suppliers instead
                    supplier: '',
                    equipment: item.equipment,
                    model: item.model,
                    serialNumber: item.serialNumber || '',
                    longCode: item.longCode || '',
                    specification: item.specification || '',
                    detailedDescription: item.detailedDescription || '',
                    currency: item.currency,
                    quantity,
                    fob: nonNegativeNumber(item.fobForeign),
                    freight: nonNegativeNumber(item.freightForeign),
                    freightPercent: nonNegativeNumber(item.freightPercent),
                    totalCost: nonNegativeNumber(item.totalCost),
                    marginPercent: nonNegativeNumber(item.marginPercent),
                    markupPercent: nonNegativeNumber(item.marginPercent),
                    suggestedPrice: effectivePrice,
                    totalPrice: effectivePrice * quantity,
                    // Full cost breakdown for persistence
                    exchangeRate: toFiniteNumber(item.exchangeRate, 1),
                    fobBHD: nonNegativeNumber(item.fobBHD),
                    freightBHD: nonNegativeNumber(item.freightBHD),
                    insurance: nonNegativeNumber(item.insurance),
                    customsPercent: nonNegativeNumber(item.customsPercent),
                    customsBHD: nonNegativeNumber(item.customsBHD),
                    handlingPercent: nonNegativeNumber(item.handlingPercent),
                    handlingBHD: nonNegativeNumber(item.handlingBHD),
                    financePercent: nonNegativeNumber(item.financePercent),
                    financeBHD: nonNegativeNumber(item.financeBHD),
                    otherCosts: nonNegativeNumber(item.otherCosts),
                    userPrice,
                    userPriceSet: item.userPriceSet && userPrice > 0,
                };
            }),
            subtotal: toFiniteNumber(subtotal),
            discount: effectiveDiscount,
            netAmount: toFiniteNumber(netAmount),
            vat: toFiniteNumber(vat),
            grandTotal: toFiniteNumber(grandTotal),
            totalCost: toFiniteNumber(totalCost),
            profit: toFiniteNumber(profit),
            profitPercent: toFiniteNumber(profitPercent),
            opportunityId: getSelectedRFQId(),
            opportunityRecordId: getSelectedOpportunityRecordId(),
            projectName: selectedOpportunity?.title || selectedOpportunity?.project_name || selectedOpportunity?.project || '',
            body: quotationBody,
            termsAndConditions: termsAndConditions,
            quoteType: header.quoteType || 'Quotation',
            vatRate: effectiveVatRate,
            hiddenCharges: effectiveHiddenCharges,
            // VAT/Tax compliance
            placeOfSupply: header.placeOfSupply || 'Kingdom of Bahrain',
            taxCategory: header.taxCategory || 'Standard',
            customerTRN: header.customerTRN || '',
        };
    }

    async function handleExportPDF() {
        const validItems = getValidLineItems();
        if (validItems.length === 0) {
            toast.warning('Please add at least one line item with equipment or price');
            return;
        }

        exporting = true;
        try {
            const data = buildExportData();
            const filePath = await ExportCostingToPDF(data as any);
            if (filePath) {
                toast.success('PDF quotation generated!');
                await OpenExportedFile(filePath);
            }
        } catch (err) {
            console.error('Export PDF failed:', err);
            toast.danger('Failed to generate PDF: ' + String(err));
        } finally {
            exporting = false;
        }
    }

    async function handleExportExcel() {
        const validItems = getValidLineItems();
        if (validItems.length === 0) {
            toast.warning('Please add at least one line item with equipment or price');
            return;
        }

        exporting = true;
        try {
            const data = buildExportData();
            const filePath = await ExportCostingToExcel(data as any);
            if (filePath) {
                toast.success('Excel costing sheet generated!');
                await OpenExportedFile(filePath);
            }
        } catch (err) {
            console.error('Export Excel failed:', err);
            toast.danger('Failed to generate Excel: ' + String(err));
        } finally {
            exporting = false;
        }
    }

    let savingOffer = $state(false);
    let currentCostingId: number | null = null;

    async function handleSaveAsOffer() {
        if (savingOffer) return; // Prevent double-click

        const validItems = getValidLineItems();
        if (validItems.length === 0) {
            toast.warning('Please add at least one line item with equipment or price');
            return;
        }

        // Wave 9.5 B3d: SaveCostingAsOffer keys on OfferID, so when sourceOfferId
        // is set this call silently overwrites an existing offer. Confirm first;
        // the create-new path (no sourceOfferId) stays unconfirmed.
        if (sourceOfferId) {
            const confirmed = await confirm.ask({
                title: 'Overwrite Existing Offer',
                message: `This will overwrite offer ${sourceOfferNumber || sourceOfferId} — continue?`,
                confirmLabel: 'Overwrite',
                variant: 'warning',
            });
            if (!confirmed) return;
        }

        savingOffer = true;
        // Wave 9.5 B3c: track whether the (already non-blocking) costing-history
        // write failed so we can surface it instead of only console.warn'ing —
        // the offer save itself stays non-blocking on this.
        let costingHistorySaveFailed = false;
        try {
            const persistedCosting = buildExportData();

            // Save costing sheet to DB (links back to RFQ)
            const rfqId = getSelectedRFQId();
            if (rfqId > 0) {
                try {
                    // Wave 9.3 B2: never fall back to a "System" ghost identity — if
                    // Prepared By is genuinely unset, persist it blank rather than fake it.
                    // This inner save is already non-blocking (see catch below), so we
                    // don't halt the offer save over it.
                    if (selectedRevision?.id || currentCostingId) {
                        await UpdateCostingSheet(selectedRevision?.id || currentCostingId, {
                            ...selectedRevision,
                            items: JSON.stringify(persistedCosting),
                            created_by: header.preparedBy || '',
                        } as any);
                    } else {
                        const saved = await CreateCostingSheet(rfqId, JSON.stringify(persistedCosting), header.preparedBy || '');
                        currentCostingId = saved?.id || null;
                    }
                } catch (err) {
                    console.warn('Costing sheet save warning:', err);
                    // Non-blocking - continue to save offer even if costing sheet fails
                    costingHistorySaveFailed = true;
                }
            }

            // Save as offer
            const data = buildExportData();
            const offer = await SaveCostingAsOffer(data as any);
            if (offer) {
                const actionText = sourceOfferId ? 'updated' : 'created';
                toast.success(`Offer ${offer.offer_number} ${actionText}! View it in Sales Hub - Offers`);
                if (costingHistorySaveFailed) {
                    toast.warning('Offer saved, but the costing history could not be saved.');
                }

                // Refresh recent sheets list
                costingSheets = await GetCostingSheets(20).catch(() => []);

                showCostingForm = false;
                selectedOpportunityId = '';
                selectedOpportunity = null;
                selectedRevision = null;
                currentCostingId = null;
                sourceOfferId = '';
                sourceOfferNumber = '';
                isDirty = false;
                clearCostingDraft();
                dispatch('navigate', { screen: 'opportunities', tab: 'offers' });
            }
        } catch (err) {
            console.error('Save as offer failed:', err);
            toast.danger('Failed to save offer: ' + String(err));
        } finally {
            savingOffer = false;
        }
    }

    onMount(() => {
        window.addEventListener('beforeunload', handleBeforeUnload);
        if (!canView) {
            loading = false;
        }
    });

    onDestroy(() => {
        window.removeEventListener('beforeunload', handleBeforeUnload);
    });

    run(() => {
        if (canView && active && !hasLoadedInitialData) {
            hasLoadedInitialData = true;
            void loadData();
        }
    });
</script>

<div class="costing-page" class:embedded>
    <header class="page-header">
        <div>
            <h1>Costing Sheet</h1>
            <p class="subtitle">Acme Instrumentation W.L.L. - Pricing Calculator</p>
        </div>
        <div class="header-actions">
            <Button variant="secondary" on:click={loadData}>Refresh</Button>
            {#if showCostingForm}
                <Button variant="secondary" on:click={handleExportExcel} disabled={exporting}>
                    {exporting ? 'Exporting...' : 'Export Excel'}
                </Button>
                <Button variant="secondary" on:click={handleExportPDF} disabled={exporting}>
                    {exporting ? 'Exporting...' : 'Export PDF'}
                </Button>
                {#if isRFQOpportunity(selectedOpportunity)}
                    <Button
                        variant="secondary"
                        on:click={handleCreateNewRevision}
                        disabled={savingRevision}
                        title="Save this costing as a revision, without creating an offer"
                    >
                        {savingRevision ? 'Saving...' : 'Save Costing'}
                    </Button>
                {/if}
                <Button variant="primary" on:click={handleSaveAsOffer} disabled={savingOffer}>
                    {savingOffer ? 'Saving...' : 'Save as Offer'}
                </Button>
            {/if}
        </div>
    </header>

    {#if loading}
        <div class="loading-state">
            <WabiSpinner size="lg" />
            <p>Loading data...</p>
        </div>
    {:else}
        <!-- Opportunity Selection Section -->
        {#if !showCostingForm}
            <Card title="Select Opportunity" variant="elevated">
                <div class="opportunity-selector">
                    <p class="selector-description">
                        Select an opportunity to create a costing sheet. Customer details will be pre-filled automatically.
                    </p>
                    {#if hasStoredDraft}
                        <div class="draft-recovery-panel">
                            <div>
                                <span class="field-label">Saved Draft</span>
                                <p>Unsaved costing draft saved on this device.</p>
                            </div>
                            <div class="draft-recovery-actions">
                                <Button variant="secondary" size="sm" on:click={resumeStoredCostingDraft}>
                                    Resume Draft
                                </Button>
                                <Button variant="danger" size="sm" on:click={discardStoredCostingDraft}>
                                    Discard
                                </Button>
                            </div>
                        </div>
                    {/if}
                    <div class="selector-grid">
                        <div class="form-group selector-dropdown">
                            <span class="field-label">Opportunity / RFQ</span>
                            <select bind:value={selectedOpportunityId} onchange={handleOpportunitySelect} class="input">
                                <option value="">-- Select an Opportunity --</option>
                                {#each opportunities as opp}
                                    <option value={String(opp.id)}>
                                        {getOpportunityDisplayReference(opp)} • {opp.client || opp.customer || 'Unknown'} - {opp.project || opp.project_name || 'New RFQ'} ({opp.status})
                                    </option>
                                {/each}
                            </select>
                        </div>
                        <div class="selector-divider">
                            <span>OR</span>
                        </div>
                        <Button variant="secondary" on:click={startNewCosting}>
                            Start Blank Costing
                        </Button>
                    </div>

                    <!-- Opportunities List Preview -->
                    {#if opportunities.length > 0}
                        <div class="opp-preview-list">
                            <h4>Recent Opportunities</h4>
                            {#each previewOpportunities as opp}
                                <button
                                    class="opp-preview-item"
                                    class:selected={selectedOpportunityId === String(opp.id)}
                                    onclick={() => { selectedOpportunityId = String(opp.id); handleOpportunitySelect(); }}
                                >
                                    <div class="opp-preview-header">
                                        <span class="opp-customer">{opp.client || opp.customer || 'Unknown'}</span>
                                        <span class="opp-status badge badge-{opp.status?.toLowerCase()}">{opp.status || 'New'}</span>
                                    </div>
                                    <div class="opp-preview-body">
                                        <span class="opp-project-ref">{getOpportunityDisplayReference(opp)}</span>
                                        <span class="opp-project">{opp.project || opp.project_name || 'New Requirements'}</span>
                                    </div>
                                    <div class="opp-preview-footer">
                                        <span class="opp-value" class:pending={getOpportunityValue(opp) <= 0}>{formatOpportunityValue(opp)}</span>
                                        <span class="opp-date">{opp.created_at ? new Date(opp.created_at).toLocaleDateString() : ''}</span>
                                    </div>
                                </button>
                            {/each}
                        </div>
                    {:else}
                        <div class="empty-opportunities">
                            <p>No opportunities found. Create an opportunity in the Sales Hub first, or start a blank costing.</p>
                        </div>
                    {/if}
                </div>
            </Card>
        {:else}
            <!-- Back Button and Selected Opportunity Info -->
            <div class="selected-opportunity-bar">
                <Button variant="ghost" on:click={handleBackToOpportunities}>
                    Back to Opportunities
                </Button>
                {#if selectedOpportunity}
                    <div class="selected-info">
                        <span class="label">Linked Opportunity:</span>
                        <span class="reference">{getOpportunityDisplayReference(selectedOpportunity)}</span>
                        <span class="customer">{selectedOpportunity.client || selectedOpportunity.customer}</span>
                        <span class="project">- {selectedOpportunity.project || selectedOpportunity.project_name}</span>
                        <span class="badge badge-{selectedOpportunity.status?.toLowerCase()}">{selectedOpportunity.status}</span>
                    </div>
                    <div class="opportunity-bar-actions">
                        <Button variant="secondary" size="sm" on:click={disconnectFromOpportunity}>
                            Disconnect from Opportunity
                        </Button>
                        <Button variant="ghost" size="sm" on:click={requestStartFresh}>
                            Start Fresh
                        </Button>
                    </div>
                {:else}
                    <div class="selected-info">
                        <span class="label">Blank Costing</span>
                        <span class="hint">(not linked to an opportunity)</span>
                    </div>
                    <div class="opportunity-bar-actions">
                        <Button variant="ghost" size="sm" on:click={requestStartFresh}>
                            Start Fresh
                        </Button>
                    </div>
                {/if}
            </div>

            {#if showStartFreshConfirm}
                <div
                    class="modal-backdrop costing-backdrop"
                    role="button"
                    tabindex="0"
                    onclick={self(() => (showStartFreshConfirm = false))}
                    onkeydown={(event) => {
                        if (event.currentTarget !== event.target) return;
                        if (event.key === 'Escape') {
                            event.preventDefault();
                            showStartFreshConfirm = false;
                        }
                    }}
                >
                    <section class="confirm-card" role="dialog" aria-modal="true" aria-labelledby="costing-start-fresh-title">
                        <span class="confirm-kicker">Start a Blank Costing?</span>
                        <h2 id="costing-start-fresh-title">Discard this costing and start fresh?</h2>
                        <p>
                            This clears the header, all line items and any linked opportunity to begin a blank costing. Unsaved changes here will be lost.
                        </p>
                        <div class="confirm-actions">
                            <Button variant="secondary" on:click={() => (showStartFreshConfirm = false)}>
                                Stay on Sheet
                            </Button>
                            <Button variant="warning" on:click={confirmStartFresh}>
                                Discard &amp; Start Fresh
                            </Button>
                        </div>
                    </section>
                </div>
            {/if}

            {#if showBackWarning}
                <div
                    class="modal-backdrop costing-backdrop"
                    role="button"
                    tabindex="0"
                    onclick={self(() => (showBackWarning = false))}
                    onkeydown={(event) => {
                        if (event.currentTarget !== event.target) return;
                        if (event.key === 'Escape') {
                            event.preventDefault();
                            showBackWarning = false;
                        }
                    }}
                >
                    <section class="confirm-card" role="dialog" aria-modal="true" aria-labelledby="costing-back-title">
                        <span class="confirm-kicker">Unsaved Changes</span>
                        <h2 id="costing-back-title">Leave this costing sheet?</h2>
                        <p>
                            Your changes have not been saved as an offer yet. A local draft will be kept on this device so it can be resumed from the Costing page.
                        </p>
                        <div class="confirm-actions">
                            <Button variant="secondary" on:click={() => (showBackWarning = false)}>
                                Stay on Sheet
                            </Button>
                            <Button variant="warning" on:click={returnToCostingList}>
                                Keep Draft & Go Back
                            </Button>
                        </div>
                    </section>
                </div>
            {/if}

            <!-- Revision Selector (Feature D - only show if RFQ selected) -->
            {#if isRFQOpportunity(selectedOpportunity) && rfqCostings.length > 0}
                <Card title="Costing Revisions">
                    <div class="revision-list">
                        {#each rfqCostings as rev}
                            <div
                                class="revision-item"
                                class:active={rev.is_active}
                                class:selected={selectedRevision?.id === rev.id}
                                onclick={() => selectRevision(rev)}
                                onkeydown={(event) => (event.key === 'Enter' || event.key === ' ') && selectRevision(rev)}
                                role="button"
                                tabindex="0"
                            >
                                <div class="rev-header">
                                    <span class="rev-number">Rev {rev.revision_number}</span>
                                    {#if rev.is_active}
                                        <span class="badge badge-success">Current</span>
                                    {/if}
                                    <span class="rev-status badge">{rev.status}</span>
                                </div>
                                <div class="rev-details">
                                    <span>{formatDate(rev.created_at)}</span>
                                    <span>by {rev.created_by}</span>
                                    <span>{formatBHD(rev.final_price || 0)}</span>
                                </div>
                            </div>
                        {/each}
                    </div>

                    <div class="revision-actions">
                        <Button
                            variant="secondary"
                            size="sm"
                            on:click={handleCreateNewRevision}
                            title="Create a new revision based on the current one"
                        >
                            + New Revision
                        </Button>

                        {#if selectedRevision && !selectedRevision.is_active}
                            <Button
                                variant="primary"
                                size="sm"
                                on:click={() => handleSetActiveRevision(selectedRevision.id)}
                            >
                                Make Current
                            </Button>
                        {/if}
                    </div>
                </Card>
            {/if}

        <div class="costing-layout" oninput={markDirty} onchange={markDirty}>
            <!-- Left: Costing Form -->
            <div class="main-form">
                <!-- Header Section -->
                <Card>
                    <!-- Row 1: Core fields always visible -->
                    <div class="header-row-3col tight">
                        <div class="field-group">
                            <span class="field-label">Customer *</span>
                            <select bind:value={header.customerId} onchange={handleCustomerChange} class="input">
                                <option value="">Select customer...</option>
                                {#each customers as c}
                                    <option value={c.id}>{c.business_name}</option>
                                {/each}
                            </select>
                        </div>
                        <div class="field-group">
                            <span class="field-label">Contact Person</span>
                            <input type="text" bind:value={header.contactPerson} class="input" placeholder="Name" />
                        </div>
                        <div class="field-group">
                            <span class="field-label">RFQ Reference</span>
                            <input type="text" bind:value={header.rfqReference} class="input" placeholder="RFQ / Enquiry ref" />
                        </div>
                    </div>

                    <!-- Row 2: Document + commercial -->
                    <div class="header-row-6col tight">
                        <div class="field-group">
                            <span class="field-label">Division</span>
                            <select bind:value={header.division} class="input-sm">
                                {#each divisionOptions as div}<option value={div}>{div}</option>{/each}
                            </select>
                        </div>
                        <div class="field-group">
                            <span class="field-label">Date</span>
                            <input type="date" bind:value={header.date} class="input-sm" />
                        </div>
                        <div class="field-group">
                            <span class="field-label">Prepared By</span>
                            <select bind:value={header.preparedBy} class="input-sm">
                                <option value="">Select...</option>
                                {#each preparedByOptions as name}<option value={name}>{name}</option>{/each}
                            </select>
                        </div>
                        <div class="field-group">
                            <span class="field-label">Doc Type</span>
                            <select bind:value={header.quoteType} class="input-sm">
                                <option value="Quotation">Quotation</option>
                                <option value="Budgetary Quote">Budgetary Quote</option>
                                <option value="Budgetary Estimate">Budgetary Est.</option>
                                <option value="Technical Offer">Technical</option>
                                <option value="Commercial Offer">Commercial</option>
                            </select>
                        </div>
                        <div class="field-group">
                            <span class="field-label">Folder No.</span>
                            <input type="text" bind:value={header.folderNumber} class="input-sm" placeholder="Enter folder no." />
                        </div>
                        <div class="field-group">
                            <span class="field-label">Costing ID</span>
                            <input type="text" bind:value={header.costingId} class="input-sm" placeholder="Enter costing ID" />
                        </div>
                    </div>

                    <!-- Row 3: Terms (always visible - needed for every quote) -->
                    <div class="header-row-3col tight">
                        <div class="field-group">
                            <span class="field-label">Payment Terms</span>
                            <select bind:value={header.paymentTerms} class="input-sm">
                                {#each paymentTermsOptions as opt}<option value={opt}>{opt}</option>{/each}
                            </select>
                        </div>
                        <div class="field-group">
                            <span class="field-label">Delivery Terms</span>
                            <select bind:value={header.deliveryTerms} class="input-sm">
                                {#each deliveryTermsOptions as opt}<option value={opt}>{opt}</option>{/each}
                            </select>
                        </div>
                        <div class="field-group">
                            <span class="field-label">Est. Delivery</span>
                            <select bind:value={header.estDelivery} class="input-sm">
                                {#each deliveryOptions as opt}<option value={opt}>{opt}</option>{/each}
                            </select>
                        </div>
                    </div>
                    <div class="header-row-3col tight">
                        <div class="field-group full-span">
                            <span class="field-label">Subject</span>
                            <input type="text" bind:value={header.subject} class="input-sm" placeholder="Subject line for customer PDF" />
                        </div>
                    </div>
                    <div class="header-row-3col tight">
                        <div class="field-group full-span">
                            <span class="field-label">PDF Body</span>
                            <textarea bind:value={quotationBody} class="input-sm textarea-sm" rows="4" placeholder="Opening body / cover note shown before the line items"></textarea>
                        </div>
                    </div>

                    <!-- Expandable: Compliance & certificates -->
                    <button class="expand-toggle" onclick={() => showAdvancedHeader = !showAdvancedHeader}>
                        {showAdvancedHeader ? '▾' : '▸'} Compliance, Certificates & VAT
                    </button>

                    {#if showAdvancedHeader}
                    <div class="advanced-section">
                        <div class="header-row-6col tight">
                            <div class="field-group">
                                <span class="field-label">Order Type</span>
                                <select bind:value={header.orderType} class="input-sm">
                                    {#each orderTypeOptions as opt}<option value={opt}>{opt}</option>{/each}
                                </select>
                            </div>
                            <div class="field-group">
                                <span class="field-label">Origin</span>
                                <select bind:value={header.countryOfOrigin} class="input-sm">
                                    {#each countryOptions as opt}<option value={opt.code}>{opt.code}</option>{/each}
                                </select>
                            </div>
                            <div class="field-group">
                                <span class="field-label">COC/COO</span>
                                <select bind:value={header.cocCoo} class="input-sm">
                                    {#each certificateOptions as opt}<option value={opt}>{opt}</option>{/each}
                                </select>
                            </div>
                            <div class="field-group">
                                <span class="field-label">Test Cert</span>
                                <select bind:value={header.testCertificate} class="input-sm">
                                    {#each certificateOptions as opt}<option value={opt}>{opt}</option>{/each}
                                </select>
                            </div>
                            <div class="field-group">
                                <span class="field-label">Install</span>
                                <select bind:value={header.installation} class="input-sm">
                                    <option value="Yes">Yes</option><option value="No">No</option>
                                </select>
                            </div>
                            <div class="field-group">
                                <span class="field-label">Commission</span>
                                <select bind:value={header.commissioning} class="input-sm">
                                    <option value="Yes">Yes</option><option value="No">No</option>
                                </select>
                            </div>
                        </div>
                        <div class="header-row-3col tight">
                            <div class="field-group">
                                <span class="field-label">Place of Supply</span>
                                <select bind:value={header.placeOfSupply} class="input-sm">
                                    <option value="Kingdom of Bahrain">Kingdom of Bahrain</option>
                                    <option value="GCC">GCC Member State</option>
                                    <option value="Export">Export (Outside GCC)</option>
                                </select>
                            </div>
                            <div class="field-group">
                                <span class="field-label">Tax Category</span>
                                <select bind:value={header.taxCategory} class="input-sm">
                                    <option value="Standard">Standard Rate (10%)</option>
                                    <option value="Zero-rated">Zero-rated</option>
                                    <option value="Exempt">Exempt</option>
                                    <option value="Out-of-scope">Out of Scope</option>
                                </select>
                            </div>
                            <div class="field-group">
                                <span class="field-label">Customer TRN</span>
                                <input type="text" bind:value={header.customerTRN} class="input-sm" placeholder="Tax Reg. Number" />
                            </div>
                        </div>
                    </div>
                    {/if}
                </Card>

                <!-- Line Items Section (Wave 9.5 B4: extracted to the canonical
                     LineItemsEditor — this screen keeps ALL calculation and
                     just passes items + calc callbacks; the component is
                     presentation only). -->
                <LineItemsEditor
                    mode="costing"
                    items={lineItems}
                    {currencyOptions}
                    maxItems={MAX_LINE_ITEMS}
                    {formatBHD}
                    {formatPercent}
                    {formatNumber}
                    onRecalculate={calculateAll}
                    onCurrencyChange={handleCurrencyChange}
                    onUserPriceInput={handleUserPriceInput}
                    onFreightPercentInput={() => { isDirty = true; }}
                    onRemoveItem={removeLineItem}
                    onCopyFirstToAll={copyFirstItemCostsToAll}
                    onAddItem={addLineItem}
                />
            </div>

            <!-- Right: Summary Panel -->
            <div class="summary-panel">
                <Card title="Summary" variant="elevated">
                    <div class="summary-rows">
                        <div class="summary-row">
                            <span>Subtotal</span>
                            <span class="money">{formatBHD(subtotal)}</span>
                        </div>
                        <div class="summary-row">
                            <span>Discount</span>
                            <input type="number" bind:value={discount} class="input-sm money" step="0.001" min="0" oninput={() => isDirty = true} onchange={calculateAll} />
                        </div>
                        <div class="summary-row">
                            <span title="Additional charges not shown on quotation">Hidden Charges</span>
                            <input type="number" bind:value={hiddenCharges} class="input-sm money" step="0.001" min="0" oninput={() => isDirty = true} />
                        </div>
                        <div class="summary-row vat-row">
                            <div class="vat-control">
                                <span>VAT</span>
                                <input type="number" bind:value={vatRate} class="vat-input" min="0" max="100" step="0.5" oninput={() => isDirty = true} />
                                <span>%</span>
                            </div>
                            <span class="money">{formatBHD(vat)}</span>
                        </div>
                        <div class="summary-row total">
                            <span>Grand Total</span>
                            <span class="money">{formatBHD(grandTotal)}</span>
                        </div>
                    </div>

                    <div class="profit-section">
                        <h4>Profit Analysis</h4>
                        <div class="profit-grid">
                            <div class="profit-item">
                                <span class="label">PO Expected</span>
                                <span class="value">{formatBHD(netAmount)}</span>
                            </div>
                            <div class="profit-item">
                                <span class="label">PH Cost</span>
                                <span class="value">{formatBHD(totalCost)}</span>
                            </div>
                            <div class="profit-item muted">
                                <span class="label">Internal Hidden Charges</span>
                                <span class="value">{formatBHD(effectiveHiddenCharges)}</span>
                            </div>
                            <div class="profit-item highlight">
                                <span class="label">Profit</span>
                                <span class="value">{formatBHD(profit)}</span>
                            </div>
                            <div class="profit-item highlight">
                                <span class="label">Profit %</span>
                                <span class="value">{formatPercent(profitPercent)}</span>
                            </div>
                        </div>
                    </div>
                </Card>

                <!-- Recent Sheets -->
                <Card title="Recent Sheets">
                    <div class="recent-list">
                        {#if costingSheets.length === 0}
                            <p class="empty">No recent costing sheets</p>
                        {:else}
                            {#each costingSheets.slice(0, 5) as sheet}
                                <div class="recent-item">
                                    <span class="ref">{sheet.ReferenceNo || 'N/A'}</span>
                                    <span class="customer">{sheet.CustomerName || '-'}</span>
                                    <span class="amount">{formatBHD(sheet.TotalSellBHD || 0)}</span>
                                </div>
                            {/each}
                        {/if}
                    </div>
                </Card>

                <!-- Terms & Conditions -->
                <Card title="Terms & Conditions">
                    <div class="tc-section">
                        <p class="tc-note">Printed on a separate page in PDF exports</p>
                        <textarea
                            class="tc-textarea"
                            bind:value={termsAndConditions}
                            rows="8"
                            placeholder="Enter terms and conditions..."
                        ></textarea>
                    </div>
                </Card>
            </div>
        </div>
        {/if}
    {/if}
</div>

<style>
    .costing-page {
        padding: var(--page-padding);
        height: 100%;
        display: flex;
        flex-direction: column;
        background: var(--bg-base);
        overflow-y: auto;
    }

    .costing-page.embedded {
        padding: 0;
    }

    .page-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: var(--spacing-lg);
        padding-bottom: var(--spacing-md);
        border-bottom: 1px solid var(--border);
    }

    .page-header h1 {
        font-size: var(--page-title-size);
        font-weight: var(--page-title-weight);
        margin: 0;
    }

    .subtitle {
        color: var(--text-secondary);
        font-size: var(--label-size);
        margin: 4px 0 0;
    }

    .header-actions {
        display: flex;
        gap: var(--spacing-sm);
    }

    .loading-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        height: 300px;
        gap: var(--spacing-md);
        color: var(--text-secondary);
    }

    .costing-layout {
        display: grid;
        grid-template-columns: 1fr 350px;
        gap: var(--spacing-lg);
        flex: 1;
        min-height: 0;
    }

    .main-form {
        display: flex;
        flex-direction: column;
        gap: var(--spacing-md);
        overflow-y: auto;
    }

    .header-grid {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 12px 16px;
        margin-bottom: 16px;
    }

    .header-grid.compact {
        grid-template-columns: repeat(3, 1fr);
        gap: 8px 12px;
        margin-bottom: 12px;
    }

    .header-row-3col {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 6px 14px;
        margin-bottom: 6px;
    }

    .header-row-3col.tight {
        margin-bottom: 4px;
    }

    .header-row-6col {
        display: grid;
        grid-template-columns: repeat(6, 1fr);
        gap: 6px 10px;
        margin-bottom: 4px;
    }

    .header-row-6col.tight {
        margin-bottom: 4px;
    }

    .field-group {
        display: flex;
        flex-direction: column;
        gap: 1px;
    }

    .field-group.full-span {
        grid-column: 1 / -1;
    }

    .field-group .field-label {
        font-size: 10px;
        font-weight: 600;
        color: var(--text-secondary, #6b7280);
        text-transform: uppercase;
        letter-spacing: 0.3px;
    }

    .field-group .input,
    .field-group .input-sm {
        font-size: 12px;
        padding: 4px 6px;
        border: 1px solid var(--border-subtle, #e5e7eb);
        border-radius: 4px;
        background: var(--surface-base, #fff);
        height: 28px;
    }

    .field-group .input:focus,
    .field-group .input-sm:focus {
        border-color: var(--accent-primary, #6366f1);
        outline: none;
        box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.1);
    }

    .field-group .textarea-sm {
        min-height: 96px;
        height: auto;
        resize: vertical;
        padding-top: 8px;
        line-height: 1.4;
    }

    .expand-toggle {
        background: none;
        border: none;
        color: var(--text-secondary, #6b7280);
        font-size: 11px;
        cursor: pointer;
        padding: 4px 0;
        margin-top: 2px;
        text-align: left;
    }

    .expand-toggle:hover {
        color: var(--accent-primary, #6366f1);
    }

    .advanced-section {
        padding-top: 6px;
        border-top: 1px solid var(--border-subtle, #e5e7eb);
        margin-top: 4px;
    }

    .form-group {
        display: flex;
        flex-direction: column;
        gap: 4px;
    }

    .form-group .field-label {
        font-size: 11px;
        color: var(--text-secondary);
        text-transform: uppercase;
        letter-spacing: 0.05em;
    }

    .input {
        padding: 8px 12px;
        border: 1px solid var(--border);
        border-radius: var(--border-radius-sm);
        font-size: 13px;
        background: var(--bg-base);
    }

    .input:focus {
        outline: none;
        border-color: var(--brand-indigo);
        box-shadow: 0 0 0 2px var(--brand-indigo-tint);
    }

    .input.readonly {
        background: var(--surface-elevated);
        color: var(--text-secondary);
    }

    .input-sm {
        padding: 6px 8px;
        border: 1px solid var(--border);
        border-radius: var(--border-radius-sm);
        font-size: 12px;
        background: var(--bg-base);
        width: 100%;
        min-width: 0;
        box-sizing: border-box;
        min-height: 32px;
        height: 32px;
        line-height: 1.2;
    }

    /* Line Items */
    /* Wave 9.5 B4: .input-xs / .input-sm.qty / .input-sm.money /
       .input-sm.sell-price(+focus) now live in LineItemsEditor.svelte — the
       line-item row markup that used them moved there. .input-sm (base,
       above) stays here too since the header rows still use it. */
    .line-items-header {
        display: grid;
        /* Sl No | Equipment | Model | Qty | Currency | Unit Price | Freight | Extra Cost | Manual Unit Price | Total (BHD) | Remove */
        grid-template-columns: 35px 2fr 120px 45px 60px 80px 70px 70px 85px 90px 28px;
        gap: 8px;
        padding: 8px 10px;
        font-size: 10px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.03em;
        color: var(--text-secondary);
        border-bottom: 2px solid var(--border);
        margin-bottom: 8px;
        align-items: end;
    }

    /* Suggested price display */
    .suggested-price {
        background: var(--ether, #F5F5F7);
        padding: 6px 8px;
        border-radius: var(--border-radius-sm);
        font-weight: 500;
        color: var(--steel, #86868B);
    }

    /* Margin input styling */
    .input-sm.margin {
        text-align: center;
        font-weight: 500;
    }

    /* Wave 9.5 B4: .calc-value(+highlight) / .btn-remove(+hover) /
       .remove-slot / .cost-breakdown / .cost-row / .cost-label(+suggested-
       highlight) / .cost-input(+field-label) / .btn-copy-costs(+hover) /
       .add-item-row / .item-count now live in LineItemsEditor.svelte
       alongside the markup that used them. */

    /* Summary Panel */
    .summary-panel {
        display: flex;
        flex-direction: column;
        gap: var(--spacing-md);
    }

    .summary-rows {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .summary-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        font-size: 14px;
    }

    .summary-row.total {
        padding-top: 12px;
        border-top: 2px solid #059669;
        font-weight: 700;
        font-size: 18px;
        color: #059669;
    }

    .summary-row .money {
        font-family: var(--font-mono, 'JetBrains Mono', monospace);
        font-weight: 500;
    }

    .profit-section {
        margin-top: 20px;
        padding-top: 16px;
        border-top: 1px solid var(--border);
    }

    .profit-section h4 {
        font-size: 12px;
        text-transform: uppercase;
        color: var(--text-secondary);
        margin: 0 0 12px;
    }

    .profit-grid {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 12px;
    }

    .profit-item {
        padding: 12px;
        background: var(--surface-elevated);
        border-radius: var(--border-radius-sm);
    }

    .profit-item.highlight {
        background: var(--brand-indigo-tint);
    }

    .profit-item.muted {
        background: rgba(15, 23, 42, 0.04);
    }

    .profit-item .label {
        display: block;
        font-size: 10px;
        color: var(--text-secondary);
        text-transform: uppercase;
        margin-bottom: 4px;
    }

    .profit-item .value {
        font-size: 16px;
        font-weight: 600;
        font-family: var(--font-mono, 'JetBrains Mono', monospace);
    }

    .profit-item.highlight .value {
        color: var(--brand-indigo);
    }

    /* Recent Sheets */
    .recent-list {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .recent-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 8px;
        background: var(--surface-elevated);
        border-radius: var(--border-radius-sm);
        font-size: 12px;
    }

    .recent-item .ref {
        font-family: var(--font-mono, 'JetBrains Mono', monospace);
        font-weight: 500;
    }

    .recent-item .customer {
        color: var(--text-secondary);
        flex: 1;
        margin: 0 8px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .recent-item .amount {
        font-family: var(--font-mono, 'JetBrains Mono', monospace);
        font-weight: 600;
    }

    .empty {
        color: var(--text-muted);
        font-style: italic;
        font-size: 13px;
        text-align: center;
        padding: 16px;
    }

    /* Opportunity Selector Styles */
    .opportunity-selector {
        padding: 16px 0;
    }

    .selector-description {
        color: var(--text-secondary);
        font-size: 14px;
        margin: 0 0 20px;
    }

    .draft-recovery-panel {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 16px;
        padding: 14px 16px;
        margin-bottom: 20px;
        border: 1px solid var(--border);
        border-left: 3px solid var(--brand-indigo);
        border-radius: var(--border-radius-sm);
        background: var(--surface-elevated);
    }

    .draft-recovery-panel p {
        margin: 4px 0 0;
        color: var(--text-secondary);
        font-size: 13px;
        line-height: 1.4;
    }

    .draft-recovery-actions {
        display: flex;
        align-items: center;
        gap: 8px;
        flex-shrink: 0;
    }

    .selector-grid {
        display: flex;
        align-items: flex-end;
        gap: 16px;
        margin-bottom: 24px;
    }

    .selector-dropdown {
        flex: 1;
        max-width: 500px;
    }

    .selector-divider {
        display: flex;
        align-items: center;
        padding: 0 8px 8px;
        color: var(--text-muted);
        font-size: 12px;
        text-transform: uppercase;
    }

    .opp-preview-list {
        margin-top: 24px;
        padding-top: 20px;
        border-top: 1px solid var(--border);
    }

    .opp-preview-list h4 {
        font-size: 12px;
        text-transform: uppercase;
        color: var(--text-secondary);
        letter-spacing: 0.05em;
        margin: 0 0 12px;
    }

    .opp-preview-item {
        display: block;
        width: 100%;
        background: var(--surface);
        border: 1px solid var(--border);
        border-radius: var(--border-radius);
        padding: 12px 16px;
        margin-bottom: 8px;
        text-align: left;
        cursor: pointer;
        transition: all var(--transition-fast);
    }

    .opp-preview-item:hover {
        border-color: var(--text-muted);
        box-shadow: var(--shadow-sm);
    }

    .opp-preview-item.selected {
        border-color: var(--brand-indigo);
        background: var(--brand-indigo-tint);
    }

    .opp-preview-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 4px;
    }

    .opp-customer {
        font-weight: 600;
        font-size: 14px;
        color: var(--text-primary);
    }

    .opp-status {
        font-size: 10px;
        padding: 2px 8px;
        border-radius: 10px;
        text-transform: uppercase;
    }

    .badge-new { background: var(--brand-indigo-tint); color: var(--brand-indigo); }
    .badge-quoted { background: rgba(245, 158, 11, 0.1); color: #d97706; }
    .badge-won { background: rgba(34, 197, 94, 0.1); color: #16a34a; }
    .badge-lost { background: rgba(239, 68, 68, 0.1); color: #ef4444; }

    .opp-preview-body {
        margin-bottom: 8px;
    }

    .opp-project {
        font-size: 13px;
        color: var(--text-secondary);
    }

    .opp-preview-footer {
        display: flex;
        justify-content: space-between;
        align-items: center;
    }

    .opp-value {
        font-family: var(--font-mono, 'JetBrains Mono', monospace);
        font-size: 13px;
        font-weight: 600;
        color: var(--text-primary);
    }

    .opp-value.pending {
        color: var(--text-muted);
        font-weight: 500;
    }

    .opp-date {
        font-size: 11px;
        color: var(--text-muted);
    }

    .empty-opportunities {
        padding: 32px;
        text-align: center;
        color: var(--text-secondary);
        background: var(--surface-elevated);
        border-radius: var(--border-radius);
        margin-top: 16px;
    }

    .empty-opportunities p {
        margin: 0;
        font-size: 14px;
    }

    /* Selected Opportunity Bar */
    .opportunity-bar-actions {
        margin-left: auto;
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .selected-opportunity-bar {
        display: flex;
        align-items: center;
        gap: 16px;
        padding: 12px 16px;
        background: var(--surface-elevated);
        border: 1px solid var(--border);
        border-radius: var(--border-radius);
        margin-bottom: 16px;
    }

    .selected-info {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 13px;
    }

    .selected-info .label {
        color: var(--text-muted);
    }

    .selected-info .customer {
        font-weight: 600;
        color: var(--text-primary);
    }

    .selected-info .project {
        color: var(--text-secondary);
    }

    .selected-info .hint {
        color: var(--text-muted);
        font-style: italic;
    }

    .costing-backdrop {
        position: fixed;
        inset: 0;
        z-index: 1200;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 24px;
        background: rgba(15, 23, 42, 0.36);
    }

    .confirm-card {
        width: min(440px, 100%);
        background: var(--surface);
        border: 1px solid var(--border);
        border-radius: var(--border-radius);
        box-shadow: var(--shadow-xl, 0 24px 60px rgba(15, 23, 42, 0.22));
        padding: 24px;
    }

    .confirm-kicker {
        display: block;
        color: #b45309;
        font-size: 11px;
        font-weight: 700;
        letter-spacing: 0.08em;
        text-transform: uppercase;
        margin-bottom: 8px;
    }

    .confirm-card h2 {
        margin: 0 0 10px;
        font-size: 20px;
        line-height: 1.25;
        color: var(--text-primary);
    }

    .confirm-card p {
        margin: 0;
        color: var(--text-secondary);
        font-size: 14px;
        line-height: 1.6;
    }

    .confirm-actions {
        display: flex;
        justify-content: flex-end;
        gap: 10px;
        margin-top: 22px;
    }

    @media (max-width: 1200px) {
        .costing-layout {
            grid-template-columns: 1fr;
        }

        .summary-panel {
            order: -1;
        }

        .header-grid {
            grid-template-columns: repeat(2, 1fr);
        }
    }

    /* Disable pop animations on costing sheet - scale transforms look buggy */
    .costing-page :global(.onyx-card:hover),
    .costing-page :global([class*="phi-card"]:hover),
    .costing-page :global(.card:hover) {
        transform: none !important;
    }

    /* FIX #23: Disable hover animation "dance" on costing inputs */
    /* Prevents headache-inducing box highlighting when hovering over header
       fields. The .cost-breakdown / .input-xs equivalents now live in
       LineItemsEditor.svelte alongside the elements they style. */
    .input-sm:hover,
    .input-sm:focus {
        transition: none !important;
        transform: none !important;
    }

    /* Remove any parent container hover effects that cause visual dance */
    .line-item-row:hover {
        transform: none !important;
        transition: none !important;
        border-left-color: var(--brand-indigo, #6366f1);
        background: transparent !important;
    }

    /* Terms & Conditions Section */
    .tc-section {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .tc-note {
        font-size: 11px;
        color: var(--text-muted);
        margin: 0;
    }

    .tc-textarea {
        width: 100%;
        min-height: 150px;
        padding: 12px;
        font-size: 12px;
        font-family: inherit;
        line-height: 1.5;
        border: 1px solid var(--border);
        border-radius: var(--border-radius-sm);
        background: var(--bg-base);
        color: var(--text-primary);
        resize: vertical;
    }

    .tc-textarea:focus {
        outline: none;
        border-color: var(--onyx);
    }

    /* VAT editable control */
    .vat-row .vat-control {
        display: flex;
        align-items: center;
        gap: 4px;
    }

    .vat-input {
        width: 50px;
        padding: 2px 6px;
        border: 1px solid var(--border);
        border-radius: var(--border-radius-sm);
        font-size: 13px;
        text-align: center;
        background: var(--bg-base);
    }

    .vat-input:focus {
        outline: none;
        border-color: var(--brand-indigo);
    }

    /* Global anti-flicker: disable ALL transitions and animations on the costing page */
    .costing-page * {
        animation: none !important;
    }

    .costing-page :global(*) {
        animation: none !important;
    }

    /* Revision Selector (Feature D) */
    .revision-list {
        display: flex;
        flex-direction: column;
        gap: 8px;
        margin-bottom: 16px;
    }

    .revision-item {
        padding: 12px;
        border-radius: 8px;
        background: var(--surface-elevated);
        cursor: pointer;
        transition: all 0.15s;
        border: 2px solid transparent;
    }

    .revision-item:hover {
        background: var(--surface-hover);
    }

    .revision-item.selected {
        border-color: var(--accent-primary);
        background: var(--accent-bg);
    }

    .revision-item.active {
        border-left: 4px solid var(--success);
    }

    .rev-header {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 4px;
    }

    .rev-number {
        font-weight: 600;
        font-size: 0.95rem;
    }

    .rev-details {
        display: flex;
        gap: 12px;
        font-size: 0.85rem;
        color: var(--text-muted);
    }

    .revision-actions {
        display: flex;
        gap: 12px;
        padding-top: 12px;
        border-top: 1px solid var(--border-color);
    }

    .badge {
        padding: 2px 8px;
        border-radius: 4px;
        font-size: 0.7rem;
        text-transform: uppercase;
    }

    .badge-success {
        background: rgba(46, 213, 115, 0.2);
        color: #2ed573;
    }
</style>
