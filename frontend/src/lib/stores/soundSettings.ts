import { writable } from "svelte/store";
import { GetSettings } from "../../../wailsjs/go/main/DocumentsService";

/**
 * Wave 10 / B4 — tiny cache of the one sound-related setting
 * (`sounds.sound_on_paid_enabled`), so `playPaidSettle()` in `lib/sound.ts`
 * can check the opt-out without a full settings-page round trip.
 *
 * Default ON (matches the backend default in
 * app_setup_documents_surface.go GetSettings). SettingsScreen.svelte owns
 * the authoritative read/write via GetSettings/UpdateSettings; this store
 * just hydrates once at app boot so other screens (PaymentsScreen) can read
 * the flag synchronously.
 */
export const soundOnPaidEnabled = writable(true);

export async function initSoundSettings(): Promise<void> {
  try {
    const res: any = await GetSettings();
    const enabled = res?.sounds?.sound_on_paid_enabled;
    if (typeof enabled === "boolean") {
      soundOnPaidEnabled.set(enabled);
    }
  } catch {
    // Keep the default-ON value if settings can't be loaded (e.g. not
    // logged in yet) — the sound opt-out is not security-sensitive.
  }
}

// Self-hydrate on first import. There is no shared app-boot hook in scope
// for this change (App.svelte is out of file scope for Wave 10 / B4), so
// the store loads its own value lazily the first time any screen imports
// it (PaymentsScreen via lib/sound.ts, or SettingsScreen directly) rather
// than requiring a dedicated wiring point elsewhere.
void initSoundSettings();
