import type { LayoutServerLoad } from './$types';

// Public, unauthenticated read of site chrome (docs/proposals/bos-site-chrome.md §5) — same
// direct apiUrl fetch pattern as the home page's own load.
const API_URL = process.env.PUBLIC_API_URL || 'http://localhost:5010';

const fallback = {
	site_title: process.env.PUBLIC_SITE_NAME || 'bos-demo',
	header: { enabled: false, preset: 'default', nav: [] },
	footer: { enabled: false, preset: 'default', sections: [] }
};

export const load: LayoutServerLoad = async ({ fetch }) => {
	const res = await fetch(`${API_URL}/api/settings`);
	const settings = res.ok ? await res.json() : fallback;
	return { settings };
};
