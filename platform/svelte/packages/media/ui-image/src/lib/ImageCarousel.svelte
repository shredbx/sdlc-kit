<script lang="ts">
	/**
	 * ImageCarousel Component
	 *
	 * Horizontal scrollable carousel for image collections.
	 * Used for property galleries, HOT DEALS sections, and R2 browser.
	 *
	 * @view-level blocks
	 * @foundation-type image
	 */

	import type { Snippet } from 'svelte';
	import type { ImageData, ImageAspectRatio } from '@sbx/core-ui/types';
	import ImageCard from './ImageCard.svelte';

	interface ImageCarouselProps {
		/** Array of images to display */
		images: ImageData[];
		/** Card width */
		cardWidth?: string;
		/** Aspect ratio for cards */
		aspectRatio?: ImageAspectRatio;
		/** Gap between cards */
		gap?: string;
		/** Show navigation arrows */
		showArrows?: boolean;
		/** Click handler per image */
		onImageClick?: (image: ImageData, index: number) => void;
		/** Custom card overlay */
		cardOverlay?: Snippet<[ImageData]>;
		/** Additional CSS classes */
		class?: string;
	}

	let {
		images,
		cardWidth = '280px',
		aspectRatio = '4:3',
		gap = '0.75rem',
		showArrows = true,
		onImageClick,
		cardOverlay,
		class: className = ''
	}: ImageCarouselProps = $props();

	let scrollContainer: HTMLElement;
	let canScrollLeft = $state(false);
	let canScrollRight = $state(true);

	function updateScrollState() {
		if (!scrollContainer) return;
		canScrollLeft = scrollContainer.scrollLeft > 0;
		canScrollRight = scrollContainer.scrollLeft < scrollContainer.scrollWidth - scrollContainer.clientWidth - 1;
	}

	function scroll(direction: 'left' | 'right') {
		if (!scrollContainer) return;
		const amount = scrollContainer.clientWidth * 0.8;
		scrollContainer.scrollBy({
			left: direction === 'left' ? -amount : amount,
			behavior: 'smooth'
		});
	}
</script>

<div class="image-carousel {className}">
	{#if showArrows && canScrollLeft}
		<button class="carousel-arrow carousel-arrow-left" onclick={() => scroll('left')} aria-label="Scroll left">
			<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="m15 18-6-6 6-6" />
			</svg>
		</button>
	{/if}

	<div
		class="carousel-track"
		bind:this={scrollContainer}
		onscroll={updateScrollState}
		style:gap={gap}
	>
		{#each images as image, index (image.id)}
			<div class="carousel-item" style:min-width={cardWidth} style:max-width={cardWidth}>
				<ImageCard
					{image}
					{aspectRatio}
					onclick={onImageClick ? () => onImageClick(image, index) : undefined}
				>
					{#snippet overlay()}
						{#if cardOverlay}
							{@render cardOverlay(image)}
						{/if}
					{/snippet}
				</ImageCard>
			</div>
		{/each}
	</div>

	{#if showArrows && canScrollRight}
		<button class="carousel-arrow carousel-arrow-right" onclick={() => scroll('right')} aria-label="Scroll right">
			<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="m9 18 6-6-6-6" />
			</svg>
		</button>
	{/if}
</div>

<style>
	.image-carousel {
		position: relative;
	}

	.carousel-track {
		display: flex;
		overflow-x: auto;
		scroll-snap-type: x mandatory;
		-webkit-overflow-scrolling: touch;
		scrollbar-width: none;
	}

	.carousel-track::-webkit-scrollbar {
		display: none;
	}

	.carousel-item {
		flex-shrink: 0;
		scroll-snap-align: start;
	}

	.carousel-arrow {
		position: absolute;
		top: 50%;
		transform: translateY(-50%);
		z-index: 2;
		display: flex;
		align-items: center;
		justify-content: center;
		width: 36px;
		height: 36px;
		border-radius: 50%;
		border: none;
		background: var(--carousel-arrow-bg, rgba(0, 0, 0, 0.6));
		color: var(--carousel-arrow-color, #fff);
		cursor: pointer;
		transition: background 0.2s ease;
	}

	.carousel-arrow:hover {
		background: var(--carousel-arrow-hover-bg, rgba(0, 0, 0, 0.8));
	}

	.carousel-arrow-left {
		left: 0.5rem;
	}

	.carousel-arrow-right {
		right: 0.5rem;
	}
</style>
