/**
 * Ambient Shimmer — barely perceptible drifting glow
 *
 * Very subtle, same-tone blurred masses that drift slowly like gas.
 * Stays within 40% brightness zone — never brighter than the surface,
 * never contrasting with background. Just adds life without attention.
 *
 * No cursor tracking. No bright spots. No color contrast.
 *
 * @example
 * ```svelte
 * <section use:ambientShimmer>...</section>
 * <footer use:ambientShimmer={{ opacity: 0.02, speed: 'slow' }}>...</footer>
 * ```
 */

import type { Action } from 'svelte/action';

export interface AmbientShimmerParams {
  /** Base opacity of blobs (default: 0.04, range 0.01-0.08) */
  opacity?: number;
  /** Animation speed preset (default: 'normal') */
  speed?: 'slow' | 'normal' | 'fast';
  /** Blur radius in px (default: 180) */
  blur?: number;
  /** Number of blobs (default: 3, max 5) */
  count?: number;
}

let shimmerCounter = 0;

const SPEED_MAP = {
  slow: { min: 25, max: 40 },
  normal: { min: 16, max: 28 },
  fast: { min: 10, max: 18 },
};

export const ambientShimmer: Action<HTMLElement, AmbientShimmerParams | undefined> = (node, params) => {
  const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  let config = resolveConfig(params);
  let layer: HTMLDivElement | null = null;
  let styleEl: HTMLStyleElement | null = null;
  const origPos = getComputedStyle(node).position;

  function resolveConfig(p: AmbientShimmerParams | undefined) {
    return {
      opacity: Math.min(0.08, Math.max(0.01, p?.opacity ?? 0.04)),
      speed: p?.speed ?? 'normal',
      blur: p?.blur ?? 180,
      count: Math.min(5, Math.max(2, p?.count ?? 3)),
    };
  }

  function setup() {
    const { opacity, speed, blur, count } = config;
    const prefix = `sbx-shimmer-${++shimmerCounter}`;
    const speeds = SPEED_MAP[speed];

    if (origPos === 'static') node.style.position = 'relative';
    node.style.overflow = 'hidden';

    if (!reduceMotion) {
      styleEl = document.createElement('style');
      styleEl.id = prefix;
      let keyframes = '';
      for (let i = 0; i < count; i++) {
        const dx1 = 30 + Math.random() * 60 * (Math.random() > 0.5 ? 1 : -1);
        const dy1 = 20 + Math.random() * 40 * (Math.random() > 0.5 ? 1 : -1);
        const s = 1 + Math.random() * 0.1;
        keyframes += `@keyframes ${prefix}-d${i} { 0% { transform: translate(0,0) scale(1); } 50% { transform: translate(${dx1}px,${dy1}px) scale(${s}); } 100% { transform: translate(${-dx1 * 0.6}px,${-dy1 * 0.4}px) scale(1); } }\n`;
      }
      styleEl.textContent = keyframes;
      document.head.appendChild(styleEl);
    }

    layer = document.createElement('div');
    layer.setAttribute('aria-hidden', 'true');
    layer.style.cssText = 'position: absolute; inset: 0; pointer-events: none; z-index: 1; overflow: hidden;';

    const positions = [
      { top: '-15%', left: '-10%' },
      { bottom: '-20%', right: '-10%' },
      { top: '25%', right: '15%' },
      { bottom: '10%', left: '20%' },
      { top: '50%', left: '50%' },
    ];

    for (let i = 0; i < count; i++) {
      const blob = document.createElement('div');
      const size = 300 + Math.random() * 300;
      const dur = speeds.min + Math.random() * (speeds.max - speeds.min);
      const delay = i * 3;
      const pos = positions[i % positions.length];
      const posCSS = Object.entries(pos).map(([k, v]) => `${k}: ${v};`).join(' ');

      blob.style.cssText = `
        position: absolute; border-radius: 50%;
        filter: blur(${blur}px);
        width: ${size}px; height: ${size}px;
        background: currentColor;
        opacity: ${opacity};
        ${posCSS}
        ${!reduceMotion ? `animation: ${prefix}-d${i} ${dur}s ease-in-out ${delay}s infinite;` : ''}
      `;
      layer.appendChild(blob);
    }

    node.appendChild(layer);
  }

  function teardown() {
    if (layer) { layer.remove(); layer = null; }
    if (styleEl) { styleEl.remove(); styleEl = null; }
    if (origPos === 'static') node.style.position = '';
    node.style.overflow = '';
  }

  setup();

  return {
    update(newParams: AmbientShimmerParams | undefined) {
      teardown();
      config = resolveConfig(newParams);
      setup();
    },
    destroy() { teardown(); },
  };
};
