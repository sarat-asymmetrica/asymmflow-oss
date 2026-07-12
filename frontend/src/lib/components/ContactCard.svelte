<script lang="ts">
    let { contact = {} } = $props();
</script>

<div class="contact-card">
    <div class="avatar">{contact.name ? contact.name[0] : '?'}</div>
    <div class="info">
        <div class="name">{contact.name}</div>
        <div class="role">{contact.role}</div>
        {#if contact.email}
            <div class="detail">{contact.email}</div>
        {/if}
        {#if contact.phone}
            <div class="detail">{contact.phone}</div>
        {/if}
    </div>
</div>

<style>
    .contact-card { display: flex; gap: 1rem; align-items: center; padding: 0.5rem; background: rgba(255,255,255,0.5); border: 1px solid rgba(0,0,0,0.05); }
    .avatar { width: 40px; height: 40px; background: var(--color-ink); color: var(--color-paper); display: flex; align-items: center; justify-content: center; font-family: var(--font-serif); font-size: 1.2rem; border-radius: 50%; }
    .info { display: flex; flex-direction: column; }
    .name { font-family: var(--font-serif); font-weight: bold; font-size: 0.9rem; }
    .role { font-family: var(--font-mono); font-size: 0.7rem; text-transform: uppercase; color: var(--color-ink-light); }
    .detail { font-family: var(--font-mono); font-size: 0.7rem; color: var(--color-ink-light); }
</style>
