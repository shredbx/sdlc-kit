/**
 * Presenter Types - Re-exports for convenient importing
 *
 * This module provides type aliases and re-exports commonly used
 * presenter types for cleaner imports in presenter files.
 *
 * @module presenter/types
 */

export type {
	IPresenter,
	IVariant as Variant,
	IPreset as Preset,
	IPropDef as PropDef,
	PropType,
	IPreviewConfig as PreviewConfig,
	PreviewBackground,
	ISourceConfig as SourceConfig,
	IFixtureConfig as FixtureConfig,
	PresenterType,
	PresenterCategory,
	PresenterStatus
} from './IPresenter';
