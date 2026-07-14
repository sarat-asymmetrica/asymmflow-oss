<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { gsap } from 'gsap';
  import { getDefaultDivisionKey } from '$lib/divisions.svelte';

  interface Props {
    activeRoute?: string;
    collapsed?: boolean;
  }

  let { activeRoute = 'dashboard', collapsed = false }: Props = $props();

  const dispatch = createEventDispatcher();

  let isOpen = $state(!collapsed);
  let segments = $state<HTMLElement[]>([]);
  let toggleButton = $state<HTMLButtonElement | null>(null);

  // Navigation routes mapped to origami segments
  const routes = [
    { id: 'dashboard', label: 'Dashboard', icon: 'dashboard' },
    { id: 'opportunities', label: 'Opportunities', icon: 'lightbulb' },
    { id: 'orders', label: 'Offers & Orders', icon: 'shopping_cart' },
    { id: 'customers', label: 'Customers', icon: 'group' },
    { id: 'suppliers', label: 'Suppliers', icon: 'inventory' }
  ];

  function navigate(id: string) {
    dispatch('navigate', id);
  }

  function toggleCollapse() {
    isOpen = !isOpen;
    dispatch('collapse', !isOpen);

    if (isOpen) {
      unfoldMenu();
    } else {
      foldMenu();
    }
  }

  function unfoldMenu() {
    // UNFOLD SEQUENCE - Elastic cascade from top to bottom
    gsap.to(segments, {
      duration: 0.8,
      rotationX: 0,
      opacity: 1,
      y: 0,
      stagger: 0.1,
      ease: "elastic.out(1, 0.75)",
      overwrite: true
    });

    // Shadow disappears as segments flatten
    gsap.to('.crease-shadow', {
      opacity: 0,
      duration: 0.8,
      delay: 0.2
    });

    // Animate toggle button
    gsap.to(toggleButton, {
      rotation: 180,
      duration: 0.3
    });
  }

  function foldMenu() {
    // FOLD SEQUENCE - Quick collapse from bottom to top
    gsap.to(segments, {
      duration: 0.5,
      rotationX: -90,
      opacity: 0,
      y: -20,
      stagger: {
        amount: 0.3,
        from: "end"
      },
      ease: "power2.in",
      overwrite: true
    });

    // Shadow deepens as segments fold
    gsap.to('.crease-shadow', {
      opacity: 0.5,
      duration: 0.3
    });

    // Animate toggle button
    gsap.to(toggleButton, {
      rotation: 0,
      duration: 0.3
    });
  }

  onMount(() => {
    // Initial state - folded if collapsed
    if (!isOpen) {
      gsap.set(segments, {
        rotationX: -90,
        transformOrigin: "top center",
        opacity: 0,
        y: -20
      });
    } else {
      gsap.set(segments, {
        rotationX: 0,
        transformOrigin: "top center",
        opacity: 1,
        y: 0
      });
    }

    // 3D hover tilt effect for each segment
    segments.forEach(seg => {
      seg.addEventListener('mousemove', (e) => {
        if (!isOpen) return;

        const rect = seg.getBoundingClientRect();
        const y = e.clientY - rect.top;
        const tilt = (y / rect.height - 0.5) * 10; // +/- 5 deg

        gsap.to(seg, {
          rotationX: tilt,
          duration: 0.3,
          ease: "power1.out"
        });
      });

      seg.addEventListener('mouseleave', () => {
        if (!isOpen) return;

        gsap.to(seg, {
          rotationX: 0,
          duration: 0.5,
          ease: "elastic.out(1, 0.5)"
        });
      });
    });
  });

  // Inline SVGs (sovereign, no external deps)
  const icons = {
    dashboard: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7"></rect><rect x="14" y="3" width="7" height="7"></rect><rect x="14" y="14" width="7" height="7"></rect><rect x="3" y="14" width="7" height="7"></rect></svg>`,
    lightbulb: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="17" x2="12" y2="17"></line><path d="M15 14c.2-1 .7-1.7 1.5-2.5 1-1 1.5-2.2 1.5-3.5A6 6 0 0 0 6 8c0 1 .2 2.2 1.5 3.5.7.7 1.3 1.5 1.5 2.5"></path><path d="M9 18h6"></path><path d="M10 22h4"></path></svg>`,
    shopping_cart: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="9" cy="21" r="1"></circle><circle cx="20" cy="21" r="1"></circle><path d="M1 1h4l2.68 13.39a2 2 0 0 0 2 1.61h9.72a2 2 0 0 0 2-1.61L23 6H6"></path></svg>`,
    group: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle><path d="M23 21v-2a4 4 0 0 0-3-3.87"></path><path d="M16 3.13a4 4 0 0 1 0 7.75"></path></svg>`,
    inventory: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path><polyline points="3.27 6.96 12 12.01 20.73 6.96"></polyline><line x1="12" y1="22.08" x2="12" y2="12"></line></svg>`,
    chevron_left: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"></polyline></svg>`,
    chevron_right: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"></polyline></svg>`
  };
</script>

<nav
  class="origami-nav"
  class:collapsed={!isOpen}
  style="perspective: 1000px; transform-style: preserve-3d;"
>
  <!-- Brand Header with Toggle -->
  <div class="nav-header">
    <div class="brand-container">
      <div class="brand-icon">P</div>
      <h1 class="brand-text" class:hidden={!isOpen}>{getDefaultDivisionKey()}</h1>
    </div>

    <!-- Toggle Button -->
    <button
      bind:this={toggleButton}
      class="toggle-btn"
      onclick={toggleCollapse}
      aria-label="Toggle navigation"
    >
      {@html isOpen ? icons.chevron_left : icons.chevron_right}
    </button>
  </div>

  <!-- Origami Segments Container -->
  <div class="origami-container">
    {#each routes as route, i}
      <button
        bind:this={segments[i]}
        class="fold-segment"
        class:active={activeRoute === route.id}
        style="top: {i * 20}%; z-index: {5 - i};"
        onclick={() => navigate(route.id)}
        title={!isOpen ? route.label : ''}
      >
        <!-- Crease shadow (visible when folded) -->
        <div class="crease-shadow"></div>

        <!-- Content -->
        <div class="segment-content">
          <span class="segment-icon">{@html icons[route.icon]}</span>
          <span class="segment-label" class:hidden={!isOpen}>{route.label}</span>
        </div>
      </button>
    {/each}
  </div>

  <!-- System Status Footer -->
  <div class="nav-footer" class:collapsed={!isOpen}>
    <div class="footer-label" class:hidden={!isOpen}>System</div>
    <div class="footer-status">
      <span class="version-text" class:hidden={!isOpen}>v2.0</span>
      <span class="status-indicator"></span>
    </div>
  </div>
</nav>

<style>
  .origami-nav {
    position: relative;
    height: 100%;
    width: 16rem; /* 256px when open */
    background: var(--color-paper, #fdfbf7);
    border-right: 1px solid rgba(0, 0, 0, 0.05);
    display: flex;
    flex-direction: column;
    padding: 1.5rem 0;
    transition: width 0.3s cubic-bezier(0.25, 0.46, 0.45, 0.94);
    overflow: visible;
  }

  .origami-nav.collapsed {
    width: 5rem; /* 80px when collapsed */
  }

  /* Brand Header */
  .nav-header {
    position: relative;
    padding: 0 1.5rem;
    margin-bottom: 2rem;
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 2.5rem;
  }

  .brand-container {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .brand-icon {
    width: 2rem;
    height: 2rem;
    background: #000;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #fff;
    font-weight: 700;
    font-family: var(--font-serif, 'Cormorant Garamond', serif);
    font-size: 1.125rem;
    flex-shrink: 0;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  }

  .brand-text {
    font-family: var(--font-serif, 'Cormorant Garamond', serif);
    font-size: 1.125rem;
    letter-spacing: 0.1em;
    white-space: nowrap;
    transition: opacity 0.2s, width 0.2s;
  }

  .toggle-btn {
    position: absolute;
    right: -0.75rem;
    top: 0.25rem;
    width: 1.5rem;
    height: 1.5rem;
    background: #fff;
    border: 1px solid rgba(0, 0, 0, 0.1);
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    color: rgba(0, 0, 0, 0.4);
    cursor: pointer;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    z-index: 10;
    transition: all 0.2s;
  }

  .toggle-btn:hover {
    color: #000;
    box-shadow: 0 2px 6px rgba(0, 0, 0, 0.15);
  }

  /* Origami Container - 3D perspective */
  .origami-container {
    flex: 1;
    position: relative;
    perspective: 1000px;
    transform-style: preserve-3d;
    overflow: visible;
  }

  /* Individual Fold Segments */
  .fold-segment {
    position: absolute;
    width: 100%;
    height: 20%; /* 5 segments */
    left: 0;
    transform-origin: top center;
    transform-style: preserve-3d;
    background: linear-gradient(135deg, var(--color-paper, #fdfbf7) 0%, #f4f1ea 100%);
    border: none;
    border-bottom: 1px solid rgba(0, 0, 0, 0.05);
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.05);
    cursor: pointer;
    transition: background 0.3s, box-shadow 0.3s;
    padding: 0 1.5rem;
    display: flex;
    align-items: center;
  }

  .fold-segment:hover {
    background: #fff;
    box-shadow: 0 6px 12px rgba(0, 0, 0, 0.08);
  }

  .fold-segment.active {
    background: #000;
    color: #fff;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
  }

  .fold-segment.active .segment-icon {
    transform: scale(1.1);
  }

  /* Crease Shadow (visible when folded) */
  .crease-shadow {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: linear-gradient(to bottom, rgba(0, 0, 0, 0.2), transparent 20%);
    pointer-events: none;
    opacity: 0;
  }

  /* Segment Content */
  .segment-content {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    width: 100%;
    position: relative;
    z-index: 1;
  }

  .segment-icon {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: transform 0.2s;
  }

  .segment-label {
    font-family: var(--font-serif, 'Cormorant Garamond', serif);
    font-size: 0.875rem;
    white-space: nowrap;
    transition: opacity 0.2s, width 0.2s;
  }

  /* Footer */
  .nav-footer {
    padding: 0 1.5rem;
    padding-top: 1rem;
    border-top: 1px solid rgba(0, 0, 0, 0.05);
    margin-top: auto;
  }

  .footer-label {
    font-size: 0.625rem;
    font-family: monospace;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: rgba(0, 0, 0, 0.4);
    margin-bottom: 0.25rem;
    transition: opacity 0.2s;
  }

  .footer-status {
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 0.75rem;
    color: rgba(0, 0, 0, 0.6);
  }

  .version-text {
    transition: opacity 0.2s;
  }

  .status-indicator {
    width: 0.5rem;
    height: 0.5rem;
    border-radius: 50%;
    background: #22c55e;
    box-shadow: 0 0 8px rgba(34, 197, 94, 0.6);
    animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
  }

  @keyframes pulse {
    0%, 100% {
      opacity: 1;
    }
    50% {
      opacity: 0.5;
    }
  }

  /* Hidden state for text elements when collapsed */
  .hidden {
    opacity: 0;
    width: 0;
    overflow: hidden;
  }
</style>
