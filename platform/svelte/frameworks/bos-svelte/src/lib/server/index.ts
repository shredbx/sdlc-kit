import type { Handle } from '@sveltejs/kit';
import { sequence } from '@sveltejs/kit/hooks';
import { apiPassThroughHandle } from './api-pass-through';
import { securityHeadersHandle } from './security-headers';

export { applySecurityHeaders, isHttpsRequest, securityHeadersHandle } from './security-headers';
export { apiPassThroughHandle } from './api-pass-through';

export interface BosHandleOptions {
	/** Where the Go API listens, for example http://localhost:5000. */
	apiUrl: string;
}

/**
 * The framework's server hook: security headers on every page response, then the
 * same-origin /api pass-through. The app's `hooks.server.ts` exports it as `handle`.
 */
export function createHandle(options: BosHandleOptions): Handle {
	return sequence(securityHeadersHandle, apiPassThroughHandle(options.apiUrl));
}
