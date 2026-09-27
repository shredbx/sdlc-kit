import type { AnimationController, CharRevealOptions } from '../types';
import { createNoopController, prefersReducedMotion, wrapCharacters } from '../_internal/utils';

/**
 * Staggered character reveal animation.
 *
 * Characters start invisible (opacity 0, translated down) and appear
 * one by one with a configurable stagger delay between each.
 *
 * @example
 * ```ts
 * import { createDelayedReveal } from '@sbx/animations/text';
 * const ctrl = createDelayedReveal(document.querySelector('h1')!, { stagger: 30 });
 * ctrl.play();
 * ```
 */
export function createDelayedReveal(
  element: HTMLElement,
  options: CharRevealOptions = {},
): AnimationController {
  const {
    duration = 400,
    delay = 0,
    easing = 'cubic-bezier(0.22, 1, 0.36, 1)',
    stagger = 25,
    offsetY = 12,
    reducedMotion = true,
  } = options;

  if (reducedMotion && prefersReducedMotion()) {
    return createNoopController();
  }

  let playing = false;
  const { chars, restore } = wrapCharacters(element);

  // Initialize: all characters hidden
  chars.forEach((span, i) => {
    const charDelay = delay + i * stagger;
    span.style.transition = `opacity ${duration}ms ${easing} ${charDelay}ms, transform ${duration}ms ${easing} ${charDelay}ms`;
    span.style.opacity = '0';
    span.style.transform = `translateY(${offsetY}px)`;
  });

  const controller: AnimationController = {
    play() {
      playing = true;
      chars.forEach((span) => {
        span.style.opacity = '1';
        span.style.transform = 'translateY(0)';
      });
    },
    pause() {
      playing = false;
      // Freeze current state
      chars.forEach((span) => {
        const currentOpacity = getComputedStyle(span).opacity;
        const currentTransform = getComputedStyle(span).transform;
        span.style.transition = 'none';
        span.style.opacity = currentOpacity;
        span.style.transform = currentTransform;
        // Restore transition on next frame
        requestAnimationFrame(() => {
          const i = chars.indexOf(span);
          const charDelay = delay + i * stagger;
          span.style.transition = `opacity ${duration}ms ${easing} ${charDelay}ms, transform ${duration}ms ${easing} ${charDelay}ms`;
        });
      });
    },
    reset() {
      playing = false;
      chars.forEach((span) => {
        span.style.opacity = '0';
        span.style.transform = `translateY(${offsetY}px)`;
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
