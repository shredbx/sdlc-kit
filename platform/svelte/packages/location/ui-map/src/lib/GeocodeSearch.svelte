<script lang="ts">
	import type { GeocodeResult, GeocodeService } from './types.js';

	interface Props {
		geocode: GeocodeService;
		onselect: (result: GeocodeResult) => void;
		placeholder?: string;
		debounceMs?: number;
		minChars?: number;
		country?: string;
		limit?: number;
		// Externally-pushed text — typically the reverse-geocode result of
		// the map center. Synced into the input only when the user is NOT
		// actively typing (input not focused, OR focused but its value
		// matches the prior external push). User-typed text wins.
		externalValue?: string;
	}

	let {
		geocode,
		onselect,
		placeholder = 'Search address or place',
		debounceMs = 300,
		minChars = 3,
		country,
		limit = 5,
		externalValue
	}: Props = $props();

	type Status = 'idle' | 'loading' | 'ready' | 'empty' | 'error';

	let value = $state('');
	let results = $state<GeocodeResult[]>([]);
	let status = $state<Status>('idle');
	let open = $state(false);
	let timer: ReturnType<typeof setTimeout> | null = null;
	let inflight = 0;
	let focused = $state(false);
	// Last value we accepted from the externalValue prop. Tracking this
	// lets us tell "user has typed since the last sync" apart from "user
	// hasn't touched the input since we last wrote into it".
	let lastSyncedExternal = $state<string | undefined>(undefined);

	// Push external value into the input when it changes AND we're not
	// stomping the user's typed text.
	$effect(() => {
		const next = externalValue;
		if (next === undefined) return;
		if (next === lastSyncedExternal) return;
		// User is actively typing if focused AND has typed something
		// different from what we last pushed in. Don't overwrite.
		if (focused && value !== lastSyncedExternal && value !== '') return;
		value = next;
		lastSyncedExternal = next;
		// External sync should not open the dropdown — there's nothing to
		// show until the user types.
		open = false;
		status = 'idle';
		results = [];
	});

	function clearTimer() {
		if (timer) {
			clearTimeout(timer);
			timer = null;
		}
	}

	function scheduleSearch(q: string) {
		clearTimer();
		if (q.length < minChars) {
			results = [];
			status = 'idle';
			open = false;
			return;
		}
		timer = setTimeout(() => runSearch(q), debounceMs);
	}

	async function runSearch(q: string) {
		const seq = ++inflight;
		status = 'loading';
		open = true;
		try {
			const hits = await geocode.search({ q, country, limit });
			if (seq !== inflight) return; // stale
			results = hits;
			status = hits.length === 0 ? 'empty' : 'ready';
		} catch {
			if (seq !== inflight) return;
			results = [];
			status = 'error';
		}
	}

	function handleInput(e: Event) {
		const next = (e.target as HTMLInputElement).value;
		value = next;
		scheduleSearch(next.trim());
	}

	function handleSelect(r: GeocodeResult) {
		value = r.display_name;
		results = [];
		status = 'idle';
		open = false;
		onselect(r);
	}

	function handleBlur() {
		focused = false;
		setTimeout(() => {
			open = false;
		}, 150);
	}

	function handleFocus() {
		focused = true;
		if (results.length || status === 'empty' || status === 'error') open = true;
	}
</script>

<div class="ui-map-geocode-search">
	<input
		type="text"
		role="combobox"
		aria-controls="ui-map-geocode-listbox"
		name="ui-map-geocode-query"
		autocomplete="off"
		value={value}
		{placeholder}
		oninput={handleInput}
		onfocus={handleFocus}
		onblur={handleBlur}
		aria-autocomplete="list"
		aria-expanded={open}
	/>
	{#if open}
		<ul id="ui-map-geocode-listbox" class="ui-map-geocode-results" role="listbox">
			{#if status === 'loading'}
				<li class="ui-map-geocode-results__row ui-map-geocode-results__row--loading">
					Searching…
				</li>
			{:else if status === 'error'}
				<li class="ui-map-geocode-results__row ui-map-geocode-results__row--error" role="alert">
					Search failed — try again
				</li>
			{:else if status === 'empty'}
				<li class="ui-map-geocode-results__row ui-map-geocode-results__row--empty">
					No matches for "{value}"
				</li>
			{:else if status === 'ready'}
				{#each results as r (r.display_name + r.lat + r.lng)}
					<li
						class="ui-map-geocode-results__row"
						role="option"
						aria-selected={false}
						tabindex="0"
					>
						<button type="button" onclick={() => handleSelect(r)}>{r.display_name}</button>
					</li>
				{/each}
			{/if}
		</ul>
	{/if}
</div>

<style>
	.ui-map-geocode-search {
		position: relative;
		width: 100%;
	}
	input {
		width: 100%;
		padding: 0.5rem 0.75rem;
		border: 1px solid var(--color-border, #d4d4d8);
		border-radius: 0.375rem;
		font: inherit;
		background: var(--color-surface, #fff);
	}
	input:focus {
		outline: 2px solid var(--color-focus, #2563eb);
		outline-offset: -1px;
	}
	.ui-map-geocode-results {
		position: absolute;
		top: 100%;
		left: 0;
		right: 0;
		margin: 0.25rem 0 0;
		padding: 0.25rem 0;
		list-style: none;
		background: var(--color-surface, #fff);
		border: 1px solid var(--color-border, #d4d4d8);
		border-radius: 0.375rem;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
		z-index: 20;
		max-height: 18rem;
		overflow-y: auto;
	}
	.ui-map-geocode-results__row {
		padding: 0;
	}
	.ui-map-geocode-results__row button {
		display: block;
		width: 100%;
		padding: 0.5rem 0.75rem;
		text-align: left;
		background: none;
		border: 0;
		cursor: pointer;
		font: inherit;
		color: inherit;
	}
	.ui-map-geocode-results__row button:hover,
	.ui-map-geocode-results__row button:focus {
		background: var(--color-hover, #f4f4f5);
		outline: none;
	}
	.ui-map-geocode-results__row--loading,
	.ui-map-geocode-results__row--empty {
		padding: 0.5rem 0.75rem;
		color: var(--color-muted, #71717a);
		font-style: italic;
	}
	.ui-map-geocode-results__row--error {
		padding: 0.5rem 0.75rem;
		color: var(--color-error, #b91c1c);
	}
</style>
