/**
 * ChatMessage security helpers — pure, framework-only (no client/business specifics).
 *
 * Assistant/system message text is LLM output rendered through markdown. The library
 * (`@humanspeak/svelte-markdown`) already blocks `javascript:`/`data:` URLs and `on*`
 * handlers by default. These helpers add two things on top, exposed as plain functions
 * so they can be unit-tested without mounting a component:
 *
 *  1. {@link makeImageSrcSanitizer} — an image-only HOST allowlist on top of the
 *     library's `defaultSanitizeUrl`. The default validates the URL *scheme* but not
 *     the *host*, so an off-site `<img src="https://evil.example/beacon.png">` in LLM
 *     output would still issue a request (IP / load-ping exfiltration). This restricts
 *     rendered image `src` to same-origin (relative URLs + the current host, ALWAYS
 *     allowed) plus a caller-supplied host allowlist; a disallowed host collapses the
 *     URL to `''`, which the library's renderers treat as "no src" (markdown `![]()`
 *     gets an empty `href`; raw `<img>` has its `src` attribute dropped) — so no
 *     request is issued. Links (`<a href>`) are NOT host-restricted.
 *
 *  2. {@link buildBlockedHtmlRenderers} — collapses the structural raw-HTML tags in
 *     {@link BLOCKED_HTML_TAGS} (iframe/embed/object) to escaped text (they otherwise
 *     render live even with handlers stripped: off-site loads / clickjacking / phishing).
 *
 * Same-origin is resolved from `window.location.host` when in a browser; under SSR
 * (`typeof window === 'undefined'`) there is no current origin, so only relative URLs
 * and the explicit allowlist are honoured for absolute hosts.
 */

import { defaultSanitizeUrl, excludeHtmlOnly, UnsupportedHTML } from '@humanspeak/svelte-markdown';
import type { HtmlRenderers, SanitizeContext, SanitizeUrlFn } from '@humanspeak/svelte-markdown';

/**
 * Raw-HTML structural tags stripped from rendered LLM output. `@humanspeak/svelte-markdown`
 * renders raw HTML elements structurally; even with `on*` handlers and `srcdoc` stripped,
 * these stay live (off-site resource loads, clickjacking, phishing frames), so each is
 * collapsed to escaped plain text by {@link buildBlockedHtmlRenderers}.
 *
 * `iframe` and `embed` have default renderers in the library; `object` does NOT — and a
 * tag with no default renderer takes the Parser's "render live" fast-path (see
 * {@link buildBlockedHtmlRenderers}), so it cannot be blocked by `excludeHtmlOnly` alone.
 * It is force-mapped to the placeholder there. Listing it also future-proofs the policy
 * against a library version that later adds an `object` renderer.
 */
export const BLOCKED_HTML_TAGS: readonly string[] = ['iframe', 'embed', 'object'];

/**
 * Builds the `renderers.html` map that neutralizes {@link BLOCKED_HTML_TAGS}.
 *
 * `excludeHtmlOnly` alone is insufficient: it only swaps tags that already have a
 * default renderer (`iframe`, `embed`), silently skipping a tag with no default entry
 * (`object`). For an unknown tag the Parser's fast-path compares
 * `renderers.html[tag] === Html[tag]` — `undefined === undefined` — and renders it live
 * via `<svelte:element>`. So every blocked tag is ALSO force-mapped to `UnsupportedHTML`
 * here: that makes `renderers.html[tag] !== Html[tag]` (defeating the live fast-path) and
 * puts the tag `in renderers.html` (so the main branch renders the escaped-text
 * placeholder). Net: known and unknown structural tags alike collapse to inert text.
 */
export function buildBlockedHtmlRenderers(): HtmlRenderers {
	const html = excludeHtmlOnly([...BLOCKED_HTML_TAGS]);
	for (const tag of BLOCKED_HTML_TAGS) html[tag] = UnsupportedHTML;
	return html;
}

/** True when the sanitize context describes an image `src` (markdown image or raw `<img>`). */
function isImageContext(context: SanitizeContext): boolean {
	return context.type === 'image' || context.tag === 'img';
}

/**
 * Extracts the lowercased host of an absolute URL, or `null` when the URL is relative
 * (no authority) or unparseable. A relative URL is same-origin by construction.
 *
 * Parsed against a dummy base so protocol-relative (`//host/…`) and absolute URLs both
 * resolve a host while genuinely relative URLs (`/path`, `./x`, `#a`, `img.png`) do not.
 */
function hostOf(url: string): string | null {
	// Relative markers — the default sanitizer returns these unchanged; no host to check.
	if (/^[#/?.]/.test(url) && !url.startsWith('//')) return null;
	if (!url.includes(':') && !url.startsWith('//')) return null;
	try {
		const base = 'http://__same_origin__.invalid';
		const parsed = new URL(url, base);
		if (parsed.host === '' || parsed.host === '__same_origin__.invalid') return null;
		return parsed.host.toLowerCase();
	} catch {
		return null;
	}
}

/** The current page host (lowercased), or `null` under SSR / no DOM. */
function currentHost(): string | null {
	if (typeof window === 'undefined' || !window.location?.host) return null;
	return window.location.host.toLowerCase();
}

/**
 * Builds a `sanitizeUrl` function for `<SvelteMarkdown sanitizeUrl=…>` that keeps the
 * library's default scheme allowlist AND restricts image hosts to same-origin + the
 * supplied allowlist. Non-image URLs (links) keep only the default scheme check.
 *
 * @param allowedImageHosts Extra hosts permitted for image `src` (e.g. a media/CDN host).
 *   Matched case-insensitively against the URL host. Same-origin and relative URLs are
 *   always allowed regardless of this list. Entries may be bare hosts (`cdn.example.com`)
 *   or include a port (`cdn.example.com:8443`); a leading `http(s)://` is tolerated.
 */
export function makeImageSrcSanitizer(allowedImageHosts: readonly string[] = []): SanitizeUrlFn {
	const allow = new Set(
		allowedImageHosts
			.map((h) => h.trim().toLowerCase().replace(/^https?:\/\//, '').replace(/\/+$/, ''))
			.filter((h) => h.length > 0)
	);

	return (url: string, context: SanitizeContext): string => {
		// 1. Always apply the library's default first — preserves the scheme allowlist
		//    (blocks javascript:, data:, vbscript:, …). Never bypass it.
		const safe = defaultSanitizeUrl(url, context);
		if (!safe) return '';

		// 2. Links and other URL-bearing attributes: default scheme check is the policy.
		if (!isImageContext(context)) return safe;

		// 3. Images: enforce the host allowlist on top of the scheme check.
		const host = hostOf(safe);
		if (host === null) return safe; // relative → same-origin → allowed

		const origin = currentHost();
		if (origin !== null && host === origin) return safe; // explicit same-origin
		if (allow.has(host)) return safe; // caller-allowed host

		// Off-allowlist image host → neutralize the src so no request is issued.
		return '';
	};
}
