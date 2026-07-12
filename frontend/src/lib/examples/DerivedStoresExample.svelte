<!--
  Derived Stores Example

  Demonstrates how to use derived stores to eliminate manual state synchronization.

  BEFORE: Manual reactive statements (bug-prone!)
  AFTER: Derived stores (automatic, bulletproof!)
-->

<script>
    import { writable } from 'svelte/store';
    import { filtered, sorted, paginated, grouped, aggregated } from '../stores/derived';

    // Sample customer data
    const customersStore = writable([
        { id: 1, name: 'Gulf Smelting Co.', status: 'Active', revenue: 45000, grade: 'A', region: 'North' },
        { id: 2, name: 'Delta Petrochemicals', status: 'Active', revenue: 32000, grade: 'B', region: 'Central' },
        { id: 3, name: 'National Petroleum Co.', status: 'Inactive', revenue: 55000, grade: 'C', region: 'South' },
        { id: 4, name: 'Highland Energy', status: 'Active', revenue: 85000, grade: 'A', region: 'North' },
        { id: 5, name: 'Coastal Shipyard', status: 'Active', revenue: 28500, grade: 'B', region: 'Central' },
        { id: 6, name: 'Demo Bank A', status: 'Pending', revenue: 12000, grade: 'D', region: 'Central' },
    ]);

    // Filter state
    const statusFilter = writable('All');
    const gradeFilter = writable('All');
    const sortBy = writable('revenue'); // 'name' | 'revenue'
    const page = writable(0);
    const pageSize = writable(3);

    // ===== DERIVED STORES (AUTOMATIC!) =====

    // Filter by status
    const statusFiltered = filtered(
        customersStore,
        statusFilter,
        (customer, filter) => filter === 'All' || customer.status === filter
    );

    // Filter by grade
    const gradeFiltered = filtered(
        statusFiltered,
        gradeFilter,
        (customer, filter) => filter === 'All' || customer.grade === filter
    );

    // Sort by selected field
    const sortedCustomers = sorted(gradeFiltered, (a, b) => {
        const field = $sortBy;
        if (field === 'revenue') {
            return b.revenue - a.revenue; // Descending
        }
        return a.name.localeCompare(b.name); // Ascending
    });

    // Paginate
    const paginatedCustomers = paginated(sortedCustomers, page, pageSize);

    // Group by region
    const customersByRegion = grouped(customersStore, (c) => c.region);

    // Aggregate stats
    const stats = aggregated(customersStore, {
        total: (customers) => customers.length,
        active: (customers) => customers.filter((c) => c.status === 'Active').length,
        totalRevenue: (customers) => customers.reduce((sum, c) => sum + c.revenue, 0),
        avgRevenue: (customers) => {
            const sum = customers.reduce((sum, c) => sum + c.revenue, 0);
            return sum / customers.length;
        },
    });

    // ===== COMPARISON: MANUAL REACTIVE (OLD WAY) =====
    /*
    // BAD: Manual state sync (bug-prone!)
    let filteredCustomers = [];
    let sortedCustomers = [];
    let paginatedCustomers = [];
    let stats = {};

    $: {
        // Filter
        filteredCustomers = $customersStore.filter(c => {
            if ($statusFilter !== 'All' && c.status !== $statusFilter) return false;
            if ($gradeFilter !== 'All' && c.grade !== $gradeFilter) return false;
            return true;
        });

        // Sort
        sortedCustomers = [...filteredCustomers].sort((a, b) => {
            if ($sortBy === 'revenue') return b.revenue - a.revenue;
            return a.name.localeCompare(b.name);
        });

        // Paginate
        const start = $page * $pageSize;
        const end = start + $pageSize;
        paginatedCustomers = sortedCustomers.slice(start, end);

        // Stats
        stats = {
            total: $customersStore.length,
            active: $customersStore.filter(c => c.status === 'Active').length,
            totalRevenue: $customersStore.reduce((sum, c) => sum + c.revenue, 0),
            avgRevenue: $customersStore.reduce((sum, c) => sum + c.revenue, 0) / $customersStore.length
        };
    }

    // PROBLEMS WITH MANUAL APPROACH:
    // 1. Lots of boilerplate code
    // 2. Easy to forget dependencies (subtle bugs!)
    // 3. No memoization (recomputes even when not needed)
    // 4. Harder to test in isolation
    */

    // Functions
    function nextPage() {
        if ($page < $paginatedCustomers.totalPages - 1) {
            page.update((n) => n + 1);
        }
    }

    function prevPage() {
        if ($page > 0) {
            page.update((n) => n - 1);
        }
    }
</script>

<div class="example">
    <h2>Derived Stores Example</h2>
    <p class="subtitle">Zero manual state synchronization - all reactive!</p>

    <!-- Stats Bar -->
    <div class="stats-bar">
        <div class="stat">
            <span class="stat-label">Total</span>
            <span class="stat-value">{$stats.total}</span>
        </div>
        <div class="stat">
            <span class="stat-label">Active</span>
            <span class="stat-value">{$stats.active}</span>
        </div>
        <div class="stat">
            <span class="stat-label">Total Revenue</span>
            <span class="stat-value">BHD {$stats.totalRevenue.toLocaleString()}</span>
        </div>
        <div class="stat">
            <span class="stat-label">Avg Revenue</span>
            <span class="stat-value">BHD {$stats.avgRevenue.toFixed(0)}</span>
        </div>
    </div>

    <!-- Filters -->
    <div class="filters">
        <label>
            Status:
            <select bind:value={$statusFilter}>
                <option value="All">All</option>
                <option value="Active">Active</option>
                <option value="Inactive">Inactive</option>
                <option value="Pending">Pending</option>
            </select>
        </label>

        <label>
            Grade:
            <select bind:value={$gradeFilter}>
                <option value="All">All</option>
                <option value="A">A</option>
                <option value="B">B</option>
                <option value="C">C</option>
                <option value="D">D</option>
            </select>
        </label>

        <label>
            Sort By:
            <select bind:value={$sortBy}>
                <option value="name">Name</option>
                <option value="revenue">Revenue</option>
            </select>
        </label>

        <label>
            Page Size:
            <select bind:value={$pageSize}>
                <option value="2">2</option>
                <option value="3">3</option>
                <option value="5">5</option>
            </select>
        </label>
    </div>

    <!-- Customer List -->
    <div class="customer-list">
        <h3>
            Customers (Page {$paginatedCustomers.page + 1} of {$paginatedCustomers.totalPages})
        </h3>

        {#each $paginatedCustomers.items as customer}
            <div class="customer-card">
                <div class="customer-header">
                    <strong>{customer.name}</strong>
                    <span class="badge grade-{customer.grade.toLowerCase()}">{customer.grade}</span>
                </div>
                <div class="customer-details">
                    <span>Status: {customer.status}</span>
                    <span>Revenue: BHD {customer.revenue.toLocaleString()}</span>
                    <span>Region: {customer.region}</span>
                </div>
            </div>
        {/each}

        <!-- Pagination -->
        <div class="pagination">
            <button onclick={prevPage} disabled={$page === 0}>Previous</button>
            <span>Page {$page + 1} of {$paginatedCustomers.totalPages}</span>
            <button onclick={nextPage} disabled={$page >= $paginatedCustomers.totalPages - 1}>
                Next
            </button>
        </div>
    </div>

    <!-- Grouped View -->
    <div class="grouped-view">
        <h3>Customers by Region</h3>
        {#each Object.entries($customersByRegion) as [region, customers]}
            <div class="region-group">
                <h4>{region} ({customers.length})</h4>
                <ul>
                    {#each customers as customer}
                        <li>{customer.name} - {customer.grade}</li>
                    {/each}
                </ul>
            </div>
        {/each}
    </div>

    <!-- Code Comparison -->
    <div class="code-comparison">
        <h3>Code Comparison</h3>
        <div class="comparison-grid">
            <div class="old-way">
                <h4>Manual Reactive (Old Way)</h4>
                <pre><code>{`$: filteredCustomers = customers.filter(c => {
    if (statusFilter !== 'All' && c.status !== statusFilter) return false;
    if (gradeFilter !== 'All' && c.grade !== gradeFilter) return false;
    return true;
});

$: sortedCustomers = [...filteredCustomers].sort((a, b) => {
    if (sortBy === 'revenue') return b.revenue - a.revenue;
    return a.name.localeCompare(b.name);
});

$: {
    const start = page * pageSize;
    const end = start + pageSize;
    paginatedCustomers = sortedCustomers.slice(start, end);
}

// PROBLEMS:
// - Lots of boilerplate
// - Easy to miss dependencies
// - No memoization
// - Harder to test`}</code></pre>
            </div>

            <div class="new-way">
                <h4>Derived Stores (New Way)</h4>
                <pre><code>{`const statusFiltered = filtered(
    customers,
    statusFilter,
    (c, f) => f === 'All' || c.status === f
);

const gradeFiltered = filtered(
    statusFiltered,
    gradeFilter,
    (c, f) => f === 'All' || c.grade === f
);

const sortedCustomers = sorted(
    gradeFiltered,
    (a, b) => sortBy === 'revenue'
        ? b.revenue - a.revenue
        : a.name.localeCompare(b.name)
);

const paginatedCustomers = paginated(
    sortedCustomers,
    page,
    pageSize
);

// BENEFITS:
// - Less code
// - Automatic reactivity
// - Composable
// - Testable in isolation`}</code></pre>
            </div>
        </div>
    </div>
</div>

<style>
    .example {
        max-width: 1200px;
        margin: 0 auto;
        padding: 2rem;
        font-family: system-ui, -apple-system, sans-serif;
    }

    h2 {
        margin: 0 0 0.5rem;
        font-size: 2rem;
        color: #1c1c1c;
    }

    .subtitle {
        margin: 0 0 2rem;
        color: #666;
        font-size: 1.1rem;
    }

    .stats-bar {
        display: grid;
        grid-template-columns: repeat(4, 1fr);
        gap: 1rem;
        margin-bottom: 2rem;
    }

    .stat {
        background: rgba(0, 0, 0, 0.03);
        padding: 1rem;
        border-radius: 8px;
        text-align: center;
    }

    .stat-label {
        display: block;
        font-size: 0.75rem;
        text-transform: uppercase;
        color: #666;
        margin-bottom: 0.5rem;
    }

    .stat-value {
        display: block;
        font-size: 1.5rem;
        font-weight: 600;
        color: #1c1c1c;
    }

    .filters {
        display: flex;
        gap: 1rem;
        margin-bottom: 2rem;
        padding: 1rem;
        background: rgba(0, 0, 0, 0.02);
        border-radius: 8px;
    }

    .filters label {
        display: flex;
        flex-direction: column;
        gap: 0.25rem;
        font-size: 0.85rem;
        color: #666;
        text-transform: uppercase;
    }

    .filters select {
        padding: 0.5rem;
        border: 1px solid #ccc;
        border-radius: 4px;
        font-size: 1rem;
    }

    .customer-list {
        margin-bottom: 2rem;
    }

    .customer-list h3 {
        margin-bottom: 1rem;
        color: #1c1c1c;
    }

    .customer-card {
        background: white;
        border: 1px solid #e0e0e0;
        border-radius: 8px;
        padding: 1rem;
        margin-bottom: 1rem;
    }

    .customer-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 0.5rem;
    }

    .customer-header strong {
        font-size: 1.1rem;
    }

    .badge {
        padding: 0.25rem 0.5rem;
        border-radius: 4px;
        font-size: 0.75rem;
        font-weight: 600;
        text-transform: uppercase;
    }

    .badge.grade-a {
        background: #d4edda;
        color: #155724;
    }
    .badge.grade-b {
        background: #fff3cd;
        color: #856404;
    }
    .badge.grade-c {
        background: #f8d7da;
        color: #721c24;
    }
    .badge.grade-d {
        background: #f5c6cb;
        color: #721c24;
    }

    .customer-details {
        display: flex;
        gap: 1rem;
        font-size: 0.9rem;
        color: #666;
    }

    .pagination {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-top: 1rem;
    }

    .pagination button {
        padding: 0.5rem 1rem;
        background: #1c1c1c;
        color: white;
        border: none;
        border-radius: 4px;
        cursor: pointer;
    }

    .pagination button:disabled {
        background: #ccc;
        cursor: not-allowed;
    }

    .grouped-view {
        margin-bottom: 2rem;
    }

    .grouped-view h3 {
        margin-bottom: 1rem;
        color: #1c1c1c;
    }

    .region-group {
        background: white;
        border: 1px solid #e0e0e0;
        border-radius: 8px;
        padding: 1rem;
        margin-bottom: 1rem;
    }

    .region-group h4 {
        margin: 0 0 0.5rem;
        color: #1c1c1c;
    }

    .region-group ul {
        margin: 0;
        padding-left: 1.5rem;
    }

    .code-comparison {
        margin-top: 3rem;
        padding-top: 2rem;
        border-top: 2px solid #e0e0e0;
    }

    .code-comparison h3 {
        margin-bottom: 1.5rem;
        color: #1c1c1c;
    }

    .comparison-grid {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 1.5rem;
    }

    .old-way,
    .new-way {
        background: #f8f9fa;
        border-radius: 8px;
        padding: 1rem;
    }

    .old-way h4 {
        color: #dc3545;
        margin: 0 0 1rem;
    }

    .new-way h4 {
        color: #28a745;
        margin: 0 0 1rem;
    }

    pre {
        margin: 0;
        overflow-x: auto;
    }

    code {
        font-family: 'Courier New', monospace;
        font-size: 0.85rem;
        line-height: 1.5;
        color: #333;
    }

    @media (max-width: 768px) {
        .stats-bar {
            grid-template-columns: repeat(2, 1fr);
        }

        .filters {
            flex-direction: column;
        }

        .comparison-grid {
            grid-template-columns: 1fr;
        }
    }
</style>
