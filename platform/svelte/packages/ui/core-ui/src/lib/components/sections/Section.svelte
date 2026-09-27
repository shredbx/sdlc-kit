<script lang="ts">
	/**
	 * Section — Primitive page-section wrapper.
	 *
	 * Owns: background color, outer padding-block, inner max-width + padding-inline,
	 * anchor id + scroll-margin-top, optional stripe background.
	 *
	 * Does NOT own: the content pattern inside. Compose with <SectionHeader>,
	 * raw markup, or any presentational component.
	 *
	 * @example
	 * <Section id="intro">
	 *   <SectionHeader title="Intro" />
	 *   <p>Body copy…</p>
	 * </Section>
	 *
	 * @example bare={true} — drops padding + max-width for components that own their own layout.
	 * <Section id="fdd-layers" bare>
	 *   <LayerMap {layers} />
	 * </Section>
	 */
	import type { Snippet } from 'svelte';

	let {
		id,
		label,
		bare = false,
		compact = false,
		stripe = false,
		noDotNav = false,
		class: className = '',
		children,
	}: {
		id?: string;
		/** Human-readable label for nav tracking (DotNav, ContentSidebar scroll-spy).
		 *  Falls back to id when omitted. */
		label?: string;
		bare?: boolean;
		compact?: boolean;
		stripe?: boolean;
		/** Suppress data-dot-nav-* attributes — keeps id for anchoring but hides from DotNav. */
		noDotNav?: boolean;
		class?: string;
		children: Snippet;
	} = $props();
</script>

<section
	{id}
	data-dot-nav-section={noDotNav ? undefined : id}
	data-dot-nav-label={noDotNav || !id ? undefined : (label ?? id)}
	class="section {bare ? 'section--bare' : ''} {compact ? 'section--compact' : ''} {stripe ? 'section--stripe' : ''} {className}"
>
	{#if bare}
		{@render children()}
	{:else}
		<div class="section__inner">
			{@render children()}
		</div>
	{/if}
</section>

<style>
	.section {
		width: 100%;
		background: transparent;
		padding-block: var(--section-pad-y, 64px);
		scroll-margin-top: var(--layout-anchor-offset, 5rem);
	}

	.section--bare {
		padding: 0;
	}

	.section--compact {
		padding-block: var(--section-pad-y-compact, 24px);
	}

	.section--stripe {
		background: var(--section-stripe-bg);
	}

	.section__inner {
		max-width: var(--section-max-width, 1160px);
		margin-inline: auto;
		padding-inline: var(--section-pad-x, 48px);
	}

	@media (max-width: 768px) {
		.section__inner {
			padding-inline: var(--section-pad-x-mobile, 20px);
		}
	}
</style>
