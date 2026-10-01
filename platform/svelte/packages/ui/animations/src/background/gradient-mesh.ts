import type { AnimationController, BackgroundOptions } from '../types';
import { createNoopController, prefersReducedMotion, injectStyle } from '../_internal/utils';

let meshCounter = 0;

/**
 * Animated mesh gradient background.
 *
 * Creates multiple radial-gradient blobs positioned absolutely inside
 * the target element. Each blob floats on a different timing period
 * (12s, 15s, 18s) creating an organic, living mesh gradient effect.
 *
 * @example
 * ```ts
 * import { createGradientMesh } from '@sbx/animations/background';
 * const ctrl = createGradientMesh(document.querySelector('.hero')!);
 * ctrl.play();
 * ```
 */
export function createGradientMesh(
  element: HTMLElement,
  options: BackgroundOptions = {},
): AnimationController {
  const {
    duration = 12000,
    delay = 0,
    opacity = 1,
    colors = [
      'rgba(233, 69, 96, 0.12)',
      'rgba(96, 165, 250, 0.08)',
      'rgba(167, 139, 250, 0.06)',
    ],
    blur = 80,
    reducedMotion = true,
  } = options;

  if (reducedMotion && prefersReducedMotion()) {
    return createNoopController();
  }

  let playing = false;
  const id = `sbx-mesh-${++meshCounter}`;
  const originalPosition = getComputedStyle(element).position;
  const originalOverflow = element.style.overflow;

  if (originalPosition === 'static') {
    element.style.position = 'relative';
  }
  element.style.overflow = 'hidden';

  // Create container for blobs
  const container = document.createElement('div');
  Object.assign(container.style, {
    position: 'absolute',
    top: '0',
    left: '0',
    width: '100%',
    height: '100%',
    pointerEvents: 'none',
    zIndex: '0',
    opacity: String(opacity),
  });
  container.setAttribute('aria-hidden', 'true');

  // Blob configurations: position, size, color, animation period
  const blobConfigs = [
    { x: '20%', y: '30%', size: '60%', color: colors[0] || 'rgba(233,69,96,0.12)', period: duration },
    { x: '70%', y: '20%', size: '50%', color: colors[1] || 'rgba(96,165,250,0.08)', period: Math.round(duration * 1.25) },
    { x: '50%', y: '70%', size: '55%', color: colors[2] || 'rgba(167,139,250,0.06)', period: Math.round(duration * 1.5) },
  ];

  // Generate keyframes for each blob's float path
  let css = '';
  const blobs: HTMLDivElement[] = [];

  blobConfigs.forEach((config, i) => {
    const blobId = `${id}-blob-${i}`;

    // Each blob has a unique circular float path
    const offsetA = 15 + i * 10;
    const offsetB = 10 + i * 8;

    css += `
@keyframes ${blobId}-float {
  0%, 100% { transform: translate(0, 0); }
  25% { transform: translate(${offsetA}px, -${offsetB}px); }
  50% { transform: translate(-${offsetB}px, ${offsetA}px); }
  75% { transform: translate(-${offsetA}px, -${offsetB / 2}px); }
}
`;

    const blob = document.createElement('div');
    Object.assign(blob.style, {
      position: 'absolute',
      left: config.x,
      top: config.y,
      width: config.size,
      height: config.size,
      borderRadius: '50%',
      background: `radial-gradient(circle, ${config.color} 0%, transparent 70%)`,
      filter: `blur(${blur}px)`,
      animation: `${blobId}-float ${config.period}ms ease-in-out infinite`,
      animationDelay: `${delay + i * 200}ms`,
      animationPlayState: 'paused',
      transform: 'translate(-50%, -50%)',
    });

    blobs.push(blob);
    container.appendChild(blob);
  });

  const removeStyle = injectStyle(css);

  // Insert container as first child so content renders above
  element.insertBefore(container, element.firstChild);

  const controller: AnimationController = {
    play() {
      playing = true;
      blobs.forEach((blob) => {
        blob.style.animationPlayState = 'running';
      });
    },
    pause() {
      playing = false;
      blobs.forEach((blob) => {
        blob.style.animationPlayState = 'paused';
      });
    },
    reset() {
      playing = false;
      blobs.forEach((blob) => {
        blob.style.animationPlayState = 'paused';
        // Force restart by removing/re-adding animation
        const anim = blob.style.animation;
        blob.style.animation = 'none';
        // Force reflow
        void blob.offsetHeight;
        blob.style.animation = anim;
        blob.style.animationPlayState = 'paused';
      });
    },
    destroy() {
      playing = false;
      container.remove();
      removeStyle();
      if (originalPosition === 'static') {
        element.style.position = '';
      }
      element.style.overflow = originalOverflow;
    },
    get isPlaying() {
      return playing;
    },
  };

  return controller;
}
