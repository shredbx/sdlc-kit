// Security headers for every SvelteKit-rendered response. The Go API sets its own via
// its security-headers middleware — this covers the HTML/page side.
//
// CSP is deliberately MINIMAL-ENFORCED: only the four directives that cannot
// break a legitimate page — no script-src/style-src (those need a nonce
// pipeline + a violation-report collector to roll out honestly). What these four buy:
//   frame-ancestors 'self'  — clickjacking: nobody frames the site cross-origin
//                             (admin same-origin previews keep working);
//                             X-Frame-Options mirrors it for older engines.
//   base-uri 'self'         — a markup injection can't retarget relative URLs
//                             via <base href>.
//   object-src 'none'       — no <object>/<embed> plugin execution, ever.
//   form-action 'self'      — an injected <form> can't exfiltrate credentials
//                             to a foreign origin (all real forms post same-
//                             origin). OUTGOING iframes are frame-src
//                             territory — unrestricted here, unaffected.
//
// HSTS only when the browser-facing hop is HTTPS (X-Forwarded-Proto): a dev
// http response must never teach localhost an HSTS pin. No includeSubDomains.

import type { Handle } from '@sveltejs/kit';

const HSTS_MAX_AGE_S = 31536000; // 1 year

const BASE_HEADERS: ReadonlyArray<readonly [string, string]> = [
	['X-Content-Type-Options', 'nosniff'],
	['Referrer-Policy', 'strict-origin-when-cross-origin'],
	['X-Frame-Options', 'SAMEORIGIN'],
	['Permissions-Policy', 'camera=(), microphone=(), geolocation=()'],
	[
		'Content-Security-Policy',
		"frame-ancestors 'self'; base-uri 'self'; object-src 'none'; form-action 'self'"
	]
];

/**
 * Stamp the security header set onto a response's headers (overwrites — the
 * policy is ours, not the page's). Applied by the hooks sequence to every
 * non-/api response; /api responses carry the Go API's own header set
 * untouched, so the two boundaries never fight over policy.
 */
export function applySecurityHeaders(headers: Headers, isHttps: boolean): void {
	for (const [name, value] of BASE_HEADERS) {
		headers.set(name, value);
	}
	if (isHttps) {
		headers.set('Strict-Transport-Security', `max-age=${HSTS_MAX_AGE_S}`);
	}
}

/**
 * Determine whether the inbound request reached us over HTTPS.
 *
 * In prod, a proxy terminates TLS and forwards plain HTTP to the
 * SvelteKit Node process — so `event.url.protocol` is `http:` even when the
 * user's browser URL bar shows `https://`. We must consult `X-Forwarded-Proto`
 * to set the `Secure` cookie flag correctly.
 */
export function isHttpsRequest(event: { url: URL; request: Request }): boolean {
	if (event.url.protocol === 'https:') return true;
	return event.request.headers.get('x-forwarded-proto') === 'https';
}

// Security headers on every SvelteKit-rendered response. Runs as the
// OUTER hook so it stamps whatever the main handle returns — page renders,
// redirects, error pages — without touching its many return sites. /api
// responses are skipped: they are Go API responses passed through verbatim,
// and the Go side owns its own SecurityHeaders policy.
export const securityHeadersHandle: Handle = async ({ event, resolve }) => {
	const response = await resolve(event);
	if (!event.url.pathname.startsWith('/api/')) {
		applySecurityHeaders(response.headers, isHttpsRequest(event));
	}
	return response;
};
