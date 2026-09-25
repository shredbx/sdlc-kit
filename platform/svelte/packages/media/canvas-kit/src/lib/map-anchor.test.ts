// TDD — map-anchor (annotation → map surface anchoring, Decision #0297 slice 1B).
//
// CASE ENUMERATION
//   TC-AN-01 success  mapLayerAt returns the TOPMOST map containing the point
//   TC-AN-02 edge     mapLayerAt returns null outside any map box
//   TC-AN-03 regress  mapLayerAt skips non-'map', system, and hidden layers
//   TC-AN-04 success  anchor marker → map_surface_id + coord_system + geo_ref;
//                     ROUND-TRIP projectToPixel ≈ local drop pixel
//   TC-AN-05 success  anchor outline → 4 anchors, closed, coord_system 'map';
//                     each anchor projects back to ~its corner pixel
//   TC-AN-06 success  anchor callout → geo_ref + map_surface_id set
//   TC-AN-07 regress  non-annotation (image/text) returned unchanged (===)
//   TC-AN-08 edge     map without map_config → annotation returned unchanged (===)

import { describe, expect, it } from 'vitest';
import { mapLayerAt, anchorAnnotationToMap, resolveAnchorTarget } from './map-anchor.js';
import { projectToPixel } from './geometry.js';
import type { Layer } from './types/layer.js';

const MAP_ID = 'map-1';

function makeMapLayer(overrides: Partial<Layer> = {}): Layer {
	return {
		id: MAP_ID,
		type: 'map',
		name: 'Map',
		visible: true,
		locked: false,
		x: 100,
		y: 50,
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

function makeMarkerLayer(overrides: Partial<Layer> = {}): Layer {
	return {
		id: 'marker-1',
		type: 'marker',
		name: 'Marker',
		visible: true,
		locked: false,
		x: 0,
		y: 0,
		width: 32,
		height: 40,
		coord_system: 'map',
		opacity: 1,
		...overrides
	};
}

function makeOutlineLayer(overrides: Partial<Layer> = {}): Layer {
	return {
		id: 'outline-1',
		type: 'outline',
		name: 'Region',
		visible: true,
		locked: false,
		x: 0,
		y: 0,
		width: 200,
		height: 200,
		anchors: [],
		closed: true,
		coord_system: 'map',
		opacity: 1,
		...overrides
	};
}

function makeCalloutLayer(overrides: Partial<Layer> = {}): Layer {
	return {
		id: 'callout-1',
		type: 'callout',
		name: 'Callout',
		visible: true,
		locked: false,
		x: 0,
		y: 0,
		width: 160,
		height: 40,
		content: 'Label',
		coord_system: 'map',
		opacity: 1,
		...overrides
	};
}

describe('mapLayerAt', () => {
	it('TC-AN-01 returns the topmost map containing the point', () => {
		const lower = makeMapLayer({ id: 'map-lower' });
		const upper = makeMapLayer({ id: 'map-upper' });
		// z-order back→front: a point inside BOTH boxes resolves to the topmost (last).
		const hit = mapLayerAt([lower, upper], 420, 250);
		expect(hit?.id).toBe('map-upper');
	});

	it('TC-AN-02 returns null outside any map box', () => {
		expect(mapLayerAt([makeMapLayer()], 5, 5)).toBeNull();
		// Just past the bottom-right corner (100+640, 50+400) → outside.
		expect(mapLayerAt([makeMapLayer()], 741, 451)).toBeNull();
	});

	it('TC-AN-03 skips non-map, system, and hidden layers', () => {
		const notMap = makeMarkerLayer({ x: 100, y: 50, width: 640, height: 400 });
		const systemMap = makeMapLayer({ id: 'sys', system: true });
		const hiddenMap = makeMapLayer({ id: 'hidden', visible: false });
		// All three cover the point but none qualify → null.
		expect(mapLayerAt([notMap, systemMap, hiddenMap], 420, 250)).toBeNull();
		// Add a real visible map → it wins.
		const realMap = makeMapLayer({ id: 'real' });
		expect(mapLayerAt([systemMap, hiddenMap, realMap], 420, 250)?.id).toBe('real');
	});
});

describe('anchorAnnotationToMap', () => {
	const map = makeMapLayer();
	const size = { width: map.width!, height: map.height! };
	const cfg = map.map_config!;
	// An artboard drop point inside the map box.
	const dropX = 300;
	const dropY = 200;
	const localX = dropX - map.x!;
	const localY = dropY - map.y!;

	it('TC-AN-04 marker: sets surface link + geo_ref; round-trips to the drop pixel', () => {
		const out = anchorAnnotationToMap(makeMarkerLayer(), dropX, dropY, map);
		expect(out.map_surface_id).toBe(MAP_ID);
		expect(out.coord_system).toBe('map');
		expect(out.geo_ref).toBeDefined();
		const back = projectToPixel(out.geo_ref!.lat, out.geo_ref!.lng, cfg.center, cfg.zoom, size);
		expect(back.x).toBeCloseTo(localX, 1);
		expect(back.y).toBeCloseTo(localY, 1);
		expect(out.geo_ref!.anchor_lats).toEqual([]);
		expect(out.geo_ref!.anchor_lngs).toEqual([]);
	});

	it('TC-AN-05 outline: 4 closed map-coord anchors span the layer BOX, each projecting back to its corner', () => {
		const out = anchorAnnotationToMap(makeOutlineLayer(), dropX, dropY, map);
		expect(out.map_surface_id).toBe(MAP_ID);
		expect(out.coord_system).toBe('map');
		expect(out.closed).toBe(true);
		expect(out.anchors).toHaveLength(4);
		// The ring spans the layer's box (makeOutlineLayer is 200×200 → ±100 around the drop).
		const HALF_W = 200 / 2;
		const HALF_H = 200 / 2;
		const corners = [
			{ x: localX - HALF_W, y: localY - HALF_H },
			{ x: localX + HALF_W, y: localY - HALF_H },
			{ x: localX + HALF_W, y: localY + HALF_H },
			{ x: localX - HALF_W, y: localY + HALF_H }
		];
		out.anchors!.forEach((a, i) => {
			// Map-coord anchors store x = lng, y = lat.
			const back = projectToPixel(a.y, a.x, cfg.center, cfg.zoom, size);
			expect(back.x).toBeCloseTo(corners[i].x, 1);
			expect(back.y).toBeCloseTo(corners[i].y, 1);
			expect(typeof a.id).toBe('string');
			expect(a.id.length).toBeGreaterThan(0);
		});
	});

	it('TC-AN-05b outline: a DRAWN box size (not the default) flows into the geo ring', () => {
		// Drawing a 400×300 region must produce a ring spanning ±200 / ±150 — not a fixed square.
		const out = anchorAnnotationToMap(makeOutlineLayer({ width: 400, height: 300 }), dropX, dropY, map);
		const corners = [
			{ x: localX - 200, y: localY - 150 },
			{ x: localX + 200, y: localY - 150 },
			{ x: localX + 200, y: localY + 150 },
			{ x: localX - 200, y: localY + 150 }
		];
		out.anchors!.forEach((a, i) => {
			const back = projectToPixel(a.y, a.x, cfg.center, cfg.zoom, size);
			expect(back.x).toBeCloseTo(corners[i].x, 1);
			expect(back.y).toBeCloseTo(corners[i].y, 1);
		});
	});

	it('TC-AN-06 callout: sets geo_ref + map_surface_id', () => {
		const out = anchorAnnotationToMap(makeCalloutLayer(), dropX, dropY, map);
		expect(out.map_surface_id).toBe(MAP_ID);
		expect(out.coord_system).toBe('map');
		expect(out.geo_ref).toBeDefined();
		const back = projectToPixel(out.geo_ref!.lat, out.geo_ref!.lng, cfg.center, cfg.zoom, size);
		expect(back.x).toBeCloseTo(localX, 1);
		expect(back.y).toBeCloseTo(localY, 1);
	});

	it('TC-AN-07 non-annotation layers are returned unchanged', () => {
		const image: Layer = {
			id: 'i1',
			type: 'image',
			name: 'Image',
			visible: true,
			locked: false,
			x: 0,
			y: 0,
			width: 100,
			height: 100,
			opacity: 1
		};
		const text: Layer = { ...image, id: 't1', type: 'text', name: 'Text', content: '' };
		expect(anchorAnnotationToMap(image, dropX, dropY, map)).toBe(image);
		expect(anchorAnnotationToMap(text, dropX, dropY, map)).toBe(text);
	});

	it('TC-AN-08 map without map_config returns the annotation unchanged', () => {
		const noCfg = makeMapLayer({ map_config: undefined });
		const marker = makeMarkerLayer();
		expect(anchorAnnotationToMap(marker, dropX, dropY, noCfg)).toBe(marker);
	});
});

describe('resolveAnchorTarget', () => {
	// Default map box: x100 y50 w640 h400 → centre (420, 250).
	it('TC-AN-09 point over a map → that map, anchored AT the point', () => {
		const map = makeMapLayer();
		const target = resolveAnchorTarget([map], 300, 200);
		expect(target?.map.id).toBe(MAP_ID);
		expect(target?.x).toBe(300);
		expect(target?.y).toBe(200);
	});

	it('TC-AN-10 point off every map → nearest map, anchored at ITS centre (R5 floor)', () => {
		const map = makeMapLayer();
		const target = resolveAnchorTarget([map], 5, 5); // outside the box
		expect(target?.map.id).toBe(MAP_ID);
		expect(target?.x).toBe(420);
		expect(target?.y).toBe(250);
	});

	it('TC-AN-11 no anchorable map on the page → null', () => {
		expect(resolveAnchorTarget([], 10, 10)).toBeNull();
		expect(resolveAnchorTarget([makeMarkerLayer()], 10, 10)).toBeNull();
	});

	it('TC-AN-12 an UNCONFIGURED map under the point is never chosen (falls to a configured one)', () => {
		// Topmost map under the point has no map_config → must NOT be picked; the
		// configured map wins instead (anchored at its centre — never a silent no-op).
		const configured = makeMapLayer({ id: 'configured' });
		const noCfg = makeMapLayer({ id: 'nocfg', x: 0, y: 0, width: 200, height: 200, map_config: undefined });
		const target = resolveAnchorTarget([configured, noCfg], 100, 100); // inside BOTH boxes
		expect(target?.map.id).toBe('configured');
		expect(target?.x).toBe(420);
		expect(target?.y).toBe(250);
	});

	it('TC-AN-13 distance tie → the TOPMOST map wins (z-order consistent with mapLayerAt)', () => {
		const a = makeMapLayer({ id: 'a' }); // identical box → identical centre
		const b = makeMapLayer({ id: 'b' });
		const target = resolveAnchorTarget([a, b], 5, 5); // off both → fallback, equidistant
		expect(target?.map.id).toBe('b'); // later in z-order = topmost
	});
});
