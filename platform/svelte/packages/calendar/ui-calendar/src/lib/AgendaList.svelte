<script lang="ts">
	// AgendaList — a chronological list of CalendarItem[], used in TWO places from ONE
	// component (design-5): the month view's compact "Upcoming" rail (grouped=false) and
	// the Agenda view's day-grouped cards (grouped=true). PRESENTATION ONLY: it renders
	// items + emits onItemClick / onEdit; it imports NO data source and knows nothing about
	// contacts/properties — the person / place / tag lines are derived purely from the
	// generic RefChip.relation (attendee / about / material), so it stays decoupled and
	// works with attachment sources off. Colours + labels come from the host-supplied
	// typeColors map (the same map the Calendar grid uses); nothing is hard-coded.
	import type { Snippet } from 'svelte';
	import { today as currentDay } from '@internationalized/date';
	import type { CalendarItem, EventTypeStyle } from './types.js';
	import { dayHeading, groupByDay } from './utils/agenda.js';
	import { DEFAULT_TIME_ZONE } from './utils/timegrid.js';

	interface Props {
		items: CalendarItem[];
		/** event_type → muted style + label; supplies each row's accent, dot and type label. */
		typeColors?: Record<string, EventTypeStyle>;
		/** true → day-group headers + counts (Agenda body); false → flat list (Upcoming rail). */
		grouped?: boolean;
		/** Bare YYYY-MM-DD for the "Today ·" heading; defaults to the real day in timeZone. */
		today?: string;
		timeZone?: string;
		locale?: string;
		timeFormat?: '12h' | '24h';
		/** Copy shown when there are no items. */
		emptyHint?: string;
		onItemClick?: (item: CalendarItem) => void;
		/** Per-row edit shortcut (the ✎). Rendered only when provided. */
		onEdit?: (item: CalendarItem) => void;
		/** Optional host-owned trailing actions per row (e.g. a ⋯ menu). */
		rowActions?: Snippet<[CalendarItem]>;
	}

	let {
		items,
		typeColors,
		grouped = false,
		today,
		timeZone = DEFAULT_TIME_ZONE,
		locale = 'en-US',
		timeFormat = '12h',
		emptyHint,
		onItemClick,
		onEdit,
		rowActions
	}: Props = $props();

	const todayISO = $derived(today ?? currentDay(timeZone).toString());
	// Flat mode renders items as-given (the host windows/sorts the rail); grouped mode
	// buckets them by day (groupByDay sorts days + items deterministically).
	const groups = $derived(groupByDay(items));

	function timeOpts(): Intl.DateTimeFormatOptions {
		return timeFormat === '24h'
			? { hour: '2-digit', minute: '2-digit', hour12: false, timeZone }
			: { hour: 'numeric', minute: '2-digit', hour12: true, timeZone };
	}
	// Items are display-zone ISO with an explicit offset, so new Date(iso) is an unambiguous
	// instant and formatting in timeZone yields the wall-clock the agent expects.
	function fmtTime(iso: string): string {
		return new Intl.DateTimeFormat(locale, timeOpts()).format(new Date(iso));
	}
	// Flat (rail) rows need a day marker since the list spans many days: "Today" or "Jun 8".
	function shortDay(iso: string): string {
		if (iso.slice(0, 10) === todayISO) return 'Today';
		return new Intl.DateTimeFormat(locale, { month: 'short', day: 'numeric', timeZone }).format(
			new Date(iso)
		);
	}

	// The type accent (left bar + dot): per-event item.color (legacy solid) wins; else the
	// type map's accent/text; else a neutral fallback so an uncoloured host still reads.
	function accentOf(item: CalendarItem): string {
		if (item.color) return item.color;
		const style = typeColors?.[item.eventType];
		return style?.accent ?? style?.text ?? 'var(--color-text-muted, #8a8a85)';
	}
	// Human type label — the map's label, else the raw code (CSS capitalises the fallback).
	function typeLabelOf(item: CalendarItem): string {
		return typeColors?.[item.eventType]?.label ?? item.eventType;
	}
	// Title falls back to the event type when blank (matches the adapter default).
	function titleOf(item: CalendarItem): string {
		return item.title?.trim() ? item.title : item.eventType;
	}

	// Refs split by RELATION (generic — no entity-type knowledge):
	//   attendee → people · about → subject/place · material → context tags.
	const peopleOf = (item: CalendarItem) => item.refs.filter((r) => r.relation === 'attendee');
	const placesOf = (item: CalendarItem) => item.refs.filter((r) => r.relation === 'about');
	const tagsOf = (item: CalendarItem) => item.refs.filter((r) => r.relation === 'material');
</script>

{#snippet personIcon()}
	<svg class="ag-ic" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
		<circle cx="12" cy="8" r="4" /><path d="M4 20c0-4 4-6 8-6s8 2 8 6" stroke-linecap="round" />
	</svg>
{/snippet}
{#snippet pinIcon()}
	<svg class="ag-ic" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
		<path d="M12 21s-6-5.5-6-10a6 6 0 0 1 12 0c0 4.5-6 10-6 10z" stroke-linejoin="round" /><circle cx="12" cy="11" r="2.5" />
	</svg>
{/snippet}

{#snippet row(item: CalendarItem)}
	<li class="ag-item" data-status={item.status} style="--ag-accent: {accentOf(item)};">
		<button type="button" class="ag-row" onclick={() => onItemClick?.(item)}>
			<span class="ag-time">
				<span class="ag-time-start">{fmtTime(item.start)}</span>
				<span class="ag-time-sub">{grouped ? fmtTime(item.end) : shortDay(item.start)}</span>
			</span>
			<span class="ag-bar" aria-hidden="true"></span>
			<span class="ag-body">
				<span class="ag-tags">
					<span class="ag-type"><span class="ag-dot" aria-hidden="true"></span>{typeLabelOf(item)}</span>
					{#each tagsOf(item) as tag (tag.referenceId)}
						<span class="ag-tag">{tag.label}</span>
					{/each}
				</span>
				<span class="ag-title">{titleOf(item)}</span>
				{#if peopleOf(item).length || placesOf(item).length}
					<span class="ag-meta">
						{#each peopleOf(item) as p (p.referenceId)}
							<span class="ag-meta-item">{@render personIcon()}{p.label}{#if p.asType}<span class="ag-as"> · {p.asType}</span>{/if}</span>
						{/each}
						{#each placesOf(item) as pl (pl.referenceId)}
							<span class="ag-meta-item">{@render pinIcon()}{pl.label}</span>
						{/each}
					</span>
				{/if}
			</span>
		</button>
		{#if onEdit || rowActions}
			<span class="ag-actions">
				{#if onEdit}
					<button type="button" class="ag-action" aria-label={`Edit ${titleOf(item)}`} onclick={() => onEdit?.(item)}>
						<svg class="ag-ic" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
							<path d="M4 20h4L18 10l-4-4L4 16v4z" stroke-linejoin="round" /><path d="M13.5 6.5l4 4" />
						</svg>
					</button>
				{/if}
				{#if rowActions}{@render rowActions(item)}{/if}
			</span>
		{/if}
	</li>
{/snippet}

{#if items.length === 0}
	<p class="ag-empty">{emptyHint ?? 'Nothing scheduled.'}</p>
{:else if grouped}
	<div class="ag-groups">
		{#each groups as group (group.day)}
			<section class="ag-group">
				<header class="ag-group-head">
					<h3 class="ag-group-title">{dayHeading(group.day, todayISO, locale, timeZone)}</h3>
					<span class="ag-group-count">{group.items.length} event{group.items.length === 1 ? '' : 's'}</span>
				</header>
				<ul class="ag-list">
					{#each group.items as item (item.id)}{@render row(item)}{/each}
				</ul>
			</section>
		{/each}
	</div>
{:else}
	<ul class="ag-list ag-list-flat">
		{#each items as item (item.id)}{@render row(item)}{/each}
	</ul>
{/if}

<style>
	/* Generic tokens only (--color-* / --font-heading) so a host themes via its own
	   bridge; design-5 structure (borders, spacing, accent bar, mono time, serif title). */
	.ag-groups {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}
	.ag-group-head {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 0.75rem;
		margin-bottom: 0.6rem;
	}
	.ag-group-title {
		margin: 0;
		font-family: var(--font-heading, Georgia, serif);
		font-size: 1.02rem;
		font-weight: 700;
		color: var(--color-text, #1d1d1b);
	}
	.ag-group-count {
		font-size: 0.72rem;
		color: var(--color-text-muted, #8a8a85);
		white-space: nowrap;
	}

	.ag-list {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
	}
	.ag-list-flat {
		gap: 0;
	}

	.ag-item {
		display: flex;
		align-items: stretch;
		gap: 0.25rem;
	}
	/* Grouped → bordered cards; flat (rail) → hairline-separated rows. */
	.ag-groups .ag-item {
		border: 1px solid var(--color-border, #e4e4e2);
		border-radius: 0.7rem;
		background: var(--color-surface, #fff);
		transition:
			box-shadow 0.14s ease,
			border-color 0.14s ease;
	}
	.ag-groups .ag-item:hover {
		border-color: var(--color-border-strong, #d0d0c8);
		box-shadow: 0 4px 14px rgba(0, 0, 0, 0.06);
	}
	.ag-list-flat .ag-item {
		border-bottom: 1px solid var(--color-border, #e4e4e2);
	}
	.ag-list-flat .ag-item:last-child {
		border-bottom: none;
	}

	.ag-row {
		flex: 1 1 auto;
		min-width: 0;
		display: grid;
		grid-template-columns: auto auto 1fr;
		gap: 0.7rem;
		align-items: start;
		padding: 0.7rem 0.5rem 0.7rem 0.7rem;
		text-align: left;
		background: transparent;
		border: none;
		border-radius: inherit;
		cursor: pointer;
		font: inherit;
		color: inherit;
	}
	.ag-row:focus-visible {
		outline: 2px solid var(--ag-accent);
		outline-offset: -2px;
		border-radius: 0.5rem;
	}

	.ag-time {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
		min-width: 3.6rem;
		font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
	}
	.ag-time-start {
		font-size: 0.76rem;
		font-weight: 700;
		color: var(--color-text, #1d1d1b);
		white-space: nowrap;
	}
	.ag-time-sub {
		font-size: 0.68rem;
		color: var(--color-text-muted, #8a8a85);
		white-space: nowrap;
	}

	.ag-bar {
		width: 3px;
		align-self: stretch;
		border-radius: 2px;
		background: var(--ag-accent);
	}

	.ag-body {
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
	}
	.ag-tags {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.4rem;
	}
	.ag-type {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		font-size: 0.7rem;
		font-weight: 600;
		text-transform: capitalize;
		color: var(--color-text-muted, #707068);
	}
	.ag-dot {
		width: 0.5rem;
		height: 0.5rem;
		border-radius: 50%;
		background: var(--ag-accent);
		flex: 0 0 auto;
	}
	.ag-tag {
		font-size: 0.68rem;
		font-weight: 600;
		padding: 0.05rem 0.4rem;
		border-radius: 0.5rem;
		background: color-mix(in srgb, var(--ag-accent) 14%, transparent);
		color: color-mix(in srgb, var(--ag-accent) 75%, var(--color-text, #1d1d1b));
		/* NEVER clip — tags wrap fully. */
		overflow-wrap: anywhere;
	}
	.ag-title {
		font-family: var(--font-heading, Georgia, serif);
		font-size: 0.95rem;
		font-weight: 600;
		line-height: 1.3;
		color: var(--color-text, #1d1d1b);
		overflow-wrap: anywhere;
	}
	.ag-meta {
		display: flex;
		flex-wrap: wrap;
		gap: 0.15rem 0.85rem;
		margin-top: 0.1rem;
	}
	.ag-meta-item {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		font-size: 0.76rem;
		color: var(--color-text-muted, #707068);
		overflow-wrap: anywhere;
	}
	.ag-as {
		color: var(--color-text-muted, #8a8a85);
	}
	.ag-ic {
		width: 0.92rem;
		height: 0.92rem;
		flex: 0 0 auto;
		opacity: 0.8;
	}

	.ag-actions {
		display: flex;
		align-items: center;
		gap: 0.15rem;
		padding-right: 0.35rem;
	}
	.ag-action {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 1.9rem;
		height: 1.9rem;
		color: var(--color-text-muted, #8a8a85);
		background: transparent;
		border: 1px solid transparent;
		border-radius: 0.45rem;
		cursor: pointer;
	}
	.ag-action:hover {
		color: var(--color-text, #1d1d1b);
		border-color: var(--color-border, #e4e4e2);
		background: var(--color-bg, #fff);
	}
	.ag-action:focus-visible {
		outline: 2px solid var(--ag-accent);
		outline-offset: 1px;
	}

	/* Status: cancelled / no_show read as VOID (line-through + dim); done stays normal —
	   only a cancelled event is "greyed" per the contract. The colour stays the type accent
	   (design-5 keeps the hue, marks state via strike/opacity, never the base colour). */
	.ag-item[data-status='cancelled'] .ag-title,
	.ag-item[data-status='no_show'] .ag-title {
		text-decoration: line-through;
	}
	.ag-item[data-status='cancelled'],
	.ag-item[data-status='no_show'] {
		opacity: 0.7;
	}

	.ag-empty {
		margin: 0;
		font-size: 0.85rem;
		line-height: 1.5;
		color: var(--color-text-muted, #8a8a85);
	}
</style>
