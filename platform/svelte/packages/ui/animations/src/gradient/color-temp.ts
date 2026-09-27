import type { AnimationController, GradientOptions } from '../types';
import { createNoopController, prefersReducedMotion, injectStyle } from '../_internal/utils';

let colorTempCounter = 0;

/**
 * Warm/cool color temperature shift.
 *
 * Smoothly transitions text color through a warm (reds, oranges, yellows)
 * to cool (blues, purples) spectrum using CSS hue rotation. Creates a
 * breathing color effect.
 *
 * @example
 * ```ts
 * import { createColorTemp } from '@sbx/animations/gradient';
 * const ctrl = createColorTemp(document.querySelector('h1')!);
 * ctrl.play();
 * ```
 */
export function createColorTemp(
  element: HTMLElement,
  options: GradientOptions = {},
): AnimationController {
  const {
    duration = 6000,
    delay = 0,
    colors = ['#ef4444', '#f59e0b', '#eab308', '#22c55e', '#22d3ee', '#6366f1', '#a78bfa', '#ef4444'],
    speed = 1,
    reducedMotion = true,
  } = options;

  if (reducedMotion && prefersReducedMotion()) {
    return createNoopController();
  }

  let playing = false;
  const id = `sbx-colortemp-${++colorTempCounter}`;
  const effectiveDuration = Math.round(duration / speed);

  const origColor = element.style.color;

  // Generate keyframes with explicit color stops
  const stepCount = colors.length;
  let keyframeCSS = '';
  colors.forEach((color, i) => {
    const pct = Math.round((i / (stepCount - 1)) * 100);
    keyframeCSS += `${pct}% { color: ${color}; }\n`;
  });

  const css = `
@keyframes ${id}-shift {
  ${keyframeCSS}
}
.${id}-active {
  animation: ${id}-shift ${effectiveDuration}ms ease-in-out infinite !important;
  animation-delay: ${delay}ms !important;
}
`;

  const removeStyle = injectStyle(css);

  const controller: AnimationController = {
    play() {
      playing = true;
      element.classList.add(`${id}-active`);
    },
    pause() {
      playing = false;
      const currentColor = getComputedStyle(element).color;
      element.classList.remove(`${id}-active`);
      element.style.color = currentColor;
    },
    reset() {
      playing = false;
      element.classList.remove(`${id}-active`);
      element.style.color = origColor;
    },
    destroy() {
      playing = false;
      element.classList.remove(`${id}-active`);
      element.style.color = origColor;
      removeStyle();
    },
    get isPlaying() {
      return playing;
    },
  };

  return controller;
}
