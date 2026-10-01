/**
 * Hero Background: Grid Pattern — Svelte action
 *
 * Renders a CSS grid pattern with radial fade and optional center glow.
 * Ported from hero-backgrounds-catalog.html section 1.
 *
 * @example
 * ```svelte
 * <section use:heroBgGrid>...</section>
 * <section use:heroBgGrid={{ gridSize: 32, glow: true }}>...</section>
 * ```
 */

import type { Action } from 'svelte/action';

export interface HeroBgGridParams {
  /** Grid cell size in px (default: 40) */
  gridSize?: number;
  /** Show center glow (default: true) */
  glow?: boolean;
}

export const heroBgGrid: Action<HTMLElement, HeroBgGridParams | undefined> = (node, params = {}) => {
  let config = resolveConfig(params);
  let fadeEl: HTMLDivElement | null = null;
  let glowEl: HTMLDivElement | null = null;
  const origPos = getComputedStyle(node).position;

  function resolveConfig(p: HeroBgGridParams | undefined) {
    return {
      gridSize: p?.gridSize ?? 40,
      glow: p?.glow ?? true,
    };
  }

  function setup() {
    const { gridSize, glow } = config;

    if (origPos === 'static') node.style.position = 'relative';
    node.style.overflow = 'hidden';

    // Grid lines via background-image
    node.style.backgroundSize = `${gridSize}px ${gridSize}px`;
    node.style.backgroundImage = [
      'linear-gradient(to right, var(--grid-line, rgba(255,255,255,0.07)) 1px, transparent 1px)',
      'linear-gradient(to bottom, var(--grid-line, rgba(255,255,255,0.07)) 1px, transparent 1px)',
    ].join(',');

    // Radial fade overlay
    fadeEl = document.createElement('div');
    fadeEl.setAttribute('aria-hidden', 'true');
    fadeEl.style.cssText = `
      position: absolute; inset: 0; pointer-events: none;
      background: radial-gradient(ellipse 80% 60% at 50% 50%, transparent var(--fade-start, 30%), var(--fade-color, #000) var(--fade-end, 80%));
    `;
    node.appendChild(fadeEl);

    // Optional center glow
    if (glow) {
      glowEl = document.createElement('div');
      glowEl.setAttribute('aria-hidden', 'true');
      glowEl.style.cssText = `
        position: absolute; inset: 0; pointer-events: none; z-index: 1;
        background: radial-gradient(600px circle at 50% 50%, var(--spot-glow, rgba(255,255,255,0.08)), transparent 60%);
      `;
      node.appendChild(glowEl);
    }
  }

  function teardown() {
    if (fadeEl) { fadeEl.remove(); fadeEl = null; }
    if (glowEl) { glowEl.remove(); glowEl = null; }
    node.style.backgroundImage = '';
    node.style.backgroundSize = '';
    if (origPos === 'static') node.style.position = '';
    node.style.overflow = '';
  }

  setup();

  return {
    update(newParams: HeroBgGridParams | undefined) {
      teardown();
      config = resolveConfig(newParams);
      setup();
    },
    destroy() { teardown(); },
  };
};
