/**
 * IPresentable Interface - Common Contract for Presentable Entities
 *
 * IPresentable defines the minimum contract for anything that can be presented
 * in the hub UI: components, fixtures, brands, packages, etc.
 *
 * Architecture:
 *   IPresentable (common contract)
 *       ↓ extends
 *   IView (runtime behavior)     IPresenter (documentation)     IBrandConfig (theming)
 *       ↓                              ↓                              ↓
 *   Component Runtime            /components page              /branding page
 *
 * Key Innovation:
 * - SINGLE CONTRACT: All presentable entities share same identity fields
 * - PROGRESSIVE ENHANCEMENT: Minimal required, rich optional
 * - TYPE-SAFE REGISTRY: Generic registry works with any IPresentable
 *
 * @example
 * // Minimal presentable
 * const item: IPresentable = {
 *   id: 'button-primary',
 *   name: 'Primary Button',
 *   category: 'primitives'
 * };
 *
 * @example
 * // Extended for branding
 * interface IBrand extends IPresentable {
 *   colors: ColorPalette;
 *   typography: TypographyConfig;
 * }
 */

// =============================================================================
// PRESENTABLE CATEGORY
// =============================================================================

/**
 * Common categories for presentable items.
 * Specific interfaces may extend with domain-specific categories.
 */
export type PresentableCategory =
	// Component categories (Modified Atomic Design)
	| 'primitives'
	| 'blocks'
	| 'sections'
	| 'layouts'
	// Package categories
	| 'ui'
	| 'utils'
	| 'adapters'
	// Branding categories
	| 'brand'
	| 'theme'
	| 'tokens'
	// Fixture categories
	| 'fixture'
	| 'test-data'
	// Generic
	| string;

// =============================================================================
// PRESENTABLE STATUS
// =============================================================================

/**
 * Development/release status of presentable item.
 */
export type PresentableStatus = 'draft' | 'beta' | 'stable' | 'deprecated';

// =============================================================================
// IPRESENTABLE INTERFACE
// =============================================================================

/**
 * IPresentable - Core contract for anything presentable in hub.
 *
 * Required fields (3):
 * - id: Unique identifier
 * - name: Display name
 * - category: Grouping category
 *
 * All other fields are optional for progressive enhancement.
 */
export interface IPresentable {
	// =========================================================================
	// IDENTITY (Required)
	// =========================================================================

	/** Unique identifier (slug format: lowercase-kebab-case) */
	id: string;

	/** Display name for UI */
	name: string;

	/** Grouping category */
	category: PresentableCategory;

	// =========================================================================
	// METADATA (Optional)
	// =========================================================================

	/** Brief description */
	description?: string;

	/** Searchable tags */
	tags?: string[];

	/** Development status */
	status?: PresentableStatus;

	/** Semantic version */
	version?: string;

	/** Entity type for discrimination (e.g., 'component', 'brand', 'fixture') */
	entityType?: string;
}

// =============================================================================
// PRESENTABLE WITH PREVIEW
// =============================================================================

/**
 * Preview configuration for presentable items.
 */
export interface IPreviewConfig {
	/** Preview width in pixels */
	width?: number;
	/** Preview height in pixels */
	height?: number;
	/** Background style */
	background?: 'light' | 'dark' | 'transparent' | 'checkerboard';
	/** Whether preview is interactive */
	interactive?: boolean;
}

/**
 * Presentable with preview capability.
 */
export interface IPresentableWithPreview extends IPresentable {
	/** Preview configuration */
	preview?: IPreviewConfig;
	/** Preview thumbnail URL */
	thumbnail?: string;
}

// =============================================================================
// PRESENTABLE WITH SOURCE
// =============================================================================

/**
 * Source code reference.
 */
export interface ISourceRef {
	/** File path relative to project root */
	path: string;
	/** Repository name if different */
	repo?: string;
	/** Line number */
	line?: number;
}

/**
 * Presentable with source code link.
 */
export interface IPresentableWithSource extends IPresentable {
	/** Source code reference */
	source?: ISourceRef;
}

// =============================================================================
// PRESENTABLE REGISTRY
// =============================================================================

/**
 * Generic registry for presentable items.
 *
 * @example
 * const registry = new PresentableRegistry<IBrand>();
 * registry.register(hubBrand);
 * registry.get('hub-brand');
 */
export interface IPresentableRegistry<T extends IPresentable> {
	/** All registered items */
	items: Map<string, T>;

	/** Register an item */
	register(item: T): void;

	/** Get item by id */
	get(id: string): T | undefined;

	/** Get all items in category */
	getByCategory(category: PresentableCategory): T[];

	/** Get all categories */
	getCategories(): string[];

	/** Search items by query */
	search(query: string): T[];

	/** Get all registered items */
	getAll(): T[];
}

// =============================================================================
// PRESENTABLE REGISTRY IMPLEMENTATION
// =============================================================================

/**
 * Generic registry implementation.
 */
export class PresentableRegistry<T extends IPresentable> implements IPresentableRegistry<T> {
	items = new Map<string, T>();

	register(item: T): void {
		this.items.set(item.id, item);
	}

	get(id: string): T | undefined {
		return this.items.get(id);
	}

	getByCategory(category: PresentableCategory): T[] {
		return Array.from(this.items.values()).filter((item) => item.category === category);
	}

	getCategories(): string[] {
		const categories = new Set<string>();
		this.items.forEach((item) => categories.add(item.category));
		return Array.from(categories).sort();
	}

	search(query: string): T[] {
		const q = query.toLowerCase();
		return Array.from(this.items.values()).filter(
			(item) =>
				item.name.toLowerCase().includes(q) ||
				item.id.includes(q) ||
				item.category.toLowerCase().includes(q) ||
				item.description?.toLowerCase().includes(q) ||
				item.tags?.some((t) => t.toLowerCase().includes(q))
		);
	}

	/** Get all registered items */
	getAll(): T[] {
		return Array.from(this.items.values());
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

/**
 * Create a minimal presentable item.
 */
export function createPresentable(
	id: string,
	name: string,
	category: PresentableCategory,
	extras?: Partial<IPresentable>
): IPresentable {
	return {
		id,
		name,
		category,
		...extras
	};
}

/**
 * Check if an object is a valid IPresentable.
 */
export function isPresentable(obj: unknown): obj is IPresentable {
	if (typeof obj !== 'object' || obj === null) return false;
	const p = obj as Record<string, unknown>;
	return (
		typeof p.id === 'string' &&
		typeof p.name === 'string' &&
		typeof p.category === 'string'
	);
}

/**
 * Merge two presentables, with second overriding first.
 */
export function mergePresentable<T extends IPresentable>(base: T, overrides: Partial<T>): T {
	return {
		...base,
		...overrides,
		tags: [...(base.tags ?? []), ...(overrides.tags ?? [])]
	};
}
