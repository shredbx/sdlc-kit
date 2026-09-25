<script lang="ts" module>
	import type {
		SourceProvider,
		SourceRecord
	} from '@sbx/canvas-kit';

	/** How rows render — a flat list of rows, or a responsive thumb-on-top grid. */
	export type SourcePickerView = 'list' | 'card';
	/** Single click-and-close, or multi checkbox-select with a sticky confirm. */
	export type SourcePickerSelect = 'single' | 'multi';
	/** Category rail rendering — `auto` picks tabs (≤6 groups) or a dropdown. */
	export type SourcePickerCategoryDisplay = 'auto' | 'tabs' | 'dropdown';

	/** Public prop contract — consumers import this for typed usage. */
	export interface SourcePickerProps {
		/** The data-source adapter — the picker drives ONLY this kit contract. */
		provider: SourceProvider;
		/** 'single' (default) = click picks + closes · 'multi' = checkbox-select + confirm. */
		select?: SourcePickerSelect;
		/** Single-select WITH a confirm step (radio + footer) instead of click-and-close —
		 *  pre-highlights `selectedIds`, and the footer button emits the chosen record via
		 *  `onpick`. Ignored when select='multi' (multi always confirms). Use for "replace
		 *  the current selection" flows where an accidental click must not commit. */
		confirm?: boolean;
		/** Footer button label for the single-confirm step (default 'Select') — e.g. 'Replace'. */
		confirmLabel?: string;
		/** Pre-selected record ids — multi pre-checks each; single+confirm pre-highlights the
		 *  current selection (only the first id is honoured in single-confirm radio mode). */
		selectedIds?: string[];
		/** 'list' (default) = rows · 'card' = thumb-on-top grid. Toggled in-header; no refetch. */
		view?: SourcePickerView;
		/** 'auto' (default) tabs ≤6 categories else dropdown · 'tabs'/'dropdown' force it. */
		categoryDisplay?: SourcePickerCategoryDisplay;
		/** A record was chosen (select='single') — the caller resolves + maps the verb. */
		onpick?: (record: SourceRecord) => void;
		/** Selection confirmed (select='multi') — the caller maps the verb for each record. */
		onconfirm?: (records: SourceRecord[]) => void;
	}
</script>

<script lang="ts">
	// Source record picker (slices 4a/4b + SP-2/3/4, D21/D22) — the attach-modal body.
	// Generic: it drives ONLY the kit SourceProvider interface, so any project's provider
	// works. Search (debounced) + optional category rail/dropdown (browse) + optional facet
	// checkboxes (e.g. drafts) + paginated load-more, in either a list or a card grid. Three
	// selection modes: single click-and-close (default) · multi checkbox-select + Add(N) ·
	// single-confirm (radio + footer, pre-highlights the current selection — the "replace"
	// flow). Picking bubbles up; the caller resolves the snapshot + maps the verb
	// (attach · bind · replace).
	//
	// SCOPE MODEL (4b): a provider that implements categories() gets a browse rail
	// (groups + "All", with counts). SEARCH OVERRIDES BROWSE — a non-empty query is
	// always a global flat list() (the kit's listByCategory takes no query); selecting
	// a category clears the search box and browses that group via listByCategory().
	// Load-more re-calls whichever scope produced the current page, with its cursor.
	import { onMount, onDestroy, untrack } from 'svelte';
	import { Input, Button, Icon, Select } from '@sbx/core-ui/components/primitives';
	import type { SourceCategory, SourceFacet, SourceFilters, SourcePage } from '@sbx/canvas-kit';
	// `SourceProvider`, `SourceRecord` (from @sbx/canvas-kit) and `SourcePickerProps` are declared
	// in the <script module> block above and are in scope here — re-importing them duplicates the
	// identifiers (TS: "Duplicate identifier" / "conflicts with local declaration").

	let {
		provider,
		select = 'single',
		confirm = false,
		confirmLabel = 'Select',
		selectedIds,
		view = 'list',
		categoryDisplay = 'auto',
		onpick,
		onconfirm
	}: SourcePickerProps = $props();

	// One provider per mount (the modal opens for a single kind); seed facets + filters
	// once (untrack makes the one-time read explicit — mirrors SettingsPanel's idiom).
	const facets: SourceFacet[] = untrack(() => provider.facets?.() ?? []);
	const hasCategories = untrack(() => typeof provider.categories === 'function');
	let filters = $state<SourceFilters>(
		untrack(() => Object.fromEntries(facets.map((f) => [f.key, f.default ?? false])))
	);

	let categories = $state<SourceCategory[]>([]);
	// Selected category id, or null for "All" (the flat list). Persists while searching.
	let activeCategory = $state<string | null>(null);

	let query = $state('');
	let records = $state<SourceRecord[]>([]);
	let nextCursor = $state<string | null>(null);
	let loading = $state(false);
	let loadingMore = $state(false);
	let error = $state<string | null>(null);

	// View is session state — the header toggle flips it WITHOUT refetching (records are
	// already in memory; only the markup differs). Seeded from the prop once.
	let activeView = $state<SourcePickerView>(untrack(() => view));

	// Multi-select state. Track selection by record id (a Set for O(1) toggles) and keep a
	// parallel id→record map so confirm() can return full SourceRecords even for rows that
	// have since scrolled out of `records` or were selected on an earlier page. Seed from
	// `selectedIds` once; the ids alone preselect — their records resolve as pages load and
	// are merged in `mergeSelectedRecords()` so a confirm without re-seeing them still works.
	// Seeded ONCE at mount — the picker does NOT react to later `selectedIds` changes; re-seed
	// by remounting (e.g. an `open`-keyed modal). Fits the modal-open-on-current-selection use (design §3).
	let selectedIdSet = $state<Set<string>>(untrack(() => new Set(selectedIds ?? [])));
	let selectedRecords = $state<Map<string, SourceRecord>>(new Map());

	// While a query is active the results are global flat search, so the rail shows
	// "All" as selected (the category selection is suspended, not lost).
	const displayActive = $derived(query.trim() ? null : activeCategory);

	const isMulti = $derived(select === 'multi');
	// Single-select WITH a confirm step (radio semantics): pre-highlight + footer, emits onpick.
	const isConfirm = $derived(select === 'single' && confirm);
	// Modes that carry a persistent selection (highlight + footer + selectedIds seed + record-merge).
	const showSelection = $derived(isMulti || isConfirm);
	const selectedCount = $derived(selectedIdSet.size);
	// Single-confirm guard: the footer must NOT commit until the chosen record is actually
	// loaded, so confirmSelection() always emits a full SourceRecord — never a silent dead click
	// when the pre-highlighted record still sits on an unfetched page. It self-enables the moment
	// that record arrives in a page (merge) or the user clicks any visible row (choose populates it).
	const confirmReady = $derived.by(() => {
		if (!isConfirm) return true;
		const id = [...selectedIdSet][0];
		return id != null && (selectedRecords.has(id) || records.some((r) => r.id === id));
	});

	// Category presentation: 'auto' → dropdown when more than 6 groups, else tabs.
	const showCategoryControl = $derived(hasCategories && categories.length > 0);
	const useDropdown = $derived(
		categoryDisplay === 'dropdown' || (categoryDisplay === 'auto' && categories.length > 6)
	);
	// Options for the dropdown form of the rail — leading "All" maps to the flat list.
	const categoryOptions = $derived([
		{ value: '', label: 'All' },
		...categories.map((c) => ({
			value: c.id,
			label: c.count != null ? `${c.label} (${c.count})` : c.label
		}))
	]);

	let debounce: ReturnType<typeof setTimeout> | null = null;
	// Monotonic guard — a newer fetch (search / category switch / facet toggle)
	// invalidates an in-flight older one (out-of-order responses, stale scope).
	let runId = 0;

	/**
	 * Pick the right provider call for the current scope. Search overrides browse:
	 * a non-empty query is always a flat list(); a selected category with no query
	 * browses that group via listByCategory().
	 */
	function fetchScope(cursor?: string): Promise<SourcePage> {
		if (activeCategory && !query.trim() && provider.listByCategory) {
			return provider.listByCategory(activeCategory, cursor, filters);
		}
		return provider.list(query, cursor, filters);
	}

	// Merge any freshly-seen records into the selection map for currently-selected ids, so
	// confirmSelection() returns full SourceRecords even for preselected ids whose row only
	// just arrived in a page (selection-by-id, record-on-sight). Applies to both selection
	// modes — single-confirm pre-highlights the current record, which must resolve on sight
	// so an unchanged "Replace" still emits it.
	function mergeSelectedRecords(page: SourceRecord[]): void {
		if (!showSelection || selectedIdSet.size === 0) return;
		let changed = false;
		const next = new Map(selectedRecords);
		for (const r of page) {
			if (selectedIdSet.has(r.id) && !next.has(r.id)) {
				next.set(r.id, r);
				changed = true;
			}
		}
		if (changed) selectedRecords = next;
	}

	async function loadCategories(): Promise<void> {
		if (!provider.categories) return;
		try {
			categories = await provider.categories();
		} catch {
			categories = []; // rail just hides; the flat list still works.
		}
	}

	async function search(): Promise<void> {
		const id = ++runId;
		loading = true;
		error = null;
		try {
			const page = await fetchScope();
			if (id !== runId) return;
			records = page.records;
			nextCursor = page.nextCursor ?? null;
			mergeSelectedRecords(page.records);
		} catch {
			if (id !== runId) return;
			error = 'Could not load records.';
			records = [];
			nextCursor = null;
		} finally {
			if (id === runId) loading = false;
		}
	}

	async function loadMore(): Promise<void> {
		if (!nextCursor || loadingMore) return;
		// Same out-of-order guard as search(): if a fresh search / scope switch fires
		// while this load-more is in flight, drop the stale page rather than append it.
		const id = runId;
		loadingMore = true;
		error = null;
		try {
			const page = await fetchScope(nextCursor);
			if (id !== runId) return;
			records = [...records, ...page.records];
			nextCursor = page.nextCursor ?? null;
			mergeSelectedRecords(page.records);
		} catch {
			if (id === runId) error = 'Could not load more.';
		} finally {
			if (id === runId) loadingMore = false;
		}
	}

	function onQueryInput(v: string): void {
		query = v;
		if (debounce) clearTimeout(debounce);
		debounce = setTimeout(search, 250);
	}

	function selectCategory(id: string | null): void {
		activeCategory = id;
		query = ''; // selecting a category is browse mode — clear any active search.
		if (debounce) clearTimeout(debounce);
		search();
	}

	function toggleFacet(key: string): void {
		filters = { ...filters, [key]: !filters[key] };
		// Cancel any pending debounced search so the facet change isn't followed by a
		// redundant duplicate fetch (symmetry with selectCategory).
		if (debounce) clearTimeout(debounce);
		search();
	}

	// Choosing a record: single picks-and-closes (caller closes); single-confirm sets the lone
	// radio selection (replace, never toggles off); multi toggles checked.
	function choose(record: SourceRecord): void {
		if (isConfirm) {
			// Radio: exactly one selection. Clicking another row replaces it; clicking the
			// current one keeps it selected (a radio never deselects on re-click).
			selectedIdSet = new Set([record.id]);
			selectedRecords = new Map([[record.id, record]]);
			return;
		}
		if (!isMulti) {
			onpick?.(record);
			return;
		}
		const nextSet = new Set(selectedIdSet);
		const nextRecords = new Map(selectedRecords);
		if (nextSet.has(record.id)) {
			nextSet.delete(record.id);
			nextRecords.delete(record.id);
		} else {
			nextSet.add(record.id);
			nextRecords.set(record.id, record);
		}
		selectedIdSet = nextSet;
		selectedRecords = nextRecords;
	}

	// Footer confirm — named `confirmSelection` (not `confirm`) so it never shadows the
	// `confirm` prop. Single-confirm emits the lone record via onpick; multi emits the
	// ordered array via onconfirm.
	function confirmSelection(): void {
		if (selectedIdSet.size === 0) return;
		if (isConfirm) {
			const id = [...selectedIdSet][0];
			const rec = selectedRecords.get(id) ?? records.find((r) => r.id === id);
			if (rec) onpick?.(rec);
			return;
		}
		// Preserve picker order: emit selected records in the order they appear in the
		// current list, then any selected-but-unseen records (e.g. preselected ids on a
		// later/closed page) appended in selection order.
		const seen = new Set<string>();
		const ordered: SourceRecord[] = [];
		for (const r of records) {
			if (selectedIdSet.has(r.id) && !seen.has(r.id)) {
				ordered.push(r);
				seen.add(r.id);
			}
		}
		for (const [id, r] of selectedRecords) {
			if (selectedIdSet.has(id) && !seen.has(id)) {
				ordered.push(r);
				seen.add(id);
			}
		}
		onconfirm?.(ordered);
	}

	function onCategoryChange(value: string): void {
		selectCategory(value === '' ? null : value);
	}

	onMount(() => {
		void loadCategories();
		void search();
	});
	onDestroy(() => {
		if (debounce) clearTimeout(debounce);
	});
</script>

<div class="picker" class:picker--multi={isMulti}>
	<div class="picker__header">
		<div class="picker__search">
			<Icon name="search" size="sm" />
			<Input
				name="source-search"
				value={query}
				placeholder={`Search ${provider.label.toLowerCase()}s…`}
				aria-label={`Search ${provider.label.toLowerCase()}s`}
				oninput={onQueryInput}
			/>
		</div>

		<!-- View toggle — list / card. Right-aligned (CTA-side), no refetch on switch. -->
		<div class="picker__view" role="group" aria-label="View">
			<button
				type="button"
				class="view-btn"
				class:view-btn--active={activeView === 'list'}
				aria-pressed={activeView === 'list'}
				title="List view"
				onclick={() => (activeView = 'list')}
			>
				<Icon name="list" size="sm" label="List view" />
			</button>
			<button
				type="button"
				class="view-btn"
				class:view-btn--active={activeView === 'card'}
				aria-pressed={activeView === 'card'}
				title="Card view"
				onclick={() => (activeView = 'card')}
			>
				<Icon name="layout-grid" size="sm" label="Card view" />
			</button>
		</div>
	</div>

	{#if showCategoryControl}
		{#if useDropdown}
			<div class="picker__catselect">
				<!-- Visually-hidden label gives the native select an accessible name
				     (core-ui Select has no aria-label prop; this is the a11y-correct form). -->
				<label class="picker__sr-only" for="source-picker-category">{provider.label} categories</label>
				<Select
					id="source-picker-category"
					options={categoryOptions}
					value={displayActive ?? ''}
					size="sm"
					onchange={onCategoryChange}
				/>
			</div>
		{:else}
			<div class="picker__cats" role="tablist" aria-label={`${provider.label} categories`}>
				<button
					type="button"
					class="cat"
					class:cat--active={displayActive === null}
					role="tab"
					aria-selected={displayActive === null}
					onclick={() => selectCategory(null)}
				>
					All
				</button>
				{#each categories as category (category.id)}
					<button
						type="button"
						class="cat"
						class:cat--active={displayActive === category.id}
						role="tab"
						aria-selected={displayActive === category.id}
						onclick={() => selectCategory(category.id)}
					>
						{category.label}{#if category.count != null}<span class="cat__count">{category.count}</span>{/if}
					</button>
				{/each}
			</div>
		{/if}
	{/if}

	{#if facets.length > 0}
		<div class="picker__facets">
			{#each facets as facet (facet.key)}
				<label class="facet">
					<input type="checkbox" checked={filters[facet.key]} onchange={() => toggleFacet(facet.key)} />
					<span>{facet.label}</span>
				</label>
			{/each}
		</div>
	{/if}

	<!-- Scroll region — rows scroll ABOVE the sticky multi footer (which reserves space). -->
	<div class="picker__body">
		{#if error}
			<p class="picker__msg picker__msg--error" role="alert">{error}</p>
		{/if}

		{#if loading}
			<p class="picker__msg">Loading…</p>
		{:else if records.length === 0}
			<p class="picker__msg">No {provider.label.toLowerCase()}s match.</p>
		{:else if activeView === 'card'}
			<ul
					class="cards"
					role={isConfirm ? 'radiogroup' : undefined}
					aria-label={isConfirm ? `Pick a ${provider.label.toLowerCase()}` : undefined}
				>
				{#each records as record (record.id)}
					{@const checked = selectedIdSet.has(record.id)}
					<li>
						<button
							type="button"
							class="card"
							class:card--selected={showSelection && checked}
							role={isMulti ? 'checkbox' : isConfirm ? 'radio' : undefined}
					aria-checked={showSelection ? checked : undefined}
							onclick={() => choose(record)}
						>
							{#if showSelection}
								<span
									class="card__check"
									class:card__check--radio={isConfirm}
									class:card__check--on={checked}
									aria-hidden="true"
								>
									{#if checked}
										{#if isConfirm}<span class="card__dot"></span>{:else}<Icon name="check" size="xs" />{/if}
									{/if}
								</span>
							{/if}
							<span class="card__thumb">
								{#if record.thumb}
									<img src={record.thumb} alt="" />
								{:else}
									<Icon name="image" size="md" />
								{/if}
							</span>
							<span class="card__text">
								<span class="card__title" title={record.title}>{record.title}</span>
								{#if record.subtitle}<span class="card__subtitle">{record.subtitle}</span>{/if}
							</span>
						</button>
					</li>
				{/each}
			</ul>
		{:else}
			<ul
					class="results"
					role={isConfirm ? 'radiogroup' : undefined}
					aria-label={isConfirm ? `Pick a ${provider.label.toLowerCase()}` : undefined}
				>
				{#each records as record (record.id)}
					{@const checked = selectedIdSet.has(record.id)}
					<li>
						<button
							type="button"
							class="result"
							class:result--selected={showSelection && checked}
							role={isMulti ? 'checkbox' : isConfirm ? 'radio' : undefined}
					aria-checked={showSelection ? checked : undefined}
							onclick={() => choose(record)}
						>
							{#if showSelection}
								<span
									class="result__check"
									class:result__check--radio={isConfirm}
									class:result__check--on={checked}
									aria-hidden="true"
								>
									{#if checked}
										{#if isConfirm}<span class="result__dot"></span>{:else}<Icon name="check" size="xs" />{/if}
									{/if}
								</span>
							{/if}
							<span class="result__thumb">
								{#if record.thumb}
									<img src={record.thumb} alt="" />
								{:else}
									<Icon name="image" size="sm" />
								{/if}
							</span>
							<span class="result__text">
								<span class="result__title" title={record.title}>{record.title}</span>
								{#if record.subtitle}<span class="result__subtitle">{record.subtitle}</span>{/if}
							</span>
						</button>
					</li>
				{/each}
			</ul>
		{/if}

		{#if !loading && records.length > 0 && nextCursor}
			<!-- CTA centred for a load-more affordance (it acts on the list, not a form). -->
			<div class="picker__more">
				<Button variant="secondary" size="sm" disabled={loadingMore} onclick={loadMore}>
					{loadingMore ? 'Loading…' : 'Load more'}
				</Button>
			</div>
		{/if}
	</div>

	{#if showSelection}
		<!-- Sticky confirm — lives below the scroll region (reserves space), CTA right. Shown
		     for multi (Add N) and single-confirm (the Replace step); a plain single has none. -->
		<div class="picker__footer">
			{#if isMulti}
				<!-- Count reads naturally for checkbox multi; in single-confirm it is always 1, so
				     it is omitted (the "Replace" verb carries the meaning). CTA stays right via margin-auto. -->
				<span class="picker__count" aria-live="polite">
					{selectedCount} selected
				</span>
			{/if}
			<div class="picker__confirm">
				<Button
					variant="primary"
					size="sm"
					disabled={selectedCount === 0 || !confirmReady}
					onclick={confirmSelection}
				>
					{isMulti ? `Add (${selectedCount})` : confirmLabel}
				</Button>
			</div>
		</div>
	{/if}
</div>

<style>
	.picker {
		display: flex;
		flex-direction: column;
		gap: var(--br-space-md, 1rem);
		min-width: 22rem;
		max-height: 60vh;
		/* Multi footer is a flow sibling of the scroll body, so the body shrinks and the
		   footer never overlaps the last rows — no fixed/absolute positioning needed. */
	}

	.picker__header {
		display: flex;
		align-items: center;
		gap: var(--br-space-md, 1rem);
	}

	.picker__search {
		display: flex;
		align-items: center;
		gap: var(--br-space-sm, 0.5rem);
		color: var(--br-color-neutral-400, #a0a098);
		flex: 1;
		min-width: 0;
	}

	.picker__search :global(input) {
		flex: 1;
	}

	/* View toggle — two icon buttons, right-aligned (mirrors the .cat button idiom). */
	.picker__view {
		display: inline-flex;
		flex-shrink: 0;
		border: 1px solid var(--br-border-color, #e8e8e0);
		border-radius: var(--br-radius-sm, 0.375rem);
		overflow: hidden;
	}

	.view-btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 2rem;
		height: 1.875rem;
		border: none;
		background: transparent;
		color: var(--br-color-neutral-500, #707068);
		cursor: pointer;
	}

	.view-btn + .view-btn {
		border-left: 1px solid var(--br-border-color, #e8e8e0);
	}

	.view-btn:hover {
		background: var(--br-color-neutral-100, #f5f5f0);
	}

	.view-btn--active {
		background: var(--br-color-primary, #0d4f4f);
		color: #fff;
	}

	.view-btn--active:hover {
		background: var(--br-color-primary, #0d4f4f);
	}

	.picker__cats {
		display: flex;
		flex-wrap: wrap;
		gap: 0.375rem;
	}

	.cat {
		display: inline-flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.25rem 0.625rem;
		border: 1px solid var(--br-border-color, #e8e8e0);
		border-radius: 999px;
		background: transparent;
		font-size: 0.8125rem;
		color: var(--br-color-neutral-700, #383830);
		cursor: pointer;
	}

	.cat:hover {
		background: var(--br-color-neutral-100, #f5f5f0);
	}

	.cat--active {
		border-color: var(--br-color-primary, #0d4f4f);
		background: var(--br-color-primary, #0d4f4f);
		color: #fff;
	}

	.cat__count {
		font-size: 0.6875rem;
		font-variant-numeric: tabular-nums;
		opacity: 0.7;
	}

	.picker__catselect {
		max-width: 16rem;
	}

	/* Visually-hidden (screen-reader-only) — names the category select without visible chrome. */
	.picker__sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		margin: -1px;
		overflow: hidden;
		clip: rect(0, 0, 0, 0);
		white-space: nowrap;
		border: 0;
	}

	.picker__facets {
		display: flex;
		flex-wrap: wrap;
		gap: var(--br-space-md, 1rem);
	}

	.facet {
		display: inline-flex;
		align-items: center;
		gap: 0.375rem;
		font-size: 0.8125rem;
		color: var(--br-color-neutral-700, #383830);
		cursor: pointer;
	}

	.facet input {
		accent-color: var(--br-color-primary, #0d4f4f);
	}

	/* Scroll region — the only scrolling part; the footer sits below it. */
	.picker__body {
		flex: 1;
		min-height: 0;
		overflow-y: auto;
		display: flex;
		flex-direction: column;
		gap: var(--br-space-md, 1rem);
	}

	.picker__msg {
		margin: 0;
		padding: var(--br-space-md, 1rem) 0;
		font-size: 0.8125rem;
		color: var(--br-color-neutral-500, #707068);
		text-align: center;
	}

	.picker__msg--error {
		color: var(--br-color-error, #c0392b);
	}

	.results {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.result {
		display: flex;
		align-items: center;
		gap: var(--br-space-sm, 0.5rem);
		width: 100%;
		padding: var(--br-space-sm, 0.5rem);
		border: 1px solid transparent;
		border-radius: var(--br-radius-sm, 0.375rem);
		background: transparent;
		text-align: left;
		cursor: pointer;
	}

	.result:hover {
		background: var(--br-color-neutral-100, #f5f5f0);
		border-color: var(--br-border-color, #e8e8e0);
	}

	.result--selected {
		border-color: var(--br-color-primary, #0d4f4f);
		background: var(--br-color-primary-soft, #e6f0f0);
	}

	.result--selected:hover {
		background: var(--br-color-primary-soft, #e6f0f0);
		border-color: var(--br-color-primary, #0d4f4f);
	}

	.result__thumb {
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		width: 2.5rem;
		height: 2.5rem;
		border-radius: var(--br-radius-sm, 0.375rem);
		background: var(--br-color-neutral-100, #f5f5f0);
		color: var(--br-color-neutral-400, #a0a098);
		overflow: hidden;
	}

	.result__thumb img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.result__text {
		display: flex;
		flex-direction: column;
		min-width: 0;
	}

	.result__title {
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--br-color-neutral-800, #202018);
		overflow-wrap: anywhere;
	}

	.result__subtitle {
		font-size: 0.75rem;
		color: var(--br-color-neutral-500, #707068);
		overflow-wrap: anywhere;
	}

	/* Checkbox affordance (multi) — a square that fills on select. Shared shape for
	   list + card via the two element classes; never an inline transform. */
	.result__check,
	.card__check {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		width: 1.125rem;
		height: 1.125rem;
		border: 1.5px solid var(--br-border-color, #e8e8e0);
		border-radius: var(--br-radius-sm, 0.25rem);
		background: #fff;
		color: #fff;
	}

	.result__check--on,
	.card__check--on {
		border-color: var(--br-color-primary, #0d4f4f);
		background: var(--br-color-primary, #0d4f4f);
	}

	/* Radio variant (single-confirm) — the same affordance box made round; on select it
	   fills primary and a white dot marks the choice (radio semantics, not a checkmark). */
	.result__check--radio,
	.card__check--radio {
		border-radius: 999px;
	}

	.result__dot,
	.card__dot {
		width: 0.5rem;
		height: 0.5rem;
		border-radius: 999px;
		background: #fff;
	}

	/* Card grid — responsive thumb-on-top tiles (thumb, then title + subtitle below). */
	.cards {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(8.5rem, 1fr));
		gap: var(--br-space-sm, 0.5rem);
	}

	.card {
		position: relative;
		display: flex;
		flex-direction: column;
		gap: var(--br-space-xs, 0.375rem);
		width: 100%;
		padding: var(--br-space-sm, 0.5rem);
		border: 1px solid var(--br-border-color, #e8e8e0);
		border-radius: var(--br-radius-md, 0.5rem);
		background: transparent;
		text-align: left;
		cursor: pointer;
	}

	.card:hover {
		background: var(--br-color-neutral-100, #f5f5f0);
	}

	.card--selected {
		border-color: var(--br-color-primary, #0d4f4f);
		background: var(--br-color-primary-soft, #e6f0f0);
	}

	.card--selected:hover {
		background: var(--br-color-primary-soft, #e6f0f0);
	}

	/* Keyboard focus ring for the row/card buttons (checkbox + radio + plain) — distinct from
	   the selected border so focus and selection read separately (WCAG 2.4.7 Focus Visible). */
	.result:focus-visible,
	.card:focus-visible {
		outline: 2px solid var(--br-color-primary, #0d4f4f);
		outline-offset: 2px;
	}

	/* Card check floats top-left over the thumb so the tile stays compact. */
	.card__check {
		position: absolute;
		top: calc(var(--br-space-sm, 0.5rem) + 0.25rem);
		left: calc(var(--br-space-sm, 0.5rem) + 0.25rem);
		z-index: 1;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.15);
	}

	.card__thumb {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 100%;
		aspect-ratio: 4 / 3;
		border-radius: var(--br-radius-sm, 0.375rem);
		background: var(--br-color-neutral-100, #f5f5f0);
		color: var(--br-color-neutral-400, #a0a098);
		overflow: hidden;
	}

	.card__thumb img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.card__text {
		display: flex;
		flex-direction: column;
		gap: 0.125rem;
		min-width: 0;
	}

	.card__title {
		font-size: 0.8125rem;
		font-weight: 500;
		color: var(--br-color-neutral-800, #202018);
		overflow-wrap: anywhere;
	}

	.card__subtitle {
		font-size: 0.6875rem;
		color: var(--br-color-neutral-500, #707068);
		overflow-wrap: anywhere;
	}

	.picker__more {
		display: flex;
		justify-content: center;
	}

	/* Sticky multi footer — a flow sibling below the scroll body. CTA right-aligned. */
	.picker__footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--br-space-md, 1rem);
		flex-shrink: 0;
		padding-top: var(--br-space-sm, 0.5rem);
		border-top: 1px solid var(--br-border-color, #e8e8e0);
	}

	.picker__count {
		font-size: 0.8125rem;
		font-variant-numeric: tabular-nums;
		color: var(--br-color-neutral-600, #50504a);
	}

	.picker__confirm {
		display: flex;
		justify-content: flex-end;
		margin-left: auto;
	}
</style>
