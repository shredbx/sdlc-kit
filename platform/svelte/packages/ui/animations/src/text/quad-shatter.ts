import type { AnimationController, TextSplitOptions } from '../types';
import { createNoopController, prefersReducedMotion } from '../_internal/utils';

/**
 * 4-piece shatter effect using clip-path quadrants.
 *
 * Splits an element into four quadrants (top-left, top-right,
 * bottom-left, bottom-right). On play each piece flies outward
 * in its respective corner direction.
 *
 * @example
 * ```ts
 * import { createQuadShatter } from '@sbx/animations/text';
 * const ctrl = createQuadShatter(document.querySelector('h1')!);
 * ctrl.play();
 * ```
 */
export function createQuadShatter(
  element: HTMLElement,
  options: TextSplitOptions = {},
): AnimationController {
  const {
    duration = 600,
    delay = 0,
    easing = 'cubic-bezier(0.23, 1, 0.32, 1)',
    distance = 24,
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

  const quadrants = [
    { clipPath: 'polygon(0 0, 50% 0, 50% 50%, 0 50%)', tx: -distance, ty: -distance },     // top-left
    { clipPath: 'polygon(50% 0, 100% 0, 100% 50%, 50% 50%)', tx: distance, ty: -distance },  // top-right
    { clipPath: 'polygon(0 50%, 50% 50%, 50% 100%, 0 100%)', tx: -distance, ty: distance },   // bottom-left
    { clipPath: 'polygon(50% 50%, 100% 50%, 100% 100%, 50% 100%)', tx: distance, ty: distance }, // bottom-right
  ];

  const layers: HTMLSpanElement[] = quadrants.map((q) => {
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
      fontWeight: '900',
      fontFamily: computed.fontFamily,
      letterSpacing: computed.letterSpacing,
      lineHeight: computed.lineHeight,
      color: computed.color,
      clipPath: q.clipPath,
      pointerEvents: 'none',
      transition: `transform ${duration}ms ${easing} ${delay}ms, opacity ${duration}ms ${easing} ${delay}ms`,
    });
    span.textContent = text;
    span.setAttribute('aria-hidden', 'true');
    return span;
  });

  const origColor = element.style.color;
  element.style.color = 'transparent';

  layers.forEach((l) => element.appendChild(l));

  const controller: AnimationController = {
    play() {
      playing = true;
      layers.forEach((layer, i) => {
        const q = quadrants[i];
        layer.style.transform = `translate(${q.tx}px, ${q.ty}px)`;
        layer.style.opacity = '0.5';
      });
    },
    pause() {
      playing = false;
    },
    reset() {
      playing = false;
      layers.forEach((layer) => {
        layer.style.transform = 'translate(0, 0)';
        layer.style.opacity = '1';
      });
    },
    destroy() {
      playing = false;
      element.style.color = origColor;
      if (originalPosition === 'static') {
        element.style.position = '';
      }
      element.style.overflow = '';
      layers.forEach((l) => l.remove());
    },
    get isPlaying() {
      return playing;
    },
  };

  return controller;
}
