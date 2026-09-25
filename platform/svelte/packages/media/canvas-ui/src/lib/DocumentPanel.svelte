<script lang="ts">
	// Document context panel (IA refactor R1) — the 1st rail item, default context.
	// Replaces the dissolved Settings gear's document half. Two sections:
	//   Document settings  title, kind (document/template), the REAL artboard size
	//                       (from the page model) + a Resize… button that opens the
	//                       functional ResizeDialog (owned by EditorShell so the Modal
	//                       portals without clipping).
	//   Templates           the embedded TemplatesPanel section (favourites + all +
	//                       scoped search) — the standalone Templates rail tab is gone.
	// Sources moved OUT to the dedicated Sources panel; export defaults moved to the
	// export dialog (R4). CTAs right-aligned (hard rule).
	import { untrack } from 'svelte';
	import { Field, Input, Select, Button, Icon } from '@sbx/core-ui/components/primitives';
	import type { FormatterRegistry } from '@sbx/canvas-kit';
	import type { TemplateGallery } from './template-gallery.js';
	import Panel from './Panel.svelte';
	import CollapsibleSection from './CollapsibleSection.svelte';
	import TemplatesPanel from './TemplatesPanel.svelte';

	interface Props {
		/** Document title. */
		title: string;
		artboardWidth: number;
		artboardHeight: number;
		isTemplate?: boolean;
		/** Whether the document already has user content (drives the Templates apply confirm). */
		documentHasContent?: boolean;
		/** Real templates + lazy loader + apply (consumer-owned). Omit → empty state. */
		gallery?: TemplateGallery;
		/** Image proxy + formatters forwarded to each live template preview. */
		resolveImageSrc?: (src: string) => string;
		formatters?: FormatterRegistry;
		/** Open the functional ResizeDialog (owned by EditorShell). */
		onresize?: () => void;
		/** Commit a new document title (blur / Enter) — persistence (slice 3). */
		onrename?: (name: string) => void;
		/** Change the document kind ('document' | 'template') — persistence (slice 3). */
		onkindchange?: (kind: string) => void;
		/** Whether the Kind is user-selectable (Decision #0298). Default true = the
		 *  document/template <Select>. false renders a fixed read-only label (`kindLabel`)
		 *  with no onkindchange — the kind is pinned by the editor mode (e.g. watermark). */
		kindSelectable?: boolean;
		/** The fixed Kind label shown when `kindSelectable` is false (e.g. "Watermark"). */
		kindLabel?: string;
		/** Whether the Templates section (start-from-template gallery) is shown. Default
		 *  true (Media Canvas). The watermark mode sets it false — templates are a
		 *  Media-Canvas concept, irrelevant to overlay authoring (#0298). */
		showTemplates?: boolean;
		/** Panel heading. Default 'Document'. The merged watermark panel passes 'Documents'
		 *  (it now also hosts the sibling-documents list). */
		panelTitle?: string;
		/** Whether the Kind row (document/template select or static label) is shown at all.
		 *  Default true. The watermark properties show ONLY title + size (no type), so the
		 *  watermark surface passes false. */
		showKind?: boolean;
		/** Optional header action (a ＋ to the right of the title) — forwarded to Panel.
		 *  The watermark route passes a "create new watermark" button. */
		headerAction?: import('svelte').Snippet;
		/** Optional consumer-owned documents LIST, rendered BELOW the properties — the
		 *  sibling-documents switcher (search · rows · delete). Omit → properties only. */
		documentsList?: import('svelte').Snippet;
		/** Optional consumer-owned action at the FOOT of the Document-settings section (below the
		 *  size) — the watermark route renders a subtle "Delete watermark" link here so the CURRENT
		 *  document is deletable from its own settings. Omit → no footer action. */
		settingsAction?: import('svelte').Snippet;
		/** Optional note shown directly beneath the artboard size — a consumer-owned caption tied
		 *  to the dimensions (the watermark route shows "Only square watermarks can be published."
		 *  when the size isn't square). Omit → no note. */
		sizeNote?: string;
	}

	let {
		title,
		artboardWidth,
		artboardHeight,
		isTemplate = false,
		documentHasContent = true,
		gallery,
		resolveImageSrc = (s) => s,
		formatters = {},
		onresize,
		onrename,
		onkindchange,
		kindSelectable = true,
		kindLabel = '', // kit-neutral default; the consumer supplies the domain term (the shell passes "Watermark")
		showTemplates = true,
		panelTitle = 'Document',
		showKind = true,
		headerAction,
		documentsList,
		settingsAction,
		sizeNote
	}: Props = $props();

	// Editable title draft — seeded once from the prop; committed on blur / Enter
	// (oninput only tracks the draft, so the document isn't re-cloned per keystroke).
	let titleDraft = $state(untrack(() => title));

	// Document kind — committed via onkindchange (persistence, slice 3). Seeded ONCE
	// from the prop (untrack makes the one-time seed explicit).
	let docKind = $state(untrack(() => (isTemplate ? 'template' : 'document')));
	const kindOptions = [
		{ value: 'document', label: 'Document' },
		{ value: 'template', label: 'Template' }
	];
</script>

<Panel title={panelTitle} search={false} {headerAction}>
	<CollapsibleSection title="Document settings">
		<div class="doc">
			<Field for="document-title" label="Title">
				{#snippet children()}
					<Input
						id="document-title"
						name="doc-title"
						value={titleDraft}
						size="sm"
						placeholder="Untitled"
						oninput={(v) => (titleDraft = v)}
						onchange={(v) => onrename?.(v)}
						onkeydown={(e) => {
							if (e.key === 'Enter') onrename?.(titleDraft);
						}}
					/>
				{/snippet}
			</Field>

			{#if showKind}
				{#if kindSelectable}
					<Field for="document-kind" label="Kind">
						{#snippet children()}
							<Select
								id="document-kind"
								name="doc-kind"
								bind:value={docKind}
								options={kindOptions}
								size="sm"
								onchange={(v) => onkindchange?.(v)}
							/>
						{/snippet}
					</Field>
				{:else}
					<!-- Kind pinned by the editor mode (#0298) — a fixed read-only label, no
					     onkindchange. The watermark surface is always kind='watermark'. -->
					<div class="kind-static">
						<span class="kind-static__label">Kind</span>
						<span class="kind-static__value">{kindLabel}</span>
					</div>
				{/if}
			{/if}

			<div class="size">
				<div class="info__row">
					<span class="info__key">Artboard size</span>
					<span class="info__val">{artboardWidth} × {artboardHeight} px</span>
				</div>
				{#if sizeNote}
					<!-- Consumer note tied to the dimensions (e.g. "Only square watermarks can be
					     published.") — sits with the size so the constraint reads where it applies. -->
					<p class="size__note">{sizeNote}</p>
				{/if}
				<!-- CTA right-aligned (hard rule). -->
				<div class="size__cta">
					<Button variant="secondary" size="sm" onclick={() => onresize?.()}>
						<span class="size__btn"><Icon name="maximize" size="sm" />Resize…</span>
					</Button>
				</div>
			</div>

				{#if settingsAction}
					<!-- Consumer-owned footer action (e.g. a subtle "Delete watermark" link). A
					     hairline sets a destructive link apart from the controls above; right-aligned
					     (CTA hard rule). The consumer owns the action + its confirm + any redirect. -->
					<div class="doc__action">{@render settingsAction()}</div>
				{/if}
		</div>
	</CollapsibleSection>

	{#if showTemplates}
		<CollapsibleSection title="Templates">
			<TemplatesPanel {gallery} {resolveImageSrc} {formatters} {documentHasContent} />
		</CollapsibleSection>
	{/if}

	<!-- Sibling-documents list (the consumer owns its semantics): the searchable switcher,
	     rendered BELOW the current-doc properties so this is ONE merged panel. Wrapped in the
	     same CollapsibleSection rhythm as the other sections. Omitted → a properties-only
	     Document panel (Media Canvas), unchanged. -->
	{#if documentsList}
		<CollapsibleSection title="All documents">
			{@render documentsList()}
		</CollapsibleSection>
	{/if}
</Panel>

<style>
	.doc {
		display: flex;
		flex-direction: column;
		gap: var(--cv-space-md, 1rem);
	}

	/* Read-only Kind row (#0298) — mirrors the Field label/value rhythm without an
	   editable control (the kind is pinned by the editor mode). */
	.kind-static {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.kind-static__label {
		font-size: 0.75rem;
		font-weight: 500;
		color: var(--cv-color-neutral-600, #505050);
	}

	.kind-static__value {
		font-size: 0.8125rem;
		font-weight: 600;
		color: var(--cv-color-neutral-800, #202020);
	}

	.size {
		display: flex;
		flex-direction: column;
		gap: var(--cv-space-sm, 0.5rem);
	}

	.size__cta {
		display: flex;
		justify-content: flex-end;
	}

	/* Dimension caption (e.g. the watermark publish-requires-square note) — muted, full width,
	   wraps freely (never clipped). Sits between the size row and the Resize CTA. */
	.size__note {
		margin: 0;
		font-size: 0.75rem;
		line-height: 1.4;
		color: var(--cv-color-neutral-600, #505050);
	}

	/* Footer action slot (e.g. the watermark "Delete" link) — a hairline above sets a
	   destructive action apart from the controls; right-aligned (CTA hard rule). */
	.doc__action {
		display: flex;
		justify-content: flex-end;
		padding-top: var(--cv-space-sm, 0.5rem);
		border-top: 1px solid var(--cv-border-color, #e0e0e0);
	}

	.size__btn {
		display: inline-flex;
		align-items: center;
		gap: 0.375rem;
	}

	.info__row {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: var(--cv-space-sm, 0.5rem);
	}

	.info__key {
		font-size: 0.75rem;
		color: var(--cv-color-neutral-500, #707070);
	}

	.info__val {
		font-size: 0.8125rem;
		font-weight: 500;
		color: var(--cv-color-neutral-800, #202020);
		text-align: right;
		overflow-wrap: anywhere;
	}
</style>
