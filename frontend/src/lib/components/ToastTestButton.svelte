<script lang="ts">
  import { toast } from '$lib/stores/toasts';

  function testSuccess() {
    console.log(' Test: Triggering SUCCESS toast');
    toast.success('Success! Everything is working.');
  }

  function testWarning() {
    console.log(' Test: Triggering WARNING toast');
    toast.warning('Warning! Something needs attention.');
  }

  function testDanger() {
    console.log(' Test: Triggering DANGER toast');
    toast.danger('Error! Something went wrong.');
  }

  function testInfo() {
    console.log(' Test: Triggering INFO toast');
    toast.info('Info: Here is some information.');
  }

  function testMultiple() {
    console.log(' Test: Triggering MULTIPLE toasts');
    toast.success('First toast!');
    setTimeout(() => toast.warning('Second toast!'), 200);
    setTimeout(() => toast.info('Third toast!'), 400);
  }
</script>

<div class="toast-test-panel">
  <h3>Toast Test Panel</h3>
  <div class="button-grid">
    <button onclick={testSuccess} class="btn-success">Success Toast</button>
    <button onclick={testWarning} class="btn-warning">Warning Toast</button>
    <button onclick={testDanger} class="btn-danger">Danger Toast</button>
    <button onclick={testInfo} class="btn-info">Info Toast</button>
    <button onclick={testMultiple} class="btn-multi">Multiple Toasts</button>
  </div>
</div>

<style>
  .toast-test-panel {
    position: fixed;
    bottom: 20px;
    left: 20px;
    background: var(--surface, #1a1a1a);
    border: 1px solid var(--border, #333);
    border-radius: 8px;
    padding: 16px;
    z-index: 10000;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  }

  h3 {
    margin: 0 0 12px;
    font-size: 14px;
    color: var(--text-primary, #fff);
  }

  .button-grid {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  button {
    padding: 8px 12px;
    border: none;
    border-radius: 4px;
    font-size: 12px;
    cursor: pointer;
    transition: opacity 0.2s;
  }

  button:hover {
    opacity: 0.8;
  }

  .btn-success {
    background: #4ade80;
    color: #000;
  }

  .btn-warning {
    background: #fbbf24;
    color: #000;
  }

  .btn-danger {
    background: #f87171;
    color: #fff;
  }

  .btn-info {
    background: #93c5fd;
    color: #000;
  }

  .btn-multi {
    background: #a78bfa;
    color: #fff;
  }
</style>
