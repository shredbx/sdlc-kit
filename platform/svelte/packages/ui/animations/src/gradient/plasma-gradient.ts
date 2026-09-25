import type { AnimationController, GradientOptions } from '../types';
import { createNoopController, prefersReducedMotion, injectStyle } from '../_internal/utils';

let plasmaCounter = 0;

/**
 * Animated gradient text fill.
 *
 * Applies an animated multi-stop linear-gradient as the text fill color
 * using background-clip: text. The gradient rotates through its stops,
 * creating a plasma-like flowing effect.
 *
 * @example
 * ```ts
 * import { createPlasmaGradient } from '@sbx/animations/gradient';
 * const ctrl = createPlasmaGradient(document.querySelector('h1')!);
 * ctrl.play();
 * ```
 */
export function createPlasmaGradient(
  element: HTMLElement,
  options: GradientOptions = {},
): AnimationController {
  const {
    duration = 4000,
    delay = 0,
    colors = ['#ef4444', '#f59e0b', '#22d3ee', '#a78bfa', '#ef4444'],
    angle = 90,
    speed = 1,
    reducedMotion = true,
  } = options;

  if (reducedMotion && prefersReducedMotion()) {
    return createNoopController();
  }

  let playing = false;
  const id = `sbx-plasma-${++plasmaCounter}`;
  const effectiveDuration = Math.round(duration / speed);

  // Save original styles
  const origStyles = {
    background: element.style.background,
    backgroundClip: element.style.backgroundClip,
    webkitBackgroundClip: (element.style as any).webkitBackgroundClip,
    webkitTextFillColor: (element.style as any).webkitTextFillColor,
    color: element.style.color,
    backgroundSize: element.style.backgroundSize,
  };

  const colorStops = colors.join(', ');

  // Keyframes: shift background-position across 200% width
  const css = `
@keyframes ${id}-flow {
  0% { background-position: 0% 50%; }
  50% { background-position: 100% 50%; }
  100% { background-position: 0% 50%; }
}
.${id}-active {
  background: linear-gradient(${angle}deg, ${colorStops}) !important;
  background-size: 200% 200% !important;
  -webkit-background-clip: text !important;
  background-clip: text !important;
  -webkit-text-fill-color: transparent !important;
  animation: ${id}-flow ${effectiveDuration}ms ease infinite !important;
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
      element.style.animationPlayState = 'paused';
    },
    reset() {
      playing = false;
      element.classList.remove(`${id}-active`);
      element.style.animationPlayState = '';
      // Restore originals
      Object.assign(element.style, origStyles);
    },
    destroy() {
      playing = false;
      element.classList.remove(`${id}-active`);
      element.style.animationPlayState = '';
      Object.assign(element.style, origStyles);
      removeStyle();
    },
    get isPlaying() {
      return playing;
    },
  };

  return controller;
}
