// TDD RED — frame export pipeline (FDD4.PD.TEST_RED).
// Scenarios: MC-SC-13 (export a frame at time t = on-canvas render with resolved
//            bindings + animation; hidden layers excluded),
//            MC-SC-14 (export waits for fonts before rasterizing).
//
// CASE ENUMERATION
//   TC-EX-01 success  resolveFrame applies binding + animation at t (MC-SC-13)
//   TC-EX-02 regress  resolveFrame excludes hidden layers (MC-SC-13)
//   TC-EX-03 edge     resolveFrame at an exact keyframe time uses that value
//   TC-EX-04 success  exportPage awaits fontsReady BEFORE rasterizing (MC-SC-14)
//   TC-EX-05 success  exportPage rasterizes the resolveFrame output (MC-SC-13)

import { describe, expect, it, vi } from 'vitest';
import { exportMarkUnit, exportPage, resolveFrame } from './export.js';
import type { Rasterizer } from './export.js';
import type { Page, SourceRef } from './types/document.js';
import type { Layer } from './types/layer.js';
import { makeImageLayer, makeTextLayer, testFormatters } from './fixtures.js';
import type { MapStaticUrlBuilder } from './map-source.js';
import { projectToPixel } from './geometry.js';

const sources: SourceRef[] = [
	{ id: 's1', kind: 'property', refId: 'r1', alias: 'primary', snapshot: { price: 28_500_000 } }
];

const animatedBoundPage = (): Page => ({
	id: 'p1',
	width: 1080,
	height: 1350,
	layers: [
		{
			...makeTextLayer({ id: 'price', opacity: 0 }),
			bindings: [{ property: 'content', sourceAlias: 'primary', token: 'price', format: 'currency:THB' }],
			animations: [
				{ property: 'opacity', keyframes: [
					{ time: 0, value: 0, easing: 'linear' },
					{ time: 1000, value: 1, easing: 'linear' }
				] }
			]
		}
	]
});

describe('resolveFrame (MC-SC-13)', () => {
	it('TC-EX-01 applies binding then animation at time t', () => {
		const out = resolveFrame(animatedBoundPage(), 500, sources, testFormatters);
		expect(out[0].content).toBe('฿28,500,000');
		expect(out[0].opacity).toBeCloseTo(0.5, 5);
	});

	it('TC-EX-06 resolves a content_template against the primary source (INC-C)', () => {
		const page: Page = {
			id: 'p1', width: 100, height: 100,
			layers: [
				makeTextLayer({
					id: 'tpl',
					content_template: [
						{ type: 'text', text: 'From ' },
						{ type: 'token', token: 'price', format: 'currency:THB' }
					]
				})
			]
		};
		const out = resolveFrame(page, 0, sources, testFormatters);
		expect(out[0].content).toBe('From ฿28,500,000');
	});

	it('TC-EX-02 excludes hidden layers from the frame', () => {
		const page: Page = {
			id: 'p1', width: 1, height: 1,
			layers: [makeImageLayer({ id: 'shown', visible: true }), makeImageLayer({ id: 'hidden', visible: false })]
		};
		const out = resolveFrame(page, 0);
		expect(out.map((l) => l.id)).toEqual(['shown']);
	});

	it('TC-EX-03 uses the exact keyframe value at an exact keyframe time', () => {
		const out = resolveFrame(animatedBoundPage(), 1000, sources, testFormatters);
		expect(out[0].opacity).toBe(1);
	});

	it('TC-EX-06 animate=false keeps bindings but ignores tracks — renders the layer BASE, not the t=0 keyframe', () => {
		const page: Page = {
			id: 'p1',
			width: 1080,
			height: 1350,
			layers: [
				{
					...makeTextLayer({ id: 'price', opacity: 1 }), // base opacity = 1
					bindings: [{ property: 'content', sourceAlias: 'primary', token: 'price', format: 'currency:THB' }],
					animations: [
						{ property: 'opacity', keyframes: [
							{ time: 0, value: 0, easing: 'linear' },
							{ time: 1000, value: 1, easing: 'linear' }
						] }
					]
				}
			]
		};
		// motion at t=0 → the t=0 keyframe (opacity 0)
		expect(resolveFrame(page, 0, sources, testFormatters, true)[0].opacity).toBe(0);
		// static (animate=false) → the layer's BASE opacity (1), track ignored; bindings still applied
		const staticOut = resolveFrame(page, 0, sources, testFormatters, false);
		expect(staticOut[0].opacity).toBe(1);
		expect(staticOut[0].content).toBe('฿28,500,000');
	});
});

// ── Map src resolution via buildMapUrl ────────────────────────────────────────

describe('resolveFrame — map layer src resolution (TC-EX-MAP)', () => {
	const MAP_ID = 'map-layer-1';

	function makeMapLayer(overrides: Partial<Layer> = {}): Layer {
		return {
			id: MAP_ID,
			type: 'map',
			name: 'Map',
			visible: true,
			locked: false,
			x: 0, y: 0,
			width: 640,
			height: 400,
			map_config: {
				center: { lat: 13.7563, lng: 100.5018 },
				zoom: 14,
				mapType: 'roadmap'
			},
			opacity: 1,
			...overrides
		};
	}

	function makeAnchoredMarker(): Layer {
		return {
			id: 'marker-1',
			type: 'marker',
			name: 'Pin',
			visible: true,
			locked: false,
			x: 0, y: 0,
			map_surface_id: MAP_ID,
			geo_ref: { lat: 13.7563, lng: 100.5018, anchor_lats: [], anchor_lngs: [] },
			stroke: { color: '#e5392b', thickness: 2, dash_type: 'solid', glow_intensity: 0 },
			opacity: 1
		};
	}

	function makeAnchoredOutline(): Layer {
		return {
			id: 'outline-1',
			type: 'outline',
			name: 'Region',
			visible: true,
			locked: false,
			x: 0, y: 0,
			map_surface_id: MAP_ID,
			coord_system: 'map',
			closed: true,
			anchors: [
				{ id: 'a1', x: 100.5, y: 13.75 },
				{ id: 'a2', x: 100.6, y: 13.75 },
				{ id: 'a3', x: 100.6, y: 13.85 }
			],
			stroke: { color: '#0d4f4f', thickness: 3, dash_type: 'solid', glow_intensity: 0 },
			opacity: 1
		};
	}

	it('TC-EX-MAP-01 resolveFrame sets src on map layer when buildMapUrl provided', () => {
		const buildMapUrl: MapStaticUrlBuilder = vi.fn(() => '/api/map/static?stub=1');
		const page: Page = {
			id: 'p1', width: 640, height: 400,
			layers: [makeMapLayer()]
		};
		const out = resolveFrame(page, 0, [], {}, true, buildMapUrl);
		expect(out[0].type).toBe('map');
		expect(out[0].src).toBe('/api/map/static?stub=1');
		expect(buildMapUrl).toHaveBeenCalledOnce();
	});

	it('TC-EX-MAP-02 anchored marker + outline passed to buildMapUrl (baked annotations)', () => {
		const calls: Parameters<MapStaticUrlBuilder>[] = [];
		const buildMapUrl: MapStaticUrlBuilder = vi.fn((desc) => {
			calls.push([desc]);
			return '/api/map/static?baked=1';
		});
		const marker = makeAnchoredMarker();
		const outline = makeAnchoredOutline();
		const page: Page = {
			id: 'p1', width: 640, height: 400,
			layers: [makeMapLayer(), marker, outline]
		};
		resolveFrame(page, 0, [], {}, true, buildMapUrl);
		// The builder is called once, for the map layer.
		expect(calls).toHaveLength(1);
		// CONTRACT: the descriptor must actually carry the baked annotations — assert the
		// content, not just that the builder was called (a no-annotation descriptor would
		// otherwise pass silently and the bake would be a no-op on export).
		const desc = calls[0][0];
		expect(desc.markers).toHaveLength(1);
		expect(desc.markers?.[0]).toMatchObject({ lat: 13.7563, lng: 100.5018, color: '#e5392b' });
		expect(desc.polygons).toHaveLength(1);
		// 3 anchors auto-closed → a 4-position ring.
		expect(desc.polygons?.[0].ring).toHaveLength(4);
	});

	it('TC-EX-MAP-03 map layer src is NOT set when buildMapUrl is omitted', () => {
		const page: Page = {
			id: 'p1', width: 640, height: 400,
			layers: [makeMapLayer()]
		};
		const out = resolveFrame(page, 0, [], {}, true);
		expect(out[0].src).toBeUndefined();
	});

	it('TC-EX-MAP-04 export snapshot: map layer + pin + region → exportPage bakes all', async () => {
		// End-to-end snapshot: the export pipeline resolves map src + bakes annotations.
		const rasterizeCalls: Layer[][] = [];
		const rasterize: Rasterizer = async (layers) => {
			rasterizeCalls.push([...layers]);
			return new Blob(['x'], { type: 'image/png' });
		};
		const buildMapUrl: MapStaticUrlBuilder = vi.fn(() => '/api/map/static?markers=1&path=1');
		const page: Page = {
			id: 'p1', width: 640, height: 400,
			layers: [makeMapLayer(), makeAnchoredMarker(), makeAnchoredOutline()]
		};
		await exportPage(page, { format: 'png', width: 640, height: 400, retina: false }, {
			rasterize,
			buildMapUrl
		});
		// The rasterizer received all 3 layers (map + marker + outline) after resolution.
		expect(rasterizeCalls[0]).toHaveLength(3);
		// The map layer has a resolved src (the baked static-map URL).
		const mapOut = rasterizeCalls[0].find((l) => l.type === 'map');
		expect(mapOut?.src).toBe('/api/map/static?markers=1&path=1');
		// The marker + outline src are untouched.
		expect(buildMapUrl).toHaveBeenCalledOnce();
	});

	it('TC-EX-MAP-05 anchored callout is projected onto the PARENT map size (not its own box)', () => {
		// REGRESSION (T4 reviewer Finding 1): a callout label can't bake into the static
		// image, so resolveFrame projects geo_ref → canvas pixels. It MUST use the MAP
		// layer's real size (640×400), never the callout's text-box size (160×40). An
		// off-centre geo_ref makes the two diverge, so a wrong size is caught. No buildMapUrl
		// is passed → proves projection is independent of map-src baking.
		const callout: Layer = {
			id: 'callout-1',
			type: 'callout',
			name: 'Label',
			visible: true,
			locked: false,
			x: 0, y: 0,
			width: 160, height: 40,
			content: 'Sea view',
			map_surface_id: MAP_ID,
			geo_ref: { lat: 13.78, lng: 100.55, anchor_lats: [], anchor_lngs: [] }, // OFF-centre
			opacity: 1
		};
		const map = makeMapLayer({ x: 50, y: 20 }); // non-zero origin exercises the offset
		const page: Page = { id: 'p1', width: 640, height: 400, layers: [map, callout] };
		const out = resolveFrame(page, 0, [], {}, true);
		const resolvedCallout = out.find((l) => l.id === 'callout-1')!;
		// Expected = map origin + projectToPixel(geo, map center/zoom, MAP size), centred on the box.
		const px = projectToPixel(13.78, 100.55, map.map_config!.center, map.map_config!.zoom, { width: 640, height: 400 });
		expect(resolvedCallout.x).toBeCloseTo(50 + px.x - 80, 5);
		expect(resolvedCallout.y).toBeCloseTo(20 + px.y - 20, 5);
		// It must NOT equal the buggy callout-box (160×40) projection — they diverge by mapW/2 - boxW/2.
		const pxBug = projectToPixel(13.78, 100.55, map.map_config!.center, map.map_config!.zoom, { width: 160, height: 40 });
		expect(resolvedCallout.x).not.toBeCloseTo(50 + pxBug.x - 80, 1);
	});

	it('TC-EX-MAP-06 free-floating callout (no map_surface_id) is left at its own x/y', () => {
		const callout: Layer = {
			id: 'callout-free',
			type: 'callout',
			name: 'Label',
			visible: true,
			locked: false,
			x: 123, y: 45,
			width: 160, height: 40,
			content: 'Free',
			geo_ref: { lat: 13.78, lng: 100.55, anchor_lats: [], anchor_lngs: [] },
			opacity: 1
		};
		const page: Page = { id: 'p1', width: 640, height: 400, layers: [makeMapLayer(), callout] };
		const out = resolveFrame(page, 0, [], {}, true);
		const free = out.find((l) => l.id === 'callout-free')!;
		expect(free.x).toBe(123);
		expect(free.y).toBe(45);
	});
});

describe('exportPage (MC-SC-14 / MC-SC-13)', () => {
	const target = { format: 'png' as const, width: 1080, height: 1350, retina: false };

	it('TC-EX-04 awaits fontsReady before rasterizing', async () => {
		let fontsResolved = false;
		let rasterizedAfterFonts: boolean | null = null;
		const fontsReady = new Promise<void>((resolve) =>
			setTimeout(() => {
				fontsResolved = true;
				resolve();
			}, 5)
		);
		const rasterize: Rasterizer = async () => {
			rasterizedAfterFonts = fontsResolved;
			return new Blob([], { type: 'image/png' });
		};
		await exportPage(animatedBoundPage(), target, { t: 500, sources, formatters: testFormatters, fontsReady, rasterize });
		expect(rasterizedAfterFonts).toBe(true);
	});

	it('TC-EX-05 rasterizes the resolveFrame output and returns its Blob', async () => {
		let received: number | null = null;
		const rasterize: Rasterizer = async (layers) => {
			received = layers.length;
			return new Blob(['x'], { type: 'image/png' });
		};
		const blob = await exportPage(animatedBoundPage(), target, {
			t: 500, sources, formatters: testFormatters, fontsReady: Promise.resolve(), rasterize
		});
		expect(received).toBe(1);
		expect(blob.type).toBe('image/png');
	});
});

// ── Preview-backdrop EXCLUSION proof (Slice A2.3 · #0298, the backdrop-leakage threat) ──
// The preview backdrop is a DOCUMENT-level preview preference rendered as STAGE CHROME.
// The export path operates on a PAGE (`exportPage(page, …)` → `resolveFrame(page, …)`)
// and only ever reads `page.layers` — the document's `previewBackdrop` is structurally
// unreachable from here. These tests pin that contract so a future change that tried to
// route the backdrop into the export would fail loudly.
describe('preview backdrop is EXCLUDED from the export path (#0298)', () => {
	const target = { format: 'png' as const, width: 100, height: 100, retina: false };

	it('TC-EX-PB-01 resolveFrame returns ONLY the page layers — no backdrop layer is injected', () => {
		const page: Page = {
			id: 'p1', width: 100, height: 100,
			layers: [makeImageLayer({ id: 'mark', visible: true })]
		};
		// A document carrying a non-none preview backdrop alongside this page (the watermark
		// authoring state) — the doc value is intentionally NOT an argument to resolveFrame.
		const out = resolveFrame(page, 0);
		expect(out.map((l) => l.id)).toEqual(['mark']);
		// No synthetic backdrop layer (e.g. a full-bleed colour rect) leaked into the frame.
		expect(out.some((l) => l.id.includes('backdrop'))).toBe(false);
	});

	it('TC-EX-PB-02 exportPage rasterizes only the page layers (the rasterizer never sees a backdrop)', async () => {
		const seen: Layer[][] = [];
		const rasterize: Rasterizer = async (layers) => {
			seen.push([...layers]);
			return new Blob(['x'], { type: 'image/png' });
		};
		const page: Page = {
			id: 'p1', width: 100, height: 100,
			layers: [makeImageLayer({ id: 'mark', visible: true })]
		};
		await exportPage(page, target, { rasterize });
		// Exactly the page's own layer reached the rasterizer — the backdrop, being stage
		// chrome and not part of `page.layers`, is structurally absent from the export blob.
		expect(seen).toHaveLength(1);
		expect(seen[0].map((l) => l.id)).toEqual(['mark']);
	});
});

// ── exportMarkUnit — the #0299 MARK UNIT crop ─────────────────────────────────
// The apply pipeline re-tiles the mark's content bbox at the photo's pixels (never
// the full-frame overlay). exportMarkUnit resolves the frame, crops to markBounds,
// and rasterizes a transparent sub-page sized to the bbox with layers translated to
// a bbox-relative origin. Source of truth for the publish-time mark-unit render.
//
//   TC-EX-MU-01 success  crops to the mark bbox; layers translated to (0,0)-origin
//   TC-EX-MU-02 regress  does NOT mutate the input page's layers
//   TC-EX-MU-03 edge     blank mark (no visible non-system layers) → null, no raster
describe('exportMarkUnit (#0299 mark unit)', () => {
	// Two image layers whose union bbox is {x:100, y:50, w:200, h:80}: layer A spans
	// 100..300 × 50..130; layer B (120..220 × 60..100) is fully inside it.
	const markPage = (): Page => ({
		id: 'p1', width: 1080, height: 1350,
		layers: [
			makeImageLayer({ id: 'A', x: 100, y: 50, width: 200, height: 80 }),
			makeImageLayer({ id: 'B', x: 120, y: 60, width: 100, height: 40 })
		]
	});

	it('TC-EX-MU-01 crops to the mark bbox and translates layers to a bbox-relative origin', async () => {
		let seenLayers: Layer[] = [];
		let seenPage: Page | null = null;
		let seenTarget: { width: number; height: number; format: string; retina?: boolean } | null = null;
		const rasterize: Rasterizer = async (layers, page, target) => {
			seenLayers = layers;
			seenPage = page;
			seenTarget = target;
			return new Blob(['x'], { type: 'image/png' });
		};

		const blob = await exportMarkUnit(markPage(), { rasterize });

		expect(blob).not.toBeNull();
		// Sub-page + target sized to the bbox (200×80), retina default true, PNG.
		expect(seenTarget).toMatchObject({ width: 200, height: 80, format: 'png', retina: true });
		expect(seenPage!.width).toBe(200);
		expect(seenPage!.height).toBe(80);
		// Layers translated by −bounds.x/−bounds.y: A→(0,0), B→(20,10).
		const byId = Object.fromEntries(seenLayers.map((l) => [l.id, l]));
		expect(byId.A).toMatchObject({ x: 0, y: 0 });
		expect(byId.B).toMatchObject({ x: 20, y: 10 });
	});

	it('TC-EX-MU-02 does not mutate the input page layers', async () => {
		const page = markPage();
		const rasterize: Rasterizer = async () => new Blob(['x'], { type: 'image/png' });
		await exportMarkUnit(page, { rasterize });
		// Originals keep their absolute coordinates — the translation built NEW objects.
		expect(page.layers[0]).toMatchObject({ x: 100, y: 50 });
		expect(page.layers[1]).toMatchObject({ x: 120, y: 60 });
	});

	it('TC-EX-MU-03 returns null without rasterizing when the mark is empty', async () => {
		const rasterize = vi.fn(async () => new Blob(['x'], { type: 'image/png' }));
		const blank: Page = {
			id: 'p1', width: 100, height: 100,
			layers: [makeImageLayer({ id: 'hidden', visible: false })]
		};
		const blob = await exportMarkUnit(blank, { rasterize });
		expect(blob).toBeNull();
		expect(rasterize).not.toHaveBeenCalled();
	});
});
