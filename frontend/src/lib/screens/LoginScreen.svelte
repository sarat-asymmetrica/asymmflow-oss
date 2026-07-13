<script lang="ts">
    import { preventDefault } from 'svelte/legacy';

    import { createEventDispatcher } from "svelte";
    import { LoginDevice } from "../../../wailsjs/go/main/App";
    import { brand } from "$lib/brand";

    const dispatch = createEventDispatcher();

    let username = $state("");
    let password = $state("");
    let error = $state("");
    let loading = $state(false);

    // P1-4 FIX: Password visibility toggle
    let showPassword = $state(false);

    // P2-6 FIX: Focus management
    let usernameInput: HTMLInputElement = $state();

    // P0-8 FIX: Rate limiting to prevent brute force attacks
    let loginAttempts = 0;
    let lastAttemptTime = 0;
    let lockedUntil = 0;
    const MAX_ATTEMPTS = 5;
    const LOCKOUT_DURATION = 60000; // 1 minute in milliseconds
    const ATTEMPT_WINDOW = 30000; // Reset counter if no attempts in 30 seconds

    function isRateLimited(): boolean {
        const now = Date.now();
        if (lockedUntil > now) {
            const remaining = Math.ceil((lockedUntil - now) / 1000);
            error = `Too many attempts. Please wait ${remaining} seconds.`;
            return true;
        }
        return false;
    }

    function recordAttempt() {
        const now = Date.now();

        // Reset counter if enough time has passed
        if (now - lastAttemptTime > ATTEMPT_WINDOW) {
            loginAttempts = 0;
        }

        loginAttempts++;
        lastAttemptTime = now;

        // Lock out if too many attempts
        if (loginAttempts >= MAX_ATTEMPTS) {
            lockedUntil = now + LOCKOUT_DURATION;
            loginAttempts = 0; // Reset for next lockout period
        }
    }

    import { onMount } from "svelte";
    onMount(() => {
        usernameInput?.focus();
    });

    async function handleLogin() {
        error = "";

        // P0-8 FIX: Check rate limiting before attempting login
        if (isRateLimited()) {
            return;
        }

        if (!username || !password) {
            error = "Please enter username and password";
            return;
        }

        loading = true;

        try {
            const result = await LoginDevice(username, password);
            // Reset rate limiting on successful login
            loginAttempts = 0;
            lockedUntil = 0;
            dispatch("login-success", result);
        } catch (err: any) {
            error = err.message || "Login failed";
            // Record failed attempt for rate limiting
            recordAttempt();
        } finally {
            loading = false;
        }
    }
</script>

<div class="login-container">
    <div class="login-card">
        <div class="login-header">
            <div class="logo">
                <div class="logo-mark" style="background: {brand.accentVar}">{brand.mark}</div>
                <span class="brand">{brand.wordmark}</span>
            </div>
            <h1>Sign In</h1>
            <p class="subtitle">Enter your credentials to continue</p>
        </div>

        <form onsubmit={preventDefault(handleLogin)} class="login-form">
            {#if error}
                <div class="error-message">{error}</div>
            {/if}

            <div class="form-group">
                <label for="username">Username</label>
                <input
                    id="username"
                    type="text"
                    bind:value={username}
                    bind:this={usernameInput}
                    placeholder="Enter username"
                    autocomplete="username"
                />
            </div>

            <div class="form-group">
                <label for="password">Password</label>
                <div class="password-input">
                    {#if showPassword}
                        <input
                            id="password"
                            type="text"
                            bind:value={password}
                            placeholder="Enter password"
                            autocomplete="current-password"
                        />
                    {:else}
                        <input
                            id="password"
                            type="password"
                            bind:value={password}
                            placeholder="Enter password"
                            autocomplete="current-password"
                        />
                    {/if}
                    <button
                        type="button"
                        class="toggle-password"
                        onclick={() => showPassword = !showPassword}
                        aria-label={showPassword ? "Hide password" : "Show password"}
                    >
                        {#if showPassword}
                            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/>
                                <line x1="1" y1="1" x2="23" y2="23"/>
                            </svg>
                        {:else}
                            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                                <circle cx="12" cy="12" r="3"/>
                            </svg>
                        {/if}
                    </button>
                </div>
            </div>

            <button type="submit" class="btn-primary" disabled={loading}>
                {#if loading}
                    Signing in...
                {:else}
                    Sign In
                {/if}
            </button>
        </form>
    </div>
</div>

<style>
    .login-container {
        min-height: 100vh;
        display: flex;
        align-items: center;
        justify-content: center;
        background: var(--bg-base, #f5f5f7);
        padding: 20px;
    }

    .login-card {
        background: var(--surface, #fff);
        border-radius: 16px;
        padding: 48px;
        max-width: 400px;
        width: 100%;
        box-shadow: 0 4px 24px rgba(0, 0, 0, 0.08);
    }

    .login-header {
        text-align: center;
        margin-bottom: 32px;
    }

    .logo {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 12px;
        margin-bottom: 24px;
    }

    .logo-mark {
        width: 48px;
        height: 48px;
        background: var(--carbon, #000);
        color: white;
        border-radius: 12px;
        display: flex;
        align-items: center;
        justify-content: center;
        font-weight: 700;
        font-size: 18px;
    }

    .brand {
        font-size: 24px;
        font-weight: 600;
        color: var(--text-primary, #1d1d1f);
    }

    h1 {
        font-size: 24px;
        font-weight: 700;
        color: var(--text-primary, #1d1d1f);
        margin: 0 0 8px;
    }

    .subtitle {
        color: var(--text-secondary, #86868b);
        font-size: 14px;
        margin: 0;
    }

    .login-form {
        display: flex;
        flex-direction: column;
        gap: 20px;
    }

    .form-group {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    label {
        font-size: 13px;
        font-weight: 500;
        color: var(--text-secondary, #86868b);
    }

    input {
        padding: 12px 16px;
        border: 1px solid var(--border, #e5e5e5);
        border-radius: 8px;
        font-size: 15px;
        transition: border-color 0.2s;
    }

    input:focus {
        outline: none;
        border-color: var(--carbon, #000);
    }

    .btn-primary {
        padding: 14px 24px;
        background: var(--carbon, #000);
        color: white;
        border: none;
        border-radius: 8px;
        font-size: 15px;
        font-weight: 600;
        cursor: pointer;
        transition: opacity 0.2s;
        margin-top: 8px;
    }

    .btn-primary:hover:not(:disabled) {
        opacity: 0.9;
    }

    .btn-primary:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .error-message {
        background: #fef2f2;
        color: #dc2626;
        padding: 12px 16px;
        border-radius: 8px;
        font-size: 14px;
    }

    /* P1-4 FIX: Password visibility toggle */
    .password-input {
        position: relative;
    }

    .password-input input {
        padding-right: 48px;
    }

    .toggle-password {
        position: absolute;
        right: 12px;
        top: 50%;
        transform: translateY(-50%);
        background: none;
        border: none;
        padding: 4px;
        cursor: pointer;
        color: var(--text-secondary, #86868b);
        display: flex;
        align-items: center;
        justify-content: center;
        transition: color 0.2s;
    }

    .toggle-password:hover {
        color: var(--onyx, #1d1d1f);
    }

    .toggle-password:focus {
        outline: 2px solid var(--carbon, #000);
        outline-offset: 2px;
        border-radius: 4px;
    }
</style>
