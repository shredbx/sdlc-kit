import type { AdminModule } from '@sbx/bos-svelte/admin';

/** This kit's admin-surface manifest (bos-system-design.md D18). The consumer's
 * admin layout imports this — the same way a Go consumer's main.go imports and
 * calls cms.RegisterAdmin — to compose its sidebar, instead of hand-typing the
 * nav entry. See docs/proposals/bos-admin-modules.md §6. */
export const registeredAdminModule: AdminModule = {
	id: 'cms',
	navItem: { label: 'Pages', href: '/admin/pages' }
};
