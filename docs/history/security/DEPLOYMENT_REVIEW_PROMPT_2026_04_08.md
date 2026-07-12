# Deployment Review Prompt

Use the following prompt for the next review pass. It is designed to use both:

- [SESSION_NOTE_2026_04_08.md](/Users/developer/House_of_Projects/ph_holdings/docs/SESSION_NOTE_2026_04_08.md)
- [DEPLOYMENT_RED_TEAM_CHECKLIST_2026_04_08.md](/Users/developer/House_of_Projects/ph_holdings/docs/DEPLOYMENT_RED_TEAM_CHECKLIST_2026_04_08.md)

---

## Prompt

You are conducting a full deployment-readiness review of the AsymmFlow / Acme Instrumentation ERP application.

Before you start, read these two documents completely and use them as the operating context for your review:

1. `/Users/developer/House_of_Projects/ph_holdings/docs/SESSION_NOTE_2026_04_08.md`
2. `/Users/developer/House_of_Projects/ph_holdings/docs/DEPLOYMENT_RED_TEAM_CHECKLIST_2026_04_08.md`

Your task is to execute the deployment review using those documents together, not in isolation.

### Objectives

1. Verify that the changes captured in the session note are actually working in the application.
2. Execute the deployment checklist end to end, including happy-path and adversarial tests.
3. Perform a red-team pass across all key workflows to find:
   - broken buttons
   - silent failures
   - navigation dead ends
   - RBAC leaks
   - missing child records
   - schema/workflow mismatches
   - duplicate or placeholder data surfacing as real operational records
4. Review whether the database schema fully supports:
   - all workflows
   - all visible screens
   - all modal actions
   - all data-entry points
   - all downstream state transitions and document linkages

### Required Coverage

You must review and test, at minimum:

- Opportunities
- Costing
- Offers
- Customer Orders
- Operations flows
- Customer Invoices
- Payments Received
- Payments Made
- Expenses
- Payroll
- Work
- People
- Notifications
- Relationships / CRM detail views
- Deployment page
- License / startup / database path behavior

### Red-Team Expectations

Do not stop at happy-path testing.

You must intentionally try to break the system by testing:

- repeated clicks on actions
- partial data entry
- missing optional/required combinations
- duplicate-looking data
- long values and edge-case values
- stale records and imported records
- role-based access boundaries
- modal open/close/reopen flows
- downstream document creation after upstream edits
- app reopen / relaunch consistency

### Output Format

Produce the review output in this structure:

1. **Findings**
   - List concrete issues first, ordered by severity.
   - Include file/screen/workflow references where possible.
   - Distinguish between:
     - UI wiring issue
     - backend logic issue
     - data issue
     - schema gap
     - RBAC issue

2. **Checklist Status**
   - For each major checklist area, mark:
     - Passed
     - Failed
     - Partial
     - Not Tested

3. **Red-Team Results**
   - Summarize adversarial tests performed.
   - Note what broke, what held, and what still feels risky.

4. **Schema Coverage Review**
   - Explicitly state whether the schema appears to cover:
     - all workflows
     - all user inputs
     - all downstream side effects
   - Call out any missing tables, columns, or relationships.

5. **Release Recommendation**
   - One of:
     - Ready to deploy
     - Deploy with caution
     - Not ready to deploy
   - Include a short rationale.

### Important Rules

- Be skeptical.
- Verify instead of assuming.
- Treat silent failures as high-risk.
- Treat visible but hollow records as real bugs.
- Treat schema gaps and missing persistence as deployment blockers.
- If a flow appears to work in UI but does not persist correctly in DB, mark it as failed.
- If a record appears complete in DB but is unusable in UI, mark it as failed.

### Final Instruction

Use the session note as the “what changed” source of truth.
Use the deployment checklist as the “what must be verified” source of truth.
Your review should combine both into one coherent deployment-readiness assessment.

