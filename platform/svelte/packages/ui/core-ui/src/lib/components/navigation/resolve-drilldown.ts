import type { ContentSidebarSection } from './ContentSidebar.svelte';

export interface DrillDownResult {
	drilledInto: ContentSidebarSection | null;
	parentLabel: string | null;
	parentHref: string | null;
}

export interface ResolveDrillDownOptions {
	rootLabel?: string;
	rootHref?: string;
}

const DEFAULT_ROOT_LABEL = 'Dashboard';
const DEFAULT_ROOT_HREF = '/';

/**
 * Determine which top-level section (if any) the current pathname is inside.
 *
 * A section "drills in" when:
 *   1. It has children or items (something to render at Level 2).
 *   2. Its href is set and prefixes the current pathname (with `/` or `?` boundary).
 *
 * The first matching section wins (top-level tree only — does not recurse into
 * grandchildren). Returns `drilledInto: null` when no section matches; consumer
 * renders the Level 1 (root) sidebar in that case.
 *
 * Pure function — no DOM, no localStorage, no side effects.
 */
export function resolveDrillDown(
	tree: ContentSidebarSection[],
	pathname: string,
	options: ResolveDrillDownOptions = {}
): DrillDownResult {
	const rootLabel = options.rootLabel ?? DEFAULT_ROOT_LABEL;
	const rootHref = options.rootHref ?? DEFAULT_ROOT_HREF;

	for (const section of tree) {
		if (!section.href) continue;
		if (!hasDrillContent(section)) continue;
		if (!matchesPrefix(pathname, section.href)) continue;
		return { drilledInto: section, parentLabel: rootLabel, parentHref: rootHref };
	}

	return { drilledInto: null, parentLabel: null, parentHref: null };
}

function hasDrillContent(section: ContentSidebarSection): boolean {
	return (
		(section.children !== undefined && section.children.length > 0) ||
		(section.items !== undefined && section.items.length > 0)
	);
}

function matchesPrefix(pathname: string, href: string): boolean {
	if (pathname === href) return true;
	return pathname.startsWith(href + '/') || pathname.startsWith(href + '?');
}
