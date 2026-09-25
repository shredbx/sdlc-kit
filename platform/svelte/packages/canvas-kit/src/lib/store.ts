// Headless editor-state reducers — pure functions over a plain EditorState.
// The Svelte 5 reactive wrapper ($state) lives in @sbx/canvas-ui; the kit stays
// framework-agnostic (MUST NOT import the Svelte runtime). Every reducer returns
// a NEW state (never mutates input) so the UI layer can diff cheaply. The document
// is JSON-serializable, so a JSON clone is a correct deep clone.

import type { Document, Page, SourceRef } from './types/document.js';
import type { Layer } from './types/layer.js';
import type { Binding } from './types/binding.js';
import type { SourceSnapshot } from './types/source.js';
import type { FillStyle } from './types/style.js';
import { DEFAULT_WATERMARK_PARAMS, type WatermarkParams } from './types/watermark.js';
import { DEFAULT_PREVIEW_BACKDROP, type PreviewBackdrop } from './types/preview-backdrop.js';

/** The editor's mutable state: the open document plus the current selection. */
export interface EditorState {
	document: Document;
	/** Currently selected layer id, or null when nothing is selected. */
	selectedLayerId: string | null;
}

/** Create a new document with a single empty page of the given dimensions. The
 *  page seeds a 5s duration (design §12.7 / D19) so animation timing has a sane
 *  default; `DEFAULT_FRAME_MS` stays the bare fallback for pages that declare none. */
export function createDocument(name: string, width: number, height: number): Document {
	const pageId = uuid();
	return {
		id: uuid(),
		name,
		pages: [
			{
				id: pageId,
				width,
				height,
				duration: 5000,
				// Every page carries its system background (user directive 2026-06-06).
				layers: [createBackgroundLayer(pageId, width, height)]
			}
		]
	};
}

/** Like {@link createDocument}, but seeds a fully TRANSPARENT system background
 *  (Slice A · #0298). The watermark surface authors a transparent overlay — the
 *  published mark is stamped onto property photos — so the canvas must start clear:
 *  the preview backdrop shows through, and the published PNG is transparent wherever
 *  the mark is absent. Identical to createDocument otherwise. */
export function createWatermarkDocument(name: string, width: number, height: number): Document {
	const doc = createDocument(name, width, height);
	for (const page of doc.pages) {
		const bg = backgroundFor(page);
		if (bg?.fill) bg.fill = { ...bg.fill, transparency: 1 };
	}
	return doc;
}

/** Append a layer to a page's z-stack (end = front) and select it (MC-SC-01). */
export function addLayer(state: EditorState, pageId: string, layer: Layer): EditorState {
	if (!findPage(state.document, pageId)) return state;
	const document = clone(state.document);
	const page = findPage(document, pageId)!;
	page.layers = [...page.layers, layer];
	return { document, selectedLayerId: layer.id };
}

/** Move a layer within a page's z-order from one index to another (MC-SC-02). */
export function reorderLayer(
	state: EditorState,
	pageId: string,
	fromIndex: number,
	toIndex: number
): EditorState {
	const page = findPage(state.document, pageId);
	if (!page) return state;
	const n = page.layers.length;
	if (fromIndex < 0 || fromIndex >= n || toIndex < 0 || toIndex >= n) return state;
	// The system background never moves nor gets displaced from the bottom.
	if (page.layers[0]?.system && (fromIndex === 0 || toIndex === 0)) return state;
	const document = clone(state.document);
	const layers = findPage(document, pageId)!.layers;
	const [moved] = layers.splice(fromIndex, 1);
	layers.splice(toIndex, 0, moved);
	return { ...state, document };
}

/** Shallow-merge a patch into a layer by id (MC-SC-03). Unknown id = no-op. */
export function updateLayer(
	state: EditorState,
	layerId: string,
	patch: Partial<Layer>
): EditorState {
	if (!findLayer(state.document, layerId)) return state;
	const document = clone(state.document);
	for (const page of document.pages) {
		const layer = page.layers.find((l) => l.id === layerId);
		if (layer) {
			Object.assign(layer, patch);
			break;
		}
	}
	return { ...state, document };
}

/**
 * Remove a layer by id (the Delete key + Inspector trash call this). No-op for
 * an unknown id and for `system` layers (background etc. are not user-deletable,
 * mirroring land-canvas). Clears the selection when the deleted layer was
 * selected. Pure — clones, never mutates the input.
 */
export function deleteLayer(state: EditorState, layerId: string): EditorState {
	const target = findLayer(state.document, layerId);
	if (!target || target.system) return state;
	const document = clone(state.document);
	for (const page of document.pages) {
		const idx = page.layers.findIndex((l) => l.id === layerId);
		if (idx !== -1) {
			page.layers.splice(idx, 1);
			break;
		}
	}
	const selectedLayerId = state.selectedLayerId === layerId ? null : state.selectedLayerId;
	return { document, selectedLayerId };
}

/** Resize a page's artboard (the Resize dialog calls this). Unknown id = no-op. */
export function resizePage(
	state: EditorState,
	pageId: string,
	width: number,
	height: number
): EditorState {
	if (!findPage(state.document, pageId)) return state;
	if (!(width > 0) || !(height > 0)) return state;
	const document = clone(state.document);
	const page = findPage(document, pageId)!;
	page.width = width;
	page.height = height;
	// The background stays pinned to the artboard (land-canvas parity).
	const bg = backgroundFor(page);
	if (bg) {
		bg.width = width;
		bg.height = height;
	}
	return { ...state, document };
}

/** Set a page's frame duration in ms (the timeline's `Page duration` control —
 *  §12.9). Unknown id / non-positive / NaN = no-op. Pure (clones, never mutates). */
export function setPageDuration(state: EditorState, pageId: string, durationMs: number): EditorState {
	if (!findPage(state.document, pageId)) return state;
	if (!(durationMs > 0)) return state;
	const document = clone(state.document);
	findPage(document, pageId)!.duration = durationMs;
	return { ...state, document };
}

/** Set (or clear, with null) the selected layer. */
export function selectLayer(state: EditorState, layerId: string | null): EditorState {
	return { ...state, selectedLayerId: layerId };
}

/**
 * Patch document-level metadata — the editable `name` (title), `isTemplate` (the
 * document/template kind), `animated` (R3 animated-mode flag), and `watermark` (the
 * document-level watermark params; Slice A · #0298). Only the keys present in `patch`
 * are written; an empty/whitespace-only `name` is ignored (a document always keeps a
 * non-blank title). The `watermark` and `previewBackdrop` patches are each a PARTIAL
 * merge over the existing value (the defaults seed the base when the doc has none yet),
 * so a single knob never drops the others. Pure (clones, never mutates the input).
 *
 * `previewBackdrop` is a PREVIEW-ONLY per-doc preference (#0298) — it persists in the
 * doc JSONB but is rendered as stage chrome, never as a layer, so it is structurally
 * excluded from the export path (`resolveFrame` reads only `page.layers`).
 */
export function setDocumentMeta(
	state: EditorState,
	patch: {
		name?: string;
		isTemplate?: boolean;
		animated?: boolean;
		watermark?: Partial<WatermarkParams>;
		previewBackdrop?: Partial<PreviewBackdrop>;
	}
): EditorState {
	const document = clone(state.document);
	if (patch.name !== undefined && patch.name.trim() !== '') {
		document.name = patch.name;
	}
	if (patch.isTemplate !== undefined) {
		document.isTemplate = patch.isTemplate;
	}
	if (patch.animated !== undefined) {
		document.animated = patch.animated;
	}
	if (patch.watermark !== undefined) {
		// Seed from the defaults on first edit, then merge the partial — so a single
		// knob change never drops the other params.
		document.watermark = { ...(document.watermark ?? DEFAULT_WATERMARK_PARAMS()), ...patch.watermark };
	}
	if (patch.previewBackdrop !== undefined) {
		// Same partial-merge-over-defaults as watermark: seed `none` on first edit, then
		// merge — switching to 'color' keeps a previously picked `imageSrc` cached, and
		// vice-versa, so toggling kinds never loses the other face's value.
		document.previewBackdrop = {
			...(document.previewBackdrop ?? DEFAULT_PREVIEW_BACKDROP()),
			...patch.previewBackdrop
		};
	}
	return { ...state, document };
}

// --- Sources & data-binding (D20 / D21) ----------------------------------
// A document attaches sources (SourceRef[]) under aliases; layers carry bindings
// that read a token from an alias's snapshot. The snapshot is filled live on open
// (re-resolve) and is NOT persisted by the consumer — only refs + per-binding
// `override` are. All reducers below are pure (clone, never mutate the input).

/**
 * The deterministic alias ladder for additive source attaches. The first attach is
 * 'primary' — the Active record that drives the canvas preview + Media panel.
 */
export const SOURCE_ALIAS_LADDER = ['primary', 'secondary', 'tertiary', 'quaternary', 'quinary', 'senary'] as const;

/**
 * The next free alias given the aliases already in use — the ladder first, then a
 * COLLISION-CHECKED `source-N` fallback. Never returns an alias already present in `used`
 * (a count-based `source-${n+1}` could collide after a mid-list detach, silently overwriting
 * an attached source via attachSource's replace-by-alias). Pure.
 */
export function nextSourceAlias(used: readonly string[]): string {
	const taken = new Set(used);
	const fromLadder = SOURCE_ALIAS_LADDER.find((a) => !taken.has(a));
	if (fromLadder) return fromLadder;
	// Continue the ordinal past the named ladder (source-7, source-8, …), scanning up to the
	// lowest FREE slot — deterministic (anchored to ladder length, not the fluctuating count)
	// and gap-backfilling, so a mid-list detach reuses the freed name instead of growing N.
	let n = SOURCE_ALIAS_LADDER.length + 1;
	while (taken.has(`source-${n}`)) n++;
	return `source-${n}`;
}

/**
 * Aliases reserved for BUILT-IN sources (#0286 refinement): always-on sources the
 * consumer passes by code (e.g. Branding). Injected at the editor seam, never
 * persisted in doc.sources, never offered in the attach picker — these guards make
 * a user attach/detach unable to shadow or remove one. The alias ladder above can
 * never emit a reserved name (disjoint sets, by construction).
 */
export const BUILTIN_SOURCE_ALIASES = ['branding', 'watermark-library'] as const;

/** One of the reserved built-in source aliases. */
export type BuiltinSourceAlias = (typeof BUILTIN_SOURCE_ALIASES)[number];

/** True when an alias belongs to a built-in (code-provided, always-on) source. */
export function isBuiltinSourceAlias(alias: string): boolean {
	return (BUILTIN_SOURCE_ALIASES as readonly string[]).includes(alias);
}

/**
 * Attach a source record under its alias, or replace the existing ref with the same
 * alias (re-attaching 'primary' swaps the Active record that drives the preview). Pure.
 * Built-in ALIASES and KINDS are both refused (state returned unchanged): the alias
 * guard stops a doc attach shadowing an injected source; the kind guard stops a
 * built-in KIND becoming a persisted doc-level source under a normal alias when a
 * consumer registers the provider but forgets to inject `builtinSources` (built-ins
 * follow the kind === alias convention).
 */
export function attachSource(state: EditorState, ref: SourceRef): EditorState {
	if (isBuiltinSourceAlias(ref.alias) || isBuiltinSourceAlias(ref.kind)) return state;
	const document = clone(state.document);
	const sources = document.sources ? [...document.sources] : [];
	const idx = sources.findIndex((s) => s.alias === ref.alias);
	if (idx === -1) sources.push(ref);
	else sources[idx] = ref;
	document.sources = sources;
	return { ...state, document };
}

/**
 * Remove the attached source with this alias. Bindings that referenced it are left
 * in place — they resolve to their `fallback` (D20: fallback covers a deleted source).
 * No-op if the alias is not attached. Pure.
 */
export function detachSource(state: EditorState, alias: string): EditorState {
	if (!state.document.sources?.some((s) => s.alias === alias)) return state;
	const document = clone(state.document);
	document.sources = (document.sources ?? []).filter((s) => s.alias !== alias);
	return { ...state, document };
}

/**
 * Fill the in-memory snapshot for an attached source (live re-resolve on open). The
 * snapshot is recomputed each open, not persisted. No-op if the alias is not attached.
 */
export function setSourceSnapshot(
	state: EditorState,
	alias: string,
	snapshot: SourceSnapshot
): EditorState {
	if (!state.document.sources?.some((s) => s.alias === alias)) return state;
	const document = clone(state.document);
	const ref = (document.sources ?? []).find((s) => s.alias === alias)!;
	ref.snapshot = snapshot;
	return { ...state, document };
}

/**
 * Add or replace a layer's binding for `binding.property` (one binding per property).
 * Replacement covers both first-link and re-link (point at another source/field); the
 * UI restricts re-link to type-matched fields. Unknown layer = no-op. Pure.
 */
export function bindLayer(state: EditorState, layerId: string, binding: Binding): EditorState {
	if (!findLayer(state.document, layerId)) return state;
	const document = clone(state.document);
	const layer = findLayer(document, layerId)!;
	layer.bindings = [...(layer.bindings ?? []).filter((b) => b.property !== binding.property), binding];
	return { ...state, document };
}

/**
 * Remove a layer's binding for `property` (Unlink). When `value` is supplied it is written
 * onto the layer as the new static value so nothing visually changes — the UI passes the
 * current resolved value (which already accounts for any override). Drops the `bindings`
 * array entirely once empty. No-op if no such binding. Pure.
 */
export function unbindLayer(
	state: EditorState,
	layerId: string,
	property: string,
	value?: string
): EditorState {
	const current = findLayer(state.document, layerId);
	if (!current?.bindings?.some((b) => b.property === property)) return state;
	const document = clone(state.document);
	const layer = findLayer(document, layerId)!;
	const remaining = (layer.bindings ?? []).filter((b) => b.property !== property);
	if (remaining.length > 0) layer.bindings = remaining;
	else delete layer.bindings;
	// Flat write — the only real binding targets are top-level Layer keys (`content`, `src`).
	// A dotted `property` path is unsupported here (would create a literal "a.b" key); none exist in 4a.
	if (value !== undefined) (layer as unknown as Record<string, unknown>)[property] = value;
	return { ...state, document };
}

/** Set a manual override on a layer's binding for `property` (fine-tune). No-op if no such binding. */
export function setBindingOverride(
	state: EditorState,
	layerId: string,
	property: string,
	override: string
): EditorState {
	const current = findLayer(state.document, layerId);
	if (!current?.bindings?.some((b) => b.property === property)) return state;
	const document = clone(state.document);
	const binding = findLayer(document, layerId)!.bindings!.find((b) => b.property === property)!;
	binding.override = override;
	return { ...state, document };
}

/** Clear a binding's override (Restore → live value). No-op if no such binding / no override. */
export function clearBindingOverride(
	state: EditorState,
	layerId: string,
	property: string
): EditorState {
	const binding = findLayer(state.document, layerId)?.bindings?.find((b) => b.property === property);
	if (!binding || binding.override === undefined) return state;
	const document = clone(state.document);
	delete findLayer(document, layerId)!.bindings!.find((b) => b.property === property)!.override;
	return { ...state, document };
}

// ── Background layer (user directive 2026-06-06; land-canvas parity) ────────────
// Every page carries ONE system background layer at index 0 — `bg_mode` picks its
// face: color (shape rect fill), image (src), map (src derived from a map token).
// The flag + guards predate this (Layer.system "e.g. background"; deleteLayer
// no-ops, CanvasStage never moves/resizes system layers); these reducers add the
// creation + invariants land-canvas kept in its document store.

/** What the background should show. Image/map carry only the STATIC face — the
 *  live link (a `src` Binding) is layered on via `bindLayer`, exactly like any
 *  other image layer. */
export type BackgroundSpec =
	| { mode: 'color'; fill: FillStyle }
	| { mode: 'image' | 'map'; src?: string; mime_type?: string };

/** The canonical system background for a page — white, locked, page-sized. */
export function createBackgroundLayer(pageId: string, width: number, height: number): Layer {
	return {
		id: `bg-${pageId}`,
		type: 'shape',
		shape_type: 'rectangle',
		name: 'Background',
		system: true,
		visible: true,
		locked: true,
		x: 0,
		y: 0,
		width,
		height,
		bg_mode: 'color',
		fill: { color: '#ffffff', transparency: 0 }
	};
}

/** A page's background — the system layer (index 0 once normalized). */
export function backgroundFor(page: Page): Layer | undefined {
	return page.layers.find((l) => l.system);
}

/** Normalize every page to carry its background at index 0 — prepends a default
 *  white one when missing, moves a stray one back to the bottom. Idempotent and
 *  cheap: returns the SAME state object when nothing needs fixing (docs from the
 *  server/templates predate backgrounds; run this wherever a document enters). */
export function ensureBackgroundLayers(state: EditorState): EditorState {
	const needsFix = state.document.pages.some((p) => {
		const bg = backgroundFor(p);
		return !bg || p.layers[0] !== bg;
	});
	if (!needsFix) return state;
	const document = clone(state.document);
	for (const page of document.pages) {
		const bg = backgroundFor(page);
		if (!bg) {
			page.layers = [createBackgroundLayer(page.id, page.width, page.height), ...page.layers];
		} else if (page.layers[0] !== bg) {
			page.layers = [bg, ...page.layers.filter((l) => l !== bg)];
		}
	}
	return { ...state, document };
}

/** Swap the background's face (color | image | map). Identity + geometry are
 *  preserved; color mode strips the static src AND its binding (resolution would
 *  re-inject it otherwise); image/map set the static face and leave binding to
 *  the caller's `bindLayer` (same flow as every bound image). Unknown page or a
 *  page without a background = no-op. */
export function setBackground(state: EditorState, pageId: string, spec: BackgroundSpec): EditorState {
	const page = findPage(state.document, pageId);
	if (!page || !backgroundFor(page)) return state;
	const document = clone(state.document);
	const bg = backgroundFor(findPage(document, pageId)!)!;
	// Any FACE change drops the old src binding — a photo link under bg_mode 'map'
	// (or a map link under 'image') would make the discriminator lie. Same-mode
	// re-applies keep it (the caller is about to rebind anyway).
	const modeChanged = (bg.bg_mode ?? 'color') !== spec.mode;
	if (modeChanged) {
		const kept = bg.bindings?.filter((b) => b.property !== 'src');
		if (kept && kept.length > 0) bg.bindings = kept;
		else delete bg.bindings;
	}
	bg.bg_mode = spec.mode;
	if (spec.mode === 'color') {
		bg.type = 'shape';
		bg.shape_type = 'rectangle';
		bg.fill = spec.fill;
		delete bg.src;
		delete bg.mime_type;
	} else {
		bg.type = 'image';
		bg.src = spec.src ?? '';
		if (spec.mime_type) bg.mime_type = spec.mime_type;
	}
	return { ...state, document };
}

/** Is this background still the untouched default (white color)? Drives the
 *  replace-confirmation policy: replacing a user-set background (any image/map,
 *  or an explicitly chosen color) asks first; covering the pristine white never
 *  does. */
export function isDefaultBackground(layer: Layer): boolean {
	if (!layer.system) return false;
	if ((layer.bg_mode ?? 'color') !== 'color') return false;
	const color = (layer.fill?.color ?? '#ffffff').toLowerCase();
	return color === '#ffffff' || color === '#fff';
}

/** Find a page by id (returns undefined if absent). */
export function findPage(doc: Document, pageId: string): Page | undefined {
	return doc.pages.find((p) => p.id === pageId);
}

function findLayer(doc: Document, layerId: string): Layer | undefined {
	for (const page of doc.pages) {
		const layer = page.layers.find((l) => l.id === layerId);
		if (layer) return layer;
	}
	return undefined;
}

/** Deep clone — the document model is pure JSON (no functions/dates/blobs-as-objects). */
function clone(doc: Document): Document {
	return JSON.parse(JSON.stringify(doc)) as Document;
}

function uuid(): string {
	return crypto.randomUUID();
}
