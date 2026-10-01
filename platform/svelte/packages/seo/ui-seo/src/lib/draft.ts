// @sbx/ui-seo · draft — the SeoEditor working copy + its SeoMeta converters.
//
// Form controls bind to concrete strings/booleans (no `undefined`), so the SeoEditor
// edits a SeoDraft. The consumer seeds it from the stored SeoMeta (toSeoDraft) and maps
// it back on save (fromSeoDraft): empty fields are omitted, and an all-empty draft
// becomes `null` — "no overrides" — matching the backend's NULL-for-empty seo_meta
// JSONB semantics. meta_keywords is never edited here (dropped per the 2025-26 verdict)
// but is carried through unchanged so a stored value is preserved.

import type { SeoMeta } from './types';

/** Editor working copy — every field present and concrete for clean two-way binding. */
export interface SeoDraft {
	meta_title: string;
	meta_description: string;
	og_title: string;
	og_description: string;
	og_image: string;
	canonical_url: string;
	noindex: boolean;
}

/** Seed a draft from a (possibly null) stored override. Absent fields → empty. */
export function toSeoDraft(meta: SeoMeta | null | undefined): SeoDraft {
	return {
		meta_title: meta?.meta_title ?? '',
		meta_description: meta?.meta_description ?? '',
		og_title: meta?.og_title ?? '',
		og_description: meta?.og_description ?? '',
		og_image: meta?.og_image ?? '',
		canonical_url: meta?.canonical_url ?? '',
		noindex: meta?.noindex ?? false
	};
}

/**
 * Map the draft back to a SeoMeta override. Each string is trimmed; empties are omitted.
 * `keepKeywords` carries a stored meta_keywords through unchanged (the editor never edits
 * it). Returns `null` when nothing is set, so the backend clears seo_meta to SQL NULL.
 */
export function fromSeoDraft(draft: SeoDraft, keepKeywords?: string[]): SeoMeta | null {
	const clean = (value: string): string | undefined => {
		const trimmed = value.trim();
		return trimmed.length > 0 ? trimmed : undefined;
	};

	const meta: SeoMeta = {
		meta_title: clean(draft.meta_title),
		meta_description: clean(draft.meta_description),
		og_title: clean(draft.og_title),
		og_description: clean(draft.og_description),
		og_image: clean(draft.og_image),
		canonical_url: clean(draft.canonical_url),
		meta_keywords: keepKeywords && keepKeywords.length > 0 ? keepKeywords : undefined,
		noindex: draft.noindex ? true : undefined
	};

	const hasOverride = Object.values(meta).some((v) => v !== undefined);
	return hasOverride ? meta : null;
}
