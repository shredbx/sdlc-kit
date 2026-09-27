/**
 * Hero Background: Dot Pattern — Svelte action
 *
 * Radial dot grid with cursor-following glow.
 * Ported from hero-backgrounds-catalog.html section 2.
 *
 * @example
 * ```svelte
 * <section use:heroBgDots>...</section>
 * <section use:heroBgDots={{ dotSize: 1.5, spacing: 20 }}>...</section>
 * ```
 */

import type { Action } from 'svelte/action';

export interface HeroBgDotsParams {
  /** Dot radius in px (default: 1.2) */
  dotSize?: number;
  /** Dot spacing in px (default: 24) */
  spacing?: number;
  /** Enable cursor glow (default: true) */
  cursor?: boolean;
  /** Cursor glow radius in px (default: 400) */
  glowRadius?: number;
}

export const heroBgDots: Action<HTMLElement, HeroBgDotsParams | undefined> = (node, params = {}) => {
  const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  let config = resolveConfig(params);
  let fadeEl: HTMLDivElement | null = null;
  let glowEl: HTMLDivElement | null = null;
  let onMove: ((e: MouseEvent) => void) | null = null;
  const origPos = getComputedStyle(node).position;

  function resolveConfig(p: HeroBgDotsParams | undefined) {
    return {
      dotSize: p?.dotSize ?? 1.2,
      spacing: p?.spacing ?? 24,
      cursor: p?.cursor ?? true,
      glowRadius: p?.glowRadius ?? 400,
    };
  }

  function setup() {
    const { dotSize, spacing, cursor, glowRadius } = config;

    if (origPos === 'static') node.style.position = 'relative';
    node.style.overflow = 'hidden';

    node.style.backgroundImage = `radial-gradient(var(--dot-color, rgba(255,255,255,0.15)) ${dotSize}px, transparent ${dotSize}px)`;
    node.style.backgroundSize = `${spacing}px ${spacing}px`;

    // Radial fade
    fadeEl = document.createElement('div');
    fadeEl.setAttribute('aria-hidden', 'true');
    fadeEl.style.cssText = `
      position: absolute; inset: 0; pointer-events: none;
      background: radial-gradient(ellipse 70% 55% at 50% 50%, transparent var(--fade-start, 35%), var(--fade-color, #000) var(--fade-end, 80%));
    `;
    node.appendChild(fadeEl);

    // Cursor glow layer (skip if user prefers reduced motion)
    if (cursor && !reduceMotion) {
      glowEl = document.createElement('div');
      glowEl.setAttribute('aria-hidden', 'true');
      glowEl.style.cssText = 'position: absolute; inset: 0; pointer-events: none; z-index: 1;';
      node.appendChild(glowEl);

      onMove = (e: MouseEvent) => {
        const rect = node.getBoundingClientRect();
        const x = e.clientX - rect.left;
        const y = e.clientY - rect.top;
        glowEl!.style.background = `radial-gradient(${glowRadius}px circle at ${x}px ${y}px, var(--spot-glow, rgba(255,255,255,0.08)), transparent 60%)`;
      };
      node.addEventListener('mousemove', onMove);
    }
  }

  function teardown() {
    if (onMove) { node.removeEventListener('mousemove', onMove); onMove = null; }
    if (fadeEl) { fadeEl.remove(); fadeEl = null; }
    if (glowEl) { glowEl.remove(); glowEl = null; }
    node.style.backgroundImage = '';
    node.style.backgroundSize = '';
    if (origPos === 'static') node.style.position = '';
    node.style.overflow = '';
  }

  setup();

  return {
    update(newParams: HeroBgDotsParams | undefined) {
      teardown();
      config = resolveConfig(newParams);
      setup();
    },
    destroy() { teardown(); },
  };
};
