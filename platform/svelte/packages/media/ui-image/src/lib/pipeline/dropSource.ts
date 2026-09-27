// Drag-and-drop source extraction — the single, shared parser every dropzone
// uses so cross-tab/cross-app drags behave identically wherever an ImagePicker
// (or GalleryManager) accepts a drop.
//
// The platform drag payload (DataTransfer) carries several representations of
// the same drop, in priority order:
//   1. dataTransfer.files            — real File objects (OS file drag).
//   2. application/x-image-id        — an INTERNAL gallery-item drag (our own
//                                      reorder/designate protocol). The caller
//                                      short-circuits on this BEFORE touching
//                                      files/urls so an internal reorder never
//                                      turns into a re-upload.
//   3. text/uri-list                 — the canonical cross-tab URL drag (one or
//                                      more URLs, '#'-prefixed comment lines).
//   4. text/html                     — a dragged <img>; we parse its src.
//   5. text/plain                    — last resort; used only if it looks like a
//                                      URL or data-URI.
// Classification splits resolved URLs into http(s) (`urls`, may need a network
// fetch / server import) and `data:` (`dataUris`, decodable offline).

export interface DropSources {
	files: File[];
	urls: string[];
	dataUris: string[];
	internalImageId?: string;
}

// Typed failure so callers can distinguish "couldn't fetch the remote bytes"
// (→ fall back to the server-side import proxy) from a genuine bad input.
export class DropSourceFetchError extends Error {
	readonly url: string;
	constructor(url: string, message: string) {
		super(message);
		this.name = 'DropSourceFetchError';
		this.url = url;
	}
}

const DATA_URI_RE = /^data:image\/[a-z0-9.+-]+;/i;
const HTTP_URL_RE = /^https?:\/\//i;

function isDataUri(s: string): boolean {
	return DATA_URI_RE.test(s.trim());
}

function isHttpUrl(s: string): boolean {
	return HTTP_URL_RE.test(s.trim());
}

// Pull <img src> out of a dragged text/html fragment without a full DOM parse
// dependency — the browser sets this when an <img> is dragged from a page.
function extractImgSrcFromHtml(html: string): string | undefined {
	const match = html.match(/<img[^>]+\bsrc\s*=\s*["']([^"']+)["']/i);
	return match?.[1];
}

// Read every offered representation and classify. Synchronous — no network. The
// caller resolves http(s) URLs to bytes later (sourceToFile) so it can choose
// between a direct fetch and the server import proxy on CORS failure.
export function extractDropSources(e: DragEvent): DropSources {
	const dt = e.dataTransfer;
	const result: DropSources = { files: [], urls: [], dataUris: [] };
	if (!dt) return result;

	// 1. Real files.
	result.files = Array.from(dt.files ?? []);

	// 2. Internal gallery-item drag — AUTHORITATIVE. A gallery tile drag also
	//    causes the browser to auto-attach text/uri-list + text/html (the tile's
	//    <img> src), but an internal id is NEVER an upload intent — that
	//    co-payload is noise. Short-circuit before mining text/* so every
	//    consumer's "is this internal?" check is reliable and the ImagePicker /
	//    GalleryManager never re-upload a dragged library image. (files is always
	//    empty for an internal drag — the tile's dragstart sets only id + plain.)
	const internalId = dt.getData('application/x-image-id');
	if (internalId) {
		result.internalImageId = internalId;
		return result;
	}

	// Files present → the OS file drag wins; don't also mine text/* (browsers
	// sometimes attach a bogus text/plain alongside a real file).
	if (result.files.length > 0) return result;

	// 3. text/uri-list — the canonical multi-URL drag payload.
	const uriList = dt.getData('text/uri-list');
	const candidates: string[] = [];
	if (uriList) {
		for (const raw of uriList.split(/\r?\n/)) {
			const line = raw.trim();
			if (!line || line.startsWith('#')) continue; // '#' = comment per RFC 2483
			candidates.push(line);
		}
	}

	// 4. text/html — a dragged <img>. Only consult when uri-list gave nothing.
	if (candidates.length === 0) {
		const html = dt.getData('text/html');
		if (html) {
			const src = extractImgSrcFromHtml(html);
			if (src) candidates.push(src.trim());
		}
	}

	// 5. text/plain — last resort, only if URL/data-URI shaped.
	if (candidates.length === 0) {
		const plain = dt.getData('text/plain')?.trim();
		if (plain && (isHttpUrl(plain) || isDataUri(plain))) candidates.push(plain);
	}

	for (const c of candidates) {
		if (isDataUri(c)) result.dataUris.push(c);
		else if (isHttpUrl(c)) result.urls.push(c);
	}

	return result;
}

// Decode a data: URI into a File entirely offline — no network. Throws on a
// malformed payload.
function dataUriToFile(dataUri: string, name: string): File {
	const comma = dataUri.indexOf(',');
	if (comma === -1) throw new Error('Malformed data URI');
	const header = dataUri.slice(5, comma); // strip leading "data:"
	const isBase64 = /;base64$/i.test(header);
	const mime = header.replace(/;base64$/i, '') || 'application/octet-stream';
	const data = dataUri.slice(comma + 1);

	let buffer: ArrayBuffer;
	if (isBase64) {
		const binary = atob(data);
		const bytes = new Uint8Array(binary.length);
		for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
		buffer = bytes.buffer;
	} else {
		buffer = new TextEncoder().encode(decodeURIComponent(data)).buffer as ArrayBuffer;
	}
	const blob = new Blob([buffer], { type: mime });
	return new File([blob], name, { type: mime });
}

// Best-effort filename from a URL path (kept for the File label only — the
// pipeline re-derives the real extension from the processed output format).
function filenameFromUrl(url: string, fallback: string): string {
	try {
		const u = new URL(url);
		const last = u.pathname.split('/').filter(Boolean).pop();
		return last && /\.[a-z0-9]+$/i.test(last) ? last : fallback;
	} catch {
		return fallback;
	}
}

// Resolve a single dropped source (a data: URI or an http(s) URL) into a File
// the pipeline can process. data: decodes offline; http(s) is fetched. A failed
// http(s) fetch (CORS / opaque / network) throws DropSourceFetchError so the
// caller can fall back to the server-side import proxy.
export async function sourceToFile(src: string, name = 'dropped-image'): Promise<File> {
	if (isDataUri(src)) {
		return dataUriToFile(src, name);
	}
	if (!isHttpUrl(src)) {
		throw new Error('Unsupported drop source');
	}
	let res: Response;
	try {
		res = await fetch(src, { mode: 'cors', credentials: 'omit' });
	} catch (err) {
		throw new DropSourceFetchError(src, err instanceof Error ? err.message : 'Fetch failed');
	}
	// An opaque response (mode mismatch) or a non-OK status can't be read as
	// usable bytes — surface as a fetch error so the caller imports server-side.
	if (!res.ok || res.type === 'opaque') {
		throw new DropSourceFetchError(src, `Could not fetch image (${res.status || res.type})`);
	}
	const blob = await res.blob();
	if (!blob.type.startsWith('image/')) {
		throw new DropSourceFetchError(src, 'Dropped URL is not an image');
	}
	return new File([blob], filenameFromUrl(src, name), { type: blob.type });
}
