import { error } from '@sveltejs/kit';
import type { PageListItem } from '@sbx/ui-cms/types';
import type { PageServerLoad } from './$types';

const API_URL = process.env.PUBLIC_API_URL || 'http://localhost:5010';
const headers = { 'X-Requested-With': 'XMLHttpRequest' };

// Backs the Pages list — the cms kit's own List capability (platform/go/packages/cms's
// Repository.List + GET /admin/cms), added specifically to fix the regression left by
// the earlier flat-pages-to-cms port: the old model's admin route had a hardcoded slug
// list to choose from; the cms port's replacement had none, so the list "disappeared"
// even though pages could still be created/edited by slug.
export const load: PageServerLoad = async ({ fetch }) => {
	const res = await fetch(`${API_URL}/api/admin/cms`, { headers });
	if (!res.ok) throw error(res.status, 'could not load pages');
	const pages: PageListItem[] = await res.json();
	return { pages };
};
