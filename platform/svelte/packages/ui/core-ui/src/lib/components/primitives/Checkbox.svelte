<script lang="ts">
	/**
	 * Checkbox — a branded, accessible boolean control.
	 *
	 * Renders a REAL native `<input type="checkbox">` (visually hidden, sr-only)
	 * inside a clickable `<label class="checkbox">` row, with the custom box drawn
	 * alongside via a sibling `.checkbox__box` (an inline SVG tick that shows only
	 * when checked). Keeping the native input means assistive tech, the keyboard
	 * (Space toggles, Tab focuses), form submission (`name`/`value`), and tests all
	 * get a genuine toggleable control — the styling is purely presentational.
	 *
	 * The whole row is the click target (label wraps the input). The label text
	 * WRAPS and is never clipped (no ellipsis / line-clamp) per the no-truncate rule.
	 *
	 * Unchecked: 18px white box, 1px --br-border-color, radius --br-radius-sm.
	 * Checked:   --br-color-primary fill + white tick.
	 * Focus:     --br-focus-ring around the box (driven off the input's :focus-visible).
	 *
	 * Tokens (themeable per consumer; brand defaults shown):
	 *   --br-color-primary, --br-color-surface, --br-border-color, --br-border-width,
	 *   --br-radius-sm, --br-focus-ring, --br-color-neutral-700, --br-color-neutral-400,
	 *   --br-size-sm, --br-space-sm.
	 *
	 * @example
	 *   <Checkbox label="Sea View" name="amenities" bind:checked={hasSeaView} />
	 */

	interface Props {
		/** Visible row label. Wraps; never clipped. */
		label: string;
		/** Two-way bound checked state (uncontrolled use). */
		checked?: boolean;
		/** Disable the control (blocks toggle, dims the row). */
		disabled?: boolean;
		/** Optional form field name (e.g. for an uncontrolled group submit). */
		name?: string;
		/** Optional submitted value when checked. */
		value?: string;
		/** Extra class on the wrapping label. */
		class?: string;
		/**
		 * Controlled-mode callback: fires with the next checked state on toggle.
		 * Lets a consumer drive `checked` from external state (e.g. a Set membership)
		 * without two-way binding. Used alongside a read-only `checked` prop.
		 */
		onCheckedChange?: (checked: boolean) => void;
	}

	let {
		label,
		checked = $bindable(false),
		disabled = false,
		name,
		value,
		class: className = '',
		onCheckedChange
	}: Props = $props();
</script>

<label class="checkbox {className}" class:checkbox--disabled={disabled}>
	<input
		class="checkbox__input"
		type="checkbox"
		{name}
		{value}
		{disabled}
		bind:checked
		onchange={(e) => onCheckedChange?.((e.currentTarget as HTMLInputElement).checked)}
	/>
	<span class="checkbox__box" aria-hidden="true">
		<svg class="checkbox__tick" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
			<path
				d="M3.5 8.5L6.5 11.5L12.5 5"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
			/>
		</svg>
	</span>
	<span class="checkbox__label">{label}</span>
</label>

<style>
	.checkbox {
		display: inline-flex;
		align-items: flex-start;
		gap: var(--br-space-sm, 0.5rem);
		cursor: pointer;
		/* Let a long label wrap inside the row instead of forcing the row wide. */
		min-width: 0;
		color: var(--br-color-neutral-700, #383830);
		font-family: var(--br-font-body, inherit);
		font-size: var(--br-size-sm, 0.875rem);
		line-height: 1.4;
	}

	.checkbox--disabled {
		cursor: not-allowed;
		opacity: 0.55;
	}

	/* Native input: visually hidden but a real, focusable, submittable control.
	   Not display:none / visibility:hidden so it stays in the a11y + tab order. */
	.checkbox__input {
		position: absolute;
		width: 1px;
		height: 1px;
		margin: -1px;
		padding: 0;
		border: 0;
		overflow: hidden;
		clip: rect(0 0 0 0);
		clip-path: inset(50%);
		white-space: nowrap;
	}

	/* The drawn box — sits where the OS checkbox would, aligned to the first
	   text line so a wrapped label keeps the box at the top. */
	.checkbox__box {
		flex-shrink: 0;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 18px;
		height: 18px;
		margin-top: 0.05em; /* optical-align to the cap height of the label */
		background: var(--br-color-surface, #fff);
		border: var(--br-border-width, 1px) solid var(--br-border-color, #e8e8e0);
		border-radius: var(--br-radius-sm, 0.25rem);
		color: var(--br-color-surface, #fff); /* tick colour (white on teal) */
		transition:
			background-color 0.12s ease,
			border-color 0.12s ease;
	}

	.checkbox__tick {
		width: 14px;
		height: 14px;
		opacity: 0;
		transition: opacity 0.1s ease;
	}

	/* Checked: teal fill + white tick. */
	.checkbox__input:checked + .checkbox__box {
		background: var(--br-color-primary, #0d4f4f);
		border-color: var(--br-color-primary, #0d4f4f);
	}
	.checkbox__input:checked + .checkbox__box .checkbox__tick {
		opacity: 1;
	}

	/* Keyboard focus ring on the drawn box (the input itself is hidden). */
	.checkbox__input:focus-visible + .checkbox__box {
		outline: none;
		box-shadow: 0 0 0 3px var(--br-focus-ring, rgba(13, 79, 79, 0.12));
		border-color: var(--br-color-primary, #0d4f4f);
	}

	.checkbox:hover:not(.checkbox--disabled) .checkbox__box {
		border-color: var(--br-color-primary, #0d4f4f);
	}

	/* Label wraps; explicitly NOT truncated (no ellipsis / line-clamp). */
	.checkbox__label {
		min-width: 0;
		overflow-wrap: anywhere;
	}
</style>
