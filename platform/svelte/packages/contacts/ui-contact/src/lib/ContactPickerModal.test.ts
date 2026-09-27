// ContactPickerModal — behavioural contract suite.
//
// TEST APPROACH (why no component mount): @testing-library/svelte is NOT in this workspace
// (no package depends on it, it is absent from the lockfile), so a `.svelte` mount + DOM
// assertions cannot run under vitest here. Following the sibling package's proven idiom
// (@sbx/ui-calendar tests its pure ./utils + its mock adapter, NOT the components), the
// component's interaction is implemented as a thin shell over the PURE, DOM-free selection +
// scope machine in ./selection — and that is what we exercise here, driven by the real
// `mockContactProvider`. Each handler the component binds (choose / confirm / selectCategory /
// search input / onCreated) is a one-liner over a function tested below, so this covers the
// required behaviours: single pick, multi select-two-and-confirm-in-order, category switch
// → listByCategory, search → list, and the createForm onCreated selection path.

import { describe, expect, it, vi } from 'vitest';
import { mockContactProvider } from './adapters/mock';
import type { ContactPickerRecord } from './types';
import {
	deselect,
	fetchScope,
	mergeSelectedRecords,
	orderedSelectedRecords,
	seedSelection,
	setSingleSelection,
	singleConfirmReady,
	singleSelectedRecord,
	toggleSelection
} from './selection';

const REC = (id: string, name: string): ContactPickerRecord => ({
	id,
	name,
	categoryCodes: ['landlord']
});

// ── single mode: choosing a row → onpick fires with that record ────────────────────────────
describe('single mode — pick', () => {
	it('choosing a row selects exactly one and singleSelectedRecord resolves it (→ onpick)', () => {
		const onpick = vi.fn<(r: ContactPickerRecord) => void>();
		const visible = [REC('1', 'Somchai'), REC('2', 'Anong')];

		// choose() in single mode === setSingleSelection; the footer emits singleSelectedRecord.
		let selection = setSingleSelection(visible[1]);
		expect([...selection.ids]).toEqual(['2']);

		const chosen = singleSelectedRecord(selection, visible);
		expect(chosen?.id).toBe('2');
		if (chosen) onpick(chosen);
		expect(onpick).toHaveBeenCalledTimes(1);
		expect(onpick).toHaveBeenCalledWith(expect.objectContaining({ id: '2', name: 'Anong' }));

		// Radio semantics: re-selecting the same row keeps it (never deselects).
		selection = setSingleSelection(visible[1]);
		expect([...selection.ids]).toEqual(['2']);
		// Selecting another replaces it.
		selection = setSingleSelection(visible[0]);
		expect([...selection.ids]).toEqual(['1']);
	});

	it('confirm guard stays closed until the chosen record is loaded, then opens', () => {
		// Preselected id whose row has not been seen yet → not ready (no dead click).
		let selection = seedSelection(['2']);
		expect(singleConfirmReady(selection, [])).toBe(false);

		// Once the row arrives in a page (merge), it resolves and the footer enables.
		selection = mergeSelectedRecords(selection, [REC('2', 'Anong')]);
		expect(singleConfirmReady(selection, [])).toBe(true);
		expect(singleSelectedRecord(selection, [])?.name).toBe('Anong');
	});
});

// ── multi mode: select two → confirm fires onconfirm with both, in pick order ───────────────
describe('multi mode — select two and confirm in order', () => {
	it('toggling two rows then confirming yields both records in the order picked', () => {
		const onconfirm = vi.fn<(r: ContactPickerRecord[]) => void>();
		const visible = [REC('1', 'Somchai'), REC('2', 'Anong'), REC('3', 'John')];

		let selection = seedSelection([]);
		selection = toggleSelection(selection, visible[2]); // John first
		selection = toggleSelection(selection, visible[0]); // Somchai second
		expect(selection.ids.size).toBe(2);

		// orderedSelectedRecords is list-order for visible rows (the component's confirm payload).
		const ordered = orderedSelectedRecords(selection, visible);
		onconfirm(ordered);
		expect(onconfirm).toHaveBeenCalledTimes(1);
		expect(onconfirm.mock.calls[0][0].map((r) => r.id)).toEqual(['1', '3']);
	});

	it('unseen preselected ids append in selection order after the visible ones', () => {
		// '9' is preselected but never appears in `visible`; merge resolves it from a later page.
		let selection = seedSelection(['9']);
		selection = mergeSelectedRecords(selection, [REC('9', 'Maria')]);
		const visible = [REC('1', 'Somchai')];
		selection = toggleSelection(selection, visible[0]);

		const ordered = orderedSelectedRecords(selection, visible);
		// Visible row first (id 1), then the unseen-but-selected (id 9).
		expect(ordered.map((r) => r.id)).toEqual(['1', '9']);
	});

	it('a removable chip deselects (deselect drops id + cached record)', () => {
		let selection = seedSelection([]);
		const a = REC('1', 'Somchai');
		const b = REC('2', 'Anong');
		selection = toggleSelection(selection, a);
		selection = toggleSelection(selection, b);
		selection = deselect(selection, '1');
		expect([...selection.ids]).toEqual(['2']);
		expect(selection.records.has('1')).toBe(false);
		expect(orderedSelectedRecords(selection, [a, b]).map((r) => r.id)).toEqual(['2']);
	});
});

// ── category switch → provider.listByCategory; search → provider.list ───────────────────────
describe('scope routing — search overrides browse', () => {
	it('selecting a category (empty query) browses via listByCategory', async () => {
		const provider = mockContactProvider();
		const listByCategory = vi.spyOn(provider, 'listByCategory');
		const list = vi.spyOn(provider, 'list');

		const page = await fetchScope(provider, { query: '', activeCategory: 'landlord' });
		expect(listByCategory).toHaveBeenCalledWith('landlord', undefined);
		expect(list).not.toHaveBeenCalled();
		// The mock returns only landlord rows.
		expect(page.records.every((r) => r.categoryCodes.includes('landlord'))).toBe(true);
		expect(page.records.map((r) => r.id)).toEqual(['1', '2']);
	});

	it('a non-empty query is always a flat list(), even with a category active', async () => {
		const provider = mockContactProvider();
		const listByCategory = vi.spyOn(provider, 'listByCategory');
		const list = vi.spyOn(provider, 'list');

		const page = await fetchScope(provider, { query: 'anong', activeCategory: 'landlord' });
		expect(list).toHaveBeenCalledWith('anong', undefined);
		expect(listByCategory).not.toHaveBeenCalled();
		expect(page.records.map((r) => r.name)).toEqual(['Anong Pol']);
	});

	it('a locked `category` prop browses that code (rail hidden, query empty)', async () => {
		const provider = mockContactProvider();
		const listByCategory = vi.spyOn(provider, 'listByCategory');

		await fetchScope(provider, { query: '', activeCategory: null, lockedCategory: 'buyer' });
		expect(listByCategory).toHaveBeenCalledWith('buyer', undefined);
	});

	// `defaultCategory` mode (browsable, NOT locked): the modal seeds activeCategory = defaultCategory
	// (no lockedCategory), so the FIRST fetch browses that role — yet the rail can switch to "All"
	// (activeCategory = null) to list EVERYONE, so a non-member of the role is still selectable.
	it('a soft default category opens on that group but "All" still lists every contact', async () => {
		const provider = mockContactProvider();
		const listByCategory = vi.spyOn(provider, 'listByCategory');
		const list = vi.spyOn(provider, 'list');

		// Opening filtered to the default role (e.g. a deal's "seller"): empty query, no lock,
		// activeCategory seeded to the default → browse that group via listByCategory.
		const seeded = await fetchScope(provider, { query: '', activeCategory: 'buyer' });
		expect(listByCategory).toHaveBeenCalledWith('buyer', undefined);
		expect(list).not.toHaveBeenCalled();
		expect(seeded.records.map((r) => r.id)).toEqual(['2', '3']); // only buyers

		// User clicks "All" → activeCategory = null, empty query → flat list() of EVERY contact,
		// including a non-buyer (Maria, tenant) the locked picker could never have surfaced.
		const all = await fetchScope(provider, { query: '', activeCategory: null });
		expect(list).toHaveBeenCalledWith('', undefined);
		expect(all.records.map((r) => r.id)).toEqual(['1', '2', '3', '4']);
		expect(all.records.some((r) => !r.categoryCodes.includes('buyer'))).toBe(true);
	});

	it('load-more passes the cursor through the resolved scope', async () => {
		const provider = mockContactProvider();
		const list = vi.spyOn(provider, 'list');
		await fetchScope(provider, { query: 'a', activeCategory: null }, 'CURSOR_42');
		expect(list).toHaveBeenCalledWith('a', 'CURSOR_42');
	});
});

// ── createForm onCreated → selects the new record (single picks it / multi adds it) ─────────
describe('createForm onCreated path', () => {
	it('single mode selects + emits the freshly-created record via onpick', () => {
		const onpick = vi.fn<(r: ContactPickerRecord) => void>();
		const created = REC('new-1', 'Created Contact');

		// onCreated (single): setSingleSelection + onpick.
		const selection = setSingleSelection(created);
		onpick(created);

		expect([...selection.ids]).toEqual(['new-1']);
		expect(onpick).toHaveBeenCalledWith(expect.objectContaining({ id: 'new-1' }));
	});

	it('multi mode adds the freshly-created record to the selection', () => {
		const created = REC('new-2', 'Created Contact');
		let selection = seedSelection(['1']);
		selection = mergeSelectedRecords(selection, [REC('1', 'Somchai')]);

		// onCreated (multi): toggle the new record into the selection.
		selection = toggleSelection(selection, created);

		expect(selection.ids.has('new-2')).toBe(true);
		expect(orderedSelectedRecords(selection, []).map((r) => r.id)).toContain('new-2');
	});
});
