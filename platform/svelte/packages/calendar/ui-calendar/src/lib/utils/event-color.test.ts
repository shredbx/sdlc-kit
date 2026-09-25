import { describe, expect, it } from 'vitest';
import { resolveEventStyle } from './event-color.js';
import type { CalendarItem } from '../types.js';

const base: CalendarItem = {
	id: 'e1',
	start: '2026-06-06T10:00:00+07:00',
	end: '2026-06-06T10:30:00+07:00',
	title: 'X',
	eventType: 'viewing',
	status: 'scheduled',
	refs: []
};

describe('resolveEventStyle', () => {
	it('returns the muted type style when the event type is in the map', () => {
		const style = resolveEventStyle(base, {
			viewing: { fill: 'var(--f)', text: 'var(--t)', accent: 'var(--a)' }
		});
		expect(style).toEqual({ fill: 'var(--f)', text: 'var(--t)', accent: 'var(--a)' });
	});

	it('per-event item.color (legacy solid fill) wins over the type map', () => {
		const style = resolveEventStyle(
			{ ...base, color: '#123456' },
			{ viewing: { fill: 'var(--f)', text: 'var(--t)' } }
		);
		expect(style).toEqual({ fill: '#123456', text: '#ffffff' });
	});

	it('is undefined when there is no override and the type is absent (status fallback)', () => {
		expect(resolveEventStyle(base, { meeting: { fill: 'a', text: 'b' } })).toBeUndefined();
		expect(resolveEventStyle(base, undefined)).toBeUndefined();
	});
});
