import { describe, it, expect } from 'vitest';
import { refSummary } from './ref-summary.js';
import type { RefChip } from '../types.js';

// RED skeleton (task 2606-003 / TC-001): the pure ref-summary builder the compact event chip
// uses instead of rendering one DOM chip per ref. refSummary(refs) → counts by relation so the
// chip can show e.g. "👥 2 · 📍 1" rather than a tower of labels. Fleshed in the FDD4 build.
function ref(relation: string, label: string): RefChip {
	return { referenceId: label, label, relation } as RefChip;
}

describe('ref-summary — compact event-chip ref counts', () => {
	it('ref-summary-counts-by-relation', () => {
		const s = refSummary([
			ref('attendee', 'Somchai Jaidee'),
			ref('attendee', 'Modal Tester'),
			ref('about', 'Freehold Hillside')
		]);
		expect(s.attendees).toBe(2);
		expect(s.properties).toBe(1);
	});

	it('is empty for no refs', () => {
		const s = refSummary([]);
		expect(s.attendees).toBe(0);
		expect(s.properties).toBe(0);
	});
});
