/**
 * Hero Background: Electric Grid Pulse — Svelte action
 *
 * Grid pattern with click-to-pulse electric bolt animation.
 * Click anywhere to send an electric pulse along the grid lines
 * from the click point outward like a circuit signal.
 * Ported from hero-backgrounds-catalog.html section 10.
 *
 * @example
 * ```svelte
 * <section use:heroBgElectric>...</section>
 * <section use:heroBgElectric={{ gridSize: 40, cursor: true }}>...</section>
 * ```
 */

import type { Action } from 'svelte/action';

export interface HeroBgElectricParams {
  /** Grid cell size in px (default: 40) */
  gridSize?: number;
  /** Enable cursor glow (default: true) */
  cursor?: boolean;
  /** Enable click pulse (default: true) */
  clickPulse?: boolean;
  /** Number of grid segments each pulse travels (default: 18) */
  pulseReach?: number;
}

interface Pulse {
  x: number;
  y: number;
  dx: number;
  dy: number;
  step: number;
  maxStep: number;
  progress: number;
  speed: number;
  life: number;
  branch: boolean;
}

export const heroBgElectric: Action<HTMLElement, HeroBgElectricParams | undefined> = (node, params = {}) => {
  const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  let config = resolveConfig(params);
  let fadeEl: HTMLDivElement | null = null;
  let cursorEl: HTMLDivElement | null = null;
  let canvas: HTMLCanvasElement | null = null;
  let ctx: CanvasRenderingContext2D | null = null;
  let pulses: Pulse[] = [];
  let rafId = 0;
  let onMove: ((e: MouseEvent) => void) | null = null;
  let onClick: ((e: MouseEvent) => void) | null = null;
  let onResize: (() => void) | null = null;
  const origPos = getComputedStyle(node).position;

  function resolveConfig(p: HeroBgElectricParams | undefined) {
    return {
      gridSize: p?.gridSize ?? 40,
      cursor: p?.cursor ?? true,
      clickPulse: p?.clickPulse ?? true,
      pulseReach: p?.pulseReach ?? 18,
    };
  }

  function resizeCanvas() {
    if (!canvas) return;
    canvas.width = node.offsetWidth;
    canvas.height = node.offsetHeight;
  }

  /** Resolve the electric base/glow color from CSS vars if possible.
   *  Games (sole consumer): neon cyan tuned to match the arcade accent
   *  palette (var(--arcade-cyan) = #00d4ff). Bright core + softer glow. */
  function getElectricColors(): { base: [number, number, number]; glow: [number, number, number] } {
    const fadeProp = getComputedStyle(node).getPropertyValue('--fade-color').trim();
    const isLight = fadeProp.startsWith('#f') || fadeProp.startsWith('rgb(2') || fadeProp.startsWith('rgb(24') || fadeProp.startsWith('white');
    if (isLight) {
      return { base: [100, 60, 220], glow: [120, 80, 240] };
    }
    // Neon cyan: bright core (near-white tint on cyan), soft blue-cyan glow.
    // Matches --arcade-cyan (#00d4ff) dialed up for vivid pulse tips.
    return { base: [120, 230, 255], glow: [0, 180, 240] };
  }

  function drawFrame() {
    if (!canvas || !ctx) return;
    // Bind locals so the narrowed non-null types survive the forEach closure.
    const c = ctx;
    const cv = canvas;
    const w = cv.width;
    const h = cv.height;
    c.clearRect(0, 0, w, h);

    const { base, glow } = getElectricColors();
    const GRID = config.gridSize;

    pulses.forEach(p => {
      p.progress += p.speed;
      if (p.progress < 0) return;

      const dist = p.progress;
      const segLen = GRID;
      const segIdx = Math.floor(dist / segLen);
      const segFrac = (dist % segLen) / segLen;

      if (segIdx > p.maxStep) { p.life = 0; return; }

      const sx = p.x + p.dx * segIdx * GRID;
      const sy = p.y + p.dy * segIdx * GRID;
      const ex = sx + p.dx * GRID * segFrac;
      const ey = sy + p.dy * GRID * segFrac;

      if (sx < -GRID || sx > w + GRID || sy < -GRID || sy > h + GRID) return;

      const fadeDist = (segIdx + segFrac) / p.maxStep;
      const alpha = p.life * (1 - fadeDist * 0.8);
      if (alpha <= 0) return;

      // Branch pulses zigzag along grid axes
      let fx = sx, fy = sy, tx = ex, ty = ey;
      if (p.branch) {
        if (segIdx % 2 === 0) {
          tx = sx + p.dx * GRID * segFrac;
          ty = sy;
        } else {
          tx = sx;
          ty = sy + p.dy * GRID * segFrac;
        }
      }

      // Outer glow — thinner + softer than the old thick "rope" look
      c.beginPath();
      c.moveTo(fx, fy);
      c.lineTo(tx, ty);
      c.strokeStyle = `rgba(${glow[0]},${glow[1]},${glow[2]},${alpha * 0.22})`;
      c.lineWidth = p.branch ? 2 : 3;
      c.stroke();

      // Core line — sharp 1px filament
      c.beginPath();
      c.moveTo(fx, fy);
      c.lineTo(tx, ty);
      c.strokeStyle = `rgba(${base[0]},${base[1]},${base[2]},${alpha * 0.85})`;
      c.lineWidth = p.branch ? 0.5 : 0.9;
      c.stroke();

      // Bright tip dot
      c.beginPath();
      c.arc(tx, ty, p.branch ? 1.25 : 1.75, 0, Math.PI * 2);
      c.fillStyle = `rgba(${base[0]},${base[1]},${base[2]},${alpha})`;
      c.fill();

      // Tiny intersection spark at grid nodes (much smaller than before)
      if (segFrac < 0.08 && segIdx > 0) {
        c.beginPath();
        c.arc(sx, sy, 2, 0, Math.PI * 2);
        c.fillStyle = `rgba(255,255,255,${alpha * 0.4})`;
        c.fill();
      }
    });

    pulses = pulses.filter(p => p.life > 0);
    rafId = requestAnimationFrame(drawFrame);
  }

  function spawnPulses(cx: number, cy: number) {
    const GRID = config.gridSize;
    const reach = config.pulseReach;
    const gx = Math.round(cx / GRID) * GRID;
    const gy = Math.round(cy / GRID) * GRID;

    const dirs: { dx: number; dy: number }[] = [
      { dx: 1, dy: 0 }, { dx: -1, dy: 0 },
      { dx: 0, dy: 1 }, { dx: 0, dy: -1 },
    ];
    dirs.forEach(d => {
      for (let step = 0; step < reach; step++) {
        pulses.push({
          x: gx, y: gy,
          dx: d.dx, dy: d.dy,
          step,
          maxStep: reach,
          progress: -step * 2,
          speed: 3 + Math.random(),
          life: 1,
          branch: false,
        });
      }
    });

    const diags: { dx: number; dy: number }[] = [
      { dx: 1, dy: 1 }, { dx: -1, dy: -1 },
      { dx: 1, dy: -1 }, { dx: -1, dy: 1 },
    ];
    const diagReach = Math.ceil(reach * 0.45);
    diags.forEach(d => {
      for (let step = 0; step < diagReach; step++) {
        pulses.push({
          x: gx, y: gy,
          dx: d.dx, dy: d.dy,
          step,
          maxStep: diagReach,
          progress: -step * 3,
          speed: 2 + Math.random(),
          life: 0.6,
          branch: true,
        });
      }
    });
  }

  function setup() {
    const { gridSize, cursor, clickPulse } = config;

    if (origPos === 'static') node.style.position = 'relative';
    node.style.overflow = 'hidden';

    // Grid background
    node.style.backgroundSize = `${gridSize}px ${gridSize}px`;
    node.style.backgroundImage = [
      'linear-gradient(to right, var(--grid-line, rgba(255,255,255,0.05)) 1px, transparent 1px)',
      'linear-gradient(to bottom, var(--grid-line, rgba(255,255,255,0.05)) 1px, transparent 1px)',
    ].join(',');

    // Radial fade overlay
    fadeEl = document.createElement('div');
    fadeEl.setAttribute('aria-hidden', 'true');
    fadeEl.style.cssText = `
      position: absolute; inset: 0; pointer-events: none; z-index: 1;
      background: radial-gradient(ellipse 80% 60% at 50% 50%, transparent var(--fade-start, 30%), var(--fade-color, #050508) var(--fade-end, 80%));
    `;
    node.appendChild(fadeEl);

    // Canvas for electric bolts
    canvas = document.createElement('canvas');
    canvas.setAttribute('aria-hidden', 'true');
    canvas.style.cssText = 'position: absolute; inset: 0; z-index: 2; pointer-events: none;';
    node.appendChild(canvas);
    ctx = canvas.getContext('2d');
    resizeCanvas();

    // Cursor glow layer
    cursorEl = document.createElement('div');
    cursorEl.setAttribute('aria-hidden', 'true');
    cursorEl.style.cssText = 'position: absolute; inset: 0; pointer-events: none; z-index: 3;';
    node.appendChild(cursorEl);

    if (reduceMotion) return;

    if (cursor) {
      onMove = (e: MouseEvent) => {
        const rect = node.getBoundingClientRect();
        const x = e.clientX - rect.left;
        const y = e.clientY - rect.top;
        cursorEl!.style.background = `radial-gradient(300px circle at ${x}px ${y}px, var(--spot-glow, rgba(100,200,255,0.1)), transparent 50%)`;
      };
      node.addEventListener('mousemove', onMove);
    }

    if (clickPulse) {
      onClick = (e: MouseEvent) => {
        const rect = node.getBoundingClientRect();
        spawnPulses(e.clientX - rect.left, e.clientY - rect.top);
      };
      node.addEventListener('click', onClick);
    }

    onResize = () => resizeCanvas();
    window.addEventListener('resize', onResize);

    rafId = requestAnimationFrame(drawFrame);
  }

  function teardown() {
    cancelAnimationFrame(rafId);
    if (onMove) { node.removeEventListener('mousemove', onMove); onMove = null; }
    if (onClick) { node.removeEventListener('click', onClick); onClick = null; }
    if (onResize) { window.removeEventListener('resize', onResize); onResize = null; }
    if (fadeEl) { fadeEl.remove(); fadeEl = null; }
    if (canvas) { canvas.remove(); canvas = null; ctx = null; }
    if (cursorEl) { cursorEl.remove(); cursorEl = null; }
    pulses = [];
    node.style.backgroundImage = '';
    node.style.backgroundSize = '';
    if (origPos === 'static') node.style.position = '';
    node.style.overflow = '';
  }

  setup();

  return {
    update(newParams: HeroBgElectricParams | undefined) {
      teardown();
      config = resolveConfig(newParams);
      setup();
    },
    destroy() { teardown(); },
  };
};
