// Template gallery contract (#4) — the consumer-owned data the Templates panel renders.
// The kit/ui stay project-agnostic: the consumer (BR) loads the lightweight summaries
// (id/title — kind='template' rows), lazy-fetches each doc for the live preview, and
// owns Apply (typically: create a new document FROM the template, then open it). Bundled
// as one object so it threads cleanly EditorShell → DocumentPanel → TemplatesPanel.

import type { Document } from '@sbx/canvas-kit';

/** One template row — the lightweight list projection (no `doc` payload; that loads
 *  lazily per visible card for the live preview). */
export interface TemplateSummary {
	id: string;
	title: string;
	/** Optional baked thumbnail; when absent the card live-renders the doc. */
	previewUrl?: string;
}

/** Everything the Templates panel needs to show REAL templates with live previews.
 *  Omit it (undefined) and the panel renders its empty state. */
export interface TemplateGallery {
	/** The available templates (already excludes the doc being edited). */
	templates: TemplateSummary[];
	/** Lazily fetch a template's full kit Document for the live preview — called once
	 *  per card (cache-friendly on the consumer side). Returns null on any failure. */
	loadDoc: (id: string) => Promise<Document | null>;
	/** Apply a template — the consumer creates a new document from it and opens it. */
	onApply: (id: string) => void;
}
