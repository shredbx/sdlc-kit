import type { Component } from 'svelte';

/** Mirrors bos-demo/apps/api/internal/settings/settings.go's NavItem (docs/proposals/bos-site-chrome.md
 * §3). page_id resolution to a real page's route (fetching the page and resolving its slug) is not
 * built yet — the admin only offers custom_path today, on purpose, so it never creates a nav item
 * this renderer can't resolve; today only custom_path renders to a real href. */
export interface NavItem {
	type: 'link' | 'separator';
	title: string;
	page_id?: string;
	custom_path?: string;
	icon?: string;
	enabled: boolean;
	web: boolean;
	mobile: boolean;
	new_tab: boolean;
}

/** Mirrors settings.go's Header. */
export interface HeaderConfig {
	enabled: boolean;
	preset: string;
	nav: NavItem[];
}

/** One entry in the header-preset registry (docs/proposals/bos-page-designer.md §5's mechanism,
 * applied to chrome instead of a page renderer). `siteTitle` is passed alongside `config` (not
 * inside it) since it's the site's own identity, not a per-preset editable field — the same
 * settings.site_title the admin's General tab already edits (bos-site-chrome.md §4: "site
 * title/logo + the nav[] list"). */
export interface RegisteredHeader {
	id: string;
	label: string;
	component: Component<{ config: HeaderConfig; siteTitle?: string }>;
}
