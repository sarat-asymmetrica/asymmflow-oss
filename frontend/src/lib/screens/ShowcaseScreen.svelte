<script lang="ts">
  import Button from '$lib/components/ui/Button.svelte';
  import Card from '$lib/components/ui/Card.svelte';
  import ActionProposalCard from '$lib/components/ui/ActionProposalCard.svelte';
  import EvidenceSourceList from '$lib/components/ui/EvidenceSourceList.svelte';
  import KpiStatusStrip from '$lib/components/ui/KpiStatusStrip.svelte';
  import Table from '$lib/components/ui/Table.svelte';
  import Tabs from '$lib/components/ui/Tabs.svelte';
  import type { Tab } from '$lib/types/components';

  // Tab data for demos
  const tabs1: Tab[] = [
    { id: 'inbox', label: 'Inbox' },
    { id: 'opportunities', label: 'Opportunities' },
    { id: 'casting', label: 'Casting & Offers' }
  ];
  const tabs2: Tab[] = [
    { id: 'orders', label: 'Orders Hub' },
    { id: 'delivery', label: 'Delivery' },
    { id: 'inventory', label: 'Inventory' },
    { id: 'supplies', label: 'Supplies' }
  ];

  // State for tabs demo
  let activeTab1 = $state('inbox');
  let activeTab2 = $state('orders');
  let demoButtonState = $state('No demo action selected');
  let proposalReviewState = $state('Review queue ready');

  const productKpis = [
    { label: 'Attention', value: 'BHD 42.8K', meta: 'BHD 18.2K overdue', status: 'review' },
    { label: 'Posting', value: '3 missing', meta: '2 drafts', status: 'review' },
    { label: 'Bank Match', value: '4 open', meta: 'BHD 6.1K', status: 'review' },
    { label: 'Evidence Pack', value: '17 items', meta: '5 follow-ups', status: 'ready' },
  ];

  const productSources = [
    { source_type: 'receivables', label: 'Receivables', required: 14, present: 9, missing: 5, confidence: 0.64, status: 'review', priority: 'high' },
    { source_type: 'banking', label: 'Bank Match', required: 8, present: 6, missing: 2, confidence: 0.75, status: 'review', priority: 'medium' },
    { source_type: 'traceability', label: 'Invoice Links', required: 11, present: 11, missing: 0, confidence: 1, status: 'ready', priority: 'low' },
  ];

  const productProposal = {
    source_type: 'cashflow evidence',
    label: 'Draft missing receivables follow-up',
    reason: 'Aging exposure and missing evidence indicate review before the next evidence-pack export.',
    priority: 'high',
    required_deterministic_service: 'FollowUpTaskService',
  };

  // Theme toggle
  function toggleTheme() {
    const html = document.documentElement;
    const currentTheme = html.getAttribute('data-theme');
    const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
    html.setAttribute('data-theme', newTheme);
    localStorage.setItem('theme', newTheme);
  }
</script>

<div class="showcase">
  <button class="btn btn-primary theme-toggle" onclick={toggleTheme}>
    Toggle Dark Mode
  </button>

  <header class="showcase-header">
    <h1 class="showcase-title">Enterprise ERP/CRM Design System</h1>
    <p class="showcase-subtitle">Component Showcase - Apple-level polish × Bloomberg-level data density</p>
  </header>

  <!-- COLORS -->
  <section class="component-section">
    <h2 class="component-section-title">Colors</h2>
    <p class="component-section-desc">Brand colors and semantic palette</p>
    
    <div class="component-demo">
      <h3 class="section-title" style="margin-bottom: 16px;">Brand Colors</h3>
      <div class="color-swatch">
        <div class="color-box" style="background: #2F2DFF;"></div>
        <div>
          <div style="font-weight: 600; font-size: 13px;">Deep Indigo</div>
          <div class="meta">#2F2DFF</div>
        </div>
      </div>
      <div class="color-swatch">
        <div class="color-box" style="background: #2624D9;"></div>
        <div>
          <div style="font-weight: 600; font-size: 13px;">Indigo Hover</div>
          <div class="meta">#2624D9</div>
        </div>
      </div>
      <div class="color-swatch">
        <div class="color-box" style="background: #1E1CB3;"></div>
        <div>
          <div style="font-weight: 600; font-size: 13px;">Indigo Pressed</div>
          <div class="meta">#1E1CB3</div>
        </div>
      </div>
    </div>

    <div class="component-demo">
      <h3 class="section-title" style="margin-bottom: 16px;">Semantic Colors (Light Mode)</h3>
      <div class="color-swatch">
        <div class="color-box" style="background: #F7F8FB;"></div>
        <div>
          <div style="font-weight: 600; font-size: 13px;">Background</div>
          <div class="meta">#F7F8FB</div>
        </div>
      </div>
      <div class="color-swatch">
        <div class="color-box" style="background: #FFFFFF;"></div>
        <div>
          <div style="font-weight: 600; font-size: 13px;">Surface</div>
          <div class="meta">#FFFFFF</div>
        </div>
      </div>
      <div class="color-swatch">
        <div class="color-box" style="background: #E6E8F0;"></div>
        <div>
          <div style="font-weight: 600; font-size: 13px;">Border</div>
          <div class="meta">#E6E8F0</div>
        </div>
      </div>
    </div>
  </section>

  <!-- SPACING -->
  <section class="component-section">
    <h2 class="component-section-title">Spacing System</h2>
    <p class="component-section-desc">Critical spacing tokens for maximum density</p>
    
    <div class="component-demo">
      <div class="spacing-demo">
        <div class="spacing-box" style="width: 16px; height: 40px;">16px</div>
        <span class="text-secondary">Page Padding</span>
      </div>
      <div class="spacing-demo">
        <div class="spacing-box" style="width: 12px; height: 40px;">12px</div>
        <span class="text-secondary">Card Padding / Grid Gap</span>
      </div>
      <div class="spacing-demo">
        <div class="spacing-box" style="width: 100px; height: 42px;">42px</div>
        <span class="text-secondary">Table Row Height</span>
      </div>
      <div class="spacing-demo">
        <div class="spacing-box" style="width: 100px; height: 36px;">36px</div>
        <span class="text-secondary">Tab Height (max)</span>
      </div>
    </div>
  </section>

  <!-- TYPOGRAPHY -->
  <section class="component-section">
    <h2 class="component-section-title">Typography</h2>
    <p class="component-section-desc">Type scale and hierarchy</p>
    
    <div class="component-demo">
      <div style="margin-bottom: 20px;">
        <div class="page-title">Page Title - 22px Semibold</div>
        <div class="meta">Used for main page headings</div>
      </div>
      <div style="margin-bottom: 20px;">
        <div class="section-title">Section Title - 16px Semibold</div>
        <div class="meta">Used for card headings and sections</div>
      </div>
      <div style="margin-bottom: 20px;">
        <div style="font-size: 14px;">Table Text - 14px Regular</div>
        <div class="meta">Used for table cells and body text</div>
      </div>
      <div style="margin-bottom: 20px;">
        <div class="label">Label - 12px Medium Uppercase</div>
        <div class="meta">Used for labels and table headers</div>
      </div>
      <div>
        <div class="meta">Meta Text - 11px Regular</div>
        <div class="meta">Used for timestamps and secondary info</div>
      </div>
    </div>
  </section>

  <!-- BUTTONS -->
  <section class="component-section">
    <h2 class="component-section-title">Buttons</h2>
    <p class="component-section-desc">Primary, secondary, and ghost button variants</p>
    
    <div class="component-demo">
      <div style="display: flex; gap: 12px; align-items: center; margin-bottom: 12px;">
        <Button variant="primary" on:click={() => (demoButtonState = 'Primary demo action')}>Primary Button</Button>
        <Button variant="secondary" on:click={() => (demoButtonState = 'Secondary demo action')}>Secondary Button</Button>
        <Button variant="ghost" on:click={() => (demoButtonState = 'Ghost demo action')}>Ghost Button</Button>
      </div>
      <div style="display: flex; gap: 12px; align-items: center;">
        <Button variant="primary" disabled on:click={() => (demoButtonState = 'Disabled primary demo action')}>Disabled Primary</Button>
        <Button variant="secondary" disabled on:click={() => (demoButtonState = 'Disabled secondary demo action')}>Disabled Secondary</Button>
      </div>
      <p class="meta" style="margin-top: 10px;">{demoButtonState}</p>
    </div>
  </section>

  <!-- KPI CARDS -->
  <section class="component-section">
    <h2 class="component-section-title">KPI Cards</h2>
    <p class="component-section-desc">Compact dashboard metrics (120-140px height)</p>
    
    <div class="component-demo">
      <div class="demo-kpi-grid">
        <Card kpi accent title="Revenue" value="$2.45M" meta="YTD, +12% from target" />
        <Card kpi accent title="Cashflow" value="$850K" meta="Current Month, 85% received" />
        <Card kpi accent title="Productivity" value="92%" meta="Team Avg, +4% vs last month" />
        <Card kpi accent title="Alerts" value="15 Active" meta="3 Critical, 12 Warnings" />
      </div>
    </div>
  </section>

  <!-- PRODUCT OPERATOR COMPONENTS -->
  <section class="component-section">
    <h2 class="component-section-title">Product Operator Components</h2>
    <p class="component-section-desc">Reusable command-center primitives for evidence, approvals, and review queues</p>

    <div class="component-demo product-demo">
      <KpiStatusStrip items={productKpis} />
      <EvidenceSourceList sources={productSources} />
      <div class="demo-proposal-list">
        <ActionProposalCard
          proposal={productProposal}
          reviewLabel={proposalReviewState}
          hasReview
          onApprove={() => (proposalReviewState = 'Approved')}
          onNeedsInput={() => (proposalReviewState = 'Needs input')}
          onReject={() => (proposalReviewState = 'Rejected')}
        />
      </div>
    </div>
  </section>

  <!-- TABS -->
  <section class="component-section">
    <h2 class="component-section-title">Tabs</h2>
    <p class="component-section-desc">Horizontal navigation tabs (36px height)</p>
    
    <div class="component-demo">
      <h3 class="section-title" style="margin-bottom: 16px;">Underline Style</h3>
      <Tabs
        tabs={tabs1}
        activeTab={activeTab1}
        variant="underline"
        on:change={(e) => activeTab1 = e.detail}
      />
    </div>

    <div class="component-demo">
      <h3 class="section-title" style="margin-bottom: 16px;">Pill Style</h3>
      <Tabs
        tabs={tabs2}
        activeTab={activeTab2}
        variant="pill"
        on:change={(e) => activeTab2 = e.detail}
      />
    </div>
  </section>

  <!-- BADGES -->
  <section class="component-section">
    <h2 class="component-section-title">Badges</h2>
    <p class="component-section-desc">Status indicators and labels</p>
    
    <div class="component-demo">
      <div style="display: flex; gap: 8px; flex-wrap: wrap;">
        <span class="badge badge-indigo">Prospect</span>
        <span class="badge badge-indigo">Qualified</span>
        <span class="badge badge-indigo">Proposal</span>
        <span class="badge badge-indigo">Negotiation</span>
        <span class="badge badge-neutral">Pending</span>
        <span class="badge badge-neutral">Archived</span>
      </div>
    </div>
  </section>

  <!-- TABLES -->
  <section class="component-section">
    <h2 class="component-section-title">Data Tables</h2>
    <p class="component-section-desc">Dense tables with 42px rows, sticky headers, no zebra striping</p>
    
    <div class="component-demo">
      <div class="demo-table-container">
        <Table>
          {#snippet header()}
                    <tr >
                <th>Opportunity Name</th>
                <th>Company</th>
                <th>Stage</th>
                <th class="align-right">Amount</th>
                <th>Owner</th>
                <th>Last Updated</th>
            </tr>
                  {/snippet}
          <tr>
            <td>Enterprise Cloud Migration</td>
            <td>TechGlobal Inc.</td>
            <td><span class="badge badge-indigo">Prospect</span></td>
            <td class="align-right">$150,000</td>
            <td>Sarah Chen</td>
            <td class="text-muted">Today, 10:45 AM</td>
          </tr>
          <tr>
            <td>AI Platform License - Q3</td>
            <td>Innovate Financial</td>
            <td><span class="badge badge-indigo">Qualified</span></td>
            <td class="align-right">$750,000</td>
            <td>David Miller</td>
            <td class="text-muted">Yesterday</td>
          </tr>
          <tr>
            <td>Security Audit Services</td>
            <td>SecureBank Corp</td>
            <td><span class="badge badge-indigo">Proposal</span></td>
            <td class="align-right">$185,000</td>
            <td>Sarah Chen</td>
            <td class="text-muted">Today, 9:35 PM</td>
          </tr>
          <tr>
            <td>Database Optimization</td>
            <td>DataFlow Systems</td>
            <td><span class="badge badge-indigo">Negotiation</span></td>
            <td class="align-right">$250,000</td>
            <td>David Miller</td>
            <td class="text-muted">Yesterday</td>
          </tr>
          <tr>
            <td>Enterprise Cloud Migration</td>
            <td>TechGlobal Inc.</td>
            <td><span class="badge badge-indigo">Prospect</span></td>
            <td class="align-right">$150,000</td>
            <td>Sarah Chen</td>
            <td class="text-muted">Today, 10:45 AM</td>
          </tr>
          <tr>
            <td>Training & Development</td>
            <td>EduTech Partners</td>
            <td><span class="badge badge-indigo">Qualified</span></td>
            <td class="align-right">$95,000</td>
            <td>Alex Johnson</td>
            <td class="text-muted">2 days ago</td>
          </tr>
        </Table>
      </div>
    </div>
  </section>

  <!-- FORMS -->
  <section class="component-section">
    <h2 class="component-section-title">Form Inputs</h2>
    <p class="component-section-desc">Text inputs with indigo focus states</p>
    
    <div class="component-demo">
      <div style="max-width: 400px;">
        <div style="margin-bottom: 16px;">
          <label class="label" for="showcase-company-name" style="display: block; margin-bottom: 6px;">Company Name</label>
          <input id="showcase-company-name" type="text" class="input" placeholder="Enter company name">
        </div>
        <div style="margin-bottom: 16px;">
          <label class="label" for="showcase-email-address" style="display: block; margin-bottom: 6px;">Email Address</label>
          <input id="showcase-email-address" type="email" class="input" placeholder="user@example.com">
        </div>
        <div>
          <label class="label" for="showcase-description" style="display: block; margin-bottom: 6px;">Description</label>
          <input id="showcase-description" type="text" class="input" placeholder="Optional description">
        </div>
      </div>
    </div>
  </section>

  <!-- CARDS -->
  <section class="component-section">
    <h2 class="component-section-title">Standard Cards</h2>
    <p class="component-section-desc">Content cards with soft shadows</p>
    
    <div class="component-demo">
      <div class="grid grid-3">
        <Card>
          <div class="section-title" style="margin-bottom: 8px;">Basic Card</div>
          <p class="text-secondary" style="font-size: 14px;">
            Standard card with 12px padding, 10px border radius, and soft shadow.
          </p>
        </Card>
        
        <Card accent>
          <div class="section-title" style="margin-bottom: 8px;">Accented Card</div>
          <p class="text-secondary" style="font-size: 14px;">
            Card with deep indigo accent border on the left edge.
          </p>
        </Card>
        
        <Card elevated>
          <div class="section-title" style="margin-bottom: 8px;">Elevated Card</div>
          <p class="text-secondary" style="font-size: 14px;">
            Card with elevated surface background for subtle distinction.
          </p>
        </Card>
      </div>
    </div>
  </section>

  <!-- FOOTER -->
  <footer style="text-align: center; padding: 40px 0; border-top: 1px solid var(--border); margin-top: 60px;">
    <p class="text-muted" style="margin-bottom: 8px;">
      Enterprise ERP/CRM Design System v1.0
    </p>
    <p class="meta">
      Apple-level polish × Bloomberg-level data density
    </p>
    <p class="meta" style="margin-top: 12px;">
      Om Lokah Samastah Sukhino Bhavantu
    </p>
  </footer>
</div>

<style>
  /* Showcase specific styles */
  .showcase {
    max-width: 1400px;
    margin: 0 auto;
    padding: 40px 20px;
    background: var(--bg-base); /* Ensure background is applied */
    min-height: 100vh;
  }

  .showcase-header {
    text-align: center;
    margin-bottom: 60px;
    padding-bottom: 30px;
    border-bottom: 1px solid var(--border);
  }

  .showcase-title {
    font-size: 36px;
    font-weight: 700;
    color: var(--text-primary);
    margin-bottom: 12px;
  }

  .showcase-subtitle {
    font-size: 18px;
    color: var(--text-secondary);
  }

  .component-section {
    margin-bottom: 60px;
  }

  .component-section-title {
    font-size: 24px;
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: 8px;
  }

  .component-section-desc {
    font-size: 14px;
    color: var(--text-muted);
    margin-bottom: 24px;
  }

  .component-demo {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--border-radius);
    padding: 24px;
    margin-bottom: 16px;
  }

  .theme-toggle {
    position: fixed;
    top: 20px;
    right: 20px;
    z-index: 1000;
  }

  /* Demo-specific layouts */
  .demo-kpi-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    gap: var(--grid-gap);
  }

  .product-demo {
    display: grid;
    gap: 1px;
    overflow: hidden;
    padding: 0;
  }

  .demo-proposal-list {
    display: grid;
    gap: 1px;
    background: var(--border);
    border-top: 1px solid var(--border);
  }

  .demo-table-container {
    max-height: 400px;
    overflow-y: auto;
  }

  .color-swatch {
    display: inline-flex;
    align-items: center;
    gap: 12px;
    margin-right: 20px;
    margin-bottom: 12px;
  }

  .color-box {
    width: 60px;
    height: 40px;
    border-radius: 6px;
    border: 1px solid var(--border);
  }

  .spacing-demo {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 12px;
  }

  .spacing-box {
    background: var(--brand-indigo-tint-medium);
    border: 1px dashed var(--brand-indigo);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 11px;
    color: var(--brand-indigo);
    font-weight: 600;
  }
</style>
