import type { LayoutServerLoad } from './$types';

const API_URL = process.env.PUBLIC_API_URL || 'http://localhost:5010';
const headers = { 'X-Requested-With': 'XMLHttpRequest' };

// Which role the current request is acting as (docs/proposals/bos-site-chrome.md's content-manager
// vs administrator split) — read once here so every admin route can filter by it without its own
// fetch. undefined (whoami unreachable) means "no RBAC signal" — AdminShell shows everything rather
// than guessing.
export const load: LayoutServerLoad = async ({ fetch }) => {
	const res = await fetch(`${API_URL}/api/admin/whoami`, { headers });
	const who = res.ok ? await res.json() : {};
	return { role: who.role as string | undefined };
};
