// @vitest-environment jsdom
//
// MoneyInput leading currency-prefix contract (TC1 / SC2). Mounts the real
// component (Svelte 5 mount/flushSync, no @testing-library) and asserts the
// OBSERVABLE contract: the default keeps the trailing-currency display
// (backward compat), and `currencyPosition="leading"` renders a persistent
// currency adornment BEFORE the field while dropping the trailing currency
// from the field value (so it never reads "THB 15,000,000 THB"). The teal/box
// styling is CSS — covered by the Playwright visual pass, not jsdom.

import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mount, unmount, flushSync } from 'svelte';
import MoneyInput from './MoneyInput.svelte';

let target: HTMLDivElement;
let component: ReturnType<typeof mount> | null = null;

function field(): HTMLInputElement {
	return target.querySelector('input.money-input__field') as HTMLInputElement;
}
function prefix(): HTMLElement | null {
	return target.querySelector('.money-input__prefix');
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

describe('MoneyInput currency position', () => {
	it('defaults to trailing currency in the field value (backward compat)', () => {
		component = mount(MoneyInput, { target, props: { value: '15000000', currency: 'THB' } });
		flushSync();
		// Blur-state default display keeps the trailing code, no leading adornment.
		expect(field().value).toBe('15,000,000 THB');
		expect(prefix()).toBeNull();
	});

	it('renders a leading currency adornment when currencyPosition="leading"', () => {
		component = mount(MoneyInput, {
			target,
			props: { value: '15000000', currency: 'THB', currencyPosition: 'leading' }
		});
		flushSync();
		// The currency sits BEFORE the field as its own element…
		expect(prefix()).not.toBeNull();
		expect(prefix()?.textContent?.trim()).toBe('THB');
		// …and is NOT duplicated as a trailing code inside the field value.
		expect(field().value).toBe('15,000,000');
		expect(field().value).not.toContain('THB');
	});

	it('keeps the leading adornment visible even when empty', () => {
		component = mount(MoneyInput, {
			target,
			props: { value: '', currency: 'THB', currencyPosition: 'leading' }
		});
		flushSync();
		expect(prefix()?.textContent?.trim()).toBe('THB');
		expect(field().value).toBe('');
	});

	it('preserves a trailing suffix (e.g. " / mo") in leading mode without the currency code', () => {
		component = mount(MoneyInput, {
			target,
			props: { value: '45000', currency: 'THB', suffix: ' / mo', currencyPosition: 'leading' }
		});
		flushSync();
		expect(prefix()?.textContent?.trim()).toBe('THB');
		expect(field().value).toBe('45,000 / mo');
	});
});
