import type { AnimationController, TextSplitOptions } from '../types';
import { createNoopController, prefersReducedMotion, wrapCharacters } from '../_internal/utils';

/**
 * Character split reveal animation.
 *
 * Wraps each character in an individual span and on play splits them
 * apart with staggered delays. Characters move outward from their
 * original position based on the configured direction.
 *
 * @example
 * ```ts
 * import { createTextSplit } from '@sbx/animations/text';
 * const ctrl = createTextSplit(document.querySelector('h1')!, { direction: 'horizontal' });
 * ctrl.play();
 * ```
 */
export function createTextSplit(
  element: HTMLElement,
  options: TextSplitOptions = {},
): AnimationController {
  const {
    duration = 500,
    delay = 0,
    easing = 'cubic-bezier(0.22, 1, 0.36, 1)',
    distance = 16,
    direction = 'horizontal',
    reducedMotion = true,
  } = options;

  if (reducedMotion && prefersReducedMotion()) {
    return createNoopController();
  }

  let playing = false;
  const { chars, restore } = wrapCharacters(element);

  // Calculate stagger: 30ms per char, max 600ms total spread
  const staggerMs = Math.min(30, 600 / Math.max(chars.length, 1));
  const midpoint = (chars.length - 1) / 2;

  chars.forEach((span, i) => {
    const charDelay = delay + Math.abs(i - midpoint) * staggerMs;
    span.style.transition = `transform ${duration}ms ${easing} ${charDelay}ms, opacity ${duration}ms ${easing} ${charDelay}ms`;
    span.style.opacity = '1';
  });

  function getTranslate(index: number): string {
    // Characters move away from center
    const sign = index < midpoint ? -1 : index > midpoint ? 1 : 0;
    const magnitude = Math.abs(index - midpoint) * (distance / midpoint || distance);

    switch (direction) {
      case 'vertical':
        return `translateY(${sign * magnitude}px)`;
      case 'diagonal':
        return `translate(${sign * magnitude}px, ${sign * magnitude}px)`;
      case 'horizontal':
      default:
        return `translateX(${sign * magnitude}px)`;
    }
  }

  const controller: AnimationController = {
    play() {
      playing = true;
      chars.forEach((span, i) => {
        span.style.transform = getTranslate(i);
        span.style.opacity = '0.4';
      });
    },
    pause() {
      playing = false;
    },
    reset() {
      playing = false;
      chars.forEach((span) => {
        span.style.transform = 'translate(0, 0)';
        span.style.opacity = '1';
      });
    },
    destroy() {
      playing = false;
      restore();
    },
    get isPlaying() {
      return playing;
    },
  };

  return controller;
}
