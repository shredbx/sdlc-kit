import { describe, it, expect } from 'vitest';
import { buildBreadcrumb } from './build-breadcrumb';

const ROUTE_MAP: Record<string, string> = {
	'/admin': 'Dashboard',
	'/manage': 'Manage',
	'/manage/properties': 'Properties',
	'/manage/properties/list': 'Listings'
};

describe('buildBreadcrumb', () => {
	it('returns empty array for root pathname when no rootLabel set', () => {
		expect(buildBreadcrumb('/', { routeMap: {} })).toEqual([]);
	});

	it('returns just rootLabel for root pathname when set', () => {
		const items = buildBreadcrumb('/', { routeMap: {}, rootLabel: 'Home', rootHref: '/' });
		expect(items).toEqual([{ label: 'Home', href: '/' }]);
	});

	it('emits a single crumb for /admin (current page has no href)', () => {
		const items = buildBreadcrumb('/admin', { routeMap: ROUTE_MAP });
		expect(items).toEqual([{ label: 'Dashboard', href: undefined }]);
	});

	it('builds multi-segment breadcrumb with intermediate hrefs', () => {
		const items = buildBreadcrumb('/manage/properties/list', { routeMap: ROUTE_MAP });
		expect(items).toEqual([
			{ label: 'Manage', href: '/manage' },
			{ label: 'Properties', href: '/manage/properties' },
			{ label: 'Listings', href: undefined }
		]);
	});

	it('uses override for a specific path', () => {
		const items = buildBreadcrumb('/manage/properties/list/abc-123', {
			routeMap: ROUTE_MAP,
			overrides: { '/manage/properties/list/abc-123': 'ABC Apartments' }
		});
		const last = items[items.length - 1];
		expect(last.label).toBe('ABC Apartments');
		expect(last.href).toBeUndefined();
	});

	it('overrides take precedence over routeMap', () => {
		const items = buildBreadcrumb('/manage/properties', {
			routeMap: ROUTE_MAP,
			overrides: { '/manage/properties': 'My Properties' }
		});
		const last = items[items.length - 1];
		expect(last.label).toBe('My Properties');
	});

	it('falls back to humanized last segment when no map entry', () => {
		const items = buildBreadcrumb('/manage/unknown-section', { routeMap: ROUTE_MAP });
		expect(items[1].label).toBe('Unknown Section');
	});

	it('does NOT link an intermediate path that is missing from routeMap', () => {
		// /manage is in ROUTE_MAP → linkable
		// Build a fresh map without /manage to simulate "route does not exist".
		const ROUTE_MAP_NO_MANAGE: Record<string, string> = {
			'/manage/properties': 'Properties',
			'/manage/properties/list': 'Listings'
		};
		const items = buildBreadcrumb('/manage/properties/list', { routeMap: ROUTE_MAP_NO_MANAGE });
		// "Manage" still appears (humanized fallback) but with no href.
		expect(items[0]).toEqual({ label: 'Manage', href: undefined });
		expect(items[1]).toEqual({ label: 'Properties', href: '/manage/properties' });
		expect(items[2]).toEqual({ label: 'Listings', href: undefined });
	});

	it('overrides make an intermediate path linkable', () => {
		const items = buildBreadcrumb('/manage/properties', {
			routeMap: {},
			overrides: { '/manage': 'Manage Hub' }
		});
		// /manage is linkable via overrides (not routeMap); /manage/properties is last → no href.
		expect(items[0]).toEqual({ label: 'Manage Hub', href: '/manage' });
		expect(items[1].href).toBeUndefined();
	});

	it('hrefOverrides remaps a crumb to a different target (and makes it linkable)', () => {
		// The property-detail case: /manage/properties has NO handler (only [id]/new),
		// so it is normally inert. An hrefOverride points the "Properties" parent crumb
		// at the real hub (/manage/content/properties) instead of its dead URL parent.
		const items = buildBreadcrumb('/manage/properties/abc-123/dashboard', {
			routeMap: {},
			overrides: { '/manage/properties/abc-123': 'Test Villa' },
			hrefOverrides: { '/manage/properties': '/manage/content/properties' }
		});
		// Parent crumb links to the OVERRIDE target, not its own cumulative path.
		expect(items[1]).toEqual({ label: 'Properties', href: '/manage/content/properties' });
		// The override is additive — the entity crumb (overrides) and the leaf are unchanged.
		expect(items[2].label).toBe('Test Villa');
		expect(items.at(-1)?.href).toBeUndefined();
	});

	it('hrefOverrides never overrides the last (current-page) crumb', () => {
		const items = buildBreadcrumb('/manage/properties', {
			routeMap: {},
			hrefOverrides: { '/manage/properties': '/manage/content/properties' }
		});
		// /manage/properties is the current page → stays href-less despite the override.
		expect(items.at(-1)).toEqual({ label: 'Properties', href: undefined });
	});

	it('humanizes single-word segments to title case', () => {
		const items = buildBreadcrumb('/foo', { routeMap: {} });
		expect(items[0].label).toBe('Foo');
	});

	it('handles multi-word kebab segments', () => {
		const items = buildBreadcrumb('/some-long-path', { routeMap: {} });
		expect(items[0].label).toBe('Some Long Path');
	});

	it('prepends rootLabel when configured', () => {
		const items = buildBreadcrumb('/manage', {
			routeMap: ROUTE_MAP,
			rootLabel: 'Home',
			rootHref: '/'
		});
		expect(items[0]).toEqual({ label: 'Home', href: '/' });
		expect(items[1].label).toBe('Manage');
		expect(items[1].href).toBeUndefined();
	});

	it('last item has undefined href, others have hrefs', () => {
		const items = buildBreadcrumb('/manage/properties/list', { routeMap: ROUTE_MAP });
		expect(items[0].href).toBe('/manage');
		expect(items[1].href).toBe('/manage/properties');
		expect(items[2].href).toBeUndefined();
	});

	it('empty pathname is treated as root', () => {
		expect(buildBreadcrumb('', { routeMap: {} })).toEqual([]);
	});

	it('strips empty path components from leading/double slashes', () => {
		const items = buildBreadcrumb('//manage//properties', { routeMap: ROUTE_MAP });
		expect(items).toEqual([
			{ label: 'Manage', href: '/manage' },
			{ label: 'Properties', href: undefined }
		]);
	});
});
