<script lang="ts">
	/**
	 * ContentNavGroup Component
	 *
	 * Collapsible navigation group header for a sidebar content tree.
	 * Clicking the header toggles the open/closed state and reveals child items.
	 *
	 * Matches the `.nav-section-header` pattern from packages-detail-a.html —
	 * e.g. "DATA & PERSISTENCE" is a NavGroup, and its child items are ContentNavItems.
	 *
	 * @example
	 * <ContentNavGroup label="Data & Persistence">
	 *   {#snippet children()}
	 *     <ContentNavItem label="Databases" href="/packages/databases" count={3} />
	 *   {/snippet}
	 * </ContentNavGroup>
	 */

	import type { Snippet } from 'svelte';

	// =============================================================================
	// PROPS
	// =============================================================================

	interface Props {
		/** The group display label (shown in uppercase) */
		label: string;
		/** Optional href — when set, clicking the label navigates AND expands */
		href?: string;
		/** Whether the group is expanded (default: true) */
		open?: boolean;
		/** Optional item count badge displayed next to the label */
		count?: number;
		/** Child nav items — use ContentNavItem components here */
		children?: Snippet;
	}

	let { label, href, open = true, count, children }: Props = $props();

	// =============================================================================
	// STATE
	// =============================================================================

	// svelte-ignore state_referenced_locally
	let expanded = $state(open ?? true);

	function toggle() {
		expanded = !expanded;
	}
</script>

<div class="nav-group" class:collapsed={!expanded}>
	{#if href}
		<a
			class="nav-group-header"
			{href}
			onclick={() => { expanded = true; }}
			aria-expanded={expanded}
		>
			<span class="nav-group-label">{label}</span>
			{#if count != null}
				<span class="nav-group-count">{count}</span>
			{/if}
			<button
				class="nav-group-toggle"
				type="button"
				onclick={(e) => { e.preventDefault(); e.stopPropagation(); toggle(); }}
				aria-label={expanded ? 'Collapse' : 'Expand'}
			>
				<svg
					class="nav-group-chevron"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					aria-hidden="true"
				>
					<polyline points="6 9 12 15 18 9" />
				</svg>
			</button>
		</a>
	{:else}
		<button
			class="nav-group-header"
			onclick={toggle}
			aria-expanded={expanded}
			type="button"
		>
			<span class="nav-group-label">{label}</span>
			{#if count != null}
				<span class="nav-group-count">{count}</span>
			{/if}
			<svg
				class="nav-group-chevron"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				aria-hidden="true"
			>
				<polyline points="6 9 12 15 18 9" />
			</svg>
		</button>
	{/if}
	{#if expanded && children}
		<div class="nav-group-items">
			{@render children()}
		</div>
	{/if}
</div>

<style>
	.nav-group {
		margin-bottom: 0.25rem;
	}

	.nav-group-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		width: 100%;
		padding: 0.5rem 0.625rem;
		background: transparent;
		border: none;
		cursor: pointer;
		font-family: inherit;
		font-size: 0.8125rem;
		font-weight: 400;
		letter-spacing: 0.01em;
		text-transform: none;
		color: var(--color-text-muted, #8b8b90);
		text-align: left;
		text-decoration: none;
		user-select: none;
		transition: color 0.15s ease;
	}

	.nav-group-toggle {
		display: flex;
		align-items: center;
		justify-content: center;
		background: transparent;
		border: none;
		cursor: pointer;
		padding: 0.125rem;
		margin: -0.125rem;
		color: inherit;
		flex-shrink: 0;
	}

	.nav-group-header:hover {
		color: var(--color-text, #eeeff1);
	}

	.nav-group-label {
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.nav-group-count {
		font-family: var(--font-mono, monospace);
		font-size: 0.625rem;
		color: var(--color-text-muted, #8b8b90);
		opacity: 0.6;
		flex-shrink: 0;
		margin-right: 0.375rem;
	}

	.nav-group-chevron {
		width: 0.75rem;
		height: 0.75rem;
		flex-shrink: 0;
		transition: transform 0.2s ease;
		color: inherit;
		opacity: 0.6;
	}

	.nav-group.collapsed .nav-group-chevron {
		transform: rotate(-90deg);
	}

	.nav-group-items {
		padding: 0 0 0.375rem;
	}

	/* Reduced motion */
	@media (prefers-reduced-motion: reduce) {
		.nav-group-header,
		.nav-group-chevron {
			transition-duration: 0.01ms !important;
		}
	}
</style>
