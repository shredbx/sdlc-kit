<script lang="ts">
	/**
	 * StripedStack — Wraps a sequence of page sections and applies
	 * alternating stripe backgrounds via CSS :nth-child, no per-child
	 * state needed.
	 *
	 * Sections inside a StripedStack should NOT pass `stripe` to their
	 * `<Section>` — the stack owns the alternation. The first child is
	 * plain; every even-positioned child gets `--section-stripe-bg`.
	 *
	 * @example
	 * <KnowledgeHero ... />
	 * <StripedStack>
	 *   <Section id="featured">...</Section>
	 *   {#each l1Sections as l}
	 *     <Section id={l.id}>...</Section>
	 *   {/each}
	 * </StripedStack>
	 */
	import type { Snippet } from 'svelte';

	let { children }: { children: Snippet } = $props();
</script>

<div class="striped-stack">
	{@render children()}
</div>

<style>
	.striped-stack {
		display: contents;
	}

	.striped-stack > :global(:nth-child(even)) {
		background: var(--section-stripe-bg);
	}

	.striped-stack > :global(:first-child) {
		padding-block-start: 0;
	}
</style>
