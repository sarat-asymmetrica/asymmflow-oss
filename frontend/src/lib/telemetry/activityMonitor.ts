import { EndUserActivitySession, RecordUserActivityBatch, RecordUserActivityHeartbeat, StartUserActivitySession } from "../../../wailsjs/go/main/InfraService";

type ActivityEvent = {
    session_id?: string;
    event_time?: string;
    event_type?: string;
    category?: string;
    screen?: string;
    route?: string;
    action_label?: string;
    action_key?: string;
    resource_type?: string;
    resource_id?: string;
    search_text?: string;
    metadata?: Record<string, any>;
    active_seconds?: number;
    meaningful_seconds?: number;
    idle_seconds?: number;
};

type ActivityHeartbeat = {
    session_id: string;
    screen: string;
    active_seconds: number;
    meaningful_seconds: number;
    idle_seconds: number;
    event_count: number;
    search_count: number;
    create_count: number;
    update_count: number;
    export_count: number;
    navigation_count: number;
};

const FLUSH_INTERVAL_MS = 20000;
const HEARTBEAT_INTERVAL_MS = 60000;
const IDLE_AFTER_MS = 120000;
const MAX_BATCH_SIZE = 50;

let started = false;
let sessionId = "";
let getCurrentScreen: () => string = () => "dashboard";
let pendingEvents: ActivityEvent[] = [];
let flushTimer: number | undefined;
let heartbeatTimer: number | undefined;
let lastHeartbeatAt = Date.now();
let lastInteractionAt = Date.now();
let interactionCount = 0;
let eventCounters = {
    event_count: 0,
    search_count: 0,
    create_count: 0,
    update_count: 0,
    export_count: 0,
    navigation_count: 0,
};

const searchTimers = new WeakMap<HTMLInputElement | HTMLTextAreaElement, number>();

function wailsReady() {
    return typeof window !== "undefined" && Boolean((window as any).go);
}

function cleanText(value: string | null | undefined, maxLen = 160) {
    const cleaned = String(value || "")
        .replace(/\s+/g, " ")
        .trim();
    return cleaned.length > maxLen ? cleaned.slice(0, maxLen) : cleaned;
}

function actionKey(value: string) {
    return cleanText(value, 80)
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/^-+|-+$/g, "");
}

function classifyAction(label: string, eventType = "click") {
    const text = `${eventType} ${label}`.toLowerCase();
    if (text.includes("search") || text.includes("filter")) return "search";
    if (text.includes("create") || text.includes("add") || text.includes("new")) return "create";
    if (text.includes("save") || text.includes("update") || text.includes("edit") || text.includes("approve")) return "update";
    if (text.includes("export") || text.includes("download") || text.includes("print") || text.includes("pdf") || text.includes("excel")) return "export";
    if (text.includes("delete") || text.includes("remove")) return "delete";
    if (text.includes("navigation") || text.includes("navigate")) return "navigation";
    return "action";
}

function bumpCounter(category: string) {
    eventCounters.event_count += 1;
    if (category === "search") eventCounters.search_count += 1;
    if (category === "create") eventCounters.create_count += 1;
    if (category === "update" || category === "save" || category === "edit") eventCounters.update_count += 1;
    if (category === "export") eventCounters.export_count += 1;
    if (category === "navigation") eventCounters.navigation_count += 1;
}

function enqueueActivity(event: ActivityEvent) {
    if (!started || !sessionId) return;
    const category = event.category || classifyAction(event.action_label || "", event.event_type);
    const payload: ActivityEvent = {
        ...event,
        session_id: sessionId,
        event_time: new Date().toISOString(),
        category,
        screen: event.screen || getCurrentScreen(),
        route: window.location.hash || window.location.pathname,
    };
    pendingEvents.push(payload);
    bumpCounter(category);
    if (pendingEvents.length >= MAX_BATCH_SIZE) {
        void flushActivity();
    }
}

async function flushActivity() {
    if (!started || pendingEvents.length === 0 || !wailsReady()) return;
    const batch = pendingEvents.splice(0, MAX_BATCH_SIZE);
    try {
        await RecordUserActivityBatch(batch as any);
    } catch (error) {
        console.warn("Activity monitor batch failed:", error);
        pendingEvents = [...batch.slice(-20), ...pendingEvents].slice(0, MAX_BATCH_SIZE);
    }
}

function heartbeatPayload(): ActivityHeartbeat {
    const now = Date.now();
    const elapsedSeconds = Math.max(1, Math.round((now - lastHeartbeatAt) / 1000));
    const visible = document.visibilityState !== "hidden";
    const active = visible && now - lastInteractionAt <= IDLE_AFTER_MS;
    const activeSeconds = active ? elapsedSeconds : 0;
    const idleSeconds = active ? 0 : elapsedSeconds;
    const meaningfulSeconds = active && interactionCount > 0 ? activeSeconds : 0;

    lastHeartbeatAt = now;
    interactionCount = 0;
    const payload: ActivityHeartbeat = {
        session_id: sessionId,
        screen: getCurrentScreen(),
        active_seconds: activeSeconds,
        meaningful_seconds: meaningfulSeconds,
        idle_seconds: idleSeconds,
        ...eventCounters,
    };
    eventCounters = {
        event_count: 0,
        search_count: 0,
        create_count: 0,
        update_count: 0,
        export_count: 0,
        navigation_count: 0,
    };
    return payload;
}

async function sendHeartbeat() {
    if (!started || !sessionId || !wailsReady()) return;
    await flushActivity();
    try {
        await RecordUserActivityHeartbeat(heartbeatPayload() as any);
    } catch (error) {
        console.warn("Activity monitor heartbeat failed:", error);
    }
}

function markInteraction() {
    lastInteractionAt = Date.now();
    interactionCount += 1;
}

function labelForElement(element: Element) {
    const html = element as HTMLElement;
    return cleanText(
        html.getAttribute("aria-label") ||
            html.getAttribute("title") ||
            html.dataset?.telemetryLabel ||
            html.innerText ||
            html.textContent ||
            html.getAttribute("name") ||
            html.id ||
            element.tagName,
    );
}

function handleClick(event: MouseEvent) {
    markInteraction();
    const target = event.target as Element | null;
    const actionElement = target?.closest?.(
        "button,a,[role='button'],input[type='button'],input[type='submit'],[data-telemetry-action]",
    );
    if (!actionElement || actionElement.closest("[data-telemetry-ignore]")) return;

    const label = labelForElement(actionElement);
    if (!label) return;
    const category = classifyAction(label, "click");
    enqueueActivity({
        event_type: "click",
        category,
        action_label: label,
        action_key: actionKey(label),
        metadata: {
            tag: actionElement.tagName.toLowerCase(),
            id: (actionElement as HTMLElement).id || "",
        },
    });
}

function isSearchField(element: HTMLInputElement | HTMLTextAreaElement) {
    const type = (element as HTMLInputElement).type || "";
    if (["password", "email", "tel", "number"].includes(type.toLowerCase())) return false;
    const descriptor = [
        type,
        element.getAttribute("placeholder"),
        element.getAttribute("aria-label"),
        element.getAttribute("name"),
        element.id,
    ]
        .join(" ")
        .toLowerCase();
    return descriptor.includes("search") || descriptor.includes("filter") || descriptor.includes("query");
}

function safeSearchText(value: string) {
    const lower = value.toLowerCase();
    if (
        lower.includes("password") ||
        lower.includes("secret") ||
        lower.includes("token") ||
        lower.includes("api_key") ||
        lower.includes("apikey") ||
        lower.includes("license")
    ) {
        return "[redacted]";
    }
    return cleanText(value, 120);
}

function handleInput(event: Event) {
    markInteraction();
    const element = event.target as HTMLInputElement | HTMLTextAreaElement | null;
    if (!element || !isSearchField(element)) return;
    const existing = searchTimers.get(element);
    if (existing) window.clearTimeout(existing);
    const timer = window.setTimeout(() => {
        const search = safeSearchText(element.value || "");
        if (search.length < 2) return;
        enqueueActivity({
            event_type: "search",
            category: "search",
            action_label: labelForElement(element) || "Search",
            action_key: "search",
            search_text: search,
        });
    }, 1200);
    searchTimers.set(element, timer);
}

function handleKeydown() {
    markInteraction();
}

function handleVisibilityChange() {
    if (document.visibilityState === "hidden") {
        void sendHeartbeat();
    } else {
        lastHeartbeatAt = Date.now();
        lastInteractionAt = Date.now();
    }
}

export async function startActivityMonitor(screenGetter: () => string) {
    if (started || !wailsReady()) return;
    getCurrentScreen = screenGetter;
    try {
        const session = (await StartUserActivitySession("desktop")) as any;
        sessionId = session?.session_id || session?.SessionID || "";
        if (!sessionId) return;
        started = true;
        lastHeartbeatAt = Date.now();
        lastInteractionAt = Date.now();
        window.addEventListener("click", handleClick, true);
        window.addEventListener("input", handleInput, true);
        window.addEventListener("keydown", handleKeydown, true);
        document.addEventListener("visibilitychange", handleVisibilityChange);
        flushTimer = window.setInterval(flushActivity, FLUSH_INTERVAL_MS);
        heartbeatTimer = window.setInterval(sendHeartbeat, HEARTBEAT_INTERVAL_MS);
        recordActivityNavigation(getCurrentScreen(), {});
    } catch (error) {
        console.warn("Activity monitor failed to start:", error);
    }
}

export function recordActivityNavigation(screen: string, params: Record<string, any> = {}) {
    enqueueActivity({
        event_type: "navigation",
        category: "navigation",
        screen,
        action_label: `Open ${screen}`,
        action_key: `open-${actionKey(screen)}`,
        metadata: {
            params: Object.keys(params || {}).join(","),
        },
    });
}

export async function stopActivityMonitor() {
    if (!started) return;
    window.removeEventListener("click", handleClick, true);
    window.removeEventListener("input", handleInput, true);
    window.removeEventListener("keydown", handleKeydown, true);
    document.removeEventListener("visibilitychange", handleVisibilityChange);
    if (flushTimer) window.clearInterval(flushTimer);
    if (heartbeatTimer) window.clearInterval(heartbeatTimer);
    flushTimer = undefined;
    heartbeatTimer = undefined;
    const endingSession = sessionId;
    await flushActivity();
    try {
        if (endingSession && wailsReady()) {
            await RecordUserActivityHeartbeat(heartbeatPayload() as any);
            await EndUserActivitySession(endingSession);
        }
    } catch (error) {
        console.warn("Activity monitor failed to stop cleanly:", error);
    } finally {
        started = false;
        sessionId = "";
        pendingEvents = [];
    }
}
