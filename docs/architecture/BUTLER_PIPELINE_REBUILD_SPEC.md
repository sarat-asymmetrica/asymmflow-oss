# Butler Pipeline Rebuild Spec (AsymmFlow)

> **Superseded (Wave 13, 2026-07-22):** the AIML/Grok primary-backend references below describe
> the pre-Wave-13 pipeline. As of Wave 13 "Perception & Print," AIMLAPI/Grok has been removed
> entirely — Butler chat is Mistral-direct only (`mistral-large-latest`, with
> `mistral-small-latest` for simple queries), via the single `getMistralAPIKey()` resolver.
> There is no AIML/Grok fallback path anymore. Treat this document's AIML/Grok sections as
> historical context for the decision contracts they describe, not as current backend wiring.

## 1. Purpose

This document is a complete technical handoff for recreating the Butler assistant pipeline used in AsymmFlow ERP.

It is written for another AI engineer/agent to:
- Rebuild the pipeline from scratch
- Preserve the current behavior and decision contracts
- Avoid prior failure modes (hallucination, weak routing, broken persistence, non-executable actions)

---

## 2. Product Goals

Butler is a business copilot embedded in ERP chat that must:
- Answer grounded business questions from live app data
- Produce structured, executable actions (create/update/approve/reject/open/fetch/analyze)
- Respect permission boundaries for finance-sensitive queries
- Persist chat history and action metadata for auditability
- Continue multi-turn workflows reliably (pronouns, follow-ups, partial details)

Non-goals:
- Free-form creative chatbot behavior without data grounding
- Unverified claims outside available records

---

## 3. High-Level Architecture

### 3.1 Runtime Stack
- Frontend: Svelte (`ButlerScreen.svelte`)
- Backend: Go (Wails-bound methods)
- DB: SQLite primary (optionally synced to Supabase)
- LLM backends:
  - Primary: AIML API route (Grok model id from config)
  - Fallback: Mistral
- Optional deterministic answer path:
  - Grounded SQL fast-path for specific query types

### 3.2 Core Flow
1. User sends message in Butler UI
2. Frontend calls `ChatWithButlerPersistent(conversationId, message)`
3. Backend:
   - validates chat permission
   - creates/loads conversation
   - saves user message
   - classifies intent
   - applies finance gate where required
   - attempts grounded fast-path (if matched)
   - else builds context + calls LLM
   - parses actions + metadata
   - saves assistant message (+ action metadata)
4. Frontend renders response and action chips
5. Action chips map to executable app APIs

---

## 4. Source Files (Current Canonical)

Backend:
- `chat_service.go`  
  Persistent pipeline, conversation CRUD, permission gate, metadata persistence, action parsing/contract behaviors.
- `butler_ai.go`  
  Intent classification, context building, model call/fallback, metadata shaping, report logic, entity resolution.
- `butler_grounded_fastpath.go`  
  Deterministic SQL response paths and quote-draft continuation fast-paths.
- `database.go`  
  `Conversation` and `ChatMessage` models.

Frontend:
- `frontend/src/lib/screens/ButlerScreen.svelte`  
  Chat UI, conversation list, persistence calls, action execution routing, local fail-safe hiding.

---

## 5. Data Model Contracts

### 5.1 Conversations
Table: `conversations`
- `id`
- `title`
- `summary`
- `is_active`
- `last_msg_at`
- base fields (`created_at`, `updated_at`, soft delete fields)

### 5.2 Chat Messages
Table: `chat_messages`
- `id`
- `conversation_id`
- `role` (`user|assistant|system`)
- `content`
- `tokens_used`
- action metadata columns:
  - `message_type`
  - `action_type`
  - `action_target`
  - `action_label`
  - `action_data`
  - `action_status`
  - `action_metadata` (JSON string)

Action metadata is required for replayability and auditing.

---

## 6. Wails API Surface (Butler-Relevant)

Required backend exports:
- `ChatWithButlerPersistent(conversationID, message) -> ChatResponse`
- `ChatWithButler(message) -> ButlerResponse` (fallback path)
- `ListConversations() -> []Conversation`
- `GetConversationMessages(conversationID) -> []ChatMessage`
- `DeleteConversation(conversationID) -> error`
- `PurgeAllConversations() -> error`

Frontend must import and use these from generated Wails bindings.

---

## 7. Routing and Decision Tree

## 7.1 Primary Chat Route
- Always try persistent route first.
- If persistent route fails, retry persistent with blank conversation id.
- Then fallback to non-persistent route.
- If all fail, surface concrete error text (do not hide behind generic apology).

## 7.2 Permission Gate
- Use effective permission check:
  - `requirePermission("finance:view") == nil`
- Gate only finance-sensitive asks.
- Do not over-gate sales/operations prompts (e.g. “what sold in last 2 quarters”).

## 7.3 Grounded Fast-Path Priority
Before LLM:
1. Offer-draft continuation fast-path (if quantity/price follow-up detected)
2. Customer grounded fast-path (invoices/offers/line items/revenue projections)
3. Otherwise LLM path

Grounded path must return direct SQL-grounded facts and can attach executable actions.

---

## 8. Intent and Entity Resolution

### 8.1 Entity Resolution
Use layered resolver:
1. Exact customer match by `business_name`, `short_code`, `customer_code`, `customer_id`
2. Fuzzy wildcard match
3. Special grouped scopes:
   - `NPC` group
   - `GSC` group
   - `Riverside Power` group

Grouped scope means multiple `customer_id`s mapped into one logical customer in-query.

### 8.2 Multi-Turn Hints
When current turn lacks explicit entity:
- infer customer hint from recent user turns (`NPC`, `GSC`, `Riverside Power`)
- infer product hint from prior quote intent
- support pronouns (`them`, `they`, `those`)
- support reconciliation phrases (`all three are same`)

---

## 9. Grounded Capabilities (Current)

Implemented deterministic routes:
- Invoice snapshots per customer/group
- Offers snapshots per customer/group
- “this quarter” invoice window
- “this year” offers window
- Sold line-items rollup from `invoice_items`
- Payment received timelines from `payments`
- Revenue projection summary from historical invoice totals + active offer pipeline
- Offer draft preparation from natural language quantity/price input

Natural language parser supports variations:
- quantity: `is`, `=`, `:`, `would be`, `to be`
- price: `price`, `price per unit`, `unit price`, plus `BHD`

---

## 10. Action Contract

Actions are JSON objects with normalized fields:
- `type`
- `target`
- `label`
- `data` (payload)

Common action types:
- `create_offer_draft`
- `create_offer`
- `create_order`
- `create_followup`
- `create_stock_adjustment`
- `approve`
- `reject`
- `update`
- `navigate`
- `open`
- `fetch`
- `analyze`
- `daily_briefing`

Mandatory data by type (examples):
- `approve/reject`: `entity_id`
- `update`: `entity_id` + `status/stage`
- `create_offer_draft`: customer + line items + amount

Frontend must block execution when action state is `needs_input`/`invalid_payload`.

---

## 11. Frontend Behavior Contract

## 11.1 Message Rendering
- Support schema variations for historical payloads:
  - `content|Content|message|Message|text|Text`
  - `id|ID`
  - `role|Role`
- Render markdown tables cleanly (avoid broken one-line table outputs)
- For empty text with actions, show placeholder message

## 11.2 Conversation List
- Support `id/ID`, `title/Title`
- Keep active conversation highlight synced

## 11.3 Deletion UX
- Per-chat delete
- Clear-all delete
- If backend delete fails, hide locally using persisted hidden set (`localStorage`) so user is never blocked by stale UI artifacts

---

## 12. LLM Integration Contract

### 12.1 Backend Selection
- Try AIML key/model first
- On AIML failure, fallback to Mistral with explicit fallback reason
- Persist backend usage metadata (`used_backend`, `requested_model`, `used_model`, `fallback_reason`)

### 12.2 Prompting Strategy
- Include context, regime signals, memory snippets, and action contract prompt section
- Enforce:
  - no hallucinated facts
  - explicit uncertainty when data absent
  - structured actions only when executable

### 12.3 Known Pitfall
- AIML “model not found” 404 has occurred historically
- Must degrade gracefully and continue via fallback without user-facing crash

---

## 13. Security and Permissions

- Chat permission: `intelligence:chat`
- Finance permission: `finance:view`
- Never expose credentials in chat/log output
- Keep structured logs sanitized
- Ensure action execution routes enforce their own domain permissions

---

## 14. Persistence and Auditability

- Persist all user/assistant turns in `chat_messages`
- Persist action metadata JSON for every assistant response
- Update `conversations.last_msg_at` after each assistant save
- Conversation/history retrieval must be chronological

---

## 15. Failure Modes and Fixes (Important Lessons)

1. **Fallback route inconsistency**
- Problem: persistent failure fell to stricter/non-equivalent path
- Fix: align gate logic and return explicit errors

2. **Entity drift in multi-turn**
- Problem: “them” lost customer context
- Fix: infer hints from prior user turns + grouped scopes

3. **Overly narrow parser**
- Problem: phrasing variants not captured
- Fix: robust regex for quantity/price and product extraction

4. **Undeletable chats**
- Problem: legacy row shapes / id mismatch / backend mismatch
- Fix: unscoped deletes + local-hide fail-safe + clear-all

5. **Empty historical chat bubbles**
- Problem: content field shape mismatch
- Fix: multi-key hydration and placeholder strategy

---

## 16. Rebuild Checklist (For Another AI)

1. Implement DB models (`Conversation`, `ChatMessage`) and migrations.
2. Build persistent chat APIs (`create/list/get/delete/purge`).
3. Implement intent classifier and finance gate.
4. Implement context builders (customer/supplier/finance/ops/risk + memory).
5. Wire model backend selection (AIML primary, Mistral fallback).
6. Add action parsing + metadata persistence.
7. Add grounded SQL fast-path module for key business intents.
8. Implement frontend chat with:
   - persistence route
   - robust hydration
   - action chips + executor routes
   - delete + clear-all + local-hide fail-safe
9. Add grouped customer resolution (`NPC`, `GSC`, `Riverside Power`).
10. Add multi-turn hint carryover (customer/product/pronoun).
11. Add offer-draft continuation from quantity/price follow-ups.
12. Validate key scenarios manually in app.

---

## 17. Minimum Acceptance Scenarios

1. “What equipment sold to Riverside Power in last 2 quarters?”  
Expected: grounded answer; not blocked by finance gate.

2. “NPC invoices/offers + handlers + line items + payments received”  
Expected: grouped customer data with concrete records.

3. “Create quote to NPC for Probe FMP51, quantity would be 4, unit price 108 BHD”  
Expected: action-ready offer draft payload.

4. Follow-up: “What about their line items?”  
Expected: entity carried from prior turn.

5. Chat persistence:
- messages survive restart
- old chats render
- delete one works
- clear all works

---

## 18. Recommended Next Hardening

- Add explicit DB path indicator in Butler UI for environment mismatch diagnosis.
- Add telemetry counters:
  - grounded-hit rate
  - fallback frequency
  - action execution success rate
- Add lightweight integration tests around:
  - fast-path matching
  - permission gating
  - conversation delete/purge
  - multi-turn hint carryover

---

## 19. Handoff Note

This pipeline is intentionally hybrid:
- deterministic SQL for critical business facts and workflow actions
- LLM for narrative synthesis and broad analysis

Do not move all logic into LLM prompting.  
Maintain deterministic fast-paths for reliability, especially for:
- customer/account-specific financial/operational answers
- offer/action generation flows
- multi-turn continuation where user gives partial details.

