<script lang="ts">
	// Shared contextual-panel chrome (D17) — the fixed-width left working panel that
	// every rail context (Document · Media · Texts · Map · Components · Settings)
	// renders inside. Owns the panel container, the panel header (title), the search
	// box, and the scrollable body. The 6 panels compose CollapsibleSections into the
	// default `children` snippet — they NEVER re-declare this chrome CSS (the
	// never-copy-paste-CSS rule: one source for the panel frame).
	import PanelSearch from './PanelSearch.svelte';

	interface Props {
		/** Panel heading (the rail context name): 'Document', 'Media', … */
		title: string;
		/** Search input placeholder. */
		searchPlaceholder?: string;
		/** Show the panel-level search box (default true). Panels whose content is
		 *  itself sectioned with per-section search (Document → Templates) or has
		 *  nothing flat to search (Sources) opt out with `search={false}`. */
		search?: boolean;
		/** Optional header action (e.g. a ＋ button to the right of the title). */
		headerAction?: import('svelte').Snippet;
		/** The panel body — a stack of CollapsibleSections. */
		children?: import('svelte').Snippet;
	}

	let { title, searchPlaceholder = 'Search…', search = true, headerAction, children }: Props = $props();

	// Search is local per panel this cut — the filter wiring lands with the real
	// content iterations (this is a SHELL/IA rebuild). Kept here so every panel gets
	// the same search affordance with zero duplicated markup.
	let query = $state('');
</script>

<aside class="panel">
	<header class="panel__header">
		<h2 class="panel__title">{title}</h2>
		{#if headerAction}
			<span class="panel__header-action">{@render headerAction()}</span>
		{/if}
	</header>

	{#if search}
		<div class="panel__search">
			<PanelSearch bind:value={query} placeholder={searchPlaceholder} label={`Search ${title.toLowerCase()}`} />
		</div>
	{/if}

	<div class="panel__body">
		{@render children?.()}
	</div>
</aside>

<style>
	.panel {
		display: flex;
		flex-direction: column;
		width: 280px;
		min-width: 280px;
		background: var(--cv-color-surface, #fff);
		border-right: 1px solid var(--cv-border-color, #e0e0e0);
		overflow: hidden;
	}

	.panel__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--cv-space-sm, 0.5rem);
		padding: var(--cv-space-md, 1rem) var(--cv-space-md, 1rem) 0;
	}

	.panel__title {
		margin: 0;
		font-size: 0.6875rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--cv-color-neutral-500, #707070);
	}

	/* CTAs / header affordances right-aligned (hard rule). */
	.panel__header-action {
		margin-left: auto;
		display: inline-flex;
		align-items: center;
	}

	.panel__search {
		display: flex;
		padding: var(--cv-space-sm, 0.5rem) var(--cv-space-md, 1rem) 0;
	}

	.panel__body {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: var(--cv-space-xs, 0.25rem);
		padding: var(--cv-space-md, 1rem);
		overflow-y: auto;
	}
</style>
