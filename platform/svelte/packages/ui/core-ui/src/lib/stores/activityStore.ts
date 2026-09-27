/**
 * Activity Store - Svelte store wrapper for ActivityRegistry
 *
 * Provides reactive bindings for the ActivityRegistry, enabling UI components
 * to subscribe to activity changes and presets.
 *
 * Architecture:
 *   ActivityRegistry (runtime catalog)
 *       |
 *   activityStore (Svelte writable/derived stores)
 *       |
 *   UI Components (reactive subscriptions)
 *
 * Features:
 * - Singleton registry instance with store wrappers
 * - Reactive stores for activities, presets, current selection
 * - Derived stores for filtered/computed values
 * - DOM integration for CSS custom property application
 * - localStorage persistence for custom activities
 *
 * Usage:
 *   import {
 *     activities,
 *     currentPreset,
 *     presets,
 *     activeActivities,
 *     addActivity,
 *     setPreset,
 *     applyPresetToDOM
 *   } from '$lib/stores/activityStore';
 *
 *   // Subscribe to activities
 *   $activities.forEach(a => console.log(a.name));
 *
 *   // Change preset
 *   setPreset('hub-light');
 *
 *   // Add custom activity
 *   addActivity({ id: 'custom-fade', ... });
 *
 * @module activityStore
 */

import { browser } from '$app/environment';
import { writable, derived, type Writable, type Readable } from 'svelte/store';
import {
	ActivityRegistry,
	activityRegistry,
	type IActivity,
	type ActivityCategory,
	type ViewLevel,
	type ILevelDefaults,
	MOTION_DURATIONS,
	MOTION_EASINGS,
	getActivityDuration,
	getActivityEasing
} from '../view/ActivityRegistry';
import { stylePresetRegistry, type IStylePreset } from '../style';

// =============================================================================
// TYPE DEFINITIONS
// =============================================================================

/**
 * Activity preset configuration.
 * Maps style presets to activity configurations.
 */
export interface IActivityPreset {
	/** Unique identifier (matches style preset id) */
	id: string;
	/** Display name */
	name: string;
	/** Associated style mode */
	mode: 'light' | 'dark';
	/** Default entrance animation */
	entrance: string;
	/** Default exit animation */
	exit: string;
	/** Default emphasis animation */
	emphasis: string;
	/** Default transition animation */
	transition: string;
	/** Duration scale */
	duration: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
	/** Motion enabled */
	motionEnabled: boolean;
	/** Description */
	description?: string;
}

/**
 * Activity store state.
 */
export interface IActivityStoreState {
	/** All registered activities */
	activities: IActivity[];
	/** Current preset ID */
	presetId: string;
	/** Motion enabled state */
	motionEnabled: boolean;
	/** View level for defaults */
	viewLevel: ViewLevel;
}

// =============================================================================
// BUILTIN ACTIVITY PRESETS
// =============================================================================

/**
 * Builtin activity presets mapped to style presets.
 */
export const ACTIVITY_PRESETS: Record<string, IActivityPreset> = {
	'hub-dark': {
		id: 'hub-dark',
		name: 'Hub Dark',
		mode: 'dark',
		entrance: 'entrance-fade-up',
		exit: 'exit-fade-out',
		emphasis: 'emphasis-pulse',
		transition: 'transition-crossfade',
		duration: 'md',
		motionEnabled: true,
		description: 'Default dark theme with smooth animations'
	},
	'hub-light': {
		id: 'hub-light',
		name: 'Hub Light',
		mode: 'light',
		entrance: 'entrance-fade-up',
		exit: 'exit-fade-out',
		emphasis: 'emphasis-pulse',
		transition: 'transition-crossfade',
		duration: 'md',
		motionEnabled: true,
		description: 'Light theme with smooth animations'
	},
	'hub-high-contrast': {
		id: 'hub-high-contrast',
		name: 'Hub High Contrast',
		mode: 'dark',
		entrance: 'entrance-fade-in',
		exit: 'exit-fade-out',
		emphasis: 'emphasis-none',
		transition: 'transition-crossfade',
		duration: 'sm',
		motionEnabled: false,
		description: 'High contrast accessibility theme with reduced motion'
	},
	professional: {
		id: 'professional',
		name: 'Professional',
		mode: 'dark',
		entrance: 'entrance-fade-in',
		exit: 'exit-fade-out',
		emphasis: 'emphasis-none',
		transition: 'transition-crossfade',
		duration: 'sm',
		motionEnabled: true,
		description: 'Minimal corporate style with subtle animations'
	},
	playful: {
		id: 'playful',
		name: 'Playful',
		mode: 'dark',
		entrance: 'entrance-scale-up',
		exit: 'exit-scale-down-out',
		emphasis: 'emphasis-bounce',
		transition: 'transition-slide',
		duration: 'lg',
		motionEnabled: true,
		description: 'Vibrant and bouncy animations'
	}
};

// =============================================================================
// STORAGE
// =============================================================================

const STORAGE_KEY = 'sbx-activity-store';

/**
 * Default state.
 */
const defaultState: IActivityStoreState = {
	activities: [],
	presetId: 'hub-dark',
	motionEnabled: true,
	viewLevel: 'any'
};

/**
 * Load state from localStorage.
 */
function loadState(): Partial<IActivityStoreState> {
	if (!browser) {
		return {};
	}

	try {
		const stored = localStorage.getItem(STORAGE_KEY);
		if (stored) {
			return JSON.parse(stored) as Partial<IActivityStoreState>;
		}
	} catch {
		console.warn('[activityStore] Failed to load from localStorage');
	}

	return {};
}

/**
 * Save state to localStorage.
 */
function saveState(state: Partial<IActivityStoreState>): void {
	if (!browser) {
		return;
	}

	try {
		// Only persist user preferences, not the full activity list
		const toSave = {
			presetId: state.presetId,
			motionEnabled: state.motionEnabled,
			viewLevel: state.viewLevel
		};
		localStorage.setItem(STORAGE_KEY, JSON.stringify(toSave));
	} catch {
		console.warn('[activityStore] Failed to save to localStorage');
	}
}

// =============================================================================
// SINGLETON REGISTRY
// =============================================================================

/**
 * Use the global activity registry singleton.
 */
const registry = activityRegistry;

// =============================================================================
// WRITABLE STORES
// =============================================================================

/**
 * Initialize state from localStorage.
 */
const initialState = loadState();

/**
 * All registered activities.
 * Subscribe to get updated activity list.
 */
export const activities: Writable<IActivity[]> = writable(registry.list());

/**
 * Current preset ID.
 * Subscribe to track active preset.
 */
export const currentPreset: Writable<string> = writable(
	initialState.presetId ?? defaultState.presetId
);

/**
 * Available activity presets.
 */
export const presets: Writable<IActivityPreset[]> = writable(Object.values(ACTIVITY_PRESETS));

/**
 * Motion enabled state.
 */
export const motionEnabled: Writable<boolean> = writable(
	initialState.motionEnabled ?? defaultState.motionEnabled
);

/**
 * Current view level for defaults.
 */
export const viewLevel: Writable<ViewLevel> = writable(
	initialState.viewLevel ?? defaultState.viewLevel
);

// =============================================================================
// DERIVED STORES
// =============================================================================

/**
 * Active preset configuration.
 */
export const activePreset: Readable<IActivityPreset | undefined> = derived(
	currentPreset,
	($preset) => ACTIVITY_PRESETS[$preset]
);

/**
 * Activities filtered by current preset.
 * Returns activities appropriate for the current preset's animation style.
 */
export const activeActivities: Readable<IActivity[]> = derived(
	[activities, currentPreset],
	([$activities, $preset]) => {
		const preset = ACTIVITY_PRESETS[$preset];
		if (!preset) {
			return $activities;
		}

		// Return activities that match the preset's motion characteristics
		return $activities.filter((activity) => {
			// Include all style activities
			const category = activity.category as string;
			if (category === 'style') return true;

			// For motion activities, filter based on preset motion settings
			if (!preset.motionEnabled) {
				// If motion disabled, only include 'none' variants
				return activity.id.includes('none') || category === 'style';
			}

			return true;
		});
	}
);

/**
 * Level defaults for current view level.
 */
export const levelDefaults: Readable<ILevelDefaults> = derived(viewLevel, ($level) =>
	registry.getDefaults($level)
);

/**
 * Activities grouped by category.
 */
export const activitiesByCategory: Readable<Record<ActivityCategory, IActivity[]>> = derived(
	activities,
	($activities) => {
		const grouped: Partial<Record<ActivityCategory, IActivity[]>> = {};
		const categories: ActivityCategory[] = [
			'entrance',
			'exit',
			'emphasis',
			'transition',
			'style',
			'state',
			'gesture'
		];

		for (const category of categories) {
			grouped[category] = $activities.filter((a) => a.category === category);
		}

		return grouped as Record<ActivityCategory, IActivity[]>;
	}
);

/**
 * Custom activities only.
 */
export const customActivities: Readable<IActivity[]> = derived(activities, ($activities) =>
	$activities.filter((a) => a.custom === true)
);

/**
 * Preset mode (light/dark).
 */
export const presetMode: Readable<'light' | 'dark'> = derived(
	activePreset,
	($preset) => $preset?.mode ?? 'dark'
);

// =============================================================================
// ACTIONS
// =============================================================================

/**
 * Refresh activities store from registry.
 */
function refreshActivities(): void {
	activities.set(registry.list());
}

/**
 * Add a new activity to the registry.
 *
 * @param activity - Activity to register
 *
 * @example
 * addActivity({
 *   id: 'entrance-custom-bounce',
 *   name: 'Custom Bounce',
 *   category: 'entrance',
 *   duration: 'md',
 *   easing: 'spring',
 *   keyframes: [...],
 *   custom: true
 * });
 */
export function addActivity(activity: IActivity): void {
	registry.register(activity);
	registry.saveToStorage();
	refreshActivities();
}

/**
 * Update an existing activity.
 *
 * @param id - Activity ID to update
 * @param changes - Partial activity updates
 * @returns true if update succeeded
 *
 * @example
 * updateActivity('entrance-fade-up', { duration: 'lg' });
 */
export function updateActivity(id: string, changes: Partial<IActivity>): boolean {
	const success = registry.update(id, changes);
	if (success) {
		registry.saveToStorage();
		refreshActivities();
	}
	return success;
}

/**
 * Remove an activity from the registry.
 *
 * @param id - Activity ID to remove
 * @returns true if removal succeeded
 *
 * @example
 * removeActivity('entrance-custom-bounce');
 */
export function removeActivity(id: string): boolean {
	const success = registry.delete(id);
	if (success) {
		registry.saveToStorage();
		refreshActivities();
	}
	return success;
}

/**
 * Set the current preset.
 *
 * @param presetId - Preset ID to activate
 *
 * @example
 * setPreset('hub-light');
 */
export function setPreset(presetId: string): void {
	const preset = ACTIVITY_PRESETS[presetId];
	if (!preset) {
		console.warn(`[activityStore] Unknown preset: ${presetId}`);
		return;
	}

	currentPreset.set(presetId);
	motionEnabled.set(preset.motionEnabled);

	// Persist preference
	saveState({
		presetId,
		motionEnabled: preset.motionEnabled
	});

	// Apply to DOM
	applyPresetToDOM();
}

/**
 * Set motion enabled state.
 *
 * @param enabled - Whether motion is enabled
 *
 * @example
 * setMotionEnabled(false); // Disable animations
 */
export function setMotionEnabled(enabled: boolean): void {
	motionEnabled.set(enabled);
	saveState({ motionEnabled: enabled });
	applyMotionToDOM(enabled);
}

/**
 * Toggle motion enabled state.
 */
export function toggleMotion(): void {
	motionEnabled.update((current) => {
		const next = !current;
		saveState({ motionEnabled: next });
		applyMotionToDOM(next);
		return next;
	});
}

/**
 * Set the view level for defaults.
 *
 * @param level - View level to set
 *
 * @example
 * setViewLevel('blocks');
 */
export function setViewLevel(level: ViewLevel): void {
	viewLevel.set(level);
	saveState({ viewLevel: level });
}

/**
 * Get an activity by ID.
 *
 * @param id - Activity ID
 * @returns Activity or undefined
 */
export function getActivity(id: string): IActivity | undefined {
	return registry.get(id);
}

/**
 * Get activities by category.
 *
 * @param category - Activity category
 * @returns Activities in category
 */
export function getActivitiesByCategory(category: ActivityCategory): IActivity[] {
	return registry.getByCategory(category);
}

/**
 * Search activities.
 *
 * @param query - Search query
 * @returns Matching activities
 */
export function searchActivities(query: string): IActivity[] {
	return registry.search(query);
}

// =============================================================================
// DOM APPLICATION
// =============================================================================

/**
 * Apply current preset to DOM.
 * Sets CSS custom properties for motion tokens.
 */
export function applyPresetToDOM(): void {
	if (!browser) return;

	let presetId: string = 'hub-dark';
	currentPreset.subscribe((p) => (presetId = p))();

	const preset = ACTIVITY_PRESETS[presetId];
	if (!preset) return;

	const html = document.documentElement;

	// Set preset data attribute
	html.setAttribute('data-activity-preset', presetId);

	// Apply duration scale
	const durationMs = MOTION_DURATIONS[preset.duration];
	html.style.setProperty('--motion-duration-default', `${durationMs}ms`);

	// Apply motion enabled state
	applyMotionToDOM(preset.motionEnabled);

	// Apply default animations as CSS custom properties
	const entranceActivity = registry.get(preset.entrance);
	const exitActivity = registry.get(preset.exit);
	const transitionActivity = registry.get(preset.transition);

	if (entranceActivity) {
		html.style.setProperty('--motion-entrance-duration', `${getActivityDuration(entranceActivity)}ms`);
		html.style.setProperty('--motion-entrance-easing', getActivityEasing(entranceActivity));
	}

	if (exitActivity) {
		html.style.setProperty('--motion-exit-duration', `${getActivityDuration(exitActivity)}ms`);
		html.style.setProperty('--motion-exit-easing', getActivityEasing(exitActivity));
	}

	if (transitionActivity) {
		html.style.setProperty('--motion-transition-duration', `${getActivityDuration(transitionActivity)}ms`);
		html.style.setProperty('--motion-transition-easing', getActivityEasing(transitionActivity));
	}
}

/**
 * Apply motion state to DOM.
 */
function applyMotionToDOM(enabled: boolean): void {
	if (!browser) return;

	const html = document.documentElement;
	html.setAttribute('data-motion', enabled ? 'enabled' : 'reduced');

	if (!enabled) {
		// Set zero duration for reduced motion
		html.style.setProperty('--motion-duration-default', '0ms');
		html.style.setProperty('--motion-entrance-duration', '0ms');
		html.style.setProperty('--motion-exit-duration', '0ms');
		html.style.setProperty('--motion-transition-duration', '0ms');
	}
}

/**
 * Apply activity CSS to DOM.
 * Injects keyframe animation for an activity.
 *
 * @param activity - Activity to apply
 */
export function applyActivityCSS(activity: IActivity): void {
	if (!browser) return;
	if (!activity.keyframes || activity.keyframes.length === 0) return;

	// Generate CSS keyframes
	const keyframeCSS = generateKeyframeCSS(activity);
	if (!keyframeCSS) return;

	// Check if style element exists
	let styleEl = document.getElementById('sbx-activity-styles');
	if (!styleEl) {
		styleEl = document.createElement('style');
		styleEl.id = 'sbx-activity-styles';
		document.head.appendChild(styleEl);
	}

	// Append keyframe if not already present
	if (!styleEl.textContent?.includes(`@keyframes ${activity.id}`)) {
		styleEl.textContent += keyframeCSS;
	}
}

/**
 * Generate CSS keyframes string for an activity.
 */
function generateKeyframeCSS(activity: IActivity): string | null {
	if (!activity.keyframes || activity.keyframes.length === 0) {
		return null;
	}

	const keyframeRules = activity.keyframes
		.map((kf) => {
			const props = Object.entries(kf)
				.filter(([key]) => key !== 'offset')
				.map(([key, value]) => {
					// Convert camelCase to kebab-case
					const cssKey = key.replace(/([A-Z])/g, '-$1').toLowerCase();
					return `${cssKey}: ${value}`;
				})
				.join('; ');

			const offset =
				kf.offset !== undefined ? `${(kf.offset * 100).toFixed(0)}%` : 'from';
			return `${offset} { ${props} }`;
		})
		.join('\n  ');

	return `
@keyframes ${activity.id} {
  ${keyframeRules}
}
`;
}

// =============================================================================
// INITIALIZATION
// =============================================================================

/**
 * Initialize activity store.
 * Call this in root layout or app entry.
 */
export function initActivityStore(): void {
	if (!browser) return;

	// Load custom activities from localStorage
	registry.loadFromStorage();
	refreshActivities();

	// Apply initial preset
	applyPresetToDOM();

	// Listen for storage changes (cross-tab sync)
	window.addEventListener('storage', (event) => {
		if (event.key === STORAGE_KEY && event.newValue) {
			try {
				const newState = JSON.parse(event.newValue) as Partial<IActivityStoreState>;
				if (newState.presetId) {
					currentPreset.set(newState.presetId);
				}
				if (newState.motionEnabled !== undefined) {
					motionEnabled.set(newState.motionEnabled);
				}
				if (newState.viewLevel) {
					viewLevel.set(newState.viewLevel);
				}
				applyPresetToDOM();
			} catch {
				// Ignore parse errors
			}
		}
	});

	// Listen for prefers-reduced-motion changes
	const mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)');
	mediaQuery.addEventListener('change', (e) => {
		if (e.matches) {
			// System requests reduced motion
			setMotionEnabled(false);
		}
	});

	// Check initial reduced motion preference
	if (mediaQuery.matches) {
		setMotionEnabled(false);
	}
}

// Auto-initialize in browser
if (browser) {
	queueMicrotask(initActivityStore);
}

// =============================================================================
// RE-EXPORTS
// =============================================================================

export {
	type IActivity,
	type ActivityCategory,
	type ViewLevel,
	type ILevelDefaults,
	MOTION_DURATIONS,
	MOTION_EASINGS
} from '@sbx/core-ui/view/ActivityRegistry';
