<script lang="ts">
	// Segmented icon control (design pass A, 2026-06-07) — the text-editor-style
	// enum/toggle row: ALIGN [⫷|≡|⫸], FORMAT [B][I]. ONE shared component for both
	// behaviors (never-copy-paste-CSS):
	//   single-select  pass `value`   — radio behavior, the matching option is active
	//                                   (ALIGN: exactly one alignment).
	//   independent    omit `value`   — each option toggles on its own `pressed`
	//                                   flag (FORMAT: bold and italic combine).
	// Either way `onchange` emits the CLICKED option's value; the consumer decides
	// what it means (set vs flip). Buttons carry aria-pressed; the group is labelled.
	import { Icon } from '@sbx/core-ui/components/primitives';

	interface SegmentOption {
		value: string;
		/** Lucide icon name (the usual face). */
		icon?: string;
		/** Text face (used when no icon fits — also the fallback). */
		label?: string;
		/** Tooltip; falls back to label. */
		title?: string;
		/** Independent mode: this segment's on/off state (ignored when `value` is set). */
		pressed?: boolean;
	}

	interface Props {
		options: SegmentOption[];
		/** Single-select mode: the active value. Omit for independent toggles. */
		value?: string;
		onchange?: (value: string) => void;
		disabled?: boolean;
		'aria-label'?: string;
	}

	let { options, value, onchange, disabled = false, 'aria-label': ariaLabel }: Props = $props();

	function isActive(opt: SegmentOption): boolean {
		return value !== undefined ? opt.value === value : !!opt.pressed;
	}
</script>

<div class="seg" role="group" aria-label={ariaLabel}>
	{#each options as opt (opt.value)}
		<button
			type="button"
			class="seg__btn"
			class:seg__btn--active={isActive(opt)}
			{disabled}
			aria-pressed={isActive(opt)}
			aria-label={opt.title ?? opt.label ?? opt.value}
			title={opt.title ?? opt.label ?? opt.value}
			onclick={() => {
				// Single-select: re-clicking the active segment is a no-op — re-firing
				// would issue a same-value update that dirties the doc for nothing.
				if (value !== undefined && opt.value === value) return;
				onchange?.(opt.value);
			}}
		>
			{#if opt.icon}
				<Icon name={opt.icon} size="xs" />
			{:else}
				{opt.label ?? opt.value}
			{/if}
		</button>
	{/each}
</div>

<style>
	/* One bordered pill, hairline-divided segments — Xcode/Figma density (28px).
	   width: fit-content — in a stretching flex column the pill must hug its
	   segments, never span the row (a stretched pill reads as one giant last
	   segment — user call 2026-06-07). */
	.seg {
		display: inline-flex;
		align-items: stretch;
		width: fit-content;
		height: 1.75rem;
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: var(--cv-color-surface, #fff);
		overflow: hidden;
	}

	.seg__btn {
		display: flex;
		align-items: center;
		justify-content: center;
		min-width: 1.75rem;
		padding: 0 0.375rem;
		border: none;
		background: transparent;
		font: inherit;
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--cv-color-neutral-500, #707070);
		cursor: pointer;
	}

	.seg__btn + .seg__btn {
		border-left: 1px solid var(--cv-border-color, #e0e0e0);
	}

	.seg__btn:hover:not(:disabled):not(.seg__btn--active) {
		background: var(--cv-color-neutral-100, #f5f5f5);
		color: var(--cv-color-neutral-700, #383838);
	}

	.seg__btn--active {
		background: var(--cv-color-primary, #333333);
		color: var(--cv-color-surface, #fff);
	}

	.seg__btn:disabled {
		cursor: not-allowed;
		opacity: 0.45;
	}

	.seg__btn:focus-visible {
		outline: 2px solid var(--cv-color-primary, #333333);
		outline-offset: -2px;
	}
</style>
