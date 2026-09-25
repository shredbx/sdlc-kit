// Browser download helpers for the export flow (S-EXPORT). Kept separate from the
// rasterizer so the blob → file step is independently testable + reusable.

/** Trigger a browser download of `blob` as `filename` (object URL → anchor click
 *  → revoke). No-op outside a browser. */
export function downloadBlob(blob: Blob, filename: string): void {
	if (typeof document === 'undefined') return;
	const url = URL.createObjectURL(blob);
	const a = document.createElement('a');
	a.href = url;
	a.download = filename;
	document.body.appendChild(a);
	a.click();
	a.remove();
	URL.revokeObjectURL(url);
}

/** Derive a safe download filename from a document title + export format. Strips
 *  path/illegal characters so a title can never escape the download or carry an
 *  unexpected extension. Falls back to "canvas" when the title is empty. */
export function exportFilename(title: string, format: ExportFormat): string {
	const base =
		(title ?? '')
			.trim()
			.replace(/[^\w.-]+/g, '-')
			.replace(/^-+|-+$/g, '')
			.slice(0, 80) || 'canvas';
	const ext = format === 'jpeg' ? 'jpg' : format;
	return `${base}.${ext}`;
}

/** The export formats the editor offers (mirrors the kit ExportTarget.format). */
export type ExportFormat = 'png' | 'jpeg' | 'pdf';
