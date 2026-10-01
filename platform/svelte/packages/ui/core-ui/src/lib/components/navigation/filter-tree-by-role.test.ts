import { describe, it, expect } from 'vitest';
import type { ContentSidebarSection } from './ContentSidebar.svelte';
import { filterTreeByRole } from './filter-tree-by-role';

describe('filterTreeByRole', () => {
	it('keeps sections with no roles field', () => {
		const tree: ContentSidebarSection[] = [{ id: 'a', label: 'A' }];
		expect(filterTreeByRole(tree, 'agent')).toEqual([{ id: 'a', label: 'A' }]);
	});

	it('keeps sections with empty roles array', () => {
		const tree: ContentSidebarSection[] = [{ id: 'a', label: 'A', roles: [] }];
		const filtered = filterTreeByRole(tree, 'agent');
		expect(filtered).toHaveLength(1);
		expect(filtered[0].id).toBe('a');
	});

	it('keeps sections when user role is in the roles list', () => {
		const tree: ContentSidebarSection[] = [
			{ id: 'a', label: 'A', roles: ['super-admin', 'manager'] }
		];
		expect(filterTreeByRole(tree, 'manager')).toHaveLength(1);
	});

	it('drops sections when user role is not in the roles list', () => {
		const tree: ContentSidebarSection[] = [{ id: 'a', label: 'A', roles: ['super-admin'] }];
		expect(filterTreeByRole(tree, 'agent')).toEqual([]);
	});

	it('recursively filters children', () => {
		const tree: ContentSidebarSection[] = [
			{
				id: 'a',
				label: 'A',
				children: [
					{ id: 'a1', label: 'A1' },
					{ id: 'a2', label: 'A2', roles: ['super-admin'] }
				]
			}
		];
		const filtered = filterTreeByRole(tree, 'agent');
		expect(filtered).toHaveLength(1);
		expect(filtered[0].children).toHaveLength(1);
		expect(filtered[0].children?.[0].id).toBe('a1');
	});

	it('preserves order', () => {
		const tree: ContentSidebarSection[] = [
			{ id: 'a', label: 'A' },
			{ id: 'b', label: 'B', roles: ['super-admin'] },
			{ id: 'c', label: 'C' }
		];
		const filtered = filterTreeByRole(tree, 'agent');
		expect(filtered.map((s) => s.id)).toEqual(['a', 'c']);
	});

	it('does not mutate the original tree', () => {
		const tree: ContentSidebarSection[] = [
			{
				id: 'a',
				label: 'A',
				children: [{ id: 'a1', label: 'A1', roles: ['super-admin'] }]
			}
		];
		const snapshot = JSON.parse(JSON.stringify(tree));
		filterTreeByRole(tree, 'agent');
		expect(tree).toEqual(snapshot);
	});

	it('handles deeply nested role gates', () => {
		const tree: ContentSidebarSection[] = [
			{
				id: 'a',
				label: 'A',
				children: [
					{
						id: 'a1',
						label: 'A1',
						children: [
							{ id: 'a1a', label: 'A1A', roles: ['super-admin'] },
							{ id: 'a1b', label: 'A1B' }
						]
					}
				]
			}
		];
		const filtered = filterTreeByRole(tree, 'agent');
		expect(filtered[0].children?.[0].children).toHaveLength(1);
		expect(filtered[0].children?.[0].children?.[0].id).toBe('a1b');
	});

	it('keeps undefined children as undefined', () => {
		const tree: ContentSidebarSection[] = [{ id: 'a', label: 'A' }];
		const filtered = filterTreeByRole(tree, 'agent');
		expect(filtered[0].children).toBeUndefined();
	});

	it('returns empty array for empty input', () => {
		expect(filterTreeByRole([], 'any')).toEqual([]);
	});
});
