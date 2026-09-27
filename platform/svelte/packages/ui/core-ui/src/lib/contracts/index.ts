/**
 * Contracts Index
 * ===============
 *
 * Export all contract interfaces for type-safe development.
 *
 * Architecture:
 *   IPresentable (base contract)
 *       ↓ extends
 *   IView (runtime)  |  IPresenter (docs)  |  IBrandConfig (theming)
 */

// =============================================================================
// PRESENTABLE CONTRACT (Base)
// =============================================================================

export type {
	PresentableCategory,
	PresentableStatus,
	IPresentable,
	IPreviewConfig,
	IPresentableWithPreview,
	ISourceRef,
	IPresentableWithSource,
	IPresentableRegistry
} from './IPresentable.js';

export { PresentableRegistry, createPresentable, isPresentable, mergePresentable } from './IPresentable.js';
