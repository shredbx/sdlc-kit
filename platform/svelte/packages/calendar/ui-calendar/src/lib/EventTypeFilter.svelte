<script lang="ts">
	// EventTypeFilter — a checkbox list of event types (design-5/B Agenda sidebar). Each row
	// is a coloured dot + label + optional count; unchecking hides that type from the list it
	// drives. PRESENTATION ONLY + fully controlled: the host owns the `hidden` set and the
	// items; this just renders the toggles and emits onToggle(code). Domain-agnostic — the
	// types, labels, colours and counts all come from the host (the same typeColors map the
	// grid/agenda use), so it ships no palette and knows no event_type vocabulary.
	import type { EventTypeStyle } from './types.js';

	interface Props {
		/** Types to list, in display order (code + human label). */
		types: { code: string; label: string }[];
		/** event_type → style; supplies each row's dot colour (accent). */
		typeColors?: Record<string, EventTypeStyle>;
		/** Optional per-code counts shown at the row end. */
		counts?: Record<string, number>;
		/** Codes currently hidden (unchecked). Absent/empty → all shown. */
		hidden?: Set<string>;
		/** Toggle a code's visibility. */
		onToggle?: (code: string) => void;
		/** Clear all filters — rendered (when provided) only while something is hidden. */
		onReset?: () => void;
		/** Optional group heading. */
		title?: string;
	}

	let { types, typeColors, counts, hidden, onToggle, onReset, title }: Props = $props();

	function dotColor(code: string): string {
		const style = typeColors?.[code];
		return style?.accent ?? style?.text ?? 'var(--color-text-muted, #8a8a85)';
	}
	const anyHidden = $derived(!!hidden && hidden.size > 0);
</script>

<div class="etf" role="group" aria-label={title ?? 'Filter by event type'}>
	{#if title}<p class="etf-title">{title}</p>{/if}
	<ul class="etf-list">
		{#each types as t (t.code)}
			<li class="etf-row" class:is-off={hidden?.has(t.code)}>
				<label class="etf-label">
					<input
						type="checkbox"
						class="etf-check"
						checked={!hidden?.has(t.code)}
						onchange={() => onToggle?.(t.code)}
					/>
					<span class="etf-dot" style="background: {dotColor(t.code)};"></span>
					<span class="etf-name">{t.label}</span>
					{#if counts}<span class="etf-count">{counts[t.code] ?? 0}</span>{/if}
				</label>
			</li>
		{/each}
	</ul>
	{#if onReset && anyHidden}
		<button type="button" class="etf-reset" onclick={onReset}>Show all</button>
	{/if}
</div>

<style>
	/* Generic tokens only (--color-* / --cal-accent-color) — host-themed, no palette here. */
	.etf {
		display: flex;
		flex-direction: column;
		gap: 0.3rem;
	}
	.etf-title {
		margin: 0 0 0.15rem;
		font-size: 0.68rem;
		font-weight: 700;
		letter-spacing: 0.05em;
		text-transform: uppercase;
		color: var(--color-text-muted, #8a8a85);
	}
	.etf-list {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
	}
	.etf-label {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.28rem 0.3rem;
		border-radius: 0.4rem;
		cursor: pointer;
		font-size: 0.82rem;
		color: var(--color-text, #1d1d1b);
	}
	.etf-label:hover {
		background: color-mix(in srgb, var(--color-text, #1d1d1b) 5%, transparent);
	}
	.etf-check {
		flex: 0 0 auto;
		width: 0.95rem;
		height: 0.95rem;
		margin: 0;
		cursor: pointer;
		/* checked tick uses the host accent (BR gold) when set, else a neutral. */
		accent-color: var(--cal-accent-color, var(--color-text-muted, #707068));
	}
	.etf-dot {
		flex: 0 0 auto;
		width: 0.6rem;
		height: 0.6rem;
		border-radius: 50%;
	}
	.etf-name {
		flex: 1 1 auto;
		min-width: 0;
		overflow-wrap: anywhere;
	}
	.etf-count {
		flex: 0 0 auto;
		font-size: 0.74rem;
		font-weight: 600;
		color: var(--color-text-muted, #8a8a85);
		font-variant-numeric: tabular-nums;
	}
	/* Hidden type: dim the dot + label (the count stays so you still see what's filtered out). */
	.etf-row.is-off .etf-dot,
	.etf-row.is-off .etf-name {
		opacity: 0.45;
	}
	.etf-reset {
		align-self: flex-start;
		margin-top: 0.2rem;
		padding: 0.15rem 0.1rem;
		font: inherit;
		font-size: 0.76rem;
		font-weight: 600;
		color: var(--cal-accent-color, var(--color-text, #1d1d1b));
		background: transparent;
		border: none;
		cursor: pointer;
		text-decoration: underline;
	}
	.etf-reset:focus-visible {
		outline: 2px solid var(--cal-accent-color, #e5392b);
		outline-offset: 2px;
	}
</style>
