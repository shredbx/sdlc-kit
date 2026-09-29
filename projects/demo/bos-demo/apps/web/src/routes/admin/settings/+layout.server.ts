import { error } from '@sveltejs/kit';
import type { AdminSettings } from '@sbx/ui-settings/types';
import type { LayoutServerLoad } from './$types';

const API_URL = process.env.PUBLIC_API_URL || 'http://localhost:5010';
const headers = { 'X-Requested-With': 'XMLHttpRequest' };

// Administrator-only (docs/proposals/bos-site-chrome.md's content-manager vs administrator split,
// enforced server-side by RequireRole("admin") on the API). A content-manager landing here should
// see a clear Forbidden page, not a quietly empty settings form.
export const load: LayoutServerLoad = async ({ fetch }) => {
	const res = await fetch(`${API_URL}/api/admin/settings`, { headers });
	if (res.status === 403) error(403, 'Site configuration is for administrators only.');
	if (!res.ok) error(res.status, 'Could not load settings.');
	const settings: AdminSettings = await res.json();
	return { settings };
};
