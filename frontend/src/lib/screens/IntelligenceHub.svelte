<script>
    import ButlerScreen from "./ButlerScreen.svelte";
    import ErrorBoundary from "../components/ErrorBoundary.svelte";
    import { onMount } from "svelte";

    let mounted = $state(false);

    onMount(() => {
        requestAnimationFrame(() => { mounted = true; });
    });
</script>

<ErrorBoundary name="Intelligence Hub">
    <div class="hub" class:flourish={mounted}>
        <header class="header">
            <h1 class="page-title">Intelligence</h1>
        </header>

        <main class="content">
            <ButlerScreen />
        </main>
    </div>
</ErrorBoundary>

<style>
    .hub {
        min-height: 100vh;
        background: var(--bg-base);
        display: flex;
        flex-direction: column;
        opacity: 0;
        transform: scale(0.97) translateY(6px);
        transition: opacity 0.5s cubic-bezier(0.34, 1.56, 0.64, 1),
                    transform 0.5s cubic-bezier(0.34, 1.56, 0.64, 1);
    }

    .hub.flourish {
        opacity: 1;
        transform: scale(1) translateY(0);
    }

    .header {
        padding: var(--page-padding);
        border-bottom: 1px solid var(--border);
        background: var(--surface);
    }

    .header h1 {
        margin: 0;
    }

    .content {
        padding: var(--page-padding);
        flex: 1;
        display: flex;
        flex-direction: column;
    }
</style>
