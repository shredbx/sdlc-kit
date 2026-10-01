import type { AnimationController, GlitchOptions } from '../types';
import { createNoopController, prefersReducedMotion, injectStyle } from '../_internal/utils';

let glitchCounter = 0;

/**
 * Pure CSS glitch effect with rapid position jitter and chromatic aberration.
 *
 * Uses CSS @keyframes with pseudo-random steps for a jittery, broken-display
 * look. Text-shadow provides the chromatic aberration color split.
 *
 * @example
 * ```ts
 * import { createGlitch } from '@sbx/animations/text';
 * const ctrl = createGlitch(document.querySelector('h1')!, { intensity: 0.8 });
 * ctrl.play();
 * ```
 */
export function createGlitch(
  element: HTMLElement,
  options: GlitchOptions = {},
): AnimationController {
  const {
    duration = 2000,
    delay = 0,
    intensity = 0.6,
    channelOffset = 3,
    chromatic = true,
    colors = { red: '#ef4444', cyan: '#22d3ee' },
    reducedMotion = true,
  } = options;

  if (reducedMotion && prefersReducedMotion()) {
    return createNoopController();
  }

  let playing = false;
  const id = `sbx-glitch-${++glitchCounter}`;
  const jitter = Math.round(intensity * 6);
  const offset = channelOffset;

  // Generate pseudo-random keyframes for jitter
  const steps = 20;
  let jitterFrames = '';
  for (let i = 0; i <= steps; i++) {
    const pct = Math.round((i / steps) * 100);
    // Deterministic pseudo-random based on step index
    const x = Math.round(Math.sin(i * 7.3) * jitter);
    const y = Math.round(Math.cos(i * 4.1) * jitter);
    const shadowR = chromatic
      ? `${Math.round(Math.sin(i * 3.7) * offset)}px ${Math.round(Math.cos(i * 5.2) * offset)}px ${colors.red}`
      : 'none';
    const shadowC = chromatic
      ? `, ${Math.round(Math.cos(i * 2.9) * offset)}px ${Math.round(Math.sin(i * 6.1) * offset)}px ${colors.cyan}`
      : '';
    jitterFrames += `${pct}% { transform: translate(${x}px, ${y}px); text-shadow: ${shadowR}${shadowC}; }\n`;
  }

  const css = `
@keyframes ${id}-jitter {
  ${jitterFrames}
}
.${id}-active {
  animation: ${id}-jitter ${duration}ms steps(${steps}) infinite;
  animation-delay: ${delay}ms;
}
`;

  const removeStyle = injectStyle(css);
  element.classList.remove(`${id}-active`);

  const controller: AnimationController = {
    play() {
      playing = true;
      element.classList.add(`${id}-active`);
    },
    pause() {
      playing = false;
      element.classList.remove(`${id}-active`);
      // Freeze at current position by keeping last computed transform
    },
    reset() {
      playing = false;
      element.classList.remove(`${id}-active`);
      element.style.transform = '';
      element.style.textShadow = '';
    },
    destroy() {
      playing = false;
      element.classList.remove(`${id}-active`);
      element.style.transform = '';
      element.style.textShadow = '';
      removeStyle();
    },
    get isPlaying() {
      return playing;
    },
  };

  return controller;
}
