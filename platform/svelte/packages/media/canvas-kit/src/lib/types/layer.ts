// Universal layer node — EXTRACTED from clients/andrei/projects/land-canvas
// (src/lib/types.ts) and EXTENDED for Media Canvas. The base shape is a faithful
// copy (extract, not rewrite); Media-Canvas additions are marked `[MC]`.
// Land-specific fields (anchors, edge_labels, map_config, geo_ref, marker…) were
// previously dormant; they ARE NOW surfaced by the Media Canvas map editor
// (Decision #0297). type:'map' is a placeable layer; annotation layers
// (marker/outline/callout) carry map_surface_id to anchor to a specific map surface.

import type { AnimationTrack } from './animation.js';
import type { Binding } from './binding.js';
import type { StrokeStyle, FillStyle, ShadowStyle, GlowStyle, ImageTint } from './style.js';
import type { DistanceUnit, AreaUnit } from './measurement.js';
import type { Template } from '@sbx/text-template';

/** Discriminator enum for the Layer type field. */
export type LayerType = 'image' | 'shape' | 'outline' | 'text' | 'callout' | 'group' | 'marker' | 'map';

/**
 * Map type discriminator — ui-map vocabulary (Decision #0297).
 * Reconciled from the legacy land-canvas 'streets'|'satellite'|'terrain'|'dark' values
 * to match @sbx/ui-map's mapType string and Google Maps MapTypeId.
 */
export type MapType = 'roadmap' | 'satellite' | 'terrain' | 'hybrid';

/**
 * Map the legacy land-canvas style values to the canonical MapType (ui-map vocab).
 * Back-compat: old documents that stored style:'streets' or style:'dark' continue to
 * resolve correctly. Exported so consumers can migrate stored documents.
 *   streets  → roadmap   (OSM streets ≈ Google roadmap)
 *   satellite → satellite (direct match)
 *   terrain   → terrain   (direct match)
 *   dark      → roadmap   (no dark mode in Google Maps JS; falls back to roadmap)
 */
export function styleToMapType(style: string): MapType {
	switch (style) {
		case 'satellite':
			return 'satellite';
		case 'terrain':
			return 'terrain';
		case 'streets':
		case 'dark':
		default:
			return 'roadmap';
	}
}

/**
 * Map configuration — used on both the background layer (bg_mode:'map') AND on
 * placeable type:'map' layers (Decision #0297). Vocabulary reconciled to ui-map
 * (Google Maps): center object + mapType enum instead of the legacy flat lat/lng +
 * style fields.
 *
 * Back-compat: `style` is retained as an optional deprecated field so existing
 * documents that serialized the old shape continue to parse without data-loss.
 * Readers should prefer `mapType`; if absent, call styleToMapType(style) to derive it.
 */
export interface MapConfig {
	/** Map center — matches @sbx/ui-map LngLat. */
	center: { lat: number; lng: number };
	zoom: number;
	/** Map display type — ui-map / Google Maps vocabulary (Decision #0297). */
	mapType: MapType;
	/**
	 * @deprecated Legacy land-canvas style field. Use `mapType` instead.
	 * Retained for back-compat with documents serialized before Decision #0297.
	 * Call styleToMapType(style) to migrate. Will be removed in a future slice.
	 */
	style?: 'streets' | 'satellite' | 'terrain' | 'dark';
	/** Map rotation in degrees (0-360). Deferred — not yet surfaced in UI (Decision #0297). */
	rotation?: number;
}

/** Background display mode. */
export type BgMode = 'color' | 'image' | 'map';

/** A point in 2D space — building block for outlines. */
export interface AnchorPoint {
	id: string;
	x: number;
	y: number;
}

/** Text label rendered along an outline edge. */
export interface EdgeLabel {
	id: string;
	anchor_from_id: string;
	anchor_to_id: string;
	text: string;
	font: string;
	size: number;
	color: string;
	offset: number;
	/** 'custom' for user text, 'distance' for auto-computed haversine distance. */
	mode?: 'custom' | 'distance';
}

/** Area label display configuration for closed map outlines. */
export interface AreaLabel {
	visible: boolean;
	unit: AreaUnit;
	font: string;
	size: number;
	color: string;
}

/**
 * [WM] Tiling effect — repeats the layer (or group) across the artboard as a
 * watermark pattern. A GENERIC transform effect: attach it to ANY single element
 * OR a group (to tile composed elements together as one unit). NOT a text- or
 * image-specific field.
 *
 * Semantics (standard media-editor mirroring):
 *  - The source layer renders where it is placed; copies repeat OUTWARD from it.
 *  - `repeatH` → copies extend left AND right (mirror horizontally); `repeatV` →
 *    up AND down (full coverage when both are on).
 *  - `gapX` / `gapY` = the empty space (px) between adjacent copies on each axis,
 *    so pitch = sourceSize + gap. Independent per axis (the two-gap requirement).
 *  - The WHOLE pattern inherits the layer's own `rotation` (rotate once → every
 *    copy rotates together, around the source centre — not per-copy).
 *
 * Copies are pure repeats of the source: selecting/clicking any copy resolves to
 * the SOURCE layer (the one you placed), never a phantom layer.
 */
export interface TileEffect {
	/** Repeat horizontally (mirror the source left and right). */
	repeatH: boolean;
	/** Repeat vertically (mirror the source up and down). */
	repeatV: boolean;
	/** Horizontal gap between copies, in px (pitch = source width + gapX). */
	gapX: number;
	/** Vertical gap between copies, in px (pitch = source height + gapY). */
	gapY: number;
}

/** Universal tree node — discriminated union by `type`. */
export interface Layer {
	id: string;
	/** Discriminator. */
	type: LayerType;
	name: string;
	visible: boolean;
	locked: boolean;
	/** System layer — not deletable/movable by user (e.g. background). */
	system?: boolean;
	x: number;
	y: number;
	width?: number;
	height?: number;
	/** image: blob/object URL */
	src?: string;
	/** image: file MIME type */
	mime_type?: string;
	/** image: solid recolor of the alpha silhouette — one logo asset reads on both light
	 *  and dark backgrounds (watermark legibility, #0300). Composited offscreen at render. */
	tint?: ImageTint;
	/** image: centred glow halo — the sanctioned image legibility effect (images take a
	 *  glow but never an offset box shadow, user call 2026-06-07). A soft contrasting halo
	 *  that keeps the mark readable on any background (#0300). */
	glow?: GlowStyle;
	/** shape: rectangle | line | ellipse | arrow | polygon | star */
	shape_type?: 'rectangle' | 'line' | 'ellipse' | 'arrow' | 'polygon' | 'star';
	/** polygon: number of sides (3-12, default 6) */
	sides?: number;
	/** star: number of points (3-12, default 5) */
	points?: number;
	/** star: inner radius ratio (0.1-0.9, default 0.4) */
	innerRadiusRatio?: number;
	/** shape/outline: stroke config */
	stroke?: StrokeStyle;
	/** shape/outline: fill config */
	fill?: FillStyle;
	/** outline: path points */
	anchors?: AnchorPoint[];
	/** outline: shape is closed */
	closed?: boolean;
	/** shape/outline: fill region (interior vs exterior) */
	fill_mode?: 'interior' | 'exterior';
	/** shape/outline: drop shadow */
	shadow?: ShadowStyle;
	/** outline: edge text labels */
	edge_labels?: EdgeLabel[];
	/** text/callout: text content */
	content?: string;
	/** text: font family */
	font?: string;
	/** text/callout: font size in pixels */
	font_size?: number;
	/** text/callout: font weight — 'normal' | 'bold' | 'medium' | '100'-'900' */
	font_weight?: string;
	/** text/callout: font style — 'normal' | 'italic' */
	font_style?: string;
	/** text/callout: word wrapping mode */
	text_wrap?: 'none' | 'word';
	/** text/callout: how content fits its box when it would overflow (S-FIT). Default
	 *  (undefined) = 'none' (draw at the set size — may overflow). 'fit' SHRINKS the
	 *  font (down to 50%) so it never overflows but NEVER grows past the set size;
	 *  'fill' scales the font UP and down (50%–400%) to fill the box; both ellipsise
	 *  at the floor. 'crop' clips to the box at the natural size; 'truncate' =
	 *  single-line ellipsis. */
	text_fit?: 'fit' | 'fill' | 'crop' | 'truncate' | 'none';
	/** text/callout: horizontal text alignment */
	text_align?: 'left' | 'center' | 'right';
	/** text/callout + rectangle shapes: corner radius (px) of the box — the text
	 *  background/border and rectangle fills/strokes draw with rounded corners. */
	corner_radius?: number;
	/** Per-edge inset crop (px from each edge of the layer box) — the layer draws
	 *  normally and is CLIPPED to box-minus-insets (renderLayer, central — every
	 *  type). Flat scalars so each edge is directly animatable (wipe/reveal). */
	crop_top?: number;
	crop_right?: number;
	crop_bottom?: number;
	crop_left?: number;
	/** text/callout: text color (independent from fill) */
	text_color?: string;
	/** text/callout: text drop shadow */
	text_shadow?: { color: string; blur: number; offset_x: number; offset_y: number; opacity: number };
	/** text/callout: text glow effect */
	text_glow?: GlowStyle;
	/** marker: which marker icon to render */
	marker_id?: string;
	/**
	 * marker/outline/callout: identifies WHICH map surface this annotation anchors to.
	 * 'bg' = the background map (bg_mode:'map'); any other value = the id of a
	 * placeable type:'map' layer. Absent → not anchored to any map surface
	 * (geo_ref alone has no parent reference; this field closes that gap).
	 */
	map_surface_id?: string;
	/** background: display mode (color | image | map) */
	bg_mode?: BgMode;
	/**
	 * background (bg_mode:'map') OR type:'map' placeable layer: map configuration.
	 * When on a type:'map' layer, the map viewport is rendered within the layer's
	 * x/y/width/height box, and crop_top/right/bottom/left clip the viewport from
	 * its edges — the same per-edge inset crop semantics as image layers (Decision #0297).
	 */
	map_config?: MapConfig;
	/** Coordinate system for anchor positions: 'canvas' (pixels) or 'map' (lng/lat) */
	coord_system?: 'canvas' | 'map';
	/** Layer opacity (0-1, default 1) */
	opacity?: number;
	/** callout: ID of outline layer this callout is attached to */
	attached_to?: string;
	/** callout: geo-referenced position when attached to a map outline */
	geo_ref?: { lat: number; lng: number; anchor_lats: number[]; anchor_lngs: number[] };
	/** outline: distance display unit for edge labels */
	distance_unit?: DistanceUnit;
	/** outline: area display unit */
	area_unit?: AreaUnit;
	/** outline: area label display config for closed map outlines */
	area_label?: AreaLabel;
	/** group: nested layers */
	children?: Layer[];

	// --- [MC] Media Canvas extensions (additive; base stays a static value) ---
	/** [MC] rotation in degrees (default 0) — animatable. The tiling pattern (when
	 *  `tile` is set) rotates as a whole by this angle, around the source centre. */
	rotation?: number;
	/** [WM] Tiling effect — when set, the layer repeats across the artboard per
	 *  TileEffect (repeat-H/V + per-axis gaps), the whole pattern rotated by
	 *  `rotation`. Attach to any element or a group. Absent = a single placement. */
	tile?: TileEffect;
	/** [MC] uniform scale (default 1) — animatable. */
	scale?: number;
	/** [MC] property tracks driving animation over the page duration. */
	animations?: AnimationTrack[];
	/** [MC] data bindings mapping properties to attached-source field tokens. */
	bindings?: Binding[];
	/** [MC] text/callout: a token-interpolating template for `content` (INC-C). When set (and
	 *  non-empty) the content is resolveTemplate()'d from the bound source's snapshot — the field
	 *  is a 'template' kind (fieldKindFor), mutually exclusive with a `content` binding. */
	content_template?: Template;
}
