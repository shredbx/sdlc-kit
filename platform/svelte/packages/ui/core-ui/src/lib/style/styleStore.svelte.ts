/**
 * Style Store - Persisted style state with localStorage
 *
 * Manages the active style preset with browser persistence.
 * Updates are reflected immediately in the DOM via data attributes.
 *
 * Usage:
 *   import { styleStore, setStylePreset, getActivePreset } from './styleStore';
 *
 *   // Get current preset ID
 *   const currentId = styleStore.presetId;
 *
 *   // Change preset
 *   setStylePreset('hub-light');
 *
 *   // Get full preset config
 *   const preset = getActivePreset();
 *
 * Persistence:
 *   - Saved to localStorage under 'sbx-style' key
 *   - Syncs across tabs via storage event
 *   - Falls back to default if localStorage unavailable
 *
 * DOM Integration:
 *   - Sets data-theme attribute on <html>
 *   - Sets data-brand attribute on <html>
 *   - Sets color-scheme CSS property
 */

import { browser } from '$app/environment';
import { stylePresetRegistry, type IStylePreset, type StyleMode } from './IStylePreset';

// =============================================================================
// STORAGE KEY
// =============================================================================

const STORAGE_KEY = 'sbx-style';
const DEFAULT_PRESET_ID = 'hub-dark';

// =============================================================================
// STYLE STATE INTERFACE
// =============================================================================

/**
 * Persisted style state.
 */
export interface IStyleState {
	/** Active preset ID */
	presetId: string;
	/** User preference for system mode */
	systemMode: StyleMode;
	/** Reduced motion preference */
	motionEnabled: boolean;
	/** Custom token overrides (user modifications) */
	customOverrides: Record<string, string>;
	/** Last updated timestamp */
	updatedAt?: number;
}

// =============================================================================
// DEFAULT STATE
// =============================================================================

const defaultState: IStyleState = {
	presetId: DEFAULT_PRESET_ID,
	systemMode: 'system',
	motionEnabled: true,
	customOverrides: {},
	updatedAt: Date.now()
};

// =============================================================================
// STORE IMPLEMENTATION
// =============================================================================

/**
 * Load state from localStorage.
 */
function loadState(): IStyleState {
	if (!browser) {
		return defaultState;
	}

	try {
		const stored = localStorage.getItem(STORAGE_KEY);
		if (stored) {
			const parsed = JSON.parse(stored) as Partial<IStyleState>;
			return { ...defaultState, ...parsed };
		}
	} catch {
		console.warn('Failed to load style state from localStorage');
	}

	return defaultState;
}

/**
 * Save state to localStorage.
 */
function saveState(state: IStyleState): void {
	if (!browser) {
		return;
	}

	try {
		state.updatedAt = Date.now();
		localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
	} catch {
		console.warn('Failed to save style state to localStorage');
	}
}

// Create reactive state using Svelte 5 runes
let state = $state<IStyleState>(loadState());

// =============================================================================
// PUBLIC API
// =============================================================================

/**
 * Style store - reactive state object.
 * Access properties directly (e.g., styleStore.presetId).
 */
export const styleStore = {
	get presetId() {
		return state.presetId;
	},
	get systemMode() {
		return state.systemMode;
	},
	get motionEnabled() {
		return state.motionEnabled;
	},
	get customOverrides() {
		return state.customOverrides;
	}
};

/**
 * Set the active style preset.
 */
export function setStylePreset(presetId: string): void {
	const preset = stylePresetRegistry.get(presetId);
	if (!preset) {
		console.warn(`Unknown preset: ${presetId}`);
		return;
	}

	state.presetId = presetId;
	saveState(state);
	applyPresetToDOM(preset);
}

/**
 * Set system mode preference.
 */
export function setSystemMode(mode: StyleMode): void {
	state.systemMode = mode;
	saveState(state);
	applyModeToDOM(mode);
}

/**
 * Toggle motion preference.
 */
export function toggleMotion(): void {
	state.motionEnabled = !state.motionEnabled;
	saveState(state);
	applyMotionToDOM(state.motionEnabled);
}

/**
 * Set custom token override.
 */
export function setTokenOverride(token: string, value: string): void {
	state.customOverrides = { ...state.customOverrides, [token]: value };
	saveState(state);
	applyOverridesToDOM(state.customOverrides);
}

/**
 * Clear all custom overrides.
 */
export function clearOverrides(): void {
	state.customOverrides = {};
	saveState(state);
	applyOverridesToDOM({});
}

/**
 * Reset to default state.
 */
export function resetStyleState(): void {
	state = { ...defaultState };
	saveState(state);
	applyAllToDOM();
}

/**
 * Get the currently active preset.
 */
export function getActivePreset(): IStylePreset | undefined {
	return stylePresetRegistry.get(state.presetId);
}

/**
 * Get resolved mode (respects system preference if mode is 'system').
 */
export function getResolvedMode(): 'light' | 'dark' {
	if (state.systemMode === 'system' && browser) {
		return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
	}
	const preset = getActivePreset();
	return preset?.mode ?? 'dark';
}

// =============================================================================
// DOM APPLICATION
// =============================================================================

/**
 * Apply preset to DOM.
 */
function applyPresetToDOM(preset: IStylePreset): void {
	if (!browser) return;

	const html = document.documentElement;
	html.setAttribute('data-theme', preset.mode);
	html.setAttribute('data-brand', preset.brandId);
	html.setAttribute('data-preset', preset.id);
	html.style.colorScheme = preset.mode;
}

/**
 * Apply mode preference to DOM.
 */
function applyModeToDOM(mode: StyleMode): void {
	if (!browser) return;

	const resolved = mode === 'system' ? getResolvedMode() : mode;
	document.documentElement.setAttribute('data-mode-preference', mode);
	document.documentElement.style.colorScheme = resolved;
}

/**
 * Apply motion preference to DOM.
 */
function applyMotionToDOM(enabled: boolean): void {
	if (!browser) return;

	document.documentElement.setAttribute('data-motion', enabled ? 'enabled' : 'reduced');
}

/**
 * Convert camelCase token name to CSS custom property name.
 * colorPrimary → --color-primary
 * shadowSmall  → --shadow-small
 */
function tokenToCSSProperty(token: string): string {
	return '--' + token.replace(/([A-Z])/g, '-$1').toLowerCase();
}

/** Track applied override properties for cleanup */
let appliedOverrideProps: string[] = [];

/**
 * Apply custom overrides to DOM as CSS custom properties.
 * Overrides are applied directly to brand token names (no prefix).
 */
function applyOverridesToDOM(overrides: Record<string, string>): void {
	if (!browser) return;

	const style = document.documentElement.style;

	// Remove previously applied overrides
	for (const prop of appliedOverrideProps) {
		style.removeProperty(prop);
	}
	appliedOverrideProps = [];

	// Apply new overrides directly to CSS custom properties
	for (const [token, value] of Object.entries(overrides)) {
		const cssProp = tokenToCSSProperty(token);
		style.setProperty(cssProp, value);
		appliedOverrideProps.push(cssProp);
	}
}

/**
 * Apply all state to DOM.
 */
function applyAllToDOM(): void {
	const preset = getActivePreset();
	if (preset) {
		applyPresetToDOM(preset);
	}
	applyModeToDOM(state.systemMode);
	applyMotionToDOM(state.motionEnabled);
	applyOverridesToDOM(state.customOverrides);
}

// =============================================================================
// INITIALIZATION
// =============================================================================

/**
 * Initialize style store on client.
 * Call this in root layout or app entry.
 */
export function initStyleStore(): void {
	if (!browser) return;

	// Apply initial state
	applyAllToDOM();

	// Listen for storage changes (cross-tab sync)
	window.addEventListener('storage', (event) => {
		if (event.key === STORAGE_KEY && event.newValue) {
			try {
				const newState = JSON.parse(event.newValue) as IStyleState;
				state = { ...defaultState, ...newState };
				applyAllToDOM();
			} catch {
				// Ignore parse errors
			}
		}
	});

	// Listen for system color scheme changes
	const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
	mediaQuery.addEventListener('change', () => {
		if (state.systemMode === 'system') {
			const preset = getActivePreset();
			if (preset) {
				document.documentElement.style.colorScheme = getResolvedMode();
			}
		}
	});
}

// Auto-initialize in browser
if (browser) {
	// Defer to avoid hydration issues
	queueMicrotask(initStyleStore);
}
