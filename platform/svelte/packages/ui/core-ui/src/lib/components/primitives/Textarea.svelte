<script lang="ts">
	/**
	 * Textarea — multi-line text input primitive.
	 *
	 * Shares the EXACT field visual language with Input / Select / MoneyInput /
	 * SizeInput (border, radius, padding, focus ring, placeholder, disabled,
	 * readonly) via the shared `--field-*` token set, so a Textarea sits flush
	 * with single-line controls in the same form. Vertical resize only.
	 *
	 * Used by the CMS About editor (markdown body) and any long-form field.
	 *
	 * @example
	 *   <Textarea name="about_body" bind:value={body} rows={14}
	 *     placeholder="Write the About page in Markdown…" />
	 */

	interface Props {
		value?: string;
		name?: string;
		id?: string;
		placeholder?: string;
		rows?: number;
		disabled?: boolean;
		readonly?: boolean;
		required?: boolean;
		maxlength?: number;
		autocomplete?: AutoFill;
		/** Marks the control invalid (red border) + aria-invalid. */
		ariaInvalid?: boolean;
		/** id of an error/help node for aria-describedby. */
		'aria-describedby'?: string;
		'aria-label'?: string;
		class?: string;
		oninput?: (value: string) => void;
		onchange?: (value: string) => void;
		onblur?: () => void;
		onfocus?: () => void;
	}

	let {
		value = $bindable(''),
		name,
		id,
		placeholder = '',
		rows = 6,
		disabled = false,
		readonly = false,
		required = false,
		maxlength,
		autocomplete,
		ariaInvalid = false,
		'aria-describedby': ariaDescribedBy,
		'aria-label': ariaLabel,
		class: className = '',
		oninput,
		onchange,
		onblur,
		onfocus
	}: Props = $props();

	const autoId = `textarea-${Math.random().toString(36).slice(2, 9)}`;
	const fieldId = $derived(id ?? autoId);

	function handleInput(e: Event) {
		value = (e.target as HTMLTextAreaElement).value;
		oninput?.(value);
	}
	function handleChange(e: Event) {
		value = (e.target as HTMLTextAreaElement).value;
		onchange?.(value);
	}
</script>

<textarea
	id={fieldId}
	{name}
	{placeholder}
	{rows}
	{disabled}
	{readonly}
	{required}
	{maxlength}
	{autocomplete}
	aria-invalid={ariaInvalid || undefined}
	aria-describedby={ariaDescribedBy}
	aria-label={ariaLabel}
	bind:value
	oninput={handleInput}
	onchange={handleChange}
	onfocus={() => onfocus?.()}
	onblur={() => onblur?.()}
	class="field-control field-textarea {className}"
	class:is-invalid={ariaInvalid}
></textarea>

<style>
	/* field-control — the shared box language. Kept byte-identical to the same
	   block in Input/Select/MoneyInput/SizeInput so every control matches. The
	   --field-* token set chains to the legacy input and color tokens so
	   existing consumer themes keep working without any change. */
	.field-control {
		width: 100%;
		font-family: var(--field-font, inherit);
		font-size: var(--field-font-size, 0.9375rem);
		color: var(--field-text, var(--input-text, var(--color-text, #1f2937)));
		background: var(--field-bg, var(--input-bg, var(--color-surface, #fff)));
		border: var(--field-border-width, 1px) solid
			var(--field-border-color, var(--input-border, var(--color-border, #d4d4d8)));
		border-radius: var(--field-radius, var(--input-radius, var(--radius-md, 0.5rem)));
		outline: none;
		transition:
			border-color 0.15s ease,
			box-shadow 0.15s ease;
	}

	.field-control::placeholder {
		color: var(--field-placeholder, var(--input-placeholder, var(--color-text-muted, #9ca3af)));
		opacity: 1;
	}

	.field-control:hover:not(:disabled):not(:focus):not(:read-only) {
		border-color: var(--field-border-hover, var(--input-border-hover, var(--color-border-hover, #b8b8b8)));
	}

	.field-control:focus {
		border-color: var(--field-border-focus, var(--input-border-focus, var(--color-accent, #2563eb)));
		box-shadow: 0 0 0 3px var(--field-ring, var(--input-ring, rgba(37, 99, 235, 0.18)));
	}

	.field-control:disabled {
		opacity: 0.6;
		cursor: not-allowed;
		background: var(--field-disabled-bg, var(--input-bg-readonly, var(--color-bg-secondary, #f5f5f5)));
	}

	.field-control:read-only {
		background: var(--field-readonly-bg, var(--input-bg-readonly, var(--color-bg-secondary, #fafafa)));
	}

	.field-control.is-invalid,
	.field-control[aria-invalid='true'] {
		border-color: var(--field-border-error, var(--color-error, #dc2626));
	}

	.field-control.is-invalid:focus,
	.field-control[aria-invalid='true']:focus {
		box-shadow: 0 0 0 3px var(--field-ring-error, rgba(220, 38, 38, 0.18));
	}

	/* Textarea-specific: comfortable line-height + vertical-only resize. */
	.field-textarea {
		display: block;
		padding: var(--field-padding-y, 0.5rem) var(--field-padding-x, 0.75rem);
		line-height: 1.55;
		resize: vertical;
		min-height: calc(var(--field-height, 2.5rem) * 2);
	}
</style>
