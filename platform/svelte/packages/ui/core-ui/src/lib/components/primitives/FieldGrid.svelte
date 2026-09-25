<script lang="ts">
	/**
	 * FieldGrid — responsive form layout primitive.
	 *
	 * Lays out <Field> children in an even, gap-consistent grid so consumer
	 * pages never hand-author `.field-grid` / `.cascade-row` CSS. A field marked
	 * `full` (or the equivalent `.field--full`) spans every column; everything
	 * else flows into the column tracks.
	 *
	 * Collapses to a single column under `--field-grid-collapse` (default 640px)
	 * so address/room grids stack cleanly on mobile.
	 *
	 * @example
	 *   <FieldGrid columns={2}>
	 *     <Field label="Title" for="title" full><Input id="title" ... /></Field>
	 *     <Field label="Unit" for="unit"><Input id="unit" ... /></Field>
	 *     <Field label="Postal" for="postal"><Input id="postal" ... /></Field>
	 *     <!-- a sub-grid for a 3-up cascade -->
	 *     <FieldGrid columns={3} full>…</FieldGrid>
	 *   </FieldGrid>
	 *
	 * Tokens:
	 *   --field-grid-gap        gap between cells (default 1.25rem)
	 *   --field-grid-collapse   max-width below which it stacks (default 640px)
	 */

	import type { Snippet } from 'svelte';

	interface Props {
		/** Number of equal columns at full width (1–4). */
		columns?: 1 | 2 | 3 | 4;
		/** Span all columns of a parent FieldGrid (for nested sub-grids). */
		full?: boolean;
		class?: string;
		children: Snippet;
	}

	let { columns = 2, full = false, class: className = '', children }: Props = $props();
</script>

<div
	class="field-grid {className}"
	class:field-grid--full={full}
	style="--_cols: {columns};"
>
	{@render children()}
</div>

<style>
	.field-grid {
		display: grid;
		grid-template-columns: repeat(var(--_cols, 2), minmax(0, 1fr));
		gap: var(--field-grid-gap, 1.25rem);
		align-items: start;
	}

	.field-grid--full {
		grid-column: 1 / -1;
	}

	/* Stack to a single column on narrow viewports. Container-query would be
	   ideal but viewport width is the consistent signal across all consumers
	   today; the breakpoint is tokenised so a consumer can tune it. */
	@media (max-width: 640px) {
		.field-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
