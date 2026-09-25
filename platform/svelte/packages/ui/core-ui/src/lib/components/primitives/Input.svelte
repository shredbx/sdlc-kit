<script lang="ts">
	/**
	 * Input Primitive Component
	 *
	 * A flexible text input supporting various types, states, and configurations.
	 * Supports validation, icons, IView conformance, and proper accessibility.
	 *
	 * @example
	 * <Input placeholder="Enter text..." />
	 * <Input type="password" label="Password" />
	 * <Input value={query} leftIcon="🔍" clearable />
	 * <Input error="Required field" required />
	 */

	import type { Snippet } from 'svelte';
	import type { HTMLInputAttributes } from 'svelte/elements';

	type InputType =
		| 'text'
		| 'password'
		| 'email'
		| 'number'
		| 'search'
		| 'tel'
		| 'url'
		| 'date'
		| 'time'
		| 'datetime-local'
		| 'month'
		| 'week';
	type InputSize = 'sm' | 'md' | 'lg';
	type InputVariant = 'default' | 'filled' | 'outlined' | 'ghost';

	// Extends the native input attribute set so any valid HTML attribute the
	// caller needs (autocomplete, inputmode, pattern, min, max, …) is accepted
	// and correctly typed, and flows through `...restProps` onto the <input>.
	interface InputProps
		extends Omit<
			HTMLInputAttributes,
			| 'type'
			| 'size'
			| 'value'
			| 'class'
			| 'onchange'
			| 'oninput'
			| 'onfocus'
			| 'onblur'
			| 'onkeydown'
			| 'children'
		> {
		/** Current input value (bindable) */
		value?: string;
		/** Input type */
		type?: InputType;
		/** Placeholder text */
		placeholder?: string;
		/** Visible label above input */
		label?: string;
		/** Helper text below input */
		helperText?: string;
		/** Error message (displays error state) */
		error?: string;
		/** Size preset */
		size?: InputSize;
		/** Visual variant */
		variant?: InputVariant;
		/** Disabled state */
		disabled?: boolean;
		/** Read-only state */
		readonly?: boolean;
		/** Required field */
		required?: boolean;
		/** Auto-focus on mount */
		autofocus?: boolean;
		/** Maximum character length */
		maxlength?: number;
		/** Icon on left side */
		leftIcon?: string;
		/** Icon on right side */
		rightIcon?: string;
		/** Show clear button when has value */
		clearable?: boolean;
		/** Show character count */
		showCount?: boolean;
		/** Name attribute for forms */
		name?: string;
		/** Input ID (auto-generated if not provided) */
		id?: string;
		/** Additional CSS classes */
		class?: string;
		/** Custom left slot content */
		left?: Snippet;
		/** Custom right slot content */
		right?: Snippet;
		/** Change handler */
		onchange?: (value: string) => void;
		/** Input handler (fires on each keystroke) */
		oninput?: (value: string) => void;
		/** Focus handler */
		onfocus?: () => void;
		/** Blur handler */
		onblur?: () => void;
		/** Keydown handler */
		onkeydown?: (event: KeyboardEvent) => void;
		/** Accessible label for screen readers */
		'aria-label'?: string;
		/** IView: Data attributes for dev mode */
		'data-view-id'?: string;
	}

	let {
		value = $bindable(''),
		type = 'text',
		placeholder = '',
		label,
		helperText,
		error,
		size = 'md',
		variant = 'default',
		disabled = false,
		readonly = false,
		required = false,
		autofocus = false,
		maxlength,
		leftIcon,
		rightIcon,
		clearable = false,
		showCount = false,
		name,
		id,
		class: className = '',
		left,
		right,
		onchange,
		oninput,
		onfocus,
		onblur,
		onkeydown,
		'aria-label': ariaLabel,
		'data-view-id': viewId,
		...restProps
	}: InputProps = $props();

	// Generate unique ID if not provided
	let inputId = $derived(id ?? `input-${Math.random().toString(36).slice(2, 9)}`);

	// Character count
	let charCount = $derived(value?.length ?? 0);
	let showClearButton = $derived(clearable && value && value.length > 0 && !disabled && !readonly);

	// Handlers
	function handleInput(e: Event) {
		const target = e.target as HTMLInputElement;
		value = target.value;
		oninput?.(value);
	}

	function handleChange(e: Event) {
		const target = e.target as HTMLInputElement;
		value = target.value;
		onchange?.(value);
	}

	function handleClear() {
		value = '';
		onchange?.('');
		oninput?.('');
	}
</script>

<div
	class="input-wrapper input-{size} input-{variant}"
	class:has-error={!!error}
	class:disabled
	class:has-left-icon={!!leftIcon || !!left}
	class:has-right-icon={!!rightIcon || !!right || showClearButton}
	data-view-id={viewId}
>
	{#if label}
		<label for={inputId} class="input-label">
			{label}
			{#if required}
				<span class="required-marker">*</span>
			{/if}
		</label>
	{/if}

	<div class="input-container">
		{#if leftIcon}
			<span class="input-icon left">{leftIcon}</span>
		{:else if left}
			<span class="input-slot left">
				{@render left()}
			</span>
		{/if}

		<!-- svelte-ignore a11y_autofocus -->
		<input
			{...restProps}
			id={inputId}
			{type}
			{name}
			{placeholder}
			{disabled}
			{readonly}
			{required}
			{maxlength}
			bind:value
			class="input-field {className}"
			aria-invalid={!!error}
			aria-describedby={error ? `${inputId}-error` : helperText ? `${inputId}-helper` : undefined}
			autofocus={autofocus ? true : undefined}
			oninput={handleInput}
			onchange={handleChange}
			onfocus={onfocus}
			onblur={onblur}
			onkeydown={onkeydown}
			aria-label={ariaLabel}
		/>

		{#if showClearButton}
			<button
				type="button"
				class="input-clear"
				onclick={handleClear}
				aria-label="Clear input"
				tabindex="-1"
			>
				✕
			</button>
		{:else if rightIcon}
			<span class="input-icon right">{rightIcon}</span>
		{:else if right}
			<span class="input-slot right">
				{@render right()}
			</span>
		{/if}
	</div>

	{#if error || helperText || showCount}
		<div class="input-footer">
			{#if error}
				<span id="{inputId}-error" class="input-error" role="alert">{error}</span>
			{:else if helperText}
				<span id="{inputId}-helper" class="input-helper">{helperText}</span>
			{/if}
			{#if showCount}
				<span class="input-count">
					{charCount}{#if maxlength}/{maxlength}{/if}
				</span>
			{/if}
		</div>
	{/if}
</div>

<style>
	.input-wrapper {
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
		width: 100%;
	}

	.input-label {
		font-size: var(--field-label-size, 0.875rem);
		font-weight: var(--field-label-weight, 500);
		color: var(--field-label-color, var(--input-label-color, var(--color-text, #fff)));
	}

	.required-marker {
		color: var(--field-required-color, var(--color-error, #ef4444));
		margin-left: 0.125rem;
	}

	.input-container {
		position: relative;
		display: flex;
		align-items: center;
	}

	/* --field-* is the shared cross-primitive contract; it chains to the legacy
	   input and color tokens so existing consumers (which never set --field-*)
	   render exactly as before, while consumers that adopt --field-* get the
	   unified look shared with Select/Textarea/MoneyInput/SizeInput. */
	.input-field {
		width: 100%;
		background: var(--field-bg, var(--input-bg, var(--color-bg-tertiary, #252540)));
		border: var(--field-border-width, 1px) solid
			var(--field-border-color, var(--input-border, var(--color-border, #2a2a4a)));
		border-radius: var(--field-radius, var(--input-radius, 0.375rem));
		color: var(--field-text, var(--input-text, var(--color-text, #fff)));
		font-size: inherit;
		font-family: var(--field-font, inherit);
		transition: all 0.15s ease;
	}

	.input-field::placeholder {
		color: var(--field-placeholder, var(--input-placeholder, var(--color-text-muted, #6b7280)));
	}

	.input-field:hover:not(:disabled):not(:focus) {
		border-color: var(--field-border-hover, var(--input-border-hover, var(--color-border-hover, #3a3a5a)));
	}

	.input-field:focus {
		outline: none;
		border-color: var(--field-border-focus, var(--input-border-focus, var(--color-accent, #6366f1)));
		box-shadow: 0 0 0 3px var(--field-ring, var(--input-ring, rgba(99, 102, 241, 0.2)));
	}

	.input-field:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.input-field:read-only {
		background: var(--input-bg-readonly, var(--color-bg-secondary, #1a1a2e));
	}

	/* Sizes. The default `md` honours the shared --field-* sizing tokens (height,
	   padding, font-size) when a consumer sets them, falling back to the original
	   values otherwise. sm/lg keep their explicit scale. */
	.input-sm .input-field {
		padding: 0.375rem 0.5rem;
		font-size: 0.75rem;
	}

	.input-md .input-field {
		height: var(--field-height, auto);
		padding: var(--field-padding-y, 0.5rem) var(--field-padding-x, 0.75rem);
		font-size: var(--field-font-size, 0.875rem);
	}

	.input-lg .input-field {
		padding: 0.625rem 1rem;
		font-size: 1rem;
	}

	/* Icons */
	.input-icon,
	.input-slot {
		position: absolute;
		display: flex;
		align-items: center;
		justify-content: center;
		pointer-events: none;
		color: var(--input-icon-color, var(--color-text-muted, #9ca3af));
	}

	.input-icon.left,
	.input-slot.left {
		left: 0.75rem;
	}

	.input-icon.right,
	.input-slot.right {
		right: 0.75rem;
	}

	.has-left-icon .input-field {
		padding-left: 2.25rem;
	}

	.has-right-icon .input-field {
		padding-right: 2.25rem;
	}

	/* Clear button */
	.input-clear {
		position: absolute;
		right: 0.5rem;
		display: flex;
		align-items: center;
		justify-content: center;
		width: 1.25rem;
		height: 1.25rem;
		padding: 0;
		background: var(--input-clear-bg, rgba(255, 255, 255, 0.1));
		border: none;
		border-radius: 50%;
		color: var(--input-clear-color, var(--color-text-muted, #9ca3af));
		font-size: 0.625rem;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.input-clear:hover {
		background: var(--input-clear-bg-hover, rgba(255, 255, 255, 0.2));
		color: var(--input-clear-color-hover, var(--color-text, #fff));
	}

	/* Footer */
	.input-footer {
		display: flex;
		justify-content: space-between;
		font-size: 0.75rem;
	}

	.input-helper {
		color: var(--input-helper-color, var(--color-text-muted, #9ca3af));
	}

	.input-error {
		color: var(--color-error, #ef4444);
	}

	.input-count {
		color: var(--input-count-color, var(--color-text-muted, #6b7280));
		margin-left: auto;
	}

	/* Error state */
	.has-error .input-field {
		border-color: var(--color-error, #ef4444);
	}

	.has-error .input-field:focus {
		box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.2);
	}

	/* Variants */
	.input-filled .input-field {
		border-color: transparent;
		background: var(--input-filled-bg, var(--color-bg-tertiary, #252540));
	}

	.input-filled .input-field:hover:not(:disabled):not(:focus) {
		background: var(--input-filled-bg-hover, rgba(255, 255, 255, 0.05));
	}

	.input-outlined .input-field {
		background: transparent;
	}

	.input-ghost .input-field {
		background: transparent;
		border-color: transparent;
	}

	.input-ghost .input-field:hover:not(:disabled):not(:focus) {
		background: var(--input-ghost-bg-hover, rgba(255, 255, 255, 0.05));
	}

	/* Disabled wrapper */
	.disabled {
		pointer-events: none;
	}
</style>
