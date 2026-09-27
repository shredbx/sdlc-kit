<script lang="ts">
	/**
	 * CheckboxGroup — a labelled cluster of related Checkboxes.
	 *
	 * Renders an uppercase category subhead followed by its children laid out in
	 * `columns` sub-columns (default 2). Composes the amenity quadrant: a parent
	 * grid places groups 2-up; each group lays its own checkboxes in two columns.
	 *
	 * Dumb by design — it owns only the subhead + the inner column grid. The caller
	 * passes the Checkboxes as the default children snippet, so the group never
	 * knows about amenity/highlight domain shape.
	 *
	 * Tokens: --br-size-xs, --br-space-xs, --br-space-sm, --br-space-md,
	 * --br-color-neutral-500, --br-font-body.
	 *
	 * @example
	 *   <CheckboxGroup label="Building">
	 *     {#each items as opt (opt.code)}
	 *       <Checkbox label={opt.label} checked={set.has(opt.code)} ... />
	 *     {/each}
	 *   </CheckboxGroup>
	 */

	import type { Snippet } from 'svelte';

	interface Props {
		/** Category subhead (rendered uppercase). */
		label: string;
		/** Number of sub-columns the checkboxes flow into (default 2). */
		columns?: 1 | 2 | 3 | 4;
		/** Extra class on the group wrapper. */
		class?: string;
		/** The Checkbox rows. */
		children: Snippet;
	}

	let { label, columns = 2, class: className = '', children }: Props = $props();
</script>

<div class="checkbox-group {className}">
	<p class="checkbox-group__label">{label}</p>
	<div class="checkbox-group__items" style="--_cols: {columns};">
		{@render children()}
	</div>
</div>

<style>
	.checkbox-group {
		min-width: 0;
	}

	.checkbox-group__label {
		margin: 0 0 var(--br-space-sm, 0.5rem);
		font-family: var(--br-font-body, inherit);
		font-size: var(--br-size-xs, 0.75rem);
		font-weight: 600;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: var(--br-color-neutral-500, #707068);
	}

	.checkbox-group__items {
		display: grid;
		grid-template-columns: repeat(var(--_cols, 2), minmax(0, 1fr));
		gap: var(--br-space-sm, 0.5rem) var(--br-space-md, 1rem);
		align-items: start;
	}

	/* Step the column count down on narrower viewports so labels never crowd:
	   4/3 → 2 below 900px, → 1 below 560px. (2-col groups are unaffected at 900.) */
	@media (max-width: 900px) {
		.checkbox-group__items {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}
	@media (max-width: 560px) {
		.checkbox-group__items {
			grid-template-columns: 1fr;
		}
	}
</style>
