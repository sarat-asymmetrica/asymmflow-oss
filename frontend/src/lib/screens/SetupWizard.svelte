<script lang="ts">
  import { onMount } from "svelte";
  import { motionMs } from "$lib/motion";
  import { fade, fly } from "svelte/transition";
  import { toast } from "$lib/stores/toasts";
  import WabiSpinner from "../components/ui/WabiSpinner.svelte";
  import {
    GetFolderPaths } from "../../../wailsjs/go/main/App";
import { UpdateFolderPaths, DetectGPU, DetectOffice, RunInitialScan, CompleteSetup, BrowseFolder, UpdateSettings, ValidateFolder } from "../../../wailsjs/go/main/DocumentsService";

  // Mapped functions
  const GetSuggestedFolders = GetFolderPaths;
  const SetFolders = UpdateFolderPaths;
  const SetAPIKeys = async (keys) => {
    try {
      await UpdateSettings({ ai: keys });
      return { success: true };
    } catch (e) {
      return { success: false, error: e };
    }
  };

  let currentStep = $state(0);
  let loading = $state(false);

  // State
  let folders = $state({
    rfq_path: "",
    offers_path: "",
    invoices_path: "",
    eh_xml_path: "",
    customers_path: "",
    reports_path: "",
  });
  let apiKeys = $state({ aimlapi_key: "", openai_key: "", anthropic_key: "" });
  let gpuConfig = $state({ detected: false, device_name: "", vendor: "", vram_mb: 0 });
  let officeConfig = $state({
    outlook_enabled: false,
    excel_enabled: false,
    word_enabled: false,
    powerpoint_enabled: false,
  });
  let scanResult = $state({ total_files: 0, scan_duration_ms: 0 });

  const steps = [
    { id: "welcome", title: "Welcome" },
    { id: "folders", title: "Folders" },
    { id: "api", title: "AI Connect" },
    { id: "gpu", title: "Hardware" },
    { id: "office", title: "Apps" },
    { id: "scan", title: "Scan" },
    { id: "ready", title: "Finish" },
  ];

  async function loadInitial() {
    try {
      const sug = await GetSuggestedFolders();
      folders = { ...folders, ...sug };
    } catch (e) {
      console.error('Failed to load suggested folders:', e);
      // Use defaults - not critical
    }
  }

  async function nextStep() {
    loading = true;
    try {
      if (currentStep === 1) await SetFolders(folders);
      if (currentStep === 2) await SetAPIKeys(apiKeys);
      if (currentStep === 3) {
        const detected = await DetectGPU();
        gpuConfig = detected || { detected: false, device_name: "", vendor: "", vram_mb: 0 };
      }
      if (currentStep === 4) {
        const detected = await DetectOffice();
        officeConfig = detected || { outlook_enabled: false, excel_enabled: false, word_enabled: false, powerpoint_enabled: false };
      }
      if (currentStep === 5) {
        const res = await RunInitialScan();
        scanResult = {
          total_files: res?.total_files || 0,
          scan_duration_ms: res?.scan_duration_ms || 0,
        };
      }
      if (currentStep < steps.length - 1) currentStep++;
    } catch (e) {
      toast.danger("Error: " + e);
    } finally {
      loading = false;
    }
  }

  function prevStep() {
    if (currentStep > 0) currentStep--;
  }

  async function finish() {
    try {
      await CompleteSetup();
      window.dispatchEvent(new CustomEvent("setup-complete"));
    } catch (e) {
      toast.danger("Setup Error");
    }
  }

  async function browse(key) {
    try {
      const path = await BrowseFolder();
      if (path) folders[key] = path;
    } catch (e) {
      console.error('Failed to browse folder:', e);
      toast.danger('Failed to open folder browser');
    }
  }

  onMount(loadInitial);
</script>

<div class="page">
  <div class="wizard-container">
    <!-- Sidebar -->
    <aside class="wizard-nav">
      <h1>Setup.</h1>
      <div class="steps-list">
        {#each steps as s, i}
          <div
            class="step-item"
            class:active={i === currentStep}
            class:done={i < currentStep}
          >
            <div class="dot">{i < currentStep ? "Done" : i + 1}</div>
            <span class="label">{s.title}</span>
          </div>
          {#if i < steps.length - 1}
            <div class="line" class:done={i < currentStep}></div>
          {/if}
        {/each}
      </div>
    </aside>

    <!-- Main -->
    <main class="wizard-content">
      {#if currentStep === 0}
        <div class="slide" in:fade={{ duration: motionMs(400) }}>
          <h2>Welcome to Asymmetrica.</h2>
          <p class="intro">
            Let's configure your environment for maximum efficiency. We will set
            up your workspace folders, connect AI services, and optimize for
            your hardware.
          </p>
          <div class="features">
            <div class="feat">GPU Accelerated</div>
            <div class="feat">Multi-Model AI</div>
            <div class="feat">Local-First Privacy</div>
          </div>
        </div>
      {:else if currentStep === 1}
        <div class="slide" in:fade={{ duration: motionMs(400) }}>
          <h2>Configure Folders</h2>
          <p class="desc">Where should we store your business data?</p>
          <div class="form-grid">
            {#each Object.keys(folders) as key}
              <div class="field">
                <label for={`folder-${key}`}>{key.replace("_path", "").toUpperCase()}</label>
                <div class="input-row">
                  <input
                    id={`folder-${key}`}
                    type="text"
                    bind:value={folders[key]}
                    class="input-clean"
                  />
                  <button class="btn-icon" onclick={() => browse(key)}
                    >Browse</button
                  >
                </div>
              </div>
            {/each}
          </div>
        </div>
      {:else if currentStep === 2}
        <div class="slide" in:fade={{ duration: motionMs(400) }}>
          <h2>AI Connections</h2>
          <p class="desc">
            Enter API keys for enhanced cloud capabilities (Optional).
          </p>
          <div class="form-stack">
            <div class="field">
              <label for="setup-aimlapi-key">AIMLAPI Key</label>
              <input
                id="setup-aimlapi-key"
                type="password"
                bind:value={apiKeys.aimlapi_key}
                class="input-clean"
                placeholder="sk-..."
              />
            </div>
            <div class="field">
              <label for="setup-openai-key">OpenAI Key</label>
              <input
                id="setup-openai-key"
                type="password"
                bind:value={apiKeys.openai_key}
                class="input-clean"
                placeholder="sk-..."
              />
            </div>
            <div class="field">
              <label for="setup-anthropic-key">Anthropic Key</label>
              <input
                id="setup-anthropic-key"
                type="password"
                bind:value={apiKeys.anthropic_key}
                class="input-clean"
                placeholder="sk-ant-..."
              />
            </div>
          </div>
        </div>
      {:else if currentStep === 3}
        <div class="slide" in:fade={{ duration: motionMs(400) }}>
          <h2>Hardware Detection</h2>
          <div class="result-box">
            {#if loading}
              <WabiSpinner size="md" /> Detecting...
            {:else if gpuConfig.detected}
              <div class="success">
                <h3>GPU Detected</h3>
                <p class="mono">{gpuConfig.device_name}</p>
                <p class="sub">{gpuConfig.vram_mb} MB VRAM</p>
              </div>
            {:else}
              <div class="info">
                No Dedicated GPU detected. Using CPU optimization.
              </div>
            {/if}
          </div>
        </div>
      {:else if currentStep === 4}
        <div class="slide" in:fade={{ duration: motionMs(400) }}>
          <h2>Office Integration</h2>
          <div class="result-box">
            {#if loading}
              <WabiSpinner size="md" /> Checking Apps...
            {:else}
              <div class="apps-grid">
                <div
                  class="app-card"
                  class:enabled={officeConfig.outlook_enabled}
                >
                  <span>Outlook</span>
                  {officeConfig.outlook_enabled ? "Yes" : "No"}
                </div>
                <div
                  class="app-card"
                  class:enabled={officeConfig.excel_enabled}
                >
                  <span>Excel</span>
                  {officeConfig.excel_enabled ? "Yes" : "No"}
                </div>
                <div class="app-card" class:enabled={officeConfig.word_enabled}>
                  <span>Word</span>
                  {officeConfig.word_enabled ? "Yes" : "No"}
                </div>
              </div>
            {/if}
          </div>
        </div>
      {:else if currentStep === 5}
        <div class="slide" in:fade={{ duration: motionMs(400) }}>
          <h2>Initial Scan</h2>
          <div class="result-box">
            {#if loading}
              <WabiSpinner size="md" /> Scanning Workspace...
            {:else}
              <div class="stat-big">
                <span class="num">{scanResult.total_files}</span>
                <span class="lbl">Files Indexed</span>
              </div>
              <p class="mono-sm">Duration: {scanResult.scan_duration_ms}ms</p>
            {/if}
          </div>
        </div>
      {:else if currentStep === 6}
        <div class="slide" in:fade={{ duration: motionMs(400) }}>
          <h2>Ready to Launch</h2>
          <p class="intro">Configuration complete. Your system is optimized.</p>
          <button class="btn-giant" onclick={finish}>Launch Asymmetrica</button
          >
        </div>
      {/if}
    </main>

    <!-- Actions -->
    <div class="wizard-actions">
      {#if currentStep > 0}
        <button class="btn-ghost" onclick={prevStep} disabled={loading}
          >Back</button
        >
      {/if}
      {#if currentStep < steps.length - 1}
        <button class="btn-primary" onclick={nextStep} disabled={loading}>
          {loading ? "Processing..." : "Next"}
        </button>
      {/if}
    </div>
  </div>
</div>

<style>
  .page {
    height: 100vh;
    background: var(--paper);
    color: var(--ink);
    display: flex;
    align-items: center;
    justify-content: center;
    font-family: var(--font-sans);
  }

  .wizard-container {
    width: 900px;
    height: 600px;
    background: var(--paper-subtle);
    border-radius: var(--radius-xl);
    border: 1px solid var(--border-subtle);
    display: grid;
    grid-template-columns: 240px 1fr;
    grid-template-rows: 1fr 80px;
    box-shadow: 0 20px 40px rgba(0, 0, 0, 0.05);
    overflow: hidden;
  }

  .wizard-nav {
    grid-row: 1 / -1;
    background: var(--paper);
    border-right: 1px solid var(--border-subtle);
    padding: 32px;
    display: flex;
    flex-direction: column;
  }
  .wizard-nav h1 {
    font-size: 24px;
    font-weight: 300;
    margin: 0 0 32px;
    letter-spacing: -0.02em;
  }

  .steps-list {
    display: flex;
    flex-direction: column;
    gap: 0;
  }
  .step-item {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 4px;
    opacity: 0.5;
    transition: all 0.2s;
  }
  .step-item.active {
    opacity: 1;
    font-weight: 500;
  }
  .step-item.done {
    opacity: 1;
    color: var(--ink-light);
  }

  .dot {
    width: 24px;
    height: 24px;
    border-radius: 50%;
    border: 1px solid var(--border-medium);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 11px;
  }
  .step-item.active .dot {
    background: var(--ink);
    color: var(--paper);
    border-color: var(--ink);
  }
  .step-item.done .dot {
    background: var(--ink-subtle);
    color: var(--ink);
    border-color: transparent;
  }

  .line {
    width: 1px;
    height: 16px;
    background: var(--border-medium);
    margin: 0 0 4px 12px;
  }
  .line.done {
    background: var(--ink);
  }

  .wizard-content {
    padding: 48px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }

  .slide {
    max-width: 500px;
    margin: 0 auto;
    width: 100%;
  }
  .slide h2 {
    font-size: 28px;
    font-weight: 300;
    margin: 0 0 16px;
  }
  .intro,
  .desc {
    color: var(--ink-light);
    line-height: 1.5;
    margin-bottom: 32px;
  }

  .features {
    display: flex;
    gap: 12px;
    flex-wrap: wrap;
  }
  .feat {
    padding: 8px 16px;
    background: var(--paper);
    border: 1px solid var(--border-medium);
    border-radius: 20px;
    font-size: 12px;
  }

  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }
  .field {
    margin-bottom: 12px;
  }
  .field label {
    display: block;
    font-size: 11px;
    text-transform: uppercase;
    margin-bottom: 4px;
    color: var(--ink-light);
  }

  .input-row {
    display: flex;
    gap: 8px;
  }
  .input-clean {
    width: 100%;
    padding: 8px;
    border: 1px solid var(--border-medium);
    border-radius: 6px;
    box-sizing: border-box;
  }
  .btn-icon {
    background: var(--paper);
    border: 1px solid var(--border-medium);
    border-radius: 6px;
    cursor: pointer;
  }

  .result-box {
    background: var(--paper);
    padding: 32px;
    border-radius: 12px;
    text-align: center;
    border: 1px solid var(--border-subtle);
  }
  .mono {
    font-family: var(--font-mono);
    font-size: 16px;
    margin: 8px 0;
  }
  .sub {
    color: var(--ink-light);
    font-size: 13px;
  }

  .apps-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }
  .app-card {
    padding: 12px;
    border: 1px solid var(--border-medium);
    border-radius: 8px;
    display: flex;
    justify-content: space-between;
    opacity: 0.5;
  }
  .app-card.enabled {
    opacity: 1;
    border-color: var(--ink);
    background: var(--paper-subtle);
    font-weight: 500;
  }

  .stat-big {
    font-size: 40px;
    font-weight: 300;
    display: flex;
    flex-direction: column;
  }
  .lbl {
    font-size: 12px;
    text-transform: uppercase;
    color: var(--ink-light);
  }

  .btn-giant {
    padding: 16px 32px;
    font-size: 18px;
    background: var(--ink);
    color: var(--paper);
    border: none;
    border-radius: 30px;
    cursor: pointer;
    transition: transform 0.2s;
    width: 100%;
  }
  .btn-giant:hover {
    transform: scale(1.02);
  }

  .wizard-actions {
    border-top: 1px solid var(--border-subtle);
    padding: 0 48px;
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 16px;
  }

  .btn-primary {
    padding: 10px 24px;
    background: var(--ink);
    color: var(--paper);
    border: none;
    border-radius: 20px;
    cursor: pointer;
    font-size: 14px;
  }
  .btn-ghost {
    background: transparent;
    border: none;
    cursor: pointer;
    color: var(--ink-light);
    font-size: 14px;
  }
</style>
