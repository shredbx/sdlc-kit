/**
 * ActivityRegistry - Unified Registry for View Activities
 *
 * Provides a runtime catalog of all view activities (motion tokens, style tokens,
 * state behaviors, gesture responses) with CRUD operations and multi-backend output.
 *
 * Architecture:
 *   Activity (IR) → ActivityRegistry → Adapters → CSS/Svelte/GSAP/Motion.dev
 *
 * Features:
 * - CRUD operations: register, get, update, delete
 * - Query operations: getByCategory, search, getDefaults
 * - Serialization: toJSON, fromJSON for persistence
 * - Backward compatibility: wraps existing MotionTokens.ts
 *
 * Standards:
 * - Material Design 3: Motion categories and easing
 * - W3C Design Tokens Format (DTCG): Token naming
 * - WCAG 2.1 SC 2.3.3: prefers-reduced-motion support
 *
 * @example
 * // Get activity by id
 * const fadeUp = activityRegistry.get('entrance-fade-up');
 *
 * @example
 * // Get all entrance animations
 * const entrances = activityRegistry.getByCategory('entrance');
 *
 * @example
 * // Register custom activity
 * activityRegistry.register({
 *   id: 'entrance-custom-bounce',
 *   name: 'Custom Bounce',
 *   category: 'entrance',
 *   duration: 'md',
 *   easing: 'spring',
 *   keyframes: [...],
 *   custom: true
 * });
 */

import {
	type IMotionToken,
	type MotionCategory,
	type MotionDuration,
	type MotionEasing,
	type Keyframe,
	type AnimationName,
	ENTRANCE_TOKENS,
	EXIT_TOKENS,
	EMPHASIS_TOKENS,
	TRANSITION_TOKENS,
	MOTION_DURATIONS,
	MOTION_EASINGS,
	prefersReducedMotion
} from './MotionTokens';

// =============================================================================
// TYPES
// =============================================================================

/**
 * Activity category - extends MotionCategory with style, state, gesture.
 */
export type ActivityCategory =
	| MotionCategory // entrance, exit, emphasis, transition
	| 'style' // CSS custom properties
	| 'state' // interaction states
	| 'gesture'; // touch/mouse gestures

/**
 * View level for cascade defaults.
 */
export type ViewLevel = 'primitives' | 'blocks' | 'sections' | 'layouts' | 'any';

/**
 * State trigger for state activities.
 */
export type StateTrigger = 'hover' | 'focus' | 'active' | 'disabled' | 'selected' | 'loading';

/**
 * Gesture type for gesture activities.
 */
export type GestureType = 'drag' | 'pan' | 'swipe' | 'pinch' | 'rotate' | 'long-press';

/**
 * Token format hint for style activities.
 */
export type TokenFormat =
	| 'color'
	| 'dimension'
	| 'fontFamily'
	| 'fontWeight'
	| 'duration'
	| 'cubicBezier'
	| 'number'
	| 'string';

/**
 * Spring configuration for physics-based animations.
 */
export interface ISpringConfig {
	/** Spring stiffness (0-1 normalized or 100-500 physical) */
	stiffness?: number;
	/** Spring damping (0-1 normalized) */
	damping?: number;
	/** Mass of animated value */
	mass?: number;
	/** Threshold for completion */
	precision?: number;
}

/**
 * Style token for CSS custom properties.
 */
export interface IStyleToken {
	/** CSS custom property name (--name) */
	property: string;
	/** Token value */
	value: string;
	/** Format hint for editor */
	format?: TokenFormat;
}

/**
 * Activity definition - unified abstraction for all view behaviors.
 *
 * Extends IMotionToken with additional categories (style, state, gesture).
 */
export interface IActivity {
	// =========================================================================
	// IDENTITY (Required)
	// =========================================================================

	/** Unique identifier (format: {category}-{name}) */
	id: string;

	/** Display name */
	name: string;

	/** Activity category */
	category: ActivityCategory;

	// =========================================================================
	// MOTION PROPERTIES (entrance, exit, emphasis, transition)
	// =========================================================================

	/** Duration scale or numeric ms */
	duration?: MotionDuration | number;

	/** Duration in milliseconds (override scale) */
	durationMs?: number;

	/** Easing preset or custom string */
	easing?: MotionEasing | string;

	/** Custom easing curve */
	easingCurve?: string;

	/** Delay before animation (ms) */
	delay?: number;

	/** Iteration count (Infinity for loop) */
	iterations?: number;

	/** Playback direction */
	direction?: 'normal' | 'reverse' | 'alternate' | 'alternate-reverse';

	/** Fill mode */
	fill?: 'none' | 'forwards' | 'backwards' | 'both';

	/** CSS keyframes */
	keyframes?: Keyframe[];

	/** Spring physics config (alternative to keyframes) */
	spring?: ISpringConfig;

	// =========================================================================
	// STYLE PROPERTIES (category = style)
	// =========================================================================

	/** CSS custom property tokens */
	tokens?: IStyleToken[];

	// =========================================================================
	// STATE PROPERTIES (category = state)
	// =========================================================================

	/** Interaction trigger */
	trigger?: StateTrigger;

	/** State transition keyframes */
	stateKeyframes?: Keyframe[];

	// =========================================================================
	// GESTURE PROPERTIES (category = gesture)
	// =========================================================================

	/** Gesture type */
	gestureType?: GestureType;

	// =========================================================================
	// METADATA
	// =========================================================================

	/** Description for dev tools */
	description?: string;

	/** Classification tags */
	tags?: string[];

	/** Source of definition */
	source?: string;

	/** User-created flag */
	custom?: boolean;

	/** Recommended view level */
	viewLevel?: ViewLevel;
}

/**
 * Level defaults for cascade resolution.
 */
export interface ILevelDefaults {
	entrance: string;
	exit: string;
	emphasis: string;
	transition: string;
	duration: MotionDuration;
}

/**
 * Serialized registry state for persistence.
 */
export interface IActivityRegistryState {
	version: number;
	activities: IActivity[];
	timestamp: string;
}

// =============================================================================
// ACTIVITY REGISTRY INTERFACE
// =============================================================================

/**
 * Activity registry interface.
 */
export interface IActivityRegistry {
	// CRUD Operations
	register(activity: IActivity): void;
	get(id: string): IActivity | undefined;
	getByName(category: ActivityCategory, name: string): IActivity | undefined;
	update(id: string, updates: Partial<IActivity>): boolean;
	delete(id: string): boolean;
	has(id: string): boolean;

	// Query Operations
	getByCategory(category: ActivityCategory): IActivity[];
	getByTag(tag: string): IActivity[];
	getByViewLevel(level: ViewLevel): IActivity[];
	search(query: string): IActivity[];
	getDefaults(level: ViewLevel): ILevelDefaults;
	getCustomActivities(): IActivity[];

	// Bulk Operations
	registerAll(activities: IActivity[]): void;
	clear(): void;
	count(): number;
	list(): IActivity[];

	// Serialization
	toJSON(): IActivityRegistryState;
	fromJSON(state: IActivityRegistryState): void;

	// Persistence
	saveToStorage(): void;
	loadFromStorage(): void;
}

// =============================================================================
// LEVEL DEFAULTS
// =============================================================================

/**
 * Default activities per view level.
 */
export const LEVEL_DEFAULTS: Record<ViewLevel, ILevelDefaults> = {
	primitives: {
		entrance: 'entrance-fade-in',
		exit: 'exit-fade-out',
		emphasis: 'emphasis-pulse',
		transition: 'transition-crossfade',
		duration: 'sm'
	},
	blocks: {
		entrance: 'entrance-fade-up',
		exit: 'exit-fade-out',
		emphasis: 'emphasis-pulse',
		transition: 'transition-crossfade',
		duration: 'md'
	},
	sections: {
		entrance: 'entrance-slide-up',
		exit: 'exit-slide-down-out',
		emphasis: 'emphasis-pulse',
		transition: 'transition-slide',
		duration: 'lg'
	},
	layouts: {
		entrance: 'entrance-fade-in',
		exit: 'exit-fade-out',
		emphasis: 'emphasis-none',
		transition: 'transition-crossfade',
		duration: 'lg'
	},
	any: {
		entrance: 'entrance-fade-up',
		exit: 'exit-fade-out',
		emphasis: 'emphasis-pulse',
		transition: 'transition-crossfade',
		duration: 'md'
	}
};

// =============================================================================
// ACTIVITY REGISTRY IMPLEMENTATION
// =============================================================================

/**
 * Activity registry implementation.
 */
export class ActivityRegistry implements IActivityRegistry {
	private activities = new Map<string, IActivity>();
	private readonly STORAGE_KEY = 'activity-registry-custom';
	private readonly VERSION = 1;

	constructor() {
		// Initialize with built-in motion tokens
		this.initializeBuiltInActivities();
	}

	// =========================================================================
	// INITIALIZATION
	// =========================================================================

	/**
	 * Convert existing MotionTokens to Activities.
	 */
	private initializeBuiltInActivities(): void {
		// Entrance tokens
		for (const [name, token] of Object.entries(ENTRANCE_TOKENS)) {
			this.register(this.motionTokenToActivity(token, 'entrance', name));
		}

		// Exit tokens
		for (const [name, token] of Object.entries(EXIT_TOKENS)) {
			this.register(this.motionTokenToActivity(token, 'exit', name));
		}

		// Emphasis tokens
		for (const [name, token] of Object.entries(EMPHASIS_TOKENS)) {
			this.register(this.motionTokenToActivity(token, 'emphasis', name));
		}

		// Transition tokens
		for (const [name, token] of Object.entries(TRANSITION_TOKENS)) {
			this.register(this.motionTokenToActivity(token, 'transition', name));
		}
	}

	/**
	 * Convert IMotionToken to IActivity.
	 */
	private motionTokenToActivity(
		token: IMotionToken,
		category: MotionCategory,
		name: string
	): IActivity {
		return {
			id: `${category}-${name}`,
			name: this.formatName(name),
			category,
			duration: token.duration,
			easing: token.easing,
			delay: token.delay,
			iterations: token.iterations,
			direction: token.direction,
			fill: token.fill,
			keyframes: token.keyframes,
			description: token.description,
			source: 'MotionTokens.ts',
			custom: false,
			viewLevel: this.inferViewLevel(category, name)
		};
	}

	/**
	 * Format token name to display name.
	 */
	private formatName(name: string): string {
		return name
			.split('-')
			.map((word) => word.charAt(0).toUpperCase() + word.slice(1))
			.join(' ');
	}

	/**
	 * Infer recommended view level from activity.
	 */
	private inferViewLevel(category: string, name: string): ViewLevel {
		// Slide animations are for sections
		if (name.startsWith('slide')) return 'sections';

		// Scale/zoom for blocks
		if (name.startsWith('scale') || name.startsWith('zoom')) return 'blocks';

		// Fade for primitives
		if (name === 'fade-in' || name === 'fade-out') return 'primitives';

		// Default based on category
		return 'any';
	}

	// =========================================================================
	// CRUD OPERATIONS
	// =========================================================================

	register(activity: IActivity): void {
		this.activities.set(activity.id, activity);
	}

	get(id: string): IActivity | undefined {
		return this.activities.get(id);
	}

	getByName(category: ActivityCategory, name: string): IActivity | undefined {
		const id = `${category}-${name}`;
		return this.activities.get(id);
	}

	update(id: string, updates: Partial<IActivity>): boolean {
		const existing = this.activities.get(id);
		if (!existing) return false;

		const updated = { ...existing, ...updates, id }; // Preserve id
		this.activities.set(id, updated);
		return true;
	}

	delete(id: string): boolean {
		return this.activities.delete(id);
	}

	has(id: string): boolean {
		return this.activities.has(id);
	}

	// =========================================================================
	// QUERY OPERATIONS
	// =========================================================================

	getByCategory(category: ActivityCategory): IActivity[] {
		return Array.from(this.activities.values()).filter((a) => a.category === category);
	}

	getByTag(tag: string): IActivity[] {
		const lowerTag = tag.toLowerCase();
		return Array.from(this.activities.values()).filter((a) =>
			a.tags?.some((t) => t.toLowerCase() === lowerTag)
		);
	}

	getByViewLevel(level: ViewLevel): IActivity[] {
		return Array.from(this.activities.values()).filter(
			(a) => a.viewLevel === level || a.viewLevel === 'any'
		);
	}

	search(query: string): IActivity[] {
		const q = query.toLowerCase();
		return Array.from(this.activities.values()).filter(
			(a) =>
				a.name.toLowerCase().includes(q) ||
				a.id.includes(q) ||
				a.category.includes(q) ||
				a.description?.toLowerCase().includes(q) ||
				a.tags?.some((t) => t.toLowerCase().includes(q))
		);
	}

	getDefaults(level: ViewLevel): ILevelDefaults {
		return LEVEL_DEFAULTS[level] ?? LEVEL_DEFAULTS.any;
	}

	getCustomActivities(): IActivity[] {
		return Array.from(this.activities.values()).filter((a) => a.custom === true);
	}

	// =========================================================================
	// BULK OPERATIONS
	// =========================================================================

	registerAll(activities: IActivity[]): void {
		for (const activity of activities) {
			this.register(activity);
		}
	}

	clear(): void {
		this.activities.clear();
		this.initializeBuiltInActivities();
	}

	count(): number {
		return this.activities.size;
	}

	list(): IActivity[] {
		return Array.from(this.activities.values());
	}

	// =========================================================================
	// SERIALIZATION
	// =========================================================================

	toJSON(): IActivityRegistryState {
		// Only serialize custom activities
		const customActivities = this.getCustomActivities();
		return {
			version: this.VERSION,
			activities: customActivities,
			timestamp: new Date().toISOString()
		};
	}

	fromJSON(state: IActivityRegistryState): void {
		if (state.version !== this.VERSION) {
			console.warn(`ActivityRegistry: version mismatch (expected ${this.VERSION}, got ${state.version})`);
		}

		for (const activity of state.activities) {
			// Mark as custom when loading
			this.register({ ...activity, custom: true });
		}
	}

	// =========================================================================
	// PERSISTENCE (localStorage)
	// =========================================================================

	saveToStorage(): void {
		if (typeof localStorage === 'undefined') return;

		try {
			const state = this.toJSON();
			localStorage.setItem(this.STORAGE_KEY, JSON.stringify(state));
		} catch (e) {
			console.error('ActivityRegistry: failed to save to localStorage', e);
		}
	}

	loadFromStorage(): void {
		if (typeof localStorage === 'undefined') return;

		try {
			const stored = localStorage.getItem(this.STORAGE_KEY);
			if (stored) {
				const state = JSON.parse(stored) as IActivityRegistryState;
				this.fromJSON(state);
			}
		} catch (e) {
			console.error('ActivityRegistry: failed to load from localStorage', e);
		}
	}
}

// =============================================================================
// GLOBAL REGISTRY
// =============================================================================

/** Global activity registry instance */
export const activityRegistry = new ActivityRegistry();

// Auto-load custom activities from localStorage
if (typeof window !== 'undefined') {
	activityRegistry.loadFromStorage();
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

/**
 * Create a motion activity.
 */
export function createMotionActivity(
	category: MotionCategory,
	name: string,
	keyframes: Keyframe[],
	config?: Partial<Omit<IActivity, 'id' | 'category' | 'keyframes'>>
): IActivity {
	return {
		id: `${category}-${name}`,
		name: config?.name ?? name.split('-').map((w) => w.charAt(0).toUpperCase() + w.slice(1)).join(' '),
		category,
		duration: config?.duration ?? 'md',
		easing: config?.easing ?? 'ease-out',
		fill: config?.fill ?? 'both',
		keyframes,
		custom: true,
		...config
	};
}

/**
 * Create a style activity.
 */
export function createStyleActivity(
	name: string,
	tokens: IStyleToken[],
	config?: Partial<Omit<IActivity, 'id' | 'category' | 'tokens'>>
): IActivity {
	const slug = name.toLowerCase().replace(/\s+/g, '-');
	return {
		id: `style-${slug}`,
		name,
		category: 'style',
		tokens,
		custom: true,
		...config
	};
}

/**
 * Create a state activity.
 */
export function createStateActivity(
	trigger: StateTrigger,
	name: string,
	stateKeyframes: Keyframe[],
	config?: Partial<Omit<IActivity, 'id' | 'category' | 'trigger' | 'stateKeyframes'>>
): IActivity {
	return {
		id: `state-${trigger}-${name}`,
		name: config?.name ?? `${trigger.charAt(0).toUpperCase() + trigger.slice(1)} ${name}`,
		category: 'state',
		trigger,
		stateKeyframes,
		duration: config?.duration ?? 'sm',
		easing: config?.easing ?? 'ease-out',
		custom: true,
		...config
	};
}

/**
 * Resolve activity with reduced motion support.
 */
export function resolveActivity(
	activity: IActivity,
	respectReducedMotion = true
): IActivity {
	if (respectReducedMotion && prefersReducedMotion()) {
		// Return activity with minimal motion
		return {
			...activity,
			duration: 'xs',
			durationMs: 0,
			keyframes: [],
			stateKeyframes: []
		};
	}
	return activity;
}

/**
 * Get duration in milliseconds.
 */
export function getActivityDuration(activity: IActivity): number {
	if (activity.durationMs !== undefined) return activity.durationMs;
	const dur = activity.duration ?? 'md';
	if (typeof dur === 'number') return dur;
	return MOTION_DURATIONS[dur];
}

/**
 * Get easing as CSS timing function.
 */
export function getActivityEasing(activity: IActivity): string {
	if (activity.easingCurve) return activity.easingCurve;
	const eas = activity.easing ?? 'ease-out';
	if (typeof eas === 'string' && !(eas in MOTION_EASINGS)) return eas;
	return MOTION_EASINGS[eas as MotionEasing];
}

// =============================================================================
// EXPORTS
// =============================================================================

export {
	// Re-export from MotionTokens for convenience
	MOTION_DURATIONS,
	MOTION_EASINGS,
	prefersReducedMotion
};
