#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const screensDir = path.join(repoRoot, "frontend/src/lib/screens");
const componentsDir = path.join(repoRoot, "frontend/src/lib/components");
const outDir = path.join(repoRoot, "docs/testing");
const outFile = path.join(outDir, "UI_BUTTON_INVENTORY_2026_04_17.md");
const outJsonFile = path.join(outDir, "UI_BUTTON_INVENTORY_2026_04_17.json");

const screenPageLabels = new Map([
  ["DashboardScreen.svelte", "Dashboard"],
  ["SalesHub.svelte", "Opportunities Hub"],
  ["OpportunitiesScreen.svelte", "Opportunities > RFQs"],
  ["CostingSheetScreen.svelte", "Opportunities > Costing"],
  ["OffersScreen.svelte", "Opportunities > Offers"],
  ["OrdersScreen.svelte", "Opportunities > Customer Orders"],
  ["OperationsHub.svelte", "Operations Hub"],
  ["PurchaseOrdersScreen.svelte", "Operations > Supplier POs"],
  ["SupplierInvoicesScreen.svelte", "Operations > Supplier Invoices"],
  ["DeliveryNotesScreen.svelte", "Operations > Delivery Notes"],
  ["FinanceHub.svelte", "Finance Hub"],
  ["FinancialDashboard.svelte", "Finance > Dashboard (Acme Instrumentation)"],
  ["AHSDashboard.svelte", "Finance > Dashboard (Beacon Controls)"],
  ["InvoicesScreen.svelte", "Finance > Customer Invoices"],
  ["PaymentsScreen.svelte", "Finance > Payments Received"],
  ["SupplierPaymentsScreen.svelte", "Finance > Payments Made"],
  ["ExpensesScreen.svelte", "Finance > Expenses / Approvals"],
  ["PayrollScreen.svelte", "Finance > Payroll"],
  ["BankReconciliationScreen.svelte", "Finance > Bank Recon"],
  ["BookBankReconciliationScreen.svelte", "Finance > Book-Bank Recon (inactive tab)"],
  ["ChequeRegisterScreen.svelte", "Finance > Cheque Register (inactive tab)"],
  ["FXRevaluationScreen.svelte", "Finance > FX Revaluation (inactive tab)"],
  ["AuditTrailViewer.svelte", "Finance > Audit Trail (inactive tab)"],
  ["CRMHub.svelte", "Relationships Hub"],
  ["CRMCustomerDashboard.svelte", "Relationships > Customers"],
  ["CRMSupplierDashboard.svelte", "Relationships > Suppliers"],
  ["CustomerDetailView.svelte", "Relationships > Customer Detail"],
  ["SupplierDetailView.svelte", "Relationships > Supplier Detail"],
  ["IntelligenceHub.svelte", "Intelligence Hub"],
  ["ButlerScreen.svelte", "Intelligence > Butler"],
  ["ArchaeologistScreen.svelte", "Intelligence > Archaeologist"],
  ["EntityDiscoveryScreen.svelte", "Intelligence > Entity Discovery"],
  ["WorkHub.svelte", "Work"],
  ["PeopleHub.svelte", "People"],
  ["NotificationsScreen.svelte", "Notifications"],
  ["DeploymentHub.svelte", "Deployment"],
  ["SettingsScreen.svelte", "Settings"],
  ["UserManagementScreen.svelte", "User Management"],
  ["ReportsScreen.svelte", "Reports"],
  ["OneDriveImportScreen.svelte", "OneDrive Import"],
  ["RFQScreen.svelte", "RFQ Management"],
  ["PricingScreen.svelte", "Pricing"],
  ["Customer360.svelte", "Customer 360"],
  ["CustomersScreen.svelte", "Customer Master"],
  ["SuppliersScreen.svelte", "Supplier Master"],
  ["GRNScreen.svelte", "GRN (deprecated/inactive)"],
  ["DeliveryTrackingScreen.svelte", "Delivery Tracking"],
  ["AccountingScreen.svelte", "Accounting"],
  ["PayrollScreen.svelte", "Payroll"],
  ["LoginScreen.svelte", "Login"],
  ["LicenseActivationScreen.svelte", "License Activation"],
  ["PendingApprovalScreen.svelte", "Pending Approval"],
  ["SetupAdminScreen.svelte", "Setup Admin"],
  ["SetupWizard.svelte", "Setup Wizard"],
  ["ArrivalCeremony.svelte", "Arrival Ceremony"],
  ["ShowcaseScreen.svelte", "Design Showcase"],
  ["QuotationScreen.svelte", "Quotation"],
  ["CashPositionWidget.svelte", "Cash Position Widget"],
]);

function walk(dir, suffix = ".svelte") {
  const results = [];
  if (!fs.existsSync(dir)) return results;
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      results.push(...walk(full, suffix));
    } else if (entry.name.endsWith(suffix)) {
      results.push(full);
    }
  }
  return results.sort();
}

function lineNumber(source, index) {
  return source.slice(0, index).split(/\r?\n/).length;
}

function stripSvelte(text) {
  return text
    .replace(/<script[\s\S]*?<\/script>/gi, "")
    .replace(/<style[\s\S]*?<\/style>/gi, "")
    .replace(/<!--[\s\S]*?-->/g, "")
    .replace(/<[^>]+>/g, " ")
    .replace(/\{#if[\s\S]*?\}/g, "")
    .replace(/\{\/if\}/g, "")
    .replace(/\{#each[\s\S]*?\}/g, "")
    .replace(/\{\/each\}/g, "")
    .replace(/\{:[\s\S]*?\}/g, "")
    .replace(/\s+/g, " ")
    .trim();
}

function cleanLabel(raw, attrs) {
  const fromText = stripSvelte(raw);
  if (fromText) return fromText;
  const aria = attrs.match(/aria-label\s*=\s*["']([^"']+)["']/)?.[1];
  if (aria) return aria;
  const title = attrs.match(/title\s*=\s*["']([^"']+)["']/)?.[1];
  if (title) return title;
  const value = attrs.match(/value\s*=\s*["']([^"']+)["']/)?.[1];
  if (value) return value;
  return "(icon/dynamic label)";
}

function attrValue(attrs, name) {
  const stringMatch = attrs.match(new RegExp(`${name}\\s*=\\s*["']([^"']+)["']`));
  if (stringMatch) return stringMatch[1].replace(/\s+/g, " ").trim();
  const exprMatch = attrs.match(new RegExp(`${name}\\s*=\\s*\\{([^}]+)\\}`));
  if (exprMatch) return `{${exprMatch[1].replace(/\s+/g, " ").trim()}}`;
  return "";
}

function clickHandler(attrs) {
  const idx = attrs.indexOf("on:click");
  if (idx === -1) return "";
  const after = attrs.slice(idx);
  const brace = after.indexOf("{");
  if (brace === -1) return "(click event forwarded/no explicit handler)";
  let depth = 0;
  for (let i = brace; i < after.length; i += 1) {
    const ch = after[i];
    if (ch === "{") depth += 1;
    if (ch === "}") depth -= 1;
    if (depth === 0) {
      return after.slice(brace + 1, i).replace(/\s+/g, " ").trim();
    }
  }
  return after.slice(brace + 1).replace(/\s+/g, " ").trim();
}

function findOpeningTagEnd(source, start) {
  let quote = "";
  let braceDepth = 0;
  for (let i = start; i < source.length; i += 1) {
    const ch = source[i];
    const prev = i > 0 ? source[i - 1] : "";
    if (quote) {
      if (ch === quote && prev !== "\\") quote = "";
      continue;
    }
    if (ch === '"' || ch === "'") {
      quote = ch;
      continue;
    }
    if (ch === "{") {
      braceDepth += 1;
      continue;
    }
    if (ch === "}" && braceDepth > 0) {
      braceDepth -= 1;
      continue;
    }
    if (ch === ">" && braceDepth === 0) return i;
  }
  return -1;
}

function findClosingTag(source, tagName, bodyStart) {
  const closeRe = new RegExp(`</${tagName}\\s*>`, "g");
  closeRe.lastIndex = bodyStart;
  const match = closeRe.exec(source);
  if (!match) return { start: -1, end: -1 };
  return { start: match.index, end: match.index + match[0].length };
}

function findElements(source, tagName) {
  const elements = [];
  const openRe = new RegExp(`<${tagName}\\b`, "g");
  let match;
  while ((match = openRe.exec(source))) {
    const openStart = match.index;
    const openEnd = findOpeningTagEnd(source, openStart);
    if (openEnd === -1) continue;
    const attrs = source.slice(openStart + tagName.length + 1, openEnd);
    if (/\/\s*$/.test(attrs)) {
      elements.push({ index: openStart, attrs, body: "" });
      openRe.lastIndex = openEnd + 1;
      continue;
    }
    const bodyStart = openEnd + 1;
    const close = findClosingTag(source, tagName, bodyStart);
    if (close.start === -1) {
      openRe.lastIndex = openEnd + 1;
      continue;
    }
    elements.push({
      index: openStart,
      attrs,
      body: source.slice(bodyStart, close.start),
    });
    openRe.lastIndex = close.end;
  }
  return elements;
}

function findRoleButtonElements(source) {
  const elements = [];
  const openRe = /<([a-zA-Z][a-zA-Z0-9:-]*)\b/g;
  let match;
  while ((match = openRe.exec(source))) {
    const tagName = match[1];
    if (["button", "Button", "WabiButton"].includes(tagName)) continue;
    const openStart = match.index;
    const openEnd = findOpeningTagEnd(source, openStart);
    if (openEnd === -1) continue;
    const attrs = source.slice(openStart + tagName.length + 1, openEnd);
    openRe.lastIndex = openEnd + 1;
    if (!/role\s*=\s*["']button["']/.test(attrs) || !attrs.includes("on:click")) continue;
    if (/modal-(overlay|backdrop)/.test(attrs) || attrs.includes("on:click|self")) continue;
    const bodyStart = openEnd + 1;
    const close = findClosingTag(source, tagName, bodyStart);
    if (close.start === -1) continue;
    elements.push({
      tagName,
      index: openStart,
      attrs,
      body: source.slice(bodyStart, close.start),
    });
  }
  return elements;
}

function backendImports(source) {
  const imports = new Set();
  const re = /import\s*\{([^}]+)\}\s*from\s*["'][^"']*wailsjs\/go\/main\/App["'];?/g;
  let match;
  while ((match = re.exec(source))) {
    match[1]
      .split(",")
      .map((name) => name.trim().replace(/\sas\s.+$/, ""))
      .filter(Boolean)
      .forEach((name) => imports.add(name));
  }
  return [...imports].sort();
}

function importedLocalComponents(source) {
  const components = new Set();
  const re = /import\s+([A-Z][A-Za-z0-9_]*)\s+from\s+["']([^"']+\.svelte)["'];?/g;
  let match;
  while ((match = re.exec(source))) {
    const importPath = match[2];
    if (!importPath.includes("components")) continue;
    components.add(`${match[1]} (${importPath})`);
  }
  return [...components].sort();
}

function extractButtons(source) {
  const buttons = [];
  const patterns = [
    { kind: "button", tag: "button" },
    { kind: "Button", tag: "Button" },
    { kind: "WabiButton", tag: "WabiButton" },
  ];

  for (const { kind, tag } of patterns) {
    for (const element of findElements(source, tag)) {
      const attrs = element.attrs || "";
      const body = element.body || "";
      buttons.push({
        kind,
        line: lineNumber(source, element.index),
        label: cleanLabel(body, attrs),
        click: clickHandler(attrs),
        dataAction: attrValue(attrs, "data-action"),
        type: attrValue(attrs, "type") || (kind === "button" ? "" : "component default"),
        title: attrValue(attrs, "title"),
        aria: attrValue(attrs, "aria-label"),
        disabled: /\bdisabled(?:=|\s|>)/.test(attrs),
        loading: /\bloading(?:=|\s|>)/.test(attrs),
      });
    }
  }

  for (const element of findRoleButtonElements(source)) {
    const attrs = element.attrs || "";
    buttons.push({
      kind: `${element.tagName} role=button`,
      line: lineNumber(source, element.index),
      label: cleanLabel(element.body || "", attrs),
      click: clickHandler(attrs),
      dataAction: attrValue(attrs, "data-action"),
      type: "",
      title: attrValue(attrs, "title"),
      aria: attrValue(attrs, "aria-label"),
      disabled: /\bdisabled(?:=|\s|>)/.test(attrs),
      loading: /\bloading(?:=|\s|>)/.test(attrs),
    });
  }

  return buttons.sort((a, b) => a.line - b.line);
}

function extractTabs(source) {
  const tabs = [];
  const literalTabs = source.match(/const\s+tabs\s*[:\w\s\[\]]*=\s*\[([\s\S]*?)\];/);
  if (!literalTabs) return tabs;
  const itemRe = /\{\s*id\s*:\s*["']([^"']+)["']\s*,\s*label\s*:\s*["']([^"']+)["']/g;
  let match;
  while ((match = itemRe.exec(literalTabs[1]))) {
    tabs.push({ id: match[1], label: match[2] });
  }
  return tabs;
}

function possibleBackendCalls(source, imported) {
  const calls = [];
  for (const fn of imported) {
    const re = new RegExp(`\\b${fn}\\s*\\(`, "g");
    let match;
    while ((match = re.exec(source))) {
      calls.push({ fn, line: lineNumber(source, match.index) });
    }
  }
  return calls.sort((a, b) => a.line - b.line);
}

function row(values) {
  return `| ${values.map((value) => String(value || "").replace(/\|/g, "\\|").replace(/\n/g, " ")).join(" | ")} |`;
}

function renderScreenSection(file, source) {
  const base = path.basename(file);
  const rel = path.relative(repoRoot, file);
  const pageLabel = screenPageLabels.get(base) || base.replace(/\.svelte$/, "");
  const buttons = extractButtons(source);
  const tabs = extractTabs(source);
  const backend = backendImports(source);
  const backendCalls = possibleBackendCalls(source, backend);
  const components = importedLocalComponents(source);

  const lines = [];
  lines.push(`### ${pageLabel}`);
  lines.push(`File: \`${rel}\``);
  lines.push("");
  if (tabs.length > 0) {
    lines.push(`Tabs/views detected: ${tabs.map((tab) => `\`${tab.label}\``).join(", ")}`);
    lines.push("");
  }
  if (backend.length > 0) {
    lines.push(`Backend bindings imported: ${backend.map((name) => `\`${name}\``).join(", ")}`);
    lines.push("");
  }
  if (backendCalls.length > 0) {
    const uniqueCalls = [...new Set(backendCalls.map((call) => call.fn))];
    lines.push(`Backend calls seen in source: ${uniqueCalls.map((name) => `\`${name}\``).join(", ")}`);
    lines.push("");
  }
  if (components.length > 0) {
    lines.push(`Button-bearing nested components to inspect in rendered UI: ${components.map((name) => `\`${name}\``).join(", ")}`);
    lines.push("");
  }
  if (buttons.length === 0) {
    lines.push("_No literal `<button>`, `<Button>`, or `<WabiButton>` usage detected in this screen file._");
  } else {
    lines.push(row(["Line", "Kind", "Visible Label", "Click Handler / Action", "State Notes", "Backend Test Status", "App Test Status", "Issue ID"]));
    lines.push(row(["---", "---", "---", "---", "---", "---", "---", "---"]));
    for (const button of buttons) {
      const state = [
        button.disabled ? "disabled-bound/static" : "",
        button.loading ? "loading-bound/static" : "",
        button.type ? `type ${button.type}` : "",
        button.title ? `title ${button.title}` : "",
        button.aria ? `aria ${button.aria}` : "",
      ].filter(Boolean).join("; ");
      lines.push(row([
        button.line,
        button.kind,
        button.label,
        button.click || "(no explicit click handler)",
        state,
        "Untested",
        "Untested",
        "",
      ]));
    }
  }
  lines.push("");
  return {
    pageLabel,
    rel,
    buttons,
    tabs,
    backend,
    backendCalls,
    markdown: lines.join("\n"),
  };
}

function renderComponentSection(file, source) {
  const rel = path.relative(repoRoot, file);
  const buttons = extractButtons(source);
  if (buttons.length === 0) return "";
  const lines = [];
  lines.push(`### ${path.basename(file)}`);
  lines.push(`File: \`${rel}\``);
  lines.push("");
  lines.push(row(["Line", "Kind", "Visible Label", "Click Handler / Action", "Notes"]));
  lines.push(row(["---", "---", "---", "---", "---"]));
  for (const button of buttons) {
    lines.push(row([
      button.line,
      button.kind,
      button.label,
      button.click || "(no explicit click handler)",
      [button.disabled ? "disabled-bound/static" : "", button.loading ? "loading-bound/static" : ""].filter(Boolean).join("; "),
    ]));
  }
  lines.push("");
  return lines.join("\n");
}

const screenFiles = walk(screensDir);
const componentFiles = walk(componentsDir);
const screenSections = screenFiles.map((file) => renderScreenSection(file, fs.readFileSync(file, "utf8")));
const componentSections = componentFiles
  .map((file) => renderComponentSection(file, fs.readFileSync(file, "utf8")))
  .filter(Boolean);

const totalButtons = screenSections.reduce((sum, section) => sum + section.buttons.length, 0);
const screensWithButtons = screenSections.filter((section) => section.buttons.length > 0).length;
const screensWithBackend = screenSections.filter((section) => section.backend.length > 0).length;
const totalComponentButtonSections = componentSections.length;

const activePageMap = [
  "Top nav: Dashboard, Opportunities, Operations, Finance, Work, People, Notifications, Relationships, Intelligence, Settings/Deployment by permission.",
  "Opportunities tabs: RFQs, Costing, Offers, Customer Orders.",
  "Operations tabs: Supplier POs, Supplier Invoices, Delivery Notes.",
  "Finance tabs: Dashboard, Customer Invoices, Payments Received, Payments Made, Expenses, Approvals, Payroll, Bank Recon.",
  "Relationships tabs: Customers, Suppliers; detail views open after row/card selection.",
  "Entry/admin pages: Login, License Activation, Pending Approval, Setup Admin, Setup Wizard, User Management, Deployment.",
];

const doc = [
  "# UI Button Inventory - 2026-04-17",
  "",
  "Purpose: static first-pass inventory of page buttons for the backend-first/manual-app testing methodology.",
  "",
  "This file is generated by `node scripts/button_inventory.mjs`. It intentionally keeps `Backend Test Status`, `App Test Status`, and `Issue ID` columns open so we can mark results as we test.",
  "",
  "## Summary",
  "",
  `- Screen files scanned: ${screenSections.length}`,
  `- Screen files with detected buttons: ${screensWithButtons}`,
  `- Screen-level buttons detected: ${totalButtons}`,
  `- Screen files importing Wails backend bindings: ${screensWithBackend}`,
  `- Reusable component files with buttons: ${totalComponentButtonSections}`,
  "",
  "## Active Page Map",
  "",
  ...activePageMap.map((item) => `- ${item}`),
  "",
  "## Extraction Rules And Gaps",
  "",
  "- Captured: literal `<button>`, design-system `<Button>`, `<WabiButton>`, and elements with `role=\"button\"` plus `on:click`.",
  "- Captured: static `const tabs = [...]` labels, Wails imports, and direct calls to imported Wails bindings.",
  "- Needs manual/runtime pass: labels rendered from variables such as `{tab.label}` or `{action.label}`.",
  "- Needs manual/runtime pass: row actions produced through `DataTable` render functions, `{@html ...}`, string templates, or nested components.",
  "- Needs manual/runtime pass: modal buttons that are mounted only after a specific data row is selected or an API result arrives.",
  "- Backend status should be marked against the Wails method/function behavior, not just against click availability.",
  "",
  "## Screen Inventory",
  "",
  ...screenSections.map((section) => section.markdown),
  "## Reusable Button-Bearing Components",
  "",
  "These components can appear inside pages and may add buttons not visible in the parent screen source.",
  "",
  ...componentSections,
].join("\n");

fs.mkdirSync(outDir, { recursive: true });
fs.writeFileSync(outFile, doc);
fs.writeFileSync(outJsonFile, `${JSON.stringify({
  generatedAt: new Date().toISOString(),
  screenCount: screenSections.length,
  screenLevelButtonCount: totalButtons,
  screensWithButtons,
  screensWithBackend,
  reusableComponentButtonSectionCount: totalComponentButtonSections,
  screens: screenSections.map((section) => ({
    pageLabel: section.pageLabel,
    file: section.rel,
    buttons: section.buttons,
    tabs: section.tabs,
    backendImports: section.backend,
    backendCalls: section.backendCalls,
  })),
}, null, 2)}\n`);
console.log(`Wrote ${path.relative(repoRoot, outFile)}`);
console.log(`Wrote ${path.relative(repoRoot, outJsonFile)}`);
console.log(`Screens scanned: ${screenSections.length}`);
console.log(`Screen-level buttons: ${totalButtons}`);
console.log(`Reusable components with buttons: ${totalComponentButtonSections}`);
