<script lang="ts">
	/**
	 * PresetPill — provenance chip for a prompt section (dumb, controlled). Renders one of three
	 * states derived by the host from a block's link/based_on:
	 *   linked  — content mirrors a catalog preset live      (iris)
	 *   edited  — detached from a preset, now diverged        (amber)
	 *   custom  — authored inline, no preset                  (muted)
	 * Labels are passed in so the host owns i18n; the pill itself imports no store/API.
	 */
	type PresetStatus = 'linked' | 'edited' | 'custom';

	interface Props {
		status: PresetStatus;
		/** Preset name shown for linked/edited (e.g. "evaluator", "warm"). Absent for custom. */
		presetName?: string;
		/** Localized "preset ·" prefix for the linked state. */
		prefixLabel?: string;
		/** Localized "edited — was" lead for the edited state. */
		editedLabel?: string;
		/** Localized "custom" word for the custom state. */
		customLabel?: string;
		onclick?: () => void;
	}

	let {
		status,
		presetName,
		prefixLabel = 'preset ·',
		editedLabel = 'edited — was',
		customLabel = 'custom',
		onclick
	}: Props = $props();
</script>

<button
	type="button"
	class="presetsel {status}"
	data-testid="preset-pill-{status}"
	{onclick}
>
	{#if status === 'linked'}
		<span class="lead" aria-hidden="true"></span>
		<span class="k">{prefixLabel}</span>
		{presetName}
	{:else if status === 'edited'}
		<span class="k">{editedLabel}</span>
		{presetName}
	{:else}
		{customLabel}
	{/if}
	<span class="chev" aria-hidden="true">▼</span>
</button>

<style>
	.presetsel {
		font-family: var(--font-family-mono, ui-monospace, monospace);
		font-size: 11.5px;
		font-weight: 600;
		padding: 3px 7px 3px 9px;
		border-radius: 999px;
		display: inline-flex;
		align-items: center;
		gap: 6px;
		white-space: nowrap;
		border: 1px solid transparent;
		cursor: pointer;
		color: var(--color-text-muted);
	}
	.presetsel.linked {
		background: var(--color-accent-soft);
		color: var(--color-accent-ink);
	}
	.presetsel.linked .lead {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: var(--color-primary);
	}
	.presetsel.edited {
		background: var(--color-amber-soft);
		color: var(--color-amber);
	}
	.presetsel.custom {
		background: var(--color-subtle);
		color: var(--color-text-muted);
	}
	.presetsel:hover {
		border-color: var(--color-line-strong);
	}
	.presetsel .chev {
		opacity: 0.55;
		font-size: 9px;
	}
</style>
