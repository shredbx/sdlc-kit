import type { PageServerLoad } from './$types';

// The MVP's flat title+body placeholder is gone: the home route now renders the page's ordered
// sections through the renderer registry (docs/proposals/bos-page-designer.md). Layout presets
// (topbar/hero/footer chrome) are still deferred — that's the next, separate offline design pass.
// Direct apiUrl fetch, same pattern as loadApiHealth.
const API_URL = process.env.PUBLIC_API_URL || 'http://localhost:5010';

export const load: PageServerLoad = async ({ fetch }) => {
	const res = await fetch(`${API_URL}/api/pages/home`);
	if (!res.ok) {
		return { found: false as const, siteName: process.env.PUBLIC_SITE_NAME || 'bos-demo' };
	}
	const page = await res.json();
	return { found: true as const, page };
};
