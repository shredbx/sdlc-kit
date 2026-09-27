<script lang="ts">
	// Failure-aware thumbnail <img> (I-2, Decision #0286 sprint). A failed load
	// swaps to an EXPLICIT broken state (alert glyph + tooltip + hatch) instead of
	// the platform's broken-image glyph — silent/ambiguous blanks are banned. One
	// shared component so every thumb surface (Media grid, source cells) renders
	// failure identically (never-copypaste). The wrapper element owns the box; this
	// component always fills it (100% × 100%).
	import { untrack } from 'svelte';
	import { Icon } from '@sbx/core-ui/components/primitives';

	let {
		src,
		alt = '',
		/** Identity folded into the failed tile's name ("Cover 1 — image failed to
		 *  load") — without it the fixed label would shadow the host button's
		 *  accessible name and make two failed thumbs indistinguishable. */
		label
	}: { src: string; alt?: string; label?: string } = $props();

	let imgEl = $state<HTMLImageElement | null>(null);
	let failed = $state(false);

	// A new src resets the state — grids/cells are keyed and reused on source
	// switch. Also catches an error that completed BEFORE hydration attached
	// onerror (panel chrome is SSR'd): a complete-but-zero-size img is a failure.
	// imgEl is read UNTRACKED: tracking it would re-run this effect when the
	// failed swap unmounts the img (bind:this → null) and overwrite `failed`.
	$effect(() => {
		void src;
		const el = untrack(() => imgEl);
		failed = el != null && el.complete && el.naturalWidth === 0;
	});

	const failedText = $derived(label ? `${label} — image failed to load` : 'Image failed to load');
</script>

{#if failed}
	<span class="failed" role="img" aria-label={failedText} title={failedText}>
		<Icon name="alert-circle" size="sm" />
	</span>
{:else}
	<img bind:this={imgEl} {src} {alt} loading="lazy" onerror={() => (failed = true)} />
{/if}

<style>
	img {
		display: block;
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.failed {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 100%;
		height: 100%;
		color: var(--cv-color-neutral-500, #707070);
		background: repeating-linear-gradient(
			45deg,
			transparent,
			transparent 6px,
			rgba(0, 0, 0, 0.05) 6px,
			rgba(0, 0, 0, 0.05) 12px
		);
	}
</style>
