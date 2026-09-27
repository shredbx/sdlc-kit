import { describe, it, expect } from 'vitest';
import type { ContentSidebarSection } from './ContentSidebar.svelte';
import { resolveDrillDown } from './resolve-drilldown';

const NAV: ContentSidebarSection[] = [
	{ id: 'dashboard', label: 'Dashboard', href: '/admin' },
	{
		id: 'properties',
		label: 'Properties',
		href: '/manage/properties',
		children: [
			{ id: 'overview', label: 'Overview', href: '/manage/properties' },
			{ id: 'list', label: 'Listings', href: '/manage/properties/list' }
		]
	},
	{ id: 'users', label: 'Users', href: '/admin/users' }
];

describe('resolveDrillDown', () => {
	it('returns no drill when pathname is root', () => {
		const result = resolveDrillDown(NAV, '/');
		expect(result.drilledInto).toBeNull();
		expect(result.parentLabel).toBeNull();
		expect(result.parentHref).toBeNull();
	});

	it('does not drill into a section without children/items', () => {
		const result = resolveDrillDown(NAV, '/admin');
		expect(result.drilledInto).toBeNull();
	});

	it('drills into Properties when pathname equals section href', () => {
		const result = resolveDrillDown(NAV, '/manage/properties');
		expect(result.drilledInto?.id).toBe('properties');
		expect(result.parentLabel).toBe('Dashboard');
		expect(result.parentHref).toBe('/');
	});

	it('drills into Properties when pathname is a subpath', () => {
		const result = resolveDrillDown(NAV, '/manage/properties/list');
		expect(result.drilledInto?.id).toBe('properties');
	});

	it('drills into Properties for deep subpaths', () => {
		const result = resolveDrillDown(NAV, '/manage/properties/list/abc-123/edit');
		expect(result.drilledInto?.id).toBe('properties');
	});

	it('respects custom rootLabel + rootHref', () => {
		const result = resolveDrillDown(NAV, '/manage/properties', {
			rootLabel: 'Main',
			rootHref: '/home'
		});
		expect(result.parentLabel).toBe('Main');
		expect(result.parentHref).toBe('/home');
	});

	it('skips sections without href', () => {
		const tree: ContentSidebarSection[] = [
			{ id: 'a', label: 'A', children: [{ id: 'b', label: 'B', href: '/b' }] }
		];
		const result = resolveDrillDown(tree, '/a');
		expect(result.drilledInto).toBeNull();
	});

	it('considers sections with items only (no children)', () => {
		const tree: ContentSidebarSection[] = [
			{
				id: 'a',
				label: 'A',
				href: '/a',
				items: [{ id: 'b', label: 'B', href: '/a/b' }]
			}
		];
		const result = resolveDrillDown(tree, '/a');
		expect(result.drilledInto?.id).toBe('a');
	});

	it('does NOT match an unrelated pathname with a similar prefix', () => {
		const result = resolveDrillDown(NAV, '/manage/propertiesXYZ');
		expect(result.drilledInto).toBeNull();
	});

	it('matches when href is followed by a query string', () => {
		const result = resolveDrillDown(NAV, '/manage/properties?filter=draft');
		expect(result.drilledInto?.id).toBe('properties');
	});

	it('first matching section wins', () => {
		const tree: ContentSidebarSection[] = [
			{ id: 'a', label: 'A', href: '/x', children: [{ id: 'a1', label: 'A1', href: '/x/1' }] },
			{ id: 'b', label: 'B', href: '/x/1', children: [{ id: 'b1', label: 'B1', href: '/x/1/y' }] }
		];
		const result = resolveDrillDown(tree, '/x/1');
		expect(result.drilledInto?.id).toBe('a');
	});

	it('returns the same section reference (no clone)', () => {
		const result = resolveDrillDown(NAV, '/manage/properties');
		expect(result.drilledInto).toBe(NAV[1]);
	});
});
