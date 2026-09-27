<script lang="ts">
	import type { ImageFacet, ImageOwner, PipelineConfig, PipelineResult, ProcessedImage } from './pipeline/types';
	import { processImage } from './pipeline/pipeline';
	import { uploadImage } from './pipeline/upload';

	interface UploadFile {
		id: string;
		file: File;
		status: 'queued' | 'processing' | 'uploading' | 'done' | 'error';
		progress: number;
		error?: string;
		result?: { url: string; id: string };
	}

	interface UploadZoneProps {
		pipeline: PipelineConfig;
		// Owner-scoped key inputs forwarded to uploadImage (Decision #0271). The
		// consumer that mounts the zone knows which entity owns the asset, so it
		// supplies owner + ownerId; the zone never invents a 'general' fallback.
		owner: ImageOwner;
		ownerId?: string; // required for every owner except 'system'
		facet: ImageFacet;
		uploadUrl?: string;
		accept?: string;
		maxFiles?: number;
		disabled?: boolean;
		onupload?: (results: { url: string; id: string }[]) => void;
		onerror?: (errors: { file: string; message: string }[]) => void;
		class?: string;
	}

	let {
		pipeline,
		owner,
		ownerId,
		facet,
		uploadUrl = '/api/images',
		accept = 'image/*',
		maxFiles = 20,
		disabled = false,
		onupload,
		onerror,
		class: className = ''
	}: UploadZoneProps = $props();

	let files = $state<UploadFile[]>([]);
	let dragOver = $state(false);
	let inputEl: HTMLInputElement;

	let isProcessing = $derived(files.some(f => f.status === 'processing' || f.status === 'uploading'));
	let completedFiles = $derived(files.filter(f => f.status === 'done'));
	let errorFiles = $derived(files.filter(f => f.status === 'error'));

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		if (!disabled) dragOver = true;
	}

	function handleDragLeave() {
		dragOver = false;
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		dragOver = false;
		if (disabled) return;
		const dropped = Array.from(e.dataTransfer?.files ?? []);
		addFiles(dropped);
	}

	function handleInputChange(e: Event) {
		const input = e.target as HTMLInputElement;
		const selected = Array.from(input.files ?? []);
		addFiles(selected);
		input.value = '';
	}

	function addFiles(incoming: File[]) {
		const remaining = maxFiles - files.length;
		if (remaining <= 0) return;

		const sorted = [...incoming].sort((a, b) => a.name.localeCompare(b.name, undefined, { numeric: true }));

		const batch = sorted.slice(0, remaining).map(file => ({
			id: crypto.randomUUID(),
			file,
			status: 'queued' as const,
			progress: 0
		}));

		files = [...files, ...batch];
		processBatch(batch);
	}

	async function processBatch(batch: UploadFile[]) {
		const batchIds = new Set(batch.map(b => b.id));

		for (const item of batch) {
			const idx = files.findIndex(f => f.id === item.id);
			if (idx === -1) continue;

			files[idx] = { ...files[idx], status: 'processing', progress: 10 };
			files = [...files];

			const pipelineResult: PipelineResult = await processImage(item.file, pipeline);

			if (!pipelineResult.ok || !pipelineResult.image) {
				const msg = pipelineResult.errors[0]?.message ?? 'Processing failed';
				files[idx] = { ...files[idx], status: 'error', error: msg };
				files = [...files];
				continue;
			}

			files[idx] = { ...files[idx], status: 'uploading', progress: 50 };
			files = [...files];

			const uploadResult = await uploadImage(pipelineResult.image, { owner, ownerId, facet, endpoint: uploadUrl });

			if (!uploadResult.ok) {
				files[idx] = { ...files[idx], status: 'error', error: uploadResult.error ?? 'Upload failed' };
				files = [...files];
				continue;
			}

			files[idx] = {
				...files[idx],
				status: 'done',
				progress: 100,
				result: { url: uploadResult.image!.url, id: uploadResult.image!.id }
			};
			files = [...files];
		}

		const batchSuccesses = files
			.filter(f => batchIds.has(f.id) && f.status === 'done' && f.result)
			.map(f => f.result!);
		const batchErrors = files
			.filter(f => batchIds.has(f.id) && f.status === 'error')
			.map(f => ({ file: f.file.name, message: f.error ?? 'Unknown error' }));

		if (batchSuccesses.length > 0) onupload?.(batchSuccesses);
		if (batchErrors.length > 0) onerror?.(batchErrors);
	}

	function removeFile(id: string) {
		files = files.filter(f => f.id !== id);
	}

	function clear() {
		files = [];
	}
</script>

<div
	class="upload-zone {className}"
	class:drag-over={dragOver}
	class:disabled
	role="button"
	tabindex="0"
	ondragover={handleDragOver}
	ondragleave={handleDragLeave}
	ondrop={handleDrop}
	onclick={(e) => { if (!disabled && !(e.target as HTMLElement).closest('.upload-zone__list')) inputEl?.click(); }}
	onkeydown={(e) => e.key === 'Enter' && !disabled && inputEl?.click()}
>
	<input
		bind:this={inputEl}
		type="file"
		{accept}
		multiple
		hidden
		onchange={handleInputChange}
	/>

	{#if files.length === 0}
		<div class="upload-zone__prompt">
			<svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
				<polyline points="17 8 12 3 7 8" />
				<line x1="12" y1="3" x2="12" y2="15" />
			</svg>
			<p class="upload-zone__text">Drop images here or click to browse</p>
			<p class="upload-zone__hint">Max {maxFiles} files, up to {Math.round(pipeline.maxBytes / 1024 / 1024)}MB each</p>
		</div>
	{:else}
		<div class="upload-zone__list">
			{#each files as f (f.id)}
				<div class="upload-item" class:upload-item--error={f.status === 'error'} class:upload-item--done={f.status === 'done'}>
					<span class="upload-item__name">{f.file.name}</span>
					<span class="upload-item__status">
						{#if f.status === 'processing'}
							Processing…
						{:else if f.status === 'uploading'}
							Uploading…
						{:else if f.status === 'done'}
							✓
						{:else if f.status === 'error'}
							{f.error}
						{/if}
					</span>
					<button class="upload-item__remove" onclick={() => removeFile(f.id)} aria-label="Remove">×</button>
				</div>
			{/each}
		</div>
	{/if}
</div>

{#if files.length > 0}
	<div class="upload-zone__footer">
		<span class="upload-zone__count">{completedFiles.length}/{files.length} uploaded</span>
		{#if !isProcessing}
			<button class="upload-zone__clear" onclick={clear}>Clear all</button>
		{/if}
	</div>
{/if}

<style>
	.upload-zone {
		border: 2px dashed var(--color-border, #333);
		border-radius: var(--radius-lg, 12px);
		padding: 2rem;
		text-align: center;
		cursor: pointer;
		transition: border-color 0.2s, background 0.2s;
		background: var(--color-bg-secondary, #1a1a1b);
	}

	.upload-zone:hover,
	.upload-zone.drag-over {
		border-color: var(--color-accent, #6366f1);
		background: color-mix(in srgb, var(--color-accent, #6366f1) 5%, transparent);
	}

	.upload-zone.disabled {
		opacity: 0.5;
		pointer-events: none;
	}

	.upload-zone__prompt {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.5rem;
		color: var(--color-text-muted, #888);
	}

	.upload-zone__text {
		font-size: 0.875rem;
		margin: 0;
	}

	.upload-zone__hint {
		font-size: 0.75rem;
		opacity: 0.6;
		margin: 0;
	}

	.upload-zone__list {
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
		text-align: left;
		max-height: 200px;
		overflow-y: auto;
	}

	.upload-item {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.375rem 0.5rem;
		border-radius: var(--radius-sm, 4px);
		background: var(--color-bg-tertiary, #222);
		font-size: 0.75rem;
	}

	.upload-item--done {
		border-left: 3px solid var(--color-success, #22c55e);
	}

	.upload-item--error {
		border-left: 3px solid var(--color-error, #ef4444);
	}

	.upload-item__name {
		flex: 1;
		word-break: break-all;
		color: var(--color-text, #eee);
	}

	.upload-item__status {
		font-size: 0.6875rem;
		color: var(--color-text-muted, #888);
		white-space: nowrap;
	}

	.upload-item--error .upload-item__status {
		color: var(--color-error, #ef4444);
	}

	.upload-item__remove {
		background: none;
		border: none;
		color: var(--color-text-muted, #888);
		cursor: pointer;
		font-size: 1rem;
		line-height: 1;
		padding: 0 0.25rem;
	}

	.upload-item__remove:hover {
		color: var(--color-error, #ef4444);
	}

	.upload-zone__footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-top: 0.5rem;
		font-size: 0.75rem;
		color: var(--color-text-muted, #888);
	}

	.upload-zone__clear {
		background: none;
		border: none;
		color: var(--color-text-muted, #888);
		cursor: pointer;
		font-size: 0.75rem;
		text-decoration: underline;
	}

	.upload-zone__clear:hover {
		color: var(--color-text, #eee);
	}
</style>
