// Frame export — Media Canvas (D11 / slice 6). Two seams:
//   resolveFrame() — PURE: collapse a page's visible layers at time t with
//                    bindings + animation applied (BASE → BINDING → ANIMATION@t).
//                    Also resolves map layer `src` via the injected MapSrcResolver
//                    so the static-map URL is baked before rasterization.
//                    Canvas-free → unit-testable. This is "what the export shows".
//   exportPage()   — orchestrate: resolve the frame, wait for fonts, then hand the
//                    resolved layers to a Rasterizer. The kit is HEADLESS and cannot
//                    touch a canvas, so the rasterizer is INJECTED by the UI layer
//                    (@sbx/canvas-ui owns the renderer) — modular-architecture-first.

import type { Page, SourceRef } from './types/document.js';
import type { Layer } from './types/layer.js';
import type { ExportTarget } from './types/template.js';
import type { FormatterRegistry, SourceSnapshot } from './types/source.js';
import { resolveBindings, resolveLayerAtTime, resolveTemplateContent } from './resolve.js';
import type { MapStaticUrlBuilder } from './map-source.js';
import { mapLayerStaticUrl } from './map-source.js';
import { projectToPixel } from './geometry.js';
import { markBounds } from './watermark-tiling.js';

/**
 * Fully-resolved visible layers of a page at time `t` — bindings substituted from
 * the attached sources (per layer binding's `sourceAlias`), then (when `animate`)
 * animation interpolated. Hidden layers excluded (never exported — MC-SC-13).
 *
 * When `buildMapUrl` is supplied, type:'map' layers have their `src` set to the
 * resolved /api/map/static URL (baking center/zoom/mapType + anchored markers and
 * outlines). The UI then loads that URL into the ImageSource cache before the
 * rasterizer draws. Without `buildMapUrl`, map layers are resolved without `src`.
 *
 * Map-anchored callouts (type:'callout' + map_surface_id + geo_ref) are then projected
 * onto their PARENT map layer's pixel space (Web Mercator via the map's real
 * center/zoom/size) — their resolved x/y is the on-canvas position over the map. This
 * runs regardless of `buildMapUrl` (it needs only the parent map's map_config + size).
 *
 * `animate=false` → STATIC render (R3 animated-mode OFF): each layer keeps its base
 * values, ignoring its animation tracks entirely — NOT the t=0 keyframe (a fade-in's
 * t=0 value is invisible, which is not the intended static look).
 */
export function resolveFrame(
	page: Page,
	t: number,
	sources: SourceRef[] = [],
	formatters: FormatterRegistry = {},
	animate: boolean = true,
	buildMapUrl?: MapStaticUrlBuilder
): Layer[] {
	const byAlias = new Map<string, SourceSnapshot>(sources.map((s) => [s.alias, s.snapshot ?? {}]));
	const allVisible = page.layers.filter((l) => l.visible !== false);
	// Pass 1 — resolve bindings + animation; bake each map layer's static-map src.
	const resolved = allVisible.map((l) => {
		let r = resolveLayerBindings(l, byAlias, formatters);
		if (animate) r = resolveLayerAtTime(r, t);
		// Map layer src resolution — bake the static-map URL (center+annotations).
		if (r.type === 'map' && buildMapUrl) {
			const annotations = allVisible.filter((a) => a.map_surface_id === r.id);
			const w = r.width ?? 0;
			const h = r.height ?? 0;
			const src = mapLayerStaticUrl(r, annotations, { width: w, height: h }, buildMapUrl);
			if (src) r = { ...r, src };
		}
		return r;
	});
	// Pass 2 — project map-anchored callouts onto their PARENT map's pixel space.
	// A callout's text label can't be baked into the static image (Static Maps renders
	// only pins/paths), so it draws on the canvas OVER the map. Look up the parent map
	// by map_surface_id and project geo_ref via the map's REAL center/zoom/SIZE — never
	// the callout's own box (using the callout box was the T4 reviewer's CRITICAL bug:
	// wrong pixel for any off-centre point). The callout carries NO denormalized copy of
	// the map config/size — the parent map layer is the single source of truth, so a
	// map pan/zoom/resize re-projects every anchored callout on the next resolveFrame.
	const mapsById = new Map<string, Layer>(
		resolved.filter((l) => l.type === 'map').map((m) => [m.id, m])
	);
	return resolved.map((l) => {
		if (l.type !== 'callout' || !l.map_surface_id || !l.geo_ref) return l;
		const map = mapsById.get(l.map_surface_id);
		if (!map?.map_config || !map.width || !map.height) return l;
		const px = projectToPixel(
			l.geo_ref.lat,
			l.geo_ref.lng,
			map.map_config.center,
			map.map_config.zoom,
			{ width: map.width, height: map.height }
		);
		// Absolute artboard position = map origin + projected pixel, centred on the text box.
		const x = map.x + px.x - (l.width ?? 0) / 2;
		const y = map.y + px.y - (l.height ?? 0) / 2;
		return { ...l, x, y };
	});
}

/** Resolve a layer's bindings (each routed to its source alias's snapshot) AND its content
 *  template (INC-C). Bindings group by alias; the template resolves CROSS-source — each token
 *  routes to its own source by an optional '<alias>.' prefix (bare tokens → 'primary'). */
function resolveLayerBindings(
	layer: Layer,
	byAlias: Map<string, SourceSnapshot>,
	formatters: FormatterRegistry
): Layer {
	const hasTemplate = !!layer.content_template?.length;
	if (!layer.bindings?.length && !hasTemplate) return layer;
	let out = layer;
	// Bindings, grouped by alias — the template is stripped here so it isn't resolved per-alias.
	const aliases = [...new Set((layer.bindings ?? []).map((b) => b.sourceAlias))];
	for (const alias of aliases) {
		const snapshot = byAlias.get(alias) ?? {};
		const subset: Layer = {
			...out,
			bindings: out.bindings!.filter((b) => b.sourceAlias === alias),
			content_template: undefined
		};
		out = resolveBindings(subset, snapshot, formatters);
	}
	// Template content → resolved CROSS-source: each token routes to its own source via an
	// optional '<alias>.' prefix (bare/legacy tokens → 'primary'), so one template can mix
	// sources (e.g. '[primary.title] — [branding.title]'). The binding loop stripped the
	// template, so it runs once here over all snapshots.
	if (hasTemplate) {
		out = { ...out, content: resolveTemplateContent(layer.content_template!, byAlias, formatters, 'primary') };
	}
	return { ...out, bindings: layer.bindings };
}

/** Turns resolved layers into a raster Blob; supplied by the UI layer's canvas renderer. */
export type Rasterizer = (layers: Layer[], page: Page, target: ExportTarget) => Promise<Blob>;

export interface ExportOptions {
	/** Playhead time (ms) for animated frames; default 0. */
	t?: number;
	/** Apply animation tracks at `t` (default true); false → static base-value render
	 *  (R3 animated-mode OFF — exports the document's static look, not the t=0 keyframe). */
	animate?: boolean;
	sources?: SourceRef[];
	formatters?: FormatterRegistry;
	/** Awaited before rasterizing so brand fonts render (MC-SC-14); e.g. document.fonts.ready. */
	fontsReady?: Promise<void>;
	/** Required — the headless kit cannot rasterize; @sbx/canvas-ui injects the canvas renderer. */
	rasterize: Rasterizer;
	/**
	 * Optional — injected static-map URL builder (wired to buildStaticMapPath from @sbx/ui-map
	 * by the UI layer). When supplied, type:'map' layers have their `src` resolved to the
	 * keyless /api/map/static URL (baking center/zoom/mapType + anchored annotations) before
	 * the rasterizer draws them. Without this, map layers export without a src (blank/placeholder).
	 */
	buildMapUrl?: MapStaticUrlBuilder;
}

/**
 * Export a page to a Blob (PNG/JPEG/PDF) at a chosen time. Resolves the frame, then
 * awaits `fontsReady` BEFORE rasterizing so text renders with the correct font
 * (MC-SC-13 / MC-SC-14).
 */
export async function exportPage(
	page: Page,
	target: ExportTarget,
	opts: ExportOptions
): Promise<Blob> {
	const { t = 0, animate = true, sources = [], formatters = {}, fontsReady, rasterize, buildMapUrl } = opts;
	const layers = resolveFrame(page, t, sources, formatters, animate, buildMapUrl);
	if (fontsReady) await fontsReady;
	return rasterize(layers, page, target);
}

/**
 * Export the watermark MARK UNIT — the CROPPED, transparent PNG of the mark's content
 * bbox (#0299), rendered at PUBLISH time so the headless Go apply worker (which can't run
 * canvas-kit) re-tiles a full-fidelity tile at each photo's exact pixels (Slice C). Resolves
 * the frame statically (`animate:false` — a watermark is static), computes `markBounds` over
 * the resolved layers, and returns null when the mark is empty (a blank overlay). Otherwise it
 * builds a sub-page sized to the bbox with every resolved layer translated by −bounds.x/−bounds.y
 * (new objects — the originals are never mutated) and rasterizes it as a transparent PNG at the
 * bbox dimensions. Mirrors exportPage's structure (resolve → await fonts → rasterize) so the
 * mark-unit logic stays in the headless kit (testable).
 */
export async function exportMarkUnit(
	page: Page,
	opts: ExportOptions & { retina?: boolean }
): Promise<Blob | null> {
	const { t = 0, sources = [], formatters = {}, fontsReady, rasterize, buildMapUrl, retina } = opts;
	// A watermark is static — resolve the base frame (animate:false), never the t=0 keyframe.
	const layers = resolveFrame(page, t, sources, formatters, false, buildMapUrl);
	const bounds = markBounds(layers, page.width, page.height);
	if (!bounds) return null; // empty mark → nothing to publish (the full-frame overlay is blank).
	// CROPPED sub-page: the mark bbox becomes the page, each layer translated into its space.
	// Map to NEW objects so the caller's resolved layers (and the full-frame export) are untouched.
	const cropped = layers.map((l) => ({ ...l, x: l.x - bounds.x, y: l.y - bounds.y }));
	const subPage: Page = { ...page, width: bounds.w, height: bounds.h, layers: cropped };
	const target: ExportTarget = {
		format: 'png',
		width: bounds.w,
		height: bounds.h,
		retina: retina ?? true
	};
	if (fontsReady) await fontsReady;
	return rasterize(cropped, subPage, target);
}
