// ref-summary — the PURE count builder the compact event chip uses instead of
// rendering one DOM chip per ref (the #1 "events become towers" complaint).
// Buckets a RefChip[] into three counts by `relation`: attendee → attendees,
// about → properties, anything else (or absent relation) → materials. The chip
// then shows a single compact summary line (e.g. "👥 2 · 📍 1") rather than a
// per-ref tower; the full ref list lives in the preview modal + the detail page.
//
// No `any`, no env, no fetch — safe to import anywhere. Domain-agnostic: the
// package never learns BR's relation labels, only the canonical dict codes.
import type { RefChip } from '../types.js';

export interface RefSummary {
	/** refs whose relation is "attendee" (people on the event). */
	attendees: number;
	/** refs whose relation is "about" (the property/thing the event concerns). */
	properties: number;
	/** every other ref — a different relation, or none recorded. */
	materials: number;
}

export function refSummary(refs: RefChip[]): RefSummary {
	const summary: RefSummary = { attendees: 0, properties: 0, materials: 0 };
	for (const ref of refs) {
		if (ref.relation === 'attendee') summary.attendees += 1;
		else if (ref.relation === 'about') summary.properties += 1;
		else summary.materials += 1;
	}
	return summary;
}
