/**
 * Hub Brand Fixture
 * =================
 *
 * Default branding for the Hub application.
 * Dark theme primary, with light mode variant.
 *
 * Standards:
 * - Material Design 3 Typography Scale
 * - W3C Design Tokens semantic naming
 * - Open Props spacing scale
 *
 * Usage:
 * 1. Import and register: brandRegistry.register(hubBrand)
 * 2. Generate CSS: generateBrandCSS(hubBrand)
 * 3. Apply via attribute: <div data-brand="hub-brand">
 * 4. Switch variant: <div data-brand="hub-brand" data-theme="light">
 */

import type { IBrandConfig } from './IBrandConfig';
import { brandRegistry } from './IBrandConfig';

/**
 * Hub Brand Configuration
 */
export const hubBrand: IBrandConfig = {
	// =========================================================================
	// IDENTITY (IPresentable)
	// =========================================================================
	id: 'hub-brand',
	name: 'Hub Application',
	displayName: 'Hub Application',
	category: 'brand',
	entityType: 'brand',
	description: 'Default branding for Hub - dark mode development environment',
	tags: ['hub', 'dark', 'developer', 'workspace'],
	status: 'stable',
	version: '1.0.0',
	project: 'hub',

	// =========================================================================
	// TYPOGRAPHY
	// =========================================================================
	typography: {
		fontFamilySans: 'Inter, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
		fontFamilySerif: 'Georgia, Cambria, "Times New Roman", Times, serif',
		fontFamilyMono: '"JetBrains Mono", ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace',
		fontFamilyEmail: 'Arial, Helvetica, sans-serif',
		baseSize: '16px',
		scaleRatio: '1.25', // Major Third

		scales: {
			// Display (Hero text)
			display_large: {
				fontSize: '57px',
				lineHeight: '64px',
				fontWeight: 400,
				letterSpacing: '-0.25px'
			},
			display_medium: {
				fontSize: '45px',
				lineHeight: '52px',
				fontWeight: 400
			},
			display_small: {
				fontSize: '36px',
				lineHeight: '44px',
				fontWeight: 400
			},

			// Headline (Section headers)
			headline_large: {
				fontSize: '32px',
				lineHeight: '40px',
				fontWeight: 400
			},
			headline_medium: {
				fontSize: '28px',
				lineHeight: '36px',
				fontWeight: 400
			},
			headline_small: {
				fontSize: '24px',
				lineHeight: '32px',
				fontWeight: 400
			},

			// Title (Card titles)
			title_large: {
				fontSize: '22px',
				lineHeight: '28px',
				fontWeight: 500
			},
			title_medium: {
				fontSize: '16px',
				lineHeight: '24px',
				fontWeight: 500
			},
			title_small: {
				fontSize: '14px',
				lineHeight: '20px',
				fontWeight: 500
			},

			// Body (Paragraphs)
			body_large: {
				fontSize: '16px',
				lineHeight: '24px',
				fontWeight: 400
			},
			body_medium: {
				fontSize: '14px',
				lineHeight: '20px',
				fontWeight: 400
			},
			body_small: {
				fontSize: '12px',
				lineHeight: '16px',
				fontWeight: 400
			},

			// Label (Buttons, form labels)
			label_large: {
				fontSize: '14px',
				lineHeight: '20px',
				fontWeight: 500
			},
			label_medium: {
				fontSize: '12px',
				lineHeight: '16px',
				fontWeight: 500
			},
			label_small: {
				fontSize: '11px',
				lineHeight: '16px',
				fontWeight: 500
			}
		}
	},

	// =========================================================================
	// COLORS (Dark Theme Base)
	// =========================================================================
	colors: {
		// Primary (Indigo)
		primary: '#6366f1',
		primaryHover: '#4f46e5',
		primaryActive: '#4338ca',

		// Secondary (Pink)
		secondary: '#ec4899',

		// Backgrounds
		background: '#0f0f1a',
		surface: '#1a1a2e',
		surfaceElevated: '#252540',

		// Text
		text: '#ffffff',
		textMuted: '#9ca3af',
		textDisabled: '#6b7280',

		// Borders
		border: '#2a2a4a',
		borderFocus: '#6366f1',

		// Status
		statusSuccess: '#10b981',
		statusWarning: '#f59e0b',
		statusError: '#ef4444',
		statusInfo: '#3b82f6'
	},

	// =========================================================================
	// SPACING (4px base)
	// =========================================================================
	spacing: {
		unit: '4px',
		// 0, 4, 8, 12, 16, 24, 32, 48, 64, 96, 128, 192, 256
		scale: [0, 1, 2, 3, 4, 6, 8, 12, 16, 24, 32, 48, 64]
	},

	// =========================================================================
	// BORDERS
	// =========================================================================
	borders: {
		radiusNone: '0',
		radiusSmall: '4px',
		radiusMedium: '8px',
		radiusLarge: '12px',
		radiusFull: '9999px',
		widthDefault: '1px',
		widthThick: '2px'
	},

	// =========================================================================
	// SHADOWS (Subtle for dark mode)
	// =========================================================================
	shadows: {
		none: 'none',
		small: '0 1px 2px rgba(0, 0, 0, 0.3)',
		medium: '0 4px 6px rgba(0, 0, 0, 0.4)',
		large: '0 10px 15px rgba(0, 0, 0, 0.5)',
		xl: '0 20px 25px rgba(0, 0, 0, 0.6)'
	},

	// =========================================================================
	// CONTEXT RULES
	// =========================================================================
	contextRules: [
		{
			context: 'navigation',
			allowedElements: ['nav', 'a', 'ul', 'li', 'button', 'span'],
			semanticRules: [
				{ rule: 'Primary nav uses <nav> as direct child of body or layout' },
				{ rule: 'Use aria-label for multiple nav elements' },
				{ rule: 'Links must have href, no nested interactive elements' }
			],
			typographyRole: 'label',
			fontStack: 'sans'
		},
		{
			context: 'content',
			allowedElements: ['section', 'article', 'header', 'footer', 'aside', 'main', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'p', 'ul', 'ol', 'li'],
			semanticRules: [
				{ rule: 'Sections must have heading (h2-h6)' },
				{ rule: 'Never skip heading levels' },
				{ rule: 'One <h1> per page' }
			],
			typographyRole: 'body',
			fontStack: 'sans'
		},
		{
			context: 'form',
			allowedElements: ['form', 'label', 'input', 'textarea', 'select', 'button', 'fieldset', 'legend'],
			semanticRules: [
				{ rule: 'Labels must have for="" matching input id' },
				{ rule: 'Inputs must have type, name, id' },
				{ rule: 'Buttons need explicit type="submit" or type="button"' }
			],
			typographyRole: 'body',
			fontStack: 'sans'
		}
	],

	// =========================================================================
	// VARIANTS
	// =========================================================================
	variants: [
		{
			name: 'light',
			description: 'Light mode for bright environments',
			colors: {
				background: '#ffffff',
				surface: '#f9fafb',
				surfaceElevated: '#ffffff',
				text: '#111827',
				textMuted: '#6b7280',
				textDisabled: '#9ca3af',
				border: '#e5e7eb',
				borderFocus: '#6366f1'
			},
			shadows: {
				small: '0 1px 2px rgba(0, 0, 0, 0.05)',
				medium: '0 4px 6px rgba(0, 0, 0, 0.1)',
				large: '0 10px 15px rgba(0, 0, 0, 0.1)',
				xl: '0 20px 25px rgba(0, 0, 0, 0.15)'
			}
		},
		{
			name: 'high-contrast',
			description: 'High contrast mode for accessibility',
			colors: {
				text: '#ffffff',
				textMuted: '#e5e7eb',
				background: '#000000',
				surface: '#0a0a0a',
				border: '#ffffff',
				primary: '#818cf8'
			}
		}
	]
};

// Register on import
brandRegistry.register(hubBrand);

export default hubBrand;
