<script lang="ts">
    import { run, self, preventDefault } from 'svelte/legacy';

    import { createEventDispatcher } from "svelte";
    import { fade, scale } from "svelte/transition";
    import { cubicOut } from "svelte/easing";
    import { motionMs } from "$lib/motion";
    import { getDefaultDivisionKey, getDivisionKeys, normalizeDivision } from "$lib/divisions.svelte";

    // Motion vocabulary (Wave 10 B2): timing mirrors design-tokens.css --motion-base (200ms).
    // Svelte transitions run in JS and cannot read CSS custom properties directly, so the
    // numeric value is hardcoded — keep it equal to --motion-base. cubicOut approximates
    // --ease-decelerate (fast start, slow settle, no overshoot).
    const MODAL_MOTION_MS = 200;
    import { toast } from "$lib/stores/toasts";
    import { confirm } from "$lib/stores/confirm";
    import WabiSpinner from "$lib/components/ui/WabiSpinner.svelte";
    import { formatNumber } from "$lib/utils/formatters";
    import { AnalyzeDocumentWithButler } from "../../../wailsjs/go/main/App";
import { CheckDuplicateOpportunity, CheckDuplicateRFQ, ListCustomers, ListSuppliers } from "../../../wailsjs/go/main/CRMService";
import { GetActiveBankAccounts } from "../../../wailsjs/go/main/FinanceService";

    interface Props {
        show?: boolean;
        processing?: boolean;
        ocrResult?: any;
        fileName?: string;
    }

    let {
        show = $bindable(false),
        processing = false,
        ocrResult = null,
        fileName = ""
    }: Props = $props();

    const dispatch = createEventDispatcher();

    // Butler AI analysis state
    let butlerAnalyzing = $state(false);
    let butlerInsights: any = $state(null);

    // Data initialization guard - prevents field reset on paste/edit
    let isDataInitialized = $state(false);

    // Dirty state tracking - prevents accidental data loss on outside click
    let isDirty = $state(false);
    let showCloseConfirmation = $state(false);

    // Customer/Supplier dropdown state - uses Wails-generated types (id is string from GORM)
    let customers: Array<{ id: string; customer_id: string; business_name: string; customer_type: string }> = $state([]);
    let suppliers: Array<{ id: string; supplier_code: string; supplier_name: string }> = $state([]);
    let bankAccounts: Array<{ id: string; division?: string; bank_name: string; account_name?: string; account_number: string; iban?: string; currency: string }> = $state([]);
    let loadingEntities = $state(false);
    let customerSearchTerm = $state("");
    let supplierSearchTerm = $state("");
    let showCustomerDropdown = $state(false);
    let showSupplierDropdown = $state(false);
    let selectedCustomerId: string | null = $state(null);
    let selectedSupplierId: string | null = $state(null);
    let selectedBankAccountId = $state("");
    let loadingBankAccounts = $state(false);


    // Editable document data (populated from OCR + Butler)
    let editableData = $state({
        division: getDefaultDivisionKey(),
        customer_name: "",
        supplier_name: "",
        project: "",
        invoice_number: "",
        po_number: "",
        dn_number: "",
        rfq_number: "",
        invoice_date: "",
        delivery_date: "",
        due_date: "",
        total: "",
        vat: "",
        currency: "BHD",
        notes: "",
        // Bank statement specific fields
        bank_name: "",
        account_number: "",
        opening_balance: "",
        closing_balance: "",
        period_start: "",
        period_end: "",
    });


    // Load customers and suppliers when modal opens
    async function loadEntities() {
        if (customers.length > 0 && suppliers.length > 0) return;
        loadingEntities = true;
        try {
            const [custResult, suppResult] = await Promise.all([
                ListCustomers(1000, 0),
                ListSuppliers(500, 0)
            ]);
            customers = (custResult || []).sort((a, b) => (a.business_name || '').localeCompare(b.business_name || ''));
            suppliers = (suppResult || []).sort((a, b) => (a.supplier_name || '').localeCompare(b.supplier_name || ''));
            console.log(`Loaded ${customers.length} customers, ${suppliers.length} suppliers`);
        } catch (err) {
            console.error('Failed to load entities:', err);
        } finally {
            loadingEntities = false;
        }
    }

    async function loadBankAccounts() {
        if (bankAccounts.length > 0 || loadingBankAccounts) return;
        loadingBankAccounts = true;
        try {
            const bankResult = await GetActiveBankAccounts();
            bankAccounts = (bankResult || []).sort((a, b) => `${a.bank_name || ''}${a.account_number || ''}`.localeCompare(`${b.bank_name || ''}${b.account_number || ''}`));
            syncDetectedBankAccountSelection();
        } catch (err) {
            bankAccounts = [];
            console.warn('Bank accounts unavailable for this user or device:', err);
        } finally {
            loadingBankAccounts = false;
        }
    }



    // Reset state on modal open to prevent state leak
    function resetStateOnOpen() {
        // Only reset if this is a fresh modal open (not just a reactive update)
        if (!isDataInitialized) {
            resetState();
        }
    }


    // Select customer from dropdown
    function selectCustomer(customer: typeof customers[0]) {
        editableData.customer_name = customer.business_name;
        selectedCustomerId = customer.id;
        customerSearchTerm = customer.business_name;
        showCustomerDropdown = false;
    }

    // Select supplier from dropdown
    function selectSupplier(supplier: typeof suppliers[0]) {
        editableData.supplier_name = supplier.supplier_name;
        selectedSupplierId = supplier.id;
        supplierSearchTerm = supplier.supplier_name;
        showSupplierDropdown = false;
    }

    // Handle customer input change
    function handleCustomerInput(e: Event) {
        const value = (e.target as HTMLInputElement).value;
        customerSearchTerm = value;
        editableData.customer_name = value;
        showCustomerDropdown = true;
        // Check if exact match exists
        const exactMatch = customers.find(c => c.business_name?.toLowerCase() === value.toLowerCase());
        selectedCustomerId = exactMatch?.id || null;
    }

    // Handle supplier input change
    function handleSupplierInput(e: Event) {
        const value = (e.target as HTMLInputElement).value;
        supplierSearchTerm = value;
        editableData.supplier_name = value;
        showSupplierDropdown = true;
        // Check if exact match exists
        const exactMatch = suppliers.find(s => s.supplier_name?.toLowerCase() === value.toLowerCase());
        selectedSupplierId = exactMatch?.id || null;
    }

    // Editable line items - flexible type for both RFQ items and bank transactions
    // RFQ: {description, quantity, unit, part_number, unit_price}
    // Bank: {date, description, reference, debit, credit, balance}
    let lineItems: Array<any> = $state([]);

    function parseNum(v: any): number {
        if (typeof v === "number") return Number.isFinite(v) ? v : 0;
        if (typeof v === "string") {
            const match = v
                .replace(/\b(BHD|USD|EUR|GBP|SAR|AED|CHF|KWD|OMR|QAR)\b/gi, "")
                .replace(/,/g, "")
                .match(/[-+]?\d*\.?\d+/);
            return match ? parseFloat(match[0]) || 0 : 0;
        }
        return 0;
    }

    function isCurrencyCode(value: any): boolean {
        return /^(BHD|USD|EUR|GBP|SAR|AED|CHF|KWD|OMR|QAR)$/i.test(String(value || "").trim());
    }

    function normalizeLineUnit(value: any): string {
        const unit = String(value || "").trim();
        if (!unit || isCurrencyCode(unit)) return "pcs";
        return unit;
    }

    function normalizeCommercialLineItem(item: any) {
        const rawUnit = item.unit || item.uom || item.unit_of_measure;
        const rawCurrency = item.currency || (isCurrencyCode(rawUnit) ? rawUnit : "");
        const description =
            item.description ||
            item.item_description ||
            item.item ||
            item.name ||
            item.equipment ||
            item.product ||
            item.product_name ||
            item.model ||
            item.part_number ||
            "";
        const partNumber =
            item.part_number ||
            item.part_no ||
            item.item_code ||
            item.product_code ||
            item.model ||
            item.long_code ||
            "";
        const unitPrice = parseNum(item.unit_price ?? item.unit_price_bhd ?? item.price ?? item.rate);
        const totalPrice = parseNum(item.total_price ?? item.total ?? item.line_total);
        const quantity = parseNum(item.quantity ?? item.qty) || (totalPrice > 0 && unitPrice > 0 ? totalPrice / unitPrice : 1);

        return {
            description: String(description || "").trim(),
            quantity,
            unit: normalizeLineUnit(rawUnit),
            part_number: String(partNumber || "").trim(),
            unit_price: unitPrice || (quantity > 0 && totalPrice > 0 ? totalPrice / quantity : 0),
            currency: isCurrencyCode(rawCurrency) ? String(rawCurrency).trim().toUpperCase() : undefined,
        };
    }

    // Duplicate RFQ detection state
    let duplicateRFQ: any = $state(null);
    let showDuplicateWarning = $state(false);
    let checkingDuplicate = $state(false);
    let duplicateType: "rfq" | "opportunity" | null = $state(null);

    // UI state
    let showRawText = false;
    let showLineItems = true;
    let activeTab: 'details' | 'items' | 'raw' = $state('details');

    // Document type options
    const documentTypes = [
        { id: "rfq", label: "RFQ / Inquiry", icon: "", screen: "/opportunities" },
        { id: "quotation", label: "Quotation", icon: "", screen: "/opportunities" },
        { id: "costing", label: "Costing Sheet", icon: "", screen: "/costing" },
        { id: "invoice", label: "Customer Invoice", icon: "", screen: "/finance" },
        { id: "supplier_invoice", label: "Supplier Invoice", icon: "", screen: "/finance" },
        { id: "purchase_order", label: "Purchase Order", icon: "", screen: "/operations" },
        { id: "delivery_note", label: "Delivery Note", icon: "", screen: "/operations" },
        { id: "bank_statement", label: "Bank Statement", icon: "", screen: "/finance" },
        { id: "contract", label: "Contract / Agreement", icon: "", screen: "/intelligence" },
        { id: "report", label: "Report / Summary", icon: "", screen: "/intelligence" },
        { id: "excel_data", label: "Excel Data", icon: "", screen: "/opportunities" },
    ];

    let selectedType = $state("rfq");



    // Auto-analyze with Butler when modal opens (debounced)
    let butlerAutoTriggered = $state(false);
    let hasStructuredBankTransactions = $state(false);

    // Map backend PascalCase types to frontend snake_case IDs
    function mapDocType(backendType: string): string {
        const typeMap: Record<string, string> = {
            'RFQ': 'rfq',
            'Invoice': 'invoice',
            'SupplierInvoice': 'supplier_invoice',
            'PurchaseOrder': 'purchase_order',
            'Quotation': 'quotation',
            'DeliveryNote': 'delivery_note',
            'BankStatement': 'bank_statement',
            'Contract': 'contract',
            'Report': 'report',
            'Other': 'rfq',
        };
        return typeMap[backendType] || backendType.toLowerCase();
    }


    function initializeEditableData() {
        const data = ocrResult?.extracted_data || {};

        // Populate editable fields from OCR
        editableData = {
            division: (typeof data.division === 'string' && data.division.trim() !== '') ? data.division : getDefaultDivisionKey(),
            customer_name: data.customer_name || data.company_name || "",
            supplier_name: data.supplier_name || "",
            project: data.project || data.subject || extractProjectFromFileName(fileName),
            invoice_number: data.invoice_number || "",
            po_number: data.po_number || "",
            dn_number: data.dn_number || "",
            rfq_number: data.rfq_number || "",
            invoice_date: data.invoice_date || new Date().toISOString().split('T')[0],
            delivery_date: data.delivery_date || "",
            due_date: data.due_date || "",
            total: data.total || "",
            vat: data.vat || "",
            currency: data.currency || "BHD",
            notes: data.notes || data.butler_summary || data.summary || "",
            // Bank statement fields
            bank_name: data.bank_name || "",
            account_number: data.account_number || data.iban || "",
            opening_balance: data.opening_balance || "",
            closing_balance: data.closing_balance || "",
            period_start: data.period_start || "",
            period_end: data.period_end || "",
        };

        // Fallback: parse bank statement fields from raw OCR text if backend extraction missed them
        const rawText = ocrResult?.text || '';
        const docType = ocrResult?.document_type || '';
        const isBankStatement = docType === 'BankStatement' || docType === 'bank_statement' ||
            rawText.toLowerCase().includes('bank statement') ||
            (rawText.toLowerCase().includes('opening balance') && rawText.toLowerCase().includes('closing balance'));

        if (isBankStatement && rawText) {
            if (!editableData.bank_name) {
                const bankMatch = rawText.match(/(?:National Bank of Bahrain[\s\w.]*|NBB[\s\w.]*|BBK[\s\w.]*|HSBC[\s\w.]*|Standard Chartered[\s\w.]*|Al Salam[\s\w.]*|Kuwait Finance[\s\w.]*|Ithmaar[\s\w.]*|Ahli United[\s\w.]*|Arab Banking[\s\w.]*)/i);
                if (bankMatch) editableData.bank_name = bankMatch[0].trim();
            }
            if (!editableData.account_number) {
                const acctMatch = rawText.match(/(?:account\s*(?:no|number|#|:))[:\s]*\n?\s*([0-9][0-9A-Z-]+)/i) ||
                                  rawText.match(/(?:a\/c\s*(?:no|number|#)?)[:\s]*([0-9][0-9-]+)/i);
                if (acctMatch) editableData.account_number = acctMatch[1].trim();
            }
            if (!editableData.account_number) {
                // Try IBAN as fallback for account number
                const ibanMatch = rawText.match(/IBAN[:\s]*([A-Z]{2}\d{2}[A-Z0-9]+)/i);
                if (ibanMatch) editableData.account_number = ibanMatch[1].trim();
            }
            if (!editableData.opening_balance) {
                const openMatch = rawText.match(/opening\s*balance[:\s]*\n?\s*([0-9][0-9,.]+)/i) ||
                                  rawText.match(/brought?\s*forward[:\s]*\n?\s*([0-9][0-9,.]+)/i);
                if (openMatch) editableData.opening_balance = openMatch[1].trim();
            }
            if (!editableData.closing_balance) {
                const closeMatch = rawText.match(/closing\s*balance[:\s]*\n?\s*([0-9][0-9,.]+)/i) ||
                                   rawText.match(/carried?\s*forward[:\s]*\n?\s*([0-9][0-9,.]+)/i);
                if (closeMatch) editableData.closing_balance = closeMatch[1].trim();
            }
            if (!editableData.period_start || !editableData.period_end) {
                const periodMatch = rawText.match(/(\d{1,2}[/-]\d{1,2}[/-]\d{2,4})\s*(?:to|through|-)\s*(\d{1,2}[/-]\d{1,2}[/-]\d{2,4})/i);
                if (periodMatch) {
                    if (!editableData.period_start) editableData.period_start = periodMatch[1].trim();
                    if (!editableData.period_end) editableData.period_end = periodMatch[2].trim();
                }
            }
            if (!editableData.currency) {
                if (rawText.includes('BHD') || rawText.includes('Bahraini Dinar')) editableData.currency = 'BHD';
                else if (rawText.includes('USD')) editableData.currency = 'USD';
                else if (rawText.includes('EUR')) editableData.currency = 'EUR';
            }
            console.log('[OCR] Bank statement fields parsed from raw text:', {
                bank_name: editableData.bank_name,
                account_number: editableData.account_number,
                opening_balance: editableData.opening_balance,
                closing_balance: editableData.closing_balance,
                period_start: editableData.period_start,
                period_end: editableData.period_end,
            });
        }

        // Set search terms for dropdowns
        customerSearchTerm = editableData.customer_name;
        supplierSearchTerm = editableData.supplier_name;

        // Try to match OCR-detected names with database entries
        if (editableData.customer_name && customers.length > 0) {
            const match = customers.find(c =>
                c.business_name?.toLowerCase().includes(editableData.customer_name.toLowerCase()) ||
                editableData.customer_name.toLowerCase().includes(c.business_name?.toLowerCase() || '')
            );
            if (match) {
                selectedCustomerId = match.id;
                editableData.customer_name = match.business_name;
                customerSearchTerm = match.business_name;
            }
        }
        if (editableData.supplier_name && suppliers.length > 0) {
            const match = suppliers.find(s =>
                s.supplier_name?.toLowerCase().includes(editableData.supplier_name.toLowerCase()) ||
                editableData.supplier_name.toLowerCase().includes(s.supplier_name?.toLowerCase() || '')
            );
            if (match) {
                selectedSupplierId = match.id;
                editableData.supplier_name = match.supplier_name;
                supplierSearchTerm = match.supplier_name;
            }
        }

        // Reset selection IDs
        if (!editableData.customer_name) selectedCustomerId = null;
        if (!editableData.supplier_name) selectedSupplierId = null;
        selectedBankAccountId = typeof data.bank_account_id === 'string' ? data.bank_account_id : "";

        // Initialize line items from OCR if available
        if (data.line_items && Array.isArray(data.line_items)) {
            if (isBankStatement) {
                lineItems = data.line_items.map((item: any) => ({
                    date: item.date || item.value_date || "",
                    description: item.description || "",
                    reference: item.reference || item.ref || "",
                    debit: parseFloat(String(item.debit ?? 0).replace(/,/g, '')) || 0,
                    credit: parseFloat(String(item.credit ?? 0).replace(/,/g, '')) || 0,
                    balance: parseFloat(String(item.balance ?? 0).replace(/,/g, '')) || 0
                }));
            } else {
                lineItems = data.line_items.map(normalizeCommercialLineItem);
            }
        } else {
            lineItems = [];
        }

        // For bank statements, try to parse transactions from raw text if no line items
        if (isBankStatement && lineItems.length === 0 && rawText) {
            const parsedTransactions = parseBankTransactionsFromText(rawText);
            if (parsedTransactions.length > 0) {
                lineItems = parsedTransactions;
                console.log(`[OCR] Parsed ${parsedTransactions.length} bank transactions from raw text`);
            }
        }

        syncDetectedBankAccountSelection();
    }

    function normalizeIdentifier(value: string): string {
        return (value || '').replace(/[^A-Za-z0-9]/g, '').toUpperCase();
    }

    function normalizeBankName(value: string): string {
        return (value || '')
            .toLowerCase()
            .replace(/\b(bank|bsc|b\.s\.c|wll|w\.l\.l|account|current|checking|call)\b/g, ' ')
            .replace(/[^a-z0-9]+/g, ' ')
            .trim();
    }

    function formatBankAccountOption(account: { bank_name?: string; account_name?: string; account_number?: string; currency?: string }): string {
        const bank = account?.bank_name || 'Bank Account';
        const accountName = account?.account_name ? ` ${account.account_name}` : '';
        const accountNumber = account?.account_number ? ` - ${account.account_number}` : '';
        const currency = account?.currency ? ` (${account.currency})` : '';
        return `${bank}${accountName}${accountNumber}${currency}`;
    }

    function findMatchingBankAccountId(): string {
        const accountNeedle = normalizeIdentifier(editableData.account_number);
        const bankNeedle = normalizeBankName(editableData.bank_name);
        let bestId = "";
        let bestScore = 0;
        let tied = false;

        for (const account of filteredBankAccounts) {
            const normalizedAccount = normalizeIdentifier(account.account_number);
            const normalizedIBAN = normalizeIdentifier(account.iban || '');
            const normalizedBank = normalizeBankName(account.bank_name);
            let score = 0;

            if (accountNeedle) {
                if (accountNeedle === normalizedAccount) {
                    score += 300;
                } else if (normalizedIBAN && accountNeedle === normalizedIBAN) {
                    score += 300;
                } else if (accountNeedle.length >= 6 && normalizedAccount.endsWith(accountNeedle)) {
                    score += 260;
                } else if (normalizedIBAN && accountNeedle.length >= 6 && normalizedIBAN.endsWith(accountNeedle)) {
                    score += 220;
                }
            }

            if (bankNeedle) {
                if (bankNeedle === normalizedBank) {
                    score += 90;
                } else if ((normalizedBank && normalizedBank.includes(bankNeedle)) || (bankNeedle && bankNeedle.includes(normalizedBank))) {
                    score += 45;
                }
            }

            if (score > bestScore) {
                bestScore = score;
                bestId = account.id;
                tied = false;
            } else if (score > 0 && score === bestScore) {
                tied = true;
            }
        }

        if (tied) return "";
        return bestScore > 0 ? bestId : "";
    }

    function syncDetectedBankAccountSelection() {
        if (selectedType !== 'bank_statement' || selectedBankAccountId || filteredBankAccounts.length === 0) {
            return;
        }

        const matchedAccountId = findMatchingBankAccountId();
        if (matchedAccountId) {
            selectedBankAccountId = matchedAccountId;
        }
    }

    function syncEditableBankFieldsFromSelection() {
        if (selectedType !== 'bank_statement' || !selectedBankAccountId || filteredBankAccounts.length === 0) {
            return;
        }

        const selectedAccount = filteredBankAccounts.find((account) => account.id === selectedBankAccountId);
        if (!selectedAccount) {
            return;
        }

        editableData.bank_name = selectedAccount.bank_name || editableData.bank_name;
        editableData.account_number = selectedAccount.account_number || editableData.account_number;
        if (selectedAccount.currency) {
            editableData.currency = selectedAccount.currency;
        }
    }



    function inferBankDirectionFromText(description: string, amount: number, balance: number, previousBalance: number, previousBalanceKnown: boolean) {
        if (!amount) {
            return { debit: 0, credit: 0 };
        }

        if (previousBalanceKnown && balance) {
            const delta = balance - previousBalance;
            if (Math.abs(Math.abs(delta) - amount) <= 0.051) {
                return delta >= 0
                    ? { debit: 0, credit: amount }
                    : { debit: amount, credit: 0 };
            }
        }

        const lower = (description || '').toLowerCase();
        const creditHints = [
            'credit',
            'transfer in',
            'deposit',
            'payment | from',
            'payment from',
            'fawri ordinary transfer',
            'sundry credit',
            'received from',
            'inward',
            'receipt'
        ];
        for (const hint of creditHints) {
            if (lower.includes(hint)) {
                return { debit: 0, credit: amount };
            }
        }

        const debitHints = [
            'debit(tf)',
            'debit',
            'payment to',
            'transfer to',
            'charges',
            'charge',
            'fee',
            'withdrawal',
            'fawri to'
        ];
        for (const hint of debitHints) {
            if (lower.includes(hint)) {
                return { debit: amount, credit: 0 };
            }
        }

        return { debit: amount, credit: 0 };
    }

    function scoreBankTransactionPolarity(transactions: any[]) {
        let mismatches = 0;
        let ambiguous = 0;
        let previousBalance = 0;
        let previousBalanceKnown = false;

        for (const transaction of transactions) {
            const debit = Number(transaction?.debit) || 0;
            const credit = Number(transaction?.credit) || 0;
            const balance = Number(transaction?.balance) || 0;

            if ((debit > 0 && credit > 0) || (debit === 0 && credit === 0 && balance !== 0)) {
                ambiguous += 1;
            }

            if (previousBalanceKnown && balance) {
                const expectedBalance = previousBalance + credit - debit;
                if (Math.abs(expectedBalance - balance) > 0.051) {
                    mismatches += 1;
                }
            }

            if (balance) {
                previousBalance = balance;
                previousBalanceKnown = true;
            }
        }

        return (mismatches * 100) + (ambiguous * 25);
    }

    function repairBankTransactionPolarity(transactions: any[]) {
        if (transactions.length < 2) {
            return transactions;
        }

        const swapped = transactions.map((transaction) => ({
            ...transaction,
            debit: Number(transaction?.credit) || 0,
            credit: Number(transaction?.debit) || 0,
        }));

        return scoreBankTransactionPolarity(swapped) + 1 < scoreBankTransactionPolarity(transactions)
            ? swapped
            : transactions;
    }

    // Parse bank transactions from raw OCR text
    function parseBankTransactionsFromText(text: string): any[] {
        const transactions: any[] = [];
        const lines = text.split('\n');
        let previousBalance = 0;
        let previousBalanceKnown = false;

        // Look for transaction lines: date pattern followed by description and amounts
        // Common formats:
        // "02/01/2026  Reference  Description  1,234.567"
        // "02-01-2026  PAYMENT  Some desc  500.000  70,058.251"
        const txnPattern = /^(\d{1,2}[/-]\d{1,2}[/-]\d{2,4})\s+(.+)/;

        for (const line of lines) {
            const match = line.trim().match(txnPattern);
            if (!match) continue;

            // Convert DD/MM/YYYY to YYYY-MM-DD for HTML date input
            const rawDate = match[1];
            const dm = rawDate.match(/^(\d{1,2})[/-](\d{1,2})[/-](\d{4})$/);
            const date = dm ? `${dm[3]}-${dm[2].padStart(2,'0')}-${dm[1].padStart(2,'0')}` : rawDate;
            const rest = match[2].trim();

            // Try to extract amounts from the end of the line
            // Numbers with commas and decimals (BHD uses 3 decimal places)
            const amounts = rest.match(/[\d,]+\.\d{2,3}/g) || [];
            if (amounts.length === 0) continue;

            // Extract description (everything before the first amount)
            const firstAmountIdx = rest.indexOf(amounts[0]);
            const descPart = rest.substring(0, firstAmountIdx).trim();

            // Extract reference if present (usually a short code before description)
            const parts = descPart.split(/\s{2,}/);
            const reference = parts.length > 1 ? parts[0] : '';
            const description = parts.length > 1 ? parts.slice(1).join(' ') : descPart;

            // Parse amounts - last is usually balance, before that debit or credit
            const parsedAmounts = amounts.map(a => parseFloat(a.replace(/,/g, '')));
            let debit = 0, credit = 0, balance = 0;

            if (parsedAmounts.length >= 3) {
                // Format: debit, credit, balance OR amount, balance, ...
                debit = parsedAmounts[0] || 0;
                credit = parsedAmounts[1] || 0;
                balance = parsedAmounts[parsedAmounts.length - 1];
            } else if (parsedAmounts.length === 2) {
                // Format: amount, balance
                const amount = parsedAmounts[0];
                balance = parsedAmounts[1];
                ({ debit, credit } = inferBankDirectionFromText(rest, amount, balance, previousBalance, previousBalanceKnown));
            } else if (parsedAmounts.length === 1) {
                balance = parsedAmounts[0];
            }

            transactions.push({
                date,
                reference,
                description: description || 'Transaction',
                debit: debit,
                credit: credit,
                balance: balance,
            });

            if (balance) {
                previousBalance = balance;
                previousBalanceKnown = true;
            }
        }

        return repairBankTransactionPolarity(transactions);
    }

    function extractProjectFromFileName(name: string): string {
        // Try to extract meaningful project name from filename
        const cleaned = name.replace(/\.[^/.]+$/, "") // Remove extension
            .replace(/[_-]/g, " ") // Replace separators with spaces
            .replace(/\d{4}-?\d{2}-?\d{2}/g, "") // Remove dates
            .trim();
        return cleaned || "New Project";
    }

    function detectDocumentType(text: string): string {
        const lower = text.toLowerCase();
        // Bank statement detection - check early as these have distinctive keywords
        if (lower.includes("bank statement") || lower.includes("account statement") ||
            lower.includes("opening balance") || lower.includes("closing balance") ||
            (lower.includes("credit") && lower.includes("debit") && lower.includes("balance"))) return "bank_statement";
        if (lower.includes("monthly report") || lower.includes("weekly report") ||
            lower.includes("management report") || lower.includes("executive summary") ||
            lower.includes("analysis report") || lower.includes("summary report") ||
            lower.includes("performance report")) return "report";
        if (lower.includes("terms and conditions") || lower.includes("service agreement") ||
            lower.includes("contract") || lower.includes("hereby agree")) return "contract";
        if (lower.includes("costing") || lower.includes("cost sheet")) return "costing";
        if (lower.includes("purchase order") || lower.includes("p.o. number") || lower.includes("p.o number")) return "purchase_order";
        if (lower.includes("delivery note") || lower.includes("packing list") || lower.includes("dn number")) return "delivery_note";
        if (lower.includes("tax invoice") || lower.includes("invoice no") || lower.includes("invoice number")) return "invoice";
        if (lower.includes("supplier") && lower.includes("invoice")) return "supplier_invoice";
        // FIX #6: Check quotation BEFORE RFQ - quotations often contain RFQ-like keywords
        // Quotations have: "quotation", "quote ref", "proforma", "validity", "valid until", "offer", "proposal"
        if (lower.includes("quotation") || lower.includes("quote ref") || lower.includes("proforma") ||
            lower.includes("validity") || lower.includes("valid until") || lower.includes("our offer") ||
            lower.includes("price proposal") || lower.includes("commercial offer")) return "quotation";
        // RFQ detection comes AFTER quotation
        if (lower.includes("request for quotation") || lower.includes("rfq") || lower.includes("enquiry") || lower.includes("inquiry")) return "rfq";
        // Excel sheets with quantities often are RFQs or costing data
        if (lower.includes("qty") || lower.includes("quantity") || lower.includes("unit price")) return "rfq";
        return "rfq"; // Default to RFQ for unknown docs
    }

    // Get fields relevant to current document type
    function getRelevantFields(): string[] {
        const fieldsByType: Record<string, string[]> = {
            rfq: ['division', 'customer_name', 'project', 'rfq_number', 'delivery_date', 'notes'],
            quotation: ['division', 'customer_name', 'project', 'rfq_number', 'delivery_date', 'total', 'notes'],
            costing: ['division', 'customer_name', 'project', 'notes'],
            invoice: ['division', 'customer_name', 'invoice_number', 'invoice_date', 'po_number', 'total', 'vat', 'due_date'],
            supplier_invoice: ['division', 'supplier_name', 'invoice_number', 'invoice_date', 'po_number', 'total', 'vat', 'due_date', 'currency'],
            purchase_order: ['division', 'supplier_name', 'po_number', 'invoice_date', 'delivery_date', 'total', 'currency'],
            delivery_note: ['customer_name', 'dn_number', 'delivery_date', 'po_number'],
            bank_statement: ['division', 'bank_name', 'account_number', 'period_start', 'period_end', 'opening_balance', 'closing_balance', 'currency', 'notes'],
            contract: ['customer_name', 'supplier_name', 'project', 'notes'],
            report: ['project', 'notes'],
            excel_data: ['division', 'customer_name', 'project', 'notes'],
        };
        return fieldsByType[selectedType] || fieldsByType.rfq;
    }

    function formatFieldLabel(key: string): string {
        const labels: Record<string, string> = {
            customer_name: "Customer",
            division: "Company",
            supplier_name: "Supplier",
            project: "Project Name",
            invoice_number: "Invoice #",
            po_number: "PO #",
            dn_number: "DN #",
            rfq_number: "RFQ #",
            invoice_date: "Date",
            delivery_date: "Delivery Date",
            due_date: "Due Date",
            total: "Total Amount",
            vat: "VAT",
            currency: "Currency",
            notes: "Notes",
            bank_name: "Bank Name",
            account_number: "Account Number",
            opening_balance: "Opening Balance",
            closing_balance: "Closing Balance",
            period_start: "Period Start",
            period_end: "Period End",
        };
        return labels[key] || key.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
    }

    function handleClose() {
        show = false;
        resetState();
        dispatch('close');
    }

    // Handle backdrop click - show confirmation if dirty
    function handleBackdropClick() {
        if (isDirty) {
            showCloseConfirmation = true;
        } else {
            handleClose();
        }
    }

    // Confirm close - discard changes
    function confirmClose() {
        showCloseConfirmation = false;
        handleClose();
    }

    // Cancel close - keep editing
    function cancelClose() {
        showCloseConfirmation = false;
    }

    function resetState() {
        butlerInsights = null;
        duplicateRFQ = null;
        showDuplicateWarning = false;
        duplicateType = null;
        lineItems = [];
        activeTab = 'details';
        // Reset dropdown state
        customerSearchTerm = "";
        supplierSearchTerm = "";
        selectedCustomerId = null;
        selectedSupplierId = null;
        showCustomerDropdown = false;
        showSupplierDropdown = false;
        // Reset initialization flag so next modal open re-initializes
        isDataInitialized = false;
        // Reset close confirmation state
        showCloseConfirmation = false;
        // Reset Butler auto-trigger flag
        butlerAutoTriggered = false;
        // Reset editable data to clean state (includes bank-specific fields)
        editableData = {
            division: getDefaultDivisionKey(),
            customer_name: "",
            supplier_name: "",
            project: "",
            invoice_number: "",
            po_number: "",
            dn_number: "",
            rfq_number: "",
            invoice_date: "",
            delivery_date: "",
            due_date: "",
            total: "",
            vat: "",
            currency: "BHD",
            notes: "",
            bank_name: "",
            account_number: "",
            opening_balance: "",
            closing_balance: "",
            period_start: "",
            period_end: "",
        };
        selectedBankAccountId = "";
        // Reset document type to default
        selectedType = "rfq";
    }

    // Add new line item
    function addLineItem() {
        if (selectedType === 'bank_statement') {
            lineItems = [...lineItems, {
                date: new Date().toISOString().split('T')[0],
                description: "",
                reference: "",
                debit: 0,
                credit: 0,
                balance: 0,
            }];
        } else {
            lineItems = [...lineItems, {
                description: "",
                quantity: 1,
                unit: "pcs",
                part_number: "",
                unit_price: 0
            }];
        }
    }

    // Remove line item
    function removeLineItem(index: number) {
        lineItems = lineItems.filter((_, i) => i !== index);
    }

    // Duplicate line item
    function duplicateLineItem(index: number) {
        const item = { ...lineItems[index] };
        lineItems = [...lineItems.slice(0, index + 1), item, ...lineItems.slice(index + 1)];
    }

    // Calculate total from line items
    function calculateTotal(): number {
        return lineItems.reduce((sum, item) => sum + (item.quantity * item.unit_price), 0);
    }

    // Merge Butler insights into editable data
    async function mergeButlerInsights() {
        if (!butlerInsights) return;
        console.log('[Butler Merge] Starting merge, selectedType:', selectedType);
        console.log('[Butler Merge] extracted_items:', butlerInsights.extracted_items?.length, 'items');
        console.log('[Butler Merge] metadata:', butlerInsights.metadata);

        const assignIfBlank = (key: keyof typeof editableData, value: any) => {
            const nextValue = value == null ? "" : String(value).trim();
            if (!nextValue) return;
            if (!String(editableData[key] || "").trim()) {
                editableData[key] = nextValue;
            }
        };

        // Update fields from Butler if they're empty or Butler has better data
        assignIfBlank('customer_name', butlerInsights.detected_customer);
        assignIfBlank('project', butlerInsights.detected_project || butlerInsights.project);
        assignIfBlank('delivery_date', butlerInsights.required_deadline);
        if (selectedType === 'bank_statement' && butlerInsights.summary) {
            assignIfBlank('notes', butlerInsights.summary);
        }

        // Merge bank statement specific fields from Butler
        if (butlerInsights.metadata) {
            const meta = butlerInsights.metadata;
            assignIfBlank('supplier_name', meta.supplier_name || meta.vendor_name);
            assignIfBlank('invoice_number', meta.invoice_number || meta.reference_number);
            assignIfBlank('po_number', meta.po_number || meta.purchase_order_number);
            assignIfBlank('rfq_number', meta.rfq_number || meta.reference || meta.folder_number);
            assignIfBlank('invoice_date', meta.invoice_date || meta.date);
            assignIfBlank('delivery_date', meta.delivery_date || meta.required_by || meta.deadline);
            assignIfBlank('due_date', meta.due_date);
            assignIfBlank('total', meta.total || meta.total_amount);
            assignIfBlank('vat', meta.vat || meta.vat_amount);
            assignIfBlank('currency', meta.currency);
            assignIfBlank('bank_name', meta.bank_name);
            assignIfBlank('account_number', meta.account_number || meta.iban);
            assignIfBlank('opening_balance', meta.opening_balance);
            assignIfBlank('closing_balance', meta.closing_balance);
            assignIfBlank('period_start', meta.period_start);
            assignIfBlank('period_end', meta.period_end);
        }

        // Merge line items from Butler
        if (butlerInsights.extracted_items && Array.isArray(butlerInsights.extracted_items)) {
            // Helper: convert DD/MM/YYYY or DD-MM-YYYY to YYYY-MM-DD for HTML date input
            const toISODate = (d: string): string => {
                if (!d) return "";
                // Already ISO format?
                if (/^\d{4}-\d{2}-\d{2}$/.test(d)) return d;
                // DD/MM/YYYY or DD-MM-YYYY
                const m = d.match(/^(\d{1,2})[/-](\d{1,2})[/-](\d{4})$/);
                if (m) return `${m[3]}-${m[2].padStart(2,'0')}-${m[1].padStart(2,'0')}`;
                // MM/DD/YYYY fallback
                const m2 = d.match(/^(\d{1,2})[/-](\d{1,2})[/-](\d{2})$/);
                if (m2) return `20${m2[3]}-${m2[1].padStart(2,'0')}-${m2[2].padStart(2,'0')}`;
                return d;
            };

            const butlerItems = selectedType === 'bank_statement'
                ? butlerInsights.extracted_items.map((item: any) => ({
                    date: toISODate(item.date || ""),
                    description: item.description || "",
                    reference: item.reference || item.ref || "",
                    debit: parseNum(item.debit),
                    credit: parseNum(item.credit),
                    balance: parseNum(item.balance),
                }))
                : butlerInsights.extracted_items.map(normalizeCommercialLineItem);

            console.log('[Butler Merge] Mapped', butlerItems.length, 'butler items, existing lineItems:', lineItems.length);
            if (butlerItems.length > 0) {
                console.log('[Butler Merge] First butler item:', JSON.stringify(butlerItems[0]));
            }

            // If we have no items, use Butler's; otherwise ask user
            if (selectedType === 'bank_statement' && lineItems.length > 0) {
                console.log('[Butler Merge] Preserving existing structured bank transactions; Butler items ignored for safety');
            } else if (lineItems.length === 0) {
                lineItems = butlerItems;
            } else if (butlerItems.length > lineItems.length) {
                // Butler found more items - offer to replace
                if (await confirm.ask({
                    title: 'Replace Line Items?',
                    message: `Butler found ${butlerItems.length} items (you have ${lineItems.length}). Replace with Butler's items?`,
                    confirmLabel: 'Replace',
                    variant: 'warning'
                })) {
                    lineItems = butlerItems;
                }
            }
            console.log('[Butler Merge] Final lineItems count:', lineItems.length);
        }

        // If Butler detected it as a different type, consider switching
        if (butlerInsights.document_type) {
            const butlerType = butlerInsights.document_type.toLowerCase();
            if (butlerType.includes('bank') || butlerType.includes('statement')) selectedType = 'bank_statement';
            else if (butlerType.includes('supplier') && butlerType.includes('invoice')) selectedType = 'supplier_invoice';
            else if (butlerType.includes('invoice')) selectedType = 'invoice';
            else if (butlerType.includes('quotation') || butlerType.includes('quote')) selectedType = 'quotation';
            else if (butlerType.includes('purchase') || butlerType.includes('po')) selectedType = 'purchase_order';
            else if (butlerType.includes('delivery')) selectedType = 'delivery_note';
            else if (butlerType.includes('rfq') || butlerType.includes('enquiry')) selectedType = 'rfq';
        }

        editableData = { ...editableData }; // Trigger reactivity
    }

    // Analyze document with Butler AI
    async function analyzeWithButler() {
        if (!ocrResult?.text || butlerAnalyzing) return;

        butlerAnalyzing = true;
        butlerInsights = null;

        console.log('[Butler] Starting analysis, selectedType:', selectedType);

        try {
            const result = await AnalyzeDocumentWithButler(
                ocrResult.text,
                selectedType,
                ocrResult.extracted_data || {}
            );

            console.log('[Butler] Raw result:', JSON.stringify(result).substring(0, 2000));
            console.log('[Butler] extracted_items count:', result?.extracted_items?.length || 0);
            console.log('[Butler] metadata:', result?.metadata);

            butlerInsights = result;
            await mergeButlerInsights();

            // Post-Butler fallback: if bank statement and still no lineItems, parse from text
            if (selectedType === 'bank_statement' && lineItems.length === 0 && ocrResult?.text) {
                console.log('[Butler] No transactions from Butler or merge, trying text parser fallback...');
                const parsed = parseBankTransactionsFromText(ocrResult.text);
                if (parsed.length > 0) {
                    lineItems = parsed;
                    console.log(`[Butler] Text parser fallback found ${parsed.length} transactions`);
                }
            }

            const itemCount = lineItems.length;
            if (itemCount > 0) {
                toast.success(`Butler found ${itemCount} ${selectedType === 'bank_statement' ? 'transactions' : 'line items'}!`);
            } else {
                toast.warning("Butler could not extract line items. Please review the text before saving.");
            }

            // Auto-switch to items tab if items were found
            if (itemCount > 0) {
                activeTab = 'items';
            }
        } catch (err: any) {
            console.error("Butler analysis failed:", err);
            toast.danger(`Analysis failed: ${err?.message || err}`);
        } finally {
            butlerAnalyzing = false;
        }
    }

    // Check for duplicate RFQ
    async function checkDuplicate(): Promise<boolean> {
        if (selectedType !== 'rfq' && selectedType !== 'quotation' && selectedType !== 'costing' && selectedType !== 'excel_data') {
            return false; // Only check duplicates for RFQ-type docs
        }

        checkingDuplicate = true;
        try {
            const reference = editableData.rfq_number || editableData.po_number || "";
            const existingOpp = await CheckDuplicateOpportunity(reference, editableData.customer_name, editableData.project);
            if (existingOpp && existingOpp.id) {
                duplicateRFQ = existingOpp;
                duplicateType = 'opportunity';
                showDuplicateWarning = true;
                return true;
            }

            const result = await CheckDuplicateRFQ(editableData.customer_name, editableData.project, "");
            if (result && result.id) {
                duplicateRFQ = result;
                duplicateType = 'rfq';
                showDuplicateWarning = true;
                return true;
            }
        } catch (err: any) {
            console.log("Duplicate check passed (no match)");
        } finally {
            checkingDuplicate = false;
        }
        return false;
    }

    // Handle save action
    async function handleSave() {
        // Validate required fields based on document type
        if (selectedType === 'rfq' || selectedType === 'quotation' || selectedType === 'costing' || selectedType === 'excel_data') {
            if (!editableData.project.trim()) {
                toast.danger("Project name is required");
                activeTab = 'details';
                return;
            }
        }
        if (selectedType === 'bank_statement') {
            if (!editableData.bank_name.trim() && !editableData.account_number.trim()) {
                toast.danger("Bank name or account number is required");
                activeTab = 'details';
                return;
            }
            if (!selectedBankAccountId) {
                toast.danger("Please choose the bank account this statement belongs to");
                activeTab = 'details';
                return;
            }
        }

        // Check for duplicates
        const isDuplicate = await checkDuplicate();
        if (isDuplicate) return;

        saveDocument();
    }

    function saveDocument() {
        showDuplicateWarning = false;

        const docType = documentTypes.find(t => t.id === selectedType);

        // Calculate total if we have line items with prices
        const calculatedTotal = calculateTotal();
        const finalTotal = calculatedTotal > 0 ? calculatedTotal : parseFloat(editableData.total.replace(/[^0-9.]/g, '')) || 0;

        dispatch('save', {
            type: selectedType,
            screen: docType?.screen || '/dashboard',
            result: ocrResult,
            fileName: fileName,
            // Pass all editable data
            projectName: editableData.project,
            customerName: editableData.customer_name,
            supplierName: editableData.supplier_name,
            estimatedValue: finalTotal,
            // Pass linked entity IDs for proper database relations
            customerId: selectedCustomerId,
            supplierId: selectedSupplierId,
            // Pass structured data for backend
            documentData: {
                ...editableData,
                division: editableData.division || getDefaultDivisionKey(),
                total: finalTotal,
                line_items: lineItems,
                bank_account_id: selectedBankAccountId,
                // Include IDs in document data for backend routing
                customer_id: selectedCustomerId,
                supplier_id: selectedSupplierId
            },
            // Pass line items separately for easy access
            lineItems: lineItems,
            // Butler insights for reference
            butlerInsights: butlerInsights
        });

        handleClose();
    }

    function handleViewExistingRFQ() {
        showDuplicateWarning = false;
        dispatch('viewRFQ', {
            rfqId: duplicateRFQ?.id,
            rfq: duplicateRFQ,
            duplicateType
        });
        handleClose();
    }

    function handleKeydown(e: KeyboardEvent) {
        if (e.key === 'Escape') handleClose();
    }

    function handleBackdropKeydown(event: KeyboardEvent, callback: () => void) {
        if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault();
            callback();
        }
    }
    // Filtered lists for search
    let filteredCustomers = $derived(customerSearchTerm.length > 0
        ? customers.filter(c => c.business_name?.toLowerCase().includes(customerSearchTerm.toLowerCase()))
        : customers.slice(0, 50));
    let filteredSuppliers = $derived(supplierSearchTerm.length > 0
        ? suppliers.filter(s => s.supplier_name?.toLowerCase().includes(supplierSearchTerm.toLowerCase()))
        : suppliers.slice(0, 50));
    let filteredBankAccounts = $derived(bankAccounts.filter((account) => normalizeDivision(account.division || getDefaultDivisionKey()) === normalizeDivision(editableData.division)));
    // Reset state when modal opens (prevent state leak between opens)
    run(() => {
        if (show) {
            resetStateOnOpen();
            loadEntities();
        }
    });
    // Reset state when modal closes
    run(() => {
        if (!show) {
            resetState();
        }
    });
    // Track if user has made changes (dirty state)
    run(() => {
        isDirty = (
            editableData.customer_name !== '' ||
            editableData.project !== '' ||
            editableData.supplier_name !== '' ||
            lineItems.length > 0 ||
            butlerInsights !== null
        );
    });
    // Initialize editable data ONCE when OCR result first arrives
    // Guard prevents field reset on paste/edit operations
    run(() => {
        if (ocrResult && !isDataInitialized) {
            initializeEditableData();
            isDataInitialized = true;
        }
    });
    // Auto-detect document type
    run(() => {
        if (ocrResult?.document_type && ocrResult.document_type !== "auto") {
            selectedType = mapDocType(ocrResult.document_type);
        } else if (ocrResult?.text) {
            selectedType = detectDocumentType(ocrResult.text);
        }
    });
    run(() => {
        if (show && selectedType === 'bank_statement' && selectedBankAccountId && !filteredBankAccounts.some((account) => account.id === selectedBankAccountId)) {
            selectedBankAccountId = "";
            syncDetectedBankAccountSelection();
        }
    });
    run(() => {
        if (show && selectedType === 'bank_statement' && bankAccounts.length > 0 && !selectedBankAccountId) {
            syncDetectedBankAccountSelection();
        }
    });
    run(() => {
        if (show && selectedType === 'bank_statement' && bankAccounts.length === 0 && !loadingBankAccounts) {
            loadBankAccounts();
        }
    });
    run(() => {
        hasStructuredBankTransactions = selectedType === 'bank_statement' && Array.isArray(ocrResult?.extracted_data?.line_items) && ocrResult.extracted_data.line_items.length > 0;
    });
    run(() => {
        if (show && ocrResult?.text && !butlerInsights && !butlerAnalyzing && !butlerAutoTriggered && !hasStructuredBankTransactions) {
            butlerAutoTriggered = true;
            console.log('[Butler Auto] Trigger fired, selectedType:', selectedType, 'docType:', ocrResult?.document_type);
            // Debounce: wait 300ms for modal to settle and selectedType to be set
            setTimeout(() => {
                console.log('[Butler Auto] setTimeout executing, selectedType now:', selectedType);
                if (ocrResult?.text && !butlerInsights && !butlerAnalyzing && !hasStructuredBankTransactions) {
                    analyzeWithButler();
                }
            }, 300);
        }
    });
    run(() => {
        if (show && selectedType === 'bank_statement' && selectedBankAccountId) {
            syncEditableBankFieldsFromSelection();
        }
    });
</script>

<svelte:window onkeydown={handleKeydown} />

{#if show}
    <div
        class="modal-backdrop"
        transition:fade={{ duration: motionMs(MODAL_MOTION_MS), easing: cubicOut }}
        onclick={self(handleBackdropClick)}
        onkeydown={(event) => handleBackdropKeydown(event, handleBackdropClick)}
        role="button"
        tabindex="0"
    >
        <div
            class="modal"
            class:bank-statement-modal={selectedType === 'bank_statement'}
            transition:scale={{ duration: motionMs(MODAL_MOTION_MS), start: 0.98, easing: cubicOut }}
            role="dialog"
            aria-modal="true"
            aria-labelledby="modal-title"
        >
            <header class="modal-header">
                <h2 id="modal-title">Document Capture</h2>
                <div class="header-actions">
                    <span class="confidence-badge">
                        {((ocrResult?.confidence || 0) * 100).toFixed(0)}% OCR
                    </span>
                    <button class="close-btn" onclick={handleClose} aria-label="Close modal">&times;</button>
                </div>
            </header>

            <div class="modal-content">
                {#if processing}
                    <div class="processing-state">
                        <WabiSpinner size="lg" tempo="calm" />
                        <p class="processing-text">Analyzing document...</p>
                        <p class="processing-file">{fileName}</p>
                    </div>
                {:else if ocrResult}
                    <!-- Document Type Selector -->
                    <div class="type-selector">
                        <div class="type-label">Document Type</div>
                        <div class="type-options">
                            {#each documentTypes as docType}
                                <button
                                    class="type-option"
                                    class:selected={selectedType === docType.id}
                                    onclick={() => selectedType = docType.id}
                                >
                                    <span class="type-icon">{docType.icon}</span>
                                    <span class="type-name">{docType.label}</span>
                                </button>
                            {/each}
                        </div>
                    </div>

                    {#if butlerAnalyzing}
                        <div class="butler-status-bar">
                            <WabiSpinner size="sm" tempo="calm" />
                            <span>AI analyzing document...</span>
                        </div>
                    {/if}

                    <!-- Tab Navigation -->
                    <div class="tabs">
                        <button
                            class="tab"
                            class:active={activeTab === 'details'}
                            onclick={() => activeTab = 'details'}
                        >
                            Details
                        </button>
                        <button
                            class="tab"
                            class:active={activeTab === 'items'}
                            onclick={() => activeTab = 'items'}
                        >
                            {selectedType === 'bank_statement' ? 'Transactions' : 'Line Items'} ({lineItems.length})
                        </button>
                        <button
                            class="tab"
                            class:active={activeTab === 'raw'}
                            onclick={() => activeTab = 'raw'}
                        >
                            Raw Text
                        </button>
                    </div>

                    <!-- Details Tab: Editable Fields -->
                    {#if activeTab === 'details'}
                        <div class="form-panel" transition:fade={{ duration: motionMs(150) }}>
                            <div class="form-grid">
                                {#if selectedType === 'bank_statement'}
                                    <div class="form-group full-width">
                                        <label for="bank_account_id">Bank Account</label>
                                        <select id="bank_account_id" bind:value={selectedBankAccountId}>
                                            <option value="">Select the account to save into...</option>
                                            {#each filteredBankAccounts as account}
                                                <option value={account.id}>{formatBankAccountOption(account)}</option>
                                            {/each}
                                        </select>
                                        <p class="field-hint">Required so the statement is stored under the correct bank account.</p>
                                    </div>
                                {/if}
                                {#each getRelevantFields() as fieldKey}
                                    <div class="form-group" class:full-width={fieldKey === 'notes'}>
                                        <label for={fieldKey}>{formatFieldLabel(fieldKey)}</label>
                                        {#if fieldKey === 'division'}
                                            <select id={fieldKey} bind:value={editableData[fieldKey]}>
                                                {#each getDivisionKeys() as divKey}
                                                    <option value={divKey}>{divKey}</option>
                                                {/each}
                                            </select>
                                        {:else if fieldKey === 'notes'}
                                            <textarea
                                                id={fieldKey}
                                                bind:value={editableData[fieldKey]}
                                                placeholder="Additional notes..."
                                                rows="2"
                                            ></textarea>
                                        {:else if fieldKey === 'customer_name'}
                                            <!-- Customer searchable dropdown -->
                                            <div class="autocomplete-wrapper">
                                                <input
                                                    type="text"
                                                    id={fieldKey}
                                                    value={customerSearchTerm}
                                                    oninput={handleCustomerInput}
                                                    onfocus={() => showCustomerDropdown = true}
                                                    onblur={() => setTimeout(() => showCustomerDropdown = false, 200)}
                                                    placeholder="Search customers..."
                                                    autocomplete="off"
                                                />
                                                {#if selectedCustomerId}
                                                    <span class="linked-badge" title="Linked to database">Linked</span>
                                                {/if}
                                                {#if showCustomerDropdown && filteredCustomers.length > 0}
                                                    <div class="autocomplete-dropdown">
                                                        {#each filteredCustomers as customer}
                                                            <button
                                                                class="dropdown-item"
                                                                class:selected={selectedCustomerId === customer.id}
                                                                onmousedown={preventDefault(() => selectCustomer(customer))}
                                                            >
                                                                <span class="item-name">{customer.business_name}</span>
                                                                {#if customer.customer_type}
                                                                    <span class="item-type">{customer.customer_type}</span>
                                                                {/if}
                                                            </button>
                                                        {/each}
                                                        {#if filteredCustomers.length >= 50}
                                                            <div class="dropdown-hint">Type to filter more...</div>
                                                        {/if}
                                                    </div>
                                                {/if}
                                                {#if loadingEntities}
                                                    <span class="loading-indicator">
                                                        <WabiSpinner size="sm" tempo="calm" />
                                                        <span>Loading</span>
                                                    </span>
                                                {/if}
                                            </div>
                                        {:else if fieldKey === 'supplier_name'}
                                            <!-- Supplier searchable dropdown -->
                                            <div class="autocomplete-wrapper">
                                                <input
                                                    type="text"
                                                    id={fieldKey}
                                                    value={supplierSearchTerm}
                                                    oninput={handleSupplierInput}
                                                    onfocus={() => showSupplierDropdown = true}
                                                    onblur={() => setTimeout(() => showSupplierDropdown = false, 200)}
                                                    placeholder="Search suppliers..."
                                                    autocomplete="off"
                                                />
                                                {#if selectedSupplierId}
                                                    <span class="linked-badge" title="Linked to database">Linked</span>
                                                {/if}
                                                {#if showSupplierDropdown && filteredSuppliers.length > 0}
                                                    <div class="autocomplete-dropdown">
                                                        {#each filteredSuppliers as supplier}
                                                            <button
                                                                class="dropdown-item"
                                                                class:selected={selectedSupplierId === supplier.id}
                                                                onmousedown={preventDefault(() => selectSupplier(supplier))}
                                                            >
                                                                <span class="item-name">{supplier.supplier_name}</span>
                                                            </button>
                                                        {/each}
                                                        {#if filteredSuppliers.length >= 50}
                                                            <div class="dropdown-hint">Type to filter more...</div>
                                                        {/if}
                                                    </div>
                                                {/if}
                                                {#if loadingEntities}
                                                    <span class="loading-indicator">
                                                        <WabiSpinner size="sm" tempo="calm" />
                                                        <span>Loading</span>
                                                    </span>
                                                {/if}
                                            </div>
                                        {:else if fieldKey.includes('date')}
                                            <input
                                                type="date"
                                                id={fieldKey}
                                                bind:value={editableData[fieldKey]}
                                            />
                                        {:else if fieldKey === 'currency'}
                                            <select id={fieldKey} bind:value={editableData[fieldKey]}>
                                                <option value="BHD">BHD</option>
                                                <option value="USD">USD</option>
                                                <option value="EUR">EUR</option>
                                                <option value="GBP">GBP</option>
                                            </select>
                                        {:else}
                                            <input
                                                type="text"
                                                id={fieldKey}
                                                bind:value={editableData[fieldKey]}
                                                placeholder={formatFieldLabel(fieldKey)}
                                            />
                                        {/if}
                                    </div>
                                {/each}
                            </div>
                        </div>
                    {/if}

                    <!-- Line Items / Transactions Tab -->
                    {#if activeTab === 'items'}
                        <div class="items-panel" transition:fade={{ duration: motionMs(150) }}>
                            {#if lineItems.length === 0}
                                <div class="empty-items">
                                    <p>No {selectedType === 'bank_statement' ? 'transactions' : 'line items'} yet.</p>
                                    <p class="hint">Click "Extract Details with AI" to auto-detect {selectedType === 'bank_statement' ? 'transactions' : 'items'}, or add manually.</p>
                                </div>
                            {:else if selectedType === 'bank_statement'}
                                <!-- Bank Statement Transaction Table -->
                                <div class="items-table-wrapper">
                                    <table class="items-table">
                                        <thead>
                                            <tr>
                                                <th class="col-date">Date</th>
                                                <th class="col-desc">Description</th>
                                                <th class="col-ref">Reference</th>
                                                <th class="col-debit">Debit</th>
                                                <th class="col-credit">Credit</th>
                                                <th class="col-balance">Balance</th>
                                                <th class="col-actions"></th>
                                            </tr>
                                        </thead>
                                        <tbody>
                                            {#each lineItems as item, i}
                                                <tr>
                                                    <td>
                                                        <input type="date" bind:value={item.date} />
                                                    </td>
                                                    <td>
                                                        <input type="text" bind:value={item.description} placeholder="Transaction description" />
                                                    </td>
                                                    <td>
                                                        <input type="text" bind:value={item.reference} placeholder="Ref" />
                                                    </td>
                                                    <td>
                                                        <input class="number-input" type="number" bind:value={item.debit} min="0" step="0.001" placeholder="0.000" />
                                                    </td>
                                                    <td>
                                                        <input class="number-input" type="number" bind:value={item.credit} min="0" step="0.001" placeholder="0.000" />
                                                    </td>
                                                    <td>
                                                        <input class="number-input" type="number" bind:value={item.balance} min="0" step="0.001" placeholder="0.000" />
                                                    </td>
                                                    <td class="actions-cell">
                                                        <button class="btn-icon danger" onclick={() => removeLineItem(i)} title="Remove" aria-label="Remove transaction {i + 1}">&times;</button>
                                                    </td>
                                                </tr>
                                            {/each}
                                        </tbody>
                                    </table>
                                </div>
                            {:else}
                                <!-- Standard Line Items Table -->
                                <div class="items-table-wrapper">
                                    <table class="items-table">
                                        <thead>
                                            <tr>
                                                <th class="col-desc">Description</th>
                                                <th class="col-qty">Qty</th>
                                                <th class="col-unit">Unit</th>
                                                <th class="col-part">Part #</th>
                                                <th class="col-price">Unit Price</th>
                                                <th class="col-actions"></th>
                                            </tr>
                                        </thead>
                                        <tbody>
                                            {#each lineItems as item, i}
                                                <tr>
                                                    <td>
                                                        <input
                                                            type="text"
                                                            bind:value={item.description}
                                                            placeholder="Item description"
                                                        />
                                                    </td>
                                                    <td>
                                                        <input
                                                            type="number"
                                                            bind:value={item.quantity}
                                                            min="0"
                                                            step="1"
                                                        />
                                                    </td>
                                                    <td>
                                                        <select bind:value={item.unit}>
                                                            <option value="pcs">pcs</option>
                                                            <option value="sets">sets</option>
                                                            <option value="m">m</option>
                                                            <option value="kg">kg</option>
                                                            <option value="lot">lot</option>
                                                        </select>
                                                    </td>
                                                    <td>
                                                        <input
                                                            type="text"
                                                            bind:value={item.part_number}
                                                            placeholder="Part #"
                                                        />
                                                    </td>
                                                    <td>
                                                        <input
                                                            type="number"
                                                            bind:value={item.unit_price}
                                                            min="0"
                                                            step="0.001"
                                                            placeholder="0.000"
                                                        />
                                                    </td>
                                                    <td class="actions-cell">
                                                        <button class="btn-icon" onclick={() => duplicateLineItem(i)} title="Duplicate" aria-label="Duplicate line item {i + 1}">Copy</button>
                                                        <button class="btn-icon danger" onclick={() => removeLineItem(i)} title="Remove" aria-label="Remove line item {i + 1}">&times;</button>
                                                    </td>
                                                </tr>
                                            {/each}
                                        </tbody>
                                    </table>
                                </div>
                            {/if}

                            <div class="items-footer">
                                <button class="btn-add-item" onclick={addLineItem}>
                                    + Add {selectedType === 'bank_statement' ? 'Transaction' : 'Line Item'}
                                </button>
                                {#if lineItems.length > 0}
                                    <div class="items-summary">
                                        <span>{lineItems.length} {selectedType === 'bank_statement' ? 'transactions' : 'items'}</span>
                                        {#if selectedType === 'bank_statement'}
                                            <span class="total">Debits: {editableData.currency} {formatNumber(lineItems.reduce((s, t) => s + (parseFloat(t.debit) || 0), 0), 3)} | Credits: {editableData.currency} {formatNumber(lineItems.reduce((s, t) => s + (parseFloat(t.credit) || 0), 0), 3)}</span>
                                        {:else}
                                            <span class="total">Total: {editableData.currency} {formatNumber(calculateTotal(), 3)}</span>
                                        {/if}
                                    </div>
                                {/if}
                            </div>
                        </div>
                    {/if}

                    <!-- Raw Text Tab -->
                    {#if activeTab === 'raw'}
                        <div class="raw-text-panel" transition:fade={{ duration: motionMs(150) }}>
                            <pre>{ocrResult.text || 'No text extracted'}</pre>
                        </div>
                    {/if}

                    <!-- Butler Summary (if analyzed) -->
                    {#if butlerInsights?.summary}
                        <div class="butler-summary">
                            <span class="summary-label">AI Summary:</span>
                            <span class="summary-text">{butlerInsights.summary}</span>
                        </div>
                    {/if}
                {:else}
                    <div class="error-state">
                        <span class="error-icon">!</span>
                        <p>No OCR result available</p>
                    </div>
                {/if}
            </div>

            <footer class="modal-footer">
                <button class="btn-secondary" onclick={handleClose}>Cancel</button>
                {#if ocrResult && !processing}
                    <button
                        class="btn-primary"
                        onclick={handleSave}
                        disabled={checkingDuplicate}
                    >
                        {#if checkingDuplicate}
                            <WabiSpinner size="sm" tempo="calm" />
                            Checking...
                        {:else}
                            Save as {documentTypes.find(t => t.id === selectedType)?.label}
                        {/if}
                    </button>
                {/if}
            </footer>
        </div>
    </div>
{/if}

<!-- Close Confirmation Modal -->
{#if showCloseConfirmation}
    <div
        class="modal-backdrop confirmation-backdrop"
        transition:fade={{ duration: motionMs(MODAL_MOTION_MS), easing: cubicOut }}
        onclick={self(cancelClose)}
        onkeydown={(event) => handleBackdropKeydown(event, cancelClose)}
        role="button"
        tabindex="0"
    >
        <div class="confirmation-modal" transition:scale={{ duration: motionMs(MODAL_MOTION_MS), start: 0.98, easing: cubicOut }}>
            <h3>Unsaved Changes</h3>
            <p>You have unsaved changes. Are you sure you want to close?</p>
            <div class="confirmation-actions">
                <button class="btn-secondary" onclick={cancelClose}>Keep Editing</button>
                <button class="btn-danger" onclick={confirmClose}>Discard & Close</button>
            </div>
        </div>
    </div>
{/if}

<!-- Duplicate Warning Modal -->
{#if showDuplicateWarning}
    <div
        class="modal-backdrop"
        transition:fade={{ duration: motionMs(MODAL_MOTION_MS), easing: cubicOut }}
        onclick={self(() => showDuplicateWarning = false)}
        onkeydown={(event) => handleBackdropKeydown(event, () => (showDuplicateWarning = false))}
        role="button"
        tabindex="0"
    >
        <div class="warning-modal" transition:scale={{ duration: motionMs(MODAL_MOTION_MS), start: 0.98, easing: cubicOut }} role="alertdialog">
            <div class="warning-header">
                <span class="warning-icon">!</span>
                <h3>Similar Document Exists</h3>
            </div>
            <div class="warning-body">
                <p>A similar RFQ/Opportunity was found:</p>
                <div class="existing-rfq-card">
                    <div class="rfq-detail">
                        <span class="detail-label">ID</span>
                        <span class="detail-value">#{duplicateRFQ?.id}</span>
                    </div>
                    <div class="rfq-detail">
                        <span class="detail-label">Customer</span>
                        <span class="detail-value">{duplicateRFQ?.Client || duplicateRFQ?.customer_name || 'Unknown'}</span>
                    </div>
                    <div class="rfq-detail">
                        <span class="detail-label">{duplicateType === 'opportunity' ? 'Title' : 'Project'}</span>
                        <span class="detail-value">{duplicateRFQ?.Project || duplicateRFQ?.title || duplicateRFQ?.folder_name || 'Unknown'}</span>
                    </div>
                    <div class="rfq-detail">
                        <span class="detail-label">Status</span>
                        <span class="detail-value status-badge">{duplicateRFQ?.Status || duplicateRFQ?.stage || 'pending'}</span>
                    </div>
                    {#if duplicateType === 'opportunity'}
                    <div class="rfq-detail">
                        <span class="detail-label">Reference</span>
                        <span class="detail-value">{duplicateRFQ?.folder_number || '—'}</span>
                    </div>
                    <div class="rfq-detail">
                        <span class="detail-label">Value</span>
                        <span class="detail-value">{duplicateRFQ?.revenue_bhd ? Number(duplicateRFQ.revenue_bhd).toLocaleString('en-US', { minimumFractionDigits: 3, maximumFractionDigits: 3 }) + ' BHD' : '—'}</span>
                    </div>
                    {/if}
                </div>
                <p class="warning-question">What would you like to do?</p>
            </div>
            <div class="warning-footer">
                <button class="btn-secondary" onclick={() => showDuplicateWarning = false}>Cancel</button>
                <button class="btn-outline" onclick={handleViewExistingRFQ}>View Existing</button>
                <button class="btn-primary" onclick={saveDocument}>Create New Anyway</button>
            </div>
        </div>
    </div>
{/if}

<style>
    .modal-backdrop {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.6);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 2000;
        backdrop-filter: blur(4px);
    }

    .modal {
        background: var(--paper, #fff);
        border-radius: 12px;
        width: 95%;
        max-width: 1100px;
        max-height: 90vh;
        overflow: hidden;
        display: flex;
        flex-direction: column;
        box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
    }

    .modal.bank-statement-modal {
        width: min(96vw, 1580px);
        max-width: min(96vw, 1580px);
    }

    .modal-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 16px 20px;
        border-bottom: 1px solid var(--border-subtle, #e5e5e5);
        background: var(--paper-subtle, #fafafa);
    }

    .modal-header h2 {
        font-size: 16px;
        font-weight: 600;
        margin: 0;
        color: var(--ink, #1a1a1a);
    }

    .header-actions {
        display: flex;
        align-items: center;
        gap: 12px;
    }

    .close-btn {
        background: none;
        border: none;
        font-size: 18px;
        cursor: pointer;
        color: var(--ink-light, #666);
        padding: 4px 8px;
        border-radius: 4px;
    }
    .close-btn:hover {
        background: var(--paper, #fff);
    }

    .modal-content {
        padding: 20px;
        overflow-y: auto;
        flex: 1;
    }

    .modal-footer {
        display: flex;
        justify-content: flex-end;
        gap: 12px;
        padding: 16px 20px;
        border-top: 1px solid var(--border-subtle, #e5e5e5);
        background: var(--paper-subtle, #fafafa);
    }

    /* Processing State */
    .processing-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 60px 20px;
        gap: 12px;
    }

    @keyframes spin {
        to { transform: rotate(360deg); }
    }

    .processing-text {
        font-size: 14px;
        color: var(--ink, #1a1a1a);
        margin: 0;
    }

    .processing-file {
        font-family: var(--font-mono, monospace);
        font-size: 12px;
        color: var(--ink-light, #666);
        margin: 0;
    }

    /* Type Selector */
    .type-selector {
        margin-bottom: 16px;
    }

    .type-label {
        display: block;
        font-size: 11px;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: var(--ink-light, #666);
        margin-bottom: 8px;
    }

    .type-options {
        display: grid;
        grid-template-columns: repeat(4, 1fr);
        gap: 8px;
    }

    .type-option {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 4px;
        padding: 10px 6px;
        background: var(--paper-subtle, #f5f5f5);
        border: 2px solid transparent;
        border-radius: 8px;
        cursor: pointer;
        transition: all 0.2s ease;
    }

    .type-option:hover {
        background: var(--paper, #fff);
        border-color: var(--border-medium, #ddd);
    }

    .type-option.selected {
        background: var(--paper, #fff);
        border-color: #15803d;
    }

    .type-icon {
        font-size: 20px;
    }

    .type-name {
        font-size: 10px;
        font-weight: 500;
        color: var(--ink, #1a1a1a);
        text-align: center;
    }

    /* Butler Status Bar (auto-analysis indicator) */
    .butler-status-bar {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 8px 16px;
        background: linear-gradient(135deg, #f5f3ff 0%, #ede9fe 100%);
        border: 1px solid #c4b5fd;
        border-radius: 8px;
        margin-bottom: 16px;
        font-size: 13px;
        color: #4f46e5;
        font-weight: 500;
    }

    .butler-summary {
        background: linear-gradient(135deg, #f5f3ff 0%, #ede9fe 100%);
        border: 1px solid #c4b5fd;
        border-radius: 8px;
        padding: 12px 16px;
        margin-top: 16px;
        font-size: 13px;
    }

    .summary-label {
        font-weight: 600;
        color: #5b21b6;
    }

    .summary-text {
        color: #1e1b4b;
    }

    /* Tabs */
    .tabs {
        display: flex;
        border-bottom: 1px solid var(--border-subtle, #e5e5e5);
        margin-bottom: 16px;
    }

    .tab {
        padding: 10px 16px;
        background: none;
        border: none;
        border-bottom: 2px solid transparent;
        font-size: 13px;
        font-weight: 500;
        color: var(--ink-light, #666);
        cursor: pointer;
        transition: all 0.2s ease;
    }

    .tab:hover {
        color: var(--ink, #1a1a1a);
    }

    .tab.active {
        color: #15803d;
        border-bottom-color: #15803d;
    }

    /* Form Panel */
    .form-panel {
        background: var(--paper-subtle, #f5f5f5);
        border-radius: 8px;
        padding: 16px;
    }

    .form-grid {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 12px;
    }

    .form-group {
        display: flex;
        flex-direction: column;
        gap: 4px;
    }

    .form-group.full-width {
        grid-column: 1 / -1;
    }

    .form-group label {
        font-size: 11px;
        font-weight: 500;
        color: var(--ink-light, #666);
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .form-group input,
    .form-group select,
    .form-group textarea {
        padding: 10px 12px;
        border: 1px solid var(--border-medium, #ddd);
        border-radius: 6px;
        font-size: 14px;
        background: var(--paper, #fff);
        outline: none;
        transition: border-color 0.2s ease;
    }

    .form-group input:focus,
    .form-group select:focus,
    .form-group textarea:focus {
        border-color: #15803d;
    }

    .form-group textarea {
        resize: vertical;
        min-height: 60px;
    }

    /* Autocomplete Dropdown */
    .autocomplete-wrapper {
        position: relative;
        width: 100%;
    }

    .autocomplete-wrapper input {
        width: 100%;
        padding-right: 28px;
    }

    .linked-badge {
        position: absolute;
        right: 10px;
        top: 50%;
        transform: translateY(-50%);
        color: #15803d;
        font-size: 14px;
        font-weight: bold;
    }

    .loading-indicator {
        position: absolute;
        right: 10px;
        top: 50%;
        transform: translateY(-50%);
        display: inline-flex;
        align-items: center;
        gap: 6px;
        font-size: 11px;
        color: var(--ink-light, #666);
        background: rgba(255, 255, 255, 0.92);
        border-radius: 999px;
        padding: 2px 6px;
    }

    .field-hint {
        margin-top: 6px;
        color: var(--ink-light, #666);
        font-size: 12px;
        line-height: 1.4;
    }

    .autocomplete-dropdown {
        position: absolute;
        top: 100%;
        left: 0;
        right: 0;
        max-height: 200px;
        overflow-y: auto;
        background: var(--paper, #fff);
        border: 1px solid var(--border-medium, #ddd);
        border-top: none;
        border-radius: 0 0 6px 6px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
        z-index: 100;
    }

    .dropdown-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        width: 100%;
        padding: 10px 12px;
        background: none;
        border: none;
        border-bottom: 1px solid var(--border-subtle, #eee);
        cursor: pointer;
        text-align: left;
        font-size: 13px;
        transition: background 0.15s ease;
    }

    .dropdown-item:hover {
        background: var(--paper-subtle, #f5f5f5);
    }

    .dropdown-item.selected {
        background: #dcfce7;
    }

    .dropdown-item:last-child {
        border-bottom: none;
    }

    .item-name {
        color: var(--ink, #1a1a1a);
        font-weight: 500;
    }

    .item-type {
        font-size: 10px;
        color: var(--ink-light, #666);
        background: var(--paper-subtle, #f0f0f0);
        padding: 2px 6px;
        border-radius: 4px;
    }

    .dropdown-hint {
        padding: 8px 12px;
        font-size: 11px;
        color: var(--ink-faint, #999);
        text-align: center;
        background: var(--paper-subtle, #f5f5f5);
    }

    /* Items Panel */
    .items-panel {
        background: var(--paper-subtle, #f5f5f5);
        border-radius: 8px;
        padding: 16px;
    }

    .empty-items {
        text-align: center;
        padding: 40px 20px;
        color: var(--ink-light, #666);
    }

    .empty-items p {
        margin: 0 0 8px 0;
    }

    .empty-items .hint {
        font-size: 12px;
        color: var(--ink-faint, #999);
    }

    .items-table-wrapper {
        overflow-x: auto;
        margin-bottom: 12px;
    }

    .items-table {
        width: 100%;
        border-collapse: collapse;
        font-size: 13px;
    }

    .items-table th {
        text-align: left;
        padding: 8px;
        background: var(--paper, #fff);
        border-bottom: 2px solid var(--border-medium, #ddd);
        font-size: 11px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: var(--ink-light, #666);
    }

    .items-table td {
        padding: 6px 8px;
        border-bottom: 1px solid var(--border-subtle, #e5e5e5);
    }

    .items-table input,
    .items-table select {
        width: 100%;
        padding: 6px 8px;
        border: 1px solid var(--border-subtle, #e5e5e5);
        border-radius: 4px;
        font-size: 12px;
        background: var(--paper, #fff);
    }

    .items-table input:focus,
    .items-table select:focus {
        border-color: #15803d;
        outline: none;
    }

    .col-desc { width: 35%; }
    .col-qty { width: 10%; }
    .col-unit { width: 10%; }
    .col-part { width: 15%; }
    .col-price { width: 15%; }
    .col-actions { width: 15%; }
    .col-ref { min-width: 140px; }
    .col-debit,
    .col-credit { min-width: 140px; }
    .col-balance { min-width: 156px; }

    .items-table .number-input {
        min-width: 132px;
        text-align: right;
        font-family: var(--font-mono, monospace);
        font-variant-numeric: tabular-nums;
    }

    .actions-cell {
        white-space: nowrap;
    }

    .btn-icon {
        background: none;
        border: none;
        padding: 4px 6px;
        cursor: pointer;
        font-size: 14px;
        border-radius: 4px;
    }

    .btn-icon:hover {
        background: var(--paper, #fff);
    }

    .btn-icon.danger:hover {
        background: #fee2e2;
    }

    .items-footer {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding-top: 12px;
        border-top: 1px solid var(--border-subtle, #e5e5e5);
    }

    .btn-add-item {
        padding: 8px 16px;
        background: var(--paper, #fff);
        border: 1px dashed var(--border-medium, #ddd);
        border-radius: 6px;
        font-size: 13px;
        cursor: pointer;
        color: var(--ink-light, #666);
        transition: all 0.2s ease;
    }

    .btn-add-item:hover {
        border-color: #15803d;
        color: #15803d;
    }

    .items-summary {
        display: flex;
        gap: 16px;
        font-size: 13px;
        color: var(--ink-light, #666);
    }

    .items-summary .total {
        font-weight: 600;
        color: var(--ink, #1a1a1a);
    }

    /* Raw Text Panel */
    .raw-text-panel {
        background: #1a1a1a;
        border-radius: 8px;
        padding: 16px;
        max-height: 300px;
        overflow-y: auto;
    }

    .raw-text-panel pre {
        margin: 0;
        font-size: 11px;
        font-family: var(--font-mono, monospace);
        color: #a3e635;
        white-space: pre-wrap;
        word-break: break-word;
    }

    /* Badges */
    .confidence-badge {
        font-size: 10px;
        font-weight: 500;
        padding: 4px 10px;
        background: #dcfce7;
        color: #166534;
        border-radius: 12px;
    }

    /* Buttons */
    .btn-secondary {
        padding: 10px 20px;
        background: var(--paper, #fff);
        border: 1px solid var(--border-medium, #ddd);
        border-radius: 6px;
        font-size: 13px;
        cursor: pointer;
        color: var(--ink, #1a1a1a);
    }
    .btn-secondary:hover {
        background: var(--paper-subtle, #f5f5f5);
    }

    .btn-primary {
        padding: 10px 20px;
        background: #15803d;
        border: none;
        border-radius: 6px;
        font-size: 13px;
        font-weight: 500;
        cursor: pointer;
        color: white;
        display: flex;
        align-items: center;
        gap: 8px;
    }
    .btn-primary:hover:not(:disabled) {
        background: #166534;
    }
    .btn-primary:disabled {
        opacity: 0.6;
        cursor: not-allowed;
    }

    .btn-outline {
        background: transparent;
        color: #15803d;
        border: 1px solid #15803d;
        padding: 10px 16px;
        border-radius: 6px;
        font-size: 13px;
        font-weight: 500;
        cursor: pointer;
    }
    .btn-outline:hover {
        background: #f0fdf4;
    }

    /* Error State */
    .error-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 60px 20px;
        gap: 8px;
    }

    .error-icon {
        font-size: 32px;
    }

    .error-state p {
        color: var(--ink-light, #666);
        margin: 0;
    }

    /* Warning Modal */
    .warning-modal {
        background: var(--paper, #fff);
        border-radius: 12px;
        width: 90%;
        max-width: 420px;
        box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
        overflow: hidden;
    }

    .warning-header {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 16px 20px;
        border-bottom: 1px solid var(--border-subtle, #e5e5e5);
        background: #fef3c7;
    }

    .warning-icon {
        font-size: 24px;
    }

    .warning-header h3 {
        font-size: 16px;
        font-weight: 600;
        margin: 0;
        color: #92400e;
    }

    .warning-body {
        padding: 20px;
    }

    .warning-body p {
        margin: 0 0 12px 0;
        font-size: 14px;
        color: var(--ink, #1a1a1a);
    }

    .existing-rfq-card {
        background: var(--paper-subtle, #f5f5f5);
        border-radius: 8px;
        padding: 12px 16px;
    }

    .rfq-detail {
        display: flex;
        justify-content: space-between;
        padding: 6px 0;
        border-bottom: 1px solid var(--border-subtle, #e5e5e5);
    }

    .rfq-detail:last-child {
        border-bottom: none;
    }

    .detail-label {
        font-size: 12px;
        color: var(--ink-light, #666);
    }

    .detail-value {
        font-size: 12px;
        font-weight: 500;
        color: var(--ink, #1a1a1a);
    }

    .status-badge {
        background: #dbeafe;
        color: #1e40af;
        padding: 2px 8px;
        border-radius: 10px;
        font-size: 11px;
    }

    .warning-question {
        font-weight: 500;
        margin-top: 16px !important;
    }

    .warning-footer {
        display: flex;
        justify-content: flex-end;
        gap: 12px;
        padding: 16px 20px;
        border-top: 1px solid var(--border-subtle, #e5e5e5);
        background: var(--paper-subtle, #fafafa);
    }

    /* Responsive */
    @media (max-width: 640px) {
        .type-options {
            grid-template-columns: repeat(2, 1fr);
        }
        .form-grid {
            grid-template-columns: 1fr;
        }
    }

    /* Close Confirmation Modal */
    .confirmation-backdrop {
        z-index: 2001;
    }

    .confirmation-modal {
        background: var(--surface-elevated, #fff);
        border-radius: 12px;
        padding: 24px;
        max-width: 400px;
        text-align: center;
        box-shadow: 0 8px 32px rgba(0,0,0,0.2);
    }

    .confirmation-modal h3 {
        margin: 0 0 12px;
        color: var(--text-primary, #1a1a1a);
        font-size: 18px;
        font-weight: 600;
    }

    .confirmation-modal p {
        margin: 0 0 20px;
        color: var(--text-secondary, #666);
        font-size: 14px;
    }

    .confirmation-actions {
        display: flex;
        gap: 12px;
        justify-content: center;
    }

    .btn-danger {
        background: #dc2626;
        color: white;
        border: none;
        padding: 10px 20px;
        border-radius: 6px;
        cursor: pointer;
        font-size: 13px;
        font-weight: 500;
    }

    .btn-danger:hover {
        background: #b91c1c;
    }
</style>
