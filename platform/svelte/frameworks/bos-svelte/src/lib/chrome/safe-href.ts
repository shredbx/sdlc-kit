// custom_path is free-text admin input (docs/proposals/bos-site-chrome.md §3) rendered straight into
// a public <a href> — a value like "javascript:alert(1)" would execute for any visitor who clicks it.
// Shared by both header and footer default presets rather than duplicated per component.
const ALLOWED_SCHEMES = new Set(['http:', 'https:', 'mailto:']);

export function safeHref(path: string | undefined): string | undefined {
	if (!path) return undefined;
	if (path.startsWith('/') && !path.startsWith('//')) return path;
	try {
		return ALLOWED_SCHEMES.has(new URL(path).protocol) ? path : '#';
	} catch {
		return '#';
	}
}
