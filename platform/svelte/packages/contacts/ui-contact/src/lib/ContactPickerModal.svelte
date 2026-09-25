<script lang="ts">
	// ContactPickerModal — reusable contact picker as a CENTERED MODAL (backdrop + panel),
	// decoupled from any app. It drives ONLY the ContactProvider contract (./types), so any
	// project maps its domain record to ContactPickerRecord and implements a provider over
	// its own API — the picker never fetches directly.
	//
	// Modeled on @sbx/ui-source-picker/SourcePicker (debounced search · monotonic runId guard ·
	// category rail with auto tabs/dropdown · selection-by-id + record-on-sight Map · load-more),
	// but with a CONTACT-NATIVE presentation: avatar (photo or initials) + wrapping name +
	// trailing category badge with a deterministic color dot + muted phone meta. Multi mode adds
	// a removable SELECTED-chips row above the list (the "Add contacts" mockup); single mode is a
	// radio list with an "Assign" footer (the "Assign contact" mockup).
	//
	// The selection/ordering/scope machine lives in ./selection (pure, unit-tested) — this file
	// is the thin presentational + IO shell over it. The modal shell is self-contained (the
	// package cannot import an app's AdminModal); its CSS mirrors BR's ContactPicker modal.
	//
	// SCOPE MODEL: SEARCH OVERRIDES BROWSE — a non-empty query is always a flat list(); selecting
	// a category clears the search and browses that group via listByCategory(). A `category` prop
	// locks the picker to one category: the rail hides and inline-create receives that code.

	import { onMount, onDestroy, untrack } from 'svelte';
	import { Avatar, Button, Icon, Input, Select } from '@sbx/core-ui/components/primitives';
	import type { ContactCategory, ContactPage, ContactPickerProps, ContactPickerRecord } from './types';
	import { categoryColor } from './palette';
	import {
		deselect,
		fetchScope,
		mergeSelectedRecords,
		orderedSelectedRecords,
		seedSelection,
		setSingleSelection,
		singleConfirmReady,
		singleSelectedRecord,
		toggleSelection,
		type SelectionState
	} from './selection';

	let {
		provider,
		open = $bindable(false),
		mode = 'single',
		category,
		defaultCategory,
		confirmLabel = 'Assign',
		selectedIds,
		onpick,
		onconfirm,
		onclose,
		createForm
	}: ContactPickerProps = $props();

	const isMulti = $derived(mode === 'multi');
	const headingId = $props.id();
	const title = $derived(isMulti ? 'Add contacts' : 'Assign contact');

	// Categories are seeded once per mount. The rail renders only when the provider implements
	// categories() AND the picker is NOT hard-locked to a single `category` (locked → rail hidden).
	// A `defaultCategory` is NOT a lock (category is unset), so the rail stays browsable — it only
	// pre-selects the opening group below.
	const hasCategories = untrack(() => typeof provider.categories === 'function');
	const railEnabled = $derived(hasCategories && category == null);

	let categories = $state<ContactCategory[]>([]);
	// Selected category id, or null for "All" (the flat list). Persists while searching. Opens on
	// `category` (hard lock) ?? `defaultCategory` (soft default) ?? null ("All"), so the modal's
	// FIRST search() pre-filters to that role — the user can then click "All"/another group to broaden.
	let activeCategory = $state<string | null>(untrack(() => category ?? defaultCategory ?? null));

	let query = $state('');
	let records = $state<ContactPickerRecord[]>([]);
	let nextCursor = $state<string | null>(null);
	let loading = $state(false);
	let loadingMore = $state(false);
	let error = $state<string | null>(null);

	// "+ Create new" inline form visibility (only meaningful when a createForm snippet is given).
	let creating = $state(false);

	// Selection state — a Set<id> + a parallel id→record Map (see ./selection). Seeded once from
	// `selectedIds`; the picker does not react to later prop changes (re-seed via an open-keyed
	// remount). Multi pre-checks each id; single pre-highlights the first.
	let selection = $state<SelectionState>(
		untrack(() => {
			const seeded = seedSelection(selectedIds);
			// Single mode honours only the first preselected id (radio semantics).
			if (!isMulti && seeded.ids.size > 1) {
				const first = [...seeded.ids][0];
				return seedSelection(first ? [first] : []);
			}
			return seeded;
		})
	);

	// While a query is active the results are a flat search, so the rail shows "All" selected
	// (the category selection is suspended, not lost).
	const displayActive = $derived(query.trim() ? null : activeCategory);

	// Single-mode footer guard — only commit once the chosen record is actually loaded.
	const confirmReady = $derived.by(() =>
		isMulti ? true : singleConfirmReady(selection, records)
	);

	// Ordered selected records (multi chips + confirm payload) — list order then unseen order.
	const selectedList = $derived(orderedSelectedRecords(selection, records));
	// Footer count == exactly what confirm emits, so the label can never overstate the payload:
	// multi counts RESOLVED selected records (an unseen preselected id is excluded until it loads);
	// single counts the lone id (the actual emit is guarded separately by confirmReady).
	const confirmCount = $derived(isMulti ? selectedList.length : selection.ids.size);

	// Category presentation: auto → dropdown when more than 6 groups, else tabs (SourcePicker's rule).
	const showCategoryControl = $derived(railEnabled && categories.length > 0);
	const useDropdown = $derived(categories.length > 6);
	const categoryOptions = $derived([
		{ value: '', label: 'All' },
		...categories.map((c) => ({
			value: c.code,
			label: c.count != null ? `${c.label} (${c.count})` : c.label
		}))
	]);

	let debounce: ReturnType<typeof setTimeout> | null = null;
	// Monotonic guard — a newer fetch (search / category switch) invalidates an older in-flight one.
	let runId = 0;

	function scope(cursor?: string): Promise<ContactPage> {
		return fetchScope(provider, { query, activeCategory, lockedCategory: category }, cursor);
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
			const page = await scope();
			if (id !== runId) return;
			records = page.records;
			nextCursor = page.nextCursor ?? null;
			selection = mergeSelectedRecords(selection, page.records);
		} catch {
			if (id !== runId) return;
			error = 'Could not load contacts.';
			records = [];
			nextCursor = null;
		} finally {
			if (id === runId) loading = false;
		}
	}

	async function loadMore(): Promise<void> {
		if (!nextCursor || loadingMore) return;
		const id = runId;
		loadingMore = true;
		error = null;
		try {
			const page = await scope(nextCursor);
			if (id !== runId) return;
			records = [...records, ...page.records];
			nextCursor = page.nextCursor ?? null;
			selection = mergeSelectedRecords(selection, page.records);
		} catch {
			if (id === runId) error = 'Could not load more.';
		} finally {
			if (id === runId) loadingMore = false;
		}
	}

	function onQueryInput(value: string): void {
		query = value;
		// Invalidate any in-flight fetch synchronously: a Load-more click within the 250ms debounce
		// window would otherwise run with a stale runId and feed a browse cursor into the now-active
		// flat search. Bumping runId makes that interleaved load-more bail.
		runId += 1;
		if (debounce) clearTimeout(debounce);
		debounce = setTimeout(search, 250);
	}

	function selectCategory(code: string | null): void {
		activeCategory = code;
		query = ''; // selecting a category is browse mode — clear any active search.
		if (debounce) clearTimeout(debounce);
		void search();
	}

	function onCategoryChange(value: string): void {
		selectCategory(value === '' ? null : value);
	}

	// Choosing a row: single sets the lone radio selection; multi toggles the checkbox.
	function choose(record: ContactPickerRecord): void {
		selection = isMulti ? toggleSelection(selection, record) : setSingleSelection(record);
	}

	function removeSelected(id: string): void {
		selection = deselect(selection, id);
	}

	function confirmSelection(): void {
		if (selection.ids.size === 0) return;
		if (isMulti) {
			onconfirm?.(selectedList);
		} else {
			const rec = singleSelectedRecord(selection, records);
			if (rec) onpick?.(rec);
		}
	}

	// Inline create-form result — select the new contact, then return to the list. Single picks
	// it immediately (commits via onpick); multi adds it to the selection (the user still confirms).
	function onCreated(record: ContactPickerRecord): void {
		creating = false;
		if (isMulti) {
			// Add-if-absent — a freshly-created contact is never already in the selection.
			if (!selection.ids.has(record.id)) selection = toggleSelection(selection, record);
		} else {
			selection = setSingleSelection(record);
			onpick?.(record);
		}
	}

	function openCreate(): void {
		creating = true;
	}

	function cancelCreate(): void {
		creating = false;
	}

	function close(): void {
		open = false;
		onclose?.();
	}

	function onBackdropClick(event: MouseEvent): void {
		if (event.target === event.currentTarget) close();
	}

	function onKeydown(event: KeyboardEvent): void {
		if (event.key === 'Escape') {
			event.preventDefault();
			close();
		}
	}

	// Trailing badge label + the muted "+N" when a contact spans multiple categories
	// (computed in script — no inline transforms in markup).
	function primaryLabel(record: ContactPickerRecord): string {
		const code = record.categoryCodes[0];
		if (!code) return '';
		return provider.categoryLabel?.(code) ?? code;
	}
	function extraCategoryCount(record: ContactPickerRecord): number {
		return Math.max(0, record.categoryCodes.length - 1);
	}

	onMount(() => {
		void loadCategories();
		void search();
	});
	onDestroy(() => {
		if (debounce) clearTimeout(debounce);
	});
</script>

{#if open}
	{#if creating && createForm}
		<!-- Create REPLACES the picker overlay — the snippet owns its own full surface (a
		     FullscreenCard), so the rich section form gets real width instead of the 30rem
		     panel. onCreated selects the new record + returns; cancel() returns to the list. -->
		{@render createForm({ onCreated, cancel: cancelCreate, category })}
	{:else}
		<!-- Backdrop closes on an outside click; Escape bubbles to onKeydown. The panel needs no
		     stopPropagation (the backdrop only closes when the click lands on itself). -->
		<div
			class="cpicker__backdrop"
			role="presentation"
			onclick={onBackdropClick}
			onkeydown={onKeydown}
		>
			<div class="cpicker__panel" role="dialog" aria-modal="true" aria-labelledby={headingId}>
				<header class="cpicker__header">
					<h2 id={headingId} class="cpicker__title">{title}</h2>
					<button type="button" class="cpicker__close" onclick={close} aria-label="Close">
						<Icon name="x" size="sm" />
					</button>
				</header>

				<div class="cpicker__search">
					<Icon name="search" size="sm" />
					<Input
						name="contact-picker-search"
						type="search"
						autocomplete="off"
						value={query}
						placeholder="Search contacts…"
						aria-label="Search contacts"
						oninput={onQueryInput}
					/>
				</div>

				{#if showCategoryControl}
					{#if useDropdown}
						<div class="cpicker__catselect">
							<label class="cpicker__sr-only" for="contact-picker-category">Contact categories</label>
							<Select
								id="contact-picker-category"
								options={categoryOptions}
								value={displayActive ?? ''}
								size="sm"
								onchange={onCategoryChange}
							/>
						</div>
					{:else}
						<div class="cpicker__cats" role="tablist" aria-label="Contact categories">
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
							{#each categories as cat (cat.code)}
								<button
									type="button"
									class="cat"
									class:cat--active={displayActive === cat.code}
									role="tab"
									aria-selected={displayActive === cat.code}
									onclick={() => selectCategory(cat.code)}
								>
									<span class="cat__dot" style="background:{categoryColor(cat.code)}"></span>
									{cat.label}{#if cat.count != null}<span class="cat__count">{cat.count}</span>{/if}
								</button>
							{/each}
						</div>
					{/if}
				{/if}

				{#if isMulti && selectedList.length > 0}
					<!-- SELECTED chips (multi) — each currently-selected contact as a removable chip.
					     Required by the "Add contacts" mockup; absent from SourcePicker. -->
					<div class="cpicker__chips" aria-label="Selected contacts">
						{#each selectedList as record (record.id)}
							<span class="chip">
								<span class="chip__name">{record.name}</span>
								<button
									type="button"
									class="chip__remove"
									onclick={() => removeSelected(record.id)}
									aria-label={`Remove ${record.name}`}
								>
									<Icon name="x" size="xs" />
								</button>
							</span>
						{/each}
					</div>
				{/if}

				<div class="cpicker__body">
					{#if error}
						<p class="cpicker__msg cpicker__msg--error" role="alert">{error}</p>
					{/if}

					{#if loading}
						<p class="cpicker__msg">Loading…</p>
					{:else if records.length === 0}
						<div class="cpicker__empty">
							<p class="cpicker__msg">No contacts match.</p>
							{#if createForm}
								<Button variant="secondary" size="sm" onclick={openCreate}>+ Create new</Button>
							{/if}
						</div>
					{:else}
						<ul class="rows" role={isMulti ? undefined : 'radiogroup'} aria-label={isMulti ? undefined : 'Pick a contact'}>
							{#each records as record (record.id)}
								{@const checked = selection.ids.has(record.id)}
								{@const label = primaryLabel(record)}
								{@const extra = extraCategoryCount(record)}
								<li>
									<button
										type="button"
										class="row"
										class:row--selected={checked}
										role={isMulti ? 'checkbox' : 'radio'}
										aria-checked={checked}
										onclick={() => choose(record)}
									>
										<span
											class="row__check"
											class:row__check--radio={!isMulti}
											class:row__check--on={checked}
											aria-hidden="true"
										>
											{#if checked}
												{#if isMulti}<Icon name="check" size="xs" />{:else}<span class="row__dot"></span>{/if}
											{/if}
										</span>
										<span class="row__avatar">
											<Avatar
												src={record.photoUrl ?? undefined}
												name={record.name}
												initials={record.initials}
												alt={record.name}
												size="md"
											/>
										</span>
										<span class="row__main">
											<span class="row__name">{record.name}</span>
											{#if record.phone}<span class="row__phone">{record.phone}</span>{/if}
										</span>
										{#if label}
											<span class="row__badge">
												<span class="row__badge-dot" style="background:{categoryColor(record.categoryCodes[0])}"></span>
												<span class="row__badge-label">{label}</span>
												{#if extra > 0}<span class="row__badge-extra">+{extra}</span>{/if}
											</span>
										{/if}
									</button>
								</li>
							{/each}
						</ul>

						{#if nextCursor}
							<div class="cpicker__more">
								<Button variant="secondary" size="sm" disabled={loadingMore} onclick={loadMore}>
									{loadingMore ? 'Loading…' : 'Load more'}
								</Button>
							</div>
						{/if}
					{/if}
				</div>

				<footer class="cpicker__footer">
					{#if createForm && records.length > 0}
						<!-- Always-available create affordance (the empty state has its own). Left-side,
						     a tertiary action — the primary CTA stays right per the CTA rule. -->
						<Button variant="ghost" size="sm" onclick={openCreate}>+ Create new</Button>
					{/if}
					{#if isMulti}
						<span class="cpicker__count" aria-live="polite">{confirmCount} selected</span>
					{/if}
					<div class="cpicker__confirm">
						<Button
							variant="primary"
							size="sm"
							disabled={confirmCount === 0 || !confirmReady}
							onclick={confirmSelection}
						>
							{isMulti ? `Add (${confirmCount})` : confirmLabel}
						</Button>
					</div>
				</footer>
			</div>
		</div>
	{/if}
{/if}

<style>
	/* Centered modal shell — self-contained (the package cannot import an app's AdminModal).
	   Mirrors BR ContactPicker's proven backdrop/panel: fixed full-viewport backdrop, click
	   outside closes, Escape closes, high z-index. */
	.cpicker__backdrop {
		position: fixed;
		inset: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: var(--br-space-xl, 2rem);
		background: rgba(0, 0, 0, 0.5);
		z-index: 1000;
		overflow-y: auto;
	}

	.cpicker__panel {
		display: flex;
		flex-direction: column;
		gap: var(--br-space-md, 1rem);
		width: 100%;
		max-width: 30rem;
		max-height: calc(100vh - 4 * var(--br-space-xl, 2rem));
		padding: var(--br-space-lg, 1.5rem);
		background: var(--br-color-surface, #ffffff);
		border-radius: var(--br-radius-lg, 0.75rem);
		box-shadow: var(--br-shadow-card, 0 10px 40px rgba(0, 0, 0, 0.25));
	}

	.cpicker__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--br-space-md, 1rem);
	}

	.cpicker__title {
		margin: 0;
		font-family: var(--br-font-heading, inherit);
		font-size: var(--br-size-xl, 1.25rem);
		font-weight: 700;
		color: var(--br-color-neutral-900, #14140f);
	}

	.cpicker__close {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		width: 2rem;
		height: 2rem;
		border: 1px solid var(--br-border-color, #e8e8e0);
		border-radius: 999px;
		background: transparent;
		color: var(--br-color-neutral-600, #50504a);
		cursor: pointer;
	}

	.cpicker__close:hover {
		background: var(--br-color-neutral-100, #f5f5f0);
	}

	.cpicker__close:focus-visible {
		outline: 2px solid var(--br-color-primary, #0d4f4f);
		outline-offset: 2px;
	}

	.cpicker__search {
		display: flex;
		align-items: center;
		gap: var(--br-space-sm, 0.5rem);
		color: var(--br-color-neutral-400, #a0a098);
	}

	.cpicker__search :global(input) {
		flex: 1;
	}

	.cpicker__catselect {
		max-width: 16rem;
	}

	.cpicker__sr-only {
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

	.cpicker__cats {
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

	.cat:focus-visible {
		outline: 2px solid var(--br-color-primary, #0d4f4f);
		outline-offset: 2px;
	}

	.cat__dot {
		width: 0.5rem;
		height: 0.5rem;
		border-radius: 999px;
		flex-shrink: 0;
	}

	.cat--active .cat__dot {
		box-shadow: 0 0 0 1px #fff;
	}

	.cat__count {
		font-size: 0.6875rem;
		font-variant-numeric: tabular-nums;
		opacity: 0.7;
	}

	/* SELECTED chips row (multi) — removable contact chips above the list. */
	.cpicker__chips {
		display: flex;
		flex-wrap: wrap;
		gap: 0.375rem;
	}

	.chip {
		display: inline-flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.25rem 0.25rem 0.25rem 0.625rem;
		border: 1px solid var(--br-color-primary, #0d4f4f);
		border-radius: 999px;
		background: var(--br-color-primary-soft, #e6f0f0);
		font-size: 0.8125rem;
		color: var(--br-color-neutral-800, #202018);
	}

	.chip__name {
		overflow-wrap: anywhere;
	}

	.chip__remove {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 1.125rem;
		height: 1.125rem;
		border: none;
		border-radius: 999px;
		background: transparent;
		color: var(--br-color-neutral-600, #50504a);
		cursor: pointer;
	}

	.chip__remove:hover {
		background: rgba(0, 0, 0, 0.08);
	}

	.chip__remove:focus-visible {
		outline: 2px solid var(--br-color-primary, #0d4f4f);
		outline-offset: 1px;
	}

	/* Scroll region — the only scrolling part; the footer sits below it. */
	.cpicker__body {
		flex: 1;
		min-height: 0;
		overflow-y: auto;
		display: flex;
		flex-direction: column;
		gap: var(--br-space-md, 1rem);
	}

	.cpicker__msg {
		margin: 0;
		padding: var(--br-space-md, 1rem) 0;
		font-size: 0.8125rem;
		color: var(--br-color-neutral-500, #707068);
		text-align: center;
	}

	.cpicker__msg--error {
		color: var(--br-color-error, #c0392b);
	}

	.cpicker__empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--br-space-sm, 0.5rem);
		padding: var(--br-space-md, 1rem) 0;
	}

	.rows {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.row {
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

	.row:hover {
		background: var(--br-color-neutral-100, #f5f5f0);
		border-color: var(--br-border-color, #e8e8e0);
	}

	.row--selected {
		border-color: var(--br-color-primary, #0d4f4f);
		background: var(--br-color-primary-soft, #e6f0f0);
	}

	.row--selected:hover {
		background: var(--br-color-primary-soft, #e6f0f0);
		border-color: var(--br-color-primary, #0d4f4f);
	}

	.row:focus-visible {
		outline: 2px solid var(--br-color-primary, #0d4f4f);
		outline-offset: 2px;
	}

	/* Checkbox affordance (multi) — square that fills on select. Round variant = radio (single). */
	.row__check {
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

	.row__check--radio {
		border-radius: 999px;
	}

	.row__check--on {
		border-color: var(--br-color-primary, #0d4f4f);
		background: var(--br-color-primary, #0d4f4f);
	}

	.row__dot {
		width: 0.5rem;
		height: 0.5rem;
		border-radius: 999px;
		background: #fff;
	}

	.row__avatar {
		display: inline-flex;
		flex-shrink: 0;
	}

	.row__main {
		display: flex;
		flex-direction: column;
		min-width: 0;
		flex: 1;
	}

	.row__name {
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--br-color-neutral-800, #202018);
		overflow-wrap: anywhere;
	}

	.row__phone {
		font-size: 0.75rem;
		color: var(--br-color-neutral-500, #707068);
		overflow-wrap: anywhere;
	}

	/* Trailing category badge — color dot + label (+N when a contact spans more categories). */
	.row__badge {
		display: inline-flex;
		align-items: center;
		gap: 0.375rem;
		flex-shrink: 0;
		padding: 0.1875rem 0.5rem;
		border: 1px solid var(--br-border-color, #e8e8e0);
		border-radius: 999px;
		background: var(--br-color-surface, #fff);
		font-size: 0.75rem;
		color: var(--br-color-neutral-700, #383830);
	}

	.row__badge-dot {
		width: 0.5rem;
		height: 0.5rem;
		border-radius: 999px;
		flex-shrink: 0;
	}

	.row__badge-label {
		overflow-wrap: anywhere;
	}

	.row__badge-extra {
		font-size: 0.6875rem;
		font-variant-numeric: tabular-nums;
		color: var(--br-color-neutral-500, #707068);
	}

	.cpicker__more {
		display: flex;
		justify-content: center;
	}

	/* Footer — CTA right-aligned. Optional left-side "+ Create new" (ghost) + multi count. */
	.cpicker__footer {
		display: flex;
		align-items: center;
		gap: var(--br-space-md, 1rem);
		flex-shrink: 0;
		padding-top: var(--br-space-sm, 0.5rem);
		border-top: 1px solid var(--br-border-color, #e8e8e0);
	}

	.cpicker__count {
		font-size: 0.8125rem;
		font-variant-numeric: tabular-nums;
		color: var(--br-color-neutral-600, #50504a);
	}

	.cpicker__confirm {
		display: flex;
		justify-content: flex-end;
		margin-left: auto;
	}
</style>
