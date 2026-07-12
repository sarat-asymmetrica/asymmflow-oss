/**
 * Portal action — mounts node children into a target outside the current DOM tree.
 * Defaults to document.body so overlays escape stacking-context traps.
 *
 * Usage: <div use:portal>…</div>
 *        <div use:portal={{ target: '#my-portal-root' }}>…</div>
 *
 * Constitution §2.6: layered surfaces must escape stacking contexts.
 */

export interface PortalOptions {
  /** CSS selector or HTMLElement to portal into. Defaults to document.body. */
  target?: string | HTMLElement;
}

export function portal(node: HTMLElement, options: PortalOptions = {}) {
  function resolve(opts: PortalOptions): HTMLElement {
    if (!opts.target) return document.body;
    if (typeof opts.target === 'string') {
      const el = document.querySelector<HTMLElement>(opts.target);
      if (!el) {
        console.warn(`[portal] target "${opts.target}" not found — falling back to document.body`);
        return document.body;
      }
      return el;
    }
    return opts.target;
  }

  let target = resolve(options);

  // Move node into target immediately (synchronous — no flash-of-wrong-position).
  target.appendChild(node);

  return {
    update(next: PortalOptions = {}) {
      const nextTarget = resolve(next);
      if (nextTarget !== target) {
        nextTarget.appendChild(node);
        target = nextTarget;
      }
    },
    destroy() {
      // Only remove if still a child (guard against external DOM mutations).
      if (node.parentNode === target) {
        target.removeChild(node);
      }
    },
  };
}
