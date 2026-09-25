/**
 * View Activity Adapters
 *
 * Adapters transform IActivity intermediate representation (IR) to
 * framework-specific output (CSS, Svelte, GSAP, Motion.dev).
 *
 * Pattern:
 *   ActivityRegistry → Adapter → Framework-specific output
 *
 * Available Adapters:
 * - cssAdapter: CSS keyframes + custom properties
 * - svelteAdapter: Svelte spring/transition configs
 * - (future) gsapAdapter: GSAP timeline configs
 * - (future) motionAdapter: Motion.dev configs
 */

// CSS Adapter
export {
	// Keyframe generation
	activityToKeyframes,
	activityToAnimation,
	generateAllKeyframes,

	// Style token generation
	styleTokensToCSS,
	styleActivityToCSS,

	// Preset generation
	presetToCSS,
	activityToClass,
	generateUtilityClasses,

	// Spring approximation
	springToCubicBezier,

	// Reduced motion
	generateReducedMotionCSS,

	// Types
	type IStylePreset,
	type ICSSOptions
} from './cssAdapter';

// Svelte Adapter
export {
	// Easing
	getEasingFunction,

	// Spring generation
	activityToSpring,
	springConfigToSvelte,

	// Tween generation
	activityToTween,

	// Transition generation
	getTransitionType,
	activityToFlyParams,
	activityToFadeParams,
	activityToSlideParams,
	activityToScaleParams,
	activityToTransition,
	activityToTransitionConfig,

	// Preset integration
	presetToThemeContext,

	// Svelte easing re-exports
	cubicOut,
	cubicIn,
	cubicInOut,
	linear,
	backOut,
	elasticOut,

	// Types
	type ISvelteSpringConfig,
	type ISvelteTweenConfig,
	type IFlyParams,
	type IFadeParams,
	type ISlideParams,
	type IScaleParams,
	type TransitionParams,
	type TransitionType,
	type ISvelteThemeContext
} from './svelteAdapter';
