// @sbx/canvas-kit — headless canvas engine: types + pure resolvers (render &
// export land in later slices). EXTRACTED & EXTENDED from
// clients/andrei/projects/land-canvas — see docs/plans/2026-06-01-media-canvas-design.md.
// MUST NOT import any Svelte runtime — UI components live in @sbx/canvas-ui.

// --- Core model (ported from land-canvas) ---
export type { Document, Page, SourceRef } from './types/document.js';
// --- Document-level watermark params (Slice A · Decision #0298) ---
export { DEFAULT_WATERMARK_PARAMS, SHUTTERSTOCK_CLASSIC_PARAMS } from './types/watermark.js';
export type {
	WatermarkParams,
	WatermarkTiling,
	WatermarkPosition,
	WatermarkTheme,
	WatermarkBlend
} from './types/watermark.js';
// --- Document-level PREVIEW backdrop (Slice A2.3 · Decision #0298) — PREVIEW-ONLY,
//     stage chrome, structurally excluded from the export path (resolveFrame reads
//     only page.layers). See types/preview-backdrop.ts. ---
export { DEFAULT_PREVIEW_BACKDROP } from './types/preview-backdrop.js';
export type { PreviewBackdrop, PreviewBackdropKind } from './types/preview-backdrop.js';
export type { Layer, LayerType, TileEffect, AnchorPoint, EdgeLabel, AreaLabel, MapConfig, BgMode, MapType } from './types/layer.js';
export { styleToMapType } from './types/layer.js';
export type { StrokeStyle, FillStyle, ShadowStyle, GlowStyle, ImageTint, ColorSwatch, StylePreset } from './types/style.js';
export type { Template, BrandKit, BrandColor, BrandFont, BrandLogo, BrandSource, ExportTarget, Tool, ShapeToolType } from './types/template.js';
export type { DistanceUnit, AreaUnit } from './types/measurement.js';

// --- Media Canvas extensions ---
export type { AnimationTrack, Keyframe, Easing } from './types/animation.js';
export type { Binding } from './types/binding.js';
export type {
	SourceKind,
	SourceProvider,
	SourceRecord,
	SourcePage,
	SourceCategory,
	SourceFacet,
	SourceFilters,
	SourceSnapshot,
	ResolvedImage,
	ImageRenderRole,
	ImageCategory,
	FieldDescriptor,
	FieldType,
	FormatOption,
	FormatOpts,
	FormatterDescriptor,
	FormatterRegistry
} from './types/source.js';
// Value export — the pure role→URL picker (graceful fallback to `url`).
export { pickImageVariant } from './types/source.js';

// --- Pure resolvers (BASE → BINDING → ANIMATION@t) ---
export {
	resolveLayerAtTime,
	resolveBindings,
	resolveTemplateContent,
	splitSourceToken,
	descriptorDefaults,
	formatTokenValue
} from './resolve.js';

// --- Inspector format-knob choice labels + previews (S-FORMAT) ---
export { formatChoiceLabel, formatChoicePreview, formatKnobChoices } from './format-labels.js';

// --- Field-type model (text | linked | template) — INC-B ---
export { fieldKindFor } from './field-kind.js';
export type { CanvasFieldKind } from './field-kind.js';

// --- Editor mode → feature bundle (Slice A · Decision #0298) ---
export { resolveEditorFeatures } from './editor-mode.js';
export type { EditorMode, EditorFeatures } from './editor-mode.js';

// --- Enum/list display formatter factory (INC-D) ---
export { makeEnumFormatter } from './enum-formatter.js';
export type { EnumAliasMap } from './enum-formatter.js';

// --- Editor-state reducers (headless; @sbx/canvas-ui adds Svelte reactivity) ---
export {
	createDocument,
	createWatermarkDocument,
	addLayer,
	reorderLayer,
	updateLayer,
	deleteLayer,
	resizePage,
	setPageDuration,
	setDocumentMeta,
	selectLayer,
	nextSourceAlias,
	attachSource,
	detachSource,
	setSourceSnapshot,
	BUILTIN_SOURCE_ALIASES,
	isBuiltinSourceAlias,
	bindLayer,
	unbindLayer,
	setBindingOverride,
	clearBindingOverride,
	findPage,
	createBackgroundLayer,
	backgroundFor,
	ensureBackgroundLayers,
	setBackground,
	isDefaultBackground
} from './store.js';
export type { EditorState, BuiltinSourceAlias, BackgroundSpec } from './store.js';

// --- Animation authoring reducers + helpers (slice-5a, design §12.9) ---
export { addAnimation, updateAnimation, removeAnimation, snapMs, easingPreset, easingToPresetName } from './animate.js';
export type { AnimationSpec, AnimatableProperty, EasingPresetName } from './animate.js';

// --- Preset → Layer factory (insert palette materialisation, D15) + injected
//     brand defaults (Decision #0286 Phase A) ---
export { presetToLayer, NEUTRAL_LAYER_DEFAULTS, EMPTY_BRAND_KIT } from './factory.js';
export type { PresetInput, PresetBaseType, LayerDefaults } from './factory.js';

// --- Geometry (resize handles + pure resize math; Web Mercator projection) ---
export { handleRects, resizeLayer, cropLayer, rotationAtPointer, pointInLayer, projectToPixel, unprojectFromPixel, HANDLE_CURSORS, CROP_KEYS, MIN_SIZE } from './geometry.js';
export type { Handle, HandleRect, Geometry, CropEdge, PixelPoint, HitRegion } from './geometry.js';

// --- Playback / storyboard sequencing ---
export { frameAtTime, totalDuration, DEFAULT_FRAME_MS } from './timeline.js';
export type { FrameAt } from './timeline.js';

// --- Map layer static-URL resolver (canvas-kit seam, injected builder) ---
export { mapLayerStaticUrl } from './map-source.js';
export type { MapStaticUrlBuilder, MapStaticDescriptor, MapMarkerInput, MapPolygonInput } from './map-source.js';

// --- Map annotation anchoring (marker/outline/callout → map surface) ---
export { mapLayerAt, anchorAnnotationToMap, resolveAnchorTarget } from './map-anchor.js';
export type { AnchorTarget } from './map-anchor.js';

// --- Frame export (resolve → rasterize, both seams injectable) ---
export { resolveFrame, exportPage, exportMarkUnit } from './export.js';
export type { Rasterizer, ExportOptions } from './export.js';

// --- Watermark tiling geometry (the canonical mark-unit bbox — preview · publish · apply) ---
export { markBounds, tilePositions } from './watermark-tiling.js';
export type { TilePlacement } from './watermark-tiling.js';

// --- Canvas 2D renderer (the seam the export Rasterizer reuses) ---
export { renderLayer, renderPage, renderCropGhost, tileFillRegion, CROP_GHOST_ALPHA, renderShapeLayer, renderTextLayer, renderImageLayer, wrapText, truncateToWidth, fitFontSize, IMAGE_FAILED } from './render.js';
export type { ImageSource, ImageSourceResult, TileRegion } from './render.js';
