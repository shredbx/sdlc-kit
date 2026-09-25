/**
 * Hero Background: Hero Highlight — Svelte action
 *
 * Grid pattern with cursor-following highlight glow (accent-colored).
 * Ported from hero-backgrounds-catalog.html section 5.
 *
 * @example
 * ```svelte
 * <section use:heroBgHighlight>...</section>
 * ```
 */

import type { Action } from 'svelte/action';

export interface HeroBgHighlightParams {
  /** Grid cell size in px (default: 32) */
  gridSize?: number;
  /** Glow radius in px (default: 600) */
  glowRadius?: number;
}

export const heroBgHighlight: Action<HTMLElement, HeroBgHighlightParams | undefined> = (node, params = {}) => {
  const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  let config = resolveConfig(params);
  let glowEl: HTMLDivElement | null = null;
  let onMove: ((e: MouseEvent) => void) | null = null;
  const origPos = getComputedStyle(node).position;

  function resolveConfig(p: HeroBgHighlightParams | undefined) {
    return {
      gridSize: p?.gridSize ?? 32,
      glowRadius: p?.glowRadius ?? 600,
    };
  }

  function setup() {
    const { gridSize, glowRadius } = config;

    if (origPos === 'static') node.style.position = 'relative';
    node.style.overflow = 'hidden';

    node.style.backgroundSize = `${gridSize}px ${gridSize}px`;
    node.style.backgroundImage = [
      'linear-gradient(to right, var(--grid-line, rgba(255,255,255,0.07)) 1px, transparent 1px)',
      'linear-gradient(to bottom, var(--grid-line, rgba(255,255,255,0.07)) 1px, transparent 1px)',
    ].join(',');

    // Highlight glow layer
    glowEl = document.createElement('div');
    glowEl.setAttribute('aria-hidden', 'true');
    glowEl.style.cssText = 'position: absolute; inset: 0; pointer-events: none; z-index: 1; transition: opacity 0.3s;';
    node.appendChild(glowEl);

    if (reduceMotion) return;

    onMove = (e: MouseEvent) => {
      const rect = node.getBoundingClientRect();
      const x = e.clientX - rect.left;
      const y = e.clientY - rect.top;
      glowEl!.style.background = `radial-gradient(${glowRadius}px circle at ${x}px ${y}px, var(--hl-glow, rgba(120,80,255,0.12)), transparent 50%)`;
    };
    node.addEventListener('mousemove', onMove);
  }

  function teardown() {
    if (onMove) { node.removeEventListener('mousemove', onMove); onMove = null; }
    if (glowEl) { glowEl.remove(); glowEl = null; }
    node.style.backgroundImage = '';
    node.style.backgroundSize = '';
    if (origPos === 'static') node.style.position = '';
    node.style.overflow = '';
  }

  setup();

  return {
    update(newParams: HeroBgHighlightParams | undefined) {
      teardown();
      config = resolveConfig(newParams);
      setup();
    },
    destroy() { teardown(); },
  };
};
