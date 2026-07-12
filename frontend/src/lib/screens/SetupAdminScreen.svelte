<script lang="ts">
    import { preventDefault } from 'svelte/legacy';

    import { createEventDispatcher } from "svelte";
    import { SetupAdminAccount } from "../../../wailsjs/go/main/App";

    const dispatch = createEventDispatcher();

    let username = $state("");
    let password = $state("");
    let confirmPassword = $state("");
    let fullName = $state("");
    let email = $state("");
    let error = $state("");
    let loading = $state(false);

    // P1-4 FIX: Password visibility toggle
    let showPassword = $state(false);
    let showConfirmPassword = $state(false);

    // P2-6 FIX: Focus management
    let fullNameInput: HTMLInputElement = $state();

    import { onMount } from "svelte";
    onMount(() => {
        fullNameInput?.focus();
    });

    async function handleSubmit() {
        error = "";

        // Validation
        if (!username || !password || !fullName) {
            error = "Please fill in all required fields";
            return;
        }

        // P2-1 FIX: Email validation
        if (email) {
            const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
            if (!emailRegex.test(email)) {
                error = "Please enter a valid email address";
                return;
            }
        }

        if (password.length < 8) {
            error = "Password must be at least 8 characters";
            return;
        }

        if (password !== confirmPassword) {
            error = "Passwords do not match";
            return;
        }

        loading = true;

        try {
            await SetupAdminAccount(username, password, fullName, email);
            dispatch("setup-complete");
        } catch (err: any) {
            error = err.message || "Failed to create admin account";
        } finally {
            loading = false;
        }
    }
</script>

<div class="setup-container">
    <div class="setup-card">
        <div class="setup-header">
            <div class="logo">
                <div class="logo-mark">PH</div>
                <span class="brand">Holdings</span>
            </div>
            <h1>Welcome</h1>
            <p class="subtitle">Set up your administrator account to get started</p>
        </div>

        <form onsubmit={preventDefault(handleSubmit)} class="setup-form">
            {#if error}
                <div class="error-message">{error}</div>
            {/if}

            <div class="form-group">
                <label for="fullName">Full Name *</label>
                <input
                    id="fullName"
                    type="text"
                    bind:value={fullName}
                    bind:this={fullNameInput}
                    placeholder="e.g., Stanislaus Vaz"
                    required
                    autocomplete="name"
                />
            </div>

            <div class="form-group">
                <label for="email">Email</label>
                <input
                    id="email"
                    type="email"
                    bind:value={email}
                    placeholder="e.g., admin@phtrading.com"
                />
            </div>

            <div class="form-group">
                <label for="username">Username *</label>
                <input
                    id="username"
                    type="text"
                    bind:value={username}
                    placeholder="e.g., admin"
                    required
                />
            </div>

            <div class="form-row">
                <div class="form-group">
                    <label for="password">Password *</label>
                    <div class="password-input">
                        {#if showPassword}
                            <input
                                id="password"
                                type="text"
                                bind:value={password}
                                placeholder="Min 8 characters"
                                required
                                autocomplete="new-password"
                            />
                        {:else}
                            <input
                                id="password"
                                type="password"
                                bind:value={password}
                                placeholder="Min 8 characters"
                                required
                                autocomplete="new-password"
                            />
                        {/if}
                        <button
                            type="button"
                            class="toggle-password"
                            onclick={() => showPassword = !showPassword}
                            aria-label={showPassword ? "Hide password" : "Show password"}
                        >
                            {#if showPassword}
                                <!-- Eye Off Icon -->
                                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/>
                                    <line x1="1" y1="1" x2="23" y2="23"/>
                                </svg>
                            {:else}
                                <!-- Eye Icon -->
                                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                                    <circle cx="12" cy="12" r="3"/>
                                </svg>
                            {/if}
                        </button>
                    </div>
                </div>

                <div class="form-group">
                    <label for="confirmPassword">Confirm Password *</label>
                    <div class="password-input">
                        {#if showConfirmPassword}
                            <input
                                id="confirmPassword"
                                type="text"
                                bind:value={confirmPassword}
                                placeholder="Confirm password"
                                required
                                autocomplete="new-password"
                            />
                        {:else}
                            <input
                                id="confirmPassword"
                                type="password"
                                bind:value={confirmPassword}
                                placeholder="Confirm password"
                                required
                                autocomplete="new-password"
                            />
                        {/if}
                        <button
                            type="button"
                            class="toggle-password"
                            onclick={() => showConfirmPassword = !showConfirmPassword}
                            aria-label={showConfirmPassword ? "Hide password" : "Show password"}
                        >
                            {#if showConfirmPassword}
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
            </div>

            <button type="submit" class="btn-primary" disabled={loading}>
                {#if loading}
                    Creating Account...
                {:else}
                    Create Administrator Account
                {/if}
            </button>
        </form>

        <div class="setup-footer">
            <p>This device will become the system administrator device.</p>
            <p>Other installations will require approval from this account.</p>
        </div>
    </div>
</div>

<style>
    .setup-container {
        min-height: 100vh;
        display: flex;
        align-items: center;
        justify-content: center;
        background: var(--bg-base, #f5f5f7);
        padding: 20px;
    }

    .setup-card {
        background: var(--surface, #fff);
        border-radius: 16px;
        padding: 48px;
        max-width: 480px;
        width: 100%;
        box-shadow: 0 4px 24px rgba(0, 0, 0, 0.08);
    }

    .setup-header {
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
        font-size: 28px;
        font-weight: 700;
        color: var(--text-primary, #1d1d1f);
        margin: 0 0 8px;
    }

    .subtitle {
        color: var(--text-secondary, #86868b);
        font-size: 15px;
        margin: 0;
    }

    .setup-form {
        display: flex;
        flex-direction: column;
        gap: 20px;
    }

    .form-group {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    .form-row {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 16px;
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

    input::placeholder {
        color: var(--text-muted, #c7c7c7);
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

    .setup-footer {
        margin-top: 32px;
        padding-top: 24px;
        border-top: 1px solid var(--border, #e5e5e5);
        text-align: center;
    }

    .setup-footer p {
        color: var(--text-muted, #c7c7c7);
        font-size: 13px;
        margin: 4px 0;
    }

    /* P1-4 FIX: Password visibility toggle */
    .password-input {
        position: relative;
    }

    .password-input input {
        padding-right: 48px; /* Space for toggle button */
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
