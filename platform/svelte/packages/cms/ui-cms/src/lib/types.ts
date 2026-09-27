import type { SectionData } from '@sbx/bos-svelte/renderers';
import type { SeoMeta } from '@sbx/ui-seo/types';

/** One row of the admin Pages list — matches the cms kit's Go-side pageListItem
 * projection (GET /admin/cms). */
export interface PageListItem {
	slug: string;
	title?: string;
	published: boolean;
	updated_at: string;
}

// SeoMeta is re-exported from @sbx/ui-seo rather than redeclared here — it was
// previously a narrower local duplicate (meta_title/meta_description only), which
// undersold what the cms kit's Go side already stores (the full shared seo.SeoMeta:
// og_*/canonical_url/noindex too). Re-exporting keeps this the single TS declaration.
export type { SeoMeta };

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
