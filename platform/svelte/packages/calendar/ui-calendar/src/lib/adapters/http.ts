// HTTP CalendarAdapter — the real backend behind the SAME contract createMockAdapter
// satisfies. The frontend-first build ran against the mock; this slots in unchanged
// (Level 1 § ① — "createBookingHttpAdapter(baseUrl) will satisfy the SAME interface
// later"). It talks to the Go manage API (api-chi EventHandler), whose JSON is shaped
// 1:1 to these types, so no translation layer is needed.
//
// Defaults target a same-origin reverse proxy (SvelteKit hooks forward the auth
// cookie to the Go API) with the `X-Requested-With` header the API's write-CSRF gate
// requires — the established BR admin convention. A consumer on a different topology
// overrides baseUrl / headers / fetch. No `any`: errors surface as thrown Errors the
// host's loading/try-catch already handles.

import type {
	AddRefInput,
	CalendarAdapter,
	CalendarItem,
	CreateEventInput,
	DateRange,
	EventStatus,
	MutationResult,
	UpdateEventInput
} from '../types.js';

export interface HttpAdapterOptions {
	/** Origin prefix for the API. Default '' → same-origin (the proxy path /api/...). */
	baseUrl?: string;
	/** Fetch implementation (injectable for tests/SSR). Default the global fetch. */
	fetch?: typeof fetch;
	/** Extra headers merged into every WRITE (e.g. a CSRF token). The default carries
	 *  the X-Requested-With header the API's write-CSRF gate checks. */
	headers?: Record<string, string>;
}

/**
 * createEventHttpAdapter — a CalendarAdapter backed by the manage events API:
 *   GET    /api/manage/events?start=&end=&scope=     list
 *   POST   /api/manage/events                        create
 *   PUT    /api/manage/events/{id}                   update (type/title/notes/all-day)
 *   PATCH  /api/manage/events/{id}/move              re-schedule
 *   PATCH  /api/manage/events/{id}/status            set status
 *   POST   /api/manage/events/{id}/refs              add reference
 *   DELETE /api/manage/events/{id}/refs/{refId}      remove reference
 */
export function createEventHttpAdapter(opts: HttpAdapterOptions = {}): CalendarAdapter {
	const base = opts.baseUrl ?? '';
	const doFetch = opts.fetch ?? fetch;
	const root = `${base}/api/manage/events`;
	const writeHeaders: Record<string, string> = {
		'Content-Type': 'application/json',
		'X-Requested-With': 'XMLHttpRequest',
		...(opts.headers ?? {})
	};

	async function request<T>(url: string, init?: RequestInit): Promise<T> {
		const res = await doFetch(url, { credentials: 'same-origin', ...init });
		if (!res.ok) {
			const body = (await res.json().catch(() => null)) as { error?: string } | null;
			throw new Error(body?.error ?? `calendar API error (${res.status})`);
		}
		return (await res.json()) as T;
	}

	const write = (url: string, method: string, body?: unknown): Promise<MutationResult> =>
		request<MutationResult>(url, {
			method,
			headers: writeHeaders,
			body: body === undefined ? undefined : JSON.stringify(body)
		});

	return {
		async list(range: DateRange, scope: 'mine' | 'all'): Promise<CalendarItem[]> {
			const qs = new URLSearchParams({ start: range.start, end: range.end, scope });
			return request<CalendarItem[]>(`${root}?${qs.toString()}`);
		},
		create(input: CreateEventInput): Promise<MutationResult> {
			return write(root, 'POST', input);
		},
		update(id: string, patch: UpdateEventInput): Promise<MutationResult> {
			return write(`${root}/${id}`, 'PUT', patch);
		},
		move(id: string, start: string, end: string): Promise<MutationResult> {
			return write(`${root}/${id}/move`, 'PATCH', { start, end });
		},
		setStatus(id: string, status: EventStatus): Promise<MutationResult> {
			return write(`${root}/${id}/status`, 'PATCH', { status });
		},
		addReference(id: string, ref: AddRefInput): Promise<MutationResult> {
			return write(`${root}/${id}/refs`, 'POST', ref);
		},
		removeReference(id: string, referenceId: string): Promise<MutationResult> {
			return write(`${root}/${id}/refs/${encodeURIComponent(referenceId)}`, 'DELETE');
		}
	};
}
