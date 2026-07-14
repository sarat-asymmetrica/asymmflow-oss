<script>
    import { createEventDispatcher } from "svelte";
    import { motionMs } from "$lib/motion";
    import { fade, fly } from "svelte/transition";
    import { toast } from "../stores/toasts";
    import { getDefaultDivisionKey } from "$lib/divisions.svelte";

    const dispatch = createEventDispatcher();

    // State
    let step = $state(0);
    let userName = $state("");
    let selectedRole = "";
    let isConfiguring = false;
    let progress = $state(0);
    let currentTask = $state("");

    const roles = [
        {
            id: "sales",
            label: "Sales & BD",
            description: "RFQs, Costing, Pipeline",
            icon: "",
        },
        {
            id: "ops",
            label: "Operations",
            description: "Orders, Shipments, Suppliers",
            icon: "",
        },
        {
            id: "finance",
            label: "Finance",
            description: "Invoices, Accounting",
            icon: "",
        },
        {
            id: "management",
            label: "Management",
            description: "Full oversight",
            icon: "",
        },
    ];

    function nextStep() {
        if (step === 1 && !userName) {
            toast.warning("Please enter your name to continue.");
            return;
        }
        step++;
    }

    async function selectRole(roleId) {
        selectedRole = roleId;
        step = 3;
        await startConfiguration();
    }

    async function startConfiguration() {
        isConfiguring = true;
        const tasks = [
            "Initializing product index...",
            "Syncing regional pricing...",
            "Calibrating analytics...",
            "Securing database...",
            "Personalizing workspace...",
        ];

        for (let i = 0; i < tasks.length; i++) {
            progress = ((i + 1) / tasks.length) * 100;
            currentTask = tasks[i];
            await new Promise((r) => setTimeout(r, 600));
        }

        setTimeout(() => {
            dispatch("complete", { userName, selectedRole });
        }, 400);
    }

    function focusOnMount(node) {
        requestAnimationFrame(() => node.focus());
        return {
            destroy() {},
        };
    }
</script>

<div class="ceremony" in:fade={{ duration: motionMs(400) }}>
    <div class="content">
        {#if step === 0}
            <!-- Welcome -->
            <div class="step" in:fly={{ y: 30, duration: motionMs(400) }}>
                <div class="logo">PH</div>
                <h1>Welcome to<br />PH Sovereign.</h1>
                <p class="intro">
                    The unified professional infrastructure for {getDefaultDivisionKey()}.
                </p>
                <button class="btn-primary" onclick={nextStep}>
                    Begin Setup
                </button>
            </div>
        {:else if step === 1}
            <!-- Name Input -->
            <div class="step" in:fly={{ y: 30, duration: motionMs(400) }}>
                <span class="label">Identity</span>
                <h2>Who's here?</h2>
                <div class="input-container">
                    <input
                        type="text"
                        bind:value={userName}
                        placeholder="Enter your name"
                        onkeydown={(e) => e.key === "Enter" && nextStep()}
                        use:focusOnMount
                    />
                    <button class="btn-circle" onclick={nextStep}>&gt;</button>
                </div>
            </div>
        {:else if step === 2}
            <!-- Role Selection -->
            <div class="step wide" in:fly={{ y: 30, duration: motionMs(400) }}>
                <span class="label">Role</span>
                <h2>Welcome, {userName}.</h2>
                <p>Select your primary domain.</p>

                <div class="role-grid">
                    {#each roles as role}
                        <button
                            class="role-card"
                            onclick={() => selectRole(role.id)}
                        >
                            <span class="role-icon">{role.icon}</span>
                            <div class="role-text">
                                <h3>{role.label}</h3>
                                <p>{role.description}</p>
                            </div>
                            <span class="arrow">&gt;</span>
                        </button>
                    {/each}
                </div>
            </div>
        {:else if step === 3}
            <!-- Configuration -->
            <div class="step" in:fly={{ y: 30, duration: motionMs(400) }}>
                <span class="label">Setup</span>
                <h2>Configuring<br />your workspace.</h2>

                <div class="progress-section">
                    <div class="progress-track">
                        <div
                            class="progress-bar"
                            style="width: {progress}%"
                        ></div>
                    </div>
                    <p class="task">{currentTask}</p>
                </div>
            </div>
        {/if}
    </div>

    <!-- Bottom Branding -->
    <div class="footer">
        <span>Powered by Asymmetrica</span>
    </div>
</div>

<style>
    .ceremony {
        position: fixed;
        inset: 0;
        background: #ffffff;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        z-index: 1000;
    }

    .content {
        width: 100%;
        max-width: 560px;
        padding: 40px;
    }

    .step {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 24px;
        text-align: center;
    }

    .step.wide {
        max-width: 640px;
    }

    .logo {
        width: 80px;
        height: 80px;
        background: #1c1c1c;
        color: #ffffff;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-family: "DM Sans", sans-serif;
        font-weight: 600;
        font-size: 28px;
        letter-spacing: 2px;
        margin-bottom: 16px;
    }

    h1 {
        font-family: "DM Sans", sans-serif;
        font-size: 56px;
        font-weight: 300;
        line-height: 1.1;
        letter-spacing: -0.02em;
        color: #1c1c1c;
        margin: 0;
    }

    h2 {
        font-family: "DM Sans", sans-serif;
        font-size: 40px;
        font-weight: 300;
        line-height: 1.2;
        letter-spacing: -0.02em;
        color: #1c1c1c;
        margin: 0;
    }

    p {
        font-family: "DM Sans", sans-serif;
        font-size: 16px;
        color: #666;
        line-height: 1.5;
        margin: 0;
    }

    .intro {
        font-size: 18px;
        max-width: 360px;
    }

    .label {
        font-family: "DM Sans", sans-serif;
        font-size: 12px;
        font-weight: 500;
        letter-spacing: 0.1em;
        text-transform: uppercase;
        color: #999;
    }

    .btn-primary {
        padding: 16px 48px;
        background: #1c1c1c;
        color: #ffffff;
        border: none;
        border-radius: 100px;
        font-family: "DM Sans", sans-serif;
        font-size: 15px;
        font-weight: 500;
        cursor: pointer;
        transition:
            transform 0.15s ease,
            background 0.15s ease;
    }

    .btn-primary:hover {
        transform: translateY(-2px);
        background: #333;
    }

    .input-container {
        display: flex;
        align-items: center;
        gap: 16px;
        width: 100%;
        max-width: 400px;
        padding: 16px 24px;
        background: #e5e5e5;
        border-radius: 100px;
    }

    input {
        flex: 1;
        background: transparent;
        border: none;
        font-family: "DM Sans", sans-serif;
        font-size: 18px;
        color: #1c1c1c;
        outline: none;
    }

    input::placeholder {
        color: #999;
    }

    .btn-circle {
        width: 44px;
        height: 44px;
        border-radius: 50%;
        background: #1c1c1c;
        color: #ffffff;
        border: none;
        font-size: 20px;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: transform 0.15s ease;
    }

    .btn-circle:hover {
        transform: scale(1.05);
    }

    .role-grid {
        display: flex;
        flex-direction: column;
        gap: 12px;
        width: 100%;
        margin-top: 16px;
    }

    .role-card {
        display: flex;
        align-items: center;
        gap: 20px;
        padding: 20px 24px;
        background: #e5e5e5;
        border: none;
        border-radius: 40px;
        text-align: left;
        cursor: pointer;
        transition: all 0.15s ease;
    }

    .role-card:hover {
        background: #d5d5d5;
        transform: translateX(8px);
    }

    .role-icon {
        font-size: 32px;
        width: 48px;
        height: 48px;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .role-text {
        flex: 1;
    }

    .role-text h3 {
        font-family: "DM Sans", sans-serif;
        font-size: 18px;
        font-weight: 500;
        color: #1c1c1c;
        margin: 0;
    }

    .role-text p {
        font-size: 14px;
        color: #666;
        margin: 4px 0 0;
    }

    .arrow {
        font-size: 20px;
        color: #999;
    }

    .role-card:hover .arrow {
        color: #1c1c1c;
    }

    .progress-section {
        width: 100%;
        max-width: 360px;
        margin-top: 24px;
    }

    .progress-track {
        height: 4px;
        background: #e5e5e5;
        border-radius: 2px;
        overflow: hidden;
    }

    .progress-bar {
        height: 100%;
        background: #1c1c1c;
        transition: width 0.3s ease;
    }

    .task {
        margin-top: 16px;
        font-size: 14px;
        color: #999;
    }

    .footer {
        position: fixed;
        bottom: 32px;
        font-family: "DM Sans", sans-serif;
        font-size: 12px;
        color: #ccc;
        letter-spacing: 0.05em;
    }

    @media (max-width: 640px) {
        h1 {
            font-size: 36px;
        }
        h2 {
            font-size: 28px;
        }
        .content {
            padding: 24px;
        }
        .role-card {
            border-radius: 24px;
            padding: 16px 20px;
        }
    }
</style>
