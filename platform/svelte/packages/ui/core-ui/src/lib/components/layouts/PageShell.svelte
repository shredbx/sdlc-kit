<script lang="ts">
	/**
	 * PageShell — Two-slot page composition shell (hero + content).
	 *
	 * Every page is a <PageShell> owning two slots:
	 *   hero    — full-bleed, flush under the global nav (optional)
	 *   content — zero-chrome column; consumer fills it with <Section> wrappers
	 *
	 * Optional sidebar/toolbar are delegated to CollapsibleSidebarLayout.
	 * PageShell adds NO padding, NO background, NO max-width to the content area —
	 * spacing is the <Section> primitive's responsibility.
	 *
	 * @example
	 * <PageShell>
	 *   {#snippet hero()}<PackagesHero ... />{/snippet}
	 *   {#snippet content()}
	 *     <Section id="intro">…</Section>
	 *     <Section id="cta" stripe>…</Section>
	 *   {/snippet}
	 * </PageShell>
	 *
	 * @layer layouts (level 4 in atomic design)
	 */
	import type { Snippet } from 'svelte';
	import CollapsibleSidebarLayout from './CollapsibleSidebarLayout.svelte';

	let {
		hero,
		content,
		sidebar: sidebarProp,
		toolbar: toolbarProp,
		floatingToolbar = true,
	}: {
		/** Hero zone — full-bleed, flush under the global nav. Optional. */
		hero?: Snippet;
		/** Content zone — zero-chrome column. Fill with <Section> wrappers. Required. */
		content: Snippet;
		/** Sidebar snippet — delegated to CollapsibleSidebarLayout. Optional. */
		sidebar?: Snippet;
		/** Toolbar snippet — delegated to CollapsibleSidebarLayout. Optional. */
		toolbar?: Snippet;
		/** When true (default) and a toolbar is present, the toolbar floats as a
		 *  fixed overlay on desktop so the hero sits flush under the nav. */
		floatingToolbar?: boolean;
	} = $props();
</script>

<CollapsibleSidebarLayout
	shell={true}
	hasSidebar={!!sidebarProp}
	hasToolbar={!!toolbarProp}
	floatingToolbar={!!toolbarProp && floatingToolbar}
	stickyToolbar={!!toolbarProp && !floatingToolbar}
>
	{#snippet sidebar()}{#if sidebarProp}{@render sidebarProp()}{/if}{/snippet}
	{#snippet toolbar()}{#if toolbarProp}{@render toolbarProp()}{/if}{/snippet}

	{#if hero}{@render hero()}{/if}
	<div class="page-content">
		{@render content()}
	</div>
</CollapsibleSidebarLayout>

<style>
	/* Zero chrome — content owns all geometry via <Section>. */
	.page-content {
		display: flex;
		flex-direction: column;
		width: 100%;
		padding: 0;
		margin: 0;
		max-width: none;
		background: transparent;
	}
</style>
