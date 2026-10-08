<script lang="ts">
	// Field names match platform/go/packages/cms/section_hero.go's HeroSection payload exactly
	// (eyebrow/headline/sub) — this is the wire shape, not this renderer's own invention.
	let { content }: { content: { eyebrow?: string; headline?: string; sub?: string } } = $props();
</script>

<section class="renderer-hero">
	<div class="renderer-hero__inner">
		{#if content.eyebrow}
			<p class="eyebrow">{content.eyebrow}</p>
		{/if}
		<!-- The page's own route-level <h1> was removed with the layout registry
		     (bos-layout-presets.md §3: "content declares its headline through a block") — a
		     hero is the one section kind that carries the page's visible top-level heading. -->
		<h1>{content.headline ?? ''}</h1>
		{#if content.sub}
			<p class="sub">{content.sub}</p>
		{/if}
	</div>
</section>

<style>
	.renderer-hero {
		padding: var(--spacing-10, 4rem) var(--spacing-lg, 1.5rem);
		text-align: center;
		background: var(--color-bg-secondary, #f7f7f9);
		border-bottom: 1px solid var(--color-border, #eee);
		font-family: var(--font-sans, sans-serif);
	}
	.renderer-hero__inner {
		max-width: 42rem;
		margin: 0 auto;
	}
	.renderer-hero h1 {
		font-size: 2rem;
		line-height: 1.15;
		font-weight: 700;
		letter-spacing: -0.02em;
		margin: 0 0 var(--spacing-sm, 0.75rem);
		color: var(--color-text, inherit);
	}
	.renderer-hero .eyebrow {
		text-transform: uppercase;
		letter-spacing: 0.08em;
		font-size: 0.8rem;
		font-weight: 600;
		color: var(--color-accent, #6366f1);
		margin: 0 0 var(--spacing-sm, 0.5rem);
	}
	.renderer-hero .sub {
		font-size: 1.0625rem;
		color: var(--color-text-muted, #666);
		margin: 0;
	}
</style>
