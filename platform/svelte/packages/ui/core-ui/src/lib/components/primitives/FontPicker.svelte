<script lang="ts" module>
	/** A brand/system font row — `label` overrides the displayed name. */
	export interface BrandFont {
		family: string;
		label?: string;
	}

	/** A catalogue (Google) font row. `category` is shown as a quiet hint. */
	export interface CatalogueFont {
		family: string;
		category?: string;
	}

	const RECENT_KEY = 'sbx.fonts.recent.v1';
	const RECENT_CAP = 5;
	const WINDOW_STEP = 40;

	/** Read the persisted recent-families list (newest first). SSR-safe. */
	function readRecent(): string[] {
		if (typeof localStorage === 'undefined') return [];
		try {
			const raw = localStorage.getItem(RECENT_KEY);
			if (!raw) return [];
			const parsed: unknown = JSON.parse(raw);
			if (!Array.isArray(parsed)) return [];
			return parsed.filter((x): x is string => typeof x === 'string').slice(0, RECENT_CAP);
		} catch {
			return [];
		}
	}

	/** Push `family` to the front of the recent list (dedupe, cap). Returns the new list. */
	function pushRecent(list: string[], family: string): string[] {
		const next = [family, ...list.filter((f) => f !== family)].slice(0, RECENT_CAP);
		if (typeof localStorage !== 'undefined') {
			try {
				localStorage.setItem(RECENT_KEY, JSON.stringify(next));
			} catch {
				/* private mode / quota — recent is best-effort */
			}
		}
		return next;
	}
</script>

<script lang="ts">
	/**
	 * FontPicker — a searchable typeahead over the full Google Fonts catalogue,
	 * with Brand and Recent shortcuts, replacing a brand-only `<Select>`.
	 *
	 * Composes the core-ui `Input` + `Icon` primitives and the `@sbx/core-ui/fonts`
	 * loader: each visible Recent/All row lazily `ensureGoogleFont`s itself (via an
	 * IntersectionObserver) so its label previews in its OWN face; selecting awaits
	 * the same loader before firing `onselect`. The All-fonts list is WINDOWED
	 * (first ~40 matches + "Show more") so a 1500-font catalogue never all renders.
	 *
	 * Theme-agnostic: uses the shared `--field-*` / `--color-*` core-ui tokens (the
	 * same contract Input/Select/Textarea read), so a consumer that themes those —
	 * including the canvas chrome — restyles this with zero prop changes.
	 */
	import Input from './Input.svelte';
	import Icon from './Icon.svelte';
	import { ensureGoogleFont } from '../../fonts';

	interface Props {
		/** Current font family. */
		value: string;
		/** Brand/system fonts, shown first (assumed already loaded). */
		brandFonts?: BrandFont[];
		/** Full Google catalogue (may be `[]` until loaded). */
		catalogue?: CatalogueFont[];
		/** Disable the trigger. */
		disabled?: boolean;
		/** Accessible name for the trigger (the picker has no associated <label>); the
		 *  chosen family rides as the visible text. Default 'Font family'. */
		label?: string;
		/** Called with the chosen family. */
		onselect: (family: string) => void;
	}

	let {
		value,
		brandFonts = [],
		catalogue = [],
		disabled = false,
		label = 'Font family',
		onselect
	}: Props = $props();

	let open = $state(false);
	let query = $state('');
	let recent = $state<string[]>([]);
	let windowSize = $state(WINDOW_STEP);

	let rootEl: HTMLDivElement | undefined = $state();
	let listEl: HTMLUListElement | undefined = $state();

	const listboxId = `fontpicker-list-${Math.random().toString(36).slice(2, 9)}`;

	// Recent is read once on mount (localStorage), then kept in sync on select.
	$effect(() => {
		recent = readRecent();
	});

	const q = $derived(query.trim().toLowerCase());

	// Brand rows visible for the current query (filter on family OR label).
	const brandMatches = $derived(
		brandFonts.filter(
			(f) =>
				q === '' ||
				f.family.toLowerCase().includes(q) ||
				(f.label?.toLowerCase().includes(q) ?? false)
		)
	);

	// Recent families that still match the query AND aren't already a brand row
	// (brand takes precedence so a family isn't listed twice).
	const brandFamilySet = $derived(new Set(brandFonts.map((f) => f.family)));
	const recentMatches = $derived(
		recent.filter((f) => !brandFamilySet.has(f) && (q === '' || f.toLowerCase().includes(q)))
	);

	// Catalogue filtered by query — the full match set (windowed at render).
	const catalogueMatches = $derived(
		catalogue.filter((f) => q === '' || f.family.toLowerCase().includes(q))
	);
	const visibleCatalogue = $derived(catalogueMatches.slice(0, windowSize));
	const hasMore = $derived(catalogueMatches.length > windowSize);

	function openMenu() {
		if (disabled) return;
		open = true;
		windowSize = WINDOW_STEP;
		// The search Input auto-focuses on mount; since the menu is conditionally
		// rendered, it remounts (and re-focuses) on every open.
	}

	function closeMenu() {
		open = false;
		query = '';
	}

	function toggleMenu() {
		if (open) closeMenu();
		else openMenu();
	}

	async function choose(family: string) {
		// Make sure the face is paintable before we commit + close.
		await ensureGoogleFont(family);
		recent = pushRecent(recent, family);
		onselect(family);
		closeMenu();
	}

	function showMore() {
		windowSize += WINDOW_STEP;
	}

	// Close on outside click / Escape; basic keyboard open from the trigger.
	function onTriggerKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' || e.key === ' ' || e.key === 'ArrowDown') {
			e.preventDefault();
			openMenu();
		}
	}

	function onMenuKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			e.preventDefault();
			closeMenu();
		}
	}

	function onDocPointerDown(e: PointerEvent) {
		if (!open) return;
		if (rootEl && !rootEl.contains(e.target as Node)) closeMenu();
	}

	$effect(() => {
		if (typeof document === 'undefined') return;
		document.addEventListener('pointerdown', onDocPointerDown, true);
		return () => document.removeEventListener('pointerdown', onDocPointerDown, true);
	});

	// Reset the window when the query changes so a new search starts at the top.
	$effect(() => {
		void q;
		windowSize = WINDOW_STEP;
	});

	/**
	 * Svelte action — lazily load `family` when the row scrolls into view so its
	 * label previews in its own face. One observer per row; disconnects on destroy.
	 * Falls back to an eager load when IntersectionObserver is unavailable (jsdom).
	 */
	function previewFont(node: HTMLElement, family: string) {
		let io: IntersectionObserver | null = null;
		const load = () => {
			void ensureGoogleFont(family);
		};
		if (typeof IntersectionObserver === 'undefined') {
			load();
		} else {
			io = new IntersectionObserver(
				(entries) => {
					for (const entry of entries) {
						if (entry.isIntersecting) {
							load();
							io?.disconnect();
							io = null;
							break;
						}
					}
				},
				{ root: listEl ?? null, rootMargin: '64px' }
			);
			io.observe(node);
		}
		return {
			destroy() {
				io?.disconnect();
			}
		};
	}

	const displayValue = $derived(value || 'Select font');
</script>

<div class="font-picker" bind:this={rootEl}>
	<button
		type="button"
		class="font-picker__trigger"
		{disabled}
		aria-label={`${label}: ${displayValue}`}
		aria-haspopup="listbox"
		aria-expanded={open}
		aria-controls={listboxId}
		onclick={toggleMenu}
		onkeydown={onTriggerKeydown}
	>
		<span class="font-picker__value" style="font-family: '{value}';" title={displayValue}>
			{displayValue}
		</span>
		<Icon name="chevron-down" size="sm" class="font-picker__chevron" />
	</button>

	{#if open}
		<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
		<div class="font-picker__menu" onkeydown={onMenuKeydown} role="presentation">
			<div class="font-picker__search">
				<Input
					bind:value={query}
					type="search"
					name="font-search"
					autocomplete="off"
					placeholder="Search fonts…"
					size="sm"
					leftIcon="🔍"
					autofocus
					aria-label="Search fonts"
				/>
			</div>

			<ul
				class="font-picker__list"
				id={listboxId}
				role="listbox"
				aria-label="Fonts"
				bind:this={listEl}
			>
				{#if brandMatches.length > 0}
					<li class="font-picker__group" role="presentation">Brand</li>
					{#each brandMatches as f (f.family)}
						<li role="presentation">
							<button
								type="button"
								class="font-picker__option"
								class:is-current={f.family === value}
								role="option"
								aria-selected={f.family === value}
								onclick={() => choose(f.family)}
							>
								<span class="font-picker__name" style="font-family: '{f.family}';">
									{f.label ?? f.family}
								</span>
							</button>
						</li>
					{/each}
				{/if}

				{#if recentMatches.length > 0}
					<li class="font-picker__group" role="presentation">Recent</li>
					{#each recentMatches as family (family)}
						<li role="presentation" use:previewFont={family}>
							<button
								type="button"
								class="font-picker__option"
								class:is-current={family === value}
								role="option"
								aria-selected={family === value}
								onclick={() => choose(family)}
							>
								<span class="font-picker__name" style="font-family: '{family}';">{family}</span>
							</button>
						</li>
					{/each}
				{/if}

				<li class="font-picker__group" role="presentation">All fonts</li>
				{#if catalogue.length === 0}
					<li class="font-picker__empty" role="presentation">Catalogue unavailable</li>
				{:else if catalogueMatches.length === 0}
					<li class="font-picker__empty" role="presentation">No fonts match "{query}"</li>
				{:else}
					{#each visibleCatalogue as f (f.family)}
						<li role="presentation" use:previewFont={f.family}>
							<button
								type="button"
								class="font-picker__option"
								class:is-current={f.family === value}
								role="option"
								aria-selected={f.family === value}
								onclick={() => choose(f.family)}
							>
								<span class="font-picker__name" style="font-family: '{f.family}';">{f.family}</span>
								{#if f.category}
									<span class="font-picker__category">{f.category}</span>
								{/if}
							</button>
						</li>
					{/each}
					{#if hasMore}
						<li class="font-picker__more-row" role="presentation">
							<button type="button" class="font-picker__more" onclick={showMore}>
								Show more ({catalogueMatches.length - windowSize})
							</button>
						</li>
					{/if}
				{/if}
			</ul>
		</div>
	{/if}
</div>

<style>
	.font-picker {
		position: relative;
		width: 100%;
	}

	/* Trigger mirrors the Select field surface via the shared --field-* tokens, so
	   it sits flush beside other core-ui form controls in any theme. */
	.font-picker__trigger {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem;
		width: 100%;
		height: var(--field-height, auto);
		padding: var(--field-padding-y, 0.5rem) var(--field-padding-x, 0.75rem);
		background: var(--field-bg, var(--input-bg, var(--color-bg-tertiary, #252540)));
		border: var(--field-border-width, 1px) solid
			var(--field-border-color, var(--input-border, var(--color-border, #2a2a4a)));
		border-radius: var(--field-radius, var(--input-radius, 0.375rem));
		color: var(--field-text, var(--input-text, var(--color-text, #fff)));
		font-size: var(--field-font-size, 0.875rem);
		cursor: pointer;
		transition: border-color 0.15s ease;
	}

	.font-picker__trigger:hover:not(:disabled) {
		border-color: var(
			--field-border-hover,
			var(--input-border-hover, var(--color-border-hover, #3a3a5a))
		);
	}

	.font-picker__trigger:focus-visible {
		outline: none;
		border-color: var(--field-border-focus, var(--input-border-focus, var(--color-accent, #6366f1)));
		box-shadow: 0 0 0 3px var(--field-ring, var(--input-ring, rgba(99, 102, 241, 0.2)));
	}

	.font-picker__trigger:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.font-picker__value {
		min-width: 0;
		text-align: left;
		/* Show the whole family name — never clip per the no-truncate rule. */
		white-space: normal;
		overflow-wrap: anywhere;
	}

	:global(.font-picker__chevron) {
		flex-shrink: 0;
		color: var(--color-text-muted, #6b7280);
	}

	.font-picker__menu {
		position: absolute;
		top: calc(100% + 0.25rem);
		left: 0;
		right: 0;
		z-index: 30;
		display: flex;
		flex-direction: column;
		max-height: 22rem;
		padding: 0.375rem;
		background: var(--field-bg, var(--color-surface, var(--color-bg-secondary, #1a1a2e)));
		border: 1px solid var(--field-border-color, var(--color-border, #2a2a4a));
		border-radius: var(--field-radius, 0.5rem);
		box-shadow: 0 12px 32px rgba(0, 0, 0, 0.32);
	}

	.font-picker__search {
		margin-bottom: 0.375rem;
	}

	.font-picker__list {
		margin: 0;
		padding: 0;
		list-style: none;
		overflow-y: auto;
		min-height: 0;
	}

	.font-picker__group {
		padding: 0.5rem 0.5rem 0.25rem;
		font-size: 0.6875rem;
		font-weight: 600;
		letter-spacing: 0.04em;
		text-transform: uppercase;
		color: var(--color-text-muted, #9ca3af);
	}

	.font-picker__option {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 0.75rem;
		width: 100%;
		padding: 0.4375rem 0.5rem;
		background: none;
		border: 0;
		border-radius: var(--field-radius, 0.375rem);
		color: inherit;
		text-align: left;
		font-size: 0.9375rem;
		cursor: pointer;
	}

	.font-picker__option:hover,
	.font-picker__option:focus-visible {
		background: var(--color-hover, rgba(255, 255, 255, 0.06));
		outline: none;
	}

	.font-picker__option.is-current {
		background: var(--color-accent-soft, rgba(99, 102, 241, 0.16));
	}

	.font-picker__name {
		min-width: 0;
		overflow-wrap: anywhere;
	}

	.font-picker__category {
		flex-shrink: 0;
		font-size: 0.6875rem;
		color: var(--color-text-muted, #9ca3af);
		font-family: inherit;
	}

	.font-picker__empty {
		padding: 0.5rem;
		font-size: 0.8125rem;
		font-style: italic;
		color: var(--color-text-muted, #9ca3af);
	}

	/* "Show more" is a row-level CTA — right-aligned per the CTA rule. */
	.font-picker__more-row {
		display: flex;
		justify-content: flex-end;
		padding: 0.25rem;
	}

	.font-picker__more {
		padding: 0.3125rem 0.625rem;
		background: none;
		border: 0;
		border-radius: var(--field-radius, 0.375rem);
		color: var(--color-accent, #6366f1);
		font-size: 0.8125rem;
		font-weight: 500;
		cursor: pointer;
	}

	.font-picker__more:hover,
	.font-picker__more:focus-visible {
		background: var(--color-hover, rgba(255, 255, 255, 0.06));
		outline: none;
	}
</style>
