<script lang="ts">
	/**
	 * ContentNavItem Component
	 *
	 * Leaf navigation item within a ContentNavGroup. Shows a small colored dot or
	 * icon on the left, label text in the middle, and an optional count badge.
	 *
	 * Matches the `.nav-item` pattern inside nav-groups in packages-detail-a.html —
	 * e.g. "Databases", "Object Storage" with optional pending-dot indicators.
	 *
	 * @example
	 * <ContentNavItem label="Databases" href="/packages/databases" count={3} active />
	 * <ContentNavItem label="Search" href="/packages/search" color="#f59e0b" />
	 */

	// =============================================================================
	// PROPS
	// =============================================================================

	interface Props {
		/** Navigation href */
		href: string;
		/** Display label */
		label: string;
		/** Whether this item is currently active/selected */
		active?: boolean;
		/** Optional emoji or unicode icon (overrides dot indicator) */
		icon?: string;
		/** Optional color for the dot indicator (CSS color value using var() recommended) */
		color?: string;
		/** Optional count badge shown on the right */
		count?: number;
	}

	let { href, label, active = false, icon, color, count }: Props = $props();

	/** Show a dot indicator if color is provided and no icon */
	let showDot = $derived(!icon && !!color);
</script>

<a
	{href}
	class="content-nav-item"
	class:active
	aria-current={active ? 'page' : undefined}
>
	{#if icon}
		<span class="content-nav-icon" aria-hidden="true">{icon}</span>
	{:else if showDot}
		<span
			class="content-nav-dot"
			style="background-color: {color};"
			aria-hidden="true"
		></span>
	{:else}
		<span class="content-nav-indent" aria-hidden="true"></span>
	{/if}

	<span class="content-nav-label">{label}</span>

	{#if count !== undefined}
		<span class="content-nav-count" aria-label="{count} items">{count}</span>
	{/if}
</a>

<style>
	.content-nav-item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.3125rem 0.625rem 0.3125rem 1.25rem;
		font-size: 0.75rem;
		color: var(--color-text-muted, #8b8b90);
		text-decoration: none;
		border-left: 2px solid transparent;
		cursor: pointer;
		transition:
			color 0.15s ease,
			background-color 0.15s ease,
			border-color 0.15s ease;
	}

	.content-nav-item:hover {
		color: var(--color-text, #eeeff1);
		background: rgba(255, 255, 255, 0.02);
	}

	.content-nav-item.active {
		color: var(--color-accent, #e94560);
		background: var(--color-accent-subtle, rgba(233, 69, 96, 0.12));
		border-left-color: var(--color-accent, #e94560);
	}

	.content-nav-icon {
		font-size: 0.875rem;
		width: 1rem;
		text-align: center;
		flex-shrink: 0;
		margin-right: 0.25rem;
		line-height: 1;
	}

	.content-nav-dot {
		width: 0.375rem;
		height: 0.375rem;
		border-radius: 50%;
		flex-shrink: 0;
		margin-right: 0.25rem;
	}

	.content-nav-indent {
		width: 0.625rem;
		flex-shrink: 0;
	}

	.content-nav-label {
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.content-nav-count {
		font-size: 0.6875rem;
		font-family: var(--font-mono, monospace);
		color: var(--color-text-dim, #55555a);
		background: rgba(255, 255, 255, 0.04);
		padding: 0.0625rem 0.375rem;
		border-radius: 4px;
		flex-shrink: 0;
		font-variant-numeric: tabular-nums;
	}

	.content-nav-item.active .content-nav-count {
		color: var(--color-accent, #e94560);
		background: var(--color-accent-subtle, rgba(233, 69, 96, 0.12));
	}

	/* Reduced motion */
	@media (prefers-reduced-motion: reduce) {
		.content-nav-item {
			transition-duration: 0.01ms !important;
		}
	}
</style>
