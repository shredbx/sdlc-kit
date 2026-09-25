/**
 * Component Representation System
 *
 * Handles different representations for components at various atomic levels.
 * Primitives can be shown inline, while sections need more space.
 *
 * Representation Contexts:
 * - sidebar: Compact list view (icon + name)
 * - card: Medium view with mini preview
 * - detail: Full view with live preview and props
 * - thumbnail: Visual-only thumbnail for galleries
 *
 * Level-Specific Defaults:
 * - primitives: Can render inline, small previews (50x30px)
 * - blocks: Card view with medium previews (150x80px)
 * - sections: Require significant space (300x150px)
 * - layouts: Full diagram or screenshot (500x300px)
 *
 * @example
 * const rep = getRepresentation('button', 'sidebar');
 * // { mode: 'inline', preview: { width: 50, height: 30 }, showProps: false }
 */

import type { IPresenter, PresenterCategory } from './IPresenter';

// =============================================================================
// REPRESENTATION TYPES
// =============================================================================

/** Context where the component is being displayed */
export type RepresentationContext = 'sidebar' | 'card' | 'detail' | 'thumbnail' | 'tree';

/** How the component should be rendered */
export type RepresentationMode = 'inline' | 'icon' | 'preview' | 'full' | 'placeholder';

/** Preview size configuration */
export interface PreviewSize {
	width: number;
	height: number;
	/** Whether preview is interactive */
	interactive?: boolean;
}

/** Full representation configuration */
export interface ComponentRepresentation {
	/** Rendering mode */
	mode: RepresentationMode;
	/** Preview dimensions */
	preview: PreviewSize;
	/** Show property editors */
	showProps: boolean;
	/** Show variants selector */
	showVariants: boolean;
	/** Show presets */
	showPresets: boolean;
	/** Show documentation */
	showDocs: boolean;
	/** Max items before collapse in list views */
	maxItems?: number;
	/** Loading behavior */
	loading: LoadingBehavior;
}

/** Loading behavior for SSR-first pattern */
export type LoadingBehavior =
	| 'skeleton'      // Show placeholder skeleton
	| 'spinner'       // Show loading spinner
	| 'silent'        // Load without indicator
	| 'blocking'      // Block UI until loaded
	| 'progressive';  // Load in stages

/** Loading state configuration */
export interface LoadingConfig {
	behavior: LoadingBehavior;
	/** Minimum display time for loading state (prevents flash) */
	minDisplayMs?: number;
	/** Timeout before showing error state */
	timeoutMs?: number;
	/** Whether to preserve previous content during reload */
	preserveContent?: boolean;
}

// =============================================================================
// LEVEL CONFIGURATIONS
// =============================================================================

/** Default preview sizes per atomic level */
const LEVEL_PREVIEW_SIZES: Record<PresenterCategory, PreviewSize> = {
	primitives: { width: 60, height: 40, interactive: false },
	blocks: { width: 150, height: 80, interactive: false },
	sections: { width: 300, height: 150, interactive: true },
	layouts: { width: 500, height: 300, interactive: true },
	templates: { width: 600, height: 400, interactive: true },
	utilities: { width: 100, height: 60, interactive: false },
	features: { width: 250, height: 120, interactive: true }
};

/** Default representations per context and level */
const LEVEL_CONTEXT_MODES: Record<PresenterCategory, Record<RepresentationContext, RepresentationMode>> = {
	primitives: {
		sidebar: 'icon',
		card: 'inline',
		detail: 'preview',
		thumbnail: 'inline',
		tree: 'icon'
	},
	blocks: {
		sidebar: 'icon',
		card: 'preview',
		detail: 'preview',
		thumbnail: 'preview',
		tree: 'icon'
	},
	sections: {
		sidebar: 'icon',
		card: 'preview',
		detail: 'full',
		thumbnail: 'preview',
		tree: 'icon'
	},
	layouts: {
		sidebar: 'icon',
		card: 'placeholder',
		detail: 'full',
		thumbnail: 'placeholder',
		tree: 'icon'
	},
	templates: {
		sidebar: 'icon',
		card: 'placeholder',
		detail: 'full',
		thumbnail: 'placeholder',
		tree: 'icon'
	},
	utilities: {
		sidebar: 'icon',
		card: 'icon',
		detail: 'preview',
		thumbnail: 'icon',
		tree: 'icon'
	},
	features: {
		sidebar: 'icon',
		card: 'preview',
		detail: 'full',
		thumbnail: 'preview',
		tree: 'icon'
	}
};

/** Context-specific display options */
const CONTEXT_OPTIONS: Record<RepresentationContext, Partial<ComponentRepresentation>> = {
	sidebar: {
		showProps: false,
		showVariants: false,
		showPresets: false,
		showDocs: false,
		maxItems: 20,
		loading: 'silent'
	},
	card: {
		showProps: false,
		showVariants: false,
		showPresets: true,
		showDocs: false,
		maxItems: 12,
		loading: 'skeleton'
	},
	detail: {
		showProps: true,
		showVariants: true,
		showPresets: true,
		showDocs: true,
		loading: 'progressive'
	},
	thumbnail: {
		showProps: false,
		showVariants: false,
		showPresets: false,
		showDocs: false,
		maxItems: 50,
		loading: 'skeleton'
	},
	tree: {
		showProps: false,
		showVariants: false,
		showPresets: false,
		showDocs: false,
		maxItems: 100,
		loading: 'silent'
	}
};

// =============================================================================
// REPRESENTATION FUNCTIONS
// =============================================================================

/**
 * Get representation configuration for a presenter in a specific context.
 */
export function getRepresentation(
	presenter: IPresenter,
	context: RepresentationContext
): ComponentRepresentation {
	const category = presenter.category as PresenterCategory;
	const mode = LEVEL_CONTEXT_MODES[category]?.[context] ?? 'icon';
	const previewSize = LEVEL_PREVIEW_SIZES[category] ?? LEVEL_PREVIEW_SIZES.primitives;
	const contextOptions = CONTEXT_OPTIONS[context];

	// Adjust preview size based on context
	let adjustedPreview = { ...previewSize };
	if (context === 'sidebar' || context === 'tree') {
		adjustedPreview = { width: 24, height: 24, interactive: false };
	} else if (context === 'thumbnail') {
		adjustedPreview = {
			width: Math.min(previewSize.width, 200),
			height: Math.min(previewSize.height, 100),
			interactive: false
		};
	}

	return {
		mode,
		preview: adjustedPreview,
		showProps: contextOptions.showProps ?? false,
		showVariants: contextOptions.showVariants ?? false,
		showPresets: contextOptions.showPresets ?? false,
		showDocs: contextOptions.showDocs ?? false,
		maxItems: contextOptions.maxItems,
		loading: contextOptions.loading ?? 'skeleton'
	};
}

/**
 * Get loading configuration for a representation.
 */
export function getLoadingConfig(
	representation: ComponentRepresentation,
	override?: Partial<LoadingConfig>
): LoadingConfig {
	const defaults: Record<LoadingBehavior, LoadingConfig> = {
		skeleton: {
			behavior: 'skeleton',
			minDisplayMs: 200,
			timeoutMs: 10000,
			preserveContent: false
		},
		spinner: {
			behavior: 'spinner',
			minDisplayMs: 0,
			timeoutMs: 10000,
			preserveContent: false
		},
		silent: {
			behavior: 'silent',
			minDisplayMs: 0,
			timeoutMs: 30000,
			preserveContent: true
		},
		blocking: {
			behavior: 'blocking',
			minDisplayMs: 0,
			timeoutMs: 5000,
			preserveContent: false
		},
		progressive: {
			behavior: 'progressive',
			minDisplayMs: 100,
			timeoutMs: 15000,
			preserveContent: true
		}
	};

	return {
		...defaults[representation.loading],
		...override
	};
}

/**
 * Check if a component can be rendered inline (without dedicated container).
 */
export function canRenderInline(presenter: IPresenter): boolean {
	const category = presenter.category as PresenterCategory;
	return category === 'primitives' || category === 'utilities';
}

/**
 * Check if a component needs significant space for meaningful preview.
 */
export function needsFullPreview(presenter: IPresenter): boolean {
	const category = presenter.category as PresenterCategory;
	return category === 'sections' || category === 'layouts' || category === 'features';
}

/**
 * Get appropriate icon for a category.
 */
export function getCategoryIcon(category: PresenterCategory): string {
	const icons: Record<PresenterCategory, string> = {
		primitives: 'box',
		blocks: 'grid-2x2',
		sections: 'layout',
		layouts: 'layout-template',
		templates: 'layers',
		utilities: 'tool',
		features: 'puzzle'
	};
	return icons[category] ?? 'component';
}

/**
 * Get human-readable label for a category.
 */
export function getCategoryLabel(category: PresenterCategory): string {
	const labels: Record<PresenterCategory, string> = {
		primitives: 'Primitives',
		blocks: 'Blocks',
		sections: 'Sections',
		layouts: 'Layouts',
		templates: 'Templates',
		utilities: 'Utilities',
		features: 'Features'
	};
	return labels[category] ?? category;
}

/**
 * Get category description for tooltips/help.
 */
export function getCategoryDescription(category: PresenterCategory): string {
	const descriptions: Record<PresenterCategory, string> = {
		primitives: 'Atomic UI elements like buttons, inputs, badges',
		blocks: 'Compositions of primitives with semantic meaning',
		sections: 'Functional UI regions like panels and sidebars',
		layouts: 'Page-level arrangements for single-entity views',
		templates: 'Multi-layout compositions for complex workflows',
		utilities: 'Cross-cutting concerns like transitions and loaders',
		features: 'Domain-specific components organized by feature'
	};
	return descriptions[category] ?? '';
}

// =============================================================================
// TREE ITEM CONVERSION
// =============================================================================

/**
 * Convert presenters to tree items for sidebar navigation.
 */
export function presentersToTreeItems(presenters: IPresenter[]): TreeItem[] {
	// Group by category
	const byCategory = new Map<string, IPresenter[]>();
	for (const p of presenters) {
		const list = byCategory.get(p.category) ?? [];
		list.push(p);
		byCategory.set(p.category, list);
	}

	// Convert to tree structure
	const items: TreeItem[] = [];
	for (const [category, members] of byCategory) {
		items.push({
			id: category,
			label: getCategoryLabel(category as PresenterCategory),
			icon: getCategoryIcon(category as PresenterCategory),
			expanded: true,
			children: members.map(p => ({
				id: p.slug,
				label: p.name,
				icon: p.status === 'deprecated' ? 'alert-circle' : 'component',
				badge: p.status === 'beta' ? 'beta' : undefined,
				data: { presenter: p }
			}))
		});
	}

	return items;
}

/** Tree item structure for sidebar */
export interface TreeItem {
	id: string;
	label: string;
	icon?: string;
	badge?: string | number;
	expanded?: boolean;
	selected?: boolean;
	children?: TreeItem[];
	data?: Record<string, unknown>;
}

// =============================================================================
// CARD VIEW HELPERS
// =============================================================================

/**
 * Get card view configuration for a presenter.
 */
export function getCardConfig(presenter: IPresenter): CardConfig {
	const rep = getRepresentation(presenter, 'card');
	return {
		preview: rep.preview,
		showPresets: rep.showPresets && (presenter.presets?.length ?? 0) > 0,
		presetsCount: presenter.presets?.length ?? 0,
		variantsCount: presenter.variants?.length ?? 0,
		status: presenter.status ?? 'stable',
		tags: presenter.tags ?? []
	};
}

/** Card view configuration */
export interface CardConfig {
	preview: PreviewSize;
	showPresets: boolean;
	presetsCount: number;
	variantsCount: number;
	status: string;
	tags: string[];
}

// =============================================================================
// EXPORTS
// =============================================================================

export {
	LEVEL_PREVIEW_SIZES,
	LEVEL_CONTEXT_MODES,
	CONTEXT_OPTIONS
};
