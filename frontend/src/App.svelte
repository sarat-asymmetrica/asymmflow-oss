<script lang="ts">
    import { run } from 'svelte/legacy';

    import { devLog } from "$lib/utils/devLog";
    import { onMount, onDestroy, tick } from "svelte";

    // ============================================
    // DESIGN MODE - For frontend-only development
    // Set to false when running with Wails backend
    // ============================================
    const DESIGN_MODE = false; // Disabled for backend testing

    // Set to true to demo the setup/onboarding screen
    const DEMO_SETUP_SCREEN = false; // Toggle to true to show ArrivalCeremony

    // Core screens used directly in routing
    // OneDrive import disabled — manual folder download + extraction in next session
    // import OneDriveImportScreen from "./lib/screens/OneDriveImportScreen.svelte";
    import ArrivalCeremony from "./lib/screens/ArrivalCeremony.svelte";
    import DashboardScreen from "./lib/screens/DashboardScreen.svelte";

    // Auth Context
    import {
        initAuthContext, navVisibility, currentUser, permissions, } from "./lib/stores/authContext";
    // Single shared nav source of truth (Wave 9.5 B8) — drives sidebar order,
    // Alt+N shortcut order, and the shell-level permission gate together.
    import { NAV_ITEMS, SCREEN_PERMISSIONS } from "./lib/config/navItems";
    import { t } from "$lib/i18n";

    // Error Boundary - Catch uncaught errors gracefully
    import ErrorBoundary from "./lib/components/ErrorBoundary.svelte";

    // Wabi-Sabi Toast System
    import ToastContainer from "./lib/components/ui/ToastContainer.svelte";
    import ConfirmHost from "./lib/components/ui/ConfirmHost.svelte";
    import { toast } from "./lib/stores/toasts";
    // ToastTestButton removed - debug only

    // Magic Cursor & Floating Nav
    import CursorFollower from "./lib/components/CursorFollower.svelte";
    import FloatingNav from "./lib/components/FloatingNav.svelte";

    // Enterprise Design Components
    import EnterpriseHeader from "./lib/components/ui/EnterpriseHeader.svelte";
    import EnterpriseSidebar from "./lib/components/ui/EnterpriseSidebar.svelte";

    // Wails bindings
    import { EventsOn, EventsOff, OnFileDrop, OnFileDropOff } from "../wailsjs/runtime/runtime";
    import { NeedsSetup } from "../wailsjs/go/main/App";
import { ProcessDocumentWithOCR, SaveDocumentToEntity } from "../wailsjs/go/main/DocumentsService";
import { RegisterDevice, ValidateLicense, NeedsLicenseActivation } from "../wailsjs/go/main/InfraService";

    // Device Registration Screens (legacy)
    import SetupAdminScreen from "./lib/screens/SetupAdminScreen.svelte";
    import PendingApprovalScreen from "./lib/screens/PendingApprovalScreen.svelte";
    import LoginScreen from "./lib/screens/LoginScreen.svelte";

    // License Activation Screen (new key-based system)
    import LicenseActivationScreen from "./lib/screens/LicenseActivationScreen.svelte";

    // Global file drop store
    import { fileDrop } from "./lib/stores/fileDrop";
    import { authNotice } from "./lib/stores/authNotice";

    // Global OCR Modal
    import QuickCaptureModal from "./lib/components/QuickCaptureModal.svelte";
    import {
        recordActivityNavigation,
        startActivityMonitor,
        stopActivityMonitor,
    } from "./lib/telemetry/activityMonitor";
    import { initTextScale } from "./lib/stores/textScale";

    // Global OCR State - Works on ANY screen
    let showGlobalOCRModal = $state(false);
    let globalOCRProcessing = $state(false);
    let globalOCRResult = $state(null);
    let globalOCRFileName = $state("");
    let globalOCRSessionKey = $state(0);

    // Butler event handler - background file-watcher/import events.
    // Wave 10 B6 (Article IV.4): a toast may only echo a user action, so
    // this no longer announces file events via toast. No notifications/
    // digest surface exists yet to route this to; left as a no-op handler
    // (still registered below so future routing has a single hook point).
    function handleButlerEvent(event) {
        // Intentionally no toast — see comment above.
    }

    // Setup state
    let needsSetup = $state(false);
    // FORCED: Start as false to prevent loading screen
    let checkingSetup = false;

    // Device registration state
    // Status: "checking", "first_setup", "pending", "approved", "blocked", "login", "error", "license_needed"
    let deviceStatus = $state("checking");
    let deviceInfo = null;
    let deviceError = $state(""); // P1-2 FIX: Store error message for retry
    let licenseRole = ""; // Role from license (admin, manager, sales, operations)
    let licensePermissions: string[] = []; // Permissions from license

    // Currently active screen
    let currentScreen = $state("dashboard");
    let currentParams = $state({});
    let screenParamsById: Record<string, Record<string, any>> = $state({ dashboard: {} });
    const persistentScreenIDs = new Set([
        "dashboard",
        "opportunities",
        "operations",
        "finance",
        "work",
        "people",
        "notifications",
        "deployment",
        "relationships",
        "intelligence",
        "settings",
        "showcase",
        "usermanagement",
        "rfqs",
        // Wave 9.6 Sh4: keep Accounting/Reports mounted so their filter/tab/scroll
        // state survives navigation (parity with the other persistent screens).
        "accounting",
        "reports",
    ]);
    let mountedPersistentScreens: string[] = $state(["dashboard"]);

    function isPersistentScreen(screen: string) {
        return persistentScreenIDs.has(screen);
    }

    function setCurrentScreen(screen: string, params: Record<string, any> = {}) {
        currentScreen = screen;
        currentParams = params;
        screenParamsById = { ...screenParamsById, [screen]: params };
        recordActivityNavigation(screen, params);
    }

    function startMonitoringIfReady() {
        if (DESIGN_MODE || deviceStatus !== "approved") return;
        void startActivityMonitor(() => currentScreen);
    }

    async function ensureScreenComponent(screen: string) {
        const loader = screenLoaders[screen];
        if (!loader) {
            activeScreenComponent = null;
            activeScreenLoading = false;
            return;
        }

        if (loadedScreenComponents[screen]) {
            activeScreenComponent = loadedScreenComponents[screen];
            if (isPersistentScreen(screen) && !mountedPersistentScreens.includes(screen)) {
                mountedPersistentScreens = [...mountedPersistentScreens, screen];
            }
            activeScreenLoading = false;
            return;
        }

        const loadToken = ++activeScreenLoadToken;
        activeScreenLoading = true;
        activeScreenComponent = null;
        try {
            const mod = await loader();
            if (loadToken !== activeScreenLoadToken) return;
            loadedScreenComponents[screen] = mod.default;
            activeScreenComponent = mod.default;
            if (isPersistentScreen(screen) && !mountedPersistentScreens.includes(screen)) {
                mountedPersistentScreens = [...mountedPersistentScreens, screen];
            }
        } catch (err) {
            if (loadToken !== activeScreenLoadToken) return;
            console.error("Failed to load screen component", screen, err);
            toast.danger(`Failed to load ${screen} screen`);
            activeScreenComponent = null;
        } finally {
            if (loadToken === activeScreenLoadToken) {
                activeScreenLoading = false;
            }
        }
    }

    let navigateToScreenListener: ((event: Event) => void) | null = null;
    let requestGlobalOCRListener: ((event: Event) => void) | null = null;
    let appLogoutListener: (() => void) | null = null;

    // For simple hash-based routing support (uses navigate for permission checking)
    function handleHashChange() {
        const hash = window.location.hash.substring(1); // Remove '#'
        if (hash.startsWith("/customers/")) {
            const id = hash.split("/")[2];
            // P0-4 FIX: Check permission via navigate before showing screen
            if (hasScreenPermission("customer360")) {
                setCurrentScreen("customer360", { id });
            } else {
                toast.danger("Access denied: You don't have permission to view customer details");
            }
        } else if (hash === "/customers") {
            if (hasScreenPermission("relationships")) {
                setCurrentScreen("relationships", {});
            } else {
                toast.danger("Access denied: You don't have permission to view customers");
            }
        } else if (hash && screenPermissions[hash] !== undefined) {
            // Handle direct screen hashes like #finance, #operations
            if (hasScreenPermission(hash)) {
                setCurrentScreen(hash, {});
            } else {
                toast.danger(`Access denied: You don't have permission to view ${hash}`);
            }
        }
    }

    run(() => {
        if (deviceStatus === "approved" && !needsSetup) {
            void ensureScreenComponent(currentScreen);
        }
    });

    // Handle setup completion - transition from ceremony to main app
    function handleSetupComplete(event) {
        const { userName, selectedRole } = event.detail || {};
        needsSetup = false;

        // Personalize initial screen based on role
        if (selectedRole === "sales") setCurrentScreen("opportunities");
        else if (selectedRole === "ops") setCurrentScreen("operations");
        else if (selectedRole === "finance") setCurrentScreen("finance");
        else setCurrentScreen("dashboard");

        toast.success(
            `Welcome back, ${userName || "Commander"}. Your workspace is ready.`,
        );
    }

    // ═══════════════════════════════════════════════════════════════════
    // GLOBAL OCR FILE DROP HANDLER - Works on ANY screen!
    // ═══════════════════════════════════════════════════════════════════
    async function handleGlobalFileDrop(filePath) {
        if (!filePath || globalOCRProcessing) {
            console.log("Global OCR: Skipping (no path or already processing)");
            return;
        }

        console.log("Global OCR: Processing file:", filePath);

        const fileName = filePath.split(/[\\/]/).pop() || "document";
        const validExts = [".pdf", ".docx", ".xlsx", ".png", ".jpg", ".jpeg", ".msg", ".eml"];
        const fileExt = "." + fileName.split(".").pop()?.toLowerCase();

        if (!validExts.includes(fileExt)) {
            toast.danger(`Unsupported file type: ${fileExt}`);
            return;
        }

        if (showGlobalOCRModal) {
            showGlobalOCRModal = false;
            globalOCRResult = null;
            globalOCRFileName = "";
            globalOCRProcessing = false;
            await tick();
        }

        // Show modal in processing state
        globalOCRSessionKey += 1;
        globalOCRFileName = fileName;
        globalOCRResult = null;
        globalOCRProcessing = true;
        showGlobalOCRModal = true;

        try {
            console.log("Global OCR: Calling ProcessDocumentWithOCR...");
            const result = await ProcessDocumentWithOCR(filePath, "auto");
            console.log("Global OCR: Result received:", result);

            globalOCRResult = result;
            globalOCRProcessing = false;

            const confidence = result?.confidence ?? 0;
            toast.success(`Document analyzed! ${(confidence * 100).toFixed(0)}% confidence`);
        } catch (err) {
            console.error("Global OCR: Failed:", err);
            toast.danger(`OCR failed: ${err?.message || err}`);
            globalOCRProcessing = false;
            showGlobalOCRModal = false;
        }
    }

    async function handleGlobalOCRSave(event) {
        const {
            type, screen, result, fileName,
            projectName, customerName, supplierName, estimatedValue,
            documentData, lineItems, butlerInsights
        } = event.detail;

        console.log("Global OCR: Saving document:", {
            type, fileName, projectName, customerName,
            lineItemsCount: lineItems?.length || 0,
            documentData
        });

        // Build comprehensive extracted data from user-edited fields
        const mergedData = {
            ...(result?.extracted_data || {}),
            ...(documentData || {}),
        };

        // Ensure key fields are set from user input
        if (projectName) mergedData.project = projectName;
        if (customerName) mergedData.customer_name = customerName;
        if (supplierName) mergedData.supplier_name = supplierName;
        if (estimatedValue && estimatedValue > 0) mergedData.total = estimatedValue;

        // Include line items for costing/RFQ creation
        if (lineItems && lineItems.length > 0) {
            mergedData.line_items = lineItems;
        }

        // Include Butler insights metadata
        if (butlerInsights) {
            mergedData.butler_summary = butlerInsights.summary;
            mergedData.butler_confidence = butlerInsights.confidence;
        }

        let extractedDataJSON = "";
        try {
            extractedDataJSON = JSON.stringify(mergedData);
        } catch (e) {
            console.warn("Failed to serialize extracted_data:", e);
        }

        try {
            // Route to correct domain table (RFQ→rfq_data, Invoice→invoices, etc.)
            const routeResult = await SaveDocumentToEntity(
                fileName,
                "",
                type,
                result?.text || "",
                result?.confidence || 0,
                result?.processing_time_ms || 0,
                result?.engine || "unknown",
                extractedDataJSON
            );

            const typeName = type.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
            const itemCount = lineItems?.length || 0;
            if (routeResult?.routed) {
                toast.success(`${typeName} saved with ${itemCount} items!`);
            } else {
                toast.success(`${typeName} saved!`);
            }

            showGlobalOCRModal = false;
            globalOCRResult = null;
            globalOCRFileName = "";

            // Dispatch custom event to refresh relevant screens
            window.dispatchEvent(new CustomEvent('ocr-document-saved', {
                detail: {
                    type, fileName,
                    entityId: routeResult?.entity_id,
                    lineItems: lineItems,
                    documentData: mergedData
                }
            }));

            // Navigate to the appropriate screen based on document type
            const typeToScreen = {
                "invoice": "finance",
                "supplier_invoice": "finance",
                "bank_statement": "finance",
                "rfq": "opportunities",
                "quotation": "opportunities",
                "costing": "costing",
                "excel_data": "opportunities",
                "purchase_order": "operations",
                "delivery_note": "operations",
            };
            const targetScreen = typeToScreen[type];
            if (targetScreen) {
                if (targetScreen === "finance" && type === "invoice") {
                    navigate({ screen: "finance", tab: "invoices", company: mergedData.division || "Acme Instrumentation" });
                } else if (targetScreen === "finance" && type === "bank_statement") {
                    navigate({ screen: "finance", tab: "bank_recon", company: mergedData.division || "Acme Instrumentation" });
                } else {
                    navigate(targetScreen);
                }
                toast.info(`Navigate to ${typeName} to continue editing`);
            }
        } catch (err) {
            console.error("Global OCR Save failed:", err);
            toast.danger(`Failed to save: ${err?.message || err}`);
        }
    }

    function handleGlobalOCRModalClose() {
        showGlobalOCRModal = false;
        globalOCRResult = null;
        globalOCRFileName = "";
        globalOCRProcessing = false;
    }

    // Keyboard shortcuts (Zen navigation)
    function handleKeyboard(e) {
        // Don't trigger shortcuts when typing in inputs
        const isInputFocused = ["INPUT", "TEXTAREA", "SELECT"].includes(
            document.activeElement?.tagName,
        );

        // Escape: Close modals, collapse mobile nav
        if (e.key === "Escape") {
            if (isMobile && showNav) {
                showNav = false;
                e.preventDefault();
            }
        }

        // Ctrl/Cmd + K: Quick navigation toggle
        if ((e.ctrlKey || e.metaKey) && e.key === "k" && !isInputFocused) {
            e.preventDefault();
            showNav = !showNav;
        }

        // Number keys 1-9: Quick screen navigation (Alt + Number)
        if (e.altKey && !isInputFocused) {
            const num = parseInt(e.key);
            if (num >= 1 && num <= 9 && screens[num - 1]) {
                e.preventDefault();
                navigate(screens[num - 1].id);
            }
        }
    }

    type ScreenModule = { default: any };

    const screenLoaders: Record<string, () => Promise<ScreenModule>> = {
        dashboard: () => Promise.resolve({ default: DashboardScreen }),
        opportunities: () => import("./lib/screens/SalesHub.svelte"),
        operations: () => import("./lib/screens/OperationsHub.svelte"),
        finance: () => import("./lib/screens/FinanceHub.svelte"),
        // Wave 8 P4 slices 2-3: two fully-built screens that were registered
        // nowhere (orphaned). Routing them in makes them reachable via the sidebar.
        accounting: () => import("./lib/screens/AccountingScreen.svelte"),
        reports: () => import("./lib/screens/ReportsScreen.svelte"),
        work: () => import("./lib/screens/WorkHub.svelte"),
        people: () => import("./lib/screens/PeopleHub.svelte"),
        notifications: () => import("./lib/screens/NotificationsScreen.svelte"),
        deployment: () => import("./lib/screens/DeploymentHub.svelte"),
        relationships: () => import("./lib/screens/CRMHub.svelte"),
        intelligence: () => import("./lib/screens/IntelligenceHub.svelte"),
        settings: () => import("./lib/screens/SettingsScreen.svelte"),
        showcase: () => import("./lib/screens/ShowcaseScreen.svelte"),
        customer360: () => import("./lib/screens/Customer360.svelte"),
        usermanagement: () => import("./lib/screens/UserManagementScreen.svelte"),
        rfqs: () => import("./lib/screens/RFQScreen.svelte"),
    };
    const loadedScreenComponents: Record<string, any> = $state({ dashboard: DashboardScreen });
    let activeScreenComponent: any = $state(DashboardScreen);
    let activeScreenLoading = $state(false);
    let activeScreenLoadToken = 0;

    // Primary navigation flows — the Alt+N shortcut targets are exactly the
    // permission-visible nav items, in the same order the sidebar renders them,
    // so "shortcut order == visual order" always holds (Wave 9.5 B8). Derived
    // from the shared NAV_ITEMS via the same permission filter the sidebar uses.
    let screens = $derived(NAV_ITEMS.filter((item) => hasScreenPermission(item.id)));

    // Human title for a screen id, resolved from the shared nav list via i18n
    // (used by the loading/placeholder states). Falls back to the id for
    // non-nav routes.
    function screenLabel(id: string): string {
        const item = NAV_ITEMS.find((s) => s.id === id);
        return item ? t(item.labelKey) : id;
    }

    let showNav = false;
    let isMobile = false;

    function handleResize() {
        isMobile = window.innerWidth < 1024;
        if (!isMobile) showNav = true;
    }

    // SINGLE UNIFIED onMount - Consolidates all initialization
    onMount(() => {
        let cleanup: (() => void) | undefined;
        initTextScale();

        void (async () => {

        // In DESIGN_MODE, skip all Wails backend calls for rapid UI iteration
        if (DESIGN_MODE) {
            devLog.info(
                "DESIGN MODE ACTIVE - Skipping backend calls for rapid UI iteration",
            );
            checkingSetup = false;
            deviceStatus = "approved"; // Skip device registration in design mode
            // If DEMO_SETUP_SCREEN is true, show the ArrivalCeremony
            needsSetup = DEMO_SETUP_SCREEN;
        } else {
            // ============================================
            // LICENSE-BASED ACTIVATION (Production Mode)
            // ============================================
            devLog.info("Checking license status...");

            try {
                // Check if device has a valid license
                // Use timeout wrapper to prevent infinite hang after Windows updates
                const LICENSE_TIMEOUT_MS = 10000; // 10 second timeout
                const timeoutPromise = new Promise((_, reject) =>
                    setTimeout(() => reject(new Error('License check timed out')), LICENSE_TIMEOUT_MS)
                );

                devLog.info(`Starting license validation (${LICENSE_TIMEOUT_MS}ms timeout)...`);
                const licenseResult = await Promise.race([
                    ValidateLicense(),
                    timeoutPromise
                ]) as { valid: boolean; role?: string; permissions?: string[]; display_name?: string };

                if (licenseResult && licenseResult.valid) {
                    // License is valid - set up user context from license
                    devLog.info(`Valid license found: role=${licenseResult.role}, name=${licenseResult.display_name}`);
                    deviceStatus = "approved";
                    licenseRole = licenseResult.role;
                    licensePermissions = licenseResult.permissions || [];

                    // Set user context - use employee name from license if available
                    const roleDisplayNames: Record<string, string> = {
                        'admin': 'Administrator',
                        'manager': 'Manager',
                        'sales': 'Sales Rep',
                        'operations': 'Ops User',
                        'dev': 'Developer',
                    };
                    const displayName = licenseResult.display_name || roleDisplayNames[licenseResult.role] || 'PH User';
                    currentUser.set({
                        id: `license-${licenseResult.role}`,
                        full_name: displayName,
                        role_name: licenseResult.role,
                    });

                    // Set permissions from license (do NOT call initAuthContext - it would overwrite these)
                    permissions.set(licensePermissions);

                    devLog.info(`Permissions set from license: ${JSON.stringify(licensePermissions)}`);

                    toast.success(`Welcome, ${displayName}!`);

                    // Register file drop handler now that license is valid
                    registerFileDropHandler();
                    startMonitoringIfReady();
                } else {
                    // No valid license - show activation screen
                    devLog.info("No valid license found - showing activation screen");
                    deviceStatus = "license_needed";
                }
            } catch (err: any) {
                devLog.error("License check failed:", err);
                // Check if it was a timeout
                if (err.message === 'License check timed out') {
                    devLog.warn("License check timed out - showing activation screen");
                    toast.danger("License verification timed out. Please try again.");
                }
                // Fallback: show license activation screen
                deviceStatus = "license_needed";
            }

            checkingSetup = false;
        }

        // Initialize resize handling
        handleResize();
        window.addEventListener("resize", handleResize);

        // Initialize routing
        window.addEventListener("hashchange", handleHashChange);
        handleHashChange(); // Check on load

        // Initialize keyboard shortcuts
        window.addEventListener("keydown", handleKeyboard);

        // Initialize setup flow
        window.addEventListener("setup-complete", handleSetupComplete);

        // Subscribe to screen navigation events (from OrdersScreen, etc.)
        navigateToScreenListener = (e: Event) => {
            const detail = (e as CustomEvent).detail;
            navigate(detail);
        };
        window.addEventListener("navigateToScreen", navigateToScreenListener);

        requestGlobalOCRListener = (e: Event) => {
            const detail = (e as CustomEvent).detail || {};
            void handleGlobalFileDrop(detail.filePath);
        };
        window.addEventListener("requestGlobalOCR", requestGlobalOCRListener);

        // Wave 6 Mission C.1: the header's Sign out button invalidates
        // the backend session, then dispatches this event — clear auth
        // state and return to the login screen.
        appLogoutListener = () => {
            currentUser.set(null);
            permissions.set([]);
            deviceStatus = "login";
            toast.info("Signed out.");
        };
        window.addEventListener("app:logout", appLogoutListener);

        // Subscribe to Butler file watcher events for real-time notifications
        // Only in non-design mode (requires Wails runtime)
        if (!DESIGN_MODE) {
            EventsOn("butler:event", handleButlerEvent);

            // Wave 5 Mission B: the backend refuses bound calls after 30
            // minutes of inactivity and emits this event — return to login.
            EventsOn("auth:session-expired", () => {
                currentUser.set(null);
                permissions.set([]);
                deviceStatus = "login";
                // Wave 10 B6 (Article IV.4/V): a 30-min inactivity timeout is a
                // background state transition, not a user action — so it must not
                // arrive as a toast. Carry the reason to the login surface instead.
                authNotice.set("Your session timed out after 30 minutes of inactivity. Please sign in again.");
            });

            // NOTE: File drop handler is registered in registerFileDropHandler()
            // AFTER license validation to avoid interfering with license input
        }

        // Cleanup function - CRITICAL: Remove all event listeners and Wails handlers
        cleanup = () => {
            window.removeEventListener("resize", handleResize);
            window.removeEventListener("hashchange", handleHashChange);
            window.removeEventListener("setup-complete", handleSetupComplete);
            window.removeEventListener("keydown", handleKeyboard);
            if (navigateToScreenListener) {
                window.removeEventListener("navigateToScreen", navigateToScreenListener);
            }
            if (requestGlobalOCRListener) {
                window.removeEventListener("requestGlobalOCR", requestGlobalOCRListener);
            }
            if (appLogoutListener) {
                window.removeEventListener("app:logout", appLogoutListener);
            }

            // Clean up Wails event handlers
            if (!DESIGN_MODE) {
                EventsOff("butler:event");
                EventsOff("auth:session-expired");
                OnFileDropOff(); // CRITICAL: Removes drag/drop listeners
                void stopActivityMonitor();
            }
        };

        })();

        return () => {
            cleanup?.();
        };
    });

    // Screen permission mapping — derived from the single shared nav source of
    // truth (Wave 9.5 B8), so it can never drift from the sidebar again. This
    // closed a latent shell-gate gap: accounting/reports were sidebar entries
    // but were MISSING from this map, so hasScreenPermission() returned
    // undefined for them and fell through the "no permission required" branch
    // (a hash/deep-link could reach #accounting/#reports without finance:view/
    // reports:view — same class as the Wave 9.4 usermanagement fix). The
    // orphan #rfqs route is now gated too. See lib/config/navItems.ts.
    const screenPermissions: Record<string, string | null> = SCREEN_PERMISSIONS;

    // Check if user has permission for a screen
    function hasScreenPermission(screenID: string): boolean {
        const requiredPerm = screenPermissions[screenID];
        if (!requiredPerm) return true; // No permission required

        const perms = Array.isArray($permissions) ? $permissions : [];
        if (perms.length === 0) return false;

        // Wildcard (admin)
        if (perms.includes('*')) return true;

        // Direct match
        if (perms.includes(requiredPerm)) return true;

        // Resource wildcard (e.g., "finance:*" matches "finance:view")
        const [resource] = requiredPerm.split(':');
        if (perms.includes(`${resource}:*`)) return true;

        return false;
    }

    function normalizeNavigationTarget(target) {
        if (!target) return { screen: "", params: {} };
        if (typeof target === "string") return { screen: target, params: {} };

        const { screen = "", ...rest } = target;
        const params = {};
        for (const [key, value] of Object.entries(rest)) {
            if (value !== undefined && value !== null && value !== "") {
                params[key] = value;
            }
        }
        return { screen, params };
    }

    function navigate(target) {
        const { screen, params } = normalizeNavigationTarget(target);
        if (!screen) return;

        console.log(
            "Navigation clicked:",
            screen,
            "params:",
            params,
            "changing from",
            currentScreen,
        );

        // P0-4 FIX: Check permission before navigating
        if (!hasScreenPermission(screen)) {
            toast.danger(`Access denied: You don't have permission to view ${screen}`);
            console.warn(`Navigation blocked: no permission for ${screen}`);
            return;
        }

        if (screen === "costing" || screen === "offers" || screen === "orders") {
            setCurrentScreen("opportunities", { tab: screen, ...params });
        } else {
            setCurrentScreen(screen, params);
        }
        console.log("Screen now:", currentScreen, "params:", currentParams);
        if (isMobile) showNav = false;
    }

    function handleFloatingNav(event) {
        navigate(event.detail);
    }

    // Handle admin setup completion
    async function handleAdminSetupComplete() {
        devLog.info("Admin setup complete, initializing app...");
        deviceStatus = "approved";
        await initAuthContext();
        toast.success("Administrator account created successfully!");
        startMonitoringIfReady();
    }

    // Handle device approval (from pending screen)
    async function handleDeviceApproved(event) {
        devLog.info("Device approved, initializing...");
        deviceInfo = event.detail;
        deviceStatus = "login"; // Show login screen after approval
    }

    // Handle login success
    async function handleLoginSuccess(event) {
        devLog.info("Login successful, initializing app...");
        const result = event.detail;

        // Set user context
        currentUser.set({
            id: result.user_id,
            full_name: result.user_name,
            role_name: result.role_name,
        });

        // Set permissions
        if (result.permissions) {
            permissions.set(result.permissions);
        }

        deviceStatus = "approved";
        await initAuthContext();
        toast.success(`Welcome, ${result.user_name}!`);
        startMonitoringIfReady();
    }

    // Handle blocked device
    function handleDeviceBlocked() {
        deviceStatus = "blocked";
    }

    // Handle license activation success
    async function handleLicenseActivated(event: CustomEvent) {
        const { role, permissions: permsFromLicense, deviceHash, display_name } = event.detail;
        devLog.info(`License activated: role=${role}, name=${display_name}`);

        deviceStatus = "approved";
        licenseRole = role;
        licensePermissions = permsFromLicense || [];

        // Set user context - use employee name from license if available
        const roleDisplayNames: Record<string, string> = {
            'admin': 'Administrator',
            'manager': 'Manager',
            'sales': 'Sales Rep',
            'operations': 'Ops User',
            'dev': 'Developer',
        };
        const displayName = display_name || roleDisplayNames[role] || 'PH User';
        currentUser.set({
            id: `license-${role}`,
            full_name: displayName,
            role_name: role,
        });

        // Set permissions from license (do NOT call initAuthContext - it would overwrite these)
        permissions.set(licensePermissions);

        devLog.info(`Permissions set from license: ${JSON.stringify(licensePermissions)}`);

        // Register file drop handler now that license is valid
        registerFileDropHandler();
        startMonitoringIfReady();
    }

    // Register file drop handler - only after license is validated
    let fileDropRegistered = false;
    function registerFileDropHandler() {
        if (fileDropRegistered || DESIGN_MODE) return;
        fileDropRegistered = true;

        // Register GLOBAL file drop handler - processes OCR directly!
        // useDropTarget=false means ANY drop anywhere triggers this
        OnFileDrop((x, y, paths) => {
            console.log("App: Global file drop detected", { x, y, paths, screen: currentScreen });

            if (paths && paths.length > 0) {
                // Process the first dropped file with OCR
                handleGlobalFileDrop(paths[0]);

                // Also publish to store for any screen-specific handling
                fileDrop.drop(x, y, paths);
            }
        }, false);

        devLog.info("File drop handler registered");
    }

    // P1-2 FIX: Retry device registration
    async function retryDeviceRegistration() {
        deviceStatus = "checking";
        deviceError = "";
        checkingSetup = true;

        try {
            devLog.info("Retrying device registration...");
            const result = await RegisterDevice();
            deviceInfo = result;
            devLog.info("Device status:", result.status);

            switch (result.status) {
                case "first_setup":
                    deviceStatus = "first_setup";
                    break;
                case "pending":
                    deviceStatus = "pending";
                    break;
                case "blocked":
                    deviceStatus = "blocked";
                    break;
                case "approved":
                    if (result.user_id) {
                        deviceStatus = "approved";
                        currentUser.set({
                            id: result.user_id,
                            full_name: result.user_name,
                            role_name: result.role_name,
                        });
                        if (result.permissions) {
                            permissions.set(result.permissions);
                        }
                    } else {
                        deviceStatus = "login";
                    }
                    break;
                default:
                    deviceStatus = "approved";
            }

            checkingSetup = false;

            if (deviceStatus === "approved") {
                try {
                    await initAuthContext();
                    toast.success("Device registered successfully!");
                } catch (e: any) {
                    devLog.error("Failed to initialize auth context:", e);
                    toast.danger(`Authentication initialization failed: ${e?.message || "Unknown error"}`);
                    deviceStatus = "error";
                    deviceError = e?.message || "Unknown error";
                }
            }
        } catch (e: any) {
            devLog.error("Retry failed:", e);
            deviceError = e?.message || "Unknown error";
            toast.danger(`Retry failed: ${deviceError}`);
            deviceStatus = "error";
            checkingSetup = false;
        }
    }
</script>

<!-- Error Boundary - Catches uncaught errors to prevent white screen of death -->
<ErrorBoundary>
    <!-- Magic Cursor Follower - Black dot that follows mouse -->
    <CursorFollower />

    <!-- Wabi-Sabi Toast Container - Global notifications with sumi-e brush strokes -->
    <ToastContainer position="top-right" />

    <!-- Canonical confirm primitive host (Design Constitution III.6 / VI.2) -->
    <ConfirmHost />


    <!-- DEVICE REGISTRATION FLOW -->
    {#if deviceStatus === "checking"}
        <div class="loading-screen">
            <div class="loading-spinner"></div>
            <p>Verifying license...</p>
        </div>
    {:else if deviceStatus === "license_needed"}
        <!-- License Activation Required -->
        <LicenseActivationScreen on:activated={handleLicenseActivated} />
    {:else if deviceStatus === "first_setup"}
        <!-- First Installation - Admin Setup -->
        <SetupAdminScreen on:setup-complete={handleAdminSetupComplete} />
    {:else if deviceStatus === "pending"}
        <!-- Device Pending Approval -->
        <PendingApprovalScreen
            on:approved={handleDeviceApproved}
            on:blocked={handleDeviceBlocked}
        />
    {:else if deviceStatus === "blocked"}
        <!-- Device Blocked -->
        <div class="blocked-screen">
            <div class="blocked-card">
                <div class="blocked-icon">Access Blocked</div>
                <h1>Access Denied</h1>
                <p>This device has been blocked by the administrator.</p>
                <p class="contact">Please contact your system administrator for assistance.</p>
            </div>
        </div>
    {:else if deviceStatus === "login"}
        <!-- Device Approved - Need Login -->
        <LoginScreen on:login-success={handleLoginSuccess} />
    {:else if deviceStatus === "error"}
        <!-- Device Registration Error -->
        <div class="error-screen">
            <div class="error-card">
                <div class="error-icon">Error</div>
                <h1>Connection Error</h1>
                <p>Unable to register this device with the server.</p>
                <div class="error-details">
                    <p><strong>Error:</strong> {deviceError}</p>
                </div>
                <button class="btn-primary" onclick={retryDeviceRegistration}>
                    Retry Connection
                </button>
                <p class="help-text">
                    If the problem persists, please check your internet connection
                    or contact your system administrator.
                </p>
            </div>
        </div>
    {:else if needsSetup}
        <!-- The Arrival Ceremony - Professional onboarding experience -->
        <ArrivalCeremony on:complete={handleSetupComplete} />
    {:else}
        <div class="app-layout">
            <!-- Sidebar Navigation -->
            <EnterpriseSidebar
                {currentScreen}
                on:navigate={(e) => navigate(e.detail)}
            />

            <!-- Main Content Area -->
            <div class="main-content-wrapper">
                <!-- Header with Title & Actions -->
                <EnterpriseHeader />
                
                <!-- Screen Content -->
                <main class="screen-content">
                    {#if isPersistentScreen(currentScreen)}
                        {#if !loadedScreenComponents[currentScreen]}
                            {#if activeScreenLoading}
                                <div class="loading-screen inline-screen-loader">
                                    <div class="loading-spinner"></div>
                                    <p>Loading {screenLabel(currentScreen)}...</p>
                                </div>
                            {:else}
                                <div class="coming-soon">
                                    <h2>
                                        {screenLabel(currentScreen)}
                                    </h2>
                                    <p class="subtitle">Screen did not finish loading. Try another section or restart the app.</p>
                                </div>
                            {/if}
                        {:else if mountedPersistentScreens.length > 0}
                            {#each mountedPersistentScreens as screenID (screenID)}
                                {#if loadedScreenComponents[screenID]}
                                    {@const SvelteComponent = loadedScreenComponents[screenID]}
                                    <div
                                        class="screen-host"
                                        class:screen-hidden={screenID !== currentScreen}
                                    >
                                        <SvelteComponent
                                            params={{
                                                ...(screenParamsById[screenID] || {}),
                                                __active: screenID === currentScreen,
                                            }}
                                            on:navigate={(e) => navigate(e.detail)}
                                        />
                                    </div>
                                {/if}
                            {/each}
                        {:else}
                            {@const SvelteComponent_1 = loadedScreenComponents[currentScreen]}
                            <div class="screen-host">
                                <SvelteComponent_1
                                    params={screenParamsById[currentScreen] || {}}
                                    on:navigate={(e) => navigate(e.detail)}
                                />
                            </div>
                        {/if}
                    {:else if activeScreenComponent}
                        {@const SvelteComponent_2 = activeScreenComponent}
                        <div class="screen-host">
                            <SvelteComponent_2
                                params={currentParams}
                                on:navigate={(e) => navigate(e.detail)}
                            />
                        </div>
                    {:else if activeScreenLoading}
                        <div class="loading-screen inline-screen-loader">
                            <div class="loading-spinner"></div>
                            <p>Loading {screenLabel(currentScreen)}...</p>
                        </div>
                    {:else}
                        <!-- Placeholder for screens not yet implemented -->
                        <div class="coming-soon">
                            <h2>
                                {screenLabel(currentScreen)}
                            </h2>
                            <p class="subtitle">COMING IN NEXT WAVE</p>
                        </div>
                    {/if}
                </main>
            </div>
        </div>
    {/if}

    <!-- GLOBAL OCR Quick Capture Modal - Works on ANY screen! -->
    {#key globalOCRSessionKey}
        <QuickCaptureModal
            bind:show={showGlobalOCRModal}
            processing={globalOCRProcessing}
            ocrResult={globalOCRResult}
            fileName={globalOCRFileName}
            on:save={handleGlobalOCRSave}
            on:close={handleGlobalOCRModalClose}
        />
    {/key}
</ErrorBoundary>

<style>
    :global(body) {
        margin: 0;
        padding: 0;
        background-color: var(--bg-base);
        color: var(--text-primary);
        font-family: var(--font-family);
        -webkit-font-smoothing: antialiased;
    }

    .app-layout {
        display: flex;
        height: 100vh;
        background-color: var(--bg-base);
        overflow: hidden;
    }

    .main-content-wrapper {
        flex: 1;
        display: flex;
        flex-direction: column;
        margin-left: var(--sidebar-width); /* 220px */
        min-width: 0; /* Prevent flex blowout */
        height: 100vh;
        background-color: var(--bg-base);
    }

    .screen-content {
        flex: 1;
        overflow-y: auto;
        min-height: 0; /* Allow flex shrinking for overflow to work */
        position: relative;
        z-index: 1;
        /* Padding is handled by individual screens now */
    }

    /* Coming Soon Placeholder */
    .coming-soon {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        height: 70vh;
        text-align: center;
    }

    .coming-soon h2 {
        font-size: 24px;
        font-weight: 600;
        margin: 0 0 16px;
        color: var(--text-primary);
    }

    .coming-soon .subtitle {
        font-size: 12px;
        color: var(--text-secondary);
        text-transform: uppercase;
        letter-spacing: 0.1em;
        background: var(--surface);
        padding: 8px 16px;
        border-radius: 99px;
        border: 1px solid var(--border);
    }

    /* Loading Screen */
    .loading-screen {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        min-height: 100vh;
        background-color: var(--bg-base);
    }

    .loading-screen p {
        font-size: 14px;
        color: var(--text-secondary);
        margin-top: 16px;
    }

    .inline-screen-loader {
        min-height: 320px;
    }

    .screen-host {
        min-height: 100%;
    }

    .screen-hidden {
        display: none;
    }

    @media (max-width: 1024px) {
        .app-layout {
            flex-direction: column;
        }

        .main-content-wrapper {
            margin-left: 0;
            padding-top: 72px;
        }
    }

    .loading-spinner {
        width: 32px;
        height: 32px;
        border: 2px solid var(--border);
        border-top-color: var(--onyx);
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }

    @keyframes spin {
        to {
            transform: rotate(360deg);
        }
    }

    /* Blocked Device Screen */
    .blocked-screen {
        min-height: 100vh;
        display: flex;
        align-items: center;
        justify-content: center;
        background: var(--bg-base, #f5f5f7);
        padding: 20px;
    }

    .blocked-card {
        background: var(--surface, #fff);
        border-radius: 16px;
        padding: 48px;
        max-width: 400px;
        width: 100%;
        text-align: center;
        box-shadow: 0 4px 24px rgba(0, 0, 0, 0.08);
    }

    .blocked-icon {
        font-size: 64px;
        margin-bottom: 24px;
    }

    .blocked-card h1 {
        font-size: 24px;
        font-weight: 700;
        color: var(--text-primary, #1d1d1f);
        margin: 0 0 12px;
    }

    .blocked-card p {
        color: var(--text-secondary, #86868b);
        font-size: 15px;
        margin: 0 0 8px;
        line-height: 1.5;
    }

    .blocked-card .contact {
        margin-top: 24px;
        padding-top: 24px;
        border-top: 1px solid var(--border, #e5e5e5);
        font-size: 13px;
    }

    /* Error Screen Styles (P1-2 FIX) */
    .error-screen {
        min-height: 100vh;
        display: flex;
        align-items: center;
        justify-content: center;
        background: var(--bg-base, #f5f5f7);
        padding: 20px;
    }

    .error-card {
        background: var(--surface, #fff);
        border-radius: 16px;
        padding: 48px;
        max-width: 480px;
        width: 100%;
        text-align: center;
        box-shadow: 0 4px 24px rgba(0, 0, 0, 0.08);
    }

    .error-icon {
        font-size: 64px;
        margin-bottom: 24px;
    }

    .error-card h1 {
        font-size: 24px;
        font-weight: 700;
        color: var(--text-primary, #1d1d1f);
        margin: 0 0 12px;
    }

    .error-card p {
        color: var(--text-secondary, #86868b);
        font-size: 15px;
        margin: 0 0 8px;
        line-height: 1.5;
    }

    .error-details {
        background: #fef2f2;
        border: 1px solid #fee;
        border-radius: 8px;
        padding: 16px;
        margin: 24px 0;
        text-align: left;
    }

    .error-details p {
        margin: 0;
        color: #dc2626;
        font-size: 13px;
        font-family: "JetBrains Mono", monospace;
    }

    .error-card .btn-primary {
        width: 100%;
        margin: 24px 0 16px;
    }

    .error-card .help-text {
        margin-top: 24px;
        padding-top: 24px;
        border-top: 1px solid var(--border, #e5e5e5);
        font-size: 13px;
        color: var(--text-muted, #c7c7c7);
    }
</style>
