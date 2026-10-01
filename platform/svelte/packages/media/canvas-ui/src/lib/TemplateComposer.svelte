<script lang="ts">
	// TemplateComposer — a contenteditable WYSIWYG for @sbx/text-template templates. The
	// surface mixes free-typed literal text with atomic, non-editable TOKEN badges: each badge
	// shows the catalog LABEL on top and the resolver's RESOLVED value below, so the author edits
	// a live-looking sentence ("Welcome to [Bangkok]") instead of raw [property.address.country].
	//
	// CONTRACT (consumer-agnostic — driven ONLY by the @sbx/text-template protocols, so the same
	// composer drives the Media Canvas today and post/content tools next; never a dep on a concrete
	// data source):
	//   • RENDER   nodes → DOM: text node → a TEXT_NODE; token node → a `span.ttc-badge`
	//     [contenteditable=false] carrying data-token / data-format / data-format-options. Badges
	//     are built IMPERATIVELY (one `buildBadge` factory) — never via Svelte `{#each}` — because
	//     Svelte-managed DOM and user contenteditable mutations fight over the same nodes.
	//   • SERIALIZE DOM → nodes (walk childNodes): TEXT_NODE → {type:'text'} (empties dropped);
	//     `<br>` → a '\n' text node; `span.ttc-badge` → {type:'token', token, format?, formatOptions?}.
	//     `onchange(nodes)` fires on every edit (input / insert / badge-remove / paste).
	//   • SYNC     re-render only when the incoming `template` differs STRUCTURALLY from the DOM
	//     (serializeTemplate compare) AND the surface is NOT focused — so an external change reflects
	//     without ever clobbering the caret mid-type.
	// Per-knob format EDITING is out of scope (the Inspector's Format knobs own that); the badge
	// popover is read-only props + Remove, and an optional onselecttoken lets a host open its own editor.

	import { tick, type Snippet } from 'svelte';
	import Modal from '@sbx/core-ui/components/primitives/Modal.svelte';
	import { Button, Select, Icon } from '@sbx/core-ui/components/primitives';
	import {
		serializeTemplate,
		type Template,
		type TemplateNode,
		type TokenResolver,
		type TokenCatalog
	} from '@sbx/text-template';
	import { FIELD_BIND_MIME, type FieldBindDrag } from './dnd.js';

	/** Context handed to the host's `badgeFormat` snippet — the active badge's token + its current
	 *  format hint/options, plus a writer to persist a format change back onto the badge. Lets the
	 *  host (the Inspector) render its OWN rich format knobs inside the badge popover without the
	 *  composer ever depending on a concrete formatter model. */
	export interface BadgeFormatContext {
		token: string;
		format?: string;
		formatOptions?: Record<string, string>;
		setFormat: (format: string | undefined, formatOptions?: Record<string, string>) => void;
	}

	interface Props {
		/** Current value (nodes). The persisted, canonical form. */
		template: Template;
		/** Insertable tokens — drives the "+ Insert field" picker AND the badge titles. */
		catalog: TokenCatalog;
		/** Token → resolved display string (the badge value + live preview). '' for missing. */
		resolver: TokenResolver;
		/** Emit the serialized nodes on EVERY edit. */
		onchange: (template: Template) => void;
		/** Optional empty-state hint shown when there are no nodes. */
		placeholder?: string;
		/** OPTIONAL — a host can open its own per-token editor (e.g. the Inspector's Format knobs)
		 *  instead of, or alongside, the built-in read-only popover. Index = node index in `template`. */
		onselecttoken?: (index: number) => void;
		/** OPTIONAL — host-rendered format controls for the ACTIVE badge, shown in its popover.
		 *  Receives the badge's token + current format/options + a writer (see {@link BadgeFormatContext}).
		 *  When omitted the popover shows the read-only format line only. */
		badgeFormat?: Snippet<[BadgeFormatContext]>;
	}

	let { template, catalog, resolver, onchange, placeholder, onselecttoken, badgeFormat }: Props = $props();

	// The contenteditable surface (imperatively read + written — never bound to reactive markup).
	let surface = $state<HTMLDivElement | null>(null);
	// True while WE are mutating the DOM (render) so the input handler doesn't echo our own write.
	let rendering = false;
	// The serialized string the DOM currently shows — the sync diff baseline.
	let domSerialized = $state('');

	// --- Insert picker + catalog ---------------------------------------------
	const catalogTokens = $derived(catalog.tokens());
	// The catalog's tokens, mapped to Select options. Labels disambiguate by source (a field label
	// like "Price" can exist on several sources) using the group's alias.
	const insertOptions = $derived(
		catalogTokens.map((t) => ({ value: t.token, label: optionLabel(t.label, t.group) }))
	);
	// Reset to placeholder after each pick (it's an action menu, never a persisted selection).
	let insertValue = $state('');
	// Format hint per token, so an inserted badge copies the field's declared format (catalog wins).
	const formatByToken = $derived(new Map(catalogTokens.map((t) => [t.token, t.format])));
	const labelByToken = $derived(new Map(catalogTokens.map((t) => [t.token, t.label])));

	// Disambiguated option label: "Price · secondary" when the token belongs to a named source group.
	function optionLabel(label: string, group?: string): string {
		const alias = group?.split(' · ')[0];
		return alias ? `${label} · ${alias}` : label;
	}

	// --- Fullscreen expand (#1) ----------------------------------------------
	// Render the SAME editor body inside a size=full Modal (the template prop is the source of truth,
	// so the surface re-renders from it on toggle — no shared-DOM conflict).
	let expanded = $state(false);

	// --- @-mention menu (#1) — type "@" to pick source → field inline ---------
	let mentionOpen = $state(false);
	let mentionQuery = $state('');
	let mentionIndex = $state(0);
	let mentionX = $state(0);
	let mentionY = $state(0);
	// The caret text-node + the @-char offset of the live "@query" — so selecting deletes it cleanly.
	let mentionAnchor: { node: Text; start: number } | null = null;
	// Catalog tokens matching the query (label OR source group contains it), capped for a tidy menu.
	const mentionItems = $derived.by(() => {
		const q = mentionQuery.trim().toLowerCase();
		const hit = q === '' ? catalogTokens : catalogTokens.filter((t) => `${t.label} ${t.group ?? ''}`.toLowerCase().includes(q));
		return hit.slice(0, 8);
	});

	// --- Badge popover -------------------------------------------------------
	let popoverOpen = $state(false);
	// The badge element a click opened the popover for (the Remove / reattach / reformat target).
	let activeBadge: HTMLElement | null = null;
	let activeToken = $state('');
	let activeFormat = $state<string | undefined>(undefined);
	let activeFormatOptions = $state<Record<string, string> | undefined>(undefined);

	const activeLabel = $derived(
		activeToken ? labelByToken.get(activeToken) ?? humanizeToken(activeToken) : ''
	);
	const activeValue = $derived(
		activeToken ? resolver.resolve(activeToken, activeFormat, activeFormatOptions) : ''
	);
	// The context the host's `badgeFormat` snippet renders its knobs from (the active badge's
	// token + format + a writer that persists a format change back onto the badge).
	const badgeFormatContext = $derived<BadgeFormatContext>({
		token: activeToken,
		format: activeFormat,
		formatOptions: activeFormatOptions,
		setFormat: applyActiveBadgeFormat
	});

	// --- Empty state ---------------------------------------------------------
	const isEmpty = $derived(template.length === 0);

	// === Render (nodes → DOM) ===============================================
	// Humanize a bare token for the title fallback: 'property.address.country' → 'Country'.
	function humanizeToken(token: string): string {
		const leaf = token.split('.').pop() ?? token;
		const spaced = leaf.replace(/[_-]+/g, ' ').replace(/([a-z0-9])([A-Z])/g, '$1 $2');
		return spaced.charAt(0).toUpperCase() + spaced.slice(1);
	}

	// The badge's icon-CTA glyph (#1) — a small ⚙ (lucide settings) drawn inline since badges are
	// built imperatively (no Svelte <Icon> in raw DOM). Opens the props / format / reattach popover.
	const COG_SVG =
		'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>';

	// Build one atomic badge element. Two stacked lines: muted TITLE (catalog label / humanized)
	// over the RESOLVED value (bracketed [token] fallback when the resolver returns ''). The whole
	// pill is contenteditable=false → a single caret stop; adjacent backspace deletes it whole.
	function buildBadge(node: Extract<TemplateNode, { type: 'token' }>): HTMLSpanElement {
		const el = document.createElement('span');
		el.className = 'ttc-badge';
		el.contentEditable = 'false';
		el.setAttribute('data-token', node.token);
		if (node.format) el.setAttribute('data-format', node.format);
		if (node.formatOptions) el.setAttribute('data-format-options', JSON.stringify(node.formatOptions));
		// Prevent the badge from being a drop target / selection anchor inside the surface.
		el.setAttribute('role', 'button');
		el.setAttribute('tabindex', '-1');

		const title = labelByToken.get(node.token) ?? humanizeToken(node.token);
		const resolved = resolver.resolve(node.token, node.format, node.formatOptions);
		const value = resolved !== '' ? resolved : `[${node.token}]`;
		el.setAttribute('title', `${title}: ${value}`);
		el.setAttribute('aria-label', `${title}: ${value}`);

		const titleEl = document.createElement('span');
		titleEl.className = 'ttc-badge__title';
		titleEl.textContent = title;
		const valueEl = document.createElement('span');
		valueEl.className = 'ttc-badge__value';
		valueEl.textContent = value;

		el.appendChild(titleEl);
		el.appendChild(valueEl);

		// Icon-CTA (#1) — a ⚙ in the badge's top-right that opens its props/format/reattach popover.
		// The whole badge is clickable too; the cog is the explicit, discoverable affordance.
		const cog = document.createElement('span');
		cog.className = 'ttc-badge__cog';
		cog.setAttribute('aria-hidden', 'true');
		cog.innerHTML = COG_SVG;
		el.appendChild(cog);
		return el;
	}

	// Render the given nodes into the surface (full rebuild — templates are short; a diff would
	// add bug surface for no perceptible gain). `rendering` suppresses the input echo.
	function renderNodes(nodes: Template): void {
		if (!surface) return;
		rendering = true;
		surface.replaceChildren();
		for (const node of nodes) {
			if (node.type === 'text') {
				// Split on newlines → text + <br>, so a multi-line template round-trips.
				const parts = node.text.split('\n');
				parts.forEach((part, i) => {
					if (part !== '') surface!.appendChild(document.createTextNode(part));
					if (i < parts.length - 1) surface!.appendChild(document.createElement('br'));
				});
			} else {
				// Caret-landing nodes (#1a/#1b): a zero-width space BEFORE + AFTER every badge
				// gives the caret a text position on each side, so a click adjacent to the chip
				// lands a caret (instead of hitting the atomic, surface-filling badge → opening
				// its popover) and a space can be typed next to it. insertBadgeAtCaret adds the
				// trailing ZWSP on the insert path; rendering an existing template (open, layer
				// switch, linked→template convert) skipped it — the bug. Stripped on serialize.
				surface.appendChild(document.createTextNode('​'));
				surface.appendChild(buildBadge(node));
				surface.appendChild(document.createTextNode('​'));
			}
		}
		domSerialized = serializeTemplate(nodes);
		rendering = false;
	}

	// === Serialize (DOM → nodes) ============================================
	// Walk the surface's direct children → Template. TEXT_NODE → text (empties dropped); a badge
	// span → a token node (reading the data-* back); <br> → a '\n' text node. Adjacent text/<br>
	// runs are NOT merged here — serializeTemplate compares structurally, and parse/render are
	// tolerant — but empties are dropped so blank text nodes never accumulate.
	// Zero-width chars (ZWSP U+200B, ZWNBSP/BOM U+FEFF) are the caret-landing spaces we inject
	// after a badge — strip them so they never leak into the persisted text.
	const ZERO_WIDTH = /[​﻿]/g;
	function serializeSurface(): Template {
		const nodes: Template = [];
		if (!surface) return nodes;
		for (const child of Array.from(surface.childNodes)) {
			if (child.nodeType === Node.TEXT_NODE) {
				const text = (child.textContent ?? '').replace(ZERO_WIDTH, '');
				if (text !== '') nodes.push({ type: 'text', text });
			} else if (child.nodeType === Node.ELEMENT_NODE) {
				const el = child as HTMLElement;
				if (el.tagName === 'BR') {
					nodes.push({ type: 'text', text: '\n' });
				} else if (el.classList.contains('ttc-badge')) {
					const token = el.getAttribute('data-token') ?? '';
					if (token === '') continue; // defensive: a badge with no token is dropped
					const format = el.getAttribute('data-format') ?? undefined;
					const optsRaw = el.getAttribute('data-format-options');
					let formatOptions: Record<string, string> | undefined;
					if (optsRaw) {
						try {
							formatOptions = JSON.parse(optsRaw) as Record<string, string>;
						} catch {
							formatOptions = undefined;
						}
					}
					nodes.push(format ? { type: 'token', token, format, formatOptions } : { type: 'token', token, formatOptions });
				}
				// Any other stray element (e.g. a pasted <div>/<span>) contributes its text only,
				// flattened — keeps the surface from ever harbouring nested editable structure.
				else {
					const text = (el.textContent ?? '').replace(ZERO_WIDTH, '');
					if (text !== '') nodes.push({ type: 'text', text });
				}
			}
		}
		return nodes;
	}

	// Read the DOM, update the baseline, emit. Called after every user edit + every insert/remove.
	function emit(): void {
		const nodes = serializeSurface();
		domSerialized = serializeTemplate(nodes);
		onchange(nodes);
	}

	// === Events ==============================================================
	function onInput(): void {
		if (rendering) return; // our own render — not a user edit
		emit();
		updateMention();
	}

	// === @-mention menu =====================================================
	// Detect a live "@query" ending at the caret → open the inline source→field menu. The query is
	// the run of (non-space) word chars after an "@" that starts at a word boundary (node start or
	// whitespace), so ordinary prose with an stray "@" mid-word never triggers it.
	function updateMention(): void {
		const range = currentRange();
		if (!range || !range.collapsed || range.startContainer.nodeType !== Node.TEXT_NODE) {
			closeMention();
			return;
		}
		const node = range.startContainer as Text;
		const upto = (node.textContent ?? '').slice(0, range.startOffset);
		const m = /(^|\s)@([\p{L}\p{N}_.-]*)$/u.exec(upto);
		if (!m) {
			closeMention();
			return;
		}
		mentionQuery = m[2];
		mentionAnchor = { node, start: m.index + m[1].length }; // index of the '@'
		mentionIndex = 0;
		const rect = range.getBoundingClientRect();
		mentionX = rect.left;
		mentionY = rect.bottom;
		mentionOpen = true;
	}

	function closeMention(): void {
		mentionOpen = false;
		mentionQuery = '';
		mentionAnchor = null;
		mentionIndex = 0;
	}

	// Replace the live "@query" text with the chosen field's badge.
	function selectMention(token: string): void {
		if (mentionAnchor && surface?.contains(mentionAnchor.node)) {
			const len = (mentionAnchor.node.textContent ?? '').length;
			const r = document.createRange();
			r.setStart(mentionAnchor.node, Math.min(mentionAnchor.start, len));
			r.setEnd(mentionAnchor.node, Math.min(mentionAnchor.start + 1 + mentionQuery.length, len));
			r.deleteContents();
			applyRange(r);
		}
		closeMention();
		insertBadgeAtCaret(token);
	}

	// Paste as PLAINTEXT only — never let rich HTML (with its own editable spans/structure) into
	// the surface. insertText keeps the caret + undo stack native; the input handler then emits.
	function onPaste(event: ClipboardEvent): void {
		event.preventDefault();
		const text = event.clipboardData?.getData('text/plain') ?? '';
		if (text === '') return;
		// document.execCommand is deprecated but remains the only cross-browser caret-preserving
		// plaintext insert; the fallback covers engines that have dropped it.
		const inserted = insertTextAtCaret(text);
		if (!inserted) return;
		emit();
	}

	// Accept a dropped field-bind chip (the canvas Texts-panel convention) as a NEW badge at the
	// drop point — isolated here so the core insert/serialize path needs only @sbx/text-template.
	// A host that doesn't produce FIELD_BIND_MIME simply never triggers this.
	function onDrop(event: DragEvent): void {
		const raw = event.dataTransfer?.getData(FIELD_BIND_MIME);
		if (!raw) return; // not a field chip — let the default happen (nothing editable to drop)
		event.preventDefault();
		let payload: FieldBindDrag;
		try {
			payload = JSON.parse(raw) as FieldBindDrag;
		} catch {
			return;
		}
		const token = payload.field?.token;
		if (!token) return;
		placeCaretFromPoint(event.clientX, event.clientY);
		insertBadgeAtCaret(token, payload.field.format);
	}

	function onDragOver(event: DragEvent): void {
		if (event.dataTransfer?.types.includes(FIELD_BIND_MIME)) {
			event.preventDefault(); // allow the drop
			event.dataTransfer.dropEffect = 'copy';
		}
	}

	// Open a badge's props popover (props + reattach + host format knobs + Remove), and notify a host
	// via onselecttoken with the badge's node-array index (see serializeIndexOf). Click + keyboard.
	function openBadge(badge: HTMLElement): void {
		if (!surface?.contains(badge)) return;
		closeMention();
		activeBadge = badge;
		activeToken = badge.getAttribute('data-token') ?? '';
		activeFormat = badge.getAttribute('data-format') ?? undefined;
		const optsRaw = badge.getAttribute('data-format-options');
		activeFormatOptions = optsRaw ? safeParseOptions(optsRaw) : undefined;
		popoverOpen = true;
		if (onselecttoken) {
			const idx = serializeIndexOf(badge);
			if (idx >= 0) onselecttoken(idx);
		}
	}

	function safeParseOptions(raw: string): Record<string, string> | undefined {
		try {
			return JSON.parse(raw) as Record<string, string>;
		} catch {
			return undefined;
		}
	}

	// Rebuild a badge in place with a new token/format/options (one construction path), refreshing
	// its title + resolved value so a reattach/reformat reflects immediately. Tracks the fresh node.
	function rewriteBadge(badge: HTMLElement, token: string, format?: string, formatOptions?: Record<string, string>): void {
		const fresh = buildBadge({ type: 'token', token, format, formatOptions });
		badge.replaceWith(fresh);
		activeBadge = fresh;
	}

	// Persist a format change from the host's knobs onto the active badge (data-format /
	// data-format-options), rebuild it so the value re-resolves, and emit.
	function applyActiveBadgeFormat(format: string | undefined, formatOptions?: Record<string, string>): void {
		if (!activeBadge || !surface?.contains(activeBadge)) return;
		rewriteBadge(activeBadge, activeToken, format, formatOptions);
		activeFormat = format;
		activeFormatOptions = formatOptions;
		emit();
	}

	// Point the active badge at another source/field — adopt the new field's default format and drop
	// the old per-knob options (they belonged to the previous formatter).
	function reattachActiveBadge(token: string): void {
		if (!activeBadge || !surface?.contains(activeBadge) || token === activeToken) return;
		const fmt = formatByToken.get(token);
		rewriteBadge(activeBadge, token, fmt, undefined);
		activeToken = token;
		activeFormat = fmt;
		activeFormatOptions = undefined;
		emit();
	}

	// Clicking a badge opens its popover. Delegated on the surface so imperatively-built badges
	// need no per-node wiring.
	function onSurfaceClick(event: MouseEvent): void {
		const badge = (event.target as HTMLElement)?.closest?.('.ttc-badge') as HTMLElement | null;
		if (!badge || !surface?.contains(badge)) return;
		event.preventDefault();
		openBadge(badge);
	}

	// Keyboard parity (a11y): when the caret sits adjacent to a badge, ENTER opens its props
	// popover — so keyboard-only users reach the same affordance as a click. Space is NEVER
	// hijacked — it must type a literal space next to a badge (#1) — and all other keys flow to
	// the native contenteditable (typing, arrows, backspace-deletes-the-atomic-badge, etc.).
	function onSurfaceKeydown(event: KeyboardEvent): void {
		// @-menu navigation takes priority while it's open.
		if (mentionOpen && mentionItems.length > 0) {
			if (event.key === 'ArrowDown') {
				event.preventDefault();
				mentionIndex = (mentionIndex + 1) % mentionItems.length;
				return;
			}
			if (event.key === 'ArrowUp') {
				event.preventDefault();
				mentionIndex = (mentionIndex - 1 + mentionItems.length) % mentionItems.length;
				return;
			}
			if (event.key === 'Enter' || event.key === 'Tab') {
				event.preventDefault();
				selectMention(mentionItems[mentionIndex].token);
				return;
			}
			if (event.key === 'Escape') {
				event.preventDefault();
				closeMention();
				return;
			}
		}
		// Space ALWAYS types (even with the caret hugging a badge) — only ENTER opens the
		// props popover, so it's no longer impossible to type a space next to a badge (#1).
		if (event.key !== 'Enter') return;
		const badge = badgeAtCaret();
		if (!badge) return; // not on a badge → let Enter insert a newline
		event.preventDefault();
		openBadge(badge);
	}

	// The badge element immediately before/after the collapsed caret, if any (so Enter on it opens
	// its popover). Returns null when the caret isn't touching a badge.
	function badgeAtCaret(): HTMLElement | null {
		const range = currentRange();
		if (!range || !range.collapsed || !surface) return null;
		const { startContainer, startOffset } = range;
		const isBadge = (n: Node | null): HTMLElement | null =>
			n && n.nodeType === Node.ELEMENT_NODE && (n as HTMLElement).classList.contains('ttc-badge')
				? (n as HTMLElement)
				: null;
		// Caret directly inside the surface element: check the children on either side of the offset.
		if (startContainer === surface) {
			return isBadge(surface.childNodes[startOffset] ?? null) ?? isBadge(surface.childNodes[startOffset - 1] ?? null);
		}
		// Caret inside a text node: check the element siblings touching the caret's edges.
		if (startContainer.nodeType === Node.TEXT_NODE) {
			const text = startContainer as Text;
			if (startOffset === 0) return isBadge(text.previousSibling);
			if (startOffset === (text.textContent ?? '').length) return isBadge(text.nextSibling);
		}
		return null;
	}

	// The node-array index of a given badge element (counts preceding non-empty text / br / badge
	// DOM children the same way serializeSurface does), so onselecttoken aligns with `template`.
	function serializeIndexOf(badge: HTMLElement): number {
		let idx = 0;
		if (!surface) return -1;
		for (const child of Array.from(surface.childNodes)) {
			if (child === badge) return idx;
			if (child.nodeType === Node.TEXT_NODE) {
				if ((child.textContent ?? '') !== '') idx++;
			} else if (child.nodeType === Node.ELEMENT_NODE) {
				const el = child as HTMLElement;
				if (el.tagName === 'BR' || el.classList.contains('ttc-badge') || (el.textContent ?? '') !== '') {
					idx++;
				}
			}
		}
		return -1;
	}

	// === Caret + insertion helpers ==========================================
	function currentRange(): Range | null {
		const sel = window.getSelection();
		if (!sel || sel.rangeCount === 0) return null;
		const range = sel.getRangeAt(0);
		// Only honour a selection that lives inside our surface (else insert appends at the end).
		if (surface && surface.contains(range.commonAncestorContainer)) return range;
		return null;
	}

	function insertTextAtCaret(text: string): boolean {
		surface?.focus();
		const range = currentRange();
		if (!range) {
			// No caret in-surface → append at the end.
			surface?.appendChild(document.createTextNode(text));
			placeCaretAtEnd();
			return true;
		}
		range.deleteContents();
		const node = document.createTextNode(text);
		range.insertNode(node);
		// Caret after the inserted text.
		range.setStartAfter(node);
		range.collapse(true);
		applyRange(range);
		return true;
	}

	// Insert a token badge at the caret (or end). Copies the catalog's declared format unless an
	// explicit one is given (drop path). Re-focuses + drops the caret AFTER the badge, then emits.
	function insertBadgeAtCaret(token: string, format?: string): void {
		if (!surface) return;
		const fmt = format ?? formatByToken.get(token);
		const badge = buildBadge(fmt ? { type: 'token', token, format: fmt } : { type: 'token', token });
		surface.focus();
		const range = currentRange();
		if (range) {
			range.deleteContents();
			range.insertNode(badge);
		} else {
			surface.appendChild(badge);
		}
		// A zero-width space after the badge gives the caret a text landing spot (so typing after a
		// trailing badge doesn't get swallowed into it); it serializes away as an empty text run.
		const after = document.createTextNode('​');
		badge.after(after);
		const sel = window.getSelection();
		const r = document.createRange();
		r.setStartAfter(after);
		r.collapse(true);
		sel?.removeAllRanges();
		sel?.addRange(r);
		emit();
	}

	function placeCaretAtEnd(): void {
		if (!surface) return;
		const range = document.createRange();
		range.selectNodeContents(surface);
		range.collapse(false);
		applyRange(range);
	}

	function placeCaretFromPoint(x: number, y: number): void {
		surface?.focus();
		// caretRangeFromPoint (WebKit/Blink) / caretPositionFromPoint (Gecko); fall back to end.
		const doc = document as Document & {
			caretRangeFromPoint?: (x: number, y: number) => Range | null;
			caretPositionFromPoint?: (x: number, y: number) => { offsetNode: Node; offset: number } | null;
		};
		let range: Range | null = null;
		if (doc.caretRangeFromPoint) {
			range = doc.caretRangeFromPoint(x, y);
		} else if (doc.caretPositionFromPoint) {
			const pos = doc.caretPositionFromPoint(x, y);
			if (pos) {
				range = document.createRange();
				range.setStart(pos.offsetNode, pos.offset);
				range.collapse(true);
			}
		}
		if (range && surface?.contains(range.commonAncestorContainer)) {
			applyRange(range);
		} else {
			placeCaretAtEnd();
		}
	}

	function applyRange(range: Range): void {
		const sel = window.getSelection();
		sel?.removeAllRanges();
		sel?.addRange(range);
	}

	// Insert from the picker — fires on Select change; resets the picker after.
	function onInsertPick(value: string): void {
		if (value === '') return;
		insertBadgeAtCaret(value);
		insertValue = ''; // back to the placeholder — it's an action, not a persisted value
	}

	function removeActiveBadge(): void {
		if (activeBadge && surface?.contains(activeBadge)) {
			activeBadge.remove();
			emit();
		}
		closePopover();
	}

	function closePopover(): void {
		popoverOpen = false;
		activeBadge = null;
		activeToken = '';
		activeFormat = undefined;
		activeFormatOptions = undefined;
	}

	// === External sync =======================================================
	// Re-render ONLY when the incoming template differs structurally from what the DOM shows AND
	// the surface isn't focused — so a host-driven change (binding edit, undo) reflects without
	// stomping the caret during the user's own typing. The `surface` dep makes this run on mount.
	$effect(() => {
		const incoming = serializeTemplate(template);
		if (!surface) return;
		const focused = typeof document !== 'undefined' && document.activeElement === surface;
		if (incoming !== domSerialized && !focused) {
			renderNodes(template);
		}
	});

	// Backspace/Delete adjacent to a badge: the browser already treats a contenteditable=false
	// span as atomic (one Delete removes the whole badge), so we DON'T intercept the key — we just
	// let the resulting `input` emit. (Documented as an edge to verify in a real browser.)

	// Keep the live badge VALUES fresh when the resolver changes (e.g. the bound record updates)
	// even though the structure (serialized string) is identical — re-render off-focus so a value
	// like ฿7,000,000 → ฿7,200,000 reflects. Guarded by focus so it never interrupts typing.
	// `prevResolver` is intentionally a plain (non-reactive) latch — the effect tracks the
	// `resolver` PROP read below; the latch just remembers the last identity we rendered for.
	let prevResolver: TokenResolver | null = null;
	$effect(() => {
		const r = resolver; // tracked dep — re-runs when the host swaps the resolver
		if (!surface) return;
		if (r === prevResolver) return;
		prevResolver = r;
		const focused = typeof document !== 'undefined' && document.activeElement === surface;
		if (!focused) renderNodes(template);
	});

	// Belt-and-braces: after the first paint, ensure the surface reflects the initial template.
	$effect(() => {
		if (surface && domSerialized === '' && template.length > 0) {
			tick().then(() => renderNodes(template));
		}
	});
</script>

<!-- The editor body is a snippet so the SAME surface renders inline OR inside a full-screen Modal
     (expand #1). The template prop is the source of truth, so the surface re-renders from it on
     toggle — no two-instance / shared-DOM conflict. -->
{#snippet editorBody()}
	<div class="ttc" class:ttc--expanded={expanded}>
		<div class="ttc__toolbar">
			<span class="ttc__toolbar-label">Insert field</span>
			<div class="ttc__insert">
				<Select
					options={insertOptions}
					bind:value={insertValue}
					placeholder="+ Add field…"
					size="sm"
					aria-label="Insert a field as a badge"
					onchange={onInsertPick}
				/>
			</div>
			<button
				type="button"
				class="ttc__expand"
				title={expanded ? 'Collapse' : 'Expand to full screen'}
				aria-label={expanded ? 'Collapse the editor' : 'Expand the editor to full screen'}
				onclick={() => (expanded = !expanded)}
			>
				<Icon name={expanded ? 'minimize-2' : 'maximize-2'} size="sm" />
			</button>
		</div>

		<div class="ttc__surface-wrap" class:is-empty={isEmpty}>
			<div
				bind:this={surface}
				class="ttc__surface"
				role="textbox"
				tabindex="0"
				contenteditable="true"
				aria-multiline="true"
				aria-label="Template editor"
				data-placeholder={placeholder ?? ''}
				oninput={onInput}
				onpaste={onPaste}
				ondrop={onDrop}
				ondragover={onDragOver}
				onclick={onSurfaceClick}
				onkeydown={onSurfaceKeydown}
				onblur={() => setTimeout(closeMention, 120)}
			></div>
			{#if isEmpty && placeholder}
				<span class="ttc__placeholder" aria-hidden="true">{placeholder}</span>
			{/if}
		</div>
	</div>
{/snippet}

<!-- Expand is a CSS overlay (NOT a node swap): the SAME surface element grows to full screen, so
     its content + caret persist (recreating it would drop the rendered badges). Backdrop sits below
     the overlay; both stay under the badge popover Modal (z-index 1000). -->
{#if expanded}
	<button type="button" class="ttc-backdrop" aria-label="Collapse the editor" onclick={() => (expanded = false)}></button>
{/if}
{@render editorBody()}

<!-- @-mention menu — fixed under the caret; mousedown (not click) so picking keeps the surface
     focused and runs before the blur-driven close. Type "@" then filter by field or source. -->
{#if mentionOpen && mentionItems.length > 0}
	<div class="ttc-mention" style="left: {mentionX}px; top: {mentionY}px;" role="listbox" aria-label="Insert a field">
		{#each mentionItems as item, i (item.token)}
			<button
				type="button"
				class="ttc-mention__item"
				class:ttc-mention__item--active={i === mentionIndex}
				role="option"
				aria-selected={i === mentionIndex}
				onmousedown={(e) => {
					e.preventDefault();
					selectMention(item.token);
				}}
			>
				<span class="ttc-mention__label">{item.label}</span>
				{#if item.group}<span class="ttc-mention__group">{item.group}</span>{/if}
			</button>
		{/each}
	</div>
{/if}

<Modal open={popoverOpen} title="Field" onclose={closePopover}>
	<div class="ttc-pop">
		<div class="ttc-pop__row">
			<span class="ttc-pop__key">Label</span>
			<span class="ttc-pop__val">{activeLabel}</span>
		</div>
		<!-- Reattach (#1) — point this badge at another source / field. -->
		<div class="ttc-pop__row">
			<span class="ttc-pop__key">Field</span>
			<div class="ttc-pop__control">
				<Select options={insertOptions} value={activeToken} size="sm" aria-label="Bound field" onchange={reattachActiveBadge} />
			</div>
		</div>
		<!-- Format (#1) — the host's rich knobs (the reused Inspector Format block) when provided,
		     else a read-only hint. -->
		{#if badgeFormat}
			{@render badgeFormat(badgeFormatContext)}
		{:else}
			<div class="ttc-pop__row">
				<span class="ttc-pop__key">Format</span>
				<span class="ttc-pop__val">{activeFormat ?? '—'}</span>
			</div>
		{/if}
		<div class="ttc-pop__row">
			<span class="ttc-pop__key">Preview</span>
			<span class="ttc-pop__val">{activeValue !== '' ? activeValue : `[${activeToken}]`}</span>
		</div>

		<!-- Destructive action lives in the BODY, left-aligned (the footer holds the Done CTA,
		     which stays right-aligned per the hard rule — mirrors AnimationInfoPopover). -->
		<div class="ttc-pop__delete">
			<button class="ttc-pop__remove" type="button" onclick={removeActiveBadge}>Remove field</button>
		</div>
	</div>

	{#snippet footer()}
		<Button variant="secondary" size="sm" onclick={closePopover}>Done</Button>
	{/snippet}
</Modal>

<style>
	.ttc {
		display: flex;
		flex-direction: column;
		gap: var(--cv-space-sm, 0.5rem);
	}

	/* Expand (#1) — the composer grows to a centred full-screen panel (same surface element,
	   just restyled), over a dimmed backdrop, beneath the badge popover Modal (z 1000). */
	.ttc--expanded {
		position: fixed;
		inset: 2.5rem;
		z-index: 900;
		padding: 1.25rem;
		background: var(--cv-color-surface, #fff);
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-lg, 0.75rem);
		box-shadow: 0 24px 64px rgba(0, 0, 0, 0.28);
		overflow: auto;
	}

	.ttc--expanded .ttc__surface {
		min-height: calc(100vh - 12rem);
	}

	.ttc-backdrop {
		position: fixed;
		inset: 0;
		z-index: 899;
		border: none;
		padding: 0;
		background: rgba(0, 0, 0, 0.45);
		cursor: pointer;
	}

	/* Toolbar — label + insert picker + the expand-to-full-screen button. */
	.ttc__toolbar {
		display: flex;
		align-items: center;
		gap: var(--cv-space-sm, 0.5rem);
	}

	.ttc__toolbar-label {
		font-size: 0.6875rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--cv-color-neutral-500, #707070);
		white-space: nowrap;
	}

	.ttc__insert {
		flex: 1;
		min-width: 8rem;
	}

	/* Expand / collapse — a quiet square button, right of the insert picker. */
	.ttc__expand {
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		width: 1.75rem;
		height: 1.75rem;
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: var(--cv-color-surface, #fff);
		color: var(--cv-color-neutral-500, #707070);
		cursor: pointer;
	}

	.ttc__expand:hover {
		background: var(--cv-color-neutral-100, #f5f5f5);
		color: var(--cv-color-neutral-700, #383838);
	}

	/* @-mention menu — fixed under the caret; source → field suggestions. */
	.ttc-mention {
		position: fixed;
		z-index: 1000;
		min-width: 13rem;
		max-width: 20rem;
		max-height: 16rem;
		overflow-y: auto;
		padding: 0.25rem;
		background: var(--cv-color-surface, #fff);
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-md, 0.5rem);
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.14);
	}

	.ttc-mention__item {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 0.0625rem;
		width: 100%;
		padding: 0.3125rem 0.5rem;
		border: none;
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: transparent;
		text-align: left;
		cursor: pointer;
	}

	.ttc-mention__item--active,
	.ttc-mention__item:hover {
		background: var(--cv-accent-soft, rgba(37, 99, 235, 0.1));
	}

	.ttc-mention__label {
		font-size: 0.8125rem;
		font-weight: 600;
		color: var(--cv-color-neutral-800, #1f1f1f);
	}

	.ttc-mention__group {
		font-size: 0.6875rem;
		color: var(--cv-color-neutral-500, #707070);
	}

	/* The editable surface. position:relative anchors the empty-state placeholder overlay. */
	.ttc__surface-wrap {
		position: relative;
	}

	.ttc__surface {
		min-height: 4.5rem;
		padding: var(--cv-space-md, 0.75rem);
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-md, 0.5rem);
		background: var(--cv-color-surface, #fff);
		color: var(--cv-color-neutral-800, #1f1f1f);
		font-size: 0.9375rem;
		line-height: 2.1; /* generous so inline badges never collide vertically */
		/* Inline badges flow with the text. Wrapping is fine; text is NEVER clipped/truncated. */
		white-space: pre-wrap;
		word-break: break-word;
		overflow-wrap: anywhere;
		cursor: text;
		outline: none;
		transition: border-color 0.15s ease, box-shadow 0.15s ease;
	}

	.ttc__surface:focus-visible,
	.ttc__surface:focus {
		border-color: var(--cv-accent, #2563eb);
		box-shadow: 0 0 0 3px var(--cv-accent-soft, rgba(37, 99, 235, 0.15));
	}

	/* Empty-state hint — an overlay (not the contenteditable's own text, which would be a real
	   node the user edits). Sits under the caret, non-interactive. */
	.ttc__placeholder {
		position: absolute;
		top: var(--cv-space-md, 0.75rem);
		left: var(--cv-space-md, 0.75rem);
		right: var(--cv-space-md, 0.75rem);
		color: var(--cv-color-neutral-400, #9a9a9a);
		font-size: 0.9375rem;
		line-height: 2.1;
		pointer-events: none;
		user-select: none;
	}

	/* === The badge (imperatively built by buildBadge — these classes are its sole styling) ===
	   A COMPACT pill/chip: muted TITLE on top, RESOLVED value below, kept to ONE short line so a
	   long binding never balloons the editor (#1). The chip is an editor CONTROL, not output — the
	   FULL value is preserved on the title/aria tooltip, in the ⚙ popover preview, and on the canvas
	   render — so truncating the on-chip preview drops no content (distinct from the never-clip rule,
	   which governs PUBLISHED text). Atomic (contenteditable=false). */
	.ttc :global(.ttc-badge) {
		position: relative;
		display: inline-flex;
		flex-direction: row;
		align-items: baseline;
		gap: 0.3rem;
		/* Cap the chip well under the surface width so a caret-clickable gutter always
		   remains beside it (#1a/#1b) — the value ellipsis absorbs the rest (#1c). */
		max-width: 12rem;
		vertical-align: baseline;
		margin: 0 0.1875rem;
		/* Extra right room reserves space for the ⚙ icon-CTA so it never overlaps the value. */
		padding: 0.0625rem 0.95rem 0.0625rem 0.4rem;
		border: 1px solid var(--cv-accent, #2563eb);
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: var(--cv-accent-soft, rgba(37, 99, 235, 0.1));
		color: var(--cv-accent-strong, #1d4ed8);
		line-height: 1.3;
		cursor: pointer;
		user-select: none;
		/* Keep the caret OUT of the badge: it's a single atomic stop. */
		caret-color: transparent;
		transition: background 0.12s ease, border-color 0.12s ease, box-shadow 0.12s ease;
	}

	.ttc :global(.ttc-badge:hover) {
		background: var(--cv-accent-soft-strong, rgba(37, 99, 235, 0.18));
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
	}

	/* Icon-CTA (#1) — the ⚙ in the badge's right gutter. pointer-events:none so a click falls
	   through to the badge (→ openBadge); shown subtly, full on hover/focus for discoverability. */
	.ttc :global(.ttc-badge__cog) {
		position: absolute;
		top: 50%;
		right: 3px;
		transform: translateY(-50%);
		display: inline-flex;
		width: 0.8125rem;
		height: 0.8125rem;
		color: var(--cv-accent, #2563eb);
		opacity: 0.4;
		pointer-events: none;
		transition: opacity 0.12s ease;
	}

	.ttc :global(.ttc-badge:hover .ttc-badge__cog),
	.ttc :global(.ttc-badge:focus .ttc-badge__cog) {
		opacity: 0.9;
	}

	.ttc :global(.ttc-badge__cog svg) {
		width: 100%;
		height: 100%;
	}

	.ttc :global(.ttc-badge__title) {
		/* The field label is the fixed anchor; the value (right) absorbs the trim. */
		flex: 0 0 auto;
		font-size: 0.5625rem;
		font-weight: 700;
		letter-spacing: 0.04em;
		text-transform: uppercase;
		opacity: 0.7;
		/* short label — ellipsis if a field label runs long (full value stays in the tooltip) */
		max-width: 5rem;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.ttc :global(.ttc-badge__value) {
		flex: 0 1 auto;
		min-width: 0;
		font-size: 0.6875rem;
		font-weight: 600;
		/* ONE compact line, a few words — long values truncate on the chip (#1c); the full
		   value stays in the title/aria tooltip, the ⚙ popover, and the canvas render. */
		max-width: 7rem;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	/* === Popover body (core-ui Modal) === */
	.ttc-pop {
		display: flex;
		flex-direction: column;
		gap: var(--cv-space-sm, 0.5rem);
	}

	.ttc-pop__row {
		display: grid;
		grid-template-columns: 5rem 1fr;
		align-items: center;
		gap: var(--cv-space-md, 1rem);
	}

	.ttc-pop__key {
		font-size: 0.6875rem;
		font-weight: 700;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: var(--cv-color-neutral-500, #707070);
	}

	.ttc-pop__val {
		font-size: 0.875rem;
		color: var(--cv-color-neutral-800, #1f1f1f);
		/* never clip the resolved preview */
		word-break: break-word;
	}

	.ttc-pop__control {
		min-width: 0;
	}

	/* Destructive action sits in the body, left-aligned (Modal footer is justify-content:flex-end,
	   so the Done CTA there stays right per the hard rule). Brand red only. */
	.ttc-pop__delete {
		display: flex;
		margin-top: var(--cv-space-sm, 0.5rem);
	}

	.ttc-pop__remove {
		display: inline-flex;
		align-items: center;
		padding: 0.375rem 0.875rem;
		border: 1px solid #e5392b;
		border-radius: var(--cv-radius-md, 0.5rem);
		background: transparent;
		color: #e5392b;
		font-size: 0.8125rem;
		font-weight: 600;
		cursor: pointer;
		transition: background 0.15s ease, color 0.15s ease;
	}

	.ttc-pop__remove:hover {
		background: #e5392b;
		color: #fff;
	}
</style>
