<script lang="ts">
	// Plain text, not rendered Markdown: the real Markdown pipeline (the old app's sanitizing
	// rules) is roadmap M2-M4 territory — this renderer proves the registry/section mechanism,
	// not the final Markdown rendering (docs/proposals/bos-demo-mvp.md carries the same note for
	// the MVP's own homepage render, which this replaces).
	// Field name matches platform/go/packages/cms/section_prose.go's ProseSection payload
	// exactly (markdown) — this is the wire shape, not this renderer's own invention.
	let { content }: { content: { markdown?: string } } = $props();
</script>

<section class="renderer-prose">
	<div class="renderer-prose__inner body">{content.markdown ?? ''}</div>
</section>

<style>
	.renderer-prose {
		font-family: var(--font-sans, sans-serif);
	}
	.renderer-prose__inner {
		max-width: 40rem;
		margin: 0 auto;
		padding: var(--spacing-lg, 1.5rem) 0;
		font-size: 1rem;
		line-height: 1.65;
		color: var(--color-text, inherit);
		white-space: pre-wrap;
	}
</style>
