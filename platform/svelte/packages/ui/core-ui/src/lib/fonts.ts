/**
 * Google Fonts on-demand loader — `@sbx/core-ui/fonts`.
 *
 * One job: make the browser able to PAINT an arbitrary Google font family so a
 * preview row (FontPicker) or a rendered/exported canvas layer (canvas-ui) shows
 * the real face instead of a fallback. Injects a `fonts.googleapis.com/css2`
 * stylesheet `<link>` into `<head>`, then waits on the FontFace set so callers can
 * `await` until the glyphs are actually available.
 *
 * Idempotent — a family is requested at most once per page (a module `Set` tracks
 * the in-flight/loaded families); a second call for the same family resolves off
 * the same promise. SSR-safe — with no `document` it resolves `false` and injects
 * nothing. Brand/system families already present in the page (Inter, system-ui …)
 * are honoured: `document.fonts.load` resolves against the already-registered face
 * and no network stylesheet is needed for them — pass them through `markFontLoaded`
 * (or just let the first call no-op gracefully when the face is already available).
 *
 * Not a React/Svelte concern — pure DOM, usable from any consumer (picker preview,
 * render, export).
 */

/**
 * Families we have already requested this session (keyed by the exact family
 * string the caller passed). Presence here means "don't inject again" — the
 * matching entry in {@link inflight} holds the shared promise.
 */
const requested = new Set<string>();

/**
 * family → the in-flight/settled load promise, so concurrent callers for the same
 * family share one network round-trip and one `document.fonts.load`.
 */
const inflight = new Map<string, Promise<boolean>>();

/** Build the css2 stylesheet URL for a family, weights 400+700, swap display. */
function css2Href(family: string): string {
	// `Open Sans` → `Open+Sans`. encodeURIComponent first so any stray special
	// chars are escaped, then spaces become `+` (the css2 API's convention).
	const name = encodeURIComponent(family.trim()).replace(/%20/g, '+');
	return `https://fonts.googleapis.com/css2?family=${name}:wght@400;700&display=swap`;
}

/** True when the document already has a usable face for `family` (no load needed). */
function alreadyAvailable(family: string): boolean {
	if (typeof document === 'undefined' || !document.fonts) return false;
	try {
		return document.fonts.check(`1em "${family}"`);
	} catch {
		return false;
	}
}

/**
 * Inject the css2 `<link>` for `family` once. No-op when one is already present
 * (covers a stylesheet a previous session/SSR shell rendered) or off-DOM.
 */
function injectStylesheet(family: string): void {
	if (typeof document === 'undefined') return;
	const href = css2Href(family);
	const existing = document.querySelector(`link[data-sbx-font][href="${href}"]`);
	if (existing) return;
	const link = document.createElement('link');
	link.rel = 'stylesheet';
	link.href = href;
	link.setAttribute('data-sbx-font', family);
	document.head.appendChild(link);
}

/**
 * Ensure a Google font `family` is loaded so the browser can paint it.
 *
 * @returns a promise that resolves `true` once a usable face is available, or
 *   `false` when it could not be loaded (SSR / no FontFace set / network/timeout).
 *   Idempotent: repeat calls for the same family share one promise.
 */
export function ensureGoogleFont(family: string): Promise<boolean> {
	const name = family?.trim();
	if (!name) return Promise.resolve(false);

	// SSR / non-DOM — nothing to paint, never throw.
	if (typeof document === 'undefined' || !document.fonts) {
		return Promise.resolve(false);
	}

	// Already requested this session → hand back the shared promise.
	const cached = inflight.get(name);
	if (cached) return cached;

	requested.add(name);

	// Already paintable (brand/system face baked into the page, or a stylesheet
	// from a prior shell) → resolve true without a network stylesheet.
	if (alreadyAvailable(name)) {
		const ready = Promise.resolve(true);
		inflight.set(name, ready);
		return ready;
	}

	injectStylesheet(name);

	// `document.fonts.load` triggers loading of the @font-face matching the spec
	// and resolves when the face(s) are ready; we then re-check availability.
	const promise = document.fonts
		.load(`1em "${name}"`)
		.then(() => (document.fonts ? document.fonts.check(`1em "${name}"`) : false))
		.catch(() => false);

	inflight.set(name, promise);
	return promise;
}

/**
 * Register a family as already-loaded WITHOUT a network fetch — for brand/system
 * faces the host page bundles (e.g. Inter, the BR brand stack). A later
 * {@link ensureGoogleFont} for the same family then resolves `true` immediately
 * and never injects a Google stylesheet.
 */
export function markFontLoaded(family: string): void {
	const name = family?.trim();
	if (!name) return;
	requested.add(name);
	if (!inflight.has(name)) inflight.set(name, Promise.resolve(true));
}

/** True if `ensureGoogleFont`/`markFontLoaded` was already called for `family`. */
export function isFontRequested(family: string): boolean {
	return requested.has(family?.trim());
}

/**
 * Test seam — clear the module's request/in-flight caches so each test starts
 * from a clean slate. NOT for production use.
 */
export function __resetFontCacheForTests(): void {
	requested.clear();
	inflight.clear();
}
