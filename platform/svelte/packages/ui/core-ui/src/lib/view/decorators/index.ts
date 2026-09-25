/**
 * View Decorators - Plugin system for extending component behavior
 *
 * Decorators follow the IDecorator interface and can:
 * - Hook into component lifecycle (mount, update, destroy)
 * - Intercept and modify props
 * - Wrap rendering with additional elements
 * - Handle events before components
 *
 * @example
 * import { createDevModeDecorator, createBrandingDecorator } from './decorators';
 *
 * const view: IView = {
 *   id: 'my-component',
 *   name: 'My Component',
 *   component: MyComponent,
 *   category: 'blocks',
 *   decorators: [
 *     createDevModeDecorator({ showBounds: true }),
 *     createBrandingDecorator({ brand: 'hub' })
 *   ]
 * };
 */

// =============================================================================
// DECORATORS
// =============================================================================

export {
	default as createDevModeDecorator,
	getSelectedViewId,
	isViewSelected,
	selectViewById,
	clearSelection,
	type DevModeConfig
} from './DevModeDecorator';

export {
	default as createBrandingDecorator,
	getAvailableBrands,
	getBrandPreset,
	registerBrand,
	getSystemMode,
	switchTheme,
	getThemeCSS,
	BRAND_PRESETS,
	type BrandingConfig
} from './BrandingDecorator';

export {
	default as createRecordingDecorator,
	startSession,
	endSession,
	getCurrentSession,
	getSession,
	recordPropChange,
	recordInteraction,
	saveSession,
	exportToJSON,
	exportToYAML,
	undo,
	redo,
	clearHistory,
	type RecordingConfig,
	type ChangeRecord,
	type ChangeType,
	type RecordingSession
} from './RecordingDecorator';

// =============================================================================
// RE-EXPORTS FROM IView
// =============================================================================

export type {
	IDecorator,
	IDecoratorHooks,
	IDecoratorContext
} from '../IView';

export { createDecorator, applyDecorators } from '../IView';
