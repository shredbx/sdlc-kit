import type { AnimationController, BaseAnimationOptions } from '../types';
import { createNoopController, prefersReducedMotion } from '../_internal/utils';

/** Options specific to accent-line-reveal. Direction = which way the line draws. */
interface AccentLineRevealOptions extends BaseAnimationOptions {
  direction?: 'left' | 'right';
  distance?: number;
  stagger?: number;
}

/**
 * Accent line draws then content reveals.
 *
 * A horizontal accent line draws from left to right using scaleX.
 * Once the line animation completes, the child content fades in
 * with a slight translateY lift. The sequence is line-first, content-second.
 *
 * @example
 * ```ts
 * import { createAccentLineReveal } from '@sbx/animations/reveal';
 * const ctrl = createAccentLineReveal(document.querySelector('.section')!, {
 *   direction: 'right',
 *   stagger: 100,
 * });
 * ctrl.play();
 * ```
 */
export function createAccentLineReveal(
  element: HTMLElement,
  options: AccentLineRevealOptions = {},
): AnimationController {
  const {
    duration = 600,
    delay = 0,
    easing = 'cubic-bezier(0.22, 1, 0.36, 1)',
    direction = 'right',
    distance = 12,
    stagger = 80,
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
  element.style.overflow = 'hidden';

  // Create accent line element
  const line = document.createElement('div');
  const lineOrigin = direction === 'left' ? 'right center' : 'left center';
  Object.assign(line.style, {
    position: 'absolute',
    top: '0',
    left: '0',
    width: '100%',
    height: '2px',
    background: 'currentColor',
    transformOrigin: lineOrigin,
    transform: 'scaleX(0)',
    transition: `transform ${duration}ms ${easing} ${delay}ms`,
  });
  line.setAttribute('aria-hidden', 'true');

  element.insertBefore(line, element.firstChild);

  // Prepare child elements for reveal
  const children = Array.from(element.children).filter((child) => child !== line) as HTMLElement[];
  const contentDelay = delay + duration; // content starts after line finishes

  children.forEach((child, i) => {
    const childDelay = contentDelay + i * stagger;
    child.style.transition = `opacity ${duration}ms ${easing} ${childDelay}ms, transform ${duration}ms ${easing} ${childDelay}ms`;
    child.style.opacity = '0';
    child.style.transform = `translateY(${distance}px)`;
  });

  const controller: AnimationController = {
    play() {
      playing = true;
      // Phase 1: draw the line
      line.style.transform = 'scaleX(1)';
      // Phase 2: reveal content (handled by CSS delays)
      children.forEach((child) => {
        child.style.opacity = '1';
        child.style.transform = 'translateY(0)';
      });
    },
    pause() {
      playing = false;
    },
    reset() {
      playing = false;
      line.style.transform = 'scaleX(0)';
      children.forEach((child) => {
        child.style.opacity = '0';
        child.style.transform = `translateY(${distance}px)`;
      });
    },
    destroy() {
      playing = false;
      line.remove();
      children.forEach((child) => {
        child.style.transition = '';
        child.style.opacity = '';
        child.style.transform = '';
      });
      if (originalPosition === 'static') {
        element.style.position = '';
      }
      element.style.overflow = '';
    },
    get isPlaying() {
      return playing;
    },
  };

  return controller;
}
