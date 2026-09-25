<script lang="ts">
	/**
	 * PageBreadcrumb — canonical page-chrome breadcrumb wrapper.
	 *
	 * Owns the single source of truth for breadcrumb placement + padding
	 * across all shell-mode pages (packages L3, knowledge L2/L3, and any
	 * future detail route). Aligns the crumb's left edge with the hero's
	 * outer padding so the trail reads flush with the title column.
	 *
	 * Usage:
	 *   // In a layout toolbar snippet, or inline above a hero:
	 *   <PageBreadcrumb items={breadcrumb} />
	 *
	 * DO NOT re-implement the padded wrapper per-page. If the canonical
	 * chrome needs to change, change it here once.
	 */
	import Breadcrumb, { type BreadcrumbItem } from './Breadcrumb.svelte';

	let {
		items = [],
		separator = '/',
	}: {
		items: BreadcrumbItem[];
		separator?: string;
	} = $props();
</script>

{#if items.length > 0}
	<div class="page-breadcrumb">
		<Breadcrumb {items} {separator} />
	</div>
{/if}

<style>
	/* Padding mirrors SectionDetailHero's outer horizontal padding
	   (clamp(40px, 5vw, 72px) 48px ...) so the breadcrumb's left edge
	   aligns with the hero's title column pixel-for-pixel. The vertical
	   top padding (0.75rem) creates breathing room under the global nav;
	   bottom padding is 0 because the hero below supplies its own top pad. */
	.page-breadcrumb {
		padding: 0 48px;
		font-size: 0.8125rem;
		width: 100%;
		min-width: 0;
		flex: 1;
	}

	@media (max-width: 768px) {
		.page-breadcrumb {
			padding: 0;
		}
	}
</style>
