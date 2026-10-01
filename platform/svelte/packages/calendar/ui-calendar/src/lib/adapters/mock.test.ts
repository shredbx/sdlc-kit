// Contract suite for the in-memory CalendarAdapter, mapping L0 scenarios.
// This SAME suite will later run against createBookingHttpAdapter — the
// frontend-first guarantee. Asia/Bangkok wall-clock (no DST) in the fixtures.

import { describe, expect, it } from 'vitest';
import { createMockAdapter, createMockDictionary } from './mock.js';

// A full month range so list() returns everything we create in these tests.
const MONTH: { start: string; end: string } = {
	start: '2026-05-01T00:00:00+07:00',
	end: '2026-06-01T00:00:00+07:00'
};

describe('createMockAdapter — CalendarAdapter contract', () => {
	// S1 — create a viewing → scheduled, no warnings.
	it('create viewing → status scheduled, warnings empty (S1)', async () => {
		const cal = createMockAdapter();
		const { item, warnings } = await cal.create({
			eventType: 'viewing',
			start: '2026-05-12T14:00:00+07:00',
			end: '2026-05-12T14:30:00+07:00'
		});
		expect(item.status).toBe('scheduled');
		expect(item.eventType).toBe('viewing');
		expect(item.title).toBe('viewing'); // O2 — title defaults to event_type
		expect(item.refs).toEqual([]);
		expect(warnings).toEqual([]);
	});

	// S3 — calendar CRUD: move reschedules; setStatus('cancelled') keeps the item listed.
	it('move updates times; setStatus(cancelled) keeps the item in list (S3)', async () => {
		const cal = createMockAdapter();
		const { item } = await cal.create({
			eventType: 'viewing',
			start: '2026-05-12T14:00:00+07:00',
			end: '2026-05-12T14:30:00+07:00'
		});

		const moved = await cal.move(item.id, '2026-05-13T09:00:00+07:00', '2026-05-13T10:00:00+07:00');
		expect(moved.item.start).toBe('2026-05-13T09:00:00+07:00');
		expect(moved.item.end).toBe('2026-05-13T10:00:00+07:00');

		const cancelled = await cal.setStatus(item.id, 'cancelled');
		expect(cancelled.item.status).toBe('cancelled');

		// Greyed, NOT deleted — still returned by list().
		const listed = await cal.list(MONTH, 'all');
		const found = listed.find((it) => it.id === item.id);
		expect(found).toBeDefined();
		expect(found?.status).toBe('cancelled');
	});

	// A1 — overlapping creates → 2nd warns 'overlap' AND is still created (warn, never block).
	it('two overlapping creates → 2nd has overlap warning and is still created (A1)', async () => {
		const cal = createMockAdapter();
		const first = await cal.create({
			eventType: 'viewing',
			start: '2026-05-12T14:00:00+07:00',
			end: '2026-05-12T15:00:00+07:00'
		});
		expect(first.warnings).toEqual([]);

		const second = await cal.create({
			eventType: 'meeting',
			start: '2026-05-12T14:30:00+07:00',
			end: '2026-05-12T15:30:00+07:00'
		});
		expect(second.warnings.some((w) => w.code === 'overlap')).toBe(true);

		// Still created — both appear on the calendar.
		const listed = await cal.list(MONTH, 'all');
		expect(listed.map((it) => it.id)).toEqual(
			expect.arrayContaining([first.item.id, second.item.id])
		);
	});

	// A7 — bad times (end ≤ start) → hard validation error, no item created.
	it('create with end <= start → rejects (A7)', async () => {
		const cal = createMockAdapter();
		await expect(
			cal.create({
				eventType: 'viewing',
				start: '2026-05-12T14:30:00+07:00',
				end: '2026-05-12T14:00:00+07:00'
			})
		).rejects.toThrow();

		// Zero-length is also rejected.
		await expect(
			cal.create({
				eventType: 'viewing',
				start: '2026-05-12T14:00:00+07:00',
				end: '2026-05-12T14:00:00+07:00'
			})
		).rejects.toThrow();

		const listed = await cal.list(MONTH, 'all');
		expect(listed).toEqual([]);
	});

	// A3 — same (refType, refId, relation) twice → exactly one ref + duplicate_ref warning.
	it('addReference twice (same refType/refId/relation) → one ref + duplicate_ref warning (A3)', async () => {
		const cal = createMockAdapter();
		const { item } = await cal.create({
			eventType: 'viewing',
			start: '2026-05-12T14:00:00+07:00',
			end: '2026-05-12T14:30:00+07:00'
		});

		const first = await cal.addReference(item.id, {
			refType: 'contact',
			refId: 'ct-1',
			relation: 'attendee',
			asType: 'Landlord'
		});
		expect(first.item.refs).toHaveLength(1);
		expect(first.warnings).toEqual([]);

		const second = await cal.addReference(item.id, {
			refType: 'contact',
			refId: 'ct-1',
			relation: 'attendee',
			asType: 'Landlord'
		});
		expect(second.item.refs).toHaveLength(1); // still exactly one
		expect(second.warnings.some((w) => w.code === 'duplicate_ref')).toBe(true);
	});

	// removeReference — the only contract method previously uncovered. Add → remove →
	// gone; and an unknown referenceId is a silent no-op (pins the contract the HTTP
	// adapter must match, not a throw).
	it('removeReference deletes the ref; unknown id is a silent no-op', async () => {
		const cal = createMockAdapter();
		const { item } = await cal.create({
			eventType: 'viewing',
			start: '2026-05-12T14:00:00+07:00',
			end: '2026-05-12T14:30:00+07:00'
		});
		const added = await cal.addReference(item.id, { refType: 'contact', refId: 'ct-9' });
		expect(added.item.refs).toHaveLength(1);
		const refId = added.item.refs[0].referenceId;

		const removed = await cal.removeReference(item.id, refId);
		expect(removed.item.refs).toEqual([]);

		// Unknown referenceId → item returned unchanged, no throw.
		const noop = await cal.removeReference(item.id, 'ref-does-not-exist');
		expect(noop.item.refs).toEqual([]);
	});

	// A5/A6 — the cached label is a SNAPSHOT (refType:refId), independent of any source
	// entity, and survives later mutations. This is the riskiest thing for the HTTP
	// adapter to get right (snapshot at addReference time, not join-at-list time).
	it('ref label is a cached snapshot that survives move + setStatus (A5/A6)', async () => {
		const cal = createMockAdapter();
		const { item } = await cal.create({
			eventType: 'viewing',
			start: '2026-05-12T14:00:00+07:00',
			end: '2026-05-12T14:30:00+07:00'
		});
		await cal.addReference(item.id, { refType: 'contact', refId: 'ct-7', relation: 'attendee' });

		await cal.move(item.id, '2026-05-13T09:00:00+07:00', '2026-05-13T10:00:00+07:00');
		const after = await cal.setStatus(item.id, 'confirmed');
		expect(after.item.refs).toHaveLength(1);
		expect(after.item.refs[0].label).toBe('contact:ct-7'); // snapshot, unchanged
	});

	// When the caller supplies a label (the picker has the real display name), it is
	// snapshotted verbatim; absent a label it falls back to refType:refId.
	it('addReference snapshots a supplied label, else synthesizes refType:refId', async () => {
		const cal = createMockAdapter();
		const { item } = await cal.create({
			eventType: 'viewing',
			start: '2026-05-12T14:00:00+07:00',
			end: '2026-05-12T14:30:00+07:00'
		});
		const withLabel = await cal.addReference(item.id, { refType: 'contact', refId: 'ct-1', label: 'Somchai' });
		expect(withLabel.item.refs[0].label).toBe('Somchai');

		const noLabel = await cal.addReference(item.id, { refType: 'property', refId: 'pr-9' });
		expect(noLabel.item.refs.find((r) => r.refId === 'pr-9')?.label).toBe('property:pr-9');
	});

	// move() promises the SAME hard-error semantics as create() (A7). A drag/resize can
	// produce end <= start (e.g. an upward resize past the start); it must reject and
	// leave the original times intact.
	it('move with end <= start → rejects, original times unchanged (A7 on move)', async () => {
		const cal = createMockAdapter();
		const { item } = await cal.create({
			eventType: 'viewing',
			start: '2026-05-12T14:00:00+07:00',
			end: '2026-05-12T15:00:00+07:00'
		});
		await expect(
			cal.move(item.id, '2026-05-12T15:00:00+07:00', '2026-05-12T14:00:00+07:00')
		).rejects.toThrow();
		// Zero-length too.
		await expect(
			cal.move(item.id, '2026-05-12T14:00:00+07:00', '2026-05-12T14:00:00+07:00')
		).rejects.toThrow();

		const listed = await cal.list(MONTH, 'all');
		const found = listed.find((it) => it.id === item.id);
		expect(found?.start).toBe('2026-05-12T14:00:00+07:00'); // untouched
		expect(found?.end).toBe('2026-05-12T15:00:00+07:00');
	});

	// The soft-state lifecycle (D9/F1): every status transition keeps the item listed,
	// not just 'cancelled'. Proves the whole scheduled→confirmed→done|no_show path.
	it('setStatus to confirmed/done/no_show each keeps the item in list', async () => {
		const cal = createMockAdapter();
		const { item } = await cal.create({
			eventType: 'viewing',
			start: '2026-05-12T14:00:00+07:00',
			end: '2026-05-12T14:30:00+07:00'
		});
		for (const status of ['confirmed', 'done', 'no_show'] as const) {
			const res = await cal.setStatus(item.id, status);
			expect(res.item.status).toBe(status);
			const listed = await cal.list(MONTH, 'all');
			expect(listed.find((it) => it.id === item.id)?.status).toBe(status);
		}
	});

	// intersectsRange half-open boundary contract (the most-called method). A backend
	// WHERE clause (`start < $end AND $start < end`) must match these exactly: touching
	// edges are EXCLUDED; a zero-length range strictly inside is INCLUDED.
	it('list() range boundaries are half-open (touching edges excluded)', async () => {
		const cal = createMockAdapter();
		const { item } = await cal.create({
			eventType: 'viewing',
			start: '2026-05-12T10:00:00+07:00',
			end: '2026-05-12T11:00:00+07:00'
		});
		const has = async (start: string, end: string) =>
			(await cal.list({ start, end }, 'all')).some((it) => it.id === item.id);

		// range end == item.start → excluded.
		expect(await has('2026-05-12T09:00:00+07:00', '2026-05-12T10:00:00+07:00')).toBe(false);
		// range start == item.end → excluded.
		expect(await has('2026-05-12T11:00:00+07:00', '2026-05-12T12:00:00+07:00')).toBe(false);
		// zero-length range strictly inside → included.
		expect(await has('2026-05-12T10:30:00+07:00', '2026-05-12T10:30:00+07:00')).toBe(true);
	});

	// scope:'mine'|'all' is intentionally ignored by the mock (no user model yet). Pin
	// that invariant so a future contributor can't silently diverge the two paths.
	it("scope is ignored — list('mine') equals list('all')", async () => {
		const cal = createMockAdapter();
		await cal.create({
			eventType: 'viewing',
			start: '2026-05-12T14:00:00+07:00',
			end: '2026-05-12T14:30:00+07:00'
		});
		const mine = await cal.list(MONTH, 'mine');
		const all = await cal.list(MONTH, 'all');
		expect(mine).toEqual(all);
	});

	// notes round-trip — create persists notes (was dropped before), and they survive
	// list(). The HTTP adapter must do the same (the editor relies on it).
	it('create persists notes and they survive list()', async () => {
		const cal = createMockAdapter();
		const { item } = await cal.create({
			eventType: 'viewing',
			start: '2026-05-12T14:00:00+07:00',
			end: '2026-05-12T14:30:00+07:00',
			notes: 'Bring the lease draft'
		});
		expect(item.notes).toBe('Bring the lease draft');
		const listed = await cal.list(MONTH, 'all');
		expect(listed.find((it) => it.id === item.id)?.notes).toBe('Bring the lease draft');
	});

	// update() is the editor contract — patch type/title/notes/all-day independently;
	// move (times) and setStatus (status) are untouched. Only present keys apply.
	it('update patches type/title/notes/allDay; times + status untouched', async () => {
		const cal = createMockAdapter();
		const { item } = await cal.create({
			eventType: 'meeting',
			start: '2026-05-12T14:00:00+07:00',
			end: '2026-05-12T14:30:00+07:00'
		});
		const edited = await cal.update(item.id, {
			eventType: 'viewing',
			title: 'Viewing — Condo A',
			notes: 'Top floor',
			allDay: false
		});
		expect(edited.item.eventType).toBe('viewing');
		expect(edited.item.title).toBe('Viewing — Condo A');
		expect(edited.item.notes).toBe('Top floor');
		// Untouched by update.
		expect(edited.item.start).toBe('2026-05-12T14:00:00+07:00');
		expect(edited.item.status).toBe('scheduled');

		// A partial patch only touches its keys (notes left as-is here).
		const partial = await cal.update(item.id, { title: 'Renamed' });
		expect(partial.item.title).toBe('Renamed');
		expect(partial.item.notes).toBe('Top floor');
	});

	// O2 — clearing the title falls back to the event type (matches create()).
	it('update with a blank title falls back to the event type (O2)', async () => {
		const cal = createMockAdapter();
		const { item } = await cal.create({
			eventType: 'viewing',
			title: 'Custom',
			start: '2026-05-12T14:00:00+07:00',
			end: '2026-05-12T14:30:00+07:00'
		});
		const cleared = await cal.update(item.id, { title: '   ' });
		expect(cleared.item.title).toBe('viewing');
	});
});

// ② DictionaryPort contract (S2 create-on-the-fly, A8 dedupe+normalize). The same
// suite later runs against an HTTP-backed dictionary.
describe('createMockDictionary — DictionaryPort contract', () => {
	it('options(event_type) returns the canonical seed', async () => {
		const dict = createMockDictionary();
		const opts = await dict.options('event_type');
		expect(opts.map((o) => o.code)).toEqual(
			expect.arrayContaining(['meeting', 'viewing', 'key-handover'])
		);
	});

	// S2 — create-on-the-fly: a new label becomes a usable option immediately.
	it('create(event_type, "Site inspection") → new normalized code, then listed (S2)', async () => {
		const dict = createMockDictionary();
		const created = await dict.create('event_type', 'Site inspection');
		expect(created.code).toBe('site-inspection');
		expect(created.label).toBe('Site inspection');
		const opts = await dict.options('event_type');
		expect(opts.some((o) => o.code === 'site-inspection')).toBe(true);
	});

	// A8 — dedupe+normalize: a duplicate label (any case) returns the EXISTING option,
	// not a near-duplicate row.
	it('create dedupes by code and case-insensitive label (A8)', async () => {
		const dict = createMockDictionary();
		const before = (await dict.options('event_type')).length;

		// "Key handover" already exists as code 'key-handover' → returns it.
		const dup = await dict.create('event_type', 'KEY handover');
		expect(dup.code).toBe('key-handover');
		// Creating the same new label twice yields one row.
		await dict.create('event_type', 'Open house');
		await dict.create('event_type', 'open house');
		const opts = await dict.options('event_type');
		expect(opts.filter((o) => o.code === 'open-house')).toHaveLength(1);
		// Only ONE net-new row ('open-house') was added.
		expect(opts.length).toBe(before + 1);
	});

	it('options returns a defensive copy — mutating it does not corrupt the port', async () => {
		const dict = createMockDictionary();
		const opts = await dict.options('event_type');
		opts.push({ code: 'hacked', label: 'Hacked' });
		const again = await dict.options('event_type');
		expect(again.some((o) => o.code === 'hacked')).toBe(false);
	});
});
