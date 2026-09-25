import { describe, it, expect } from 'vitest';
import { extractDropSources, sourceToFile, DropSourceFetchError } from './dropSource';

// Minimal DataTransfer/DragEvent stand-in — jsdom-free unit harness. Only the
// surface extractDropSources reads (files, types, getData) is modelled.
function makeDragEvent(opts: {
	files?: File[];
	data?: Record<string, string>;
}): DragEvent {
	const data = opts.data ?? {};
	const files = opts.files ?? [];
	const dt = {
		files,
		types: Object.keys(data).concat(files.length > 0 ? ['Files'] : []),
		getData: (type: string) => data[type] ?? ''
	};
	return { dataTransfer: dt } as unknown as DragEvent;
}

function fakeFile(name = 'a.png'): File {
	return new File([new Uint8Array([1, 2, 3])], name, { type: 'image/png' });
}

describe('extractDropSources', () => {
	it('returns files when the drop carries OS files', () => {
		const f = fakeFile();
		const res = extractDropSources(makeDragEvent({ files: [f] }));
		expect(res.files).toEqual([f]);
		expect(res.urls).toEqual([]);
		expect(res.dataUris).toEqual([]);
	});

	it('files win over text payloads', () => {
		const f = fakeFile();
		const res = extractDropSources(
			makeDragEvent({ files: [f], data: { 'text/uri-list': 'https://x.com/a.jpg' } })
		);
		expect(res.files).toEqual([f]);
		expect(res.urls).toEqual([]);
	});

	it('captures the internal gallery-item id', () => {
		const res = extractDropSources(
			makeDragEvent({ data: { 'application/x-image-id': 'img-123' } })
		);
		expect(res.internalImageId).toBe('img-123');
	});

	it('treats an internal id as authoritative, dropping the browser auto-attached img payload', () => {
		// A gallery tile drag sets application/x-image-id AND the browser
		// auto-attaches text/uri-list + text/html (the tile's <img> src). The id
		// must win so a dropzone never re-uploads a dragged library image — the
		// rail→cover "uploading progress, nothing happened" bug.
		const res = extractDropSources(
			makeDragEvent({
				data: {
					'application/x-image-id': 'img-123',
					'text/uri-list': 'https://cdn.example.com/img-123.webp',
					'text/html': '<img src="https://cdn.example.com/img-123.webp">'
				}
			})
		);
		expect(res.internalImageId).toBe('img-123');
		expect(res.urls).toEqual([]);
		expect(res.dataUris).toEqual([]);
		expect(res.files).toEqual([]);
	});

	it('parses text/uri-list, ignoring comment lines', () => {
		const res = extractDropSources(
			makeDragEvent({
				data: { 'text/uri-list': '# comment\r\nhttps://cdn.example.com/p.jpg\r\n' }
			})
		);
		expect(res.urls).toEqual(['https://cdn.example.com/p.jpg']);
	});

	it('classifies a data: URI into dataUris', () => {
		const uri = 'data:image/png;base64,AAAA';
		const res = extractDropSources(makeDragEvent({ data: { 'text/uri-list': uri } }));
		expect(res.dataUris).toEqual([uri]);
		expect(res.urls).toEqual([]);
	});

	it('falls back to <img src> in text/html', () => {
		const res = extractDropSources(
			makeDragEvent({
				data: { 'text/html': '<div><img src="https://x.com/y.png" alt=""></div>' }
			})
		);
		expect(res.urls).toEqual(['https://x.com/y.png']);
	});

	it('falls back to text/plain only when URL-shaped', () => {
		expect(
			extractDropSources(makeDragEvent({ data: { 'text/plain': 'https://x.com/z.webp' } })).urls
		).toEqual(['https://x.com/z.webp']);
		expect(
			extractDropSources(makeDragEvent({ data: { 'text/plain': 'just some text' } })).urls
		).toEqual([]);
	});

	it('returns empty result with no dataTransfer', () => {
		const res = extractDropSources({ dataTransfer: null } as unknown as DragEvent);
		expect(res.files).toEqual([]);
		expect(res.urls).toEqual([]);
	});
});

describe('sourceToFile', () => {
	it('decodes a base64 data URI offline (no network)', async () => {
		// "hi" base64 → aGk=
		const file = await sourceToFile('data:image/png;base64,aGk=', 'pic.png');
		expect(file).toBeInstanceOf(File);
		expect(file.type).toBe('image/png');
		expect(await file.text()).toBe('hi');
	});

	it('throws on an unsupported (non-url, non-data) source', async () => {
		await expect(sourceToFile('ftp://nope')).rejects.toThrow('Unsupported drop source');
	});

	it('wraps a fetch failure as DropSourceFetchError', async () => {
		const original = globalThis.fetch;
		globalThis.fetch = (() => Promise.reject(new Error('CORS'))) as typeof fetch;
		try {
			await expect(sourceToFile('https://blocked.example.com/a.jpg')).rejects.toBeInstanceOf(
				DropSourceFetchError
			);
		} finally {
			globalThis.fetch = original;
		}
	});

	it('rejects an opaque/non-ok response as a fetch error', async () => {
		const original = globalThis.fetch;
		globalThis.fetch = (() =>
			Promise.resolve({ ok: false, status: 0, type: 'opaque' } as Response)) as typeof fetch;
		try {
			await expect(sourceToFile('https://blocked.example.com/a.jpg')).rejects.toBeInstanceOf(
				DropSourceFetchError
			);
		} finally {
			globalThis.fetch = original;
		}
	});

	it('builds a File from a CORS-permissive image response', async () => {
		const original = globalThis.fetch;
		globalThis.fetch = (() =>
			Promise.resolve({
				ok: true,
				type: 'cors',
				status: 200,
				blob: () => Promise.resolve(new Blob([new Uint8Array([1])], { type: 'image/jpeg' }))
			} as unknown as Response)) as typeof fetch;
		try {
			const file = await sourceToFile('https://ok.example.com/photo.jpg');
			expect(file.type).toBe('image/jpeg');
			expect(file.name).toBe('photo.jpg');
		} finally {
			globalThis.fetch = original;
		}
	});

	it('rejects a non-image response', async () => {
		const original = globalThis.fetch;
		globalThis.fetch = (() =>
			Promise.resolve({
				ok: true,
				type: 'cors',
				status: 200,
				blob: () => Promise.resolve(new Blob(['<html>'], { type: 'text/html' }))
			} as unknown as Response)) as typeof fetch;
		try {
			await expect(sourceToFile('https://ok.example.com/not.png')).rejects.toBeInstanceOf(
				DropSourceFetchError
			);
		} finally {
			globalThis.fetch = original;
		}
	});
});
