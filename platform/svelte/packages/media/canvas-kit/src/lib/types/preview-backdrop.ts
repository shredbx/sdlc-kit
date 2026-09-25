// Document-level PREVIEW backdrop (Slice A2.3 · Decision #0298).
//
// A watermark design is a TRANSPARENT overlay. This backdrop lets the author judge
// the mark's legibility by previewing it over a chosen background — a solid colour
// OR a picked image. It rides the canvas document's opaque JSONB
// (`Document.previewBackdrop`), so it round-trips through the existing persistence
// path with no Go / migration change (a per-doc preview PREFERENCE, like the editor
// zoom would be).
//
// ⚠️ PREVIEW-ONLY — NEVER EXPORTED / PUBLISHED (Decision #0298, the backdrop-leakage
// threat). The backdrop is rendered by `CanvasStage` as STAGE CHROME *behind* the
// artboard — it is NOT a document layer and is NOT in `editor.layers`. The export
// path (`exportPage` → `resolveFrame`) rasterizes only a page's LAYERS, so stage
// chrome is structurally excluded from the published overlay (`overlay_r2_key`):
// `resolveFrame` is handed a `Page` and never sees the document's `previewBackdrop`.
// Persisting it in the doc JSONB is fine (it's a preview preference); it just must
// never reach the exported pixels — and by construction it cannot.

/** Which preview background the stage shows behind the transparent mark. */
export type PreviewBackdropKind = 'none' | 'color' | 'image';

/** Document-level preview backdrop. `none` = the transparent checker (no background);
 *  `color` reads `color`; `image` reads `imageSrc` (a src already available to the
 *  canvas — picked from the doc's attached image sources, routed through the same
 *  hardened image proxy as layer images, never a raw client URL). */
export interface PreviewBackdrop {
	kind: PreviewBackdropKind;
	/** Solid fill (kind:'color'). */
	color?: string;
	/** Image src (kind:'image') — a canvas-available image token's resolved URL. */
	imageSrc?: string;
}

/** A FRESH default preview-backdrop object (mirrors `DEFAULT_WATERMARK_PARAMS` — never
 *  a shared singleton, so a caller can seed a doc and mutate it without touching the
 *  defaults). The default is `none` — a transparent preview shows the mark as it
 *  publishes (over nothing). */
export function DEFAULT_PREVIEW_BACKDROP(): PreviewBackdrop {
	return { kind: 'none' };
}
