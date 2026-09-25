<script lang="ts">
	/**
	 * Panel — a bordered header+body card (dumb). The Run pane's repeating `.card` shell: a title
	 * (with optional right-aligned sub / status snippet) over a padded body. Imports no store/API.
	 */
	import type { Snippet } from 'svelte';

	interface Props {
		title?: string;
		/** Muted caption shown at the right of the header. */
		sub?: string;
		/** Optional right-aligned header slot (e.g. a status pill) — overrides `sub`. */
		header?: Snippet;
		children: Snippet;
		'data-testid'?: string;
	}

	let { title, sub, header, children, 'data-testid': testid }: Props = $props();
</script>

<div class="card" data-testid={testid}>
	{#if title || sub || header}
		<div class="ch">
			{#if title}<h4>{title}</h4>{/if}
			{#if header}{@render header()}{:else if sub}<span class="sub">{sub}</span>{/if}
		</div>
	{/if}
	<div class="cb">{@render children()}</div>
</div>

<style>
	.card {
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 12px;
		box-shadow: 0 1px 2px rgba(16, 24, 40, 0.04), 0 1px 3px rgba(16, 24, 40, 0.06);
	}
	.ch {
		display: flex;
		align-items: center;
		gap: 9px;
		padding: 13px 16px;
		border-bottom: 1px solid var(--color-border);
	}
	.ch h4 {
		margin: 0;
		font-size: 13px;
		color: var(--color-text);
	}
	.ch .sub {
		font-size: 12px;
		color: var(--color-faint);
		margin-left: auto;
	}
	.cb {
		padding: 15px 16px;
	}
</style>
