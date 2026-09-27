/**
 * IBrandConfig - Brand Configuration Interface
 *
 * Extends IPresentable with brand-specific configuration for typography,
 * colors, spacing, borders, shadows, and theme variants.
 *
 * Architecture:
 *   branding.yml (schema)
 *       ↓ generates
 *   IBrandConfig (TypeScript interface)
 *       ↓ implements
 *   hub-brand.fixture.ts, shredbx-brand.fixture.ts
 *       ↓ transforms to
 *   CSS Custom Properties (:root, [data-brand="name"])
 *
 * Standards Referenced:
 * - W3C Design Tokens Format (DTCG 2025.10)
 * - Material Design 3 Typography
 * - Open Props naming conventions
 *
 * @example
 * const hubBrand: IBrandConfig = {
 *   id: 'hub-brand',
 *   name: 'Hub Application',
 *   category: 'brand',
 *   typography: { ... },
 *   colors: { ... },
 *   spacing: { ... }
 * };
 */

import type { IPresentable } from '../contracts/IPresentable';
import { PresentableRegistry } from '../contracts/IPresentable';
import type { TransitionPresetName } from '@sbx/animations';

// =============================================================================
// TYPOGRAPHY TYPES
// =============================================================================

/**
 * Typography scale entry following Material Design 3.
 */
export interface ITypographyScale {
	/** Font size with unit (e.g., "16px", "1rem", "clamp(1rem, 4vw, 1.5rem)") */
	fontSize: string;
	/** Line height with unit or unitless multiplier */
	lineHeight: string;
	/** Font weight (100-900) */
	fontWeight: number;
	/** Letter spacing (optional) */
	letterSpacing?: string;
	/** Font family override (optional) */
	fontFamily?: string;
}

/**
 * Typography configuration.
 */
export interface ITypographyConfig {
	/** Sans-serif font stack */
	fontFamilySans: string;
	/** Serif font stack (optional) */
	fontFamilySerif?: string;
	/** Monospace font stack (optional) */
	fontFamilyMono?: string;
	/** Email-safe font stack (optional) */
	fontFamilyEmail?: string;
	/** Base font size for body text */
	baseSize: string;
	/** Type scale ratio (e.g., "1.25" for Major Third) */
	scaleRatio?: string;
	/** Typography scales by name */
	scales: Record<string, ITypographyScale>;
}

// =============================================================================
// COLOR TYPES
// =============================================================================

/**
 * Semantic color palette.
 * Colors named by purpose, not appearance.
 */
export interface IColorPalette {
	/** Primary brand color */
	primary: string;
	/** Primary hover state */
	primaryHover?: string;
	/** Primary active state */
	primaryActive?: string;
	/** Secondary accent color */
	secondary?: string;
	/** Page background color */
	background: string;
	/** Surface color (cards, panels) */
	surface: string;
	/** Elevated surface color */
	surfaceElevated?: string;
	/** Primary text color */
	text: string;
	/** Muted text color */
	textMuted: string;
	/** Disabled text color */
	textDisabled?: string;
	/** Border color */
	border: string;
	/** Focus ring color */
	borderFocus?: string;
	/** Success status color */
	statusSuccess?: string;
	/** Warning status color */
	statusWarning?: string;
	/** Error status color */
	statusError?: string;
	/** Info status color */
	statusInfo?: string;
}

// =============================================================================
// SPACING TYPES
// =============================================================================

/**
 * Spacing scale configuration.
 */
export interface ISpacingConfig {
	/** Base spacing unit (e.g., "4px") */
	unit: string;
	/** Scale multipliers (generates --space-0, --space-1, etc.) */
	scale: number[];
}

// =============================================================================
// BORDER TYPES
// =============================================================================

/**
 * Border configuration.
 */
export interface IBorderConfig {
	/** No radius */
	radiusNone?: string;
	/** Small radius (inputs, badges) */
	radiusSmall: string;
	/** Medium radius (buttons, cards) */
	radiusMedium: string;
	/** Large radius (modals, panels) */
	radiusLarge: string;
	/** Full/pill radius */
	radiusFull: string;
	/** Default border width */
	widthDefault?: string;
	/** Thick border width */
	widthThick?: string;
}

// =============================================================================
// SHADOW TYPES
// =============================================================================

/**
 * Shadow/elevation configuration.
 */
export interface IShadowConfig {
	/** No shadow */
	none?: string;
	/** Subtle shadow (buttons, inputs) */
	small: string;
	/** Standard shadow (cards, dropdowns) */
	medium: string;
	/** Prominent shadow (modals, tooltips) */
	large: string;
	/** Maximum shadow (floating panels) */
	xl?: string;
}

// =============================================================================
// CONTEXT RULE TYPES
// =============================================================================

/**
 * HTML context types.
 */
export type HTMLContext = 'navigation' | 'email' | 'content' | 'form' | 'data' | 'hero' | 'footer';

/**
 * Typography roles.
 */
export type TypographyRole = 'display' | 'headline' | 'title' | 'body' | 'label';

/**
 * Font stack types.
 */
export type FontStack = 'sans' | 'serif' | 'mono' | 'email_safe';

/**
 * Context-specific HTML governance rule.
 */
export interface IContextRule {
	/** Context identifier */
	context: HTMLContext;
	/** Permitted HTML elements */
	allowedElements: string[];
	/** Semantic HTML rules */
	semanticRules: Array<{ rule: string }>;
	/** Default typography role */
	typographyRole: TypographyRole;
	/** Font stack for this context */
	fontStack: FontStack;
}

// =============================================================================
// VARIANT TYPES
// =============================================================================

/**
 * Theme variant (e.g., dark mode).
 * Overrides base tokens.
 */
export interface IBrandVariant {
	/** Variant identifier (used in data-theme attribute) */
	name: string;
	/** Human-readable description */
	description?: string;
	/** Color overrides */
	colors?: Partial<IColorPalette>;
	/** Typography overrides */
	typography?: Partial<Record<string, Partial<ITypographyScale>>>;
	/** Shadow overrides */
	shadows?: Partial<IShadowConfig>;
}

// =============================================================================
// IBRANDCONFIG INTERFACE
// =============================================================================

/**
 * IBrandConfig - Full brand configuration.
 * Extends IPresentable with brand-specific fields.
 */
export interface IBrandConfig extends IPresentable {
	// Override category to be more specific
	category: 'brand';
	entityType: 'brand';

	/** Human-readable display name */
	displayName: string;

	// =========================================================================
	// DESIGN TOKENS (Required)
	// =========================================================================

	/** Typography configuration */
	typography: ITypographyConfig;

	/** Color palette */
	colors: IColorPalette;

	/** Spacing scale */
	spacing: ISpacingConfig;

	/** Border configuration */
	borders: IBorderConfig;

	/** Shadow configuration */
	shadows: IShadowConfig;

	// =========================================================================
	// GOVERNANCE (Optional)
	// =========================================================================

	/** HTML context rules */
	contextRules?: IContextRule[];

	// =========================================================================
	// VARIANTS (Optional)
	// =========================================================================

	/** Theme variants (dark, high-contrast, etc.) */
	variants?: IBrandVariant[];

	// =========================================================================
	// METADATA (Optional)
	// =========================================================================

	/** Associated project */
	project?: string;

	/** Parent brand to extend */
	extends?: string;

	// =========================================================================
	// MOTION (Optional)
	// =========================================================================

	/** Transition / motion preset for scroll-reveal animations (Decision #0220). */
	transitions?: {
		/** Named preset applied to bare <Reveal> components. Default: 'minimal'. */
		preset: TransitionPresetName;
	};
}

// =============================================================================
// BRAND REGISTRY
// =============================================================================

/**
 * Global brand registry.
 */
export const brandRegistry = new PresentableRegistry<IBrandConfig>();

// =============================================================================
// CSS GENERATION
// =============================================================================

/**
 * Generate CSS custom properties from brand config.
 *
 * @example
 * const css = brandToCSS(hubBrand);
 * // Returns:
 * // :root, [data-brand="hub-brand"] {
 * //   --color-primary: #6366f1;
 * //   --typography-body-large-size: 16px;
 * // }
 */
export function brandToCSS(brand: IBrandConfig): string {
	const props: string[] = [];

	// Typography
	props.push(`--font-family-sans: ${brand.typography.fontFamilySans};`);
	if (brand.typography.fontFamilySerif) {
		props.push(`--font-family-serif: ${brand.typography.fontFamilySerif};`);
	}
	if (brand.typography.fontFamilyMono) {
		props.push(`--font-family-mono: ${brand.typography.fontFamilyMono};`);
	}
	props.push(`--typography-base-size: ${brand.typography.baseSize};`);

	// Typography scales
	for (const [name, scale] of Object.entries(brand.typography.scales)) {
		const prefix = `--typography-${name.replace(/_/g, '-')}`;
		props.push(`${prefix}-size: ${scale.fontSize};`);
		props.push(`${prefix}-line-height: ${scale.lineHeight};`);
		props.push(`${prefix}-weight: ${scale.fontWeight};`);
		if (scale.letterSpacing) {
			props.push(`${prefix}-letter-spacing: ${scale.letterSpacing};`);
		}
	}

	// Colors
	props.push(`--color-primary: ${brand.colors.primary};`);
	if (brand.colors.primaryHover) props.push(`--color-primary-hover: ${brand.colors.primaryHover};`);
	if (brand.colors.primaryActive) props.push(`--color-primary-active: ${brand.colors.primaryActive};`);
	if (brand.colors.secondary) props.push(`--color-secondary: ${brand.colors.secondary};`);
	props.push(`--color-background: ${brand.colors.background};`);
	props.push(`--color-surface: ${brand.colors.surface};`);
	if (brand.colors.surfaceElevated) props.push(`--color-surface-elevated: ${brand.colors.surfaceElevated};`);
	props.push(`--color-text: ${brand.colors.text};`);
	props.push(`--color-text-muted: ${brand.colors.textMuted};`);
	if (brand.colors.textDisabled) props.push(`--color-text-disabled: ${brand.colors.textDisabled};`);
	props.push(`--color-border: ${brand.colors.border};`);
	if (brand.colors.borderFocus) props.push(`--color-border-focus: ${brand.colors.borderFocus};`);
	if (brand.colors.statusSuccess) props.push(`--color-status-success: ${brand.colors.statusSuccess};`);
	if (brand.colors.statusWarning) props.push(`--color-status-warning: ${brand.colors.statusWarning};`);
	if (brand.colors.statusError) props.push(`--color-status-error: ${brand.colors.statusError};`);
	if (brand.colors.statusInfo) props.push(`--color-status-info: ${brand.colors.statusInfo};`);

	// Spacing
	brand.spacing.scale.forEach((multiplier, index) => {
		const value = multiplier === 0 ? '0' : `calc(${multiplier} * ${brand.spacing.unit})`;
		props.push(`--space-${index}: ${value};`);
	});

	// Borders
	props.push(`--border-radius-none: ${brand.borders.radiusNone ?? '0'};`);
	props.push(`--border-radius-small: ${brand.borders.radiusSmall};`);
	props.push(`--border-radius-medium: ${brand.borders.radiusMedium};`);
	props.push(`--border-radius-large: ${brand.borders.radiusLarge};`);
	props.push(`--border-radius-full: ${brand.borders.radiusFull};`);
	props.push(`--border-width-default: ${brand.borders.widthDefault ?? '1px'};`);
	props.push(`--border-width-thick: ${brand.borders.widthThick ?? '2px'};`);

	// Shadows
	props.push(`--shadow-none: ${brand.shadows.none ?? 'none'};`);
	props.push(`--shadow-small: ${brand.shadows.small};`);
	props.push(`--shadow-medium: ${brand.shadows.medium};`);
	props.push(`--shadow-large: ${brand.shadows.large};`);
	if (brand.shadows.xl) props.push(`--shadow-xl: ${brand.shadows.xl};`);

	// Build selector
	const selector = `:root, [data-brand="${brand.id}"]`;

	return `${selector} {\n  ${props.join('\n  ')}\n}`;
}

/**
 * Generate CSS for a brand variant.
 */
export function variantToCSS(brand: IBrandConfig, variant: IBrandVariant): string {
	const props: string[] = [];

	// Color overrides
	if (variant.colors) {
		for (const [key, value] of Object.entries(variant.colors)) {
			if (value) {
				const cssKey = key.replace(/([A-Z])/g, '-$1').toLowerCase();
				props.push(`--color-${cssKey}: ${value};`);
			}
		}
	}

	// Shadow overrides
	if (variant.shadows) {
		for (const [key, value] of Object.entries(variant.shadows)) {
			if (value) {
				props.push(`--shadow-${key}: ${value};`);
			}
		}
	}

	const selector = `[data-brand="${brand.id}"][data-theme="${variant.name}"]`;

	return `${selector} {\n  ${props.join('\n  ')}\n}`;
}

/**
 * Generate complete CSS for brand including all variants.
 */
export function generateBrandCSS(brand: IBrandConfig): string {
	const parts: string[] = [brandToCSS(brand)];

	if (brand.variants) {
		for (const variant of brand.variants) {
			parts.push(variantToCSS(brand, variant));
		}
	}

	return parts.join('\n\n');
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

/**
 * Create a minimal brand configuration.
 */
export function createBrand(
	id: string,
	name: string,
	config: Omit<IBrandConfig, 'id' | 'name' | 'category' | 'entityType' | 'displayName'>
): IBrandConfig {
	return {
		id,
		name,
		displayName: name,
		category: 'brand',
		entityType: 'brand',
		...config
	};
}

/**
 * Merge brand with variant overrides.
 */
export function applyVariant(brand: IBrandConfig, variantName: string): IBrandConfig {
	const variant = brand.variants?.find((v) => v.name === variantName);
	if (!variant) return brand;

	return {
		...brand,
		colors: { ...brand.colors, ...variant.colors },
		shadows: { ...brand.shadows, ...variant.shadows }
	};
}

/**
 * Get all variant names for a brand.
 */
export function getVariantNames(brand: IBrandConfig): string[] {
	return brand.variants?.map((v) => v.name) ?? [];
}
