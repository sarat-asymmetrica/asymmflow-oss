<script lang="ts">
  /**
   * Design System Component Showcase
   * Tests Button, Card, and KPICard components
   * Philosophy: Apple-level polish × Bloomberg-level data density
   */
  import Button from './Button.svelte';
  import Card from './Card.svelte';
  import KPICard from './KPICard.svelte';

  let loading = $state(false);

  function handleClick() {
    console.log('Button clicked!');
  }

  function simulateLoading() {
    loading = true;
    setTimeout(() => {
      loading = false;
    }, 2000);
  }
</script>

<div class="showcase-container">
  <div class="showcase-header">
    <h1 class="page-title">Design System Components</h1>
    <p class="text-secondary">Enterprise ERP/CRM Design System - Button, Card, and KPICard</p>
  </div>

  <!-- Button Component Section -->
  <section class="section">
    <h2 class="section-title">Button Component</h2>

    <div class="demo-grid">
      <!-- Variants -->
      <div class="demo-block">
        <h3 class="label">Variants</h3>
        <div class="button-row">
          <Button variant="primary" on:click={handleClick}>Primary Button</Button>
          <Button variant="secondary" on:click={handleClick}>Secondary Button</Button>
          <Button variant="ghost" on:click={handleClick}>Ghost Button</Button>
          <Button variant="danger" on:click={handleClick}>Danger Button</Button>
        </div>
      </div>

      <!-- Sizes -->
      <div class="demo-block">
        <h3 class="label">Sizes</h3>
        <div class="button-row">
          <Button size="sm" on:click={handleClick}>Small</Button>
          <Button size="md" on:click={handleClick}>Medium</Button>
          <Button size="lg" on:click={handleClick}>Large</Button>
        </div>
      </div>

      <!-- States -->
      <div class="demo-block">
        <h3 class="label">States</h3>
        <div class="button-row">
          <Button on:click={handleClick}>Normal</Button>
          <Button disabled on:click={handleClick}>Disabled</Button>
          <Button loading={loading} on:click={simulateLoading}>
            {loading ? 'Loading...' : 'Click to Load'}
          </Button>
        </div>
      </div>

      <!-- Full Width -->
      <div class="demo-block">
        <h3 class="label">Full Width</h3>
        <Button fullWidth on:click={handleClick}>Full Width Button</Button>
      </div>
    </div>
  </section>

  <!-- Card Component Section -->
  <section class="section">
    <h2 class="section-title">Card Component</h2>

    <div class="card-grid">
      <!-- Default Card -->
      <Card variant="default" padding="md">
        <div class="card-content-demo">
          <h3 class="label">Default Card</h3>
          <p class="text-primary">This is a standard card with default styling.</p>
          <p class="meta">Shadow: sm, Background: surface</p>
        </div>
      </Card>

      <!-- Elevated Card -->
      <Card variant="elevated" padding="md">
        <div class="card-content-demo">
          <h3 class="label">Elevated Card</h3>
          <p class="text-primary">This card has elevated styling with higher shadow.</p>
          <p class="meta">Shadow: md, Background: surface-elevated</p>
        </div>
      </Card>

      <!-- Accent Card -->
      <Card variant="accent" padding="md">
        <div class="card-content-demo">
          <h3 class="label">Accent Card</h3>
          <p class="text-primary">This card features a 3px indigo left border accent.</p>
          <p class="meta">Border-left: 3px solid brand-indigo</p>
        </div>
      </Card>

      <!-- Hoverable Card -->
      <Card variant="default" padding="md" hoverable on:click={handleClick}>
        <div class="card-content-demo">
          <h3 class="label">Hoverable Card</h3>
          <p class="text-primary">This card is interactive and responds to hover/click.</p>
          <p class="meta">Cursor: pointer, Hover shadow elevation</p>
        </div>
      </Card>

      <!-- Different Padding Sizes -->
      <Card variant="default" padding="sm">
        <div class="card-content-demo">
          <h3 class="label">Small Padding</h3>
          <p class="text-primary">padding="sm"</p>
        </div>
      </Card>

      <Card variant="default" padding="lg">
        <div class="card-content-demo">
          <h3 class="label">Large Padding</h3>
          <p class="text-primary">padding="lg"</p>
        </div>
      </Card>
    </div>
  </section>

  <!-- KPI Card Component Section -->
  <section class="section">
    <h2 class="section-title">KPI Card Component</h2>

    <div class="kpi-grid">
      <!-- Revenue KPI -->
      <KPICard
        label="Total Revenue"
        value="$2,450,000"
        meta="YTD Performance, +12.5%"
        trend="up"
      />

      <!-- Opportunities KPI -->
      <KPICard
        label="Active Opportunities"
        value="34"
        meta="Pipeline Count"
      />

      <!-- Conversion KPI -->
      <KPICard
        label="Conversion Rate"
        value="68.2%"
        meta="Last 30 Days, +5.2%"
        trend="up"
      />

      <!-- Cash Runway KPI -->
      <KPICard
        label="Cash Runway"
        value="18 Months"
        meta="Based on Burn Rate, -2 months"
        trend="down"
      />

      <!-- No Trend -->
      <KPICard
        label="Active Customers"
        value="127"
        meta="Total Count"
      />

      <!-- Large Number -->
      <KPICard
        label="Processed Invoices"
        value="1,847"
        meta="This Quarter, +234"
        trend="up"
      />
    </div>
  </section>

  <!-- Design Tokens Reference -->
  <section class="section">
    <h2 class="section-title">Design Tokens Reference</h2>

    <Card variant="accent" padding="lg">
      <div class="tokens-grid">
        <div class="token-section">
          <h4 class="label">Colors</h4>
          <div class="token-item">
            <div class="color-swatch" style="background: var(--brand-indigo)"></div>
            <span class="meta">--brand-indigo: #2F2DFF</span>
          </div>
          <div class="token-item">
            <div class="color-swatch" style="background: var(--text-primary)"></div>
            <span class="meta">--text-primary: #0E1020</span>
          </div>
          <div class="token-item">
            <div class="color-swatch" style="background: var(--text-secondary)"></div>
            <span class="meta">--text-secondary: #5B5F7A</span>
          </div>
        </div>

        <div class="token-section">
          <h4 class="label">Spacing</h4>
          <div class="meta">--page-padding: 16px</div>
          <div class="meta">--card-padding: 12px</div>
          <div class="meta">--grid-gap: 12px</div>
          <div class="meta">--section-spacing: 16px</div>
        </div>

        <div class="token-section">
          <h4 class="label">Typography</h4>
          <div class="meta">--page-title-size: 22px</div>
          <div class="meta">--section-title-size: 16px</div>
          <div class="meta">--table-text-size: 14px</div>
          <div class="meta">--label-size: 12px</div>
        </div>

        <div class="token-section">
          <h4 class="label">Borders & Radius</h4>
          <div class="meta">--border-radius: 10px</div>
          <div class="meta">--border-radius-lg: 12px</div>
          <div class="meta">--border-radius-sm: 8px</div>
        </div>
      </div>
    </Card>
  </section>

  <!-- Philosophy Section -->
  <section class="section">
    <Card variant="elevated" padding="lg">
      <h3 class="label" style="margin-bottom: 12px;">Design Philosophy</h3>
      <p class="text-primary" style="margin-bottom: 12px; line-height: 1.5;">
        <strong>Apple-level polish × Bloomberg-level data density</strong>
      </p>
      <ul class="philosophy-list">
        <li class="text-secondary">
          <strong>Information Density:</strong> Maximum data on screen, compact cards (≤160px), tight spacing
        </li>
        <li class="text-secondary">
          <strong>Consistency:</strong> Components look identical across ALL modules
        </li>
        <li class="text-secondary">
          <strong>Calm Confidence:</strong> Neutral grays, soft elevation, 150-200ms transitions
        </li>
        <li class="text-secondary">
          <strong>Purposeful Indigo:</strong> Deep indigo (#2F2DFF) reserved for primary actions and brand moments
        </li>
      </ul>
    </Card>
  </section>
</div>

<style>
  .showcase-container {
    padding: var(--page-padding);
    background: var(--bg-base);
    min-height: 100vh;
  }

  .showcase-header {
    margin-bottom: 32px;
  }

  .section {
    margin-bottom: var(--section-spacing);
  }

  .demo-grid {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .demo-block {
    background: var(--surface);
    padding: 16px;
    border-radius: var(--border-radius);
    box-shadow: var(--shadow-sm);
  }

  .button-row {
    display: flex;
    gap: 12px;
    flex-wrap: wrap;
    margin-top: 12px;
  }

  .card-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: var(--grid-gap);
  }

  .card-content-demo {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .kpi-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
    gap: var(--grid-gap);
  }

  .tokens-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 24px;
  }

  .token-section {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .token-item {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .color-swatch {
    width: 24px;
    height: 24px;
    border-radius: 4px;
    border: 1px solid var(--border);
  }

  .philosophy-list {
    list-style: none;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .philosophy-list li {
    padding-left: 20px;
    position: relative;
    line-height: 1.5;
  }

  .philosophy-list li::before {
    content: '•';
    position: absolute;
    left: 0;
    color: var(--brand-indigo);
    font-weight: 700;
  }
</style>
