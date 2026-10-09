// Resolves the active locale ONCE for every page beneath it. This is what replaces the six
// per-locale wrapper route files (/, /ru, /th and their /cookies twins) with one group.
import type { Locale } from '$lib/content';
import { MASTER_LOCALE } from '$lib/content';

export const prerender = true;

export const load = ({ params }) => {
	// The matcher (src/params/lang.ts) has already rejected anything that is not ru|th,
	// so an absent param means English at the bare root.
	const locale = (params.lang ?? MASTER_LOCALE) as Locale;
	return { locale };
};
