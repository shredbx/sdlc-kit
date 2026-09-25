import type { ContentSidebarSection } from './ContentSidebar.svelte';

/**
 * Recursively prune sections whose `roles?` field excludes the current user role.
 *
 * Visibility rules:
 *   - Section with no `roles` field (or empty array) → visible to all
 *   - Section with `roles` set → visible only when `userRole ∈ roles`
 *
 * Returns a new tree (does not mutate inputs).
 *
 * IMPORTANT: This is UX filtering only. Server-side handlers MUST still enforce
 * role-based access on the protected routes themselves; hiding nav items is not
 * a security perimeter.
 */
export function filterTreeByRole(
	tree: ContentSidebarSection[],
	userRole: string
): ContentSidebarSection[] {
	const result: ContentSidebarSection[] = [];
	for (const section of tree) {
		if (!isVisible(section, userRole)) continue;
		const filtered: ContentSidebarSection = { ...section };
		if (section.children !== undefined) {
			filtered.children = filterTreeByRole(section.children, userRole);
		}
		result.push(filtered);
	}
	return result;
}

function isVisible(section: ContentSidebarSection, userRole: string): boolean {
	if (!section.roles || section.roles.length === 0) return true;
	return section.roles.includes(userRole);
}
