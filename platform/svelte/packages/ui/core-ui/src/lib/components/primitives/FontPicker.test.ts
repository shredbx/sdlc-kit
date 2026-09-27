// @vitest-environment jsdom
//
// FontPicker component test. Mounts the real component (Svelte 5 mount/flushSync,
// no @testing-library) and asserts the contract the canvas integration depends on:
//   - the trigger shows `value` and opens a listbox on click (aria-expanded);
//   - Brand fonts render first, in their own face (inline font-family);
//   - typing in the search filters Brand + All (case-insensitive includes);
//   - the All list is WINDOWED — only the first ~40 of a large catalogue render,
//     and "Show more" reveals the next slice;
//   - an empty catalogue shows the quiet "Catalogue unavailable" note (no crash);
//   - selecting a row fires `onselect` with the family and persists it to Recent.
//
// `../../fonts` is the REAL loader — safe in jsdom: it guards on `document.fonts`
// (absent here) and resolves false, injecting nothing. IntersectionObserver is
// also absent → the row preview action takes its eager-load fallback (a no-op
// ensureGoogleFont). No mocks needed.

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { mount, unmount, flushSync } from 'svelte';
import FontPicker from './FontPicker.svelte';

let target: HTMLDivElement;
let component: ReturnType<typeof mount> | null = null;

function trigger(): HTMLButtonElement {
	return target.querySelector('.font-picker__trigger') as HTMLButtonElement;
}
function options(): HTMLButtonElement[] {
	return Array.from(target.querySelectorAll('.font-picker__option')) as HTMLButtonElement[];
}
function groups(): string[] {
	return Array.from(target.querySelectorAll('.font-picker__group')).map(
		(n) => n.textContent?.trim() ?? ''
	);
}

beforeEach(() => {
	target = document.createElement('div');
	document.body.appendChild(target);
	localStorage.clear();
});

afterEach(() => {
	if (component) {
		unmount(component);
		component = null;
	}
	target.remove();
});

const BRAND = [{ family: 'Inter', label: 'Inter (Brand)' }, { family: 'Georgia' }];

describe('FontPicker — trigger', () => {
	it('shows the current value and is collapsed by default', () => {
		component = mount(FontPicker, {
			target,
			props: { value: 'Inter', brandFonts: BRAND, catalogue: [], onselect: () => {} }
		});
		flushSync();
		expect(trigger().getAttribute('aria-expanded')).toBe('false');
		expect(trigger().getAttribute('aria-haspopup')).toBe('listbox');
		expect(target.querySelector('.font-picker__value')?.textContent?.trim()).toBe('Inter');
		expect(target.querySelector('[role="listbox"]')).toBeNull();
	});

	it('opens a listbox on click', async () => {
		component = mount(FontPicker, {
			target,
			props: { value: 'Inter', brandFonts: BRAND, catalogue: [], onselect: () => {} }
		});
		flushSync();
		trigger().click();
		flushSync();
		expect(trigger().getAttribute('aria-expanded')).toBe('true');
		expect(target.querySelector('[role="listbox"]')).not.toBeNull();
	});
});

describe('FontPicker — brand + preview face', () => {
	it('renders brand fonts first, each in its own font-family', () => {
		component = mount(FontPicker, {
			target,
			props: { value: 'Inter', brandFonts: BRAND, catalogue: [], onselect: () => {} }
		});
		flushSync();
		trigger().click();
		flushSync();

		expect(groups()[0]).toBe('Brand');
		const names = Array.from(target.querySelectorAll('.font-picker__name')) as HTMLElement[];
		// Brand label override is honoured, and the face is the family's own.
		const inter = names.find((n) => n.textContent?.includes('Inter (Brand)'));
		expect(inter).toBeTruthy();
		// The browser normalises the quote style ('Inter' → "Inter"); assert the
		// family is set in the row's own face, quote-agnostically.
		expect(inter?.getAttribute('style')?.replace(/["']/g, '')).toContain('font-family: Inter');
	});
});

describe('FontPicker — search filtering', () => {
	const CATALOGUE = [
		{ family: 'Roboto', category: 'sans-serif' },
		{ family: 'Lobster', category: 'display' },
		{ family: 'Lato', category: 'sans-serif' }
	];

	it('filters Brand + All by a case-insensitive substring', async () => {
		component = mount(FontPicker, {
			target,
			props: { value: 'Inter', brandFonts: BRAND, catalogue: CATALOGUE, onselect: () => {} }
		});
		flushSync();
		trigger().click();
		flushSync();

		const search = target.querySelector('input[name="font-search"]') as HTMLInputElement;
		search.value = 'lat';
		search.dispatchEvent(new Event('input', { bubbles: true }));
		flushSync();

		const labels = options().map((o) => o.textContent?.trim() ?? '');
		// 'Lato' matches; 'Roboto'/'Lobster' do not; brand 'Inter'/'Georgia' drop out.
		expect(labels.some((l) => l.includes('Lato'))).toBe(true);
		expect(labels.some((l) => l.includes('Roboto'))).toBe(false);
		expect(labels.some((l) => l.includes('Inter'))).toBe(false);
	});

	it('shows a quiet empty note when nothing matches', async () => {
		component = mount(FontPicker, {
			target,
			props: { value: 'Inter', brandFonts: [], catalogue: CATALOGUE, onselect: () => {} }
		});
		flushSync();
		trigger().click();
		flushSync();

		const search = target.querySelector('input[name="font-search"]') as HTMLInputElement;
		search.value = 'zzzznomatch';
		search.dispatchEvent(new Event('input', { bubbles: true }));
		flushSync();

		expect(target.querySelector('.font-picker__empty')?.textContent).toContain('No fonts match');
	});
});

describe('FontPicker — windowing', () => {
	const BIG = Array.from({ length: 120 }, (_, i) => ({ family: `Font ${i}` }));

	it('renders only the first window and reveals more on "Show more"', () => {
		component = mount(FontPicker, {
			target,
			props: { value: '', brandFonts: [], catalogue: BIG, onselect: () => {} }
		});
		flushSync();
		trigger().click();
		flushSync();

		// 40 windowed rows (no brand/recent here).
		expect(options().length).toBe(40);
		const more = target.querySelector('.font-picker__more') as HTMLButtonElement;
		expect(more).not.toBeNull();

		more.click();
		flushSync();
		expect(options().length).toBe(80);
	});
});

describe('FontPicker — empty catalogue', () => {
	it('still shows Brand and a "Catalogue unavailable" note (no crash)', () => {
		component = mount(FontPicker, {
			target,
			props: { value: 'Inter', brandFonts: BRAND, catalogue: [], onselect: () => {} }
		});
		flushSync();
		trigger().click();
		flushSync();

		expect(groups()).toContain('Brand');
		expect(groups()).toContain('All fonts');
		expect(target.querySelector('.font-picker__empty')?.textContent).toContain(
			'Catalogue unavailable'
		);
	});
});

describe('FontPicker — selection', () => {
	it('fires onselect with the family, closes, and persists to Recent', async () => {
		const onselect = vi.fn();
		component = mount(FontPicker, {
			target,
			props: {
				value: 'Inter',
				brandFonts: BRAND,
				catalogue: [{ family: 'Roboto' }],
				onselect
			}
		});
		flushSync();
		trigger().click();
		flushSync();

		const roboto = options().find((o) => o.textContent?.includes('Roboto')) as HTMLButtonElement;
		roboto.click();
		// choose() awaits ensureGoogleFont (a resolved promise in jsdom) before
		// firing onselect/closing — let the microtask drain.
		await Promise.resolve();
		await Promise.resolve();
		flushSync();

		expect(onselect).toHaveBeenCalledWith('Roboto');
		expect(target.querySelector('[role="listbox"]')).toBeNull(); // closed
		expect(JSON.parse(localStorage.getItem('sbx.fonts.recent.v1') ?? '[]')).toContain('Roboto');
	});
});
