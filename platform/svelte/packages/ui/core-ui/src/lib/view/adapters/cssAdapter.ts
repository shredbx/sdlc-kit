/**
 * CSS Adapter - Transform Activities to CSS Keyframes and Custom Properties
 *
 * Converts IActivity definitions to CSS output:
 * - Motion activities → @keyframes rules + animation shorthand
 * - Style activities → CSS custom properties
 * - Presets → Complete CSS variable set
 *
 * Architecture:
 *   IActivity (IR) → cssAdapter → CSS string
 *
 * Features:
 * - GPU-accelerated keyframes (transform, opacity only)
 * - Reduced motion support
 * - Preset inheritance resolution
 * - CSS custom property generation
 *
 * @example
 * // Generate keyframes for an activity
 * const css = activityToKeyframes(fadeUpActivity);
 * // Output: @keyframes motion-fade-up { ... }
 *
 * @example
 * // Generate animation shorthand
 * const animation = activityToAnimation(fadeUpActivity);
 * // Output: motion-fade-up 200ms ease-out 0ms 1 normal both
 *
 * @example
 * // Generate preset CSS
 * const presetCSS = presetToCSS(hubDarkPreset);
 * // Output: :root { --color-primary: #6366f1; ... }
 */

import type { IActivity, IStyleToken, ISpringConfig } from '../ActivityRegistry';
import { MOTION_DURATIONS, MOTION_EASINGS, prefersReducedMotion } from '../ActivityRegistry';
import type { Keyframe } from '../MotionTokens';

// =============================================================================
// TYPES
// =============================================================================

/**
 * Style preset structure for CSS generation.
 */
export interface IStylePreset {
	id: string;
	name: string;
	mode: 'light' | 'dark';
	motion?: {
		entrance?: string;
		exit?: string;
		emphasis?: string;
		transition?: string;
		duration?: string;
		easing?: string;
		stagger?: number;
		reducedMotion?: boolean;
	};
	colors?: Record<string, string>;
	typography?: {
		fontFamily?: string;
		fontFamilyMono?: string;
		fontSizeBase?: string;
		lineHeight?: string;
		fontWeightNormal?: number;
		fontWeightBold?: number;
	};
	spacing?: Record<string, string>;
	effects?: {
		borderRadius?: string;
		borderRadiusLg?: string;
		shadowSm?: string;
		shadowMd?: string;
		shadowLg?: string;
		blur?: string;
	};
	extends?: string;
}

/**
 * CSS generation options.
 */
export interface ICSSOptions {
	/** Include @keyframes rules */
	includeKeyframes?: boolean;
	/** Include CSS custom properties */
	includeCustomProperties?: boolean;
	/** Prefix for custom properties (default: '') */
	propertyPrefix?: string;
	/** Selector for scoping (default: ':root') */
	selector?: string;
	/** Respect prefers-reduced-motion */
	respectReducedMotion?: boolean;
}

// =============================================================================
// KEYFRAME GENERATION
// =============================================================================

/**
 * Convert keyframe to CSS rule.
 */
function keyframeToCSS(keyframe: Keyframe): string {
	const offset = keyframe.offset ?? 0;
	const percentage = Math.round(offset * 100);

	const props = Object.entries(keyframe)
		.filter(([key]) => key !== 'offset' && keyframe[key] !== undefined)
		.map(([key, value]) => {
			// Convert camelCase to kebab-case
			const cssKey = key.replace(/([A-Z])/g, '-$1').toLowerCase();
			return `${cssKey}: ${value}`;
		})
		.join('; ');

	return `${percentage}% { ${props} }`;
}

/**
 * Generate @keyframes rule for an activity.
 */
export function activityToKeyframes(activity: IActivity): string {
	// No keyframes for style, state without stateKeyframes, or none activities
	if (!activity.keyframes || activity.keyframes.length === 0) {
		return '';
	}

	const keyframeName = `motion-${activity.id}`;
	const keyframeRules = activity.keyframes.map(keyframeToCSS);

	return `@keyframes ${keyframeName} {\n  ${keyframeRules.join('\n  ')}\n}`;
}

/**
 * Generate CSS animation shorthand for an activity.
 */
export function activityToAnimation(
	activity: IActivity,
	options: { respectReducedMotion?: boolean } = {}
): string {
	// Handle reduced motion
	if (options.respectReducedMotion !== false && prefersReducedMotion()) {
		return 'none';
	}

	// No animation for activities without keyframes
	if (!activity.keyframes || activity.keyframes.length === 0) {
		return 'none';
	}

	const name = `motion-${activity.id}`;
	const duration = activity.durationMs ?? MOTION_DURATIONS[activity.duration ?? 'md'];
	const easing = activity.easingCurve ?? MOTION_EASINGS[activity.easing ?? 'ease-out'];
	const delay = activity.delay ?? 0;
	const iterations = activity.iterations ?? 1;
	const direction = activity.direction ?? 'normal';
	const fill = activity.fill ?? 'both';

	// CSS animation: name duration easing delay iterations direction fill
	return `${name} ${duration}ms ${easing} ${delay}ms ${iterations === Infinity ? 'infinite' : iterations} ${direction} ${fill}`;
}

/**
 * Generate all @keyframes rules for a list of activities.
 */
export function generateAllKeyframes(activities: IActivity[]): string {
	return activities
		.map(activityToKeyframes)
		.filter(Boolean)
		.join('\n\n');
}

// =============================================================================
// STYLE TOKEN GENERATION
// =============================================================================

/**
 * Convert style tokens to CSS custom properties.
 */
export function styleTokensToCSS(
	tokens: IStyleToken[],
	options: { prefix?: string } = {}
): string {
	const prefix = options.prefix ?? '';

	return tokens
		.map((token) => {
			const property = prefix
				? token.property.replace('--', `--${prefix}-`)
				: token.property;
			return `${property}: ${token.value};`;
		})
		.join('\n');
}

/**
 * Generate CSS for a style activity.
 */
export function styleActivityToCSS(
	activity: IActivity,
	options: { prefix?: string } = {}
): string {
	if (activity.category !== 'style' || !activity.tokens) {
		return '';
	}

	return styleTokensToCSS(activity.tokens, options);
}

// =============================================================================
// PRESET GENERATION
// =============================================================================

/**
 * Generate CSS custom properties for a preset.
 */
export function presetToCSS(
	preset: IStylePreset,
	options: ICSSOptions = {}
): string {
	const {
		includeKeyframes = false,
		includeCustomProperties = true,
		propertyPrefix = '',
		selector = ':root',
		respectReducedMotion = true
	} = options;

	const lines: string[] = [];
	const properties: string[] = [];

	// Color tokens
	if (preset.colors) {
		for (const [name, value] of Object.entries(preset.colors)) {
			const prop = propertyPrefix
				? `--${propertyPrefix}-color-${name}`
				: `--color-${name}`;
			properties.push(`${prop}: ${value};`);
		}
	}

	// Typography tokens
	if (preset.typography) {
		const typo = preset.typography;
		if (typo.fontFamily) {
			properties.push(`--font-family: ${typo.fontFamily};`);
		}
		if (typo.fontFamilyMono) {
			properties.push(`--font-family-mono: ${typo.fontFamilyMono};`);
		}
		if (typo.fontSizeBase) {
			properties.push(`--font-size-base: ${typo.fontSizeBase};`);
		}
		if (typo.lineHeight) {
			properties.push(`--line-height: ${typo.lineHeight};`);
		}
		if (typo.fontWeightNormal !== undefined) {
			properties.push(`--font-weight-normal: ${typo.fontWeightNormal};`);
		}
		if (typo.fontWeightBold !== undefined) {
			properties.push(`--font-weight-bold: ${typo.fontWeightBold};`);
		}
	}

	// Spacing tokens
	if (preset.spacing) {
		for (const [name, value] of Object.entries(preset.spacing)) {
			const prop = propertyPrefix
				? `--${propertyPrefix}-space-${name}`
				: `--space-${name}`;
			properties.push(`${prop}: ${value};`);
		}
	}

	// Effect tokens
	if (preset.effects) {
		const fx = preset.effects;
		if (fx.borderRadius) {
			properties.push(`--border-radius: ${fx.borderRadius};`);
		}
		if (fx.borderRadiusLg) {
			properties.push(`--border-radius-lg: ${fx.borderRadiusLg};`);
		}
		if (fx.shadowSm) {
			properties.push(`--shadow-sm: ${fx.shadowSm};`);
		}
		if (fx.shadowMd) {
			properties.push(`--shadow-md: ${fx.shadowMd};`);
		}
		if (fx.shadowLg) {
			properties.push(`--shadow-lg: ${fx.shadowLg};`);
		}
		if (fx.blur) {
			properties.push(`--blur: ${fx.blur};`);
		}
	}

	// Motion tokens (as CSS custom properties for reference)
	if (preset.motion) {
		const motion = preset.motion;
		const shouldDisable = motion.reducedMotion ||
			(respectReducedMotion && prefersReducedMotion());

		if (shouldDisable) {
			properties.push(`--motion-duration: 0ms;`);
			properties.push(`--motion-enabled: 0;`);
		} else {
			if (motion.entrance) {
				properties.push(`--motion-entrance: motion-entrance-${motion.entrance};`);
			}
			if (motion.exit) {
				properties.push(`--motion-exit: motion-exit-${motion.exit};`);
			}
			if (motion.emphasis) {
				properties.push(`--motion-emphasis: motion-emphasis-${motion.emphasis};`);
			}
			if (motion.duration) {
				const durationMs = MOTION_DURATIONS[motion.duration as keyof typeof MOTION_DURATIONS] ?? 200;
				properties.push(`--motion-duration: ${durationMs}ms;`);
			}
			if (motion.easing) {
				const easingValue = MOTION_EASINGS[motion.easing as keyof typeof MOTION_EASINGS] ?? motion.easing;
				properties.push(`--motion-easing: ${easingValue};`);
			}
			if (motion.stagger !== undefined) {
				properties.push(`--motion-stagger: ${motion.stagger}ms;`);
			}
			properties.push(`--motion-enabled: 1;`);
		}
	}

	// Mode-specific properties
	properties.push(`--color-scheme: ${preset.mode};`);

	// Build CSS
	if (includeCustomProperties && properties.length > 0) {
		lines.push(`${selector} {\n  ${properties.join('\n  ')}\n}`);
	}

	return lines.join('\n\n');
}

/**
 * Generate CSS class for applying an activity.
 */
export function activityToClass(activity: IActivity): string {
	if (activity.category === 'style') {
		// Style activities don't have classes - they're applied via custom properties
		return '';
	}

	const animation = activityToAnimation(activity);
	if (animation === 'none') {
		return '';
	}

	return `.apply-${activity.id} {\n  animation: ${animation};\n}`;
}

/**
 * Generate utility classes for all activities.
 */
export function generateUtilityClasses(activities: IActivity[]): string {
	return activities
		.map(activityToClass)
		.filter(Boolean)
		.join('\n\n');
}

// =============================================================================
// SPRING TO CSS APPROXIMATION
// =============================================================================

/**
 * Approximate spring physics with CSS cubic-bezier.
 *
 * Note: CSS cannot truly represent spring physics, this is an approximation.
 * For true spring physics, use Svelte spring() or Motion.dev.
 */
export function springToCubicBezier(spring: ISpringConfig): string {
	const { stiffness = 0.3, damping = 0.8 } = spring;

	// Approximate spring with cubic-bezier
	// This is a rough approximation - real springs need JS
	if (stiffness > 0.5 && damping < 0.5) {
		// High stiffness, low damping = bouncy
		return 'cubic-bezier(0.68, -0.55, 0.265, 1.55)';
	} else if (stiffness > 0.5) {
		// High stiffness, high damping = snappy
		return 'cubic-bezier(0.175, 0.885, 0.32, 1.275)';
	} else if (damping > 0.8) {
		// Low stiffness, high damping = smooth
		return 'cubic-bezier(0.25, 1, 0.5, 1)';
	} else {
		// Default spring approximation
		return 'cubic-bezier(0.175, 0.885, 0.32, 1.275)';
	}
}

// =============================================================================
// REDUCED MOTION SUPPORT
// =============================================================================

/**
 * Generate CSS media query for reduced motion.
 */
export function generateReducedMotionCSS(): string {
	return `@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}`;
}

// =============================================================================
// EXPORTS
// =============================================================================

export type { Keyframe };
