<script lang="ts">
    import { stopPropagation, createBubbler } from 'svelte/legacy';

    const bubble = createBubbler();
    import { onMount, createEventDispatcher } from "svelte";
    import { fade } from "svelte/transition";
	import {
	    ListCustomers } from "../../../wailsjs/go/main/App";
import { CreateCustomer, DeleteCustomer } from "../../../wailsjs/go/main/CRMService";
    import { toast } from "$lib/stores/toasts";
    import { confirm } from "$lib/stores/confirm";
    import WabiSpinner from "../components/ui/WabiSpinner.svelte";
    import CursorFollower from "../components/CursorFollower.svelte";
    import { main, time, crm } from "../../../wailsjs/go/models";

    
    interface Props {
        // Props
        embedded?: boolean;
    }

    let { embedded = false }: Props = $props();

    const dispatch = createEventDispatcher();

    // Fallback nav (only used when not embedded)
    const navigate = (id) => {
        if (embedded) {
            dispatch('select', { id });
        } else {
            window.location.hash = `#/customers/${id}`;
        }
    };

    // State
    let customers = $state([]);
    let loading = $state(true);
    let showCreate = $state(false);
    let creating = $state(false);
    let filters = $state({ search: "", type: "All" });

    const customerTypes = [
        { code: "EC", label: "End Customer" },
        { code: "CO", label: "Consultant" },
        { code: "EP", label: "Engineering/EPC" },
        { code: "IR", label: "Intl Reseller" },
        { code: "NR", label: "Natl Reseller" },
        { code: "PB", label: "Plant Builder" },
        { code: "SI", label: "System Integrator" },
        { code: "SP", label: "Service Provider" },
        { code: "PH", label: "Acme Instrumentation" },
    ];

    // Payment terms options per Bahrain business rules
    const paymentTermsOptions = [
        { value: "cash", label: "Cash" },
        { value: "pdc", label: "PDC (Post-Dated Cheque)" },
        { value: "net30", label: "Net 30" },
        { value: "net60", label: "Net 60" },
        { value: "net90", label: "Net 90" },
    ];

    // Customer status options
    const statusOptions = ["Active", "Inactive", "Blacklisted"];

	function emptyCustomerForm() {
	    return {
	    // Identity Section
	    customer_id: "",
	    business_name: "",
        trading_name: "",
        cr_number: "",
        trn: "",
        short_code: "",

	    // Classification Section
	    customer_type: "",
	    payment_grade: "",
	    status: "",

        // Contact Section
        primary_phone: "",
        mobile_number: "",
        primary_email: "",
        website: "",

	    // Address Section (Bahrain format)
	    building_flat: "",
	    road_street: "",
	    block: "",
	    area: "",
	    city: "",
	    country: "",
	    address_line1: "", // Combined for backend

	    // Financial Section
	    payment_terms: "",
	    payment_terms_days: 0,
	    credit_limit: "",

        // Business Details
        industry: "",
        relation_years: 0,

	    // Contacts (nested)
	    contacts: [],
	    };
	}

	let newCustomer = $state(emptyCustomerForm());

    async function loadCustomers() {
        loading = true;
        try {
            const res = await ListCustomers(500, 0); // simplistic pagination
            customers = res || [];
        } catch (e) {
            toast.danger("Failed to load customers");
        } finally {
            loading = false;
        }
    }

    async function handleCreate() {
        if (!newCustomer.business_name) {
            toast.warning("Company Name is required");
            return;
        }
        // Auto-generate customer code if not provided
        if (!newCustomer.customer_id) {
            const prefix = newCustomer.business_name.substring(0, 4).toUpperCase().replace(/[^A-Z]/g, '');
            const suffix = Date.now().toString().slice(-6) + Math.random().toString(36).slice(-2);
            newCustomer.customer_id = `CUST-${prefix}${suffix}`;
        }
        creating = true;
        try {
            // Compose address_line1 from Bahrain address parts
            const addressParts = [
                newCustomer.building_flat,
                newCustomer.road_street,
                newCustomer.block ? `Block ${newCustomer.block}` : "",
                newCustomer.area,
            ].filter(Boolean);
            newCustomer.address_line1 = addressParts.join(", ");

            // Map payment terms to days
	            const termsMap = {
	                cash: 0,
	                pdc: 7,
	                net30: 30,
	                net60: 60,
	                net90: 90,
	            };
	            newCustomer.payment_terms_days =
	                termsMap[newCustomer.payment_terms] || 0;

            // Create proper CustomerMaster object
            const customerData = crm.CustomerMaster.createFrom({
                id: "",
                created_at: new time.Time(),
                updated_at: new time.Time(),
                version: 0,
                created_by: "system",
                customer_id: newCustomer.customer_id,
                customer_code: newCustomer.short_code || newCustomer.customer_id,
                customer_type: newCustomer.customer_type,
                business_name: newCustomer.business_name,
                short_code: newCustomer.short_code || "",
                trading_name: newCustomer.trading_name || "",
                cr_number: newCustomer.cr_number || "",
	                status: newCustomer.status || "Active",
                primary_phone: newCustomer.primary_phone || "",
                primary_email: newCustomer.primary_email || "",
                website: newCustomer.website || "",
                address_line1: newCustomer.address_line1,
                city: newCustomer.city,
                country: newCustomer.country,
                trn: newCustomer.trn || "",
                mobile_number: newCustomer.mobile_number || "",
                industry: newCustomer.industry || "",
                relation_years: newCustomer.relation_years || 0,
	                payment_grade: newCustomer.payment_grade || "C",
	                customer_grade: newCustomer.payment_grade || "C",
	                payment_terms_days: newCustomer.payment_terms_days,
                avg_payment_days: 0,
                dispute_count: 0,
                total_orders_value: 0,
                total_orders_count: 0,
                avg_order_value: 0,
                ar_risk_tier: "low",
                outstanding_bhd: 0,
                overdue_days: 0,
	                credit_limit_bhd: Number(newCustomer.credit_limit) || 0,
                is_credit_blocked: false,
                requires_prepayment: false,
                has_abb_competition: false,
                is_emergency_only: false,
            });

            await CreateCustomer(customerData);
            toast.success("Customer created");
            showCreate = false;

	            newCustomer = emptyCustomerForm();
	            await loadCustomers();
        } catch (e) {
            console.error(e);
            toast.danger("Create failed: " + String(e));
	        } finally {
	            creating = false;
	        }
	    }

	    async function handleDeleteCustomer(customer, event) {
	        event?.stopPropagation();
	        const name = customer?.business_name || 'this customer';
	        if (!customer?.id || !(await confirm.ask({
	            title: 'Delete Customer',
	            message: `Delete ${name}? This cannot be undone.`,
	            confirmLabel: 'Delete',
	            variant: 'danger'
	        }))) {
	            return;
	        }
	        try {
	            await DeleteCustomer(customer.id);
	            toast.success(`Deleted customer: ${name}`);
	            await loadCustomers();
	        } catch (err) {
	            toast.danger(`Delete failed: ${String(err)}`);
	        }
	    }

    let filteredCustomers = $derived(customers.filter((c) => {
        const matchSearch =
            !filters.search ||
            c.business_name
                .toLowerCase()
                .includes(filters.search.toLowerCase());
        const matchType =
            filters.type === "All" || c.customer_type === filters.type;
        return matchSearch && matchType;
    }));

    onMount(loadCustomers);
</script>

<div class="page">
    <header class="header">
        <div class="header-content">
            <h1>Customers.</h1>
            <p class="subtitle">Directory & Relationships</p>
        </div>
        <button class="btn-primary" onclick={() => { newCustomer = emptyCustomerForm(); showCreate = true; }}>
            + New Customer
        </button>
    </header>

    <div class="layout-split">
        <!-- Sidebar Filters -->
        <aside class="sidebar">
            <div class="search-box">
                <input
                    type="text"
                    placeholder="Search customers..."
                    bind:value={filters.search}
                    class="input-clean"
                />
            </div>

            <div class="filter-group">
                <h3>Type</h3>
                <button
                    class="filter-item"
                    class:active={filters.type === "All"}
                    onclick={() => (filters.type = "All")}
                >
                    All Types
                </button>
                {#each customerTypes as t}
                    <button
                        class="filter-item"
                        class:active={filters.type === t.code}
                        onclick={() => (filters.type = t.code)}
                    >
                        {t.label}
                    </button>
                {/each}
            </div>
        </aside>

        <!-- Main List -->
        <main class="main-content">
            {#if loading}
                <div class="loading">
                    <WabiSpinner size="lg" tempo="calm" />
                </div>
            {:else if filteredCustomers.length === 0}
                <div class="empty-state">No customers yet — add one to begin quoting.</div>
            {:else}
                <div class="customer-grid">
                    {#each filteredCustomers as c}
                        <div
                            class="customer-card"
                            role="button"
                            tabindex="0"
                            onclick={() => navigate(c.id)}
                            onkeydown={(event) =>
                                (event.key === "Enter" || event.key === " ") &&
                                navigate(c.id)}
                        >
	                            <div class="card-head">
	                                <span class="customer-name"
	                                    >{c.business_name}</span
	                                >
	                                <div class="card-actions">
	                                    <span class="grade-badge {c.payment_grade}"
	                                        >Grade {c.payment_grade || "B"}</span
	                                    >
	                                    <button
	                                        class="delete-link"
	                                        type="button"
	                                        aria-label="Delete customer"
	                                        onclick={stopPropagation((event) => handleDeleteCustomer(c, event))}
	                                    >
	                                        Delete
	                                    </button>
	                                </div>
	                            </div>
                            <div class="card-meta">
                                <span class="type-tag">{c.customer_type}</span>
                                {#if c.city}
                                    <span class="loc">{c.city}</span>
                                {/if}
                            </div>
                            <!-- Future metrics like Total Revenue could go here -->
                        </div>
                    {/each}
                </div>
            {/if}
        </main>
    </div>
</div>

{#if showCreate}
    <div
        class="modal-backdrop"
        transition:fade
        role="button"
        tabindex="0"
        onclick={() => (showCreate = false)}
        onkeydown={(event) =>
            (event.key === "Enter" || event.key === " ") &&
            (showCreate = false)}
    >
        <CursorFollower />
        <div class="modal" role="presentation" tabindex="-1" onclick={stopPropagation(bubble('click'))} onkeydown={stopPropagation(bubble('keydown'))}>
            <h3>New Customer</h3>
            <div class="form-scroll">
                <!-- IDENTITY SECTION -->
                <div class="section-header">Identity</div>
                <div class="row">
                    <div class="form-group third">
                        <label for="customer-code">Customer Code</label>
                        <input
                            id="customer-code"
                            type="text"
                            bind:value={newCustomer.customer_id}
                            class="input-clean"
                            placeholder="CUST-001"
                        />
                    </div>
                    <div class="form-group third">
                        <label for="customer-short-code">Short Code</label>
                        <input
                            id="customer-short-code"
                            type="text"
                            bind:value={newCustomer.short_code}
                            class="input-clean"
                            placeholder="CUST"
                        />
                    </div>
                    <div class="form-group third">
                        <label for="customer-cr-number">CR Number</label>
                        <input
                            id="customer-cr-number"
                            type="text"
                            bind:value={newCustomer.cr_number}
                            class="input-clean"
                            placeholder="CR-12345"
                        />
                    </div>
                </div>
                <div class="form-group">
                    <label for="customer-company-name">Company Name *</label>
                    <input
                        id="customer-company-name"
                        type="text"
                        bind:value={newCustomer.business_name}
                        class="input-clean"
                    />
                </div>
                <div class="row">
                    <div class="form-group half">
                        <label for="customer-trading-name">Trading Name</label>
                        <input
                            id="customer-trading-name"
                            type="text"
                            bind:value={newCustomer.trading_name}
                            class="input-clean"
                            placeholder="DBA name"
                        />
                    </div>
                    <div class="form-group half">
                        <label for="customer-trn">VAT / TRN</label>
                        <input
                            id="customer-trn"
                            type="text"
                            bind:value={newCustomer.trn}
                            class="input-clean"
                            placeholder="100000000000003"
                        />
                    </div>
                </div>

                <!-- CLASSIFICATION SECTION -->
                <div class="section-header">Classification</div>
                <div class="row">
                    <div class="form-group third">
                        <label for="customer-type">Type</label>
                        <select
                            id="customer-type"
                            bind:value={newCustomer.customer_type}
                            class="input-clean"
	                        >
	                            <option value="" disabled>Select type</option>
	                            {#each customerTypes as t}
	                                <option value={t.code}>{t.label}</option>
                            {/each}
                        </select>
                    </div>
                    <div class="form-group third">
                        <label for="customer-grade">Grade</label>
                        <select
                            id="customer-grade"
                            bind:value={newCustomer.payment_grade}
                            class="input-clean"
	                        >
	                            <option value="" disabled>Select grade</option>
	                            <option>A</option><option>B</option><option
                                >C</option
                            ><option>D</option>
                        </select>
                    </div>
                    <div class="form-group third">
                        <label for="customer-status">Status</label>
                        <select
                            id="customer-status"
                            bind:value={newCustomer.status}
                            class="input-clean"
	                        >
	                            <option value="" disabled>Select status</option>
	                            {#each statusOptions as s}
                                <option>{s}</option>
                            {/each}
                        </select>
                    </div>
                </div>
                <div class="form-group">
                    <label for="customer-industry">Industry</label>
                    <input
                        id="customer-industry"
                        type="text"
                        bind:value={newCustomer.industry}
                        class="input-clean"
                        placeholder="e.g. Oil & Gas"
                    />
                </div>

                <!-- CONTACT SECTION -->
                <div class="section-header">Contact</div>
                <div class="row">
                    <div class="form-group quarter">
                        <label for="customer-primary-phone">Primary Phone</label>
                        <input
                            id="customer-primary-phone"
                            type="tel"
                            bind:value={newCustomer.primary_phone}
                            class="input-clean"
                            placeholder="+973 1234 5678"
                        />
                    </div>
                    <div class="form-group quarter">
                        <label for="customer-mobile-number">Mobile Number</label>
                        <input
                            id="customer-mobile-number"
                            type="tel"
                            bind:value={newCustomer.mobile_number}
                            class="input-clean"
                            placeholder="+973 3XXX XXXX"
                        />
                    </div>
                    <div class="form-group quarter">
                        <label for="customer-primary-email">Primary Email</label>
                        <input
                            id="customer-primary-email"
                            type="email"
                            bind:value={newCustomer.primary_email}
                            class="input-clean"
                            placeholder="contact@company.bh"
                        />
                    </div>
                    <div class="form-group quarter">
                        <label for="customer-website">Website</label>
                        <input
                            id="customer-website"
                            type="url"
                            bind:value={newCustomer.website}
                            class="input-clean"
                            placeholder="www.company.bh"
                        />
                    </div>
                </div>

                <!-- ADDRESS SECTION (Bahrain) -->
                <div class="section-header">Address (Bahrain)</div>
                <div class="row">
                    <div class="form-group half">
                        <label for="customer-building-flat">Building / Flat</label>
                        <input
                            id="customer-building-flat"
                            type="text"
                            bind:value={newCustomer.building_flat}
                            class="input-clean"
                            placeholder="Building 123"
                        />
                    </div>
                    <div class="form-group half">
                        <label for="customer-road-street">Road / Street</label>
                        <input
                            id="customer-road-street"
                            type="text"
                            bind:value={newCustomer.road_street}
                            class="input-clean"
                            placeholder="Road 456"
                        />
                    </div>
                </div>
                <div class="row">
                    <div class="form-group quarter">
                        <label for="customer-block">Block</label>
                        <input
                            id="customer-block"
                            type="text"
                            bind:value={newCustomer.block}
                            class="input-clean"
                            placeholder="789"
                        />
                    </div>
                    <div class="form-group quarter">
                        <label for="customer-area">Area</label>
                        <input
                            id="customer-area"
                            type="text"
                            bind:value={newCustomer.area}
                            class="input-clean"
                            placeholder="Seef"
                        />
                    </div>
                    <div class="form-group quarter">
                        <label for="customer-city">City</label>
                        <input
                            id="customer-city"
                            type="text"
                            bind:value={newCustomer.city}
                            class="input-clean"
                        />
                    </div>
                    <div class="form-group quarter">
                        <label for="customer-country">Country</label>
                        <input
                            id="customer-country"
                            type="text"
                            bind:value={newCustomer.country}
                            class="input-clean"
                        />
                    </div>
                </div>

                <!-- FINANCIAL SECTION -->
                <div class="section-header">Financial</div>
                <div class="row">
                    <div class="form-group half">
                        <label for="customer-payment-terms">Payment Terms</label>
                        <select
                            id="customer-payment-terms"
                            bind:value={newCustomer.payment_terms}
                            class="input-clean"
	                        >
	                            <option value="" disabled>Select terms</option>
	                            {#each paymentTermsOptions as pt}
                                <option value={pt.value}>{pt.label}</option>
                            {/each}
                        </select>
                    </div>
                    <div class="form-group half">
                        <label for="customer-credit-limit">Credit Limit (BHD)</label>
                        <input
                            id="customer-credit-limit"
                            type="number"
                            bind:value={newCustomer.credit_limit}
                            class="input-clean"
                            placeholder="0"
                        />
                    </div>
                </div>
            </div>

            <div class="modal-actions">
                <button class="btn-ghost" onclick={() => (showCreate = false)}
                    >Cancel</button
                >
                <button
                    class="btn-primary"
                    onclick={handleCreate}
                    disabled={creating}
                >
                    {creating ? "Creating..." : "Create Customer"}
                </button>
            </div>
        </div>
    </div>
{/if}

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
        margin-bottom: var(--space-4);
        flex-shrink: 0;
    }
    h1 {
        font-size: var(--text-3xl);
        font-weight: var(--font-weight-light);
        margin: 0;
        letter-spacing: -0.01em;
    }
    .subtitle {
        color: var(--ink-faint);
        margin-top: var(--space-1);
        font-size: var(--text-sm);
    }

    .btn-primary {
        background: var(--ink);
        color: var(--paper);
        border: none;
        padding: 10px 20px;
        border-radius: var(--radius-pill);
        cursor: pointer;
        transition: all var(--duration-normal) var(--ease-out);
    }
    .btn-primary:hover {
        transform: translateY(-2px) scale(1.02);
        box-shadow: var(--shadow-md);
    }

    .layout-split {
        display: grid;
        grid-template-columns: 200px 1fr;
        gap: var(--space-4);
        flex: 1;
        min-height: 0;
    }

    .sidebar {
        border-right: 1px solid var(--border-subtle);
        padding-right: var(--space-3);
        display: flex;
        flex-direction: column;
        gap: var(--space-4);
        overflow-y: auto;
    }

    .input-clean {
        width: 100%;
        padding: 8px 12px;
        border: 1px solid var(--border-medium);
        border-radius: var(--radius-md);
        font-family: var(--font-sans);
        box-sizing: border-box;
        font-size: 14px;
    }

    .filter-group h3 {
        font-size: 11px;
        text-transform: uppercase;
        color: var(--ink-light);
        margin-bottom: 8px;
    }
    .filter-item {
        display: block;
        width: 100%;
        text-align: left;
        padding: 8px 12px;
        margin-bottom: 2px;
        border: none;
        background: transparent;
        border-radius: var(--radius-md);
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
    }

    .main-content {
        overflow-y: auto;
        padding-right: 4px;
    }
    .loading,
    .empty-state {
        display: flex;
        align-items: center;
        justify-content: center;
        height: 100%;
        color: var(--ink-light);
    }

    .customer-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
        gap: var(--space-4);
    }

    .customer-card {
        background: var(--paper-subtle);
        padding: var(--space-5);
        border-radius: var(--radius-lg);
        border: 1px solid var(--border-subtle);
        cursor: pointer;
        transition:
            transform 0.2s,
            box-shadow 0.2s;
        display: flex;
        flex-direction: column;
        gap: 8px;
    }
    .customer-card:hover {
        transform: translateY(-2px);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
        border-color: var(--border-medium);
    }

	    .card-head {
	        display: flex;
	        justify-content: space-between;
	        align-items: flex-start;
	        gap: 8px;
	    }
	    .card-actions {
	        display: flex;
	        align-items: center;
	        gap: 8px;
	        flex-shrink: 0;
	    }
	    .customer-name {
	        font-weight: 500;
	        font-size: 16px;
	    }
	    .delete-link {
	        border: 1px solid rgba(185, 28, 28, 0.24);
	        background: rgba(254, 242, 242, 0.88);
	        color: #991b1b;
	        border-radius: 6px;
	        padding: 4px 8px;
	        font-size: 11px;
	        font-weight: 700;
	        cursor: pointer;
	    }
	    .delete-link:hover {
	        background: #fee2e2;
	        border-color: rgba(185, 28, 28, 0.42);
	    }

	    .grade-badge {
        font-size: 10px;
        padding: 2px 6px;
        border-radius: 4px;
        font-weight: 600;
        text-transform: uppercase;
    }
    .grade-badge.A {
        background: #dcfce7;
        color: #166534;
    }
    .grade-badge.B {
        background: #dbeafe;
        color: #1e40af;
    }
    .grade-badge.C {
        background: #fef9c3;
        color: #854d0e;
    }
    .grade-badge.D {
        background: #fee2e2;
        color: #991b1b;
    }

    .card-meta {
        font-size: 12px;
        color: var(--ink-light);
        display: flex;
        gap: 8px;
    }
    .type-tag {
        background: var(--paper);
        border: 1px solid var(--border-medium);
        padding: 1px 6px;
        border-radius: 4px;
    }

    /* Modal */
    .modal-backdrop {
        position: fixed;
        inset: 0;
        background: rgba(0, 0, 0, 0.6);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
        backdrop-filter: blur(4px);
    }
    .modal {
        background: var(--paper);
        padding: 24px;
        border-radius: var(--radius-xl);
        width: 720px;
        max-width: 95%;
        max-height: 90vh;
        display: flex;
        flex-direction: column;
        box-shadow: var(--shadow-xl);
    }
    .modal h3 {
        margin: 0 0 16px 0;
        font-weight: 400;
        font-size: 22px;
        font-family: var(--font-heading);
    }
    .form-scroll {
        flex: 1;
        overflow-y: auto;
        padding-right: 8px;
        max-height: 60vh;
    }

    /* Section Headers */
    .section-header {
        font-size: 11px;
        text-transform: uppercase;
        letter-spacing: 0.08em;
        color: var(--ink-light);
        font-weight: 500;
        margin-top: 16px;
        margin-bottom: 10px;
        padding-bottom: 4px;
        border-bottom: 1px solid var(--border-subtle);
    }
    .section-header:first-child {
        margin-top: 0;
    }

    .form-group {
        margin-bottom: 12px;
    }
    .form-group label {
        display: block;
        font-size: 10px;
        text-transform: uppercase;
        color: var(--ink-faint);
        margin-bottom: 3px;
        letter-spacing: 0.02em;
    }

    .row {
        display: flex;
        gap: 12px;
    }
    .half {
        flex: 1;
    }
    .third {
        flex: 1;
        min-width: 0;
    }
    .quarter {
        flex: 1;
        min-width: 0;
    }

    .modal-actions {
        display: flex;
        justify-content: flex-end;
        gap: 12px;
        margin-top: 16px;
        padding-top: 16px;
        border-top: 1px solid var(--border-subtle);
    }
    .btn-ghost {
        background: transparent;
        border: none;
        cursor: pointer;
        color: var(--ink-light);
        font-size: 14px;
        padding: 8px 16px;
    }
    .btn-ghost:hover {
        color: var(--ink);
    }
</style>
