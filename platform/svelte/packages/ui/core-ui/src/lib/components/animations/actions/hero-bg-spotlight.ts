/**
 * Hero Background: Cursor Spotlight — Svelte action
 *
 * Grid pattern with dual-layer cursor-following glow.
 * Ported from hero-backgrounds-catalog.html section 3.
 *
 * @example
 * ```svelte
 * <section use:heroBgSpotlight>...</section>
 * ```
 */

import type { Action } from 'svelte/action';

export interface HeroBgSpotlightParams {
  /** Grid cell size in px (default: 32) */
  gridSize?: number;
  /** Outer glow radius in px (default: 700) */
  outerRadius?: number;
  /** Inner glow radius in px (default: 300) */
  innerRadius?: number;
}

export const heroBgSpotlight: Action<HTMLElement, HeroBgSpotlightParams | undefined> = (node, params = {}) => {
  const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  let config = resolveConfig(params);
  let fadeEl: HTMLDivElement | null = null;
  let spotEl: HTMLDivElement | null = null;
  let onMove: ((e: MouseEvent) => void) | null = null;
  const origPos = getComputedStyle(node).position;

  function resolveConfig(p: HeroBgSpotlightParams | undefined) {
    return {
      gridSize: p?.gridSize ?? 32,
      outerRadius: p?.outerRadius ?? 700,
      innerRadius: p?.innerRadius ?? 300,
    };
  }

  function setup() {
    const { gridSize, outerRadius, innerRadius } = config;

    if (origPos === 'static') node.style.position = 'relative';
    node.style.overflow = 'hidden';

    node.style.backgroundSize = `${gridSize}px ${gridSize}px`;
    node.style.backgroundImage = [
      'linear-gradient(to right, var(--grid-line, rgba(255,255,255,0.07)) 1px, transparent 1px)',
      'linear-gradient(to bottom, var(--grid-line, rgba(255,255,255,0.07)) 1px, transparent 1px)',
    ].join(',');

    // Fade
    fadeEl = document.createElement('div');
    fadeEl.setAttribute('aria-hidden', 'true');
    fadeEl.style.cssText = `
      position: absolute; inset: 0; pointer-events: none;
      background: radial-gradient(ellipse 70% 55% at 50% 50%, transparent var(--fade-start, 30%), var(--fade-surface, var(--fade-color, #000)) var(--fade-end, 80%));
    `;
    node.appendChild(fadeEl);

    // Spotlight layer
    spotEl = document.createElement('div');
    spotEl.setAttribute('aria-hidden', 'true');
    spotEl.style.cssText = 'position: absolute; inset: 0; pointer-events: none; z-index: 1; transition: opacity 0.3s;';
    node.appendChild(spotEl);

    if (reduceMotion) return;

    onMove = (e: MouseEvent) => {
      const rect = node.getBoundingClientRect();
      const x = e.clientX - rect.left;
      const y = e.clientY - rect.top;
      spotEl!.style.background = [
        `radial-gradient(${outerRadius}px circle at ${x}px ${y}px, var(--spot-glow-strong, rgba(255,255,255,0.15)), transparent 50%)`,
        `radial-gradient(${innerRadius}px circle at ${x}px ${y}px, var(--spot-glow, rgba(255,255,255,0.08)), transparent 40%)`,
      ].join(',');
    };
    node.addEventListener('mousemove', onMove);
  }

  function teardown() {
    if (onMove) { node.removeEventListener('mousemove', onMove); onMove = null; }
    if (fadeEl) { fadeEl.remove(); fadeEl = null; }
    if (spotEl) { spotEl.remove(); spotEl = null; }
    node.style.backgroundImage = '';
    node.style.backgroundSize = '';
    if (origPos === 'static') node.style.position = '';
    node.style.overflow = '';
  }

  setup();

  return {
    update(newParams: HeroBgSpotlightParams | undefined) {
      teardown();
      config = resolveConfig(newParams);
      setup();
    },
    destroy() { teardown(); },
  };
};
