# Conflict Resolution System — Future Implementation

**Status:** PLANNED (not yet implemented)
**Priority:** After deployment stabilizes
**Estimated effort:** 1.5-2 days

---

## Problem

When two users on different PCs edit the same record between sync cycles (10 min), the last sync push silently overwrites the other person's changes. No notification, no merge, no admin review.

## Proposed Solution: Admin-Arbitrated Conflict Queue

### Flow

1. During sync push, if Supabase record's version != expected (someone else pushed first), create a `sync_conflicts` row instead of overwriting
2. First writer's version stays live in Supabase (stable state)
3. Second writer's version is queued as "pending admin review"
4. Admin gets a notification badge — opens Conflict Review screen
5. Admin sees side-by-side diff, picks a winner (or merges)
6. Resolution propagates to all devices on next sync
7. Both parties get a toast notification explaining the outcome

### New Tables

```sql
-- Conflict queue
CREATE TABLE sync_conflicts (
    id TEXT PRIMARY KEY,
    table_name TEXT NOT NULL,         -- e.g. "customers"
    record_id TEXT NOT NULL,          -- the conflicting record's ID
    field_name TEXT,                  -- optional per-field tracking
    local_data TEXT NOT NULL,         -- JSON blob of the rejected push
    remote_data TEXT NOT NULL,        -- JSON blob of current Supabase data
    local_device_hash TEXT,
    remote_device_hash TEXT,
    local_user TEXT,                  -- display name "Riley"
    remote_user TEXT,                 -- display name "Jamie"
    status TEXT DEFAULT 'pending',    -- pending / resolved_local / resolved_remote / resolved_merged
    resolved_by TEXT,
    resolved_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Notifications to affected parties
CREATE TABLE conflict_notifications (
    id TEXT PRIMARY KEY,
    conflict_id TEXT NOT NULL,
    device_hash TEXT NOT NULL,
    message TEXT NOT NULL,
    read INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Backend Functions

| Function | Purpose |
|----------|---------|
| `detectConflict()` | Called during sync push — version mismatch creates conflict row |
| `GetPendingConflicts()` | Admin-only — returns pending conflicts with both data versions |
| `ResolveConflict(id, choice)` | Admin picks winner, updates record to v+1, creates notifications |
| `GetMyNotifications()` | Each device polls during sync for unread notifications |

### Frontend

- Conflict Review screen (Admin only) — side-by-side diff with Accept buttons
- Notification badge in sidebar (conflict count)
- Toast on sync when a conflict affecting this user is resolved

### Design Decision: First-Writer Holds

While conflict is pending, the first writer's version stays live in Supabase. The second writer sees their local version with a "pending review" indicator. No record is ever in an invalid state.

### Prerequisites

Before implementing:
1. All update functions must increment `Version` (currently only 2 of 15+ do)
2. Switch background sync from `db_manager.go` (blind upsert) to `sync_service_impl.go` (version-aware)
3. Add `WHERE version = ?` optimistic locking to update functions

---

## Architecture Diagram

```
  PC 1 (Jamie)                    Supabase                     PC 2 (Riley)
  ───────────                    ────────                     ───────────
  Edit NPC                                                  Edit NPC
  phone -> "111"                                              phone -> "222"
       |                                                           |
       v sync push                                                 |
  Push "111" ──────────────> Supabase: "111" (v2)                  |
                                                                   v sync push
                              <──────────────── Push "222" (conflict!)
                              Supabase has v2 from different device
                                    |
                                    v
                           sync_conflicts table
                           status: pending
                                    |
                              Admin reviews
                              picks "222"
                                    |
                                    v
                           Supabase: "222" (v3)
                           conflict: resolved
                                    |
       Next sync pulls v3 ──────────┼──────────── Conflict cleared
       toast: "resolved"                          toast: "approved"
```
