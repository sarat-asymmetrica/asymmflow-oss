<script lang="ts">
    import { currentUser } from "$lib/stores/authContext";
    import { setTextScalePreset, textScalePreset, type TextScalePreset } from "$lib/stores/textScale";
    import { LogoutInteractiveSession } from "../../../../wailsjs/go/main/App";

    // Reactive user data
    let userName = $derived($currentUser?.full_name || "User");
    let userInitial = $derived(userName.charAt(0).toUpperCase());

    let loggingOut = $state(false);

    // Wave 6 Mission C.1: visible logout — invalidate the DB session
    // (audit trail: user_logout), then hand control back to App.svelte,
    // which clears auth state and returns to the login screen. The
    // frontend transition happens even if the backend call fails, so a
    // user is never trapped in a session the backend already lost.
    async function handleLogout() {
        if (loggingOut) return;
        loggingOut = true;
        try {
            await LogoutInteractiveSession();
        } catch (error) {
            console.error("Logout failed:", error);
        } finally {
            loggingOut = false;
            window.dispatchEvent(new CustomEvent("app:logout"));
        }
    }

    const textSizeButtons: Array<{ preset: TextScalePreset; label: string; title: string }> = [
        { preset: "standard", label: "A", title: "Standard text" },
        { preset: "comfortable", label: "A+", title: "Comfortable text" },
        { preset: "large", label: "A++", title: "Large text" },
    ];
</script>

<header class="header">
    <!-- Left: spacer -->
    <div class="header-left"></div>

    <!-- Right: Actions -->
    <div class="header-right">
        <div class="text-size-control" aria-label="Text size">
            {#each textSizeButtons as option}
                <button
                    type="button"
                    class:active={$textScalePreset === option.preset}
                    title={option.title}
                    aria-label={option.title}
                    aria-pressed={$textScalePreset === option.preset}
                    onclick={() => setTextScalePreset(option.preset)}
                >
                    {option.label}
                </button>
            {/each}
        </div>

        <!-- User -->
        <div class="user-menu">
            <div class="avatar">{userInitial}</div>
            <span class="user-name">{userName}</span>
        </div>

        <button
            type="button"
            class="logout-button"
            title="Sign out of AsymmFlow"
            disabled={loggingOut}
            onclick={handleLogout}
        >
            {loggingOut ? "Signing out…" : "Sign out"}
        </button>
    </div>
</header>

<style>
    .header {
        height: var(--header-height); /* 56px */
        background: var(--surface);
        border-bottom: 1px solid var(--border);
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 0 var(--page-padding);
    }

    .header-right {
        display: flex;
        align-items: center;
        gap: 12px;
    }

    .text-size-control {
        display: inline-flex;
        align-items: center;
        gap: 2px;
        padding: 2px;
        border: 1px solid var(--border);
        border-radius: 8px;
        background: var(--surface-elevated);
    }

    .text-size-control button {
        min-width: 34px;
        height: 28px;
        border: 0;
        border-radius: 6px;
        background: transparent;
        color: var(--text-secondary);
        font-family: var(--font-display);
        font-size: 12px;
        font-weight: 700;
        cursor: pointer;
    }

    .text-size-control button:nth-child(2) {
        font-size: 13px;
    }

    .text-size-control button:nth-child(3) {
        font-size: 14px;
    }

    .text-size-control button:hover,
    .text-size-control button.active {
        background: var(--onyx, #1d1d1f);
        color: #fff;
    }

    .user-menu {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .avatar {
        width: 32px;
        height: 32px;
        background: var(--onyx, #1D1D1F);
        color: #fff;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 13px;
        font-weight: 600;
    }

    .user-name {
        font-size: 13px;
        font-weight: 500;
        color: var(--text-primary);
    }

    .logout-button {
        height: 32px;
        padding: 0 12px;
        border: 1px solid var(--border);
        border-radius: 8px;
        background: var(--surface-elevated);
        color: var(--text-secondary);
        font-family: var(--font-display);
        font-size: 12px;
        font-weight: 600;
        cursor: pointer;
    }

    .logout-button:hover:not(:disabled) {
        background: var(--onyx, #1d1d1f);
        color: #fff;
    }

    .logout-button:disabled {
        opacity: 0.6;
        cursor: default;
    }
</style>
