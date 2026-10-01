<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import Icon from '../primitives/Icon.svelte';

	export interface Tab {
		id: string;
		label: string;
		icon?: string;
		count?: number;
	}

	interface Props {
		tabs: Tab[];
		activeTab: string;
		onchange: (id: string) => void;
		/** When set, syncs activeTab to ?{urlParam}=X in the URL (replaceState, no history pollution) */
		urlParam?: string;
		/** Visual variant: 'pills' (default, rounded background) or 'underline' (bottom border) */
		variant?: 'pills' | 'underline';
	}

	let {
		tabs,
		activeTab,
		onchange,
		urlParam,
		variant = 'pills'
	}: Props = $props();

	// On mount: read initial tab from URL if urlParam is configured
	onMount(() => {
		if (!urlParam) return;
		const paramValue = page.url.searchParams.get(urlParam);
		if (paramValue && tabs.some((t) => t.id === paramValue) && paramValue !== activeTab) {
			onchange(paramValue);
		}
	});

	function handleClick(id: string) {
		onchange(id);
		if (urlParam) {
			const url = new URL(page.url);
			url.searchParams.set(urlParam, id);
			goto(url.toString(), { replaceState: true, noScroll: true });
		}
	}
</script>

<div class="tabs" class:underline={variant === 'underline'} role="tablist">
	{#each tabs as tab (tab.id)}
		<button
			class="tab"
			class:active={activeTab === tab.id}
			role="tab"
			aria-selected={activeTab === tab.id}
			onclick={() => handleClick(tab.id)}
		>
			{#if tab.icon}<Icon name={tab.icon} size="xs" />{/if}
			{tab.label}
			{#if tab.count !== undefined && tab.count > 0}
				<span class="tab-count">{tab.count}</span>
			{/if}
		</button>
	{/each}
</div>

<style>
	/* ===== Base (pills variant) ===== */
	.tabs {
		display: flex;
		gap: 0.25rem;
		padding: 0.25rem;
		background: var(--color-bg-tertiary);
		border-radius: 0.5rem;
		margin-bottom: 1.5rem;
		/* Many tabs scroll inside their own row — the page never widens and the
		   labels never squish (e.g. a language switcher offering many locales).
		   contain: inline-size zeroes the row's min-content contribution, so grid/
		   flex ancestors (min-width:auto) can never be inflated by many tabs. */
		overflow-x: auto;
		scrollbar-width: thin;
		contain: inline-size;
	}

	.tab {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.5rem 1rem;
		border: none;
		border-radius: 0.375rem;
		background: transparent;
		color: var(--color-text-muted);
		font-size: 0.875rem;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.15s ease;
		flex-shrink: 0;
		white-space: nowrap;
	}

	.tab:hover {
		color: var(--color-text);
		background: var(--color-bg-hover);
	}

	.tab.active {
		background: var(--color-bg-secondary);
		color: var(--color-text);
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
	}

	/* ===== Underline variant ===== */
	.tabs.underline {
		background: none;
		padding: 0;
		border-radius: 0;
		border-bottom: 1px solid var(--color-border);
		gap: var(--spacing-xs, 0.25rem);
	}

	.underline .tab {
		border-radius: 0;
		border-bottom: 2px solid transparent;
		padding: var(--spacing-sm, 0.5rem) var(--spacing-md, 1rem);
		font-size: 0.8rem;
	}

	.underline .tab:hover {
		background: none;
		color: var(--color-text);
	}

	.underline .tab.active {
		background: none;
		box-shadow: none;
		color: var(--color-accent);
		border-bottom-color: var(--color-accent);
	}

	/* ===== Count badge ===== */
	.tab-count {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-width: 1.25rem;
		height: 1.125rem;
		padding: 0 0.25rem;
		background: var(--color-bg-tertiary);
		border-radius: 9999px;
		font-size: 0.7rem;
		font-weight: 600;
		color: var(--color-text-muted);
	}

	.tab.active .tab-count {
		background: var(--color-accent);
		color: var(--color-bg);
	}

	.underline .tab-count {
		background: var(--color-bg-secondary);
	}

	.underline .tab.active .tab-count {
		background: var(--color-accent);
		color: var(--color-bg);
	}
</style>
