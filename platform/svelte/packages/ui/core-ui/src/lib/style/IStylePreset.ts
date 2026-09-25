/**
 * IStylePreset - Style Preset Interface
 *
 * Style presets are concrete configurations that combine:
 * - A brand reference (design tokens source)
 * - A brand variant (light/dark)
 * - Optional token overrides
 *
 * Presets enable users to switch themes without changing the underlying brand.
 *
 * Architecture:
 *   IBrandConfig (base tokens)
 *       ↓ referenced by
 *   IStylePreset (preset config)
 *       ↓ resolved by
 *   styleStore (runtime state + localStorage)
 *       ↓ applies
 *   CSS custom properties via data attributes
 *
 * @example
 * // Hub Dark preset - uses hub-brand with no overrides
 * const hubDark: IStylePreset = {
 *   id: 'hub-dark',
 *   name: 'Hub Dark',
 *   mode: 'dark',
 *   brandId: 'hub-brand',
 *   brandVariant: 'dark'
 * };
 */

// =============================================================================
// STYLE MODE TYPE
// =============================================================================

/** Color scheme mode */
export type StyleMode = 'light' | 'dark' | 'system';

/** Preset status */
export type PresetStatus = 'draft' | 'stable' | 'deprecated';

// =============================================================================
// TOKEN OVERRIDES
// =============================================================================

/**
 * Overridable design tokens.
 * Only include tokens you want to override from brand defaults.
 */
export interface ITokenOverrides {
	// Colors
	colorPrimary?: string;
	colorBackground?: string;
	colorSurface?: string;
	colorSurfaceElevated?: string;
	colorText?: string;
	colorTextMuted?: string;
	colorTextDisabled?: string;
	colorBorder?: string;
	// Shadows
	shadowSmall?: string;
	shadowMedium?: string;
	shadowLarge?: string;
	// Add more as needed
}

// =============================================================================
// ISTYLEPRESET INTERFACE
// =============================================================================

/**
 * Style preset configuration.
 */
export interface IStylePreset {
	/** Unique identifier (kebab-case) */
	id: string;

	/** Display name */
	name: string;

	/** Short description */
	description?: string;

	/** Color mode (light/dark) */
	mode: 'light' | 'dark';

	/** Reference to brand configuration */
	brandId: string;

	/** Brand variant to use (e.g., 'dark', 'light') */
	brandVariant?: string;

	/** Token overrides (optional) */
	tokens?: ITokenOverrides;

	/** Preset status */
	status?: PresetStatus;

	/** Whether this is a user-created preset */
	isCustom?: boolean;
}

// =============================================================================
// STYLE PRESET REGISTRY
// =============================================================================

/**
 * Registry interface for style presets.
 */
export interface IStylePresetRegistry {
	/** All registered presets */
	presets: Map<string, IStylePreset>;

	/** Register a preset */
	register(preset: IStylePreset): void;

	/** Get preset by ID */
	get(id: string): IStylePreset | undefined;

	/** Get all presets */
	getAll(): IStylePreset[];

	/** Get presets by mode */
	getByMode(mode: 'light' | 'dark'): IStylePreset[];

	/** Get presets by brand */
	getByBrand(brandId: string): IStylePreset[];

	/** Get default preset */
	getDefault(): IStylePreset | undefined;
}

/**
 * Style preset registry implementation.
 */
export class StylePresetRegistry implements IStylePresetRegistry {
	presets = new Map<string, IStylePreset>();
	private defaultId = 'hub-dark';

	register(preset: IStylePreset): void {
		this.presets.set(preset.id, preset);
	}

	get(id: string): IStylePreset | undefined {
		return this.presets.get(id);
	}

	getAll(): IStylePreset[] {
		return Array.from(this.presets.values());
	}

	getByMode(mode: 'light' | 'dark'): IStylePreset[] {
		return Array.from(this.presets.values()).filter((p) => p.mode === mode);
	}

	getByBrand(brandId: string): IStylePreset[] {
		return Array.from(this.presets.values()).filter((p) => p.brandId === brandId);
	}

	getDefault(): IStylePreset | undefined {
		return this.presets.get(this.defaultId);
	}

	setDefault(id: string): void {
		if (this.presets.has(id)) {
			this.defaultId = id;
		}
	}
}

// =============================================================================
// GLOBAL REGISTRY INSTANCE
// =============================================================================

/** Global style preset registry */
export const stylePresetRegistry = new StylePresetRegistry();

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

/**
 * Create a style preset.
 */
export function createStylePreset(
	id: string,
	name: string,
	mode: 'light' | 'dark',
	brandId: string,
	options?: Partial<IStylePreset>
): IStylePreset {
	return {
		id,
		name,
		mode,
		brandId,
		status: 'stable',
		...options
	};
}

// =============================================================================
// BUILTIN PRESETS
// =============================================================================

/** Hub Dark - Default workspace theme */
export const hubDarkPreset: IStylePreset = {
	id: 'hub-dark',
	name: 'Hub Dark',
	description: 'Default dark theme for developer productivity',
	mode: 'dark',
	brandId: 'hub',
	brandVariant: 'dark',
	status: 'stable'
};

/** Hub Light - Light mode variant */
export const hubLightPreset: IStylePreset = {
	id: 'hub-light',
	name: 'Hub Light',
	description: 'Light theme for bright environments',
	mode: 'light',
	brandId: 'hub',
	brandVariant: 'light',
	status: 'stable'
};

/** Hub High Contrast - Accessibility variant */
export const hubHighContrastPreset: IStylePreset = {
	id: 'hub-high-contrast',
	name: 'Hub High Contrast',
	description: 'High contrast for accessibility',
	mode: 'dark',
	brandId: 'hub',
	brandVariant: 'dark',
	tokens: {
		colorText: '#ffffff',
		colorTextMuted: '#e5e7eb',
		colorBackground: '#000000',
		colorSurface: '#0a0a0a',
		colorBorder: '#ffffff'
	},
	status: 'stable'
};

/** shredbx Dark - Developer-focused dark theme */
export const shredbxDarkPreset: IStylePreset = {
	id: 'shredbx-dark',
	name: 'shredbx Dark',
	description: 'Developer-focused dark theme with red accents',
	mode: 'dark',
	brandId: 'shredbx',
	brandVariant: 'dark',
	status: 'stable'
};

/** shredbx Light - Light mode variant */
export const shredbxLightPreset: IStylePreset = {
	id: 'shredbx-light',
	name: 'shredbx Light',
	description: 'Light theme for documentation reading',
	mode: 'light',
	brandId: 'shredbx',
	brandVariant: 'light',
	status: 'stable'
};

// Register builtins
stylePresetRegistry.register(hubDarkPreset);
stylePresetRegistry.register(hubLightPreset);
stylePresetRegistry.register(hubHighContrastPreset);
stylePresetRegistry.register(shredbxDarkPreset);
stylePresetRegistry.register(shredbxLightPreset);
