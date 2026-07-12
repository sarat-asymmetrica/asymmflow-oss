/**
 * Dismiss-on-outside-click for popovers (Dropdown, DatePicker, Tooltip).
 * Usage: <div use:clickOutside={() => (open = false)}>
 */
export function clickOutside(node: HTMLElement, handler: () => void) {
  let currentHandler = handler;

  function onPointerDown(e: PointerEvent) {
    if (!node.contains(e.target as Node)) currentHandler();
  }

  document.addEventListener('pointerdown', onPointerDown, true);

  return {
    update(next: () => void) {
      currentHandler = next;
    },
    destroy() {
      document.removeEventListener('pointerdown', onPointerDown, true);
    },
  };
}
