<script lang="ts">
	// Locked page-title primitive (#0290 F2 — promoted from BR-local AdminTitle, 2606-012).
	// The page title typography lives HERE and nowhere else, so the heading scale can't
	// drift between the surfaces that compose it — the same role AdminPageShell's
	// `.admin-page-shell__title` once played, extracted into a reusable <h1>. The DEFAULT
	// scale tokens are the generic core-ui token contract (--font-heading, --text-3xl,
	// weight 700, --color-text-strong) and are LOCKED; a consumer themes them by mapping
	// those tokens (BR maps them back to its --br-* values, so the look is unchanged).
	//
	// `scale` is additive: 'default' preserves the locked tokens; 'landing' reproduces the
	// larger landing title (--text-4xl, weight 600, --color-accent, tight leading) so a
	// scaffold/landing surface keeps its distinct look without a second component.
	// Typography only — vertical spacing stays owned by the shell that renders this title.
	//
	// Accepts EITHER a `title` string (the common case) or a `children` snippet (when the
	// heading needs inline markup). The snippet wins when both are passed; passing neither
	// renders an empty heading — callers always supply one.
	import type { Snippet } from 'svelte';

	let {
		title,
		children,
		scale = 'default'
	}: { title?: string; children?: Snippet; scale?: 'default' | 'landing' } = $props();
</script>

<h1 class="page-title" class:page-title--landing={scale === 'landing'}>
	{#if children}
		{@render children()}
	{:else}
		{title}
	{/if}
</h1>

<style>
	/* The single source of the page-title scale — promoted verbatim from BR's locked
	   AdminTitle (which carried it from AdminPageShell's .admin-page-shell__title), with
	   every --br-* token swapped for the generic core-ui token contract. */
	.page-title {
		font-family: var(--font-heading);
		font-size: var(--text-3xl);
		font-weight: 700;
		color: var(--color-text-strong);
		margin: 0;
	}

	/* Landing scale — the larger, accent-coloured landing title (verbatim type tokens,
	   generalized: 4xl, weight 600, accent, tight leading). Spacing is the shell's job,
	   so this overrides typography only. */
	.page-title--landing {
		font-size: var(--text-4xl);
		font-weight: 600;
		color: var(--color-accent);
		line-height: var(--leading-tight);
	}
</style>
