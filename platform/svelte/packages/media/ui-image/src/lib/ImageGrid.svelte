<script lang="ts">
	/**
	 * ImageGrid Component
	 *
	 * Responsive grid layout for displaying multiple images.
	 * Adapts column count based on container width.
	 *
	 * @view-level blocks
	 * @foundation-type image
	 */

	import type { Snippet } from 'svelte';
	import type { ImageData, ImageAspectRatio } from '@sbx/core-ui/types';
	import ImageCard from './ImageCard.svelte';

	interface ImageGridProps {
		/** Array of images to display */
		images: ImageData[];
		/** Number of columns (auto-responsive if not set) */
		columns?: number;
		/** Gap between items */
		gap?: string;
		/** Aspect ratio for cards */
		aspectRatio?: ImageAspectRatio;
		/** Click handler per image */
		onImageClick?: (image: ImageData) => void;
		/** Custom card overlay */
		cardOverlay?: Snippet<[ImageData]>;
		/** Additional CSS classes */
		class?: string;
		/** Empty state message */
		emptyMessage?: string;
	}

	let {
		images,
		columns,
		gap = '0.75rem',
		aspectRatio = '4:3',
		onImageClick,
		cardOverlay,
		class: className = '',
		emptyMessage = 'No images'
	}: ImageGridProps = $props();
</script>

{#if images.length === 0}
	<div class="image-grid-empty {className}">
		<p>{emptyMessage}</p>
	</div>
{:else}
	<div
		class="image-grid {className}"
		style:gap={gap}
		style:--grid-columns={columns ?? 'auto-fill'}
	>
		{#each images as image (image.id)}
			<ImageCard
				{image}
				{aspectRatio}
				onclick={onImageClick ? () => onImageClick(image) : undefined}
			>
				{#snippet overlay()}
					{#if cardOverlay}
						{@render cardOverlay(image)}
					{/if}
				{/snippet}
			</ImageCard>
		{/each}
	</div>
{/if}

<style>
	.image-grid {
		display: grid;
		grid-template-columns: repeat(var(--grid-columns, auto-fill), minmax(var(--image-grid-min-width, 200px), 1fr));
	}

	.image-grid-empty {
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: 200px;
		color: var(--color-text-muted, #666);
		font-size: 0.875rem;
	}
</style>
