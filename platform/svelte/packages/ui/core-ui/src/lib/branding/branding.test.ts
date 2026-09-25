/**
 * Branding System Tests (TDD)
 * ===========================
 *
 * Test suite for the branding system.
 * Follows TDD approach: tests define expected behavior.
 *
 * Test Categories:
 * 1. Token Existence - Required tokens must exist
 * 2. CSS Generation - Valid CSS output
 * 3. Variant Switching - Variants override base correctly
 * 4. Registry Operations - CRUD on brand registry
 * 5. IPresentable Contract - Brands satisfy base interface
 */

import { describe, it, expect, beforeEach } from 'vitest';
import {
	hubBrand,
	shredbxBrand,
	brandRegistry,
	generateBrandCSS,
	brandToCSS,
	variantToCSS,
	applyVariant,
	getVariantNames,
	createBrand
} from './index';
import type { IBrandConfig, IColorPalette } from './IBrandConfig';
import { isPresentable } from '../contracts/IPresentable';

describe('Branding System', () => {
	// =========================================================================
	// 1. TOKEN EXISTENCE TESTS
	// =========================================================================
	describe('Token Existence', () => {
		it('hubBrand has all required color tokens', () => {
			const colors = hubBrand.colors;
			expect(colors.primary).toBeDefined();
			expect(colors.background).toBeDefined();
			expect(colors.surface).toBeDefined();
			expect(colors.text).toBeDefined();
			expect(colors.textMuted).toBeDefined();
			expect(colors.border).toBeDefined();
		});

		it('hubBrand has all required typography tokens', () => {
			const typography = hubBrand.typography;
			expect(typography.fontFamilySans).toBeDefined();
			expect(typography.baseSize).toBeDefined();
			expect(typography.scales).toBeDefined();

			// Check for core scale categories
			expect(typography.scales.body_large).toBeDefined();
			expect(typography.scales.headline_large).toBeDefined();
			expect(typography.scales.label_large).toBeDefined();
		});

		it('hubBrand has all required spacing tokens', () => {
			const spacing = hubBrand.spacing;
			expect(spacing.unit).toBeDefined();
			expect(spacing.scale).toBeDefined();
			expect(spacing.scale.length).toBeGreaterThanOrEqual(5);
		});

		it('hubBrand has all required border tokens', () => {
			const borders = hubBrand.borders;
			expect(borders.radiusSmall).toBeDefined();
			expect(borders.radiusMedium).toBeDefined();
			expect(borders.radiusLarge).toBeDefined();
			expect(borders.radiusFull).toBeDefined();
		});

		it('hubBrand has all required shadow tokens', () => {
			const shadows = hubBrand.shadows;
			expect(shadows.small).toBeDefined();
			expect(shadows.medium).toBeDefined();
			expect(shadows.large).toBeDefined();
		});

		it('shredbxBrand has all required tokens', () => {
			expect(shredbxBrand.colors.primary).toBeDefined();
			expect(shredbxBrand.typography.fontFamilySans).toBeDefined();
			expect(shredbxBrand.spacing.unit).toBeDefined();
			expect(shredbxBrand.borders.radiusMedium).toBeDefined();
			expect(shredbxBrand.shadows.medium).toBeDefined();
		});
	});

	// =========================================================================
	// 2. CSS GENERATION TESTS
	// =========================================================================
	describe('CSS Generation', () => {
		it('generates valid CSS selector', () => {
			const css = brandToCSS(hubBrand);
			expect(css).toContain(':root, [data-brand="hub-brand"]');
		});

		it('generates color tokens', () => {
			const css = brandToCSS(hubBrand);
			expect(css).toContain('--color-primary:');
			expect(css).toContain('--color-background:');
			expect(css).toContain('--color-text:');
		});

		it('generates typography tokens', () => {
			const css = brandToCSS(hubBrand);
			expect(css).toContain('--font-family-sans:');
			expect(css).toContain('--typography-body-large-size:');
			expect(css).toContain('--typography-body-large-line-height:');
		});

		it('generates spacing tokens', () => {
			const css = brandToCSS(hubBrand);
			expect(css).toContain('--space-0:');
			expect(css).toContain('--space-4:');
		});

		it('generates border tokens', () => {
			const css = brandToCSS(hubBrand);
			expect(css).toContain('--border-radius-small:');
			expect(css).toContain('--border-radius-medium:');
		});

		it('generates shadow tokens', () => {
			const css = brandToCSS(hubBrand);
			expect(css).toContain('--shadow-small:');
			expect(css).toContain('--shadow-medium:');
		});

		it('generateBrandCSS includes variants', () => {
			const css = generateBrandCSS(hubBrand);
			expect(css).toContain('[data-brand="hub-brand"][data-theme="light"]');
		});
	});

	// =========================================================================
	// 3. VARIANT SWITCHING TESTS
	// =========================================================================
	describe('Variant Switching', () => {
		it('getVariantNames returns all variant names', () => {
			const names = getVariantNames(hubBrand);
			expect(names).toContain('light');
			expect(names).toContain('high-contrast');
		});

		it('applyVariant overrides base colors', () => {
			const lightBrand = applyVariant(hubBrand, 'light');

			// Light variant should have light background
			expect(lightBrand.colors.background).toBe('#ffffff');
			expect(lightBrand.colors.text).toBe('#111827');
		});

		it('applyVariant preserves non-overridden tokens', () => {
			const lightBrand = applyVariant(hubBrand, 'light');

			// Primary should remain unchanged (not in light variant overrides)
			expect(lightBrand.colors.primary).toBe(hubBrand.colors.primary);
		});

		it('applyVariant returns base if variant not found', () => {
			const result = applyVariant(hubBrand, 'nonexistent');
			expect(result).toEqual(hubBrand);
		});

		it('variantToCSS generates correct selector', () => {
			const variant = hubBrand.variants![0]; // light
			const css = variantToCSS(hubBrand, variant);
			expect(css).toContain('[data-brand="hub-brand"][data-theme="light"]');
		});
	});

	// =========================================================================
	// 4. REGISTRY TESTS
	// =========================================================================
	describe('Brand Registry', () => {
		beforeEach(() => {
			// Reset registry for each test
			brandRegistry.items.clear();
			brandRegistry.register(hubBrand);
			brandRegistry.register(shredbxBrand);
		});

		it('registers brands correctly', () => {
			expect(brandRegistry.items.size).toBe(2);
		});

		it('retrieves brand by id', () => {
			const brand = brandRegistry.get('hub-brand');
			expect(brand).toBeDefined();
			expect(brand?.name).toBe('Hub Application');
		});

		it('getByCategory returns brands', () => {
			const brands = brandRegistry.getByCategory('brand');
			expect(brands.length).toBe(2);
		});

		it('search finds brands by name', () => {
			const results = brandRegistry.search('hub');
			expect(results.length).toBeGreaterThanOrEqual(1);
			expect(results[0].id).toBe('hub-brand');
		});

		it('search finds brands by tag', () => {
			const results = brandRegistry.search('rose');
			expect(results.length).toBeGreaterThanOrEqual(1);
			expect(results[0].id).toBe('shredbx-brand');
		});
	});

	// =========================================================================
	// 5. IPRESENTABLE CONTRACT TESTS
	// =========================================================================
	describe('IPresentable Contract', () => {
		it('hubBrand satisfies IPresentable', () => {
			expect(isPresentable(hubBrand)).toBe(true);
		});

		it('shredbxBrand satisfies IPresentable', () => {
			expect(isPresentable(shredbxBrand)).toBe(true);
		});

		it('brands have required identity fields', () => {
			expect(hubBrand.id).toBeDefined();
			expect(hubBrand.name).toBeDefined();
			expect(hubBrand.category).toBe('brand');
		});

		it('brands have optional metadata fields', () => {
			expect(hubBrand.description).toBeDefined();
			expect(hubBrand.tags).toBeDefined();
			expect(hubBrand.status).toBeDefined();
			expect(hubBrand.version).toBeDefined();
		});
	});

	// =========================================================================
	// 6. FACTORY TESTS
	// =========================================================================
	describe('Brand Factory', () => {
		it('createBrand generates valid brand', () => {
			const brand = createBrand('test-brand', 'Test Brand', {
				typography: {
					fontFamilySans: 'Arial, sans-serif',
					baseSize: '16px',
					scales: {}
				},
				colors: {
					primary: '#000000',
					background: '#ffffff',
					surface: '#f0f0f0',
					text: '#000000',
					textMuted: '#666666',
					border: '#cccccc'
				},
				spacing: {
					unit: '4px',
					scale: [0, 1, 2, 3, 4]
				},
				borders: {
					radiusSmall: '2px',
					radiusMedium: '4px',
					radiusLarge: '8px',
					radiusFull: '9999px'
				},
				shadows: {
					small: '0 1px 2px rgba(0,0,0,0.1)',
					medium: '0 2px 4px rgba(0,0,0,0.1)',
					large: '0 4px 8px rgba(0,0,0,0.1)'
				}
			});

			expect(brand.id).toBe('test-brand');
			expect(brand.name).toBe('Test Brand');
			expect(brand.category).toBe('brand');
			expect(brand.entityType).toBe('brand');
			expect(isPresentable(brand)).toBe(true);
		});
	});

	// =========================================================================
	// 7. ACCESSIBILITY TESTS (Contrast)
	// =========================================================================
	describe('Accessibility', () => {
		/**
		 * Helper to calculate relative luminance
		 * Formula: https://www.w3.org/TR/WCAG20/#relativeluminancedef
		 */
		function getLuminance(hex: string): number {
			const rgb = hex.slice(1).match(/.{2}/g)!.map(c => parseInt(c, 16) / 255);
			const [r, g, b] = rgb.map(c =>
				c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4)
			);
			return 0.2126 * r + 0.7152 * g + 0.0722 * b;
		}

		/**
		 * Calculate contrast ratio between two colors
		 * Formula: https://www.w3.org/TR/WCAG20/#contrast-ratiodef
		 */
		function getContrastRatio(color1: string, color2: string): number {
			const l1 = getLuminance(color1);
			const l2 = getLuminance(color2);
			const lighter = Math.max(l1, l2);
			const darker = Math.min(l1, l2);
			return (lighter + 0.05) / (darker + 0.05);
		}

		it('hubBrand text/background meets WCAG AA (4.5:1)', () => {
			const ratio = getContrastRatio(hubBrand.colors.text, hubBrand.colors.background);
			expect(ratio).toBeGreaterThanOrEqual(4.5);
		});

		it('hubBrand light variant meets WCAG AA', () => {
			const lightBrand = applyVariant(hubBrand, 'light');
			const ratio = getContrastRatio(lightBrand.colors.text, lightBrand.colors.background);
			expect(ratio).toBeGreaterThanOrEqual(4.5);
		});

		it('shredbxBrand text/background meets WCAG AA', () => {
			const ratio = getContrastRatio(shredbxBrand.colors.text, shredbxBrand.colors.background);
			expect(ratio).toBeGreaterThanOrEqual(4.5);
		});
	});
});
