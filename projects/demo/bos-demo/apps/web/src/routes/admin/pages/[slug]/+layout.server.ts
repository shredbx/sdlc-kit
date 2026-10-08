import { error } from '@sveltejs/kit';
import type { CmsPage } from '@sbx/ui-cms/types';
import type { LayoutServerLoad } from './$types';

const API_URL = process.env.PUBLIC_API_URL || 'http://localhost:5010';
const headers = { 'X-Requested-With': 'XMLHttpRequest' };

// Shared by this page's own Content tab and its sibling SEO tab. A 404 from the cms kit means
// this slug has no live row yet — NOT a dead end here (unlike the public route): the cms
// model's Repository.UpsertBySlug creates on first save, so this returns page: null and lets
// the Content tab render a fresh, unsaved editing form for the slug instead of erroring.
export const load: LayoutServerLoad = async ({ fetch, params }) => {
	const res = await fetch(`${API_URL}/api/admin/cms/${params.slug}`, { headers });
	if (res.status === 404) {
		return { slug: params.slug, page: null };
	}
	if (!res.ok) throw error(res.status, 'could not load page');
	const page: CmsPage = await res.json();
	return { slug: params.slug, page };
};
