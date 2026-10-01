// The @sbx/canvas-ui implementation of the kit's injected `Rasterizer` (S-EXPORT).
// The kit is HEADLESS — it resolves a frame (resolveFrame, inside exportPage) but
// cannot touch a canvas; this module owns the canvas renderer, so the kit stays
// DOM-free. It reuses the SAME kit renderer the live stage uses (renderPage) — no
// duplicate renderer (cf. land-canvas, which re-implemented one inside export).
//
// Images are loaded CORS-clean (crossOrigin + the consumer's same-origin
// resolveImageSrc proxy) so the offscreen canvas is NOT tainted and toBlob/
// toDataURL succeed. The kit renderer is synchronous and draws from a preloaded
// cache, so we preload every image src FIRST (await), then render in one pass.

import {
	renderPage,
	resolveFrame,
	IMAGE_FAILED,
	type Layer,
	type Page,
	type Document,
	type FormatterRegistry,
	type ExportTarget,
	type Rasterizer,
	type ImageSource,
	type ImageSourceResult
} from '@sbx/canvas-kit';

/** Load one image CORS-clean. Resolves to the element on load, or null on error
 *  (the layer then renders the explicit broken-image state — never a silent blank). */
function loadImage(url: string): Promise<HTMLImageElement | null> {
	return new Promise((resolve) => {
		const img = new Image();
		img.crossOrigin = 'anonymous';
		img.onload = () => resolve(img);
		img.onerror = () => resolve(null);
		img.src = url;
	});
}

/** Preload every distinct image-layer src (routed through resolveSrc) into a sync
 *  cache keyed by the ORIGINAL src — that's what renderPage's ImageSource is called
 *  with (layer.src). Failed loads are cached as IMAGE_FAILED so the export paints
 *  the kit's broken-image state (I-2) — a failed asset must never export as a
 *  silent blank the user only discovers after sharing the file. */
async function preloadImages(layers: Layer[], resolveSrc: (src: string) => string): Promise<ImageSource> {
	const srcs = [...new Set(layers.filter((l) => l.type === 'image' && l.src).map((l) => l.src as string))];
	const cache = new Map<string, ImageSourceResult>();
	await Promise.all(
		srcs.map(async (src) => {
			const img = await loadImage(resolveSrc(src));
			cache.set(src, img ?? IMAGE_FAILED);
		})
	);
	return (src: string) => cache.get(src);
}

const MIME: Record<ExportTarget['format'], string> = {
	png: 'image/png',
	jpeg: 'image/jpeg',
	pdf: 'image/png' // PDF embeds a PNG snapshot
};

/**
 * Live-render a document's FIRST page into a (small) on-screen canvas — the template
 * gallery preview (#4). Reuses the EXACT export pipeline (resolveFrame → preload →
 * renderPage) so a template card is byte-identical to what the editor/export produce,
 * just scaled to fit the canvas (no snapshot file, no duplicate renderer). Bindings
 * resolve against the doc's own sources (templates carry refs, NOT live snapshots — D20
 * — so bound fields fall back to their fallback text, which is the honest template
 * preview). Returns false when there's nothing to draw (no page / no 2D context).
 */
export async function renderDocPreview(
	canvas: HTMLCanvasElement,
	doc: Document,
	resolveImageSrc: (src: string) => string = (s) => s,
	formatters: FormatterRegistry = {}
): Promise<boolean> {
	const page = doc.pages?.[0];
	const ctx = canvas.getContext('2d');
	if (!page || !ctx || !page.width || !page.height) return false;
	const layers = resolveFrame(page, 0, doc.sources ?? [], formatters, false);
	// Contain the page within the canvas's pixel box (DPR already baked into width/height).
	const scale = Math.min(canvas.width / page.width, canvas.height / page.height);
	ctx.clearRect(0, 0, canvas.width, canvas.height);
	ctx.save();
	ctx.scale(scale, scale);
	const images = await preloadImages(layers, resolveImageSrc);
	renderPage(ctx, page, layers, images);
	ctx.restore();
	return true;
}

function canvasToBlob(canvas: HTMLCanvasElement, mime: string, quality?: number): Promise<Blob> {
	return new Promise((resolve, reject) => {
		canvas.toBlob(
			(blob) => (blob ? resolve(blob) : reject(new Error('canvas.toBlob returned null'))),
			mime,
			quality
		);
	});
}

/**
 * Build the kit `Rasterizer`: resolved layers → offscreen 2D canvas → PNG/JPEG
 * (toBlob) or PDF (jsPDF, lazy-imported). `resolveImageSrc` routes layer images
 * through the consumer's same-origin proxy so the canvas is export-clean; the
 * default identity is fine for an image-free or already-same-origin doc.
 */
export function createRasterizer(resolveImageSrc: (src: string) => string = (s) => s): Rasterizer {
	return async (layers: Layer[], page: Page, target: ExportTarget): Promise<Blob> => {
		const scale = target.retina ? 2 : 1;
		const canvas = document.createElement('canvas');
		canvas.width = target.width * scale;
		canvas.height = target.height * scale;
		const ctx = canvas.getContext('2d');
		if (!ctx) throw new Error('canvas export: failed to acquire a 2D context');
		ctx.scale(scale, scale);

		// JPEG/PDF cannot be transparent → white floor-fill (a page background layer,
		// if present, paints over it). PNG keeps transparency.
		if (target.format !== 'png') {
			ctx.fillStyle = '#ffffff';
			ctx.fillRect(0, 0, target.width, target.height);
		}

		const images = await preloadImages(layers, resolveImageSrc);
		renderPage(ctx, page, layers, images);

		if (target.format === 'pdf') {
			const { jsPDF } = await import('jspdf');
			const orientation = target.width >= target.height ? 'landscape' : 'portrait';
			const pdf = new jsPDF({ orientation, unit: 'px', format: [target.width, target.height] });
			pdf.addImage(canvas.toDataURL('image/png'), 'PNG', 0, 0, target.width, target.height);
			return pdf.output('blob');
		}

		const quality =
			target.format === 'jpeg' ? (target.quality != null ? target.quality / 100 : 0.92) : undefined;
		return canvasToBlob(canvas, MIME[target.format], quality);
	};
}
