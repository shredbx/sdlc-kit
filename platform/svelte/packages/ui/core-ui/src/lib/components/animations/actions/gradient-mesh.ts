/**
 * Gradient Mesh — Svelte action wrapper
 *
 * Multiple radial-gradient blobs floating on CSS @keyframes.
 * Creates a soft, organic background animation.
 *
 * Default behaviour: blobs are statically visible at `staticOpacity` — zero animation cost.
 * On hover: opacity rises to `opacity` and blobs animate. On leave: animation stops, static
 * preview remains. This lets hundreds of cards coexist on one page with no performance cost.
 *
 * @example
 * ```svelte
 * <div use:gradientMesh>...</div>
 * <div use:gradientMesh={{ trigger: 'hover' }}>...</div>
 * ```
 */

import type { Action } from 'svelte/action';

export interface GradientMeshParams {
  duration?: number;
  delay?: number;
  colors?: string[];
  opacity?: number;
  staticOpacity?: number;
  blur?: number;
  trigger?: 'hover' | 'scroll' | 'load' | 'manual';
}

let keyframeCounter = 0;

export const gradientMesh: Action<HTMLElement, GradientMeshParams | undefined> = (node, params = {}) => {
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    return {};
  }

  let config = resolveConfig(params);
  let meshContainer: HTMLDivElement | null = null;
  let blobs: HTMLDivElement[] = [];
  let styleEl: HTMLStyleElement | null = null;
  let observer: IntersectionObserver | null = null;
  const originalStyles = node.style.cssText;
  const originalPosition = getComputedStyle(node).position;

  function resolveConfig(p: GradientMeshParams | undefined) {
    return {
      duration: p?.duration ?? 8000,
      delay: p?.delay ?? 0,
      colors: p?.colors ?? ['rgba(255, 0, 64, 0.3)', 'rgba(0, 255, 255, 0.3)', 'rgba(255, 0, 255, 0.3)', 'rgba(255, 200, 0, 0.3)'],
      opacity: p?.opacity ?? 1.0,
      staticOpacity: p?.staticOpacity ?? 0.75,
      blur: p?.blur ?? 30,
      trigger: p?.trigger ?? 'load'
    };
  }

  function setup() {
    const { duration, delay, colors, opacity, staticOpacity, blur, trigger } = config;

    const prefix = `sbx-mesh-${++keyframeCounter}`;

    if (originalPosition === 'static') {
      node.style.position = 'relative';
    }
    node.style.overflow = 'hidden';

    meshContainer = document.createElement('div');
    meshContainer.setAttribute('aria-hidden', 'true');
    meshContainer.style.cssText = `
      position: absolute;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
      pointer-events: none;
      z-index: 0;
      opacity: ${staticOpacity};
      transition: opacity 600ms ease;
      filter: blur(${blur}px);
    `;

    let cssRules = '';
    blobs = [];
    if (!meshContainer) return;

    colors.forEach((color, i) => {
      const animName = `${prefix}-blob-${i}`;
      const startX = 20 + (i * 20) % 60;
      const startY = 20 + (i * 15) % 60;
      const midX1 = (startX + 30) % 80 + 10;
      const midY1 = (startY + 25) % 80 + 10;
      const midX2 = (startX + 50) % 80 + 10;
      const midY2 = (startY + 45) % 80 + 10;

      cssRules += `
        @keyframes ${animName} {
          0%, 100% { transform: translate(${startX}%, ${startY}%) scale(1); }
          33% { transform: translate(${midX1}%, ${midY1}%) scale(1.1); }
          66% { transform: translate(${midX2}%, ${midY2}%) scale(0.9); }
        }
      `;

      const blob = document.createElement('div');
      // Position blobs at their initial keyframe position (no transform = translate(startX%, startY%))
      blob.style.cssText = `
        position: absolute;
        width: 70%;
        height: 70%;
        border-radius: 50%;
        background: radial-gradient(circle, ${color} 0%, transparent 70%);
        will-change: transform;
        transform: translate(${startX}%, ${startY}%) scale(1);
      `;
      blob.dataset.animName = animName;

      blobs.push(blob);
      meshContainer!.appendChild(blob);
    });

    styleEl = document.createElement('style');
    styleEl.id = prefix;
    styleEl.textContent = cssRules;
    document.head.appendChild(styleEl);

    node.insertBefore(meshContainer, node.firstChild);

    Array.from(node.children).forEach((child) => {
      if (child !== meshContainer && child instanceof HTMLElement) {
        if (getComputedStyle(child).position === 'static') {
          child.style.position = 'relative';
        }
        child.style.zIndex = child.style.zIndex || '1';
      }
    });

    if (trigger === 'hover') {
      node.addEventListener('mouseenter', activate);
      node.addEventListener('mouseleave', deactivate);
    } else if (trigger === 'scroll') {
      observer = new IntersectionObserver(([entry]) => {
        if (entry.isIntersecting) activate();
        else deactivate();
      }, { threshold: 0.1 });
      observer.observe(node);
    } else if (trigger === 'load') {
      setTimeout(activate, delay);
    }
    // trigger === 'manual': blobs are visible statically, caller activates via returned handle
  }

  function activate() {
    const { duration, opacity } = config;
    if (meshContainer) meshContainer.style.opacity = String(opacity);
    blobs.forEach((blob, i) => {
      const animName = blob.dataset.animName;
      const blobDuration = duration + i * 1000;
      blob.style.animation = `${animName} ${blobDuration}ms ease-in-out infinite`;
    });
  }

  function deactivate() {
    const { staticOpacity } = config;
    // Keep mesh visible at static opacity — just stop animating
    if (meshContainer) meshContainer.style.opacity = String(staticOpacity);
    blobs.forEach((blob) => {
      blob.style.animation = '';
    });
  }

  function teardown() {
    node.removeEventListener('mouseenter', activate);
    node.removeEventListener('mouseleave', deactivate);
    if (observer) { observer.disconnect(); observer = null; }
    if (styleEl) { styleEl.remove(); styleEl = null; }
    if (meshContainer) { meshContainer.remove(); meshContainer = null; }
    blobs = [];
    node.style.cssText = originalStyles;
  }

  setup();

  return {
    update(newParams: GradientMeshParams | undefined) {
      teardown();
      config = resolveConfig(newParams);
      setup();
    },
    destroy() {
      teardown();
    }
  };
};
