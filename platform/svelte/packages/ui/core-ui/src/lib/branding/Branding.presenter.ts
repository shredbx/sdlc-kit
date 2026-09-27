/**
 * Branding Page Presenter
 * =======================
 *
 * Presenter for the /branding page.
 * Provides metadata, variants, and presets for branding documentation.
 */

import type { IPresenter } from '../presenter/IPresenter';

/**
 * Branding page presenter configuration.
 */
export const brandingPresenter: IPresenter = {
	// =========================================================================
	// REQUIRED FIELDS
	// =========================================================================
	name: 'Branding',
	slug: 'branding',
	category: 'pages',
	presenterType: 'package',

	// =========================================================================
	// ENRICHMENT
	// =========================================================================
	description: 'Design tokens and brand configuration for the workspace. Configure typography, colors, spacing, borders, and shadows.',

	documentation: `
# Branding System

The branding system provides a centralized way to manage design tokens and visual consistency across the workspace.

## Key Concepts

### IPresentable
Base interface for all presentable entities (components, brands, fixtures).

### IBrandConfig
Brand configuration extending IPresentable with typography, colors, spacing, borders, and shadows.

### Token Cascade
Tokens cascade from Global → Layout → Section → Component, allowing contextual overrides.

## Standards

- **W3C Design Tokens Format (DTCG 2025.10)** - Platform-agnostic token format
- **Material Design 3 Typography** - 15+ tokens across 5 semantic categories
- **Open Props** - CSS custom properties naming conventions

## Integration

1. Import brand fixture
2. Register with brandRegistry
3. Generate CSS with generateBrandCSS()
4. Apply via data-brand attribute

## Testing (TDD)

- Verify required tokens exist
- Test WCAG contrast compliance
- Validate CSS generation
- Test variant switching
`,

	tags: [
		'branding',
		'design-tokens',
		'typography',
		'colors',
		'theming',
		'accessibility',
		'wcag'
	],

	status: 'stable',
	version: '1.0.0',

	source: {
		path: 'projects/sbx/apps/sbx/svelte/src/routes/branding/+page.svelte'
	}
};

export default brandingPresenter;
