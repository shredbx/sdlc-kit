import type { AnimationController, GlitchOptions } from '../types';
import { createNoopController, prefersReducedMotion } from '../_internal/utils';

/**
 * Glitch + slice combo with chromatic aberration.
 *
 * Combines diagonal clip-path splitting with red/cyan chromatic
 * aberration overlays using mix-blend-mode: screen.
 *
 * @example
 * ```ts
 * import { createGlitchSlice } from '@sbx/animations/text';
 * const ctrl = createGlitchSlice(document.querySelector('h1')!);
 * ctrl.play();
 * ```
 */
export function createGlitchSlice(
  element: HTMLElement,
  options: GlitchOptions = {},
): AnimationController {
  const {
    duration = 600,
    delay = 0,
    easing = 'cubic-bezier(0.23, 1, 0.32, 1)',
    intensity = 0.7,
    channelOffset = 4,
    chromatic = true,
    colors = { red: '#ef4444', cyan: '#22d3ee' },
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
  element.style.overflow = 'visible';

  const text = element.textContent || '';
  const computed = getComputedStyle(element);

  // Build shared style for overlay layers
  function makeLayer(): HTMLSpanElement {
    const span = document.createElement('span');
    Object.assign(span.style, {
      position: 'absolute',
      top: '0',
      left: '0',
      width: '100%',
      height: '100%',
      display: 'flex',
      alignItems: 'center',
      justifyContent: computed.textAlign === 'center' ? 'center' : 'flex-start',
      fontSize: computed.fontSize,
      fontWeight: computed.fontWeight,
      fontFamily: computed.fontFamily,
      letterSpacing: computed.letterSpacing,
      lineHeight: computed.lineHeight,
      pointerEvents: 'none',
      transition: `transform ${duration}ms ${easing} ${delay}ms, opacity ${duration}ms ${easing} ${delay}ms`,
    });
    span.textContent = text;
    span.setAttribute('aria-hidden', 'true');
    return span;
  }

  // Top half (clip-path polygon for upper diagonal)
  const topHalf = makeLayer();
  topHalf.style.color = computed.color;
  topHalf.style.clipPath = 'polygon(0 0, 100% 0, 100% 35%, 0 65%)';

  // Bottom half
  const bottomHalf = makeLayer();
  bottomHalf.style.color = computed.color;
  bottomHalf.style.clipPath = 'polygon(0 65%, 100% 35%, 100% 100%, 0 100%)';

  // Chromatic aberration layers
  const redLayer = makeLayer();
  redLayer.style.color = colors.red || '#ef4444';
  redLayer.style.clipPath = 'polygon(0 0, 100% 0, 100% 35%, 0 65%)';
  redLayer.style.mixBlendMode = 'screen';
  redLayer.style.opacity = '0';

  const cyanLayer = makeLayer();
  cyanLayer.style.color = colors.cyan || '#22d3ee';
  cyanLayer.style.clipPath = 'polygon(0 65%, 100% 35%, 100% 100%, 0 100%)';
  cyanLayer.style.mixBlendMode = 'screen';
  cyanLayer.style.opacity = '0';

  // Hide original text
  const origColor = element.style.color;
  element.style.color = 'transparent';

  element.appendChild(topHalf);
  element.appendChild(bottomHalf);
  if (chromatic) {
    element.appendChild(redLayer);
    element.appendChild(cyanLayer);
  }

  const dist = 16 * intensity;
  const offset = channelOffset;

  const controller: AnimationController = {
    play() {
      playing = true;
      // Split halves apart
      topHalf.style.transform = `translate(-${dist}px, -${dist}px)`;
      bottomHalf.style.transform = `translate(${dist}px, ${dist}px)`;
      topHalf.style.opacity = '0.7';
      bottomHalf.style.opacity = '0.7';

      if (chromatic) {
        // Chromatic aberration offset
        redLayer.style.opacity = '0.5';
        redLayer.style.transform = `translate(-${offset}px, -${offset / 2}px)`;
        cyanLayer.style.opacity = '0.5';
        cyanLayer.style.transform = `translate(${offset}px, ${offset / 2}px)`;
      }
    },
    pause() {
      playing = false;
    },
    reset() {
      playing = false;
      topHalf.style.transform = 'translate(0, 0)';
      bottomHalf.style.transform = 'translate(0, 0)';
      topHalf.style.opacity = '1';
      bottomHalf.style.opacity = '1';
      redLayer.style.opacity = '0';
      redLayer.style.transform = 'translate(0, 0)';
      cyanLayer.style.opacity = '0';
      cyanLayer.style.transform = 'translate(0, 0)';
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
      redLayer.remove();
      cyanLayer.remove();
    },
    get isPlaying() {
      return playing;
    },
  };

  return controller;
}
