<script lang="ts">
	// EventPreviewPopover — a reusable, anchored event-detail popup (design-5, Google-
	// Calendar model). PACKAGE-OWNED so ANY surface can preview an event: the calendar grid,
	// a dashboard "today" widget, a property page's activity list. It renders a CalendarItem
	// and drives the contract verbs it's wired for — setStatus (status quick-set),
	// removeReference (chip ✕), plus Edit/Close — exactly the half of the contract the grid
	// alone can't reach. It imports NO data source and knows nothing about contacts/
	// properties: refs render generically by RefChip.relation, and ADD-reference (the only
	// host-specific, registry-driven part) is a snippet slot the host fills with its own
	// picker. `readonly` collapses it to a pure preview (dashboards). Positioning is the
	// unit-tested placePopover() helper; it dismisses on Esc / outside-click / scroll-away.
	import type { Snippet } from 'svelte';
	import type { CalendarItem, EventStatus, EventTypeStyle, PreviewAnchor } from './types.js';
	import { placePopover, type Placement } from './utils/popover.js';
	import { DEFAULT_TIME_ZONE } from './utils/timegrid.js';

	interface Props {
		item: CalendarItem;
		/** Viewport rect of the clicked event (from onItemClick). Omit → centered. */
		anchor?: PreviewAnchor;
		/** event_type → style + label (the same map the grid/agenda use) for the type dot. */
		typeColors?: Record<string, EventTypeStyle>;
		timeZone?: string;
		locale?: string;
		timeFormat?: '12h' | '24h';
		/** Preview-only: hide the status control, ref remove, add-ref slot and Edit. */
		readonly?: boolean;
		/**
		 * Render a dimmed full-screen backdrop behind the centered card (Google-Calendar-
		 * style modal). Clicking the backdrop or pressing Esc closes. Use with anchor=undefined
		 * so the card centers; an anchor + modal still centers the backdrop but anchors the card.
		 */
		modal?: boolean;
		/** "Open full details →" link target (the event's detail page). Renders a footer link when set. */
		detailHref?: string;
		/** "Add to calendar" export — an .ics download URL. Renders a footer download link when set. */
		icsHref?: string;
		/** Status quick-set → adapter.setStatus. Rendered (unless readonly) when provided. */
		onStatusChange?: (status: EventStatus) => void | Promise<void>;
		/** Remove a linked ref → adapter.removeReference. Renders chip ✕ when provided. */
		onRemoveReference?: (referenceId: string) => void | Promise<void>;
		/** Opens the host's editor (modal now, dedicated page later). Renders Edit when set. */
		onEdit?: () => void;
		onClose?: () => void;
		/** Host-specific add-reference control (its registry-driven picker) — package-agnostic. */
		addReference?: Snippet;
		/** Extra context fields (e.g. a property link) rendered under the meta. */
		extra?: Snippet;
	}

	let {
		item,
		anchor,
		typeColors,
		timeZone = DEFAULT_TIME_ZONE,
		locale = 'en-US',
		timeFormat = '12h',
		readonly = false,
		modal = false,
		detailHref,
		icsHref,
		onStatusChange,
		onRemoveReference,
		onEdit,
		onClose,
		addReference,
		extra
	}: Props = $props();

	const STATUS_LABELS: Record<EventStatus, string> = {
		scheduled: 'Scheduled',
		confirmed: 'Confirmed',
		done: 'Done',
		cancelled: 'Cancelled',
		no_show: 'No-show'
	};
	const STATUS_ORDER: EventStatus[] = ['scheduled', 'confirmed', 'done', 'cancelled', 'no_show'];

	const typeStyle = $derived(typeColors?.[item.eventType]);
	const accent = $derived(typeStyle?.accent ?? typeStyle?.text ?? 'var(--color-text-muted, #8a8a85)');
	const typeLabel = $derived(typeStyle?.label ?? item.eventType);
	const title = $derived(item.title?.trim() ? item.title : item.eventType);

	function timeOpts(): Intl.DateTimeFormatOptions {
		return timeFormat === '24h'
			? { hour: '2-digit', minute: '2-digit', hour12: false, timeZone }
			: { hour: 'numeric', minute: '2-digit', hour12: true, timeZone };
	}
	const fmtTime = (iso: string) => new Intl.DateTimeFormat(locale, timeOpts()).format(new Date(iso));
	const dayLabel = $derived(
		new Intl.DateTimeFormat(locale, {
			weekday: 'short',
			month: 'long',
			day: 'numeric',
			timeZone
		}).format(new Date(item.start))
	);
	const whenText = $derived(
		item.allDay
			? `${dayLabel} · All day`
			: `${dayLabel} · ${fmtTime(item.start)} – ${fmtTime(item.end)}`
	);

	let busy = $state(false);
	async function selectStatus(e: Event) {
		const next = (e.currentTarget as HTMLSelectElement).value as EventStatus;
		if (next === item.status || !onStatusChange) return;
		busy = true;
		try {
			await onStatusChange(next);
		} finally {
			busy = false;
		}
	}
	async function removeRef(referenceId: string) {
		if (!onRemoveReference) return;
		busy = true;
		try {
			await onRemoveReference(referenceId);
		} finally {
			busy = false;
		}
	}

	// ── Positioning (pure placePopover) + dismiss ────────────────────────────────
	let cardEl = $state<HTMLElement | null>(null);
	let pos = $state<Placement | null>(null);

	$effect(() => {
		const a = anchor; // track so re-anchoring (different event) repositions
		if (!cardEl) return;
		const el = cardEl;
		function place() {
			pos = placePopover(
				a,
				{ width: el.offsetWidth, height: el.offsetHeight },
				{ width: window.innerWidth, height: window.innerHeight }
			);
		}
		place();
		// Focus the card so Esc works immediately and screen readers announce the dialog.
		el.focus({ preventScroll: true });
		// Re-place when the card's own height changes (e.g. the add-ref picker expands) so a
		// grown card clamps back into the viewport instead of spilling past the bottom edge.
		const ro = new ResizeObserver(place);
		ro.observe(el);
		return () => ro.disconnect();
	});

	$effect(() => {
		function onKey(e: KeyboardEvent) {
			if (e.key === 'Escape') onClose?.();
		}
		// pointerdown (capture) attaches AFTER the opening click resolved, so it only catches
		// a subsequent press; close when it lands outside the card.
		function onPointer(e: PointerEvent) {
			if (cardEl && !cardEl.contains(e.target as Node)) onClose?.();
		}
		// A scroll OUTSIDE the card detaches the anchor → close; inner scroll is ignored.
		function onScroll(e: Event) {
			if (cardEl && cardEl.contains(e.target as Node)) return;
			onClose?.();
		}
		function onResize() {
			onClose?.();
		}
		window.addEventListener('keydown', onKey);
		window.addEventListener('pointerdown', onPointer, true);
		window.addEventListener('scroll', onScroll, true);
		window.addEventListener('resize', onResize);
		return () => {
			window.removeEventListener('keydown', onKey);
			window.removeEventListener('pointerdown', onPointer, true);
			window.removeEventListener('scroll', onScroll, true);
			window.removeEventListener('resize', onResize);
		};
	});
</script>

{#snippet personIcon()}
	<svg class="pp-ic" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
		<circle cx="12" cy="8" r="4" /><path d="M4 20c0-4 4-6 8-6s8 2 8 6" stroke-linecap="round" />
	</svg>
{/snippet}
{#snippet pinIcon()}
	<svg class="pp-ic" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
		<path d="M12 21s-6-5.5-6-10a6 6 0 0 1 12 0c0 4.5-6 10-6 10z" stroke-linejoin="round" /><circle cx="12" cy="11" r="2.5" />
	</svg>
{/snippet}
{#snippet tagIcon()}
	<svg class="pp-ic" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
		<path d="M20.6 13.4 11 3.8a2 2 0 0 0-1.4-.6H4v5.6a2 2 0 0 0 .6 1.4l9.6 9.6a2 2 0 0 0 2.8 0l3.6-3.6a2 2 0 0 0 0-2.8z" stroke-linejoin="round" /><circle cx="7.5" cy="7.5" r="1.2" />
	</svg>
{/snippet}

{#if modal}
	<!-- Dimmed full-screen backdrop (Google-Calendar modal). Click closes; aria-hidden
	     because the dialog itself carries the accessible name. The outside-click handler
	     also fires, but the backdrop makes the dismiss target explicit + visible. -->
	<div class="pp-backdrop" aria-hidden="true" onclick={() => onClose?.()}></div>
{/if}

<div
	bind:this={cardEl}
	class="pp"
	class:pp--modal={modal}
	role="dialog"
	aria-label="Event preview"
	tabindex="-1"
	style="top: {pos?.top ?? 0}px; left: {pos?.left ?? 0}px; visibility: {pos ? 'visible' : 'hidden'};"
>
	<header class="pp-head">
		<span class="pp-type"><span class="pp-dot" style="background: {accent};"></span>{typeLabel}</span>
		<span class="pp-head-actions">
			{#if onEdit && !readonly}
				<button type="button" class="pp-iconbtn" aria-label="Edit event" onclick={onEdit}>
					<svg class="pp-ic" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
						<path d="M4 20h4L18 10l-4-4L4 16v4z" stroke-linejoin="round" /><path d="M13.5 6.5l4 4" />
					</svg>
				</button>
			{/if}
			{#if onClose}
				<button type="button" class="pp-iconbtn" aria-label="Close preview" onclick={onClose}>
					<svg class="pp-ic" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
						<path d="M6 6l12 12M18 6 6 18" stroke-linecap="round" />
					</svg>
				</button>
			{/if}
		</span>
	</header>

	<h2 class="pp-title">{title}</h2>
	<p class="pp-when">{whenText}</p>

	{#if !readonly && onStatusChange}
		<label class="pp-status">
			<span class="pp-status-dot" data-status={item.status}></span>
			<select value={item.status} disabled={busy} onchange={selectStatus} aria-label="Change status">
				{#each STATUS_ORDER as s (s)}
					<option value={s}>{STATUS_LABELS[s]}</option>
				{/each}
			</select>
		</label>
	{:else}
		<p class="pp-status-ro"><span class="pp-status-dot" data-status={item.status}></span>{STATUS_LABELS[item.status]}</p>
	{/if}

	{#if item.refs.length}
		<ul class="pp-refs">
			{#each item.refs as ref (ref.referenceId)}
				<li class="pp-ref">
					<span class="pp-ref-ic">
						{#if ref.relation === 'attendee'}{@render personIcon()}
						{:else if ref.relation === 'about'}{@render pinIcon()}
						{:else}{@render tagIcon()}{/if}
					</span>
					<span class="pp-ref-label">{ref.label}{#if ref.asType}<span class="pp-ref-as"> · {ref.asType}</span>{/if}</span>
					{#if !readonly && onRemoveReference}
						<button
							type="button"
							class="pp-ref-remove"
							aria-label={`Remove ${ref.label}`}
							disabled={busy}
							onclick={() => removeRef(ref.referenceId)}
						>
							<svg class="pp-ic" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
								<path d="M6 6l12 12M18 6 6 18" stroke-linecap="round" />
							</svg>
						</button>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}

	{#if addReference && !readonly}
		<div class="pp-addref">{@render addReference()}</div>
	{/if}

	{#if item.notes}
		<p class="pp-notes">{item.notes}</p>
	{/if}

	{#if extra}
		<div class="pp-extra">{@render extra()}</div>
	{/if}

	{#if detailHref || icsHref}
		<!-- Footer bar — drill-down to the full detail page + an .ics export download.
		     Right-aligned (CTA discipline); the export carries `download` so the browser
		     saves the file rather than navigating to it. -->
		<footer class="pp-foot">
			{#if icsHref}
				<a class="pp-foot-link" href={icsHref} download>
					<svg class="pp-ic" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
						<path d="M12 3v12" stroke-linecap="round" /><path d="M7 12l5 5 5-5" stroke-linecap="round" stroke-linejoin="round" /><path d="M5 21h14" stroke-linecap="round" />
					</svg>
					Add to calendar
				</a>
			{/if}
			{#if detailHref}
				<a class="pp-foot-link pp-foot-link--primary" href={detailHref}>Open full details →</a>
			{/if}
		</footer>
	{/if}
</div>

<style>
	/* Generic tokens only (--color-* / --font-heading) so a host themes via its own bridge —
	   identical token surface to AgendaList. design-5 card: hairline border, soft shadow. */
	.pp {
		position: fixed;
		z-index: 50;
		width: 19rem;
		max-width: calc(100vw - 1rem);
		/* Never spill past the viewport: cap height + scroll internally (the dismiss
		   handler ignores scrolls inside the card, so this inner scroll won't close it). */
		max-height: calc(100vh - 1rem);
		overflow-y: auto;
		display: flex;
		flex-direction: column;
		gap: 0.55rem;
		padding: 0.85rem 0.9rem;
		background: var(--color-surface, #ffffff);
		border: 1px solid var(--color-border, #e4e4e2);
		border-radius: 0.75rem;
		box-shadow: 0 10px 30px rgba(0, 0, 0, 0.12);
	}
	.pp:focus-visible {
		outline: none;
	}

	/* ── Modal mode (centered card + dimmed backdrop) ────────────────────────────
	   The backdrop sits just below the card; both above the app. In modal mode the
	   centered card reads as a true dialog — a touch wider, a stronger shadow, and a
	   ring so it lifts off the dim. */
	.pp-backdrop {
		position: fixed;
		inset: 0;
		z-index: 49;
		background: rgba(15, 15, 12, 0.45);
	}
	.pp--modal {
		width: 22rem;
		box-shadow: 0 18px 50px rgba(0, 0, 0, 0.3);
	}

	.pp-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem;
	}
	.pp-type {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		font-size: 0.72rem;
		font-weight: 600;
		text-transform: capitalize;
		color: var(--color-text-muted, #707068);
	}
	.pp-dot {
		width: 0.55rem;
		height: 0.55rem;
		border-radius: 50%;
		flex: 0 0 auto;
	}
	.pp-head-actions {
		display: inline-flex;
		gap: 0.15rem;
	}
	.pp-iconbtn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 1.8rem;
		height: 1.8rem;
		color: var(--color-text-muted, #8a8a85);
		background: transparent;
		border: 1px solid transparent;
		border-radius: 0.45rem;
		cursor: pointer;
	}
	.pp-iconbtn:hover {
		color: var(--color-text, #1d1d1b);
		border-color: var(--color-border, #e4e4e2);
		background: var(--color-bg, #ffffff);
	}
	.pp-iconbtn:focus-visible {
		outline: 2px solid var(--color-text, #1d1d1b);
		outline-offset: 1px;
	}

	.pp-title {
		margin: 0;
		font-family: var(--font-heading, Georgia, serif);
		font-size: 1.05rem;
		font-weight: 700;
		line-height: 1.25;
		color: var(--color-text, #1d1d1b);
		/* NEVER truncate. */
		overflow-wrap: anywhere;
	}
	.pp-when {
		margin: 0;
		font-size: 0.82rem;
		color: var(--color-text-muted, #707068);
	}

	/* Status — a styled native select (the 5 soft states) + a state-coloured dot. */
	.pp-status {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		align-self: flex-start;
		padding: 0.05rem 0.1rem 0.05rem 0.45rem;
		border: 1px solid var(--color-border, #e4e4e2);
		border-radius: 0.45rem;
	}
	.pp-status select {
		appearance: none;
		border: none;
		background: transparent;
		font: inherit;
		font-size: 0.78rem;
		font-weight: 600;
		color: var(--color-text, #1d1d1b);
		padding: 0.25rem 1.4rem 0.25rem 0.1rem;
		cursor: pointer;
		/* chevron */
		background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='%23707068' stroke-width='2' stroke-linecap='round'%3E%3Cpath d='M6 9l6 6 6-6'/%3E%3C/svg%3E");
		background-repeat: no-repeat;
		background-position: right 0.35rem center;
		background-size: 0.85rem;
	}
	.pp-status select:disabled {
		opacity: 0.6;
		cursor: default;
	}
	.pp-status-ro {
		margin: 0;
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		font-size: 0.8rem;
		font-weight: 600;
		color: var(--color-text, #1d1d1b);
	}
	.pp-status-dot {
		width: 0.5rem;
		height: 0.5rem;
		border-radius: 50%;
		flex: 0 0 auto;
		background: var(--color-text-muted, #8a8a85);
	}
	.pp-status-dot[data-status='scheduled'] {
		background: var(--color-info, #2563eb);
	}
	.pp-status-dot[data-status='confirmed'] {
		background: var(--color-success, #16a34a);
	}
	.pp-status-dot[data-status='done'] {
		background: var(--color-text-muted, #6b7280);
	}
	.pp-status-dot[data-status='cancelled'] {
		background: var(--color-error, #dc2626);
	}
	.pp-status-dot[data-status='no_show'] {
		background: var(--color-warning, #d97706);
	}

	.pp-refs {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0.3rem;
	}
	.pp-ref {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		font-size: 0.8rem;
		color: var(--color-text, #1d1d1b);
	}
	.pp-ref-ic {
		display: inline-flex;
		color: var(--color-text-muted, #8a8a85);
		flex: 0 0 auto;
	}
	.pp-ref-label {
		flex: 1 1 auto;
		min-width: 0;
		overflow-wrap: anywhere;
	}
	.pp-ref-as {
		color: var(--color-text-muted, #8a8a85);
	}
	.pp-ref-remove {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 1.4rem;
		height: 1.4rem;
		flex: 0 0 auto;
		color: var(--color-text-muted, #8a8a85);
		background: transparent;
		border: none;
		border-radius: 50%;
		cursor: pointer;
	}
	.pp-ref-remove:hover:not(:disabled) {
		color: #ffffff;
		background: var(--color-error, #c0392b);
	}
	.pp-ref-remove:disabled {
		opacity: 0.5;
		cursor: default;
	}

	.pp-addref {
		margin-top: 0.1rem;
	}

	.pp-notes {
		margin: 0;
		font-size: 0.82rem;
		line-height: 1.45;
		color: var(--color-text, #1d1d1b);
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}

	/* ── Footer bar — detail drill-down + .ics export. Right-aligned (CTA discipline). ── */
	.pp-foot {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		flex-wrap: wrap;
		gap: 0.4rem 0.65rem;
		margin-top: 0.15rem;
		padding-top: 0.6rem;
		border-top: 1px solid var(--color-border, #e4e4e2);
	}
	.pp-foot-link {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		font-size: 0.78rem;
		font-weight: 600;
		text-decoration: none;
		color: var(--color-text-muted, #707068);
		padding: 0.3rem 0.55rem;
		border-radius: 0.45rem;
	}
	.pp-foot-link:hover {
		color: var(--color-text, #1d1d1b);
		background: var(--color-bg, #f5f5f0);
	}
	.pp-foot-link:focus-visible {
		outline: 2px solid var(--color-text, #1d1d1b);
		outline-offset: 1px;
	}
	.pp-foot-link--primary {
		color: var(--color-accent, #0d4f4f);
	}
	.pp-foot-link--primary:hover {
		color: var(--color-accent, #0d4f4f);
		background: color-mix(in srgb, var(--color-accent, #0d4f4f) 10%, transparent);
	}

	.pp-ic {
		width: 0.95rem;
		height: 0.95rem;
		flex: 0 0 auto;
	}
</style>
