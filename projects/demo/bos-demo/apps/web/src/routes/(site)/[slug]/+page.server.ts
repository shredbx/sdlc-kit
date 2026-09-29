import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

// Any published page beyond "home" (which the root route serves) reads through this one
// catch-all, the same way bos-constructor.md's "a page is one file, one catch-all route"
// principle works for the file-sourced model — here sourced from Postgres instead.
const API_URL = process.env.PUBLIC_API_URL || 'http://localhost:5010';

export const load: PageServerLoad = async ({ fetch, params }) => {
	const res = await fetch(`${API_URL}/api/pages/${params.slug}`);
	if (!res.ok) throw error(404, 'page not found');
	return { page: await res.json() };
};
