/**
 * Hero Background: Dot + Cursor Ripple — Svelte action
 *
 * Dot grid with dual-layer cursor glow + click-to-expand ring.
 * Ported from hero-backgrounds-catalog.html section 7.
 *
 * @example
 * ```svelte
 * <section use:heroBgDotRipple>...</section>
 * ```
 */

import type { Action } from 'svelte/action';

export interface HeroBgDotRippleParams {
  /** Dot spacing in px (default: 20) */
  spacing?: number;
  /** Enable cursor glow (default: true) */
  cursor?: boolean;
  /** Enable click ripple (default: true) */
  clickRipple?: boolean;
}

export const heroBgDotRipple: Action<HTMLElement, HeroBgDotRippleParams | undefined> = (node, params = {}) => {
  const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  let config = resolveConfig(params);
  let fadeEl: HTMLDivElement | null = null;
  let rippleLayer: HTMLDivElement | null = null;
  let styleEl: HTMLStyleElement | null = null;
  let onMove: ((e: MouseEvent) => void) | null = null;
  let onClick: ((e: MouseEvent) => void) | null = null;
  const origPos = getComputedStyle(node).position;

  function resolveConfig(p: HeroBgDotRippleParams | undefined) {
    return {
      spacing: p?.spacing ?? 20,
      cursor: p?.cursor ?? true,
      clickRipple: p?.clickRipple ?? true,
    };
  }

  function setup() {
    const { spacing, cursor, clickRipple } = config;

    if (origPos === 'static') node.style.position = 'relative';
    node.style.overflow = 'hidden';

    node.style.backgroundImage = `radial-gradient(var(--dot-color, rgba(255,255,255,0.15)) 1px, transparent 1px)`;
    node.style.backgroundSize = `${spacing}px ${spacing}px`;

    // Fade
    fadeEl = document.createElement('div');
    fadeEl.setAttribute('aria-hidden', 'true');
    fadeEl.style.cssText = `
      position: absolute; inset: 0; pointer-events: none;
      background: radial-gradient(ellipse 70% 55% at 50% 50%, transparent var(--fade-start, 35%), var(--fade-surface, var(--fade-color, #000)) var(--fade-end, 80%));
    `;
    node.appendChild(fadeEl);

    // Cursor glow + ripple container
    rippleLayer = document.createElement('div');
    rippleLayer.setAttribute('aria-hidden', 'true');
    rippleLayer.style.cssText = 'position: absolute; inset: 0; pointer-events: none; z-index: 1;';
    node.appendChild(rippleLayer);

    if (cursor && !reduceMotion) {
      onMove = (e: MouseEvent) => {
        const rect = node.getBoundingClientRect();
        const x = e.clientX - rect.left;
        const y = e.clientY - rect.top;
        rippleLayer!.style.background = [
          `radial-gradient(300px circle at ${x}px ${y}px, rgba(167,139,250,0.08), transparent 50%)`,
          `radial-gradient(120px circle at ${x}px ${y}px, rgba(167,139,250,0.15), transparent 40%)`,
        ].join(',');
      };
      node.addEventListener('mousemove', onMove);
    }

    if (clickRipple && !reduceMotion) {
      // Inject keyframes once
      if (!document.getElementById('sbx-ripple-expand')) {
        styleEl = document.createElement('style');
        styleEl.id = 'sbx-ripple-expand';
        styleEl.textContent = `
          @keyframes sbxRippleExpand {
            0% { width: 0; height: 0; opacity: 1; }
            100% { width: 400px; height: 400px; opacity: 0; }
          }
        `;
        document.head.appendChild(styleEl);
      }

      onClick = (e: MouseEvent) => {
        const rect = node.getBoundingClientRect();
        const x = e.clientX - rect.left;
        const y = e.clientY - rect.top;
        const ring = document.createElement('div');
        ring.style.cssText = `
          position: absolute; left: ${x}px; top: ${y}px;
          width: 0; height: 0; border-radius: 50%;
          border: 1.5px solid rgba(167,139,250,0.4);
          transform: translate(-50%, -50%);
          pointer-events: none; z-index: 2;
          animation: sbxRippleExpand 1s ease-out forwards;
        `;
        node.appendChild(ring);
        setTimeout(() => ring.remove(), 1000);
      };
      node.addEventListener('click', onClick);
    }
  }

  function teardown() {
    if (onMove) { node.removeEventListener('mousemove', onMove); onMove = null; }
    if (onClick) { node.removeEventListener('click', onClick); onClick = null; }
    if (fadeEl) { fadeEl.remove(); fadeEl = null; }
    if (rippleLayer) { rippleLayer.remove(); rippleLayer = null; }
    // Don't remove shared keyframe style — other instances may use it
    node.style.backgroundImage = '';
    node.style.backgroundSize = '';
    if (origPos === 'static') node.style.position = '';
    node.style.overflow = '';
  }

  setup();

  return {
    update(newParams: HeroBgDotRippleParams | undefined) {
      teardown();
      config = resolveConfig(newParams);
      setup();
    },
    destroy() { teardown(); },
  };
};
