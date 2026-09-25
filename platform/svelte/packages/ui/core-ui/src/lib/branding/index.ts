/**
 * Branding Module Index
 * =====================
 *
 * Central export for all branding functionality.
 *
 * Usage:
 *   import { hubBrand, brandRegistry, generateBrandCSS } from '.';
 */

// =============================================================================
// INTERFACES
// =============================================================================

export type {
	ITypographyScale,
	ITypographyConfig,
	IColorPalette,
	ISpacingConfig,
	IBorderConfig,
	IShadowConfig,
	HTMLContext,
	TypographyRole,
	FontStack,
	IContextRule,
	IBrandVariant,
	IBrandConfig
} from './IBrandConfig.js';

// =============================================================================
// REGISTRY & HELPERS
// =============================================================================

export {
	brandRegistry,
	brandToCSS,
	variantToCSS,
	generateBrandCSS,
	createBrand,
	applyVariant,
	getVariantNames
} from './IBrandConfig.js';

// =============================================================================
// FIXTURES
// =============================================================================

export { hubBrand } from './hub-brand.fixture.js';
export { shredbxBrand } from './shredbx-brand.fixture.js';
