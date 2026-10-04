<script lang="ts">
	import { safeUrl } from './safeUrl';
	import type { CardData } from './types';

	interface Props {
		card: CardData;
		/** Base URL a relative `link` (e.g. "/p/{id}") is resolved against - the card's own link
		 * is meant for the real web app's routing, not necessarily this page's. Omit to leave a
		 * relative link relative (resolves against the current origin). */
		webBaseUrl?: string;
	}

	let { card, webBaseUrl = '' }: Props = $props();

	const safeImageUrl = $derived(safeUrl(card.image_url));
	const resolvedLink = $derived.by(() => {
		const link = safeUrl(card.link);
		if (!link) return null;
		return link.startsWith('/') ? `${webBaseUrl}${link}` : link;
	});
</script>

{#snippet body()}
	<div class="card">
		{#if safeImageUrl}<img src={safeImageUrl} alt={card.title} />{/if}
		<div>
			<div class="card-title">{card.title}</div>
			{#if card.subtitle}<div class="card-subtitle">{card.subtitle}</div>{/if}
			{#if card.price_display}<div class="card-price">{card.price_display}</div>{/if}
		</div>
	</div>
{/snippet}

{#if resolvedLink}
	<a href={resolvedLink} target="_blank" rel="noopener noreferrer" class="card-link">
		{@render body()}
	</a>
{:else}
	{@render body()}
{/if}

<style>
	.card-link {
		text-decoration: none;
		color: inherit;
		display: block;
	}
	.card-link:hover .card {
		opacity: 0.85;
	}
	.card {
		display: flex;
		gap: 0.5rem;
		background: #1a1a1a;
		border-radius: 4px;
		padding: 0.5rem;
	}
	.card img {
		width: 48px;
		height: 48px;
		object-fit: cover;
		border-radius: 4px;
	}
	.card-title {
		font-size: 0.9rem;
	}
	.card-subtitle {
		font-size: 0.75rem;
		color: #999;
	}
	.card-price {
		font-size: 0.85rem;
		font-weight: 600;
	}
</style>
