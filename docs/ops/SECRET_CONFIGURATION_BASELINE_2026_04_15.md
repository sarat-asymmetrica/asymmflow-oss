# Secret Configuration Baseline - 2026-04-15

This document records the current secret/configuration situation before any repo-hygiene or secret-loading refactor work.

Purpose:
- preserve the exact secret-loading model currently used by the app
- record the active source locations and precedence rules
- provide a safe tracked reference without re-committing plaintext secrets

Companion local snapshot:
- Exact plaintext values are stored locally in `.agent/secret-baselines/SECRET_CONFIGURATION_SNAPSHOT_2026_04_15.md`
- That file is intentionally ignored by git and should remain local-only

## 1. Current Runtime Secret Sources

### 1.1 Environment file load order

The app already loads runtime config from `.env` files instead of requiring hardcoded values in source.

Primary loader:
- [config.go](/Users/developer/House_of_Projects/ph_holdings/config.go:280)

Observed load order:
1. executable-adjacent `.env`
2. current working directory `.env`
3. `~/.local/share/AsymmFlow/.env`

Important implication:
- Removing secrets from git is not the same as removing secrets from runtime
- The app still requires secrets to exist in one of these runtime locations when that integration is enabled

### 1.2 Database path behavior

Primary resolver:
- [config.go](/Users/developer/House_of_Projects/ph_holdings/config.go:213)
- [app.go](/Users/developer/House_of_Projects/ph_holdings/app.go:332)

Observed behavior:
1. `PH_DB_PATH`
2. `DATABASE_PATH`
3. existing machine-level app-data DB
4. packaged DB near executable or app bundle
5. executable search directories
6. local `./ph_holdings.db`
7. new app-data DB fallback

Important implication:
- packaged deployments can work without a repo-root DB, as long as the packaged `.env` and bundled DB stay aligned

### 1.3 Supabase/cloud-sync loading

Primary loader:
- [config.go](/Users/developer/House_of_Projects/ph_holdings/config.go:639)

Observed precedence:
1. `DATABASE_URL`
2. individual `SUPABASE_*` variables

Observed enable rule:
- cloud sync is enabled only when `ENABLE_CLOUD_SYNC=true` and required DB credentials are present

### 1.4 AI key precedence

Mistral:
- provider registration: [app.go](/Users/developer/House_of_Projects/ph_holdings/app.go:817)
- env/hardcoded fallback: [butler_ai.go](/Users/developer/House_of_Projects/ph_holdings/butler_ai.go:6449)

Observed precedence:
1. encrypted settings DB `apiKeys.mistral_key`
2. `settings.json` value `apiKeys.mistral_key`
3. env `MISTRAL_API_KEY`
4. hardcoded fallback in source

AIML / Butler primary backend:
- provider registration: [app.go](/Users/developer/House_of_Projects/ph_holdings/app.go:848)
- env fallback: [butler_ai.go](/Users/developer/House_of_Projects/ph_holdings/butler_ai.go:5608)

Observed precedence:
1. encrypted settings DB `apiKeys.aimlapi_key`
2. `settings.json` value `apiKeys.aimlapi_key`
3. env `ASYMM_AIML_API_KEY`
4. env `AIML_API_KEY`

OCR AIML fallback:
- [app.go](/Users/developer/House_of_Projects/ph_holdings/app.go:13553)

Observed behavior:
- OCR engine reads `AIMLAPI_KEY` without underscore

Important implication:
- current codebase contains a naming mismatch between `AIML_API_KEY` and `AIMLAPI_KEY`
- Butler chat can succeed while OCR fallback still fails, depending on which variable is set

## 2. Observed State On 2026-04-15

### 2.1 Repo root `.env`

Observed state:
- repo-root `.env` is currently absent on disk
- git still knows about `.env` historically, but the live file is missing in this workspace

### 2.2 Packaged deployment env

Observed file:
- `deploy_package/.env`

Observed characteristics:
- exists and is currently the most deployment-relevant env file in the repo workspace
- last modified: `Apr 14 23:49:24 2026`
- SHA-256: `<redacted>`
- contains active cloud DB credentials (values redacted)
- contains `ENABLE_CLOUD_SYNC=true`
- does not pin the runtime DB; packaged `data/ph_holdings.db` is now a first-run seed
- does not currently provide an active `MISTRAL_API_KEY`

### 2.3 Machine-level app-data env

Observed file:
- `~/.local/share/AsymmFlow/.env`

Observed characteristics:
- exists
- last modified: `Mar 18 10:46:11 2026`
- SHA-256: `<redacted>`
- contains a machine-level `DATABASE_PATH`
- contains `ENABLE_CLOUD_SYNC=false`
- contains an AIML key variable
- does not match the packaged deployment env

### 2.4 Settings DB and user settings file

Observed state in repo-root `ph_holdings.db`:
- `settings` table exists
- `SELECT COUNT(*) FROM settings WHERE key LIKE 'apiKeys.%'` returned `0`

Observed state for user settings file:
- `~/.local/share/AsymmFlow/settings.json` is currently absent

Important implication:
- current active secret behavior is mostly env-driven, not DB-driven

## 3. Embedded / Hardcoded Secret-Like Values

### 3.1 Developer master license key

Source:
- [license_service.go](/Users/developer/House_of_Projects/ph_holdings/license_service.go:98)

Observed behavior:
- a reusable developer master key exists in source (value redacted)
- it is gated by `ENABLE_DEVELOPER_MASTER_KEY`

### 3.2 Hardcoded Mistral fallback

Source:
- [butler_ai.go](/Users/developer/House_of_Projects/ph_holdings/butler_ai.go:6461)

Observed behavior:
- if no valid Mistral key is found via settings DB, `settings.json`, or env, Butler falls back to a hardcoded key in source

Important implication:
- Mistral can continue to work even when env configuration appears incomplete
- changing secret-loading logic without accounting for this fallback can create confusing regressions

## 4. Packaging-Specific Notes

Primary packager logic:
- [manual_deployment_package_test.go](/Users/developer/House_of_Projects/ph_holdings/manual_deployment_package_test.go:518)

Observed behavior:
- deployment packaging reads a source env file
- strips existing DB path overrides
- leaves packaged `data/ph_holdings.db` as a first-run seed
- writes a packaged `.env` into the output bundle without forcing the live DB path

Important implication:
- the deployment package is already designed to rely on packaged runtime config, not on hardcoded source secrets

## 5. Recovery Guidance

If a future secret/config cleanup causes regressions:
1. Restore the exact values from `.agent/secret-baselines/SECRET_CONFIGURATION_SNAPSHOT_2026_04_15.md`
2. Restore the packaged env behavior first:
   - packaged `.env`
   - no forced `DATABASE_PATH` / `PH_DB_PATH`
   - bundled seed `data/ph_holdings.db`
3. Restore the machine-level env only if reproducing the previous local-machine behavior is necessary
4. Verify there are still no unexpected `apiKeys.*` rows in the active DB and no surprise `settings.json` overrides
5. Re-test:
   - cloud sync
   - Butler chat
   - OCR fallback
   - packaged-app first run

## 6. Change-Risk Flags To Remember

- The current app relies on runtime env files, not hardcoded config, for Supabase.
- Packaged and machine-level envs currently disagree on cloud-sync state.
- Mistral currently has a hardcoded fallback in source.
- AIML naming is inconsistent across subsystems:
  - `ASYMM_AIML_API_KEY`
  - `AIML_API_KEY`
  - `AIMLAPI_KEY`
- A future cleanup must preserve behavior before removing any fallback or moving any env source.
