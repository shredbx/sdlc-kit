// Visual style value-types — ported as-is from land-canvas.

/** Visual properties for element borders/outlines */
export interface StrokeStyle {
	/** Hex color code */
	color: string;
	/** Width in pixels (1-20) */
	thickness: number;
	/** solid | dashed | dotted */
	dash_type: string;
	/** Glow effect (0-10) */
	glow_intensity: number;
}

/** Visual properties for element interior */
export interface FillStyle {
	/** Hex color code */
	color: string;
	/** Opacity (0-1) */
	transparency: number;
	/** Fill pattern type */
	pattern?: 'solid' | 'hatched' | 'crosshatch' | 'dots';
	/** Pattern line/dot spacing in pixels */
	pattern_spacing?: number;
}

/** Drop shadow effect for shapes/outlines */
export interface ShadowStyle {
	color: string;
	blur: number;
	offset_x: number;
	offset_y: number;
	opacity: number;
}

/** Centred halo glow — used by text AND image layers (no offset; blur ∝ intensity).
 *  Deliberately distinct from ShadowStyle: images may take a glow but NEVER an offset
 *  box shadow (user call 2026-06-07). The glow is the sanctioned image legibility
 *  effect — a contrasting halo that reads on any background (watermark, #0300). */
export interface GlowStyle {
	/** 0–10 halo strength (render blur = intensity × 3). */
	intensity: number;
	color: string;
}

/** Solid recolor of an image's alpha silhouette — composited offscreen at render time
 *  so one logo asset reads on both light and dark backgrounds (#0300). `strength`
 *  blends the original toward the tint colour (1 = flat recolor, 0 = untinted). */
export interface ImageTint {
	color: string;
	/** 0–1 blend toward `color`. */
	strength: number;
}

/** Named color swatch */
export interface ColorSwatch {
	id: string;
	name: string;
	color: string;
	created_at: number;
}

/** Named stroke+fill configuration for consistent branding */
export interface StylePreset {
	id: string;
	name: string;
	stroke: StrokeStyle;
	fill: FillStyle;
	fill_mode?: 'interior' | 'exterior';
	shadow?: ShadowStyle;
	/** text/callout: text color captured from source layer */
	text_color?: string;
	/** text/callout: text drop shadow captured from source layer */
	text_shadow?: { color: string; blur: number; offset_x: number; offset_y: number; opacity: number };
	/** text/callout: text glow effect captured from source layer */
	text_glow?: GlowStyle;
	/** text/callout: font weight captured from source layer */
	font_weight?: string;
	/** text/callout: font style captured from source layer */
	font_style?: string;
	/** Source layer type for compatibility UX (e.g. 'outline', 'text', 'callout') */
	layer_type?: string;
	created_at?: number;
}
