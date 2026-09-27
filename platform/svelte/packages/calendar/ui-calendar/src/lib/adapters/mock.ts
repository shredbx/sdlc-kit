// In-memory CalendarAdapter — the frontend-first build runs against this.
// Deterministic (monotonic counters, never Date.now()/Math.random()) so the
// contract suite is repeatable. It is the SAME contract a real HTTP adapter will
// satisfy later, so every L0 scenario proven here transfers to the backend.
//
// No `any` — soft guidance is carried as typed Warning[]; only hard, data-
// corrupting failures (end ≤ start) throw.

import type {
	AddRefInput,
	CalendarAdapter,
	CalendarItem,
	CreateEventInput,
	DateRange,
	DictName,
	DictionaryPort,
	DictOption,
	EventStatus,
	MutationResult,
	RefChip,
	UpdateEventInput,
	Warning
} from '../types.js';

// Two events overlap when one starts strictly before the other ends and vice
// versa. Touching edges (a.end === b.start) is NOT an overlap.
function overlaps(aStart: string, aEnd: string, bStart: string, bEnd: string): boolean {
	return aStart < bEnd && bStart < aEnd;
}

// A range [start, end) intersects an item when their intervals overlap on the
// wall-clock timeline; a zero-length range still includes items covering it.
function intersectsRange(item: CalendarItem, range: DateRange): boolean {
	return item.start < range.end && range.start < item.end;
}

/**
 * createMockAdapter — an in-memory, deterministic CalendarAdapter.
 *
 * @param seed optional initial items (deep-copied so the caller's array is never
 *             mutated). Seeded items keep their own ids; new items get `ev-N`.
 *
 * NOTE on `scope`: the mock has no user model yet, so `scope: 'mine' | 'all'` is
 * accepted but ignored — every item is returned regardless of scope. The real
 * adapter will filter by `created_by` (Decision #D12). Documented intentionally.
 */
export function createMockAdapter(seed: CalendarItem[] = []): CalendarAdapter {
	// Deep copy the seed so the adapter owns its state.
	const items: CalendarItem[] = seed.map((it) => ({
		...it,
		refs: it.refs.map((r) => ({ ...r })),
		meta: it.meta ? { ...it.meta } : undefined
	}));

	let eventCounter = 0;
	let refCounter = 0;

	function nextEventId(): string {
		eventCounter += 1;
		return `ev-${eventCounter}`;
	}
	function nextRefId(): string {
		refCounter += 1;
		return `ref-${refCounter}`;
	}

	function find(id: string): CalendarItem {
		const item = items.find((it) => it.id === id);
		if (!item) throw new Error(`event not found: ${id}`);
		return item;
	}

	// Return a defensive copy so callers can't mutate adapter state by reference.
	function clone(item: CalendarItem): CalendarItem {
		return {
			...item,
			refs: item.refs.map((r) => ({ ...r })),
			meta: item.meta ? { ...item.meta } : undefined
		};
	}

	// Soft overlap warning against every OTHER item (A1 — warn, never block).
	function overlapWarnings(item: CalendarItem): Warning[] {
		const clash = items.some(
			(other) => other.id !== item.id && overlaps(item.start, item.end, other.start, other.end)
		);
		return clash
			? [{ code: 'overlap', message: 'You have an overlapping event in this time range.' }]
			: [];
	}

	return {
		// scope is intentionally ignored by the mock (no user model yet).
		async list(range: DateRange, _scope: 'mine' | 'all'): Promise<CalendarItem[]> {
			void _scope;
			return items.filter((it) => intersectsRange(it, range)).map(clone);
		},

		async create(input: CreateEventInput): Promise<MutationResult> {
			// A7 — bad times (end ≤ start, zero-length) are a hard validation error.
			if (input.end <= input.start) {
				throw new Error('event end must be after start');
			}
			const item: CalendarItem = {
				id: nextEventId(),
				start: input.start,
				end: input.end,
				title: input.title?.trim() ? input.title : input.eventType,
				eventType: input.eventType,
				status: 'scheduled',
				// Persist notes so the round-trip is lossless (was dropped before).
				notes: input.notes?.trim() ? input.notes : undefined,
				refs: []
			};
			// Compute overlap BEFORE pushing so the new item isn't compared to itself.
			const warnings = overlapWarnings(item);
			items.push(item);
			return { item: clone(item), warnings };
		},

		// Patch an event's own fields (type/title/notes/all-day). Only the keys
		// present in the patch are applied; times stay with move(), status with
		// setStatus(). Clearing the title falls back to the event type, matching
		// create()'s default (O2).
		async update(id: string, patch: UpdateEventInput): Promise<MutationResult> {
			const item = find(id);
			if (patch.eventType !== undefined && patch.eventType.trim()) {
				item.eventType = patch.eventType;
			}
			if (patch.title !== undefined) {
				item.title = patch.title.trim() ? patch.title : item.eventType;
			}
			if (patch.notes !== undefined) {
				item.notes = patch.notes.trim() ? patch.notes : undefined;
			}
			if (patch.allDay !== undefined) {
				item.allDay = patch.allDay;
			}
			return { item: clone(item), warnings: [] };
		},

		async move(id: string, start: string, end: string): Promise<MutationResult> {
			if (end <= start) {
				throw new Error('event end must be after start');
			}
			const item = find(id);
			item.start = start;
			item.end = end;
			return { item: clone(item), warnings: overlapWarnings(item) };
		},

		async setStatus(id: string, status: EventStatus): Promise<MutationResult> {
			const item = find(id);
			item.status = status;
			// A cancelled item STAYS in the list (S3) — status change only.
			return { item: clone(item), warnings: [] };
		},

		async addReference(id: string, ref: AddRefInput): Promise<MutationResult> {
			const item = find(id);
			// A3 — dedupe on (refType, refId, relation). If already present, return
			// the item unchanged + a duplicate_ref warning (one ref, not two).
			const existing = item.refs.find(
				(r) => r.refType === ref.refType && r.refId === ref.refId && r.relation === ref.relation
			);
			if (existing) {
				return {
					item: clone(item),
					warnings: [
						{
							code: 'duplicate_ref',
							message: 'This entity is already linked to the event with the same relation.'
						}
					]
				};
			}
			// Snapshot a durable cached label so the chip survives a future delete of the
			// target entity (A5/A6). Prefer the caller's snapshotted label (the picker has
			// the real display name); fall back to `refType:refId` when none was supplied.
			const chip: RefChip = {
				referenceId: nextRefId(),
				refType: ref.refType,
				refId: ref.refId,
				relation: ref.relation,
				asType: ref.asType,
				label: ref.label?.trim() ? ref.label : `${ref.refType}:${ref.refId}`
			};
			item.refs.push(chip);
			return { item: clone(item), warnings: [] };
		},

		async removeReference(id: string, referenceId: string): Promise<MutationResult> {
			const item = find(id);
			item.refs = item.refs.filter((r) => r.referenceId !== referenceId);
			return { item: clone(item), warnings: [] };
		}
	};
}

// Canonical event types (D6). A consumer can pass its own seed; these mirror the
// design's examples + the BR booking seed (viewing/meeting/key-handover/reminder).
const DEFAULT_DICT_SEED: Partial<Record<DictName, DictOption[]>> = {
	event_type: [
		{ code: 'meeting', label: 'Meeting' },
		{ code: 'viewing', label: 'Viewing' },
		{ code: 'visit', label: 'Visit' },
		{ code: 'call', label: 'Call' },
		{ code: 'key-handover', label: 'Key handover' },
		{ code: 'reminder', label: 'Reminder' }
	]
};

// Normalize a free-text label into a stable dict code: lowercased, spaces → '-',
// non-[a-z0-9-] stripped, collapsed dashes. "Key Handover!" → "key-handover".
function toDictCode(label: string): string {
	return label
		.trim()
		.toLowerCase()
		.replace(/\s+/g, '-')
		.replace(/[^a-z0-9-]/g, '')
		.replace(/-+/g, '-')
		.replace(/^-|-$/g, '');
}

/**
 * createMockDictionary — an in-memory, deterministic DictionaryPort (Level 1 § ②).
 *
 * @param seed optional initial options per dict (deep-copied). Defaults seed
 *             `event_type` with the canonical list above; other dicts start empty.
 *
 * `create` normalizes the label to a code and DEDUPES (A8): an existing code OR a
 * case-insensitively equal label returns the existing option — never a near-dup row.
 */
export function createMockDictionary(
	seed: Partial<Record<DictName, DictOption[]>> = DEFAULT_DICT_SEED
): DictionaryPort {
	const store = new Map<DictName, DictOption[]>();
	for (const [dict, opts] of Object.entries(seed) as [DictName, DictOption[]][]) {
		store.set(
			dict,
			opts.map((o) => ({ ...o }))
		);
	}

	function list(dict: DictName): DictOption[] {
		let opts = store.get(dict);
		if (!opts) {
			opts = [];
			store.set(dict, opts);
		}
		return opts;
	}

	return {
		async options(dict: DictName): Promise<DictOption[]> {
			return list(dict).map((o) => ({ ...o }));
		},

		async create(dict: DictName, label: string): Promise<DictOption> {
			const opts = list(dict);
			const code = toDictCode(label);
			const trimmed = label.trim();
			// A8 — dedupe by code or case-insensitive label; return the existing row.
			const existing = opts.find(
				(o) => o.code === code || o.label.toLowerCase() === trimmed.toLowerCase()
			);
			if (existing) return { ...existing };
			const option: DictOption = { code, label: trimmed };
			opts.push(option);
			return { ...option };
		}
	};
}
