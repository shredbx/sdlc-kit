// Mock ContactProvider — for the package's own unit tests and isolated dev/storybook use
// (mirrors @sbx/ui-calendar's ./adapters/mock). Real consumers implement ContactProvider
// over their API; this never ships to a real surface.

import type {
	ContactCategory,
	ContactPage,
	ContactPickerRecord,
	ContactProvider
} from '../types';

const CATEGORIES: ContactCategory[] = [
	{ code: 'landlord', label: 'Landlord', count: 2 },
	{ code: 'buyer', label: 'Buyer', count: 2 },
	{ code: 'tenant', label: 'Tenant', count: 1 }
];

const ROWS: ContactPickerRecord[] = [
	{ id: '1', name: 'Somchai Jaidee', phone: '+66 81 234 5678', categoryCodes: ['landlord'], initials: 'SJ' },
	{ id: '2', name: 'Anong Pol', phone: '+66 82 345 6789', categoryCodes: ['landlord', 'buyer'], initials: 'AP' },
	{ id: '3', name: 'John Carter', phone: '+1 415 555 0102', categoryCodes: ['buyer'], initials: 'JC' },
	{ id: '4', name: 'Maria Silva', phone: '+55 11 99999 0000', categoryCodes: ['tenant'], initials: 'MS' }
];

/** A small in-memory provider with search + category browse (no real pagination). */
export function mockContactProvider(): ContactProvider {
	return {
		async list(query: string): Promise<ContactPage> {
			const q = query.trim().toLowerCase();
			const records = q ? ROWS.filter((r) => r.name.toLowerCase().includes(q)) : ROWS;
			return { records, nextCursor: null };
		},
		async categories(): Promise<ContactCategory[]> {
			return CATEGORIES;
		},
		async listByCategory(code: string): Promise<ContactPage> {
			return { records: ROWS.filter((r) => r.categoryCodes.includes(code)), nextCursor: null };
		},
		categoryLabel(code: string): string {
			return CATEGORIES.find((c) => c.code === code)?.label ?? code;
		}
	};
}
