import type { TransitionType } from './tokens.js';

/** Controller returned by all animation factory functions */
export interface AnimationController {
  /** Start the animation */
  play(): void;
  /** Pause the animation */
  pause(): void;
  /** Reset to initial state */
  reset(): void;
  /** Clean up DOM changes and listeners */
  destroy(): void;
  /** Whether animation is currently playing */
  readonly isPlaying: boolean;
}

/** Base options shared by all animations */
export interface BaseAnimationOptions {
  /** Duration in milliseconds */
  duration?: number;
  /** Delay before starting in milliseconds */
  delay?: number;
  /** CSS easing function */
  easing?: string;
  /** Respect prefers-reduced-motion */
  reducedMotion?: boolean;
}

export interface TextSplitOptions extends BaseAnimationOptions {
  /** Split angle in degrees (for diagonal-slice) */
  angle?: number;
  /** Split distance in pixels */
  distance?: number;
  /** Split direction */
  direction?: 'horizontal' | 'vertical' | 'diagonal';
}

export interface GlitchOptions extends BaseAnimationOptions {
  /** Glitch intensity 0-1 */
  intensity?: number;
  /** Color channel offset in pixels */
  channelOffset?: number;
  /** Whether to use chromatic aberration */
  chromatic?: boolean;
  /** Glitch colors */
  colors?: { red?: string; cyan?: string };
}

export interface GradientOptions extends BaseAnimationOptions {
  /** Gradient colors array */
  colors?: string[];
  /** Gradient angle in degrees */
  angle?: number;
  /** Animation speed multiplier */
  speed?: number;
}

export interface RevealOptions extends BaseAnimationOptions {
  /** Reveal motion style (Decision #0220 — named component per type). */
  type?: TransitionType;
  /** Translate distance in pixels. */
  distance?: number;
  /** Stagger delay between children in ms. */
  stagger?: number;
}

export interface HideoutOptions extends BaseAnimationOptions {
  /** Exit motion style (excludes stagger — Hideout is per-element). */
  type?: Exclude<TransitionType, 'stagger'>;
  /** Translate distance in pixels. */
  distance?: number;
}

export interface LazyOptions {
  /** IO root margin before instantiating slot. Default '200px'. */
  rootMargin?: string;
  /** Placeholder HTML string while waiting. Default: pulsing skeleton div. */
  placeholder?: string;
}

export interface BackgroundOptions extends BaseAnimationOptions {
  /** Opacity 0-1 */
  opacity?: number;
  /** Colors for the background animation */
  colors?: string[];
  /** Blur radius for mesh gradient */
  blur?: number;
}

export interface WeightShiftOptions extends BaseAnimationOptions {
  /** Starting font-weight */
  weightFrom?: number;
  /** Target font-weight */
  weightTo?: number;
  /** Stagger delay per character in ms */
  stagger?: number;
}

export interface DoubleVisionOptions extends BaseAnimationOptions {
  /** Horizontal offset in pixels */
  offsetX?: number;
  /** Vertical offset in pixels */
  offsetY?: number;
  /** Overlay opacity 0-1 */
  overlayOpacity?: number;
  /** Mix blend mode for the duplicate */
  blendMode?: string;
}

export interface CharRevealOptions extends BaseAnimationOptions {
  /** Stagger delay per character in ms */
  stagger?: number;
  /** Vertical offset for entrance in pixels */
  offsetY?: number;
}

export interface DotGridOptions extends BaseAnimationOptions {
  /** Grid spacing in pixels */
  spacing?: number;
  /** Dot color */
  color?: string;
  /** Whether to fade edges with a mask */
  fadeMask?: boolean;
}
