/**
 * IView Interface - Runtime Interface for UI Components
 *
 * IView defines how components BEHAVE at runtime, complementing IPresenter
 * which defines how components are DOCUMENTED in /components.
 *
 * Key Features:
 * - DECORATOR PATTERN: Extend behavior via plugins without modifying components
 * - THEME OVERRIDES: Per-instance visual customization
 * - DEV MODE: Inspection, selection, recording for development
 * - FIXTURE LINKING: Load test data from entity fixtures
 *
 * Architecture:
 *   IView (runtime behavior)  +  IPresenter (documentation)  =  Complete Component
 *
 * @example
 * // Basic view configuration
 * const buttonView: IView = {
 *   id: 'submit-button',
 *   name: 'Submit Button',
 *   component: 'Button',
 *   category: 'primitives',
 *   props: { variant: 'primary' }
 * };
 *
 * @example
 * // View with decorators
 * const interactiveView: IView = {
 *   id: 'interactive-button',
 *   name: 'Interactive Button',
 *   component: 'Button',
 *   category: 'primitives',
 *   decorators: [
 *     { id: 'dev-mode', name: 'DevMode', priority: 100, config: { selectable: true } },
 *     { id: 'branding', name: 'Branding', priority: 200, config: { theme: 'dark' } }
 *   ],
 *   devMode: { recordable: true }
 * };
 */

import type { Component, Snippet } from 'svelte';
import type { IMotionConfig } from './MotionTokens';
import { motionConfigToCSS } from './MotionTokens';

// =============================================================================
// DECORATOR INTERFACE
// =============================================================================

/**
 * Lifecycle hooks that decorators can implement.
 * All hooks are optional - implement only what you need.
 */
export interface IDecoratorHooks<T = unknown> {
	/** Called before component mounts to DOM */
	onBeforeMount?: (view: IView, context: IDecoratorContext) => void | boolean;

	/** Called after component mounts to DOM */
	onAfterMount?: (view: IView, context: IDecoratorContext, element: HTMLElement) => void;

	/** Called before props update */
	onBeforeUpdate?: (view: IView, context: IDecoratorContext, nextProps: T) => T | void;

	/** Called after props update and re-render */
	onAfterUpdate?: (view: IView, context: IDecoratorContext, prevProps: T) => void;

	/** Called before component unmounts */
	onBeforeDestroy?: (view: IView, context: IDecoratorContext) => void;

	/** Intercept DOM events on component. Return false to stop propagation. */
	onEvent?: (view: IView, context: IDecoratorContext, event: Event) => boolean | void;

	/** Wrap component rendering with additional elements */
	wrapRender?: (view: IView, context: IDecoratorContext, children: Snippet) => Snippet;
}

/**
 * Context passed to decorator hooks.
 */
export interface IDecoratorContext {
	/** Global dev mode flag */
	devMode: boolean;

	/** Current theme settings */
	theme: IThemeConfig;

	/** Registered decorators */
	decorators: Map<string, IDecorator>;

	/** Event bus for inter-decorator communication */
	emit: (event: string, payload?: unknown) => void;

	/** Subscribe to events */
	on: (event: string, handler: (payload?: unknown) => void) => () => void;

	/** Get element reference */
	getElement: () => HTMLElement | null;

	/** Record a change for persistence */
	recordChange?: (path: string, value: unknown) => void;
}

/**
 * Decorator definition - plugin that extends view behavior.
 *
 * @example
 * const devModeDecorator: IDecorator = {
 *   id: 'dev-mode',
 *   name: 'DevMode',
 *   priority: 100,
 *   enabled: true,
 *   config: { highlightColor: '#6366f1' },
 *   hooks: {
 *     onAfterMount: (view, ctx, el) => {
 *       if (ctx.devMode) el.dataset.viewId = view.id;
 *     }
 *   }
 * };
 */
export interface IDecorator<TConfig = Record<string, unknown>> {
	/** Unique identifier for this decorator instance */
	id: string;

	/** Display name for debugging and dev tools */
	name: string;

	/** Execution order (lower = runs first). Default: 1000 */
	priority?: number;

	/** Toggle decorator on/off. Default: true */
	enabled?: boolean;

	/** Decorator-specific configuration */
	config?: TConfig;

	/** Lifecycle hook implementations */
	hooks?: IDecoratorHooks;
}

// =============================================================================
// THEME INTERFACE
// =============================================================================

/**
 * Theme override configuration.
 * Merges with global theme, doesn't replace.
 */
export interface IThemeOverrides {
	/** Color overrides (mapped to --color-{key}) */
	colors?: Record<string, string>;

	/** Spacing overrides (mapped to --spacing-{key}) */
	spacing?: Record<string, string>;

	/** Typography overrides (mapped to --typography-{key}) */
	typography?: Record<string, string>;

	/** Border overrides (mapped to --border-{key}) */
	borders?: Record<string, string>;

	/** Shadow overrides (mapped to --shadow-{key}) */
	shadows?: Record<string, string>;

	/** Motion overrides (entrance, exit, emphasis animations) */
	motion?: IMotionConfig;
}

/**
 * Full theme configuration (global + overrides merged).
 */
export interface IThemeConfig extends IThemeOverrides {
	/** Theme name/identifier */
	name: string;

	/** Theme mode */
	mode: 'light' | 'dark' | 'system';

	/** Custom CSS variables */
	customProperties?: Record<string, string>;
}

// =============================================================================
// ANIMATION INTERFACE
// =============================================================================

/**
 * Animation/transition configuration for component.
 */
export interface IAnimation {
	/** Animation when component enters (mount) */
	enter?: string;

	/** Animation when component exits (unmount) */
	exit?: string;

	/** Duration in milliseconds. Default: 200 */
	duration?: number;

	/** CSS timing function. Default: 'ease-out' */
	easing?: string;

	/** Delay before animation starts. Default: 0 */
	delay?: number;
}

// =============================================================================
// CONTEXT MENU INTERFACE
// =============================================================================

/**
 * Context menu item for right-click actions.
 */
export interface IContextMenuItem {
	/** Unique identifier */
	id: string;

	/** Display text */
	label: string;

	/** Icon (emoji or icon name) */
	icon?: string;

	/** Keyboard shortcut hint */
	shortcut?: string;

	/** Show separator line before this item */
	separator?: boolean;

	/** Disable item while showing */
	disabled?: boolean;

	/** Action handler when clicked */
	action?: (view: IView, context: IDecoratorContext) => void;

	/** Submenu items */
	children?: IContextMenuItem[];
}

// =============================================================================
// DEV MODE INTERFACE
// =============================================================================

/**
 * Developer tools configuration.
 */
export interface IDevModeConfig {
	/** Allow component selection in dev tools. Default: true */
	selectable?: boolean;

	/** Show highlight on hover. Default: true */
	highlightable?: boolean;

	/** Show props in inspector. Default: true */
	inspectable?: boolean;

	/** Allow recording prop changes. Default: false */
	recordable?: boolean;

	/** Additional visual overlays */
	overlays?: ('bounds' | 'spacing' | 'grid' | 'accessibility')[];

	/** Highlight color override */
	highlightColor?: string;
}

// =============================================================================
// FIXTURE INTERFACE
// =============================================================================

/**
 * Reference to entity fixture for preview/test data.
 */
export interface IFixtureRef {
	/** Entity type that owns the fixture */
	entity: string;

	/** Fixture name within entity */
	fixture: string;

	/** How this fixture is used */
	use?: 'preview' | 'e2e' | 'both';

	/** JSONPath or transform to extract subset */
	transform?: string;
}

// =============================================================================
// LOCALE INTERFACE
// =============================================================================

/**
 * Localization configuration for views.
 *
 * When locale is configured, the view system automatically translates
 * all text fields. No per-field _i18n suffixes needed - i18n is implicit.
 *
 * @example
 * const localizedView: IView = {
 *   id: 'sidebar',
 *   name: 'Sidebar',
 *   component: 'Sidebar',
 *   category: 'sections',
 *   locale: {
 *     namespace: 'nav',
 *     enabled: true
 *   }
 * };
 */
export interface ILocaleConfig {
	/** i18n namespace for this view's translations. Default: "common" */
	namespace?: string;

	/** Toggle automatic translation. Default: true when locale object present */
	enabled?: boolean;

	/** Behavior when translation key missing. Default: "key" */
	fallback?: 'key' | 'original';
}

// =============================================================================
// VIEW INTERFACE
// =============================================================================

/** Component category in Modified Atomic Design */
export type ViewCategory = 'primitives' | 'blocks' | 'sections' | 'layouts';

/**
 * IView - Runtime interface for UI components.
 *
 * Defines how components behave at runtime:
 * - Decorators extend behavior without modification
 * - Theme overrides customize appearance
 * - Dev mode enables inspection and recording
 * - Fixtures provide preview/test data
 */
export interface IView<TProps = Record<string, unknown>> {
	// =========================================================================
	// IDENTITY (Required)
	// =========================================================================

	/** Unique identifier for view instance */
	id: string;

	/** Display name for dev tools */
	name: string;

	/** Component reference (name or actual component) */
	component: string | Component;

	/** Component hierarchy level */
	category: ViewCategory;

	// =========================================================================
	// BEHAVIOR (Optional)
	// =========================================================================

	/** Decorators to apply (plugins) */
	decorators?: IDecorator[];

	/** Theme customization */
	theme?: IThemeOverrides;

	/** Animation configuration */
	animation?: IAnimation;

	/** Context menu items */
	contextMenu?: IContextMenuItem[];

	/** Developer tools configuration */
	devMode?: IDevModeConfig;

	/** Localization - when set, all text fields auto-translate */
	locale?: ILocaleConfig;

	// =========================================================================
	// DATA (Optional)
	// =========================================================================

	/** Entity fixture references */
	fixtures?: IFixtureRef[];

	/** Default props for component */
	props?: TProps;

	/** Named slot content */
	slots?: Record<string, string | Snippet>;

	// =========================================================================
	// METADATA (Optional)
	// =========================================================================

	/** Link to presenter for documentation */
	presenter?: string;

	/** Tags for categorization */
	tags?: string[];

	/** Description */
	description?: string;
}

// =============================================================================
// VIEW REGISTRY
// =============================================================================

/**
 * Registry for managing view configurations.
 */
export interface IViewRegistry {
	/** All registered views */
	views: Map<string, IView>;

	/** Register a view configuration */
	register(view: IView): void;

	/** Get view by id */
	get(id: string): IView | undefined;

	/** Get all views by category */
	getByCategory(category: ViewCategory): IView[];

	/** Get all views with a specific decorator */
	getWithDecorator(decoratorId: string): IView[];

	/** Search views by query */
	search(query: string): IView[];
}

/**
 * View registry implementation.
 */
export class ViewRegistry implements IViewRegistry {
	views = new Map<string, IView>();

	register(view: IView): void {
		this.views.set(view.id, view);
	}

	get(id: string): IView | undefined {
		return this.views.get(id);
	}

	getByCategory(category: ViewCategory): IView[] {
		return Array.from(this.views.values()).filter((v) => v.category === category);
	}

	getWithDecorator(decoratorId: string): IView[] {
		return Array.from(this.views.values()).filter((v) =>
			v.decorators?.some((d) => d.id === decoratorId)
		);
	}

	search(query: string): IView[] {
		const q = query.toLowerCase();
		return Array.from(this.views.values()).filter(
			(v) =>
				v.name.toLowerCase().includes(q) ||
				v.id.includes(q) ||
				v.category.includes(q) ||
				v.description?.toLowerCase().includes(q) ||
				v.tags?.some((t) => t.toLowerCase().includes(q))
		);
	}
}

// =============================================================================
// GLOBAL REGISTRY
// =============================================================================

/** Global view registry */
export const viewRegistry = new ViewRegistry();

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

/**
 * Create a minimal view configuration.
 */
export function createView<TProps = Record<string, unknown>>(
	id: string,
	name: string,
	component: string | Component,
	category: ViewCategory,
	config?: Partial<IView<TProps>>
): IView<TProps> {
	return {
		id,
		name,
		component,
		category,
		...config
	};
}

/**
 * Create a decorator.
 */
export function createDecorator<TConfig = Record<string, unknown>>(
	id: string,
	name: string,
	hooks: IDecoratorHooks,
	config?: TConfig
): IDecorator<TConfig> {
	return {
		id,
		name,
		hooks,
		config
	};
}

/**
 * Apply decorators to a view in priority order.
 */
export function applyDecorators(view: IView, context: IDecoratorContext): void {
	const decorators = view.decorators?.filter((d) => d.enabled !== false) ?? [];

	// Sort by priority (lower = first)
	decorators.sort((a, b) => (a.priority ?? 1000) - (b.priority ?? 1000));

	// Execute onBeforeMount hooks
	for (const decorator of decorators) {
		decorator.hooks?.onBeforeMount?.(view, context);
	}
}

/**
 * Merge theme overrides with base theme.
 */
export function mergeTheme(base: IThemeConfig, overrides?: IThemeOverrides): IThemeConfig {
	if (!overrides) return base;

	return {
		...base,
		colors: { ...base.colors, ...overrides.colors },
		spacing: { ...base.spacing, ...overrides.spacing },
		typography: { ...base.typography, ...overrides.typography },
		borders: { ...base.borders, ...overrides.borders },
		shadows: { ...base.shadows, ...overrides.shadows },
		motion: overrides.motion ? { ...base.motion, ...overrides.motion } : base.motion
	};
}

/**
 * Convert theme to CSS custom properties.
 */
export function themeToCSS(theme: IThemeConfig): string {
	const props: string[] = [];

	if (theme.colors) {
		for (const [key, value] of Object.entries(theme.colors)) {
			props.push(`--color-${key}: ${value};`);
		}
	}

	if (theme.spacing) {
		for (const [key, value] of Object.entries(theme.spacing)) {
			props.push(`--spacing-${key}: ${value};`);
		}
	}

	if (theme.typography) {
		for (const [key, value] of Object.entries(theme.typography)) {
			props.push(`--typography-${key}: ${value};`);
		}
	}

	if (theme.borders) {
		for (const [key, value] of Object.entries(theme.borders)) {
			props.push(`--border-${key}: ${value};`);
		}
	}

	if (theme.shadows) {
		for (const [key, value] of Object.entries(theme.shadows)) {
			props.push(`--shadow-${key}: ${value};`);
		}
	}

	if (theme.customProperties) {
		for (const [key, value] of Object.entries(theme.customProperties)) {
			props.push(`${key}: ${value};`);
		}
	}

	if (theme.motion) {
		const motionCSS = motionConfigToCSS(theme.motion);
		if (motionCSS) {
			props.push(motionCSS);
		}
	}

	return props.join('\n');
}
