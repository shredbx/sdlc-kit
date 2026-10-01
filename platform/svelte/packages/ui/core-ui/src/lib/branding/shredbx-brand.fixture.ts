/**
 * Shredbx Brand Fixture — "Dark Rose"
 * ====================================
 *
 * Personal developer portfolio & knowledge base branding.
 * Near-black with warm rose primary, muted teal secondary.
 *
 * Context: Software Development, Knowledge Base, Portfolio, Services.
 * Design: Sophisticated dark mode default, warm light variant.
 *
 * Palette rationale:
 * - Near-black (#0C0A0F) with subtle purple warmth — not flat zinc
 * - Warm off-white (#EDE8E2) — cream undertone, not sterile white
 * - Dark rose (#D4365C) — between crimson and pink, bold but refined
 * - Muted teal (#5B8F96) — cool complement, doesn't compete
 *
 * Standards:
 * - Material Design 3 Typography Scale
 * - W3C Design Tokens semantic naming
 * - WCAG 2.1 AA contrast compliance
 *
 * Usage:
 * 1. Import and register: brandRegistry.register(shredbxBrand)
 * 2. Generate CSS: generateBrandCSS(shredbxBrand)
 * 3. Apply via attribute: <div data-brand="shredbx-brand">
 * 4. Switch variant: <div data-brand="shredbx-brand" data-theme="light">
 */

import type { IBrandConfig } from './IBrandConfig';
import { brandRegistry } from './IBrandConfig';

/**
 * Shredbx Brand Configuration
 */
export const shredbxBrand: IBrandConfig = {
	// =========================================================================
	// IDENTITY (IPresentable)
	// =========================================================================
	id: 'shredbx-brand',
	name: 'Shredbx',
	displayName: 'Shredbx',
	category: 'brand',
	entityType: 'brand',
	description: 'Developer portfolio & knowledge base — dark rose accent theme',
	tags: ['shredbx', 'dark', 'portfolio', 'developer', 'rose'],
	status: 'stable',
	version: '1.0.0',
	project: 'shredbx',

	// =========================================================================
	// TYPOGRAPHY
	// =========================================================================
	typography: {
		fontFamilySans: '"Inter", system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
		fontFamilySerif: '"Lora", Georgia, Cambria, "Times New Roman", Times, serif',
		fontFamilyMono: '"JetBrains Mono", "SF Mono", ui-monospace, SFMono-Regular, Menlo, Consolas, monospace',
		fontFamilyEmail: 'Arial, Helvetica, sans-serif',
		baseSize: '16px',
		scaleRatio: '1.25', // Major Third — more dramatic for portfolio

		scales: {
			// Display (Hero text)
			display_large: {
				fontSize: '56px',
				lineHeight: '64px',
				fontWeight: 700,
				letterSpacing: '-1px'
			},
			display_medium: {
				fontSize: '44px',
				lineHeight: '52px',
				fontWeight: 700,
				letterSpacing: '-0.5px'
			},
			display_small: {
				fontSize: '36px',
				lineHeight: '44px',
				fontWeight: 600,
				letterSpacing: '-0.25px'
			},

			// Headline (Section headers)
			headline_large: {
				fontSize: '30px',
				lineHeight: '38px',
				fontWeight: 600
			},
			headline_medium: {
				fontSize: '24px',
				lineHeight: '32px',
				fontWeight: 600
			},
			headline_small: {
				fontSize: '20px',
				lineHeight: '28px',
				fontWeight: 600
			},

			// Title (Card titles)
			title_large: {
				fontSize: '18px',
				lineHeight: '26px',
				fontWeight: 600
			},
			title_medium: {
				fontSize: '16px',
				lineHeight: '22px',
				fontWeight: 600
			},
			title_small: {
				fontSize: '14px',
				lineHeight: '20px',
				fontWeight: 600
			},

			// Body (Paragraphs — slightly relaxed line-height for readability)
			body_large: {
				fontSize: '18px',
				lineHeight: '28px',
				fontWeight: 400
			},
			body_medium: {
				fontSize: '16px',
				lineHeight: '26px',
				fontWeight: 400
			},
			body_small: {
				fontSize: '14px',
				lineHeight: '22px',
				fontWeight: 400
			},

			// Label (Buttons, form labels)
			label_large: {
				fontSize: '14px',
				lineHeight: '20px',
				fontWeight: 600,
				letterSpacing: '0.25px'
			},
			label_medium: {
				fontSize: '12px',
				lineHeight: '16px',
				fontWeight: 600,
				letterSpacing: '0.25px'
			},
			label_small: {
				fontSize: '11px',
				lineHeight: '14px',
				fontWeight: 600,
				letterSpacing: '0.5px'
			}
		}
	},

	// =========================================================================
	// COLORS — Dark Rose (Dark default)
	// =========================================================================
	colors: {
		// Primary (Dark Rose — between crimson and pink)
		primary: '#D4365C',
		primaryHover: '#C0304F',
		primaryActive: '#A82847',

		// Secondary (Muted Teal — cool complement)
		secondary: '#5B8F96',

		// Backgrounds (Near-black with subtle purple warmth)
		background: '#0C0A0F',
		surface: '#15131A',
		surfaceElevated: '#1E1C24',

		// Text (Warm off-white — cream undertone)
		text: '#EDE8E2',
		textMuted: '#908A84',
		textDisabled: '#5C5856',

		// Borders
		border: '#28262E',
		borderFocus: '#D4365C',

		// Status
		statusSuccess: '#34D399',
		statusWarning: '#FBBF24',
		statusError: '#F87171',
		statusInfo: '#60A5FA'
	},

	// =========================================================================
	// SPACING (4px base)
	// =========================================================================
	spacing: {
		unit: '4px',
		scale: [0, 1, 2, 3, 4, 6, 8, 10, 12, 16, 20, 24, 32, 40, 48, 64]
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
	// SHADOWS (Deep for dark mode — uses primary tint)
	// =========================================================================
	shadows: {
		none: 'none',
		small: '0 1px 3px rgba(0, 0, 0, 0.5)',
		medium: '0 4px 12px rgba(0, 0, 0, 0.6)',
		large: '0 12px 28px rgba(0, 0, 0, 0.7)',
		xl: '0 24px 48px rgba(0, 0, 0, 0.8)'
	},

	// =========================================================================
	// CONTEXT RULES
	// =========================================================================
	contextRules: [
		{
			context: 'navigation',
			allowedElements: ['nav', 'a', 'ul', 'li', 'button', 'span', 'div'],
			semanticRules: [
				{ rule: 'Primary nav uses <nav> as direct child of layout' },
				{ rule: 'Use aria-label for multiple nav elements' },
				{ rule: 'Links must have href, buttons for actions' }
			],
			typographyRole: 'label',
			fontStack: 'sans'
		},
		{
			context: 'hero',
			allowedElements: ['section', 'h1', 'p', 'a', 'button', 'div', 'span'],
			semanticRules: [
				{ rule: 'Hero section uses display typography' },
				{ rule: 'CTA buttons use primary color with light text' },
				{ rule: 'Limit to one h1 per page' }
			],
			typographyRole: 'display',
			fontStack: 'sans'
		},
		{
			context: 'content',
			allowedElements: ['article', 'section', 'h2', 'h3', 'h4', 'p', 'ul', 'ol', 'li', 'pre', 'code', 'blockquote'],
			semanticRules: [
				{ rule: 'Articles use serif for body, mono for code blocks' },
				{ rule: 'Code blocks get surfaceElevated background' },
				{ rule: 'Blockquotes use border-left with primary color' }
			],
			typographyRole: 'body',
			fontStack: 'sans'
		},
		{
			context: 'footer',
			allowedElements: ['footer', 'nav', 'a', 'p', 'span', 'div'],
			semanticRules: [
				{ rule: 'Footer uses muted text colors' },
				{ rule: 'Social links use icon buttons' },
				{ rule: 'Copyright uses label typography' }
			],
			typographyRole: 'label',
			fontStack: 'sans'
		}
	],

	// =========================================================================
	// VARIANTS
	// =========================================================================
	variants: [
		{
			name: 'light',
			description: 'Light mode — warm cream background, dark text',
			colors: {
				background: '#F5F0EA',
				surface: '#FDFBF8',
				surfaceElevated: '#FFFFFF',
				text: '#0C0A0F',
				textMuted: '#5A5550',
				textDisabled: '#9E9892',
				border: '#E0DBD4',
				borderFocus: '#D4365C',
				secondary: '#4A7A80'
			},
			shadows: {
				small: '0 1px 3px rgba(12, 10, 15, 0.06)',
				medium: '0 4px 12px rgba(12, 10, 15, 0.08)',
				large: '0 12px 28px rgba(12, 10, 15, 0.1)',
				xl: '0 24px 48px rgba(12, 10, 15, 0.12)'
			}
		}
	]
};

// Register on import
brandRegistry.register(shredbxBrand);

export default shredbxBrand;
