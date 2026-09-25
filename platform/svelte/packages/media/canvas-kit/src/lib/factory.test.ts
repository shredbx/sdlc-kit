// TDD — presetToLayer (preset → Layer factory, slice 2).
//
// CASE ENUMERATION
//   TC-FA-01 success  text preset → a valid text layer (id, type, x/y, content)
//   TC-FA-02 success  text.heading is bigger than text.body (size mapping)
//   TC-FA-03 success  shape.rectangle → shape layer, shape_type 'rectangle'
//   TC-FA-04 success  shape.ellipse → shape_type 'ellipse'
//   TC-FA-05 success  shape.line → shape_type 'line', thin height
//   TC-FA-06 success  shape.triangle → polygon with sides:3
//   TC-FA-07 success  image preset → image layer, placeholder fill, no src
//   TC-FA-08 failure  widget → null (only widget has no renderer; map/marker/region/callout now materialise — TC-FA-12..23)
//   TC-FA-09 edge     x/y are used as the layer origin (top-left)
//   TC-FA-10 regress  ids are unique across calls
//   TC-FA-11 success  injected LayerDefaults land on text/shape/image layers

import { describe, expect, it } from 'vitest';
import { NEUTRAL_LAYER_DEFAULTS, presetToLayer, type LayerDefaults, type PresetInput } from './factory.js';

const preset = (id: string, baseType: PresetInput['baseType']): PresetInput => ({ id, baseType });

describe('presetToLayer — text', () => {
	it('TC-FA-01 produces a valid text layer', () => {
		const layer = presetToLayer(preset('text.body', 'text'), 10, 20);
		expect(layer).not.toBeNull();
		expect(layer!.id).toBeTruthy();
		expect(layer!.type).toBe('text');
		expect(layer!.x).toBe(10);
		expect(layer!.y).toBe(20);
		expect(layer!.content).toBe('Body text');
		expect(layer!.font).toBe(NEUTRAL_LAYER_DEFAULTS.textFont);
		expect(layer!.text_color).toBe(NEUTRAL_LAYER_DEFAULTS.textColor);
	});

	it('TC-FA-02 maps heading to a larger font than body', () => {
		const heading = presetToLayer(preset('text.heading', 'text'), 0, 0)!;
		const body = presetToLayer(preset('text.body', 'text'), 0, 0)!;
		expect(heading.font_size!).toBeGreaterThan(body.font_size!);
		expect(heading.font_weight).toBe('bold');
	});
});

describe('presetToLayer — shape', () => {
	it('TC-FA-03 produces a shape layer with shape_type rectangle', () => {
		const layer = presetToLayer(preset('shape.rectangle', 'shape'), 0, 0)!;
		expect(layer.type).toBe('shape');
		expect(layer.shape_type).toBe('rectangle');
		expect(layer.fill?.color).toBe(NEUTRAL_LAYER_DEFAULTS.shapeFill);
		expect(layer.stroke?.color).toBe(NEUTRAL_LAYER_DEFAULTS.shapeStroke);
		expect(layer.stroke?.thickness).toBe(2);
	});

	it('TC-FA-04 derives ellipse from the preset id', () => {
		expect(presetToLayer(preset('shape.ellipse', 'shape'), 0, 0)!.shape_type).toBe('ellipse');
	});

	it('TC-FA-05 derives line with a thin height', () => {
		const layer = presetToLayer(preset('shape.line', 'shape'), 0, 0)!;
		expect(layer.shape_type).toBe('line');
		expect(layer.height).toBe(4);
	});

	it('TC-FA-06 derives triangle as a 3-sided polygon', () => {
		const layer = presetToLayer(preset('shape.triangle', 'shape'), 0, 0)!;
		expect(layer.shape_type).toBe('polygon');
		expect(layer.sides).toBe(3);
	});
});

describe('presetToLayer — image', () => {
	it('TC-FA-07 produces an image layer with a placeholder fill and no src', () => {
		const layer = presetToLayer(preset('image.placeholder', 'image'), 0, 0)!;
		expect(layer.type).toBe('image');
		expect(layer.src).toBeUndefined();
		expect(layer.fill?.color).toBe(NEUTRAL_LAYER_DEFAULTS.imagePlaceholderFill);
		expect(layer.width).toBe(480);
		expect(layer.height).toBe(360);
	});
});

describe('presetToLayer — unsupported base types', () => {
	it('TC-FA-08 returns null ONLY for widget', () => {
		expect(presetToLayer(preset('x.widget', 'widget'), 0, 0)).toBeNull();
	});
});

describe('presetToLayer — map layer', () => {
	it('TC-FA-12 produces a type:map layer with map_config', () => {
		const layer = presetToLayer(preset('map.basic', 'map'), 50, 80);
		expect(layer).not.toBeNull();
		expect(layer!.type).toBe('map');
		expect(layer!.x).toBe(50);
		expect(layer!.y).toBe(80);
		expect(layer!.width).toBeGreaterThan(0);
		expect(layer!.height).toBeGreaterThan(0);
		expect(layer!.map_config).toBeDefined();
		expect(layer!.map_config!.center.lat).toBeGreaterThan(0); // non-null-island
		expect(layer!.map_config!.zoom).toBeGreaterThan(0);
		expect(layer!.map_config!.mapType).toBe('roadmap');
	});

	it('TC-FA-13 map layer has no src set (resolved later by mapLayerStaticUrl)', () => {
		const layer = presetToLayer(preset('map.basic', 'map'), 0, 0)!;
		expect(layer.src).toBeUndefined();
	});

	it('TC-FA-14 map layer ids are unique', () => {
		const a = presetToLayer(preset('map.basic', 'map'), 0, 0)!;
		const b = presetToLayer(preset('map.basic', 'map'), 0, 0)!;
		expect(a.id).not.toBe(b.id);
	});
});

describe('presetToLayer — marker layer', () => {
	it('TC-FA-15 produces a type:marker layer in map coord_system', () => {
		const layer = presetToLayer(preset('marker.pin', 'marker'), 10, 20);
		expect(layer).not.toBeNull();
		expect(layer!.type).toBe('marker');
		expect(layer!.coord_system).toBe('map');
		expect(layer!.x).toBe(10);
		expect(layer!.y).toBe(20);
	});

	it('TC-FA-16 marker has no map_surface_id (editor sets it on drop)', () => {
		const layer = presetToLayer(preset('marker.pin', 'marker'), 0, 0)!;
		expect(layer.map_surface_id).toBeUndefined();
	});

	it('TC-FA-17 marker stroke color inherits from LayerDefaults.shapeFill', () => {
		const layer = presetToLayer(preset('marker.pin', 'marker'), 0, 0, NEUTRAL_LAYER_DEFAULTS)!;
		expect(layer.stroke?.color).toBe(NEUTRAL_LAYER_DEFAULTS.shapeFill);
	});
});

describe('presetToLayer — outline (region preset)', () => {
	it('TC-FA-18 region preset produces a type:outline layer', () => {
		const layer = presetToLayer(preset('region.area', 'region'), 5, 10);
		expect(layer).not.toBeNull();
		expect(layer!.type).toBe('outline');
		expect(layer!.x).toBe(5);
		expect(layer!.y).toBe(10);
	});

	it('TC-FA-19 outline layer is closed in map coord_system with empty anchors', () => {
		const layer = presetToLayer(preset('region.area', 'region'), 0, 0)!;
		expect(layer.closed).toBe(true);
		expect(layer.coord_system).toBe('map');
		expect(layer.anchors).toEqual([]);
	});

	it('TC-FA-20 outline inherits stroke/fill from defaults', () => {
		const custom: LayerDefaults = { ...NEUTRAL_LAYER_DEFAULTS, shapeStroke: '#aabbcc', shapeFill: '#112233' };
		const layer = presetToLayer(preset('region.area', 'region'), 0, 0, custom)!;
		expect(layer.stroke?.color).toBe('#aabbcc');
		expect(layer.fill?.color).toBe('#112233');
	});
});

describe('presetToLayer — callout layer', () => {
	it('TC-FA-21 produces a type:callout layer with default content', () => {
		const layer = presetToLayer(preset('callout.label', 'callout'), 30, 40);
		expect(layer).not.toBeNull();
		expect(layer!.type).toBe('callout');
		expect(layer!.content).toBeTruthy();
		expect(layer!.x).toBe(30);
		expect(layer!.y).toBe(40);
	});

	it('TC-FA-22 callout text_color inherits from defaults', () => {
		const custom: LayerDefaults = { ...NEUTRAL_LAYER_DEFAULTS, textColor: '#ff0000' };
		const layer = presetToLayer(preset('callout.label', 'callout'), 0, 0, custom)!;
		expect(layer.text_color).toBe('#ff0000');
	});

	it('TC-FA-23 callout has no map_surface_id (editor sets it on drop)', () => {
		const layer = presetToLayer(preset('callout.label', 'callout'), 0, 0)!;
		expect(layer.map_surface_id).toBeUndefined();
	});
});

describe('presetToLayer — invariants', () => {
	it('TC-FA-09 uses x/y as the top-left origin', () => {
		const layer = presetToLayer(preset('shape.rectangle', 'shape'), 100, 200)!;
		expect(layer.x).toBe(100);
		expect(layer.y).toBe(200);
	});

	it('TC-FA-10 produces a unique id on every call', () => {
		const a = presetToLayer(preset('text.body', 'text'), 0, 0)!;
		const b = presetToLayer(preset('text.body', 'text'), 0, 0)!;
		expect(a.id).not.toBe(b.id);
	});
});

describe('presetToLayer — injected LayerDefaults (Decision #0286 Phase A)', () => {
	it('TC-FA-11 lands custom defaults on text, shape and image layers', () => {
		const custom: LayerDefaults = {
			textFont: 'Georgia, serif',
			textColor: '#123456',
			shapeFill: '#abcdef',
			shapeStroke: '#fedcba',
			imagePlaceholderFill: '#0f0f0f'
		};
		const text = presetToLayer(preset('text.body', 'text'), 0, 0, custom)!;
		expect(text.font).toBe(custom.textFont);
		expect(text.text_color).toBe(custom.textColor);
		const shape = presetToLayer(preset('shape.rectangle', 'shape'), 0, 0, custom)!;
		expect(shape.fill?.color).toBe(custom.shapeFill);
		expect(shape.stroke?.color).toBe(custom.shapeStroke);
		const image = presetToLayer(preset('image.placeholder', 'image'), 0, 0, custom)!;
		expect(image.fill?.color).toBe(custom.imagePlaceholderFill);
	});
});
