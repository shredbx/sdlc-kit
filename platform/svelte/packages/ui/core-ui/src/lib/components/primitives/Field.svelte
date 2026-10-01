<script lang="ts">
	/**
	 * Field — form-field wrapper primitive.
	 *
	 * Provides the consistent label + required-marker + help/error layout and
	 * spacing that every form control shares, so consumer pages never re-author
	 * `.form-label` / `.field-error` CSS. Wrap ANY control (Input, Select,
	 * Textarea, MoneyInput, SizeInput, a custom combobox, a checkbox row) in a
	 * <Field> and pass the control as the default children snippet.
	 *
	 * The control is responsible for its OWN box styling (border, focus ring,
	 * height). Field owns only the surrounding chrome: label typography, the
	 * required `*`, the helper/error line, and the vertical rhythm between them.
	 *
	 * Accessibility contract:
	 *  - Pass `for` (the control's id) so the <label> associates correctly.
	 *  - When `error` is set, render an id'd error node; the control should set
	 *    `aria-describedby={`${id}-error`}` + `aria-invalid` itself. Field exposes
	 *    the derived ids via the `children` snippet param for convenience.
	 *
	 * Tokens (all fall back to sane neutrals — themeable per consumer):
	 *  --field-label-color, --field-label-size, --field-label-weight,
	 *  --field-label-gap (label→control), --field-help-color, --field-help-size,
	 *  --field-error-color, --field-required-color.
	 *
	 * @example
	 *   <Field label="Street" for="street" help="Building number + road">
	 *     <Input id="street" name="street" autocomplete="street-address" bind:value={street} />
	 *   </Field>
	 *
	 *   <Field label="Sale Price" for="sale_price" required error={salePriceError}>
	 *     {#snippet children(ids)}
	 *       <MoneyInput id="sale_price" name="sale_price" bind:value={salePrice}
	 *         ariaInvalid={!!salePriceError} aria-describedby={ids.describedBy} />
	 *     {/snippet}
	 *   </Field>
	 */

	import type { Snippet } from 'svelte';

	interface FieldIds {
		/** id to wire onto the control's `aria-describedby` (error wins over help). */
		describedBy: string | undefined;
		errorId: string;
		helpId: string;
	}

	interface Props {
		/** Visible label text. Omit for a control that supplies its own label. */
		label?: string;
		/** The control's id, used for <label for> association. */
		for?: string;
		/** Mark the field required (renders a `*` after the label). */
		required?: boolean;
		/** Helper text under the control (hidden when `error` is set). */
		help?: string;
		/** Error message under the control (replaces help, role=alert). */
		error?: string;
		/** Make the field span all columns of a parent FieldGrid. */
		full?: boolean;
		/** Extra class on the wrapper. */
		class?: string;
		/** The control. Receives derived a11y ids as its snippet param. */
		children: Snippet<[FieldIds]>;
	}

	let {
		label,
		for: forId,
		required = false,
		help,
		error,
		full = false,
		class: className = '',
		children
	}: Props = $props();

	const errorId = $derived(forId ? `${forId}-error` : '');
	const helpId = $derived(forId ? `${forId}-help` : '');
	const ids = $derived<FieldIds>({
		describedBy: error ? errorId || undefined : help ? helpId || undefined : undefined,
		errorId,
		helpId
	});
</script>

<div class="field {className}" class:field--full={full} class:field--error={!!error}>
	{#if label}
		<label class="field__label" for={forId}>
			{label}{#if required}<span class="field__required" aria-hidden="true">*</span>{/if}
		</label>
	{/if}

	{@render children(ids)}

	{#if error}
		<span class="field__error" id={errorId || undefined} role="alert">{error}</span>
	{:else if help}
		<span class="field__help" id={helpId || undefined}>{help}</span>
	{/if}
</div>

<style>
	.field {
		display: flex;
		flex-direction: column;
		gap: var(--field-label-gap, 0.375rem);
		min-width: 0; /* lets the control shrink inside a grid track without overflow */
	}

	.field--full {
		grid-column: 1 / -1;
	}

	.field__label {
		font-family: var(--field-label-font, inherit);
		font-size: var(--field-label-size, 0.875rem);
		font-weight: var(--field-label-weight, 600);
		line-height: 1.3;
		color: var(--field-label-color, var(--color-text, #1f2937));
	}

	.field__required {
		color: var(--field-required-color, var(--color-error, #dc2626));
		margin-left: 0.15em;
	}

	.field__help {
		font-size: var(--field-help-size, 0.75rem);
		line-height: 1.4;
		color: var(--field-help-color, var(--color-text-muted, #6b7280));
	}

	.field__error {
		font-size: var(--field-help-size, 0.75rem);
		line-height: 1.4;
		color: var(--field-error-color, var(--color-error, #dc2626));
	}
</style>
