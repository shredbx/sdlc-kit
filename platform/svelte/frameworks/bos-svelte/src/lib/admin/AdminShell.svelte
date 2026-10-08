<script lang="ts">
	/**
	 * AdminShell — the one admin chrome every bos consumer gets for free (framework work, not
	 * app work; docs/proposals/bos-page-designer.md §2). Calls createTopbarSlots() once so
	 * descendant pages (via TabbedPageShell, or useTopbarSlots() directly) can register a
	 * toolbar/actions snippet that renders here, in this shell's own topbar row — without that
	 * provider, useTopbarSlots() throws ("no provider found"), which is exactly what blocked
	 * TabbedPageShell from being used before this component existed.
	 */
	import type { Snippet } from 'svelte';
	import { createTopbarSlots } from '@sbx/core-ui/stores/topbarSlots';
	import { SidebarLayout } from '@sbx/core-ui/components/layouts';

	/** One sidebar link. `roles` omitted = visible regardless of role (or when the app passes no
	 * role at all, i.e. no RBAC wired up yet) — only listed here to keep the app in control of
	 * what its own admin areas are called and who reaches them; this shell only filters. */
	export interface AdminNavItem {
		href: string;
		label: string;
		roles?: string[];
	}

	/** A kit's own admin-surface manifest (bos-system-design.md D18: "a Svelte manifest declares
	 * ... admin navigation entries"). A kit's Svelte package exports one of these (see
	 * @sbx/ui-cms, @sbx/ui-settings); the consumer app's own admin layout — its composition root,
	 * mirroring how main.go explicitly calls each enabled kit's RegisterAdmin on the Go side —
	 * imports the modules it wants enabled and builds navItems from them, instead of hand-typing
	 * the array. Only navItem exists so far because no module needs more than a nav entry yet;
	 * add list/configure component slots when a real module needs AdminShell to render one
	 * inline, not speculatively (docs/proposals/bos-admin-modules.md §6, §8). */
	export interface AdminModule {
		id: string;
		navItem: AdminNavItem;
	}

	let {
		children,
		navItems,
		role
	}: { children: Snippet; navItems: AdminNavItem[]; role?: string } = $props();

	// Sets the context TabbedPageShell (and any page calling useTopbarSlots() directly) reads.
	const topbar = createTopbarSlots();

	const visibleItems = $derived(
		navItems.filter((item) => !item.roles || (role !== undefined && item.roles.includes(role)))
	);
</script>

{#snippet nav()}
	<nav class="admin-nav" aria-label="Admin">
		{#each visibleItems as item (item.href)}
			<a href={item.href}>{item.label}</a>
		{/each}
	</nav>
{/snippet}

<SidebarLayout sidebar={nav} toolbar={$topbar.toolbar} actions={$topbar.actions} storageKey="bos-admin-sidebar-collapsed">
	{@render children()}
</SidebarLayout>

<style>
	.admin-nav {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		padding: 1rem;
	}
	.admin-nav a {
		padding: 0.5rem 0.75rem;
		border-radius: 0.375rem;
		text-decoration: none;
		color: inherit;
		font-family: sans-serif;
		font-size: 0.9rem;
	}
	.admin-nav a:hover {
		background: rgba(0, 0, 0, 0.06);
	}
</style>
