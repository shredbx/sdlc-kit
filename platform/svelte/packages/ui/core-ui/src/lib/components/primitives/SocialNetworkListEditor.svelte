<script lang="ts">
	import { untrack } from 'svelte';
	import SocialIcon from './SocialIcon.svelte';
	import type { SocialNetworkEntry, SocialNetworkPlatform } from './SocialNetworkList.svelte';

	let {
		entries = $bindable([]),
		platforms = [],
		addStyle = 'button'
	}: {
		entries?: SocialNetworkEntry[];
		platforms?: SocialNetworkPlatform[];
		/** The add affordance: a labelled "Add" button (default) or a compact (+)
		 *  icon button. Opt-in per consumer — existing consumers keep the button. */
		addStyle?: 'button' | 'icon';
	} = $props();

	let pendingPlatform = $state(untrack(() => platforms[0]?.code ?? ''));
	let pendingHandle = $state('');

	$effect(() => {
		if (pendingPlatform === '' && platforms.length > 0) {
			pendingPlatform = platforms[0].code;
		}
	});

	function addEntry() {
		const handle = pendingHandle.trim();
		if (!pendingPlatform || handle === '') return;
		entries = [...entries, { platform: pendingPlatform, handle }];
		pendingHandle = '';
	}

	function removeEntry(index: number) {
		entries = entries.filter((_, i) => i !== index);
	}

	function updateHandle(index: number, value: string) {
		entries = entries.map((entry, i) => (i === index ? { ...entry, handle: value } : entry));
	}

	function updatePlatform(index: number, value: string) {
		entries = entries.map((entry, i) => (i === index ? { ...entry, platform: value } : entry));
	}

	function onKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			event.preventDefault();
			addEntry();
		}
	}

	const platformIndex = $derived(new Map(platforms.map((p) => [p.code, p])));
	// The selected add-row platform's metadata — drives the leading icon preview so
	// the add row matches the committed rows' layout.
	const pendingMeta = $derived(platformIndex.get(pendingPlatform));
</script>

<div class="social-network-editor">
	{#if entries.length > 0}
		<ul class="social-network-rows">
			{#each entries as entry, i (i)}
				{@const platform = platformIndex.get(entry.platform)}
				<li class="social-network-row">
					<span class="social-network-row__icon" style:--brand-color={platform?.color ?? 'currentColor'}>
						<SocialIcon platform={platform?.icon ?? entry.platform} size={16} />
					</span>
					<select
						class="social-network-row__platform"
						value={entry.platform}
						onchange={(e) => updatePlatform(i, (e.target as HTMLSelectElement).value)}
					>
						{#each platforms as p (p.code)}
							<option value={p.code}>{p.label}</option>
						{/each}
					</select>
					<input
						class="social-network-row__handle"
						type="text"
						value={entry.handle}
						placeholder="https://… (full profile URL)"
						oninput={(e) => updateHandle(i, (e.target as HTMLInputElement).value)}
					/>
					<button type="button" class="social-network-row__remove" onclick={() => removeEntry(i)} aria-label="Remove">×</button>
				</li>
			{/each}
		</ul>
	{/if}

	<div class="social-network-add">
		<span class="social-network-row__icon" style:--brand-color={pendingMeta?.color ?? 'currentColor'}>
			<SocialIcon platform={pendingMeta?.icon ?? pendingPlatform} size={16} />
		</span>
		<select
			class="social-network-add__platform"
			bind:value={pendingPlatform}
			aria-label="Platform"
		>
			{#each platforms as p (p.code)}
				<option value={p.code}>{p.label}</option>
			{/each}
		</select>
		<input
			class="social-network-add__handle"
			type="text"
			bind:value={pendingHandle}
			placeholder="https://… (full profile URL)"
			onkeydown={onKeydown}
		/>
		{#if addStyle === 'icon'}
			<button
				type="button"
				class="social-network-add__icon"
				onclick={addEntry}
				aria-label="Add social network"
				title="Add"
			>
				<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
					<line x1="12" y1="5" x2="12" y2="19" />
					<line x1="5" y1="12" x2="19" y2="12" />
				</svg>
			</button>
		{:else}
			<button type="button" class="social-network-add__btn" onclick={addEntry}>
				Add
			</button>
		{/if}
	</div>
</div>

<style>
	.social-network-editor {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}
	.social-network-rows {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
	}
	.social-network-row {
		display: grid;
		grid-template-columns: auto auto 1fr auto;
		gap: 0.5rem;
		align-items: center;
	}
	.social-network-row__icon {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		color: var(--brand-color);
	}
	.social-network-row__platform,
	.social-network-add__platform,
	.social-network-row__handle,
	.social-network-add__handle {
		font: inherit;
		padding: 0.35rem 0.5rem;
		border: 1px solid var(--input-border, #ccc);
		border-radius: 0.375rem;
		background: var(--input-bg, transparent);
		color: inherit;
	}
	/* Selects: drop the OS arrow (it renders flush to the right edge) and draw a
	   custom chevron with breathing room from the border. */
	.social-network-row__platform,
	.social-network-add__platform {
		appearance: none;
		-webkit-appearance: none;
		padding-right: 1.85rem;
		background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%236b7280' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpolyline points='6 9 12 15 18 9'/%3E%3C/svg%3E");
		background-repeat: no-repeat;
		background-position: right 0.6rem center;
		background-size: 0.7rem;
	}
	.social-network-row__remove {
		background: transparent;
		border: 1px solid var(--input-border, #ccc);
		border-radius: 999px;
		width: 28px;
		height: 28px;
		cursor: pointer;
		font-size: 1rem;
		line-height: 1;
	}
	.social-network-row__remove:hover { background: rgba(0, 0, 0, 0.05); }
	.social-network-add {
		display: grid;
		grid-template-columns: auto auto 1fr auto;
		gap: 0.5rem;
		align-items: center;
	}
	.social-network-add__btn {
		font: inherit;
		font-weight: 600;
		padding: 0.35rem 0.85rem;
		border: 1px solid currentColor;
		border-radius: 0.375rem;
		background: transparent;
		color: inherit;
		cursor: pointer;
	}
	.social-network-add__btn:hover { opacity: 0.8; }

	/* Compact (+) icon variant (addStyle="icon") — a circular outline button that
	   replaces the labelled "Add". Occupies the same trailing grid cell; opt-in. */
	.social-network-add__icon {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 34px;
		height: 34px;
		padding: 0;
		border: 1px solid currentColor;
		border-radius: 999px;
		background: transparent;
		color: inherit;
		cursor: pointer;
		transition: opacity 0.15s ease;
	}
	.social-network-add__icon:hover { opacity: 0.7; }
</style>
