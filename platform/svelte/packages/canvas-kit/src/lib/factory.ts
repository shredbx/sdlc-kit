// Preset → Layer factory — materialises a palette preset (D15) into a concrete
// Layer. Sizes/seeds ported from clients/andrei/projects/land-canvas
// (apps/web/svelte/src/lib/utils/create.ts: createText ~76-98, createShape
// ~194-227) per the slice-2 spec (docs/plans/2026-06-01-media-canvas-design.md
// §4/§7). Colours/fonts are INJECTED (Decision #0286 Phase A): the consumer
// passes brand-derived LayerDefaults; the kit itself stays brand-neutral.
//
// Renderable base types only: text · image · shape. Geo (region/marker/callout/
// map) and widget presets are NOT renderable until their own later slices, so
// they return null here (never faked). HEADLESS — no Svelte, no DOM; ids via the
// kit's crypto.randomUUID() (matches store.ts / land-canvas create.ts).

import type { Layer } from './types/layer.js';
import type { BrandKit } from './types/template.js';

/** The base layer type a preset materialises (mirrors the UI palette's PresetBaseType). */
export type PresetBaseType =
	| 'text'
	| 'image'
	| 'shape'
	| 'region'
	| 'marker'
	| 'callout'
	| 'map'
	| 'widget';

/**
 * The subset of a palette Preset the factory needs. Structural — the UI's
 * `Preset` (canvas-ui/palette.ts) is assignable to this, so the kit never
 * imports the UI layer.
 */
export interface PresetInput {
	id: string;
	baseType: PresetBaseType;
}

// --- Injected defaults (Decision #0286 Phase A) -------------------------------

/** Colour/font defaults a NEW layer is born with. The consumer derives these from
 *  its BrandKit (e.g. BR's buildLayerDefaults) and threads them through the editor;
 *  the kit ships only the brand-neutral fallback below. */
export interface LayerDefaults {
	textFont: string;
	textColor: string;
	shapeFill: string;
	shapeStroke: string;
	imagePlaceholderFill: string;
}

/** Brand-neutral defaults — what presetToLayer uses when no consumer kit is wired.
 *  Frozen: shared singleton — a consumer mutation would poison every consumer. */
export const NEUTRAL_LAYER_DEFAULTS: LayerDefaults = Object.freeze({
	textFont: 'system-ui, sans-serif',
	textColor: '#111111',
	shapeFill: '#888888',
	shapeStroke: '#666666',
	imagePlaceholderFill: '#D9D9D9'
});

/** No brand entries — the default when a consumer passes no BrandKit.
 *  Lives here (not types/template.ts) — type files stay type-only.
 *  Deep-frozen: shared singleton (also returned by SSR buildBrandKit paths). */
export const EMPTY_BRAND_KIT: BrandKit = Object.freeze({
	colors: Object.freeze([]) as never[],
	fonts: Object.freeze([]) as never[],
	logos: Object.freeze([]) as never[]
});

/** Per-text-preset seed: size, weight, content, and a proportional box. */
interface TextSeed {
	font_size: number;
	font_weight: string;
	content: string;
	width: number;
	height: number;
}

function textSeed(presetId: string): TextSeed {
	switch (presetId) {
		case 'text.heading':
			return { font_size: 64, font_weight: 'bold', content: 'Heading', width: 600, height: 80 };
		case 'text.subheading':
			return { font_size: 40, font_weight: '600', content: 'Subheading', width: 500, height: 56 };
		case 'text.body':
			return { font_size: 24, font_weight: 'normal', content: 'Body text', width: 400, height: 40 };
		default:
			// text.box / text.textbox / any other text preset
			return { font_size: 24, font_weight: 'normal', content: 'Text', width: 400, height: 40 };
	}
}

/** Map a shape preset id to a renderable shape_type (+ sides for triangle). */
function shapeKind(presetId: string): { shape_type: NonNullable<Layer['shape_type']>; sides?: number } {
	switch (presetId) {
		case 'shape.ellipse':
			return { shape_type: 'ellipse' };
		case 'shape.line':
			return { shape_type: 'line' };
		case 'shape.triangle':
			return { shape_type: 'polygon', sides: 3 };
		default:
			// shape.rectangle / any other shape preset
			return { shape_type: 'rectangle' };
	}
}

/**
 * Materialise a preset into a Layer with origin (x, y) = top-left. The caller
 * computes (x, y) (e.g. to centre the layer on the page) and MAY pass brand-derived
 * LayerDefaults (colours/fonts the new layer is born with — brand-neutral when
 * omitted). Only `widget` returns null (no renderer yet — it lands in its own later
 * slice and MUST NOT be faked); map/marker/region/callout return real layers.
 */
export function presetToLayer(
	preset: PresetInput,
	x: number,
	y: number,
	defaults: LayerDefaults = NEUTRAL_LAYER_DEFAULTS
): Layer | null {
	switch (preset.baseType) {
		case 'text': {
			const seed = textSeed(preset.id);
			return {
				id: crypto.randomUUID(),
				type: 'text',
				name: seed.content,
				visible: true,
				locked: false,
				x,
				y,
				width: seed.width,
				height: seed.height,
				content: seed.content,
				font: defaults.textFont,
				font_size: seed.font_size,
				font_weight: seed.font_weight,
				font_style: 'normal',
				text_wrap: 'word',
				text_align: 'left',
				text_color: defaults.textColor,
				opacity: 1
			};
		}
		case 'shape': {
			const { shape_type, sides } = shapeKind(preset.id);
			const isLine = shape_type === 'line';
			const layer: Layer = {
				id: crypto.randomUUID(),
				type: 'shape',
				name: shape_type.charAt(0).toUpperCase() + shape_type.slice(1),
				visible: true,
				locked: false,
				x,
				y,
				width: 240,
				height: isLine ? 4 : 240,
				shape_type,
				fill: { color: defaults.shapeFill, transparency: 0, pattern: 'solid' },
				stroke: { color: defaults.shapeStroke, thickness: 2, dash_type: 'solid', glow_intensity: 0 },
				opacity: 1
			};
			if (sides !== undefined) layer.sides = sides;
			return layer;
		}
		case 'image': {
			return {
				id: crypto.randomUUID(),
				type: 'image',
				name: 'Image',
				visible: true,
				locked: false,
				x,
				y,
				width: 480,
				height: 360,
				// No src yet — renders as a neutral placeholder box (mock cover parity).
				fill: { color: defaults.imagePlaceholderFill, transparency: 0, pattern: 'solid' },
				opacity: 1
			};
		}
		// Map layer — a placeable static-map surface. Born with a sensible default
		// map config (Bangkok center, zoom 14, roadmap). The editor sets map_surface_id
		// on any annotation layers dropped onto this map. `src` is resolved by
		// resolveMapSrc before rendering/export — the factory does NOT set it.
		case 'map': {
			return {
				id: crypto.randomUUID(),
				type: 'map',
				name: 'Map',
				visible: true,
				locked: false,
				x,
				y,
				width: 480,
				height: 360,
				map_config: {
					center: { lat: 13.7563, lng: 100.5018 }, // Bangkok — neutral default
					zoom: 14,
					mapType: 'roadmap'
				},
				opacity: 1
			};
		}

		// Marker annotation — a geo-anchored pin on a map surface. geo_ref is set
		// by the editor on drop (when the user places the marker on a map layer).
		// map_surface_id is set by the editor after drop.
		case 'marker': {
			return {
				id: crypto.randomUUID(),
				type: 'marker',
				name: 'Marker',
				visible: true,
				locked: false,
				x,
				y,
				width: 32,
				height: 40,
				stroke: { color: defaults.shapeFill, thickness: 2, dash_type: 'solid', glow_intensity: 0 },
				coord_system: 'map',
				opacity: 1
			};
		}

		// Outline annotation — a geo-anchored region polygon on a map surface.
		// `region` is the preset base type; the Layer discriminator is 'outline'
		// (the layer type mirrors land-canvas: the outline layer IS the region).
		// anchors and coord_system:'map' are set by the editor on drop.
		case 'region': {
			return {
				id: crypto.randomUUID(),
				type: 'outline',
				name: 'Region',
				visible: true,
				locked: false,
				x,
				y,
				width: 200,
				height: 200,
				anchors: [],
				closed: true,
				coord_system: 'map',
				stroke: { color: defaults.shapeStroke, thickness: 3, dash_type: 'solid', glow_intensity: 0 },
				fill: { color: defaults.shapeFill, transparency: 0.8, pattern: 'solid' },
				opacity: 1
			};
		}

		// Callout annotation — a geo-referenced text label on a map surface.
		// geo_ref carries the anchor position; map_surface_id links it to the map.
		// The render pass projects the geo_ref to canvas pixels via projectToPixel.
		case 'callout': {
			return {
				id: crypto.randomUUID(),
				type: 'callout',
				name: 'Callout',
				visible: true,
				locked: false,
				x,
				y,
				width: 160,
				height: 40,
				content: 'Label',
				font_size: 14,
				font_weight: 'bold',
				font_style: 'normal',
				text_align: 'center',
				text_color: defaults.textColor,
				coord_system: 'map',
				opacity: 1
			};
		}

		// widget has no renderer — return null.
		case 'widget':
		default:
			return null;
	}
}
