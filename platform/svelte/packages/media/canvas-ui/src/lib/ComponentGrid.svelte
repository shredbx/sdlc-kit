<script lang="ts">
	// Shared palette grid (Fix 4, §11.6) — the responsive ~3-column container that
	// every insertable palette uses (Text · Shapes · Widgets · Map · Templates ·
	// Images). Holds the grid CSS ONCE (never-copy-paste-CSS); panels fill it with
	// ComponentCards via the default children snippet. `cols` lets a panel widen
	// the columns for thumbnail grids (templates/images) without re-authoring CSS.
	interface Props {
		/** Column count (default 3, matching the old .palette grid). */
		cols?: number;
		/** The cards — a row of ComponentCards. */
		children?: import('svelte').Snippet;
	}

	let { cols = 3, children }: Props = $props();
</script>

<div class="grid" style="--grid-cols: {cols};">
	{@render children?.()}
</div>

<style>
	.grid {
		display: grid;
		grid-template-columns: repeat(var(--grid-cols, 3), minmax(0, 1fr));
		gap: var(--cv-space-sm, 0.5rem);
	}
</style>
