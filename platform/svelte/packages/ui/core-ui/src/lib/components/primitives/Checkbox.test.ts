// @vitest-environment jsdom
//
// Checkbox primitive contract (TC6 / SC1). Mounts the real component
// (Svelte 5 mount/flushSync, no @testing-library) and asserts the OBSERVABLE
// contract consumers depend on — a native <input type="checkbox"> for a11y +
// state, a rendered label, click toggle, checked reflection, and disabled
// blocking. (The teal fill + white tick are CSS — verified by the Playwright
// visual pass + screenshot-compare, not in jsdom.)

import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mount, unmount, flushSync } from 'svelte';
import Checkbox from './Checkbox.svelte';

let target: HTMLDivElement;
let component: ReturnType<typeof mount> | null = null;

function input(): HTMLInputElement {
	return target.querySelector('input[type="checkbox"]') as HTMLInputElement;
}

beforeEach(() => {
	target = document.createElement('div');
	document.body.appendChild(target);
});

afterEach(() => {
	if (component) {
		unmount(component);
		component = null;
	}
	target.remove();
});

describe('Checkbox', () => {
	it('renders the label text and a native checkbox input', () => {
		component = mount(Checkbox, { target, props: { label: 'Sea View' } });
		flushSync();
		expect(target.textContent).toContain('Sea View');
		expect(input()).toBeTruthy();
		expect(input().checked).toBe(false);
	});

	it('reflects the checked prop', () => {
		component = mount(Checkbox, { target, props: { label: 'Pool', checked: true } });
		flushSync();
		expect(input().checked).toBe(true);
	});

	it('toggles the native control on click', () => {
		component = mount(Checkbox, { target, props: { label: 'Pool', checked: false } });
		flushSync();
		input().click();
		flushSync();
		expect(input().checked).toBe(true);
	});

	it('does not toggle when disabled', () => {
		component = mount(Checkbox, { target, props: { label: 'Locked', checked: false, disabled: true } });
		flushSync();
		expect(input().disabled).toBe(true);
		input().click();
		flushSync();
		expect(input().checked).toBe(false);
	});

	it('does not clip a long label (no truncation styling)', () => {
		component = mount(Checkbox, {
			target,
			props: { label: 'A very long amenity label that must wrap and never be truncated or clipped' }
		});
		flushSync();
		expect(target.querySelector('.checkbox')).not.toBeNull();
		const html = target.innerHTML;
		expect(html).not.toContain('text-overflow: ellipsis');
		expect(html).not.toContain('line-clamp');
	});
});
