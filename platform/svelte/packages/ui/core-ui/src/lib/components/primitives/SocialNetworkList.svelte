<script lang="ts" module>
	/** Entry shape matches pkg/socialnetwork.SocialNetwork JSON serialization
	 *  (platform / handle / url). The `platform` field stores the dictionary
	 *  code (e.g. "line", "whatsapp") that links to a SocialNetworkPlatform
	 *  for icon + color + url_template metadata. */
	export interface SocialNetworkEntry {
		platform: string;
		handle: string;
		url?: string;
	}

	/** Platform metadata from the social_networks dictionary. `code` matches
	 *  SocialNetworkEntry.platform. */
	export interface SocialNetworkPlatform {
		code: string;
		label: string;
		icon: string;
		color?: string;
		url_template?: string;
	}

	/** Resolve an entry's href: the stored value (the `url` field if set, else the
	 *  `handle` — the single editor input now holds the FULL profile URL) is used
	 *  verbatim, scheme-normalized (a scheme-less value gets an https:// prefix).
	 *  No handle→URL templating: that double-prefixed pasted URLs and broke on
	 *  custom profile paths. `_platform` is retained for call-site compatibility. */
	export function resolveSocialUrl(entry: SocialNetworkEntry, _platform?: SocialNetworkPlatform): string | undefined {
		const value = entry.url?.trim() || entry.handle?.trim() || '';
		if (!value) return undefined;
		return /^https?:\/\//i.test(value) ? value : `https://${value.replace(/^\/+/, '')}`;
	}
</script>

<script lang="ts">
	import SocialIcon from './SocialIcon.svelte';

	let {
		entries,
		platforms = [],
		size = 18,
		showHandle = true
	}: {
		entries: SocialNetworkEntry[];
		platforms?: SocialNetworkPlatform[];
		size?: number;
		showHandle?: boolean;
	} = $props();

	const platformIndex = $derived(
		new Map(platforms.map((p) => [p.code, p]))
	);
</script>

<ul class="social-network-list">
	{#each entries as entry (entry.platform + ':' + entry.handle)}
		{@const platform = platformIndex.get(entry.platform)}
		{@const href = resolveSocialUrl(entry, platform)}
		{@const color = platform?.color}
		<li class="social-network-item">
			{#if href}
				<a class="social-network-link" href={href} target="_blank" rel="noopener noreferrer" style:--brand-color={color ?? 'currentColor'}>
					<SocialIcon platform={platform?.icon ?? entry.platform} {size} />
					{#if showHandle}<span class="social-network-handle">{entry.handle}</span>{/if}
				</a>
			{:else}
				<span class="social-network-link social-network-link--inert" style:--brand-color={color ?? 'currentColor'}>
					<SocialIcon platform={platform?.icon ?? entry.platform} {size} />
					{#if showHandle}<span class="social-network-handle">{entry.handle}</span>{/if}
				</span>
			{/if}
		</li>
	{/each}
</ul>

<style>
	.social-network-list {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.social-network-item { display: inline-flex; }
	.social-network-link {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		padding: 0.25rem 0.5rem;
		border: 1px solid currentColor;
		border-radius: 999px;
		color: var(--brand-color);
		text-decoration: none;
		font-size: 0.85rem;
		line-height: 1;
	}
	.social-network-link:hover { opacity: 0.8; }
	.social-network-link--inert { opacity: 0.85; }
	.social-network-handle { color: inherit; }
</style>
