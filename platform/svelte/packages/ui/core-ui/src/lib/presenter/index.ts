/**
 * Presenter Module - Auto-Registration for Components
 *
 * The Presenter pattern enables components to be automatically
 * listed in hub's /components page with minimal configuration.
 *
 * @module presenter
 */

// Core presenter interface and registry
export {
	type IPresenter,
	type IVariant,
	type IPreset,
	type IPropDef,
	type PropType,
	type IPreviewConfig,
	type PreviewBackground,
	type ISourceConfig,
	type IFixtureConfig,
	type PresenterType,
	type PresenterCategory,
	type PresenterStatus,
	type IPresenterRegistry,
	PresenterRegistry,
	presenterRegistry,
	createComponentPresenter,
	createPackagePresenter
} from './IPresenter';

// Component registry - auto-discovery (side-effect: registers all presenters on import)
export {
	getPresenterCounts,
	getComponentsByCategory,
	getCategoryCounts
} from './componentRegistry';

// Component representation system
export {
	type RepresentationContext,
	type RepresentationMode,
	type PreviewSize,
	type ComponentRepresentation,
	type LoadingBehavior,
	type LoadingConfig,
	type TreeItem,
	type CardConfig,
	getRepresentation,
	getLoadingConfig,
	canRenderInline,
	needsFullPreview,
	getCategoryIcon,
	getCategoryLabel,
	getCategoryDescription,
	presentersToTreeItems,
	getCardConfig,
	LEVEL_PREVIEW_SIZES,
	LEVEL_CONTEXT_MODES,
	CONTEXT_OPTIONS
} from './ComponentRepresentation';
