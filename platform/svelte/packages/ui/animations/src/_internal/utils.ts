import type { AnimationController } from '../types';

/** Returns true if the user prefers reduced motion */
export function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined') return false;
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

/** A no-op controller for reduced motion or SSR contexts */
export function createNoopController(): AnimationController {
  return {
    play() {},
    pause() {},
    reset() {},
    destroy() {},
    get isPlaying() {
      return false;
    },
  };
}

let styleCounter = 0;

/**
 * Injects a <style> tag into the document head with an auto-generated ID.
 * Returns a cleanup function that removes the style element.
 */
export function injectStyle(css: string): () => void {
  if (typeof document === 'undefined') return () => {};

  const id = `sbx-anim-${++styleCounter}`;
  const style = document.createElement('style');
  style.id = id;
  style.textContent = css;
  document.head.appendChild(style);

  return () => {
    const el = document.getElementById(id);
    if (el) el.remove();
  };
}

/**
 * Wraps each character of an element's text content in individual spans.
 * Returns the spans and a cleanup function that restores the original content.
 */
export function wrapCharacters(element: HTMLElement): {
  chars: HTMLSpanElement[];
  restore: () => void;
} {
  const originalHTML = element.innerHTML;
  const text = element.textContent || '';
  const chars: HTMLSpanElement[] = [];

  element.innerHTML = '';

  for (let i = 0; i < text.length; i++) {
    const span = document.createElement('span');
    span.textContent = text[i];
    span.style.display = 'inline-block';
    span.setAttribute('aria-hidden', 'true');
    // Preserve spaces
    if (text[i] === ' ') {
      span.style.width = '0.3em';
    }
    chars.push(span);
    element.appendChild(span);
  }

  // Keep element accessible
  element.setAttribute('aria-label', text);

  return {
    chars,
    restore() {
      element.innerHTML = originalHTML;
      element.removeAttribute('aria-label');
    },
  };
}

/**
 * Creates a unique keyframes name and injects the CSS animation.
 * Returns the animation name and a cleanup function.
 */
export function createKeyframes(
  name: string,
  keyframeCSS: string,
): { animationName: string; cleanup: () => void } {
  const animationName = `sbx-${name}-${++styleCounter}`;
  const css = `@keyframes ${animationName} { ${keyframeCSS} }`;
  const cleanup = injectStyle(css);
  return { animationName, cleanup };
}
