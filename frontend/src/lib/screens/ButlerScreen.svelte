<script lang="ts">
    import { stopPropagation } from 'svelte/legacy';

    import { onMount, onDestroy, createEventDispatcher } from "svelte";
    import { fly } from "svelte/transition";
    import WabiSpinner from "../components/ui/WabiSpinner.svelte";
    import {
        ChatWithButler } from "../../../wailsjs/go/main/App";
import { ListConversations, GetConversationMessages, ChatWithButlerPersistent, DeleteConversation, PurgeAllConversations } from "../../../wailsjs/go/main/ButlerService";
import { ListCustomers, GetCustomer } from "../../../wailsjs/go/main/CRMService";
    import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime";

    const dispatch = createEventDispatcher();

    let messages = $state([]);
    let userInput = $state("");
    let loading = $state(false);
    let confidence = 0.85;
    let insights = [];

    // Conversation persistence
    let conversations: any[] = $state([]);
    let activeConversationId = $state("");
    let showConversations = true;
    let messageSequence = 0;
    let hiddenConversationKeys = new Set<string>();
    let loadingConversation = $state(false);
    const ACTIVE_CONVERSATION_STORAGE_KEY = "butler_active_conversation_id";
    const ACTION_CONFIRMATION_WINDOW_MS = 6000;
    let pendingActionKey = "";
    let pendingActionTimer: ReturnType<typeof setTimeout> | null = null;

    function convId(conv: any): string {
        return String(conv?.id || conv?.ID || "");
    }

    function convTitle(conv: any): string {
        return String(conv?.title || conv?.Title || "Untitled");
    }

    function convKey(conv: any): string {
        return convId(conv) || convTitle(conv);
    }

    function loadHiddenConversationKeys() {
        try {
            const raw = localStorage.getItem("butler_hidden_conversations");
            if (!raw) return;
            const parsed = JSON.parse(raw);
            if (Array.isArray(parsed)) {
                hiddenConversationKeys = new Set(parsed.map((v) => String(v)));
            }
        } catch {
            hiddenConversationKeys = new Set();
        }
    }

    function saveHiddenConversationKeys() {
        try {
            localStorage.setItem("butler_hidden_conversations", JSON.stringify(Array.from(hiddenConversationKeys)));
        } catch {
            // no-op
        }
    }

    function loadActiveConversationId() {
        try {
            activeConversationId = String(localStorage.getItem(ACTIVE_CONVERSATION_STORAGE_KEY) || "").trim();
        } catch {
            activeConversationId = "";
        }
    }

    function saveActiveConversationId() {
        try {
            if (activeConversationId) {
                localStorage.setItem(ACTIVE_CONVERSATION_STORAGE_KEY, activeConversationId);
            } else {
                localStorage.removeItem(ACTIVE_CONVERSATION_STORAGE_KEY);
            }
        } catch {
            // no-op
        }
    }

    function nextLocalMessageId() {
        messageSequence += 1;
        return `local-${Date.now()}-${messageSequence}`;
    }

    function addMessage(role, text, actions = [], messageId = "") {
        const id = String(messageId || "").trim() || nextLocalMessageId();
        messages = [...messages, { id, role, text, actions }];
        scrollToBottom();
    }

    function escapeHtml(value) {
        return String(value || "")
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;")
            .replace(/"/g, "&quot;")
            .replace(/'/g, "&#39;");
    }

    function isMarkdownTableSeparator(line) {
        const raw = String(line || "").trim();
        if (!raw.includes("-")) return false;
        return /^\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)+\|?$/.test(raw);
    }

    function splitMarkdownTableRow(line) {
        let raw = String(line || "").trim();
        if (raw.startsWith("|")) raw = raw.slice(1);
        if (raw.endsWith("|")) raw = raw.slice(0, -1);
        return raw.split("|").map((cell) => cell.trim());
    }

    function renderMarkdownTable(lines) {
        if (!Array.isArray(lines) || lines.length < 2) return "";
        const headerCells = splitMarkdownTableRow(lines[0]);
        if (headerCells.length === 0) return "";

        const bodyLines = lines.slice(2);
        const headerHtml = headerCells.map((cell) => `<th>${escapeHtml(cell)}</th>`).join("");
        const rowsHtml = bodyLines
            .map((line) => splitMarkdownTableRow(line))
            .filter((cells) => cells.length > 0)
            .map((cells) => {
                const normalized = headerCells.map((_, idx) => escapeHtml(cells[idx] ?? ""));
                return `<tr>${normalized.map((cell) => `<td>${cell}</td>`).join("")}</tr>`;
            })
            .join("");

        return `<div class="table-wrap"><table style="width:100%;border-collapse:collapse;table-layout:fixed;font-size:12px;background:var(--surface);border:1px solid var(--border);border-radius:8px;overflow:hidden;"><thead><tr>${headerHtml.replace(/<th>/g, '<th style="padding:8px 10px;border:1px solid var(--border);text-align:left;vertical-align:top;word-break:break-word;background:var(--surface-elevated);color:var(--text-primary);font-weight:600;">')}</tr></thead><tbody>${rowsHtml.replace(/<td>/g, '<td style="padding:8px 10px;border:1px solid var(--border);text-align:left;vertical-align:top;word-break:break-word;">')}</tbody></table></div>`;
    }

    function formatInlineButlerMarkdown(line) {
        let safe = escapeHtml(line);
        safe = safe.replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>");
        safe = safe.replace(/`([^`]+)`/g, "<code>$1</code>");
        return safe;
    }

    function stripDecorativeMarkdown(line) {
        return String(line || "")
            .replace(/^#{1,6}\s+/, "")
            .replace(/^[-*_]{3,}\s*$/, "")
            .trim();
    }

    function formatButlerMessage(text) {
        const input = String(text || "");
        if (!input) return "";

        const lines = input.split(/\r?\n/);
        const output = [];
        let i = 0;

        while (i < lines.length) {
            const line = lines[i];
            const next = i + 1 < lines.length ? lines[i + 1] : "";
            const isTableHeader = line.includes("|") && isMarkdownTableSeparator(next);

            if (isTableHeader) {
                const tableLines = [line, next];
                i += 2;
                while (i < lines.length && lines[i].includes("|")) {
                    tableLines.push(lines[i]);
                    i += 1;
                }
                output.push(renderMarkdownTable(tableLines));
                continue;
            }

            const trimmed = line.trim();

            if (!trimmed) {
                output.push("<br/>");
                i += 1;
                continue;
            }

            if (/^[-*_]{3,}\s*$/.test(trimmed)) {
                i += 1;
                continue;
            }

            const headingMatch = trimmed.match(/^(#{1,6})\s+(.+)$/);
            if (headingMatch) {
                const text = stripDecorativeMarkdown(headingMatch[2]);
                output.push(`<div class="butler-section-title">${formatInlineButlerMarkdown(text)}</div>`);
                i += 1;
                continue;
            }

            const bulletMatch = trimmed.match(/^[-*•]\s+(.+)$/);
            if (bulletMatch) {
                output.push(`<div class="butler-list-item"><span></span><p>${formatInlineButlerMarkdown(stripDecorativeMarkdown(bulletMatch[1]))}</p></div>`);
                i += 1;
                continue;
            }

            const numberedMatch = trimmed.match(/^(\d+)[.)]\s+(.+)$/);
            if (numberedMatch) {
                output.push(`<div class="butler-list-item numbered"><span>${escapeHtml(numberedMatch[1])}</span><p>${formatInlineButlerMarkdown(stripDecorativeMarkdown(numberedMatch[2]))}</p></div>`);
                i += 1;
                continue;
            }

            output.push(`<span>${formatInlineButlerMarkdown(stripDecorativeMarkdown(line))}</span>`);
            i += 1;
        }

        return output.join("<br/>");
    }

    function scrollToBottom() {
        setTimeout(() => {
            const el = document.getElementById("chat-feed");
            if (el) el.scrollTop = el.scrollHeight;
        }, 100);
    }

    const ACTION_TYPE_ALIAS: Record<string, string> = {
        "create_offer": "create",
        "createoffer": "create",
        "create_offer_draft": "create",
        "create_quotation": "create",
        "createoffer_draft": "create",
        "createfollowup": "create",
        "create_follow_up": "create",
        "create_followup_task": "create",
        "create_stock_adjustment": "create",
        "create_stockadjustment": "create",
        "create_stock": "create",
        "createorder": "create",
        "create_order": "create",
        "createfollowuptask": "create",
        "daily_briefing": "daily_briefing",
    };

    const ACTION_TARGET_ALIAS: Record<string, string> = {
        "offer_draft": "offer",
        "offerdraft": "offer",
        "quotation": "offer",
        "quote": "offer",
        "opportunities": "opportunity",
        "customercontact": "customer_contact",
        "customer contact": "customer_contact",
        "customer_contacts": "customer_contact",
        "suppliercontact": "supplier_contact",
        "supplier contact": "supplier_contact",
        "supplier_contacts": "supplier_contact",
        "followup": "follow_up",
        "followup_task": "follow_up",
        "follow_up_task": "follow_up",
        "follow-up task": "follow_up",
        "follow up task": "follow_up",
        "followuptask": "follow_up",
        "stockadjustment": "stock_adjustment",
        "stock-adjustment": "stock_adjustment",
        "stock_adjustments": "stock_adjustment",
        "stockadjust": "stock_adjustment",
        "daily briefing": "daily_briefing",
    };

    function normalizeActionType(value) {
        const normalized = String(value || "")
            .trim()
            .toLowerCase()
            .replace(/\s+/g, "_")
            .replace(/-/g, "_");

        return ACTION_TYPE_ALIAS[normalized] || normalized;
    }

    function resolveActionTarget(rawTarget) {
        const target = String(rawTarget || "").trim().toLowerCase();
        if (ACTION_TARGET_ALIAS[target]) {
            return ACTION_TARGET_ALIAS[target];
        }
        switch (target) {
            case "offer_draft":
            case "offerdraft":
                return "offer_draft";
            case "offer":
            case "offers":
            case "quotation":
            case "quote":
            case "quotations":
                return "offer";
            case "po":
            case "purchaseorder":
            case "purchase_orders":
            case "purchase-order":
                return "purchase_order";
            case "invoic":
            case "invoice":
            case "invoices":
                return "invoice";
            case "follow-up":
            case "followup":
            case "followup_task":
            case "follow_up_task":
            case "follow up task":
            case "follow-up task":
            case "task":
            case "tasks":
                return "follow_up";
            case "rfq":
                return "rfq";
            case "costingsheet":
                return "costing_sheet";
            case "supplierinvoice":
                return "supplier_invoice";
            case "supplier_invoice":
                return "supplier_invoice";
            case "stock_adjustment":
            case "stockadjustment":
            case "stock-adjustment":
                return "stock_adjustment";
            case "stock adjustment":
                return "stock_adjustment";
            case "order":
            case "orders":
                return "order";
            case "opportunity":
            case "opportunities":
                return "opportunity";
            case "costings":
            case "costingsheets":
                return "costing_sheet";
            case "costing sheet":
                return "costing_sheet";
            case "follow-up":
            case "followup":
                return "follow_up";
            case "customer":
            case "customers":
                return "customer";
            case "customer_contact":
            case "customer contacts":
            case "customer contact":
                return "customer_contact";
            case "supplier":
            case "suppliers":
                return "supplier";
            case "supplier_contact":
            case "supplier contacts":
            case "supplier contact":
                return "supplier_contact";
            case "contact":
            case "contacts":
                return "contact";
            case "opportunity":
            case "opportunities":
                return "opportunity";
            case "costingsheet":
                return "costing_sheet";
            case "finance":
                return "finance";
            case "operations":
                return "operations";
            case "dashboard":
            case "home":
                return "dashboard";
            default:
                return target || "";
        }
    }

    function getActionWorkflowKey(action: any) {
        const type = normalizeActionType(action?.type);
        const rawTarget = String(action?.target || "").trim().toLowerCase();
        const target = String(resolveActionTarget(rawTarget)).toLowerCase();

        if (type === "daily_briefing") return "daily_briefing";
        if (type === "approve" || type === "reject") return `${type}_action`;
        if (type === "update") return `update_${target}`;

        if (type === "create") {
            if (target === "follow_up") return "create_follow_up";
            if (target === "offer") return "create_offer_draft";
            if (target === "order") return "create_order";
            if (target === "opportunity") return "create_opportunity";
            if (target === "customer_contact") return "create_customer_contact";
            if (target === "supplier_contact") return "create_supplier_contact";
            if (target === "contact") return "create_contact";
            if (target === "stock_adjustment") return "create_stock_adjustment";
            return "create";
        }

        if (type === "navigate" || type === "open") return `open_${target || "screen"}`;
        if (type === "analyze" || type === "fetch") return type;

        if ((type === "invoke" || type === "generate" || type === "briefing") && (target.includes("daily") || String(type) === "daily_briefing")) {
            return "daily_briefing";
        }

        return type;
    }

    function normalizeToPlainText(value) {
        if (value === null || value === undefined) return "";
        if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") return String(value);
        return "";
    }

    function parseNumeric(value, fallback = 0) {
        const raw = Number(normalizeToPlainText(value));
        return Number.isFinite(raw) ? raw : fallback;
    }

    function normalizeDueDate(value, fallback = "") {
        const raw = normalizeToPlainText(value).trim();
        if (!raw) return fallback;

        const asDate = new Date(raw);
        if (Number.isNaN(asDate.getTime())) {
            return fallback;
        }

        return asDate.toISOString().split("T")[0];
    }

    async function resolveCustomerIdFromHint(rawHint, fallbackNameHint = "") {
        const hint = toActionIdValue(rawHint);
        if (hint && hint === String(rawHint).trim() && !/\s/.test(hint)) {
            return hint;
        }

        const customers = await loadCustomerLookupCache();
        const nameHint = normalizeToPlainText(hint || fallbackNameHint).trim();
        if (!nameHint) return "";

        const normalizedHint = normalizeCustomerLookupText(nameHint);
        const exact = customers.find((item) => item.normalizedName === normalizedHint);
        if (exact?.id) return exact.id;

        const partial = customers.find((item) => {
            const normalized = item.normalizedName;
            return normalized.includes(normalizedHint) || normalizedHint.includes(normalized);
        });
        return partial?.id || "";
    }

    async function resolveActionCustomerIdentity(action) {
        const data = getActionDataObject(action);
        const customerIdHint = toActionIdValue(
            data.customer_id || data.customerId || data.customer_id_text || data.customerId_text || data.customer || data.customer_name || data.customerName,
        );
        const customerNameHint = normalizeToPlainText(data.customer_name || data.customerName || data.customer || data.contact || data.client);

        let customerId = "";
        if (toActionIdValue(customerIdHint)) {
            customerId = customerIdHint;
        } else if (customerNameHint) {
            customerId = await resolveCustomerIdFromHint(customerNameHint);
        }

        let customerName = customerNameHint;
        if (!customerName && customerId) {
            customerName = await resolveCustomerNameFromId(customerId);
        }

        return {
            customerId: toActionIdValue(customerId),
            customerName,
        };
    }

    function toActionIdValue(value) {
        const raw = String(value ?? "").trim();
        if (!raw) return "";
        if (/^\d+$/.test(raw)) return raw;
        const parsed = Number(raw);
        return Number.isFinite(parsed) ? String(parsed) : raw;
    }

    function toNumericId(value) {
        const num = Number(toActionIdValue(value));
        return Number.isFinite(num) ? num : null;
    }

    function parseActionListPayload(rawActionData) {
        if (rawActionData === null || rawActionData === undefined) return [];
        if (typeof rawActionData !== "string") return [];

        const raw = rawActionData.trim();
        if (!raw) return [];

        try {
            const parsed = JSON.parse(raw);
            if (Array.isArray(parsed)) return parsed;
            if (parsed && Array.isArray(parsed.actions)) return parsed.actions;
            if (parsed && typeof parsed === "object") return [parsed];
        } catch {
            // Keep legacy plain-text payloads ignored for action hydration.
        }
        return [];
    }

    function toStringArray(raw) {
        if (!raw) return [];

        if (Array.isArray(raw)) {
            return raw
                .map((item) => String(item || "").trim())
                .filter(Boolean);
        }

        if (typeof raw === "string") {
            return raw
                .split(",")
                .map((item) => item.trim())
                .filter(Boolean);
        }

        return [];
    }

    function toBoolean(value) {
        if (typeof value === "boolean") return value;
        if (typeof value === "number") return value === 1;
        if (typeof value === "string") {
            const raw = value.trim().toLowerCase();
            return raw === "1" || raw === "true" || raw === "yes";
        }
        return false;
    }

    function parseStoredActionMetadata(rawMetadata) {
        if (rawMetadata === null || rawMetadata === undefined) return [];
        if (typeof rawMetadata !== "string") return [];
        const raw = rawMetadata.trim();
        if (!raw) return [];

        try {
            const parsed = JSON.parse(raw);
            if (!parsed || typeof parsed !== "object") return [];
            if (Array.isArray(parsed.actions)) return parsed.actions;
            if (Array.isArray(parsed.data?.actions)) return parsed.data.actions;
            if (Array.isArray(parsed.action_contract?.actions)) return parsed.action_contract.actions;
            if (Array.isArray(parsed.action_contract?.data?.actions)) return parsed.action_contract.data.actions;
            return [];
        } catch {
            return [];
        }
    }

    function parseActionDataToObject(rawData) {
        if (rawData === null || rawData === undefined) return {};
        if (typeof rawData === "string") {
            try {
                const parsed = JSON.parse(rawData);
                if (parsed && typeof parsed === "object") return parsed;
            } catch {
                return {};
            }
        }
        if (typeof rawData === "object") return rawData;
        return {};
    }

    function normalizeCustomerLookupText(value) {
        return String(value || "")
            .toLowerCase()
            .replace(/\b(ltd|llc|inc|co|pvt|ltd\.|inc\.|co\.)\b/g, "")
            .replace(/[^a-z0-9 ]/g, " ")
            .replace(/\s+/g, " ")
            .trim();
    }

    let customerLookupCache = [];
    let customerLookupLoadedAt = 0;
    const CUSTOMER_LOOKUP_TTL_MS = 5 * 60 * 1000;
    let supplierLookupCache = [];
    let supplierLookupLoadedAt = 0;
    const SUPPLIER_LOOKUP_TTL_MS = 5 * 60 * 1000;

    async function loadCustomerLookupCache(forceRefresh = false) {
        const now = Date.now();
        if (!forceRefresh && customerLookupCache.length > 0 && now - customerLookupLoadedAt < CUSTOMER_LOOKUP_TTL_MS) {
            return customerLookupCache;
        }

        try {
            const customers = await ListCustomers(500, 0);
            customerLookupCache = Array.isArray(customers)
                ? customers
                      .map((customer: any) => {
                          const name = String(customer?.business_name || "").trim();
                          return {
                              id: String(customer?.id || ""),
                              name,
                              normalizedName: normalizeCustomerLookupText(name),
                          };
                      })
                      .filter((customer) => customer.name && customer.id)
                : [];
            customerLookupLoadedAt = now;
            return customerLookupCache;
        } catch {
            return customerLookupCache;
        }
    }

    async function resolveCustomerNameFromId(customerId) {
        const id = toActionIdValue(customerId);
        if (!id) return "";

        try {
            const customer: any = await GetCustomer(id);
            return String(customer?.business_name || "").trim();
        } catch {
            return "";
        }
    }

    async function loadSupplierLookupCache(forceRefresh = false) {
        const now = Date.now();
        if (!forceRefresh && supplierLookupCache.length > 0 && now - supplierLookupLoadedAt < SUPPLIER_LOOKUP_TTL_MS) {
            return supplierLookupCache;
        }

        try {
            const suppliers = await invokeAppBridge("ListSuppliers", 500, 0);
            supplierLookupCache = Array.isArray(suppliers)
                ? suppliers
                      .map((supplier: any) => {
                          const name = String(supplier?.supplier_name || "").trim();
                          return {
                              id: String(supplier?.id || ""),
                              name,
                              normalizedName: normalizeCustomerLookupText(name),
                          };
                      })
                      .filter((supplier) => supplier.name && supplier.id)
                : [];
            supplierLookupLoadedAt = now;
            return supplierLookupCache;
        } catch {
            return supplierLookupCache;
        }
    }

    async function resolveSupplierIdFromHint(rawHint, fallbackNameHint = "") {
        const hint = toActionIdValue(rawHint);
        if (hint && hint === String(rawHint).trim() && !/\s/.test(hint)) {
            return hint;
        }

        const suppliers = await loadSupplierLookupCache();
        const nameHint = normalizeToPlainText(hint || fallbackNameHint).trim();
        if (!nameHint) return "";

        const normalizedHint = normalizeCustomerLookupText(nameHint);
        const exact = suppliers.find((item) => item.normalizedName === normalizedHint);
        if (exact?.id) return exact.id;

        const partial = suppliers.find((item) => {
            const normalized = item.normalizedName;
            return normalized.includes(normalizedHint) || normalizedHint.includes(normalized);
        });
        return partial?.id || "";
    }

    async function resolveSupplierNameFromId(supplierId) {
        const id = toActionIdValue(supplierId);
        if (!id) return "";

        try {
            const supplier = await invokeAppBridge("GetSupplier", id);
            return String(supplier?.supplier_name || "").trim();
        } catch {
            return "";
        }
    }

    async function resolveActionSupplierIdentity(action) {
        const data = getActionDataObject(action);
        const supplierIdHint = toActionIdValue(
            data.supplier_id || data.supplierId || data.supplier_id_text || data.supplierId_text || data.supplier || data.supplier_name || data.supplierName,
        );
        const supplierNameHint = normalizeToPlainText(data.supplier_name || data.supplierName || data.supplier || data.vendor);

        let supplierId = "";
        if (toActionIdValue(supplierIdHint)) {
            supplierId = supplierIdHint;
        } else if (supplierNameHint) {
            supplierId = await resolveSupplierIdFromHint(supplierNameHint);
        }

        let supplierName = supplierNameHint;
        if (!supplierName && supplierId) {
            supplierName = await resolveSupplierNameFromId(supplierId);
        }

        return {
            supplierId: toActionIdValue(supplierId),
            supplierName,
        };
    }

    function normalizeStoredAction(action) {
        if (!action || typeof action !== "object") return null;

        const type = String(action.type || action.Type || "").toLowerCase().trim();
        if (!type) return null;

        const normalizedType = normalizeActionType(type);

        const inferredTarget = resolveActionTarget(
            action.target || action.Target || action.entity || action.Entity || action.entity_type || action.EntityType || action.target_type || action.targetType,
        );
        const runtimeStatus = String(action.execution_status || action.executionStatus || action.status || "").trim().toLowerCase();
        const statusConstraints = Array.isArray(action.status_constraints)
            ? action.status_constraints
            : Array.isArray(action.statusConstraints)
                ? action.statusConstraints
                : [];
        const missingFields = toStringArray(action.missing_fields || action.missingFields);
        const requiredFields = toStringArray(action.required_fields || action.requiredFields);

        let executionStatus = runtimeStatus;
        if (!executionStatus) {
            executionStatus = String(action.ready_for_execution ? "ready_for_execution" : "").trim().toLowerCase();
        }
        if (!executionStatus) {
            executionStatus = "";
        }
        const rawData = parseActionDataToObject(action.data ?? action.parameters ?? action.payload ?? action.rawData);
        const normalizedData =
            rawData && typeof rawData === "object" && "payload" in rawData && typeof rawData.payload === "object" ? rawData.payload : rawData;

        return {
            type: normalizedType,
            target: inferredTarget,
            data: normalizedData ?? {},
            label: String(action.label || action.action_label || action.Label || action.ActionLabel || action.name || action.Name || "").trim(),
            execution_status: executionStatus,
            requires_approval: toBoolean(action.requires_approval || action.requiresApproval),
            requires_confirmation: toBoolean(action.requires_confirmation || action.requiresConfirmation),
            status_constraints: statusConstraints,
            missing_fields: missingFields,
            required_fields: requiredFields.length > 0 ? requiredFields : toStringArray(action.requiredFieldsByType || action.required_fields_by_type),
            invalid_reason: String(action.invalid_reason || action.invalidReason || action.reason || "").trim(),
            runtime_verification: action.runtime_verification || action.runtimeVerification || null,
        };
    }

    function hydrateActionsFromMessage(msg) {
        if (!msg || msg.message_type !== "assistant_actionable") return [];

        const metadataActions = parseStoredActionMetadata(msg.action_metadata);
        if (metadataActions.length > 0) {
            const normalizedMetadataActions = metadataActions.map(normalizeStoredAction).filter(Boolean);
            if (normalizedMetadataActions.length > 0) return normalizedMetadataActions;
        }

        const parsed = parseActionListPayload(msg.action_data);
        const normalized = parsed.map(normalizeStoredAction).filter(Boolean);
        if (normalized.length > 0) return normalized;

        const actionType = String(msg.action_type || "").trim();
        if (!actionType) return [];

        return [
            {
                type: actionType.toLowerCase(),
                target: resolveActionTarget(msg.action_target || ""),
                data: toActionIdValue(msg.action_data),
                label: String(msg.action_label || "").trim() || `Action: ${actionType}`,
            },
        ];
    }

    function summarizeActionData(data) {
        if (!data) return "";
        if (typeof data === "string") return data.trim();
        if (typeof data !== "object") return String(data).trim();

        const obj = data as Record<string, unknown>;
        const pieces: string[] = [];

        const pick = (...keys: string[]) => {
            for (const key of keys) {
                const value = obj[key];
                if (value !== undefined && value !== null && String(value).trim()) {
                    return String(value).trim();
                }
            }
            return "";
        };

        const customer = pick("customer", "customer_name", "customerName", "company", "account");
        const opportunity = pick("opportunity", "opportunity_name", "opportunityName", "title", "subject");
        const date = pick("date", "briefing_date", "day", "for_date", "period", "range");
        const id = pick("id", "offer_id", "offerId", "customer_id", "customerId");

        if (customer) pieces.push(customer);
        if (opportunity) pieces.push(opportunity);
        if (date) pieces.push(date);
        if (id) pieces.push(`ID ${id}`);

        for (const [key, value] of Object.entries(obj)) {
            if (value === undefined || value === null) continue;
            if (typeof value !== "string") continue;
            const trimmed = value.trim();
            if (!trimmed) continue;
            if (["customer", "customer_name", "customerName", "company", "account", "opportunity", "opportunity_name", "opportunityName", "title", "subject", "date", "briefing_date", "day", "for_date", "period", "range", "id", "offer_id", "offerId", "customer_id", "customerId"].includes(key)) {
                continue;
            }
            pieces.push(`${key}: ${trimmed}`);
        }

        return pieces.join(", ");
    }

    function actionLabelOrFallback(action, fallback = "Action") {
        const normalizedLabel = String(action?.label || "").trim();
        if (normalizedLabel) return normalizedLabel;
        return fallback;
    }

    function normalizeEntityIdForAction(action) {
        const { id, idNumeric } = extractEntityId(action);
        return { id, idNumeric };
    }

    function parseStatusPayload(action) {
        const data = getActionDataObject(action);
        return toActionIdValue(
            data.status ||
                data.stage ||
                data.new_status ||
                data.new_stage ||
                data.target_status ||
                data.stage_to ||
                data.approved_to ||
                data.to,
        );
    }

    function normalizeStatusValue(status) {
        return String(status || "").trim();
    }

    function isStatusAllowedForTarget(target, status) {
        const normalizedTarget = String(target || "").toLowerCase();
        const normalizedStatus = normalizeStatusValue(status).toLowerCase().replace(/[-\s]+/g, "");
        if (!normalizedStatus) return true;

        const allowed = STATUS_CONSTRAINTS[normalizedTarget];
        if (!allowed || allowed.length === 0) return true;

        return allowed.some((candidate) => String(candidate).toLowerCase().replace(/[-\s]+/g, "") === normalizedStatus);
    }

    const STATUS_CONSTRAINTS = {
        opportunity: ["New", "Qualified", "Proposal", "Quoted", "Won", "Lost", "On Hold"],
        purchase_order: [
            "Draft",
            "Pending Approval",
            "Approved",
            "Sent",
            "Acknowledged",
            "Partially Received",
            "Received",
            "Closed",
            "Cancelled",
        ],
        order: [
            "Draft",
            "Confirmed",
            "Processing",
            "InProgress",
            "Shipped",
            "PartiallyDelivered",
            "FullyDelivered",
            "Delivered",
            "Invoiced",
            "Complete",
            "Cancelled",
        ],
        rfq: [
            "RFQ Received",
            "Offer Sent",
            "Follow-up/Eval",
            "PO/LOI Received",
            "Order Placed",
            "In Process",
            "Delivered",
            "Closed (Payment)",
            "Closed (Lost)",
        ],
        costing_sheet: ["draft", "pending_approval", "approved", "rejected"],
        offer: ["draft", "quoted", "sent", "accepted", "rejected", "won", "lost"],
        quotation: ["draft", "quoted", "sent", "accepted", "rejected", "won", "lost"],
        follow_up: ["pending", "in_progress", "completed", "cancelled", "overdue"],
        stock_adjustment: ["pending", "approved", "rejected"],
    };

    function getActionWorkflowMode(action, type = "", target = "") {
        const resolvedType = String(type || action?.type || "").toLowerCase();
        const resolvedTarget = String(target || action?.target || "").toLowerCase();
        if (resolvedType === "approve" || resolvedType === "reject") return resolvedType;
        if (resolvedType === "update") return "update";
        if (resolvedType === "create") {
            if (resolvedTarget === "offer") return "create_offer_draft";
            if (resolvedTarget === "follow_up") return "create_follow_up";
            if (resolvedTarget === "order") return "create_order";
            if (resolvedTarget === "stock_adjustment") return "create_stock_adjustment";
            if (resolvedTarget === "quotation") return "create_offer_draft";
            return "create";
        }
        return resolvedType || "workflow";
    }

    function getActionRuntimeState(action) {
        const type = normalizeActionType(action?.type || "");
        const target = resolveActionTarget(action?.target || "");
        const storedStatus = String(action?.execution_status || action?.executionStatus || action?.status || "").trim().toLowerCase();
        const requiredApproval = toBoolean(action?.requires_approval || action?.requiresApproval);
        const metadataMissing = toStringArray(action?.missing_fields || action?.missingFields);
        const invalidReason = String(action?.invalid_reason || action?.invalidReason || "").trim();

        const runtimeMode = getActionWorkflowMode(action, type, target);
        if (["ready_for_execution", "pending_execution", "needs_input", "needs_approval", "invalid_payload"].includes(storedStatus)) {
            return {
                status: storedStatus,
                missing: metadataMissing,
                reason: invalidReason,
                mode: runtimeMode,
                target,
            };
        }

        const validation = validateActionPayload(action, runtimeMode);
        let status = "ready_for_execution";
        if (requiredApproval) status = "needs_approval";
        if (validation.missing.length > 0) status = "needs_input";

        return {
            status,
            missing: metadataMissing.length > 0 ? metadataMissing : validation.missing,
            reason: invalidReason || (validation.missing.length > 0 ? `Missing: ${validation.missing.join(", ")}` : ""),
            mode: runtimeMode,
            target,
        };
    }

    function actionChipStatusLabel(state) {
        switch (state) {
            case "pending_execution":
            case "ready_for_execution":
                return "Ready";
            case "needs_input":
                return "Needs input";
            case "needs_approval":
                return "Needs approval";
            case "invalid_payload":
                return "Invalid";
            default:
                return "Review";
        }
    }

    function validateActionPayload(action: any, mode: string) {
        const target = String(resolveActionTarget(action?.target || "")).toLowerCase();
        const data = getActionDataObject(action);
        const id =
            toActionIdValue(
                data.entity_id ||
                    data.id ||
                    data.offer_id ||
                    data.order_id ||
                    data.purchase_order_id ||
                    data.supplier_invoice_id ||
                    data.costing_sheet_id ||
                    data.stock_adjustment_id,
            ) || toActionIdValue(action?.id);
        const statusValue = parseStatusPayload(action);
        const actionType = String(action?.type || "").toLowerCase();
        const reason = String(data.reason || data.note || data.notes || data.rejection_reason || "").trim();
        const missing = [];
        const allowedStatuses = STATUS_CONSTRAINTS[target] || [];

        if (mode === "approve") {
            if (!id) missing.push("entity id");
        }

        if (mode === "update") {
            if (!id) missing.push("entity id");
            if (target === "opportunity") {
                const comment = String(data.comment || data.notes || data.description || "").trim();
                const ownerNotes = String(data.owner_notes || data.ownerNotes || "").trim();
                if (!statusValue && !comment && !ownerNotes) {
                    missing.push("stage/status or comment/owner_notes");
                } else if (statusValue && !isStatusAllowedForTarget(target, statusValue)) {
                    const expected = allowedStatuses.length > 0 ? `Expected: ${allowedStatuses.join(", ")}` : "";
                    missing.push(`valid status/stage for ${target} ${expected}`);
                }
            } else if (!statusValue) {
                missing.push("status/stage");
            } else if (!isStatusAllowedForTarget(target, statusValue)) {
                const expected = allowedStatuses.length > 0 ? `Expected: ${allowedStatuses.join(", ")}` : "";
                missing.push(`valid status/stage for ${target} ${expected}`);
            }
        }

        const lineItemCount = Array.isArray(data.line_items) ? data.line_items.length : Array.isArray(data.lineItems) ? data.lineItems.length : Array.isArray(data.items) ? data.items.length : 0;
        const amount = parseNumeric(data.grand_total || data.total || data.total_amount || data.amount || data.amount_bhd || data.value, 0);

        if (mode === "create_offer_draft" || mode === "create_offer") {
            if (
                !toActionIdValue(data.customer_id || data.customerId || data.customer_id_text || data.customerId_text) &&
                !toActionIdValue(data.customer_name || data.customerName || data.customer)
            ) {
                missing.push("customer");
            }
            if (!lineItemCount && (!Number.isFinite(amount) || amount <= 0)) {
                missing.push("amount");
            }
        }

        if (mode === "create_follow_up") {
            if (
                !toActionIdValue(data.customer_id || data.customerId || data.customer_id_text || data.customerId_text) &&
                !toActionIdValue(data.customer_name || data.customerName || data.customer)
            ) {
                missing.push("customer");
            }
            if (!toActionIdValue(data.title || data.subject)) {
                missing.push("follow-up title");
            }
        }

        if (mode === "create_order") {
            if (!toActionIdValue(data.order_number || data.orderNumber || data.reference)) {
                missing.push("order number");
            }
            if (!toActionIdValue(data.customer_name || data.customer || data.customer_id || data.customerId || data.customer_id_text)) {
                missing.push("customer");
            }
            const amountValue = parseNumeric(data.amount || data.total_amount || data.totalAmount || data.amount_bhd || data.value, 0);
            if (!Number.isFinite(amountValue) || amountValue <= 0) {
                missing.push("amount");
            }
        }

        if (mode === "create_opportunity") {
            if (!toActionIdValue(data.customer_name || data.customer || data.customer_id || data.customerId || data.customer_id_text)) {
                missing.push("customer");
            }
            if (!toActionIdValue(data.title || data.project || data.opportunity_name || data.name)) {
                missing.push("project/title");
            }
        }

        if (mode === "create_customer_contact") {
            if (!toActionIdValue(data.customer_name || data.customer || data.customer_id || data.customerId || data.customer_id_text)) {
                missing.push("customer");
            }
            if (!toActionIdValue(data.contact_name || data.name || data.person || data.primary_contact)) {
                missing.push("contact name");
            }
        }

        if (mode === "create_supplier_contact") {
            if (!toActionIdValue(data.supplier_name || data.supplier || data.supplier_id || data.supplierId || data.supplier_id_text)) {
                missing.push("supplier");
            }
            if (!toActionIdValue(data.contact_name || data.name || data.person || data.primary_contact)) {
                missing.push("contact name");
            }
        }

        if (mode === "create_stock_adjustment") {
            const inventoryItem = toActionIdValue(
                data.inventory_item_id || data.item_id || data.inventoryItemId || data.itemId || data.item_code || data.inventoryItem,
            );
            const reason = String(data.reason || data.notes || "").trim();
            const variance = parseNumeric(data.variance, NaN);
            const systemQuantity = parseNumeric(data.system_quantity, NaN);
            const physicalQuantity = parseNumeric(data.physical_quantity, NaN);
            if (!inventoryItem) {
                missing.push("inventory item id");
            }
            if (!reason) {
                missing.push("reason");
            }
            if (!Number.isFinite(variance) && (!Number.isFinite(systemQuantity) || !Number.isFinite(physicalQuantity))) {
                missing.push("variance/system_quantity/physical_quantity");
            }
        }

        if (mode === "create" && !target) {
            missing.push("target");
        }

        if (mode === "create") {
            if (target === "offer") {
                if (
                    !toActionIdValue(data.customer_id || data.customerId || data.customer_id_text || data.customerId_text) &&
                    !toActionIdValue(data.customer_name || data.customerName || data.customer)
                ) {
                    missing.push("customer");
                }
                if (!lineItemCount && (!Number.isFinite(amount) || amount <= 0)) {
                    missing.push("amount");
                }
            } else if (target === "follow_up") {
                if (
                    !toActionIdValue(data.customer_id || data.customerId || data.customer_id_text || data.customerId_text) &&
                    !toActionIdValue(data.customer_name || data.customerName || data.customer)
                ) {
                    missing.push("customer");
                }
                if (!toActionIdValue(data.title || data.subject)) {
                    missing.push("follow-up title");
                }
            } else if (target === "order") {
                if (!toActionIdValue(data.order_number || data.orderNumber || data.reference)) {
                    missing.push("order number");
                }
                if (!toActionIdValue(data.customer_name || data.customer || data.customer_id || data.customerId || data.customer_id_text)) {
                    missing.push("customer");
                }
                const amountValue = parseNumeric(data.amount || data.total_amount || data.totalAmount || data.amount_bhd || data.value, 0);
                if (!Number.isFinite(amountValue) || amountValue <= 0) {
                    missing.push("amount");
                }
            } else if (target === "opportunity") {
                if (!toActionIdValue(data.customer_name || data.customer || data.customer_id || data.customerId || data.customer_id_text)) {
                    missing.push("customer");
                }
                if (!toActionIdValue(data.title || data.project || data.opportunity_name || data.name)) {
                    missing.push("project/title");
                }
            } else if (target === "customer_contact" || target === "contact") {
                if (
                    !toActionIdValue(data.customer_name || data.customer || data.customer_id || data.customerId || data.customer_id_text) &&
                    !toActionIdValue(data.supplier_name || data.supplier || data.supplier_id || data.supplierId || data.supplier_id_text)
                ) {
                    missing.push("customer or supplier");
                }
                if (!toActionIdValue(data.contact_name || data.name || data.person || data.primary_contact)) {
                    missing.push("contact name");
                }
            } else if (target === "supplier_contact") {
                if (!toActionIdValue(data.supplier_name || data.supplier || data.supplier_id || data.supplierId || data.supplier_id_text)) {
                    missing.push("supplier");
                }
                if (!toActionIdValue(data.contact_name || data.name || data.person || data.primary_contact)) {
                    missing.push("contact name");
                }
            } else if (target === "stock_adjustment") {
                const inventoryItem = toActionIdValue(
                    data.inventory_item_id || data.item_id || data.inventoryItemId || data.itemId || data.item_code || data.inventoryItem,
                );
                const reason = String(data.reason || data.notes || "").trim();
                const variance = parseNumeric(data.variance, NaN);
                const systemQuantity = parseNumeric(data.system_quantity, NaN);
                const physicalQuantity = parseNumeric(data.physical_quantity, NaN);
                if (!inventoryItem) {
                    missing.push("inventory item id");
                }
                if (!reason) {
                    missing.push("reason");
                }
                if (!Number.isFinite(variance) && (!Number.isFinite(systemQuantity) || !Number.isFinite(physicalQuantity))) {
                    missing.push("variance/system_quantity/physical_quantity");
                }
            }
        }

        return {
            type: actionType,
            target,
            id,
            statusValue,
            reason,
            missing,
            allowedStatuses,
            data,
        };
    }

    function statusLabelForToast(target, candidate) {
        if (!candidate) return "status";
        return `${target} ${candidate}`;
    }

    function requireNumericActionId(idNumeric, label) {
        if (idNumeric === null) {
            throw new Error(`${label} id must be numeric.`);
        }
        return idNumeric;
    }

    function buildWorkflowPrompt(workflow, action) {
        const summary = summarizeActionData(action?.data);

        if (workflow === "create_offer_draft") {
            return summary
                ? `Create an offer draft for ${summary}. Include recommended line items, commercial terms, and flag any missing details before finalizing.`
                : "Create an offer draft. Use the current Butler context and ask for any missing customer, product, quantity, or pricing details before finalizing.";
        }

        if (workflow === "create_follow_up") {
            return summary
                ? `Create this follow-up from Butler data: ${summary}.`
                : "Create a follow-up task with a clear customer, title, and due date.";
        }

        if (workflow === "create_order") {
            return summary
                ? `Create an order for ${summary}. Validate required fields and confirmation details before saving.`
                : "Create an order from Butler data with required order number, customer, and amount.";
        }

        if (workflow === "create_opportunity") {
            return summary
                ? `Create an opportunity for ${summary}. Ask for any missing customer, project, or reference details before finalizing.`
                : "Create an opportunity from Butler data with a customer and project/title. Ask for missing details before finalizing.";
        }

        if (workflow === "create_customer_contact") {
            return summary
                ? `Create a customer contact for ${summary}. Confirm the customer and contact details before saving.`
                : "Create a customer contact from Butler data. I need the customer and contact details.";
        }

        if (workflow === "create_supplier_contact" || workflow === "create_contact") {
            return summary
                ? `Create a contact for ${summary}. Confirm the parent company and contact details before saving.`
                : "Create a supplier or customer contact from Butler data. I need the parent company and contact details.";
        }

        if (workflow === "create_stock_adjustment") {
            return summary
                ? `Create a stock adjustment for ${summary}.`
                : "Create a stock adjustment action from Butler data. I need inventory item, reason, and quantity variance.";
        }

        if (workflow === "daily_briefing") {
            return summary
                ? `Generate the daily briefing for ${summary}. Summarize priorities, risks, and next actions in a concise briefing format.`
                : "Generate today's daily briefing. Summarize priorities, risks, and next actions in a concise briefing format.";
        }

        return summary ? `Proceed with ${workflow} for ${summary}.` : `Proceed with ${workflow.replace(/_/g, " ")}.`;
    }

    function getAppBridge() {
        if (typeof window === "undefined") return null;
        return window.go?.main?.App || null;
    }

    async function invokeAppBridge(method, ...args) {
        const app = getAppBridge();
        if (!app || typeof app[method] !== "function") {
            throw new Error(`Butler action '${method}' is not available in this build yet.`);
        }
        return await app[method](...args);
    }

    function getActionDataObject(action: any): Record<string, any> {
        return typeof action?.data === "object" && action.data ? action.data : {};
    }

    function buildOfferDraftPayload(action: any = {}) {
        const data = getActionDataObject(action);
        const rawLineItems = Array.isArray(data.line_items)
            ? data.line_items
            : Array.isArray(data.lineItems)
                ? data.lineItems
                : Array.isArray(data.items)
                    ? data.items
                    : [];

        // Map line items to match Go ButlerOfferDraftLineItem json tags
        const line_items = rawLineItems.map((item) => ({
            equipment: item.equipment || item.description || "",
            description: item.description || item.equipment || "",
            model: item.model || "",
            specification: item.specification || "",
            quantity: parseInt(item.quantity || item.qty || 1, 10),
            unit_price_bhd: parseFloat(item.unit_price_bhd || item.unit_price || item.unitPrice || item.unitPriceBHD || 0),
            optional: item.optional || false,
        }));

        // Keys MUST match Go ButlerOfferDraftRequest json tags (snake_case)
        return {
            customer_id: data.customer_id || data.customerId || "",
            customer_name: data.customer_name || data.customerName || data.customer || action.target || "",
            quote_type: data.quote_type || data.quoteType || "Quotation",
            vat_rate: data.vat_rate ?? data.vatRate ?? 10,
            payment_terms: data.payment_terms || data.paymentTerms || "",
            delivery_terms: data.delivery_terms || data.deliveryTerms || "",
            est_delivery: data.est_delivery || data.estDelivery || "",
            contact_person: data.contact_person || data.contactPerson || "",
            rfq_reference: data.rfq_reference || data.rfqReference || "",
            prepared_by: data.prepared_by || data.preparedBy || "",
            country_of_origin: data.country_of_origin || data.countryOfOrigin || "",
            division: data.division || "",
            line_items,
        };
    }

    function buildOpportunityPayload(action: any = {}) {
        const data = getActionDataObject(action);
        return {
            customer_name: data.customer_name || data.customerName || data.customer || "",
            customer_id: data.customer_id || data.customerId || data.customer_id_text || "",
            project: data.project || data.title || data.opportunity_name || data.name || "",
            reference: data.reference || data.folder_number || data.rfq_reference || data.rfq_number || "",
            value: parseNumeric(data.amount ?? data.total_amount ?? data.totalAmount ?? data.amount_bhd ?? data.value, 0),
            notes: data.notes || data.comment || data.description || "",
        };
    }

    function buildCustomerContactPayload(action: any = {}) {
        const data = getActionDataObject(action);
        return {
            customer_id: toActionIdValue(data.customer_id || data.customerId || data.customer_id_text || ""),
            contact_name: data.contact_name || data.name || data.person || data.primary_contact || "",
            job_title: data.job_title || data.jobTitle || data.title || "",
            email: data.email || data.primary_email || "",
            phone: data.phone || data.primary_phone || "",
            address: data.address || data.address_line1 || "",
            is_primary_contact: toBoolean(data.is_primary_contact || data.isPrimaryContact),
        };
    }

    function buildSupplierContactPayload(action: any = {}) {
        const data = getActionDataObject(action);
        return {
            supplier_id: toActionIdValue(data.supplier_id || data.supplierId || data.supplier_id_text || ""),
            contact_name: data.contact_name || data.name || data.person || data.primary_contact || "",
            job_title: data.job_title || data.jobTitle || data.title || "",
            email: data.email || "",
            phone: data.phone || data.mobile || "",
            address: data.address || data.address_line1 || "",
            is_primary_contact: toBoolean(data.is_primary_contact || data.isPrimaryContact),
        };
    }

    function buildFollowUpPayload(action: any = {}) {
        const data = getActionDataObject(action);
        const target = String(action?.target || "").trim();
        const dueDateRaw = String(data.due_date || data.dueDate || "").trim();
        const normalizedPriority = String(data.priority || "medium")
            .toLowerCase()
            .trim();
        const allowedPriority = ["low", "medium", "high", "urgent"].includes(normalizedPriority)
            ? normalizedPriority
            : "medium";

        return {
            customer_id: toActionIdValue(data.customer_id || data.customerId || data.customer_id_text || data.customerId_text || data.customer || ""),
            customerId: toActionIdValue(data.customer_id || data.customerId || data.customer_id_text || data.customerId_text || data.customer || ""),
            title: data.title || data.subject || `Butler follow-up${target ? `: ${target}` : ""}`,
            description: data.description || data.notes || data.message || "",
            notes: data.notes || data.description || "",
            contact: data.contact || data.contactPerson || data.person || "",
            due_date: dueDateRaw || new Date().toISOString().split("T")[0],
            dueDate: dueDateRaw || new Date().toISOString().split("T")[0],
            priority: allowedPriority,
            status: "pending",
            type: data.type || "Follow-up",
            source: "Butler",
        };
    }

    function buildStockAdjustmentPayload(action: any = {}) {
        const data = getActionDataObject(action);
        const systemQuantity = parseNumeric(data.system_quantity, NaN);
        const physicalQuantity = parseNumeric(data.physical_quantity, NaN);
        const variance = parseNumeric(data.variance, NaN);
        const itemId = toActionIdValue(data.inventory_item_id || data.item_id || data.itemId || data.inventoryItemId || data.item_code);
        const computedVariance = Number.isFinite(variance) ? variance : Number.isFinite(systemQuantity) && Number.isFinite(physicalQuantity) ? physicalQuantity - systemQuantity : 0;

        return {
            InventoryItemID: itemId,
            inventory_item_id: itemId,
            adjustment_type: data.adjustment_type || data.adjustmentType || "physical_count",
            AdjustmentType: data.adjustment_type || data.adjustmentType || "physical_count",
            reason: data.reason || data.note || "",
            Reason: data.reason || data.note || "",
            variance: computedVariance,
            Variance: computedVariance,
            system_quantity: Number.isFinite(systemQuantity) ? systemQuantity : 0,
            physical_quantity: Number.isFinite(physicalQuantity) ? physicalQuantity : 0,
            SystemQuantity: Number.isFinite(systemQuantity) ? systemQuantity : 0,
            PhysicalQuantity: Number.isFinite(physicalQuantity) ? physicalQuantity : 0,
            unit_cost: parseNumeric(data.unit_cost, 0),
            unitCost: parseNumeric(data.unit_cost, 0),
            notes: data.notes || data.note || "",
            source: "Butler",
        };
    }

    async function executeOfferDraftAction(action) {
        const payload = buildOfferDraftPayload(action);

        try {
            const result = await invokeAppBridge("CreateOfferDraftFromButler", payload);
            const offerRef =
                result?.OfferNumber ||
                result?.offerNumber ||
                result?.OfferID ||
                result?.offerID ||
                result?.id ||
                payload.customer_name ||
                "offer draft";
            addMessage("butler", `Created ${offerRef} from the Butler action.`);
        } catch (err) {
            addMessage("butler", `I couldn't create the offer draft: ${String(err)}`);
        }
    }

    async function executeFollowUpAction(action) {
        const payload = buildFollowUpPayload(action);

        try {
            const result = await invokeAppBridge("CreateFollowUp", payload);
            const ref = result?.id || result?.ID || result?.followUpId || result?.FollowUpID;
            addMessage("butler", ref ? `Created follow-up #${ref}.` : "Created the follow-up task.");
        } catch (err) {
            addMessage("butler", `I couldn't create the follow-up: ${String(err)}`);
        }
    }

    async function executeCreateStockAdjustmentAction(action) {
        const payload: any = buildStockAdjustmentPayload(action);
        const validated = validateActionPayload(action, "create_stock_adjustment");
        const summary = summarizeActionData(payload);
        const itemId = payload.inventory_item_id || payload.InventoryItemID || payload.item_id;
        const reason = payload.reason || "";
        const inventoryDisplay = itemId || "unknown inventory item";
        if (!itemId || !reason || !Number.isFinite(payload.variance) || validated.missing.length > 0) {
            const missing = validated.missing.length > 0 ? validated.missing.join(", ") : "required stock adjustment details";
            addMessage("butler", `I can create this stock adjustment only with: ${missing}.`);
            return;
        }

        try {
            await invokeAppBridge("CreateStockAdjustment", payload);
            addMessage("butler", "Created stock adjustment from Butler action.");
        } catch (err) {
            addMessage("butler", `I couldn't create the stock adjustment: ${String(err)}`);
        }
    }

    async function executeCreateCustomerAction(action) {
        const data = getActionDataObject(action);
        const payload = {
            business_name: data.business_name || data.businessName || data.customer_name || data.name || "",
            customer_type: data.customer_type || data.customerType || "Corporate",
            payment_grade: data.payment_grade || data.paymentGrade || "B",
            city: data.city || "",
            country: data.country || "Bahrain",
            primary_contact: data.primary_contact || data.contact || "",
            primary_email: data.primary_email || data.email || "",
            primary_phone: data.primary_phone || data.phone || "",
            mobile_number: data.mobile_number || data.mobileNumber || "",
            industry: data.industry || "",
            address_line1: data.address_line1 || data.address || "",
            trn: data.trn || "",
        };
        if (!payload.business_name) {
            addMessage("butler", "I need a business name to create a customer. Please provide the company name.");
            return;
        }
        try {
            const result = await invokeAppBridge("CreateCustomerFromButler", payload);
            addMessage("butler", `Created customer: ${result?.business_name || payload.business_name} (Code: ${result?.customer_code || "auto"})`);
        } catch (err) {
            addMessage("butler", `Couldn't create customer: ${String(err)}`);
        }
    }

    async function executeCreateSupplierAction(action) {
        const data = getActionDataObject(action);
        const payload = {
            supplier_name: data.supplier_name || data.supplierName || data.name || "",
            supplier_type: data.supplier_type || data.supplierType || "Manufacturer",
            country: data.country || "Bahrain",
            primary_contact: data.primary_contact || data.contact || "",
            email: data.email || "",
            phone: data.phone || "",
            address: data.address || "",
            tax_id: data.tax_id || data.taxId || data.trn || "",
            brands_handled: data.brands_handled || data.brandsHandled || data.brands || "",
            lead_time_days: data.lead_time_days || data.leadTimeDays || 0,
        };
        if (!payload.supplier_name) {
            addMessage("butler", "I need a supplier name to create a supplier. Please provide the company name.");
            return;
        }
        try {
            const result = await invokeAppBridge("CreateSupplierFromButler", payload);
            addMessage("butler", `Created supplier: ${result?.supplier_name || payload.supplier_name} (Code: ${result?.supplier_code || "auto"})`);
        } catch (err) {
            addMessage("butler", `Couldn't create supplier: ${String(err)}`);
        }
    }

    function extractEntityId(action) {
        const data = getActionDataObject(action);
        const idValue =
            toActionIdValue(data.id) ||
            toActionIdValue(data.entity_id) ||
            toActionIdValue(data.offer_id) ||
            toActionIdValue(data.order_id) ||
            toActionIdValue(data.purchase_order_id) ||
            toActionIdValue(data.supplier_invoice_id) ||
            toActionIdValue(data.invoice_id) ||
            toActionIdValue(data.costing_sheet_id) ||
            toActionIdValue(data.stock_adjustment_id);

        return { data, id: idValue, idNumeric: toNumericId(idValue) };
    }

    async function executeUpdateAction(action) {
        const target = String(resolveActionTarget(action?.target || "")).toLowerCase();
        const { id, idNumeric } = normalizeEntityIdForAction(action);
        const actionType = String(action?.type || "").toLowerCase();
        const statusValue = parseStatusPayload(action);

        if (!id) {
            addMessage("butler", `This ${actionType} action needs an entity id.`);
            return;
        }

        if (
            (target === "order" || target === "rfq" || target === "offer" || target === "costing_sheet") &&
            !statusValue
        ) {
            addMessage("butler", `I need a status or stage value to update ${target} ${id}.`);
            return;
        }

        try {
            if (target === "purchase_order") {
                const status = statusValue || "Cancelled";
                if (actionType === "approve") {
                    const approver = "Butler";
                    await invokeAppBridge("ApprovePurchaseOrder", String(id), approver);
                    addMessage("butler", `Approved purchase order #${id}.`);
                    return;
                }

                if (actionType === "reject") {
                    await invokeAppBridge("UpdatePOStatus", String(id), status || "Cancelled");
                    addMessage("butler", `Updated purchase order #${id} to ${status}.`);
                    return;
                }

                if (status) {
                    await invokeAppBridge("UpdatePOStatus", String(id), status);
                    addMessage("butler", `Updated purchase order #${id} status to ${status}.`);
                    return;
                }
            }

            if (target === "opportunity") {
                const data = getActionDataObject(action);
                const comment = String(data.comment || data.notes || data.description || "").trim();
                const ownerNotes = String(data.owner_notes || data.ownerNotes || "").trim();

                if (!statusValue && !comment && !ownerNotes) {
                    addMessage("butler", `I need a stage/status or note update to modify opportunity ${id}.`);
                    return;
                }

                if (statusValue) {
                    await invokeAppBridge("UpdateOpportunityStage", String(id), statusValue);
                }
                if (comment || ownerNotes) {
                    await invokeAppBridge("UpdateOpportunityDetails", String(id), comment, ownerNotes);
                }

                if (statusValue && (comment || ownerNotes)) {
                    addMessage("butler", `Updated opportunity ${id} stage to ${statusValue} and refreshed its notes.`);
                } else if (statusValue) {
                    addMessage("butler", `Updated opportunity ${id} stage to ${statusValue}.`);
                } else {
                    addMessage("butler", `Updated opportunity ${id} notes.`);
                }
                return;
            }

            if (target === "order") {
                requireNumericActionId(idNumeric, "Order");
                await invokeAppBridge("UpdateOrderStage", idNumeric, statusValue);
                addMessage("butler", `Updated order #${id} to stage ${statusValue}.`);
                return;
            }

            if (target === "rfq") {
                requireNumericActionId(idNumeric, "RFQ");
                await invokeAppBridge("UpdateRFQStage", idNumeric, statusValue);
                addMessage("butler", `Updated RFQ #${id} to stage ${statusValue}.`);
                return;
            }

            if (target === "costing_sheet") {
                requireNumericActionId(idNumeric, "Costing sheet");
                await invokeAppBridge("UpdateCostingSheet", idNumeric, { id: idNumeric, status: statusValue });
                addMessage("butler", `Updated costing sheet #${id} status to ${statusValue}.`);
                return;
            }

            if (target === "offer") {
                requireNumericActionId(idNumeric, "Offer");
                await invokeAppBridge("UpdateOfferStatus", idNumeric, statusValue);
                addMessage("butler", `Updated offer #${id} status to ${statusValue}.`);
                return;
            }

            if (target === "quotation") {
                requireNumericActionId(idNumeric, "Quotation");
                await invokeAppBridge("UpdateOfferStatus", idNumeric, statusValue);
                addMessage("butler", `Updated quotation #${id} status to ${statusValue}.`);
                return;
            }

            if (target === "stock_adjustment") {
                requireNumericActionId(idNumeric, "Stock adjustment");
                if (statusValue.toLowerCase() === "approved") {
                    await invokeAppBridge("ApproveStockAdjustment", idNumeric);
                    addMessage("butler", `Approved stock adjustment #${id}.`);
                    return;
                }
                addMessage("butler", `Stock adjustment update only supports approval action for now. This status change to "${statusValue}" is not supported.`);
                return;
            }

            addMessage("butler", `No update execution path is configured for target '${target}'.`);
        } catch (err) {
            addMessage("butler", `I couldn't perform that update: ${String(err)}`);
        }
    }

    async function executeCreateOrderAction(action) {
        const data = getActionDataObject(action);
        const customerIdentity = await resolveActionCustomerIdentity(action);
        const orderNumber =
            toActionIdValue(data.order_number || data.orderNumber || data.reference || data.order_no || `ORD-${Date.now()}`).trim();
        const amount = parseNumeric(data.amount ?? data.total_amount ?? data.totalAmount ?? data.amount_bhd, Number.NaN);
        const customerId = customerIdentity.customerId;
        const customerName = customerIdentity.customerName || toActionIdValue(data.customer_name || data.customerName || data.customer || data.contact || data.client || "");
        const resolvedCustomerName = customerName || (await resolveCustomerNameFromId(customerId)) || "";

        if (!orderNumber || Number.isNaN(amount) || amount <= 0 || (!customerName && !customerId)) {
            addMessage(
                "butler",
                "I can create this order only with an order number, amount and customer (name or id). Please add missing fields and run the action again.",
            );
            return;
        }

        if (customerId && !resolvedCustomerName) {
            addMessage(
                "butler",
                "I found a customer id but could not resolve the customer name. Please refresh customer context and regenerate the action so this order links correctly.",
            );
            return;
        }

        try {
            const orderDate = normalizeDueDate(data.order_date || data.orderDate || "", "");
            const result = await invokeAppBridge(
                "CreateOrder",
                orderNumber,
                resolvedCustomerName || customerName || `Customer ${customerId}`,
                amount,
                orderDate,
                data.status || "Pending",
            );
            const ref = result?.order_number || result?.OrderNumber || result?.id || orderNumber;
            addMessage("butler", `Created order ${ref} from the Butler action.`);
        } catch (err) {
            addMessage("butler", `I couldn't create that order: ${String(err)}`);
        }
    }

    async function executeCreateOpportunityAction(action) {
        const payload = buildOpportunityPayload(action);
        const customerIdentity = await resolveActionCustomerIdentity(action);
        const customerName = customerIdentity.customerName || payload.customer_name;
        const project = String(payload.project || "").trim();
        const reference = String(payload.reference || "").trim();

        if (!customerName || !project) {
            addMessage("butler", "I can create this opportunity only with a customer and project/title. Please provide the missing details.");
            return;
        }

        try {
            const duplicate = await invokeAppBridge("CheckDuplicateOpportunity", reference, customerName, project);
            if (duplicate && (duplicate.id || duplicate.ID)) {
                const existingRef = duplicate.folder_number || duplicate.rfq_number || duplicate.id || duplicate.ID;
                const existingStage = duplicate.stage || "New";
                addMessage(
                    "butler",
                    `I found an existing opportunity instead of creating a duplicate: ${existingRef} | ${duplicate.customer_name || customerName} | ${duplicate.title || duplicate.folder_name || project} | ${existingStage}.`,
                );
                return;
            }

            // Wave 9.6: CreateRFQ gained a 5th productDetails arg (Sa2). Butler has no
            // structured line items to seed, so pass "" — keeps the binding arity correct.
            const result = await invokeAppBridge("CreateRFQ", customerName, project, payload.value || 0, payload.notes || "", "");
            const ref = result?.rfq_number || result?.RFQNumber || result?.id || project;
            addMessage("butler", `Created opportunity ${ref} for ${customerName}.`);
        } catch (err) {
            addMessage("butler", `I couldn't create that opportunity: ${String(err)}`);
        }
    }

    async function executeCreateCustomerContactAction(action) {
        const payload: any = buildCustomerContactPayload(action);
        const customerIdentity = await resolveActionCustomerIdentity(action);
        const customerId = customerIdentity.customerId || payload.customer_id;
        const customerName = customerIdentity.customerName;

        if (!customerId && !customerName) {
            addMessage("butler", "I need the customer record before I can add this contact. Please specify the customer.");
            return;
        }
        if (!payload.contact_name) {
            addMessage("butler", "I need the contact name before I can create this customer contact.");
            return;
        }

        payload.customer_id = customerId;
        if (!payload.customer_id && customerName) {
            payload.customer_id = await resolveCustomerIdFromHint(customerName);
        }
        if (!payload.customer_id) {
            addMessage("butler", `I couldn't resolve the customer record for ${customerName || "this contact"}. Please clarify the customer.`);
            return;
        }

        try {
            await invokeAppBridge("AddCustomerContact", payload);
            addMessage("butler", `Added ${payload.contact_name} to ${customerName || "the customer"} contacts.`);
        } catch (err) {
            addMessage("butler", `I couldn't create the customer contact: ${String(err)}`);
        }
    }

    async function executeCreateSupplierContactAction(action) {
        const payload: any = buildSupplierContactPayload(action);
        const supplierIdentity = await resolveActionSupplierIdentity(action);
        const supplierId = supplierIdentity.supplierId || payload.supplier_id;
        const supplierName = supplierIdentity.supplierName;

        if (!supplierId && !supplierName) {
            addMessage("butler", "I need the supplier record before I can add this contact. Please specify the supplier.");
            return;
        }
        if (!payload.contact_name) {
            addMessage("butler", "I need the contact name before I can create this supplier contact.");
            return;
        }

        payload.supplier_id = supplierId;
        if (!payload.supplier_id && supplierName) {
            payload.supplier_id = await resolveSupplierIdFromHint(supplierName);
        }
        if (!payload.supplier_id) {
            addMessage("butler", `I couldn't resolve the supplier record for ${supplierName || "this contact"}. Please clarify the supplier.`);
            return;
        }

        try {
            await invokeAppBridge("AddSupplierContact", payload);
            addMessage("butler", `Added ${payload.contact_name} to ${supplierName || "the supplier"} contacts.`);
        } catch (err) {
            addMessage("butler", `I couldn't create the supplier contact: ${String(err)}`);
        }
    }

    async function executeGenericApprovalAction(action) {
        const actionType = String(action?.type || "").toLowerCase();
        const validated = validateActionPayload(action, "approve");
        const target = validated.target;
        const { id } = validated;
        const reason = validated.reason || "No reason provided";
        const approver = "Butler";

        if (!id) {
            addMessage("butler", `This ${actionType} action is missing entity id for ${target}.`);
            return;
        }

        if (validated.missing.length > 0) {
            addMessage("butler", `I need: ${validated.missing.join(", ")} to execute this ${actionType} action.`);
            return;
        }

        if (target === "purchase_order" && id) {
            if (actionType === "approve") {
                await invokeAppBridge("ApprovePurchaseOrder", id, approver);
                addMessage("butler", `Approved purchase order #${id}.`);
            } else {
                const status = parseStatusPayload(action) || "Cancelled";
                await invokeAppBridge("UpdatePOStatus", id, status);
                addMessage("butler", `Updated purchase order #${id} to ${status}.`);
            }
            return;
        }

        if (target === "order") {
            const status = actionType === "approve" ? "Confirmed" : "Cancelled";
            await invokeAppBridge("UpdateOrderStage", id, status);
            addMessage("butler", `${statusLabelForToast(target, status)} actioned.`);
            return;
        }

        if (target === "offer") {
            if (actionType === "approve") {
                // Wave 9.6: MarkOfferWon's 2nd arg is the customer PO number (required,
                // persisted onto the Order, part of its idempotency key) — NOT an approver.
                // Refuse rather than write the literal "Butler" as a customer PO.
                const customerPO = String(
                    action?.data?.customer_po || action?.data?.customerPO || validated?.data?.customer_po || "",
                ).trim();
                if (!customerPO) {
                    addMessage("butler", `I need the customer PO number to mark offer #${id} as won.`);
                    return;
                }
                await invokeAppBridge("MarkOfferWon", id, customerPO);
                addMessage("butler", `Marked offer #${id} as won.`);
            } else {
                await invokeAppBridge("MarkOfferLost", id, reason || "Lost from Butler action");
                addMessage("butler", `Marked offer #${id} as lost.`);
            }
            return;
        }

        if (target === "supplier_invoice") {
            if (actionType === "approve") {
                await invokeAppBridge("ApproveSupplierInvoice", id, approver);
                addMessage("butler", `Approved supplier invoice #${id}.`);
            } else {
                await invokeAppBridge("DisputeSupplierInvoice", id, reason || "Disputed from Butler");
                addMessage("butler", `Disputed supplier invoice #${id}.`);
            }
            return;
        }

        if (target === "rfq") {
            if (actionType === "approve") {
                await invokeAppBridge("UpdateRFQStage", toNumericId(id), "Closed (Payment)");
            } else {
                await invokeAppBridge("UpdateRFQStage", toNumericId(id), "Closed (Lost)");
            }
            addMessage("butler", `RFQ #${id} marked ${actionType === "approve" ? "Closed (Payment)" : "Closed (Lost)"}.`);
            return;
        }

        if (target === "stock_adjustment") {
            if (actionType === "approve") {
                const adjustmentId = toNumericId(id);
                if (adjustmentId === null) {
                    addMessage("butler", "I need a valid stock adjustment id to approve.");
                    return;
                }
                await invokeAppBridge("ApproveStockAdjustment", adjustmentId);
                addMessage("butler", `Approved stock adjustment #${id}.`);
            } else {
                addMessage("butler", "Stock adjustment rejection is not supported yet. Re-map to a supported status update.");
            }
            return;
        }

        if (target === "costing_sheet") {
            const { idNumeric } = extractEntityId(action);
            if (actionType === "approve") {
                if (idNumeric === null) throw new Error("Invalid costing sheet id");
                await invokeAppBridge("ApproveCostingSheet", idNumeric, approver);
                addMessage("butler", `Approved costing sheet #${id}.`);
            } else {
                if (idNumeric === null) throw new Error("Invalid costing sheet id");
                await invokeAppBridge("RejectCostingSheet", idNumeric, reason || "Rejected from Butler");
                addMessage("butler", `Rejected costing sheet #${id}.`);
            }
            return;
        }

        addMessage("butler", `This Butler action is recognized, but the execution path is not mapped for target '${target}'.`);
    }

    async function executeCreateAction(action) {
        const target = String(resolveActionTarget(action?.target || action?.target_type || "")).toLowerCase();

        if (target === "follow_up" || target === "followup" || target === "follow-up") {
            return executeFollowUpAction(action);
        }

        if (target === "offer" || target === "offer_draft") {
            return executeOfferDraftAction(action);
        }

        if (target === "quotation") {
            return executeOfferDraftAction(action);
        }

        if (target === "order" || target === "orders") {
            return executeCreateOrderAction(action);
        }

        if (target === "opportunity" || target === "opportunities") {
            return executeCreateOpportunityAction(action);
        }

        if (target === "customer_contact") {
            return executeCreateCustomerContactAction(action);
        }

        if (target === "supplier_contact") {
            return executeCreateSupplierContactAction(action);
        }

        if (target === "contact" || target === "contacts") {
            const data = getActionDataObject(action);
            if (toActionIdValue(data.supplier_id || data.supplierId || data.supplier_name || data.supplier || data.vendor)) {
                return executeCreateSupplierContactAction(action);
            }
            return executeCreateCustomerContactAction(action);
        }

        if (target === "stock_adjustment") {
            return executeCreateStockAdjustmentAction(action);
        }

        if (target === "customer" || target === "customers") {
            return executeCreateCustomerAction(action);
        }

        if (target === "supplier" || target === "suppliers") {
            return executeCreateSupplierAction(action);
        }

        addMessage(
            "butler",
            `I can create offer drafts, follow-ups, orders, opportunities, contacts, stock adjustments, customers, and suppliers from Butler actions. Please refine this action with the required details.`,
        );
    }

    async function runWorkflowAction(workflow, action = {}) {
        const prompt = buildWorkflowPrompt(workflow, action);
        if (!prompt) return;

        userInput = prompt;
        await send();
    }

    function isWriteAction(action) {
        const type = normalizeActionType(action?.type || "");
        return type === "create" || type === "update" || type === "approve" || type === "reject";
    }

    function actionExecutionKey(action) {
        return JSON.stringify({
            type: normalizeActionType(action?.type || ""),
            target: resolveActionTarget(action?.target || ""),
            label: actionLabelOrFallback(action, "Action"),
            data: getActionDataObject(action),
        });
    }

    function clearPendingActionArm() {
        pendingActionKey = "";
        if (pendingActionTimer) {
            clearTimeout(pendingActionTimer);
            pendingActionTimer = null;
        }
    }

    function armActionConfirmation(action) {
        clearPendingActionArm();
        pendingActionKey = actionExecutionKey(action);
        pendingActionTimer = setTimeout(() => {
            pendingActionKey = "";
            pendingActionTimer = null;
        }, ACTION_CONFIRMATION_WINDOW_MS);
    }

    function isActionArmed(action) {
        return pendingActionKey !== "" && pendingActionKey === actionExecutionKey(action);
    }

    function buildActionPreview(action) {
        const state = getActionRuntimeState(action);
        const type = normalizeActionType(action?.type || "");
        const target = resolveActionTarget(action?.target || "");
        const summary = summarizeActionData(getActionDataObject(action));
        const intro =
            type === "create"
                ? "I'm ready to create this record."
                : type === "update"
                    ? "I'm ready to update this record."
                    : type === "approve"
                        ? "I'm ready to approve this action."
                        : type === "reject"
                            ? "I'm ready to reject this action."
                            : "I'm ready to run this Butler action.";
        const details = summary ? ` Details: ${summary}.` : "";
        const missing = state.missing?.length ? ` Missing: ${state.missing.join(", ")}.` : "";
        return `${intro} ${actionLabelOrFallback(action, target || "Action")}.${details}${missing} Click the same action again to confirm.`;
    }

    async function send() {
        if (loading) return; // Prevent double-send (key repeat creates duplicate conversations)
        if (!userInput.trim()) return;
        clearPendingActionArm();
        const txt = userInput;
        userInput = "";
        addMessage("user", txt);
        loading = true;

        try {
            if (!window.go) {
                await new Promise((r) => setTimeout(r, 1000));
                addMessage("butler", "I have noted your request. Is there anything else?");
            } else {
                // Use persistent chat if we have a conversation
                const res = await ChatWithButlerPersistent(activeConversationId, txt);
                // Set conversation ID IMMEDIATELY to prevent duplicate conversation creation
                if (res?.conversation_id && !activeConversationId) {
                    activeConversationId = res.conversation_id;
                    saveActiveConversationId();
                }
                if (res && res.response) {
                    addMessage("butler", res.response, res.actions || []);
                    // Refresh sidebar (non-blocking — don't await to avoid delaying next message)
                    loadConversations();
                }
                if (res?.confidence) confidence = res.confidence;
            }
        } catch (e) {
            // Retry on a fresh persistent conversation before giving up.
            try {
                const res = await ChatWithButlerPersistent("", txt);
                if (res && res.response) {
                    addMessage("butler", res.response, res.actions || []);
                    if (res.conversation_id) {
                        activeConversationId = res.conversation_id;
                        saveActiveConversationId();
                        await loadConversations();
                    }
                }
            } catch (persistentRetryErr) {
                try {
                    const fallback = await ChatWithButler(txt);
                    if (fallback && fallback.message) {
                        addMessage("butler", fallback.message, fallback.actions || []);
                    } else {
                        addMessage("butler", `Butler could not complete this request. Error: ${String(persistentRetryErr || e)}`);
                    }
                } catch (fallbackErr) {
                    addMessage(
                        "butler",
                        `Butler could not complete this request. Persistent error: ${String(persistentRetryErr || e)}. Fallback error: ${String(fallbackErr)}`
                    );
                }
            }
        } finally {
            loading = false;
        }
    }

    async function loadConversations() {
        try {
            const loaded = await ListConversations() || [];
            conversations = loaded.filter((c: any) => !hiddenConversationKeys.has(convKey(c)));
        } catch {
            conversations = [];
        }
    }

    async function selectConversation(conv: any) {
        const id = convId(conv);
        if (!id) return;
        clearPendingActionArm();
        activeConversationId = id;
        saveActiveConversationId();
        loadingConversation = true;
        try {
            const msgs = await GetConversationMessages(id);
            const loadedMessages = [];
            if (msgs && msgs.length > 0) {
                for (const rawMessage of msgs as any[]) {
                    const m: any = rawMessage;
                    const role = String(m?.role || "").toLowerCase();
                    const text = m?.content ?? "";
                    const messageId = m?.id || "";
                    const actions = role === "assistant" ? hydrateActionsFromMessage(m) : [];
                    const renderedText = String(text).trim() === "" && actions.length > 0
                        ? "Action details available in this historical message."
                        : String(text);
                    loadedMessages.push({
                        id: String(messageId || nextLocalMessageId()),
                        role: role === "user" ? "user" : "butler",
                        text: renderedText,
                        actions,
                    });
                }
            }
            messages = loadedMessages.length > 0
                ? loadedMessages
                : [{ id: `empty-${id}`, role: "butler", text: "This conversation has no persisted messages yet.", actions: [] }];
            scrollToBottom();
        } catch {
            messages = [{ id: `error-${id}`, role: "butler", text: "Failed to load conversation history.", actions: [] }];
        } finally {
            loadingConversation = false;
        }
    }

    async function startNewConversation(showGreeting = true) {
        clearPendingActionArm();
        activeConversationId = "";
        messages = [];
        saveActiveConversationId();
        if (showGreeting) {
            addMessage("butler", "Good day. How can I help you?");
        }
        await loadConversations();
    }

    // Double-click-to-confirm state (window.confirm doesn't work in Wails WebView)
    let pendingDeleteKey = $state("");
    let pendingDeleteTimer: any = null;
    let pendingClearAll = $state(false);
    let pendingClearAllTimer: any = null;
    let deletingConvId = "";

    function armDelete(conv: any) {
        const key = convKey(conv);
        if (pendingDeleteKey === key) {
            // Second click — confirmed
            clearTimeout(pendingDeleteTimer);
            pendingDeleteKey = "";
            doDeleteConv(conv);
        } else {
            // First click — arm
            clearTimeout(pendingDeleteTimer);
            pendingDeleteKey = key;
            pendingDeleteTimer = setTimeout(() => { pendingDeleteKey = ""; }, 3000);
        }
    }

    async function doDeleteConv(conv: any) {
        const id = convId(conv);
        const identifier = id || convTitle(conv);
        if (!identifier || deletingConvId === id) return;
        deletingConvId = id;
        try {
            await DeleteConversation(identifier);
            hiddenConversationKeys.add(convKey(conv));
            saveHiddenConversationKeys();
            conversations = conversations.filter((c) => convKey(c) !== convKey(conv));
            if (activeConversationId === id) {
                startNewConversation();
            }
            await loadConversations();
        } catch (err) {
            addMessage("butler", `Delete failed: ${String(err)}`);
        } finally {
            deletingConvId = "";
        }
    }

    function armClearAll() {
        if (pendingClearAll) {
            clearTimeout(pendingClearAllTimer);
            pendingClearAll = false;
            doClearAll();
        } else {
            pendingClearAll = true;
            pendingClearAllTimer = setTimeout(() => { pendingClearAll = false; }, 3000);
        }
    }

    async function doClearAll() {
        try {
            await PurgeAllConversations();
            hiddenConversationKeys.clear();
            saveHiddenConversationKeys();
            conversations = [];
            activeConversationId = "";
            messages = [];
            await loadConversations();
            addMessage("butler", "All conversations have been cleared.");
        } catch (err) {
            addMessage("butler", `Purge failed: ${String(err)}`);
        }
    }

    function handleButlerEvent(e) {
        const msg = `Observation: ${e.Type} at ${e.Path}`;
        insights = [
            { id: Date.now(), text: msg, type: "observation" },
            ...insights,
        ].slice(0, 10);
    }

    function handleAction(action) {
        const type = normalizeActionType(action?.type);
        const target = String(resolveActionTarget(action?.target || "")).toLowerCase();
        const workflowKey = getActionWorkflowKey(action);
        if (type === "clarify") {
            const data = getActionDataObject(action);
            const prompt = String(data.prompt || data.command || action?.prompt || action?.label || "").trim();
            if (!prompt) {
                addMessage("butler", "I need a prompt attached to this command choice. Please ask Butler to regenerate the options.");
                return;
            }
            userInput = prompt;
            send();
            return;
        }

        const actionState = getActionRuntimeState(action);
        if (actionState?.status === "needs_input" || actionState?.status === "invalid_payload") {
            addMessage(
                "butler",
                `This action is not ready: ${actionState.status.replace("_", " ")}. ${actionState.reason || "Please add missing details and ask Butler to regenerate this action."}`,
            );
            return;
        }

        if (isWriteAction(action) && !isActionArmed(action)) {
            armActionConfirmation(action);
            addMessage("butler", buildActionPreview(action));
            return;
        }

        if (isWriteAction(action)) {
            clearPendingActionArm();
        }

        if (type === "navigate") {
            dispatch("navigate", { screen: action.target });
        } else if (type === "analyze" || type === "fetch") {
            userInput = `Tell me more about ${action.target}`;
            send();
        } else if (type === "approve" || type === "reject") {
            executeGenericApprovalAction(action);
        } else if (type === "update") {
            executeUpdateAction(action);
        } else if (
            type === "create" ||
            type === "create_order" ||
            type === "create_stock_adjustment" ||
            workflowKey === "create_offer_draft" ||
            workflowKey === "create_follow_up" ||
            type === "create_offer_draft" ||
            type === "create_follow_up"
        ) {
            executeCreateAction(action);
        } else if (workflowKey === "daily_briefing") {
            runWorkflowAction(workflowKey, action);
        } else {
            if (workflowKey) {
                runWorkflowAction(workflowKey, action);
                return;
            }

            if (target) {
                runWorkflowAction(String(target).replace(/\s+/g, "_").toLowerCase(), action);
                return;
            }

            if (typeof action.data === "string") {
                userInput = `Proceed with ${action.data}`;
                send();
                return;
            }
        }
    }

    onMount(async () => {
        loadHiddenConversationKeys();
        loadActiveConversationId();
        await loadConversations();
        if (activeConversationId) {
            const activeConversation = conversations.find((conv) => convId(conv) === activeConversationId);
            if (activeConversation) {
                await selectConversation(activeConversation);
            } else {
                activeConversationId = "";
                saveActiveConversationId();
            }
        }
        if (!activeConversationId && conversations.length > 0) {
            await selectConversation(conversations[0]);
        }
        if (!activeConversationId && conversations.length === 0) {
            addMessage("butler", "Good day. I can answer questions about customers, suppliers, financials, and operations. I can also generate detailed PDF intelligence reports, create offer drafts, and prepare daily briefings on request.");
        }
        if (window.runtime) EventsOn("butler:event", handleButlerEvent);
    });

    onDestroy(() => {
        if (pendingDeleteTimer) clearTimeout(pendingDeleteTimer);
        if (pendingClearAllTimer) clearTimeout(pendingClearAllTimer);
        if (pendingActionTimer) clearTimeout(pendingActionTimer);
        if (window.runtime) EventsOff("butler:event");
    });
</script>

<div class="butler-layout">
    <!-- Conversations Sidebar -->
    <div class="conv-sidebar">
        <button class="new-conv-btn" onclick={() => startNewConversation()}>+ New Chat</button>
        <button class="new-conv-btn danger-btn" onclick={armClearAll}>
            {pendingClearAll ? "Click again to confirm" : "Clear All Chats"}
        </button>
        <div class="panel-card quick-actions-card">
            <p class="panel-label">Quick actions</p>
            <div class="prompts">
                <button class="prompt-chip" type="button" disabled={loading} onclick={() => runWorkflowAction("create_offer_draft")}>
                    Create offer draft
                </button>
                <button class="prompt-chip" type="button" disabled={loading} onclick={() => runWorkflowAction("daily_briefing")}>
                    Daily briefing
                </button>
            </div>
        </div>
        <div class="conv-list">
            {#each conversations as conv (convKey(conv))}
                <div
                    class="conv-item"
                    class:active={activeConversationId === convId(conv)}
                    onclick={() => selectConversation(conv)}
                    onkeydown={(event) => (event.key === "Enter" || event.key === " ") && selectConversation(conv)}
                    role="button"
                    tabindex="0"
                >
                    <span class="conv-title">{convTitle(conv)}</span>
                    <button
                        class="conv-delete"
                        class:conv-delete-armed={pendingDeleteKey === convKey(conv)}
                        onclick={stopPropagation(() => armDelete(conv))}
                        title={pendingDeleteKey === convKey(conv) ? "Click again to delete" : "Delete conversation"}
                    >{pendingDeleteKey === convKey(conv) ? "?" : "\u00d7"}</button>
                </div>
            {/each}
            {#if conversations.length === 0}
                <p class="conv-empty">No conversations yet</p>
            {/if}
        </div>
    </div>

    <!-- Main Chat Panel -->
    <div class="chat-panel">
        <div class="messages" id="chat-feed">
        {#if loadingConversation}
            <div class="conversation-loading">
                <WabiSpinner size="md" />
                <p>Loading conversation...</p>
            </div>
        {/if}
        {#each messages as m (m.id)}
            <div class="msg-row {m.role}" in:fly={{ y: 8, duration: 200 }}>
                {#if m.role === "butler"}
                    <div class="avatar">B</div>
                {/if}
                    <div class="bubble-wrap">
                        <div class="bubble">
                            {#if m.role === "butler"}
                                {@html formatButlerMessage(m.text)}
                            {:else}
                                {m.text}
                            {/if}
                        </div>
                        {#if m.actions && m.actions.length > 0}
                            <div class="actions-row">
                                {#each m.actions as action}
                                    {@const actionType = normalizeActionType(action?.type || "")}
                                    {@const state = getActionRuntimeState(action)}
                                    {@const actionArmed = isActionArmed(action)}
                                    <button
                                        class="action-chip"
                                        class:chip-clarify={actionType === "clarify"}
                                        class:chip-disabled={state.status === "needs_input" || state.status === "invalid_payload"}
                                        class:chip-warning={state.status === "needs_approval"}
                                        class:chip-ready={state.status === "ready_for_execution"}
                                        class:chip-armed={actionArmed}
                                        disabled={state.status === "needs_input" || state.status === "invalid_payload"}
                                        title={state.reason || action.label || ""}
                                        onclick={() => handleAction(action)}
                                    >
                                        {#if actionType === "navigate"}
                                            <span class="chip-icon">&rarr;</span>
                                        {:else if actionType === "clarify"}
                                            <span class="chip-icon">&rarr;</span>
                                        {:else if actionType === "analyze"}
                                            <span class="chip-icon">&bull;</span>
                                        {:else}
                                        <span class="chip-icon">+</span>
                                        {/if}
                                        <span>{actionLabelOrFallback(action, typeof action.data === "string" ? action.data : (action.target || "Action"))}</span>
                                        <span class="chip-state">
                                            {actionArmed ? "Confirm" : actionType === "clarify" ? "Choose" : actionChipStatusLabel(state.status)}
                                        </span>
                                    </button>
                                {/each}
                            </div>
                        {/if}
                    </div>
                </div>
            {/each}
            {#if loading}
                <div class="msg-row butler">
                    <div class="avatar">B</div>
                    <div class="bubble typing">
                        <span class="dot-pulse"></span>
                    </div>
                </div>
            {/if}
        </div>

        <div class="input-area">
            <input
                class="chat-input"
                type="text"
                bind:value={userInput}
                placeholder="Ask about financials, customers, suppliers..."
                onkeydown={(e) => e.key === "Enter" && send()}
                disabled={loading}
            />
            <button class="send-btn" onclick={send} disabled={loading || !userInput}>
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="12" y1="19" x2="12" y2="5"/>
                    <polyline points="5 12 12 5 19 12"/>
                </svg>
            </button>
        </div>
    </div>

</div>

<style>
    .butler-layout {
        display: grid;
        grid-template-columns: 200px 1fr;
        gap: 16px;
        flex: 1;
        min-height: 0;
        height: min(calc(100vh - 230px), 760px);
        max-height: calc(100vh - 210px);
    }

    /* Conversations Sidebar */
    .conv-sidebar {
        background: var(--surface, #fff);
        border-radius: var(--border-radius, 8px);
        border: 1px solid var(--border, #E5E5E5);
        display: flex;
        flex-direction: column;
        overflow: hidden;
    }

    .new-conv-btn {
        padding: 12px 16px;
        background: var(--carbon, #000);
        color: #fff;
        border: none;
        font-size: 13px;
        font-weight: 500;
        cursor: pointer;
        transition: opacity 0.15s;
    }

    .new-conv-btn:hover {
        opacity: 0.85;
    }

    .conv-list {
        flex: 1;
        overflow-y: auto;
        padding: 8px;
    }

    .quick-actions-card {
        margin: 8px;
    }

    .conv-item {
        display: flex;
        align-items: center;
        padding: 10px 12px;
        border-radius: 6px;
        cursor: pointer;
        font-size: 13px;
        color: var(--steel, #86868B);
        transition: all 0.15s;
    }

    .conv-item:hover {
        background: var(--ether, #F5F5F7);
        color: var(--onyx, #1D1D1F);
    }

    .conv-item.active {
        background: var(--ether, #F5F5F7);
        color: var(--onyx, #1D1D1F);
        font-weight: 500;
    }

    .conv-title {
        flex: 1;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .conv-delete {
        opacity: 0;
        background: none;
        border: none;
        color: var(--steel, #86868B);
        font-size: 16px;
        cursor: pointer;
        padding: 2px 6px;
        border-radius: 4px;
    }

    .conv-item:hover .conv-delete {
        opacity: 1;
    }

    .conv-delete:hover {
        color: #d00;
    }

    .conv-delete-armed {
        opacity: 1 !important;
        color: #d00;
        font-weight: bold;
        animation: pulse-red 0.6s ease-in-out infinite alternate;
    }

    @keyframes pulse-red {
        from { color: #d00; }
        to { color: #f55; }
    }

    .conv-empty {
        padding: 16px;
        text-align: center;
        font-size: 12px;
        color: var(--steel, #86868B);
    }

    /* Chat Panel */
    .chat-panel {
        background: var(--surface);
        border-radius: var(--border-radius);
        border: 1px solid var(--border);
        display: flex;
        flex-direction: column;
        overflow: hidden;
        min-height: 0;
    }

    .messages {
        flex: 1;
        overflow-y: auto;
        padding: 20px;
        display: flex;
        flex-direction: column;
        gap: 14px;
        scroll-behavior: smooth;
    }

    .conversation-loading {
        display: flex;
        align-items: center;
        gap: 10px;
        color: var(--steel, #86868B);
        font-size: 13px;
    }

    .msg-row {
        display: flex;
        gap: 10px;
        align-items: flex-end;
        max-width: 75%;
    }

    .msg-row.user {
        align-self: flex-end;
        flex-direction: row-reverse;
    }

    .avatar {
        width: 26px;
        height: 26px;
        background: var(--carbon);
        color: var(--canvas);
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 11px;
        font-weight: 600;
        flex-shrink: 0;
    }

    .bubble-wrap {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    .bubble {
        padding: 10px 14px;
        border-radius: 14px;
        font-size: 13px;
        line-height: 1.55;
        white-space: pre-wrap;
        overflow-wrap: anywhere;
    }

    .bubble :global(.butler-section-title) {
        margin: 8px 0 4px;
        color: var(--text-primary);
        font-size: 13px;
        font-weight: 700;
    }

    .bubble :global(.butler-section-title:first-child) {
        margin-top: 0;
    }

    .bubble :global(.butler-list-item) {
        display: grid;
        grid-template-columns: 14px minmax(0, 1fr);
        gap: 6px;
        align-items: start;
        margin: 3px 0;
    }

    .bubble :global(.butler-list-item > span) {
        width: 5px;
        height: 5px;
        margin-top: 8px;
        border-radius: 50%;
        background: var(--text-secondary);
    }

    .bubble :global(.butler-list-item.numbered > span) {
        width: auto;
        height: auto;
        margin-top: 0;
        border-radius: 0;
        background: transparent;
        color: var(--text-secondary);
        font-size: 11px;
        font-weight: 700;
    }

    .bubble :global(.butler-list-item p) {
        margin: 0;
    }

    .bubble :global(code) {
        padding: 1px 4px;
        border-radius: 4px;
        background: rgba(0, 0, 0, 0.06);
        font-family: var(--font-mono, monospace);
        font-size: 12px;
    }

    .table-wrap {
        margin: 8px 0;
        width: 100%;
        overflow-x: auto;
    }

    .butler-table {
        width: 100%;
        border-collapse: collapse;
        table-layout: fixed;
        font-size: 12px;
        background: var(--surface);
        border: 1px solid var(--border);
        border-radius: 8px;
        overflow: hidden;
    }

    .msg-row.butler .bubble {
        background: var(--surface-elevated);
        border: 1px solid var(--border);
        border-bottom-left-radius: 4px;
        color: var(--text-primary);
    }

    .msg-row.user .bubble {
        background: var(--carbon);
        color: var(--canvas);
        border-bottom-right-radius: 4px;
    }

    .typing {
        display: flex;
        align-items: center;
        padding: 12px 16px;
    }

    .dot-pulse {
        width: 6px;
        height: 6px;
        background: var(--text-muted);
        border-radius: 50%;
        animation: pulse 1s infinite;
    }

    @keyframes pulse {
        0%, 100% { opacity: 0.3; }
        50% { opacity: 1; }
    }

    .actions-row {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
    }

    .action-chip {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        padding: 4px 10px;
        font-size: 11px;
        font-family: var(--font-family);
        background: var(--surface);
        border: 1px solid var(--border);
        border-radius: 100px;
        cursor: pointer;
        color: var(--text-secondary);
        transition: all var(--transition-fast);
    }

    .action-chip .chip-state {
        padding: 2px 6px;
        border-radius: 999px;
        background: var(--surface-elevated);
        border: 1px solid var(--border);
        font-size: 9px;
        color: var(--text-muted);
    }

    .action-chip:hover {
        background: var(--carbon);
        color: var(--canvas);
        border-color: var(--carbon);
    }

    .chip-clarify {
        background: color-mix(in srgb, var(--accent) 10%, var(--surface));
        border-color: color-mix(in srgb, var(--accent) 35%, var(--border));
        color: var(--text-primary);
    }

    .chip-clarify .chip-state {
        background: var(--accent);
        border-color: var(--accent);
        color: var(--canvas);
    }

    .chip-disabled {
        cursor: not-allowed;
        opacity: 0.7;
    }

    .chip-disabled:hover {
        background: var(--surface);
        color: var(--text-secondary);
        border-color: var(--border);
    }

    .chip-ready .chip-state {
        background: rgba(5, 150, 105, 0.18);
        border-color: rgba(5, 150, 105, 0.35);
        color: #047857;
    }

    .chip-warning .chip-state {
        background: rgba(217, 119, 6, 0.18);
        border-color: rgba(217, 119, 6, 0.4);
        color: #b45309;
    }

    .chip-armed {
        background: rgba(29, 78, 216, 0.08);
        border-color: rgba(37, 99, 235, 0.45);
        color: #1d4ed8;
    }

    .chip-armed .chip-state {
        background: rgba(37, 99, 235, 0.14);
        border-color: rgba(37, 99, 235, 0.35);
        color: #1d4ed8;
    }

    .chip-icon {
        font-weight: 700;
        font-size: 12px;
    }

    /* Input Area */
    .input-area {
        padding: 12px 16px;
        border-top: 1px solid var(--border);
        display: flex;
        gap: 10px;
        align-items: center;
        background: var(--surface);
    }

    .chat-input {
        flex: 1;
        border: none;
        outline: none;
        font-size: 13px;
        background: transparent;
        font-family: var(--font-family);
        color: var(--text-primary);
    }

    .chat-input::placeholder {
        color: var(--text-muted);
    }

    .send-btn {
        width: 32px;
        height: 32px;
        border-radius: 50%;
        background: var(--carbon);
        color: var(--canvas);
        border: none;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: opacity var(--transition-fast);
    }

    .send-btn:disabled {
        opacity: 0.3;
        cursor: default;
    }

    /* Side Panel */
    .side-panel {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .panel-card {
        background: var(--surface);
        border: 1px solid var(--border);
        border-radius: var(--border-radius);
        padding: 14px;
    }

    .panel-label {
        font-size: var(--label-size);
        font-weight: var(--label-weight);
        color: var(--text-secondary);
        text-transform: uppercase;
        letter-spacing: 0.08em;
        margin: 0 0 10px;
    }

    .meter-track {
        height: 4px;
        background: var(--surface-elevated);
        border-radius: 2px;
        overflow: hidden;
        margin-bottom: 6px;
    }

    .meter-fill {
        height: 100%;
        background: var(--carbon);
        border-radius: 2px;
        transition: width 0.3s var(--easing-smooth);
    }

    .meter-value {
        font-size: 11px;
        color: var(--text-muted);
        font-variant-numeric: tabular-nums;
    }

    .empty-feed {
        font-size: 12px;
        color: var(--text-muted);
        font-style: italic;
        margin: 0;
    }

    .feed-list {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    .feed-item {
        font-size: 11px;
        padding: 6px 8px;
        background: var(--surface-elevated);
        border-radius: 6px;
        color: var(--text-primary);
        border: 1px solid var(--border);
    }

    /* Prompt Chips */
    .prompts {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    .prompt-chip {
        width: 100%;
        text-align: left;
        padding: 8px 12px;
        font-size: 12px;
        font-family: var(--font-family);
        background: var(--surface-elevated);
        border: 1px solid var(--border);
        border-radius: 8px;
        cursor: pointer;
        color: var(--text-secondary);
        transition: all var(--transition-fast);
    }

    .prompt-chip:hover {
        border-color: var(--onyx);
        color: var(--text-primary);
        background: var(--surface);
    }
</style>
