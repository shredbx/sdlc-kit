/**
 * IPresenter Interface - Auto-Registration for Components and Packages
 *
 * The Presenter pattern enables components and packages to be automatically
 * listed in hub's /components and /packages pages with minimal configuration.
 *
 * Key Innovation:
 * - MINIMAL BARRIER: 4 required fields to get listed
 * - PROGRESSIVE ENHANCEMENT: Add variants, presets, editing when needed
 * - AUTO-DISCOVERY: Glob pattern finds all *.presenter.ts files
 * - RICH PRESENTATION: Single file = docs + preview + playground
 *
 * @example
 * // Button.presenter.ts - Minimal conformance
 * export default {
 *   name: 'Button',
 *   slug: 'button',
 *   category: 'primitives',
 *   presenterType: 'component',
 *   component: Button
 * } satisfies IPresenter;
 *
 * @example
 * // Button.presenter.ts - With enrichment
 * export default {
 *   name: 'Button',
 *   slug: 'button',
 *   category: 'primitives',
 *   presenterType: 'component',
 *   component: Button,
 *   variants: [
 *     { name: 'Primary', props: { variant: 'primary' } },
 *     { name: 'Ghost', props: { variant: 'ghost' } }
 *   ],
 *   props: [
 *     { name: 'variant', type: 'select', options: ['primary', 'secondary', 'ghost'] }
 *   ]
 * } satisfies IPresenter;
 */

import type { Component, SvelteComponent } from 'svelte';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyComponent = Component<any, any, any> | typeof SvelteComponent<any>;

// =============================================================================
// VARIANT DEFINITION
// =============================================================================

/**
 * A variant showcases a specific state/configuration of a component.
 * Each variant is rendered separately in the presenter view.
 */
export interface IVariant {
	/** Unique identifier for this variant */
	id?: string;
	/** Display name for this variant */
	name: string;
	/** Optional description of what this variant demonstrates */
	description?: string;
	/** Props to pass to component for this variant */
	props: Record<string, unknown>;
}

// =============================================================================
// PRESET DEFINITION
// =============================================================================

/**
 * A preset is a saved prop combination for quick reuse.
 * Presets can be copied to clipboard for use in code.
 */
export interface IPreset {
	/** Display name for this preset */
	name: string;
	/** Optional description of when to use this preset */
	description?: string;
	/** Complete prop configuration for this preset */
	props: Record<string, unknown>;
	/** Optional pre-generated code snippet */
	code?: string;
}

// =============================================================================
// PROP DEFINITION
// =============================================================================

/** Input types for editing props in the presenter UI */
export type PropType = 'string' | 'number' | 'boolean' | 'select' | 'color' | 'range' | 'json' | 'array' | 'object' | 'function';

/**
 * Defines how a prop appears in the real-time editing panel.
 */
export interface IPropDef {
	/** Prop name as used in component (must match exactly) */
	name: string;
	/** Display label in editing UI (defaults to titleCase of name) */
	label?: string;
	/** Input type for editing this prop */
	type: PropType;
	/** Help text explaining what prop does */
	description?: string;
	/** Default value for this prop */
	default?: unknown;
	/** Whether this prop is required */
	required?: boolean;
	/** Available options for select type */
	options?: string[];
	/** Minimum value for number/range */
	min?: number;
	/** Maximum value for number/range */
	max?: number;
	/** Step increment for number/range */
	step?: number;
}

// =============================================================================
// PREVIEW CONFIGURATION
// =============================================================================

/** Background options for preview rendering */
export type PreviewBackground = 'light' | 'dark' | 'transparent' | 'checkerboard' | (string & {});

/**
 * Configuration for how the component preview renders.
 */
export interface IPreviewConfig {
	/** Preview container width in pixels */
	width?: number;
	/** Preview container height in pixels */
	height?: number;
	/** Preview background */
	background?: PreviewBackground;
	/** Default props for preview rendering */
	defaultProps?: Record<string, unknown>;
}

// =============================================================================
// SOURCE CONFIGURATION
// =============================================================================

/**
 * Links to the actual source code.
 */
export interface ISourceConfig {
	/** File path relative to project root */
	path: string;
	/** Repository name if different from current */
	repo?: string;
	/** Line number in source file */
	line?: number;
}

// =============================================================================
// FIXTURE CONFIGURATION
// =============================================================================

/**
 * Configuration for linking presenter to fixture data.
 * Enables loading demo data from colocated *.fixture.yml files.
 */
export interface IFixtureConfig {
	/** Default fixture key (entity:name format) */
	default?: string;
	/** Map of variant name to fixture key for different scenarios */
	variants?: Record<string, string>;
	/** Map of preset name to fixture preset for component-specific props */
	presets?: Record<string, string>;
}

// =============================================================================
// PRESENTER TYPE
// =============================================================================

/** Distinguishes between component and package presenters */
export type PresenterType = 'component' | 'package';

/** Component categories (atomic design) */
export type PresenterCategory = 'primitives' | 'blocks' | 'sections' | 'layouts' | 'templates' | 'utilities' | 'features';

/** Development status of the item */
export type PresenterStatus = 'draft' | 'beta' | 'stable' | 'deprecated';

// =============================================================================
// IPRESENTER INTERFACE
// =============================================================================

/**
 * The main IPresenter interface.
 *
 * Required fields (4):
 * - name: Display name
 * - slug: URL-safe identifier
 * - category: Grouping category
 * - presenterType: 'component' or 'package'
 *
 * Optional enrichment fields enable progressive enhancement.
 */
export interface IPresenter {
	// =========================================================================
	// REQUIRED FIELDS (Minimal Interface)
	// =========================================================================

	/** Display name for the component/package */
	name: string;

	/** URL-safe identifier (lowercase kebab-case) */
	slug: string;

	/** Grouping category (for components: primitives, blocks, sections; for packages: ui, utils, adapters) */
	category: string;

	/** Whether this presents a component or package */
	presenterType: PresenterType;

	// =========================================================================
	// OPTIONAL ENRICHMENT FIELDS
	// =========================================================================

	/** Brief description of what this item does */
	description?: string;

	/** The actual Svelte component (for component presenters) */
	component?: AnyComponent;

	/** Link to source code */
	source?: ISourceConfig;

	/** Extended documentation in markdown */
	documentation?: string;

	/** Different states/configurations to showcase */
	variants?: IVariant[];

	/** Saved prop combinations for common use cases */
	presets?: IPreset[];

	/** Props available for real-time editing */
	props?: IPropDef[];

	/** Preview configuration */
	preview?: IPreviewConfig;

	/** Fixture configuration - links to colocated *.fixture.yml */
	fixture?: IFixtureConfig;

	/** Searchable tags */
	tags?: string[];

	/** Development status */
	status?: PresenterStatus;

	/** Semantic version */
	version?: string;
}

// =============================================================================
// PRESENTER REGISTRY
// =============================================================================

/**
 * Registry for managing presenter discovery and access.
 */
export interface IPresenterRegistry {
	/** All registered presenters */
	presenters: Map<string, IPresenter>;

	/** Register a presenter */
	register(presenter: IPresenter): void;

	/** Get presenter by slug */
	get(slug: string): IPresenter | undefined;

	/** Get all presenters of a type */
	getByType(type: PresenterType): IPresenter[];

	/** Get all presenters in a category */
	getByCategory(category: string): IPresenter[];

	/** Get all categories */
	getCategories(): string[];

	/** Search presenters by query */
	search(query: string): IPresenter[];
}

// =============================================================================
// PRESENTER REGISTRY IMPLEMENTATION
// =============================================================================

/**
 * Simple in-memory presenter registry implementation.
 */
export class PresenterRegistry implements IPresenterRegistry {
	presenters = new Map<string, IPresenter>();

	register(presenter: IPresenter): void {
		const key = `${presenter.presenterType}:${presenter.slug}`;
		this.presenters.set(key, presenter);
	}

	get(slug: string): IPresenter | undefined {
		// Try component first, then package
		return this.presenters.get(`component:${slug}`) || this.presenters.get(`package:${slug}`);
	}

	getByType(type: PresenterType): IPresenter[] {
		return Array.from(this.presenters.values()).filter((p) => p.presenterType === type);
	}

	getByCategory(category: string): IPresenter[] {
		return Array.from(this.presenters.values()).filter((p) => p.category === category);
	}

	getCategories(): string[] {
		const categories = new Set<string>();
		this.presenters.forEach((p) => categories.add(p.category));
		return Array.from(categories).sort();
	}

	search(query: string): IPresenter[] {
		const q = query.toLowerCase();
		return Array.from(this.presenters.values()).filter(
			(p) =>
				p.name.toLowerCase().includes(q) ||
				p.slug.includes(q) ||
				p.category.toLowerCase().includes(q) ||
				p.description?.toLowerCase().includes(q) ||
				p.tags?.some((t) => t.toLowerCase().includes(q))
		);
	}
}

// =============================================================================
// GLOBAL REGISTRY INSTANCE
// =============================================================================

/** Global presenter registry */
export const presenterRegistry = new PresenterRegistry();

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

/**
 * Helper to create a minimal component presenter.
 */
export function createComponentPresenter(
	name: string,
	slug: string,
	category: string,
	component: AnyComponent,
	enrichment?: Partial<IPresenter>
): IPresenter {
	return {
		name,
		slug,
		category,
		presenterType: 'component',
		component,
		...enrichment
	};
}

/**
 * Helper to create a minimal package presenter.
 */
export function createPackagePresenter(
	name: string,
	slug: string,
	category: string,
	enrichment?: Partial<IPresenter>
): IPresenter {
	return {
		name,
		slug,
		category,
		presenterType: 'package',
		...enrichment
	};
}
