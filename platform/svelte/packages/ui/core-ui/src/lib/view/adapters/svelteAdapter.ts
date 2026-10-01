/**
 * Svelte Adapter - Transform Activities to Svelte Motion/Transition Configs
 *
 * Converts IActivity definitions to Svelte-native output:
 * - Motion activities → spring/tweened configurations
 * - Transitions → fade, fly, slide, scale, blur configs
 * - Presets → Theme context values
 *
 * Architecture:
 *   IActivity (IR) → svelteAdapter → Svelte config objects
 *
 * Features:
 * - Native Svelte 5 Spring/Tween classes
 * - Built-in transition parameter generation
 * - Reduced motion support via prefersReducedMotion
 * - Type-safe configurations
 *
 * @example
 * // Generate spring config
 * const spring = activityToSpring(bounceActivity);
 * // Output: { stiffness: 0.3, damping: 0.8 }
 *
 * @example
 * // Generate transition config
 * const flyConfig = activityToTransition(fadeUpActivity);
 * // Output: { y: 20, duration: 200, easing: cubicOut }
 */

import type { IActivity, ISpringConfig } from '../ActivityRegistry';
import { MOTION_DURATIONS, MOTION_EASINGS, prefersReducedMotion } from '../ActivityRegistry';
import { cubicOut, cubicIn, cubicInOut, linear, backOut, elasticOut } from 'svelte/easing';
import type { EasingFunction, TransitionConfig } from 'svelte/transition';

// =============================================================================
// TYPES
// =============================================================================

/**
 * Svelte Spring configuration.
 */
export interface ISvelteSpringConfig {
	stiffness?: number;
	damping?: number;
	precision?: number;
}

/**
 * Svelte Tweened configuration.
 */
export interface ISvelteTweenConfig {
	duration?: number;
	delay?: number;
	easing?: EasingFunction;
}

/**
 * Svelte fly transition parameters.
 */
export interface IFlyParams {
	delay?: number;
	duration?: number;
	easing?: EasingFunction;
	x?: number;
	y?: number;
	opacity?: number;
}

/**
 * Svelte fade transition parameters.
 */
export interface IFadeParams {
	delay?: number;
	duration?: number;
	easing?: EasingFunction;
}

/**
 * Svelte slide transition parameters.
 */
export interface ISlideParams {
	delay?: number;
	duration?: number;
	easing?: EasingFunction;
	axis?: 'x' | 'y';
}

/**
 * Svelte scale transition parameters.
 */
export interface IScaleParams {
	delay?: number;
	duration?: number;
	easing?: EasingFunction;
	start?: number;
	opacity?: number;
}

/**
 * Union of all transition params.
 */
export type TransitionParams = IFlyParams | IFadeParams | ISlideParams | IScaleParams;

/**
 * Transition type mapping.
 */
export type TransitionType = 'fade' | 'fly' | 'slide' | 'scale' | 'blur' | 'draw' | 'none';

// =============================================================================
// EASING MAPPING
// =============================================================================

/**
 * Map activity easing to Svelte easing function.
 */
export function getEasingFunction(easing?: string): EasingFunction {
	switch (easing) {
		case 'linear':
			return linear;
		case 'ease-in':
			return cubicIn;
		case 'ease-out':
			return cubicOut;
		case 'ease-in-out':
			return cubicInOut;
		case 'emphasized':
		case 'emphasized-decelerate':
			return cubicOut;
		case 'emphasized-accelerate':
			return cubicIn;
		case 'spring':
			return backOut;
		case 'bounce':
			return elasticOut;
		default:
			return cubicOut;
	}
}

// =============================================================================
// SPRING GENERATION
// =============================================================================

/**
 * Convert activity to Svelte Spring configuration.
 *
 * @example
 * const config = activityToSpring(activity);
 * const value = new Spring(0, config);
 */
export function activityToSpring(
	activity: IActivity,
	options: { respectReducedMotion?: boolean } = {}
): ISvelteSpringConfig {
	// Handle reduced motion
	if (options.respectReducedMotion !== false && prefersReducedMotion()) {
		return { stiffness: 1, damping: 1, precision: 0.01 };
	}

	// Use spring config if defined
	if (activity.spring) {
		return {
			stiffness: activity.spring.stiffness ?? 0.3,
			damping: activity.spring.damping ?? 0.8,
			precision: activity.spring.precision ?? 0.01
		};
	}

	// Map easing to spring approximation
	const easing = activity.easing ?? 'ease-out';
	switch (easing) {
		case 'spring':
			return { stiffness: 0.15, damping: 0.7, precision: 0.01 };
		case 'bounce':
			return { stiffness: 0.2, damping: 0.5, precision: 0.01 };
		case 'ease-in':
			return { stiffness: 0.5, damping: 0.9, precision: 0.01 };
		case 'ease-out':
		default:
			return { stiffness: 0.3, damping: 0.8, precision: 0.01 };
	}
}

/**
 * Convert spring config to Svelte Spring parameters.
 */
export function springConfigToSvelte(spring: ISpringConfig): ISvelteSpringConfig {
	return {
		stiffness: spring.stiffness ?? 0.3,
		damping: spring.damping ?? 0.8,
		precision: spring.precision ?? 0.01
	};
}

// =============================================================================
// TWEEN GENERATION
// =============================================================================

/**
 * Convert activity to Svelte Tweened configuration.
 *
 * @example
 * const config = activityToTween(activity);
 * const value = new Tweened(0, config);
 */
export function activityToTween(
	activity: IActivity,
	options: { respectReducedMotion?: boolean } = {}
): ISvelteTweenConfig {
	// Handle reduced motion
	if (options.respectReducedMotion !== false && prefersReducedMotion()) {
		return { duration: 0, delay: 0, easing: linear };
	}

	const duration = activity.durationMs ?? MOTION_DURATIONS[activity.duration ?? 'md'];
	const delay = activity.delay ?? 0;
	const easing = getEasingFunction(activity.easing);

	return { duration, delay, easing };
}

// =============================================================================
// TRANSITION GENERATION
// =============================================================================

/**
 * Determine best Svelte transition for an activity.
 */
export function getTransitionType(activity: IActivity): TransitionType {
	const id = activity.id.toLowerCase();

	// Fade transitions
	if (id.includes('fade') && !id.includes('up') && !id.includes('down') &&
	    !id.includes('left') && !id.includes('right')) {
		return 'fade';
	}

	// Fly transitions (fade + direction)
	if (id.includes('fade') || id.includes('fly')) {
		return 'fly';
	}

	// Slide transitions
	if (id.includes('slide')) {
		return 'slide';
	}

	// Scale transitions
	if (id.includes('scale') || id.includes('zoom') || id.includes('pop')) {
		return 'scale';
	}

	// Default to fade
	return 'fade';
}

/**
 * Extract direction from activity keyframes.
 */
function getDirectionFromKeyframes(activity: IActivity): { x: number; y: number } {
	const keyframes = activity.keyframes;
	if (!keyframes || keyframes.length < 2) {
		return { x: 0, y: 0 };
	}

	const firstFrame = keyframes[0];
	const transform = firstFrame.transform;

	if (!transform) {
		return { x: 0, y: 0 };
	}

	// Parse translateX/Y from transform
	const yMatch = transform.match(/translateY\((-?\d+)(?:px|%)?\)/);
	const xMatch = transform.match(/translateX\((-?\d+)(?:px|%)?\)/);

	return {
		x: xMatch ? parseInt(xMatch[1], 10) : 0,
		y: yMatch ? parseInt(yMatch[1], 10) : 0
	};
}

/**
 * Extract scale from activity keyframes.
 */
function getScaleFromKeyframes(activity: IActivity): number {
	const keyframes = activity.keyframes;
	if (!keyframes || keyframes.length < 2) {
		return 0;
	}

	const firstFrame = keyframes[0];
	const transform = firstFrame.transform;

	if (!transform) {
		return 0;
	}

	// Parse scale from transform
	const scaleMatch = transform.match(/scale\(([0-9.]+)\)/);
	if (scaleMatch) {
		return parseFloat(scaleMatch[1]);
	}

	return 0;
}

/**
 * Convert activity to Svelte transition parameters.
 *
 * @example
 * // In Svelte component
 * <div transition:fly={activityToFlyParams(activity)}>
 */
export function activityToFlyParams(
	activity: IActivity,
	options: { respectReducedMotion?: boolean } = {}
): IFlyParams {
	// Handle reduced motion
	if (options.respectReducedMotion !== false && prefersReducedMotion()) {
		return { duration: 0, x: 0, y: 0, opacity: 1 };
	}

	const duration = activity.durationMs ?? MOTION_DURATIONS[activity.duration ?? 'md'];
	const delay = activity.delay ?? 0;
	const easing = getEasingFunction(activity.easing);
	const { x, y } = getDirectionFromKeyframes(activity);

	// Get initial opacity from keyframes
	const firstFrame = activity.keyframes?.[0];
	const opacity = firstFrame?.opacity ?? 0;

	return { delay, duration, easing, x, y, opacity };
}

/**
 * Convert activity to Svelte fade parameters.
 */
export function activityToFadeParams(
	activity: IActivity,
	options: { respectReducedMotion?: boolean } = {}
): IFadeParams {
	// Handle reduced motion
	if (options.respectReducedMotion !== false && prefersReducedMotion()) {
		return { duration: 0 };
	}

	const duration = activity.durationMs ?? MOTION_DURATIONS[activity.duration ?? 'md'];
	const delay = activity.delay ?? 0;
	const easing = getEasingFunction(activity.easing);

	return { delay, duration, easing };
}

/**
 * Convert activity to Svelte slide parameters.
 */
export function activityToSlideParams(
	activity: IActivity,
	options: { respectReducedMotion?: boolean } = {}
): ISlideParams {
	// Handle reduced motion
	if (options.respectReducedMotion !== false && prefersReducedMotion()) {
		return { duration: 0, axis: 'y' };
	}

	const duration = activity.durationMs ?? MOTION_DURATIONS[activity.duration ?? 'md'];
	const delay = activity.delay ?? 0;
	const easing = getEasingFunction(activity.easing);

	// Determine axis from activity name/keyframes
	const { x, y } = getDirectionFromKeyframes(activity);
	const axis = Math.abs(x) > Math.abs(y) ? 'x' : 'y';

	return { delay, duration, easing, axis };
}

/**
 * Convert activity to Svelte scale parameters.
 */
export function activityToScaleParams(
	activity: IActivity,
	options: { respectReducedMotion?: boolean } = {}
): IScaleParams {
	// Handle reduced motion
	if (options.respectReducedMotion !== false && prefersReducedMotion()) {
		return { duration: 0, start: 1, opacity: 1 };
	}

	const duration = activity.durationMs ?? MOTION_DURATIONS[activity.duration ?? 'md'];
	const delay = activity.delay ?? 0;
	const easing = getEasingFunction(activity.easing);
	const start = getScaleFromKeyframes(activity);

	// Get initial opacity
	const firstFrame = activity.keyframes?.[0];
	const opacity = firstFrame?.opacity ?? 0;

	return { delay, duration, easing, start, opacity };
}

/**
 * Convert activity to appropriate Svelte transition parameters.
 *
 * Automatically selects the best transition type based on the activity.
 */
export function activityToTransition(
	activity: IActivity,
	options: { respectReducedMotion?: boolean } = {}
): { type: TransitionType; params: TransitionParams } {
	const type = getTransitionType(activity);

	switch (type) {
		case 'fly':
			return { type, params: activityToFlyParams(activity, options) };
		case 'slide':
			return { type, params: activityToSlideParams(activity, options) };
		case 'scale':
			return { type, params: activityToScaleParams(activity, options) };
		case 'fade':
		default:
			return { type: 'fade', params: activityToFadeParams(activity, options) };
	}
}

// =============================================================================
// TRANSITION CONFIG GENERATION
// =============================================================================

/**
 * Generate TransitionConfig for Svelte custom transitions.
 *
 * @example
 * // Custom transition function
 * function myTransition(node, params) {
 *   return activityToTransitionConfig(activity);
 * }
 */
export function activityToTransitionConfig(
	activity: IActivity,
	options: { respectReducedMotion?: boolean } = {}
): TransitionConfig {
	// Handle reduced motion
	if (options.respectReducedMotion !== false && prefersReducedMotion()) {
		return { duration: 0 };
	}

	const duration = activity.durationMs ?? MOTION_DURATIONS[activity.duration ?? 'md'];
	const delay = activity.delay ?? 0;
	const easing = getEasingFunction(activity.easing);

	// Generate CSS from keyframes
	let css: ((t: number, u: number) => string) | undefined;

	if (activity.keyframes && activity.keyframes.length >= 2) {
		css = (t: number) => {
			// Interpolate between keyframes
			const frames = activity.keyframes!;
			const segmentCount = frames.length - 1;

			// Find which segment t falls into
			const segment = Math.min(Math.floor(t * segmentCount), segmentCount - 1);
			const segmentT = (t * segmentCount) - segment;

			const startFrame = frames[segment];
			const endFrame = frames[segment + 1];

			const props: string[] = [];

			// Interpolate opacity
			if (startFrame.opacity !== undefined && endFrame.opacity !== undefined) {
				const opacity = startFrame.opacity + (endFrame.opacity - startFrame.opacity) * segmentT;
				props.push(`opacity: ${opacity}`);
			}

			// Interpolate transform (simplified - only handles basic cases)
			if (startFrame.transform && endFrame.transform) {
				// For now, just use the interpolated value based on t
				props.push(`transform: ${t < 0.5 ? startFrame.transform : endFrame.transform}`);
			}

			return props.join('; ');
		};
	}

	return { delay, duration, easing, css };
}

// =============================================================================
// PRESET INTEGRATION
// =============================================================================

/**
 * Theme context value for Svelte context API.
 */
export interface ISvelteThemeContext {
	mode: 'light' | 'dark';
	motion: {
		entrance: ISvelteTweenConfig;
		exit: ISvelteTweenConfig;
		emphasis: ISvelteTweenConfig;
		spring: ISvelteSpringConfig;
		enabled: boolean;
	};
	transitions: {
		fade: IFadeParams;
		fly: IFlyParams;
		slide: ISlideParams;
		scale: IScaleParams;
	};
}

/**
 * Generate Svelte theme context from preset.
 */
export function presetToThemeContext(
	preset: {
		mode: 'light' | 'dark';
		motion?: {
			entrance?: string;
			exit?: string;
			emphasis?: string;
			duration?: string;
			easing?: string;
			reducedMotion?: boolean;
		};
	},
	options: { respectReducedMotion?: boolean } = {}
): ISvelteThemeContext {
	const shouldDisable = preset.motion?.reducedMotion ||
		(options.respectReducedMotion !== false && prefersReducedMotion());

	const duration = preset.motion?.duration
		? MOTION_DURATIONS[preset.motion.duration as keyof typeof MOTION_DURATIONS] ?? 200
		: 200;

	const easing = getEasingFunction(preset.motion?.easing);

	const tweenConfig: ISvelteTweenConfig = shouldDisable
		? { duration: 0, delay: 0, easing: linear }
		: { duration, delay: 0, easing };

	const springConfig: ISvelteSpringConfig = shouldDisable
		? { stiffness: 1, damping: 1, precision: 0.01 }
		: { stiffness: 0.3, damping: 0.8, precision: 0.01 };

	return {
		mode: preset.mode,
		motion: {
			entrance: tweenConfig,
			exit: tweenConfig,
			emphasis: tweenConfig,
			spring: springConfig,
			enabled: !shouldDisable
		},
		transitions: {
			fade: { duration, easing },
			fly: { duration, easing, x: 0, y: 20, opacity: 0 },
			slide: { duration, easing, axis: 'y' },
			scale: { duration, easing, start: 0.95, opacity: 0 }
		}
	};
}

// =============================================================================
// EXPORTS
// =============================================================================

export {
	cubicOut,
	cubicIn,
	cubicInOut,
	linear,
	backOut,
	elasticOut
} from 'svelte/easing';
