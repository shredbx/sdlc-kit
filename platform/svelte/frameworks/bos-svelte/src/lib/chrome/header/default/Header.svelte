<script lang="ts">
	import type { HeaderConfig } from '../types';
	import { safeHref } from '../../safe-href';

	let { config, siteTitle }: { config: HeaderConfig; siteTitle?: string } = $props();

	// page_id resolution isn't built yet (see chrome/header/types.ts) — a nav item with only
	// page_id set renders nothing, rather than a broken link.
	const items = $derived(config.nav.filter((n) => n.enabled && n.web));
</script>

<header class="chrome-header">
	<div class="chrome-header__inner">
		<a class="chrome-header__brand" href="/">{siteTitle ?? 'bos'}</a>
		<nav class="chrome-header__nav">
			{#each items as item, i (item.title + i)}
				{#if item.type === 'separator'}
					<span class="sep" aria-hidden="true"></span>
				{:else if item.custom_path}
					<a href={safeHref(item.custom_path)} target={item.new_tab ? '_blank' : undefined} rel={item.new_tab ? 'noopener' : undefined}>
						{item.title}
					</a>
				{/if}
			{/each}
		</nav>
	</div>
</header>

<style>
	.chrome-header {
		position: sticky;
		top: 0;
		z-index: 40;
		background: var(--color-bg, #fff);
		border-bottom: 1px solid var(--color-border, #e2e4e8);
	}
	.chrome-header__inner {
		max-width: 72rem;
		margin: 0 auto;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--spacing-md, 1rem);
		padding: var(--spacing-sm, 0.75rem) var(--spacing-lg, 1.5rem);
		font-family: var(--font-sans, sans-serif);
	}
	.chrome-header__brand {
		font-size: 1.125rem;
		font-weight: 700;
		color: var(--color-text, inherit);
		text-decoration: none;
		letter-spacing: -0.01em;
	}
	.chrome-header__nav {
		display: flex;
		align-items: center;
		gap: var(--spacing-lg, 1.5rem);
		flex-wrap: wrap;
	}
	.chrome-header__nav a {
		font-size: 0.875rem;
		font-weight: 500;
		text-decoration: none;
		color: var(--color-text-muted, #666);
		transition: color 150ms ease;
	}
	.chrome-header__nav a:hover {
		color: var(--color-accent, var(--color-text, #111));
	}
	.sep {
		width: 1px;
		height: 1.25rem;
		background: var(--color-border, #ccc);
	}
</style>
