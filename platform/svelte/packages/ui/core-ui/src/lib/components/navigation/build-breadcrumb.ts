import type { BreadcrumbItem } from './Breadcrumb.svelte';

export interface BreadcrumbConfig {
	/** Map URL → display label, e.g. { '/manage/properties': 'Properties' }. */
	routeMap: Record<string, string>;
	/** Per-page dynamic overrides (e.g. resolve `[id]` to the entity title). */
	overrides?: Record<string, string>;
	/**
	 * Per-crumb href remaps, keyed by cumulative path. When a crumb has an entry,
	 * it links to that TARGET instead of its own cumulative path — and becomes
	 * linkable even when its own path has no handler. For surfaces whose hub lives
	 * at a different route than the URL's parent segment (e.g. a property detail at
	 * `/manage/properties/[id]` whose hub is `/manage/content/properties`): the
	 * parent crumb points at the real hub rather than its dead URL parent. Additive
	 * — crumbs without an entry are unchanged.
	 */
	hrefOverrides?: Record<string, string>;
	/** Optional first crumb prepended to all results (e.g. 'Home'). */
	rootLabel?: string;
	/** Href for the optional rootLabel crumb. */
	rootHref?: string;
}

/**
 * Walk URL segments and emit a breadcrumb item per cumulative path.
 *
 * Label resolution per segment, in order of precedence:
 *   1. `overrides[cumulativePath]` (per-page dynamic label)
 *   2. `routeMap[cumulativePath]` (static configured label)
 *   3. Humanized last segment (e.g. `unknown-section` → `Unknown Section`)
 *
 * The last item has no `href` (it's the current page). Intermediate items
 * link to the cumulative path ONLY when that path is declared in `routeMap`
 * or `overrides` — i.e. the route is known to exist. Paths that humanize via
 * the fallback (no map entry) render as plain text without an href, so the
 * breadcrumb never produces a link to a 404 page (e.g. `/manage` when only
 * `/manage/properties` has a handler).
 *
 * An intermediate crumb with a `hrefOverrides` entry links to that TARGET
 * instead of its own cumulative path (and is linkable even without a routeMap/
 * overrides entry) — for surfaces whose hub route differs from the URL's parent
 * segment.
 *
 * Pure function — no DOM, no SvelteKit deps.
 */
export function buildBreadcrumb(pathname: string, config: BreadcrumbConfig): BreadcrumbItem[] {
	if (!pathname || pathname === '/') {
		return config.rootLabel ? [{ label: config.rootLabel, href: config.rootHref }] : [];
	}

	const segments = pathname.split('/').filter(Boolean);
	const items: BreadcrumbItem[] = [];

	if (config.rootLabel) {
		items.push({ label: config.rootLabel, href: config.rootHref });
	}

	let cumulative = '';
	for (let i = 0; i < segments.length; i++) {
		cumulative += '/' + segments[i];
		const isLast = i === segments.length - 1;
		const overrideHref = config.hrefOverrides?.[cumulative];
		const isLinkable =
			!!overrideHref || !!config.routeMap[cumulative] || !!config.overrides?.[cumulative];
		items.push({
			label: resolveLabel(cumulative, config),
			href: isLast || !isLinkable ? undefined : (overrideHref ?? cumulative)
		});
	}

	return items;
}

function resolveLabel(path: string, config: BreadcrumbConfig): string {
	if (config.overrides?.[path]) return config.overrides[path];
	if (config.routeMap[path]) return config.routeMap[path];
	const lastSegment = path.split('/').pop() ?? '';
	return humanize(lastSegment);
}

function humanize(segment: string): string {
	return segment
		.split('-')
		.map((word) => (word.length > 0 ? word.charAt(0).toUpperCase() + word.slice(1) : word))
		.join(' ');
}
