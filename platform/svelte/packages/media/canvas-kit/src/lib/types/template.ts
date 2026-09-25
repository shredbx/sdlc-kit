// Template / brand / export / tool value-types — ported as-is from land-canvas.

import type { Document } from './document.js';

/** Read-only pre-designed layout — ships with the app. */
export interface Template {
	id: string;
	name: string;
	/** property-card | listing-banner | open-house-flyer */
	category: string;
	thumbnail: string;
	document: Document;
}

/** Where a brand entry comes from: 'brand' = readonly (CSS/governance-derived),
 *  'custom' = content-manager-added (Phase B). */
export type BrandSource = 'brand' | 'custom';

/** One brand colour entry. `value` is any canvas2d-paintable color string
 *  (hex/rgb/rgba/hsl) — never a CSS var()/color-mix() reference. */
export interface BrandColor {
	name: string;
	value: string;
	source: BrandSource;
}

/** One brand font entry. `family` is a CSS font-family stack — like `value`,
 *  always a literal (paintable into ctx.font), never a var() reference. */
export interface BrandFont {
	name: string;
	family: string;
	source: BrandSource;
}

/** One brand logo entry (Phase B — pickers ship later). */
export interface BrandLogo {
	name: string;
	url: string;
	source: BrandSource;
}

/** Consumer-supplied branding configuration (Decision #0286 Phase A). The kit and
 *  UI are brand-agnostic: the consumer passes its kit in, the Inspector pickers
 *  list these entries, and new-layer defaults derive from them (LayerDefaults). */
export interface BrandKit {
	colors: BrandColor[];
	fonts: BrandFont[];
	logos: BrandLogo[];
}

/** Configuration for a canvas export operation. */
export interface ExportTarget {
	/** png | jpeg | pdf */
	format: 'png' | 'jpeg' | 'pdf';
	/** JPEG quality (1-100) */
	quality?: number;
	width: number;
	height: number;
	retina: boolean;
}

/** Shape type for shape tool sub-selection. */
export type ShapeToolType = 'rectangle' | 'ellipse' | 'arrow' | 'polygon' | 'star';

/** Active tool state for the canvas editor. */
export interface Tool {
	/** select | hand | outline | map-outline | text | shape | callout | marker */
	type: 'select' | 'hand' | 'outline' | 'map-outline' | 'text' | 'shape' | 'callout' | 'marker';
	active: boolean;
	/** When type='shape', which shape sub-type is selected. */
	shapeType?: ShapeToolType;
}
