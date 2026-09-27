import type { SectionData } from '@sbx/bos-svelte/renderers';

/** One row of the admin Pages list — matches the cms kit's Go-side pageListItem
 * projection (GET /admin/cms). */
export interface PageListItem {
	slug: string;
	title?: string;
	published: boolean;
	updated_at: string;
}

export interface SeoMeta {
	meta_title?: string;
	meta_description?: string;
}

/** The full admin shape of a CMS page — matches the cms kit's Go-side CmsPage as
 * exposed by GET/PUT /admin/cms/{slug}. Only the fields this UI reads/writes. */
export interface CmsPage {
	slug: string;
	title?: string;
	body_markdown?: string;
	published: boolean;
	published_content?: SectionData[];
	seo_meta?: SeoMeta;
	/** The registered layout preset id (bos-svelte's layout registry) this page
	 * uses — "default" (hero + main) or "legal" (main only) today. */
	layout: string;
}
