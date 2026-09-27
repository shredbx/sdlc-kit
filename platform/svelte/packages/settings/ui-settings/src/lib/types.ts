import type { HeaderConfig, FooterConfig } from '@sbx/bos-svelte/chrome';

export interface AdminSettings {
	site_title: string;
	tagline: string;
	header: HeaderConfig;
	footer: FooterConfig;
}
