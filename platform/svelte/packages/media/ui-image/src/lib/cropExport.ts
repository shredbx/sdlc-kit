// cropExport — the PURE crop-geometry + output-format law behind SquareLogoCropper.svelte
// (task 2606-118). The Svelte component is a thin canvas/DOM shell; the testable LOGIC lives
// here so vitest (node, no real canvas) can prove it independently of any rendering.
//
// Two responsibilities, both side-effect-free:
//   1. computeSquareCrop  — the output master is a SQUARE whose side is the largest square
//      that fits the source, clamped DOWN to LOGO_MAX_DIMENSION (512). We never upscale past
//      the source (we don't invent pixels — mirrors the canvas image-variant semantics: a
//      master is only ever <= the source).
//   2. pickLogoOutputFormat — the SC4 format law: PNG-in → PNG-out (alpha survives); SVG → a
//      non-raster passthrough sentinel (the caller SKIPS crop/resize and sanitizes the vector
//      upstream); every other raster → a WebP master (best size/quality, no transparency need).

/** The brand-logo master cap (long edge, px). Mirrors FACET_MAX_DIMENSION['logo'] and the
 *  image-optimization policy: a logo master is always <= 512 on its longest side. */
export const LOGO_MAX_DIMENSION = 512;

/** Non-raster sentinel returned by {@link pickLogoOutputFormat} for an SVG source. It is
 *  deliberately NOT an `image/*` raster MIME so callers branch on it to skip the
 *  crop/resize/canvas path entirely and treat the (sanitized) SVG as a vector passthrough. */
export const SVG_PASSTHROUGH = 'svg-passthrough' as const;

/** The crop frame the cropper presents over the source: the top-left of the square region in
 *  SOURCE pixels plus its side length (also in source pixels). Optional — when omitted the
 *  frame defaults to the centered largest-fitting square. */
export interface CropFrame {
	x: number;
	y: number;
	side: number;
}

/** The square master geometry: `size` is the exported square's edge length in pixels. */
export interface SquareCrop {
	size: number;
}

/**
 * Compute the square master size for a source of `srcW`×`srcH`.
 *
 * The cropped square's side is `frame.side` when a frame is supplied (the cropper's chosen
 * zoom/pan region), otherwise the largest square that fits the source (`min(srcW, srcH)`).
 * The exported `size` is that side clamped DOWN to {@link LOGO_MAX_DIMENSION} — never larger
 * than the cap and never larger than the source square (no upscaling).
 */
export function computeSquareCrop(srcW: number, srcH: number, frame?: CropFrame): SquareCrop {
	const sourceSquare = Math.min(srcW, srcH);
	const cropSide = frame ? Math.min(frame.side, sourceSquare) : sourceSquare;
	const size = Math.min(cropSide, LOGO_MAX_DIMENSION);
	return { size: Math.max(0, Math.floor(size)) };
}

/**
 * Choose the logo master output format from the source MIME (SC4 format law):
 *   - `image/png`     → `'image/png'`     (PNG kept so transparency survives)
 *   - `image/svg+xml` → {@link SVG_PASSTHROUGH} (sanitized vector passthrough, no rasterize)
 *   - anything else   → `'image/webp'`    (every other raster → WebP master)
 */
export function pickLogoOutputFormat(sourceMime: string): string {
	if (sourceMime === 'image/png') return 'image/png';
	if (sourceMime === 'image/svg+xml') return SVG_PASSTHROUGH;
	return 'image/webp';
}
