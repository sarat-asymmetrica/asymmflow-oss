<script lang="ts">
  
import { devLog } from "$lib/utils/devLog";
/**
   * Ecosystem Intelligence Dashboard
   * VQC-powered repository navigation + market research
   * Built with Wabi-Sabi Design System
   * 
   * December 8th, 2025 - Making the ecosystem visible
   */
  import { onMount } from 'svelte';
  import WabiCard from '$lib/components/ui/WabiCard.svelte';
  import WabiStatCard from '$lib/components/ui/WabiStatCard.svelte';
  import WabiButton from '$lib/components/ui/WabiButton.svelte';
  import WabiSpinner from '$lib/components/ui/WabiSpinner.svelte';
  import WabiBadge from '$lib/components/ui/WabiBadge.svelte';
  import { RUNTIME_URL, API_ENDPOINTS } from '$lib/config';

  const API_BASE = RUNTIME_URL;

  // State
  let loading = $state(true);
  let scanning = $state(false);
  let summary: any = $state(null);
  let recentFiles: any[] = $state([]);
  let searchQuery = $state('');
  let searchResults: any[] = $state([]);
  let marketResearch: any[] = $state([]);
  let edgePages: any[] = $state([]);
  let error: string | null = $state(null);

  // Fetch ecosystem summary
  async function fetchSummary() {
    try {
      const res = await fetch(`${API_BASE}/api/ecosystem/summary`);
      summary = await res.json();
    } catch (e) {
      devLog.error('Failed to fetch summary:', e);
    }
  }

  // Fetch recent files
  async function fetchRecentFiles() {
    try {
      const res = await fetch(`${API_BASE}/api/nodes/tag/ecosystem:file`);
      const files = await res.json();
      recentFiles = files
        .sort((a: any, b: any) => {
          const aScore = parseFloat(a.fields?.importance_score?.toString() || '0');
          const bScore = parseFloat(b.fields?.importance_score?.toString() || '0');
          return bScore - aScore;
        })
        .slice(0, 20);
    } catch (e) {
      devLog.error('Failed to fetch files:', e);
    }
  }

  // Scan ecosystem
  async function scanEcosystem() {
    scanning = true;
    error = null;
    try {
      // Let backend determine the root path via ASYMM_MATH_ROOT env var or discovery
      const res = await fetch(`${API_BASE}/api/ecosystem/scan`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ maxResults: 200 })
      });
      const result = await res.json();
      if (result.success) {
        // PARALLEL - 2× faster!
        await Promise.all([
          fetchSummary(),
          fetchRecentFiles()
        ]);
      } else {
        error = result.error;
      }
    } catch (e: any) {
      error = e.message;
    } finally {
      scanning = false;
    }
  }

  // Search ecosystem
  async function searchEcosystem() {
    if (!searchQuery.trim()) return;
    try {
      const res = await fetch(`${API_BASE}/api/ecosystem/search?query=${encodeURIComponent(searchQuery)}&limit=20`);
      const result = await res.json();
      
      // Fetch search results
      const searchRes = await fetch(`${API_BASE}/api/nodes/tag/ecosystem:search`);
      const searches = await searchRes.json();
      searchResults = searches.filter((s: any) => 
        s.fields?.query?.toString().includes(searchQuery)
      );
    } catch (e) {
      devLog.error('Search failed:', e);
    }
  }

  // Fetch market research
  async function fetchMarketResearch() {
    try {
      const res = await fetch(`${API_BASE}/api/nodes/tag/market:competitor`);
      marketResearch = await res.json();
    } catch (e) {
      devLog.error('Failed to fetch market research:', e);
    }
  }

  // Fetch Edge pages
  async function fetchEdgePages() {
    try {
      const res = await fetch(`${API_BASE}/api/edge/current`);
      const result = await res.json();
      if (result.success) {
        edgePages = result.pages || [];
      }
    } catch (e) {
      devLog.error('Failed to fetch Edge pages:', e);
    }
  }

  // Extract page content
  async function extractPage(url: string, tag: string) {
    try {
      await fetch(`${API_BASE}/api/edge/extract`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url, targetTag: tag })
      });
      await fetchMarketResearch();
    } catch (e) {
      devLog.error('Extract failed:', e);
    }
  }

  onMount(async () => {
    await Promise.all([
      fetchSummary(),
      fetchRecentFiles(),
      fetchMarketResearch(),
      fetchEdgePages()
    ]);
    loading = false;
  });

  // Helpers
  function getImportanceColor(tier: string): string {
    switch (tier) {
      case 'critical': return 'var(--color-danger)';
      case 'high': return 'var(--color-warning)';
      case 'medium': return 'var(--color-gold)';
      default: return 'var(--color-stone)';
    }
  }

  function formatPath(path: string): string {
    // Strip common prefixes to show relative paths
    return path?.replace(/^.*[\\\/](asymm_mathematical_organism|asymm_all_math|ACE Engine)[\\\/]/, '') || '';
  }
</script>

<svelte:head>
  <title>Ecosystem Intelligence | Asymmetrica</title>
</svelte:head>

<div class="dashboard">
  <!-- Header -->
  <header class="dashboard-header">
    <div class="header-content">
      <h1>Ecosystem Intelligence</h1>
      <p class="subtitle">VQC-powered repository navigation</p>
    </div>
    <div class="header-actions">
      <WabiButton 
        variant="primary" 
        on:click={scanEcosystem}
        disabled={scanning}
      >
        {#if scanning}
          <WabiSpinner size="sm" />
          Scanning...
        {:else}
          Scan Ecosystem
        {/if}
      </WabiButton>
    </div>
  </header>

  {#if loading}
    <div class="loading-state">
      <WabiSpinner size="lg" />
      <p>Loading ecosystem data...</p>
    </div>
  {:else}
    <!-- Stats Row -->
    <section class="stats-row">
      <WabiCard variant="elevated" padding="md">
        <WabiStatCard 
          label="Total Files" 
          value={summary?.totalFiles || 0}
          format="number"
          animate={true}
        />
      </WabiCard>
      
      <WabiCard variant="elevated" padding="md">
        <WabiStatCard 
          label="Connections" 
          value={summary?.totalConnections || 0}
          format="number"
          animate={true}
        />
      </WabiCard>
      
      <WabiCard variant="elevated" padding="md">
        <WabiStatCard 
          label="Critical Files" 
          value={summary?.importanceTiers?.critical || 0}
          format="number"
          trend="up"
          trendValue="High priority"
          animate={true}
        />
      </WabiCard>
      
      <WabiCard variant="elevated" padding="md">
        <WabiStatCard 
          label="Market Research" 
          value={marketResearch.length}
          format="number"
          animate={true}
        />
      </WabiCard>
    </section>

    <!-- Quick Context -->
    {#if summary?.quickContext}
      <WabiCard variant="outlined" padding="md">
        <div class="quick-context">
          <span class="context-label">Quick Context</span>
          <p class="context-text">{summary.quickContext}</p>
        </div>
      </WabiCard>
    {/if}

    <!-- Main Grid -->
    <div class="main-grid">
      <!-- Left Column: Files -->
      <section class="files-section">
        <WabiCard padding="md">
          {#snippet header()}
                    <div  class="section-header">
              <h2>Top Files by Importance</h2>
              <span class="file-count">{recentFiles.length} files</span>
            </div>
                  {/snippet}
          
          <div class="file-list">
            {#each recentFiles as file}
              {@const importance = parseFloat(file.fields?.importance_score?.toString() || '0')}
              {@const tier = file.tags?.find((t) => t.startsWith('importance:'))?.replace('importance:', '') || 'low'}
              <div class="file-item">
                <div class="file-info">
                  <span class="file-path">{formatPath(file.fields?.path?.toString())}</span>
                  <span class="file-category">{file.fields?.category?.toString() || 'other'}</span>
                </div>
                <div class="file-score" style="color: {getImportanceColor(tier)}">
                  {(importance * 100).toFixed(0)}%
                </div>
              </div>
            {/each}
          </div>
        </WabiCard>
      </section>

      <!-- Right Column: Search & Research -->
      <section class="research-section">
        <!-- Search -->
        <WabiCard padding="md">
          {#snippet header()}
                    <div  class="section-header">
              <h2>Search Ecosystem</h2>
            </div>
                  {/snippet}
          
          <div class="search-box">
            <input 
              type="text" 
              bind:value={searchQuery}
              placeholder="quaternion, vqc, bio-resonance..."
              onkeydown={(e) => e.key === 'Enter' && searchEcosystem()}
            />
            <WabiButton variant="secondary" on:click={searchEcosystem}>
              Search
            </WabiButton>
          </div>

          {#if searchResults.length > 0}
            <div class="search-results">
              {#each searchResults as result}
                <div class="search-result">
                  <span class="result-query">"{result.fields?.query}"</span>
                  <span class="result-count">{result.fields?.result_count} matches</span>
                </div>
              {/each}
            </div>
          {/if}
        </WabiCard>

        <!-- Edge Browser -->
        <WabiCard padding="md">
          {#snippet header()}
                    <div  class="section-header">
              <h2>Edge Browser</h2>
              <WabiBadge variant={edgePages.length > 0 ? 'success' : 'default'}>
                {edgePages.length > 0 ? 'Connected' : 'Disconnected'}
              </WabiBadge>
            </div>
                  {/snippet}
          
          {#if edgePages.length > 0}
            <div class="edge-pages">
              {#each edgePages as page}
                <div class="edge-page">
                  <span class="page-title">{page.title}</span>
                  <WabiButton 
                    variant="ghost" 
                    size="sm"
                    on:click={() => extractPage(page.url, 'market:research')}
                  >
                    Extract
                  </WabiButton>
                </div>
              {/each}
            </div>
          {:else}
            <p class="empty-state">Launch Edge with debugging to enable browser automation</p>
          {/if}
        </WabiCard>

        <!-- Market Research -->
        <WabiCard padding="md">
          {#snippet header()}
                    <div  class="section-header">
              <h2>Market Research</h2>
            </div>
                  {/snippet}
          
          {#if marketResearch.length > 0}
            <div class="research-list">
              {#each marketResearch as research}
                <div class="research-item">
                  <span class="research-title">{research.fields?.title}</span>
                  <span class="research-url">{research.fields?.url}</span>
                  <div class="research-concepts">
                    {#each (research.fields?._vqc_concepts || []).slice(0, 5) as concept}
                      <WabiBadge variant="info">{concept}</WabiBadge>
                    {/each}
                  </div>
                </div>
              {/each}
            </div>
          {:else}
            <p class="empty-state">No market research data yet. Extract pages to analyze.</p>
          {/if}
        </WabiCard>
      </section>
    </div>
  {/if}

  {#if error}
    <div class="error-toast">
      <span>{error}</span>
    </div>
  {/if}
</div>

<style>
  .dashboard {
    min-height: 100vh;
    padding: var(--space-3);
    background: var(--color-paper);
  }

  .dashboard-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: var(--space-3);
    padding-bottom: var(--space-2);
    border-bottom: var(--border-subtle);
  }

  .header-content h1 {
    font-size: var(--text-3xl);
    margin: 0;
  }

  .subtitle {
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    color: var(--color-ink-light);
    margin: var(--space-0) 0 0;
  }

  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 50vh;
    gap: var(--space-2);
    color: var(--color-ink-light);
  }

  .stats-row {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: var(--space-2);
    margin-bottom: var(--space-3);
  }

  .quick-context {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .context-label {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    text-transform: uppercase;
    letter-spacing: 1px;
    color: var(--color-ink-light);
  }

  .context-text {
    font-size: var(--text-base);
    line-height: var(--leading-relaxed);
    margin: 0;
  }

  .main-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-3);
    margin-top: var(--space-3);
  }

  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .section-header h2 {
    font-size: var(--text-lg);
    margin: 0;
  }

  .file-count {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    color: var(--color-ink-light);
  }

  .file-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    max-height: 500px;
    overflow-y: auto;
  }

  .file-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--space-1);
    border-radius: var(--radius-sm);
    transition: background var(--duration-fast) var(--ease-wabi);
  }

  .file-item:hover {
    background: rgba(0, 0, 0, 0.03);
  }

  .file-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .file-path {
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .file-category {
    font-size: var(--text-xs);
    color: var(--color-ink-light);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .file-score {
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    font-weight: 600;
  }

  .search-box {
    display: flex;
    gap: var(--space-1);
    margin-bottom: var(--space-2);
  }

  .search-box input {
    flex: 1;
    padding: var(--space-1);
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    border: var(--border-light);
    border-radius: var(--radius-md);
    background: rgba(255, 255, 255, 0.5);
  }

  .search-box input:focus {
    outline: none;
    border-color: var(--color-ink);
  }

  .search-results {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .search-result {
    display: flex;
    justify-content: space-between;
    padding: var(--space-1);
    background: rgba(0, 0, 0, 0.02);
    border-radius: var(--radius-sm);
  }

  .result-query {
    font-family: var(--font-mono);
    font-size: var(--text-sm);
  }

  .result-count {
    font-size: var(--text-xs);
    color: var(--color-ink-light);
  }

  .edge-pages {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .edge-page {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--space-1);
    background: rgba(0, 0, 0, 0.02);
    border-radius: var(--radius-sm);
  }

  .page-title {
    font-size: var(--text-sm);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 200px;
  }

  .research-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .research-item {
    display: flex;
    flex-direction: column;
    gap: var(--space-0);
    padding: var(--space-1);
    background: rgba(0, 0, 0, 0.02);
    border-radius: var(--radius-sm);
  }

  .research-title {
    font-weight: 500;
    font-size: var(--text-sm);
  }

  .research-url {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    color: var(--color-ink-light);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .research-concepts {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin-top: var(--space-0);
  }

  .empty-state {
    font-size: var(--text-sm);
    color: var(--color-ink-light);
    text-align: center;
    padding: var(--space-2);
  }

  .error-toast {
    position: fixed;
    bottom: var(--space-3);
    right: var(--space-3);
    background: var(--color-danger);
    color: white;
    padding: var(--space-1) var(--space-2);
    border-radius: var(--radius-md);
    font-size: var(--text-sm);
  }

  @media (max-width: 1024px) {
    .stats-row {
      grid-template-columns: repeat(2, 1fr);
    }

    .main-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
