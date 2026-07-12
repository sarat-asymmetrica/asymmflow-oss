<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import * as d3 from "d3";
  import { fade } from "svelte/transition";
  import { toast } from "$lib/stores/toasts";
  import {
    StartArchaeologyScan } from "../../../wailsjs/go/main/App";
import { CancelScan } from "../../../wailsjs/go/main/InfraService";
  import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime";
  import WabiSpinner from "../components/ui/WabiSpinner.svelte";

  let loading = false;
  let scanning = $state(false);
  let scanPath = $state("");
  let logs = $state([]);
  let progress = $state(0);
  let stats = { files: 0, dirs: 0, size: 0, duration: 0 };
  let mockScanInterval: ReturnType<typeof setInterval> | null = null;
  let currentScanId = "";

  // Scan Types
  const scanTypes = [
    { id: "deep", label: "Deep Scan", desc: "Full content analysis" },
    { id: "quick", label: "Quick Scan", desc: "Metadata only" },
  ];
  let selectedType = $state("quick");

  async function start() {
    if (!scanPath) {
      toast.warning("Enter path");
      return;
    }
    scanning = true;
    progress = 0;
    logs = [];
    logs = [
      ...logs,
      {
        ts: new Date(),
        msg: `Starting ${selectedType} scan on ${scanPath}...`,
      },
    ];

    try {
      // Mock or Real
      if (!window.go) {
        // Mock
        let p = 0;
        mockScanInterval = setInterval(() => {
          p += 5;
          progress = p;
          logs = [...logs, { ts: new Date(), msg: `Scanning... ${p}%` }];
          if (p >= 100) {
            if (mockScanInterval) {
              clearInterval(mockScanInterval);
              mockScanInterval = null;
            }
            scanning = false;
            logs = [...logs, { ts: new Date(), msg: "Done." }];
          }
        }, 200);
      } else {
        EventsOn("archaeologist:scan:progress", (e) => {
          progress = e.percentage || 0;
          if (e.current_file)
            logs = [
              ...logs,
              { ts: new Date(), msg: `Processing: ${e.current_file}` },
            ];
          // Auto-scroll logic if needed
        });
        EventsOn("archaeologist:scan:complete", (e) => {
          scanning = false;
          toast.success("Scan Complete");
          logs = [...logs, { ts: new Date(), msg: "Scan Complete." }];
        });

        const isZip = scanPath.toLowerCase().endsWith(".zip");
        const outputDir = scanPath + "/archaeology_artifacts";
        currentScanId = await StartArchaeologyScan(scanPath, isZip, outputDir);
      }
    } catch (e) {
      toast.danger("Error starting scan");
      scanning = false;
      // Clear mock interval on error
      if (mockScanInterval) {
        clearInterval(mockScanInterval);
        mockScanInterval = null;
      }
    }
  }

  async function stop() {
    try {
      await CancelScan(currentScanId);
      scanning = false;
      logs = [...logs, { ts: new Date(), msg: "Cancelled by user." }];
      currentScanId = "";
      // Clear mock interval if running
      if (mockScanInterval) {
        clearInterval(mockScanInterval);
        mockScanInterval = null;
      }
    } catch (e) {
      console.error('Failed to cancel scan:', e);
      toast.danger('Failed to cancel scan');
    }
  }

  onDestroy(() => {
    EventsOff("archaeologist:scan:progress");
    // Clear any active mock scan intervals
    if (mockScanInterval) {
      clearInterval(mockScanInterval);
      mockScanInterval = null;
    }
  });
</script>

<div class="page">
  <header class="header">
    <div class="header-content">
      <h1>Archaeologist.</h1>
      <p class="subtitle">Data Excavation & Analysis</p>
    </div>
    <div class="status-badge" class:active={scanning}>
      <span class="dot"></span>
      {scanning ? "Scanning..." : "Ready"}
    </div>
  </header>

  <div class="layout-split">
    <!-- Sidebar: Controls -->
    <aside class="sidebar">
      <div class="control-panel">
        <div class="field">
          <label for="archaeologist-target-path">Target Path</label>
          <input
            id="archaeologist-target-path"
            type="text"
            bind:value={scanPath}
            placeholder="C:/Data..."
            class="input-clean"
            disabled={scanning}
          />
        </div>

        <div class="field">
          <div id="archaeologist-strategy-label">Scan Strategy</div>
          <div class="radio-grp" role="group" aria-labelledby="archaeologist-strategy-label">
            {#each scanTypes as t}
              <button
                class="radio-item"
                class:selected={selectedType === t.id}
                onclick={() => (selectedType = t.id)}
                disabled={scanning}
              >
                <span class="lbl">{t.label}</span>
                <span class="desc">{t.desc}</span>
              </button>
            {/each}
          </div>
        </div>

        <div class="actions">
          {#if scanning}
            <button class="btn-danger" onclick={stop}>Stop Scan</button>
          {:else}
            <button class="btn-primary" onclick={start}
              >Start Excavation</button
            >
          {/if}
        </div>
      </div>

      <div class="stats-panel">
        <div class="stat">
          <span class="val">{stats.files}</span>
          <span class="lbl">Files Found</span>
        </div>
        <div class="stat">
          <span class="val">{progress.toFixed(0)}%</span>
          <span class="lbl">Progress</span>
        </div>
      </div>
    </aside>

    <!-- Main: Console / Visuals -->
    <main class="main-content">
      <div class="console-box">
        <div class="console-header">System Log</div>
        <div class="console-body">
          {#if logs.length === 0}
            <div class="empty">Ready to scan.</div>
          {/if}
          {#each logs as l}
            <div class="log-line">
              <span class="ts">[{l.ts.toLocaleTimeString()}]</span>
              <span class="msg">{l.msg}</span>
            </div>
          {/each}
          {#if scanning}
            <div class="log-line typing">_</div>
          {/if}
        </div>
      </div>

      <!-- Future Visualization Area -->
      <div class="viz-placeholder">
        <p>Visualization Area (Force Directed Graph)</p>
      </div>
    </main>
  </div>
</div>

<style>
  .page {
    padding: var(--page-padding);
    height: 100vh;
    background: var(--paper);
    color: var(--ink);
    display: flex;
    flex-direction: column;
    box-sizing: border-box;
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: flex-end;
    margin-bottom: var(--space-6);
    flex-shrink: 0;
  }
  h1 {
    font-size: var(--text-5xl);
    font-weight: var(--font-weight-light);
    margin: 0;
    letter-spacing: -0.02em;
  }
  .subtitle {
    color: var(--ink-faint);
    margin-top: var(--space-2);
  }

  .status-badge {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    font-family: var(--font-mono);
    color: var(--ink-light);
  }
  .status-badge .dot {
    width: 8px;
    height: 8px;
    background: var(--ink-light);
    border-radius: 50%;
  }
  .status-badge.active .dot {
    background: #22c55e;
    box-shadow: 0 0 8px #22c55e;
  }
  .status-badge.active {
    color: var(--ink);
  }

  .layout-split {
    display: grid;
    grid-template-columns: 280px 1fr;
    gap: var(--space-8);
    flex: 1;
    min-height: 0;
  }

  .sidebar {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }
  .control-panel {
    background: var(--paper-subtle);
    padding: 20px;
    border-radius: 12px;
  }

  .field {
    margin-bottom: 20px;
  }
  .field label {
    display: block;
    font-size: 11px;
    text-transform: uppercase;
    margin-bottom: 6px;
    color: var(--ink-light);
  }
  .input-clean {
    width: 100%;
    padding: 8px;
    border: 1px solid var(--border-medium);
    border-radius: 6px;
    box-sizing: border-box;
    font-family: var(--font-mono);
    font-size: 12px;
  }

  .radio-grp {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .radio-item {
    background: var(--paper);
    border: 1px solid var(--border-medium);
    padding: 10px;
    border-radius: 8px;
    text-align: left;
    cursor: pointer;
    transition: all 0.2s;
  }
  .radio-item:hover {
    border-color: var(--ink-light);
  }
  .radio-item.selected {
    border-color: var(--ink);
    background: var(--ink);
    color: var(--paper);
  }
  .radio-item.selected .desc {
    color: rgba(255, 255, 255, 0.7);
  }
  .radio-item .lbl {
    font-weight: 500;
    font-size: 13px;
    display: block;
  }
  .radio-item .desc {
    font-size: 11px;
    color: var(--ink-light);
  }

  .actions {
    margin-top: 12px;
  }
  .btn-primary {
    width: 100%;
    padding: 12px;
    background: var(--ink);
    color: var(--paper);
    border: none;
    border-radius: 8px;
    cursor: pointer;
    font-weight: 500;
  }
  .btn-danger {
    width: 100%;
    padding: 12px;
    background: #dc2626;
    color: white;
    border: none;
    border-radius: 8px;
    cursor: pointer;
    font-weight: 500;
  }

  .stats-panel {
    background: var(--paper-subtle);
    padding: 20px;
    border-radius: 12px;
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }
  .stat {
    display: flex;
    flex-direction: column;
    align-items: center;
  }
  .stat .val {
    font-size: 20px;
    font-weight: 500;
  }
  .stat .lbl {
    font-size: 10px;
    text-transform: uppercase;
    color: var(--ink-light);
  }

  .main-content {
    display: flex;
    flex-direction: column;
    gap: 16px;
    min-height: 0;
  }

  .console-box {
    flex: 1;
    background: #1c1c1c;
    color: #e5e5e5;
    border-radius: 12px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    font-family: "JetBrains Mono", monospace;
    font-size: 12px;
  }
  .console-header {
    background: #2a2a2a;
    padding: 8px 12px;
    font-size: 11px;
    text-transform: uppercase;
    color: #888;
    border-bottom: 1px solid #333;
  }
  .console-body {
    padding: 12px;
    overflow-y: auto;
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .log-line {
    display: flex;
    gap: 8px;
  }
  .ts {
    color: #666;
    width: 80px;
    flex-shrink: 0;
  }
  .msg {
    word-break: break-all;
  }
  .typing {
    animation: blink 1s infinite;
  }
  @keyframes blink {
    50% {
      opacity: 0;
    }
  }

  .viz-placeholder {
    height: 150px;
    background: var(--paper-subtle);
    border-radius: 12px;
    border: 1px dashed var(--border-medium);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--ink-light);
    font-size: 12px;
  }
</style>
