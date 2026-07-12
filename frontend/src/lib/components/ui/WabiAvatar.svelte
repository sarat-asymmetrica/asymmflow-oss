<script lang="ts">
  
  
  interface Props {
    /**
   * Wabi-Sabi Avatar
   * Beautiful user/entity representation with fallback initials
   */
    src?: string;
    alt?: string;
    name?: string;
    size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
    status?: 'online' | 'offline' | 'busy' | 'away' | null;
  }

  let {
    src = '',
    alt = '',
    name = '',
    size = 'md',
    status = null
  }: Props = $props();
  
  let initials = $derived(name
    .split(' ')
    .map(n => n[0])
    .slice(0, 2)
    .join('')
    .toUpperCase());
  
  // Generate consistent color from name
  function stringToColor(str: string): string {
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
      hash = str.charCodeAt(i) + ((hash << 5) - hash);
    }
    const hue = hash % 360;
    return `hsl(${hue}, 25%, 45%)`;
  }
  
  let bgColor = $derived(name ? stringToColor(name) : '#57534e');
</script>

<div class="avatar {size}" class:has-status={status}>
  {#if src}
    <img {src} alt={alt || name} class="avatar-image" />
  {:else}
    <span class="avatar-initials" style="background-color: {bgColor}">
      {initials || '?'}
    </span>
  {/if}
  
  {#if status}
    <span class="avatar-status {status}"></span>
  {/if}
</div>

<style>
  .avatar {
    position: relative;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border-radius: 50%;
    overflow: hidden;
    flex-shrink: 0;
  }
  
  /* Sizes */
  .avatar.xs { width: 24px; height: 24px; }
  .avatar.sm { width: 32px; height: 32px; }
  .avatar.md { width: 40px; height: 40px; }
  .avatar.lg { width: 56px; height: 56px; }
  .avatar.xl { width: 80px; height: 80px; }
  
  .avatar-image {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
  
  .avatar-initials {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-family: Georgia, serif;
    color: #fdfbf7;
    background: #57534e;
  }
  
  .avatar.xs .avatar-initials { font-size: 10px; }
  .avatar.sm .avatar-initials { font-size: 12px; }
  .avatar.md .avatar-initials { font-size: 14px; }
  .avatar.lg .avatar-initials { font-size: 18px; }
  .avatar.xl .avatar-initials { font-size: 24px; }
  
  /* Status indicator */
  .avatar-status {
    position: absolute;
    bottom: 0;
    right: 0;
    width: 25%;
    height: 25%;
    min-width: 8px;
    min-height: 8px;
    border-radius: 50%;
    border: 2px solid #fdfbf7;
    box-sizing: content-box;
  }
  
  .avatar-status.online { background: #15803d; }
  .avatar-status.offline { background: #57534e; }
  .avatar-status.busy { background: #ef4444; }
  .avatar-status.away { background: #d97706; }
</style>
