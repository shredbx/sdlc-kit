<script lang="ts">
	/**
	 * MoneyInput — currency-aware numeric input with thousands-separator
	 * display and a shorthand caption ("≈ 15M THB" / "≈ 45K THB").
	 *
	 * Storage contract: binds to a string of digits in MAJOR units (whole
	 * THB, not satang). Mirrors the existing `salePrice` / `leasePrice`
	 * state on property /new + /edit — callers continue to do their own
	 * `Math.round(Number(value) * 100)` conversion to satang on submit.
	 *
	 * Display: text input with `inputmode="decimal"` so browsers accept
	 * commas. On blur, value renders with thousands separators + currency
	 * suffix (e.g. "15,000,000 THB / mo"). On focus, switches to plain
	 * digits for easy editing. The bound value stays plain digits.
	 *
	 * Caption: under the input, an alt-form shorthand renders for amounts
	 * >= 10K so the user can sanity-check a 15,000,000 THB price didn't
	 * become 150,000,000 by accident. Display-only — the user CANNOT enter
	 * shorthand (would be too easy to typo "150M" instead of "15M").
	 *
	 * @example
	 *   <MoneyInput name="sale_price" bind:value={salePrice} currency="THB" />
	 *
	 *   <MoneyInput
	 *     name="lease_price"
	 *     bind:value={leasePrice}
	 *     currency="THB"
	 *     suffix=" / month"
	 *     placeholder="0"
	 *   />
	 */

	interface Props {
		value?: string;
		name?: string;
		id?: string;
		placeholder?: string;
		disabled?: boolean;
		currency?: string;
		/** Where the currency label sits relative to the amount. 'trailing' (default)
		 * appends it to the blur-formatted value ("15,000,000 THB"); 'leading' renders
		 * it as a persistent adornment before the field ("THB  15,000,000") and drops
		 * the trailing code so it never reads "THB 15,000,000 THB". */
		currencyPosition?: 'leading' | 'trailing';
		suffix?: string;
		ariaInvalid?: boolean;
		/** id of an error/help node for aria-describedby (from <Field>). */
		'aria-describedby'?: string;
		onchange?: (value: string) => void;
	}

	import { stripToDigits, formatMoneyDisplay, moneyCaption, withCommas } from './money';

	let {
		value = $bindable(''),
		name,
		id,
		placeholder = '0',
		disabled = false,
		currency = 'THB',
		currencyPosition = 'trailing',
		suffix = '',
		ariaInvalid = false,
		'aria-describedby': ariaDescribedBy,
		onchange
	}: Props = $props();

	let focused = $state(false);

	const autoId = `money-input-${Math.random().toString(36).slice(2, 9)}`;
	const inputId = $derived(id ?? autoId);

	const isLeading = $derived(currencyPosition === 'leading');
	// Plain digits while editing. On blur: the trailing-currency display by default,
	// or grouped digits + suffix only (no trailing code) when the currency shows as a
	// leading adornment — so it never reads "THB 15,000,000 THB".
	const formattedDisplay = $derived(
		focused
			? value
			: isLeading
				? value
					? `${withCommas(value)}${suffix}`
					: ''
				: formatMoneyDisplay(value, currency, suffix)
	);
	const caption = $derived(moneyCaption(value, currency, suffix));

	function handleInput(e: Event) {
		const digits = stripToDigits((e.target as HTMLInputElement).value);
		value = digits;
		onchange?.(digits);
	}
</script>

<div class="money-input" class:is-disabled={disabled} class:has-prefix={isLeading}>
	<div class="money-input__control">
		{#if isLeading}
			<span class="money-input__prefix">{currency}</span>
		{/if}
		<input
			id={inputId}
			{name}
			type="text"
			inputmode="decimal"
			autocomplete="off"
			{placeholder}
			{disabled}
			aria-invalid={ariaInvalid || undefined}
			aria-describedby={ariaDescribedBy}
			value={formattedDisplay}
			oninput={handleInput}
			onfocus={() => { focused = true; }}
			onblur={() => { focused = false; }}
			class="field-control money-input__field"
			class:is-invalid={ariaInvalid}
			class:has-prefix={isLeading}
		/>
	</div>
	{#if caption}
		<div class="money-input__caption" aria-live="polite">{caption}</div>
	{/if}
</div>

<style>
	.money-input {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		width: 100%;
	}

	/* Leading-currency adornment: the currency code sits inside the field's left
	   inset, the field reserves room for it. Generic tokens only — no brand vars. */
	.money-input__control {
		position: relative;
		width: 100%;
	}
	.money-input__prefix {
		position: absolute;
		left: var(--field-padding-x, 0.75rem);
		top: 50%;
		transform: translateY(-50%);
		font-size: var(--field-font-size, 0.9375rem);
		color: var(--field-prefix-color, var(--color-text-muted, #6b7280));
		pointer-events: none;
		line-height: 1;
		font-variant-numeric: tabular-nums;
	}
	.money-input__field.has-prefix {
		padding-left: calc(var(--field-padding-x, 0.75rem) + 2.75rem);
	}

	/* field-control — shared box language. Identical token contract across
	   Input/Select/Textarea/MoneyInput/SizeInput so every control matches.
	   Tokens chain to the legacy input and color tokens so themes keep working. */
	.field-control {
		width: 100%;
		height: var(--field-height, 2.5rem);
		padding: var(--field-padding-y, 0.5rem) var(--field-padding-x, 0.75rem);
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
	.field-control:hover:not(:disabled):not(:focus) {
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
	.field-control.is-invalid,
	.field-control[aria-invalid='true'] {
		border-color: var(--field-border-error, var(--color-error, #dc2626));
	}
	.field-control.is-invalid:focus,
	.field-control[aria-invalid='true']:focus {
		box-shadow: 0 0 0 3px var(--field-ring-error, rgba(220, 38, 38, 0.18));
	}

	.money-input__caption {
		font-size: var(--field-help-size, 0.75rem);
		color: var(--field-help-color, var(--color-text-muted, #6b7280));
		padding-left: 0.25rem;
	}
	.is-disabled {
		opacity: 0.6;
	}
	.is-disabled .money-input__field {
		cursor: not-allowed;
		background: var(--field-disabled-bg, var(--color-bg-secondary, #fafafa));
	}
</style>
