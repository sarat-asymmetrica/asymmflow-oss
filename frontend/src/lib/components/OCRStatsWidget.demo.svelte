<script lang="ts">
  import OCRStatsWidget from "./OCRStatsWidget.svelte";
</script>

<!-- ================================================================
     OCR STATS WIDGET DEMO
     Shows how to use the OCRStatsWidget component
     ================================================================ -->

<div class="demo-container">
  <div class="demo-header">
    <h1>OCR Stats Widget Demo</h1>
    <p>Production-ready component with wabi-sabi design aesthetic</p>
  </div>

  <div class="demo-grid">
    <!-- Default Usage -->
    <div class="demo-section">
      <h2>Default Widget</h2>
      <OCRStatsWidget />
    </div>

    <!-- In a Dashboard Layout -->
    <div class="demo-section">
      <h2>Dashboard Integration Example</h2>
      <div class="dashboard-preview">
        <div class="dashboard-card">
          <h3>Revenue Overview</h3>
          <p>Sample card</p>
        </div>
        <div class="dashboard-card">
          <OCRStatsWidget />
        </div>
        <div class="dashboard-card">
          <h3>Recent Activity</h3>
          <p>Sample card</p>
        </div>
      </div>
    </div>
  </div>

  <!-- Usage Instructions -->
  <div class="usage-section">
    <h2>Usage Instructions</h2>
    <pre><code>{`<!-- Import the component -->
<script>
  import OCRStatsWidget from "./lib/components/OCRStatsWidget.svelte";
</script>

<!-- Use it anywhere -->
<OCRStatsWidget />

<!-- That's it! Auto-refreshes every 30 seconds -->
`}</code></pre>

    <h3>Features</h3>
    <ul>
      <li>Real-time stats from backend via Wails</li>
      <li>Auto-refresh every 30 seconds</li>
      <li>Loading and error states</li>
      <li>Manual refresh button</li>
      <li>Wabi-sabi design (rice paper aesthetic)</li>
      <li>φ-based spacing (8, 13, 21, 34px)</li>
      <li>Accessible (ARIA labels, semantic HTML)</li>
      <li>Responsive (mobile-friendly)</li>
      <li>Engine distribution visualization</li>
    </ul>

    <h3>Metrics Displayed</h3>
    <ul>
      <li><strong>Documents:</strong> Total OCR documents processed</li>
      <li><strong>Confidence:</strong> Average OCR confidence percentage</li>
      <li><strong>DNA Cache:</strong> Cache hit rate (10× speedup indicator)</li>
      <li><strong>GPU Usage:</strong> GPU utilization percentage (5× speedup)</li>
      <li><strong>Efficiency Gain:</strong> Combined optimization impact</li>
      <li><strong>Engine Distribution:</strong> Visual breakdown by OCR engine</li>
    </ul>

    <h3>Backend Integration</h3>
    <p>Uses the following Wails bindings:</p>
    <pre><code>{`import { GetOCRStats } from "../../../wailsjs/go/main/DocumentsService";

// Returns:
{
  total_documents: number,
  avg_confidence: number (0-1),
  total_cost: number (USD),
  dna_cache_hits: number,
  gpu_processed: number,
  engine_distribution: [
    { Engine: string, Count: number }
  ]
}`}</code></pre>
  </div>
</div>

<style>
  .demo-container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 2rem;
    font-family: var(--font-sans, 'DM Sans', sans-serif);
  }

  .demo-header {
    text-align: center;
    margin-bottom: 3rem;
  }

  .demo-header h1 {
    font-family: var(--font-heading, Georgia, serif);
    font-size: 3rem;
    margin-bottom: 0.5rem;
  }

  .demo-header p {
    color: var(--ink-light, #666);
    font-size: 1.125rem;
  }

  .demo-grid {
    display: grid;
    gap: 3rem;
  }

  .demo-section h2 {
    font-family: var(--font-heading, Georgia, serif);
    font-size: 1.5rem;
    margin-bottom: 1rem;
  }

  .dashboard-preview {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 1.5rem;
  }

  .dashboard-card {
    background: rgba(255, 255, 255, 0.5);
    border-radius: var(--radius-xl, 32px);
    padding: 1.5rem;
  }

  .dashboard-card h3 {
    margin: 0 0 0.5rem 0;
    font-family: var(--font-heading, Georgia, serif);
  }

  .usage-section {
    margin-top: 4rem;
    padding-top: 2rem;
    border-top: 2px solid rgba(0, 0, 0, 0.08);
  }

  .usage-section h2 {
    font-family: var(--font-heading, Georgia, serif);
    font-size: 1.75rem;
    margin-bottom: 1rem;
  }

  .usage-section h3 {
    font-family: var(--font-heading, Georgia, serif);
    font-size: 1.25rem;
    margin: 2rem 0 1rem 0;
  }

  pre {
    background: rgba(0, 0, 0, 0.04);
    padding: 1.5rem;
    border-radius: var(--radius-lg, 20px);
    overflow-x: auto;
    font-family: 'Courier Prime', monospace;
    font-size: 0.875rem;
    line-height: 1.5;
  }

  code {
    font-family: 'Courier Prime', monospace;
  }

  ul {
    list-style: none;
    padding: 0;
  }

  ul li {
    padding: 0.5rem 0;
    padding-left: 1.5rem;
    position: relative;
  }

  ul li::before {
    content: "•";
    position: absolute;
    left: 0;
    color: var(--color-safe, #15803d);
    font-weight: bold;
  }

  strong {
    font-weight: 600;
    color: var(--ink, #1c1c1c);
  }
</style>
