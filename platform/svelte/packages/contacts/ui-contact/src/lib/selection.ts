// Pure, DOM-free selection + scope machine for ContactPickerModal — the testable core
// (mirrors @sbx/ui-calendar's util-extraction: the component stays thin over tested
// helpers, and these run in plain vitest without @testing-library, which is not in the
// workspace). The SAME state shape (a Set<id> + a parallel Map<id,record>) is proven here
// once and reused by the radio (single) and checkbox (multi) flows in the component.
//
// Why a parallel Map: confirm/pick must return FULL records even for preselected ids whose
// row has not scrolled into view (or sits on an unfetched page). The Set drives pre-check /
// pre-highlight; the Map resolves records on sight via mergeSelectedRecords as pages load,
// so a confirm that never re-saw a preselected row still emits its record.

import type { ContactPickerRecord, ContactProvider, ContactPage } from './types';

/** The selection state shared by single (radio) and multi (checkbox) modes. */
export interface SelectionState {
	/** Selected ids — O(1) membership for pre-check / pre-highlight. */
	ids: Set<string>;
	/** id → record, resolved on sight, so confirm returns full records for unseen ids. */
	records: Map<string, ContactPickerRecord>;
}

/** Seed selection from preselected ids (records resolve later as pages load). */
export function seedSelection(selectedIds?: string[]): SelectionState {
	return { ids: new Set(selectedIds ?? []), records: new Map() };
}

/**
 * Multi toggle — flips a record in/out of the selection (returns a NEW state so a runes
 * `$state` reassignment re-renders). Removing also drops the cached record.
 */
export function toggleSelection(state: SelectionState, record: ContactPickerRecord): SelectionState {
	const ids = new Set(state.ids);
	const records = new Map(state.records);
	if (ids.has(record.id)) {
		ids.delete(record.id);
		records.delete(record.id);
	} else {
		ids.add(record.id);
		records.set(record.id, record);
	}
	return { ids, records };
}

/**
 * Single (radio) selection — exactly one. Re-selecting the current row keeps it (a radio
 * never deselects on re-click); selecting another replaces it.
 */
export function setSingleSelection(record: ContactPickerRecord): SelectionState {
	return { ids: new Set([record.id]), records: new Map([[record.id, record]]) };
}

/** Drop one id from the selection (the removable SELECTED chip in multi mode). */
export function deselect(state: SelectionState, id: string): SelectionState {
	const ids = new Set(state.ids);
	const records = new Map(state.records);
	ids.delete(id);
	records.delete(id);
	return { ids, records };
}

/**
 * Resolve full records for any selected ids first seen in `page` (record-on-sight). Returns
 * a NEW state only when something changed (so the caller can skip a no-op reassignment).
 */
export function mergeSelectedRecords(
	state: SelectionState,
	page: ContactPickerRecord[]
): SelectionState {
	if (state.ids.size === 0) return state;
	let changed = false;
	const records = new Map(state.records);
	for (const r of page) {
		if (state.ids.has(r.id) && !records.has(r.id)) {
			records.set(r.id, r);
			changed = true;
		}
	}
	return changed ? { ids: new Set(state.ids), records } : state;
}

/**
 * Multi confirm order — selected records in the order they appear in the current list, then
 * any selected-but-unseen records (e.g. preselected ids on a closed page) appended in
 * selection (Map-insertion) order. Preserves "ordered as picked" per the contract.
 */
export function orderedSelectedRecords(
	state: SelectionState,
	visible: ContactPickerRecord[]
): ContactPickerRecord[] {
	const seen = new Set<string>();
	const ordered: ContactPickerRecord[] = [];
	for (const r of visible) {
		if (state.ids.has(r.id) && !seen.has(r.id)) {
			ordered.push(r);
			seen.add(r.id);
		}
	}
	for (const [id, r] of state.records) {
		if (state.ids.has(id) && !seen.has(id)) {
			ordered.push(r);
			seen.add(id);
		}
	}
	return ordered;
}

/** Resolve the single chosen record (Map first, then the visible list). */
export function singleSelectedRecord(
	state: SelectionState,
	visible: ContactPickerRecord[]
): ContactPickerRecord | undefined {
	const id = [...state.ids][0];
	if (id == null) return undefined;
	return state.records.get(id) ?? visible.find((r) => r.id === id);
}

/**
 * Single-mode footer guard: never commit until the chosen record is actually loaded, so the
 * footer always emits a full record (no silent dead click when the pre-highlighted row sits
 * on an unfetched page). Self-enables the moment that record arrives or any row is clicked.
 */
export function singleConfirmReady(state: SelectionState, visible: ContactPickerRecord[]): boolean {
	return singleSelectedRecord(state, visible) != null;
}

/**
 * Scope router — SEARCH OVERRIDES BROWSE. A non-empty query is always a flat list(); a
 * selected category with an empty query browses that group via listByCategory(). When the
 * picker is locked to one `category` prop the rail is hidden and that code is the browse
 * scope. Identical decision tree to SourcePicker.fetchScope, made pure + testable.
 */
export function fetchScope(
	provider: ContactProvider,
	args: { query: string; activeCategory: string | null; lockedCategory?: string },
	cursor?: string
): Promise<ContactPage> {
	const q = args.query.trim();
	const browseCode = args.lockedCategory ?? args.activeCategory ?? null;
	if (!q && browseCode && provider.listByCategory) {
		return provider.listByCategory(browseCode, cursor);
	}
	return provider.list(args.query, cursor);
}
