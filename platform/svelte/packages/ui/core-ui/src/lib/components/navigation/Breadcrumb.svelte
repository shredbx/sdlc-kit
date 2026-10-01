<script lang="ts">
	/**
	 * Breadcrumb Navigation Component
	 * Shows hierarchical path for deep entity navigation.
	 * Matches the hand-rolled breadcrumb pattern used across sidebar routes.
	 *
	 * @example
	 * <Breadcrumb items={[
	 *   { label: 'Knowledge', href: '/knowledge' },
	 *   { label: 'Patterns', href: '/knowledge/patterns' },
	 *   { label: 'Observer' }
	 * ]} />
	 */

	export interface BreadcrumbItem {
		label: string;
		href?: string;
	}

	let {
		items = [],
		separator = '/',
		showHome = false
	}: {
		items: BreadcrumbItem[];
		separator?: string;
		showHome?: boolean;
	} = $props();

	const displayItems = $derived.by(() => {
		if (showHome && items.length > 0 && items[0].href !== '/') {
			return [{ label: 'Home', href: '/' }, ...items];
		}
		return items;
	});
</script>

{#if displayItems.length > 1}
	<nav class="breadcrumb" aria-label="Breadcrumb">
		{#each displayItems as item, i}
			{#if i > 0}<span class="breadcrumb-sep" aria-hidden="true">{separator}</span>{/if}
			{#if i === displayItems.length - 1}
				<span class="breadcrumb-current" aria-current="page">{item.label}</span>
			{:else if item.href}
				<a href={item.href} class="breadcrumb-link">{item.label}</a>
			{:else}
				<span class="breadcrumb-current">{item.label}</span>
			{/if}
		{/each}
	</nav>
{/if}

<style>
	.breadcrumb {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		font-size: 0.8125rem;
		overflow-x: auto;
		white-space: nowrap;
		-webkit-overflow-scrolling: touch;
		scrollbar-width: none;
	}

	.breadcrumb::-webkit-scrollbar {
		display: none;
	}

	.breadcrumb-link {
		color: var(--color-text-muted);
		text-decoration: none;
		transition: color 0.15s ease;
	}

	.breadcrumb-link:hover {
		color: var(--color-accent);
	}

	.breadcrumb-sep {
		color: var(--color-text-muted);
		opacity: 0.5;
	}

	.breadcrumb-current {
		color: var(--color-text);
		font-weight: 500;
	}
</style>
