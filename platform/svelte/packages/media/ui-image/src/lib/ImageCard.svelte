<script lang="ts">
	/**
	 * ImageCard Component
	 *
	 * Displays an image as a card with optional overlay text.
	 * Supports aspect ratios, lazy loading, and click-to-expand.
	 *
	 * @view-level blocks
	 * @foundation-type image
	 */

	import type { Snippet } from 'svelte';
	import type { ImageData, ImageAspectRatio } from '@sbx/core-ui/types';

	interface ImageCardProps {
		/** Image data from API */
		image?: ImageData;
		/** Direct URL (alternative to image object) */
		src?: string;
		/** Alt text (falls back to image.alt_text) */
		alt?: string;
		/** Aspect ratio */
		aspectRatio?: ImageAspectRatio;
		/** Width constraint in pixels */
		width?: number;
		/** Enable lazy loading */
		lazy?: boolean;
		/** Click handler */
		onclick?: (image?: ImageData) => void;
		/** Additional CSS classes */
		class?: string;
		/** Overlay content (title, price, etc.) */
		overlay?: Snippet;
		/** IView data attribute */
		'data-view-id'?: string;
	}

	let {
		image,
		src,
		alt,
		aspectRatio = '4:3',
		width,
		lazy = true,
		onclick,
		class: className = '',
		overlay,
		'data-view-id': viewId
	}: ImageCardProps = $props();

	let resolvedSrc = $derived(src ?? image?.url ?? '');
	let resolvedAlt = $derived(alt ?? image?.alt_text ?? '');
	let isLoaded = $state(false);
	let hasError = $state(false);

	function handleClick() {
		onclick?.(image);
	}

	function handleLoad() {
		isLoaded = true;
	}

	function handleError() {
		hasError = true;
	}

	let ratioClass = $derived(aspectRatio === 'auto' ? '' : `ratio-${aspectRatio.replace(':', '-')}`);
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<div
	class="image-card {ratioClass} {className}"
	class:clickable={!!onclick}
	class:loaded={isLoaded}
	class:error={hasError}
	style:width={width ? `${width}px` : undefined}
	data-view-id={viewId}
	role={onclick ? 'button' : undefined}
	tabindex={onclick ? 0 : undefined}
	onclick={handleClick}
	onkeydown={onclick ? (e) => e.key === 'Enter' && handleClick() : undefined}
>
	{#if resolvedSrc && !hasError}
		<img
			src={resolvedSrc}
			alt={resolvedAlt}
			loading={lazy ? 'lazy' : 'eager'}
			onload={handleLoad}
			onerror={handleError}
			class="image-card-img"
		/>
	{:else}
		<div class="image-card-placeholder">
			<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<rect x="3" y="3" width="18" height="18" rx="2" />
				<circle cx="8.5" cy="8.5" r="1.5" />
				<path d="m21 15-5-5L5 21" />
			</svg>
		</div>
	{/if}

	{#if overlay && isLoaded}
		<div class="image-card-overlay">
			{@render overlay()}
		</div>
	{/if}
</div>

<style>
	.image-card {
		position: relative;
		overflow: hidden;
		border-radius: var(--image-card-radius, 0.5rem);
		background: var(--image-card-bg, var(--color-surface-2, #1a1a1b));
	}

	.image-card.clickable {
		cursor: pointer;
	}

	.image-card.clickable:hover .image-card-img {
		transform: scale(1.05);
	}

	.image-card.clickable:focus-visible {
		outline: 2px solid var(--color-accent, #6366f1);
		outline-offset: 2px;
	}

	/* Aspect ratios */
	.ratio-1-1 { aspect-ratio: 1 / 1; }
	.ratio-4-3 { aspect-ratio: 4 / 3; }
	.ratio-16-9 { aspect-ratio: 16 / 9; }
	.ratio-3-2 { aspect-ratio: 3 / 2; }
	.ratio-2-3 { aspect-ratio: 2 / 3; }

	.image-card-img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		transition: transform 0.3s ease;
	}

	.image-card-placeholder {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 100%;
		height: 100%;
		min-height: 120px;
		color: var(--color-text-muted, #666);
	}

	.image-card-overlay {
		position: absolute;
		bottom: 0;
		left: 0;
		right: 0;
		padding: var(--image-card-overlay-padding, 0.75rem);
		background: var(--image-card-overlay-bg, linear-gradient(transparent, rgba(0, 0, 0, 0.7)));
		color: var(--image-card-overlay-text, #fff);
	}
</style>
