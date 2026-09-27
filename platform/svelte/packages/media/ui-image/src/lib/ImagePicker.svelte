<script lang="ts">
	// ImagePicker — the single, generic pick → process → upload control shared by
	// every consumer (avatars, property cover/gallery, static-content hero). It
	// owns ONLY the interaction (click-to-browse, drag-drop of files OR image URLs
	// from another tab) and the pipeline plumbing; the CALLER owns the surrounding
	// chrome (preview, remove, labels) and decides how to persist via onuploaded.
	//
	// Theme-neutral: no brand colours, no domain logic. It inherits currentColor
	// and reads --color-* custom properties (with neutral fallbacks) exactly like
	// UploadZone/GalleryManager so a consumer styles it by setting those vars.
	//
	// Drop handling (the cross-tab fix — every legacy dropzone read files only):
	//   • files                       → pipeline.
	//   • internal x-image-id drag    → ignored here (no upload); consumers that
	//                                    care (property cover) handle it themselves.
	//   • data: URI                   → decoded offline → pipeline.
	//   • http(s) URL                 → sourceToFile() fetch → pipeline.
	//   • http(s) URL, fetch blocked  → Layer-2: POST {url,…} to the server import
	//                                    proxy (CORS-locked CDNs e.g. Facebook).
	import {
		processImage,
		uploadImage,
		extractDropSources,
		sourceToFile,
		DropSourceFetchError,
		facetMaxDimension,
		type PipelineConfig,
		type ImageFacet,
		type ImageOwner
	} from './pipeline';
	import type { Snippet } from 'svelte';

	interface UploadedImage {
		id: string;
		url: string;
		width: number;
		height: number;
		format: string;
		size: number;
	}

	interface ImagePickerProps {
		owner: ImageOwner;
		ownerId?: string;
		facet: ImageFacet;
		/** Client-side resize ceiling (long edge). Defaults to the facet policy. */
		maxDimension?: number;
		/** Allow selecting/dropping more than one file at a time. */
		multiple?: boolean;
		/** Accept dragged image URLs / data-URIs from other tabs. */
		acceptUrl?: boolean;
		disabled?: boolean;
		label?: string;
		/** Server-side import proxy for CORS-locked hosts (Layer-2 fallback). */
		importEndpoint?: string;
		/** Upload endpoint forwarded to uploadImage (SvelteKit proxy). */
		uploadEndpoint?: string;
		altText?: string;
		onuploaded: (img: UploadedImage) => void;
		onerror?: (msg: string) => void;
		/** Fired when work starts/stops so a consumer can show its own spinner. */
		onbusy?: (busy: boolean) => void;
		/** Optional custom trigger UI; replaces the default drop prompt. */
		children?: Snippet;
		class?: string;
	}

	let {
		owner,
		ownerId,
		facet,
		maxDimension,
		multiple = false,
		acceptUrl = true,
		disabled = false,
		label = 'Upload image',
		importEndpoint = '/api/images/import-url',
		uploadEndpoint = '/api/images',
		altText,
		onuploaded,
		onerror,
		onbusy,
		children,
		class: className = ''
	}: ImagePickerProps = $props();

	let fileInput: HTMLInputElement | undefined = $state();
	let dragActive = $state(false);
	let busyCount = $state(0);

	const busy = $derived(busyCount > 0);

	$effect(() => {
		onbusy?.(busy);
	});

	const pipeline = $derived<PipelineConfig>({
		maxBytes: 10 * 1024 * 1024,
		maxDimension: maxDimension ?? facetMaxDimension(facet),
		quality: 0.85,
		// Property/avatar/hero assets never need transparency; WebP is ~half the
		// bytes of JPEG/PNG for the same visual quality. Mirrors the policy the
		// consumers used inline before unification.
		outputFormat: 'image/webp'
	});

	function fail(msg: string) {
		onerror?.(msg);
	}

	function pick() {
		if (disabled) return;
		fileInput?.click();
	}

	function handleInputChange(e: Event) {
		const input = e.target as HTMLInputElement;
		const files = Array.from(input.files ?? []);
		input.value = '';
		void ingestFiles(files);
	}

	// Validate → resize/compress → upload one File, hand the result to the caller.
	async function processAndUpload(file: File): Promise<void> {
		busyCount += 1;
		try {
			const processed = await processImage(file, pipeline);
			if (!processed.ok || !processed.image) {
				fail(processed.errors[0]?.message ?? 'Could not process image');
				return;
			}
			const uploaded = await uploadImage(processed.image, {
				owner,
				ownerId,
				facet,
				altText,
				endpoint: uploadEndpoint
			});
			if (!uploaded.ok || !uploaded.image) {
				fail(uploaded.error ?? 'Upload failed');
				return;
			}
			onuploaded(uploaded.image);
		} catch {
			fail('Network error during upload. Please try again.');
		} finally {
			busyCount -= 1;
		}
	}

	async function ingestFiles(files: File[]): Promise<void> {
		const batch = multiple ? files : files.slice(0, 1);
		for (const file of batch) {
			await processAndUpload(file);
		}
	}

	// Layer-2 fallback — the server fetches the remote bytes for us (used when a
	// direct browser fetch is blocked by CORS, e.g. Facebook CDN). Codes to the
	// import-url contract; a 404 here is expected until that endpoint integrates.
	async function importViaServer(url: string): Promise<void> {
		busyCount += 1;
		try {
			const res = await fetch(importEndpoint, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					'X-Requested-With': 'XMLHttpRequest'
				},
				credentials: 'include',
				body: JSON.stringify({
					url,
					owner,
					owner_id: ownerId,
					facet,
					alt_text: altText
				})
			});
			if (res.ok) {
				const json = await res.json();
				onuploaded({
					id: json.id,
					url: json.url,
					width: json.width ?? 0,
					height: json.height ?? 0,
					format: json.format ?? '',
					size: json.size ?? 0
				});
				return;
			}
			const body = await res.json().catch(() => null);
			fail(body?.error ?? `Could not import image (${res.status})`);
		} catch {
			fail('Could not import the dropped image URL.');
		} finally {
			busyCount -= 1;
		}
	}

	// Resolve a dropped URL/data-URI to a File and run the pipeline; on an http(s)
	// fetch failure (CORS/opaque) fall back to the server import proxy.
	async function ingestUrl(src: string): Promise<void> {
		try {
			const file = await sourceToFile(src);
			await processAndUpload(file);
		} catch (err) {
			if (err instanceof DropSourceFetchError) {
				await importViaServer(src);
			} else {
				fail(err instanceof Error ? err.message : 'Could not read the dropped image.');
			}
		}
	}

	function handleDragOver(e: DragEvent) {
		if (disabled) return;
		// An internal gallery-item drag is an assign/reorder owned by the host
		// wrapper (e.g. the property cover), not an upload here. Still allow the
		// drop (preventDefault) but skip our "Drop to upload" cue so an internal
		// drag over the picker never looks like a pending upload.
		if (e.dataTransfer?.types.includes('application/x-image-id')) {
			e.preventDefault();
			return;
		}
		e.preventDefault();
		dragActive = true;
	}

	function handleDragLeave(e: DragEvent) {
		if (e.currentTarget === e.target) dragActive = false;
	}

	async function handleDrop(e: DragEvent) {
		e.preventDefault();
		dragActive = false;
		if (disabled) return;

		const sources = extractDropSources(e);

		// Internal gallery-item drag — not an upload here. extractDropSources
		// makes the id authoritative (it drops the browser's auto-attached img
		// payload), so this bail is reliable: a library tile dropped on the picker
		// is an assign/reorder the host wrapper owns, never a re-upload.
		if (sources.internalImageId) {
			return;
		}

		if (sources.files.length > 0) {
			await ingestFiles(sources.files);
			return;
		}

		if (!acceptUrl) return;

		const urlSources = [...sources.dataUris, ...sources.urls];
		const queue = multiple ? urlSources : urlSources.slice(0, 1);
		for (const src of queue) {
			await ingestUrl(src);
		}
	}
</script>

<div
	class="image-picker {className}"
	class:is-dragover={dragActive}
	class:is-disabled={disabled}
	class:is-busy={busy}
	role="button"
	tabindex={disabled ? -1 : 0}
	aria-label={label}
	aria-busy={busy}
	ondragover={handleDragOver}
	ondragleave={handleDragLeave}
	ondrop={handleDrop}
	onclick={pick}
	onkeydown={(e) => {
		if (!disabled && (e.key === 'Enter' || e.key === ' ')) {
			e.preventDefault();
			pick();
		}
	}}
>
	<!-- Keep the input rendered (not [hidden]/display:none) so programmatic
	     .click() reliably opens the OS picker and honours `multiple`. -->
	<input
		bind:this={fileInput}
		type="file"
		accept="image/*"
		{multiple}
		class="image-picker__input"
		{disabled}
		onchange={handleInputChange}
	/>

	{#if children}
		{@render children()}
	{:else}
		<div class="image-picker__prompt">
			<svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
				<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
				<polyline points="17 8 12 3 7 8" />
				<line x1="12" y1="3" x2="12" y2="15" />
			</svg>
			<p class="image-picker__text">
				{#if dragActive}
					Drop to upload
				{:else if acceptUrl}
					Drop an image or URL here, or <strong>click to browse</strong>
				{:else}
					Drop an image here, or <strong>click to browse</strong>
				{/if}
			</p>
		</div>
	{/if}
</div>

<style>
	.image-picker {
		position: relative;
		display: flex;
		align-items: center;
		justify-content: center;
		border: 2px dashed var(--color-border, currentColor);
		border-radius: var(--radius-md, 8px);
		padding: 1.5rem;
		text-align: center;
		cursor: pointer;
		color: inherit;
		transition: border-color 0.15s ease, background 0.15s ease;
	}
	.image-picker:hover,
	.image-picker:focus-visible {
		border-color: var(--color-accent, currentColor);
		outline: none;
	}
	.image-picker.is-dragover {
		border-color: var(--color-accent, currentColor);
		background: color-mix(in srgb, var(--color-accent, currentColor) 8%, transparent);
	}
	.image-picker.is-disabled {
		opacity: 0.5;
		pointer-events: none;
	}
	.image-picker.is-busy {
		cursor: progress;
	}
	/* Visually-hidden but kept in the tree so .click() works reliably. */
	.image-picker__input {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		margin: -1px;
		overflow: hidden;
		clip: rect(0, 0, 0, 0);
		white-space: nowrap;
		border: 0;
	}
	.image-picker__prompt {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.4rem;
		color: inherit;
		opacity: 0.85;
	}
	.image-picker__text {
		margin: 0;
		font-size: 0.8125rem;
	}
</style>
