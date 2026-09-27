import type { Handle } from '@sveltejs/kit';

/**
 * Same-origin pass-through proxy for /api/* — forwards browser cookies +
 * Set-Cookie back. This runs in BOTH dev and prod (there is no Vite /api proxy,
 * so dev mirrors prod exactly — one code path). It is pure transport:
 * no composition. The Go API is the single API for web AND mobile.
 */
export function apiPassThroughHandle(apiBaseUrl: string): Handle {
	return async ({ event, resolve }) => {
		if (!event.url.pathname.startsWith('/api/')) return resolve(event);

		const target = `${apiBaseUrl}${event.url.pathname}${event.url.search}`;
		const hasBody = event.request.method !== 'GET' && event.request.method !== 'HEAD';
		// Buffer the body ONCE so a 401 can be replayed after a silent refresh
		// (the refresh itself arrives with the auth kit).
		const body = hasBody ? await event.request.arrayBuffer() : undefined;

		const headers = new Headers(event.request.headers);
		headers.delete('host');
		// Strip hop-by-hop / message-framing headers before re-issuing the
		// request. The body was re-buffered above, so undici MUST compute its
		// own framing — forwarding the edge's transfer-encoding (a proxy may send
		// the POST body chunked) or a stale content-length makes undici throw
		// UND_ERR_INVALID_ARG → "fetch failed" → 500 on EVERY body-carrying
		// request; GETs have no body so they were unaffected.
		headers.delete('transfer-encoding');
		headers.delete('content-length');
		headers.delete('connection');
		headers.delete('keep-alive');
		// Deliver the REAL client IP to the Go API. The browser never sends
		// X-Forwarded-For; SvelteKit's getClientAddress() resolves it from the
		// edge (adapter-node ADDRESS_HEADER=x-forwarded-for + XFF_DEPTH). The
		// Go API's rate-limiter and visitor-id hash both depend on this — without
		// it every visitor collapses to one IP. Set, never append, so a
		// client-supplied X-Forwarded-For can't spoof the chain.
		headers.set('x-forwarded-for', event.getClientAddress());

		const res = await fetch(target, {
			method: event.request.method,
			headers,
			body,
			// @ts-expect-error Node fetch supports duplex for streaming
			duplex: hasBody ? 'half' : undefined
		});

		return new Response(res.body, {
			status: res.status,
			statusText: res.statusText,
			headers: res.headers
		});
	};
}
