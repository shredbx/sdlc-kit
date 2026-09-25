// Editor mode → feature bundle (Slice A2.1 · Decision #0298).
//
// The canvas editor (@sbx/canvas-ui EditorShell) is a single configuration-driven
// shell. Rather than fork it for the watermark surface, a generic `mode` selects a
// COHERENT preset of feature gates — the natural extension of the shell's existing
// prop pattern (#0298). This module is pure + Svelte-free so it lives in the headless
// kit: the shell threads the resolved flags to its children, and the BR consumer
// (the watermark route) reuses the SAME contract.
//
//   design     the existing Media Canvas — full capability (the back-compat default).
//   watermark  a static overlay-authoring surface — no motion, no live data, no
//              components/map; gains Publish + Copy + a Watermark inspector block + a
//              preview backdrop.
//
// SECURITY (#0298): these flags are CLIENT-side UI gating ONLY. The authoritative
// watermark save/publish validation (reject live-source bindings, strip animation,
// enforce kind='watermark') is server-owned and lands in a later slice — never trust
// `mode` for persistence decisions.

/** Which editor experience the shell presents. */
export type EditorMode = 'design' | 'watermark';

/** The coherent bundle of capability gates a mode resolves to. Each flag is read by
 *  one shell seam (LeftRail items, the DocumentPanel kind control, the Inspector
 *  animate chips + Watermark block, the TopToolbar actions, the stage backdrop). */
export interface EditorFeatures {
	/** Motion authoring — the Timeline + the Inspector per-property animate chips. */
	animation: boolean;
	/** Live-data sources — the standalone Sources rail item (attach/detach records). */
	liveSources: boolean;
	/** The Components rail item (shape/widget palette). */
	components: boolean;
	/** The Map rail item (map surface + map background). */
	map: boolean;
	/** Whether the document Kind is user-selectable (vs a fixed, read-only label). */
	kindSelectable: boolean;
	/** The Publish action (TopToolbar) — promote the design into the registry. */
	publish: boolean;
	/** The Copy/duplicate action (TopToolbar). */
	copy: boolean;
	/** The Watermark inspector block (params; rendered at the top of the Inspector). */
	watermarkInspector: boolean;
	/** A preview-only stage backdrop (solid colour / picked image) — excluded from the
	 *  published overlay. Plumbed here; the backdrop UI lands in a later slice. */
	previewBackdrop: boolean;
	/** The in-editor TILED preview (Slice B0b · #0298) — a TopToolbar toggle that flips
	 *  the stage from authoring the single tile to seeing it tiled (per the watermark
	 *  params: rotation/density/opacity/blend) over the preview backdrop, the way the
	 *  apply pipeline will stamp it across a photo. Preview-only; never persisted. */
	watermarkTiledPreview: boolean;
	/** The Templates section inside the Document panel (start-from-template gallery).
	 *  A Media-Canvas concept — irrelevant to overlay authoring, so OFF for watermark. */
	documentTemplates: boolean;
	/** The iPhone-Photos-style document-sizing control in the Resize dialog (Slice S1).
	 *  A watermark is a full overlay sized to the image max (2048) — authors pick an
	 *  orientation + ratio rather than a preset social size. ON for watermark; OFF for
	 *  design (the Media Canvas keeps its preset-grid resize, back-compat). */
	watermarkPhotoSizing: boolean;
}

const DESIGN_FEATURES: Readonly<EditorFeatures> = {
	animation: true,
	liveSources: true,
	components: true,
	map: true,
	kindSelectable: true,
	publish: false,
	copy: false,
	watermarkInspector: false,
	previewBackdrop: false,
	watermarkTiledPreview: false,
	documentTemplates: true,
	watermarkPhotoSizing: false
};

const WATERMARK_FEATURES: Readonly<EditorFeatures> = {
	animation: false,
	liveSources: false,
	components: false,
	map: false,
	kindSelectable: false,
	publish: true,
	copy: true,
	watermarkInspector: true,
	previewBackdrop: true,
	watermarkTiledPreview: true,
	documentTemplates: false,
	watermarkPhotoSizing: true
};

/** Resolve an editor mode to its feature bundle. Pure: returns a FRESH object each
 *  call so a caller can never mutate the shared preset. */
export function resolveEditorFeatures(mode: EditorMode): EditorFeatures {
	return { ...(mode === 'watermark' ? WATERMARK_FEATURES : DESIGN_FEATURES) };
}
