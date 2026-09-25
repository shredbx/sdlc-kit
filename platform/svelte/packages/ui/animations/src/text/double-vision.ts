import type { AnimationController, DoubleVisionOptions } from '../types';
import { createNoopController, prefersReducedMotion } from '../_internal/utils';

/**
 * Offset duplicate text effect.
 *
 * Creates a semi-transparent duplicate overlay of the element's text,
 * offset by a configurable number of pixels with mix-blend-mode
 * for a double-vision / drunk effect.
 *
 * @example
 * ```ts
 * import { createDoubleVision } from '@sbx/animations/text';
 * const ctrl = createDoubleVision(document.querySelector('h1')!, { offsetX: 4 });
 * ctrl.play();
 * ```
 */
export function createDoubleVision(
  element: HTMLElement,
  options: DoubleVisionOptions = {},
): AnimationController {
  const {
    duration = 400,
    delay = 0,
    easing = 'cubic-bezier(0.4, 0, 0.2, 1)',
    offsetX = 4,
    offsetY = 0,
    overlayOpacity = 0.4,
    blendMode = 'screen',
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

  const text = element.textContent || '';
  const computed = getComputedStyle(element);

  const duplicate = document.createElement('span');
  Object.assign(duplicate.style, {
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
    color: computed.color,
    mixBlendMode: blendMode,
    opacity: '0',
    pointerEvents: 'none',
    transition: `transform ${duration}ms ${easing} ${delay}ms, opacity ${duration}ms ${easing} ${delay}ms`,
  });
  duplicate.textContent = text;
  duplicate.setAttribute('aria-hidden', 'true');

  element.appendChild(duplicate);

  const controller: AnimationController = {
    play() {
      playing = true;
      duplicate.style.transform = `translate(${offsetX}px, ${offsetY}px)`;
      duplicate.style.opacity = String(overlayOpacity);
    },
    pause() {
      playing = false;
    },
    reset() {
      playing = false;
      duplicate.style.transform = 'translate(0, 0)';
      duplicate.style.opacity = '0';
    },
    destroy() {
      playing = false;
      if (originalPosition === 'static') {
        element.style.position = '';
      }
      duplicate.remove();
    },
    get isPlaying() {
      return playing;
    },
  };

  return controller;
}
