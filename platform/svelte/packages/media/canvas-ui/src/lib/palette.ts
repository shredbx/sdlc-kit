// Shared catalogues for the editor UI: the typed PRESET catalogue (D15
// groundwork) that every insertable palette grid reads from, and the
// per-layer-type icon used by the Layers list + timeline lanes. Pure data — no
// Svelte, so the panels, the grids, and the timeline read the SAME source
// (single source of truth for the element vocabulary).

import type { LayerType, AnimatableProperty } from '@sbx/canvas-kit';

// --- Preset catalogue (D15) ------------------------------------------------
// A preset = one palette card over a base type. Adding a variant is one data
// entry, zero code (modular-architecture-first / no-discriminator). This cut
// ships the kit defaults; persistence + the real insert/anchor behaviour land
// in later slices.

/** The base layer type a preset materialises (D15/§11.3). Geo family added in
 *  the Map-panel rewrite (T5): map · marker · region · callout are now insertable
 *  presets via the Map Components grid. The kit factory materialises them all. */
export type PresetBaseType = 'text' | 'image' | 'shape' | 'widget' | 'map' | 'marker' | 'region' | 'callout';

/** How a palette card places its element (the standardized placement model):
 *  - 'tool'    arms a cursor TOOL — the next canvas click (or drag-draw) creates it
 *              where you point (Figma-style). The grid card stays lit while armed.
 *  - 'instant' drops the element in the PAGE CENTRE on click, auto-selected
 *              (Canva-style) — for content blocks you arrange rather than draw. */
export type PlacementMode = 'instant' | 'tool';

/** For mode:'tool', how the apply gesture works:
 *  - 'box'   click = default size at the point · click-DRAG = draw the bounding box.
 *  - 'point' click places at the point (no size — a pin has none). */
export type PlacementKind = 'point' | 'box';

/** One insertable palette card, grouped by `section`. */
export interface Preset {
	/** Stable id, e.g. 'text.heading', 'shape.rectangle', 'widget.price-tag'. */
	id: string;
	/** Display label shown on the card. */
	label: string;
	/** The base type this preset materialises. */
	baseType: PresetBaseType;
	/** Which rail-panel section the card appears under ('text', 'shape', …). */
	section: string;
	/** core-ui Icon name (verified against the Icon registry). */
	icon: string;
	/** Placement behaviour — defaults to 'tool' (arm a cursor tool) when omitted. */
	mode?: PlacementMode;
	/** Tool apply gesture — defaults to 'box' when omitted (ignored for mode:'instant'). */
	kind?: PlacementKind;
}

/**
 * The preset catalogue, grouped by `section`. Icon names are all verified in
 * the core-ui Icon registry — the registry has no dedicated text glyphs
 * (`type`/`heading`/`text` are absent), so the text presets reuse the closest
 * available document/list glyphs for distinct cards.
 */
export const PRESETS: Preset[] = [
	// Text (Canva standard set) — registry has no text-specific glyphs, so each
	// card uses a distinct available document/list icon.
	{ id: 'text.heading', label: 'Heading', baseType: 'text', section: 'text', icon: 'file-text' },
	{ id: 'text.subheading', label: 'Subheading', baseType: 'text', section: 'text', icon: 'book-text' },
	{ id: 'text.body', label: 'Body', baseType: 'text', section: 'text', icon: 'scroll-text' },
	{ id: 'text.box', label: 'Text box', baseType: 'text', section: 'text', icon: 'list' },

	// Shapes
	{ id: 'shape.rectangle', label: 'Rectangle', baseType: 'shape', section: 'shape', icon: 'square' },
	{ id: 'shape.ellipse', label: 'Ellipse', baseType: 'shape', section: 'shape', icon: 'circle' },
	{ id: 'shape.line', label: 'Line', baseType: 'shape', section: 'shape', icon: 'minus' },
	{ id: 'shape.triangle', label: 'Triangle', baseType: 'shape', section: 'shape', icon: 'triangle' },

	// Widgets (composite BR elements) — content blocks you DROP, not draw → instant-centre.
	{ id: 'widget.price-tag', label: 'Price tag', baseType: 'widget', section: 'widget', icon: 'tag', mode: 'instant' },
	{ id: 'widget.spec-badge', label: 'Spec badge', baseType: 'widget', section: 'widget', icon: 'badge-check', mode: 'instant' },
	{ id: 'widget.image-callout', label: 'Image callout', baseType: 'widget', section: 'widget', icon: 'message-square', mode: 'instant' },

	// Map components (T5 — Map Components grid). Icons verified in core-ui IconRegistry:
	//   map → Map (Lucide), map-pin → MapPin (teardrop LOCATION marker — R1 "location icon",
	//   matches the static-map pin glyph; NOT Pin/pushpin), hexagon → Hexagon (closest to
	//   outline/region polygon), message-square → MessageSquare (label/callout — same as widget.image-callout).
	// All are TOOLS (placed where you click on the map); Pin is a point, the rest are boxes.
	{ id: 'map.basic', label: 'Map', baseType: 'map', section: 'map', icon: 'map' },
	{ id: 'marker.pin', label: 'Pin', baseType: 'marker', section: 'map', icon: 'map-pin', kind: 'point' },
	{ id: 'region.area', label: 'Outline', baseType: 'region', section: 'map', icon: 'hexagon' },
	{ id: 'callout.label', label: 'Callout', baseType: 'callout', section: 'map', icon: 'message-square' }
];

/** All presets that belong to a given section (drives one ComponentGrid). */
export function presetsForSection(section: string): Preset[] {
	return PRESETS.filter((p) => p.section === section);
}

/** Look up a preset by id. */
export function presetById(id: string): Preset | undefined {
	return PRESETS.find((p) => p.id === id);
}

/** Resolved placement behaviour for a preset id (defaults: tool + box). Single source
 *  of truth so EditorShell (dispatch) and CanvasStage (apply gesture) never diverge. */
export function placementOf(id: string): { mode: PlacementMode; kind: PlacementKind } {
	const p = presetById(id);
	return { mode: p?.mode ?? 'tool', kind: p?.kind ?? 'box' };
}

// --- Animatable properties (slice-5a, design §12.6 / G6) -------------------
// The numeric/continuous props the timeline + Inspector ≣ can animate, with the
// display unit hint and a model→display scale (opacity/scale are 0–1 in the model
// but edited as % in the popup). Single source of truth for both surfaces.

export interface AnimatablePropMeta {
	key: AnimatableProperty;
	/** Human label shown in the dialog dropdown. */
	label: string;
	/** Display unit hint (G6): px · % · deg. */
	unit: string;
	/** model × displayScale = display value (opacity 0–1 → 0–100 %). */
	displayScale: number;
}

export const ANIMATABLE_PROPS: AnimatablePropMeta[] = [
	{ key: 'x', label: 'X', unit: 'px', displayScale: 1 },
	{ key: 'y', label: 'Y', unit: 'px', displayScale: 1 },
	{ key: 'width', label: 'Width', unit: 'px', displayScale: 1 },
	{ key: 'height', label: 'Height', unit: 'px', displayScale: 1 },
	{ key: 'opacity', label: 'Opacity', unit: '%', displayScale: 100 },
	{ key: 'rotation', label: 'Rotation', unit: 'deg', displayScale: 1 },
	{ key: 'scale', label: 'Scale', unit: '%', displayScale: 100 },
	// Per-edge inset crop (2026-06-07) — animatable wipes/reveals.
	{ key: 'crop_top', label: 'Crop top', unit: 'px', displayScale: 1 },
	{ key: 'crop_right', label: 'Crop right', unit: 'px', displayScale: 1 },
	{ key: 'crop_bottom', label: 'Crop bottom', unit: 'px', displayScale: 1 },
	{ key: 'crop_left', label: 'Crop left', unit: 'px', displayScale: 1 }
];

/** Lookup metadata for one animatable property. */
export function animatableMeta(key: AnimatableProperty): AnimatablePropMeta | undefined {
	return ANIMATABLE_PROPS.find((p) => p.key === key);
}

// --- Display ↔ model conversion (single source of truth) -------------------
// Opacity/scale live as 0–1 fractions in the model but are edited as % in every
// editor surface. AddAnimationDialog + AnimationInfoPopover BOTH route through
// these so the conversion lives in exactly one place (no copy-pasted scaling).

/** Round a display value to 2 decimals (kills float dust in the inputs). */
function roundDisplay(n: number): number {
	return Math.round(n * 100) / 100;
}

/** model value → display value (× displayScale; opacity 0–1 → 0–100 %). */
export function toDisplay(key: AnimatableProperty, model: number): number {
	const scale = animatableMeta(key)?.displayScale ?? 1;
	return roundDisplay(model * scale);
}

/** display value → model value (÷ displayScale; 0–100 % → opacity 0–1). */
export function toModel(key: AnimatableProperty, display: number): number {
	const scale = animatableMeta(key)?.displayScale ?? 1;
	return display / scale;
}

/** Inspector row label → animatable property key, or null when the row isn't
 *  animatable (CONTENT/FONT/SIZE/COLOR/ALIGN/WEIGHT/FILL/STROKE/THICKNESS). */
export function rowLabelToProperty(label: string): AnimatableProperty | null {
	const map: Record<string, AnimatableProperty> = {
		X: 'x',
		Y: 'y',
		WIDTH: 'width',
		HEIGHT: 'height',
		OPACITY: 'opacity',
		ROTATION: 'rotation',
		SCALE: 'scale',
		'CROP TOP': 'crop_top',
		'CROP RIGHT': 'crop_right',
		'CROP BOTTOM': 'crop_bottom',
		'CROP LEFT': 'crop_left'
	};
	return map[label.trim().toUpperCase()] ?? null;
}

// --- Layer-row icons (Layers list + timeline lanes) ------------------------

/** core-ui Icon name for a layer row, by layer type. System layers use a lock-ish glyph. */
export function layerIcon(type: LayerType): string {
	switch (type) {
		case 'image':
			return 'image';
		case 'text':
			return 'file-text';
		case 'callout':
			return 'message-square';
		case 'shape':
			return 'square';
		case 'group':
			return 'folder-open';
		case 'map':
			return 'map';
		case 'marker':
			return 'map-pin';
		case 'outline':
			return 'hexagon';
		default:
			return 'square';
	}
}
