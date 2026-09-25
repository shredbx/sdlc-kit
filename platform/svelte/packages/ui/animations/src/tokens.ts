/**
 * W3C Design Tokens for animation parameters.
 * Used as defaults across all animation functions.
 */
export const animationTokens = {
  duration: {
    instant: 50,
    fast: 100,
    normal: 200,
    slow: 400,
    slower: 600,
  },
  easing: {
    default: 'cubic-bezier(0.4, 0, 0.2, 1)',
    in: 'cubic-bezier(0.4, 0, 1, 1)',
    out: 'cubic-bezier(0, 0, 0.2, 1)',
    inOut: 'cubic-bezier(0.4, 0, 0.2, 1)',
    bounce: 'cubic-bezier(0.34, 1.56, 0.64, 1)',
    spring: 'cubic-bezier(0.22, 1, 0.36, 1)',
  },
  distance: {
    sm: 4,
    md: 8,
    lg: 16,
    xl: 24,
  },
} as const;

export type DurationToken = keyof typeof animationTokens.duration;
export type EasingToken = keyof typeof animationTokens.easing;
export type DistanceToken = keyof typeof animationTokens.distance;

// =============================================================================
// TRANSITION TOKENS (Decision #0220)
// =============================================================================

/** All scroll-reveal transition motion styles. */
export type TransitionType =
	| 'fade'
	| 'slide-up'
	| 'slide-down'
	| 'slide-left'
	| 'slide-right'
	| 'zoom'
	| 'stagger';

/** Named presets that brands reference. */
export type TransitionPresetName = 'minimal' | 'clean' | 'expressive';

/** Full preset descriptor consumed by brand config + RevealGroup. */
export interface TransitionPreset {
	name: string;
	/** Reveal motion style. */
	reveal: TransitionType;
	/** Duration in ms. */
	duration_ms: number;
	/** Easing token key. */
	easing: EasingToken;
	/** Translate distance in px (0 for fade). */
	distance_px: number;
	/** Per-child stagger delay in ms. */
	stagger_ms: number;
}

export const transitionPresets: Record<TransitionPresetName, TransitionPreset> = {
	minimal: {
		name: 'minimal',
		reveal: 'fade',
		duration_ms: 400,
		easing: 'out',
		distance_px: 0,
		stagger_ms: 60
	},
	clean: {
		name: 'clean',
		reveal: 'slide-up',
		duration_ms: 400,
		easing: 'out',
		distance_px: 24,
		stagger_ms: 60
	},
	expressive: {
		name: 'expressive',
		reveal: 'zoom',
		duration_ms: 350,
		easing: 'bounce',
		distance_px: 24,
		stagger_ms: 80
	}
} as const;

/** Z-index bands for stacking context governance. */
export const zTokens = {
	base: 0,
	content: 1,
	elevated: 10,
	overlay: 50,
	popover: 100,
	modal: 200,
	system: 300
} as const;

export type ZBand = keyof typeof zTokens;
