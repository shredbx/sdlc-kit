<script lang="ts">
	/**
	 * SizeInput — numeric input with a land-unit dropdown.
	 *
	 * Storage: canonical sqm (via `bind:valueSqm`). Display unit is a UI
	 * preference that does not change the stored value when switched
	 * (Pattern A — quantity preserved). Typing "200" in sqm → store 200;
	 * switching unit to rai → input shows 0.125, store stays 200.
	 *
	 * Below the input, a small caption renders the equivalent in a
	 * different unit (sqm view shows "≈ 0.125 rai"; rai view shows
	 * "≈ 200 sqm") so the user can trust the conversion.
	 *
	 * @example
	 *   <SizeInput name="land_size" bind:valueSqm={landSize} defaultUnit="sqm" />
	 *
	 *   <SizeInput
	 *     name="house_size"
	 *     bind:valueSqm={houseSize}
	 *     units={['sqm', 'sqft']}
	 *     placeholder="0"
	 *   />
	 */

	import { untrack } from 'svelte';
	import {
		LAND_SIZE_UNITS,
		LAND_SIZE_UNITS_ORDERED,
		convert,
		formatLandSize,
		formatLandSizePlain,
		fromSqm,
		toSqm,
		type LandSizeUnit
	} from '@sbx/units';
	import { formatSizeValue } from './size-format';

	interface Props {
		valueSqm?: number | undefined;
		defaultUnit?: LandSizeUnit;
		units?: LandSizeUnit[];
		name?: string;
		id?: string;
		placeholder?: string;
		disabled?: boolean;
		min?: number;
		ariaInvalid?: boolean;
		/** id of an error/help node for aria-describedby (from <Field>). */
		'aria-describedby'?: string;
		/** Display format for the numeric input.
		 *  - `plain` (default): native `type="number"` behaviour.
		 *  - `comma`: thousands separators applied to the displayed value
		 *    when the input is not focused. The internal `valueSqm` stays a
		 *    plain number; the user can paste/type with commas (they're
		 *    stripped on parse). Switches the input to `type="text"` with
		 *    `inputmode="decimal"` so browsers don't reject the comma. */
		format?: 'plain' | 'comma';
		onchange?: (sqm: number | undefined) => void;
	}

	let {
		valueSqm = $bindable(undefined),
		defaultUnit = 'sqm',
		units = LAND_SIZE_UNITS_ORDERED,
		name,
		id,
		placeholder = '0',
		disabled = false,
		min = 0,
		ariaInvalid = false,
		'aria-describedby': ariaDescribedBy,
		format = 'plain',
		onchange
	}: Props = $props();

	let focused = $state(false);

	const autoId = `size-input-${Math.random().toString(36).slice(2, 9)}`;
	const inputId = $derived(id ?? autoId);

	// Initial display is seeded ONCE from the first prop snapshot. Switching
	// `defaultUnit` later doesn't reset the user's chosen display unit —
	// that's intentional, so we read via untrack() to skip the reactive sub.
	let displayUnit = $state<LandSizeUnit>(untrack(() => defaultUnit));
	let displayText = $state<string>(
		untrack(() => initialDisplayText(valueSqm, defaultUnit))
	);

	// Sync `displayText` when `valueSqm` is updated externally (e.g., by a
	// parent applying a polygon-derived land size). The local edit path
	// already writes to displayText, so we only re-sync if the input's
	// current value would round-trip to a DIFFERENT sqm than the prop. That
	// guard prevents a feedback loop where typing "200" → valueSqm becomes
	// 200 → effect rewrites displayText to "200" → input event fires …
	$effect(() => {
		const next = valueSqm;
		const parsedDisplay = parseFloat(displayText);
		const parsedSqm = Number.isFinite(parsedDisplay)
			? toSqm(parsedDisplay, displayUnit)
			: undefined;
		if (typeof next !== 'number' || !Number.isFinite(next)) {
			if (displayText !== '') displayText = '';
			return;
		}
		// 1-sqm tolerance — display rounds for some units, the canonical
		// stays exact. Without this we'd thrash on every render at the
		// last decimal place.
		if (parsedSqm == null || Math.abs(parsedSqm - next) >= 1) {
			displayText = roundForUnit(fromSqm(next, displayUnit), displayUnit);
		}
	});

	function initialDisplayText(sqm: number | undefined, unit: LandSizeUnit): string {
		if (typeof sqm !== 'number' || !Number.isFinite(sqm)) return '';
		const v = fromSqm(sqm, unit);
		return roundForUnit(v, unit);
	}

	// PRECISION IS OWNED BY THE UNIT SYSTEM — defer to the single tested formatter
	// (formatLandSizePlain) rather than re-rounding here. The old local `Math.max(dp,5)`
	// floor leaked float tails (3342.99997, 310.57486); switching to natural precision
	// then needed a whole-number guard ("3300" must not gut to "33") — both now live in
	// one place under vitest. Unit switches stay exact because handleUnitChange
	// re-expresses from the canonical sqm (valueSqm), not from this rounded text.
	function roundForUnit(value: number, unit: LandSizeUnit): string {
		return formatLandSizePlain(value, unit);
	}

	function handleInput(e: Event) {
		const raw = (e.target as HTMLInputElement).value;
		// In comma format, strip thousand-separators before parsing — the user
		// can type "12,345.50" or "12345.50" and both yield 12345.50. The
		// stored displayText is the plain form so the comma render is purely
		// presentational.
		const normalized = format === 'comma' ? raw.replace(/,/g, '') : raw;
		displayText = normalized;
		if (normalized.trim() === '') {
			valueSqm = undefined;
			onchange?.(undefined);
			return;
		}
		const parsed = parseFloat(normalized);
		if (!Number.isFinite(parsed)) return;
		const sqm = toSqm(parsed, displayUnit);
		valueSqm = sqm;
		onchange?.(sqm);
	}

	function handleUnitChange(e: Event) {
		const next = (e.target as HTMLSelectElement).value as LandSizeUnit;
		const prev = displayUnit;
		if (next === prev) return;
		displayUnit = next;
		// Pattern A — preserve the absolute quantity. Re-express from the CANONICAL
		// sqm (valueSqm) so the switch round-trips EXACTLY at the new unit's natural
		// precision — no compounding of display rounding (this is why roundForUnit can
		// safely drop to decimalPlacesFor). Fall back to converting the typed text only
		// when there is no canonical value yet (the user is mid-entry, before onchange).
		if (typeof valueSqm === 'number' && Number.isFinite(valueSqm)) {
			displayText = roundForUnit(fromSqm(valueSqm, next), next);
		} else {
			const parsed = parseFloat(displayText);
			if (Number.isFinite(parsed)) displayText = roundForUnit(convert(parsed, prev, next), next);
		}
	}

	// Caption: show the equivalent in the most informative alt unit. For
	// sqm display, show rai (Thai market convention). For rai/ngan/wah
	// display, show sqm. For sqft, show sqm.
	function altUnitFor(unit: LandSizeUnit): LandSizeUnit {
		if (unit === 'sqm') return 'rai';
		return 'sqm';
	}

	const caption = $derived.by((): string => {
		if (typeof valueSqm !== 'number' || !Number.isFinite(valueSqm) || valueSqm <= 0) return '';
		const alt = altUnitFor(displayUnit);
		const altValue = fromSqm(valueSqm, alt);
		const formatted = formatLandSize(altValue, alt);
		if (!formatted) return '';
		return `≈ ${formatted} ${LAND_SIZE_UNITS[alt].display}`;
	});

	const inputStep = $derived(displayUnit === 'sqm' || displayUnit === 'sqft' ? '1' : '0.01');

	// format-related derivations live AFTER displayText / focused are
	// declared so the dependency order is valid in Svelte 5 / TS.
	const inputType = $derived(format === 'comma' ? 'text' : 'number');
	const inputMode = $derived('decimal' as const);
	// Plain digits while focused (easy editing); comma-grouped on blur.
	const formattedDisplay = $derived(formatSizeValue(displayText, focused ? 'plain' : format));
</script>

<div class="size-input" class:is-disabled={disabled}>
	<div class="size-input__row" class:is-invalid={ariaInvalid}>
		<input
			id={inputId}
			{name}
			type={inputType}
			inputmode={inputMode}
			step={format === 'comma' ? undefined : inputStep}
			{min}
			{placeholder}
			{disabled}
			aria-invalid={ariaInvalid || undefined}
			aria-describedby={ariaDescribedBy}
			value={formattedDisplay}
			oninput={handleInput}
			onfocus={() => { focused = true; }}
			onblur={() => { focused = false; }}
			autocomplete="off"
			class="size-input__field"
		/>
		<select
			class="size-input__unit"
			aria-label="Unit"
			{disabled}
			value={displayUnit}
			onchange={handleUnitChange}
		>
			{#each units as u (u)}
				<option value={u}>{LAND_SIZE_UNITS[u].display}</option>
			{/each}
		</select>
	</div>
	{#if caption}
		<div class="size-input__caption" aria-live="polite">{caption}</div>
	{/if}
</div>

<style>
	.size-input {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		width: 100%;
	}
	/* Row owns the shared field box (border/radius/focus); the inner input is
	   borderless. Token contract identical to .field-control so the composite
	   control aligns with Input/Select/Textarea/MoneyInput. */
	.size-input__row {
		display: flex;
		align-items: stretch;
		height: var(--field-height, 2.5rem);
		border: var(--field-border-width, 1px) solid
			var(--field-border-color, var(--input-border, var(--color-border, #d4d4d8)));
		border-radius: var(--field-radius, var(--input-radius, var(--radius-md, 0.5rem)));
		overflow: hidden;
		background: var(--field-bg, var(--input-bg, var(--color-surface, #fff)));
		transition:
			border-color 0.15s ease,
			box-shadow 0.15s ease;
	}
	.size-input__row:hover:not(:focus-within) {
		border-color: var(--field-border-hover, var(--input-border-hover, var(--color-border-hover, #b8b8b8)));
	}
	.size-input__row:focus-within {
		border-color: var(--field-border-focus, var(--input-border-focus, var(--color-accent, #2563eb)));
		box-shadow: 0 0 0 3px var(--field-ring, var(--input-ring, rgba(37, 99, 235, 0.18)));
	}
	.size-input__row.is-invalid {
		border-color: var(--field-border-error, var(--color-error, #dc2626));
	}
	.size-input__row.is-invalid:focus-within {
		box-shadow: 0 0 0 3px var(--field-ring-error, rgba(220, 38, 38, 0.18));
	}
	.size-input__field {
		flex: 1 1 auto;
		min-width: 0;
		padding: var(--field-padding-y, 0.5rem) var(--field-padding-x, 0.75rem);
		border: 0;
		background: transparent;
		font-family: var(--field-font, inherit);
		font-size: var(--field-font-size, 0.9375rem);
		color: var(--field-text, var(--input-text, var(--color-text, #1f2937)));
		outline: none;
	}
	.size-input__field::placeholder {
		color: var(--field-placeholder, var(--input-placeholder, var(--color-text-muted, #9ca3af)));
		opacity: 1;
	}
	.size-input__unit {
		flex: 0 0 auto;
		padding: var(--field-padding-y, 0.5rem) 0.5rem;
		border: 0;
		border-left: var(--field-border-width, 1px) solid
			var(--field-border-color, var(--input-border, var(--color-border, #e5e7eb)));
		background: var(--field-addon-bg, var(--color-bg-secondary, #fafafa));
		font-family: var(--field-font, inherit);
		font-size: var(--field-font-size, 0.9375rem);
		color: var(--field-text-muted, var(--color-text-muted, #4b5563));
		cursor: pointer;
		appearance: none;
		-webkit-appearance: none;
		padding-right: 1.5rem;
		background-image: url("data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 12 12' fill='%236b7280'><path d='M6 8.5 1.5 4h9z'/></svg>");
		background-repeat: no-repeat;
		background-position: right 0.4rem center;
		background-size: 0.6rem 0.6rem;
	}
	.size-input__unit:focus {
		outline: none;
	}
	.size-input__caption {
		font-size: var(--field-help-size, 0.75rem);
		color: var(--field-help-color, var(--color-text-muted, #6b7280));
		padding-left: 0.25rem;
	}
	.is-disabled {
		opacity: 0.6;
	}
	.is-disabled .size-input__field,
	.is-disabled .size-input__unit {
		cursor: not-allowed;
	}
</style>
