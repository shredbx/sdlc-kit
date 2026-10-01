<script lang="ts">
	/**
	 * SectionAddRow — name-first section creation (dumb, controlled). Two shapes:
	 *   inside=false (default) — the Edit pane's bottom row: "+ Add section" + catalog chips + a free
	 *                            "or type a name…" input.
	 *   inside=true            — a compact row shown under a section when adding a child: a name input
	 *                            + Add / cancel.
	 * A blank / whitespace-only name NEVER fires onadd (structural guarantee: no unnamed block is
	 * creatable). Imports no store/API.
	 */
	interface Props {
		inside?: boolean;
		/** Catalog quick-pick chips (bottom row only). */
		chips?: string[];
		addLabel?: string;
		sectionLabel?: string;
		typeLabel?: string;
		onadd?: (name: string) => void;
		oncancel?: () => void;
	}

	let {
		inside = false,
		chips = [],
		addLabel = 'Add section',
		sectionLabel = 'Add',
		typeLabel = 'or type a name…',
		onadd,
		oncancel
	}: Props = $props();

	let draft = $state('');

	function commit() {
		const name = draft.trim();
		if (!name) return; // no unnamed block, ever
		onadd?.(name);
		draft = '';
	}

	function onkeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			commit();
		} else if (e.key === 'Escape') {
			draft = '';
			oncancel?.();
		}
	}
</script>

{#if inside}
	<div class="addinside-row" data-testid="section-add-row">
		<!-- svelte-ignore a11y_autofocus -->
		<input
			class="fld-name"
			bind:value={draft}
			aria-label="Name"
			placeholder="Section name…"
			autofocus
			{onkeydown}
		/>
		<button class="btn-add" onclick={commit}>{sectionLabel}</button>
		<button class="btn-cancel" aria-label="Cancel" onclick={() => oncancel?.()}>✕</button>
	</div>
{:else}
	<div class="addrow" data-testid="section-add-row">
		<span class="lead"><span class="addplus">+</span> <span>{addLabel}</span></span>
		{#each chips as chip}
			<button class="chip" onclick={() => onadd?.(chip)}>{chip}</button>
		{/each}
		<input
			class="type-input"
			bind:value={draft}
			aria-label="Name"
			placeholder={typeLabel}
			{onkeydown}
		/>
	</div>
{/if}

<style>
	.addrow {
		display: flex;
		align-items: center;
		gap: 10px;
		flex-wrap: wrap;
		padding: 18px 2px 0;
	}
	.addrow .lead {
		font-size: 13.5px;
		font-weight: 600;
		display: inline-flex;
		align-items: center;
		gap: 9px;
		color: var(--color-text);
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
	}
	.chip {
		font-family: var(--font-family-mono, ui-monospace, monospace);
		font-size: 12px;
		color: var(--color-accent-ink);
		background: var(--color-accent-soft);
		border-radius: 7px;
		padding: 5px 10px;
		font-weight: 600;
		border: 0;
	}
	.chip:hover {
		box-shadow: inset 0 0 0 1.5px var(--color-primary);
	}
	.type-input {
		font-size: 12.5px;
		color: var(--color-faint);
		border: 0;
		background: transparent;
		outline: none;
		flex: 1;
		min-width: 120px;
		font-family: inherit;
	}
	.addinside-row {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 6px 8px;
	}
	.fld-name {
		flex: 1;
		border: 1px solid var(--color-primary);
		border-radius: 8px;
		padding: 7px 10px;
		font-size: 13px;
		font-family: var(--font-family-mono, ui-monospace, monospace);
		background: var(--color-surface);
		color: var(--color-text);
		outline: none;
		box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-primary) 14%, transparent);
	}
	.btn-add {
		border: 1px solid var(--color-primary);
		background: var(--color-primary);
		color: #fff;
		font-size: 13px;
		font-weight: 600;
		padding: 7px 13px;
		border-radius: 8px;
	}
	.btn-cancel {
		border: 0;
		background: transparent;
		color: var(--color-faint);
		font-size: 14px;
		padding: 4px 6px;
		border-radius: 6px;
	}
	.btn-cancel:hover {
		background: var(--color-subtle);
		color: var(--color-text-muted);
	}
</style>
