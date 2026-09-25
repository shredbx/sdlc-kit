<script lang="ts">
	import type { ImageData } from '@sbx/core-ui/types';

	interface ImageLightboxProps {
		images: ImageData[];
		activeIndex?: number;
		open?: boolean;
		onclose?: () => void;
		class?: string;
	}

	let {
		images,
		activeIndex = 0,
		open = false,
		onclose,
		class: className = ''
	}: ImageLightboxProps = $props();

	// svelte-ignore state_referenced_locally
	let currentIndex = $state(activeIndex);

	$effect(() => {
		currentIndex = activeIndex;
	});

	let currentImage = $derived(images[currentIndex]);
	let hasPrev = $derived(currentIndex > 0);
	let hasNext = $derived(currentIndex < images.length - 1);

	// ── Zoom/pan — continuous, slider-driven ──
	// scale=1 = CSS-constrained fit (72vw × 64dvh).
	// panX/panY = screen-pixel offset from centered position.
	// transform: translate(pan) scale(s) with origin: center.
	let scale = $state(1);
	let panX = $state(0);
	let panY = $state(0);
	let isZoomed = $derived(scale > 1.01);
	let imageContainer = $state<HTMLElement>();
	let backdropEl = $state<HTMLElement>();
	let imgEl = $state<HTMLElement>();

	// Small-screen mode: the image goes full-bleed (native-gallery feel) and the fit
	// multipliers below must match the CSS breakpoint values. Tracked live so a rotate /
	// resize across the breakpoint re-fits.
	let isSmall = $state(false);
	$effect(() => {
		// Guarded: unavailable in SSR and in unit-test DOM environments.
		if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return;
		const mq = window.matchMedia('(max-width: 640px)');
		isSmall = mq.matches;
		const onChange = (e: MediaQueryListEvent) => (isSmall = e.matches);
		mq.addEventListener('change', onChange);
		return () => mq.removeEventListener('change', onChange);
	});

	// What % of native pixels the CSS-constrained fit shows (computed from
	// container + native dims — no dependency on imgEl.offsetWidth which is 0
	// until image loads). Multipliers MUST mirror .lightbox-image's max-width /
	// max-height per breakpoint (desktop 72vw × 64dvh; small screens full-bleed).
	let basePercent = $derived.by(() => {
		if (!currentImage || !imageContainer) return 100;
		const nativeW = currentImage.width || 1;
		const nativeH = currentImage.height || 1;
		const maxW = imageContainer.clientWidth * (isSmall ? 1 : 0.72);
		const maxH = imageContainer.clientHeight * (isSmall ? 1 : 0.64);
		const fitRatio = Math.min(maxW / nativeW, maxH / nativeH, 1);
		return Math.max(Math.round(fitRatio * 100), 1);
	});

	let nativeScale = $derived(basePercent > 0 ? 100 / basePercent : 2);
	let maxScale = $derived(Math.max(nativeScale * 2, 4));
	let zoomPercent = $derived(Math.round(basePercent * scale));

	function resetZoom() {
		scale = 1;
		panX = 0;
		panY = 0;
	}

	function zoomTo(newScale: number, cursorX?: number, cursorY?: number) {
		if (!imageContainer) return;
		const rect = imageContainer.getBoundingClientRect();
		const centerX = rect.left + rect.width / 2;
		const centerY = rect.top + rect.height / 2;
		const cx = (cursorX ?? centerX) - centerX;
		const cy = (cursorY ?? centerY) - centerY;

		const oldScale = scale;
		newScale = Math.min(Math.max(newScale, 1), maxScale);

		panX = panX + cx * (oldScale - newScale);
		panY = panY + cy * (oldScale - newScale);
		scale = newScale;

		clampPan();
	}

	function handleSliderInput(e: Event) {
		const value = parseFloat((e.target as HTMLInputElement).value);
		zoomTo(value);
	}

	function clampPan() {
		if (!imageContainer || !imgEl) return;
		const containerW = imageContainer.clientWidth;
		const containerH = imageContainer.clientHeight;
		const imgW = imgEl.offsetWidth || imgEl.clientWidth;
		const imgH = imgEl.offsetHeight || imgEl.clientHeight;

		const overflowX = Math.max(0, (imgW * scale - containerW) / 2);
		const overflowY = Math.max(0, (imgH * scale - containerH) / 2);

		panX = Math.min(Math.max(panX, -overflowX), overflowX);
		panY = Math.min(Math.max(panY, -overflowY), overflowY);
	}

	function prev() {
		if (hasPrev) { currentIndex--; resetZoom(); }
	}

	function next() {
		if (hasNext) { currentIndex++; resetZoom(); }
	}

	function close() {
		resetZoom();
		onclose?.();
	}

	// ── Pointer tracking ──
	let pointerCache: PointerEvent[] = [];
	let dragStartX = 0;
	let dragStartY = 0;
	let dragStartPanX = 0;
	let dragStartPanY = 0;
	let isDragging = $state(false);
	let isContinuousGesture = $state(false);
	let lastTapTime = 0;
	// A mouse drag ending on the backdrop still dispatches a `click` AFTER pointerup (when
	// isDragging is already reset) — without this stamp, a swipe released over the backdrop
	// would close the lightbox.
	let lastDragEndAt = 0;

	let initialPinchDist = 0;
	let initialPinchScale = 1;

	function getDistance(e1: PointerEvent, e2: PointerEvent): number {
		return Math.hypot(e1.clientX - e2.clientX, e1.clientY - e2.clientY);
	}

	function getMidpoint(e1: PointerEvent, e2: PointerEvent) {
		return { x: (e1.clientX + e2.clientX) / 2, y: (e1.clientY + e2.clientY) / 2 };
	}

	function handlePointerDown(e: PointerEvent) {
		if ((e.target as HTMLElement).closest('button, input')) return;

		pointerCache = [...pointerCache, e];
		(e.target as HTMLElement).setPointerCapture?.(e.pointerId);

		if (pointerCache.length === 1) {
			dragStartX = e.clientX;
			dragStartY = e.clientY;
			dragStartPanX = panX;
			dragStartPanY = panY;
			isDragging = false;
		}

		if (pointerCache.length === 2) {
			initialPinchDist = getDistance(pointerCache[0], pointerCache[1]);
			initialPinchScale = scale;
		}
	}

	function handlePointerMove(e: PointerEvent) {
		pointerCache = pointerCache.map(p => p.pointerId === e.pointerId ? e : p);

		if (pointerCache.length === 2) {
			isContinuousGesture = true;
			const dist = getDistance(pointerCache[0], pointerCache[1]);
			const mid = getMidpoint(pointerCache[0], pointerCache[1]);
			const newScale = initialPinchScale * (dist / initialPinchDist);
			zoomTo(newScale, mid.x, mid.y);
			return;
		}

		if (pointerCache.length === 1) {
			const dx = e.clientX - dragStartX;
			const dy = e.clientY - dragStartY;

			if (!isDragging && Math.hypot(dx, dy) > 5) {
				isDragging = true;
				isContinuousGesture = true;
			}

			if (isDragging && isZoomed) {
				panX = dragStartPanX + dx;
				panY = dragStartPanY + dy;
				clampPan();
			} else if (isDragging) {
				// Un-zoomed horizontal drag = swipe-in-progress: the image FOLLOWS the finger
				// (native-gallery feel). Release decides navigate vs snap back — no clampPan
				// here (it would zero the offset since an un-zoomed image has no overflow).
				panX = dx;
			}
		}
	}

	function handlePointerUp(e: PointerEvent) {
		pointerCache = pointerCache.filter(p => p.pointerId !== e.pointerId);

		if (pointerCache.length === 0) {
			if (!isDragging) {
				const now = Date.now();
				if (now - lastTapTime < 300) {
					if (isZoomed) resetZoom();
					else zoomTo(Math.min(nativeScale, maxScale), e.clientX, e.clientY);
					lastTapTime = 0;
				} else {
					lastTapTime = now;
				}
			} else {
				lastDragEndAt = Date.now();
				if (!isZoomed) {
					const dx = e.clientX - dragStartX;
					const dy = e.clientY - dragStartY;
					isContinuousGesture = false; // transition back on: navigate resets pan; snap-back animates
					if (Math.abs(dx) > 50 && Math.abs(dy) < Math.abs(dx)) {
						if (dx > 0) prev();
						else next();
					} else {
						panX = 0; // not a committed swipe — glide back to center
					}
				}
			}
			isDragging = false;
			isContinuousGesture = false;
		}
	}

	function handlePointerCancel(e: PointerEvent) {
		pointerCache = pointerCache.filter(p => p.pointerId !== e.pointerId);
		isDragging = false;
		isContinuousGesture = false;
	}

	function handleWheel(e: WheelEvent) {
		if (!e.ctrlKey && !e.metaKey) return;
		e.preventDefault();
		isContinuousGesture = true;
		const factor = e.deltaY > 0 ? 0.9 : 1.1;
		zoomTo(scale * factor, e.clientX, e.clientY);
		clearTimeout(wheelEndTimer);
		wheelEndTimer = setTimeout(() => { isContinuousGesture = false; }, 150);
	}

	let wheelEndTimer: ReturnType<typeof setTimeout>;

	function handleKeydown(e: KeyboardEvent) {
		if (!open) return;
		if (e.key === 'Escape') close();
		if (e.key === 'ArrowLeft') prev();
		if (e.key === 'ArrowRight') next();
		if (e.key === '+' || e.key === '=') { e.preventDefault(); zoomTo(scale * 1.2); }
		if (e.key === '-') { e.preventDefault(); zoomTo(scale / 1.2); }
		if (e.key === '0') { e.preventDefault(); resetZoom(); }
	}

	function handleBackdropClick(e: MouseEvent) {
		if (isDragging || Date.now() - lastDragEndAt < 400) return;
		// While zoomed, a click on the dark margin is part of a zoom/pan interaction,
		// and a double-tap-to-zoom also emits a trailing click — neither should dismiss.
		if (isZoomed) return;
		// Close when the click lands on the dark area — the backdrop itself OR the
		// padded image stage (.lightbox-content) that fills it. Clicking the image,
		// toolbar, nav arrows or footer is NOT a dismiss (those targets carry their own
		// classes). Previously only `.lightbox-backdrop` matched, but .lightbox-content
		// (flex:1, full width) covers the whole dark area, so the guard never passed and
		// backdrop-click-to-close was dead.
		const target = e.target as HTMLElement;
		if (
			target.classList.contains('lightbox-backdrop') ||
			target.classList.contains('lightbox-content')
		) {
			close();
		}
	}

	$effect(() => {
		if (open) {
			document.body.style.overflow = 'hidden';
		}
		return () => {
			document.body.style.overflow = '';
		};
	});

	$effect(() => {
		if (!open) return;
		const toPreload = [currentIndex - 1, currentIndex + 1]
			.filter(i => i >= 0 && i < images.length);

		const preloaded = toPreload.map(i => {
			const img = new Image();
			img.src = images[i].url;
			return img;
		});

		return () => { preloaded.forEach(img => { img.src = ''; }); };
	});

	$effect(() => {
		if (!open) return;
		if (backdropEl) backdropEl.focus();
	});
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open && currentImage}
	<!-- Keyboard operation is handled at the window level (Escape/arrows via
	     handleKeydown on <svelte:window>); the backdrop onclick is a redundant
	     mouse affordance for dismiss, so an element-level keydown isn't needed. -->
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<div
		bind:this={backdropEl}
		class="lightbox-backdrop {className}"
		role="dialog"
		aria-modal="true"
		aria-label="Image viewer, showing {currentIndex + 1} of {images.length}"
		tabindex="-1"
		onclick={handleBackdropClick}
		onpointerdown={handlePointerDown}
		onpointermove={handlePointerMove}
		onpointerup={handlePointerUp}
		onpointercancel={handlePointerCancel}
		onwheel={handleWheel}
	>
		<div class="lightbox-toolbar">
			<div class="lightbox-zoom-control">
				<button
					class="lightbox-btn lightbox-btn-sm"
					onclick={() => zoomTo(scale / 1.3)}
					disabled={scale <= 1.01}
					aria-label="Zoom out"
				>
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
						<path d="M5 12h14" />
					</svg>
				</button>

				<input
					type="range"
					class="lightbox-slider"
					min="1"
					max={maxScale}
					step="0.01"
					value={scale}
					oninput={handleSliderInput}
					aria-label="Zoom level {zoomPercent}%"
				/>

				<button
					class="lightbox-btn lightbox-btn-sm"
					onclick={() => zoomTo(scale * 1.3)}
					disabled={scale >= maxScale - 0.01}
					aria-label="Zoom in"
				>
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
						<path d="M12 5v14" /><path d="M5 12h14" />
					</svg>
				</button>

				<span class="lightbox-zoom-label">{zoomPercent}%</span>
			</div>

			<button class="lightbox-btn" onclick={close} aria-label="Close">
				<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M18 6 6 18" /><path d="m6 6 12 12" />
				</svg>
			</button>
		</div>

		{#if hasPrev}
			<button class="lightbox-nav lightbox-prev" onclick={prev} aria-label="Previous image">
				<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="m15 18-6-6 6-6" />
				</svg>
			</button>
		{/if}

		{#if hasNext}
			<button class="lightbox-nav lightbox-next" onclick={next} aria-label="Next image">
				<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="m9 18 6-6-6-6" />
				</svg>
			</button>
		{/if}

		<div class="lightbox-content" bind:this={imageContainer}>
			<img
				bind:this={imgEl}
				src={currentImage.url}
				alt={currentImage.alt_text}
				class="lightbox-image"
				class:no-transition={isContinuousGesture}
				draggable="false"
				style:transform="translate({panX}px, {panY}px) scale({scale})"
				style:cursor={isZoomed ? (isDragging ? 'grabbing' : 'grab') : 'zoom-in'}
			/>
		</div>

		<div class="lightbox-footer">
			{#if currentImage.alt_text}
				<p class="lightbox-caption">{currentImage.alt_text}</p>
			{/if}
			<span class="lightbox-counter">
				{currentIndex + 1} / {images.length}
			</span>
		</div>

		<div class="sr-only" aria-live="polite" aria-atomic="true">
			Image {currentIndex + 1} of {images.length}
			{#if currentImage.alt_text}, {currentImage.alt_text}{/if}
		</div>
	</div>
{/if}

<style>
	.lightbox-backdrop {
		position: fixed;
		inset: 0;
		z-index: 9999;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		background: var(--lightbox-bg, rgba(0, 0, 0, 0.92));
		overscroll-behavior: contain;
	}

	.lightbox-toolbar {
		position: absolute;
		top: 0.75rem;
		right: 0.75rem;
		z-index: 10;
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.lightbox-zoom-control {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		background: rgba(0, 0, 0, 0.5);
		backdrop-filter: blur(8px);
		-webkit-backdrop-filter: blur(8px);
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 10px;
		padding: 0 0.5rem;
		height: 44px;
	}

	.lightbox-zoom-label {
		font-family: var(--font-mono, monospace);
		font-size: 0.6875rem;
		color: rgba(255, 255, 255, 0.6);
		min-width: 3ch;
		text-align: right;
		white-space: nowrap;
	}

	.lightbox-slider {
		-webkit-appearance: none;
		appearance: none;
		width: 80px;
		height: 3px;
		background: rgba(255, 255, 255, 0.2);
		border-radius: 2px;
		outline: none;
		cursor: pointer;
	}

	.lightbox-slider::-webkit-slider-thumb {
		-webkit-appearance: none;
		appearance: none;
		width: 14px;
		height: 14px;
		border-radius: 50%;
		background: rgba(255, 255, 255, 0.85);
		border: none;
		cursor: pointer;
		transition: transform 0.1s;
	}

	.lightbox-slider::-webkit-slider-thumb:hover {
		transform: scale(1.2);
	}

	.lightbox-slider::-moz-range-thumb {
		width: 14px;
		height: 14px;
		border-radius: 50%;
		background: rgba(255, 255, 255, 0.85);
		border: none;
		cursor: pointer;
	}

	.lightbox-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 44px;
		height: 44px;
		border-radius: 10px;
		border: 1px solid rgba(255, 255, 255, 0.1);
		background: rgba(0, 0, 0, 0.5);
		backdrop-filter: blur(8px);
		-webkit-backdrop-filter: blur(8px);
		color: rgba(255, 255, 255, 0.85);
		cursor: pointer;
		transition: background 0.15s, border-color 0.15s;
	}

	.lightbox-btn:hover {
		background: rgba(0, 0, 0, 0.7);
		border-color: rgba(255, 255, 255, 0.2);
		color: #fff;
	}

	.lightbox-btn:active {
		background: rgba(255, 255, 255, 0.12);
	}

	.lightbox-btn:disabled {
		opacity: 0.3;
		cursor: default;
	}

	.lightbox-btn-sm {
		width: 28px;
		height: 28px;
		border-radius: 6px;
		border: none;
		background: transparent;
		backdrop-filter: none;
		-webkit-backdrop-filter: none;
	}

	.lightbox-btn-sm:hover:not(:disabled) {
		background: rgba(255, 255, 255, 0.1);
	}

	.lightbox-content {
		position: relative;
		display: flex;
		align-items: center;
		justify-content: center;
		flex: 1;
		width: 100%;
		padding: 2rem;
		touch-action: none;
		user-select: none;
		-webkit-user-select: none;
	}

	.lightbox-image {
		max-width: 72vw;
		max-height: 64dvh;
		object-fit: contain;
		border-radius: 0.25rem;
		transform-origin: center center;
		will-change: transform;
		transition: transform 0.2s cubic-bezier(0.25, 0.46, 0.45, 0.94);
		-webkit-user-drag: none;
	}

	.lightbox-image.no-transition {
		transition: none;
	}

	.lightbox-nav {
		position: absolute;
		top: 50%;
		transform: translateY(-50%);
		z-index: 10;
		display: flex;
		align-items: center;
		justify-content: center;
		width: 44px;
		height: 44px;
		border-radius: 50%;
		border: 1px solid rgba(255, 255, 255, 0.1);
		background: rgba(0, 0, 0, 0.5);
		backdrop-filter: blur(8px);
		-webkit-backdrop-filter: blur(8px);
		color: rgba(255, 255, 255, 0.85);
		cursor: pointer;
		transition: background 0.15s, border-color 0.15s;
	}

	.lightbox-nav:hover {
		background: rgba(0, 0, 0, 0.7);
		border-color: rgba(255, 255, 255, 0.2);
		color: #fff;
	}

	.lightbox-prev { left: 0.75rem; }
	.lightbox-next { right: 0.75rem; }

	.lightbox-footer {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 1rem;
		padding: 1rem;
		color: rgba(255, 255, 255, 0.8);
		font-size: 0.875rem;
	}

	.lightbox-caption {
		margin: 0;
	}

	.lightbox-counter {
		opacity: 0.6;
	}

	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		margin: -1px;
		overflow: hidden;
		clip: rect(0, 0, 0, 0);
		white-space: nowrap;
		border-width: 0;
	}

	/* Small screens — native-gallery mode. The image fills the viewport width (the fit
	   multipliers in the script mirror these bounds); navigation is the swipe gesture,
	   zoom is pinch / double-tap, so the desktop chrome (edge arrows, zoom slider) goes.
	   Close + the counter footer remain. */
	@media (max-width: 640px) {
		.lightbox-content {
			padding: 0;
		}

		.lightbox-image {
			max-width: 100vw;
			max-height: 100%;
			border-radius: 0;
		}

		.lightbox-zoom-control {
			display: none;
		}

		.lightbox-nav {
			display: none;
		}
	}
</style>
