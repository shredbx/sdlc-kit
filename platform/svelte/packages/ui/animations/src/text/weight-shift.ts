import type { AnimationController, WeightShiftOptions } from '../types';
import { createNoopController, prefersReducedMotion, wrapCharacters, injectStyle } from '../_internal/utils';

let weightCounter = 0;

/**
 * Font-weight animation with per-character stagger.
 *
 * Animates font-weight from a light to heavy value (or vice versa)
 * with staggered timing per character. Works best with variable fonts
 * that support continuous weight axes.
 *
 * @example
 * ```ts
 * import { createWeightShift } from '@sbx/animations/text';
 * const ctrl = createWeightShift(document.querySelector('h1')!, { weightTo: 900 });
 * ctrl.play();
 * ```
 */
export function createWeightShift(
  element: HTMLElement,
  options: WeightShiftOptions = {},
): AnimationController {
  const {
    duration = 800,
    delay = 0,
    easing = 'cubic-bezier(0.4, 0, 0.2, 1)',
    weightFrom = 400,
    weightTo = 900,
    stagger = 40,
    reducedMotion = true,
  } = options;

  if (reducedMotion && prefersReducedMotion()) {
    return createNoopController();
  }

  let playing = false;
  const id = `sbx-weight-${++weightCounter}`;
  const { chars, restore } = wrapCharacters(element);

  // Create keyframes for weight oscillation
  const css = `
@keyframes ${id}-shift {
  0% { font-weight: ${weightFrom}; }
  50% { font-weight: ${weightTo}; }
  100% { font-weight: ${weightFrom}; }
}
`;
  const removeStyle = injectStyle(css);

  // Set up each character span
  chars.forEach((span) => {
    span.style.fontWeight = String(weightFrom);
    span.style.display = 'inline-block';
  });

  const controller: AnimationController = {
    play() {
      playing = true;
      chars.forEach((span, i) => {
        const charDelay = delay + i * stagger;
        span.style.animation = `${id}-shift ${duration}ms ${easing} ${charDelay}ms infinite`;
      });
    },
    pause() {
      playing = false;
      chars.forEach((span) => {
        // Freeze by capturing current computed weight
        const currentWeight = getComputedStyle(span).fontWeight;
        span.style.animation = 'none';
        span.style.fontWeight = currentWeight;
      });
    },
    reset() {
      playing = false;
      chars.forEach((span) => {
        span.style.animation = 'none';
        span.style.fontWeight = String(weightFrom);
      });
    },
    destroy() {
      playing = false;
      removeStyle();
      restore();
    },
    get isPlaying() {
      return playing;
    },
  };

  return controller;
}
