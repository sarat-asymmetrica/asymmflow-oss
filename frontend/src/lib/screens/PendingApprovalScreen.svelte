<script lang="ts">
    import { onMount, onDestroy } from "svelte";
    import { createEventDispatcher } from "svelte";
    import { CheckDeviceStatus } from "../../../wailsjs/go/main/App";
import { GetCurrentDeviceInfo } from "../../../wailsjs/go/main/InfraService";

    const dispatch = createEventDispatcher();

    let deviceName = $state("");
    let deviceId = $state("");
    let firstSeen = $state("");
    let checking = $state(false);
    let pollInterval: ReturnType<typeof setInterval>;

    // P1-7 FIX: Circuit breaker for polling
    let consecutiveFailures = 0;
    let maxConsecutiveFailures = 3;
    let pollingPaused = false;
    let connectionLost = $state(false);

    onMount(async () => {
        // Get device info
        try {
            const device = await GetCurrentDeviceInfo();
            deviceName = device.device_name || "Unknown Device";
            deviceId = device.id?.substring(0, 8) || "—";
            firstSeen = device.first_seen_at ? new Date(String(device.first_seen_at)).toLocaleString() : "—";
        } catch (err) {
            console.error("Failed to get device info:", err);
        }

        // Poll for status changes every 10 seconds
        pollInterval = setInterval(checkStatus, 10000);
    });

    onDestroy(() => {
        if (pollInterval) clearInterval(pollInterval);
    });

    async function checkStatus() {
        if (checking || pollingPaused) return;
        checking = true;

        try {
            const result = await CheckDeviceStatus();
            // P1-7 FIX: Reset failure counter on success
            consecutiveFailures = 0;
            connectionLost = false;

            if (result.status === "approved") {
                dispatch("approved", result);
            } else if (result.status === "blocked") {
                dispatch("blocked");
            }
        } catch (err) {
            console.error("Status check failed:", err);
            // P1-7 FIX: Circuit breaker logic
            consecutiveFailures++;

            if (consecutiveFailures >= maxConsecutiveFailures) {
                pollingPaused = true;
                connectionLost = true;
                if (pollInterval) clearInterval(pollInterval);
                console.warn(`Polling paused after ${consecutiveFailures} consecutive failures`);
            }
        } finally {
            checking = false;
        }
    }

    function manualCheck() {
        // P1-7 FIX: Manual check resumes polling
        consecutiveFailures = 0;
        pollingPaused = false;
        connectionLost = false;

        // Restart polling if it was stopped
        if (!pollInterval) {
            pollInterval = setInterval(checkStatus, 10000);
        }

        checkStatus();
    }
</script>

<div class="pending-container">
    <div class="pending-card">
        <div class="icon-container">
            <div class="icon">
                <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                    <circle cx="12" cy="12" r="10"/>
                    <path d="M12 6v6l4 2"/>
                </svg>
            </div>
        </div>

        <h1>Awaiting Approval</h1>
        <p class="subtitle">This device needs to be approved by an administrator before you can use the application.</p>

        {#if connectionLost}
            <!-- P1-7 FIX: Connection lost banner -->
            <div class="connection-lost">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
                    <line x1="12" y1="9" x2="12" y2="13"/>
                    <line x1="12" y1="17" x2="12.01" y2="17"/>
                </svg>
                <span>Connection lost. Click "Check Status" to retry.</span>
            </div>
        {/if}

        <div class="device-info">
            <div class="info-row">
                <span class="label">Device Name</span>
                <span class="value">{deviceName}</span>
            </div>
            <div class="info-row">
                <span class="label">Device ID</span>
                <span class="value code">{deviceId}...</span>
            </div>
            <div class="info-row">
                <span class="label">Registered</span>
                <span class="value">{firstSeen}</span>
            </div>
        </div>

        <div class="status-indicator">
            {#if !connectionLost}
                <div class="pulse"></div>
                <span>Checking for approval...</span>
            {:else}
                <div class="pulse-paused"></div>
                <span>Waiting to retry...</span>
            {/if}
        </div>

        <button class="btn-secondary" onclick={manualCheck} disabled={checking}>
            {checking ? "Checking..." : "Check Status"}
        </button>

        <div class="instructions">
            <h3>What to do next</h3>
            <ol>
                <li>Contact your system administrator</li>
                <li>Provide them with your Device ID: <code>{deviceId}</code></li>
                <li>Wait for approval — this screen will automatically update</li>
            </ol>
        </div>
    </div>
</div>

<style>
    .pending-container {
        min-height: 100vh;
        display: flex;
        align-items: center;
        justify-content: center;
        background: var(--bg-base, #f5f5f7);
        padding: 20px;
    }

    .pending-card {
        background: var(--surface, #fff);
        border-radius: 16px;
        padding: 48px;
        max-width: 480px;
        width: 100%;
        text-align: center;
        box-shadow: 0 4px 24px rgba(0, 0, 0, 0.08);
    }

    .icon-container {
        margin-bottom: 24px;
    }

    .icon {
        width: 80px;
        height: 80px;
        background: var(--bg-base, #f5f5f7);
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        margin: 0 auto;
        color: var(--text-secondary, #86868b);
    }

    h1 {
        font-size: 24px;
        font-weight: 700;
        color: var(--text-primary, #1d1d1f);
        margin: 0 0 8px;
    }

    .subtitle {
        color: var(--text-secondary, #86868b);
        font-size: 15px;
        margin: 0 0 32px;
        line-height: 1.5;
    }

    .device-info {
        background: var(--bg-base, #f5f5f7);
        border-radius: 12px;
        padding: 20px;
        margin-bottom: 24px;
    }

    .info-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 8px 0;
    }

    .info-row:not(:last-child) {
        border-bottom: 1px solid var(--border, #e5e5e5);
    }

    .label {
        color: var(--text-secondary, #86868b);
        font-size: 13px;
    }

    .value {
        color: var(--text-primary, #1d1d1f);
        font-size: 14px;
        font-weight: 500;
    }

    .value.code {
        font-family: "JetBrains Mono", monospace;
        font-size: 12px;
    }

    .status-indicator {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 12px;
        margin-bottom: 24px;
        color: var(--text-secondary, #86868b);
        font-size: 14px;
    }

    .pulse {
        width: 12px;
        height: 12px;
        background: #fbbf24;
        border-radius: 50%;
        animation: pulse 2s infinite;
    }

    @keyframes pulse {
        0%, 100% { opacity: 1; transform: scale(1); }
        50% { opacity: 0.5; transform: scale(1.1); }
    }

    .btn-secondary {
        padding: 12px 24px;
        background: transparent;
        color: var(--text-primary, #1d1d1f);
        border: 1px solid var(--border, #e5e5e5);
        border-radius: 8px;
        font-size: 14px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
        margin-bottom: 32px;
    }

    .btn-secondary:hover:not(:disabled) {
        background: var(--bg-base, #f5f5f7);
    }

    .btn-secondary:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .instructions {
        text-align: left;
        padding-top: 24px;
        border-top: 1px solid var(--border, #e5e5e5);
    }

    .instructions h3 {
        font-size: 14px;
        font-weight: 600;
        color: var(--text-primary, #1d1d1f);
        margin: 0 0 12px;
    }

    .instructions ol {
        margin: 0;
        padding-left: 20px;
        color: var(--text-secondary, #86868b);
        font-size: 13px;
        line-height: 1.8;
    }

    .instructions code {
        background: var(--bg-base, #f5f5f7);
        padding: 2px 6px;
        border-radius: 4px;
        font-family: "JetBrains Mono", monospace;
        font-size: 11px;
    }

    /* P1-7 FIX: Connection lost banner */
    .connection-lost {
        background: #fef2f2;
        border: 1px solid #fecaca;
        border-radius: 8px;
        padding: 12px 16px;
        margin-bottom: 24px;
        display: flex;
        align-items: center;
        gap: 12px;
        color: #dc2626;
        font-size: 13px;
    }

    .connection-lost svg {
        flex-shrink: 0;
    }

    .pulse-paused {
        width: 12px;
        height: 12px;
        background: #94a3b8;
        border-radius: 50%;
    }
</style>
