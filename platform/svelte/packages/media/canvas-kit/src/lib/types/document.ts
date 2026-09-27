// Aggregate root — EXTRACTED from land-canvas and EXTENDED for Media Canvas.
// Additions marked `[MC]`.

import type { Layer } from './layer.js';
import type { SourceKind, SourceSnapshot } from './source.js';
import type { WatermarkParams } from './watermark.js';
import type { PreviewBackdrop } from './preview-backdrop.js';

/** Artboard within a document — has dimensions and contains layers. */
export interface Page {
	id: string;
	width: number;
	height: number;
	layers: Layer[];
	/** [MC] frame duration in ms; absent = a still image / single frame. */
	duration?: number;
}

/** [MC] An attached domain record the document binds to. */
export interface SourceRef {
	id: string;
	kind: SourceKind;
	/** Provider-scoped record id. */
	refId: string;
	/** Local alias bindings reference; alias 'primary' = the Active record (drives preview). */
	alias: string;
	/** Cached token → value map so the document renders without re-fetching. */
	snapshot?: SourceSnapshot;
}

/** Root container for a graphic composition. */
export interface Document {
	id: string;
	name: string;
	pages: Page[];
	/** [MC] attached sources (property/guide/service records). */
	sources?: SourceRef[];
	/** [MC] true → placeholder bindings re-resolve when a new document is spawned from this. */
	isTemplate?: boolean;
	/** [MC] R3 animated mode. true → the doc plays (timeline shown, motion render);
	 *  false → static (timeline hidden, base-value render). Absent → derived: animated
	 *  iff any layer carries animation tracks. Toggling OFF keeps the tracks. */
	animated?: boolean;
	/** [WM] Document-level watermark stamping params (Slice A · #0298) — how the whole
	 *  overlay is stamped onto a target image. Absent until the watermark Inspector
	 *  block is first edited (seeded from `DEFAULT_WATERMARK_PARAMS`). Persists as part
	 *  of the doc's opaque JSONB. */
	watermark?: WatermarkParams;
	/** [WM] PREVIEW-ONLY stage backdrop (Slice A2.3 · #0298) — the colour/image the
	 *  author previews the transparent mark over to judge legibility. Rendered by
	 *  CanvasStage as STAGE CHROME *behind* the artboard, NOT as a layer, so the export
	 *  path (`exportPage` → `resolveFrame`, which reads only `page.layers`) can never
	 *  include it in the published overlay (`overlay_r2_key`). Persisting it in the doc
	 *  JSONB is a per-doc preview preference only — it never reaches the exported pixels.
	 *  Absent until the Watermark Inspector's backdrop control is first changed (seeded
	 *  from `DEFAULT_PREVIEW_BACKDROP`). */
	previewBackdrop?: PreviewBackdrop;
}
