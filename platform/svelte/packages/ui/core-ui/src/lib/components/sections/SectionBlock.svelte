<script lang="ts">
	/**
	 * SectionBlock — one prompt section on the Edit surface (dumb, controlled, recursive). Renders the
	 * mockup's `.d-sec` (top-level) or `.d-child` (nested) shape: grip · name · provenance pill ·
	 * delete/revert, an editable description + content (inline `$arg` chips), a "uses" caption, and its
	 * nested children (each a SectionBlock). All mutations fire host callbacks keyed by block name — the
	 * component never imports a store/API and never mutates its own props.
	 */
	import { tick } from 'svelte';
	import { tokenizeArgs, looksLikeJson } from './sectionContent';
	import type { SectionNode } from './sectionContent';
	// Explicit self-import for recursion: the implicit self-reference works in the main template but NOT
	// inside a {#snippet} (where the nested render now lives), so import the component by name.
	import SectionBlock from './SectionBlock.svelte';
	import PresetPill from './PresetPill.svelte';
	import ArgToken from './ArgToken.svelte';
	import SectionAddRow from './SectionAddRow.svelte';

	interface Labels {
		presetPrefix?: string;
		editedWas?: string;
		custom?: string;
		uses?: string;
		declaredIn?: string;
		addInside?: string;
		divergesFrom?: string;
		divergesActions?: string;
		emptyHint?: string;
	}

	type DropPlace = 'before' | 'after';

	interface Props {
		node: SectionNode;
		nested?: boolean;
		labels?: Labels;
		/** Read-only view (e.g. a template): render the block but disable all edit affordances —
		    no content editing, no delete/add-inside, no preset picker. */
		readonly?: boolean;
		onDelete?: (name: string) => void;
		onRevert?: (name: string) => void;
		onPickPreset?: (name: string) => void;
		onAddInside?: (parent: string, child: string) => void;
		onContentInput?: (name: string, content: string) => void;
		/** Clicking a `$var` chip in the content opens the host's variable editor. */
		onArgClick?: (name: string) => void;
		/** Reorder wiring (top-level only). The grip is the drag handle; the section is the drop zone.
		    `grabbed` dims the block being dragged; `dropIndicator` paints the insert line. */
		grabbed?: boolean;
		dropIndicator?: DropPlace | null;
		onGrab?: (name: string) => void;
		onGrabEnd?: () => void;
		onOver?: (name: string, place: DropPlace) => void;
		onDrop?: (name: string) => void;
	}

	let {
		node,
		nested = false,
		labels = {},
		readonly = false,
		onDelete,
		onRevert,
		onPickPreset,
		onAddInside,
		onContentInput,
		onArgClick,
		grabbed = false,
		dropIndicator = null,
		onGrab,
		onGrabEnd,
		onOver,
		onDrop
	}: Props = $props();

	// Drag-to-reorder (top-level sections). The whole section is the drop zone but is NOT itself
	// draggable — only the grip is — so contenteditable text selection inside the content still works.
	let secEl = $state<HTMLElement | null>(null);
	const draggable = $derived(!nested && !!onGrab);

	function grabStart(e: DragEvent) {
		if (!onGrab) return;
		onGrab(node.name);
		e.dataTransfer?.setData('text/plain', node.name);
		if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move';
		if (secEl && e.dataTransfer) e.dataTransfer.setDragImage(secEl, 20, 18);
	}
	function grabEnd() {
		onGrabEnd?.();
	}
	function dragOver(e: DragEvent) {
		if (!onOver || !secEl) return;
		e.preventDefault();
		if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
		const rect = secEl.getBoundingClientRect();
		onOver(node.name, e.clientY - rect.top < rect.height / 2 ? 'before' : 'after');
	}
	function drop(e: DragEvent) {
		if (!onDrop) return;
		e.preventDefault();
		onDrop(node.name);
	}

	const L = $derived({
		presetPrefix: labels.presetPrefix ?? 'preset ·',
		editedWas: labels.editedWas ?? 'edited — was',
		custom: labels.custom ?? 'custom',
		uses: labels.uses ?? 'Uses',
		declaredIn: labels.declaredIn ?? '· declared in Variables',
		addInside: labels.addInside ?? 'Add section inside',
		divergesFrom: labels.divergesFrom ?? 'Diverges from',
		divergesActions: labels.divergesActions ?? '— Save to preset, keep as custom, or revert',
		emptyHint: labels.emptyHint ?? 'Empty — click to write'
	});

	const segments = $derived(tokenizeArgs(node.content));
	const isJson = $derived(looksLikeJson(node.content));
	const uses = $derived(node.uses ?? []);
	const hasChildren = $derived((node.children?.length ?? 0) > 0);

	// Nested blocks are collapsible (mockup expander); top-level always shown.
	let expanded = $state(true);
	let adding = $state(false);

	// Content editing model (owner-approved): while FOCUSED the field is plain text the browser owns —
	// its body is NOT rendered from state, so a keystroke can never trigger a reactive re-render that
	// resets the caret (the bug that scrambled input). On blur we swap back to the chip-rendered view.
	let editing = $state(false);
	let editEl = $state<HTMLElement | null>(null);

	async function beginEdit() {
		if (readonly || editing) return;
		editing = true;
		await tick(); // editEl now mounted
		if (!editEl) return;
		editEl.innerText = node.content; // seed ONCE; the browser owns the DOM from here
		editEl.focus();
		const sel = typeof window !== 'undefined' ? window.getSelection() : null;
		if (sel) {
			const r = document.createRange();
			r.selectNodeContents(editEl);
			r.collapse(false); // caret at end
			sel.removeAllRanges();
			sel.addRange(r);
		}
	}
	function endEdit() {
		// Commit the final text, then re-render chips.
		if (editEl) onContentInput?.(node.name, editEl.innerText);
		editing = false;
	}
	function contentInput() {
		// Push each keystroke up WITHOUT re-rendering this element (its body is not state-bound).
		if (editEl) onContentInput?.(node.name, editEl.innerText);
	}
	// A click on a `$var` chip opens the var editor; a click on plain text enters edit mode.
	function displayPointerDown(e: PointerEvent) {
		if (readonly) return; // a template is read-only — no click-to-edit
		if ((e.target as HTMLElement).closest('[data-testid^="arg-token-"]')) return; // let the chip handle it
		e.preventDefault();
		void beginEdit();
	}
</script>

{#snippet prov()}
	<span class="prov">
		<PresetPill
			status={node.status}
			presetName={node.presetName}
			prefixLabel={L.presetPrefix}
			editedLabel={L.editedWas}
			customLabel={L.custom}
			onclick={readonly ? undefined : () => onPickPreset?.(node.name)}
		/>
		{#if !readonly}
			{#if node.status === 'edited'}
				<button class="icobtn" aria-label="Revert" title="Revert to preset" onclick={() => onRevert?.(node.name)}>↺</button>
			{/if}
			<button class="icobtn" aria-label={nested ? 'Delete' : 'Delete section'} onclick={() => onDelete?.(node.name)}>✕</button>
		{/if}
	</span>
{/snippet}

{#snippet body()}
	{#if editing}
		<!-- EDIT: plain text the browser owns. Body is NOT state-bound, so a keystroke can't trigger a
		     reactive re-render that resets the caret. `editing` is checked first so typing JSON-shaped
		     text never unmounts the editor mid-edit. -->
		<div
			class="d-content content"
			data-testid="section-content"
			bind:this={editEl}
			contenteditable="plaintext-only"
			role="textbox"
			tabindex="0"
			aria-label="Section content"
			aria-multiline="true"
			oninput={contentInput}
			onblur={endEdit}
		></div>
	{:else if isJson}
		<!-- IDLE (JSON): pretty display; focus or click enters plain-text edit. -->
		<div
			class="d-content jsonbox"
			data-testid="section-content"
			role="textbox"
			tabindex="0"
			aria-label="Section content"
			onfocus={beginEdit}
			onpointerdown={displayPointerDown}
		>{node.content}</div>
	{:else}
		<!-- IDLE: chips render; focus (tab) or a text click enters edit; a chip click edits the var. -->
		<div
			class="d-content content"
			data-testid="section-content"
			data-placeholder={L.emptyHint}
			role="textbox"
			tabindex="0"
			aria-label="Section content"
			aria-multiline="true"
			onfocus={beginEdit}
			onpointerdown={displayPointerDown}
		>{#each segments as seg}{#if seg.kind === 'arg'}<ArgToken name={seg.name} onclick={onArgClick} />{:else}{seg.value}{/if}{/each}</div>
	{/if}
	{#if uses.length}
		<div class="uses" data-testid="uses-caption">
			<span>{L.uses}</span>
			{#each uses as v, i}<span class="v">${v}</span>{i < uses.length - 1 ? ', ' : ' '}{/each}
			<span>{L.declaredIn}</span>
		</div>
	{/if}
{/snippet}

{#snippet nest()}
	<!-- Children + add-inside, shared by top-level AND nested blocks so nesting works at ANY depth and
	     the add-inside affordance is ALWAYS available (a leaf can gain its first child). -->
	<div class="d-nest">
		{#each node.children ?? [] as child (child.name)}
			<SectionBlock
				node={child}
				nested
				{readonly}
				{labels}
				{onDelete}
				{onRevert}
				{onPickPreset}
				{onAddInside}
				{onContentInput}
				{onArgClick}
			/>
		{/each}
		{#if !readonly}
			{#if adding}
				<SectionAddRow
					inside
					onadd={(name) => {
						onAddInside?.(node.name, name);
						adding = false;
					}}
					oncancel={() => (adding = false)}
				/>
			{:else}
				<button class="addinside" onclick={() => (adding = true)}>
					<span class="addplus">+</span>
					<span>{L.addInside}</span>
					<b>{node.name}</b>
				</button>
			{/if}
		{/if}
	</div>
{/snippet}

{#if nested}
	<div class="d-child" data-testid="section-block-{node.name}">
		<div class="d-childhead">
			<span class="grip" aria-hidden="true"></span>
			{#if hasChildren || node.content}
				<button class="d-expander" aria-label={expanded ? 'Collapse' : 'Expand'} onclick={() => (expanded = !expanded)}>{expanded ? '▾' : '▸'}</button>
			{/if}
			<span class="name">{node.name}</span>
			{@render prov()}
		</div>
		{#if expanded}
			<div class="d-childbody">
				{@render body()}
				{#if node.status === 'edited'}
					<div class="editnote">
						<span>{L.divergesFrom}</span>
						<b>{node.presetName}</b>
						<span>{L.divergesActions}</span> ↺.
					</div>
				{/if}
				{@render nest()}
			</div>
		{/if}
	</div>
{:else}
	<!-- The keyboard-accessible reorder affordance is the grip button; the section-level dragover/drop
	     are supplementary pointer handlers, so the static-element interaction warning is expected. -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<section
		bind:this={secEl}
		class="d-sec"
		class:grabbed
		class:drop-before={dropIndicator === 'before'}
		class:drop-after={dropIndicator === 'after'}
		data-testid="section-block-{node.name}"
		ondragover={dragOver}
		ondrop={drop}
	>
		<div class="d-head">
			{#if draggable}
				<span
					class="grip grip--handle"
					role="button"
					tabindex="0"
					aria-label="Drag to reorder {node.name}"
					title="Drag to reorder"
					draggable="true"
					ondragstart={grabStart}
					ondragend={grabEnd}
				></span>
			{:else}
				<span class="grip" aria-hidden="true"></span>
			{/if}
			<span class="name">{node.name}</span>
			{@render prov()}
		</div>
		{#if node.description}
			<div class="d-desc">{node.description}</div>
		{/if}
		<div class="d-content-wrap">{@render body()}</div>

		{@render nest()}
	</section>
{/if}

<style>
	.d-sec {
		padding: 20px 0;
		border-bottom: 1px solid var(--color-border);
		position: relative;
	}
	.d-sec:last-of-type {
		border-bottom: 0;
	}
	.d-sec:hover :global(.grip) {
		opacity: 1;
	}
	/* Reorder affordances: dim the grabbed block, draw the insert line before/after a drop target. */
	.d-sec.grabbed {
		opacity: 0.4;
	}
	.d-sec.drop-before::before,
	.d-sec.drop-after::after {
		content: '';
		position: absolute;
		left: 0;
		right: 0;
		height: 2px;
		background: var(--color-primary);
		border-radius: 2px;
	}
	.d-sec.drop-before::before {
		top: 0;
	}
	.d-sec.drop-after::after {
		bottom: 0;
	}
	.grip--handle {
		cursor: grab;
	}
	.grip--handle:active {
		cursor: grabbing;
	}
	.grip--handle:focus-visible {
		opacity: 1;
		outline: 2px solid var(--color-primary);
		outline-offset: 2px;
		border-radius: 2px;
	}
	.d-head {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.d-head .grip {
		margin-left: -16px;
	}
	.name {
		font-family: var(--font-family-mono, ui-monospace, monospace);
		font-weight: 600;
		font-size: 13px;
		color: var(--color-text);
	}
	.prov {
		margin-left: auto;
		display: flex;
		align-items: center;
		gap: 2px;
	}
	.grip {
		width: 12px;
		height: 16px;
		flex: 0 0 auto;
		opacity: 0;
		transition: opacity 0.12s;
		background: radial-gradient(circle 1.3px at 2.5px 3px, var(--color-faint) 99%, transparent),
			radial-gradient(circle 1.3px at 7.5px 3px, var(--color-faint) 99%, transparent),
			radial-gradient(circle 1.3px at 2.5px 8px, var(--color-faint) 99%, transparent),
			radial-gradient(circle 1.3px at 7.5px 8px, var(--color-faint) 99%, transparent),
			radial-gradient(circle 1.3px at 2.5px 13px, var(--color-faint) 99%, transparent),
			radial-gradient(circle 1.3px at 7.5px 13px, var(--color-faint) 99%, transparent);
		cursor: grab;
	}
	.icobtn {
		min-width: 26px;
		height: 26px;
		padding: 0 5px;
		border-radius: 6px;
		border: 0;
		background: transparent;
		color: var(--color-faint);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		font-size: 14px;
		line-height: 1;
	}
	.icobtn:hover {
		background: var(--color-subtle);
		color: var(--color-text-muted);
	}
	.d-desc {
		margin-top: 8px;
		font-size: 13px;
		line-height: 1.5;
		color: var(--color-text-muted);
		border: 1px solid transparent;
		border-radius: 7px;
		padding: 5px 8px;
		margin-left: -8px;
	}
	.d-desc:hover {
		background: var(--color-surface-2);
		border-color: var(--color-border);
	}
	.d-content-wrap {
		margin-top: 12px;
	}
	.content {
		font-size: 14.5px;
		line-height: 1.66;
		color: var(--color-text);
		white-space: pre-wrap;
		word-break: break-word;
		outline: none;
		border-radius: 7px;
		/* An EMPTY section must stay clickable to start editing — without a min-height the idle display
		   collapses to 0px and there is no target to click into. */
		min-height: 1.66em;
		cursor: text;
	}
	/* Faint hint on an empty IDLE display (the editing contenteditable is excluded via [contenteditable]). */
	.content:not([contenteditable]):empty::before {
		content: attr(data-placeholder);
		color: var(--color-faint);
	}
	.content:focus {
		box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-primary) 14%, transparent);
	}
	.jsonbox {
		font-family: var(--font-family-mono, ui-monospace, monospace);
		font-size: 12.5px;
		line-height: 1.6;
		background: var(--color-surface-2);
		border: 1px solid var(--color-border);
		border-radius: 8px;
		padding: 11px 13px;
		color: var(--color-code);
		overflow-x: auto;
	}
	.uses {
		font-size: 12px;
		color: var(--color-faint);
		margin-top: 9px;
	}
	.uses .v {
		font-family: var(--font-family-mono, ui-monospace, monospace);
		color: var(--color-text-muted);
	}

	/* Nested children */
	.d-nest {
		margin: 14px 0 2px 3px;
		padding-left: 16px;
		border-left: 2px solid var(--color-accent-soft);
		display: flex;
		flex-direction: column;
		gap: 4px;
	}
	.d-child:hover :global(.grip) {
		opacity: 1;
	}
	.d-childhead {
		display: flex;
		align-items: center;
		gap: 9px;
		padding: 9px 8px;
		border-radius: 9px;
	}
	.d-childhead:hover {
		background: var(--color-surface-2);
	}
	.d-childhead .grip {
		margin-left: -14px;
	}
	.d-expander {
		color: var(--color-faint);
		font-size: 11px;
		width: 16px;
		text-align: center;
		border: 0;
		background: transparent;
		padding: 0;
	}
	.d-childbody {
		padding: 2px 8px 12px 22px;
	}
	.d-childbody .content {
		font-size: 14px;
	}
	.editnote {
		margin-top: 9px;
		font-size: 12px;
		color: var(--color-faint);
	}
	.editnote b {
		color: var(--color-text-muted);
		font-weight: 600;
	}
	.addinside {
		display: inline-flex;
		align-items: center;
		gap: 9px;
		color: var(--color-text-muted);
		font-size: 13px;
		padding: 9px 8px;
		border: 0;
		background: transparent;
	}
	.addinside:hover {
		color: var(--color-accent-ink);
	}
	.addinside b {
		font-weight: 600;
		color: var(--color-text-muted);
	}
	.addplus {
		width: 20px;
		height: 20px;
		border: 1.5px solid var(--color-line-strong);
		border-radius: 6px;
		color: var(--color-text-muted);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		font-size: 14px;
		line-height: 1;
		flex: 0 0 auto;
		transition: border-color 0.12s, color 0.12s, background 0.12s;
	}
	.addinside:hover .addplus {
		border-color: var(--color-primary);
		color: var(--color-accent-ink);
		background: color-mix(in srgb, var(--color-primary) 8%, transparent);
	}
</style>
