<script lang="ts">
    import { onMount, onDestroy } from "svelte";
    import * as d3 from "d3";
    import WabiSpinner from "../components/ui/WabiSpinner.svelte";
    import {
        GetEntityGraph } from "../../../wailsjs/go/main/App";
import { GetGraphStats, BuildEntityGraph } from "../../../wailsjs/go/main/CRMService";
    import { toast } from "$lib/stores/toasts";

    let loading = $state(true);
    let building = $state(false);
    let graphData = { nodes: [], links: [] };
    let stats = $state({ totalNodes: 0, totalEdges: 0 });
    let activeFilter = $state("All");

    // D3 Refs
    let svgElement = $state();
    let simulation;
    let container = $state();
    let width = 800;
    let height = 600;

    const filters = [
        "All",
        "Customers",
        "Offers",
        "Products",
        "Contacts",
        "Industries",
    ];

    // Milestoners Palette for Nodes
    const nodeColors = {
        Customer: "#166534", // Green
        Offer: "#1e40af", // Blue
        Product: "#c2410c", // Orange
        Contact: "#7e22ce", // Purple
        Industry: "#a16207", // Yellow/Brown
        default: "#404040",
    };

    async function loadGraph() {
        loading = true;
        try {
            // Mock or Real
            if (!window.go) {
                // Mock
                graphData = {
                    nodes: Array.from({ length: 20 }, (_, i) => ({
                        id: i,
                        name: `Node ${i}`,
                        type: i % 2 ? "Customer" : "Product",
                    })),
                    links: Array.from({ length: 30 }, (_, i) => ({
                        source: Math.floor(Math.random() * 20),
                        target: Math.floor(Math.random() * 20),
                    })),
                };
            } else {
                const typeC =
                    activeFilter === "All" ? "" : activeFilter.slice(0, -1);
                const res = await GetEntityGraph(typeC, 500);
                graphData = {
                    nodes: (res.nodes || []).map((n) => ({
                        id: n.id,
                        name: n.label,
                        type: n.type,
                        val: 5,
                    })),
                    links: (res.links || []).map((l) => ({
                        source: l.source,
                        target: l.target,
                    })),
                };
                const st = await GetGraphStats();
                stats = {
                    totalNodes: st.total_nodes,
                    totalEdges: st.total_edges,
                };
            }
            renderGraph();
        } catch (e) {
            toast.danger("Graph load failed");
        } finally {
            loading = false;
        }
    }

    async function rebuild() {
        building = true;
        try {
            await BuildEntityGraph();
            await loadGraph();
            toast.success("Graph rebuilt");
        } catch (e) {
            toast.danger("Rebuild failed");
        } finally {
            building = false;
        }
    }

    function renderGraph() {
        if (!svgElement) return;

        // Clear old
        d3.select(svgElement).selectAll("*").remove();

        const svg = d3
            .select(svgElement)
            .attr("viewBox", [0, 0, width, height])
            .call(
                d3
                    .zoom()
                    .scaleExtent([0.1, 8])
                    .on("zoom", (event) =>
                        g.attr("transform", event.transform),
                    ),
            );

        const g = svg.append("g");

        simulation = d3
            .forceSimulation(graphData.nodes)
            .force(
                "link",
                d3
                    .forceLink(graphData.links)
                    .id((d) => d.id)
                    .distance(100),
            )
            .force("charge", d3.forceManyBody().strength(-300))
            .force("center", d3.forceCenter(width / 2, height / 2));

        const link = g
            .append("g")
            .attr("stroke", "#e5e5e5")
            .attr("stroke-opacity", 0.6)
            .selectAll("line")
            .data(graphData.links)
            .join("line")
            .attr("stroke-width", 1);

        const node = g
            .append("g")
            .attr("stroke", "#fff")
            .attr("stroke-width", 1.5)
            .selectAll("circle")
            .data(graphData.nodes)
            .join("circle")
            .attr("r", 8) // fixed radius for now
            .attr("fill", (d) => nodeColors[d.type] || nodeColors.default)
            .call(drag(simulation));

        node.append("title").text((d) => d.name);

        // Labels
        const labels = g
            .append("g")
            .attr("class", "labels")
            .selectAll("text")
            .data(graphData.nodes)
            .join("text")
            .attr("dx", 12)
            .attr("dy", ".35em")
            .text((d) => d.name)
            .style("font-size", "10px")
            .style("font-family", "DM Sans, sans-serif")
            .style("fill", "#666");

        simulation.on("tick", () => {
            link.attr("x1", (d) => d.source.x)
                .attr("y1", (d) => d.source.y)
                .attr("x2", (d) => d.target.x)
                .attr("y2", (d) => d.target.y);

            node.attr("cx", (d) => d.x).attr("cy", (d) => d.y);

            labels.attr("x", (d) => d.x).attr("y", (d) => d.y);
        });
    }

    function drag(simulation) {
        function dragstarted(event) {
            if (!event.active) simulation.alphaTarget(0.3).restart();
            event.subject.fx = event.subject.x;
            event.subject.fy = event.subject.y;
        }

        function dragged(event) {
            event.subject.fx = event.x;
            event.subject.fy = event.y;
        }

        function dragended(event) {
            if (!event.active) simulation.alphaTarget(0);
            event.subject.fx = null;
            event.subject.fy = null;
        }

        return d3
            .drag()
            .on("start", dragstarted)
            .on("drag", dragged)
            .on("end", dragended);
    }

    // Resize observer to update width/height
    function resizeAction(node) {
        const ro = new ResizeObserver((entries) => {
            const entry = entries[0];
            width = entry.contentRect.width;
            height = entry.contentRect.height;
            if (simulation && !loading) renderGraph();
        });
        ro.observe(node);
        return {
            destroy() {
                ro.disconnect();
            },
        };
    }

    onMount(loadGraph);
    onDestroy(() => simulation?.stop());
</script>

<div class="page">
    <header class="header">
        <div class="header-content">
            <h1>Discovery.</h1>
            <p class="subtitle">Entity Graph & Relationships</p>
        </div>
        <div class="header-actions">
            {#if building}
                <span class="status-txt">Indexing...</span>
            {:else}
                <button class="btn-primary" onclick={rebuild}
                    >Rebuild Index</button
                >
            {/if}
        </div>
    </header>

    <div class="layout-split">
        <aside class="sidebar">
            <div class="filters">
                <h3>Filter Nodes</h3>
                {#each filters as f}
                    <button
                        class="filter-item"
                        class:active={activeFilter === f}
                        onclick={() => {
                            activeFilter = f;
                            loadGraph();
                        }}
                    >
                        {f}
                    </button>
                {/each}
            </div>

            <div class="stats-box">
                <div class="stat">
                    <span class="val">{stats.totalNodes}</span>
                    <span class="lbl">Nodes</span>
                </div>
                <div class="stat">
                    <span class="val">{stats.totalEdges}</span>
                    <span class="lbl">Edges</span>
                </div>
            </div>

            <div class="legend">
                <h3>Legend</h3>
                {#each Object.entries(nodeColors) as [k, v]}
                    {#if k !== "default"}
                        <div class="legend-item">
                            <span class="dot" style="background: {v}"></span>
                            <span class="txt">{k}</span>
                        </div>
                    {/if}
                {/each}
            </div>
        </aside>

        <main class="graph-area" use:resizeAction bind:this={container}>
            {#if loading}
                <div class="loading">
                    <WabiSpinner size="lg" tempo="calm" />
                </div>
            {:else}
                <svg bind:this={svgElement} width="100%" height="100%"></svg>
            {/if}
        </main>
    </div>
</div>

<style>
    .page {
        padding: var(--page-padding);
        height: 100vh;
        background: var(--paper);
        color: var(--ink);
        display: flex;
        flex-direction: column;
        box-sizing: border-box;
    }

    .header {
        display: flex;
        justify-content: space-between;
        align-items: flex-end;
        margin-bottom: var(--space-6);
        flex-shrink: 0;
    }
    h1 {
        font-size: var(--text-5xl);
        font-weight: var(--font-weight-light);
        margin: 0;
        letter-spacing: -0.02em;
    }
    .subtitle {
        color: var(--ink-faint);
        margin-top: var(--space-2);
    }

    .btn-primary {
        background: var(--ink);
        color: var(--paper);
        border: none;
        padding: 10px 20px;
        border-radius: var(--radius-pill);
        cursor: pointer;
    }
    .status-txt {
        font-style: italic;
        color: var(--ink-light);
    }

    .layout-split {
        display: grid;
        grid-template-columns: 240px 1fr;
        gap: var(--space-8);
        flex: 1;
        min-height: 0;
    }

    .sidebar {
        border-right: 1px solid var(--border-subtle);
        padding-right: 16px;
        display: flex;
        flex-direction: column;
        gap: 24px;
    }

    .filters h3,
    .legend h3,
    .stats-box .lbl {
        font-size: 11px;
        text-transform: uppercase;
        color: var(--ink-light);
        margin-bottom: 12px;
    }

    .filter-item {
        display: block;
        width: 100%;
        text-align: left;
        padding: 8px 12px;
        margin-bottom: 2px;
        background: transparent;
        border: 1px solid transparent;
        border-radius: 6px;
        font-size: 13px;
        color: var(--ink-light);
        cursor: pointer;
    }
    .filter-item:hover {
        background: var(--paper-subtle);
        color: var(--ink);
    }
    .filter-item.active {
        background: var(--ink);
        color: var(--paper);
        font-weight: 500;
        border-color: var(--ink);
    }

    .stats-box {
        display: flex;
        gap: 24px;
        padding: 16px;
        background: var(--paper-subtle);
        border-radius: 12px;
    }
    .stat {
        display: flex;
        flex-direction: column;
    }
    .val {
        font-size: 20px;
        font-weight: 500;
    }

    .legend-item {
        display: flex;
        gap: 8px;
        align-items: center;
        margin-bottom: 6px;
        font-size: 12px;
        color: var(--ink-light);
    }
    .dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
    }

    .graph-area {
        background: var(--paper-subtle);
        border-radius: var(--radius-xl);
        border: 1px solid var(--border-subtle);
        position: relative;
        overflow: hidden;
    }
    .loading {
        display: flex;
        align-items: center;
        justify-content: center;
        height: 100%;
    }
</style>
