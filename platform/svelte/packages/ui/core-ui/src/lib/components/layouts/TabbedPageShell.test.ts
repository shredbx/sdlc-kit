// @vitest-environment jsdom
//
// TabbedPageShell component test (#0290 F1b). THE ONE admin shell — the merge of the
// old AdminPage + AdminPageShell + AdminScaffoldPage tests (LISTING/SCAFFOLD mode) PLUS
// the folded-in EntityDetailShell test coverage (DETAIL mode, F1b-ii). Asserts the full
// contract:
//   LISTING / SCAFFOLD —
//   - heading band (back? → title via PageTitle → subtitle?) renders by default;
//   - heading='none' drops the WHOLE band so the page owns its own <h1>, while the
//     tab strip + body still render (the old AdminPageShell headless behaviour);
//   - titleScale='landing' renders the larger primary scaffold title;
//   - the route-driven tab strip appears only with ≥2 tabs ("no fake tabs"), marks
//     the active tab aria-current for the mocked pathname, and renders a count pill
//     only for badge > 0;
//   - `actions` is FORWARDED into the topbar store on afterNavigate and cleared on
//     beforeNavigate when the pathname changes;
//   DETAIL (surfaceRoot set, folded in from EntityDetailShell) —
//   - heading='sr-only' renders a visually-hidden <h1> (in the DOM, off-screen) and no
//     visual band;
//   - cover + status register into the topbar TOOLBAR slot (beside the breadcrumb) on
//     mount, alongside the actions slot;
//   - the topbar slots PERSIST across a same-surfaceRoot facet navigation and CLEAR only
//     when navigating OUT of surfaceRoot;
//   - the facet tab strip (≥2) marks the active facet via the same activeTabId rule;
//   SHARED —
//   - the exported activeTabId rule (exact match + longest boundary-safe prefix) — now
//     the single (tabs, pathname) order for both modes.
//
// Svelte 5 mount/unmount/flushSync, no @testing-library. Three module deps are mocked so
// the component mounts in isolation: `$app/state`'s `page` (read for the active-tab
// pathname), `$app/navigation`'s afterNavigate/beforeNavigate (captured to drive the
// register/clear steps), and the topbar-slots store (so useTopbarSlots resolves without a
// parent layout). `svelte`'s onMount is the REAL one (un-mocked) — Svelte 5 fires it
// synchronously during mount(), so the detail-mode registration lands right after
// flushSync with no extra step.

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { mount, unmount, flushSync, type Snippet } from 'svelte';

// Mutable mock of $app/state's `page` — the active-tab logic reads page.url.pathname.
const { mockPage } = vi.hoisted(() => ({ mockPage: { url: { pathname: '/' } } }));
vi.mock('$app/state', () => ({ page: mockPage }));

// Capture the afterNavigate/beforeNavigate callbacks so the test can invoke the
// register step and assert what was written into the topbar store.
const { afterCbs, beforeCbs } = vi.hoisted(() => ({
	afterCbs: [] as Array<() => void>,
	beforeCbs: [] as Array<(nav: unknown) => void>
}));
vi.mock('$app/navigation', () => ({
	afterNavigate: (cb: () => void) => afterCbs.push(cb),
	beforeNavigate: (cb: (nav: unknown) => void) => beforeCbs.push(cb)
}));

// A standalone writable-like store so useTopbarSlots resolves without a shell layout.
const { topbarStore, topbarState, topbarSet } = vi.hoisted(() => {
	const state: { value: { toolbar: unknown; actions: unknown } } = {
		value: { toolbar: undefined, actions: undefined }
	};
	const set = vi.fn((v: { toolbar: unknown; actions: unknown }) => {
		state.value = v;
	});
	return {
		topbarState: state,
		topbarSet: set,
		topbarStore: {
			set,
			subscribe: (run: (v: unknown) => void) => {
				run(state.value);
				return () => {};
			}
		}
	};
});
vi.mock('@sbx/core-ui/stores/topbarSlots', () => ({ useTopbarSlots: () => topbarStore }));

// Imported AFTER the mocks are registered.
import TabbedPageShell, { activeTabId, type TabEntry } from './TabbedPageShell.svelte';

let target: HTMLDivElement;
let component: ReturnType<typeof mount> | null = null;

beforeEach(() => {
	target = document.createElement('div');
	document.body.appendChild(target);
	mockPage.url.pathname = '/';
	afterCbs.length = 0;
	beforeCbs.length = 0;
	topbarState.value = { toolbar: undefined, actions: undefined };
	topbarSet.mockClear();
});

afterEach(() => {
	if (component) {
		try {
			unmount(component);
		} catch {
			/* already torn down */
		}
		component = null;
	}
	if (target.parentNode) target.remove();
});

// Minimal body snippet — a single probe node so we can assert content renders.
const body = ((anchor: Node) => {
	const el = document.createElement('p');
	el.className = 'page-body-probe';
	el.textContent = 'content';
	anchor.parentNode?.insertBefore(el, anchor);
}) as unknown as Snippet;

const TWO_TABS: TabEntry[] = [
	{ id: 'listing', label: 'Listing', href: '/manage/inquiries' },
	{ id: 'page', label: 'Page', href: '/manage/inquiries/page' }
];

// DETAIL-mode facet fixtures (folded in from EntityDetailShell.test.ts). Property-shaped
// facets under one entity surface — used to assert the strip + active-facet rule and the
// surfaceRoot-scoped topbar persistence.
const FACETS: TabEntry[] = [
	{ id: 'information', label: 'Information', href: '/manage/properties/p1/information' },
	{ id: 'media', label: 'Media', href: '/manage/properties/p1/media' },
	{ id: 'activity', label: 'Activity', href: '/manage/properties/p1/activity' }
];
const PROPERTY_SURFACE = '/manage/properties/p1/';

function h1() {
	return target.querySelector('h1.page-title') as HTMLElement | null;
}
function srTitle() {
	return target.querySelector('h1.page-shell-sr-title') as HTMLElement | null;
}
function nav() {
	return target.querySelector('.page-shell-tabs') as HTMLElement | null;
}
function tabs() {
	return Array.from(target.querySelectorAll('.page-shell-tab')) as HTMLAnchorElement[];
}

describe('TabbedPageShell — heading band', () => {
	it('renders the title (via PageTitle) and subtitle by default', () => {
		component = mount(TabbedPageShell, {
			target,
			props: { title: 'Audit Log', subtitle: '128 events', children: body }
		});
		flushSync();
		expect(h1()?.textContent?.trim()).toBe('Audit Log');
		expect(target.querySelector('.page-shell-subtitle')?.textContent?.trim()).toBe('128 events');
		expect(target.querySelector('.page-body-probe')).toBeTruthy();
	});

	it('renders NO heading band when heading="none", but the body still renders', () => {
		component = mount(TabbedPageShell, {
			target,
			props: { title: 'Audit Log', subtitle: '128 events', heading: 'none', children: body }
		});
		flushSync();
		expect(h1()).toBeNull();
		expect(target.querySelector('.page-shell-header')).toBeNull();
		// Subtitle lives inside the header block, so it's gone too.
		expect(target.querySelector('.page-shell-subtitle')).toBeNull();
		expect(target.querySelector('.page-body-probe')).toBeTruthy();
	});

	it('keeps the tab strip + body when heading="none" with ≥2 tabs', () => {
		mockPage.url.pathname = '/manage/inquiries';
		component = mount(TabbedPageShell, {
			target,
			props: { title: 'Inquiries', heading: 'none', tabs: TWO_TABS, children: body }
		});
		flushSync();
		expect(target.querySelector('.page-shell-header')).toBeNull();
		expect(nav()).toBeTruthy();
		expect(target.querySelector('.page-body-probe')).toBeTruthy();
	});

	it('renders an sr-only <h1> (in the DOM, visually hidden) when heading="sr-only"', () => {
		// DETAIL pages (folded in from EntityDetailShell): no visual title band — the
		// breadcrumb carries identity — but exactly one accessible <h1> survives for
		// a11y/SEO. The visible PageTitle band is gone; the sr-only <h1> is present and
		// carries the standard visually-hidden clip (off-screen, not display:none).
		component = mount(TabbedPageShell, {
			target,
			props: { title: 'A property', heading: 'sr-only', children: body }
		});
		flushSync();
		expect(target.querySelector('.page-shell-header')).toBeNull();
		expect(h1()).toBeNull(); // no visible PageTitle band
		const sr = srTitle();
		expect(sr).toBeTruthy();
		// Exactly one accessible <h1>, carrying the title text and the visually-hidden
		// hook class (.page-shell-sr-title → clip-rect off-screen, NOT display:none, so
		// it stays in the a11y tree). Matches EntityDetailShell's sr-title assertion; the
		// component's scoped CSS isn't injected in the jsdom unit harness, so the contract
		// is the element + class + text (the clip rule itself is verified in source).
		expect(target.querySelectorAll('h1').length).toBe(1);
		expect(sr?.tagName).toBe('H1');
		expect(sr?.classList.contains('page-shell-sr-title')).toBe(true);
		expect(sr?.textContent?.trim()).toBe('A property');
		expect(target.querySelector('.page-body-probe')).toBeTruthy();
	});

	it('omits the subtitle element when no subtitle is passed', () => {
		component = mount(TabbedPageShell, { target, props: { title: 'Audit Log', children: body } });
		flushSync();
		expect(h1()?.textContent?.trim()).toBe('Audit Log');
		expect(target.querySelector('.page-shell-subtitle')).toBeNull();
	});

	it('renders an optional back link above the title', () => {
		component = mount(TabbedPageShell, {
			target,
			props: { title: 'Detail', back: { href: '/manage/news', label: 'Back to News' }, children: body }
		});
		flushSync();
		const back = target.querySelector('a.page-shell-back') as HTMLAnchorElement | null;
		expect(back?.getAttribute('href')).toBe('/manage/news');
		expect(back?.textContent?.replace('←', '').trim()).toBe('Back to News');
	});

	it('applies the landing title scale when titleScale="landing"', () => {
		component = mount(TabbedPageShell, {
			target,
			props: { title: 'Transactions', titleScale: 'landing', children: body }
		});
		flushSync();
		expect(h1()?.classList.contains('page-title--landing')).toBe(true);
	});

	it('uses the default (non-landing) title scale by default', () => {
		component = mount(TabbedPageShell, { target, props: { title: 'Audit Log', children: body } });
		flushSync();
		expect(h1()?.classList.contains('page-title--landing')).toBe(false);
	});
});

describe('TabbedPageShell — tab strip', () => {
	it('renders the nav with all tab labels for ≥2 tabs and marks the active one', () => {
		mockPage.url.pathname = '/manage/inquiries';
		component = mount(TabbedPageShell, {
			target,
			props: { title: 'Inquiries', tabs: TWO_TABS, children: body }
		});
		flushSync();
		expect(nav()).toBeTruthy();
		expect(tabs().map((a) => a.textContent?.trim())).toEqual(['Listing', 'Page']);
		expect(tabs()[0].getAttribute('aria-current')).toBe('page');
		expect(tabs()[0].classList.contains('page-shell-tab--active')).toBe(true);
	});

	it('marks the nested sub-route tab active (Page on /manage/inquiries/page)', () => {
		mockPage.url.pathname = '/manage/inquiries/page';
		component = mount(TabbedPageShell, {
			target,
			props: { title: 'Inquiries', tabs: TWO_TABS, children: body }
		});
		flushSync();
		const [listing, pageTab] = tabs();
		expect(pageTab.getAttribute('aria-current')).toBe('page');
		expect(listing.getAttribute('aria-current')).toBeNull();
	});

	it('renders NO tab strip for a single tab (the "no fake tabs" rule)', () => {
		mockPage.url.pathname = '/manage/inquiries';
		component = mount(TabbedPageShell, {
			target,
			props: { title: 'Inquiries', tabs: [TWO_TABS[0]], children: body }
		});
		flushSync();
		expect(nav()).toBeNull();
		expect(target.querySelector('.page-body-probe')).toBeTruthy();
	});

	it('renders NO tab strip when tabs is undefined', () => {
		component = mount(TabbedPageShell, { target, props: { title: 'Inquiries', children: body } });
		flushSync();
		expect(nav()).toBeNull();
	});
});

describe('TabbedPageShell — badge pill', () => {
	it('renders a count pill for badge > 0 with an accessible link label', () => {
		mockPage.url.pathname = '/manage/inquiries';
		const badged: TabEntry[] = [
			{ id: 'listing', label: 'Listing', href: '/manage/inquiries', badge: 3 },
			{ id: 'page', label: 'Page', href: '/manage/inquiries/page' }
		];
		component = mount(TabbedPageShell, {
			target,
			props: { title: 'Inquiries', tabs: badged, children: body }
		});
		flushSync();
		const pill = target.querySelector('.page-shell-badge') as HTMLElement | null;
		expect(pill?.textContent?.trim()).toBe('3');
		expect(tabs()[0].getAttribute('aria-label')).toBe('Listing (3)');
		expect(pill?.getAttribute('aria-hidden')).toBe('true');
	});

	it('renders no pill for badge: 0 or undefined', () => {
		mockPage.url.pathname = '/manage/inquiries';
		const badged: TabEntry[] = [
			{ id: 'listing', label: 'Listing', href: '/manage/inquiries', badge: 0 },
			{ id: 'page', label: 'Page', href: '/manage/inquiries/page' }
		];
		component = mount(TabbedPageShell, {
			target,
			props: { title: 'Inquiries', tabs: badged, children: body }
		});
		flushSync();
		expect(target.querySelector('.page-shell-badge')).toBeNull();
		expect(tabs()[0].getAttribute('aria-label')).toBeNull();
	});
});

describe('TabbedPageShell — topbar actions forwarding', () => {
	it('forwards the actions snippet into the topbar store on (after)navigate', () => {
		const actions = (() => {}) as unknown as Snippet;
		component = mount(TabbedPageShell, {
			target,
			props: { title: 'Audit Log', actions, children: body }
		});
		flushSync();

		// The component registered an afterNavigate callback; invoke it (SvelteKit
		// fires it on mount in the app). It must push the actions snippet — not a
		// self-rendered region — into the shared topbar store.
		expect(afterCbs.length).toBeGreaterThan(0);
		afterCbs.forEach((cb) => cb());
		expect(topbarState.value.actions).toBe(actions);
		expect(topbarState.value.toolbar).toBeUndefined();
	});

	it('clears the topbar slots on beforeNavigate when the pathname changes', () => {
		const actions = (() => {}) as unknown as Snippet;
		mockPage.url.pathname = '/admin/audit';
		component = mount(TabbedPageShell, {
			target,
			props: { title: 'Audit Log', actions, children: body }
		});
		flushSync();
		afterCbs.forEach((cb) => cb());
		expect(topbarState.value.actions).toBe(actions);

		// Leaving to a different route → slots cleared.
		beforeCbs.forEach((cb) => cb({ to: { url: { pathname: '/admin/users' } } }));
		expect(topbarState.value.actions).toBeUndefined();
		expect(topbarState.value.toolbar).toBeUndefined();
	});

	it('keeps the topbar slots populated on a same-route (query-only) navigation', () => {
		const actions = (() => {}) as unknown as Snippet;
		mockPage.url.pathname = '/admin/audit';
		component = mount(TabbedPageShell, {
			target,
			props: { title: 'Audit Log', actions, children: body }
		});
		flushSync();
		afterCbs.forEach((cb) => cb());
		expect(topbarState.value.actions).toBe(actions);

		// Same pathname (only the query changed) → the slot must NOT be cleared.
		beforeCbs.forEach((cb) => cb({ to: { url: { pathname: '/admin/audit' } } }));
		expect(topbarState.value.actions).toBe(actions);
	});
});

// ── DETAIL mode (surfaceRoot set) — folded in from EntityDetailShell.test.ts ─────────
// Detail mode keys off `surfaceRoot`. Registration is onMount (NOT afterNavigate), so it
// fires during mount() — topbarState reflects it right after flushSync, with NO
// afterCbs.forEach() needed. The cover + status snippet ride the TOOLBAR slot; actions
// ride the actions slot; both PERSIST across same-surfaceRoot facet navigation and CLEAR
// only when leaving surfaceRoot (the topbar-leak gotcha this shell must not regress).
describe('TabbedPageShell — detail mode (surfaceRoot)', () => {
	it('registers cover + status into the topbar TOOLBAR slot (plus actions) on mount', () => {
		const status = (() => {}) as unknown as Snippet;
		const actions = (() => {}) as unknown as Snippet;
		mockPage.url.pathname = '/manage/properties/p1/information';
		component = mount(TabbedPageShell, {
			target,
			props: {
				title: 'A property',
				heading: 'sr-only',
				surfaceRoot: PROPERTY_SURFACE,
				cover: { url: undefined, href: '/manage/properties/p1/media' },
				status,
				actions,
				children: body
			}
		});
		flushSync();
		// onMount already registered — a compact toolbar snippet (cover/status present)
		// rides the toolbar slot, the consumer's actions ride the actions slot. No
		// afterNavigate invocation needed (detail mode ignores afterNavigate).
		expect(typeof topbarState.value.toolbar).toBe('function');
		expect(topbarState.value.actions).toBe(actions);
		// Double-registration guard (carried from EntityDetailShell.test.ts): detail mode
		// registers EXACTLY once on mount; afterNavigate must never re-register.
		expect(topbarSet).toHaveBeenCalledTimes(1);
	});

	it('registers an undefined toolbar when neither cover nor status is passed', () => {
		const actions = (() => {}) as unknown as Snippet;
		mockPage.url.pathname = '/manage/properties/p1/information';
		component = mount(TabbedPageShell, {
			target,
			props: {
				title: 'A property',
				heading: 'sr-only',
				surfaceRoot: PROPERTY_SURFACE,
				actions,
				children: body
			}
		});
		flushSync();
		// Compact-only contract: onMount always registers; with no cover/status the
		// toolbar slot is undefined (visual no-op) while actions still ride along.
		expect(topbarState.value.toolbar).toBeUndefined();
		expect(topbarState.value.actions).toBe(actions);
	});

	it('PERSISTS the topbar slots across a same-surfaceRoot facet navigation', () => {
		const status = (() => {}) as unknown as Snippet;
		const actions = (() => {}) as unknown as Snippet;
		mockPage.url.pathname = '/manage/properties/p1/information';
		component = mount(TabbedPageShell, {
			target,
			props: {
				title: 'A property',
				heading: 'sr-only',
				surfaceRoot: PROPERTY_SURFACE,
				status,
				actions,
				children: body
			}
		});
		flushSync();
		const registeredToolbar = topbarState.value.toolbar;
		expect(typeof registeredToolbar).toBe('function');
		expect(topbarState.value.actions).toBe(actions);

		// Navigate Information → Media (still under surfaceRoot) → slots must NOT clear.
		beforeCbs.forEach((cb) => cb({ to: { url: { pathname: '/manage/properties/p1/media' } } }));
		expect(topbarState.value.toolbar).toBe(registeredToolbar);
		expect(topbarState.value.actions).toBe(actions);
	});

	it('CLEARS the topbar slots when navigating OUT of surfaceRoot', () => {
		const status = (() => {}) as unknown as Snippet;
		const actions = (() => {}) as unknown as Snippet;
		mockPage.url.pathname = '/manage/properties/p1/information';
		component = mount(TabbedPageShell, {
			target,
			props: {
				title: 'A property',
				heading: 'sr-only',
				surfaceRoot: PROPERTY_SURFACE,
				status,
				actions,
				children: body
			}
		});
		flushSync();
		expect(typeof topbarState.value.toolbar).toBe('function');

		// Leaving the property surface (a sibling list) → both slots cleared.
		beforeCbs.forEach((cb) => cb({ to: { url: { pathname: '/manage/properties' } } }));
		expect(topbarState.value.toolbar).toBeUndefined();
		expect(topbarState.value.actions).toBeUndefined();
	});

	it('renders the facet strip (≥2) and marks the active facet via longest prefix', () => {
		mockPage.url.pathname = '/manage/properties/p1/media';
		component = mount(TabbedPageShell, {
			target,
			props: {
				title: 'A property',
				heading: 'sr-only',
				surfaceRoot: PROPERTY_SURFACE,
				tabs: FACETS,
				children: body
			}
		});
		flushSync();
		expect(nav()).toBeTruthy();
		expect(tabs().map((a) => a.textContent?.trim())).toEqual(['Information', 'Media', 'Activity']);
		const media = tabs().find((a) => a.textContent?.trim() === 'Media');
		expect(media?.getAttribute('aria-current')).toBe('page');
		expect(media?.classList.contains('page-shell-tab--active')).toBe(true);
		// The detail body carries the section-stacking gap modifier.
		expect(target.querySelector('.page-shell--detail')).toBeTruthy();
	});

	it('renders NO facet strip for a single facet (the "no fake tabs" rule)', () => {
		mockPage.url.pathname = '/manage/properties/p1/information';
		component = mount(TabbedPageShell, {
			target,
			props: {
				title: 'A property',
				heading: 'sr-only',
				surfaceRoot: PROPERTY_SURFACE,
				tabs: [FACETS[0]],
				children: body
			}
		});
		flushSync();
		expect(nav()).toBeNull();
		expect(target.querySelector('.page-body-probe')).toBeTruthy();
	});
});

describe('activeTabId — pure rule', () => {
	const T: TabEntry[] = [
		{ id: 'listing', label: 'Listing', href: '/manage/inquiries' },
		{ id: 'page', label: 'Page', href: '/manage/inquiries/page' }
	];

	it('returns the exact match', () => {
		expect(activeTabId(T, '/manage/inquiries')).toBe('listing');
		expect(activeTabId(T, '/manage/inquiries/page')).toBe('page');
	});

	it('picks the longest boundary-safe prefix for deeper paths', () => {
		// Deeper under /page → still Page (longest prefix).
		expect(activeTabId(T, '/manage/inquiries/page/edit')).toBe('page');
		// Deeper under listing but not /page → Listing.
		expect(activeTabId(T, '/manage/inquiries/123')).toBe('listing');
	});

	it('does NOT match a non-boundary prefix', () => {
		// /manage/inquiriesX must not match /manage/inquiries.
		expect(activeTabId(T, '/manage/inquiriesX')).toBeUndefined();
	});

	it('returns undefined when nothing matches', () => {
		expect(activeTabId(T, '/manage/properties')).toBeUndefined();
	});
});
