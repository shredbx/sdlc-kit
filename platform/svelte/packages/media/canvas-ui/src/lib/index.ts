// @sbx/canvas-ui — Svelte 5 editor components for the Media Canvas (light theme,
// brand-agnostic --cv-* tokens with neutral fallbacks; the consumer maps them onto
// its own brand vars — Decision #0286 Phase A). Composes @sbx/core-ui primitives
// over the headless @sbx/canvas-kit engine. See docs/plans/2026-06-01-media-canvas-design.md §11.

// Editor shell + chrome ----------------------------------------------------
export { default as EditorShell } from './EditorShell.svelte';
export { default as TopToolbar } from './TopToolbar.svelte';
export { default as LeftRail } from './LeftRail.svelte';
export type { RailContext } from './LeftRail.svelte';
export { default as CanvasStage } from './CanvasStage.svelte';
export { default as Inspector } from './Inspector.svelte';
export { default as Timeline } from './Timeline.svelte';

// Shared panel chrome ------------------------------------------------------
export { default as Panel } from './Panel.svelte';
export { default as PanelSearch } from './PanelSearch.svelte';
export { default as CollapsibleSection } from './CollapsibleSection.svelte';
export { default as EmptyState } from './EmptyState.svelte';
export { default as ComponentGrid } from './ComponentGrid.svelte';
export { default as ComponentCard } from './ComponentCard.svelte';
export { default as SourceImageGrid } from './SourceImageGrid.svelte';
export { default as SourceCell } from './SourceCell.svelte';
export { default as ThumbImage } from './ThumbImage.svelte';
export { default as SegmentedControl } from './SegmentedControl.svelte';
export { default as ScrubUnit } from './ScrubUnit.svelte';

// Contextual panels (one per rail item) ------------------------------------
export { default as DocumentPanel } from './DocumentPanel.svelte';
export { default as SourcesPanel } from './SourcesPanel.svelte';
export { default as TemplatesPanel } from './TemplatesPanel.svelte';
export { default as TemplatePreview } from './TemplatePreview.svelte';
export type { TemplateGallery, TemplateSummary } from './template-gallery.js';
export { default as LayersPanel } from './LayersPanel.svelte';
export { default as MediaPanel } from './MediaPanel.svelte';
export { default as TextsPanel } from './TextsPanel.svelte';
export { default as ComponentsPanel } from './ComponentsPanel.svelte';
export { default as MapPanel } from './MapPanel.svelte';
export { default as ResizeDialog } from './ResizeDialog.svelte';
export { default as ExportDialog } from './ExportDialog.svelte';
export { default as AddAnimationDialog } from './AddAnimationDialog.svelte';
export { default as AnimationInfoPopover } from './AnimationInfoPopover.svelte';
export { default as TimelineLane } from './TimelineLane.svelte';
export { default as EasingCurvePreview } from './EasingCurvePreview.svelte';

// Token-template WYSIWYG — a contenteditable surface mixing free text with atomic field
// badges (title-over-resolved-value), driven only by the @sbx/text-template protocols so it
// reuses across the canvas + post/content tools. Composes core-ui Select + Modal.
export { default as TemplateComposer } from './TemplateComposer.svelte';

// Reactive editor state (Svelte 5 wrapper over the kit reducers) ------------
export { CanvasEditor } from './editor-state.svelte.js';

// Shared catalogues (preset catalogue + layer icons) -----------------------
export { PRESETS, presetsForSection, layerIcon } from './palette.js';
export type { Preset, PresetBaseType } from './palette.js';
