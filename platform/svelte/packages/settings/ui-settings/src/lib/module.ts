import type { AdminModule } from '@sbx/bos-svelte/admin';

/** This kit's admin-surface manifest (bos-system-design.md D18). See
 * @sbx/ui-cms's module.ts for the full rationale — same pattern here.
 * roles: ['admin'] mirrors the API's own RequireRole("admin") gate on
 * /api/admin/settings (docs/proposals/bos-site-chrome.md's content-manager vs
 * administrator split) — a UX hint only, the server enforces it independently. */
export const registeredAdminModule: AdminModule = {
	id: 'settings',
	navItem: { label: 'Site configuration', href: '/admin/settings', roles: ['admin'] }
};
