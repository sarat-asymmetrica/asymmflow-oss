/* Auth bridge — the license gate (the ONE live auth path per the K5 owner
 * ruling: license-only, no device-registration). `validateLicense()` runs at
 * boot; `activateLicense(key)` runs from the LicenseActivation screen. Real
 * adapters call the Wails bindings; under mock (the lab), validate returns a
 * synthetic admin so the app boots straight in, and activate accepts the demo
 * key format — so the whole gate is demoable without a backend. */

import { ValidateLicense, ActivateLicense } from '$wails/go/main/InfraService'
import { pick } from './runtime'

export interface AuthResult {
  ok: boolean
  message: string
  role: string
  displayName: string
  permissions: string[]
  deviceHash: string
}

/** Owner-visible key shape: PH-<ROLE>-<6 alnum> (ADM/MGR/SLS/OPS/STF/DEV). */
export const LICENSE_KEY_PATTERN = /^PH-(ADM|MGR|SLS|OPS|STF|DEV)-[A-Z0-9]{6}$/

async function realValidate(): Promise<AuthResult> {
  const r = await ValidateLicense()
  return {
    ok: r.valid,
    message: r.valid ? '' : 'No valid license on this device.',
    role: r.role,
    displayName: r.display_name,
    permissions: r.permissions ?? [],
    deviceHash: r.key ?? '',
  }
}

async function mockValidate(): Promise<AuthResult> {
  // Lab: a synthetic admin so the app boots straight in (mock never gates).
  return { ok: true, message: '', role: 'Administrator', displayName: 'Lab User', permissions: ['*'], deviceHash: 'lab-device' }
}

export const validateLicense = (): Promise<AuthResult> => pick(realValidate, mockValidate)()

async function realActivate(key: string): Promise<AuthResult> {
  const r = await ActivateLicense(key)
  return {
    ok: r.success,
    message: r.message,
    role: r.role,
    displayName: r.display_name,
    permissions: r.permissions ?? [],
    deviceHash: r.device_hash,
  }
}

async function mockActivate(key: string): Promise<AuthResult> {
  if (!LICENSE_KEY_PATTERN.test(key)) {
    return { ok: false, message: 'Invalid key format — expected PH-XXX-YYYYYY.', role: '', displayName: '', permissions: [], deviceHash: '' }
  }
  return { ok: true, message: 'Activated (mock).', role: 'Administrator', displayName: 'Lab User', permissions: ['*'], deviceHash: 'lab-device' }
}

export const activateLicense = (key: string): Promise<AuthResult> => pick(realActivate, mockActivate)(key)
