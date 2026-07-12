#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const screensDir = path.join(repoRoot, "frontend/src/lib/screens");
const componentsDir = path.join(repoRoot, "frontend/src/lib/components");
const outDir = path.join(repoRoot, "docs/testing");
const outMd = path.join(outDir, "UI_MODAL_INVENTORY_2026_04_17.md");
const outJson = path.join(outDir, "UI_MODAL_INVENTORY_2026_04_17.json");

function walk(dir, suffix = ".svelte") {
  const results = [];
  if (!fs.existsSync(dir)) return results;
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) results.push(...walk(full, suffix));
    else if (entry.name.endsWith(suffix)) results.push(full);
  }
  return results.sort();
}

function lineNumber(source, index) {
  return source.slice(0, index).split(/\r?\n/).length;
}

function cleanText(text) {
  return String(text || "")
    .replace(/<script[\s\S]*?<\/script>/gi, "")
    .replace(/<style[\s\S]*?<\/style>/gi, "")
    .replace(/<!--[\s\S]*?-->/g, "")
    .replace(/<[^>]+>/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

function row(values) {
  return `| ${values.map((value) => String(value || "").replace(/\|/g, "\\|").replace(/\n/g, " ")).join(" | ")} |`;
}

function attrValue(attrs, name) {
  const stringMatch = attrs.match(new RegExp(`${name}\\s*=\\s*["']([^"']+)["']`));
  if (stringMatch) return stringMatch[1].replace(/\s+/g, " ").trim();
  const exprMatch = attrs.match(new RegExp(`${name}\\s*=\\s*\\{([^}]+)\\}`));
  if (exprMatch) return `{${exprMatch[1].replace(/\s+/g, " ").trim()}}`;
  return "";
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
    const bodyStart = openEnd + 1;
    const close = findClosingTag(source, tagName, bodyStart);
    if (close.start === -1) {
      elements.push({ tagName, index: openStart, attrs, body: "" });
      openRe.lastIndex = openEnd + 1;
      continue;
    }
    elements.push({
      tagName,
      index: openStart,
      attrs,
      body: source.slice(bodyStart, close.start),
    });
    openRe.lastIndex = close.end;
  }
  return elements;
}

function findIfBlocks(source) {
  const blocks = [];
  const ifRe = /\{#if\s+([^}]*show[A-Za-z0-9_]*Modal[^}]*)\}/g;
  let match;
  while ((match = ifRe.exec(source))) {
    const start = match.index;
    const condition = match[1].trim();
    let idx = ifRe.lastIndex;
    let depth = 1;
    const tokenRe = /\{#if\b|\{\/if\}/g;
    tokenRe.lastIndex = idx;
    let token;
    while ((token = tokenRe.exec(source))) {
      if (token[0].startsWith("{#if")) depth += 1;
      else depth -= 1;
      if (depth === 0) {
        blocks.push({
          index: start,
          condition,
          body: source.slice(ifRe.lastIndex, token.index),
        });
        ifRe.lastIndex = tokenRe.lastIndex;
        break;
      }
    }
  }
  return blocks;
}

function extractBackendImports(source) {
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
      bodies.set(match[1], source.slice(openIndex + 1, closeIndex));
      fnRe.lastIndex = closeIndex + 1;
    }
  }

  const constFnRe = /(?:const|let)\s+([A-Za-z_$][\w$]*)\s*=\s*(?:async\s*)?(?:\([^)]*\)|[A-Za-z_$][\w$]*)\s*=>\s*\{/g;
  while ((match = constFnRe.exec(source))) {
    const openIndex = source.indexOf("{", match.index);
    const closeIndex = findMatchingBrace(source, openIndex);
    if (closeIndex !== -1) {
      bodies.set(match[1], source.slice(openIndex + 1, closeIndex));
      constFnRe.lastIndex = closeIndex + 1;
    }
  }

  return bodies;
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
    if (depth === 0) return after.slice(brace + 1, i).replace(/\s+/g, " ").trim();
  }
  return after.slice(brace + 1).replace(/\s+/g, " ").trim();
}

function extractButtons(source, block) {
  const buttons = [];
  for (const tag of ["button", "Button", "WabiButton"]) {
    for (const element of findElements(block, tag)) {
      buttons.push({
        line: lineNumber(source, source.indexOf(element.body, Math.max(0, source.indexOf(block) - 1))),
        kind: tag,
        label: cleanText(element.body) || attrValue(element.attrs, "aria-label") || attrValue(element.attrs, "title") || "(icon/dynamic)",
        handler: clickHandler(element.attrs),
        type: attrValue(element.attrs, "type"),
        disabled: /\bdisabled(?:=|\s|>)/.test(element.attrs),
      });
    }
  }
  return buttons;
}

function calledNames(text) {
  const names = [];
  const re = /\b([A-Za-z_$][\w$]*)\s*\(/g;
  let match;
  while ((match = re.exec(text || ""))) {
    const name = match[1];
    if (["if", "for", "while", "switch", "catch", "confirm", "alert", "setTimeout"].includes(name)) continue;
    names.push(name);
  }
  return [...new Set(names)];
}

function backendForButton(button, functionBodies, backendImports) {
  const text = button.handler || "";
  const methods = new Set();
  for (const backend of backendImports) {
    if (new RegExp(`\\b${backend}\\s*\\(`).test(text)) methods.add(backend);
  }
  const refs = new Set(calledNames(text));
  if (/^[A-Za-z_$][\w$]*$/.test(text.trim())) refs.add(text.trim());
  for (const name of refs) {
    const body = functionBodies.get(name);
    if (!body) continue;
    for (const backend of backendImports) {
      if (new RegExp(`\\b${backend}\\s*\\(`).test(body)) methods.add(backend);
    }
    for (const nested of calledNames(body)) {
      const nestedBody = functionBodies.get(nested);
      if (!nestedBody) continue;
      for (const backend of backendImports) {
        if (new RegExp(`\\b${backend}\\s*\\(`).test(nestedBody)) methods.add(backend);
      }
    }
  }
  return [...methods].sort();
}

function extractFields(block) {
  const fields = [];
  const labels = [...block.matchAll(/<label\b[^>]*>([\s\S]*?)<\/label>/g)].map((m) => cleanText(m[1]));
  fields.push(...labels.filter(Boolean));

  const controlRe = /<(input|select|textarea)\b([^>]*)>/g;
  let match;
  while ((match = controlRe.exec(block))) {
    const attrs = match[2];
    const bound = attrValue(attrs, "bind:value") || attrValue(attrs, "value") || attrValue(attrs, "name");
    const placeholder = attrValue(attrs, "placeholder");
    const type = attrValue(attrs, "type");
    const label = placeholder || bound || `${match[1]}${type ? `:${type}` : ""}`;
    if (label) fields.push(label);
  }

  return [...new Set(fields.map((field) => field.replace(/\s*\*\s*$/, " *")).filter(Boolean))];
}

function titleFromBlock(block, attrs = "") {
  const titleAttr = attrValue(attrs, "title");
  if (titleAttr) return titleAttr;
  const heading = block.match(/<h[1-4]\b[^>]*>([\s\S]*?)<\/h[1-4]>/);
  if (heading) return cleanText(heading[1]);
  return "";
}

function classifyModal(modal) {
  const lowerTitle = modal.title.toLowerCase();
  const saveButtons = modal.buttons.filter((button) => /save|create|update|submit|record|delete|confirm|approve|issue|generate|mark|apply|import|match|pay|dispatch/i.test(button.label));
  const cancelButtons = modal.buttons.filter((button) => /cancel|close|back/i.test(button.label));
  const backendButtons = modal.buttons.filter((button) => button.backendMethods.length > 0);
  const notes = [];
  if (!modal.title) notes.push("Missing/unknown title.");
  if (saveButtons.length > 0 && backendButtons.length === 0 && !/detail|view|lost|won|task/.test(lowerTitle)) {
    notes.push("Has submit-like button but no mapped backend method in first two handler hops.");
  }
  if (cancelButtons.length === 0 && !/detail|view|performing/.test(lowerTitle)) {
    notes.push("No visible cancel/close footer button detected.");
  }
  if (modal.fields.length === 0 && saveButtons.length > 0 && !/delete|confirm|lost|won|match/.test(lowerTitle)) {
    notes.push("Submit-like modal has no detected input fields.");
  }
  return notes.length > 0 ? notes.join(" ") : "No static issue detected.";
}

function screenLabel(rel) {
  return path.basename(rel).replace(/\.svelte$/, "");
}

function collectModals(file) {
  const rel = path.relative(repoRoot, file);
  const source = fs.readFileSync(file, "utf8");
  const backendImports = extractBackendImports(source);
  const functionBodies = extractFunctionBodies(source);
  const modals = [];
  const seen = new Set();

  for (const tag of ["Modal", "WabiModal", "ContextTaskModal"]) {
    for (const element of findElements(source, tag)) {
      const title = titleFromBlock(element.body, element.attrs);
      const open = attrValue(element.attrs, "bind:open") || attrValue(element.attrs, "open");
      const buttons = extractButtons(source, element.body).map((button) => ({
        ...button,
        backendMethods: backendForButton(button, functionBodies, backendImports),
      }));
      const modal = {
        screen: screenLabel(rel),
        file: rel,
        line: lineNumber(source, element.index),
        type: tag,
        title,
        open,
        fields: extractFields(element.body),
        buttons,
      };
      modal.staticStatus = classifyModal(modal);
      modals.push(modal);
      seen.add(`${element.index}:${tag}`);
    }
  }

  for (const block of findIfBlocks(source)) {
    if (!/modal-(backdrop|overlay)|class=["'][^"']*modal|role=["']dialog/.test(block.body)) continue;
    const key = `${block.index}:manual`;
    if (seen.has(key)) continue;
    const buttons = extractButtons(source, block.body).map((button) => ({
      ...button,
      backendMethods: backendForButton(button, functionBodies, backendImports),
    }));
    const modal = {
      screen: screenLabel(rel),
      file: rel,
      line: lineNumber(source, block.index),
      type: "manual",
      title: titleFromBlock(block.body),
      open: block.condition,
      fields: extractFields(block.body),
      buttons,
    };
    modal.staticStatus = classifyModal(modal);
    modals.push(modal);
  }

  return modals.sort((a, b) => a.line - b.line);
}

const files = [...walk(screensDir), ...walk(componentsDir)];
const allModals = files.flatMap(collectModals);
const screenModals = allModals.filter((modal) => modal.file.startsWith("frontend/src/lib/screens/"));
const componentModals = allModals.filter((modal) => !modal.file.startsWith("frontend/src/lib/screens/"));
const issueLike = allModals.filter((modal) => modal.staticStatus !== "No static issue detected.");
const backendMapped = allModals.filter((modal) => modal.buttons.some((button) => button.backendMethods.length > 0));

const lines = [];
lines.push("# UI Modal Inventory - 2026-04-17");
lines.push("");
lines.push("Purpose: static first-pass inventory of popup/modal surfaces, their fields, actions, and visible backend wiring.");
lines.push("");
lines.push("Generated by `node scripts/modal_inventory.mjs`.");
lines.push("");
lines.push("## Summary");
lines.push("");
lines.push(`- Screen modals detected: ${screenModals.length}`);
lines.push(`- Reusable component modals detected: ${componentModals.length}`);
lines.push(`- Modals with mapped backend action buttons: ${backendMapped.length}`);
lines.push(`- Modals with static review notes: ${issueLike.length}`);
lines.push("");
lines.push("## Static Review Notes");
lines.push("");
if (issueLike.length === 0) {
  lines.push("_No static modal notes found._");
} else {
  lines.push(row(["Screen", "File:Line", "Title", "Open State", "Note"]));
  lines.push(row(["---", "---", "---", "---", "---"]));
  for (const modal of issueLike) {
    lines.push(row([modal.screen, `${modal.file}:${modal.line}`, modal.title || "(untitled)", modal.open, modal.staticStatus]));
  }
}
lines.push("");
lines.push("## Screen Modals");
lines.push("");
for (const modal of screenModals) {
  lines.push(`### ${modal.screen} - ${modal.title || "(untitled)"}`);
  lines.push(`File: \`${modal.file}:${modal.line}\``);
  lines.push(`Type: \`${modal.type}\` | Open: \`${modal.open || "(unknown)"}\``);
  lines.push("");
  lines.push(`Fields detected: ${modal.fields.length ? modal.fields.map((field) => `\`${field}\``).join(", ") : "_none detected_"}`);
  lines.push("");
  if (modal.buttons.length === 0) {
    lines.push("_No footer/body buttons detected._");
  } else {
    lines.push(row(["Button", "Handler", "Backend Method(s)", "State"]));
    lines.push(row(["---", "---", "---", "---"]));
    for (const button of modal.buttons) {
      lines.push(row([
        button.label,
        button.handler || "(none)",
        button.backendMethods.map((method) => `\`${method}\``).join(", "),
        [button.type ? `type ${button.type}` : "", button.disabled ? "disabled-bound/static" : ""].filter(Boolean).join("; "),
      ]));
    }
  }
  lines.push("");
  lines.push(`Static status: ${modal.staticStatus}`);
  lines.push("");
}
lines.push("## Reusable Component Modals");
lines.push("");
for (const modal of componentModals) {
  lines.push(`### ${modal.screen} - ${modal.title || "(untitled)"}`);
  lines.push(`File: \`${modal.file}:${modal.line}\``);
  lines.push(`Fields detected: ${modal.fields.length ? modal.fields.map((field) => `\`${field}\``).join(", ") : "_none detected_"}`);
  lines.push("");
}

fs.mkdirSync(outDir, { recursive: true });
fs.writeFileSync(outMd, `${lines.join("\n")}\n`);
fs.writeFileSync(outJson, `${JSON.stringify({
  generatedAt: new Date().toISOString(),
  screenModalCount: screenModals.length,
  componentModalCount: componentModals.length,
  backendMappedModalCount: backendMapped.length,
  staticReviewNoteCount: issueLike.length,
  modals: allModals,
}, null, 2)}\n`);

console.log(`Wrote ${path.relative(repoRoot, outMd)}`);
console.log(`Wrote ${path.relative(repoRoot, outJson)}`);
console.log(`Screen modals: ${screenModals.length}`);
console.log(`Component modals: ${componentModals.length}`);
console.log(`Backend-mapped modals: ${backendMapped.length}`);
console.log(`Static review notes: ${issueLike.length}`);
