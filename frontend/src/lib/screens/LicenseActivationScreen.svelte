<script lang="ts">
    import { createEventDispatcher } from "svelte";
    import { ActivateLicense } from "../../../wailsjs/go/main/App";
import { FirstRunSyncWithProgress, GetFirstRunSyncStatus } from "../../../wailsjs/go/main/SyncServiceBinding";
    import { toast } from "$lib/stores/toasts";
    import SyncProgress from "../components/ui/SyncProgress.svelte";
    import { brand } from "$lib/brand";

    const dispatch = createEventDispatcher();

    let licenseKey = $state("");
    let loading = $state(false);
    let syncing = $state(false);
    let showSyncProgress = $state(false);
    let error = $state("");

    // Valid role codes for license keys
    const VALID_ROLES = ['ADM', 'MGR', 'SLS', 'OPS', 'STF', 'DEV'];

    // Validate license key format: PH-{3 letters}-{6 alphanumeric}
    const LICENSE_KEY_PATTERN = /^PH-(ADM|MGR|SLS|OPS|STF|DEV)-[A-Z0-9]{6}$/;

    // Format key as user types (uppercase, add dashes)
    function formatKey(event: Event) {
        const input = event.target as HTMLInputElement;
        let value = input.value.toUpperCase().replace(/[^A-Z0-9-]/g, "");

        // Auto-format: PH-XXX-YYYYYY
        if (value.length > 2 && !value.includes("-")) {
            value = "PH-" + value.substring(2);
        }
        if (value.length > 6 && value.split("-").length === 2) {
            const parts = value.split("-");
            value = parts[0] + "-" + parts[1].substring(0, 3) + "-" + parts[1].substring(3);
        }

        // Limit to 13 chars (PH-XXX-YYYYYY)
        licenseKey = value.substring(0, 13);

        // Validate format when complete length is reached
        if (licenseKey.length === 13 && !LICENSE_KEY_PATTERN.test(licenseKey)) {
            error = "Invalid key format. Use PH-XXX-YYYYYY where XXX is ADM, MGR, SLS, OPS, STF, or DEV";
        } else if (licenseKey.length === 13) {
            error = ""; // Clear error if valid
        }
    }

    // Reactive validation - updates whenever licenseKey changes
    let keyValid = $derived(licenseKey.length === 13 && LICENSE_KEY_PATTERN.test(licenseKey));

    async function handleActivate() {
        if (loading || !keyValid) return;

        loading = true;
        error = "";

        try {
            // Wrap activation in 15s timeout — backend can hang if DB is slow on startup
            const result = await Promise.race([
                ActivateLicense(licenseKey),
                new Promise((_, reject) => setTimeout(() => reject(new Error('Activation timed out — please try again')), 15000))
            ]) as any;

            if (result.success) {
                toast.success(result.message);

                // Skip first-run sync entirely — Supabase may be down
                // Sync will happen automatically in the background once app is running

                dispatch("activated", {
                    role: result.role,
                    display_name: result.display_name,
                    permissions: result.permissions,
                    deviceHash: result.device_hash
                });
            } else {
                error = result.message;
                toast.danger(result.message);
            }
        } catch (err: any) {
            error = err?.message || "Activation timed out. The app may still be starting up — wait 10 seconds and try again.";
            toast.danger(error);
        } finally {
            loading = false;
            syncing = false;
            showSyncProgress = false;
        }
    }

    function handleKeyPress(event: KeyboardEvent) {
        if (event.key === "Enter" && keyValid) {
            handleActivate();
        }
    }

    function focusOnMount(node: HTMLInputElement) {
        requestAnimationFrame(() => node.focus());
        return {
            destroy() {},
        };
    }
</script>

<div class="license-screen">
    <div class="license-card">
        <div class="logo-mark">{brand.mark}</div>
        <h1>{brand.wordmark}</h1>
        <p class="subtitle">Enter your license key to activate this device</p>

        <div class="form-group">
            <label for="license-key">License Key</label>
            <input
                id="license-key"
                type="password"
                value={licenseKey}
                oninput={formatKey}
                onkeypress={handleKeyPress}
                placeholder="PH-ADM-A1B2C3"
                class="license-input"
                class:error={error}
                disabled={loading}
                autocomplete="off"
                spellcheck="false"
                use:focusOnMount
            />
            <p class="hint">Contact your administrator for a license key</p>
        </div>

        {#if error}
            <div class="error-message">
                {error}
            </div>
        {/if}

        <button
            class="btn-primary"
            onclick={handleActivate}
            disabled={loading || syncing || !keyValid}
        >
            {#if syncing}
                <span class="spinner"></span>
                Syncing data...
            {:else if loading}
                <span class="spinner"></span>
                Activating...
            {:else}
                Activate License
            {/if}
        </button>

        <!-- Sync Progress Modal -->
        <SyncProgress bind:show={showSyncProgress} title="Downloading Data" />

        <div class="key-format">
            <p>Key format: <code>PH-XXX-YYYYYY</code></p>
            <ul>
                <li><code>PH-ADM-*</code> = Administrator (full access)</li>
                <li><code>PH-MGR-*</code> = Manager (finance + operations)</li>
                <li><code>PH-SLS-*</code> = Sales (sales pipeline only)</li>
                <li><code>PH-OPS-*</code> = Operations (procurement only)</li>
            </ul>
        </div>
    </div>
</div>

<style>
    .license-screen {
        min-height: 100vh;
        display: flex;
        align-items: center;
        justify-content: center;
        background: linear-gradient(135deg, #f5f5f7 0%, #e5e5ea 100%);
        padding: 20px;
        /* Ensure license screen is on top */
        position: relative;
        z-index: 100;
    }

    .license-card {
        background: white;
        border-radius: 16px;
        padding: 48px;
        max-width: 420px;
        width: 100%;
        text-align: center;
        box-shadow: 0 4px 24px rgba(0, 0, 0, 0.08);
    }

    .logo-mark {
        width: 56px;
        height: 56px;
        background: #1d1d1f;
        color: white;
        border-radius: 12px;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        font-weight: 700;
        font-size: 20px;
        margin-bottom: 24px;
    }

    h1 {
        font-size: 24px;
        font-weight: 700;
        color: #1d1d1f;
        margin: 0 0 8px;
    }

    .subtitle {
        color: #86868b;
        font-size: 15px;
        margin: 0 0 32px;
    }

    .form-group {
        text-align: left;
        margin-bottom: 24px;
    }

    .form-group label {
        display: block;
        font-size: 13px;
        font-weight: 600;
        color: #1d1d1f;
        margin-bottom: 8px;
    }

    .license-input {
        width: 100%;
        padding: 14px 16px;
        font-size: 18px;
        font-family: "JetBrains Mono", monospace;
        letter-spacing: 0.05em;
        text-align: center;
        border: 2px solid #e5e5ea;
        border-radius: 12px;
        background: #fafafa;
        color: #1d1d1f;
        transition: all 0.2s ease;
        box-sizing: border-box;
        /* Explicit pointer-events to ensure clickability */
        pointer-events: auto;
        cursor: text;
    }

    .license-input:focus {
        outline: none;
        border-color: #1d1d1f;
        background: white;
    }

    .license-input.error {
        border-color: #dc2626;
        background: #fef2f2;
    }

    .license-input:disabled {
        opacity: 0.6;
        cursor: not-allowed;
    }

    .hint {
        font-size: 12px;
        color: #86868b;
        margin: 8px 0 0;
    }

    .error-message {
        background: #fef2f2;
        border: 1px solid #fee2e2;
        color: #dc2626;
        padding: 12px 16px;
        border-radius: 8px;
        font-size: 14px;
        margin-bottom: 24px;
        text-align: left;
    }

    .btn-primary {
        width: 100%;
        padding: 14px 24px;
        font-size: 16px;
        font-weight: 600;
        color: white;
        background: #1d1d1f;
        border: none;
        border-radius: 12px;
        cursor: pointer;
        transition: all 0.2s ease;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        /* CRITICAL: position relative to contain ::after pseudo-element */
        position: relative;
        overflow: hidden;
    }

    .btn-primary:hover:not(:disabled) {
        background: #3a3a3c;
        transform: translateY(-1px);
    }

    .btn-primary:disabled {
        opacity: 0.5;
        cursor: not-allowed;
        transform: none;
    }

    .spinner {
        width: 16px;
        height: 16px;
        border: 2px solid rgba(255, 255, 255, 0.3);
        border-top-color: white;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }

    @keyframes spin {
        to { transform: rotate(360deg); }
    }

    .key-format {
        margin-top: 32px;
        padding-top: 24px;
        border-top: 1px solid #e5e5ea;
        text-align: left;
    }

    .key-format p {
        font-size: 13px;
        color: #86868b;
        margin: 0 0 12px;
    }

    .key-format ul {
        margin: 0;
        padding: 0 0 0 20px;
        font-size: 12px;
        color: #86868b;
    }

    .key-format li {
        margin: 4px 0;
    }

    .key-format code {
        background: #f5f5f7;
        padding: 2px 6px;
        border-radius: 4px;
        font-family: "JetBrains Mono", monospace;
        font-size: 11px;
        color: #1d1d1f;
    }
</style>
