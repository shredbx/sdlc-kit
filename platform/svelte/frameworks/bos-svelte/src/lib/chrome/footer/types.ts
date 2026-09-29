import type { Component } from 'svelte';

/** Mirrors bos-demo/apps/api/internal/settings/settings.go's FooterLink — same page_id/custom_path
 * duality as a header NavItem, page_id resolution deferred the same way (see header/types.ts). */
export interface FooterLink {
	title: string;
	page_id?: string;
	custom_path?: string;
	enabled: boolean;
	web: boolean;
	mobile: boolean;
}

/** Mirrors settings.go's FooterSection. */
export interface FooterSection {
	heading: string;
	links: FooterLink[];
}

/** Mirrors settings.go's Footer. */
export interface FooterConfig {
	enabled: boolean;
	preset: string;
	sections: FooterSection[];
}

/** One entry in the footer-preset registry — same shape as RegisteredHeader, kept a separate type
 * since a header preset and a footer preset are never chosen from the same list. */
export interface RegisteredFooter {
	id: string;
	label: string;
	component: Component<{ config: FooterConfig }>;
}
