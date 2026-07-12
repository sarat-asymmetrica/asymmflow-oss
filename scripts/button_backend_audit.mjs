#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const inventoryPath = path.join(repoRoot, "docs/testing/UI_BUTTON_INVENTORY_2026_04_17.json");
const outPath = path.join(repoRoot, "docs/testing/UI_BACKEND_ACTION_AUDIT_2026_04_17.md");
const outJsonPath = path.join(repoRoot, "docs/testing/UI_BACKEND_ACTION_AUDIT_2026_04_17.json");

if (!fs.existsSync(inventoryPath)) {
  console.error("Missing inventory JSON. Run: node scripts/button_inventory.mjs");
  process.exit(1);
}

const inventory = JSON.parse(fs.readFileSync(inventoryPath, "utf8"));

function lineNumber(source, index) {
  return source.slice(0, index).split(/\r?\n/).length;
}

function unique(values) {
  return [...new Set(values.filter(Boolean))];
}

function row(values) {
  return `| ${values.map((value) => String(value || "").replace(/\|/g, "\\|").replace(/\n/g, " ")).join(" | ")} |`;
}

function readSource(rel) {
  return fs.readFileSync(path.join(repoRoot, rel), "utf8");
}

function extractImports(source) {
  const imports = new Set();
  const namedRe = /import\s*\{([^}]+)\}\s*from\s*["'][^"']+["'];?/g;
  let match;
  while ((match = namedRe.exec(source))) {
    match[1]
      .split(",")
      .map((part) => part.trim().replace(/\sas\s.+$/, ""))
      .filter(Boolean)
      .forEach((name) => imports.add(name));
  }
  const defaultRe = /import\s+([A-Z_a-z][\w$]*)\s+from\s*["'][^"']+["'];?/g;
  while ((match = defaultRe.exec(source))) {
    imports.add(match[1]);
  }
  return imports;
}

function extractBackendImports(source) {
  const imports = new Set();
  const re = /import\s*\{([^}]+)\}\s*from\s*["'][^"']*wailsjs\/go\/main\/App["'];?/g;
  let match;
  while ((match = re.exec(source))) {
    match[1]
      .split(",")
      .map((part) => part.trim().replace(/\sas\s.+$/, ""))
      .filter(Boolean)
      .forEach((name) => imports.add(name));
  }
  return imports;
}

function findMatchingBrace(source, openIndex) {
  let quote = "";
  let depth = 0;
  for (let i = openIndex; i < source.length; i += 1) {
    const ch = source[i];
    const prev = i > 0 ? source[i - 1] : "";
    if (quote) {
      if (ch === quote && prev !== "\\") quote = "";
      continue;
    }
    if (ch === '"' || ch === "'" || ch === "`") {
      quote = ch;
      continue;
    }
    if (ch === "{") depth += 1;
    if (ch === "}") depth -= 1;
    if (depth === 0) return i;
  }
  return -1;
}

function extractFunctionBodies(source) {
  const bodies = new Map();
  const fnRe = /(?:async\s+)?function\s+([A-Za-z_$][\w$]*)\s*\([^)]*\)\s*\{/g;
  let match;
  while ((match = fnRe.exec(source))) {
    const openIndex = source.indexOf("{", match.index);
    const closeIndex = findMatchingBrace(source, openIndex);
    if (closeIndex !== -1) {
      bodies.set(match[1], {
        line: lineNumber(source, match.index),
        body: source.slice(openIndex + 1, closeIndex),
      });
      fnRe.lastIndex = closeIndex + 1;
    }
  }

  const constFnRe = /(?:const|let)\s+([A-Za-z_$][\w$]*)\s*=\s*(?:async\s*)?(?:\([^)]*\)|[A-Za-z_$][\w$]*)\s*=>\s*\{/g;
  while ((match = constFnRe.exec(source))) {
    const openIndex = source.indexOf("{", match.index);
    const closeIndex = findMatchingBrace(source, openIndex);
    if (closeIndex !== -1) {
      bodies.set(match[1], {
        line: lineNumber(source, match.index),
        body: source.slice(openIndex + 1, closeIndex),
      });
      constFnRe.lastIndex = closeIndex + 1;
    }
  }

  const constExprRe = /(?:const|let)\s+([A-Za-z_$][\w$]*)\s*=\s*(?:async\s*)?(?:\([^)]*\)|[A-Za-z_$][\w$]*)\s*=>\s*([^;\n]+)/g;
  while ((match = constExprRe.exec(source))) {
    if (!bodies.has(match[1])) {
      bodies.set(match[1], {
        line: lineNumber(source, match.index),
        body: match[2],
      });
    }
  }

  return bodies;
}

function extractDeclaredNames(source) {
  const declared = new Set();
  const patterns = [
    /(?:async\s+)?function\s+([A-Za-z_$][\w$]*)\s*\(/g,
    /(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*=/g,
  ];
  for (const re of patterns) {
    let match;
    while ((match = re.exec(source))) declared.add(match[1]);
  }
  return declared;
}

function calledNames(text) {
  const names = [];
  const re = /\b([A-Za-z_$][\w$]*)\s*\(/g;
  let match;
  while ((match = re.exec(text))) {
    const name = match[1];
    if (["if", "for", "while", "switch", "catch", "setTimeout", "console"].includes(name)) continue;
    names.push(name);
  }
  return unique(names);
}

function directIdentifier(click) {
  const trimmed = (click || "").trim();
  if (/^[A-Za-z_$][\w$]*$/.test(trimmed)) return trimmed;
  return "";
}

function firstFunctionCalls(click) {
  const stripped = (click || "")
    .replace(/^async\s*\([^)]*\)\s*=>\s*/, "")
    .replace(/^\([^)]*\)\s*=>\s*/, "")
    .replace(/^[A-Za-z_$][\w$]*\s*=>\s*/, "");
  return calledNames(stripped);
}

function handlerReferences(click) {
  const direct = directIdentifier(click);
  return direct ? [direct] : firstFunctionCalls(click);
}

function containsBackendCall(text, backendImports) {
  return [...backendImports].filter((name) => new RegExp(`\\b${name}\\s*\\(`).test(text));
}

function classifyButton(button, screen) {
  const click = button.click || "";
  const label = button.label || "";
  const sourceInfo = screen.sourceInfo;

  if (!click || click === "(no explicit click handler)") {
    if (button.dataAction) {
      if (sourceInfo.hasDelegatedDataAction) {
        return {
          category: "Delegated DataTable",
          status: "Needs Runtime",
          backendMethods: [],
          notes: `Rendered HTML action '${button.dataAction}' handled by delegated row/table click logic.`,
        };
      }
      return {
        category: "Suspicious",
        status: "Needs Fix/Review",
        backendMethods: [],
        notes: `Rendered HTML action '${button.dataAction}' has no detected delegated click handler.`,
      };
    }
    if (button.type === "submit" || /submit/i.test(label)) {
      return { category: "Form Submit", status: "Needs Runtime", backendMethods: [], notes: "No click handler; may submit parent form." };
    }
    return { category: "Suspicious", status: "Needs Fix/Review", backendMethods: [], notes: "No explicit click handler." };
  }

  const direct = directIdentifier(click);
  const references = direct ? [direct] : firstFunctionCalls(click);
  const missing = references.filter((name) => !sourceInfo.knownNames.has(name) && !sourceInfo.backendImports.has(name));
  const directBackend = containsBackendCall(click, sourceInfo.backendImports);
  let backendMethods = [...directBackend];
  let nestedBackend = [];
  let handlerFound = false;

  for (const ref of references) {
    const body = sourceInfo.functionBodies.get(ref)?.body;
    if (body) {
      handlerFound = true;
      backendMethods.push(...containsBackendCall(body, sourceInfo.backendImports));
      const nestedCalls = calledNames(body);
      for (const nested of nestedCalls) {
        const nestedBody = sourceInfo.functionBodies.get(nested)?.body;
        if (nestedBody) nestedBackend.push(...containsBackendCall(nestedBody, sourceInfo.backendImports));
      }
    }
  }

  backendMethods = unique([...backendMethods, ...nestedBackend]);

  if (backendMethods.length > 0) {
    return {
      category: "Backend-Backed",
      status: "Backend Method Mapped",
      backendMethods,
      notes: handlerFound ? "Handler body reaches Wails binding." : "Click expression reaches Wails binding.",
    };
  }

  if (missing.length > 0) {
    return {
      category: "Suspicious",
      status: "Needs Fix/Review",
      backendMethods: [],
      notes: `References unresolved symbol(s): ${missing.join(", ")}.`,
    };
  }

  if (/show[A-Z]\w*\s*=|active\w*\s*=|selected\w*\s*=|current\w*\s*=|dispatch\(|window\.dispatchEvent|toast\.|toggle|close|open|cancel|reset|select|filter|sort/i.test(click)) {
    return { category: "UI/Event", status: "UI Wiring Present", backendMethods: [], notes: "Appears to change local UI state, dispatch an event, or open/close a modal." };
  }

  if (references.length > 0 && references.every((name) => sourceInfo.knownNames.has(name))) {
    return { category: "Handler Present", status: "Needs Runtime", backendMethods: [], notes: "Handler exists but no direct Wails call detected in first two hops." };
  }

  return { category: "Unknown", status: "Needs Runtime", backendMethods: [], notes: "Static audit cannot classify this action." };
}

const screens = inventory.screens.map((screen) => {
  const source = readSource(screen.file);
  const imports = extractImports(source);
  const backendImports = extractBackendImports(source);
  const declared = extractDeclaredNames(source);
  const functionBodies = extractFunctionBodies(source);
  const knownNames = new Set([...imports, ...declared, ...backendImports]);
  const sourceInfo = { imports, backendImports, declared, functionBodies, knownNames };
  sourceInfo.knownNames.add("confirm");
  sourceInfo.knownNames.add("alert");
  sourceInfo.knownNames.add("prompt");
  sourceInfo.hasDelegatedDataAction = /dataset\.action|closest\(\s*['"]\[data-action\]['"]\s*\)/.test(source);
  const buttons = screen.buttons.map((button) => ({
    ...button,
    ...classifyButton(button, { ...screen, sourceInfo }),
  }));
  return { ...screen, sourceInfo, buttons };
});

const allButtons = screens.flatMap((screen) => screen.buttons.map((button) => ({ ...button, screen })));
const categoryCounts = new Map();
const statusCounts = new Map();
for (const button of allButtons) {
  categoryCounts.set(button.category, (categoryCounts.get(button.category) || 0) + 1);
  statusCounts.set(button.status, (statusCounts.get(button.status) || 0) + 1);
}

const suspicious = allButtons.filter((button) => button.category === "Suspicious");
const backendBacked = allButtons.filter((button) => button.category === "Backend-Backed");
const needsRuntime = allButtons.filter((button) => ["Needs Runtime", "Needs Fix/Review"].includes(button.status));

const lines = [];
lines.push("# UI Backend-First Button Audit - 2026-04-17");
lines.push("");
lines.push("Purpose: first backend-oriented classification of the button inventory. This is not a full click-through result yet; it identifies which actions are backend-backed, UI-only, suspicious, or runtime-only.");
lines.push("");
lines.push("Generated by `node scripts/button_backend_audit.mjs` after `node scripts/button_inventory.mjs`.");
lines.push("");
lines.push("## Summary");
lines.push("");
lines.push(`- Screen-level actions audited: ${allButtons.length}`);
lines.push(`- Backend-backed actions mapped: ${backendBacked.length}`);
lines.push(`- Suspicious/unwired actions: ${suspicious.length}`);
lines.push(`- Actions needing runtime/manual verification: ${needsRuntime.length}`);
lines.push("");
lines.push("### Category Counts");
lines.push("");
lines.push(row(["Category", "Count"]));
lines.push(row(["---", "---"]));
for (const [category, count] of [...categoryCounts.entries()].sort((a, b) => b[1] - a[1])) {
  lines.push(row([category, count]));
}
lines.push("");
lines.push("### Status Counts");
lines.push("");
lines.push(row(["Status", "Count"]));
lines.push(row(["---", "---"]));
for (const [status, count] of [...statusCounts.entries()].sort((a, b) => b[1] - a[1])) {
  lines.push(row([status, count]));
}
lines.push("");
lines.push("## Suspicious / Unwired Actions");
lines.push("");
if (suspicious.length === 0) {
  lines.push("_No suspicious actions found by static audit._");
} else {
  lines.push(row(["Screen", "File:Line", "Label", "Handler", "Reason"]));
  lines.push(row(["---", "---", "---", "---", "---"]));
  for (const button of suspicious) {
    lines.push(row([
      button.screen.pageLabel,
      `${button.screen.file}:${button.line}`,
      button.label,
      button.click,
      button.notes,
    ]));
  }
}
lines.push("");
lines.push("## Backend-Backed Actions By Screen");
lines.push("");
for (const screen of screens) {
  const screenBackend = screen.buttons.filter((button) => button.category === "Backend-Backed");
  if (screenBackend.length === 0) continue;
  lines.push(`### ${screen.pageLabel}`);
  lines.push(`File: \`${screen.file}\``);
  lines.push("");
  lines.push(row(["Line", "Label", "Handler", "Mapped Backend Method(s)", "Backend Test Status", "Notes"]));
  lines.push(row(["---", "---", "---", "---", "---", "---"]));
  for (const button of screenBackend) {
    lines.push(row([
      button.line,
      button.label,
      button.click,
      button.backendMethods.map((name) => `\`${name}\``).join(", "),
      "Untested",
      button.notes,
    ]));
  }
  lines.push("");
}
lines.push("## Runtime-Only / UI Actions By Screen");
lines.push("");
for (const screen of screens) {
  const rest = screen.buttons.filter((button) => button.category !== "Backend-Backed" && button.category !== "Suspicious");
  if (rest.length === 0) continue;
  lines.push(`### ${screen.pageLabel}`);
  lines.push(`File: \`${screen.file}\``);
  lines.push("");
  lines.push(row(["Line", "Category", "Label", "Handler", "Status", "Notes"]));
  lines.push(row(["---", "---", "---", "---", "---", "---"]));
  for (const button of rest) {
    lines.push(row([
      button.line,
      button.category,
      button.label,
      button.click,
      button.status,
      button.notes,
    ]));
  }
  lines.push("");
}

fs.writeFileSync(outPath, `${lines.join("\n")}\n`);

function serializeButton(button, index) {
  const references = handlerReferences(button.click || "");
  const resolvedHandlers = references.map((name) => {
    const body = button.screen.sourceInfo.functionBodies.get(name);
    return {
      name,
      line: body?.line || null,
      resolved: Boolean(body || button.screen.sourceInfo.backendImports.has(name) || button.screen.sourceInfo.knownNames.has(name)),
    };
  });
  return {
    audit_id: `BTN-${String(index + 1).padStart(3, "0")}`,
    screen: button.screen.pageLabel,
    file: button.screen.file,
    line: button.line,
    label: button.label,
    click: button.click,
    data_action: button.dataAction || "",
    type: button.type || "",
    category: button.category,
    status: button.status,
    backend_methods: button.backendMethods || [],
    notes: button.notes,
    handler_references: references,
    resolved_handlers: resolvedHandlers,
  };
}

const serializableButtons = allButtons.map(serializeButton);
const auditJson = {
  generated_at: new Date().toISOString(),
  summary: {
    actions_audited: allButtons.length,
    backend_backed: backendBacked.length,
    suspicious_unwired: suspicious.length,
    runtime_manual: needsRuntime.length,
  },
  category_counts: Object.fromEntries([...categoryCounts.entries()].sort((a, b) => b[1] - a[1])),
  status_counts: Object.fromEntries([...statusCounts.entries()].sort((a, b) => b[1] - a[1])),
  buttons: serializableButtons,
  runtime_manual_actions: serializableButtons
    .filter((button) => ["Needs Runtime", "Needs Fix/Review"].includes(button.status))
    .map((button, index) => ({
      ...button,
      runtime_id: `RT-${String(index + 1).padStart(3, "0")}`,
    })),
};
fs.writeFileSync(outJsonPath, `${JSON.stringify(auditJson, null, 2)}\n`);

console.log(`Wrote ${path.relative(repoRoot, outPath)}`);
console.log(`Wrote ${path.relative(repoRoot, outJsonPath)}`);
console.log(`Actions audited: ${allButtons.length}`);
console.log(`Backend-backed: ${backendBacked.length}`);
console.log(`Suspicious/unwired: ${suspicious.length}`);
console.log(`Needs runtime/manual: ${needsRuntime.length}`);
