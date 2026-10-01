import type { AnimationController, TextSplitOptions } from '../types';
import { createNoopController, prefersReducedMotion } from '../_internal/utils';

/**
 * CSS clip-path diagonal split animation.
 *
 * Splits an element into two halves along a diagonal line. On play,
 * the top half translates up-left and the bottom half translates
 * down-right, creating a shattered-open effect.
 *
 * @example
 * ```ts
 * import { createDiagonalSlice } from '@sbx/animations/text';
 * const ctrl = createDiagonalSlice(document.querySelector('h1')!, { angle: 35 });
 * ctrl.play();
 * ```
 */
export function createDiagonalSlice(
  element: HTMLElement,
  options: TextSplitOptions = {},
): AnimationController {
  const {
    duration = 500,
    delay = 0,
    easing = 'cubic-bezier(0.23, 1, 0.32, 1)',
    angle = 35,
    distance = 16,
    reducedMotion = true,
  } = options;

  if (reducedMotion && prefersReducedMotion()) {
    return createNoopController();
  }

  let playing = false;
  const originalPosition = getComputedStyle(element).position;

  // Ensure element is a positioning context
  if (originalPosition === 'static') {
    element.style.position = 'relative';
  }
  element.style.overflow = 'visible';

  const text = element.textContent || '';
  const computedStyle = getComputedStyle(element);

  // Calculate clip-path split percentages from angle
  // angle=35 gives the 35%/65% split from the prototype
  const splitTop = Math.max(5, Math.min(95, angle));
  const splitBottom = 100 - splitTop;

  // Create two overlay layers that clip the original text
  const topHalf = document.createElement('span');
  const bottomHalf = document.createElement('span');

  const sharedStyle: Partial<CSSStyleDeclaration> = {
    position: 'absolute',
    top: '0',
    left: '0',
    width: '100%',
    height: '100%',
    display: 'flex',
    alignItems: 'center',
    justifyContent: computedStyle.textAlign === 'center' ? 'center' : 'flex-start',
    fontSize: computedStyle.fontSize,
    fontWeight: computedStyle.fontWeight,
    fontFamily: computedStyle.fontFamily,
    letterSpacing: computedStyle.letterSpacing,
    lineHeight: computedStyle.lineHeight,
    color: computedStyle.color,
    pointerEvents: 'none',
    transition: `transform ${duration}ms ${easing} ${delay}ms, opacity ${duration}ms ${easing} ${delay}ms`,
  };

  Object.assign(topHalf.style, sharedStyle);
  Object.assign(bottomHalf.style, sharedStyle);

  topHalf.textContent = text;
  bottomHalf.textContent = text;

  topHalf.style.clipPath = `polygon(0 0, 100% 0, 100% ${splitTop}%, 0 ${splitBottom}%)`;
  bottomHalf.style.clipPath = `polygon(0 ${splitBottom}%, 100% ${splitTop}%, 100% 100%, 0 100%)`;

  topHalf.setAttribute('aria-hidden', 'true');
  bottomHalf.setAttribute('aria-hidden', 'true');

  // Hide original text visually but keep it for accessibility
  const origColor = element.style.color;
  element.style.color = 'transparent';

  element.appendChild(topHalf);
  element.appendChild(bottomHalf);

  const controller: AnimationController = {
    play() {
      playing = true;
      topHalf.style.transform = `translate(-${distance}px, -${distance}px)`;
      bottomHalf.style.transform = `translate(${distance}px, ${distance}px)`;
      topHalf.style.opacity = '0.6';
      bottomHalf.style.opacity = '0.6';
    },
    pause() {
      playing = false;
      // Freeze current state by removing transition temporarily
      const topTransform = getComputedStyle(topHalf).transform;
      const bottomTransform = getComputedStyle(bottomHalf).transform;
      topHalf.style.transition = 'none';
      bottomHalf.style.transition = 'none';
      topHalf.style.transform = topTransform;
      bottomHalf.style.transform = bottomTransform;
      // Restore transitions on next frame
      requestAnimationFrame(() => {
        topHalf.style.transition = sharedStyle.transition!;
        bottomHalf.style.transition = sharedStyle.transition!;
      });
    },
    reset() {
      playing = false;
      topHalf.style.transform = 'translate(0, 0)';
      bottomHalf.style.transform = 'translate(0, 0)';
      topHalf.style.opacity = '1';
      bottomHalf.style.opacity = '1';
    },
    destroy() {
      playing = false;
      element.style.color = origColor;
      if (originalPosition === 'static') {
        element.style.position = '';
      }
      element.style.overflow = '';
      topHalf.remove();
      bottomHalf.remove();
    },
    get isPlaying() {
      return playing;
    },
  };

  return controller;
}
