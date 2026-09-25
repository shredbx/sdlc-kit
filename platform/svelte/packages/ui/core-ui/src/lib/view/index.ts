/**
 * View Module Index
 *
 * Exports all view-related interfaces, utilities, and decorators.
 *
 * The view system provides:
 * - IView: Runtime interface for UI components
 * - IFixture: Entity-linked test data
 * - FixtureLoader: Bridge between YAML and TypeScript
 * - Decorators: Behavior plugins (DevMode, Branding, Recording)
 *
 * @example
 * // Import specific utilities
 * import { IView, createView, viewRegistry } from '.';
 * import { loadFixture, loadPreviewFixture } from '.';
 * import { createDevModeDecorator } from '.';
 */

// =============================================================================
// IView - Runtime Component Interface
// =============================================================================
export {
	// Types
	type IView,
	type IDecorator,
	type IDecoratorHooks,
	type IDecoratorContext,
	type IThemeOverrides,
	type IThemeConfig,
	type IAnimation,
	type IContextMenuItem,
	type IDevModeConfig,
	type IFixtureRef,
	type ILocaleConfig,
	type ViewCategory,
	type IViewRegistry,
	// Classes
	ViewRegistry,
	viewRegistry,
	// Functions
	createView,
	createDecorator,
	applyDecorators,
	mergeTheme,
	themeToCSS
} from './IView';

// =============================================================================
// IFixture - Entity-Linked Test Data
// =============================================================================
export {
	// Types
	type IFixture,
	type IFixtureField,
	type IFixtureFieldStatic,
	type IFixtureFieldFaker,
	type IFixtureFieldGenerator,
	type IFixtureFieldRef,
	type IFixtureFieldTransform,
	type IFixtureVariant,
	type IFixturePreviewUsage,
	type IFixtureE2EUsage,
	type IFixtureUsage,
	type IFixtureRelation,
	type IResolvedFixture,
	type IFixtureRegistry,
	// Classes
	FixtureRegistry,
	fixtureRegistry,
	// Functions
	createFixture,
	staticField,
	fakerField,
	refField,
	loadFixtureForPreview
} from './IFixture';

// =============================================================================
// FixtureLoader - YAML to TypeScript Bridge
// =============================================================================
export {
	// Types
	type YAMLFixture,
	type YAMLFixtureField,
	type YAMLVariant,
	type FixtureLoadOptions,
	// Functions
	loadFixture,
	loadFixtureData,
	loadFixtureDefinition,
	loadPreviewFixture,
	getFixtureVariants,
	getFixtureUsage,
	seedFixture,
	cleanupFixture,
	registerFixture,
	registerFixtures,
	convertYAMLToFixture,
	clearFixtureCache,
	getFixtureCacheStats
} from './FixtureLoader';

// =============================================================================
// Decorators
// =============================================================================
export {
	// DevMode
	createDevModeDecorator,
	type DevModeConfig,
	getSelectedViewId,
	isViewSelected,
	selectViewById,
	clearSelection
} from './decorators/DevModeDecorator';

export {
	// Branding
	createBrandingDecorator,
	type BrandingConfig,
	BRAND_PRESETS,
	getAvailableBrands,
	getBrandPreset,
	registerBrand,
	getSystemMode,
	switchTheme,
	getThemeCSS
} from './decorators/BrandingDecorator';

export {
	// Recording
	createRecordingDecorator,
	type RecordingConfig,
	type ChangeRecord,
	type ChangeType,
	type RecordingSession,
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
	clearHistory
} from './decorators/RecordingDecorator';

// =============================================================================
// Motion Tokens - Animation Catalogue
// =============================================================================
export {
	// Types
	type MotionCategory,
	type MotionDuration,
	type MotionEasing,
	type EntranceAnimation,
	type ExitAnimation,
	type EmphasisAnimation,
	type TransitionAnimation,
	type AnimationName,
	type IMotionToken,
	type IMotionConfig,
	type Keyframe,
	// Constants
	MOTION_DURATIONS,
	MOTION_EASINGS,
	ENTRANCE_TOKENS,
	EXIT_TOKENS,
	EMPHASIS_TOKENS,
	TRANSITION_TOKENS,
	MOTION_TOKENS,
	DEFAULT_MOTION_CONFIG,
	// Functions
	getMotionToken,
	resolveDuration,
	resolveEasing,
	motionToCSS,
	motionToKeyframes,
	generateAllKeyframesCSS,
	motionConfigToCSS,
	prefersReducedMotion,
	applyMotion
} from './MotionTokens';
