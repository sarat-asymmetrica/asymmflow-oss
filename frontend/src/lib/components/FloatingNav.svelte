<script lang="ts">
    import { createEventDispatcher, onMount } from "svelte";
    import { fade, fly } from "svelte/transition";

    interface Props {
        currentScreen?: string;
    }

    let { currentScreen = "" }: Props = $props();

    const dispatch = createEventDispatcher();
    let visible = $state(false);

    const quickLinks = [
        { id: "dashboard", label: "Dashboard" },
        { id: "opportunities", label: "Opportunities" },
        { id: "operations", label: "Operations" },
        { id: "finance", label: "Finance" },
        { id: "relationships", label: "Customers & Suppliers" },
        { id: "intelligence", label: "Intelligence" },
    ];

    function navigate(screenId) {
        dispatch("navigate", { screen: screenId });
    }

    // Scroll Logic
    function handleScroll(e) {
        // We use capture=true on window to catch scroll events from any child
        // If e.target is the document or window, use scrollY.
        // If it's an element (like .main-content), use scrollTop.
        let scrollTop = 0;
        if (e.target && e.target.scrollTop) {
            scrollTop = e.target.scrollTop;
        } else if (window.scrollY) {
            scrollTop = window.scrollY;
        }
        
        // Show floating nav after scrolling down 100px
        visible = scrollTop > 100;
    }

    onMount(() => {
        window.addEventListener("scroll", handleScroll, true); // Capture phase
        return () => window.removeEventListener("scroll", handleScroll, true);
    });
</script>

{#if visible}
    <nav class="floating-nav" transition:fly={{ y: 50, duration: 200 }}>
        <div class="nav-container">
            {#each quickLinks as link}
                <button
                    class="quick-link"
                    class:active={currentScreen === link.id}
                    onclick={() => navigate(link.id)}
                >
                    {link.label}
                </button>
            {/each}
        </div>
    </nav>
{/if}

<style>
    .floating-nav {
        position: fixed;
        bottom: 24px;
        left: 50%;
        transform: translateX(-50%);
        z-index: 1000;
    }

    .nav-container {
        display: flex;
        gap: 4px;
        background: rgba(28, 28, 28, 0.9);
        backdrop-filter: blur(12px);
        padding: 4px;
        border-radius: 100px;
        box-shadow:
            0 10px 30px rgba(0, 0, 0, 0.2),
            0 0 0 1px rgba(255, 255, 255, 0.1);
    }

    .quick-link {
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 8px 16px;
        background: transparent;
        border: none;
        border-radius: 100px;
        color: rgba(255, 255, 255, 0.6);
        cursor: pointer;
        transition: all 0.2s ease;
        
        font-family: var(--font-body, "DM Sans", sans-serif);
        font-size: 13px;
        font-weight: 400;
        white-space: nowrap;
    }

    .quick-link:hover {
        color: #fff;
        background: rgba(255, 255, 255, 0.1);
    }

    .quick-link.active {
        color: #1c1c1c;
        background: #fff;
        font-weight: 600;
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
    }
</style>
