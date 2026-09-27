/**
 * Hero Background: Gradient Mesh + Cursor — Svelte action
 *
 * Three animated floating gradient blobs with cursor-following glow.
 * This is the hero-page variant of gradient-mesh with specific blob
 * colors/sizes from the prototype. Uses CSS animations (not the
 * gradient-mesh action's dynamically generated blobs).
 *
 * Ported from hero-backgrounds-catalog.html section 8.
 *
 * @example
 * ```svelte
 * <section use:heroBgMesh>...</section>
 * ```
 */

import type { Action } from 'svelte/action';

export interface HeroBgMeshParams {
  /** Enable cursor glow (default: true) */
  cursor?: boolean;
  /** Cursor glow radius in px (default: 400) */
  glowRadius?: number;
}

let meshCounter = 0;

export const heroBgMesh: Action<HTMLElement, HeroBgMeshParams | undefined> = (node, params = {}) => {
  const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  let config = resolveConfig(params);
  let meshLayer: HTMLDivElement | null = null;
  let cursorEl: HTMLDivElement | null = null;
  let styleEl: HTMLStyleElement | null = null;
  let onMove: ((e: MouseEvent) => void) | null = null;
  const origPos = getComputedStyle(node).position;

  function resolveConfig(p: HeroBgMeshParams | undefined) {
    return {
      cursor: p?.cursor ?? true,
      glowRadius: p?.glowRadius ?? 400,
    };
  }

  function setup() {
    const { cursor, glowRadius } = config;
    const prefix = `sbx-mesh-hero-${++meshCounter}`;
    const shouldAnimate = !reduceMotion;

    if (origPos === 'static') node.style.position = 'relative';
    node.style.overflow = 'hidden';

    // Keyframes for blob movement
    if (shouldAnimate) {
      styleEl = document.createElement('style');
      styleEl.id = prefix;
      styleEl.textContent = `
        @keyframes ${prefix}-f1 { 0% { transform: translate(0,0) scale(1); } 100% { transform: translate(80px,40px) scale(1.15); } }
        @keyframes ${prefix}-f2 { 0% { transform: translate(0,0) scale(1); } 100% { transform: translate(-60px,-30px) scale(1.1); } }
        @keyframes ${prefix}-f3 { 0% { transform: translate(0,0) scale(1); } 100% { transform: translate(40px,-50px) scale(1.2); } }
      `;
      document.head.appendChild(styleEl);
    }

    // Mesh blob container
    meshLayer = document.createElement('div');
    meshLayer.setAttribute('aria-hidden', 'true');
    meshLayer.style.cssText = 'position: absolute; inset: 0; pointer-events: none; z-index: 1; overflow: hidden;';

    const blobs = [
      { w: 500, h: 500, bg: 'hsla(260,60%,50%,0.12)', top: '-10%', left: '-5%', anim: `${prefix}-f1 12s ease-in-out infinite alternate` },
      { w: 400, h: 400, bg: 'hsla(210,70%,50%,0.08)', bottom: '-15%', right: '-5%', anim: `${prefix}-f2 10s ease-in-out 2s infinite alternate` },
      { w: 300, h: 300, bg: 'hsla(330,60%,50%,0.06)', top: '30%', right: '20%', anim: `${prefix}-f3 14s ease-in-out 4s infinite alternate` },
    ];

    blobs.forEach(b => {
      const blob = document.createElement('div');
      blob.style.cssText = `
        position: absolute; border-radius: 50%; filter: blur(120px); opacity: 0.5;
        width: ${b.w}px; height: ${b.h}px; background: ${b.bg};
        ${b.top ? `top: ${b.top};` : ''} ${b.bottom ? `bottom: ${b.bottom};` : ''}
        ${b.left ? `left: ${b.left};` : ''} ${b.right ? `right: ${b.right};` : ''}
        ${shouldAnimate ? `animation: ${b.anim};` : ''}
      `;
      meshLayer!.appendChild(blob);
    });

    node.appendChild(meshLayer);

    // Cursor glow
    if (cursor) {
      cursorEl = document.createElement('div');
      cursorEl.setAttribute('aria-hidden', 'true');
      cursorEl.style.cssText = 'position: absolute; inset: 0; pointer-events: none; z-index: 2;';
      node.appendChild(cursorEl);

      onMove = (e: MouseEvent) => {
        const rect = node.getBoundingClientRect();
        const x = e.clientX - rect.left;
        const y = e.clientY - rect.top;
        cursorEl!.style.background = `radial-gradient(${glowRadius}px circle at ${x}px ${y}px, var(--spot-glow-strong, rgba(255,255,255,0.15)), transparent 50%)`;
      };
      node.addEventListener('mousemove', onMove);
    }
  }

  function teardown() {
    if (onMove) { node.removeEventListener('mousemove', onMove); onMove = null; }
    if (meshLayer) { meshLayer.remove(); meshLayer = null; }
    if (cursorEl) { cursorEl.remove(); cursorEl = null; }
    if (styleEl) { styleEl.remove(); styleEl = null; }
    if (origPos === 'static') node.style.position = '';
    node.style.overflow = '';
  }

  setup();

  return {
    update(newParams: HeroBgMeshParams | undefined) {
      teardown();
      config = resolveConfig(newParams);
      setup();
    },
    destroy() { teardown(); },
  };
};
