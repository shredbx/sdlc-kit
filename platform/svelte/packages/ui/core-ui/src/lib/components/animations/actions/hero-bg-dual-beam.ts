/**
 * Hero Background: Dual Beam Spotlight — Svelte action
 *
 * Animated light beams with sinusoidal motion. CSS-only, no cursor tracking.
 * Ported from hero-backgrounds-catalog.html section 4.
 *
 * @example
 * ```svelte
 * <section use:heroBgDualBeam>...</section>
 * ```
 */

import type { Action } from 'svelte/action';

export interface HeroBgDualBeamParams {
  /** Enable animations (default: true — respects prefers-reduced-motion) */
  animate?: boolean;
}

let beamCounter = 0;

export const heroBgDualBeam: Action<HTMLElement, HeroBgDualBeamParams | undefined> = (node, params = {}) => {
  const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  let config = resolveConfig(params);
  let beamLeft: HTMLDivElement | null = null;
  let beamRight: HTMLDivElement | null = null;
  let beamAccent: HTMLDivElement | null = null;
  let styleEl: HTMLStyleElement | null = null;
  const origPos = getComputedStyle(node).position;

  function resolveConfig(p: HeroBgDualBeamParams | undefined) {
    return { animate: p?.animate ?? true };
  }

  function setup() {
    const prefix = `sbx-beam-${++beamCounter}`;

    if (origPos === 'static') node.style.position = 'relative';
    node.style.overflow = 'hidden';

    const shouldAnimate = config.animate && !reduceMotion;

    // Keyframes
    if (shouldAnimate) {
      styleEl = document.createElement('style');
      styleEl.id = prefix;
      styleEl.textContent = `
        @keyframes ${prefix}-left {
          0% { transform: translateX(0) rotate(-15deg); }
          100% { transform: translateX(100px) rotate(-12deg); }
        }
        @keyframes ${prefix}-right {
          0% { transform: translateX(0) rotate(10deg); }
          100% { transform: translateX(-80px) rotate(8deg); }
        }
        @keyframes ${prefix}-center {
          0% { transform: translateX(0) rotate(-5deg); opacity: 0.6; }
          100% { transform: translateX(60px) rotate(-3deg); opacity: 1; }
        }
      `;
      document.head.appendChild(styleEl);
    }

    const beamBase = 'position: absolute; z-index: 1; pointer-events: none;';

    beamLeft = document.createElement('div');
    beamLeft.setAttribute('aria-hidden', 'true');
    beamLeft.style.cssText = `${beamBase}
      top: -200px; left: 10%; width: 560px; height: 1380px;
      background: radial-gradient(68% 69% at 55% 31%, var(--beam1, hsla(210,100%,85%,0.1)) 0, var(--beam1-mid, hsla(210,100%,55%,0.03)) 50%, transparent 80%);
      transform: rotate(-15deg);
      ${shouldAnimate ? `animation: ${prefix}-left 7s ease-in-out infinite alternate;` : ''}
    `;
    node.appendChild(beamLeft);

    beamRight = document.createElement('div');
    beamRight.setAttribute('aria-hidden', 'true');
    beamRight.style.cssText = `${beamBase}
      top: -300px; right: 5%; width: 240px; height: 1380px;
      background: radial-gradient(50% 50% at 50% 50%, var(--beam2, hsla(210,100%,85%,0.07)) 0, transparent 80%);
      transform: rotate(10deg);
      ${shouldAnimate ? `animation: ${prefix}-right 7s ease-in-out 2s infinite alternate-reverse;` : ''}
    `;
    node.appendChild(beamRight);

    beamAccent = document.createElement('div');
    beamAccent.setAttribute('aria-hidden', 'true');
    beamAccent.style.cssText = `${beamBase}
      top: -100px; left: 40%; width: 240px; height: 1000px;
      background: radial-gradient(50% 50% at 50% 50%, var(--beam3, hsla(270,80%,70%,0.05)) 0, transparent 80%);
      transform: rotate(-5deg);
      ${shouldAnimate ? `animation: ${prefix}-center 9s ease-in-out 1s infinite alternate;` : ''}
    `;
    node.appendChild(beamAccent);
  }

  function teardown() {
    if (beamLeft) { beamLeft.remove(); beamLeft = null; }
    if (beamRight) { beamRight.remove(); beamRight = null; }
    if (beamAccent) { beamAccent.remove(); beamAccent = null; }
    if (styleEl) { styleEl.remove(); styleEl = null; }
    if (origPos === 'static') node.style.position = '';
    node.style.overflow = '';
  }

  setup();

  return {
    update(newParams: HeroBgDualBeamParams | undefined) {
      teardown();
      config = resolveConfig(newParams);
      setup();
    },
    destroy() { teardown(); },
  };
};
