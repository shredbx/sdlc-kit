/**
 * Style Module - Style preset management and persistence
 *
 * Provides:
 * 1. Style preset interface and registry
 * 2. Persisted style state with localStorage
 * 3. DOM integration for theme switching
 * 4. Color contrast validation (Phase 5)
 *
 * Usage:
 *   import {
 *     styleStore,
 *     setStylePreset,
 *     getActivePreset,
 *     stylePresetRegistry
 *   } from '.';
 *
 *   // Change theme
 *   setStylePreset('hub-light');
 *
 *   // Get current preset
 *   const preset = getActivePreset();
 *
 * @see IStylePreset.ts - Interface definitions
 * @see styleStore.ts - Persisted state
 */

// =============================================================================
// INTERFACES & TYPES
// =============================================================================

export type {
	StyleMode,
	PresetStatus,
	ITokenOverrides,
	IStylePreset,
	IStylePresetRegistry
} from './IStylePreset';

export type { IStyleState } from './styleStore.svelte';

// =============================================================================
// REGISTRY
// =============================================================================

export {
	StylePresetRegistry,
	stylePresetRegistry,
	createStylePreset,
	hubDarkPreset,
	hubLightPreset,
	hubHighContrastPreset
} from './IStylePreset';

// =============================================================================
// STORE
// =============================================================================

export {
	styleStore,
	setStylePreset,
	setSystemMode,
	toggleMotion,
	setTokenOverride,
	clearOverrides,
	resetStyleState,
	getActivePreset,
	getResolvedMode,
	initStyleStore
} from './styleStore.svelte';

// =============================================================================
// CONTRAST VALIDATION
// =============================================================================

export type {
	Zone,
	ValidationResult,
	ValidationOptions,
	ContrastReport,
	ContrastIssue
} from './contrastValidator';

export {
	getZone,
	getLightness,
	validatePairing,
	validateBrandContrast,
	getZoneLabel,
	getZoneClass,
	isNearThreshold,
	suggestSaferColor,
	tailwindShadeToZone,
	validateTailwindPairing
} from './contrastValidator';
