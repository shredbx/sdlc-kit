<script lang="ts">
	import type { FooterConfig } from '../types';
	import { safeHref } from '../../safe-href';

	let { config }: { config: FooterConfig } = $props();
</script>

<footer class="chrome-footer">
	<div class="chrome-footer__inner">
		{#each config.sections as section (section.heading)}
			<div class="chrome-footer__section">
				<h3>{section.heading}</h3>
				<ul>
					{#each section.links.filter((l) => l.enabled && l.web) as link, i (link.title + i)}
						{#if link.custom_path}
							<li><a href={safeHref(link.custom_path)}>{link.title}</a></li>
						{/if}
					{/each}
				</ul>
			</div>
		{/each}
	</div>
</footer>

<style>
	.chrome-footer {
		border-top: 1px solid var(--color-border, #e2e4e8);
		background: var(--color-bg-secondary, #fafafa);
	}
	.chrome-footer__inner {
		max-width: 72rem;
		margin: 0 auto;
		display: flex;
		flex-wrap: wrap;
		gap: var(--spacing-xl, 3rem);
		padding: var(--spacing-xl, 3rem) var(--spacing-lg, 1.5rem);
		font-family: var(--font-sans, sans-serif);
	}
	.chrome-footer__section h3 {
		margin: 0 0 var(--spacing-sm, 0.75rem);
		font-size: 0.8rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.06em;
		color: var(--color-text-muted, #888);
	}
	.chrome-footer__section ul {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: var(--spacing-xs, 0.4rem);
	}
	.chrome-footer__section a {
		color: var(--color-text, inherit);
		text-decoration: none;
		font-size: 0.875rem;
	}
	.chrome-footer__section a:hover {
		color: var(--color-accent, inherit);
		text-decoration: underline;
	}
</style>
