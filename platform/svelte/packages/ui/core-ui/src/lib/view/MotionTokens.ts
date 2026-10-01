/**
 * MotionTokens - Animation Token Catalogue for IThemeOverrides
 *
 * Provides standardized motion patterns following Material Design 3 and WCAG 2.1.
 * Motion tokens define HOW elements animate, separate from WHEN (triggers).
 *
 * Architecture:
 *   Data (MotionTokens) → Transform (CSS/GSAP) → Result (Animated component)
 *
 * Standards:
 * - Material Design 3: Easing, duration, emphasis categories
 * - WCAG 2.1 SC 2.3.3: prefers-reduced-motion support
 * - CSS Transitions Level 1: timing functions
 *
 * Categories:
 * - ENTRANCE: Element enters viewport/DOM
 * - EXIT: Element leaves viewport/DOM
 * - EMPHASIS: Draw attention to state change
 * - TRANSITION: Continuous state transitions
 *
 * @example
 * // In IThemeOverrides
 * theme: {
 *   motion: {
 *     entrance: 'fade-in',
 *     exit: 'fade-out',
 *     emphasis: 'pulse'
 *   }
 * }
 *
 * @example
 * // Using motion resolver
 * const animation = resolveMotion('entrance', 'fade-up');
 * element.style.animation = motionToCSS(animation);
 */

// =============================================================================
// TYPES
// =============================================================================

/**
 * Motion category - when the animation plays.
 */
export type MotionCategory = 'entrance' | 'exit' | 'emphasis' | 'transition';

/**
 * Duration scale following Material Design 3.
 * - xs: 50ms - micro-interactions
 * - sm: 100ms - quick feedback
 * - md: 200ms - standard transitions
 * - lg: 300ms - complex transitions
 * - xl: 500ms - dramatic emphasis
 */
export type MotionDuration = 'xs' | 'sm' | 'md' | 'lg' | 'xl';

/**
 * Easing presets following Material Design 3.
 */
export type MotionEasing =
	| 'linear'
	| 'ease-in' // Acceleration
	| 'ease-out' // Deceleration (most common)
	| 'ease-in-out' // Acceleration then deceleration
	| 'emphasized' // Material Design emphasized
	| 'emphasized-decelerate' // Material Design emphasized decelerate
	| 'emphasized-accelerate' // Material Design emphasized accelerate
	| 'spring' // Spring physics approximation
	| 'bounce'; // Bouncy overshoot

/**
 * Entrance animation names.
 */
export type EntranceAnimation =
	| 'fade-in'
	| 'fade-up'
	| 'fade-down'
	| 'fade-left'
	| 'fade-right'
	| 'slide-up'
	| 'slide-down'
	| 'slide-left'
	| 'slide-right'
	| 'scale-in'
	| 'scale-up'
	| 'zoom-in'
	| 'expand-horizontal'
	| 'expand-vertical'
	| 'reveal'
	| 'pop'
	| 'none';

/**
 * Exit animation names (mirrors entrance).
 */
export type ExitAnimation =
	| 'fade-out'
	| 'fade-up-out'
	| 'fade-down-out'
	| 'fade-left-out'
	| 'fade-right-out'
	| 'slide-up-out'
	| 'slide-down-out'
	| 'slide-left-out'
	| 'slide-right-out'
	| 'scale-out'
	| 'scale-down'
	| 'zoom-out'
	| 'collapse-horizontal'
	| 'collapse-vertical'
	| 'hide'
	| 'pop-out'
	| 'none';

/**
 * Emphasis animation names.
 */
export type EmphasisAnimation =
	| 'pulse'
	| 'shake'
	| 'bounce'
	| 'wiggle'
	| 'flash'
	| 'glow'
	| 'heartbeat'
	| 'ring'
	| 'tada'
	| 'rubber-band'
	| 'none';

/**
 * Transition animation names (continuous state changes).
 */
export type TransitionAnimation =
	| 'crossfade'
	| 'morph'
	| 'slide'
	| 'flip'
	| 'rotate'
	| 'swap'
	| 'none';

/**
 * All animation names union.
 */
export type AnimationName =
	| EntranceAnimation
	| ExitAnimation
	| EmphasisAnimation
	| TransitionAnimation;

/**
 * Motion token definition.
 */
export interface IMotionToken {
	/** Animation name identifier */
	name: AnimationName;
	/** Category of motion */
	category: MotionCategory;
	/** Duration scale or milliseconds */
	duration: MotionDuration | number;
	/** Easing curve name or CSS easing string */
	easing: MotionEasing | string;
	/** Delay before animation (ms) */
	delay?: number;
	/** Number of iterations (Infinity for loop) */
	iterations?: number;
	/** Animation direction */
	direction?: 'normal' | 'reverse' | 'alternate' | 'alternate-reverse';
	/** Fill mode */
	fill?: 'none' | 'forwards' | 'backwards' | 'both';
	/** CSS keyframes or GSAP timeline */
	keyframes?: Keyframe[];
	/** Description for dev tools */
	description?: string;
}

/**
 * Keyframe for CSS animations.
 */
export interface Keyframe {
	offset?: number; // 0-1
	transform?: string;
	opacity?: number;
	[key: string]: string | number | undefined;
}

/**
 * Motion configuration for IThemeOverrides.
 */
export interface IMotionConfig {
	/** Entrance animation token */
	entrance?: EntranceAnimation;
	/** Exit animation token */
	exit?: ExitAnimation;
	/** Emphasis animation token */
	emphasis?: EmphasisAnimation;
	/** Transition animation token */
	transition?: TransitionAnimation;
	/** Override duration */
	duration?: MotionDuration | number;
	/** Override easing */
	easing?: MotionEasing | string;
	/** Override delay (ms) */
	delay?: number;
	/** Stagger delay for lists (ms) */
	stagger?: number;
	/** Disable all motion */
	disabled?: boolean;
	/** Respect prefers-reduced-motion */
	respectReducedMotion?: boolean;
}

// =============================================================================
// CONSTANTS
// =============================================================================

/**
 * Duration values in milliseconds.
 */
export const MOTION_DURATIONS: Record<MotionDuration, number> = {
	xs: 50,
	sm: 100,
	md: 200,
	lg: 300,
	xl: 500
} as const;

/**
 * Easing curves as CSS timing functions.
 */
export const MOTION_EASINGS: Record<MotionEasing, string> = {
	linear: 'linear',
	'ease-in': 'cubic-bezier(0.4, 0, 1, 1)',
	'ease-out': 'cubic-bezier(0, 0, 0.2, 1)',
	'ease-in-out': 'cubic-bezier(0.4, 0, 0.2, 1)',
	emphasized: 'cubic-bezier(0.2, 0, 0, 1)',
	'emphasized-decelerate': 'cubic-bezier(0.05, 0.7, 0.1, 1)',
	'emphasized-accelerate': 'cubic-bezier(0.3, 0, 0.8, 0.15)',
	spring: 'cubic-bezier(0.175, 0.885, 0.32, 1.275)',
	bounce: 'cubic-bezier(0.68, -0.55, 0.265, 1.55)'
} as const;

// =============================================================================
// TOKEN CATALOGUE
// =============================================================================

/**
 * Entrance animation tokens.
 */
export const ENTRANCE_TOKENS: Record<EntranceAnimation, IMotionToken> = {
	'fade-in': {
		name: 'fade-in',
		category: 'entrance',
		duration: 'md',
		easing: 'ease-out',
		description: 'Simple fade from transparent to opaque',
		keyframes: [{ offset: 0, opacity: 0 }, { offset: 1, opacity: 1 }]
	},
	'fade-up': {
		name: 'fade-up',
		category: 'entrance',
		duration: 'md',
		easing: 'ease-out',
		description: 'Fade in while sliding up from below',
		keyframes: [
			{ offset: 0, opacity: 0, transform: 'translateY(20px)' },
			{ offset: 1, opacity: 1, transform: 'translateY(0)' }
		]
	},
	'fade-down': {
		name: 'fade-down',
		category: 'entrance',
		duration: 'md',
		easing: 'ease-out',
		description: 'Fade in while sliding down from above',
		keyframes: [
			{ offset: 0, opacity: 0, transform: 'translateY(-20px)' },
			{ offset: 1, opacity: 1, transform: 'translateY(0)' }
		]
	},
	'fade-left': {
		name: 'fade-left',
		category: 'entrance',
		duration: 'md',
		easing: 'ease-out',
		description: 'Fade in while sliding from right to left',
		keyframes: [
			{ offset: 0, opacity: 0, transform: 'translateX(20px)' },
			{ offset: 1, opacity: 1, transform: 'translateX(0)' }
		]
	},
	'fade-right': {
		name: 'fade-right',
		category: 'entrance',
		duration: 'md',
		easing: 'ease-out',
		description: 'Fade in while sliding from left to right',
		keyframes: [
			{ offset: 0, opacity: 0, transform: 'translateX(-20px)' },
			{ offset: 1, opacity: 1, transform: 'translateX(0)' }
		]
	},
	'slide-up': {
		name: 'slide-up',
		category: 'entrance',
		duration: 'lg',
		easing: 'emphasized-decelerate',
		description: 'Slide up from below viewport',
		keyframes: [
			{ offset: 0, transform: 'translateY(100%)' },
			{ offset: 1, transform: 'translateY(0)' }
		]
	},
	'slide-down': {
		name: 'slide-down',
		category: 'entrance',
		duration: 'lg',
		easing: 'emphasized-decelerate',
		description: 'Slide down from above viewport',
		keyframes: [
			{ offset: 0, transform: 'translateY(-100%)' },
			{ offset: 1, transform: 'translateY(0)' }
		]
	},
	'slide-left': {
		name: 'slide-left',
		category: 'entrance',
		duration: 'lg',
		easing: 'emphasized-decelerate',
		description: 'Slide in from right edge',
		keyframes: [
			{ offset: 0, transform: 'translateX(100%)' },
			{ offset: 1, transform: 'translateX(0)' }
		]
	},
	'slide-right': {
		name: 'slide-right',
		category: 'entrance',
		duration: 'lg',
		easing: 'emphasized-decelerate',
		description: 'Slide in from left edge',
		keyframes: [
			{ offset: 0, transform: 'translateX(-100%)' },
			{ offset: 1, transform: 'translateX(0)' }
		]
	},
	'scale-in': {
		name: 'scale-in',
		category: 'entrance',
		duration: 'md',
		easing: 'ease-out',
		description: 'Scale up from 0 to full size',
		keyframes: [
			{ offset: 0, opacity: 0, transform: 'scale(0)' },
			{ offset: 1, opacity: 1, transform: 'scale(1)' }
		]
	},
	'scale-up': {
		name: 'scale-up',
		category: 'entrance',
		duration: 'md',
		easing: 'emphasized-decelerate',
		description: 'Scale from smaller to normal size',
		keyframes: [
			{ offset: 0, opacity: 0, transform: 'scale(0.95)' },
			{ offset: 1, opacity: 1, transform: 'scale(1)' }
		]
	},
	'zoom-in': {
		name: 'zoom-in',
		category: 'entrance',
		duration: 'lg',
		easing: 'spring',
		description: 'Zoom in with slight overshoot',
		keyframes: [
			{ offset: 0, opacity: 0, transform: 'scale(0.5)' },
			{ offset: 0.7, opacity: 1, transform: 'scale(1.05)' },
			{ offset: 1, opacity: 1, transform: 'scale(1)' }
		]
	},
	'expand-horizontal': {
		name: 'expand-horizontal',
		category: 'entrance',
		duration: 'md',
		easing: 'ease-out',
		description: 'Expand from center horizontally',
		keyframes: [
			{ offset: 0, transform: 'scaleX(0)' },
			{ offset: 1, transform: 'scaleX(1)' }
		]
	},
	'expand-vertical': {
		name: 'expand-vertical',
		category: 'entrance',
		duration: 'md',
		easing: 'ease-out',
		description: 'Expand from center vertically',
		keyframes: [
			{ offset: 0, transform: 'scaleY(0)' },
			{ offset: 1, transform: 'scaleY(1)' }
		]
	},
	reveal: {
		name: 'reveal',
		category: 'entrance',
		duration: 'lg',
		easing: 'emphasized',
		description: 'Reveal with clip-path wipe',
		keyframes: [
			{ offset: 0, 'clip-path': 'inset(0 100% 0 0)' },
			{ offset: 1, 'clip-path': 'inset(0 0 0 0)' }
		]
	},
	pop: {
		name: 'pop',
		category: 'entrance',
		duration: 'sm',
		easing: 'bounce',
		description: 'Pop in with bounce',
		keyframes: [
			{ offset: 0, opacity: 0, transform: 'scale(0.8)' },
			{ offset: 0.5, transform: 'scale(1.1)' },
			{ offset: 1, opacity: 1, transform: 'scale(1)' }
		]
	},
	none: {
		name: 'none',
		category: 'entrance',
		duration: 'xs',
		easing: 'linear',
		description: 'No animation',
		keyframes: []
	}
};

/**
 * Exit animation tokens (inverse of entrance).
 */
export const EXIT_TOKENS: Record<ExitAnimation, IMotionToken> = {
	'fade-out': {
		name: 'fade-out',
		category: 'exit',
		duration: 'md',
		easing: 'ease-in',
		description: 'Simple fade to transparent',
		keyframes: [{ offset: 0, opacity: 1 }, { offset: 1, opacity: 0 }]
	},
	'fade-up-out': {
		name: 'fade-up-out',
		category: 'exit',
		duration: 'md',
		easing: 'ease-in',
		description: 'Fade out while sliding up',
		keyframes: [
			{ offset: 0, opacity: 1, transform: 'translateY(0)' },
			{ offset: 1, opacity: 0, transform: 'translateY(-20px)' }
		]
	},
	'fade-down-out': {
		name: 'fade-down-out',
		category: 'exit',
		duration: 'md',
		easing: 'ease-in',
		description: 'Fade out while sliding down',
		keyframes: [
			{ offset: 0, opacity: 1, transform: 'translateY(0)' },
			{ offset: 1, opacity: 0, transform: 'translateY(20px)' }
		]
	},
	'fade-left-out': {
		name: 'fade-left-out',
		category: 'exit',
		duration: 'md',
		easing: 'ease-in',
		description: 'Fade out while sliding left',
		keyframes: [
			{ offset: 0, opacity: 1, transform: 'translateX(0)' },
			{ offset: 1, opacity: 0, transform: 'translateX(-20px)' }
		]
	},
	'fade-right-out': {
		name: 'fade-right-out',
		category: 'exit',
		duration: 'md',
		easing: 'ease-in',
		description: 'Fade out while sliding right',
		keyframes: [
			{ offset: 0, opacity: 1, transform: 'translateX(0)' },
			{ offset: 1, opacity: 0, transform: 'translateX(20px)' }
		]
	},
	'slide-up-out': {
		name: 'slide-up-out',
		category: 'exit',
		duration: 'lg',
		easing: 'emphasized-accelerate',
		description: 'Slide up out of viewport',
		keyframes: [
			{ offset: 0, transform: 'translateY(0)' },
			{ offset: 1, transform: 'translateY(-100%)' }
		]
	},
	'slide-down-out': {
		name: 'slide-down-out',
		category: 'exit',
		duration: 'lg',
		easing: 'emphasized-accelerate',
		description: 'Slide down out of viewport',
		keyframes: [
			{ offset: 0, transform: 'translateY(0)' },
			{ offset: 1, transform: 'translateY(100%)' }
		]
	},
	'slide-left-out': {
		name: 'slide-left-out',
		category: 'exit',
		duration: 'lg',
		easing: 'emphasized-accelerate',
		description: 'Slide out to left edge',
		keyframes: [
			{ offset: 0, transform: 'translateX(0)' },
			{ offset: 1, transform: 'translateX(-100%)' }
		]
	},
	'slide-right-out': {
		name: 'slide-right-out',
		category: 'exit',
		duration: 'lg',
		easing: 'emphasized-accelerate',
		description: 'Slide out to right edge',
		keyframes: [
			{ offset: 0, transform: 'translateX(0)' },
			{ offset: 1, transform: 'translateX(100%)' }
		]
	},
	'scale-out': {
		name: 'scale-out',
		category: 'exit',
		duration: 'md',
		easing: 'ease-in',
		description: 'Scale down to nothing',
		keyframes: [
			{ offset: 0, opacity: 1, transform: 'scale(1)' },
			{ offset: 1, opacity: 0, transform: 'scale(0)' }
		]
	},
	'scale-down': {
		name: 'scale-down',
		category: 'exit',
		duration: 'md',
		easing: 'emphasized-accelerate',
		description: 'Scale to smaller size',
		keyframes: [
			{ offset: 0, opacity: 1, transform: 'scale(1)' },
			{ offset: 1, opacity: 0, transform: 'scale(0.95)' }
		]
	},
	'zoom-out': {
		name: 'zoom-out',
		category: 'exit',
		duration: 'md',
		easing: 'ease-in',
		description: 'Zoom out with shrink',
		keyframes: [
			{ offset: 0, opacity: 1, transform: 'scale(1)' },
			{ offset: 1, opacity: 0, transform: 'scale(0.5)' }
		]
	},
	'collapse-horizontal': {
		name: 'collapse-horizontal',
		category: 'exit',
		duration: 'md',
		easing: 'ease-in',
		description: 'Collapse to center horizontally',
		keyframes: [
			{ offset: 0, transform: 'scaleX(1)' },
			{ offset: 1, transform: 'scaleX(0)' }
		]
	},
	'collapse-vertical': {
		name: 'collapse-vertical',
		category: 'exit',
		duration: 'md',
		easing: 'ease-in',
		description: 'Collapse to center vertically',
		keyframes: [
			{ offset: 0, transform: 'scaleY(1)' },
			{ offset: 1, transform: 'scaleY(0)' }
		]
	},
	hide: {
		name: 'hide',
		category: 'exit',
		duration: 'lg',
		easing: 'emphasized',
		description: 'Hide with clip-path wipe',
		keyframes: [
			{ offset: 0, 'clip-path': 'inset(0 0 0 0)' },
			{ offset: 1, 'clip-path': 'inset(0 100% 0 0)' }
		]
	},
	'pop-out': {
		name: 'pop-out',
		category: 'exit',
		duration: 'sm',
		easing: 'ease-in',
		description: 'Pop out with shrink',
		keyframes: [
			{ offset: 0, opacity: 1, transform: 'scale(1)' },
			{ offset: 0.5, transform: 'scale(1.1)' },
			{ offset: 1, opacity: 0, transform: 'scale(0.8)' }
		]
	},
	none: {
		name: 'none',
		category: 'exit',
		duration: 'xs',
		easing: 'linear',
		description: 'No animation',
		keyframes: []
	}
};

/**
 * Emphasis animation tokens.
 */
export const EMPHASIS_TOKENS: Record<EmphasisAnimation, IMotionToken> = {
	pulse: {
		name: 'pulse',
		category: 'emphasis',
		duration: 'lg',
		easing: 'ease-in-out',
		iterations: 2,
		description: 'Pulse scale to draw attention',
		keyframes: [
			{ offset: 0, transform: 'scale(1)' },
			{ offset: 0.5, transform: 'scale(1.05)' },
			{ offset: 1, transform: 'scale(1)' }
		]
	},
	shake: {
		name: 'shake',
		category: 'emphasis',
		duration: 'md',
		easing: 'ease-in-out',
		description: 'Shake horizontally (error indicator)',
		keyframes: [
			{ offset: 0, transform: 'translateX(0)' },
			{ offset: 0.1, transform: 'translateX(-5px)' },
			{ offset: 0.2, transform: 'translateX(5px)' },
			{ offset: 0.3, transform: 'translateX(-5px)' },
			{ offset: 0.4, transform: 'translateX(5px)' },
			{ offset: 0.5, transform: 'translateX(-5px)' },
			{ offset: 0.6, transform: 'translateX(5px)' },
			{ offset: 0.7, transform: 'translateX(-5px)' },
			{ offset: 0.8, transform: 'translateX(5px)' },
			{ offset: 0.9, transform: 'translateX(-5px)' },
			{ offset: 1, transform: 'translateX(0)' }
		]
	},
	bounce: {
		name: 'bounce',
		category: 'emphasis',
		duration: 'lg',
		easing: 'ease-out',
		description: 'Bounce vertically',
		keyframes: [
			{ offset: 0, transform: 'translateY(0)' },
			{ offset: 0.2, transform: 'translateY(-10px)' },
			{ offset: 0.4, transform: 'translateY(0)' },
			{ offset: 0.6, transform: 'translateY(-5px)' },
			{ offset: 0.8, transform: 'translateY(0)' },
			{ offset: 1, transform: 'translateY(0)' }
		]
	},
	wiggle: {
		name: 'wiggle',
		category: 'emphasis',
		duration: 'md',
		easing: 'ease-in-out',
		description: 'Wiggle rotation',
		keyframes: [
			{ offset: 0, transform: 'rotate(0deg)' },
			{ offset: 0.25, transform: 'rotate(-5deg)' },
			{ offset: 0.5, transform: 'rotate(5deg)' },
			{ offset: 0.75, transform: 'rotate(-5deg)' },
			{ offset: 1, transform: 'rotate(0deg)' }
		]
	},
	flash: {
		name: 'flash',
		category: 'emphasis',
		duration: 'sm',
		easing: 'ease-in-out',
		iterations: 2,
		description: 'Flash opacity',
		keyframes: [
			{ offset: 0, opacity: 1 },
			{ offset: 0.5, opacity: 0.5 },
			{ offset: 1, opacity: 1 }
		]
	},
	glow: {
		name: 'glow',
		category: 'emphasis',
		duration: 'xl',
		easing: 'ease-in-out',
		iterations: 2,
		description: 'Glow box-shadow pulse',
		keyframes: [
			{ offset: 0, 'box-shadow': '0 0 0 0 rgba(99, 102, 241, 0)' },
			{ offset: 0.5, 'box-shadow': '0 0 20px 5px rgba(99, 102, 241, 0.5)' },
			{ offset: 1, 'box-shadow': '0 0 0 0 rgba(99, 102, 241, 0)' }
		]
	},
	heartbeat: {
		name: 'heartbeat',
		category: 'emphasis',
		duration: 'lg',
		easing: 'ease-in-out',
		iterations: 2,
		description: 'Heartbeat double-pulse',
		keyframes: [
			{ offset: 0, transform: 'scale(1)' },
			{ offset: 0.14, transform: 'scale(1.3)' },
			{ offset: 0.28, transform: 'scale(1)' },
			{ offset: 0.42, transform: 'scale(1.3)' },
			{ offset: 0.7, transform: 'scale(1)' },
			{ offset: 1, transform: 'scale(1)' }
		]
	},
	ring: {
		name: 'ring',
		category: 'emphasis',
		duration: 'md',
		easing: 'ease-in-out',
		description: 'Ring bell motion',
		keyframes: [
			{ offset: 0, transform: 'rotate(0deg)' },
			{ offset: 0.1, transform: 'rotate(15deg)' },
			{ offset: 0.2, transform: 'rotate(-15deg)' },
			{ offset: 0.3, transform: 'rotate(10deg)' },
			{ offset: 0.4, transform: 'rotate(-10deg)' },
			{ offset: 0.5, transform: 'rotate(5deg)' },
			{ offset: 0.6, transform: 'rotate(-5deg)' },
			{ offset: 1, transform: 'rotate(0deg)' }
		]
	},
	tada: {
		name: 'tada',
		category: 'emphasis',
		duration: 'lg',
		easing: 'ease-in-out',
		description: 'Tada celebration',
		keyframes: [
			{ offset: 0, transform: 'scale(1) rotate(0deg)' },
			{ offset: 0.1, transform: 'scale(0.9) rotate(-3deg)' },
			{ offset: 0.2, transform: 'scale(0.9) rotate(-3deg)' },
			{ offset: 0.3, transform: 'scale(1.1) rotate(3deg)' },
			{ offset: 0.4, transform: 'scale(1.1) rotate(-3deg)' },
			{ offset: 0.5, transform: 'scale(1.1) rotate(3deg)' },
			{ offset: 0.6, transform: 'scale(1.1) rotate(-3deg)' },
			{ offset: 0.7, transform: 'scale(1.1) rotate(3deg)' },
			{ offset: 0.8, transform: 'scale(1.1) rotate(-3deg)' },
			{ offset: 0.9, transform: 'scale(1.1) rotate(3deg)' },
			{ offset: 1, transform: 'scale(1) rotate(0deg)' }
		]
	},
	'rubber-band': {
		name: 'rubber-band',
		category: 'emphasis',
		duration: 'lg',
		easing: 'ease-out',
		description: 'Rubber band stretch',
		keyframes: [
			{ offset: 0, transform: 'scale(1)' },
			{ offset: 0.3, transform: 'scaleX(1.25) scaleY(0.75)' },
			{ offset: 0.4, transform: 'scaleX(0.75) scaleY(1.25)' },
			{ offset: 0.5, transform: 'scaleX(1.15) scaleY(0.85)' },
			{ offset: 0.65, transform: 'scaleX(0.95) scaleY(1.05)' },
			{ offset: 0.75, transform: 'scaleX(1.05) scaleY(0.95)' },
			{ offset: 1, transform: 'scale(1)' }
		]
	},
	none: {
		name: 'none',
		category: 'emphasis',
		duration: 'xs',
		easing: 'linear',
		description: 'No animation',
		keyframes: []
	}
};

/**
 * Transition animation tokens.
 */
export const TRANSITION_TOKENS: Record<TransitionAnimation, IMotionToken> = {
	crossfade: {
		name: 'crossfade',
		category: 'transition',
		duration: 'md',
		easing: 'ease-in-out',
		description: 'Crossfade between states',
		keyframes: []
	},
	morph: {
		name: 'morph',
		category: 'transition',
		duration: 'lg',
		easing: 'emphasized',
		description: 'Morph shape/position',
		keyframes: []
	},
	slide: {
		name: 'slide',
		category: 'transition',
		duration: 'lg',
		easing: 'emphasized-decelerate',
		description: 'Slide between positions',
		keyframes: []
	},
	flip: {
		name: 'flip',
		category: 'transition',
		duration: 'lg',
		easing: 'ease-in-out',
		description: '3D flip transition',
		keyframes: [
			{ offset: 0, transform: 'perspective(400px) rotateY(0deg)' },
			{ offset: 0.5, transform: 'perspective(400px) rotateY(90deg)' },
			{ offset: 1, transform: 'perspective(400px) rotateY(180deg)' }
		]
	},
	rotate: {
		name: 'rotate',
		category: 'transition',
		duration: 'lg',
		easing: 'ease-in-out',
		description: 'Rotate between states',
		keyframes: [
			{ offset: 0, transform: 'rotate(0deg)' },
			{ offset: 1, transform: 'rotate(360deg)' }
		]
	},
	swap: {
		name: 'swap',
		category: 'transition',
		duration: 'md',
		easing: 'ease-out',
		description: 'Swap position with another element',
		keyframes: []
	},
	none: {
		name: 'none',
		category: 'transition',
		duration: 'xs',
		easing: 'linear',
		description: 'No animation',
		keyframes: []
	}
};

/**
 * All tokens combined.
 */
export const MOTION_TOKENS = {
	entrance: ENTRANCE_TOKENS,
	exit: EXIT_TOKENS,
	emphasis: EMPHASIS_TOKENS,
	transition: TRANSITION_TOKENS
} as const;

// =============================================================================
// RESOLVER FUNCTIONS
// =============================================================================

/**
 * Get motion token by category and name.
 */
export function getMotionToken(category: MotionCategory, name: AnimationName): IMotionToken | undefined {
	const categoryTokens = MOTION_TOKENS[category];
	return categoryTokens?.[name as keyof typeof categoryTokens];
}

/**
 * Resolve duration value to milliseconds.
 */
export function resolveDuration(duration: MotionDuration | number): number {
	return typeof duration === 'number' ? duration : MOTION_DURATIONS[duration];
}

/**
 * Resolve easing to CSS timing function.
 */
export function resolveEasing(easing: MotionEasing | string): string {
	return MOTION_EASINGS[easing as MotionEasing] ?? easing;
}

/**
 * Convert motion token to CSS animation shorthand.
 */
export function motionToCSS(
	token: IMotionToken,
	overrides?: Partial<Pick<IMotionToken, 'duration' | 'easing' | 'delay' | 'iterations' | 'direction' | 'fill'>>
): string {
	if (token.name === 'none' || !token.keyframes?.length) {
		return 'none';
	}

	const duration = resolveDuration(overrides?.duration ?? token.duration);
	const easing = resolveEasing(overrides?.easing ?? token.easing);
	const delay = overrides?.delay ?? token.delay ?? 0;
	const iterations = overrides?.iterations ?? token.iterations ?? 1;
	const direction = overrides?.direction ?? token.direction ?? 'normal';
	const fill = overrides?.fill ?? token.fill ?? 'both';

	// CSS animation: name duration easing delay iterations direction fill
	return `motion-${token.name} ${duration}ms ${easing} ${delay}ms ${iterations === Infinity ? 'infinite' : iterations} ${direction} ${fill}`;
}

/**
 * Generate CSS @keyframes rule for a motion token.
 */
export function motionToKeyframes(token: IMotionToken): string {
	if (token.name === 'none' || !token.keyframes?.length) {
		return '';
	}

	const frames = token.keyframes.map((kf) => {
		const offset = kf.offset ?? 0;
		const props = Object.entries(kf)
			.filter(([key]) => key !== 'offset')
			.map(([key, value]) => `${key}: ${value}`)
			.join('; ');
		return `${offset * 100}% { ${props} }`;
	});

	return `@keyframes motion-${token.name} {\n  ${frames.join('\n  ')}\n}`;
}

/**
 * Generate all keyframes CSS for injection.
 */
export function generateAllKeyframesCSS(): string {
	const allTokens = [
		...Object.values(ENTRANCE_TOKENS),
		...Object.values(EXIT_TOKENS),
		...Object.values(EMPHASIS_TOKENS),
		...Object.values(TRANSITION_TOKENS)
	];

	return allTokens
		.map(motionToKeyframes)
		.filter(Boolean)
		.join('\n\n');
}

/**
 * Convert motion config to CSS custom properties.
 */
export function motionConfigToCSS(config: IMotionConfig): string {
	const props: string[] = [];

	if (config.entrance && config.entrance !== 'none') {
		const token = ENTRANCE_TOKENS[config.entrance];
		props.push(`--motion-entrance: ${motionToCSS(token, { duration: config.duration, easing: config.easing, delay: config.delay })}`);
	}

	if (config.exit && config.exit !== 'none') {
		const token = EXIT_TOKENS[config.exit];
		props.push(`--motion-exit: ${motionToCSS(token, { duration: config.duration, easing: config.easing, delay: config.delay })}`);
	}

	if (config.emphasis && config.emphasis !== 'none') {
		const token = EMPHASIS_TOKENS[config.emphasis];
		props.push(`--motion-emphasis: ${motionToCSS(token, { duration: config.duration, easing: config.easing, delay: config.delay })}`);
	}

	if (config.stagger !== undefined) {
		props.push(`--motion-stagger: ${config.stagger}ms`);
	}

	return props.join(';\n');
}

// =============================================================================
// REDUCED MOTION SUPPORT
// =============================================================================

/**
 * Check if user prefers reduced motion.
 */
export function prefersReducedMotion(): boolean {
	if (typeof window === 'undefined') return false;
	return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

/**
 * Apply motion token respecting reduced motion preference.
 */
export function applyMotion(
	token: IMotionToken,
	config?: IMotionConfig
): IMotionToken {
	const shouldReduce = config?.respectReducedMotion !== false && prefersReducedMotion();

	if (shouldReduce || config?.disabled) {
		return {
			...token,
			duration: 'xs',
			keyframes: []
		};
	}

	return token;
}

// =============================================================================
// DEFAULT CONFIGURATION
// =============================================================================

/**
 * Default motion configuration.
 */
export const DEFAULT_MOTION_CONFIG: IMotionConfig = {
	entrance: 'fade-up',
	exit: 'fade-out',
	emphasis: 'pulse',
	transition: 'crossfade',
	duration: 'md',
	easing: 'ease-out',
	delay: 0,
	respectReducedMotion: true
};
