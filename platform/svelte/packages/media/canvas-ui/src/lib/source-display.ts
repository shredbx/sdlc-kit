// Shared source-display helpers (G6b) — one source of truth for how an attached
// SourceRef is named/shown in the UI, reused by SourceCell (Sources panel) and the
// editor-state imageSources() bundle (Media panel [Sources ▾] filter). Keeping this
// here avoids re-deriving the title in two places (never-copy-paste).

import type { SourceRef } from '@sbx/canvas-kit';

/** The record's human display title — the resolved snapshot `title`, else an honest
 *  "Untitled <kind>" (never the raw refId/UUID, which is useless to the user). */
export function sourceTitle(ref: SourceRef): string {
	const t = ref.snapshot?.title;
	return typeof t === 'string' && t.trim() ? t : `Untitled ${ref.kind}`;
}
