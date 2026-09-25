<script lang="ts">
	/**
	 * Skeleton Component
	 *
	 * Placeholder loading state component for SSR-first pattern.
	 * Shows animated placeholder while content loads.
	 *
	 * Features:
	 * - Multiple animation styles (pulse, wave, none)
	 * - Configurable dimensions
	 * - Reduced motion support
	 * - Composable for complex layouts
	 *
	 * @example
	 * <Skeleton width="100%" height="40px" animation="pulse" />
	 *
	 * @example
	 * <Skeleton variant="text" lines={3} />
	 *
	 * @example
	 * <Skeleton variant="card" />
	 */

	// =============================================================================
	// TYPES
	// =============================================================================

	export type SkeletonVariant = 'rectangle' | 'circle' | 'text' | 'card' | 'list';
	export type SkeletonAnimation = 'pulse' | 'wave' | 'none';

	export interface SkeletonProps {
		/** Predefined variant */
		variant?: SkeletonVariant;
		/** Width (CSS value) */
		width?: string;
		/** Height (CSS value) */
		height?: string;
		/** Animation style */
		animation?: SkeletonAnimation;
		/** Border radius (CSS value) */
		rounded?: string;
		/** Number of lines for text variant */
		lines?: number;
		/** Number of items for list variant */
		count?: number;
		/** Gap between items */
		gap?: string;
		/** Additional CSS class */
		class?: string;
	}

	// =============================================================================
	// PROPS
	// =============================================================================

	let {
		variant = 'rectangle',
		width = '100%',
		height = '20px',
		animation = 'pulse',
		rounded = '4px',
		lines = 3,
		count = 3,
		gap = '8px',
		class: className = ''
	}: SkeletonProps = $props();

	// =============================================================================
	// DERIVED STYLES
	// =============================================================================

	let baseStyle = $derived(
		variant === 'circle'
			? `width: ${width}; height: ${width}; border-radius: 50%;`
			: `width: ${width}; height: ${height}; border-radius: ${rounded};`
	);

	// Text variant generates multiple lines with varying widths
	let textLines = $derived(
		variant === 'text'
			? Array.from({ length: lines }, (_, i) => ({
					id: i,
					width: i === lines - 1 ? '60%' : i === 0 ? '90%' : '100%'
				}))
			: []
	);

	// List variant generates multiple items
	let listItems = $derived(
		variant === 'list'
			? Array.from({ length: count }, (_, i) => ({ id: i }))
			: []
	);
</script>

{#if variant === 'text'}
	<div class="skeleton-text" style="gap: {gap};">
		{#each textLines as line (line.id)}
			<div
				class="skeleton skeleton-{animation} {className}"
				style="width: {line.width}; height: {height}; border-radius: {rounded};"
			></div>
		{/each}
	</div>
{:else if variant === 'list'}
	<div class="skeleton-list" style="gap: {gap};">
		{#each listItems as item (item.id)}
			<div class="skeleton-list-item">
				<div
					class="skeleton skeleton-{animation}"
					style="width: 40px; height: 40px; border-radius: 50%; flex-shrink: 0;"
				></div>
				<div class="skeleton-list-content">
					<div
						class="skeleton skeleton-{animation}"
						style="width: 60%; height: 14px; border-radius: 4px;"
					></div>
					<div
						class="skeleton skeleton-{animation}"
						style="width: 40%; height: 12px; border-radius: 4px;"
					></div>
				</div>
			</div>
		{/each}
	</div>
{:else if variant === 'card'}
	<div class="skeleton-card {className}">
		<div
			class="skeleton skeleton-{animation}"
			style="width: 100%; height: 120px; border-radius: {rounded} {rounded} 0 0;"
		></div>
		<div class="skeleton-card-content">
			<div
				class="skeleton skeleton-{animation}"
				style="width: 70%; height: 16px; border-radius: 4px;"
			></div>
			<div
				class="skeleton skeleton-{animation}"
				style="width: 100%; height: 12px; border-radius: 4px;"
			></div>
			<div
				class="skeleton skeleton-{animation}"
				style="width: 50%; height: 12px; border-radius: 4px;"
			></div>
		</div>
	</div>
{:else}
	<div
		class="skeleton skeleton-{animation} {className}"
		style={baseStyle}
	></div>
{/if}

<style>
	/* ==========================================================================
	   BASE SKELETON
	   ========================================================================== */
	.skeleton {
		background: var(--color-skeleton-bg, rgba(255, 255, 255, 0.08));
		position: relative;
		overflow: hidden;
	}

	/* ==========================================================================
	   PULSE ANIMATION
	   ========================================================================== */
	.skeleton-pulse {
		animation: skeleton-pulse 1.5s ease-in-out infinite;
	}

	@keyframes skeleton-pulse {
		0%,
		100% {
			opacity: 1;
		}
		50% {
			opacity: 0.5;
		}
	}

	/* ==========================================================================
	   WAVE ANIMATION
	   ========================================================================== */
	.skeleton-wave::after {
		content: '';
		position: absolute;
		inset: 0;
		transform: translateX(-100%);
		background: linear-gradient(
			90deg,
			transparent,
			rgba(255, 255, 255, 0.08),
			transparent
		);
		animation: skeleton-wave 1.5s ease-in-out infinite;
	}

	@keyframes skeleton-wave {
		100% {
			transform: translateX(100%);
		}
	}

	/* ==========================================================================
	   TEXT VARIANT
	   ========================================================================== */
	.skeleton-text {
		display: flex;
		flex-direction: column;
	}

	/* ==========================================================================
	   LIST VARIANT
	   ========================================================================== */
	.skeleton-list {
		display: flex;
		flex-direction: column;
	}

	.skeleton-list-item {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.skeleton-list-content {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	/* ==========================================================================
	   CARD VARIANT
	   ========================================================================== */
	.skeleton-card {
		background: var(--color-bg-secondary, rgba(255, 255, 255, 0.02));
		border-radius: 8px;
		overflow: hidden;
		border: 1px solid var(--color-border, rgba(255, 255, 255, 0.08));
	}

	.skeleton-card-content {
		padding: 12px;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	/* ==========================================================================
	   REDUCED MOTION
	   ========================================================================== */
	@media (prefers-reduced-motion: reduce) {
		.skeleton-pulse,
		.skeleton-wave::after {
			animation: none;
		}

		.skeleton {
			opacity: 0.7;
		}
	}
</style>
