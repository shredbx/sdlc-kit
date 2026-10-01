// TDD — canvas-kit map layer model contracts (Decision #0297).
// Covers: placeable map layer shape, map_surface_id on annotation layers,
// styleToMapType back-compat mapper, and MapConfig vocabulary reconciliation.
//
// CASE ENUMERATION
//   TC-ML-01 success  type:'map' is a valid LayerType discriminator
//   TC-ML-02 success  a map layer carries x/y/width/height (placeable box)
//   TC-ML-03 success  a map layer carries map_config with center + zoom + mapType
//   TC-ML-04 success  mapType accepts all four ui-map values
//   TC-ML-05 success  map layer reuses crop_top/right/bottom/left (same as image layers)
//   TC-ML-06 success  marker layer carries map_surface_id pointing to 'bg'
//   TC-ML-07 success  outline layer carries map_surface_id pointing to a placeable map id
//   TC-ML-08 success  callout layer carries map_surface_id
//   TC-ML-09 success  map_surface_id is optional — annotation layers without it are valid
//   TC-ML-10 success  styleToMapType: 'streets' → 'roadmap'
//   TC-ML-11 success  styleToMapType: 'satellite' → 'satellite'
//   TC-ML-12 success  styleToMapType: 'terrain' → 'terrain'
//   TC-ML-13 success  styleToMapType: 'dark' → 'roadmap' (no dark in Google Maps)
//   TC-ML-14 edge     styleToMapType: unknown value → 'roadmap' (safe default)
//   TC-ML-15 success  legacy MapConfig with style field still parses (back-compat)

import { describe, expect, it } from 'vitest';
import { styleToMapType } from './layer.js';
import type { Layer, MapConfig, MapType } from './layer.js';

// ---------------------------------------------------------------------------
// TC-ML-01..05 — placeable map layer shape
// ---------------------------------------------------------------------------

describe('placeable map layer — shape', () => {
	/** Minimal valid placeable map layer. */
	function makeMapLayer(overrides: Partial<Layer> = {}): Layer {
		return {
			id: 'map-1',
			type: 'map',
			name: 'Map',
			visible: true,
			locked: false,
			x: 100,
			y: 200,
			width: 480,
			height: 320,
			map_config: {
				center: { lat: 13.7563, lng: 100.5018 },
				zoom: 14,
				mapType: 'roadmap'
			},
			opacity: 1,
			...overrides
		};
	}

	it('TC-ML-01 type:"map" is a valid LayerType', () => {
		const layer = makeMapLayer();
		expect(layer.type).toBe('map');
	});

	it('TC-ML-02 map layer carries x/y/width/height (placeable box)', () => {
		const layer = makeMapLayer();
		expect(layer.x).toBe(100);
		expect(layer.y).toBe(200);
		expect(layer.width).toBe(480);
		expect(layer.height).toBe(320);
	});

	it('TC-ML-03 map layer carries map_config with center + zoom + mapType', () => {
		const layer = makeMapLayer();
		expect(layer.map_config).toBeDefined();
		expect(layer.map_config!.center).toEqual({ lat: 13.7563, lng: 100.5018 });
		expect(layer.map_config!.zoom).toBe(14);
		expect(layer.map_config!.mapType).toBe('roadmap');
	});

	it('TC-ML-04 mapType accepts all four ui-map values', () => {
		const types: MapType[] = ['roadmap', 'satellite', 'terrain', 'hybrid'];
		for (const mapType of types) {
			const layer = makeMapLayer({ map_config: { center: { lat: 0, lng: 0 }, zoom: 10, mapType } });
			expect(layer.map_config!.mapType).toBe(mapType);
		}
	});

	it('TC-ML-05 map layer reuses crop_top/right/bottom/left (same semantics as image layers)', () => {
		const layer = makeMapLayer({ crop_top: 10, crop_right: 20, crop_bottom: 30, crop_left: 40 });
		expect(layer.crop_top).toBe(10);
		expect(layer.crop_right).toBe(20);
		expect(layer.crop_bottom).toBe(30);
		expect(layer.crop_left).toBe(40);
	});
});

// ---------------------------------------------------------------------------
// TC-ML-06..09 — map_surface_id on annotation layers
// ---------------------------------------------------------------------------

describe('map_surface_id on annotation layers', () => {
	it('TC-ML-06 marker layer carries map_surface_id="bg" (background map)', () => {
		const layer: Layer = {
			id: 'pin-1',
			type: 'marker',
			name: 'Pin',
			visible: true,
			locked: false,
			x: 50,
			y: 80,
			map_surface_id: 'bg'
		};
		expect(layer.map_surface_id).toBe('bg');
	});

	it('TC-ML-07 outline layer carries map_surface_id pointing to a placeable map layer id', () => {
		const layer: Layer = {
			id: 'outline-1',
			type: 'outline',
			name: 'Parcel',
			visible: true,
			locked: false,
			x: 0,
			y: 0,
			map_surface_id: 'map-1'
		};
		expect(layer.map_surface_id).toBe('map-1');
	});

	it('TC-ML-08 callout layer carries map_surface_id', () => {
		const layer: Layer = {
			id: 'callout-1',
			type: 'callout',
			name: 'Label',
			visible: true,
			locked: false,
			x: 120,
			y: 90,
			map_surface_id: 'map-1',
			content: 'Lat: 13.76 Lng: 100.50'
		};
		expect(layer.map_surface_id).toBe('map-1');
	});

	it('TC-ML-09 map_surface_id is optional — annotation layers without it are valid', () => {
		const layer: Layer = {
			id: 'pin-2',
			type: 'marker',
			name: 'Floating pin',
			visible: true,
			locked: false,
			x: 0,
			y: 0
		};
		expect(layer.map_surface_id).toBeUndefined();
	});
});

// ---------------------------------------------------------------------------
// TC-ML-10..14 — styleToMapType back-compat mapper
// ---------------------------------------------------------------------------

describe('styleToMapType — back-compat mapper', () => {
	it('TC-ML-10 streets → roadmap', () => {
		expect(styleToMapType('streets')).toBe('roadmap');
	});

	it('TC-ML-11 satellite → satellite', () => {
		expect(styleToMapType('satellite')).toBe('satellite');
	});

	it('TC-ML-12 terrain → terrain', () => {
		expect(styleToMapType('terrain')).toBe('terrain');
	});

	it('TC-ML-13 dark → roadmap (no dark mode in Google Maps JS)', () => {
		expect(styleToMapType('dark')).toBe('roadmap');
	});

	it('TC-ML-14 unknown value → roadmap (safe default)', () => {
		expect(styleToMapType('custom-style-xyz')).toBe('roadmap');
	});
});

// ---------------------------------------------------------------------------
// TC-ML-15 — back-compat: legacy MapConfig with style field parses correctly
// ---------------------------------------------------------------------------

describe('MapConfig back-compat', () => {
	it('TC-ML-15 legacy MapConfig with deprecated style field still parses', () => {
		// Simulates a document serialized before Decision #0297 that stored
		// the old flat lat/lng + style shape. The new interface accepts `style`
		// as an optional deprecated field alongside the new `mapType`.
		// A migration path: read style, call styleToMapType(style), write mapType.
		const legacy: MapConfig = {
			center: { lat: 13.7563, lng: 100.5018 },
			zoom: 12,
			mapType: styleToMapType('streets'), // migrated
			style: 'streets' // kept for back-compat; readers prefer mapType
		};
		expect(legacy.mapType).toBe('roadmap');
		expect(legacy.style).toBe('streets');
		expect(styleToMapType(legacy.style)).toBe(legacy.mapType);
	});
});
