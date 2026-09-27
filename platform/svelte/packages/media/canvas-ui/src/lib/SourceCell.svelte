<script lang="ts">
	// One attached-source cell (extracted from the dissolved SettingsPanel so the new
	// Sources panel reuses it — never-copy-paste-CSS). Shows the record's HUMAN title
	// from the resolved snapshot (the raw refId/UUID is useless to the user) + a cover
	// thumbnail for recognition; the alias·kind is the slot eyebrow. The right-aligned
	// (hard rule) actions are compact icon buttons — ⇄ change (re-pick the record,
	// same alias) and the detach ✕ — pinned to the top of the row. The title is capped
	// at two lines (user call 2026-06-06: "proper table with limited lines" — full
	// title lives in the tooltip).
	import { Icon } from '@sbx/core-ui/components/primitives';
	import type { SourceRef } from '@sbx/canvas-kit';
	import ThumbImage from './ThumbImage.svelte';
	import { sourceTitle } from './source-display.js';

	interface Props {
		source: SourceRef;
		/** Re-pick the record for this alias (keeps every binding — same-alias replace). */
		onchange?: (alias: string) => void;
		/** Detach this source by alias (its bindings then fall back). */
		ondetach?: (alias: string) => void;
	}

	let { source, onchange, ondetach }: Props = $props();

	function asUrl(v: unknown): string | undefined {
		return typeof v === 'string' && v.trim() ? v : undefined;
	}

	/** A cover thumbnail from the resolved snapshot — property emits `images.cover[]`; the content
	 *  kinds (guide/service/news) emit `coverImage.url`. A plain <img> (not canvas-drawn) → no proxy. */
	function coverThumb(snapshot: SourceRef['snapshot']): string | undefined {
		if (!snapshot) return undefined;
		const images = snapshot.images;
		if (images && typeof images === 'object') {
			const cover = (images as Record<string, unknown>).cover;
			if (Array.isArray(cover) && cover[0] && typeof cover[0] === 'object') {
				const first = cover[0] as Record<string, unknown>;
				// Thumbnail role (task 2606-001) — mirrors pickImageVariant's fallback for
				// this untyped cover record: thumbnail → legacy thumb → medium → url.
				return (
					asUrl(first.thumbnail) ?? asUrl(first.thumb) ?? asUrl(first.medium) ?? asUrl(first.url)
				);
			}
		}
		const coverImage = snapshot.coverImage;
		if (coverImage && typeof coverImage === 'object') {
			return asUrl((coverImage as Record<string, unknown>).url);
		}
		return undefined;
	}

	const thumb = $derived(coverThumb(source.snapshot));
</script>

<li class="cell">
	<span class="cell__thumb">
		{#if thumb}
			<ThumbImage src={thumb} />
		{:else}
			<Icon name="image" size="sm" />
		{/if}
	</span>
	<span class="cell__text">
		<span class="cell__eyebrow">
			{#if source.alias === source.kind}
				<span class="cell__alias">{source.kind}</span>
			{:else}
				<span class="cell__alias">{source.alias}</span> · {source.kind}
			{/if}
		</span>
		<span class="cell__title" title={sourceTitle(source)}>{sourceTitle(source)}</span>
	</span>
	<span class="cell__actions">
		{#if onchange}
			<button
				type="button"
				class="cell__change"
				aria-label={`Change ${source.alias} source`}
				title="Change source"
				onclick={() => onchange?.(source.alias)}
			>
				<Icon name="replace" size="xs" />
			</button>
		{/if}
		<button
			type="button"
			class="cell__remove"
			aria-label={`Detach ${source.alias} source`}
			title="Detach source"
			onclick={() => ondetach?.(source.alias)}
		>
			<Icon name="x" size="xs" />
		</button>
	</span>
</li>

<style>
	/* Compact source row: thumb + (alias·kind eyebrow / 2-line title) + icon actions.
	   align-items flex-start so the thumb + actions pin to the top of the row. */
	.cell {
		display: flex;
		align-items: flex-start;
		gap: var(--cv-space-sm, 0.5rem);
		padding: 0.375rem 0.5rem;
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: var(--cv-color-neutral-50, #fafafa);
	}

	.cell__thumb {
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		width: 2.25rem;
		height: 2.25rem;
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: var(--cv-color-neutral-100, #f5f5f5);
		color: var(--cv-color-neutral-400, #a0a0a0);
		overflow: hidden;
	}

	.cell__text {
		display: flex;
		flex-direction: column;
		gap: 0.0625rem;
		min-width: 0;
		flex: 1;
	}

	.cell__eyebrow {
		font-size: 0.6875rem;
		color: var(--cv-color-neutral-500, #707070);
	}

	.cell__alias {
		font-weight: 700;
		color: var(--cv-color-primary, #333333);
	}

	/* The record's human title — capped at TWO lines (user call 2026-06-06: limited
	   lines for a proper table; overrides the default no-clip stance for this row).
	   The full title is always reachable via the tooltip. */
	.cell__title {
		display: -webkit-box;
		-webkit-box-orient: vertical;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		overflow: hidden;
		font-size: 0.8125rem;
		line-height: 1.3;
		color: var(--cv-color-neutral-800, #202020);
		overflow-wrap: anywhere;
	}

	/* Actions group — Change + detach, right-aligned (hard rule); pinned to the top of a
	   multi-line cell so they never drift down beside a wrapped title. */
	.cell__actions {
		display: flex;
		align-items: flex-start;
		gap: 0.25rem;
		flex-shrink: 0;
		margin-left: auto;
	}

	/* Change — icon-only (user call 2026-06-06: the "Change" text was bulk; the ⇄
	   glyph + tooltip carry the meaning). Same hit target as the detach ✕. */
	.cell__change {
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		width: 1.25rem;
		height: 1.25rem;
		border: none;
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: transparent;
		color: var(--cv-color-neutral-500, #707070);
		cursor: pointer;
	}

	.cell__change:hover {
		background: var(--cv-color-neutral-200, #e8e8e8);
		color: var(--cv-color-primary, #333333);
	}

	.cell__change:focus-visible {
		outline: 2px solid var(--cv-color-primary, #333333);
		outline-offset: 1px;
	}

	/* Detach control; pinned to the top of a multi-line cell. */
	.cell__remove {
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		width: 1.25rem;
		height: 1.25rem;
		border: none;
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: transparent;
		color: var(--cv-color-neutral-500, #707070);
		cursor: pointer;
	}

	.cell__remove:hover {
		background: var(--cv-color-neutral-200, #e8e8e8);
		color: var(--cv-color-error, #c0392b);
	}

	.cell__remove:focus-visible {
		outline: 2px solid var(--cv-color-primary, #333333);
		outline-offset: 1px;
	}
</style>
