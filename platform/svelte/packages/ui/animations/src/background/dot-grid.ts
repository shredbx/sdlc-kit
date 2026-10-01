import type { AnimationController, DotGridOptions } from '../types';
import { createNoopController, prefersReducedMotion } from '../_internal/utils';

/**
 * Subtle dot pattern background.
 *
 * Creates a CSS dot grid overlay using repeating linear-gradient.
 * Very subtle (low opacity) with an optional radial mask for edge fade.
 * The grid spacing and dot color are configurable.
 *
 * Note: this is a static pattern, not an animation per se, but it
 * follows the AnimationController interface for consistency. play()
 * shows the grid, pause()/reset() hide it.
 *
 * @example
 * ```ts
 * import { createDotGrid } from '@sbx/animations/background';
 * const ctrl = createDotGrid(document.querySelector('.section')!, { spacing: 60 });
 * ctrl.play();
 * ```
 */
export function createDotGrid(
  element: HTMLElement,
  options: DotGridOptions = {},
): AnimationController {
  const {
    spacing = 60,
    color = 'rgba(255, 255, 255, 0.015)',
    fadeMask = true,
    reducedMotion = true,
  } = options;

  if (reducedMotion && prefersReducedMotion()) {
    return createNoopController();
  }

  let playing = false;
  const originalPosition = getComputedStyle(element).position;

  if (originalPosition === 'static') {
    element.style.position = 'relative';
  }

  // Create dot grid overlay element
  const overlay = document.createElement('div');
  Object.assign(overlay.style, {
    position: 'absolute',
    top: '0',
    left: '0',
    width: '100%',
    height: '100%',
    pointerEvents: 'none',
    zIndex: '0',
    opacity: '0',
    transition: 'opacity 400ms ease',
    // Dot grid via radial-gradient
    backgroundImage: `radial-gradient(circle, ${color} 1px, transparent 1px)`,
    backgroundSize: `${spacing}px ${spacing}px`,
  });

  if (fadeMask) {
    overlay.style.maskImage = 'radial-gradient(ellipse at center, black 40%, transparent 80%)';
    (overlay.style as any).webkitMaskImage = 'radial-gradient(ellipse at center, black 40%, transparent 80%)';
  }

  overlay.setAttribute('aria-hidden', 'true');

  // Insert as first child so content sits on top
  element.insertBefore(overlay, element.firstChild);

  const controller: AnimationController = {
    play() {
      playing = true;
      overlay.style.opacity = '1';
    },
    pause() {
      playing = false;
      overlay.style.opacity = '0';
    },
    reset() {
      playing = false;
      overlay.style.opacity = '0';
    },
    destroy() {
      playing = false;
      overlay.remove();
      if (originalPosition === 'static') {
        element.style.position = '';
      }
    },
    get isPlaying() {
      return playing;
    },
  };

  return controller;
}
