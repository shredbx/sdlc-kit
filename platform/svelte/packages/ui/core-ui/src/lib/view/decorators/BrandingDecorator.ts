/**
 * BrandingDecorator - Theme and branding customization
 *
 * Applies brand-specific styles to components at runtime.
 * Supports theme switching, custom colors, typography, and spacing.
 *
 * Features:
 * - CSS custom properties injection
 * - Theme mode switching (light/dark/system)
 * - Per-component overrides
 * - Brand preset loading
 * - CSS-in-JS integration ready
 *
 * @example
 * const view: IView = {
 *   id: 'card-1',
 *   name: 'Product Card',
 *   component: 'Card',
 *   category: 'blocks',
 *   decorators: [createBrandingDecorator({ theme: 'dark' })]
 * };
 */

import type {
	IDecorator,
	IDecoratorContext,
	IDecoratorHooks,
	IView,
	IThemeConfig,
	IThemeOverrides
} from '../IView';

// =============================================================================
// CONFIGURATION
// =============================================================================

/**
 * Branding decorator configuration.
 */
export interface BrandingConfig {
	/** Theme mode: light, dark, or follow system. Default: 'system' */
	mode?: 'light' | 'dark' | 'system';

	/** Brand preset name to load (e.g., 'hub', 'shredbx') */
	brand?: string;

	/** Color overrides */
	colors?: Record<string, string>;

	/** Spacing overrides */
	spacing?: Record<string, string>;

	/** Typography overrides */
	typography?: Record<string, string>;

	/** Border overrides */
	borders?: Record<string, string>;

	/** Shadow overrides */
	shadows?: Record<string, string>;

	/** Use CSS-in-JS (inline styles) instead of custom properties. Default: false */
	inline?: boolean;

	/** Apply to children as well. Default: true */
	cascade?: boolean;

	/** Animation duration for theme transitions in ms. Default: 200 */
	transitionDuration?: number;
}

const DEFAULT_CONFIG: BrandingConfig = {
	mode: 'system',
	inline: false,
	cascade: true,
	transitionDuration: 200
};

// =============================================================================
// BRAND PRESETS
// =============================================================================

/**
 * Built-in brand presets.
 * These can be loaded by name.
 */
export const BRAND_PRESETS: Record<string, IThemeConfig> = {
	hub: {
		name: 'Hub',
		mode: 'dark',
		colors: {
			bg: '#0f0f1a',
			'bg-secondary': '#1a1a2e',
			'bg-tertiary': '#252540',
			'bg-hover': 'rgba(255, 255, 255, 0.05)',
			text: '#ffffff',
			'text-muted': '#9ca3af',
			'text-subtle': '#6b7280',
			accent: '#6366f1',
			'accent-hover': '#818cf8',
			success: '#22c55e',
			warning: '#f59e0b',
			error: '#ef4444',
			info: '#3b82f6',
			border: '#2a2a4a'
		},
		spacing: {
			xs: '0.25rem',
			sm: '0.5rem',
			md: '1rem',
			lg: '1.5rem',
			xl: '2rem',
			'2xl': '3rem'
		},
		typography: {
			'font-sans': 'system-ui, -apple-system, sans-serif',
			'font-mono': 'JetBrains Mono, Fira Code, monospace',
			'text-xs': '0.75rem',
			'text-sm': '0.875rem',
			'text-base': '1rem',
			'text-lg': '1.125rem',
			'text-xl': '1.25rem',
			'text-2xl': '1.5rem'
		},
		borders: {
			radius: '0.375rem',
			'radius-sm': '0.25rem',
			'radius-lg': '0.5rem',
			'radius-full': '9999px',
			width: '1px'
		},
		shadows: {
			sm: '0 1px 2px rgba(0, 0, 0, 0.3)',
			md: '0 4px 6px rgba(0, 0, 0, 0.3)',
			lg: '0 10px 15px rgba(0, 0, 0, 0.3)',
			glow: '0 0 20px rgba(99, 102, 241, 0.3)'
		}
	},
	'hub-light': {
		name: 'Hub Light',
		mode: 'light',
		colors: {
			bg: '#ffffff',
			'bg-secondary': '#f9fafb',
			'bg-tertiary': '#f3f4f6',
			'bg-hover': 'rgba(0, 0, 0, 0.05)',
			text: '#111827',
			'text-muted': '#6b7280',
			'text-subtle': '#9ca3af',
			accent: '#4f46e5',
			'accent-hover': '#6366f1',
			success: '#16a34a',
			warning: '#d97706',
			error: '#dc2626',
			info: '#2563eb',
			border: '#e5e7eb'
		},
		spacing: {
			xs: '0.25rem',
			sm: '0.5rem',
			md: '1rem',
			lg: '1.5rem',
			xl: '2rem',
			'2xl': '3rem'
		},
		typography: {
			'font-sans': 'system-ui, -apple-system, sans-serif',
			'font-mono': 'JetBrains Mono, Fira Code, monospace',
			'text-xs': '0.75rem',
			'text-sm': '0.875rem',
			'text-base': '1rem',
			'text-lg': '1.125rem',
			'text-xl': '1.25rem',
			'text-2xl': '1.5rem'
		},
		borders: {
			radius: '0.375rem',
			'radius-sm': '0.25rem',
			'radius-lg': '0.5rem',
			'radius-full': '9999px',
			width: '1px'
		},
		shadows: {
			sm: '0 1px 2px rgba(0, 0, 0, 0.1)',
			md: '0 4px 6px rgba(0, 0, 0, 0.1)',
			lg: '0 10px 15px rgba(0, 0, 0, 0.1)',
			glow: '0 0 20px rgba(79, 70, 229, 0.2)'
		}
	},
	shredbx: {
		name: 'ShredBX',
		mode: 'dark',
		colors: {
			bg: '#0a0a0f',
			'bg-secondary': '#121218',
			'bg-tertiary': '#1c1c24',
			'bg-hover': 'rgba(255, 255, 255, 0.08)',
			text: '#fafafa',
			'text-muted': '#a1a1aa',
			'text-subtle': '#71717a',
			accent: '#f97316',
			'accent-hover': '#fb923c',
			success: '#22c55e',
			warning: '#eab308',
			error: '#ef4444',
			info: '#06b6d4',
			border: '#27272a'
		},
		spacing: {
			xs: '0.25rem',
			sm: '0.5rem',
			md: '1rem',
			lg: '1.5rem',
			xl: '2rem',
			'2xl': '3rem'
		},
		typography: {
			'font-sans': 'Inter, system-ui, sans-serif',
			'font-mono': 'JetBrains Mono, monospace',
			'text-xs': '0.75rem',
			'text-sm': '0.875rem',
			'text-base': '1rem',
			'text-lg': '1.125rem',
			'text-xl': '1.25rem',
			'text-2xl': '1.5rem'
		},
		borders: {
			radius: '0.5rem',
			'radius-sm': '0.25rem',
			'radius-lg': '0.75rem',
			'radius-full': '9999px',
			width: '1px'
		},
		shadows: {
			sm: '0 1px 3px rgba(0, 0, 0, 0.4)',
			md: '0 4px 8px rgba(0, 0, 0, 0.4)',
			lg: '0 10px 20px rgba(0, 0, 0, 0.4)',
			glow: '0 0 20px rgba(249, 115, 22, 0.3)'
		}
	}
};

// =============================================================================
// STATE
// =============================================================================

/** Current system theme mode */
let systemMode: 'light' | 'dark' = 'dark';

// Detect system preference
if (typeof window !== 'undefined') {
	const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
	systemMode = mediaQuery.matches ? 'dark' : 'light';

	mediaQuery.addEventListener('change', (e) => {
		systemMode = e.matches ? 'dark' : 'light';
	});
}

// =============================================================================
// THEME UTILITIES
// =============================================================================

/**
 * Get effective theme mode (resolves 'system' to actual mode).
 */
function getEffectiveMode(mode: 'light' | 'dark' | 'system'): 'light' | 'dark' {
	return mode === 'system' ? systemMode : mode;
}

/**
 * Merge theme overrides with base theme.
 */
function mergeTheme(base: IThemeConfig, overrides?: IThemeOverrides): IThemeConfig {
	if (!overrides) return base;

	return {
		...base,
		colors: { ...base.colors, ...overrides.colors },
		spacing: { ...base.spacing, ...overrides.spacing },
		typography: { ...base.typography, ...overrides.typography },
		borders: { ...base.borders, ...overrides.borders },
		shadows: { ...base.shadows, ...overrides.shadows }
	};
}

/**
 * Convert theme to CSS custom properties string.
 */
function themeToCSSProperties(theme: IThemeConfig): string {
	const props: string[] = [];

	if (theme.colors) {
		for (const [key, value] of Object.entries(theme.colors)) {
			props.push(`--color-${key}: ${value}`);
		}
	}

	if (theme.spacing) {
		for (const [key, value] of Object.entries(theme.spacing)) {
			props.push(`--spacing-${key}: ${value}`);
		}
	}

	if (theme.typography) {
		for (const [key, value] of Object.entries(theme.typography)) {
			props.push(`--${key}: ${value}`);
		}
	}

	if (theme.borders) {
		for (const [key, value] of Object.entries(theme.borders)) {
			props.push(`--border-${key}: ${value}`);
		}
	}

	if (theme.shadows) {
		for (const [key, value] of Object.entries(theme.shadows)) {
			props.push(`--shadow-${key}: ${value}`);
		}
	}

	return props.join('; ');
}

/**
 * Apply CSS custom properties to element.
 */
function applyThemeToElement(
	element: HTMLElement,
	theme: IThemeConfig,
	transitionDuration: number
): void {
	// Add transition for smooth theme changes
	element.style.transition = `all ${transitionDuration}ms ease`;

	// Apply custom properties
	if (theme.colors) {
		for (const [key, value] of Object.entries(theme.colors)) {
			element.style.setProperty(`--color-${key}`, value);
		}
	}

	if (theme.spacing) {
		for (const [key, value] of Object.entries(theme.spacing)) {
			element.style.setProperty(`--spacing-${key}`, value);
		}
	}

	if (theme.typography) {
		for (const [key, value] of Object.entries(theme.typography)) {
			element.style.setProperty(`--${key}`, value);
		}
	}

	if (theme.borders) {
		for (const [key, value] of Object.entries(theme.borders)) {
			element.style.setProperty(`--border-${key}`, value);
		}
	}

	if (theme.shadows) {
		for (const [key, value] of Object.entries(theme.shadows)) {
			element.style.setProperty(`--shadow-${key}`, value);
		}
	}

	// Set data attribute for CSS selectors
	element.dataset.theme = theme.mode;
	element.dataset.brand = theme.name.toLowerCase().replace(/\s+/g, '-');
}

// =============================================================================
// HOOKS
// =============================================================================

/**
 * Create Branding decorator hooks.
 */
function createHooks(config: BrandingConfig): IDecoratorHooks {
	return {
		onAfterMount(view: IView, context: IDecoratorContext, element: HTMLElement): void {
			// Build theme from config
			let theme: IThemeConfig;

			// Start with brand preset if specified
			if (config.brand && BRAND_PRESETS[config.brand]) {
				theme = { ...BRAND_PRESETS[config.brand] };
			} else {
				// Use context theme as base
				theme = { ...context.theme };
			}

			// Override mode if specified
			if (config.mode) {
				theme.mode = getEffectiveMode(config.mode);
			}

			// Merge view-level theme overrides
			if (view.theme) {
				theme = mergeTheme(theme, view.theme);
			}

			// Merge config-level overrides
			const overrides: IThemeOverrides = {
				colors: config.colors,
				spacing: config.spacing,
				typography: config.typography,
				borders: config.borders,
				shadows: config.shadows
			};
			theme = mergeTheme(theme, overrides);

			// Apply theme to element
			applyThemeToElement(element, theme, config.transitionDuration ?? 200);

			// Listen for theme change events
			const unsubscribe = context.on('theme:change', (payload) => {
				const newTheme = payload as IThemeConfig;
				applyThemeToElement(element, newTheme, config.transitionDuration ?? 200);
			});

			// Store cleanup for destroy
			(element as unknown as { __brandingCleanup?: () => void }).__brandingCleanup = unsubscribe;
		},

		onBeforeDestroy(view: IView, context: IDecoratorContext): void {
			const element = context.getElement();
			if (element) {
				const cleanup = (element as unknown as { __brandingCleanup?: () => void })
					.__brandingCleanup;
				if (cleanup) cleanup();
			}
		}
	};
}

// =============================================================================
// FACTORY
// =============================================================================

/**
 * Create Branding decorator with configuration.
 *
 * @example
 * // Use brand preset
 * const decorator = createBrandingDecorator({ brand: 'hub' });
 *
 * @example
 * // Custom colors
 * const decorator = createBrandingDecorator({
 *   colors: { accent: '#ff6b6b' }
 * });
 */
export function createBrandingDecorator(config?: BrandingConfig): IDecorator<BrandingConfig> {
	const mergedConfig = { ...DEFAULT_CONFIG, ...config };

	return {
		id: 'branding',
		name: 'Branding',
		priority: 200, // Run after DevMode
		enabled: true,
		config: mergedConfig,
		hooks: createHooks(mergedConfig)
	};
}

// =============================================================================
// UTILITIES
// =============================================================================

/**
 * Get available brand preset names.
 */
export function getAvailableBrands(): string[] {
	return Object.keys(BRAND_PRESETS);
}

/**
 * Get brand preset by name.
 */
export function getBrandPreset(name: string): IThemeConfig | undefined {
	return BRAND_PRESETS[name];
}

/**
 * Register a custom brand preset.
 */
export function registerBrand(name: string, theme: IThemeConfig): void {
	BRAND_PRESETS[name] = theme;
}

/**
 * Get current system color scheme preference.
 */
export function getSystemMode(): 'light' | 'dark' {
	return systemMode;
}

/**
 * Switch theme for a specific element.
 */
export function switchTheme(
	element: HTMLElement,
	theme: IThemeConfig,
	transitionDuration = 200
): void {
	applyThemeToElement(element, theme, transitionDuration);
}

/**
 * Get CSS custom properties string for a theme.
 */
export function getThemeCSS(theme: IThemeConfig): string {
	return themeToCSSProperties(theme);
}

// =============================================================================
// DEFAULT EXPORT
// =============================================================================

export default createBrandingDecorator;
