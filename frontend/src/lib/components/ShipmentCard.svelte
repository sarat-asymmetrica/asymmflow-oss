<script lang="ts">
    let { shipment, stageColors = {} } = $props();
</script>

<div class="card">
    <div class="header">
        <div class="status">
            <span class="dot" style={`background:${stageColors[shipment?.status] || 'var(--color-gold)'}`}></span>
            <span class="mono">{shipment?.status || 'Preparing'}</span>
        </div>
        <span class="mono">{shipment?.trackingNumber || 'No Tracking'}</span>
    </div>

    <div class="meta">
        <p class="muted">Carrier: {shipment?.carrier || 'TBD'}</p>
        <p class="muted">Items: {shipment?.items ? shipment.items.join(', ') : '—'}</p>
        <p class="muted">Delivery: {shipment?.deliveryDate ? new Date(shipment.deliveryDate).toLocaleDateString() : '—'}</p>
    </div>
</div>

<style>
    .card { background: rgba(255,255,255,0.65); border: 1px solid rgba(0,0,0,0.08); padding: 0.75rem; display: flex; flex-direction: column; gap: 0.25rem; }
    .header { display: flex; justify-content: space-between; align-items: center; }
    .status { display: flex; gap: 0.4rem; align-items: center; font-family: var(--font-mono); letter-spacing: 1px; text-transform: uppercase; font-size: 0.75rem; }
    .dot { width: 10px; height: 10px; border-radius: 50%; border: 1px solid rgba(0,0,0,0.1); }
    .mono { font-family: var(--font-mono); letter-spacing: 1px; text-transform: uppercase; }
    .muted { margin: 0; color: var(--color-ink-light); font-family: var(--font-serif); }
</style>
