<script lang="ts">
	/**
	 * PromptDocumentEditor — a compact, flat editor for a prompt "document".
	 *
	 * The document is an ordered list of sections, each a title + a body of plain text.
	 * The whole stack reads like ONE continuous document: tight vertical rhythm, a thin
	 * separator between sections, no large gaps and no card chrome — not a grid of boxed
	 * cards. Each section has add / delete / move-up / move-down / hide-show controls,
	 * and a preview affordance toggles the rendered output (XML or Markdown).
	 *
	 * This is the lean v1 core extracted from the Configurator card surface: title + text
	 * ONLY — no presets, no nesting/children, no variables panel.
	 *
	 * The CALLER owns persistence. The component is fully controlled: it renders the
	 * `blocks` it is given and emits the next array through `onchange` on every edit. It
	 * never autosaves, never fetches, never holds a store — the host decides what to do
	 * with each change.
	 *
	 * Theme via the `--prompt-doc-*` custom properties (neutral, dark-default; chains to
	 * the shared `--color-*` tokens). A light product remaps them on the wrapper.
	 *
	 * @layer section
	 * @uses Input (primitive) · Textarea (primitive) · IconButton (primitive) · Button (primitive)
	 *
	 * @example
	 *   <PromptDocumentEditor {blocks} onchange={(next) => (blocks = next)} />
	 */

	import Input from '../primitives/Input.svelte';
	import Textarea from '../primitives/Textarea.svelte';
	import IconButton from '../primitives/IconButton.svelte';
	import Button from '../primitives/Button.svelte';
	import {
		renderPromptDocument,
		type PromptBlock,
		type PromptDocumentFormat
	} from './PromptDocument';

	interface Props {
		/** The sections to edit (controlled — the host owns the array). */
		blocks: PromptBlock[];
		/** Read-only mode: inputs disabled, no add/delete/reorder/hide controls. */
		readonly?: boolean;
		/** Placeholder for an empty section title. */
		titlePlaceholder?: string;
		/** Placeholder for an empty section body. */
		bodyPlaceholder?: string;
		/** Emitted with the NEXT array on every edit. The caller persists. */
		onchange?: (blocks: PromptBlock[]) => void;
		/** Additional CSS classes on the root. */
		class?: string;
		/** IView: Data attributes for dev mode. */
		'data-view-id'?: string;
	}

	let {
		blocks,
		readonly = false,
		titlePlaceholder = 'Section title',
		bodyPlaceholder = 'Write this section…',
		onchange,
		class: className = '',
		'data-view-id': viewId
	}: Props = $props();

	// Preview state — local view concern, never persisted.
	let previewOpen = $state(false);
	let previewFormat = $state<PromptDocumentFormat>('xml');
	let previewText = $derived(renderPromptDocument(blocks, previewFormat));

	// Every mutation builds a NEW array (immutable update) and hands it to the host; the
	// component itself stays controlled and re-renders from the prop the host feeds back.
	function commit(next: PromptBlock[]) {
		onchange?.(next);
	}

	function setTitle(index: number, name: string) {
		commit(blocks.map((b, i) => (i === index ? { ...b, name } : b)));
	}
	function setBody(index: number, text: string) {
		commit(blocks.map((b, i) => (i === index ? { ...b, text } : b)));
	}
	function toggleHidden(index: number) {
		commit(blocks.map((b, i) => (i === index ? { ...b, hidden: !b.hidden } : b)));
	}
	function add() {
		commit([...blocks, { name: '', text: '' }]);
	}
	function remove(index: number) {
		commit(blocks.filter((_, i) => i !== index));
	}
	function move(from: number, to: number) {
		if (to < 0 || to >= blocks.length) return;
		const next = [...blocks];
		[next[from], next[to]] = [next[to], next[from]];
		commit(next);
	}
</script>

<section class="prompt-doc {className}" data-view-id={viewId}>
	<div class="prompt-doc__list">
		{#each blocks as block, i (i)}
			<article class="prompt-doc__section" class:is-hidden={block.hidden}>
				<header class="prompt-doc__head">
					<div class="prompt-doc__title">
						<Input
							value={block.name}
							placeholder={titlePlaceholder}
							variant="ghost"
							size="md"
							{readonly}
							aria-label="Section title"
							oninput={(v) => setTitle(i, v)}
						/>
					</div>

					{#if !readonly}
						<!-- Per-section controls. Right-aligned (CTA convention). The whole
						     group fades to quiet until the section is hovered/focused so the
						     stack stays calm and document-like. -->
						<div class="prompt-doc__controls">
							<IconButton
								icon={block.hidden ? 'circle-dashed' : 'check-circle'}
								label={block.hidden ? 'Include this section' : 'Exclude this section'}
								title={block.hidden ? 'Include this section' : 'Exclude this section'}
								variant="ghost"
								size="sm"
								onclick={() => toggleHidden(i)}
							/>
							<IconButton
								icon="chevron-up"
								label="Move section up"
								title="Move up"
								variant="ghost"
								size="sm"
								disabled={i === 0}
								onclick={() => move(i, i - 1)}
							/>
							<IconButton
								icon="chevron-down"
								label="Move section down"
								title="Move down"
								variant="ghost"
								size="sm"
								disabled={i === blocks.length - 1}
								onclick={() => move(i, i + 1)}
							/>
							<IconButton
								icon="trash-2"
								label="Remove section"
								title="Remove"
								variant="ghost"
								size="sm"
								onclick={() => remove(i)}
							/>
						</div>
					{/if}
				</header>

				<div class="prompt-doc__body">
					<Textarea
						value={block.text}
						placeholder={bodyPlaceholder}
						rows={3}
						{readonly}
						aria-label="Section text"
						oninput={(v) => setBody(i, v)}
					/>
				</div>
			</article>
		{/each}

		{#if blocks.length === 0}
			<p class="prompt-doc__empty">No sections yet.</p>
		{/if}
	</div>

	{#if !readonly}
		<!-- Add a section — left-aligned, low-key (it grows the document inline, it is not
		     a primary page action). -->
		<div class="prompt-doc__add">
			<Button variant="ghost" size="sm" icon="+" onclick={add}>Add section</Button>
		</div>
	{/if}

	<!-- Preview affordance: a quiet toggle that reveals the rendered document. The
	     format switch (XML | Markdown) sits with it; CTAs right-aligned. -->
	<footer class="prompt-doc__preview">
		<div class="prompt-doc__preview-bar">
			<button
				type="button"
				class="prompt-doc__preview-toggle"
				aria-expanded={previewOpen}
				onclick={() => (previewOpen = !previewOpen)}
			>
				{previewOpen ? 'Hide preview' : 'Show preview'}
			</button>

			{#if previewOpen}
				<div class="prompt-doc__formats" role="group" aria-label="Preview format">
					<button
						type="button"
						class="prompt-doc__format"
						class:is-active={previewFormat === 'xml'}
						aria-pressed={previewFormat === 'xml'}
						onclick={() => (previewFormat = 'xml')}
					>
						XML
					</button>
					<button
						type="button"
						class="prompt-doc__format"
						class:is-active={previewFormat === 'markdown'}
						aria-pressed={previewFormat === 'markdown'}
						onclick={() => (previewFormat = 'markdown')}
					>
						Markdown
					</button>
				</div>
			{/if}
		</div>

		{#if previewOpen}
			<pre class="prompt-doc__preview-out">{previewText}</pre>
		{/if}
	</footer>
</section>

<style>
	/* Tokens — neutral, dark-default; each chains to the shared --color-* set so a
	   consumer remaps either the --prompt-doc-* knob or the underlying --color-* token.
	   No hard-coded brand colours. */
	.prompt-doc {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		width: 100%;
		color: var(--prompt-doc-text, var(--color-text, #e5e7eb));
	}

	/* The list IS the document: one bordered surface, sections stacked flush and
	   divided by a thin separator. No per-section card, no gap between sections —
	   tight, continuous, document-like density. */
	.prompt-doc__list {
		display: flex;
		flex-direction: column;
		border: 1px solid var(--prompt-doc-border, var(--color-border, #2a2a4a));
		border-radius: var(--prompt-doc-radius, 0.5rem);
		background: var(--prompt-doc-bg, var(--color-bg-secondary, #1a1a2e));
		overflow: hidden;
	}

	.prompt-doc__section {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		padding: 0.5rem 0.625rem 0.625rem;
	}

	/* The thin separator between consecutive sections — the document's rhythm. */
	.prompt-doc__section + .prompt-doc__section {
		border-top: 1px solid var(--prompt-doc-separator, var(--color-border, #2a2a4a));
	}

	/* An excluded section stays fully readable (never clipped) but visually recedes so
	   it's clear it won't render into the prompt. */
	.prompt-doc__section.is-hidden {
		opacity: 0.55;
	}

	.prompt-doc__head {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	/* The title input grows; its ghost variant has no box so the title reads as a
	   heading on the document surface, not a form field. */
	.prompt-doc__title {
		flex: 1 1 auto;
		min-width: 0;
	}
	.prompt-doc__title :global(.input-field) {
		font-weight: 600;
		font-size: 0.9375rem;
		padding-left: 0.25rem;
		padding-right: 0.25rem;
	}

	/* Controls sit right (CTA convention) and stay quiet until the section is engaged,
	   keeping the resting stack calm. Reveal on hover/focus-within of the section. */
	.prompt-doc__controls {
		display: flex;
		align-items: center;
		gap: 0.125rem;
		flex: 0 0 auto;
		opacity: 0.35;
		transition: opacity 0.15s ease;
	}
	.prompt-doc__section:hover .prompt-doc__controls,
	.prompt-doc__section:focus-within .prompt-doc__controls {
		opacity: 1;
	}

	/* The body textarea sheds ALL field chrome so it reads as prose flowing under its
	   title, not a boxed input — the whole point of the one-document feel.

	   Robustness: a host themes the shared field via the --field-* set (BR maps them to
	   its brand), and the primitive's own `.field-control{,:hover,:focus,:read-only}`
	   rules paint a border / background / focus-ring / readonly fill from those tokens.
	   Matching BOTH classes (`.field-control.field-textarea`) keeps this override a hair
	   more specific than every one of those single-class primitive rules, so the body
	   stays flush no matter the host's tokens or stylesheet order — we don't merely
	   neutralise today's border-colour, we zero the entire box (background, border,
	   shadow, outline) in every interaction state. Padding collapses to the section's own
	   left inset so the prose sits directly beneath the ghost-heading title; resize is
	   dropped (its corner grip is the last form-field tell). Full width, never clipped. */
	.prompt-doc__body :global(.field-control.field-textarea),
	.prompt-doc__body :global(.field-control.field-textarea:hover),
	.prompt-doc__body :global(.field-control.field-textarea:focus),
	.prompt-doc__body :global(.field-control.field-textarea:read-only) {
		background: transparent;
		border-color: transparent;
		box-shadow: none;
		outline: none;
		padding: 0.125rem 0.25rem;
		min-height: 0;
		resize: none;
	}

	.prompt-doc__empty {
		margin: 0;
		padding: 0.75rem;
		font-size: 0.875rem;
		color: var(--prompt-doc-muted, var(--color-text-muted, #9ca3af));
	}

	/* Add-section — left-aligned, low-key (grows the document inline). */
	.prompt-doc__add {
		display: flex;
		justify-content: flex-start;
	}

	/* Preview region. */
	.prompt-doc__preview {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.prompt-doc__preview-bar {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: 0.75rem;
	}

	.prompt-doc__preview-toggle {
		background: none;
		border: none;
		padding: 0;
		font: inherit;
		font-size: 0.8125rem;
		color: var(--prompt-doc-muted, var(--color-text-muted, #9ca3af));
		cursor: pointer;
		text-decoration: underline;
		text-underline-offset: 2px;
	}
	.prompt-doc__preview-toggle:hover {
		color: var(--prompt-doc-text, var(--color-text, #e5e7eb));
	}

	.prompt-doc__formats {
		display: inline-flex;
		gap: 0.25rem;
	}

	.prompt-doc__format {
		padding: 0.1875rem 0.625rem;
		font-size: 0.75rem;
		font-weight: 500;
		color: var(--prompt-doc-muted, var(--color-text-muted, #9ca3af));
		background: var(--prompt-doc-chip-bg, var(--color-bg-tertiary, #252540));
		border: 1px solid var(--prompt-doc-border, var(--color-border, #2a2a4a));
		border-radius: 999px;
		cursor: pointer;
		transition:
			color 0.15s ease,
			background 0.15s ease,
			border-color 0.15s ease;
	}
	.prompt-doc__format:hover {
		color: var(--prompt-doc-text, var(--color-text, #e5e7eb));
	}
	.prompt-doc__format.is-active {
		color: var(--prompt-doc-accent-text, #fff);
		background: var(--prompt-doc-accent, var(--color-accent, #6366f1));
		border-color: var(--prompt-doc-accent, var(--color-accent, #6366f1));
	}

	/* The rendered output — full width, monospace, wraps (never clipped, never
	   horizontally scrolled away). */
	.prompt-doc__preview-out {
		margin: 0;
		padding: 0.75rem;
		width: 100%;
		font-family: var(--prompt-doc-mono, ui-monospace, 'SF Mono', Menlo, monospace);
		font-size: 0.8125rem;
		line-height: 1.5;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
		color: var(--prompt-doc-text, var(--color-text, #e5e7eb));
		background: var(--prompt-doc-preview-bg, var(--color-bg-tertiary, #252540));
		border: 1px solid var(--prompt-doc-border, var(--color-border, #2a2a4a));
		border-radius: var(--prompt-doc-radius, 0.5rem);
	}

	@media (prefers-reduced-motion: reduce) {
		.prompt-doc__controls,
		.prompt-doc__format {
			transition: none;
		}
	}
</style>
