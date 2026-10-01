<script lang="ts">
	/**
	 * Select Primitive Component
	 *
	 * A generic dropdown select with Svelte 5 patterns.
	 * NOT dictionary-specific — a raw <select> primitive.
	 *
	 * @example
	 * <Select options={[{value: 'a', label: 'Alpha'}]} bind:value />
	 * <Select options={items} placeholder="Choose..." size="sm" />
	 */

	type SelectSize = 'sm' | 'md' | 'lg';
	type SelectVariant = 'default' | 'filled' | 'outlined' | 'ghost';

	interface SelectOption {
		value: string;
		label: string;
		disabled?: boolean;
	}

	interface SelectProps {
		/** Selected value (bindable) */
		value?: string;
		/** Available options */
		options?: SelectOption[];
		/** Placeholder text when no value selected */
		placeholder?: string;
		/** Visible label above select */
		label?: string;
		/** Helper text below select */
		helperText?: string;
		/** Error message */
		error?: string;
		/** Size preset */
		size?: SelectSize;
		/** Visual variant */
		variant?: SelectVariant;
		/** Disabled state */
		disabled?: boolean;
		/** Required field */
		required?: boolean;
		/** Name attribute for forms */
		name?: string;
		/** Select ID */
		id?: string;
		/** Additional CSS classes */
		class?: string;
		/** Change handler */
		onchange?: (value: string) => void;
		/** Accessible label for screen readers (mirrors Input) — flows onto the
		 *  <select> via restProps for label-less form rows. */
		'aria-label'?: string;
		/** IView: Data attributes */
		'data-view-id'?: string;
	}

	let {
		value = $bindable(''),
		options = [],
		placeholder,
		label: selectLabel,
		helperText,
		error,
		size = 'md',
		variant = 'default',
		disabled = false,
		required = false,
		name,
		id,
		class: className = '',
		onchange,
		'data-view-id': viewId,
		...restProps
	}: SelectProps = $props();

	let selectId = $derived(id ?? `select-${Math.random().toString(36).slice(2, 9)}`);

	function handleChange(e: Event) {
		const target = e.target as HTMLSelectElement;
		value = target.value;
		onchange?.(value);
	}
</script>

<div
	class="select-wrapper select-{size} select-{variant}"
	class:has-error={!!error}
	class:disabled
	data-view-id={viewId}
>
	{#if selectLabel}
		<label for={selectId} class="select-label">
			{selectLabel}
			{#if required}
				<span class="required-marker">*</span>
			{/if}
		</label>
	{/if}

	<div class="select-container">
		<select
			{...restProps}
			id={selectId}
			{name}
			{disabled}
			{required}
			bind:value
			class="select-field {className}"
			aria-invalid={!!error}
			aria-describedby={error ? `${selectId}-error` : helperText ? `${selectId}-helper` : undefined}
			onchange={handleChange}
		>
			{#if placeholder}
				<option value="" disabled>{placeholder}</option>
			{/if}
			{#each options as opt}
				<option value={opt.value} disabled={opt.disabled}>{opt.label}</option>
			{/each}
		</select>
		<span class="select-chevron">&#x25BE;</span>
	</div>

	{#if error || helperText}
		<div class="select-footer">
			{#if error}
				<span id="{selectId}-error" class="select-error" role="alert">{error}</span>
			{:else if helperText}
				<span id="{selectId}-helper" class="select-helper">{helperText}</span>
			{/if}
		</div>
	{/if}
</div>

<style>
	.select-wrapper {
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
		width: 100%;
	}

	.select-label {
		font-size: var(--field-label-size, 0.875rem);
		font-weight: var(--field-label-weight, 500);
		color: var(--field-label-color, var(--input-label-color, var(--color-text, #fff)));
	}

	.required-marker {
		color: var(--field-required-color, var(--color-error, #ef4444));
		margin-left: 0.125rem;
	}

	.select-container {
		position: relative;
		display: flex;
		align-items: center;
	}

	/* --field-* shared contract chained over the legacy input/color tokens —
	   additive, so existing consumers are unchanged and adopters get the
	   unified look (see Input.svelte for the same pattern). */
	.select-field {
		width: 100%;
		appearance: none;
		background: var(--field-bg, var(--input-bg, var(--color-bg-tertiary, #252540)));
		border: var(--field-border-width, 1px) solid
			var(--field-border-color, var(--input-border, var(--color-border, #2a2a4a)));
		border-radius: var(--field-radius, var(--input-radius, 0.375rem));
		color: var(--field-text, var(--input-text, var(--color-text, #fff)));
		font-size: inherit;
		font-family: var(--field-font, inherit);
		cursor: pointer;
		transition: all 0.15s ease;
		padding-right: 2rem;
	}

	.select-field option {
		background: var(--field-bg, var(--color-bg-tertiary, #252540));
		color: var(--field-text, var(--color-text, #fff));
	}

	.select-field:hover:not(:disabled):not(:focus) {
		border-color: var(--field-border-hover, var(--input-border-hover, var(--color-border-hover, #3a3a5a)));
	}

	.select-field:focus {
		outline: none;
		border-color: var(--field-border-focus, var(--input-border-focus, var(--color-accent, #6366f1)));
		box-shadow: 0 0 0 3px var(--field-ring, var(--input-ring, rgba(99, 102, 241, 0.2)));
	}

	.select-field:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.select-chevron {
		position: absolute;
		right: 0.75rem;
		pointer-events: none;
		color: var(--color-text-muted, #6b7280);
		font-size: 0.75rem;
	}

	/* Sizes */
	.select-sm .select-field {
		padding: 0.375rem 0.5rem;
		font-size: 0.75rem;
	}

	.select-md .select-field {
		height: var(--field-height, auto);
		padding: var(--field-padding-y, 0.5rem) var(--field-padding-x, 0.75rem);
		font-size: var(--field-font-size, 0.875rem);
		padding-right: 2rem;
	}

	.select-lg .select-field {
		padding: 0.625rem 1rem;
		font-size: 1rem;
	}

	/* Footer */
	.select-footer {
		font-size: 0.75rem;
	}

	.select-helper {
		color: var(--input-helper-color, var(--color-text-muted, #9ca3af));
	}

	.select-error {
		color: var(--color-error, #ef4444);
	}

	/* Error state */
	.has-error .select-field {
		border-color: var(--color-error, #ef4444);
	}

	.has-error .select-field:focus {
		box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.2);
	}

	/* Variants */
	.select-filled .select-field {
		border-color: transparent;
		background: var(--input-filled-bg, var(--color-bg-tertiary, #252540));
	}

	.select-outlined .select-field {
		background: transparent;
	}

	.select-ghost .select-field {
		background: transparent;
		border-color: transparent;
	}

	.disabled {
		pointer-events: none;
	}
</style>
