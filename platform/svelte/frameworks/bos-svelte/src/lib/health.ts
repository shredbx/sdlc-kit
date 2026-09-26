/** What the placeholder home shows about the API: whether it answered, and what it reported about itself. */
export interface ApiHealth {
	status: 'healthy' | 'unreachable';
	service?: string;
	version?: string;
	environment?: string;
	database?: string;
}

/**
 * Ask the API for its health from the server side (a page's `load`). It never throws:
 * an API that is down, slow or answering something else is reported as `unreachable`,
 * so the page still renders and says so.
 */
export async function loadApiHealth(
	apiUrl: string,
	fetchFn: typeof fetch = fetch,
	timeoutMs = 3000
): Promise<ApiHealth> {
	try {
		const res = await fetchFn(`${apiUrl}/health`, { signal: AbortSignal.timeout(timeoutMs) });
		if (!res.ok) return { status: 'unreachable' };
		const body = await res.json();
		if (body?.status !== 'healthy') return { status: 'unreachable' };
		return {
			status: 'healthy',
			service: body.service,
			version: body.version,
			environment: body.environment,
			database: body.database
		};
	} catch {
		return { status: 'unreachable' };
	}
}
