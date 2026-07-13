<script lang="ts">
  import { run } from 'svelte/legacy';
  import { motionMs } from "$lib/motion";

  import { createEventDispatcher, onMount } from "svelte";
  import { fade } from "svelte/transition";
  import { toast } from "$lib/stores/toasts";
  import { permissions } from "$lib/stores/authContext";
  import { initI18n, localeOptions, setLocale, t, type Locale } from "$lib/i18n";
  import { setTextScale, setTextScalePreset, textScale, textScalePreset, type TextScalePreset } from "$lib/stores/textScale";
  import { soundOnPaidEnabled } from "$lib/stores/soundSettings";
  import { formatNumber } from "$lib/utils/formatters";
  import WabiSpinner from "$lib/components/ui/WabiSpinner.svelte";
  import { confirm } from "$lib/stores/confirm";
  import {
    UpdateSettings } from "../../../wailsjs/go/main/App";
import { GetSettings, BrowseFolder, TestAIConnection, DetectGPU, GetActiveCurrencyRates, SetExchangeRate, GetSupportedCurrencies } from "../../../wailsjs/go/main/DocumentsService";
import { TestMistralConnection } from "../../../wailsjs/go/main/ButlerService";
import { ImportTallyInvoices, ImportTallyPurchases, ImportARDefaulters, ImportSupplierPaymentsFromFile, ImportAllTallyData, GenerateProfitAndLoss, GenerateBalanceSheet, GetFinancialReportYears, GetAllBankAccounts, CreateBankAccount, UpdateBankAccount, DeleteBankAccount } from "../../../wailsjs/go/main/FinanceService";
import { TestSupabaseConnection, GetSyncHealth } from "../../../wailsjs/go/main/SyncServiceBinding";
import { GetCurrentUserRole, GetPhase7RolloutStatus, GetPilotReadinessSummary, ListCollaborativePendingOperations, ListPilotReadinessRows, RetryCollaborativePendingOperation, RetryCollaborativePendingOperations, RerunPhase7FollowUpBackfill, GetBackupInfo, GetBackupPolicy, SaveBackupPolicy, TriggerBackup, TriggerCollaborativeSyncNow, ExportPilotSupportBundle, GetBuildInfo } from "../../../wailsjs/go/main/InfraService";


  interface Props {
    // Props
    embedded?: boolean;
    // Wave 9.4 C1: supports deep-linking straight to a section, e.g.
    // dispatch('navigate', { screen: 'settings', section: 'accounts' }) —
    // mirrors the params?.tab convention FinanceHub already uses.
    params?: { section?: string };
  }

  let { embedded = false, params = {} }: Props = $props();
  run(() => {
    embedded;
  });
  const dispatch = createEventDispatcher();

  // State
  let loading = $state(true);
  let saving = $state(false);
  let activeSection = $state("general");
  let showKeyInput = $state(false); // toggle for AIML API key visibility
  let showMistralKey = $state(false); // toggle for Mistral API key visibility
  let testingMistral = $state(false);

  let settings = $state({
    companyName: "",
    currency: "BHD",
    language: "en",
    theme: "light",
    folders: {
      rfq_path: "",
      offers_path: "",
      invoices_path: "",
      eh_xml_path: "",
      customers_path: "",
      reports_path: "",
    },
    apiKeys: {
      aimlapi_key: "",
      aiml_model: "grok-4.1",
      openai_key: "",
      anthropic_key: "",
      mistral_key: "",
      azure_endpoint: "",
    },
    gpu: {
      detected: false,
      vendor: "",
      device_name: "",
      vram_mb: 0,
      use_gpu: true,
    },
    office: { outlook_enabled: false, excel_enabled: false },
    business: { default_margin: 20, vat_rate: 10 },
    security: { session_timeout_minutes: 30 },
    sounds: { sound_on_paid_enabled: true },
  });

  let userRole = $state('staff');
  let importYear = $state(2025);
  let importing = $state(false);
  let importResult: any = $state(null);
  let supabaseForm = $state({ url: '', anonKey: '', serviceKey: '', dbHost: '', dbPort: '5432', dbName: 'postgres', dbPassword: '' });
  let testingConnection = $state(false);
  let reportYear = $state(2025);
  let generatingReport = $state(false);
  let plReport: any = $state(null);
  let bsReport: any = $state(null);
  let availableYears: number[] = $state([]);
  let syncHealth: any = $state(null);
  let backupInfo: any = $state(null);
  let backupPolicy: any = $state(null);
  let backupRunning = $state(false);
  let savingBackupPolicy = $state(false);
  let backupAutoEnabled = $state(true);
  let backupFrequencyDays = $state(7);
  let rolloutStatus: any = $state(null);
  let rolloutOps: any[] = $state([]);
  let rolloutOpsFilter = $state('active');
  let loadingRolloutOps = $state(false);
  let rolloutActionRunning = $state(false);
  let pilotSummary: any = $state(null);
  let pilotRows: any[] = $state([]);
  let buildInfo: any = $state(null);
  let pilotOnlyIssues = $state(true);
  let loadingPilotReadiness = $state(false);
  let exportingPilotBundle = $state(false);
  const textScalePresets: Array<{ id: TextScalePreset; label: string; sample: string }> = [
    { id: "standard", label: "Standard", sample: "A" },
    { id: "comfortable", label: "Comfortable", sample: "A+" },
    { id: "large", label: "Large", sample: "A++" },
  ];

  // Currency state
  let currencyRates: any[] = $state([]);
  let supportedCurrencies: any[] = $state([]);
  let editingCurrency = $state('');
  let newRate = $state(0);
  let rateNotes = $state('');
  let savingRate = $state(false);

  // Wave 9.4 C1: bank-account CRUD relocated here from BankReconciliationScreen,
  // which now only keeps the read-only active-account picker for statement
  // import/matching. Gated on finance:create (mirrors the server-side
  // finance:create/finance:delete checks CreateBankAccount/UpdateBankAccount/
  // DeleteBankAccount already enforce).
  interface BankAccount {
    id: string;
    division?: string;
    bank_name: string;
    account_name?: string;
    account_number: string;
    iban?: string;
    swift_bic?: string;
    currency: string;
    is_active: boolean;
  }
  let bankAccounts: any[] = $state([]);
  let loadingBankAccounts = $state(false);
  let editingBankAccount: BankAccount | null = $state(null);
  let bankAccountFormData = $state({
    division: 'Acme Instrumentation',
    bank_name: '',
    account_name: '',
    account_number: '',
    iban: '',
    swift_bic: '',
    currency: 'BHD',
    booking_rate: 0
  });
  let savingBankAccount = $state(false);

  let isAdmin = $derived(userRole === 'admin');
  function hasPermission(permission: string): boolean {
    const permissionList = Array.isArray($permissions) ? $permissions : [];
    if (permissionList.length === 0) return false;
    if (permissionList.includes('*')) return true;
    if (permissionList.includes(permission)) return true;
    const [resource] = permission.split(':');
    return permissionList.includes(`${resource}:*`);
  }
  let canOpenDeployment = $derived(hasPermission('settings:update'));
  let canManageBankAccounts = $derived(hasPermission('finance:create'));
  let sections = $derived([
    { id: "general", label: t("settings.general") },
    ...(canOpenDeployment ? [{ id: "deployment", label: "Deployment" }] : []),
    { id: "folders", label: "Directories" },
    { id: "ai", label: "AI & Intelligence" },
    { id: "business", label: t("settings.business_rules") },
    { id: "currency", label: t("settings.currency_rates") },
    ...(canManageBankAccounts ? [{ id: "accounts", label: "Bank Accounts" }] : []),
    ...(isAdmin ? [{ id: "data", label: t("settings.data_import") }] : []),
    ...(isAdmin ? [{ id: "reports", label: "Financial Reports" }] : []),
    ...(isAdmin ? [{ id: "sync", label: "Supabase Sync" }] : []),
  ]);

  function openDeploymentWorkspace() {
    dispatch("navigate", { screen: "deployment" });
  }

  // Wave 9.4 C1: honor a deep-linked section (e.g. from BankReconciliationScreen's
  // "Manage bank accounts" affordance) once, on mount / whenever params change.
  run(() => {
    if (params?.section === 'accounts' && canManageBankAccounts) {
      activeSection = 'accounts';
      void loadBankAccounts();
    } else if (params?.section && params.section !== 'accounts') {
      activeSection = params.section;
    }
  });

  async function importTallyAll() {
    if (importYear < 2000 || importYear > 2030) {
      toast.danger('Year must be between 2000 and 2030');
      return;
    }
    importing = true;
    importResult = null;
    try {
      const res = await ImportAllTallyData();
      importResult = res;
      toast.success(`Import complete: ${res.imported} records imported`);
    } catch (e) {
      toast.danger('Import failed: ' + e);
    } finally {
      importing = false;
    }
  }

  async function importInvoices() {
    if (importYear < 2000 || importYear > 2030) {
      toast.danger('Year must be between 2000 and 2030');
      return;
    }
    importing = true;
    try {
      const res = await ImportTallyInvoices(importYear);
      importResult = res;
      toast.success(`Invoices: ${res.imported} imported, ${res.duplicates} duplicates`);
    } catch (e) {
      toast.danger('Invoice import failed: ' + e);
    } finally {
      importing = false;
    }
  }

  async function importPurchases() {
    if (importYear < 2000 || importYear > 2030) {
      toast.danger('Year must be between 2000 and 2030');
      return;
    }
    importing = true;
    try {
      const res = await ImportTallyPurchases(importYear);
      importResult = res;
      toast.success(`Purchases: ${res.imported} imported, ${res.duplicates} duplicates`);
    } catch (e) {
      toast.danger('Purchase import failed: ' + e);
    } finally {
      importing = false;
    }
  }

  async function importARDefaulters() {
    importing = true;
    try {
      const res = await ImportARDefaulters();
      importResult = res;
      toast.success(`AR Defaulters: ${res.imported} updated`);
    } catch (e) {
      toast.danger('AR import failed: ' + e);
    } finally {
      importing = false;
    }
  }

  async function importSupplierPayments() {
    importing = true;
    try {
      const res = await ImportSupplierPaymentsFromFile();
      importResult = res;
      toast.success(`Supplier Payments: ${res.imported} imported`);
    } catch (e) {
      toast.danger('Supplier payments import failed: ' + e);
    } finally {
      importing = false;
    }
  }

  async function loadCurrencyRates() {
    try {
      currencyRates = await GetActiveCurrencyRates() || [];
      supportedCurrencies = await GetSupportedCurrencies() || [];
    } catch (err) {
      console.error('Failed to load currency rates:', err);
    }
  }

  async function handleSaveRate() {
    if (!editingCurrency || newRate <= 0) {
      toast.danger('Please select a currency and enter a valid rate');
      return;
    }
    savingRate = true;
    try {
      await SetExchangeRate(editingCurrency, newRate, new Date().toISOString(), rateNotes);
      toast.success(`Exchange rate for ${editingCurrency} updated to ${newRate} BHD`);
      editingCurrency = '';
      newRate = 0;
      rateNotes = '';
      await loadCurrencyRates();
    } catch (err) {
      toast.danger('Failed to save rate: ' + err);
    } finally {
      savingRate = false;
    }
  }

  function getCurrencyName(code: string): string {
    const currency = supportedCurrencies.find(c => c.code === code);
    return currency?.name || code;
  }

  // Wave 9.4 C1: bank-account CRUD (moved from BankReconciliationScreen).
  async function loadBankAccounts() {
    loadingBankAccounts = true;
    try {
      const result = await GetAllBankAccounts();
      bankAccounts = result || [];
    } catch (err) {
      console.error('Failed to load bank accounts:', err);
      toast.danger('Failed to load bank accounts');
    } finally {
      loadingBankAccounts = false;
    }
  }

  function resetBankAccountForm() {
    editingBankAccount = null;
    bankAccountFormData = {
      division: 'Acme Instrumentation',
      bank_name: '',
      account_name: '',
      account_number: '',
      iban: '',
      swift_bic: '',
      currency: 'BHD',
      booking_rate: 0
    };
  }

  function editBankAccount(account: BankAccount) {
    editingBankAccount = account;
    bankAccountFormData = {
      division: account.division || 'Acme Instrumentation',
      bank_name: account.bank_name,
      account_name: account.account_name || '',
      account_number: account.account_number,
      iban: account.iban || '',
      swift_bic: account.swift_bic || '',
      currency: account.currency || 'BHD',
      booking_rate: (account as any).booking_rate || 0
    };
  }

  async function handleSaveBankAccount() {
    if (savingBankAccount) return;

    if (!bankAccountFormData.bank_name) {
      toast.warning('Bank name is required');
      return;
    }
    if (!bankAccountFormData.account_number) {
      toast.warning('Account number is required');
      return;
    }

    savingBankAccount = true;
    try {
      if (editingBankAccount) {
        await UpdateBankAccount(editingBankAccount.id, bankAccountFormData);
        toast.success('Account updated successfully');
      } else {
        await CreateBankAccount(bankAccountFormData as any);
        toast.success('Account created successfully');
      }
      await loadBankAccounts();
      resetBankAccountForm();
    } catch (err) {
      console.error('Failed to save account:', err);
      toast.danger(`Failed to save account: ${err}`);
    } finally {
      savingBankAccount = false;
    }
  }

  async function handleDeleteBankAccount(account: BankAccount) {
    if (!(await confirm.ask({
      title: 'Deactivate Bank Account',
      message: `Are you sure you want to deactivate ${account.bank_name} - ${account.account_number}?`,
      confirmLabel: 'Deactivate',
      variant: 'danger'
    }))) {
      return;
    }

    try {
      await DeleteBankAccount(account.id);
      toast.success('Account deactivated successfully');
      await loadBankAccounts();
    } catch (err) {
      console.error('Failed to deactivate account:', err);
      toast.danger(`Failed to deactivate account: ${err}`);
    }
  }

  async function testSupabase() {
    if (!supabaseForm.dbHost || !supabaseForm.dbPassword) {
      toast.warning('DB Host and Password are required');
      return;
    }
    testingConnection = true;
    try {
      const ok = await TestSupabaseConnection(
        supabaseForm.dbHost,
        supabaseForm.dbPort || '5432',
        'postgres',
        supabaseForm.dbPassword,
        supabaseForm.dbName || 'postgres',
        'require'
      );
      if (ok) {
        toast.success('Supabase connection successful');
      } else {
        toast.danger('Connection failed - check credentials');
      }
    } catch (e) {
      toast.danger('Connection test failed: ' + e);
    } finally {
      testingConnection = false;
    }
  }

  async function loadSettings() {
    loading = true;
    try {
      const res = await GetSettings();
      if (res) settings = { ...settings, ...res };
      await initI18n(settings.language);
      try { userRole = await GetCurrentUserRole() || 'staff'; } catch { userRole = 'staff'; }
      try { buildInfo = await GetBuildInfo(); } catch { buildInfo = null; }
      try { availableYears = await GetFinancialReportYears() || []; } catch { availableYears = []; }
      await refreshRolloutData();
    } catch (e) {
      toast.danger("Failed to load settings");
    } finally {
      loading = false;
    }
    // Detect GPU in background (non-blocking)
    detectHardware();
  }

  async function refreshRolloutData() {
    try { syncHealth = await GetSyncHealth(); } catch { syncHealth = null; }
    await loadBackupState();
    try { rolloutStatus = await GetPhase7RolloutStatus(); } catch { rolloutStatus = null; }
    await Promise.all([loadRolloutOps(), loadPilotReadiness()]);
  }

  async function loadBackupState() {
    try {
      backupInfo = await GetBackupInfo();
    } catch {
      backupInfo = null;
    }
    try {
      backupPolicy = await GetBackupPolicy();
      backupAutoEnabled = backupPolicy?.auto_backup_enabled ?? true;
      backupFrequencyDays = backupPolicy?.frequency_days ?? 7;
    } catch {
      backupPolicy = null;
      backupAutoEnabled = true;
      backupFrequencyDays = 7;
    }
  }

  async function runManualBackup() {
    backupRunning = true;
    try {
      const result = await TriggerBackup();
      if (result?.success) {
        toast.success('Backup created');
        await loadBackupState();
      } else {
        toast.danger(result?.error || 'Backup failed');
      }
    } catch (err) {
      toast.danger('Backup failed: ' + String(err));
    } finally {
      backupRunning = false;
    }
  }

  async function saveBackupPolicy() {
    savingBackupPolicy = true;
    try {
      backupPolicy = await SaveBackupPolicy(backupAutoEnabled, Number(backupFrequencyDays) || 7);
      backupAutoEnabled = backupPolicy?.auto_backup_enabled ?? backupAutoEnabled;
      backupFrequencyDays = backupPolicy?.frequency_days ?? backupFrequencyDays;
      toast.success('Backup schedule saved');
      await loadBackupState();
    } catch (err) {
      toast.danger('Backup schedule failed: ' + String(err));
    } finally {
      savingBackupPolicy = false;
    }
  }

  async function loadRolloutOps() {
    loadingRolloutOps = true;
    try {
      rolloutOps = await ListCollaborativePendingOperations(rolloutOpsFilter, 20) || [];
    } catch {
      rolloutOps = [];
    } finally {
      loadingRolloutOps = false;
    }
  }

  async function loadPilotReadiness() {
    loadingPilotReadiness = true;
    try {
      pilotSummary = await GetPilotReadinessSummary();
    } catch {
      pilotSummary = null;
    }
    try {
      pilotRows = await ListPilotReadinessRows(pilotOnlyIssues) || [];
    } catch {
      pilotRows = [];
    } finally {
      loadingPilotReadiness = false;
    }
  }

  async function handleRetryQueue(status: string) {
    rolloutActionRunning = true;
    try {
      const result = await RetryCollaborativePendingOperations(status, 100);
      toast.success(result?.message || 'Collaborative operations re-queued');
      await refreshRolloutData();
    } catch (e) {
      toast.danger('Failed to retry collaborative operations: ' + (e?.message || e));
    } finally {
      rolloutActionRunning = false;
    }
  }

  async function handleRetrySingle(operationID: string) {
    rolloutActionRunning = true;
    try {
      await RetryCollaborativePendingOperation(operationID);
      toast.success('Collaborative operation re-queued');
      await refreshRolloutData();
    } catch (e) {
      toast.danger('Failed to retry collaborative operation: ' + (e?.message || e));
    } finally {
      rolloutActionRunning = false;
    }
  }

  async function handleRerunBackfill() {
    rolloutActionRunning = true;
    try {
      const result = await RerunPhase7FollowUpBackfill();
      toast.success(result?.message || 'Legacy follow-up backfill completed');
      await refreshRolloutData();
    } catch (e) {
      toast.danger('Failed to re-run legacy follow-up backfill: ' + (e?.message || e));
    } finally {
      rolloutActionRunning = false;
    }
  }

  async function handleTriggerCollaborativeSync() {
    rolloutActionRunning = true;
    try {
      await TriggerCollaborativeSyncNow();
      toast.success('Collaborative sync completed');
      await refreshRolloutData();
    } catch (e) {
      toast.danger('Collaborative sync failed: ' + (e?.message || e));
    } finally {
      rolloutActionRunning = false;
    }
  }

  async function handleExportPilotBundle() {
    exportingPilotBundle = true;
    try {
      const result = await ExportPilotSupportBundle();
      toast.success(`Pilot support bundle exported to ${result?.path || 'reports directory'}`);
      await refreshRolloutData();
    } catch (e) {
      toast.danger('Failed to export pilot support bundle: ' + (e?.message || e));
    } finally {
      exportingPilotBundle = false;
    }
  }

  function formatIssue(issue: string): string {
    if (!issue) return 'Unknown issue';
    return issue.replace(/_/g, ' ').replace(/\b\w/g, (match) => match.toUpperCase());
  }

  function getPilotRowIssueSummary(row: any): string {
    const issues = Array.isArray(row?.issues) ? row.issues : [];
    if (issues.length === 0) {
      return 'Ready';
    }
    return issues.map((issue) => formatIssue(issue)).join(', ');
  }

  async function detectHardware() {
    try {
      const gpu = await DetectGPU();
      settings.gpu = { ...settings.gpu, ...gpu };
    } catch (e) {
      console.error('Failed to detect GPU:', e);
      // Silently fail - GPU detection is optional
    }
  }

  async function save() {
    saving = true;
    try {
      await UpdateSettings(settings as Record<string, any>);
      await setLocale(settings.language as Locale);
      soundOnPaidEnabled.set(settings.sounds.sound_on_paid_enabled);
      toast.success("Settings saved");
    } catch (e) {
      toast.danger("Save failed");
    } finally {
      saving = false;
    }
  }

  function handleTextScaleSlider(event: Event) {
    const input = event.currentTarget as HTMLInputElement;
    setTextScale(Number(input.value) / 100);
  }

  async function browse(field) {
    try {
      const path = await BrowseFolder();
      if (path) settings.folders[field] = path;
    } catch (e) {
      console.error('Failed to browse folder:', e);
      toast.danger('Failed to open folder browser');
    }
  }

  async function handleLocaleChange(event: Event) {
    const select = event.currentTarget as HTMLSelectElement;
    settings.language = select.value;
    await setLocale(select.value as Locale);
  }

  async function testAI() {
    try {
      const key = settings.apiKeys.aimlapi_key || '';
      if (!key || key.includes('*')) {
        toast.warning('Enter a valid API key first');
        return;
      }
      await TestAIConnection("aimlapi", key);
      toast.success("AI Connection Valid");
    } catch (e) {
      toast.danger("Test failed: " + (e?.message || e));
    }
  }

  async function testMistral() {
    // First, save settings to ensure the key is persisted before testing
    const key = settings.apiKeys.mistral_key || '';
    if (!key || key.includes('*')) {
      toast.warning('Enter a valid Mistral API key first');
      return;
    }

    testingMistral = true;
    try {
      // Save settings first so the backend has the key
      await save();

      // Test the connection
      const result = await TestMistralConnection();
      if (result) {
        toast.success("Mistral API connection successful");
      } else {
        toast.danger("Mistral API connection failed");
      }
    } catch (e: any) {
      toast.danger("Mistral test failed: " + (e?.message || e));
    } finally {
      testingMistral = false;
    }
  }

  async function generatePL() {
    if (reportYear < 2000 || reportYear > 2030) {
      toast.danger('Year must be between 2000 and 2030');
      return;
    }
    generatingReport = true;
    plReport = null;
    try {
      plReport = await GenerateProfitAndLoss(reportYear);
      toast.success(`P&L generated for ${reportYear}`);
    } catch (e) {
      toast.danger('P&L generation failed: ' + e);
    } finally {
      generatingReport = false;
    }
  }

  async function generateBS() {
    if (reportYear < 2000 || reportYear > 2030) {
      toast.danger('Year must be between 2000 and 2030');
      return;
    }
    generatingReport = true;
    bsReport = null;
    try {
      bsReport = await GenerateBalanceSheet(reportYear);
      toast.success(`Balance Sheet generated for ${reportYear}`);
    } catch (e) {
      toast.danger('Balance Sheet generation failed: ' + e);
    } finally {
      generatingReport = false;
    }
  }

  onMount(async () => {
    await loadSettings();
    await loadCurrencyRates();
  });
</script>

<div class="page">
  <header class="header">
    <div class="header-content">
      <h1>{t("settings.title")}.</h1>
      <p class="subtitle">{t("settings.subtitle")}</p>
    </div>
    <div class="actions">
      <button class="btn-primary" onclick={save} disabled={saving || loading}>
        {saving ? t("common.saving") : t("common.save_changes")}
      </button>
    </div>
  </header>

  <div class="layout-split">
    <!-- Sidebar Navigation -->
    <aside class="sidebar">
      <nav>
        {#each sections as sec}
          <button
            class="nav-item"
            class:active={activeSection === sec.id}
            onclick={() => { activeSection = sec.id; if (sec.id === 'accounts') void loadBankAccounts(); }}
          >
            {sec.label}
          </button>
        {/each}
      </nav>

      <div class="sys-info">
        <span>Ver: {buildInfo?.version || '0.1.0-alpha.1'} ({buildInfo?.channel || 'alpha'})</span>
        {#if buildInfo?.git_commit}
          <span>Build: {String(buildInfo.git_commit).slice(0, 8)}</span>
        {/if}
        <span>GPU: {settings.gpu.detected ? "Active" : "N/A"}</span>
      </div>
    </aside>

    <!-- Content Area -->
    <main class="content-panel">
      {#if loading}
        <div class="loading"><WabiSpinner /></div>
      {:else if activeSection === "general"}
        <div class="section" in:fade={{ duration: motionMs(400) }}>
          <h3>{t("settings.general")}</h3>
          {#if canOpenDeployment}
            <div class="deployment-card">
              <div class="deployment-card-copy">
                <h4>Deployment Workspace</h4>
                <p>Open rollout and production-support controls directly from Settings.</p>
              </div>
              <button class="btn-primary" onclick={openDeploymentWorkspace}>
                Open Deployment Workspace
              </button>
            </div>
          {/if}
          <div class="readability-card">
            <div class="readability-copy">
              <h4>Text Size</h4>
              <p>Applies immediately across cards, forms, tables, modals, and workflow screens on this computer.</p>
            </div>
            <div class="readability-controls">
              <div class="text-preset-buttons" aria-label="Text size preset">
                {#each textScalePresets as preset}
                  <button
                    type="button"
                    class:active={$textScalePreset === preset.id}
                    aria-pressed={$textScalePreset === preset.id}
                    onclick={() => setTextScalePreset(preset.id)}
                  >
                    <span>{preset.sample}</span>
                    {preset.label}
                  </button>
                {/each}
              </div>
              <label class="text-scale-slider" for="settings-text-scale">
                <span>{Math.round($textScale * 100)}%</span>
                <input
                  id="settings-text-scale"
                  type="range"
                  min="100"
                  max="125"
                  step="1"
                  value={Math.round($textScale * 100)}
                  oninput={handleTextScaleSlider}
                />
              </label>
            </div>
          </div>
          <div class="form-group">
            <label for="settings-language">{t("settings.language")}</label>
            <select
              id="settings-language"
              bind:value={settings.language}
              class="input-clean"
              onchange={handleLocaleChange}
            >
              {#each localeOptions as locale}
                <option value={locale.value}>{locale.label}</option>
              {/each}
            </select>
          </div>
          <div class="form-group">
            <label for="settings-company-name">Company Name</label>
            <input
              id="settings-company-name"
              type="text"
              bind:value={settings.companyName}
              class="input-clean"
            />
          </div>
          <div class="form-group">
            <label for="settings-session-timeout">Session Timeout (minutes)</label>
            <input
              id="settings-session-timeout"
              type="number"
              min="5"
              max="480"
              bind:value={settings.security.session_timeout_minutes}
              class="input-clean"
            />
            <p class="hint">Idle time before an automatic sign-out (5–480 minutes). Applies immediately when saved.</p>
          </div>
          <div class="row">
            <div class="form-group half">
              <label for="settings-base-currency">Base Currency</label>
              <select id="settings-base-currency" bind:value={settings.currency} class="input-clean">
                <option value="BHD">BHD (Bahraini Dinar)</option>
                <option value="USD">USD (US Dollar)</option>
                <option value="EUR">EUR (Euro)</option>
                <option value="CHF">CHF (Swiss Franc)</option>
                <option value="SAR">SAR (Saudi Riyal)</option>
                <option value="AED">AED (UAE Dirham)</option>
              </select>
            </div>
            <!-- Dark mode commented out - not yet functional
            <div class="form-group half">
              <label>Interface Theme</label>
              <select bind:value={settings.theme} class="input-clean">
                <option value="light">Light (Milestoners)</option>
                <option value="dark">Dark (Onyx)</option>
                <option value="system">System Default</option>
              </select>
            </div>
            -->
          </div>

          <h3>Office Integration</h3>
          <div class="toggle-row">
            <input
              type="checkbox"
              id="outlook"
              bind:checked={settings.office.outlook_enabled}
            />
            <label for="outlook">Enable Outlook Integration</label>
          </div>
          <div class="toggle-row">
            <input
              type="checkbox"
              id="excel"
              bind:checked={settings.office.excel_enabled}
            />
            <label for="excel">Enable Excel Automations</label>
          </div>

          <h3>Sound</h3>
          <div class="toggle-row">
            <input
              type="checkbox"
              id="sound-on-paid"
              bind:checked={settings.sounds.sound_on_paid_enabled}
            />
            <label for="sound-on-paid">Play a sound when an invoice is paid in full</label>
          </div>
          <p class="hint">The application's one sound — a quiet settle when your own posting click fully pays a customer invoice. On by default.</p>
        </div>
      {:else if activeSection === "folders"}
        <div class="section" in:fade={{ duration: motionMs(400) }}>
          <h3>Directory Mapping</h3>
          <p class="hint">Define where the system looks for documents.</p>

          {#each Object.keys(settings.folders) as key}
            {#if key !== "extra_paths"}
              <div class="form-group">
                <label for={`settings-folder-${key}`}>{key.replace("_path", "").toUpperCase()}</label>
                <div class="input-group">
                  <input
                    id={`settings-folder-${key}`}
                    type="text"
                    bind:value={settings.folders[key]}
                    class="input-clean"
                    readonly
                  />
                  <button class="btn-ghost" onclick={() => browse(key)}
                    >Browse</button
                  >
                </div>
              </div>
            {/if}
          {/each}
        </div>
      {:else if activeSection === "ai"}
        <div class="section" in:fade={{ duration: motionMs(400) }}>
          <h3>Artificial Intelligence</h3>

          <div class="form-group">
            <label for="settings-aiml-key">AIML API Key <span style="font-size:0.8em; color: var(--accent);">(Butler primary — Grok 4)</span></label>
            <p style="font-size:0.8em; color: var(--text-muted); margin: 2px 0 6px;">Powers Butler AI chat with Grok's large context window. Get key at api.aimlapi.com</p>
            <div class="input-group">
              {#if showKeyInput}
                <input
                  id="settings-aiml-key"
                  type="text"
                  bind:value={settings.apiKeys.aimlapi_key}
                  class="input-clean"
                  placeholder="Enter AIML API key..."
                />
              {:else}
                <input
                  id="settings-aiml-key"
                  type="password"
                  bind:value={settings.apiKeys.aimlapi_key}
                  class="input-clean"
                  placeholder="••••••••••••••••"
                />
              {/if}
              <button
                class="btn-icon"
                onclick={() => (showKeyInput = !showKeyInput)}>{showKeyInput ? 'Hide' : 'Show'}</button
              >
            </div>
          </div>

          <div class="form-group">
            <label for="settings-grok-model">Grok Model</label>
            <p style="font-size:0.8em; color: var(--text-muted); margin: 2px 0 6px;">Select the backend model for Butler chat. It will fallback automatically if the model is unavailable.</p>
            <select id="settings-grok-model" class="input-clean" bind:value={settings.apiKeys.aiml_model}>
              <option value="grok-4.1">grok-4.1</option>
              <option value="grok-4">grok-4</option>
              <option value="grok-2-vision-1212">grok-2-vision-1212</option>
              <option value="grok-2-1212">grok-2-1212</option>
              <option value="gpt-4o-mini">gpt-4o-mini</option>
            </select>
          </div>

          <div class="form-group" style="margin-top: 16px;">
            <label for="settings-mistral-key">Mistral API Key <span style="font-size:0.8em; color: var(--text-muted);">(OCR/document analysis + fallback chat)</span></label>
            <p style="font-size:0.8em; color: var(--text-muted); margin: 2px 0 6px;">Used for "Analyze with Butler" document OCR. Falls back to this if AIML key not set.</p>
            <div class="input-group">
              {#if showMistralKey}
                <input
                  id="settings-mistral-key"
                  type="text"
                  bind:value={settings.apiKeys.mistral_key}
                  class="input-clean"
                  placeholder="Enter Mistral API key..."
                />
              {:else}
                <input
                  id="settings-mistral-key"
                  type="password"
                  bind:value={settings.apiKeys.mistral_key}
                  class="input-clean"
                  placeholder="••••••••••••••••"
                />
              {/if}
              <button
                class="btn-icon"
                onclick={() => (showMistralKey = !showMistralKey)}>{showMistralKey ? 'Hide' : 'Show'}</button
              >
            </div>
          </div>

          <div class="action-row" style="gap: 8px;">
            <button class="btn-secondary" onclick={testAI}>Test AIML Connectivity</button>
            <button class="btn-secondary" onclick={testMistral} disabled={testingMistral}>
              {testingMistral ? 'Testing...' : 'Test Mistral Connection'}
            </button>
          </div>

          <h3>Hardware Acceleration</h3>
          <div class="gpu-card">
            <div class="gpu-header">
              <span class="gpu-icon">GPU</span>
              <span>{settings.gpu.device_name || "No GPU Detected"}</span>
            </div>
            {#if settings.gpu.detected}
              <div class="gpu-stats">
                <span>VRAM: {settings.gpu.vram_mb} MB</span>
                <span>Vendor: {settings.gpu.vendor}</span>
              </div>
              <div class="toggle-row">
                <input
                  type="checkbox"
                  id="use_gpu"
                  bind:checked={settings.gpu.use_gpu}
                />
                <label for="use_gpu">Use GPU for Local Inference</label>
              </div>
            {/if}
          </div>
        </div>
      {:else if activeSection === "business"}
        <div class="section" in:fade={{ duration: motionMs(400) }}>
          <h3>Financial Rules</h3>
          <div class="row">
            <div class="form-group half">
              <label for="settings-default-margin">Default Margin (%)</label>
              <input
                id="settings-default-margin"
                type="number"
                bind:value={settings.business.default_margin}
                class="input-clean"
              />
            </div>
            <div class="form-group half">
              <label for="settings-vat-rate">VAT Rate (%)</label>
              <input
                id="settings-vat-rate"
                type="number"
                bind:value={settings.business.vat_rate}
                class="input-clean"
              />
            </div>
          </div>
        </div>
      {:else if activeSection === "currency"}
        <div class="section" in:fade={{ duration: motionMs(400) }}>
          <h3>Currency Exchange Rates</h3>
          <p class="hint">Exchange rates to BHD (base currency). Historical rates are preserved for audit accuracy.</p>

          <!-- Current Rates -->
          <div class="currency-rates">
            <table class="data-table">
              <thead>
                <tr>
                  <th>Currency</th>
                  <th>Rate (to BHD)</th>
                  <th>Effective From</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {#each supportedCurrencies.filter(c => c.code !== 'BHD') as currency}
                  {@const rate = currencyRates.find(r => r.currency_code === currency.code)}
                  <tr>
                    <td>
                      <strong>{currency.code}</strong>
                      <span class="hint-inline">({currency.name})</span>
                    </td>
                    <td>
                      {#if rate}
                        1 {currency.code} = <strong>{rate.rate.toFixed(4)}</strong> BHD
                      {:else}
                        <span class="muted">Not configured</span>
                      {/if}
                    </td>
                    <td>
                      {#if rate}
                        {new Date(rate.effective_from).toLocaleDateString()}
                      {:else}
                        -
                      {/if}
                    </td>
                    <td>
                      <button
                        class="btn-ghost btn-sm"
                        onclick={() => {
                          editingCurrency = currency.code;
                          newRate = rate?.rate || 0;
                          rateNotes = '';
                        }}
                      >
                        Update
                      </button>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>

          <!-- Edit Rate Form -->
          {#if editingCurrency}
            <div class="edit-rate-form" style="margin-top: 1.5rem; padding: 1rem; background: var(--surface-elevated); border-radius: 8px;">
              <h4>Update {editingCurrency} Rate</h4>
              <div class="form-group">
                <label for="settings-new-rate">New Rate (1 {editingCurrency} = ? BHD)</label>
                <input
                  id="settings-new-rate"
                  type="number"
                  step="0.0001"
                  min="0.0001"
                  bind:value={newRate}
                  class="input-clean"
                  placeholder="e.g., 0.376"
                />
              </div>
              <div class="form-group">
                <label for="settings-rate-notes">Notes (optional)</label>
                <input
                  id="settings-rate-notes"
                  type="text"
                  bind:value={rateNotes}
                  class="input-clean"
                  placeholder="e.g., Central bank rate Feb 2026"
                />
              </div>
              <div class="action-row" style="margin-top: 1rem;">
                <button class="btn-primary" onclick={handleSaveRate} disabled={savingRate}>
                  {savingRate ? 'Saving...' : 'Save Rate'}
                </button>
                <button class="btn-ghost" onclick={() => { editingCurrency = ''; newRate = 0; }}>
                  Cancel
                </button>
              </div>
            </div>
          {/if}

          <!-- Reference Info -->
          <div class="info-box" style="margin-top: 1.5rem; padding: 1rem; background: var(--surface); border-radius: 8px; border-left: 3px solid var(--accent);">
            <p><strong>Note:</strong> BHD (Bahraini Dinar) is the base currency. All other currencies are converted to BHD for calculations.</p>
            <p style="margin-top: 0.5rem;">Exchange rates are used for:</p>
            <ul style="margin: 0.5rem 0 0 1.5rem;">
              <li>Supplier invoices in foreign currency</li>
              <li>Purchase orders with international suppliers</li>
              <li>Financial reporting and reconciliation</li>
            </ul>
          </div>
        </div>
      {:else if activeSection === "accounts" && canManageBankAccounts}
        <div class="section" in:fade={{ duration: motionMs(400) }}>
          <h3>Bank Accounts</h3>
          <p class="hint">Manage the company bank accounts used across statement import, matching, and reconciliation. Deactivating an account hides it from the active pickers without deleting history.</p>

          {#if loadingBankAccounts}
            <div class="loading-container">
              <WabiSpinner size="md" />
            </div>
          {:else}
            <div class="bank-accounts-list">
              {#if bankAccounts.length === 0}
                <p class="empty-message">No bank accounts found</p>
              {:else}
                {#each bankAccounts as account}
                  <div class="bank-account-item" class:inactive={!account.is_active}>
                    <div class="bank-account-info">
                      <div class="bank-account-main">
                        <strong>{account.account_name || account.bank_name}</strong>
                        {#if account.account_name}
                          <span class="hint-inline">{account.bank_name}</span>
                        {/if}
                        {#if !account.is_active}
                          <span class="inactive-badge">Inactive</span>
                        {/if}
                      </div>
                      <div class="bank-account-details">
                        <span>{account.account_number}</span>
                        {#if account.iban}
                          <span class="mono">{account.iban}</span>
                        {/if}
                        <span class="currency-chip">{account.currency}</span>
                        {#if account.division}
                          <span class="hint-inline">{account.division}</span>
                        {/if}
                      </div>
                    </div>
                    <div class="bank-account-actions">
                      <button class="btn-ghost btn-sm" onclick={() => editBankAccount(account)}>Edit</button>
                      {#if account.is_active}
                        <button class="btn-ghost btn-sm danger" onclick={() => handleDeleteBankAccount(account)}>Deactivate</button>
                      {/if}
                    </div>
                  </div>
                {/each}
              {/if}
            </div>
          {/if}

          <!-- Add/Edit Account Form -->
          <div class="edit-rate-form" style="margin-top: 1.5rem; padding: 1rem; background: var(--surface-elevated); border-radius: 8px;">
            <h4>{editingBankAccount ? 'Edit Account' : 'Add New Account'}</h4>
            <div class="row">
              <div class="form-group half">
                <label for="ba-division">Company</label>
                <select id="ba-division" bind:value={bankAccountFormData.division} class="input-clean">
                  <option value="Acme Instrumentation">Acme Instrumentation</option>
                  <option value="Beacon Controls">Beacon Controls</option>
                </select>
              </div>
              <div class="form-group half">
                <label for="ba-currency">Currency</label>
                <select id="ba-currency" bind:value={bankAccountFormData.currency} class="input-clean">
                  <option value="BHD">BHD</option>
                  <option value="USD">USD</option>
                  <option value="EUR">EUR</option>
                  <option value="GBP">GBP</option>
                  <option value="SAR">SAR</option>
                </select>
              </div>
            </div>
            <div class="form-group">
              <label for="ba-bank-name">Bank Name *</label>
              <input id="ba-bank-name" type="text" bind:value={bankAccountFormData.bank_name} class="input-clean" placeholder="e.g., Demo Bank A" />
            </div>
            <div class="form-group">
              <label for="ba-account-name">Display Name</label>
              <input id="ba-account-name" type="text" bind:value={bankAccountFormData.account_name} class="input-clean" placeholder="e.g., Demo Bank A - BHD Operating" />
            </div>
            <div class="form-group">
              <label for="ba-account-number">Account Number *</label>
              <input id="ba-account-number" type="text" bind:value={bankAccountFormData.account_number} class="input-clean" placeholder="e.g., 10000000001" />
            </div>
            <div class="row">
              <div class="form-group half">
                <label for="ba-iban">IBAN</label>
                <input id="ba-iban" type="text" bind:value={bankAccountFormData.iban} class="input-clean" placeholder="e.g., BH29DMOA10000000000001" />
              </div>
              <div class="form-group half">
                <label for="ba-swift">SWIFT/BIC</label>
                <input id="ba-swift" type="text" bind:value={bankAccountFormData.swift_bic} class="input-clean" placeholder="e.g., DMOABHBM" />
              </div>
            </div>
            {#if bankAccountFormData.currency !== 'BHD'}
              <div class="form-group">
                <label for="ba-booking-rate">Opening/Booking Rate</label>
                <input id="ba-booking-rate" type="number" step="0.0001" min="0" bind:value={bankAccountFormData.booking_rate} class="input-clean" placeholder="e.g., 1.4523" />
              </div>
            {/if}
            <div class="action-row" style="margin-top: 1rem;">
              {#if editingBankAccount}
                <button class="btn-ghost" onclick={resetBankAccountForm}>Cancel Edit</button>
              {/if}
              <button class="btn-primary" onclick={handleSaveBankAccount} disabled={savingBankAccount}>
                {savingBankAccount ? 'Saving...' : (editingBankAccount ? 'Update Account' : 'Add Account')}
              </button>
            </div>
          </div>
        </div>
      {:else if activeSection === "data"}
        <div class="section" in:fade={{ duration: motionMs(400) }}>
          <h3>Tally Data Import</h3>
          <p class="hint">Import historical data from Tally Excel exports. Files are loaded from Data_for_database/Tally_data directory.</p>

          <div class="form-group">
            <label for="settings-import-year">Import Year</label>
            <input id="settings-import-year" type="number" bind:value={importYear} class="input-clean" min="2000" max="2030" />
          </div>

          <div class="import-actions">
            <button class="btn-secondary" onclick={importInvoices} disabled={importing}>
              {importing ? 'Importing...' : 'Import Invoices'}
            </button>
            <button class="btn-secondary" onclick={importPurchases} disabled={importing}>
              {importing ? 'Importing...' : 'Import Purchases'}
            </button>
            <button class="btn-secondary" onclick={importARDefaulters} disabled={importing}>
              {importing ? 'Importing...' : 'Import AR Defaulters'}
            </button>
            <button class="btn-secondary" onclick={importSupplierPayments} disabled={importing}>
              {importing ? 'Importing...' : 'Import Supplier Payments'}
            </button>
          </div>

          <div class="import-divider"></div>

          <button class="btn-primary" onclick={importTallyAll} disabled={importing}>
            {importing ? 'Running Full Import...' : 'Import All Tally Data'}
          </button>

          {#if importResult}
            <div class="import-result">
              <h4>Import Result</h4>
              <div class="result-stats">
                <span>Total Rows: {importResult.total_rows}</span>
                <span>Imported: {importResult.imported}</span>
                <span>Duplicates: {importResult.duplicates}</span>
                <span>Errors: {importResult.errors}</span>
              </div>
              {#if importResult.error_details?.length > 0}
                <details>
                  <summary>Error Details ({importResult.error_details.length})</summary>
                  <ul class="error-list">
                    {#each importResult.error_details.slice(0, 20) as err}
                      <li>{err}</li>
                    {/each}
                  </ul>
                </details>
              {/if}
            </div>
          {/if}
        </div>
      {:else if activeSection === "reports"}
        <div class="section" in:fade={{ duration: motionMs(400) }}>
          <h3>Financial Report Generation</h3>
          <p class="hint">Generate Profit & Loss and Balance Sheet reports from imported Tally data.</p>

          <div class="form-group">
            <label for="settings-report-year">Report Year</label>
            {#if availableYears.length > 0}
              <select id="settings-report-year" bind:value={reportYear} class="input-clean">
                {#each availableYears as yr}
                  <option value={yr}>{yr}</option>
                {/each}
              </select>
            {:else}
              <input id="settings-report-year" type="number" bind:value={reportYear} class="input-clean" min="2000" max="2030" />
              <p class="hint-small">No imported data found. Import Tally data first.</p>
            {/if}
          </div>

          <div class="import-actions">
            <button class="btn-secondary" onclick={generatePL} disabled={generatingReport}>
              {generatingReport ? 'Generating...' : 'Generate P&L Statement'}
            </button>
            <button class="btn-secondary" onclick={generateBS} disabled={generatingReport}>
              {generatingReport ? 'Generating...' : 'Generate Balance Sheet'}
            </button>
          </div>

          {#if plReport}
            <div class="financial-report">
              <h4>Profit & Loss Statement - {plReport.year}</h4>
              <div class="report-section">
                <div class="report-line">
                  <span class="label">Sales Revenue</span>
                  <span class="value">{formatNumber(plReport.sales_revenue, 3)} {plReport.currency}</span>
                </div>
                <div class="report-line">
                  <span class="label">Other Income</span>
                  <span class="value">{formatNumber(plReport.other_income, 3)} {plReport.currency}</span>
                </div>
                <div class="report-line total">
                  <span class="label">Total Revenue</span>
                  <span class="value">{formatNumber(plReport.total_revenue, 3)} {plReport.currency}</span>
                </div>
              </div>

              <div class="report-section">
                <div class="report-line">
                  <span class="label">Cost of Goods Sold</span>
                  <span class="value">{formatNumber(plReport.cost_of_goods_sold, 3)} {plReport.currency}</span>
                </div>
                <div class="report-line total">
                  <span class="label">Gross Profit</span>
                  <span class="value">{formatNumber(plReport.gross_profit, 3)} {plReport.currency}</span>
                </div>
                <div class="report-line">
                  <span class="label">Gross Profit Margin</span>
                  <span class="value">{plReport.gross_profit_margin.toFixed(1)}%</span>
                </div>
              </div>

              <div class="report-section">
                <div class="report-line">
                  <span class="label">Operating Expenses</span>
                  <span class="value">{formatNumber(plReport.operating_expenses, 3)} {plReport.currency}</span>
                </div>
              </div>

              <div class="report-section final">
                <div class="report-line total">
                  <span class="label">Net Profit</span>
                  <span class="value net-profit">{formatNumber(plReport.net_profit, 3)} {plReport.currency}</span>
                </div>
                <div class="report-line">
                  <span class="label">Net Profit Margin</span>
                  <span class="value">{plReport.net_profit_margin.toFixed(1)}%</span>
                </div>
              </div>

              <div class="report-meta">
                <span>Generated: {new Date(plReport.generated_at).toLocaleString()}</span>
                <span>Source: {plReport.invoice_count} invoices, {plReport.purchase_count} purchases</span>
              </div>
            </div>
          {/if}

          {#if bsReport}
            <div class="financial-report">
              <h4>Balance Sheet - As of {new Date(bsReport.as_of_date).toLocaleDateString()}</h4>

              <div class="report-section">
                <h5>ASSETS</h5>
                <div class="report-line">
                  <span class="label">Cash</span>
                  <span class="value">{formatNumber(bsReport.cash, 3)} {bsReport.currency}</span>
                </div>
                <div class="report-line">
                  <span class="label">Accounts Receivable</span>
                  <span class="value">{formatNumber(bsReport.accounts_receivable, 3)} {bsReport.currency}</span>
                </div>
                <div class="report-line">
                  <span class="label">Inventory</span>
                  <span class="value">{formatNumber(bsReport.inventory, 3)} {bsReport.currency}</span>
                </div>
                <div class="report-line total">
                  <span class="label">Total Current Assets</span>
                  <span class="value">{formatNumber(bsReport.total_current_assets, 3)} {bsReport.currency}</span>
                </div>
                <div class="report-line total">
                  <span class="label">Total Assets</span>
                  <span class="value">{formatNumber(bsReport.total_assets, 3)} {bsReport.currency}</span>
                </div>
              </div>

              <div class="report-section">
                <h5>LIABILITIES</h5>
                <div class="report-line">
                  <span class="label">Accounts Payable</span>
                  <span class="value">{formatNumber(bsReport.accounts_payable, 3)} {bsReport.currency}</span>
                </div>
                <div class="report-line total">
                  <span class="label">Total Current Liabilities</span>
                  <span class="value">{formatNumber(bsReport.total_current_liabilities, 3)} {bsReport.currency}</span>
                </div>
                <div class="report-line total">
                  <span class="label">Total Liabilities</span>
                  <span class="value">{formatNumber(bsReport.total_liabilities, 3)} {bsReport.currency}</span>
                </div>
              </div>

              <div class="report-section final">
                <h5>EQUITY</h5>
                <div class="report-line">
                  <span class="label">Retained Earnings</span>
                  <span class="value">{formatNumber(bsReport.retained_earnings, 3)} {bsReport.currency}</span>
                </div>
                <div class="report-line total">
                  <span class="label">Total Equity</span>
                  <span class="value">{formatNumber(bsReport.total_equity, 3)} {bsReport.currency}</span>
                </div>
              </div>

              <div class="report-meta">
                <span>Generated: {new Date(bsReport.generated_at).toLocaleString()}</span>
              </div>
            </div>
          {/if}
        </div>
      {:else if activeSection === "sync"}
        <div class="section" in:fade={{ duration: motionMs(400) }}>
          <h3>Supabase Connection</h3>
          <p class="hint">Configure remote database sync via Supabase for multi-device access.</p>

          <div class="form-group">
            <label for="settings-supabase-url">Supabase URL</label>
            <input id="settings-supabase-url" type="text" bind:value={supabaseForm.url} class="input-clean" placeholder="https://xxxx.supabase.co" />
          </div>
          <div class="form-group">
            <label for="settings-supabase-anon-key">Anon Key</label>
            <input id="settings-supabase-anon-key" type="password" bind:value={supabaseForm.anonKey} class="input-clean" placeholder="eyJ..." />
          </div>
          <div class="form-group">
            <label for="settings-supabase-service-key">Service Key</label>
            <input id="settings-supabase-service-key" type="password" bind:value={supabaseForm.serviceKey} class="input-clean" placeholder="eyJ..." />
          </div>

          <h3>Database Connection</h3>
          <div class="row">
            <div class="form-group half">
              <label for="settings-db-host">DB Host</label>
              <input id="settings-db-host" type="text" bind:value={supabaseForm.dbHost} class="input-clean" placeholder="db.xxxx.supabase.co" />
            </div>
            <div class="form-group half">
              <label for="settings-db-port">DB Port</label>
              <input id="settings-db-port" type="text" bind:value={supabaseForm.dbPort} class="input-clean" />
            </div>
          </div>
          <div class="row">
            <div class="form-group half">
              <label for="settings-db-name">DB Name</label>
              <input id="settings-db-name" type="text" bind:value={supabaseForm.dbName} class="input-clean" placeholder="postgres" />
            </div>
            <div class="form-group half">
              <label for="settings-db-password">DB Password</label>
              <input id="settings-db-password" type="password" bind:value={supabaseForm.dbPassword} class="input-clean" placeholder="Database password" />
            </div>
          </div>

          <div class="action-row">
            <button class="btn-secondary" onclick={testSupabase} disabled={testingConnection}>
              {testingConnection ? 'Testing...' : 'Test Connection'}
            </button>
          </div>

          <h3>Operational Status</h3>
          <div class="result-stats">
            <span>Sync Online: {syncHealth?.is_online ? 'Yes' : 'No'}</span>
            <span>Tables In Sync: {syncHealth?.tables_in_sync ?? 0}</span>
            <span>Last Sync: {syncHealth?.last_sync_at ? new Date(syncHealth.last_sync_at).toLocaleString() : 'Never'}</span>
            <span>Backups: {syncHealth?.backup_count ?? 0}</span>
          </div>

          <h3>Database Backups</h3>
          <div class="backup-panel">
            <div class="result-stats">
              <span>Stored Backups: {backupInfo?.count ?? 0}</span>
              <span>Last Backup: {backupPolicy?.last_backup_at ? new Date(backupPolicy.last_backup_at).toLocaleString() : (backupInfo?.last_backup || 'Never')}</span>
              <span>Next Due: {backupPolicy?.next_backup_due_at ? new Date(backupPolicy.next_backup_due_at).toLocaleString() : 'Next startup'}</span>
              <span>Folder: {backupInfo?.backup_dir || 'App data backups'}</span>
            </div>
            <div class="row backup-controls">
              <label class="toggle-inline" for="backup-auto-enabled">
                <input id="backup-auto-enabled" type="checkbox" bind:checked={backupAutoEnabled} />
                Auto Backup
              </label>
              <div class="form-group compact">
                <label for="backup-frequency-days">Frequency</label>
                <input id="backup-frequency-days" type="number" min="1" max="30" bind:value={backupFrequencyDays} class="input-clean" />
              </div>
              <button class="btn-secondary" onclick={saveBackupPolicy} disabled={savingBackupPolicy}>
                {savingBackupPolicy ? 'Saving...' : 'Save Schedule'}
              </button>
              <button class="btn-primary" onclick={runManualBackup} disabled={backupRunning}>
                {backupRunning ? 'Backing up...' : 'Backup Now'}
              </button>
            </div>
          </div>

          <h3>Phase 7 Rollout</h3>
          <div class="result-stats">
            <span>Legacy Follow-Ups: {rolloutStatus?.legacy_followup_tasks ?? 0}</span>
            <span>Migrated Tasks: {rolloutStatus?.migrated_legacy_tasks ?? 0}</span>
            <span>Pending Ops: {rolloutStatus?.pending_collaborative_ops ?? 0}</span>
            <span>Failed Ops: {rolloutStatus?.failed_collaborative_ops ?? 0}</span>
            <span>Dead Letters: {rolloutStatus?.dead_letter_collaborative_ops ?? 0}</span>
            <span>Payroll Payouts Awaiting Recon: {rolloutStatus?.payroll_payouts_awaiting_recon ?? 0}</span>
          </div>
          {#if rolloutStatus?.followup_backfill_completed_at}
            <p class="hint-small">Legacy follow-up backfill completed on {new Date(rolloutStatus.followup_backfill_completed_at).toLocaleString()}.</p>
          {/if}

          <div class="section-divider"></div>

          <div class="rollout-ops-header">
            <h4>Pilot Readiness</h4>
            <div class="rollout-ops-controls">
              <label class="toggle-inline" for="pilot-only-issues">
                <input
                  id="pilot-only-issues"
                  type="checkbox"
                  bind:checked={pilotOnlyIssues}
                  onchange={loadPilotReadiness}
                />
                Issues Only
              </label>
              <button class="btn-ghost" onclick={loadPilotReadiness} disabled={loadingPilotReadiness}>
                {loadingPilotReadiness ? 'Refreshing...' : 'Refresh Readiness'}
              </button>
              <button class="btn-secondary" onclick={handleExportPilotBundle} disabled={exportingPilotBundle || rolloutActionRunning}>
                {exportingPilotBundle ? 'Exporting...' : 'Export Support Bundle'}
              </button>
            </div>
          </div>

          <div class="result-stats pilot-summary-grid">
            <span>Total Employees: {pilotSummary?.total_employees ?? 0}</span>
            <span>Ready: {pilotSummary?.ready_employees ?? 0}</span>
            <span>Needs Attention: {pilotSummary?.employees_with_issues ?? 0}</span>
            <span>Missing Access: {pilotSummary?.employees_missing_access ?? 0}</span>
            <span>Activated Licenses: {pilotSummary?.activated_licenses ?? 0}</span>
            <span>Unlinked Licenses: {pilotSummary?.unlinked_licenses ?? 0}</span>
            <span>Approved Devices: {pilotSummary?.approved_devices ?? 0}</span>
            <span>Pending Devices: {pilotSummary?.pending_devices ?? 0}</span>
            <span>Blocked Devices: {pilotSummary?.blocked_devices ?? 0}</span>
          </div>
          {#if pilotSummary?.generated_at}
            <p class="hint-small">Pilot readiness snapshot generated on {new Date(pilotSummary.generated_at).toLocaleString()}.</p>
          {/if}

          {#if loadingPilotReadiness}
            <p class="hint-small">Refreshing pilot readiness audit...</p>
          {:else if pilotRows.length === 0}
            <p class="hint-small">
              {pilotOnlyIssues ? 'No active rollout issues were found for current employees.' : 'No employee readiness records are available yet.'}
            </p>
          {:else}
            <div class="pilot-readiness-table">
              <div class="pilot-readiness-row pilot-readiness-head">
                <span>Employee</span>
                <span>Department</span>
                <span>Access</span>
                <span>License</span>
                <span>Device</span>
                <span>User</span>
                <span>Readiness</span>
              </div>
              {#each pilotRows as row}
                <div class="pilot-readiness-row">
                  <span class="pilot-cell">
                    <strong>{row.employee_name || 'Unknown Employee'}</strong>
                    <small>{row.employee_code || row.employee_id?.slice?.(0, 8) || '—'}</small>
                  </span>
                  <span class="pilot-cell">
                    <strong>{row.department || '—'}</strong>
                    <small>{row.job_title || row.employment_state || '—'}</small>
                  </span>
                  <span class="pilot-cell">
                    <strong>{row.access_status || 'unlinked'}</strong>
                    <small>{row.employment_state || '—'}</small>
                  </span>
                  <span class="pilot-cell">
                    <strong>{row.license_key || '—'}</strong>
                    <small>{row.license_role || (row.license_active ? 'Activated' : 'Not activated') || '—'}</small>
                  </span>
                  <span class="pilot-cell">
                    <strong>{row.device_name || row.device_id || '—'}</strong>
                    <small>{row.device_status || row.last_seen_at || '—'}</small>
                  </span>
                  <span class="pilot-cell">
                    <strong>{row.user_name || '—'}</strong>
                    <small>{row.user_id || 'No linked user'}</small>
                  </span>
                  <span class="pilot-cell">
                    <span class={`status-chip ${row.ready_for_pilot ? 'ready' : 'attention'}`}>
                      {row.ready_for_pilot ? 'Ready' : 'Attention'}
                    </span>
                    <small>{getPilotRowIssueSummary(row)}</small>
                  </span>
                </div>
              {/each}
            </div>
          {/if}

          <div class="action-row rollout-actions">
            <button class="btn-secondary" onclick={handleTriggerCollaborativeSync} disabled={rolloutActionRunning}>
              {rolloutActionRunning ? 'Working...' : 'Run Collaborative Sync'}
            </button>
            <button class="btn-secondary" onclick={() => handleRetryQueue('failed')} disabled={rolloutActionRunning}>
              Retry Failed Ops
            </button>
            <button class="btn-secondary" onclick={() => handleRetryQueue('dead_letter')} disabled={rolloutActionRunning}>
              Revive Dead Letters
            </button>
            <button class="btn-secondary" onclick={handleRerunBackfill} disabled={rolloutActionRunning}>
              Re-run Legacy Backfill
            </button>
          </div>

          <div class="rollout-ops-header">
            <h4>Collaborative Queue</h4>
            <div class="rollout-ops-controls">
              <select bind:value={rolloutOpsFilter} class="input-clean input-inline" onchange={loadRolloutOps}>
                <option value="active">Active Issues</option>
                <option value="pending">Pending</option>
                <option value="failed">Failed</option>
                <option value="dead_letter">Dead Letter</option>
                <option value="synced">Recently Synced</option>
              </select>
              <button class="btn-ghost" onclick={loadRolloutOps} disabled={loadingRolloutOps}>
                {loadingRolloutOps ? 'Refreshing...' : 'Refresh Queue'}
              </button>
            </div>
          </div>

          {#if rolloutOps.length === 0}
            <p class="hint-small">No collaborative queue items match the current filter.</p>
          {:else}
            <div class="rollout-ops-table">
              <div class="rollout-ops-row rollout-ops-head">
                <span>Status</span>
                <span>Entity</span>
                <span>Operation</span>
                <span>Attempts</span>
                <span>Updated</span>
                <span>Action</span>
              </div>
              {#each rolloutOps as op}
                <div class="rollout-ops-row">
                  <span class={`status-chip ${op.status || 'unknown'}`}>{op.status || 'unknown'}</span>
                  <span>{op.entity_type}:{op.entity_id?.slice?.(0, 8) || op.entity_id}</span>
                  <span>{op.operation}</span>
                  <span>{op.attempts ?? 0}</span>
                  <span>{op.updated_at ? new Date(op.updated_at).toLocaleString() : '—'}</span>
                  <span>
                    {#if op.status === 'failed' || op.status === 'dead_letter'}
                      <button class="btn-ghost danger" onclick={() => handleRetrySingle(op.id)} disabled={rolloutActionRunning}>
                        Retry
                      </button>
                    {:else}
                      <span class="hint-small">{op.error_message ? op.error_message.slice(0, 60) : '—'}</span>
                    {/if}
                  </span>
                </div>
                {#if op.error_message}
                  <div class="rollout-ops-error">{op.error_message}</div>
                {/if}
              {/each}
            </div>
          {/if}
        </div>
      {:else if activeSection === "deployment"}
        <div class="section" in:fade={{ duration: motionMs(400) }}>
          <h3>Deployment Workspace</h3>
          <p class="hint">Keep rollout checks and production support controls under Settings, where admin tools belong.</p>

          <div class="deployment-card">
            <div class="deployment-card-copy">
              <h4>Admin Rollout Control</h4>
              <p>Open the deployment workspace to review rollout readiness, repair sync queues, inspect pilot issues, and export support bundles.</p>
            </div>
            <button class="btn-primary" onclick={openDeploymentWorkspace}>
              Open Deployment Workspace
            </button>
          </div>

          <div class="result-stats">
            <span>Sync Online: {syncHealth?.is_online ? 'Yes' : 'No'}</span>
            <span>Pending Ops: {rolloutStatus?.pending_collaborative_ops ?? 0}</span>
            <span>Failed Ops: {rolloutStatus?.failed_collaborative_ops ?? 0}</span>
            <span>Dead Letters: {rolloutStatus?.dead_letter_collaborative_ops ?? 0}</span>
            <span>Pilot Issues: {pilotSummary?.employees_with_issues ?? 0}</span>
            <span>Backups: {syncHealth?.backup_count ?? 0}</span>
          </div>

          <div class="info-box deployment-note">
            <p><strong>Why it moved:</strong> deployment is an internal admin workspace, not an everyday navigation area. Putting it under Settings keeps the sidebar focused on business operations while preserving direct admin access.</p>
          </div>
        </div>
      {/if}
    </main>
  </div>
</div>

<style>
  .page {
    padding: var(--page-padding);
    height: 100%;
    background: var(--bg-base, #F5F5F7);
    color: var(--text-primary, #1D1D1F);
    display: flex;
    flex-direction: column;
    box-sizing: border-box;
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: flex-end;
    margin-bottom: 24px;
  }
  h1 {
    font-size: 28px;
    font-weight: 300;
    margin: 0;
    letter-spacing: -0.02em;
    color: var(--onyx, #1D1D1F);
  }
  .subtitle {
    color: var(--text-muted, #AEAEB2);
    margin-top: 4px;
    font-size: 13px;
  }

  .layout-split {
    display: grid;
    grid-template-columns: 180px 1fr;
    gap: 24px;
    flex: 1;
    min-height: 0;
  }

  /* Sidebar */
  .sidebar {
    border-right: 1px solid var(--border, #E5E5E5);
    padding-right: 24px;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    min-height: 0;
  }
  .sidebar nav {
    overflow-y: auto;
    padding-right: 4px;
  }
  .nav-item {
    display: block;
    width: 100%;
    text-align: left;
    padding: 12px 16px;
    margin-bottom: 4px;
    border: none;
    background: transparent;
    border-radius: 8px;
    font-size: 14px;
    color: var(--steel, #86868B);
    cursor: pointer;
    transition: all 0.2s;
    font-family: var(--font-family, 'Inter', system-ui, sans-serif);
  }
  .nav-item:hover {
    background: var(--surface-elevated, #FAFAFA);
    color: var(--onyx, #1D1D1F);
  }
  .nav-item.active {
    background: var(--carbon, #000);
    color: var(--canvas, #FFF);
    font-weight: 500;
  }

  .sys-info {
    font-size: 11px;
    color: var(--steel, #86868B);
    display: flex;
    flex-direction: column;
    gap: 4px;
    opacity: 0.6;
  }

  /* Content */
  .content-panel {
    overflow-y: auto;
    padding-right: 24px;
    max-width: 800px;
  }
  .section {
    margin-bottom: 48px;
  }
  h3 {
    font-size: 18px;
    font-weight: 500;
    margin-bottom: 24px;
    border-bottom: 1px solid var(--border, #E5E5E5);
    padding-bottom: 8px;
    color: var(--onyx, #1D1D1F);
  }
  .hint {
    font-size: 13px;
    color: var(--steel, #86868B);
    margin-bottom: 16px;
  }

  .form-group {
    margin-bottom: 20px;
  }
  label {
    display: block;
    font-size: 12px;
    font-weight: 500;
    margin-bottom: 6px;
    color: var(--steel, #86868B);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .input-clean {
    width: 100%;
    padding: 10px 12px;
    border: 1px solid var(--border, #E5E5E5);
    border-radius: 8px;
    font-family: var(--font-family, 'Inter', system-ui, sans-serif);
    box-sizing: border-box;
    font-size: 14px;
    color: var(--text-primary, #1D1D1F);
    background: var(--surface, #FFFFFF);
    transition: border-color 0.2s;
  }
  .input-clean:focus {
    border-color: var(--onyx, #1D1D1F);
    outline: none;
    box-shadow: 0 0 0 3px rgba(29, 29, 31, 0.06);
  }

  .row {
    display: flex;
    gap: 16px;
  }
  .half {
    flex: 1;
  }

  .input-group {
    display: flex;
    gap: 8px;
  }
  .btn-ghost {
    background: transparent;
    border: 1px solid var(--border, #E5E5E5);
    padding: 0 16px;
    border-radius: 8px;
    cursor: pointer;
    white-space: nowrap;
    font-size: 13px;
    color: var(--text-primary, #1D1D1F);
    transition: border-color 0.2s;
  }
  .btn-ghost:hover {
    border-color: var(--onyx, #1D1D1F);
  }
  .btn-ghost.danger {
    color: var(--text-danger);
    border-color: rgba(180, 35, 24, 0.2);
  }
  .btn-ghost.danger:hover {
    border-color: rgba(180, 35, 24, 0.45);
  }
  .btn-icon {
    background: transparent;
    border: none;
    cursor: pointer;
    padding: 0 8px;
    font-size: 16px;
  }

  .toggle-row {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 12px;
  }

  .gpu-card {
    background: var(--surface-elevated, #FAFAFA);
    padding: 16px;
    border-radius: 12px;
    border: 1px solid var(--border, #E5E5E5);
    margin-top: 16px;
  }
  .gpu-header {
    display: flex;
    align-items: center;
    gap: 10px;
    font-weight: 600;
    font-size: 15px;
    margin-bottom: 8px;
    color: var(--onyx, #1D1D1F);
  }
  .gpu-stats {
    font-size: 12px;
    color: var(--steel, #86868B);
    margin-bottom: 12px;
    display: flex;
    gap: 16px;
  }

  .btn-primary {
    background: var(--carbon, #000);
    color: var(--canvas, #FFF);
    border: none;
    padding: 10px 24px;
    border-radius: 100px;
    cursor: pointer;
    font-size: 14px;
    font-weight: 500;
    transition: background 0.2s;
    /* CRITICAL: position relative to contain ::after pseudo-element from global styles */
    position: relative;
    overflow: hidden;
  }
  .btn-primary:hover:not(:disabled) {
    background: var(--onyx, #1D1D1F);
  }
  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .btn-secondary {
    background: var(--surface-elevated, #FAFAFA);
    border: 1px solid var(--border, #E5E5E5);
    padding: 8px 16px;
    border-radius: 8px;
    cursor: pointer;
    font-size: 13px;
    color: var(--text-primary, #1D1D1F);
    transition: border-color 0.2s;
  }
  .btn-secondary:hover:not(:disabled) {
    border-color: var(--onyx, #1D1D1F);
  }
  .btn-secondary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .loading {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
  }

  .action-row {
    margin-top: 16px;
  }

  .import-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-bottom: 16px;
  }

  .import-divider {
    border-top: 1px solid var(--border, #E5E5E5);
    margin: 16px 0;
  }

  .import-result {
    margin-top: 16px;
    padding: 16px;
    background: var(--surface-elevated, #FAFAFA);
    border-radius: 8px;
    border: 1px solid var(--border, #E5E5E5);
  }
  .import-result h4 {
    margin: 0 0 8px;
    font-size: 14px;
    color: var(--onyx, #1D1D1F);
  }
  .result-stats {
    display: flex;
    flex-wrap: wrap;
    gap: 16px;
    font-size: 13px;
    color: var(--steel, #86868B);
  }
  .backup-panel {
    display: grid;
    gap: 14px;
    padding: 16px;
    border: 1px solid var(--border, #E5E5E5);
    border-radius: 8px;
    background: var(--surface-elevated, #FAFAFA);
    margin-bottom: 20px;
  }
  .backup-controls {
    align-items: flex-end;
    flex-wrap: wrap;
  }
  .form-group.compact {
    width: 130px;
    margin-bottom: 0;
  }
  .deployment-card {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 18px 20px;
    margin: 16px 0;
    border: 1px solid var(--border, #E5E5E5);
    border-radius: 14px;
    background: linear-gradient(135deg, rgba(248, 248, 250, 0.95), rgba(255, 255, 255, 0.98));
  }
  .deployment-card-copy h4 {
    margin: 0 0 6px;
    font-size: 15px;
    color: var(--onyx, #1D1D1F);
  }
  .deployment-card-copy p {
    margin: 0;
    font-size: 13px;
    line-height: 1.5;
    color: var(--steel, #86868B);
  }
  .readability-card {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(280px, 360px);
    gap: 16px;
    padding: 18px 20px;
    margin: 16px 0 22px;
    border: 1px solid var(--border, #E5E5E5);
    border-radius: 14px;
    background: var(--surface, #FFFFFF);
  }
  .readability-copy h4 {
    margin: 0 0 6px;
    color: var(--onyx, #1D1D1F);
    font-size: 15px;
  }
  .readability-copy p {
    margin: 0;
    color: var(--steel, #86868B);
    font-size: 13px;
    line-height: 1.5;
  }
  .readability-controls {
    display: grid;
    gap: 12px;
    align-content: center;
  }
  .text-preset-buttons {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 6px;
  }
  .text-preset-buttons button {
    display: grid;
    gap: 2px;
    justify-items: center;
    min-height: 52px;
    border: 1px solid var(--border, #E5E5E5);
    border-radius: 8px;
    background: var(--surface-elevated, #FAFAFA);
    color: var(--text-primary, #1D1D1F);
    cursor: pointer;
    font-size: 11px;
    font-weight: 600;
  }
  .text-preset-buttons button span {
    font-family: var(--font-display);
    font-size: 15px;
    font-weight: 800;
  }
  .text-preset-buttons button:nth-child(2) span {
    font-size: 18px;
  }
  .text-preset-buttons button:nth-child(3) span {
    font-size: 21px;
  }
  .text-preset-buttons button:hover,
  .text-preset-buttons button.active {
    background: var(--onyx, #1D1D1F);
    border-color: var(--onyx, #1D1D1F);
    color: var(--canvas, #fff);
  }
  .text-scale-slider {
    display: grid;
    gap: 6px;
    margin: 0;
    text-transform: none;
    letter-spacing: 0;
    color: var(--text-secondary, #86868B);
  }
  .text-scale-slider span {
    font-size: 12px;
    font-weight: 700;
    color: var(--text-primary, #1D1D1F);
  }
  .text-scale-slider input {
    width: 100%;
    accent-color: var(--onyx, #1D1D1F);
  }
  .deployment-note {
    margin-top: 16px;
  }
  .info-box {
    padding: 16px;
    background: var(--surface-elevated, #FAFAFA);
    border-radius: 12px;
    border-left: 3px solid var(--onyx, #1D1D1F);
  }
  .rollout-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }
  .pilot-summary-grid {
    margin-top: 12px;
  }
  .rollout-ops-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 16px;
    margin-top: 20px;
  }
  .rollout-ops-header h4 {
    margin: 0;
    font-size: 14px;
    color: var(--onyx, #1D1D1F);
  }
  .rollout-ops-controls {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px;
  }
  .toggle-inline {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    color: var(--steel, #86868B);
    text-transform: none;
    letter-spacing: 0;
    margin: 0;
  }
  .input-inline {
    min-width: 160px;
  }
  .pilot-readiness-table {
    margin-top: 12px;
    border: 1px solid var(--border, #E5E5E5);
    border-radius: 12px;
    overflow: hidden;
    background: var(--surface, #FFF);
  }
  .pilot-readiness-row {
    display: grid;
    grid-template-columns: 1.2fr 1fr 0.8fr 1.1fr 1.2fr 1fr 1.4fr;
    gap: 12px;
    align-items: start;
    padding: 12px 14px;
    border-bottom: 1px solid var(--border, #E5E5E5);
    font-size: 12px;
  }
  .pilot-readiness-head {
    background: var(--surface-elevated, #FAFAFA);
    font-weight: 600;
    color: var(--steel, #86868B);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .pilot-readiness-row:last-child {
    border-bottom: none;
  }
  .pilot-cell {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }
  .pilot-cell strong,
  .pilot-cell small {
    overflow-wrap: anywhere;
  }
  .rollout-ops-table {
    margin-top: 12px;
    border: 1px solid var(--border, #E5E5E5);
    border-radius: 12px;
    overflow: hidden;
    background: var(--surface, #FFF);
  }
  .rollout-ops-row {
    display: grid;
    grid-template-columns: 120px 1.4fr 110px 80px 180px 1fr;
    gap: 12px;
    align-items: center;
    padding: 12px 14px;
    border-bottom: 1px solid var(--border, #E5E5E5);
    font-size: 12px;
  }
  .rollout-ops-head {
    background: var(--surface-elevated, #FAFAFA);
    font-weight: 600;
    color: var(--steel, #86868B);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .rollout-ops-row:last-child {
    border-bottom: none;
  }
  .rollout-ops-error {
    padding: 0 14px 12px;
    font-size: 12px;
    color: var(--text-danger);
    border-bottom: 1px solid var(--border, #E5E5E5);
    white-space: pre-wrap;
  }
  .status-chip {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 84px;
    padding: 4px 10px;
    border-radius: 999px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .status-chip.pending {
    background: rgba(180, 35, 24, 0.08);
    color: var(--text-danger);
  }
  .status-chip.failed {
    background: rgba(180, 35, 24, 0.14);
    color: var(--text-danger-strong);
  }
  .status-chip.dead_letter {
    background: rgba(127, 29, 29, 0.18);
    color: var(--text-danger-deep);
  }
  .status-chip.synced {
    background: rgba(2, 122, 72, 0.12);
    color: var(--text-success);
  }
  .status-chip.ready {
    background: rgba(2, 122, 72, 0.12);
    color: var(--text-success);
  }
  .status-chip.attention {
    background: rgba(180, 35, 24, 0.12);
    color: var(--text-danger);
  }
  .status-chip.unknown {
    background: rgba(52, 64, 84, 0.08);
    color: var(--text-neutral);
  }
  .error-list {
    font-size: 12px;
    color: var(--steel, #86868B);
    max-height: 200px;
    overflow-y: auto;
    padding-left: 16px;
  }

  /* Financial Reports */
  .financial-report {
    margin-top: 24px;
    padding: 20px;
    background: var(--surface, #FFF);
    border: 1px solid var(--border, #E5E5E5);
    border-radius: 12px;
  }
  .financial-report h4 {
    margin: 0 0 16px;
    font-size: 18px;
    font-weight: 600;
    color: var(--onyx, #1D1D1F);
  }
  .financial-report h5 {
    margin: 16px 0 8px;
    font-size: 13px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--steel, #86868B);
  }
  .report-section {
    margin-bottom: 20px;
    padding-bottom: 16px;
    border-bottom: 1px solid var(--border, #E5E5E5);
  }
  .report-section.final {
    border-bottom: 2px solid var(--onyx, #1D1D1F);
  }
  .report-line {
    display: flex;
    justify-content: space-between;
    padding: 8px 0;
    font-size: 14px;
    font-variant-numeric: tabular-nums lining-nums;
  }
  .report-line.total {
    font-weight: 600;
    border-top: 1px solid var(--border, #E5E5E5);
    padding-top: 12px;
    margin-top: 4px;
  }
  .report-line .label {
    color: var(--onyx, #1D1D1F);
  }
  .report-line .value {
    color: var(--onyx, #1D1D1F);
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
  }
  .report-line .value.net-profit {
    font-weight: 700;
    font-size: 16px;
  }
  .report-meta {
    margin-top: 16px;
    padding-top: 12px;
    border-top: 1px solid var(--border, #E5E5E5);
    display: flex;
    justify-content: space-between;
    font-size: 12px;
    color: var(--steel, #86868B);
  }
  .hint-small {
    font-size: 12px;
    color: var(--steel, #86868B);
    margin-top: 4px;
  }
  .hint-inline {
    font-size: 12px;
    color: var(--steel, #86868B);
    font-weight: 400;
  }
  .section-divider {
    border-top: 1px solid var(--border, #E5E5E5);
    margin: 24px 0;
  }

  /* Wave 9.4 C1: bank-account admin section (moved from BankReconciliationScreen) */
  .bank-accounts-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-height: 420px;
    overflow-y: auto;
  }
  .bank-account-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
    padding: 12px;
    background: var(--surface-elevated, #FAFAFA);
    border: 1px solid var(--border, #E5E5E5);
    border-radius: 8px;
  }
  .bank-account-item.inactive {
    opacity: 0.6;
  }
  .bank-account-info {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .bank-account-main {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .bank-account-main strong {
    font-size: 13px;
    color: var(--text-primary, #1D1D1F);
  }
  .bank-account-details {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 11px;
    color: var(--steel, #86868B);
  }
  .bank-account-details .mono {
    font-family: var(--font-mono, monospace);
    color: var(--steel, #86868B);
  }
  .bank-account-actions {
    display: flex;
    gap: 6px;
    flex-shrink: 0;
  }
  .inactive-badge {
    font-size: 10px;
    padding: 2px 6px;
    background: var(--surface-warning);
    color: white;
    border-radius: 3px;
    text-transform: uppercase;
    font-weight: 600;
  }
  .currency-chip {
    padding: 1px 6px;
    background: var(--onyx, #1D1D1F);
    color: white;
    border-radius: 3px;
    font-weight: 600;
    font-size: 10px;
  }
  .btn-ghost.btn-sm {
    padding: 0 10px;
    height: 28px;
    font-size: 12px;
  }

  @media (max-width: 1200px) {
    .pilot-readiness-row,
    .rollout-ops-row {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .deployment-card {
      flex-direction: column;
      align-items: flex-start;
    }
    .readability-card {
      grid-template-columns: 1fr;
    }
  }
</style>
