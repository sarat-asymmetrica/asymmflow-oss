<script lang="ts">
  
import { devLog } from "$lib/utils/devLog";
import { onMount } from "svelte";
  // GetScreenLayout not yet implemented in backend
const GetScreenLayout = async (_screenId: string) => null;
  
  // Dynamic Imports for Components
  import SurvivalGarden from "./canvas/SurvivalGarden.svelte";
  import OpportunityMandala from "./canvas/OpportunityMandala.svelte";
  import ServiceRhythm from "./canvas/ServiceRhythm.svelte";
  import MetricCard from "./MetricCard.svelte";
  import DataTable from "./ui/DataTable.svelte";
  import AlchemyForm from "./ui/AlchemyForm.svelte";
  import CostingSheet from "./ui/CostingSheet.svelte";
  import TaskFlow from "./ui/TaskFlow.svelte";
  import ButlerCharacter from "./ui/ButlerCharacter.svelte";
  import KintsugiError from "./ui/KintsugiError.svelte";
  import SpaceTimeCanvas from "./canvas/SpaceTimeCanvas.svelte";

  interface Props {
    screenID?: string;
  }

  let { screenID = "dashboard" }: Props = $props();

  let layout = $state(null);
  let loading = $state(true);
  let errorMessage = $state("");

  // Component Mapping
  const COMPONENT_MAP = {
    "survival_garden": SurvivalGarden,
    "opportunity_mandala": OpportunityMandala,
    "service_rhythm": ServiceRhythm,
    "metric_card": MetricCard,
    "data_table": DataTable,
    "form": AlchemyForm,
    "costing_sheet": CostingSheet,
    "task_flow": TaskFlow,
    "butler_insight": ButlerCharacter,
  };

  // Development Mock Data (Fallback when Wails backend is unreachable)
  const MOCK_DATA = {
    "dashboard": {
        "id": "dashboard",
        "title": "Dashboard (Offline Mode)",
        "type": "generated",
        "grid_template": "\"insight insight insight\" \"tasks tasks garden\" \"tasks tasks garden\"",
        "theme": { "primary_color": "#1c1c1c", "accent_color": "#fbbf24", "background_color": "#fdfbf7" },
        "regime": {
            "name": "Morning Calm",
            "primary_color": "#fdfbf7",
            "secondary_color": "#e2e8f0",
            "geometry": { "type": "FluidPlane", "complexity": 0.2, "roughness": 0.9, "metalness": 0.1 },
            "physics": { "flow_rate": 0.1, "turbulence": 0.05, "gravity": 0.0, "viscosity": 0.8 }
        },
        "components": [
            { "id": "butler_insight", "type": "butler_insight", "grid_area": "insight", "regime": 1, "data": { "message": "Good morning. System running in Offline Mode.", "sentiment": "neutral" } },
            { "id": "survival_garden", "type": "survival_garden", "grid_area": "garden", "regime": 3, "data": { "runwayMonths": 6.2, "burnRate": 11000, "flowRate": 45.5 } },
            { "id": "todays_flow", "type": "task_flow", "grid_area": "tasks", "regime": 1, "data": { "tasks": [{ "title": "Check Backend Connection", "time": "Now", "subtitle": "Wails not detected", "color": "#ef4444" }] } }
        ]
    },
    "costing": {
        "id": "costing",
        "title": "Costing (Offline)",
        "type": "generated",
        "grid_template": "\"sheet sheet sheet\"",
        "theme": { "primary_color": "#1c1c1c", "accent_color": "#fbbf24", "background_color": "#fdfbf7" },
        "regime": { "name": "Focus", "primary_color": "#fdfbf7", "secondary_color": "#e2e8f0" },
        "components": [
            { "id": "active_costing_sheet", "type": "costing_sheet", "grid_area": "sheet", "regime": 2, "data": { "currency": "BHD", "markup": 1.2, "items": [{ "description": "Offline Item", "quantity": 1, "unitCost": 100, "margin": 20 }] } }
        ]
    }
  };

  onMount(async () => {
    try {
      layout = await GetScreenLayout(screenID);
      errorMessage = "";
    } catch (e) {
      devLog.warn("Backend unavailable, falling back to mock data:", e);
      // Fallback to mock data if backend fails
      if (MOCK_DATA[screenID]) {
          layout = MOCK_DATA[screenID];
          errorMessage = "";
      } else {
          errorMessage = `Failed to generate ${screenID} screen. Backend unavailable.`;
      }
    } finally {
      loading = false;
    }
  });
</script>

{#if loading}
  <div class="flex items-center justify-center h-screen">
    <div class="text-center">
       <div class="text-2xl font-serif animate-pulse" style="margin-bottom: var(--space-xs)">Growing {screenID}...</div>
       <div class="text-xs font-mono text-gray-400">Calculating Geodesics...</div>
    </div>
  </div>
{:else if errorMessage}
  <div class="flex items-center justify-center h-screen" style="padding: var(--space-2xl)">
    <KintsugiError message={errorMessage} show={true} />
  </div>
{:else if layout}
  <div class="w-full min-h-screen"
       style="padding: var(--space-lg); background-color: {layout.theme.background_color}">

    <header class="text-center border-b border-[#e5e0d8]" style="margin-bottom: var(--space-lg); padding-bottom: var(--space-md)">
      <h1 class="text-3xl font-serif font-normal tracking-wide capitalize">{layout.title}</h1>
    </header>

    <!-- The CSS Grid Layout -->
    <div class="grid content-container" style="gap: var(--space-md); grid-template-areas: {layout.grid_template};">

      {#each layout.components as comp}
        <!-- Enforce min-height for specific components to prevent collapse -->
        <div style="grid-area: {comp.grid_area};"
             class="relative {['survival_garden', 'opportunity_mandala', 'service_rhythm'].includes(comp.type) ? 'min-h-[350px]' : 'min-h-min'}">
          {#if COMPONENT_MAP[comp.type]}
            {@const SvelteComponent = COMPONENT_MAP[comp.type]}
            <SvelteComponent
              {...comp.data}
            />
          {:else}
             <KintsugiError message="Missing component: {comp.type}" show={true} />
          {/if}
        </div>
      {/each}

    </div>
  </div>

  <!-- THE MAGICAL BACKGROUND LAYER -->
  {#if layout && layout.regime}
    <SpaceTimeCanvas regime={layout.regime} />
  {/if}
{/if}

<style>
  .content-container {
    display: grid;
    /* Default to 3 columns as per the backend template assumption */
    grid-template-columns: repeat(3, 1fr); 
    width: 100%;
    max-width: 1600px; /* Prevent ultra-wide stretching */
    margin: 0 auto;    /* Center content */
  }

  @media (max-width: 1024px) {
    .content-container {
      grid-template-columns: 1fr !important;
      grid-template-areas: initial !important; /* Reset areas to stack vertically */
      display: flex;
      flex-direction: column;
    }
    
    /* Ensure components that rely on height (canvas) still have size */
    :global(.component-card) {
      min-height: 300px;
    }
  }
</style>
