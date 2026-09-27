<script lang="ts">
	// DatePicker — a compact mini-month popover for jumping the calendar's focus
	// date (Finding A1). PRESENTATION ONLY: it renders a month grid from the pure
	// `monthMatrix` util (NO new dependency) and emits the picked day; it imports no
	// data source. The host (Calendar) decides what "jump" means per active view.
	//
	// Its own ‹ › nav walks the displayed month independently of the calendar's
	// focus date, so a user can browse months without committing. Picking a day
	// emits onSelect(dayISO) and closes. Esc closes; a click outside closes; each
	// day is a focusable button so Tab+Enter selects (arrow-key grid nav is a
	// tracked a11y follow-up). Brand red #e5392b is the only accent.

	import { onMount, untrack } from 'svelte';
	import { monthMatrix, stepFocus } from './utils/timegrid.js';

	interface Props {
		/** The currently-selected focus day (bare YYYY-MM-DD or any ISO instant). */
		value: string;
		/** Emitted when a day is picked — a bare YYYY-MM-DD ISO. */
		onSelect: (dayISO: string) => void;
		/** Emitted when the popover should close (Esc, outside click, after pick). Optional
		 *  in `inline` mode (an embedded mini-month never closes). */
		onClose?: () => void;
		timeZone: string;
		locale?: string;
		/**
		 * The trigger element that toggles this popover. Pointer-downs on it are IGNORED
		 * by the outside-click handler, so the host's own toggle owns close-on-trigger —
		 * otherwise the outside-close + the trigger's toggle would fight and re-open.
		 */
		anchorEl?: HTMLElement | null;
		/**
		 * Inline mode — render as an embedded mini-month (Agenda view) instead of a floating
		 * popover: no shadow/outside-click/focus-grab, fills its container, never closes.
		 */
		inline?: boolean;
		/**
		 * Per-day event indicator dots: bare YYYY-MM-DD → a CSS colour. A day present here
		 * gets a small dot beneath its number (e.g. the day's dominant event-type accent).
		 */
		dayDots?: Record<string, string>;
	}

	let {
		value,
		onSelect,
		onClose,
		timeZone,
		locale = 'en-US',
		anchorEl = null,
		inline = false,
		dayDots
	}: Props = $props();

	const WEEKDAY_LABELS = ['S', 'M', 'T', 'W', 'T', 'F', 'S'];

	// The month the popover currently displays — starts on the selected value's
	// month, then walks independently via the ‹ › nav (without committing). The
	// popover is re-mounted on each open ({#if pickerOpen} in the host), so this
	// initial seed from `value` is intentionally a one-time snapshot per open.
	let viewDate = $state(untrack(() => value));
	const cells = $derived(monthMatrix(viewDate, timeZone, locale));

	const monthTitle = $derived.by(() => {
		try {
			const anchor = cells.find((c) => c.inMonth)?.date;
			if (!anchor) return '';
			return new Intl.DateTimeFormat(locale, { month: 'long', year: 'numeric', timeZone }).format(
				anchor.toDate(timeZone)
			);
		} catch {
			return '';
		}
	});

	// The bare YYYY-MM-DD of the selected value, for highlighting.
	const selectedISO = $derived(value.slice(0, 10));

	function prevMonth() {
		viewDate = stepFocus(viewDate, timeZone, 'month', -1);
	}
	function nextMonth() {
		viewDate = stepFocus(viewDate, timeZone, 'month', 1);
	}

	function pick(dayISO: string) {
		onSelect(dayISO);
		if (!inline) onClose?.();
	}

	let rootEl = $state<HTMLElement | null>(null);

	function onRootKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			if (inline) return; // embedded mini-month: Esc isn't a close
			event.stopPropagation();
			onClose?.();
			return;
		}
		// Roving arrow-key grid navigation over the day buttons: ←/→ step a day, ↑/↓ a
		// week, Home/End jump to the month's first/last day. Enter/Space select natively
		// (the focused element is a <button>), so we only move focus here.
		const NAV: Record<string, number> = {
			ArrowLeft: -1,
			ArrowRight: 1,
			ArrowUp: -7,
			ArrowDown: 7
		};
		if (!(event.key in NAV) && event.key !== 'Home' && event.key !== 'End') return;
		const days = rootEl ? Array.from(rootEl.querySelectorAll<HTMLButtonElement>('.dp-day')) : [];
		if (days.length === 0) return;
		const active = document.activeElement as HTMLElement | null;
		const current = active ? days.indexOf(active as HTMLButtonElement) : -1;
		let next: number;
		if (event.key === 'Home') next = 0;
		else if (event.key === 'End') next = days.length - 1;
		else if (current === -1) next = 0; // focus wasn't on a day yet → enter the grid
		else next = current + NAV[event.key];
		if (next < 0 || next >= days.length) return; // don't wrap past the visible grid edges
		event.preventDefault();
		days[next].focus();
	}

	// Focus-in on open: land focus on the selected day (else today, else the month's
	// first day) so the popover is immediately keyboard-drivable and Esc/arrows work
	// without a preliminary Tab. The host returns focus to the trigger on close.
	onMount(() => {
		if (inline || !rootEl) return; // embedded: don't steal focus on mount
		const days = Array.from(rootEl.querySelectorAll<HTMLButtonElement>('.dp-day'));
		const target =
			rootEl.querySelector<HTMLButtonElement>('.dp-day.is-selected') ??
			rootEl.querySelector<HTMLButtonElement>('.dp-day.is-today') ??
			days.find((d) => !d.classList.contains('is-outside')) ??
			days[0];
		target?.focus();
	});

	// Outside-click closes the popover. The trigger (anchorEl) is excluded so the
	// host's toggle — not this handler — owns close-on-trigger (no fight/re-open).
	$effect(() => {
		if (inline) return; // embedded mini-month never closes on outside click
		function onDocPointerDown(event: PointerEvent) {
			if (!(event.target instanceof Node)) return;
			if (rootEl && rootEl.contains(event.target)) return;
			if (anchorEl && anchorEl.contains(event.target)) return;
			onClose?.();
		}
		document.addEventListener('pointerdown', onDocPointerDown, true);
		return () => document.removeEventListener('pointerdown', onDocPointerDown, true);
	});
</script>

<div
	class="dp"
	class:is-inline={inline}
	role={inline ? 'group' : 'dialog'}
	aria-label="Pick a date"
	bind:this={rootEl}
	onkeydown={onRootKeydown}
	tabindex="-1"
>
	<header class="dp-head">
		<button type="button" class="dp-nav" aria-label="Previous month" onclick={prevMonth}>‹</button>
		<span class="dp-title">{monthTitle}</span>
		<button type="button" class="dp-nav" aria-label="Next month" onclick={nextMonth}>›</button>
	</header>

	<div class="dp-weekdays" aria-hidden="true">
		{#each WEEKDAY_LABELS as label, i (i)}
			<span class="dp-weekday">{label}</span>
		{/each}
	</div>

	<div class="dp-grid" role="grid">
		{#each cells as cell (cell.date.toString())}
			{@const iso = cell.date.toString()}
			<button
				type="button"
				class="dp-day"
				class:is-outside={!cell.inMonth}
				class:is-today={cell.isToday}
				class:is-selected={iso === selectedISO}
				aria-label={iso}
				aria-current={iso === selectedISO ? 'date' : undefined}
				onclick={() => pick(iso)}
			>
				<span class="dp-day-num">{cell.date.day}</span>
				{#if dayDots?.[iso]}<span class="dp-dot" style="background: {dayDots[iso]};"></span>{/if}
			</button>
		{/each}
	</div>
</div>

<style>
	.dp {
		/* Themeable accent — mirrors Calendar's --cal-accent-color (BR sets it to the brand
		   gold). Defaults to brand red so the package stands alone unchanged. */
		--cal-today-ring: var(--cal-accent-color, #e5392b);
		--cal-text: var(--color-text, #1d1d1b);
		--cal-text-muted: var(--color-text-muted, #8a8a85);
		--cal-day-border: var(--color-border, #e4e4e2);

		width: 16rem;
		padding: 0.6rem;
		background: var(--color-bg, #ffffff);
		color: var(--cal-text);
		border: 1px solid var(--cal-day-border);
		border-radius: 0.6rem;
		box-shadow: 0 8px 28px rgba(0, 0, 0, 0.18);
		box-sizing: border-box;
	}

	.dp-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 0.4rem;
	}
	.dp-title {
		font-size: 0.9rem;
		font-weight: 600;
	}
	.dp-nav {
		appearance: none;
		width: 1.9rem;
		height: 1.9rem;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		font-size: 1.1rem;
		line-height: 1;
		color: var(--cal-text-muted);
		background: transparent;
		border: none;
		border-radius: 0.4rem;
		cursor: pointer;
		transition:
			background 0.12s ease,
			color 0.12s ease;
	}
	.dp-nav:hover {
		color: var(--cal-text);
		background: color-mix(in srgb, currentColor 8%, transparent);
	}
	.dp-nav:focus-visible {
		outline: 2px solid var(--cal-today-ring);
		outline-offset: 1px;
	}

	.dp-weekdays,
	.dp-grid {
		display: grid;
		grid-template-columns: repeat(7, 1fr);
	}
	.dp-weekday {
		padding: 0.2rem 0;
		font-size: 0.7rem;
		font-weight: 600;
		text-align: center;
		color: var(--cal-text-muted);
	}

	.dp-grid {
		gap: 1px;
	}
	.dp-day {
		appearance: none;
		position: relative;
		aspect-ratio: 1 / 1;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		font: inherit;
		font-size: 0.8rem;
		color: var(--cal-text);
		background: transparent;
		border: none;
		border-radius: 0.4rem;
		cursor: pointer;
		transition:
			background 0.12s ease,
			color 0.12s ease;
	}
	/* Inline (Agenda) mini-month: embedded, not a floating card. */
	.dp.is-inline {
		width: 100%;
		padding: 0;
		background: transparent;
		border: none;
		box-shadow: none;
	}
	/* Per-day event dot — beneath the number; hidden on the selected day's filled cell. */
	.dp-dot {
		position: absolute;
		bottom: 3px;
		left: 50%;
		transform: translateX(-50%);
		width: 4px;
		height: 4px;
		border-radius: 50%;
	}
	.dp-day.is-selected .dp-dot {
		display: none;
	}
	.dp-day:hover {
		background: color-mix(in srgb, currentColor 8%, transparent);
	}
	.dp-day:focus-visible {
		outline: 2px solid var(--cal-today-ring);
		outline-offset: -1px;
	}
	.dp-day.is-outside {
		color: var(--cal-text-muted);
	}
	.dp-day.is-today {
		font-weight: 700;
		color: var(--cal-today-ring);
	}
	.dp-day.is-selected {
		color: #ffffff;
		background: var(--cal-today-ring);
	}
	.dp-day.is-selected:hover {
		background: color-mix(in srgb, var(--cal-today-ring) 88%, #000);
	}
</style>
