<script lang="ts">
	// SquareLogoCropper — the thin DOM/canvas shell over the PURE cropExport.ts geometry +
	// format law (task 2606-118). A raster brand logo is loaded into a SQUARE frame the user
	// zooms/pans; on confirm the chosen square region is drawn to an offscreen canvas sized by
	// computeSquareCrop (longest edge <= 512, never upscaled) and exported as a Blob in the
	// source-derived format (pickLogoOutputFormat: PNG-in keeps PNG+alpha, every other raster
	// -> WebP). SVG is NOT cropped here — the upload helper sanitizes + passes the vector
	// through upstream and never opens this cropper for an SVG (the SVG_PASSTHROUGH branch is a
	// defensive no-rasterize guard only).
	//
	// Theme-neutral like the rest of @sbx/ui-image: no brand colours, inherits currentColor and
	// reads --color-* custom properties with neutral fallbacks. The CALLER owns the surrounding
	// chrome; this component owns only the crop interaction + the canvas export, delegating ALL
	// sizing/format decisions to the unit-tested cropExport functions.
	import {
		computeSquareCrop,
		pickLogoOutputFormat,
		SVG_PASSTHROUGH,
		type CropFrame
	} from './cropExport';

	/** The exported square master handed back to the caller via `oncrop`. */
	export interface CroppedLogo {
		blob: Blob;
		width: number;
		height: number;
		format: string;
	}

	interface SquareLogoCropperProps {
		/** Object/data URL of the source raster to crop. */
		src: string;
		/** Source MIME — drives the export format (PNG keeps alpha, else WebP). */
		mime?: string;
		/** WebP/JPEG export quality (0–1). Ignored for lossless PNG export. */
		quality?: number;
		/** Confirm — returns the square master blob + its pixel dimensions + MIME. */
		oncrop: (result: CroppedLogo) => void;
		/** Cancel — the caller dismisses the cropper, nothing is exported. */
		oncancel?: () => void;
		/** Optional accessible label for the crop surface. */
		label?: string;
		class?: string;
	}

	let {
		src,
		mime = 'image/png',
		quality = 0.9,
		oncrop,
		oncancel,
		label = 'Crop logo',
		class: className = ''
	}: SquareLogoCropperProps = $props();

	// The square viewport edge in CSS px (the on-screen crop window). The exported master is
	// independently sized by computeSquareCrop in SOURCE pixels — this is only the UI frame.
	const VIEWPORT = 320;

	let image = $state<HTMLImageElement | null>(null);
	let naturalW = $state(0);
	let naturalH = $state(0);
	// zoom = 1 means the image is scaled to exactly COVER the square viewport; >1 zooms in.
	let zoom = $state(1);
	// Pan offset of the image's top-left relative to the viewport, in viewport CSS px.
	let offsetX = $state(0);
	let offsetY = $state(0);
	let exporting = $state(false);

	let dragging = false;
	let dragStartX = 0;
	let dragStartY = 0;

	// Base scale: the factor that makes the source COVER the square viewport (the smaller of
	// the source's two edges fills the frame). `zoom` multiplies this.
	const baseScale = $derived(naturalW && naturalH ? VIEWPORT / Math.min(naturalW, naturalH) : 1);
	const drawScale = $derived(baseScale * zoom);
	const drawW = $derived(naturalW * drawScale);
	const drawH = $derived(naturalH * drawScale);

	// Constrain the pan so the viewport square is always fully covered by the image (no gaps).
	function clampOffsets() {
		const minX = VIEWPORT - drawW;
		const minY = VIEWPORT - drawH;
		offsetX = Math.min(0, Math.max(minX, offsetX));
		offsetY = Math.min(0, Math.max(minY, offsetY));
	}

	function onImageLoad(e: Event) {
		const el = e.currentTarget as HTMLImageElement;
		image = el;
		naturalW = el.naturalWidth;
		naturalH = el.naturalHeight;
		zoom = 1;
		// Center the covered image in the viewport.
		offsetX = (VIEWPORT - drawW) / 2;
		offsetY = (VIEWPORT - drawH) / 2;
		clampOffsets();
	}

	function onZoom(e: Event) {
		zoom = Number((e.currentTarget as HTMLInputElement).value);
		clampOffsets();
	}

	function onPointerDown(e: PointerEvent) {
		dragging = true;
		dragStartX = e.clientX - offsetX;
		dragStartY = e.clientY - offsetY;
		(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
	}

	function onPointerMove(e: PointerEvent) {
		if (!dragging) return;
		offsetX = e.clientX - dragStartX;
		offsetY = e.clientY - dragStartY;
		clampOffsets();
	}

	function onPointerUp(e: PointerEvent) {
		dragging = false;
		(e.currentTarget as HTMLElement).releasePointerCapture(e.pointerId);
	}

	// Map the on-screen square viewport back to a CROP FRAME in source pixels: the viewport's
	// top-left corner in image space, and its side length in source pixels.
	function viewportToSourceFrame(): CropFrame {
		const sourceSide = VIEWPORT / drawScale; // viewport edge measured in source pixels
		const sx = -offsetX / drawScale;
		const sy = -offsetY / drawScale;
		return { x: sx, y: sy, side: sourceSide };
	}

	async function confirm() {
		if (!image || exporting) return;
		const format = pickLogoOutputFormat(mime);
		// Defensive: this cropper never rasterizes an SVG (the helper handles SVG upstream and
		// won't open the cropper for one). If somehow asked, fall back to lossless PNG export.
		const outputMime = format === SVG_PASSTHROUGH ? 'image/png' : format;

		const frame = viewportToSourceFrame();
		const { size } = computeSquareCrop(naturalW, naturalH, frame);

		const canvas = document.createElement('canvas');
		canvas.width = size;
		canvas.height = size;
		const ctx = canvas.getContext('2d');
		if (!ctx) return;
		// Draw the chosen square source region into the full square canvas (cover, no letterbox).
		ctx.drawImage(image, frame.x, frame.y, frame.side, frame.side, 0, 0, size, size);

		exporting = true;
		try {
			const blob = await new Promise<Blob | null>((resolve) =>
				canvas.toBlob(resolve, outputMime, quality)
			);
			if (blob) oncrop({ blob, width: size, height: size, format: outputMime });
		} finally {
			exporting = false;
		}
	}
</script>

<div class="logo-cropper {className}">
	<!-- The hidden source image drives natural dimensions; the visible crop happens in the
	     framed viewport below via CSS transform (the canvas re-renders it on confirm). -->
	<img class="logo-cropper__source" {src} alt="" onload={onImageLoad} />

	<div
		class="logo-cropper__viewport"
		style="width:{VIEWPORT}px;height:{VIEWPORT}px"
		role="application"
		aria-label={label}
		onpointerdown={onPointerDown}
		onpointermove={onPointerMove}
		onpointerup={onPointerUp}
		onpointercancel={onPointerUp}
	>
		{#if image}
			<img
				class="logo-cropper__preview"
				{src}
				alt=""
				style="width:{drawW}px;height:{drawH}px;transform:translate({offsetX}px,{offsetY}px)"
				draggable="false"
			/>
		{/if}
		<div class="logo-cropper__mask" aria-hidden="true"></div>
	</div>

	<label class="logo-cropper__zoom">
		<span class="logo-cropper__zoom-label">Zoom</span>
		<input
			type="range"
			min="1"
			max="3"
			step="0.01"
			value={zoom}
			oninput={onZoom}
			aria-label="Zoom"
		/>
	</label>

	<div class="logo-cropper__actions">
		<button type="button" class="logo-cropper__btn" onclick={() => oncancel?.()} disabled={exporting}>
			Cancel
		</button>
		<button
			type="button"
			class="logo-cropper__btn logo-cropper__btn--primary"
			onclick={confirm}
			disabled={!image || exporting}
			aria-busy={exporting}
		>
			{exporting ? 'Saving…' : 'Use logo'}
		</button>
	</div>
</div>

<style>
	.logo-cropper {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 1rem;
		color: inherit;
	}
	/* The source image only supplies natural dimensions — never shown. */
	.logo-cropper__source {
		display: none;
	}
	.logo-cropper__viewport {
		position: relative;
		overflow: hidden;
		border-radius: var(--radius-md, 8px);
		background: var(--color-bg-muted, #f3f4f6);
		cursor: grab;
		touch-action: none;
		user-select: none;
	}
	.logo-cropper__viewport:active {
		cursor: grabbing;
	}
	.logo-cropper__preview {
		position: absolute;
		top: 0;
		left: 0;
		transform-origin: top left;
		max-width: none;
		pointer-events: none;
	}
	/* A subtle inset ring marking the square export region. */
	.logo-cropper__mask {
		position: absolute;
		inset: 0;
		box-shadow: inset 0 0 0 1px color-mix(in srgb, currentColor 25%, transparent);
		pointer-events: none;
	}
	.logo-cropper__zoom {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		width: 100%;
		max-width: 320px;
		font-size: 0.8125rem;
	}
	.logo-cropper__zoom input[type='range'] {
		flex: 1;
		accent-color: var(--color-accent, currentColor);
	}
	.logo-cropper__actions {
		display: flex;
		justify-content: flex-end;
		gap: 0.5rem;
		width: 100%;
		max-width: 320px;
	}
	.logo-cropper__btn {
		padding: 0.4rem 0.9rem;
		border-radius: var(--radius-sm, 6px);
		border: 1px solid var(--color-border, currentColor);
		background: transparent;
		color: inherit;
		font: inherit;
		cursor: pointer;
	}
	.logo-cropper__btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
	.logo-cropper__btn--primary {
		border-color: var(--color-accent, currentColor);
		background: var(--color-accent, currentColor);
		color: var(--color-accent-contrast, #fff);
	}
</style>
